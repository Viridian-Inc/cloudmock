package xray

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- Data types ----

type TraceSegment struct {
	Document  string
	TraceID   string
	Timestamp time.Time
}

type SamplingRule struct {
	RuleName     string
	RuleARN      string
	Priority     int
	FixedRate    float64
	ReservoirSize int
	ServiceName  string
	ServiceType  string
	Host         string
	HTTPMethod   string
	URLPath      string
	Version      int
	CreatedAt    time.Time
	ModifiedAt   time.Time
	Tags         map[string]string
}

type Group struct {
	GroupName        string
	GroupARN         string
	FilterExpression string
	InsightEnabled   bool
	CreatedAt        time.Time
	ModifiedAt       time.Time
	Tags             map[string]string
}

type EncryptionConfig struct {
	KeyID  string
	Status string
	Type   string
}

// ---- Store ----

type Store struct {
	mu               sync.RWMutex
	segments         []*TraceSegment           // append-only buffer
	samplingRules    map[string]*SamplingRule   // keyed by rule name
	groups           map[string]*Group          // keyed by group name
	encryptionConfig EncryptionConfig
	tagsByARN        map[string]map[string]string
	accountID        string
	region           string
}

func NewStore(accountID, region string) *Store {
	s := &Store{
		segments:      make([]*TraceSegment, 0),
		samplingRules: make(map[string]*SamplingRule),
		groups:        make(map[string]*Group),
		tagsByARN:     make(map[string]map[string]string),
		accountID:     accountID,
		region:        region,
		encryptionConfig: EncryptionConfig{
			Status: "ACTIVE",
			Type:   "NONE",
		},
	}
	// Seed default sampling rule (AWS default)
	s.samplingRules["Default"] = &SamplingRule{
		RuleName:      "Default",
		RuleARN:       s.samplingRuleARN("Default"),
		Priority:      10000,
		FixedRate:     0.05,
		ReservoirSize: 5,
		ServiceName:   "*",
		ServiceType:   "*",
		Host:          "*",
		HTTPMethod:    "*",
		URLPath:       "*",
		Version:       1,
		CreatedAt:     time.Now().UTC(),
		ModifiedAt:    time.Now().UTC(),
		Tags:          make(map[string]string),
	}
	s.tagsByARN[s.samplingRuleARN("Default")] = make(map[string]string)
	return s
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) samplingRuleARN(name string) string {
	return fmt.Sprintf("arn:aws:xray:%s:%s:sampling-rule/%s", s.region, s.accountID, name)
}

func (s *Store) groupARN(name string) string {
	return fmt.Sprintf("arn:aws:xray:%s:%s:group//%s", s.region, s.accountID, name)
}

// ---- Trace Segments ----

func (s *Store) PutTraceSegments(docs []string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var unprocessed []string
	for _, doc := range docs {
		traceID := fmt.Sprintf("1-%08x-%s", time.Now().Unix(), newID()[:12])
		s.segments = append(s.segments, &TraceSegment{
			Document:  doc,
			TraceID:   traceID,
			Timestamp: time.Now().UTC(),
		})
	}
	return unprocessed
}

func (s *Store) GetTraceSummaries() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summaries := make([]map[string]any, 0, len(s.segments))
	for _, seg := range s.segments {
		summaries = append(summaries, map[string]any{
			"Id":        seg.TraceID,
			"Duration":  0.0,
			"HasError":  false,
			"HasFault":  false,
			"HasThrottle": false,
		})
	}
	return summaries
}

func (s *Store) BatchGetTraces(traceIDs []string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index := make(map[string]*TraceSegment)
	for _, seg := range s.segments {
		index[seg.TraceID] = seg
	}
	traces := make([]map[string]any, 0, len(traceIDs))
	for _, id := range traceIDs {
		if seg, ok := index[id]; ok {
			traces = append(traces, map[string]any{
				"Id":       seg.TraceID,
				"Segments": []map[string]any{{"Id": newID()[:16], "Document": seg.Document}},
			})
		}
	}
	return traces
}

// ---- Sampling Rules ----

func (s *Store) CreateSamplingRule(rule *SamplingRule) (*SamplingRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.samplingRules[rule.RuleName]; exists {
		return nil, service.NewAWSError("RuleAlreadyExistsException",
			fmt.Sprintf("Sampling rule %s already exists", rule.RuleName), http.StatusBadRequest)
	}
	rule.RuleARN = s.samplingRuleARN(rule.RuleName)
	rule.Version = 1
	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.ModifiedAt = now
	if rule.Tags == nil {
		rule.Tags = make(map[string]string)
	}
	s.samplingRules[rule.RuleName] = rule
	s.tagsByARN[rule.RuleARN] = rule.Tags
	return rule, nil
}

func (s *Store) UpdateSamplingRule(name string, fixedRate float64, reservoirSize int, priority int) (*SamplingRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.samplingRules[name]
	if !ok {
		return nil, service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Sampling rule %s not found", name), http.StatusBadRequest)
	}
	if fixedRate >= 0 {
		rule.FixedRate = fixedRate
	}
	if reservoirSize >= 0 {
		rule.ReservoirSize = reservoirSize
	}
	if priority > 0 {
		rule.Priority = priority
	}
	rule.ModifiedAt = time.Now().UTC()
	return rule, nil
}

func (s *Store) DeleteSamplingRule(name string) (*SamplingRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.samplingRules[name]
	if !ok {
		return nil, service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Sampling rule %s not found", name), http.StatusBadRequest)
	}
	if name == "Default" {
		return nil, service.NewAWSError("InvalidRequestException",
			"Cannot delete the default sampling rule", http.StatusBadRequest)
	}
	delete(s.samplingRules, name)
	delete(s.tagsByARN, rule.RuleARN)
	return rule, nil
}

func (s *Store) GetSamplingRules() []*SamplingRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rules := make([]*SamplingRule, 0, len(s.samplingRules))
	for _, r := range s.samplingRules {
		rules = append(rules, r)
	}
	return rules
}

// ---- Groups ----

func (s *Store) CreateGroup(name, filterExpr string, insightEnabled bool, tags map[string]string) (*Group, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.groups[name]; exists {
		return nil, service.NewAWSError("GroupAlreadyExistsException",
			fmt.Sprintf("Group %s already exists", name), http.StatusBadRequest)
	}
	now := time.Now().UTC()
	if tags == nil {
		tags = make(map[string]string)
	}
	g := &Group{
		GroupName:        name,
		GroupARN:         s.groupARN(name),
		FilterExpression: filterExpr,
		InsightEnabled:   insightEnabled,
		CreatedAt:        now,
		ModifiedAt:       now,
		Tags:             tags,
	}
	s.groups[name] = g
	s.tagsByARN[g.GroupARN] = tags
	return g, nil
}

func (s *Store) GetGroup(name string) (*Group, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Group %s not found", name), http.StatusBadRequest)
	}
	return g, nil
}

func (s *Store) GetGroups() []*Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		groups = append(groups, g)
	}
	return groups
}

func (s *Store) UpdateGroup(name, filterExpr string, insightEnabled *bool) (*Group, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Group %s not found", name), http.StatusBadRequest)
	}
	if filterExpr != "" {
		g.FilterExpression = filterExpr
	}
	if insightEnabled != nil {
		g.InsightEnabled = *insightEnabled
	}
	g.ModifiedAt = time.Now().UTC()
	return g, nil
}

func (s *Store) DeleteGroup(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Group %s not found", name), http.StatusBadRequest)
	}
	delete(s.groups, name)
	delete(s.tagsByARN, g.GroupARN)
	return nil
}

// ---- Encryption Config ----

func (s *Store) PutEncryptionConfig(keyID, encType string) EncryptionConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.encryptionConfig = EncryptionConfig{
		KeyID:  keyID,
		Status: "UPDATING",
		Type:   encType,
	}
	// immediately mark as active
	s.encryptionConfig.Status = "ACTIVE"
	return s.encryptionConfig
}

func (s *Store) GetEncryptionConfig() EncryptionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.encryptionConfig
}

// ---- Tags ----

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByARN[arn]
	if !ok {
		existing = make(map[string]string)
		s.tagsByARN[arn] = existing
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByARN[arn]
	if !ok {
		return service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusBadRequest)
	}
	for _, k := range keys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByARN[arn]
	if !ok {
		return nil, service.NewAWSError("InvalidRequestException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusBadRequest)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}
