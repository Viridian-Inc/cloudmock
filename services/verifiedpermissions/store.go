package verifiedpermissions

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// PolicyStore holds a Verified Permissions policy store.
type PolicyStore struct {
	PolicyStoreId string
	Arn           string
	Description   string
	CreatedDate   time.Time
	LastUpdatedDate time.Time
	ValidationSettings map[string]any
}

// Policy holds a policy in a policy store.
type Policy struct {
	PolicyId      string
	PolicyStoreId string
	PolicyType    string // STATIC or TEMPLATE_LINKED
	Principal     map[string]any
	Resource      map[string]any
	Definition    map[string]any
	CreatedDate   time.Time
	LastUpdatedDate time.Time
}

// Schema holds a schema for a policy store.
type Schema struct {
	PolicyStoreId string
	Schema        string
	CreatedDate   time.Time
	LastUpdatedDate time.Time
	Namespaces    []string
}

// PolicyTemplate holds a policy template.
type PolicyTemplate struct {
	PolicyTemplateId string
	PolicyStoreId    string
	Description      string
	Statement        string
	CreatedDate      time.Time
	LastUpdatedDate  time.Time
}

// IdentitySource holds an identity source.
type IdentitySource struct {
	IdentitySourceId string
	PolicyStoreId    string
	PrincipalEntityType string
	Configuration    map[string]any
	Details          map[string]any
	CreatedDate      time.Time
	LastUpdatedDate  time.Time
}

// Store is the in-memory store for Verified Permissions resources.
type Store struct {
	mu              sync.RWMutex
	policyStores    map[string]*PolicyStore
	policies        map[string]map[string]*Policy // keyed by store ID then policy ID
	schemas         map[string]*Schema
	policyTemplates map[string]map[string]*PolicyTemplate
	identitySources map[string]map[string]*IdentitySource
	accountID       string
	region          string
}

// NewStore creates an empty Verified Permissions Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		policyStores:    make(map[string]*PolicyStore),
		policies:        make(map[string]map[string]*Policy),
		schemas:         make(map[string]*Schema),
		policyTemplates: make(map[string]map[string]*PolicyTemplate),
		identitySources: make(map[string]map[string]*IdentitySource),
		accountID:       accountID,
		region:          region,
	}
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) buildPolicyStoreArn(id string) string {
	return fmt.Sprintf("arn:aws:verifiedpermissions:%s:%s:policy-store/%s", s.region, s.accountID, id)
}

// CreatePolicyStore creates a new policy store.
func (s *Store) CreatePolicyStore(description string, validationSettings map[string]any) (*PolicyStore, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	now := time.Now().UTC()
	ps := &PolicyStore{
		PolicyStoreId:      id,
		Arn:                s.buildPolicyStoreArn(id),
		Description:        description,
		CreatedDate:        now,
		LastUpdatedDate:    now,
		ValidationSettings: validationSettings,
	}
	s.policyStores[id] = ps
	s.policies[id] = make(map[string]*Policy)
	s.policyTemplates[id] = make(map[string]*PolicyTemplate)
	s.identitySources[id] = make(map[string]*IdentitySource)
	return ps, nil
}

// GetPolicyStore returns a policy store.
func (s *Store) GetPolicyStore(id string) (*PolicyStore, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps, ok := s.policyStores[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy store %s not found.", id), http.StatusNotFound)
	}
	return ps, nil
}

// ListPolicyStores returns all policy stores.
func (s *Store) ListPolicyStores() []*PolicyStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*PolicyStore, 0, len(s.policyStores))
	for _, ps := range s.policyStores {
		out = append(out, ps)
	}
	return out
}

// UpdatePolicyStore updates a policy store.
func (s *Store) UpdatePolicyStore(id, description string, validationSettings map[string]any) (*PolicyStore, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.policyStores[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy store %s not found.", id), http.StatusNotFound)
	}
	if description != "" {
		ps.Description = description
	}
	if validationSettings != nil {
		ps.ValidationSettings = validationSettings
	}
	ps.LastUpdatedDate = time.Now().UTC()
	return ps, nil
}

// DeletePolicyStore removes a policy store.
func (s *Store) DeletePolicyStore(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policyStores[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy store %s not found.", id), http.StatusNotFound)
	}
	delete(s.policyStores, id)
	delete(s.policies, id)
	delete(s.schemas, id)
	delete(s.policyTemplates, id)
	delete(s.identitySources, id)
	return nil
}

// CreatePolicy creates a new policy.
func (s *Store) CreatePolicy(storeId string, definition map[string]any) (*Policy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policyStores[storeId]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	// Validate Cedar policy statement if static
	if staticDef, ok := definition["static"].(map[string]any); ok {
		if stmt, ok := staticDef["statement"].(string); ok && stmt != "" {
			stmtLower := strings.ToLower(strings.TrimSpace(stmt))
			if !strings.HasPrefix(stmtLower, "permit") && !strings.HasPrefix(stmtLower, "forbid") {
				return nil, service.NewAWSError("ValidationException",
					"Cedar policy statement must start with 'permit' or 'forbid'.",
					http.StatusBadRequest)
			}
		}
	}

	policyId := newID()
	now := time.Now().UTC()
	policyType := "STATIC"
	if _, ok := definition["templateLinked"]; ok {
		policyType = "TEMPLATE_LINKED"
	}
	p := &Policy{
		PolicyId:        policyId,
		PolicyStoreId:   storeId,
		PolicyType:      policyType,
		Definition:      definition,
		CreatedDate:     now,
		LastUpdatedDate: now,
	}
	s.policies[storeId][policyId] = p
	return p, nil
}

// GetPolicy returns a policy.
func (s *Store) GetPolicy(storeId, policyId string) (*Policy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storePolicies, ok := s.policies[storeId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	p, ok := storePolicies[policyId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyId), http.StatusNotFound)
	}
	return p, nil
}

// ListPolicies returns all policies for a store.
func (s *Store) ListPolicies(storeId string) []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storePolicies, ok := s.policies[storeId]
	if !ok {
		return nil
	}
	out := make([]*Policy, 0, len(storePolicies))
	for _, p := range storePolicies {
		out = append(out, p)
	}
	return out
}

// UpdatePolicy updates a policy.
func (s *Store) UpdatePolicy(storeId, policyId string, definition map[string]any) (*Policy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storePolicies, ok := s.policies[storeId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	p, ok := storePolicies[policyId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyId), http.StatusNotFound)
	}
	if definition != nil {
		p.Definition = definition
	}
	p.LastUpdatedDate = time.Now().UTC()
	return p, nil
}

// DeletePolicy removes a policy.
func (s *Store) DeletePolicy(storeId, policyId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	storePolicies, ok := s.policies[storeId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	if _, ok := storePolicies[policyId]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyId), http.StatusNotFound)
	}
	delete(storePolicies, policyId)
	return nil
}

// PutSchema sets the schema for a policy store.
func (s *Store) PutSchema(storeId, schema string) (*Schema, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policyStores[storeId]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	now := time.Now().UTC()
	sc := &Schema{
		PolicyStoreId:   storeId,
		Schema:          schema,
		CreatedDate:     now,
		LastUpdatedDate: now,
		Namespaces:      []string{"default"},
	}
	s.schemas[storeId] = sc
	return sc, nil
}

// GetSchema returns the schema for a policy store.
func (s *Store) GetSchema(storeId string) (*Schema, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.schemas[storeId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Schema not found for policy store.", http.StatusNotFound)
	}
	return sc, nil
}

// IsAuthorized evaluates an authorization request (mock: always ALLOW).
func (s *Store) IsAuthorized(storeId string) (string, map[string]any) {
	return "ALLOW", map[string]any{
		"determiningPolicies": []map[string]any{},
		"errors":              []map[string]any{},
	}
}

// CreatePolicyTemplate creates a policy template.
func (s *Store) CreatePolicyTemplate(storeId, description, statement string) (*PolicyTemplate, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policyStores[storeId]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	templateId := newID()
	now := time.Now().UTC()
	pt := &PolicyTemplate{
		PolicyTemplateId: templateId,
		PolicyStoreId:    storeId,
		Description:      description,
		Statement:        statement,
		CreatedDate:      now,
		LastUpdatedDate:  now,
	}
	s.policyTemplates[storeId][templateId] = pt
	return pt, nil
}

// GetPolicyTemplate returns a policy template.
func (s *Store) GetPolicyTemplate(storeId, templateId string) (*PolicyTemplate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storeTemplates, ok := s.policyTemplates[storeId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	pt, ok := storeTemplates[templateId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy template %s not found.", templateId), http.StatusNotFound)
	}
	return pt, nil
}

// ListPolicyTemplates returns all templates for a store.
func (s *Store) ListPolicyTemplates(storeId string) []*PolicyTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storeTemplates, ok := s.policyTemplates[storeId]
	if !ok {
		return nil
	}
	out := make([]*PolicyTemplate, 0, len(storeTemplates))
	for _, pt := range storeTemplates {
		out = append(out, pt)
	}
	return out
}

// DeletePolicyTemplate removes a policy template.
func (s *Store) DeletePolicyTemplate(storeId, templateId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeTemplates, ok := s.policyTemplates[storeId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	if _, ok := storeTemplates[templateId]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Policy template %s not found.", templateId), http.StatusNotFound)
	}
	delete(storeTemplates, templateId)
	return nil
}

// CreateIdentitySource creates an identity source.
func (s *Store) CreateIdentitySource(storeId, principalEntityType string, config map[string]any) (*IdentitySource, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policyStores[storeId]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	isId := newID()
	now := time.Now().UTC()
	is := &IdentitySource{
		IdentitySourceId:    isId,
		PolicyStoreId:       storeId,
		PrincipalEntityType: principalEntityType,
		Configuration:       config,
		CreatedDate:         now,
		LastUpdatedDate:     now,
	}
	s.identitySources[storeId][isId] = is
	return is, nil
}

// GetIdentitySource returns an identity source.
func (s *Store) GetIdentitySource(storeId, isId string) (*IdentitySource, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storeSources, ok := s.identitySources[storeId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	is, ok := storeSources[isId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Identity source %s not found.", isId), http.StatusNotFound)
	}
	return is, nil
}

// ListIdentitySources returns all identity sources for a store.
func (s *Store) ListIdentitySources(storeId string) []*IdentitySource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	storeSources, ok := s.identitySources[storeId]
	if !ok {
		return nil
	}
	out := make([]*IdentitySource, 0, len(storeSources))
	for _, is := range storeSources {
		out = append(out, is)
	}
	return out
}

// DeleteIdentitySource removes an identity source.
func (s *Store) DeleteIdentitySource(storeId, isId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	storeSources, ok := s.identitySources[storeId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Policy store not found.", http.StatusNotFound)
	}
	if _, ok := storeSources[isId]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Identity source %s not found.", isId), http.StatusNotFound)
	}
	delete(storeSources, isId)
	return nil
}
