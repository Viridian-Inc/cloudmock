package shield

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// DRTAccess holds DRT (DDoS Response Team) access information.
type DRTAccess struct {
	RoleArn       string
	LogBucketList []string
}

// AttackStatistics holds aggregated attack statistics.
type AttackStatistics struct {
	FromInclusive time.Time
	ToExclusive   time.Time
	DataItems     []map[string]any
}

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// Protection holds a Shield protection.
type Protection struct {
	Id                string
	Name              string
	ResourceArn       string
	ProtectionArn     string
	HealthCheckIds    []string
	ApplicationLayerAutoResponseConfiguration map[string]any
	Tags              []Tag
}

// Subscription holds a Shield Advanced subscription.
type Subscription struct {
	StartTime         time.Time
	EndTime           time.Time
	TimeCommitmentInSeconds int64
	AutoRenew         string
	Limits            []Limit
	ProactiveEngagementStatus string
	SubscriptionState string
	SubscriptionArn   string
}

// Limit holds a Shield limit.
type Limit struct {
	Type string
	Max  int64
}

// Attack holds attack detail information.
type Attack struct {
	AttackId          string
	ResourceArn       string
	StartTime         time.Time
	EndTime           *time.Time
	AttackVectors     []AttackVector
	AttackCounters    []SummarizedCounter
	Mitigations       []Mitigation
}

// AttackVector describes a type of attack.
type AttackVector struct {
	VectorType string
}

// SummarizedCounter holds attack counter data.
type SummarizedCounter struct {
	Name    string
	Max     float64
	Average float64
	Sum     float64
	N       int
	Unit    string
}

// Mitigation describes a mitigation action.
type Mitigation struct {
	MitigationName string
}

// ProtectionGroup holds a protection group definition.
type ProtectionGroup struct {
	ProtectionGroupId  string
	Aggregation        string
	Pattern            string
	ResourceType       string
	Members            []string
	ProtectionGroupArn string
	Tags               []Tag
}

// Store is the in-memory store for Shield resources.
type Store struct {
	mu               sync.RWMutex
	protections      map[string]*Protection
	subscription     *Subscription
	attacks          map[string]*Attack
	protectionGroups map[string]*ProtectionGroup
	accountID        string
	region           string
}

// NewStore creates an empty Shield Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		protections:      make(map[string]*Protection),
		attacks:          make(map[string]*Attack),
		protectionGroups: make(map[string]*ProtectionGroup),
		accountID:        accountID,
		region:           region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) buildProtectionArn(id string) string {
	return fmt.Sprintf("arn:aws:shield::%s:protection/%s", s.accountID, id)
}

func (s *Store) buildProtectionGroupArn(id string) string {
	return fmt.Sprintf("arn:aws:shield::%s:protection-group/%s", s.accountID, id)
}

func (s *Store) buildSubscriptionArn() string {
	return fmt.Sprintf("arn:aws:shield::%s:subscription", s.accountID)
}

// CreateProtection creates a new protection.
func (s *Store) CreateProtection(name, resourceArn string, tags []Tag) (*Protection, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Name is required.")
	}
	if resourceArn == "" {
		return nil, service.ErrValidation("ResourceArn is required.")
	}

	// Check for duplicate resource
	for _, p := range s.protections {
		if p.ResourceArn == resourceArn {
			return nil, service.NewAWSError("ResourceAlreadyExistsException",
				"Resource is already protected.", http.StatusConflict)
		}
	}

	id := newUUID()
	protection := &Protection{
		Id:            id,
		Name:          name,
		ResourceArn:   resourceArn,
		ProtectionArn: s.buildProtectionArn(id),
		Tags:          tags,
	}
	s.protections[id] = protection
	return protection, nil
}

// GetProtection returns a protection by ID.
func (s *Store) GetProtection(id string) (*Protection, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.protections[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection %s not found.", id), http.StatusNotFound)
	}
	return p, nil
}

// ListProtections returns all protections.
func (s *Store) ListProtections() []*Protection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Protection, 0, len(s.protections))
	for _, p := range s.protections {
		out = append(out, p)
	}
	return out
}

// DeleteProtection removes a protection.
func (s *Store) DeleteProtection(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.protections[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection %s not found.", id), http.StatusNotFound)
	}
	delete(s.protections, id)
	return nil
}

// CreateSubscription creates a Shield Advanced subscription.
func (s *Store) CreateSubscription() (*Subscription, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subscription != nil {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			"Subscription already exists.", http.StatusConflict)
	}
	now := time.Now().UTC()
	s.subscription = &Subscription{
		StartTime:                 now,
		EndTime:                   now.AddDate(1, 0, 0),
		TimeCommitmentInSeconds:   31536000, // 1 year
		AutoRenew:                 "ENABLED",
		ProactiveEngagementStatus: "DISABLED",
		SubscriptionState:         "ACTIVE",
		SubscriptionArn:           s.buildSubscriptionArn(),
		Limits: []Limit{
			{Type: "PROTECTION", Max: 1000},
			{Type: "PROTECTION_GROUP", Max: 5000},
		},
	}
	return s.subscription, nil
}

// GetSubscription returns the subscription.
func (s *Store) GetSubscription() (*Subscription, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.subscription == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"No Shield Advanced subscription found.", http.StatusNotFound)
	}
	return s.subscription, nil
}

// GetAttack returns an attack by ID.
func (s *Store) GetAttack(attackId string) (*Attack, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.attacks[attackId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Attack %s not found.", attackId), http.StatusNotFound)
	}
	return a, nil
}

// ListAttacks returns all attacks.
func (s *Store) ListAttacks() []*Attack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Attack, 0, len(s.attacks))
	for _, a := range s.attacks {
		out = append(out, a)
	}
	return out
}

// CreateProtectionGroup creates a new protection group.
func (s *Store) CreateProtectionGroup(id, aggregation, pattern, resourceType string, members []string, tags []Tag) (*ProtectionGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.protectionGroups[id]; ok {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Protection group %s already exists.", id), http.StatusConflict)
	}
	pg := &ProtectionGroup{
		ProtectionGroupId:  id,
		Aggregation:        aggregation,
		Pattern:            pattern,
		ResourceType:       resourceType,
		Members:            members,
		ProtectionGroupArn: s.buildProtectionGroupArn(id),
		Tags:               tags,
	}
	s.protectionGroups[id] = pg
	return pg, nil
}

// GetProtectionGroup returns a protection group.
func (s *Store) GetProtectionGroup(id string) (*ProtectionGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pg, ok := s.protectionGroups[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection group %s not found.", id), http.StatusNotFound)
	}
	return pg, nil
}

// ListProtectionGroups returns all protection groups.
func (s *Store) ListProtectionGroups() []*ProtectionGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ProtectionGroup, 0, len(s.protectionGroups))
	for _, pg := range s.protectionGroups {
		out = append(out, pg)
	}
	return out
}

// UpdateProtectionGroup updates a protection group.
func (s *Store) UpdateProtectionGroup(id, aggregation, pattern, resourceType string, members []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	pg, ok := s.protectionGroups[id]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection group %s not found.", id), http.StatusNotFound)
	}
	if aggregation != "" {
		pg.Aggregation = aggregation
	}
	if pattern != "" {
		pg.Pattern = pattern
	}
	if resourceType != "" {
		pg.ResourceType = resourceType
	}
	if members != nil {
		pg.Members = members
	}
	return nil
}

// DeleteProtectionGroup removes a protection group.
func (s *Store) DeleteProtectionGroup(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.protectionGroups[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection group %s not found.", id), http.StatusNotFound)
	}
	delete(s.protectionGroups, id)
	return nil
}

// TagResource adds tags to a resource.
func (s *Store) TagResource(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.protections {
		if p.ProtectionArn == arn {
			p.Tags = mergeTags(p.Tags, tags)
			return nil
		}
	}
	for _, pg := range s.protectionGroups {
		if pg.ProtectionGroupArn == arn {
			pg.Tags = mergeTags(pg.Tags, tags)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Resource not found.", http.StatusNotFound)
}

// UntagResource removes tags from a resource.
func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	keySet := make(map[string]bool, len(tagKeys))
	for _, k := range tagKeys {
		keySet[k] = true
	}
	for _, p := range s.protections {
		if p.ProtectionArn == arn {
			p.Tags = filterTags(p.Tags, keySet)
			return nil
		}
	}
	for _, pg := range s.protectionGroups {
		if pg.ProtectionGroupArn == arn {
			pg.Tags = filterTags(pg.Tags, keySet)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Resource not found.", http.StatusNotFound)
}

// ListTagsForResource returns tags for a resource.
func (s *Store) ListTagsForResource(arn string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.protections {
		if p.ProtectionArn == arn {
			return p.Tags, nil
		}
	}
	for _, pg := range s.protectionGroups {
		if pg.ProtectionGroupArn == arn {
			return pg.Tags, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		"Resource not found.", http.StatusNotFound)
}

func mergeTags(existing, newTags []Tag) []Tag {
	for _, nt := range newTags {
		found := false
		for i, et := range existing {
			if et.Key == nt.Key {
				existing[i].Value = nt.Value
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, nt)
		}
	}
	return existing
}

func filterTags(tags []Tag, removeKeys map[string]bool) []Tag {
	out := make([]Tag, 0, len(tags))
	for _, t := range tags {
		if !removeKeys[t.Key] {
			out = append(out, t)
		}
	}
	return out
}

// UpdateSubscription updates the subscription auto-renew setting.
func (s *Store) UpdateSubscription(autoRenew string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subscription == nil {
		return service.NewAWSError("ResourceNotFoundException",
			"No Shield Advanced subscription found.", http.StatusNotFound)
	}
	if autoRenew != "" {
		s.subscription.AutoRenew = autoRenew
	}
	return nil
}

// DescribeAttackStatistics returns attack statistics (mock data).
func (s *Store) DescribeAttackStatistics() AttackStatistics {
	now := time.Now().UTC()
	return AttackStatistics{
		FromInclusive: now.AddDate(0, 0, -30),
		ToExclusive:   now,
		DataItems:     []map[string]any{},
	}
}

// DescribeDRTAccess returns the DRT access configuration.
func (s *Store) DescribeDRTAccess() DRTAccess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return DRTAccess{
		RoleArn:       "",
		LogBucketList: []string{},
	}
}

// EnableApplicationLayerAutomaticResponse enables automatic response for a resource.
func (s *Store) EnableApplicationLayerAutomaticResponse(resourceArn string, action map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if resourceArn == "" {
		return service.ErrValidation("ResourceArn is required.")
	}
	for _, p := range s.protections {
		if p.ResourceArn == resourceArn {
			p.ApplicationLayerAutoResponseConfiguration = map[string]any{
				"Action": action,
				"Status": "ENABLED",
			}
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("No protection found for resource %s.", resourceArn), http.StatusNotFound)
}

// DisableApplicationLayerAutomaticResponse disables automatic response for a resource.
func (s *Store) DisableApplicationLayerAutomaticResponse(resourceArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if resourceArn == "" {
		return service.ErrValidation("ResourceArn is required.")
	}
	for _, p := range s.protections {
		if p.ResourceArn == resourceArn {
			p.ApplicationLayerAutoResponseConfiguration = map[string]any{
				"Status": "DISABLED",
			}
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("No protection found for resource %s.", resourceArn), http.StatusNotFound)
}

// AssociateHealthCheck associates a Route 53 health check with a protection.
func (s *Store) AssociateHealthCheck(protectionID, healthCheckArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if protectionID == "" {
		return service.ErrValidation("ProtectionId is required.")
	}
	if healthCheckArn == "" {
		return service.ErrValidation("HealthCheckArn is required.")
	}
	p, ok := s.protections[protectionID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection %s not found.", protectionID), http.StatusNotFound)
	}
	for _, hc := range p.HealthCheckIds {
		if hc == healthCheckArn {
			return service.NewAWSError("ResourceAlreadyExistsException",
				"Health check already associated.", http.StatusConflict)
		}
	}
	p.HealthCheckIds = append(p.HealthCheckIds, healthCheckArn)
	return nil
}

// DisassociateHealthCheck removes a Route 53 health check from a protection.
func (s *Store) DisassociateHealthCheck(protectionID, healthCheckArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if protectionID == "" {
		return service.ErrValidation("ProtectionId is required.")
	}
	p, ok := s.protections[protectionID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Protection %s not found.", protectionID), http.StatusNotFound)
	}
	for i, hc := range p.HealthCheckIds {
		if hc == healthCheckArn {
			p.HealthCheckIds = append(p.HealthCheckIds[:i], p.HealthCheckIds[i+1:]...)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Health check not associated.", http.StatusNotFound)
}
