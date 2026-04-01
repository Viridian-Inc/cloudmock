package codeconnections

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Connection status constants.
const (
	ConnectionStatusPending   = "PENDING"
	ConnectionStatusAvailable = "AVAILABLE"
	ConnectionStatusError     = "ERROR"
)

// Connection represents a CodeConnections connection to an external SCM provider.
type Connection struct {
	Name           string
	ARN            string
	ProviderType   string
	HostARN        string
	ConnectionStatus string
	OwnerAccountID string
	CreatedAt      time.Time
	Tags           map[string]string
	lifecycle      *lifecycle.Machine
}

// Host represents a CodeConnections host for self-managed providers.
type Host struct {
	Name          string
	ARN           string
	ProviderType  string
	ProviderEndpoint string
	Status        string
	VPCConfiguration *VPCConfiguration
	CreatedAt     time.Time
	Tags          map[string]string
}

// VPCConfiguration describes VPC settings for a host.
type VPCConfiguration struct {
	VpcID             string
	SubnetIDs         []string
	SecurityGroupIDs  []string
	TLSCertificate   string
}

// Store is the in-memory store for all CodeConnections resources.
type Store struct {
	mu          sync.RWMutex
	accountID   string
	region      string
	connections map[string]*Connection // ARN -> Connection
	hosts       map[string]*Host       // ARN -> Host
	tags        map[string]map[string]string
	lcConfig    *lifecycle.Config
}

// NewStore creates an empty CodeConnections store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:   accountID,
		region:      region,
		connections: make(map[string]*Connection),
		hosts:       make(map[string]*Host),
		tags:        make(map[string]map[string]string),
		lcConfig:    lifecycle.DefaultConfig(),
	}
}

// ---- ARN builders ----

func (s *Store) connectionARN(id string) string {
	return fmt.Sprintf("arn:aws:codeconnections:%s:%s:connection/%s", s.region, s.accountID, id)
}

func (s *Store) hostARN(id string) string {
	return fmt.Sprintf("arn:aws:codeconnections:%s:%s:host/%s", s.region, s.accountID, id)
}

// ---- Connection operations ----

func (s *Store) CreateConnection(name, providerType, hostARN string, tags map[string]string) (*Connection, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Connection name is required.")
	}
	if providerType == "" && hostARN == "" {
		return nil, service.ErrValidation("Either ProviderType or HostArn is required.")
	}
	validProviders := map[string]bool{"GitHub": true, "Bitbucket": true, "GitLab": true, "GitHubEnterpriseServer": true}
	if providerType != "" && !validProviders[providerType] {
		return nil, service.ErrValidation("Invalid ProviderType. Must be one of: GitHub, Bitbucket, GitLab, GitHubEnterpriseServer")
	}

	// Check for duplicate name
	for _, c := range s.connections {
		if c.Name == name {
			return nil, service.NewAWSError("ResourceAlreadyExistsException",
				fmt.Sprintf("Connection already exists: %s", name), http.StatusConflict)
		}
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	connID := newUUID()
	conn := &Connection{
		Name:             name,
		ARN:              s.connectionARN(connID),
		ProviderType:     providerType,
		HostARN:          hostARN,
		ConnectionStatus: ConnectionStatusPending,
		OwnerAccountID:   s.accountID,
		CreatedAt:        time.Now().UTC(),
		Tags:             tags,
	}

	// Set up lifecycle: PENDING -> AVAILABLE
	transitions := []lifecycle.Transition{
		{From: lifecycle.State(ConnectionStatusPending), To: lifecycle.State(ConnectionStatusAvailable), Delay: 3 * time.Second},
	}
	conn.lifecycle = lifecycle.NewMachine(lifecycle.State(ConnectionStatusPending), transitions, s.lcConfig)
	conn.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		conn.ConnectionStatus = string(to)
	})

	s.connections[conn.ARN] = conn
	return conn, nil
}

func (s *Store) GetConnection(arn string) (*Connection, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.connections[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Connection not found: %s", arn), http.StatusNotFound)
	}
	if conn.lifecycle != nil {
		conn.ConnectionStatus = string(conn.lifecycle.State())
	}
	return conn, nil
}

func (s *Store) ListConnections(providerType, hostARN string) []*Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Connection
	for _, c := range s.connections {
		if c.lifecycle != nil {
			c.ConnectionStatus = string(c.lifecycle.State())
		}
		if providerType != "" && c.ProviderType != providerType {
			continue
		}
		if hostARN != "" && c.HostARN != hostARN {
			continue
		}
		result = append(result, c)
	}
	return result
}

func (s *Store) DeleteConnection(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.connections[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Connection not found: %s", arn), http.StatusNotFound)
	}
	if conn.lifecycle != nil {
		conn.lifecycle.Stop()
	}
	delete(s.connections, arn)
	return nil
}

func (s *Store) UpdateConnectionStatus(arn, status string) (*Connection, *service.AWSError) {
	s.mu.Lock()
	conn, ok := s.connections[arn]
	if !ok {
		s.mu.Unlock()
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Connection not found: %s", arn), http.StatusNotFound)
	}
	lc := conn.lifecycle
	conn.ConnectionStatus = status
	s.mu.Unlock()

	// ForceState triggers OnTransition callback which may acquire s.mu,
	// so we must not hold the lock here.
	if lc != nil {
		lc.Stop()
		lc.ForceState(lifecycle.State(status))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return conn, nil
}

// ---- Host operations ----

func (s *Store) CreateHost(name, providerType, providerEndpoint string, vpcConfig *VPCConfiguration, tags map[string]string) (*Host, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Host name is required.")
	}
	if providerType == "" {
		return nil, service.ErrValidation("ProviderType is required.")
	}
	if providerEndpoint == "" {
		return nil, service.ErrValidation("ProviderEndpoint is required.")
	}

	// Validate VPC configuration if provided
	if vpcConfig != nil {
		if vpcConfig.VpcID == "" {
			return nil, service.ErrValidation("VpcConfiguration.VpcId is required when VpcConfiguration is provided.")
		}
		if len(vpcConfig.SubnetIDs) == 0 {
			return nil, service.ErrValidation("VpcConfiguration.SubnetIds must have at least one subnet.")
		}
		if len(vpcConfig.SecurityGroupIDs) == 0 {
			return nil, service.ErrValidation("VpcConfiguration.SecurityGroupIds must have at least one security group.")
		}
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	hostID := newUUID()
	host := &Host{
		Name:             name,
		ARN:              s.hostARN(hostID),
		ProviderType:     providerType,
		ProviderEndpoint: providerEndpoint,
		Status:           "AVAILABLE",
		VPCConfiguration: vpcConfig,
		CreatedAt:        time.Now().UTC(),
		Tags:             tags,
	}
	s.hosts[host.ARN] = host
	return host, nil
}

func (s *Store) GetHost(arn string) (*Host, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	host, ok := s.hosts[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Host not found: %s", arn), http.StatusNotFound)
	}
	return host, nil
}

func (s *Store) ListHosts() []*Host {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Host, 0, len(s.hosts))
	for _, h := range s.hosts {
		result = append(result, h)
	}
	return result
}

func (s *Store) DeleteHost(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.hosts[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Host not found: %s", arn), http.StatusNotFound)
	}
	delete(s.hosts, arn)
	return nil
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}

	// Also update resource-level tags
	if conn, ok := s.connections[arn]; ok {
		for k, v := range tags {
			conn.Tags[k] = v
		}
	}
	if host, ok := s.hosts[arn]; ok {
		for k, v := range tags {
			host.Tags[k] = v
		}
	}
	return nil
}

func (s *Store) UntagResource(arn string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.tags[arn]
	if m != nil {
		for _, k := range keys {
			delete(m, k)
		}
	}

	if conn, ok := s.connections[arn]; ok {
		for _, k := range keys {
			delete(conn.Tags, k)
		}
	}
	if host, ok := s.hosts[arn]; ok {
		for _, k := range keys {
			delete(host.Tags, k)
		}
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.tags[arn]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
