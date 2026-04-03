package organizations

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

// Organization holds the organization metadata.
type Organization struct {
	Id                    string
	Arn                   string
	MasterAccountId       string
	MasterAccountArn      string
	MasterAccountEmail    string
	FeatureSet            string
	AvailablePolicyTypes  []PolicyTypeSummary
}

// PolicyTypeSummary describes a policy type and its status.
type PolicyTypeSummary struct {
	Type   string
	Status string
}

// Root represents the root of the organization tree.
type Root struct {
	Id         string
	Arn        string
	Name       string
	PolicyTypes []PolicyTypeSummary
}

// OrganizationalUnit represents an OU in the hierarchy.
type OrganizationalUnit struct {
	Id       string
	Arn      string
	Name     string
	ParentId string
	Tags     []Tag
}

// CreateAccountStatus tracks the status of a CreateAccount request.
type CreateAccountStatus struct {
	Id                 string
	AccountName        string
	State              string
	AccountId          string
	RequestedTimestamp time.Time
	CompletedTimestamp time.Time
}

// Account represents an AWS account in the organization.
type Account struct {
	Id              string
	Arn             string
	Name            string
	Email           string
	Status          string
	JoinedMethod    string
	JoinedTimestamp time.Time
	ParentId        string
	Tags            []Tag
}

// Policy represents an organization policy.
type Policy struct {
	PolicySummary PolicySummary
	Content       string
	Tags          []Tag
}

// PolicySummary holds policy metadata.
type PolicySummary struct {
	Id          string
	Arn         string
	Name        string
	Description string
	Type        string
	AwsManaged  bool
}

// PolicyTarget tracks policy attachments.
type PolicyTarget struct {
	TargetId string
	Arn      string
	Name     string
	Type     string // ROOT, ORGANIZATIONAL_UNIT, ACCOUNT
}

// AccountProvisioner is called when a new account is created via Organizations.
// The account.Registry implements this by provisioning an isolated account.
type AccountProvisioner interface {
	ProvisionAccount(id, name string) error
}

// Store is the in-memory store for Organizations resources.
type Store struct {
	mu                   sync.RWMutex
	organization         *Organization
	roots                map[string]*Root
	ous                  map[string]*OrganizationalUnit
	accounts             map[string]*Account
	policies             map[string]*Policy
	policyAttachments    map[string][]PolicyTarget      // keyed by policy ID
	createAccountStatuses map[string]*CreateAccountStatus // keyed by request ID
	accountID            string
	region               string
	provisioner          AccountProvisioner
}

// NewStore creates an empty Organizations Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		roots:                 make(map[string]*Root),
		ous:                   make(map[string]*OrganizationalUnit),
		accounts:              make(map[string]*Account),
		policies:              make(map[string]*Policy),
		policyAttachments:     make(map[string][]PolicyTarget),
		createAccountStatuses: make(map[string]*CreateAccountStatus),
		accountID:             accountID,
		region:                region,
	}
}

// SetProvisioner attaches an AccountProvisioner that is called when new
// accounts are created via the CreateAccount API. This enables integration
// with the multi-account registry for resource isolation.
func (s *Store) SetProvisioner(p AccountProvisioner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provisioner = p
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newAccountID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%012x", b)[:12]
}

// CreateOrganization creates a new organization.
func (s *Store) CreateOrganization(featureSet string) (*Organization, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.organization != nil {
		return nil, service.NewAWSError("AlreadyInOrganizationException",
			"The account is already a member of an organization.", http.StatusConflict)
	}

	if featureSet == "" {
		featureSet = "ALL"
	}

	orgID := newID("o")
	org := &Organization{
		Id:                   orgID,
		Arn:                  fmt.Sprintf("arn:aws:organizations::%s:organization/%s", s.accountID, orgID),
		MasterAccountId:      s.accountID,
		MasterAccountArn:     fmt.Sprintf("arn:aws:organizations::%s:account/%s/%s", s.accountID, orgID, s.accountID),
		MasterAccountEmail:   fmt.Sprintf("admin@%s.example.com", s.accountID),
		FeatureSet:           featureSet,
		AvailablePolicyTypes: []PolicyTypeSummary{{Type: "SERVICE_CONTROL_POLICY", Status: "ENABLED"}},
	}

	rootID := newID("r")
	root := &Root{
		Id:   rootID,
		Arn:  fmt.Sprintf("arn:aws:organizations::%s:root/%s/%s", s.accountID, orgID, rootID),
		Name: "Root",
		PolicyTypes: []PolicyTypeSummary{
			{Type: "SERVICE_CONTROL_POLICY", Status: "ENABLED"},
		},
	}

	s.organization = org
	s.roots[rootID] = root

	// Add master account
	masterAccount := &Account{
		Id:              s.accountID,
		Arn:             org.MasterAccountArn,
		Name:            "Management Account",
		Email:           org.MasterAccountEmail,
		Status:          "ACTIVE",
		JoinedMethod:    "INVITED",
		JoinedTimestamp: time.Now().UTC(),
		ParentId:        rootID,
	}
	s.accounts[s.accountID] = masterAccount

	return org, nil
}

// GetOrganization returns the organization.
func (s *Store) GetOrganization() (*Organization, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.organization == nil {
		return nil, service.NewAWSError("AWSOrganizationsNotInUseException",
			"Your account is not a member of an organization.", http.StatusBadRequest)
	}
	return s.organization, nil
}

// DeleteOrganization removes the organization.
func (s *Store) DeleteOrganization() *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.organization == nil {
		return service.NewAWSError("AWSOrganizationsNotInUseException",
			"Your account is not a member of an organization.", http.StatusBadRequest)
	}
	s.organization = nil
	s.roots = make(map[string]*Root)
	s.ous = make(map[string]*OrganizationalUnit)
	s.accounts = make(map[string]*Account)
	s.policies = make(map[string]*Policy)
	s.policyAttachments = make(map[string][]PolicyTarget)
	return nil
}

// ListRoots returns all roots.
func (s *Store) ListRoots() []*Root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Root, 0, len(s.roots))
	for _, r := range s.roots {
		out = append(out, r)
	}
	return out
}

// CreateOU creates a new organizational unit.
func (s *Store) CreateOU(parentID, name string) (*OrganizationalUnit, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.organization == nil {
		return nil, service.NewAWSError("AWSOrganizationsNotInUseException",
			"Your account is not a member of an organization.", http.StatusBadRequest)
	}

	if name == "" {
		return nil, service.ErrValidation("Name is required.")
	}

	// Verify parent exists
	if _, ok := s.roots[parentID]; !ok {
		if _, ok := s.ous[parentID]; !ok {
			return nil, service.NewAWSError("ParentNotFoundException",
				fmt.Sprintf("Parent %s not found.", parentID), http.StatusBadRequest)
		}
	}

	ouID := newID("ou")
	ou := &OrganizationalUnit{
		Id:       ouID,
		Arn:      fmt.Sprintf("arn:aws:organizations::%s:ou/%s/%s", s.accountID, s.organization.Id, ouID),
		Name:     name,
		ParentId: parentID,
	}

	s.ous[ouID] = ou
	return ou, nil
}

// GetOU returns an OU by ID.
func (s *Store) GetOU(ouID string) (*OrganizationalUnit, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ou, ok := s.ous[ouID]
	if !ok {
		return nil, service.NewAWSError("OrganizationalUnitNotFoundException",
			fmt.Sprintf("OU %s not found.", ouID), http.StatusBadRequest)
	}
	return ou, nil
}

// ListOUsForParent returns OUs under a parent.
func (s *Store) ListOUsForParent(parentID string) []*OrganizationalUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*OrganizationalUnit, 0)
	for _, ou := range s.ous {
		if ou.ParentId == parentID {
			out = append(out, ou)
		}
	}
	return out
}

// DeleteOU removes an OU.
func (s *Store) DeleteOU(ouID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ous[ouID]; !ok {
		return service.NewAWSError("OrganizationalUnitNotFoundException",
			fmt.Sprintf("OU %s not found.", ouID), http.StatusBadRequest)
	}
	// Check for children
	for _, ou := range s.ous {
		if ou.ParentId == ouID {
			return service.NewAWSError("OrganizationalUnitNotEmptyException",
				"OU contains child OUs.", http.StatusBadRequest)
		}
	}
	for _, acct := range s.accounts {
		if acct.ParentId == ouID {
			return service.NewAWSError("OrganizationalUnitNotEmptyException",
				"OU contains accounts.", http.StatusBadRequest)
		}
	}
	delete(s.ous, ouID)
	return nil
}

// CreateAccount creates a new account and records the creation status.
func (s *Store) CreateAccount(name, email string) (*Account, *CreateAccountStatus, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.organization == nil {
		return nil, nil, service.NewAWSError("AWSOrganizationsNotInUseException",
			"Your account is not a member of an organization.", http.StatusBadRequest)
	}

	if name == "" || email == "" {
		return nil, nil, service.ErrValidation("AccountName and Email are required.")
	}

	accountId := newAccountID()
	// Find the root
	var rootID string
	for id := range s.roots {
		rootID = id
		break
	}

	now := time.Now().UTC()
	acct := &Account{
		Id:              accountId,
		Arn:             fmt.Sprintf("arn:aws:organizations::%s:account/%s/%s", s.accountID, s.organization.Id, accountId),
		Name:            name,
		Email:           email,
		Status:          "ACTIVE",
		JoinedMethod:    "CREATED",
		JoinedTimestamp: now,
		ParentId:        rootID,
	}

	requestID := newUUID()
	status := &CreateAccountStatus{
		Id:                 requestID,
		AccountName:        name,
		State:              "SUCCEEDED",
		AccountId:          accountId,
		RequestedTimestamp: now,
		CompletedTimestamp: now,
	}

	s.accounts[accountId] = acct
	s.createAccountStatuses[requestID] = status

	// Provision an isolated account in the account registry if available.
	if s.provisioner != nil {
		// Unlock before calling external code to avoid deadlocks.
		// The account is already recorded; provisioner failure is non-fatal.
		provisioner := s.provisioner
		s.mu.Unlock()
		_ = provisioner.ProvisionAccount(accountId, name)
		s.mu.Lock()
	}

	return acct, status, nil
}

// GetCreateAccountStatus returns the status of a CreateAccount request.
func (s *Store) GetCreateAccountStatus(requestID string) (*CreateAccountStatus, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.createAccountStatuses[requestID]
	if !ok {
		return nil, service.NewAWSError("CreateAccountStatusNotFoundException",
			fmt.Sprintf("CreateAccount request %s not found.", requestID), http.StatusBadRequest)
	}
	return status, nil
}

// GetAccount returns an account by ID.
func (s *Store) GetAccount(accountID string) (*Account, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acct, ok := s.accounts[accountID]
	if !ok {
		return nil, service.NewAWSError("AccountNotFoundException",
			fmt.Sprintf("Account %s not found.", accountID), http.StatusBadRequest)
	}
	return acct, nil
}

// ListAccounts returns all accounts.
func (s *Store) ListAccounts() []*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Account, 0, len(s.accounts))
	for _, a := range s.accounts {
		out = append(out, a)
	}
	return out
}

// ListAccountsForParent returns accounts under a parent.
func (s *Store) ListAccountsForParent(parentID string) []*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Account, 0)
	for _, a := range s.accounts {
		if a.ParentId == parentID {
			out = append(out, a)
		}
	}
	return out
}

// MoveAccount moves an account to a new parent.
func (s *Store) MoveAccount(accountID, sourceParentID, destParentID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	acct, ok := s.accounts[accountID]
	if !ok {
		return service.NewAWSError("AccountNotFoundException",
			fmt.Sprintf("Account %s not found.", accountID), http.StatusBadRequest)
	}
	if acct.ParentId != sourceParentID {
		return service.NewAWSError("SourceParentNotFoundException",
			"Account is not in the specified source parent.", http.StatusBadRequest)
	}
	// Verify dest exists
	if _, ok := s.roots[destParentID]; !ok {
		if _, ok := s.ous[destParentID]; !ok {
			return service.NewAWSError("DestinationParentNotFoundException",
				fmt.Sprintf("Destination parent %s not found.", destParentID), http.StatusBadRequest)
		}
	}
	acct.ParentId = destParentID
	return nil
}

// CreatePolicy creates a new policy.
func (s *Store) CreatePolicy(name, description, content, policyType string, tags []Tag) (*Policy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.organization == nil {
		return nil, service.NewAWSError("AWSOrganizationsNotInUseException",
			"Your account is not a member of an organization.", http.StatusBadRequest)
	}

	if name == "" || content == "" {
		return nil, service.ErrValidation("Name and Content are required.")
	}
	if policyType == "" {
		policyType = "SERVICE_CONTROL_POLICY"
	}

	policyID := newID("p")
	policy := &Policy{
		PolicySummary: PolicySummary{
			Id:          policyID,
			Arn:         fmt.Sprintf("arn:aws:organizations::%s:policy/%s/%s", s.accountID, s.organization.Id, policyID),
			Name:        name,
			Description: description,
			Type:        policyType,
			AwsManaged:  false,
		},
		Content: content,
		Tags:    tags,
	}

	s.policies[policyID] = policy
	return policy, nil
}

// GetPolicy returns a policy by ID.
func (s *Store) GetPolicy(policyID string) (*Policy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[policyID]
	if !ok {
		return nil, service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}
	return policy, nil
}

// ListPolicies returns policies by type.
func (s *Store) ListPolicies(policyType string) []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Policy, 0)
	for _, p := range s.policies {
		if policyType == "" || p.PolicySummary.Type == policyType {
			out = append(out, p)
		}
	}
	return out
}

// UpdatePolicy updates a policy.
func (s *Store) UpdatePolicy(policyID, name, description, content string) (*Policy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	policy, ok := s.policies[policyID]
	if !ok {
		return nil, service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}
	if name != "" {
		policy.PolicySummary.Name = name
	}
	if description != "" {
		policy.PolicySummary.Description = description
	}
	if content != "" {
		policy.Content = content
	}
	return policy, nil
}

// DeletePolicy removes a policy.
func (s *Store) DeletePolicy(policyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[policyID]; !ok {
		return service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}
	if targets, ok := s.policyAttachments[policyID]; ok && len(targets) > 0 {
		return service.NewAWSError("PolicyInUseException",
			"Policy is still attached to targets.", http.StatusBadRequest)
	}
	delete(s.policies, policyID)
	delete(s.policyAttachments, policyID)
	return nil
}

// AttachPolicy attaches a policy to a target.
func (s *Store) AttachPolicy(policyID, targetID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.policies[policyID]; !ok {
		return service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}

	// Determine target type
	targetType := ""
	targetName := ""
	targetArn := ""
	if _, ok := s.roots[targetID]; ok {
		targetType = "ROOT"
		targetName = "Root"
		targetArn = s.roots[targetID].Arn
	} else if ou, ok := s.ous[targetID]; ok {
		targetType = "ORGANIZATIONAL_UNIT"
		targetName = ou.Name
		targetArn = ou.Arn
	} else if acct, ok := s.accounts[targetID]; ok {
		targetType = "ACCOUNT"
		targetName = acct.Name
		targetArn = acct.Arn
	} else {
		return service.NewAWSError("TargetNotFoundException",
			fmt.Sprintf("Target %s not found.", targetID), http.StatusBadRequest)
	}

	// Check for duplicate
	for _, t := range s.policyAttachments[policyID] {
		if t.TargetId == targetID {
			return service.NewAWSError("DuplicatePolicyAttachmentException",
				"Policy is already attached to this target.", http.StatusConflict)
		}
	}

	s.policyAttachments[policyID] = append(s.policyAttachments[policyID], PolicyTarget{
		TargetId: targetID,
		Arn:      targetArn,
		Name:     targetName,
		Type:     targetType,
	})
	return nil
}

// DetachPolicy detaches a policy from a target.
func (s *Store) DetachPolicy(policyID, targetID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	targets, ok := s.policyAttachments[policyID]
	if !ok {
		return service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}

	for i, t := range targets {
		if t.TargetId == targetID {
			s.policyAttachments[policyID] = append(targets[:i], targets[i+1:]...)
			return nil
		}
	}

	return service.NewAWSError("PolicyNotAttachedException",
		"Policy is not attached to this target.", http.StatusBadRequest)
}

// ListTargetsForPolicy returns targets a policy is attached to.
func (s *Store) ListTargetsForPolicy(policyID string) ([]PolicyTarget, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.policies[policyID]; !ok {
		return nil, service.NewAWSError("PolicyNotFoundException",
			fmt.Sprintf("Policy %s not found.", policyID), http.StatusBadRequest)
	}
	return s.policyAttachments[policyID], nil
}

// ListChildren returns child OUs and accounts for a given parent.
// childType is either "ACCOUNT" or "ORGANIZATIONAL_UNIT".
func (s *Store) ListChildren(parentID, childType string) ([]map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate parent exists
	parentExists := false
	if _, ok := s.roots[parentID]; ok {
		parentExists = true
	}
	if _, ok := s.ous[parentID]; ok {
		parentExists = true
	}
	if !parentExists {
		return nil, service.NewAWSError("ParentNotFoundException",
			fmt.Sprintf("Parent %s not found.", parentID), http.StatusBadRequest)
	}

	out := make([]map[string]string, 0)
	if childType == "" || childType == "ACCOUNT" {
		for _, acc := range s.accounts {
			if acc.ParentId == parentID {
				out = append(out, map[string]string{"Id": acc.Id, "Type": "ACCOUNT"})
			}
		}
	}
	if childType == "" || childType == "ORGANIZATIONAL_UNIT" {
		for _, ou := range s.ous {
			if ou.ParentId == parentID {
				out = append(out, map[string]string{"Id": ou.Id, "Type": "ORGANIZATIONAL_UNIT"})
			}
		}
	}
	return out, nil
}

// ListParents returns the parent of a given child (account or OU).
func (s *Store) ListParents(childID string) ([]map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if it's an account
	if acc, ok := s.accounts[childID]; ok {
		parentID := acc.ParentId
		parentType := "ROOT"
		if _, isOU := s.ous[parentID]; isOU {
			parentType = "ORGANIZATIONAL_UNIT"
		}
		return []map[string]string{{"Id": parentID, "Type": parentType}}, nil
	}

	// Check if it's an OU
	if ou, ok := s.ous[childID]; ok {
		parentID := ou.ParentId
		parentType := "ROOT"
		if _, isOU := s.ous[parentID]; isOU {
			parentType = "ORGANIZATIONAL_UNIT"
		}
		return []map[string]string{{"Id": parentID, "Type": parentType}}, nil
	}

	return nil, service.NewAWSError("ChildNotFoundException",
		fmt.Sprintf("Child %s not found.", childID), http.StatusBadRequest)
}

// ListPoliciesForTarget returns policies attached to a given target (account, OU, or root).
func (s *Store) ListPoliciesForTarget(targetID, filter string) ([]*Policy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate target exists
	targetExists := false
	if _, ok := s.roots[targetID]; ok {
		targetExists = true
	}
	if _, ok := s.ous[targetID]; ok {
		targetExists = true
	}
	if _, ok := s.accounts[targetID]; ok {
		targetExists = true
	}
	if !targetExists {
		return nil, service.NewAWSError("TargetNotFoundException",
			fmt.Sprintf("Target %s not found.", targetID), http.StatusBadRequest)
	}

	out := make([]*Policy, 0)
	for policyID, targets := range s.policyAttachments {
		for _, t := range targets {
			if t.TargetId == targetID {
				policy, ok := s.policies[policyID]
				if !ok {
					continue
				}
				if filter != "" && policy.PolicySummary.Type != filter {
					continue
				}
				out = append(out, policy)
				break
			}
		}
	}
	return out, nil
}

// EnablePolicyType enables a policy type on a root.
func (s *Store) EnablePolicyType(rootID, policyType string) (*Root, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root, ok := s.roots[rootID]
	if !ok {
		return nil, service.NewAWSError("RootNotFoundException",
			fmt.Sprintf("Root %s not found.", rootID), http.StatusBadRequest)
	}
	for _, pt := range root.PolicyTypes {
		if pt.Type == policyType && pt.Status == "ENABLED" {
			return nil, service.NewAWSError("PolicyTypeAlreadyEnabledException",
				"Policy type is already enabled.", http.StatusBadRequest)
		}
	}
	root.PolicyTypes = append(root.PolicyTypes, PolicyTypeSummary{Type: policyType, Status: "ENABLED"})
	return root, nil
}

// DisablePolicyType disables a policy type on a root.
func (s *Store) DisablePolicyType(rootID, policyType string) (*Root, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	root, ok := s.roots[rootID]
	if !ok {
		return nil, service.NewAWSError("RootNotFoundException",
			fmt.Sprintf("Root %s not found.", rootID), http.StatusBadRequest)
	}
	for i, pt := range root.PolicyTypes {
		if pt.Type == policyType {
			root.PolicyTypes = append(root.PolicyTypes[:i], root.PolicyTypes[i+1:]...)
			return root, nil
		}
	}
	return nil, service.NewAWSError("PolicyTypeNotEnabledException",
		"Policy type is not enabled.", http.StatusBadRequest)
}

// TagResource adds tags to a resource.
func (s *Store) TagResource(resourceID string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if acct, ok := s.accounts[resourceID]; ok {
		acct.Tags = mergeTags(acct.Tags, tags)
		return nil
	}
	if ou, ok := s.ous[resourceID]; ok {
		ou.Tags = mergeTags(ou.Tags, tags)
		return nil
	}
	if policy, ok := s.policies[resourceID]; ok {
		policy.Tags = mergeTags(policy.Tags, tags)
		return nil
	}
	return service.NewAWSError("TargetNotFoundException",
		fmt.Sprintf("Resource %s not found.", resourceID), http.StatusBadRequest)
}

// UntagResource removes tags from a resource.
func (s *Store) UntagResource(resourceID string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if acct, ok := s.accounts[resourceID]; ok {
		acct.Tags = removeTags(acct.Tags, tagKeys)
		return nil
	}
	if ou, ok := s.ous[resourceID]; ok {
		ou.Tags = removeTags(ou.Tags, tagKeys)
		return nil
	}
	if policy, ok := s.policies[resourceID]; ok {
		policy.Tags = removeTags(policy.Tags, tagKeys)
		return nil
	}
	return service.NewAWSError("TargetNotFoundException",
		fmt.Sprintf("Resource %s not found.", resourceID), http.StatusBadRequest)
}

// ListTagsForResource returns tags for a resource.
func (s *Store) ListTagsForResource(resourceID string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if acct, ok := s.accounts[resourceID]; ok {
		return acct.Tags, nil
	}
	if ou, ok := s.ous[resourceID]; ok {
		return ou.Tags, nil
	}
	if policy, ok := s.policies[resourceID]; ok {
		return policy.Tags, nil
	}
	return nil, service.NewAWSError("TargetNotFoundException",
		fmt.Sprintf("Resource %s not found.", resourceID), http.StatusBadRequest)
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

func removeTags(existing []Tag, keys []string) []Tag {
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	out := make([]Tag, 0, len(existing))
	for _, t := range existing {
		if !keySet[t.Key] {
			out = append(out, t)
		}
	}
	return out
}
