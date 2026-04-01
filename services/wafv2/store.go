package wafv2

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// WebACL holds a WAFv2 Web ACL.
type WebACL struct {
	Name                string
	Id                  string
	ARN                 string
	Description         string
	Scope               string // REGIONAL or CLOUDFRONT
	DefaultAction       map[string]any
	Rules               []map[string]any
	VisibilityConfig    map[string]any
	Capacity            int64
	LockToken           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Tags                []Tag
	AssociatedResources []string
	LoggingConfig       map[string]any
}

// RuleGroup holds a WAFv2 rule group.
type RuleGroup struct {
	Name             string
	Id               string
	ARN              string
	Description      string
	Scope            string
	Capacity         int64
	Rules            []map[string]any
	VisibilityConfig map[string]any
	LockToken        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Tags             []Tag
}

// IPSet holds a WAFv2 IP set.
type IPSet struct {
	Name             string
	Id               string
	ARN              string
	Description      string
	Scope            string
	IPAddressVersion string
	Addresses        []string
	LockToken        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Tags             []Tag
}

// RegexPatternSet holds a WAFv2 regex pattern set.
type RegexPatternSet struct {
	Name                  string
	Id                    string
	ARN                   string
	Description           string
	Scope                 string
	RegularExpressionList []string
	LockToken             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Tags                  []Tag
}

// Store is the in-memory store for WAFv2 resources.
type Store struct {
	mu               sync.RWMutex
	webACLs          map[string]*WebACL          // keyed by ID
	ruleGroups       map[string]*RuleGroup       // keyed by ID
	ipSets           map[string]*IPSet           // keyed by ID
	regexPatternSets map[string]*RegexPatternSet // keyed by ID
	sampledRequests  map[string][]SampledRequest  // keyed by WebACL ID
	rateCounter      *RateCounter
	accountID        string
	region           string
}

// NewStore creates an empty WAFv2 Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		webACLs:          make(map[string]*WebACL),
		ruleGroups:       make(map[string]*RuleGroup),
		ipSets:           make(map[string]*IPSet),
		regexPatternSets: make(map[string]*RegexPatternSet),
		sampledRequests:  make(map[string][]SampledRequest),
		rateCounter:      NewRateCounter(),
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

func newLockToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) buildWebACLArn(scope, name, id string) string {
	scopePath := "regional"
	if scope == "CLOUDFRONT" {
		scopePath = "global"
	}
	return fmt.Sprintf("arn:aws:wafv2:%s:%s:%s/webacl/%s/%s", s.region, s.accountID, scopePath, name, id)
}

func (s *Store) buildRuleGroupArn(scope, name, id string) string {
	scopePath := "regional"
	if scope == "CLOUDFRONT" {
		scopePath = "global"
	}
	return fmt.Sprintf("arn:aws:wafv2:%s:%s:%s/rulegroup/%s/%s", s.region, s.accountID, scopePath, name, id)
}

func (s *Store) buildIPSetArn(scope, name, id string) string {
	scopePath := "regional"
	if scope == "CLOUDFRONT" {
		scopePath = "global"
	}
	return fmt.Sprintf("arn:aws:wafv2:%s:%s:%s/ipset/%s/%s", s.region, s.accountID, scopePath, name, id)
}

func (s *Store) buildRegexPatternSetArn(scope, name, id string) string {
	scopePath := "regional"
	if scope == "CLOUDFRONT" {
		scopePath = "global"
	}
	return fmt.Sprintf("arn:aws:wafv2:%s:%s:%s/regexpatternset/%s/%s", s.region, s.accountID, scopePath, name, id)
}

// CreateWebACL creates a new Web ACL.
func (s *Store) CreateWebACL(name, description, scope string, defaultAction map[string]any, rules []map[string]any, visibilityConfig map[string]any, tags []Tag) (*WebACL, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate name in scope
	for _, acl := range s.webACLs {
		if acl.Name == name && acl.Scope == scope {
			return nil, service.NewAWSError("WAFDuplicateItemException",
				fmt.Sprintf("Web ACL with name %s already exists.", name), http.StatusConflict)
		}
	}

	id := newUUID()
	now := time.Now().UTC()
	acl := &WebACL{
		Name:             name,
		Id:               id,
		ARN:              s.buildWebACLArn(scope, name, id),
		Description:      description,
		Scope:            scope,
		DefaultAction:    defaultAction,
		Rules:            rules,
		VisibilityConfig: visibilityConfig,
		Capacity:         100,
		LockToken:        newLockToken(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Tags:             tags,
	}
	s.webACLs[id] = acl
	return acl, nil
}

// GetWebACL returns a Web ACL by name, scope, and ID.
func (s *Store) GetWebACL(name, scope, id string) (*WebACL, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acl, ok := s.webACLs[id]
	if !ok || acl.Name != name || acl.Scope != scope {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Web ACL not found.", http.StatusBadRequest)
	}
	return acl, nil
}

// GetWebACLByID returns a Web ACL by ID only.
func (s *Store) GetWebACLByID(id string) (*WebACL, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acl, ok := s.webACLs[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Web ACL not found.", http.StatusBadRequest)
	}
	return acl, nil
}

// ListWebACLs returns all Web ACLs for a scope.
func (s *Store) ListWebACLs(scope string) []*WebACL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*WebACL, 0)
	for _, acl := range s.webACLs {
		if scope == "" || acl.Scope == scope {
			out = append(out, acl)
		}
	}
	return out
}

// UpdateWebACL updates an existing Web ACL.
func (s *Store) UpdateWebACL(name, scope, id, lockToken string, defaultAction map[string]any, rules []map[string]any, visibilityConfig map[string]any, description string) (*WebACL, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acl, ok := s.webACLs[id]
	if !ok || acl.Name != name || acl.Scope != scope {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Web ACL not found.", http.StatusBadRequest)
	}
	if acl.LockToken != lockToken {
		return nil, service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	if defaultAction != nil {
		acl.DefaultAction = defaultAction
	}
	if rules != nil {
		acl.Rules = rules
	}
	if visibilityConfig != nil {
		acl.VisibilityConfig = visibilityConfig
	}
	if description != "" {
		acl.Description = description
	}
	acl.UpdatedAt = time.Now().UTC()
	acl.LockToken = newLockToken()
	return acl, nil
}

// DeleteWebACL removes a Web ACL.
func (s *Store) DeleteWebACL(name, scope, id, lockToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	acl, ok := s.webACLs[id]
	if !ok || acl.Name != name || acl.Scope != scope {
		return service.NewAWSError("WAFNonexistentItemException",
			"Web ACL not found.", http.StatusBadRequest)
	}
	if acl.LockToken != lockToken {
		return service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	if len(acl.AssociatedResources) > 0 {
		return service.NewAWSError("WAFAssociatedItemException",
			"Web ACL is still associated with resources.", http.StatusBadRequest)
	}
	delete(s.webACLs, id)
	return nil
}

// CreateRuleGroup creates a new rule group.
func (s *Store) CreateRuleGroup(name, description, scope string, capacity int64, rules []map[string]any, visibilityConfig map[string]any, tags []Tag) (*RuleGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	now := time.Now().UTC()
	rg := &RuleGroup{
		Name:             name,
		Id:               id,
		ARN:              s.buildRuleGroupArn(scope, name, id),
		Description:      description,
		Scope:            scope,
		Capacity:         capacity,
		Rules:            rules,
		VisibilityConfig: visibilityConfig,
		LockToken:        newLockToken(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Tags:             tags,
	}
	s.ruleGroups[id] = rg
	return rg, nil
}

// GetRuleGroup returns a rule group.
func (s *Store) GetRuleGroup(name, scope, id string) (*RuleGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rg, ok := s.ruleGroups[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Rule group not found.", http.StatusBadRequest)
	}
	return rg, nil
}

// ListRuleGroups returns all rule groups for a scope.
func (s *Store) ListRuleGroups(scope string) []*RuleGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RuleGroup, 0)
	for _, rg := range s.ruleGroups {
		if scope == "" || rg.Scope == scope {
			out = append(out, rg)
		}
	}
	return out
}

// UpdateRuleGroup updates a rule group.
func (s *Store) UpdateRuleGroup(name, scope, id, lockToken string, rules []map[string]any, visibilityConfig map[string]any, description string) (*RuleGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rg, ok := s.ruleGroups[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Rule group not found.", http.StatusBadRequest)
	}
	if rg.LockToken != lockToken {
		return nil, service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	if rules != nil {
		rg.Rules = rules
	}
	if visibilityConfig != nil {
		rg.VisibilityConfig = visibilityConfig
	}
	if description != "" {
		rg.Description = description
	}
	rg.UpdatedAt = time.Now().UTC()
	rg.LockToken = newLockToken()
	return rg, nil
}

// DeleteRuleGroup removes a rule group.
func (s *Store) DeleteRuleGroup(name, scope, id, lockToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	rg, ok := s.ruleGroups[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"Rule group not found.", http.StatusBadRequest)
	}
	if rg.LockToken != lockToken {
		return service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	delete(s.ruleGroups, id)
	return nil
}

// CreateIPSet creates a new IP set.
func (s *Store) CreateIPSet(name, description, scope, ipVersion string, addresses []string, tags []Tag) (*IPSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	now := time.Now().UTC()
	ipSet := &IPSet{
		Name:             name,
		Id:               id,
		ARN:              s.buildIPSetArn(scope, name, id),
		Description:      description,
		Scope:            scope,
		IPAddressVersion: ipVersion,
		Addresses:        addresses,
		LockToken:        newLockToken(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Tags:             tags,
	}
	s.ipSets[id] = ipSet
	return ipSet, nil
}

// GetIPSet returns an IP set.
func (s *Store) GetIPSet(name, scope, id string) (*IPSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ipSet, ok := s.ipSets[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"IP set not found.", http.StatusBadRequest)
	}
	return ipSet, nil
}

// ListIPSets returns all IP sets for a scope.
func (s *Store) ListIPSets(scope string) []*IPSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*IPSet, 0)
	for _, ipSet := range s.ipSets {
		if scope == "" || ipSet.Scope == scope {
			out = append(out, ipSet)
		}
	}
	return out
}

// UpdateIPSet updates an IP set.
func (s *Store) UpdateIPSet(name, scope, id, lockToken string, addresses []string, description string) (*IPSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ipSet, ok := s.ipSets[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"IP set not found.", http.StatusBadRequest)
	}
	if ipSet.LockToken != lockToken {
		return nil, service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	if addresses != nil {
		ipSet.Addresses = addresses
	}
	if description != "" {
		ipSet.Description = description
	}
	ipSet.UpdatedAt = time.Now().UTC()
	ipSet.LockToken = newLockToken()
	return ipSet, nil
}

// DeleteIPSet removes an IP set.
func (s *Store) DeleteIPSet(name, scope, id, lockToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ipSet, ok := s.ipSets[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"IP set not found.", http.StatusBadRequest)
	}
	if ipSet.LockToken != lockToken {
		return service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	delete(s.ipSets, id)
	return nil
}

// CreateRegexPatternSet creates a new regex pattern set.
func (s *Store) CreateRegexPatternSet(name, description, scope string, patterns []string, tags []Tag) (*RegexPatternSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	now := time.Now().UTC()
	rps := &RegexPatternSet{
		Name:                  name,
		Id:                    id,
		ARN:                   s.buildRegexPatternSetArn(scope, name, id),
		Description:           description,
		Scope:                 scope,
		RegularExpressionList: patterns,
		LockToken:             newLockToken(),
		CreatedAt:             now,
		UpdatedAt:             now,
		Tags:                  tags,
	}
	s.regexPatternSets[id] = rps
	return rps, nil
}

// GetRegexPatternSet returns a regex pattern set.
func (s *Store) GetRegexPatternSet(name, scope, id string) (*RegexPatternSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rps, ok := s.regexPatternSets[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			"Regex pattern set not found.", http.StatusBadRequest)
	}
	return rps, nil
}

// ListRegexPatternSets returns all regex pattern sets for a scope.
func (s *Store) ListRegexPatternSets(scope string) []*RegexPatternSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RegexPatternSet, 0)
	for _, rps := range s.regexPatternSets {
		if scope == "" || rps.Scope == scope {
			out = append(out, rps)
		}
	}
	return out
}

// DeleteRegexPatternSet removes a regex pattern set.
func (s *Store) DeleteRegexPatternSet(name, scope, id, lockToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	rps, ok := s.regexPatternSets[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"Regex pattern set not found.", http.StatusBadRequest)
	}
	if rps.LockToken != lockToken {
		return service.NewAWSError("WAFOptimisticLockException",
			"Lock token mismatch.", http.StatusBadRequest)
	}
	delete(s.regexPatternSets, id)
	return nil
}

// AssociateWebACL associates a Web ACL with a resource.
func (s *Store) AssociateWebACL(webACLArn, resourceArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acl := range s.webACLs {
		if acl.ARN == webACLArn {
			acl.AssociatedResources = append(acl.AssociatedResources, resourceArn)
			return nil
		}
	}
	return service.NewAWSError("WAFNonexistentItemException",
		"Web ACL not found.", http.StatusBadRequest)
}

// DisassociateWebACL removes a resource association.
func (s *Store) DisassociateWebACL(resourceArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acl := range s.webACLs {
		for i, res := range acl.AssociatedResources {
			if res == resourceArn {
				acl.AssociatedResources = append(acl.AssociatedResources[:i], acl.AssociatedResources[i+1:]...)
				return nil
			}
		}
	}
	return nil // Not an error if not associated
}

// GetWebACLForResource returns the Web ACL associated with a resource.
func (s *Store) GetWebACLForResource(resourceArn string) (*WebACL, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, acl := range s.webACLs {
		for _, res := range acl.AssociatedResources {
			if res == resourceArn {
				return acl, nil
			}
		}
	}
	return nil, service.NewAWSError("WAFNonexistentItemException",
		"No Web ACL associated with this resource.", http.StatusBadRequest)
}

// SetLoggingConfig sets logging configuration for a Web ACL.
func (s *Store) SetLoggingConfig(webACLArn string, config map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acl := range s.webACLs {
		if acl.ARN == webACLArn {
			acl.LoggingConfig = config
			return nil
		}
	}
	return service.NewAWSError("WAFNonexistentItemException",
		"Web ACL not found.", http.StatusBadRequest)
}

// GetLoggingConfig returns logging configuration for a Web ACL.
func (s *Store) GetLoggingConfig(webACLArn string) (map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, acl := range s.webACLs {
		if acl.ARN == webACLArn {
			if acl.LoggingConfig == nil {
				return nil, service.NewAWSError("WAFNonexistentItemException",
					"No logging configuration found.", http.StatusBadRequest)
			}
			return acl.LoggingConfig, nil
		}
	}
	return nil, service.NewAWSError("WAFNonexistentItemException",
		"Web ACL not found.", http.StatusBadRequest)
}

// DeleteLoggingConfig removes logging configuration.
func (s *Store) DeleteLoggingConfig(webACLArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acl := range s.webACLs {
		if acl.ARN == webACLArn {
			acl.LoggingConfig = nil
			return nil
		}
	}
	return service.NewAWSError("WAFNonexistentItemException",
		"Web ACL not found.", http.StatusBadRequest)
}

// TagResource adds tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acl := range s.webACLs {
		if acl.ARN == arn {
			acl.Tags = mergeTags(acl.Tags, tags)
			return nil
		}
	}
	for _, rg := range s.ruleGroups {
		if rg.ARN == arn {
			rg.Tags = mergeTags(rg.Tags, tags)
			return nil
		}
	}
	for _, ipSet := range s.ipSets {
		if ipSet.ARN == arn {
			ipSet.Tags = mergeTags(ipSet.Tags, tags)
			return nil
		}
	}
	for _, rps := range s.regexPatternSets {
		if rps.ARN == arn {
			rps.Tags = mergeTags(rps.Tags, tags)
			return nil
		}
	}
	return service.NewAWSError("WAFNonexistentItemException",
		"Resource not found.", http.StatusBadRequest)
}

// UntagResource removes tags from a resource.
func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	keySet := make(map[string]bool, len(tagKeys))
	for _, k := range tagKeys {
		keySet[k] = true
	}
	for _, acl := range s.webACLs {
		if acl.ARN == arn {
			acl.Tags = filterTags(acl.Tags, keySet)
			return nil
		}
	}
	for _, rg := range s.ruleGroups {
		if rg.ARN == arn {
			rg.Tags = filterTags(rg.Tags, keySet)
			return nil
		}
	}
	for _, ipSet := range s.ipSets {
		if ipSet.ARN == arn {
			ipSet.Tags = filterTags(ipSet.Tags, keySet)
			return nil
		}
	}
	for _, rps := range s.regexPatternSets {
		if rps.ARN == arn {
			rps.Tags = filterTags(rps.Tags, keySet)
			return nil
		}
	}
	return service.NewAWSError("WAFNonexistentItemException",
		"Resource not found.", http.StatusBadRequest)
}

// ListTagsForResource returns tags for a resource.
func (s *Store) ListTagsForResource(arn string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, acl := range s.webACLs {
		if acl.ARN == arn {
			return acl.Tags, nil
		}
	}
	for _, rg := range s.ruleGroups {
		if rg.ARN == arn {
			return rg.Tags, nil
		}
	}
	for _, ipSet := range s.ipSets {
		if ipSet.ARN == arn {
			return ipSet.Tags, nil
		}
	}
	for _, rps := range s.regexPatternSets {
		if rps.ARN == arn {
			return rps.Tags, nil
		}
	}
	return nil, service.NewAWSError("WAFNonexistentItemException",
		"Resource not found.", http.StatusBadRequest)
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
