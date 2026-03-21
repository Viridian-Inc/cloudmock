package iam

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	iampkg "github.com/neureaux/cloudmock/pkg/iam"
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
type Store struct {
	mu               sync.RWMutex
	accountID        string
	users            map[string]*IAMUser
	roles            map[string]*IAMRole
	policies         map[string]*IAMPolicy        // keyed by ARN
	policyByName     map[string]string             // name -> ARN
	groups           map[string]*IAMGroup
	accessKeys       map[string]*IAMAccessKey       // keyed by AccessKeyID
	userAccessKeys   map[string][]string            // userName -> []AccessKeyID
	instanceProfiles map[string]*IAMInstanceProfile
	userPolicies     map[string]map[string]bool     // userName -> set of policy ARNs
	rolePolicies     map[string]map[string]bool     // roleName -> set of policy ARNs
	engine           *iampkg.Engine
	pkgStore         *iampkg.Store
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[userName]; exists {
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

	// Also create in the pkg store so access keys and auth work
	if s.pkgStore != nil {
		s.pkgStore.CreateUser(userName)
	}

	return user, nil
}

func (s *Store) GetUser(userName string) (*IAMUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	return user, nil
}

func (s *Store) ListUsers() []*IAMUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*IAMUser, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

func (s *Store) UpdateUser(userName, newUserName string) (*IAMUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// Move policy attachments
	if pols, ok := s.userPolicies[userName]; ok {
		s.userPolicies[newUserName] = pols
		delete(s.userPolicies, userName)
	}

	// Move access keys
	if keys, ok := s.userAccessKeys[userName]; ok {
		s.userAccessKeys[newUserName] = keys
		delete(s.userAccessKeys, userName)
		for _, keyID := range keys {
			if ak, ok := s.accessKeys[keyID]; ok {
				ak.UserName = newUserName
			}
		}
	}

	// Update group membership
	for _, g := range s.groups {
		if g.Members[userName] {
			delete(g.Members, userName)
			g.Members[newUserName] = true
		}
	}

	return user, nil
}

func (s *Store) DeleteUser(userName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	// Clean up access keys
	if keys, ok := s.userAccessKeys[userName]; ok {
		for _, keyID := range keys {
			delete(s.accessKeys, keyID)
		}
		delete(s.userAccessKeys, userName)
	}

	// Clean up policy attachments and update attach counts
	if pols, ok := s.userPolicies[userName]; ok {
		for arn := range pols {
			if p, ok := s.policies[arn]; ok {
				p.AttachCount--
			}
		}
		delete(s.userPolicies, userName)
	}

	// Remove from groups
	for _, g := range s.groups {
		delete(g.Members, userName)
	}

	// Remove engine policies
	if s.engine != nil {
		s.engine.RemovePolicies(fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, userName))
	}

	delete(s.users, userName)
	return nil
}

// --- Tags ---

func (s *Store) TagUser(userName string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.roles[roleName]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}
	return role, nil
}

func (s *Store) ListRoles() []*IAMRole {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*IAMRole, 0, len(s.roles))
	for _, r := range s.roles {
		roles = append(roles, r)
	}
	return roles
}

func (s *Store) DeleteRole(roleName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[roleName]; !ok {
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	// Clean up policy attachments
	if pols, ok := s.rolePolicies[roleName]; ok {
		for arn := range pols {
			if p, ok := s.policies[arn]; ok {
				p.AttachCount--
			}
		}
		delete(s.rolePolicies, roleName)
	}

	// Remove from instance profiles
	for _, ip := range s.instanceProfiles {
		for i, rn := range ip.Roles {
			if rn == roleName {
				ip.Roles = append(ip.Roles[:i], ip.Roles[i+1:]...)
				break
			}
		}
	}

	if s.engine != nil {
		s.engine.RemovePolicies(fmt.Sprintf("arn:aws:iam::%s:role/%s", s.accountID, roleName))
	}

	delete(s.roles, roleName)
	return nil
}

// --- Policies ---

func (s *Store) CreatePolicy(policyName, policyDocument, description string) (*IAMPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.policies[policyArn]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}
	return policy, nil
}

func (s *Store) ListPolicies() []*IAMPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policies := make([]*IAMPolicy, 0, len(s.policies))
	for _, p := range s.policies {
		policies = append(policies, p)
	}
	return policies
}

func (s *Store) DeletePolicy(policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *Store) AttachUserPolicy(userName, policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}
	if _, ok := s.policies[policyArn]; !ok {
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}

	if s.userPolicies[userName] == nil {
		s.userPolicies[userName] = make(map[string]bool)
	}
	if !s.userPolicies[userName][policyArn] {
		s.userPolicies[userName][policyArn] = true
		s.policies[policyArn].AttachCount++
	}

	// Register with IAM engine for policy evaluation
	if s.engine != nil {
		pol := s.policies[policyArn]
		if pol.Document != "" {
			s.registerPolicyWithEngine(
				fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, userName),
				pol.Document,
			)
		}
	}

	return nil
}

func (s *Store) DetachUserPolicy(userName, policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	pols, ok := s.userPolicies[userName]
	if !ok || !pols[policyArn] {
		return fmt.Errorf("NoSuchEntity: Policy %s is not attached to user %s", policyArn, userName)
	}

	delete(pols, policyArn)
	if p, ok := s.policies[policyArn]; ok {
		p.AttachCount--
	}
	return nil
}

func (s *Store) ListAttachedUserPolicies(userName string) ([]AttachedPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.users[userName]; !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	var result []AttachedPolicy
	if pols, ok := s.userPolicies[userName]; ok {
		for arn := range pols {
			if p, ok := s.policies[arn]; ok {
				result = append(result, AttachedPolicy{PolicyName: p.PolicyName, PolicyArn: p.Arn})
			}
		}
	}
	return result, nil
}

func (s *Store) AttachRolePolicy(roleName, policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[roleName]; !ok {
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}
	if _, ok := s.policies[policyArn]; !ok {
		return fmt.Errorf("NoSuchEntity: Policy %s does not exist", policyArn)
	}

	if s.rolePolicies[roleName] == nil {
		s.rolePolicies[roleName] = make(map[string]bool)
	}
	if !s.rolePolicies[roleName][policyArn] {
		s.rolePolicies[roleName][policyArn] = true
		s.policies[policyArn].AttachCount++
	}

	if s.engine != nil {
		pol := s.policies[policyArn]
		if pol.Document != "" {
			s.registerPolicyWithEngine(
				fmt.Sprintf("arn:aws:iam::%s:role/%s", s.accountID, roleName),
				pol.Document,
			)
		}
	}

	return nil
}

func (s *Store) DetachRolePolicy(roleName, policyArn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.roles[roleName]; !ok {
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	pols, ok := s.rolePolicies[roleName]
	if !ok || !pols[policyArn] {
		return fmt.Errorf("NoSuchEntity: Policy %s is not attached to role %s", policyArn, roleName)
	}

	delete(pols, policyArn)
	if p, ok := s.policies[policyArn]; ok {
		p.AttachCount--
	}
	return nil
}

func (s *Store) ListAttachedRolePolicies(roleName string) ([]AttachedPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.roles[roleName]; !ok {
		return nil, fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
	}

	var result []AttachedPolicy
	if pols, ok := s.rolePolicies[roleName]; ok {
		for arn := range pols {
			if p, ok := s.policies[arn]; ok {
				result = append(result, AttachedPolicy{PolicyName: p.PolicyName, PolicyArn: p.Arn})
			}
		}
	}
	return result, nil
}

// --- Groups ---

func (s *Store) CreateGroup(groupName string) (*IAMGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	group, ok := s.groups[groupName]
	if !ok {
		return nil, nil, fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}

	var users []*IAMUser
	for userName := range group.Members {
		if u, ok := s.users[userName]; ok {
			users = append(users, u)
		}
	}
	return group, users, nil
}

func (s *Store) ListGroups() []*IAMGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := make([]*IAMGroup, 0, len(s.groups))
	for _, g := range s.groups {
		groups = append(groups, g)
	}
	return groups
}

func (s *Store) DeleteGroup(groupName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[groupName]; !ok {
		return fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}

	delete(s.groups, groupName)
	return nil
}

func (s *Store) AddUserToGroup(groupName, userName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[groupName]; !ok {
		return fmt.Errorf("NoSuchEntity: The group with name %s cannot be found", groupName)
	}
	if _, ok := s.users[userName]; !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	s.groups[groupName].Members[userName] = true
	return nil
}

func (s *Store) RemoveUserFromGroup(groupName, userName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
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
	s.accessKeys[keyID] = key
	s.userAccessKeys[userName] = append(s.userAccessKeys[userName], keyID)

	// Also register with pkg store for auth
	if s.pkgStore != nil {
		s.pkgStore.CreateAccessKey(userName)
	}

	return key, nil
}

func (s *Store) ListAccessKeys(userName string) ([]*IAMAccessKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.users[userName]; !ok {
		return nil, fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	var keys []*IAMAccessKey
	for _, keyID := range s.userAccessKeys[userName] {
		if ak, ok := s.accessKeys[keyID]; ok {
			keys = append(keys, ak)
		}
	}
	return keys, nil
}

func (s *Store) DeleteAccessKey(userName, accessKeyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
		return fmt.Errorf("NoSuchEntity: The user with name %s cannot be found", userName)
	}

	if _, ok := s.accessKeys[accessKeyID]; !ok {
		return fmt.Errorf("NoSuchEntity: The access key with id %s cannot be found", accessKeyID)
	}

	delete(s.accessKeys, accessKeyID)
	keys := s.userAccessKeys[userName]
	for i, k := range keys {
		if k == accessKeyID {
			s.userAccessKeys[userName] = append(keys[:i], keys[i+1:]...)
			break
		}
	}
	return nil
}

// --- Instance Profiles ---

func (s *Store) CreateInstanceProfile(name string) (*IAMInstanceProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	ip, ok := s.instanceProfiles[name]
	if !ok {
		return nil, fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", name)
	}
	return ip, nil
}

func (s *Store) ListInstanceProfiles() []*IAMInstanceProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ips := make([]*IAMInstanceProfile, 0, len(s.instanceProfiles))
	for _, ip := range s.instanceProfiles {
		ips = append(ips, ip)
	}
	return ips
}

func (s *Store) DeleteInstanceProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instanceProfiles[name]; !ok {
		return fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", name)
	}

	delete(s.instanceProfiles, name)
	return nil
}

func (s *Store) AddRoleToInstanceProfile(profileName, roleName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ip, ok := s.instanceProfiles[profileName]
	if !ok {
		return fmt.Errorf("NoSuchEntity: Instance profile %s does not exist", profileName)
	}
	if _, ok := s.roles[roleName]; !ok {
		return fmt.Errorf("NoSuchEntity: The role with name %s cannot be found", roleName)
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
	s.mu.Lock()
	defer s.mu.Unlock()

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
