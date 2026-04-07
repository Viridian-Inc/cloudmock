package transfer

import (
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// Server represents an AWS Transfer Family server.
type Server struct {
	ServerID              string
	Arn                   string
	Endpoint              string
	Domain                string
	EndpointType          string
	IdentityProviderType  string
	LoggingRole           string
	Protocols             []string
	State                 string
	Tags                  map[string]string
	UserCount             int
	CreatedAt             time.Time
	lifecycle             *lifecycle.Machine
}

// User represents an AWS Transfer Family user.
type User struct {
	ServerID       string
	UserName       string
	Arn            string
	HomeDirectory  string
	HomeDirectoryType string
	Role           string
	SshPublicKeys  []*SSHPublicKey
	Tags           map[string]string
	CreatedAt      time.Time
}

// SSHPublicKey represents an SSH public key for a Transfer user.
type SSHPublicKey struct {
	SSHPublicKeyID   string
	SSHPublicKeyBody string
	DateImported     time.Time
}

// Workflow represents an AWS Transfer Family workflow.
type Workflow struct {
	WorkflowID  string
	Arn         string
	Description string
	Steps       []map[string]any
	OnException []map[string]any
	Tags        map[string]string
	CreatedAt   time.Time
}

// Store manages Transfer Family resources in memory.
type Store struct {
	mu        sync.RWMutex
	servers   map[string]*Server
	users     map[string]map[string]*User // serverID -> userName -> User
	workflows map[string]*Workflow        // workflowID -> Workflow
	tagsByArn map[string]map[string]string
	accountID string
	region    string
	lcConfig  *lifecycle.Config
	keySeq    int
	wfSeq     int
}

// NewStore returns a new empty Transfer Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		servers:   make(map[string]*Server),
		users:     make(map[string]map[string]*User),
		workflows: make(map[string]*Workflow),
		tagsByArn: make(map[string]map[string]string),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:transfer:%s:%s:", s.region, s.accountID)
}

func (s *Store) newServerID() string {
	return fmt.Sprintf("s-%012d", time.Now().UnixNano()%1000000000000)
}

// CreateServer creates a new Transfer server.
func (s *Store) CreateServer(domain, endpointType, identityProvider, loggingRole string, protocols []string, tags map[string]string) (*Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate protocols
	validProtocols := map[string]bool{"SFTP": true, "FTP": true, "FTPS": true, "AS2": true}
	for _, p := range protocols {
		if !validProtocols[p] {
			return nil, fmt.Errorf("invalid protocol: %s (must be SFTP, FTP, FTPS, or AS2)", p)
		}
	}

	id := s.newServerID()
	transitions := []lifecycle.Transition{
		{From: "OFFLINE", To: "ONLINE", Delay: 2 * time.Second},
	}

	// Generate endpoint URL
	endpoint := fmt.Sprintf("%s.server.transfer.%s.amazonaws.com", id, s.region)

	srv := &Server{
		ServerID:             id,
		Arn:                  s.arnPrefix() + "server/" + id,
		Endpoint:             endpoint,
		Domain:               domain,
		EndpointType:         endpointType,
		IdentityProviderType: identityProvider,
		LoggingRole:          loggingRole,
		Protocols:            protocols,
		State:                "OFFLINE",
		Tags:                 tags,
		CreatedAt:            time.Now().UTC(),
	}
	srv.lifecycle = lifecycle.NewMachine("OFFLINE", transitions, s.lcConfig)
	srv.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		srv.State = string(to)
	})

	s.servers[id] = srv
	s.users[id] = make(map[string]*User)
	s.tagsByArn[srv.Arn] = tags
	return srv, nil
}

// GetServer retrieves a server by ID.
func (s *Store) GetServer(id string) (*Server, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.servers[id]
	return srv, ok
}

// ListServers returns all servers.
func (s *Store) ListServers() []*Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Server, 0, len(s.servers))
	for _, srv := range s.servers {
		out = append(out, srv)
	}
	return out
}

// StopServer stops a server, transitioning to STOPPING then OFFLINE.
func (s *Store) StopServer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	srv, ok := s.servers[id]
	if !ok {
		return fmt.Errorf("server not found: %s", id)
	}
	if srv.State != "ONLINE" {
		return fmt.Errorf("server %s is not online: %s", id, srv.State)
	}
	srv.lifecycle.Stop()

	transitions := []lifecycle.Transition{
		{From: "STOPPING", To: "OFFLINE", Delay: 2 * time.Second},
	}
	srv.State = "STOPPING"
	srv.lifecycle = lifecycle.NewMachine("STOPPING", transitions, s.lcConfig)
	srv.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		srv.State = string(to)
	})
	return nil
}

// StartServer starts a server, transitioning to ONLINE.
func (s *Store) StartServer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	srv, ok := s.servers[id]
	if !ok {
		return fmt.Errorf("server not found: %s", id)
	}
	if srv.State != "OFFLINE" {
		return fmt.Errorf("server %s is not offline: %s", id, srv.State)
	}
	srv.lifecycle.Stop()

	transitions := []lifecycle.Transition{
		{From: "START_PENDING", To: "ONLINE", Delay: 2 * time.Second},
	}
	srv.State = "START_PENDING"
	srv.lifecycle = lifecycle.NewMachine("START_PENDING", transitions, s.lcConfig)
	srv.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		srv.State = string(to)
	})
	return nil
}

// DeleteServer removes a server.
func (s *Store) DeleteServer(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	srv, ok := s.servers[id]
	if !ok {
		return false
	}
	srv.lifecycle.Stop()
	delete(s.servers, id)
	delete(s.users, id)
	return true
}

// CreateUser creates a new user on a server.
func (s *Store) CreateUser(serverID, userName, role, homeDir, homeDirType string, tags map[string]string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.servers[serverID]; !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	userMap := s.users[serverID]
	if _, ok := userMap[userName]; ok {
		return nil, fmt.Errorf("user already exists: %s", userName)
	}

	// Validate home directory starts with /
	if homeDir != "" && homeDir[0] != '/' {
		return nil, fmt.Errorf("home directory must start with /: %s", homeDir)
	}

	user := &User{
		ServerID:          serverID,
		UserName:          userName,
		Arn:               s.arnPrefix() + "user/" + serverID + "/" + userName,
		HomeDirectory:     homeDir,
		HomeDirectoryType: homeDirType,
		Role:              role,
		Tags:              tags,
		CreatedAt:         time.Now().UTC(),
	}
	userMap[userName] = user
	s.servers[serverID].UserCount++
	return user, nil
}

// GetUser retrieves a user.
func (s *Store) GetUser(serverID, userName string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userMap, ok := s.users[serverID]
	if !ok {
		return nil, false
	}
	user, ok := userMap[userName]
	return user, ok
}

// ListUsers returns all users for a server.
func (s *Store) ListUsers(serverID string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userMap, ok := s.users[serverID]
	if !ok {
		return nil
	}
	out := make([]*User, 0, len(userMap))
	for _, u := range userMap {
		out = append(out, u)
	}
	return out
}

// DeleteUser removes a user.
func (s *Store) DeleteUser(serverID, userName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	userMap, ok := s.users[serverID]
	if !ok {
		return false
	}
	if _, ok := userMap[userName]; !ok {
		return false
	}
	delete(userMap, userName)
	s.servers[serverID].UserCount--
	return true
}

// ImportSSHPublicKey adds an SSH public key to a user.
func (s *Store) ImportSSHPublicKey(serverID, userName, keyBody string) (*SSHPublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userMap, ok := s.users[serverID]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}
	user, ok := userMap[userName]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userName)
	}

	s.keySeq++
	key := &SSHPublicKey{
		SSHPublicKeyID:   fmt.Sprintf("key-%012d", s.keySeq),
		SSHPublicKeyBody: keyBody,
		DateImported:     time.Now().UTC(),
	}
	user.SshPublicKeys = append(user.SshPublicKeys, key)
	return key, nil
}

// DeleteSSHPublicKey removes an SSH public key from a user.
func (s *Store) DeleteSSHPublicKey(serverID, userName, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userMap, ok := s.users[serverID]
	if !ok {
		return fmt.Errorf("server not found: %s", serverID)
	}
	user, ok := userMap[userName]
	if !ok {
		return fmt.Errorf("user not found: %s", userName)
	}

	for i, key := range user.SshPublicKeys {
		if key.SSHPublicKeyID == keyID {
			user.SshPublicKeys = append(user.SshPublicKeys[:i], user.SshPublicKeys[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("SSH public key not found: %s", keyID)
}

// UpdateServer updates a server's configuration.
func (s *Store) UpdateServer(id, loggingRole string, protocols []string) (*Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	srv, ok := s.servers[id]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", id)
	}
	if loggingRole != "" {
		srv.LoggingRole = loggingRole
	}
	if len(protocols) > 0 {
		srv.Protocols = protocols
	}
	return srv, nil
}

// UpdateUser updates a user's configuration.
func (s *Store) UpdateUser(serverID, userName, homeDir, role string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userMap, ok := s.users[serverID]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}
	user, ok := userMap[userName]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userName)
	}
	if homeDir != "" {
		user.HomeDirectory = homeDir
	}
	if role != "" {
		user.Role = role
	}
	return user, nil
}

// CreateWorkflow creates a new Transfer workflow.
func (s *Store) CreateWorkflow(description string, steps, onException []map[string]any, tags map[string]string) (*Workflow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	s.wfSeq++
	id := fmt.Sprintf("w-%012d", s.wfSeq)
	arn := s.arnPrefix() + "workflow/" + id
	wf := &Workflow{
		WorkflowID:  id,
		Arn:         arn,
		Description: description,
		Steps:       steps,
		OnException: onException,
		Tags:        tags,
		CreatedAt:   time.Now().UTC(),
	}
	s.workflows[id] = wf
	s.tagsByArn[arn] = tags
	return wf, nil
}

// DescribeWorkflow retrieves a workflow by ID.
func (s *Store) DescribeWorkflow(id string) (*Workflow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wf, ok := s.workflows[id]
	return wf, ok
}

// ListWorkflows returns all workflows.
func (s *Store) ListWorkflows() []*Workflow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Workflow, 0, len(s.workflows))
	for _, wf := range s.workflows {
		out = append(out, wf)
	}
	return out
}

// DeleteWorkflow removes a workflow.
func (s *Store) DeleteWorkflow(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	wf, ok := s.workflows[id]
	if !ok {
		return false
	}
	delete(s.tagsByArn, wf.Arn)
	delete(s.workflows, id)
	return true
}

// TagResource adds tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return false
	}
	for k, v := range tags {
		existing[k] = v
	}
	return true
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return false
	}
	for _, k := range keys {
		delete(existing, k)
	}
	return true
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tags, ok := s.tagsByArn[arn]
	if !ok {
		return nil, false
	}
	cp := make(map[string]string, len(tags))
	for k, v := range tags {
		cp[k] = v
	}
	return cp, true
}
