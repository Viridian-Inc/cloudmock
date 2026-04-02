package route53resolver

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

// ResolverEndpoint represents a Route 53 Resolver endpoint.
type ResolverEndpoint struct {
	ID                string
	Arn               string
	Name              string
	Direction         string // INBOUND or OUTBOUND
	IPAddressCount    int
	IPAddresses       []string
	SecurityGroupIds  []string
	Status            string
	StatusMessage     string
	HostVPCId         string
	CreationTime      time.Time
	ModificationTime  time.Time
	lifecycle         *lifecycle.Machine
}

// ResolverRule represents a resolver rule.
type ResolverRule struct {
	ID                    string
	Arn                   string
	Name                  string
	DomainName            string
	RuleType              string // FORWARD, SYSTEM, RECURSIVE
	ResolverEndpointID    string
	Status                string
	StatusMessage         string
	TargetIPs             []TargetAddress
	CreationTime          time.Time
	ModificationTime      time.Time
}

// TargetAddress represents a target IP for a resolver rule.
type TargetAddress struct {
	IP   string
	Port int
}

// RuleAssociation represents an association between a rule and a VPC.
type RuleAssociation struct {
	ID              string
	Arn             string
	ResolverRuleID  string
	VPCId           string
	Name            string
	Status          string
	StatusMessage   string
}

// QueryLogConfig represents a query logging configuration.
type QueryLogConfig struct {
	ID               string
	Arn              string
	Name             string
	DestinationArn   string
	Status           string
	AssociationCount int
	CreationTime     time.Time
}

// QueryLogConfigAssociation represents an association between a query log config and a VPC.
type QueryLogConfigAssociation struct {
	ID         string
	ConfigID   string
	ResourceID string
	Status     string
}

// ResolverTag is a key-value tag for Route 53 Resolver resources.
type ResolverTag struct {
	Key   string
	Value string
}

// Store manages Route 53 Resolver resources in memory.
type Store struct {
	mu                  sync.RWMutex
	endpoints           map[string]*ResolverEndpoint
	rules               map[string]*ResolverRule
	associations        map[string]*RuleAssociation
	queryLogConfigs     map[string]*QueryLogConfig
	qlAssociations      map[string]*QueryLogConfigAssociation
	tags                map[string][]ResolverTag // keyed by resource ARN
	accountID           string
	region              string
	lcConfig            *lifecycle.Config
	epSeq               int
	ruleSeq             int
	assocSeq            int
	qlSeq               int
	qlAssocSeq          int
}

// NewStore returns a new empty Route 53 Resolver Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		endpoints:      make(map[string]*ResolverEndpoint),
		rules:          make(map[string]*ResolverRule),
		associations:   make(map[string]*RuleAssociation),
		queryLogConfigs: make(map[string]*QueryLogConfig),
		qlAssociations: make(map[string]*QueryLogConfigAssociation),
		tags:           make(map[string][]ResolverTag),
		accountID:      accountID,
		region:         region,
		lcConfig:       lifecycle.DefaultConfig(),
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:route53resolver:%s:%s:", s.region, s.accountID)
}

// CreateResolverEndpoint creates a new resolver endpoint.
func (s *Store) CreateResolverEndpoint(name, direction, vpcID string, securityGroupIDs []string, ipCount int) (*ResolverEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.epSeq++
	id := fmt.Sprintf("rslvr-in-%012d", s.epSeq)
	if direction == "OUTBOUND" {
		id = fmt.Sprintf("rslvr-out-%012d", s.epSeq)
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "OPERATIONAL", Delay: 3 * time.Second},
	}

	// Generate IPs in 10.x.x.x range
	if ipCount <= 0 {
		ipCount = 2
	}
	ipAddresses := make([]string, 0, ipCount)
	for i := 0; i < ipCount; i++ {
		ipAddresses = append(ipAddresses, fmt.Sprintf("10.0.%d.%d", s.epSeq, i+1))
	}

	now := time.Now().UTC()
	ep := &ResolverEndpoint{
		ID:               id,
		Arn:              s.arnPrefix() + "resolver-endpoint/" + id,
		Name:             name,
		Direction:        direction,
		IPAddressCount:   ipCount,
		IPAddresses:      ipAddresses,
		SecurityGroupIds: securityGroupIDs,
		Status:           "CREATING",
		StatusMessage:    "Creating",
		HostVPCId:        vpcID,
		CreationTime:     now,
		ModificationTime: now,
	}
	ep.lifecycle = lifecycle.NewMachine("CREATING", transitions, s.lcConfig)
	ep.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		ep.Status = string(to)
		ep.StatusMessage = string(to)
		ep.ModificationTime = time.Now().UTC()
	})

	s.endpoints[id] = ep
	return ep, nil
}

// GetResolverEndpoint retrieves an endpoint.
func (s *Store) GetResolverEndpoint(id string) (*ResolverEndpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.endpoints[id]
	return ep, ok
}

// ListResolverEndpoints returns all endpoints.
func (s *Store) ListResolverEndpoints() []*ResolverEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ResolverEndpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		out = append(out, ep)
	}
	return out
}

// DeleteResolverEndpoint removes an endpoint.
func (s *Store) DeleteResolverEndpoint(id string) (*ResolverEndpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[id]
	if !ok {
		return nil, false
	}
	if ep.lifecycle != nil {
		ep.lifecycle.Stop()
	}
	ep.Status = "DELETING"
	delete(s.endpoints, id)
	return ep, true
}

// CreateResolverRule creates a new resolver rule.
func (s *Store) CreateResolverRule(name, domainName, ruleType, endpointID string, targetIPs []TargetAddress) (*ResolverRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate target IPs are valid IPv4
	for _, t := range targetIPs {
		if t.IP != "" {
			ip := net.ParseIP(t.IP)
			if ip == nil || ip.To4() == nil {
				return nil, fmt.Errorf("invalid target IP address: %s. Must be a valid IPv4 address", t.IP)
			}
		}
	}

	s.ruleSeq++
	id := fmt.Sprintf("rslvr-rr-%012d", s.ruleSeq)
	now := time.Now().UTC()

	rule := &ResolverRule{
		ID:                 id,
		Arn:                s.arnPrefix() + "resolver-rule/" + id,
		Name:               name,
		DomainName:         domainName,
		RuleType:           ruleType,
		ResolverEndpointID: endpointID,
		Status:             "COMPLETE",
		StatusMessage:      "Creating",
		TargetIPs:          targetIPs,
		CreationTime:       now,
		ModificationTime:   now,
	}
	s.rules[id] = rule
	return rule, nil
}

// GetResolverRule retrieves a rule.
func (s *Store) GetResolverRule(id string) (*ResolverRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[id]
	return r, ok
}

// ListResolverRules returns all rules.
func (s *Store) ListResolverRules() []*ResolverRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ResolverRule, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, r)
	}
	return out
}

// DeleteResolverRule removes a rule.
func (s *Store) DeleteResolverRule(id string) (*ResolverRule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	delete(s.rules, id)
	return r, true
}

// AssociateResolverRule creates a rule-VPC association.
func (s *Store) AssociateResolverRule(ruleID, vpcID, name string) (*RuleAssociation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rules[ruleID]; !ok {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	s.assocSeq++
	id := fmt.Sprintf("rslvr-rrassoc-%012d", s.assocSeq)

	assoc := &RuleAssociation{
		ID:             id,
		Arn:            s.arnPrefix() + "resolver-rule-association/" + id,
		ResolverRuleID: ruleID,
		VPCId:          vpcID,
		Name:           name,
		Status:         "COMPLETE",
		StatusMessage:  "Associated",
	}
	s.associations[id] = assoc
	return assoc, nil
}

// GetResolverRuleAssociation retrieves an association.
func (s *Store) GetResolverRuleAssociation(id string) (*RuleAssociation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.associations[id]
	return a, ok
}

// ListResolverRuleAssociations returns all associations.
func (s *Store) ListResolverRuleAssociations() []*RuleAssociation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RuleAssociation, 0, len(s.associations))
	for _, a := range s.associations {
		out = append(out, a)
	}
	return out
}

// DisassociateResolverRule removes a rule association.
func (s *Store) DisassociateResolverRule(id string) (*RuleAssociation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.associations[id]
	if !ok {
		return nil, false
	}
	delete(s.associations, id)
	return a, true
}

// CreateQueryLogConfig creates a query log configuration.
func (s *Store) CreateQueryLogConfig(name, destinationArn string) (*QueryLogConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.qlSeq++
	id := fmt.Sprintf("rslvr-qlc-%012d", s.qlSeq)

	config := &QueryLogConfig{
		ID:             id,
		Arn:            s.arnPrefix() + "resolver-query-log-config/" + id,
		Name:           name,
		DestinationArn: destinationArn,
		Status:         "CREATED",
		CreationTime:   time.Now().UTC(),
	}
	s.queryLogConfigs[id] = config
	return config, nil
}

// GetQueryLogConfig retrieves a query log config.
func (s *Store) GetQueryLogConfig(id string) (*QueryLogConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.queryLogConfigs[id]
	return c, ok
}

// ListQueryLogConfigs returns all query log configs.
func (s *Store) ListQueryLogConfigs() []*QueryLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*QueryLogConfig, 0, len(s.queryLogConfigs))
	for _, c := range s.queryLogConfigs {
		out = append(out, c)
	}
	return out
}

// DeleteQueryLogConfig removes a query log config.
func (s *Store) DeleteQueryLogConfig(id string) (*QueryLogConfig, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.queryLogConfigs[id]
	if !ok {
		return nil, false
	}
	delete(s.queryLogConfigs, id)
	return c, true
}

// UpdateResolverEndpoint updates the name of a resolver endpoint.
func (s *Store) UpdateResolverEndpoint(id, name string) (*ResolverEndpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.endpoints[id]
	if !ok {
		return nil, false
	}
	if name != "" {
		ep.Name = name
	}
	ep.ModificationTime = time.Now().UTC()
	return ep, true
}

// UpdateResolverRule updates a resolver rule from a config map.
func (s *Store) UpdateResolverRule(id string, config map[string]any) (*ResolverRule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rules[id]
	if !ok {
		return nil, false
	}
	if config != nil {
		if name, ok := config["Name"].(string); ok && name != "" {
			r.Name = name
		}
		if endpointID, ok := config["ResolverEndpointId"].(string); ok && endpointID != "" {
			r.ResolverEndpointID = endpointID
		}
		if targetIPs, ok := config["TargetIps"].([]any); ok {
			newTargets := make([]TargetAddress, 0, len(targetIPs))
			for _, t := range targetIPs {
				if tm, ok := t.(map[string]any); ok {
					ip, _ := tm["Ip"].(string)
					port := 53
					if pv, ok := tm["Port"].(float64); ok {
						port = int(pv)
					}
					newTargets = append(newTargets, TargetAddress{IP: ip, Port: port})
				}
			}
			r.TargetIPs = newTargets
		}
	}
	r.ModificationTime = time.Now().UTC()
	return r, true
}

// AssociateQueryLogConfig associates a query log config with a resource (VPC).
func (s *Store) AssociateQueryLogConfig(configID, resourceID string) (*QueryLogConfigAssociation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queryLogConfigs[configID]; !ok {
		return nil, fmt.Errorf("query log config not found: %s", configID)
	}
	s.qlAssocSeq++
	id := fmt.Sprintf("rslvr-qlca-%012d", s.qlAssocSeq)
	assoc := &QueryLogConfigAssociation{
		ID:         id,
		ConfigID:   configID,
		ResourceID: resourceID,
		Status:     "ACTIVE",
	}
	s.qlAssociations[id] = assoc
	s.queryLogConfigs[configID].AssociationCount++
	return assoc, nil
}

// DisassociateQueryLogConfig removes a query log config association.
func (s *Store) DisassociateQueryLogConfig(assocID string) (*QueryLogConfigAssociation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	assoc, ok := s.qlAssociations[assocID]
	if !ok {
		return nil, false
	}
	if cfg, ok := s.queryLogConfigs[assoc.ConfigID]; ok {
		if cfg.AssociationCount > 0 {
			cfg.AssociationCount--
		}
	}
	delete(s.qlAssociations, assocID)
	return assoc, true
}

// TagResource adds or updates tags on a resource by ARN.
func (s *Store) TagResource(resourceArn string, tags []ResolverTag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.tags[resourceArn]
	for _, nt := range tags {
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
	s.tags[resourceArn] = existing
	return nil
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(resourceArn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	keySet := make(map[string]bool, len(tagKeys))
	for _, k := range tagKeys {
		keySet[k] = true
	}
	existing := s.tags[resourceArn]
	filtered := existing[:0]
	for _, t := range existing {
		if !keySet[t.Key] {
			filtered = append(filtered, t)
		}
	}
	s.tags[resourceArn] = filtered
	return nil
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(resourceArn string) ([]ResolverTag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tags := s.tags[resourceArn]
	if tags == nil {
		return []ResolverTag{}, nil
	}
	return tags, nil
}
