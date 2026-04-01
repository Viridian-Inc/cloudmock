package servicediscovery

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// NamespaceType represents the type of namespace.
type NamespaceType string

const (
	NamespaceHTTP       NamespaceType = "HTTP"
	NamespacePrivateDNS NamespaceType = "DNS_PRIVATE"
	NamespacePublicDNS  NamespaceType = "DNS_PUBLIC"
)

// Namespace represents a Cloud Map namespace.
type Namespace struct {
	ID          string
	Name        string
	Type        NamespaceType
	ARN         string
	Description string
	HostedZoneID string // for DNS namespaces
	VpcID       string  // for private DNS namespaces
	CreatedDate time.Time
	Tags        map[string]string
}

// ServiceEntry represents a Cloud Map service.
type ServiceEntry struct {
	ID           string
	Name         string
	ARN          string
	NamespaceID  string
	Description  string
	DnsConfig    *DnsConfig
	HealthCheck  *HealthCheckConfig
	CreateDate   time.Time
	InstanceCount int
	Tags         map[string]string
}

// DnsConfig holds DNS configuration for a service.
type DnsConfig struct {
	NamespaceID  string
	RoutingPolicy string
	DnsRecords   []DnsRecord
}

// DnsRecord describes a DNS record type and TTL.
type DnsRecord struct {
	Type string
	TTL  int64
}

// HealthCheckConfig holds health check settings for a service.
type HealthCheckConfig struct {
	Type             string
	ResourcePath     string
	FailureThreshold int
}

// Instance represents an instance registered with a service.
type Instance struct {
	ID           string
	ServiceID    string
	Attributes   map[string]string
	Tags         map[string]string
	HealthStatus string // HEALTHY, UNHEALTHY, UNKNOWN
}

// Store manages all Service Discovery state in memory.
type Store struct {
	mu         sync.RWMutex
	namespaces map[string]*Namespace            // nsID -> namespace
	services   map[string]*ServiceEntry          // svcID -> service
	instances  map[string]map[string]*Instance   // svcID -> instID -> instance
	accountID  string
	region     string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		namespaces: make(map[string]*Namespace),
		services:   make(map[string]*ServiceEntry),
		instances:  make(map[string]map[string]*Instance),
		accountID:  accountID,
		region:     region,
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ns-%08x%04x%04x%04x%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newServiceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("srv-%08x%04x%04x%04x%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newOperationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x%04x%04x%04x%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) namespaceARN(id string) string {
	return fmt.Sprintf("arn:aws:servicediscovery:%s:%s:namespace/%s", s.region, s.accountID, id)
}

func (s *Store) serviceARN(id string) string {
	return fmt.Sprintf("arn:aws:servicediscovery:%s:%s:service/%s", s.region, s.accountID, id)
}

// CreateNamespace creates a new namespace.
func (s *Store) CreateNamespace(name, description string, nsType NamespaceType, vpcID string, tags map[string]string) (*Namespace, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check for duplicate name
	for _, ns := range s.namespaces {
		if ns.Name == name && ns.Type == nsType {
			return nil, ""
		}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newID()
	arn := s.namespaceARN(id)
	hostedZoneID := ""
	if nsType == NamespacePrivateDNS || nsType == NamespacePublicDNS {
		hostedZoneID = "Z" + newOperationID()[:12]
	}
	ns := &Namespace{
		ID: id, Name: name, Type: nsType, ARN: arn,
		Description: description, HostedZoneID: hostedZoneID,
		VpcID: vpcID, CreatedDate: time.Now().UTC(), Tags: tags,
	}
	s.namespaces[id] = ns
	operationID := newOperationID()
	return ns, operationID
}

// GetNamespace returns a namespace by ID.
func (s *Store) GetNamespace(id string) (*Namespace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[id]
	return ns, ok
}

// ListNamespaces returns all namespaces.
func (s *Store) ListNamespaces() []*Namespace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Namespace, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		result = append(result, ns)
	}
	return result
}

// DeleteNamespace removes a namespace.
func (s *Store) DeleteNamespace(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[id]; !ok {
		return "", false
	}
	delete(s.namespaces, id)
	return newOperationID(), true
}

// CreateService creates a new service in a namespace.
func (s *Store) CreateService(name, namespaceID, description string, dnsConfig *DnsConfig, healthCheck *HealthCheckConfig, tags map[string]string) (*ServiceEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[namespaceID]; !ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newServiceID()
	svc := &ServiceEntry{
		ID: id, Name: name, ARN: s.serviceARN(id),
		NamespaceID: namespaceID, Description: description,
		DnsConfig: dnsConfig, HealthCheck: healthCheck,
		CreateDate: time.Now().UTC(), Tags: tags,
	}
	s.services[id] = svc
	s.instances[id] = make(map[string]*Instance)
	return svc, true
}

// GetService returns a service by ID.
func (s *Store) GetService(id string) (*ServiceEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svc, ok := s.services[id]
	return svc, ok
}

// ListServices returns all services, optionally filtered by namespace.
func (s *Store) ListServices(namespaceID string) []*ServiceEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ServiceEntry, 0)
	for _, svc := range s.services {
		if namespaceID != "" && svc.NamespaceID != namespaceID {
			continue
		}
		result = append(result, svc)
	}
	return result
}

// UpdateService updates a service's description and DNS config.
func (s *Store) UpdateService(id, description string, dnsConfig *DnsConfig, healthCheck *HealthCheckConfig) (*ServiceEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	svc, ok := s.services[id]
	if !ok {
		return nil, false
	}
	if description != "" {
		svc.Description = description
	}
	if dnsConfig != nil {
		svc.DnsConfig = dnsConfig
	}
	if healthCheck != nil {
		svc.HealthCheck = healthCheck
	}
	return svc, true
}

// DeleteService removes a service.
func (s *Store) DeleteService(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[id]; !ok {
		return false
	}
	delete(s.services, id)
	delete(s.instances, id)
	return true
}

// RegisterInstance registers an instance with a service.
func (s *Store) RegisterInstance(serviceID, instanceID string, attributes map[string]string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceID]; !ok {
		return "", false
	}
	if attributes == nil {
		attributes = make(map[string]string)
	}
	inst := &Instance{
		ID: instanceID, ServiceID: serviceID,
		Attributes: attributes, Tags: make(map[string]string),
		HealthStatus: "HEALTHY",
	}
	if s.instances[serviceID] == nil {
		s.instances[serviceID] = make(map[string]*Instance)
	}
	s.instances[serviceID][instanceID] = inst
	s.services[serviceID].InstanceCount = len(s.instances[serviceID])
	return newOperationID(), true
}

// DeregisterInstance removes an instance from a service.
func (s *Store) DeregisterInstance(serviceID, instanceID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	insts, ok := s.instances[serviceID]
	if !ok {
		return "", false
	}
	if _, ok := insts[instanceID]; !ok {
		return "", false
	}
	delete(insts, instanceID)
	if svc, ok := s.services[serviceID]; ok {
		svc.InstanceCount = len(insts)
	}
	return newOperationID(), true
}

// GetInstance returns an instance.
func (s *Store) GetInstance(serviceID, instanceID string) (*Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	insts, ok := s.instances[serviceID]
	if !ok {
		return nil, false
	}
	inst, ok := insts[instanceID]
	return inst, ok
}

// ListInstances returns all instances for a service.
func (s *Store) ListInstances(serviceID string) ([]*Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	insts, ok := s.instances[serviceID]
	if !ok {
		return nil, false
	}
	result := make([]*Instance, 0, len(insts))
	for _, inst := range insts {
		result = append(result, inst)
	}
	return result, true
}

// DiscoverInstances finds instances by namespace and service name.
// Only returns HEALTHY instances, matching AWS behavior.
func (s *Store) DiscoverInstances(namespaceName, serviceName string, queryParams map[string]string, healthFilter string) []*Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Find namespace by name
	var nsID string
	for _, ns := range s.namespaces {
		if ns.Name == namespaceName {
			nsID = ns.ID
			break
		}
	}
	if nsID == "" {
		return nil
	}
	// Find service in namespace
	var svcID string
	for _, svc := range s.services {
		if svc.NamespaceID == nsID && svc.Name == serviceName {
			svcID = svc.ID
			break
		}
	}
	if svcID == "" {
		return nil
	}
	insts := s.instances[svcID]
	result := make([]*Instance, 0, len(insts))
	for _, inst := range insts {
		// Filter by health status (default: HEALTHY only).
		if healthFilter == "" {
			healthFilter = "HEALTHY"
		}
		if healthFilter != "ALL" && inst.HealthStatus != healthFilter {
			continue
		}
		match := true
		for k, v := range queryParams {
			if inst.Attributes[k] != v {
				match = false
				break
			}
		}
		if match {
			result = append(result, inst)
		}
	}
	return result
}

// UpdateInstanceCustomHealthStatus updates the health status of an instance.
func (s *Store) UpdateInstanceCustomHealthStatus(serviceID, instanceID, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	insts, ok := s.instances[serviceID]
	if !ok {
		return false
	}
	inst, ok := insts[instanceID]
	if !ok {
		return false
	}
	inst.HealthStatus = status
	return true
}

// GetNamespaceByName returns a namespace by name.
func (s *Store) GetNamespaceByName(name string) (*Namespace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ns := range s.namespaces {
		if ns.Name == name {
			return ns, true
		}
	}
	return nil, false
}

// TagResource applies tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for k, v := range tags {
			r[k] = v
		}
		return true
	}
	return false
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for _, k := range keys {
			delete(r, k)
		}
		return true
	}
	return false
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r := s.findTagsByARN(arn); r != nil {
		cp := make(map[string]string, len(r))
		for k, v := range r {
			cp[k] = v
		}
		return cp, true
	}
	return nil, false
}

func (s *Store) findTagsByARN(arn string) map[string]string {
	for _, ns := range s.namespaces {
		if ns.ARN == arn {
			return ns.Tags
		}
	}
	for _, svc := range s.services {
		if svc.ARN == arn {
			return svc.Tags
		}
	}
	return nil
}
