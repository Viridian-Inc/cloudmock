package cloudfront

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// Distribution represents a CloudFront distribution.
type Distribution struct {
	ID                   string
	ARN                  string
	DomainName           string
	Status               string // Deployed, InProgress
	Enabled              bool
	Comment              string
	DefaultRootObject    string
	PriceClass           string
	Origins              []Origin
	DefaultCacheBehavior *CacheBehavior
	CacheBehaviors       []CacheBehavior
	CallerReference      string
	ETag                 string
	LastModified         time.Time
	Tags                 map[string]string
	Lifecycle            *lifecycle.Machine
}

// Origin represents a CloudFront origin.
type Origin struct {
	ID           string
	DomainName   string
	OriginPath   string
	S3Config     *S3OriginConfig
	CustomConfig *CustomOriginConfig
}

// S3OriginConfig holds S3-specific origin settings.
type S3OriginConfig struct {
	OriginAccessIdentity string
}

// CustomOriginConfig holds custom origin settings.
type CustomOriginConfig struct {
	HTTPPort             int
	HTTPSPort            int
	OriginProtocolPolicy string
}

// CacheBehavior represents a cache behavior.
type CacheBehavior struct {
	PathPattern          string
	TargetOriginID       string
	ViewerProtocolPolicy string
	AllowedMethods       []string
	CachedMethods        []string
	ForwardedValues      *ForwardedValues
	MinTTL               int64
	MaxTTL               int64
	DefaultTTL           int64
	Compress             bool
}

// ForwardedValues controls what is forwarded to the origin.
type ForwardedValues struct {
	QueryString bool
	Cookies     string // none, whitelist, all
	Headers     []string
}

// Invalidation represents a CloudFront invalidation.
type Invalidation struct {
	ID              string
	DistributionID  string
	Status          string // InProgress, Completed
	CallerReference string
	Paths           []string
	CreateTime      time.Time
	Lifecycle       *lifecycle.Machine
}

// CachePolicy represents a CloudFront cache policy.
type CachePolicy struct {
	ID               string
	Name             string
	Comment          string
	DefaultTTL       int64
	MaxTTL           int64
	MinTTL           int64
	ParametersInCacheKeyAndForwardedToOrigin map[string]any
	ETag             string
	LastModified     time.Time
}

// OriginRequestPolicy represents a CloudFront origin request policy.
type OriginRequestPolicy struct {
	ID               string
	Name             string
	Comment          string
	HeadersConfig    map[string]any
	CookiesConfig    map[string]any
	QueryStringsConfig map[string]any
	ETag             string
	LastModified     time.Time
}

// Function represents a CloudFront Function.
type Function struct {
	Name             string
	Comment          string
	Runtime          string // cloudfront-js-1.0, cloudfront-js-2.0
	Stage            string // DEVELOPMENT, LIVE
	Status           string // UNPUBLISHED, DEPLOYED
	ARN              string
	FunctionCode     []byte
	CreatedAt        time.Time
	LastModifiedAt   time.Time
	ETag             string
}

// Store manages all CloudFront resources.
type Store struct {
	mu                   sync.RWMutex
	distributions        map[string]*Distribution   // keyed by ID
	callerRefIndex       map[string]string           // callerReference -> distID
	invalidations        map[string][]*Invalidation  // keyed by distribution ID
	cachePolicies        map[string]*CachePolicy     // keyed by ID
	originRequestPolicies map[string]*OriginRequestPolicy // keyed by ID
	functions            map[string]*Function        // keyed by name
	accountID            string
	region               string
	lifecycleCfg         *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		distributions:        make(map[string]*Distribution),
		callerRefIndex:       make(map[string]string),
		invalidations:        make(map[string][]*Invalidation),
		cachePolicies:        make(map[string]*CachePolicy),
		originRequestPolicies: make(map[string]*OriginRequestPolicy),
		functions:            make(map[string]*Function),
		accountID:            accountID,
		region:               region,
		lifecycleCfg:         lifecycle.DefaultConfig(),
	}
}

// ---- ID/ARN helpers ----

func newDistributionID() string {
	return fmt.Sprintf("E%s", randomUpperHex(13))
}

func newInvalidationID() string {
	return fmt.Sprintf("I%s", randomUpperHex(13))
}

func newCachePolicyID() string {
	return newUUIDStr()
}

func newOriginRequestPolicyID() string {
	return newUUIDStr()
}

func newUUIDStr() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) distributionARN(id string) string {
	return fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s", s.accountID, id)
}

func (s *Store) functionARN(name string) string {
	return fmt.Sprintf("arn:aws:cloudfront::%s:function/%s", s.accountID, name)
}

func domainName(id string) string {
	// Generate realistic CloudFront distribution domain: d{13alphanumeric}.cloudfront.net
	return fmt.Sprintf("d%s.cloudfront.net", strings.ToLower(randomUpperHex(13)))
}

func newETag() string {
	return fmt.Sprintf("E%s", randomHex(16))
}

// ---- Distribution operations ----

// FindByCallerReference returns an existing distribution with the same CallerReference, if any.
func (s *Store) FindByCallerReference(callerRef string) (*Distribution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id, ok := s.callerRefIndex[callerRef]; ok {
		d, ok := s.distributions[id]
		return d, ok
	}
	return nil, false
}

func (s *Store) CreateDistribution(callerRef, comment, defaultRootObject, priceClass string, enabled bool, origins []Origin, defaultBehavior *CacheBehavior, behaviors []CacheBehavior) *Distribution {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Caller reference deduplication
	if id, ok := s.callerRefIndex[callerRef]; ok {
		if d, ok := s.distributions[id]; ok {
			return d
		}
	}

	id := newDistributionID()
	if priceClass == "" {
		priceClass = "PriceClass_All"
	}

	transitions := []lifecycle.Transition{
		{From: "InProgress", To: "Deployed", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("InProgress", transitions, s.lifecycleCfg)

	dist := &Distribution{
		ID:                   id,
		ARN:                  s.distributionARN(id),
		DomainName:           domainName(id),
		Status:               "InProgress",
		Enabled:              enabled,
		Comment:              comment,
		DefaultRootObject:    defaultRootObject,
		PriceClass:           priceClass,
		Origins:              origins,
		DefaultCacheBehavior: defaultBehavior,
		CacheBehaviors:       behaviors,
		CallerReference:      callerRef,
		ETag:                 newETag(),
		LastModified:         time.Now().UTC(),
		Tags:                 make(map[string]string),
		Lifecycle:            lm,
	}

	// Use a goroutine-safe callback that doesn't need s.mu.
	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if d, ok := s.distributions[id]; ok {
			d.Status = string(to)
		}
	})

	s.distributions[id] = dist
	if callerRef != "" {
		s.callerRefIndex[callerRef] = id
	}
	return dist
}

func (s *Store) GetDistribution(id string) (*Distribution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.distributions[id]
	return d, ok
}

func (s *Store) ListDistributions() []*Distribution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Distribution, 0, len(s.distributions))
	for _, d := range s.distributions {
		result = append(result, d)
	}
	return result
}

// UpdateDistribution updates a distribution. Returns (dist, true) on success,
// (nil, false) if not found, and specific errors for ETag/enabled state issues.
func (s *Store) UpdateDistribution(id, ifMatch, comment, defaultRootObject, priceClass string, enabled *bool, origins []Origin, defaultBehavior *CacheBehavior, behaviors []CacheBehavior) (*Distribution, bool, error) {
	s.mu.Lock()

	d, ok := s.distributions[id]
	if !ok {
		s.mu.Unlock()
		return nil, false, nil
	}

	// ETag / If-Match validation
	if ifMatch != "" && d.ETag != ifMatch {
		s.mu.Unlock()
		return nil, true, fmt.Errorf("InvalidIfMatchVersion")
	}

	if comment != "" {
		d.Comment = comment
	}
	if defaultRootObject != "" {
		d.DefaultRootObject = defaultRootObject
	}
	if priceClass != "" {
		d.PriceClass = priceClass
	}
	if enabled != nil {
		d.Enabled = *enabled
	}
	if len(origins) > 0 {
		d.Origins = origins
	}
	if defaultBehavior != nil {
		d.DefaultCacheBehavior = defaultBehavior
	}
	if len(behaviors) > 0 {
		d.CacheBehaviors = behaviors
	}
	d.ETag = newETag()
	d.LastModified = time.Now().UTC()
	d.Status = "InProgress"
	s.mu.Unlock()

	// Schedule transition back to Deployed without holding s.mu.
	// Use a simple goroutine since lifecycle machine handles this itself.
	go func() {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		if dist, ok := s.distributions[id]; ok && dist.Status == "InProgress" {
			dist.Status = "Deployed"
		}
		s.mu.Unlock()
	}()

	s.mu.RLock()
	d2 := s.distributions[id]
	s.mu.RUnlock()

	return d2, true, nil
}

// DeleteDistribution removes a distribution. Returns specific error strings for
// not-found or not-disabled cases.
func (s *Store) DeleteDistribution(id, ifMatch string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.distributions[id]
	if !ok {
		return false, nil
	}

	// ETag validation
	if ifMatch != "" && d.ETag != ifMatch {
		return true, fmt.Errorf("InvalidIfMatchVersion")
	}

	// Must be disabled before deletion
	if d.Enabled {
		return true, fmt.Errorf("DistributionNotDisabled")
	}

	if d.Lifecycle != nil {
		d.Lifecycle.Stop()
	}
	delete(s.distributions, id)
	if d.CallerReference != "" {
		delete(s.callerRefIndex, d.CallerReference)
	}
	delete(s.invalidations, id)
	return true, nil
}

// ---- Invalidation operations ----

func (s *Store) CreateInvalidation(distID, callerRef string, paths []string) (*Invalidation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.distributions[distID]; !ok {
		return nil, false
	}

	id := newInvalidationID()
	transitions := []lifecycle.Transition{
		{From: "InProgress", To: "Completed", Delay: 1 * time.Second},
	}
	lm := lifecycle.NewMachine("InProgress", transitions, s.lifecycleCfg)

	inv := &Invalidation{
		ID:              id,
		DistributionID:  distID,
		Status:          "InProgress",
		CallerReference: callerRef,
		Paths:           paths,
		CreateTime:      time.Now().UTC(),
		Lifecycle:       lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		inv.Status = string(to)
	})

	s.invalidations[distID] = append(s.invalidations[distID], inv)
	return inv, true
}

func (s *Store) GetInvalidation(distID, invID string) (*Invalidation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	invs, ok := s.invalidations[distID]
	if !ok {
		return nil, false
	}
	for _, inv := range invs {
		if inv.ID == invID {
			return inv, true
		}
	}
	return nil, false
}

func (s *Store) ListInvalidations(distID string) ([]*Invalidation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.distributions[distID]; !ok {
		return nil, false
	}
	invs := s.invalidations[distID]
	result := make([]*Invalidation, len(invs))
	copy(result, invs)
	return result, true
}

// ---- Cache Policy operations ----

func (s *Store) CreateCachePolicy(name, comment string, defaultTTL, maxTTL, minTTL int64, params map[string]any) (*CachePolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate name
	for _, cp := range s.cachePolicies {
		if cp.Name == name {
			return nil, false
		}
	}

	id := newCachePolicyID()
	cp := &CachePolicy{
		ID:               id,
		Name:             name,
		Comment:          comment,
		DefaultTTL:       defaultTTL,
		MaxTTL:           maxTTL,
		MinTTL:           minTTL,
		ParametersInCacheKeyAndForwardedToOrigin: params,
		ETag:             newETag(),
		LastModified:     time.Now().UTC(),
	}
	s.cachePolicies[id] = cp
	return cp, true
}

func (s *Store) GetCachePolicy(id string) (*CachePolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp, ok := s.cachePolicies[id]
	return cp, ok
}

func (s *Store) ListCachePolicies() []*CachePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*CachePolicy, 0, len(s.cachePolicies))
	for _, cp := range s.cachePolicies {
		out = append(out, cp)
	}
	return out
}

func (s *Store) UpdateCachePolicy(id, ifMatch, name, comment string, defaultTTL, maxTTL, minTTL int64, params map[string]any) (*CachePolicy, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.cachePolicies[id]
	if !ok {
		return nil, false, nil
	}
	if ifMatch != "" && cp.ETag != ifMatch {
		return nil, true, fmt.Errorf("InvalidIfMatchVersion")
	}
	if name != "" {
		cp.Name = name
	}
	if comment != "" {
		cp.Comment = comment
	}
	cp.DefaultTTL = defaultTTL
	cp.MaxTTL = maxTTL
	cp.MinTTL = minTTL
	if params != nil {
		cp.ParametersInCacheKeyAndForwardedToOrigin = params
	}
	cp.ETag = newETag()
	cp.LastModified = time.Now().UTC()
	return cp, true, nil
}

func (s *Store) DeleteCachePolicy(id, ifMatch string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.cachePolicies[id]
	if !ok {
		return false, nil
	}
	if ifMatch != "" && cp.ETag != ifMatch {
		return true, fmt.Errorf("InvalidIfMatchVersion")
	}
	delete(s.cachePolicies, id)
	return true, nil
}

// ---- Origin Request Policy operations ----

func (s *Store) CreateOriginRequestPolicy(name, comment string, headersConfig, cookiesConfig, queryStringsConfig map[string]any) (*OriginRequestPolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, orp := range s.originRequestPolicies {
		if orp.Name == name {
			return nil, false
		}
	}

	id := newOriginRequestPolicyID()
	orp := &OriginRequestPolicy{
		ID:                 id,
		Name:               name,
		Comment:            comment,
		HeadersConfig:      headersConfig,
		CookiesConfig:      cookiesConfig,
		QueryStringsConfig: queryStringsConfig,
		ETag:               newETag(),
		LastModified:       time.Now().UTC(),
	}
	s.originRequestPolicies[id] = orp
	return orp, true
}

func (s *Store) GetOriginRequestPolicy(id string) (*OriginRequestPolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orp, ok := s.originRequestPolicies[id]
	return orp, ok
}

func (s *Store) ListOriginRequestPolicies() []*OriginRequestPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*OriginRequestPolicy, 0, len(s.originRequestPolicies))
	for _, orp := range s.originRequestPolicies {
		out = append(out, orp)
	}
	return out
}

func (s *Store) DeleteOriginRequestPolicy(id, ifMatch string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	orp, ok := s.originRequestPolicies[id]
	if !ok {
		return false, nil
	}
	if ifMatch != "" && orp.ETag != ifMatch {
		return true, fmt.Errorf("InvalidIfMatchVersion")
	}
	delete(s.originRequestPolicies, id)
	return true, nil
}

// ---- Function operations ----

func (s *Store) CreateFunction(name, comment, runtime string, code []byte) (*Function, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.functions[name]; exists {
		return nil, false
	}

	now := time.Now().UTC()
	fn := &Function{
		Name:           name,
		Comment:        comment,
		Runtime:        runtime,
		Stage:          "DEVELOPMENT",
		Status:         "UNPUBLISHED",
		ARN:            s.functionARN(name),
		FunctionCode:   code,
		CreatedAt:      now,
		LastModifiedAt: now,
		ETag:           newETag(),
	}
	s.functions[name] = fn
	return fn, true
}

func (s *Store) GetFunction(name, stage string) (*Function, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn, ok := s.functions[name]
	if !ok {
		return nil, false
	}
	if stage != "" && fn.Stage != stage {
		return nil, false
	}
	return fn, ok
}

func (s *Store) ListFunctions(stage string) []*Function {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Function, 0, len(s.functions))
	for _, fn := range s.functions {
		if stage == "" || fn.Stage == stage {
			out = append(out, fn)
		}
	}
	return out
}

func (s *Store) UpdateFunction(name, ifMatch, comment, runtime string, code []byte) (*Function, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn, ok := s.functions[name]
	if !ok {
		return nil, false, nil
	}
	if ifMatch != "" && fn.ETag != ifMatch {
		return nil, true, fmt.Errorf("InvalidIfMatchVersion")
	}
	if comment != "" {
		fn.Comment = comment
	}
	if runtime != "" {
		fn.Runtime = runtime
	}
	if len(code) > 0 {
		fn.FunctionCode = code
	}
	fn.LastModifiedAt = time.Now().UTC()
	fn.ETag = newETag()
	return fn, true, nil
}

func (s *Store) DeleteFunction(name, ifMatch string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn, ok := s.functions[name]
	if !ok {
		return false, nil
	}
	if ifMatch != "" && fn.ETag != ifMatch {
		return true, fmt.Errorf("InvalidIfMatchVersion")
	}
	delete(s.functions, name)
	return true, nil
}

func (s *Store) PublishFunction(name, ifMatch string) (*Function, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn, ok := s.functions[name]
	if !ok {
		return nil, false, nil
	}
	if ifMatch != "" && fn.ETag != ifMatch {
		return nil, true, fmt.Errorf("InvalidIfMatchVersion")
	}
	fn.Stage = "LIVE"
	fn.Status = "DEPLOYED"
	fn.ETag = newETag()
	fn.LastModifiedAt = time.Now().UTC()
	return fn, true, nil
}

// TestFunction simulates running a CloudFront function against test event data.
// Returns computed result (event echo) or error.
func (s *Store) TestFunction(name, stage string, eventObject []byte) (map[string]any, bool) {
	s.mu.RLock()
	fn, ok := s.functions[name]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	result := map[string]any{
		"FunctionSummary": map[string]any{
			"Name":   fn.Name,
			"Stage":  fn.Stage,
			"Status": fn.Status,
		},
		"TestResult": map[string]any{
			"FunctionExecutionLogs": []string{"Function executed successfully."},
			"FunctionErrorMessage":  "",
			"FunctionOutput":        string(eventObject),
			"ComputeUtilization":    "10",
		},
	}
	return result, true
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.distributions {
		if d.ARN == arn {
			for k, v := range tags {
				d.Tags[k] = v
			}
			return true
		}
	}
	return false
}

func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.distributions {
		if d.ARN == arn {
			for _, k := range keys {
				delete(d.Tags, k)
			}
			return true
		}
	}
	return false
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, d := range s.distributions {
		if d.ARN == arn {
			result := make(map[string]string, len(d.Tags))
			for k, v := range d.Tags {
				result[k] = v
			}
			return result, true
		}
	}
	return nil, false
}

// ---- utility ----

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func randomUpperHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	hex := fmt.Sprintf("%X", b)
	if len(hex) > n {
		hex = hex[:n]
	}
	return hex
}
