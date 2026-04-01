package cloudfront

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Distribution represents a CloudFront distribution.
type Distribution struct {
	ID                string
	ARN               string
	DomainName        string
	Status            string // Deployed, InProgress
	Enabled           bool
	Comment           string
	DefaultRootObject string
	PriceClass        string
	Origins           []Origin
	DefaultCacheBehavior *CacheBehavior
	CacheBehaviors    []CacheBehavior
	CallerReference   string
	ETag              string
	LastModified      time.Time
	Tags              map[string]string
	Lifecycle         *lifecycle.Machine
}

// Origin represents a CloudFront origin.
type Origin struct {
	ID         string
	DomainName string
	OriginPath string
	S3Config   *S3OriginConfig
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
	PathPattern        string
	TargetOriginID     string
	ViewerProtocolPolicy string
	AllowedMethods     []string
	CachedMethods      []string
	ForwardedValues    *ForwardedValues
	MinTTL             int64
	MaxTTL             int64
	DefaultTTL         int64
	Compress           bool
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

// Store manages all CloudFront resources.
type Store struct {
	mu             sync.RWMutex
	distributions  map[string]*Distribution  // keyed by ID
	invalidations  map[string][]*Invalidation // keyed by distribution ID
	accountID      string
	region         string
	lifecycleCfg   *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		distributions: make(map[string]*Distribution),
		invalidations: make(map[string][]*Invalidation),
		accountID:     accountID,
		region:        region,
		lifecycleCfg:  lifecycle.DefaultConfig(),
	}
}

// ---- ID/ARN helpers ----

func newDistributionID() string {
	return fmt.Sprintf("E%s", randomUpperHex(13))
}

func newInvalidationID() string {
	return fmt.Sprintf("I%s", randomUpperHex(13))
}

func (s *Store) distributionARN(id string) string {
	return fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s", s.accountID, id)
}

func domainName(id string) string {
	// Generate realistic CloudFront distribution domain: d{13alphanumeric}.cloudfront.net
	return fmt.Sprintf("d%s.cloudfront.net", strings.ToLower(randomUpperHex(13)))
}

func newETag() string {
	return fmt.Sprintf("E%s", randomHex(16))
}

// ---- Distribution operations ----

func (s *Store) CreateDistribution(callerRef, comment, defaultRootObject, priceClass string, enabled bool, origins []Origin, defaultBehavior *CacheBehavior, behaviors []CacheBehavior) *Distribution {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := newDistributionID()
	if priceClass == "" {
		priceClass = "PriceClass_All"
	}

	transitions := []lifecycle.Transition{
		{From: "InProgress", To: "Deployed", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("InProgress", transitions, s.lifecycleCfg)

	dist := &Distribution{
		ID:                id,
		ARN:               s.distributionARN(id),
		DomainName:        domainName(id),
		Status:            "InProgress",
		Enabled:           enabled,
		Comment:           comment,
		DefaultRootObject: defaultRootObject,
		PriceClass:        priceClass,
		Origins:           origins,
		DefaultCacheBehavior: defaultBehavior,
		CacheBehaviors:    behaviors,
		CallerReference:   callerRef,
		ETag:              newETag(),
		LastModified:      time.Now().UTC(),
		Tags:              make(map[string]string),
		Lifecycle:         lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if d, ok := s.distributions[id]; ok {
			d.Status = string(to)
		}
	})

	s.distributions[id] = dist
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

func (s *Store) UpdateDistribution(id, comment, defaultRootObject, priceClass string, enabled *bool, origins []Origin, defaultBehavior *CacheBehavior, behaviors []CacheBehavior) (*Distribution, bool) {
	s.mu.Lock()

	d, ok := s.distributions[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
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
	lc := d.Lifecycle
	s.mu.Unlock()

	// ForceState may trigger OnTransition callback that acquires s.mu.
	if lc != nil {
		lc.ForceState("InProgress")
	}

	return d, true
}

func (s *Store) DeleteDistribution(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.distributions[id]
	if !ok {
		return false
	}
	if d.Lifecycle != nil {
		d.Lifecycle.Stop()
	}
	delete(s.distributions, id)
	delete(s.invalidations, id)
	return true
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
