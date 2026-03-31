package iot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

type Thing struct {
	ThingName      string
	ThingArn       string
	ThingTypeName  string
	Attributes     map[string]string
	Version        int64
	Principals     []string // attached certificate ARNs
}

type ThingType struct {
	ThingTypeName string
	ThingTypeArn  string
	Properties    map[string]any
	CreationDate  time.Time
	Deprecated    bool
}

type ThingGroup struct {
	ThingGroupName string
	ThingGroupArn  string
	Properties     map[string]any
	ParentGroupName string
	Things         map[string]bool // thing names in this group
	CreationDate   time.Time
}

type Policy struct {
	PolicyName     string
	PolicyArn      string
	PolicyDocument string
	VersionId      string
	CreationDate   time.Time
	Targets        []string // attached targets (certificate ARNs, etc.)
}

type Certificate struct {
	CertificateId    string
	CertificateArn   string
	CertificatePem   string
	KeyPair          map[string]string
	Status           string // ACTIVE, INACTIVE, REVOKED
	CreationDate     time.Time
	Policies         []string // attached policy names
}

type TopicRule struct {
	RuleName     string
	RuleArn      string
	Sql          string
	Description  string
	Actions      []map[string]any
	RuleDisabled bool
	CreationDate time.Time
}

type Store struct {
	mu           sync.RWMutex
	things       map[string]*Thing       // keyed by name
	thingTypes   map[string]*ThingType   // keyed by name
	thingGroups  map[string]*ThingGroup  // keyed by name
	policies     map[string]*Policy      // keyed by name
	certificates map[string]*Certificate // keyed by certificateId
	topicRules   map[string]*TopicRule   // keyed by rule name
	tagsByArn    map[string]map[string]string
	accountID    string
	region       string
}

func NewStore(accountID, region string) *Store {
	return &Store{
		things:       make(map[string]*Thing),
		thingTypes:   make(map[string]*ThingType),
		thingGroups:  make(map[string]*ThingGroup),
		policies:     make(map[string]*Policy),
		certificates: make(map[string]*Certificate),
		topicRules:   make(map[string]*TopicRule),
		tagsByArn:    make(map[string]map[string]string),
		accountID:    accountID,
		region:       region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newCertId() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) thingARN(name string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:thing/%s", s.region, s.accountID, name)
}
func (s *Store) thingTypeARN(name string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:thingtype/%s", s.region, s.accountID, name)
}
func (s *Store) thingGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:thinggroup/%s", s.region, s.accountID, name)
}
func (s *Store) policyARN(name string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:policy/%s", s.region, s.accountID, name)
}
func (s *Store) certificateARN(id string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:cert/%s", s.region, s.accountID, id)
}
func (s *Store) topicRuleARN(name string) string {
	return fmt.Sprintf("arn:aws:iot:%s:%s:rule/%s", s.region, s.accountID, name)
}

// Things.

func (s *Store) CreateThing(name, thingTypeName string, attributes map[string]string) (*Thing, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.things[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Thing %s already exists", name), http.StatusConflict)
	}
	if attributes == nil {
		attributes = make(map[string]string)
	}
	t := &Thing{
		ThingName:     name,
		ThingArn:      s.thingARN(name),
		ThingTypeName: thingTypeName,
		Attributes:    attributes,
		Version:       1,
	}
	s.things[name] = t
	s.tagsByArn[t.ThingArn] = make(map[string]string)
	return t, nil
}

func (s *Store) DescribeThing(name string) (*Thing, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.things[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", name), http.StatusNotFound)
	}
	return t, nil
}

func (s *Store) ListThings() []*Thing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Thing, 0, len(s.things))
	for _, t := range s.things {
		out = append(out, t)
	}
	return out
}

func (s *Store) UpdateThing(name, thingTypeName string, attributes map[string]string, removeThingType bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.things[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", name), http.StatusNotFound)
	}
	if removeThingType {
		t.ThingTypeName = ""
	} else if thingTypeName != "" {
		t.ThingTypeName = thingTypeName
	}
	if attributes != nil {
		t.Attributes = attributes
	}
	t.Version++
	return nil
}

func (s *Store) DeleteThing(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.things[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", name), http.StatusNotFound)
	}
	delete(s.things, name)
	delete(s.tagsByArn, t.ThingArn)
	return nil
}

// Thing types.

func (s *Store) CreateThingType(name string, properties map[string]any) (*ThingType, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.thingTypes[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Thing type %s already exists", name), http.StatusConflict)
	}
	tt := &ThingType{
		ThingTypeName: name,
		ThingTypeArn:  s.thingTypeARN(name),
		Properties:    properties,
		CreationDate:  time.Now().UTC(),
	}
	s.thingTypes[name] = tt
	s.tagsByArn[tt.ThingTypeArn] = make(map[string]string)
	return tt, nil
}

func (s *Store) DescribeThingType(name string) (*ThingType, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tt, ok := s.thingTypes[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing type %s not found", name), http.StatusNotFound)
	}
	return tt, nil
}

func (s *Store) ListThingTypes() []*ThingType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ThingType, 0, len(s.thingTypes))
	for _, tt := range s.thingTypes {
		out = append(out, tt)
	}
	return out
}

func (s *Store) DeleteThingType(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	tt, ok := s.thingTypes[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing type %s not found", name), http.StatusNotFound)
	}
	delete(s.thingTypes, name)
	delete(s.tagsByArn, tt.ThingTypeArn)
	return nil
}

// Thing groups.

func (s *Store) CreateThingGroup(name, parentGroupName string, properties map[string]any) (*ThingGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.thingGroups[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Thing group %s already exists", name), http.StatusConflict)
	}
	tg := &ThingGroup{
		ThingGroupName:  name,
		ThingGroupArn:   s.thingGroupARN(name),
		Properties:      properties,
		ParentGroupName: parentGroupName,
		Things:          make(map[string]bool),
		CreationDate:    time.Now().UTC(),
	}
	s.thingGroups[name] = tg
	s.tagsByArn[tg.ThingGroupArn] = make(map[string]string)
	return tg, nil
}

func (s *Store) DescribeThingGroup(name string) (*ThingGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tg, ok := s.thingGroups[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing group %s not found", name), http.StatusNotFound)
	}
	return tg, nil
}

func (s *Store) ListThingGroups() []*ThingGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ThingGroup, 0, len(s.thingGroups))
	for _, tg := range s.thingGroups {
		out = append(out, tg)
	}
	return out
}

func (s *Store) DeleteThingGroup(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	tg, ok := s.thingGroups[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing group %s not found", name), http.StatusNotFound)
	}
	delete(s.thingGroups, name)
	delete(s.tagsByArn, tg.ThingGroupArn)
	return nil
}

func (s *Store) AddThingToThingGroup(thingName, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	tg, ok := s.thingGroups[groupName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing group %s not found", groupName), http.StatusNotFound)
	}
	if _, ok := s.things[thingName]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", thingName), http.StatusNotFound)
	}
	tg.Things[thingName] = true
	return nil
}

func (s *Store) RemoveThingFromThingGroup(thingName, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	tg, ok := s.thingGroups[groupName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing group %s not found", groupName), http.StatusNotFound)
	}
	delete(tg.Things, thingName)
	return nil
}

// Policies.

func (s *Store) CreatePolicy(name, document string) (*Policy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.policies[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Policy %s already exists", name), http.StatusConflict)
	}
	p := &Policy{
		PolicyName:     name,
		PolicyArn:      s.policyARN(name),
		PolicyDocument: document,
		VersionId:      "1",
		CreationDate:   time.Now().UTC(),
	}
	s.policies[name] = p
	s.tagsByArn[p.PolicyArn] = make(map[string]string)
	return p, nil
}

func (s *Store) GetPolicy(name string) (*Policy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found", name), http.StatusNotFound)
	}
	return p, nil
}

func (s *Store) ListPolicies() []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Policy, 0, len(s.policies))
	for _, p := range s.policies {
		out = append(out, p)
	}
	return out
}

func (s *Store) DeletePolicy(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.policies[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found", name), http.StatusNotFound)
	}
	delete(s.policies, name)
	delete(s.tagsByArn, p.PolicyArn)
	return nil
}

func (s *Store) AttachPolicy(policyName, targetArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.policies[policyName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found", policyName), http.StatusNotFound)
	}
	p.Targets = append(p.Targets, targetArn)
	return nil
}

func (s *Store) DetachPolicy(policyName, targetArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.policies[policyName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found", policyName), http.StatusNotFound)
	}
	targets := make([]string, 0)
	for _, t := range p.Targets {
		if t != targetArn {
			targets = append(targets, t)
		}
	}
	p.Targets = targets
	return nil
}

func (s *Store) ListAttachedPolicies(targetArn string) []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Policy, 0)
	for _, p := range s.policies {
		for _, t := range p.Targets {
			if t == targetArn {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// Certificates.

func (s *Store) CreateKeysAndCertificate(setAsActive bool) (*Certificate, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	certId := newCertId()
	status := "INACTIVE"
	if setAsActive {
		status = "ACTIVE"
	}
	cert := &Certificate{
		CertificateId:  certId,
		CertificateArn: s.certificateARN(certId),
		CertificatePem: "-----BEGIN CERTIFICATE-----\nMOCKCERTIFICATE\n-----END CERTIFICATE-----",
		KeyPair: map[string]string{
			"PublicKey":  "-----BEGIN PUBLIC KEY-----\nMOCKPUBLICKEY\n-----END PUBLIC KEY-----",
			"PrivateKey": "-----BEGIN RSA PRIVATE KEY-----\nMOCKPRIVATEKEY\n-----END RSA PRIVATE KEY-----",
		},
		Status:       status,
		CreationDate: time.Now().UTC(),
	}
	s.certificates[certId] = cert
	s.tagsByArn[cert.CertificateArn] = make(map[string]string)
	return cert, nil
}

func (s *Store) DescribeCertificate(certId string) (*Certificate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cert, ok := s.certificates[certId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Certificate %s not found", certId), http.StatusNotFound)
	}
	return cert, nil
}

func (s *Store) ListCertificates() []*Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Certificate, 0, len(s.certificates))
	for _, cert := range s.certificates {
		out = append(out, cert)
	}
	return out
}

func (s *Store) DeleteCertificate(certId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, ok := s.certificates[certId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Certificate %s not found", certId), http.StatusNotFound)
	}
	delete(s.certificates, certId)
	delete(s.tagsByArn, cert.CertificateArn)
	return nil
}

func (s *Store) AttachThingPrincipal(thingName, principal string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.things[thingName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", thingName), http.StatusNotFound)
	}
	t.Principals = append(t.Principals, principal)
	return nil
}

func (s *Store) DetachThingPrincipal(thingName, principal string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.things[thingName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Thing %s not found", thingName), http.StatusNotFound)
	}
	principals := make([]string, 0)
	for _, p := range t.Principals {
		if p != principal {
			principals = append(principals, p)
		}
	}
	t.Principals = principals
	return nil
}

// Topic rules.

func (s *Store) CreateTopicRule(name, sql, description string, actions []map[string]any, ruleDisabled bool) (*TopicRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.topicRules[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Topic rule %s already exists", name), http.StatusConflict)
	}
	tr := &TopicRule{
		RuleName:     name,
		RuleArn:      s.topicRuleARN(name),
		Sql:          sql,
		Description:  description,
		Actions:      actions,
		RuleDisabled: ruleDisabled,
		CreationDate: time.Now().UTC(),
	}
	s.topicRules[name] = tr
	s.tagsByArn[tr.RuleArn] = make(map[string]string)
	return tr, nil
}

func (s *Store) GetTopicRule(name string) (*TopicRule, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tr, ok := s.topicRules[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Topic rule %s not found", name), http.StatusNotFound)
	}
	return tr, nil
}

func (s *Store) ListTopicRules() []*TopicRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TopicRule, 0, len(s.topicRules))
	for _, tr := range s.topicRules {
		out = append(out, tr)
	}
	return out
}

func (s *Store) DeleteTopicRule(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.topicRules[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Topic rule %s not found", name), http.StatusNotFound)
	}
	delete(s.topicRules, name)
	delete(s.tagsByArn, tr.RuleArn)
	return nil
}

// Tags.

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for _, k := range tagKeys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}
