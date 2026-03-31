package ssoadmin

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

// Instance represents an SSO instance.
type Instance struct {
	InstanceArn    string
	IdentityStoreId string
	Name           string
	CreatedDate    time.Time
	Status         string
}

// PermissionSet holds a permission set definition.
type PermissionSet struct {
	PermissionSetArn string
	Name             string
	Description      string
	SessionDuration  string
	RelayState       string
	CreatedDate      time.Time
	InstanceArn      string
	Tags             []Tag
	ManagedPolicies  []ManagedPolicy
	InlinePolicy     string
}

// ManagedPolicy represents an attached managed policy.
type ManagedPolicy struct {
	Arn  string
	Name string
}

// AccountAssignment holds an account assignment.
type AccountAssignment struct {
	InstanceArn      string
	PermissionSetArn string
	TargetId         string
	TargetType       string
	PrincipalId      string
	PrincipalType    string
}

// Store is the in-memory store for SSO Admin resources.
type Store struct {
	mu                sync.RWMutex
	instances         map[string]*Instance
	permissionSets    map[string]*PermissionSet // keyed by ARN
	accountAssignments []AccountAssignment
	accountID         string
	region            string
}

// NewStore creates an empty SSO Admin Store.
func NewStore(accountID, region string) *Store {
	// Pre-create a default SSO instance
	instanceArn := fmt.Sprintf("arn:aws:sso:::%s:instance/ssoins-%s", accountID, newShortID())
	store := &Store{
		instances:          make(map[string]*Instance),
		permissionSets:     make(map[string]*PermissionSet),
		accountAssignments: make([]AccountAssignment, 0),
		accountID:          accountID,
		region:             region,
	}
	store.instances[instanceArn] = &Instance{
		InstanceArn:     instanceArn,
		IdentityStoreId: fmt.Sprintf("d-%s", newShortID()),
		Name:            "Default SSO Instance",
		CreatedDate:     time.Now().UTC(),
		Status:          "ACTIVE",
	}
	return store
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) buildPermissionSetArn(instanceArn, psID string) string {
	return fmt.Sprintf("%s/ps-%s", instanceArn, psID)
}

// ListInstances returns all SSO instances.
func (s *Store) ListInstances() []*Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		out = append(out, inst)
	}
	return out
}

// DescribeInstance returns an SSO instance by ARN.
func (s *Store) DescribeInstance(arn string) (*Instance, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.instances[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Instance %s not found.", arn), http.StatusNotFound)
	}
	return inst, nil
}

// CreatePermissionSet creates a new permission set.
func (s *Store) CreatePermissionSet(instanceArn, name, description, sessionDuration string, tags []Tag) (*PermissionSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instances[instanceArn]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Instance not found.", http.StatusNotFound)
	}

	// Check for duplicate name
	for _, ps := range s.permissionSets {
		if ps.InstanceArn == instanceArn && ps.Name == name {
			return nil, service.NewAWSError("ConflictException",
				fmt.Sprintf("Permission set with name %s already exists.", name), http.StatusConflict)
		}
	}

	psID := newShortID()
	arn := s.buildPermissionSetArn(instanceArn, psID)
	if sessionDuration == "" {
		sessionDuration = "PT1H"
	}

	ps := &PermissionSet{
		PermissionSetArn: arn,
		Name:             name,
		Description:      description,
		SessionDuration:  sessionDuration,
		CreatedDate:      time.Now().UTC(),
		InstanceArn:      instanceArn,
		Tags:             tags,
		ManagedPolicies:  make([]ManagedPolicy, 0),
	}
	s.permissionSets[arn] = ps
	return ps, nil
}

// DescribePermissionSet returns a permission set.
func (s *Store) DescribePermissionSet(instanceArn, permissionSetArn string) (*PermissionSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	return ps, nil
}

// ListPermissionSets returns all permission sets for an instance.
func (s *Store) ListPermissionSets(instanceArn string) []*PermissionSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*PermissionSet, 0)
	for _, ps := range s.permissionSets {
		if ps.InstanceArn == instanceArn {
			out = append(out, ps)
		}
	}
	return out
}

// UpdatePermissionSet updates a permission set.
func (s *Store) UpdatePermissionSet(instanceArn, permissionSetArn, description, sessionDuration, relayState string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	if description != "" {
		ps.Description = description
	}
	if sessionDuration != "" {
		ps.SessionDuration = sessionDuration
	}
	if relayState != "" {
		ps.RelayState = relayState
	}
	return nil
}

// DeletePermissionSet removes a permission set.
func (s *Store) DeletePermissionSet(instanceArn, permissionSetArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	delete(s.permissionSets, permissionSetArn)
	return nil
}

// CreateAccountAssignment creates an account assignment.
func (s *Store) CreateAccountAssignment(instanceArn, permissionSetArn, targetId, targetType, principalId, principalType string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	assignment := AccountAssignment{
		InstanceArn:      instanceArn,
		PermissionSetArn: permissionSetArn,
		TargetId:         targetId,
		TargetType:       targetType,
		PrincipalId:      principalId,
		PrincipalType:    principalType,
	}
	s.accountAssignments = append(s.accountAssignments, assignment)
	return nil
}

// ListAccountAssignments returns assignments for an instance and permission set.
func (s *Store) ListAccountAssignments(instanceArn, accountId, permissionSetArn string) []AccountAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AccountAssignment, 0)
	for _, a := range s.accountAssignments {
		if a.InstanceArn == instanceArn && a.PermissionSetArn == permissionSetArn {
			if accountId == "" || a.TargetId == accountId {
				out = append(out, a)
			}
		}
	}
	return out
}

// DeleteAccountAssignment removes an account assignment.
func (s *Store) DeleteAccountAssignment(instanceArn, permissionSetArn, targetId, targetType, principalId, principalType string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.accountAssignments {
		if a.InstanceArn == instanceArn && a.PermissionSetArn == permissionSetArn &&
			a.TargetId == targetId && a.PrincipalId == principalId {
			s.accountAssignments = append(s.accountAssignments[:i], s.accountAssignments[i+1:]...)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Account assignment not found.", http.StatusNotFound)
}

// AttachManagedPolicy attaches a managed policy to a permission set.
func (s *Store) AttachManagedPolicy(instanceArn, permissionSetArn, policyArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	for _, mp := range ps.ManagedPolicies {
		if mp.Arn == policyArn {
			return service.NewAWSError("ConflictException",
				"Policy already attached.", http.StatusConflict)
		}
	}
	ps.ManagedPolicies = append(ps.ManagedPolicies, ManagedPolicy{Arn: policyArn})
	return nil
}

// DetachManagedPolicy detaches a managed policy from a permission set.
func (s *Store) DetachManagedPolicy(instanceArn, permissionSetArn, policyArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	for i, mp := range ps.ManagedPolicies {
		if mp.Arn == policyArn {
			ps.ManagedPolicies = append(ps.ManagedPolicies[:i], ps.ManagedPolicies[i+1:]...)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Policy not attached.", http.StatusNotFound)
}

// ListManagedPolicies returns managed policies for a permission set.
func (s *Store) ListManagedPolicies(instanceArn, permissionSetArn string) ([]ManagedPolicy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	return ps.ManagedPolicies, nil
}

// PutInlinePolicy sets the inline policy for a permission set.
func (s *Store) PutInlinePolicy(instanceArn, permissionSetArn, policy string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	ps.InlinePolicy = policy
	return nil
}

// GetInlinePolicy returns the inline policy for a permission set.
func (s *Store) GetInlinePolicy(instanceArn, permissionSetArn string) (string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return "", service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	return ps.InlinePolicy, nil
}

// DeleteInlinePolicy removes the inline policy from a permission set.
func (s *Store) DeleteInlinePolicy(instanceArn, permissionSetArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[permissionSetArn]
	if !ok || ps.InstanceArn != instanceArn {
		return service.NewAWSError("ResourceNotFoundException",
			"Permission set not found.", http.StatusNotFound)
	}
	ps.InlinePolicy = ""
	return nil
}

// TagResource adds tags to a permission set.
func (s *Store) TagResource(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Resource not found.", http.StatusNotFound)
	}
	for _, nt := range tags {
		found := false
		for i, et := range ps.Tags {
			if et.Key == nt.Key {
				ps.Tags[i].Value = nt.Value
				found = true
				break
			}
		}
		if !found {
			ps.Tags = append(ps.Tags, nt)
		}
	}
	return nil
}

// UntagResource removes tags from a permission set.
func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ps, ok := s.permissionSets[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Resource not found.", http.StatusNotFound)
	}
	keySet := make(map[string]bool, len(tagKeys))
	for _, k := range tagKeys {
		keySet[k] = true
	}
	out := make([]Tag, 0, len(ps.Tags))
	for _, t := range ps.Tags {
		if !keySet[t.Key] {
			out = append(out, t)
		}
	}
	ps.Tags = out
	return nil
}

// ListTagsForResource returns tags for a resource.
func (s *Store) ListTagsForResource(arn string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ps, ok := s.permissionSets[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Resource not found.", http.StatusNotFound)
	}
	return ps.Tags, nil
}
