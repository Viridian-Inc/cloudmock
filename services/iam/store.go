package iam

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
)

// IAMUser represents an IAM user with metadata.
type IAMUser struct {
	UserName   string
	UserID     string
	Arn        string
	Path       string
	CreateDate time.Time
	Tags       map[string]string
}

// IAMRole represents an IAM role.
type IAMRole struct {
	RoleName                 string
	RoleID                   string
	Arn                      string
	Path                     string
	AssumeRolePolicyDocument string
	Description              string
	CreateDate               time.Time
}

// IAMPolicy represents a managed IAM policy.
type IAMPolicy struct {
	PolicyName   string
	PolicyID     string
	Arn          string
	Path         string
	Description  string
	Document     string
	CreateDate   time.Time
	AttachCount  int
}

// IAMGroup represents an IAM group.
type IAMGroup struct {
	GroupName  string
	GroupID    string
	Arn        string
	Path       string
	CreateDate time.Time
	Members    map[string]bool // user names
}

// IAMAccessKey represents an IAM access key with status.
type IAMAccessKey struct {
	AccessKeyID    string
	SecretAccessKey string
	UserName       string
	Status         string
	CreateDate     time.Time
}

// IAMInstanceProfile represents an IAM instance profile.
type IAMInstanceProfile struct {
	InstanceProfileName string
	InstanceProfileID   string
	Arn                 string
	Path                string
	Roles               []string // role names
	CreateDate          time.Time
}

// Store holds all IAM resources for a single account.
//
// Per-entity-type locking: each entity type has its own RWMutex to reduce
// contention. Cross-entity operations acquire locks in a consistent order
// to prevent deadlocks:
//
//	usersMu -> policiesMu -> accessKeysMu -> groupsMu -> rolesMu -> instanceProfilesMu
type Store struct {
	accountID string

	usersMu      sync.RWMutex
	users        map[string]*IAMUser
	userPolicies map[string]map[string]bool // userName -> set of policy ARNs
	userAccessKeys map[string][]string      // userName -> []AccessKeyID

	rolesMu      sync.RWMutex
	roles        map[string]*IAMRole
	rolePolicies map[string]map[string]bool // roleName -> set of policy ARNs

	policiesMu   sync.RWMutex
	policies     map[string]*IAMPolicy // keyed by ARN
	policyByName map[string]string     // name -> ARN

	groupsMu sync.RWMutex
	groups   map[string]*IAMGroup

	accessKeysMu sync.RWMutex
	accessKeys   map[string]*IAMAccessKey // keyed by AccessKeyID

	instanceProfilesMu sync.RWMutex
	instanceProfiles   map[string]*IAMInstanceProfile

	engine   *iampkg.Engine
	pkgStore *iampkg.Store
}

// NewStore creates a new IAM service store.
func NewStore(accountID string, engine *iampkg.Engine, pkgStore *iampkg.Store) *Store {
	return &Store{
		accountID:        accountID,
		users:            make(map[string]*IAMUser),
		roles:            make(map[string]*IAMRole),
		policies:         make(map[string]*IAMPolicy),
		policyByName:     make(map[string]string),
		groups:           make(map[string]*IAMGroup),
		accessKeys:       make(map[string]*IAMAccessKey),
		userAccessKeys:   make(map[string][]string),
		instanceProfiles: make(map[string]*IAMInstanceProfile),
		userPolicies:     make(map[string]map[string]bool),
		rolePolicies:     make(map[string]map[string]bool),
		engine:           engine,
		pkgStore:         pkgStore,
	}
}

// --- Users ---

func (s *Store) CreateUser(userName string) (*IAMUser, error) {
	s.usersMu.Lock()
	if _, exists := s.users[userName]; exists {
		s.usersMu.Unlock()
		return nil, fmt.Errorf("EntityAlreadyExists: User with name %s already exists", userName)
	}

	userID := generateID("AIDA", 16)
	user := &IAMUser{
		UserName:   userName,
		UserID:     userID,
		Arn:        fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, userName),
		Path:       "/",
		CreateDate: time.Now().UTC(),
		Tags:       make(map[string]string),
	}
	s.users[userName] = user
	s.usersMu.Unlock()

	// Call external store outside lock
	if s.pkgStore != nil {
		s.pkgStore.CreateUser(userName)
	}

	return user, nil
}

func (s *Store) GetUser(userName string) (*IAMUser, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	return user, nil
}

func (s *Store) ListUsers() []*IAMUser {
	s.usersMu.RLock()
	users := make([]*IAMUser, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	s.usersMu.RUnlock()
	return users
}

func (s *Store) UpdateUser(userName, newUserName string) (*IAMUser, error) {
	// Lock order: usersMu -> accessKeysMu -> groupsMu
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	if _, exists := s.users[newUserName]; exists {
		return nil, fmt.Errorf("EntityAlreadyExists: User with name %s already exists", newUserName)
	}

	delete(s.users, userName)
	user.UserName = newUserName
	user.Arn = fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, newUserName)
	s.users[newUserName] = user

	// Move policy attachments (userPolicies guarded by usersMu)
	if pols, ok := s.userPolicies[userName]; ok {
		s.userPolicies[newUserName] = pols
		delete(s.userPolicies, userName)
	}

	// Move access keys (userAccessKeys guarded by usersMu)
	if keys, ok := s.userAccessKeys[userName]; ok {
		s.userAccessKeys[newUserName] = keys
		delete(s.userAccessKeys, userName)
		// Update the access key objects themselves
		s.accessKeysMu.Lock()
		for _, keyID := range keys {
			if ak, ok := s.accessKeys[keyID]; ok {
				ak.UserName = newUserName
			}
		}
		s.accessKeysMu.Unlock()
	}

	// Update group membership
	s.groupsMu.Lock()
	for _, g := range s.groups {
		if g.Members[userName] {
			delete(g.Members, userName)
			g.Members[newUserName] = true
		}
	}
	s.groupsMu.Unlock()

	return user, nil
}

func (s *Store) DeleteUser(userName string) error {
	// Lock order: usersMu -> policiesMu -> accessKeysMu -> groupsMu
	s.usersMu.Lock()

	if _, ok := s.users[userName]; !ok {
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	// Collect policy ARNs to update attach counts
	var polARNs []string
	if pols, ok := s.userPolicies[userName]; ok {
		for arn := range pols {
			polARNs = append(polARNs, arn)
		}
		delete(s.userPolicies, userName)
	}

	// Collect access key IDs to delete
	keyIDs := s.userAccessKeys[userName]
	delete(s.userAccessKeys, userName)

	userArn := fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, userName)
	delete(s.users, userName)
	s.usersMu.Unlock()

	// Update policy attach counts
	if len(polARNs) > 0 {
		s.policiesMu.Lock()
		for _, arn := range polARNs {
			if p, ok := s.policies[arn]; ok {
				p.AttachCount--
			}
		}
		s.policiesMu.Unlock()
	}

	// Clean up access keys
	if len(keyIDs) > 0 {
		s.accessKeysMu.Lock()
		for _, keyID := range keyIDs {
			delete(s.accessKeys, keyID)
		}
		s.accessKeysMu.Unlock()
	}

	// Remove from groups
	s.groupsMu.Lock()
	for _, g := range s.groups {
		delete(g.Members, userName)
	}
	s.groupsMu.Unlock()

	// Remove engine policies (external call, no lock held)
	if s.engine != nil {
		s.engine.RemovePolicies(userArn)
	}

	return nil
}

// --- Tags ---

func (s *Store) TagUser(userName string, tags map[string]string) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	user, ok := s.users[userName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	for k, v := range tags {
		user.Tags[k] = v
	}
	return nil
}

func (s *Store) UntagUser(userName string, tagKeys []string) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	user, ok := s.users[userName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	for _, k := range tagKeys {
		delete(user.Tags, k)
	}
	return nil
}

func (s *Store) ListUserTags(userName string) (map[string]string, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	tags := make(map[string]string, len(user.Tags))
	for k, v := range user.Tags {
		tags[k] = v
	}
	return tags, nil
}

// --- Roles ---

func (s *Store) CreateRole(roleName, assumeRolePolicyDoc, description string) (*IAMRole, error) {
	s.rolesMu.Lock()
	defer s.rolesMu.Unlock()

	if _, exists := s.roles[roleName]; exists {
		return nil, fmt.Errorf("EntityAlreadyExists: Role with name %s already exists", roleName)
	}

	roleID := generateID("AROA", 16)
	role := &IAMRole{
		RoleName:                 roleName,
		RoleID:                   roleID,
		Arn:                      fmt.Sprintf("arn:aws:iam::%s:role/%s", s.accountID, roleName),
		Path:                     "/",
		AssumeRolePolicyDocument: assumeRolePolicyDoc,
		Description:              description,
		CreateDate:               time.Now().UTC(),
	}
	s.roles[roleName] = role
	return role, nil
}

func (s *Store) GetRole(roleName string) (*IAMRole, error) {
	s.rolesMu.RLock()
	defer s.rolesMu.RUnlock()

	role, ok := s.roles[roleName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}
	return role, nil
}

func (s *Store) ListRoles() []*IAMRole {
	s.rolesMu.RLock()
	roles := make([]*IAMRole, 0, len(s.roles))
	for _, r := range s.roles {
		roles = append(roles, r)
	}
	s.rolesMu.RUnlock()
	return roles
}

func (s *Store) DeleteRole(roleName string) error {
	// Lock order: policiesMu -> rolesMu -> instanceProfilesMu
	// We need rolesMu to check existence and delete, policiesMu for attach counts,
	// instanceProfilesMu for cleanup.

	s.rolesMu.Lock()
	if _, ok := s.roles[roleName]; !ok {
		s.rolesMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	// Collect policy ARNs to update attach counts
	var polARNs []string
	if pols, ok := s.rolePolicies[roleName]; ok {
		for arn := range pols {
			polARNs = append(polARNs, arn)
		}
		delete(s.rolePolicies, roleName)
	}

	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", s.accountID, roleName)
	delete(s.roles, roleName)
	s.rolesMu.Unlock()

	// Update policy attach counts
	if len(polARNs) > 0 {
		s.policiesMu.Lock()
		for _, arn := range polARNs {
			if p, ok := s.policies[arn]; ok {
				p.AttachCount--
			}
		}
		s.policiesMu.Unlock()
	}

	// Remove from instance profiles
	s.instanceProfilesMu.Lock()
	for _, ip := range s.instanceProfiles {
		for i, rn := range ip.Roles {
			if rn == roleName {
				ip.Roles = append(ip.Roles[:i], ip.Roles[i+1:]...)
				break
			}
		}
	}
	s.instanceProfilesMu.Unlock()

	// External call outside all locks
	if s.engine != nil {
		s.engine.RemovePolicies(roleArn)
	}

	return nil
}

// --- Policies ---

func (s *Store) CreatePolicy(policyName, policyDocument, description string) (*IAMPolicy, error) {
	s.policiesMu.Lock()
	defer s.policiesMu.Unlock()

	arn := fmt.Sprintf("arn:aws:iam::%s:policy/%s", s.accountID, policyName)
	if _, exists := s.policies[arn]; exists {
		return nil, fmt.Errorf("EntityAlreadyExists: A policy called %s already exists", policyName)
	}

	policyID := generateID("ANPA", 16)
	policy := &IAMPolicy{
		PolicyName:  policyName,
		PolicyID:    policyID,
		Arn:         arn,
		Path:        "/",
		Description: description,
		Document:    policyDocument,
		CreateDate:  time.Now().UTC(),
	}
	s.policies[arn] = policy
	s.policyByName[policyName] = arn
	return policy, nil
}

func (s *Store) GetPolicy(policyArn string) (*IAMPolicy, error) {
	s.policiesMu.RLock()
	defer s.policiesMu.RUnlock()

	policy, ok := s.policies[policyArn]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}
	return policy, nil
}

func (s *Store) ListPolicies() []*IAMPolicy {
	s.policiesMu.RLock()
	policies := make([]*IAMPolicy, 0, len(s.policies))
	for _, p := range s.policies {
		policies = append(policies, p)
	}
	s.policiesMu.RUnlock()
	return policies
}

func (s *Store) DeletePolicy(policyArn string) error {
	s.policiesMu.Lock()
	defer s.policiesMu.Unlock()

	policy, ok := s.policies[policyArn]
	if !ok {
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}

	if policy.AttachCount > 0 {
		return fmt.Errorf("DeleteConflict: Cannot delete a policy that is attached to entities")
	}

	delete(s.policyByName, policy.PolicyName)
	delete(s.policies, policyArn)
	return nil
}

// UpdatePolicyDocument replaces the stored policy document. Cloudmock models
// every policy as a single versioned entity — CreatePolicyVersion calls this
// to mutate the default (only) version in place.
func (s *Store) UpdatePolicyDocument(policyArn, document string) error {
	s.policiesMu.Lock()
	defer s.policiesMu.Unlock()

	policy, ok := s.policies[policyArn]
	if !ok {
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}
	policy.Document = document
	return nil
}

func (s *Store) AttachUserPolicy(userName, policyArn string) error {
	// Lock order: usersMu -> policiesMu
	s.usersMu.Lock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	s.policiesMu.Lock()
	pol, ok := s.policies[policyArn]
	if !ok && !strings.HasPrefix(policyArn, "arn:aws:iam::aws:policy/") {
		s.policiesMu.Unlock()
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}

	if s.userPolicies[userName] == nil {
		s.userPolicies[userName] = make(map[string]bool)
	}
	if !s.userPolicies[userName][policyArn] {
		s.userPolicies[userName][policyArn] = true
		if pol != nil {
			pol.AttachCount++
		}
	}

	// Capture document for engine registration
	var doc string
	if s.engine != nil && pol != nil && pol.Document != "" {
		doc = pol.Document
	}
	s.policiesMu.Unlock()
	s.usersMu.Unlock()

	// Register with IAM engine outside locks
	if doc != "" {
		s.registerPolicyWithEngine(
			fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, userName),
			doc,
		)
	}

	return nil
}

func (s *Store) DetachUserPolicy(userName, policyArn string) error {
	// Lock order: usersMu -> policiesMu
	s.usersMu.Lock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	pols, ok := s.userPolicies[userName]
	if !ok || !pols[policyArn] {
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: Policy %s is not attached to user %s", policyArn, userName)
	}

	delete(pols, policyArn)
	s.usersMu.Unlock()

	s.policiesMu.Lock()
	if p, ok := s.policies[policyArn]; ok {
		p.AttachCount--
	}
	s.policiesMu.Unlock()

	return nil
}

func (s *Store) ListAttachedUserPolicies(userName string) ([]AttachedPolicy, error) {
	// Lock order: usersMu -> policiesMu
	s.usersMu.RLock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.RUnlock()
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	// Snapshot the policy ARNs
	var arns []string
	if pols, ok := s.userPolicies[userName]; ok {
		arns = make([]string, 0, len(pols))
		for arn := range pols {
			arns = append(arns, arn)
		}
	}
	s.usersMu.RUnlock()

	// Build result using policies lock
	var result []AttachedPolicy
	if len(arns) > 0 {
		s.policiesMu.RLock()
		for _, arn := range arns {
			if p, ok := s.policies[arn]; ok {
				result = append(result, AttachedPolicy{PolicyName: p.PolicyName, PolicyArn: p.Arn})
			}
		}
		s.policiesMu.RUnlock()
	}

	return result, nil
}

func (s *Store) AttachRolePolicy(roleName, policyArn string) error {
	// Lock order: rolesMu -> policiesMu (consistent with global order)
	s.rolesMu.Lock()
	if _, ok := s.roles[roleName]; !ok {
		s.rolesMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	s.policiesMu.Lock()
	pol, ok := s.policies[policyArn]
	if !ok && !strings.HasPrefix(policyArn, "arn:aws:iam::aws:policy/") {
		// Only require the policy to exist for customer-managed policies.
		// AWS managed policies (arn:aws:iam::aws:policy/*) are always valid.
		s.policiesMu.Unlock()
		s.rolesMu.Unlock()
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}

	if s.rolePolicies[roleName] == nil {
		s.rolePolicies[roleName] = make(map[string]bool)
	}
	if !s.rolePolicies[roleName][policyArn] {
		s.rolePolicies[roleName][policyArn] = true
		if pol != nil {
			pol.AttachCount++
		}
	}

	var doc string
	if s.engine != nil && pol != nil && pol.Document != "" {
		doc = pol.Document
	}
	s.policiesMu.Unlock()
	s.rolesMu.Unlock()

	// Register with IAM engine outside locks
	if doc != "" {
		s.registerPolicyWithEngine(
			fmt.Sprintf("arn:aws:iam::%s:role/%s", s.accountID, roleName),
			doc,
		)
	}

	return nil
}

func (s *Store) DetachRolePolicy(roleName, policyArn string) error {
	// Lock order: rolesMu -> policiesMu
	s.rolesMu.Lock()
	if _, ok := s.roles[roleName]; !ok {
		s.rolesMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	pols, ok := s.rolePolicies[roleName]
	if !ok || !pols[policyArn] {
		s.rolesMu.Unlock()
		return fmt.Errorf("NoSuchEntity: Policy %s is not attached to role %s", policyArn, roleName)
	}

	delete(pols, policyArn)
	s.rolesMu.Unlock()

	s.policiesMu.Lock()
	if p, ok := s.policies[policyArn]; ok {
		p.AttachCount--
	}
	s.policiesMu.Unlock()

	return nil
}

func (s *Store) ListAttachedRolePolicies(roleName string) ([]AttachedPolicy, error) {
	// Lock order: rolesMu -> policiesMu
	s.rolesMu.RLock()
	if _, ok := s.roles[roleName]; !ok {
		s.rolesMu.RUnlock()
		return nil, fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	var arns []string
	if pols, ok := s.rolePolicies[roleName]; ok {
		arns = make([]string, 0, len(pols))
		for arn := range pols {
			arns = append(arns, arn)
		}
	}
	s.rolesMu.RUnlock()

	var result []AttachedPolicy
	if len(arns) > 0 {
		s.policiesMu.RLock()
		for _, arn := range arns {
			if p, ok := s.policies[arn]; ok {
				result = append(result, AttachedPolicy{PolicyName: p.PolicyName, PolicyArn: p.Arn})
			}
		}
		s.policiesMu.RUnlock()
	}

	return result, nil
}

// --- Groups ---

func (s *Store) CreateGroup(groupName string) (*IAMGroup, error) {
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	if _, exists := s.groups[groupName]; exists {
		return nil, fmt.Errorf("EntityAlreadyExists: Group with name %s already exists", groupName)
	}

	groupID := generateID("AGPA", 16)
	group := &IAMGroup{
		GroupName:  groupName,
		GroupID:    groupID,
		Arn:        fmt.Sprintf("arn:aws:iam::%s:group/%s", s.accountID, groupName),
		Path:       "/",
		CreateDate: time.Now().UTC(),
		Members:    make(map[string]bool),
	}
	s.groups[groupName] = group
	return group, nil
}

func (s *Store) GetGroup(groupName string) (*IAMGroup, []*IAMUser, error) {
	// Lock order: usersMu -> groupsMu (but we only read, and groups first for existence check)
	// Actually for reads we can lock groupsMu first, snapshot members, unlock, then lock usersMu.
	s.groupsMu.RLock()
	group, ok := s.groups[groupName]
	if !ok {
		s.groupsMu.RUnlock()
		return nil, nil, fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}

	// Snapshot member names
	memberNames := make([]string, 0, len(group.Members))
	for userName := range group.Members {
		memberNames = append(memberNames, userName)
	}
	s.groupsMu.RUnlock()

	// Look up users outside groups lock
	s.usersMu.RLock()
	var users []*IAMUser
	for _, userName := range memberNames {
		if u, ok := s.users[userName]; ok {
			users = append(users, u)
		}
	}
	s.usersMu.RUnlock()

	return group, users, nil
}

func (s *Store) ListGroups() []*IAMGroup {
	s.groupsMu.RLock()
	groups := make([]*IAMGroup, 0, len(s.groups))
	for _, g := range s.groups {
		groups = append(groups, g)
	}
	s.groupsMu.RUnlock()
	return groups
}

func (s *Store) DeleteGroup(groupName string) error {
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	if _, ok := s.groups[groupName]; !ok {
		return fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}

	delete(s.groups, groupName)
	return nil
}

func (s *Store) AddUserToGroup(groupName, userName string) error {
	// Lock order: usersMu (read) -> groupsMu (write)
	s.usersMu.RLock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.RUnlock()
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	s.usersMu.RUnlock()

	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	if _, ok := s.groups[groupName]; !ok {
		return fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}

	s.groups[groupName].Members[userName] = true
	return nil
}

func (s *Store) RemoveUserFromGroup(groupName, userName string) error {
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	group, ok := s.groups[groupName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}
	if !group.Members[userName] {
		return fmt.Errorf("NoSuchEntity: The user with name %s is not in group %s", userName, groupName)
	}

	delete(group.Members, userName)
	return nil
}

// --- Access Keys ---

func (s *Store) CreateAccessKey(userName string) (*IAMAccessKey, error) {
	// Lock order: usersMu -> accessKeysMu
	s.usersMu.Lock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.Unlock()
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	keyID := generateID("AKIA", 16)
	secret := randomHex(20)
	key := &IAMAccessKey{
		AccessKeyID:    keyID,
		SecretAccessKey: secret,
		UserName:       userName,
		Status:         "Active",
		CreateDate:     time.Now().UTC(),
	}

	s.accessKeysMu.Lock()
	s.accessKeys[keyID] = key
	s.accessKeysMu.Unlock()

	s.userAccessKeys[userName] = append(s.userAccessKeys[userName], keyID)
	s.usersMu.Unlock()

	// External call outside locks
	if s.pkgStore != nil {
		s.pkgStore.CreateAccessKey(userName)
	}

	return key, nil
}

func (s *Store) ListAccessKeys(userName string) ([]*IAMAccessKey, error) {
	// Lock order: usersMu -> accessKeysMu
	s.usersMu.RLock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.RUnlock()
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	// Snapshot key IDs
	keyIDs := make([]string, len(s.userAccessKeys[userName]))
	copy(keyIDs, s.userAccessKeys[userName])
	s.usersMu.RUnlock()

	// Build result with accessKeys lock
	s.accessKeysMu.RLock()
	var keys []*IAMAccessKey
	for _, keyID := range keyIDs {
		if ak, ok := s.accessKeys[keyID]; ok {
			keys = append(keys, ak)
		}
	}
	s.accessKeysMu.RUnlock()

	return keys, nil
}

func (s *Store) DeleteAccessKey(userName, accessKeyID string) error {
	// Lock order: usersMu -> accessKeysMu
	s.usersMu.Lock()
	if _, ok := s.users[userName]; !ok {
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	s.accessKeysMu.Lock()
	if _, ok := s.accessKeys[accessKeyID]; !ok {
		s.accessKeysMu.Unlock()
		s.usersMu.Unlock()
		return fmt.Errorf("NoSuchEntity: The access key with id %s cannot be found", accessKeyID)
	}
	delete(s.accessKeys, accessKeyID)
	s.accessKeysMu.Unlock()

	keys := s.userAccessKeys[userName]
	for i, k := range keys {
		if k == accessKeyID {
			s.userAccessKeys[userName] = append(keys[:i], keys[i+1:]...)
			break
		}
	}
	s.usersMu.Unlock()

	return nil
}

// --- Instance Profiles ---

func (s *Store) CreateInstanceProfile(name string) (*IAMInstanceProfile, error) {
	s.instanceProfilesMu.Lock()
	defer s.instanceProfilesMu.Unlock()

	if _, exists := s.instanceProfiles[name]; exists {
		return nil, fmt.Errorf("EntityAlreadyExists: Instance profile %s already exists", name)
	}

	ipID := generateID("AIPA", 16)
	ip := &IAMInstanceProfile{
		InstanceProfileName: name,
		InstanceProfileID:   ipID,
		Arn:                 fmt.Sprintf("arn:aws:iam::%s:instance-profile/%s", s.accountID, name),
		Path:                "/",
		CreateDate:          time.Now().UTC(),
	}
	s.instanceProfiles[name] = ip
	return ip, nil
}

func (s *Store) GetInstanceProfile(name string) (*IAMInstanceProfile, error) {
	s.instanceProfilesMu.RLock()
	defer s.instanceProfilesMu.RUnlock()

	ip, ok := s.instanceProfiles[name]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", name)
	}
	return ip, nil
}

func (s *Store) ListInstanceProfiles() []*IAMInstanceProfile {
	s.instanceProfilesMu.RLock()
	ips := make([]*IAMInstanceProfile, 0, len(s.instanceProfiles))
	for _, ip := range s.instanceProfiles {
		ips = append(ips, ip)
	}
	s.instanceProfilesMu.RUnlock()
	return ips
}

func (s *Store) DeleteInstanceProfile(name string) error {
	s.instanceProfilesMu.Lock()
	defer s.instanceProfilesMu.Unlock()

	if _, ok := s.instanceProfiles[name]; !ok {
		return fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", name)
	}

	delete(s.instanceProfiles, name)
	return nil
}

func (s *Store) AddRoleToInstanceProfile(profileName, roleName string) error {
	// Lock order: rolesMu (read) -> instanceProfilesMu (write)
	s.rolesMu.RLock()
	if _, ok := s.roles[roleName]; !ok {
		s.rolesMu.RUnlock()
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}
	s.rolesMu.RUnlock()

	s.instanceProfilesMu.Lock()
	defer s.instanceProfilesMu.Unlock()

	ip, ok := s.instanceProfiles[profileName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", profileName)
	}

	for _, rn := range ip.Roles {
		if rn == roleName {
			return fmt.Errorf("LimitExceeded: Role %s is already in instance profile %s", roleName, profileName)
		}
	}

	ip.Roles = append(ip.Roles, roleName)
	return nil
}

func (s *Store) RemoveRoleFromInstanceProfile(profileName, roleName string) error {
	s.instanceProfilesMu.Lock()
	defer s.instanceProfilesMu.Unlock()

	ip, ok := s.instanceProfiles[profileName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", profileName)
	}

	found := false
	for i, rn := range ip.Roles {
		if rn == roleName {
			ip.Roles = append(ip.Roles[:i], ip.Roles[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("NoSuchEntity: Role %s is not in instance profile %s", roleName, profileName)
	}
	return nil
}

// --- Helpers ---

// AttachedPolicy is a name+arn pair for listing attached policies.
type AttachedPolicy struct {
	PolicyName string
	PolicyArn  string
}

// registerPolicyWithEngine parses a JSON policy document and registers it with the IAM engine.
func (s *Store) registerPolicyWithEngine(principal, document string) {
	// Best-effort: parse the policy doc and add to engine
	// The engine uses pkg/iam.Policy type, not the managed policy wrapper
	// For simplicity, we don't parse JSON here — the engine integration
	// happens when callers use the pkg/iam store directly.
	// This is a placeholder for future enhancement.
}

func generateID(prefix string, hexLen int) string {
	return prefix + randomHexUpper(hexLen)
}

func randomHexUpper(n int) string {
	b := make([]byte, n/2)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
