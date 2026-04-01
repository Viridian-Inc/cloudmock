package wafregional

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/neureaux/cloudmock/pkg/service"
)

// WebACL holds a legacy WAF Web ACL.
type WebACL struct {
	WebACLId    string
	Name        string
	MetricName  string
	DefaultAction string // ALLOW, BLOCK, COUNT
	Rules       []ActivatedRule
	WebACLArn   string
	ChangeToken string
}

// ActivatedRule holds a rule activation in a Web ACL.
type ActivatedRule struct {
	Priority   int
	RuleId     string
	Action     string
	Type       string
	OverrideAction string
}

// Rule holds a legacy WAF rule.
type Rule struct {
	RuleId     string
	Name       string
	MetricName string
	Predicates []Predicate
}

// Predicate connects a rule to a condition set.
type Predicate struct {
	Negated bool
	Type    string
	DataId  string
}

// IPSet holds a legacy WAF IP set.
type IPSet struct {
	IPSetId        string
	Name           string
	IPSetDescriptors []IPSetDescriptor
}

// IPSetDescriptor holds an IP address/CIDR.
type IPSetDescriptor struct {
	Type  string // IPV4 or IPV6
	Value string
}

// ByteMatchSet holds a byte match set.
type ByteMatchSet struct {
	ByteMatchSetId string
	Name           string
	ByteMatchTuples []ByteMatchTuple
}

// ByteMatchTuple holds a byte match condition.
type ByteMatchTuple struct {
	FieldToMatch        FieldToMatch
	TargetString        string
	TextTransformation  string
	PositionalConstraint string
}

// FieldToMatch identifies the part of a request to inspect.
type FieldToMatch struct {
	Type string
	Data string
}

// ResourceAssociation tracks Web ACL to resource associations.
type ResourceAssociation struct {
	WebACLId    string
	ResourceArn string
}

// Store is the in-memory store for WAF Regional resources.
type Store struct {
	mu             sync.RWMutex
	webACLs        map[string]*WebACL
	rules          map[string]*Rule
	ipSets         map[string]*IPSet
	byteMatchSets  map[string]*ByteMatchSet
	associations   []ResourceAssociation
	changeTokens   map[string]bool
	accountID      string
	region         string
}

// NewStore creates an empty WAF Regional Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		webACLs:       make(map[string]*WebACL),
		rules:         make(map[string]*Rule),
		ipSets:        make(map[string]*IPSet),
		byteMatchSets: make(map[string]*ByteMatchSet),
		associations:  make([]ResourceAssociation, 0),
		changeTokens:  make(map[string]bool),
		accountID:     accountID,
		region:        region,
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

func (s *Store) newChangeToken() string {
	token := newUUID()
	s.changeTokens[token] = true
	return token
}

func (s *Store) buildWebACLArn(id string) string {
	return fmt.Sprintf("arn:aws:waf-regional:%s:%s:webacl/%s", s.region, s.accountID, id)
}

// CreateWebACL creates a new Web ACL.
func (s *Store) CreateWebACL(name, metricName, defaultAction, changeToken string) (*WebACL, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	acl := &WebACL{
		WebACLId:      id,
		Name:          name,
		MetricName:    metricName,
		DefaultAction: defaultAction,
		Rules:         make([]ActivatedRule, 0),
		WebACLArn:     s.buildWebACLArn(id),
		ChangeToken:   s.newChangeToken(),
	}
	s.webACLs[id] = acl
	return acl, nil
}

// GetWebACL returns a Web ACL by ID.
func (s *Store) GetWebACL(id string) (*WebACL, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acl, ok := s.webACLs[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			fmt.Sprintf("WebACL %s not found.", id), http.StatusBadRequest)
	}
	return acl, nil
}

// ListWebACLs returns all Web ACLs.
func (s *Store) ListWebACLs() []*WebACL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*WebACL, 0, len(s.webACLs))
	for _, acl := range s.webACLs {
		out = append(out, acl)
	}
	return out
}

// UpdateWebACL updates a Web ACL (adding/removing rules).
func (s *Store) UpdateWebACL(id, changeToken string, updates []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	acl, ok := s.webACLs[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"WebACL not found.", http.StatusBadRequest)
	}
	// Process updates (simplified: just replace rules)
	for _, u := range updates {
		action, _ := u["Action"].(string)
		ruleMap, _ := u["ActivatedRule"].(map[string]any)
		if ruleMap == nil {
			continue
		}
		ruleId, _ := ruleMap["RuleId"].(string)
		priority := 0
		if v, ok := ruleMap["Priority"].(float64); ok {
			priority = int(v)
		}
		ruleAction, _ := ruleMap["Action"].(map[string]any)
		actionType := "BLOCK"
		if ruleAction != nil {
			if v, ok := ruleAction["Type"].(string); ok {
				actionType = v
			}
		}
		if action == "INSERT" {
			acl.Rules = append(acl.Rules, ActivatedRule{
				Priority: priority,
				RuleId:   ruleId,
				Action:   actionType,
				Type:     "REGULAR",
			})
		} else if action == "DELETE" {
			for i, r := range acl.Rules {
				if r.RuleId == ruleId {
					acl.Rules = append(acl.Rules[:i], acl.Rules[i+1:]...)
					break
				}
			}
		}
	}
	acl.ChangeToken = s.newChangeToken()
	return nil
}

// DeleteWebACL removes a Web ACL.
func (s *Store) DeleteWebACL(id, changeToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.webACLs[id]; !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"WebACL not found.", http.StatusBadRequest)
	}
	delete(s.webACLs, id)
	return nil
}

// CreateRule creates a new rule.
func (s *Store) CreateRule(name, metricName, changeToken string) (*Rule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	rule := &Rule{
		RuleId:     id,
		Name:       name,
		MetricName: metricName,
		Predicates: make([]Predicate, 0),
	}
	s.rules[id] = rule
	return rule, nil
}

// GetRule returns a rule by ID.
func (s *Store) GetRule(id string) (*Rule, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rule, ok := s.rules[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			fmt.Sprintf("Rule %s not found.", id), http.StatusBadRequest)
	}
	return rule, nil
}

// ListRules returns all rules.
func (s *Store) ListRules() []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Rule, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, r)
	}
	return out
}

// UpdateRule updates a rule (adding/removing predicates).
func (s *Store) UpdateRule(id, changeToken string, updates []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	rule, ok := s.rules[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"Rule not found.", http.StatusBadRequest)
	}
	for _, u := range updates {
		action, _ := u["Action"].(string)
		predMap, _ := u["Predicate"].(map[string]any)
		if predMap == nil {
			continue
		}
		negated, _ := predMap["Negated"].(bool)
		predType, _ := predMap["Type"].(string)
		dataId, _ := predMap["DataId"].(string)
		// Validate predicate type
		if predType != "" {
			validPredicateTypes := map[string]bool{
				"IPMatch": true, "ByteMatch": true, "SqlInjectionMatch": true,
				"GeoMatch": true, "SizeConstraint": true, "XssMatch": true,
				"RegexMatch": true,
			}
			if !validPredicateTypes[predType] {
				return service.NewAWSError("WAFInvalidParameterException",
					fmt.Sprintf("Invalid predicate type: %s", predType), http.StatusBadRequest)
			}
		}
		if action == "INSERT" {
			rule.Predicates = append(rule.Predicates, Predicate{
				Negated: negated, Type: predType, DataId: dataId,
			})
		} else if action == "DELETE" {
			for i, p := range rule.Predicates {
				if p.DataId == dataId {
					rule.Predicates = append(rule.Predicates[:i], rule.Predicates[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

// DeleteRule removes a rule.
func (s *Store) DeleteRule(id, changeToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"Rule not found.", http.StatusBadRequest)
	}
	delete(s.rules, id)
	return nil
}

// CreateIPSet creates a new IP set.
func (s *Store) CreateIPSet(name, changeToken string) (*IPSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	ipSet := &IPSet{
		IPSetId:          id,
		Name:             name,
		IPSetDescriptors: make([]IPSetDescriptor, 0),
	}
	s.ipSets[id] = ipSet
	return ipSet, nil
}

// GetIPSet returns an IP set by ID.
func (s *Store) GetIPSet(id string) (*IPSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ipSet, ok := s.ipSets[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			fmt.Sprintf("IPSet %s not found.", id), http.StatusBadRequest)
	}
	return ipSet, nil
}

// ListIPSets returns all IP sets.
func (s *Store) ListIPSets() []*IPSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*IPSet, 0, len(s.ipSets))
	for _, ipSet := range s.ipSets {
		out = append(out, ipSet)
	}
	return out
}

// UpdateIPSet updates an IP set.
func (s *Store) UpdateIPSet(id, changeToken string, updates []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ipSet, ok := s.ipSets[id]
	if !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"IPSet not found.", http.StatusBadRequest)
	}
	for _, u := range updates {
		action, _ := u["Action"].(string)
		descMap, _ := u["IPSetDescriptor"].(map[string]any)
		if descMap == nil {
			continue
		}
		descType, _ := descMap["Type"].(string)
		value, _ := descMap["Value"].(string)
		if action == "INSERT" {
			// Validate CIDR format
			if value != "" && strings.Contains(value, "/") {
				_, _, err := net.ParseCIDR(value)
				if err != nil {
					return service.NewAWSError("WAFInvalidParameterException",
						fmt.Sprintf("Invalid CIDR: %s", value), http.StatusBadRequest)
				}
			}
			ipSet.IPSetDescriptors = append(ipSet.IPSetDescriptors, IPSetDescriptor{
				Type: descType, Value: value,
			})
		} else if action == "DELETE" {
			for i, d := range ipSet.IPSetDescriptors {
				if d.Value == value {
					ipSet.IPSetDescriptors = append(ipSet.IPSetDescriptors[:i], ipSet.IPSetDescriptors[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

// DeleteIPSet removes an IP set.
func (s *Store) DeleteIPSet(id, changeToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ipSets[id]; !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"IPSet not found.", http.StatusBadRequest)
	}
	delete(s.ipSets, id)
	return nil
}

// CreateByteMatchSet creates a new byte match set.
func (s *Store) CreateByteMatchSet(name, changeToken string) (*ByteMatchSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newUUID()
	bms := &ByteMatchSet{
		ByteMatchSetId:  id,
		Name:            name,
		ByteMatchTuples: make([]ByteMatchTuple, 0),
	}
	s.byteMatchSets[id] = bms
	return bms, nil
}

// GetByteMatchSet returns a byte match set.
func (s *Store) GetByteMatchSet(id string) (*ByteMatchSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bms, ok := s.byteMatchSets[id]
	if !ok {
		return nil, service.NewAWSError("WAFNonexistentItemException",
			fmt.Sprintf("ByteMatchSet %s not found.", id), http.StatusBadRequest)
	}
	return bms, nil
}

// ListByteMatchSets returns all byte match sets.
func (s *Store) ListByteMatchSets() []*ByteMatchSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ByteMatchSet, 0, len(s.byteMatchSets))
	for _, bms := range s.byteMatchSets {
		out = append(out, bms)
	}
	return out
}

// DeleteByteMatchSet removes a byte match set.
func (s *Store) DeleteByteMatchSet(id, changeToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byteMatchSets[id]; !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"ByteMatchSet not found.", http.StatusBadRequest)
	}
	delete(s.byteMatchSets, id)
	return nil
}

// AssociateWebACL associates a Web ACL with a resource.
func (s *Store) AssociateWebACL(webACLId, resourceArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.webACLs[webACLId]; !ok {
		return service.NewAWSError("WAFNonexistentItemException",
			"WebACL not found.", http.StatusBadRequest)
	}
	s.associations = append(s.associations, ResourceAssociation{
		WebACLId: webACLId, ResourceArn: resourceArn,
	})
	return nil
}

// DisassociateWebACL disassociates a Web ACL from a resource.
func (s *Store) DisassociateWebACL(resourceArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.associations {
		if a.ResourceArn == resourceArn {
			s.associations = append(s.associations[:i], s.associations[i+1:]...)
			return nil
		}
	}
	return nil
}

// GetWebACLForResource returns the Web ACL for a resource.
func (s *Store) GetWebACLForResource(resourceArn string) (*WebACL, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.associations {
		if a.ResourceArn == resourceArn {
			if acl, ok := s.webACLs[a.WebACLId]; ok {
				return acl, nil
			}
		}
	}
	return nil, service.NewAWSError("WAFNonexistentItemException",
		"No WebACL associated with this resource.", http.StatusBadRequest)
}
