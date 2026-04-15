package globalaccelerator

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredAccelerator models a Global Accelerator standard accelerator.
type StoredAccelerator struct {
	Arn              string
	Name             string
	IpAddressType    string
	IpAddresses      []string
	Enabled          bool
	DnsName          string
	DualStackDnsName string
	Status           string
	IpSets           []map[string]any
	Events           []map[string]any
	CreatedTime      time.Time
	LastModifiedTime time.Time
}

// StoredCustomRoutingAccelerator models a custom routing accelerator.
type StoredCustomRoutingAccelerator struct {
	Arn              string
	Name             string
	IpAddressType    string
	IpAddresses      []string
	Enabled          bool
	DnsName          string
	Status           string
	IpSets           []map[string]any
	CreatedTime      time.Time
	LastModifiedTime time.Time
}

// StoredAcceleratorAttributes are flow log attributes for a standard accelerator.
type StoredAcceleratorAttributes struct {
	FlowLogsEnabled  bool
	FlowLogsS3Bucket string
	FlowLogsS3Prefix string
}

// StoredCustomRoutingAcceleratorAttributes are flow log attributes for a custom routing accelerator.
type StoredCustomRoutingAcceleratorAttributes struct {
	FlowLogsEnabled  bool
	FlowLogsS3Bucket string
	FlowLogsS3Prefix string
}

// StoredListener models a Global Accelerator listener.
type StoredListener struct {
	Arn            string
	AcceleratorArn string
	Protocol       string
	ClientAffinity string
	PortRanges     []map[string]any
}

// StoredCustomRoutingListener models a custom routing listener.
type StoredCustomRoutingListener struct {
	Arn            string
	AcceleratorArn string
	PortRanges     []map[string]any
}

// StoredEndpointGroup models a standard endpoint group.
type StoredEndpointGroup struct {
	Arn                        string
	ListenerArn                string
	EndpointGroupRegion        string
	HealthCheckPort            int
	HealthCheckProtocol        string
	HealthCheckPath            string
	HealthCheckIntervalSeconds int
	ThresholdCount             int
	TrafficDialPercentage      float64
	PortOverrides              []map[string]any
	EndpointDescriptions       []map[string]any
}

// StoredCustomRoutingEndpointGroup models a custom routing endpoint group.
type StoredCustomRoutingEndpointGroup struct {
	Arn                     string
	ListenerArn             string
	EndpointGroupRegion     string
	DestinationDescriptions []map[string]any
	EndpointDescriptions    []map[string]any
}

// StoredByoipCidr models a BYOIP CIDR allocation.
type StoredByoipCidr struct {
	Cidr   string
	State  string
	Events []map[string]any
}

// StoredCrossAccountAttachment models a cross-account attachment.
type StoredCrossAccountAttachment struct {
	Arn              string
	Name             string
	Principals       []string
	Resources        []map[string]any
	CreatedTime      time.Time
	LastModifiedTime time.Time
}

// ── Store ────────────────────────────────────────────────────────────────────

// Store is the in-memory data store for globalaccelerator resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	accelerators        map[string]*StoredAccelerator
	customAccelerators  map[string]*StoredCustomRoutingAccelerator
	listeners           map[string]*StoredListener
	customListeners     map[string]*StoredCustomRoutingListener
	endpointGroups      map[string]*StoredEndpointGroup
	customEndpointGroups map[string]*StoredCustomRoutingEndpointGroup
	byoipCidrs          map[string]*StoredByoipCidr
	attachments         map[string]*StoredCrossAccountAttachment

	acceleratorAttrs        map[string]*StoredAcceleratorAttributes
	customAcceleratorAttrs  map[string]*StoredCustomRoutingAcceleratorAttributes

	tags map[string]map[string]string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:               accountID,
		region:                  region,
		accelerators:            make(map[string]*StoredAccelerator),
		customAccelerators:      make(map[string]*StoredCustomRoutingAccelerator),
		listeners:               make(map[string]*StoredListener),
		customListeners:         make(map[string]*StoredCustomRoutingListener),
		endpointGroups:          make(map[string]*StoredEndpointGroup),
		customEndpointGroups:    make(map[string]*StoredCustomRoutingEndpointGroup),
		byoipCidrs:              make(map[string]*StoredByoipCidr),
		attachments:             make(map[string]*StoredCrossAccountAttachment),
		acceleratorAttrs:        make(map[string]*StoredAcceleratorAttributes),
		customAcceleratorAttrs:  make(map[string]*StoredCustomRoutingAcceleratorAttributes),
		tags:                    make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accelerators = make(map[string]*StoredAccelerator)
	s.customAccelerators = make(map[string]*StoredCustomRoutingAccelerator)
	s.listeners = make(map[string]*StoredListener)
	s.customListeners = make(map[string]*StoredCustomRoutingListener)
	s.endpointGroups = make(map[string]*StoredEndpointGroup)
	s.customEndpointGroups = make(map[string]*StoredCustomRoutingEndpointGroup)
	s.byoipCidrs = make(map[string]*StoredByoipCidr)
	s.attachments = make(map[string]*StoredCrossAccountAttachment)
	s.acceleratorAttrs = make(map[string]*StoredAcceleratorAttributes)
	s.customAcceleratorAttrs = make(map[string]*StoredCustomRoutingAcceleratorAttributes)
	s.tags = make(map[string]map[string]string)
}

// ── Accelerator ──────────────────────────────────────────────────────────────

// CreateAccelerator stores a new standard accelerator.
func (s *Store) CreateAccelerator(name, ipAddressType string, ipAddresses []string, enabled bool) *StoredAccelerator {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	arn := s.acceleratorArn(id)
	if ipAddressType == "" {
		ipAddressType = "IPV4"
	}
	if len(ipAddresses) == 0 {
		ipAddresses = defaultIPv4Addresses()
	}
	now := time.Now().UTC()
	a := &StoredAccelerator{
		Arn:              arn,
		Name:             name,
		IpAddressType:    ipAddressType,
		IpAddresses:      ipAddresses,
		Enabled:          enabled,
		DnsName:          fmt.Sprintf("%s.awsglobalaccelerator.com", strings.ToLower(id[:16])),
		DualStackDnsName: fmt.Sprintf("%s.dualstack.awsglobalaccelerator.com", strings.ToLower(id[:16])),
		Status:           "DEPLOYED",
		IpSets:           buildIpSets(ipAddressType, ipAddresses),
		Events:           []map[string]any{},
		CreatedTime:      now,
		LastModifiedTime: now,
	}
	s.accelerators[arn] = a
	s.acceleratorAttrs[arn] = &StoredAcceleratorAttributes{}
	return a
}

// GetAccelerator returns an accelerator by ARN.
func (s *Store) GetAccelerator(arn string) (*StoredAccelerator, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.accelerators[arn]
	if !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+arn, 404)
	}
	return a, nil
}

// UpdateAccelerator updates fields on an accelerator.
func (s *Store) UpdateAccelerator(arn string, name *string, ipAddressType *string, ipAddresses []string, enabled *bool) (*StoredAccelerator, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.accelerators[arn]
	if !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+arn, 404)
	}
	if name != nil {
		a.Name = *name
	}
	if ipAddressType != nil && *ipAddressType != "" {
		a.IpAddressType = *ipAddressType
	}
	if len(ipAddresses) > 0 {
		a.IpAddresses = ipAddresses
	}
	if enabled != nil {
		a.Enabled = *enabled
	}
	a.IpSets = buildIpSets(a.IpAddressType, a.IpAddresses)
	a.LastModifiedTime = time.Now().UTC()
	return a, nil
}

// DeleteAccelerator removes a standard accelerator by ARN.
func (s *Store) DeleteAccelerator(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accelerators[arn]; !ok {
		return service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+arn, 404)
	}
	delete(s.accelerators, arn)
	delete(s.acceleratorAttrs, arn)
	return nil
}

// ListAccelerators returns all standard accelerators.
func (s *Store) ListAccelerators() []*StoredAccelerator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAccelerator, 0, len(s.accelerators))
	for _, a := range s.accelerators {
		out = append(out, a)
	}
	return out
}

// GetAcceleratorAttributes returns flow log attributes for an accelerator.
func (s *Store) GetAcceleratorAttributes(arn string) (*StoredAcceleratorAttributes, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.accelerators[arn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+arn, 404)
	}
	attrs, ok := s.acceleratorAttrs[arn]
	if !ok {
		return &StoredAcceleratorAttributes{}, nil
	}
	return attrs, nil
}

// UpdateAcceleratorAttributes sets flow log attributes for an accelerator.
func (s *Store) UpdateAcceleratorAttributes(arn string, enabled *bool, bucket *string, prefix *string) (*StoredAcceleratorAttributes, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accelerators[arn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+arn, 404)
	}
	attrs, ok := s.acceleratorAttrs[arn]
	if !ok {
		attrs = &StoredAcceleratorAttributes{}
		s.acceleratorAttrs[arn] = attrs
	}
	if enabled != nil {
		attrs.FlowLogsEnabled = *enabled
	}
	if bucket != nil {
		attrs.FlowLogsS3Bucket = *bucket
	}
	if prefix != nil {
		attrs.FlowLogsS3Prefix = *prefix
	}
	return attrs, nil
}

// ── Custom Routing Accelerator ───────────────────────────────────────────────

// CreateCustomRoutingAccelerator stores a new custom routing accelerator.
func (s *Store) CreateCustomRoutingAccelerator(name, ipAddressType string, ipAddresses []string, enabled bool) *StoredCustomRoutingAccelerator {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	arn := s.customAcceleratorArn(id)
	if ipAddressType == "" {
		ipAddressType = "IPV4"
	}
	if len(ipAddresses) == 0 {
		ipAddresses = defaultIPv4Addresses()
	}
	now := time.Now().UTC()
	a := &StoredCustomRoutingAccelerator{
		Arn:              arn,
		Name:             name,
		IpAddressType:    ipAddressType,
		IpAddresses:      ipAddresses,
		Enabled:          enabled,
		DnsName:          fmt.Sprintf("%s.awsglobalaccelerator.com", strings.ToLower(id[:16])),
		Status:           "DEPLOYED",
		IpSets:           buildIpSets(ipAddressType, ipAddresses),
		CreatedTime:      now,
		LastModifiedTime: now,
	}
	s.customAccelerators[arn] = a
	s.customAcceleratorAttrs[arn] = &StoredCustomRoutingAcceleratorAttributes{}
	return a
}

// GetCustomRoutingAccelerator returns a custom routing accelerator by ARN.
func (s *Store) GetCustomRoutingAccelerator(arn string) (*StoredCustomRoutingAccelerator, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.customAccelerators[arn]
	if !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+arn, 404)
	}
	return a, nil
}

// UpdateCustomRoutingAccelerator updates fields on a custom routing accelerator.
func (s *Store) UpdateCustomRoutingAccelerator(arn string, name *string, ipAddressType *string, ipAddresses []string, enabled *bool) (*StoredCustomRoutingAccelerator, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.customAccelerators[arn]
	if !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+arn, 404)
	}
	if name != nil {
		a.Name = *name
	}
	if ipAddressType != nil && *ipAddressType != "" {
		a.IpAddressType = *ipAddressType
	}
	if len(ipAddresses) > 0 {
		a.IpAddresses = ipAddresses
	}
	if enabled != nil {
		a.Enabled = *enabled
	}
	a.IpSets = buildIpSets(a.IpAddressType, a.IpAddresses)
	a.LastModifiedTime = time.Now().UTC()
	return a, nil
}

// DeleteCustomRoutingAccelerator removes a custom routing accelerator.
func (s *Store) DeleteCustomRoutingAccelerator(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customAccelerators[arn]; !ok {
		return service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+arn, 404)
	}
	delete(s.customAccelerators, arn)
	delete(s.customAcceleratorAttrs, arn)
	return nil
}

// ListCustomRoutingAccelerators returns all custom routing accelerators.
func (s *Store) ListCustomRoutingAccelerators() []*StoredCustomRoutingAccelerator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCustomRoutingAccelerator, 0, len(s.customAccelerators))
	for _, a := range s.customAccelerators {
		out = append(out, a)
	}
	return out
}

// GetCustomRoutingAcceleratorAttributes returns flow log attributes for a custom routing accelerator.
func (s *Store) GetCustomRoutingAcceleratorAttributes(arn string) (*StoredCustomRoutingAcceleratorAttributes, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.customAccelerators[arn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+arn, 404)
	}
	attrs, ok := s.customAcceleratorAttrs[arn]
	if !ok {
		return &StoredCustomRoutingAcceleratorAttributes{}, nil
	}
	return attrs, nil
}

// UpdateCustomRoutingAcceleratorAttributes sets flow log attributes for a custom routing accelerator.
func (s *Store) UpdateCustomRoutingAcceleratorAttributes(arn string, enabled *bool, bucket *string, prefix *string) (*StoredCustomRoutingAcceleratorAttributes, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customAccelerators[arn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+arn, 404)
	}
	attrs, ok := s.customAcceleratorAttrs[arn]
	if !ok {
		attrs = &StoredCustomRoutingAcceleratorAttributes{}
		s.customAcceleratorAttrs[arn] = attrs
	}
	if enabled != nil {
		attrs.FlowLogsEnabled = *enabled
	}
	if bucket != nil {
		attrs.FlowLogsS3Bucket = *bucket
	}
	if prefix != nil {
		attrs.FlowLogsS3Prefix = *prefix
	}
	return attrs, nil
}

// ── Listener ─────────────────────────────────────────────────────────────────

// CreateListener stores a new standard listener.
func (s *Store) CreateListener(acceleratorArn, protocol, clientAffinity string, portRanges []map[string]any) (*StoredListener, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accelerators[acceleratorArn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Accelerator not found: "+acceleratorArn, 404)
	}
	id := generateID()
	arn := s.listenerArn(acceleratorArn, id)
	if clientAffinity == "" {
		clientAffinity = "NONE"
	}
	l := &StoredListener{
		Arn:            arn,
		AcceleratorArn: acceleratorArn,
		Protocol:       protocol,
		ClientAffinity: clientAffinity,
		PortRanges:     portRanges,
	}
	s.listeners[arn] = l
	return l, nil
}

// GetListener returns a listener by ARN.
func (s *Store) GetListener(arn string) (*StoredListener, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.listeners[arn]
	if !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Listener not found: "+arn, 404)
	}
	return l, nil
}

// UpdateListener updates fields on a listener.
func (s *Store) UpdateListener(arn string, protocol *string, clientAffinity *string, portRanges []map[string]any) (*StoredListener, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.listeners[arn]
	if !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Listener not found: "+arn, 404)
	}
	if protocol != nil && *protocol != "" {
		l.Protocol = *protocol
	}
	if clientAffinity != nil && *clientAffinity != "" {
		l.ClientAffinity = *clientAffinity
	}
	if len(portRanges) > 0 {
		l.PortRanges = portRanges
	}
	return l, nil
}

// DeleteListener removes a listener.
func (s *Store) DeleteListener(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.listeners[arn]; !ok {
		return service.NewAWSError("ListenerNotFoundException",
			"Listener not found: "+arn, 404)
	}
	delete(s.listeners, arn)
	return nil
}

// ListListeners returns all listeners on an accelerator.
func (s *Store) ListListeners(acceleratorArn string) []*StoredListener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredListener, 0)
	for _, l := range s.listeners {
		if acceleratorArn == "" || l.AcceleratorArn == acceleratorArn {
			out = append(out, l)
		}
	}
	return out
}

// ── Custom Routing Listener ──────────────────────────────────────────────────

// CreateCustomRoutingListener stores a new custom routing listener.
func (s *Store) CreateCustomRoutingListener(acceleratorArn string, portRanges []map[string]any) (*StoredCustomRoutingListener, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customAccelerators[acceleratorArn]; !ok {
		return nil, service.NewAWSError("AcceleratorNotFoundException",
			"Custom routing accelerator not found: "+acceleratorArn, 404)
	}
	id := generateID()
	arn := s.customListenerArn(acceleratorArn, id)
	l := &StoredCustomRoutingListener{
		Arn:            arn,
		AcceleratorArn: acceleratorArn,
		PortRanges:     portRanges,
	}
	s.customListeners[arn] = l
	return l, nil
}

// GetCustomRoutingListener returns a custom routing listener by ARN.
func (s *Store) GetCustomRoutingListener(arn string) (*StoredCustomRoutingListener, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.customListeners[arn]
	if !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Custom routing listener not found: "+arn, 404)
	}
	return l, nil
}

// UpdateCustomRoutingListener updates fields on a custom routing listener.
func (s *Store) UpdateCustomRoutingListener(arn string, portRanges []map[string]any) (*StoredCustomRoutingListener, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.customListeners[arn]
	if !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Custom routing listener not found: "+arn, 404)
	}
	if len(portRanges) > 0 {
		l.PortRanges = portRanges
	}
	return l, nil
}

// DeleteCustomRoutingListener removes a custom routing listener.
func (s *Store) DeleteCustomRoutingListener(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customListeners[arn]; !ok {
		return service.NewAWSError("ListenerNotFoundException",
			"Custom routing listener not found: "+arn, 404)
	}
	delete(s.customListeners, arn)
	return nil
}

// ListCustomRoutingListeners returns all custom routing listeners on an accelerator.
func (s *Store) ListCustomRoutingListeners(acceleratorArn string) []*StoredCustomRoutingListener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCustomRoutingListener, 0)
	for _, l := range s.customListeners {
		if acceleratorArn == "" || l.AcceleratorArn == acceleratorArn {
			out = append(out, l)
		}
	}
	return out
}

// ── Endpoint Group ───────────────────────────────────────────────────────────

// CreateEndpointGroup stores a new endpoint group.
func (s *Store) CreateEndpointGroup(listenerArn, region string, healthPort int, healthProtocol, healthPath string, healthInterval, thresholdCount int, trafficDial float64, portOverrides []map[string]any, endpoints []map[string]any) (*StoredEndpointGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.listeners[listenerArn]; !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Listener not found: "+listenerArn, 404)
	}
	id := generateID()
	arn := s.endpointGroupArn(listenerArn, id)
	if healthProtocol == "" {
		healthProtocol = "TCP"
	}
	if healthPort == 0 {
		healthPort = 80
	}
	if healthInterval == 0 {
		healthInterval = 30
	}
	if thresholdCount == 0 {
		thresholdCount = 3
	}
	if trafficDial == 0 {
		trafficDial = 100
	}
	descriptions := make([]map[string]any, 0, len(endpoints))
	for _, e := range endpoints {
		descriptions = append(descriptions, endpointConfigToDescription(e))
	}
	g := &StoredEndpointGroup{
		Arn:                        arn,
		ListenerArn:                listenerArn,
		EndpointGroupRegion:        region,
		HealthCheckPort:            healthPort,
		HealthCheckProtocol:        healthProtocol,
		HealthCheckPath:            healthPath,
		HealthCheckIntervalSeconds: healthInterval,
		ThresholdCount:             thresholdCount,
		TrafficDialPercentage:      trafficDial,
		PortOverrides:              portOverrides,
		EndpointDescriptions:       descriptions,
	}
	s.endpointGroups[arn] = g
	return g, nil
}

// GetEndpointGroup returns an endpoint group by ARN.
func (s *Store) GetEndpointGroup(arn string) (*StoredEndpointGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.endpointGroups[arn]
	if !ok {
		return nil, service.NewAWSError("EndpointGroupNotFoundException",
			"Endpoint group not found: "+arn, 404)
	}
	return g, nil
}

// UpdateEndpointGroup updates fields on an endpoint group.
func (s *Store) UpdateEndpointGroup(arn string, healthPort *int, healthProtocol, healthPath *string, healthInterval, thresholdCount *int, trafficDial *float64, portOverrides []map[string]any, endpoints []map[string]any) (*StoredEndpointGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.endpointGroups[arn]
	if !ok {
		return nil, service.NewAWSError("EndpointGroupNotFoundException",
			"Endpoint group not found: "+arn, 404)
	}
	if healthPort != nil {
		g.HealthCheckPort = *healthPort
	}
	if healthProtocol != nil && *healthProtocol != "" {
		g.HealthCheckProtocol = *healthProtocol
	}
	if healthPath != nil {
		g.HealthCheckPath = *healthPath
	}
	if healthInterval != nil {
		g.HealthCheckIntervalSeconds = *healthInterval
	}
	if thresholdCount != nil {
		g.ThresholdCount = *thresholdCount
	}
	if trafficDial != nil {
		g.TrafficDialPercentage = *trafficDial
	}
	if portOverrides != nil {
		g.PortOverrides = portOverrides
	}
	if endpoints != nil {
		descriptions := make([]map[string]any, 0, len(endpoints))
		for _, e := range endpoints {
			descriptions = append(descriptions, endpointConfigToDescription(e))
		}
		g.EndpointDescriptions = descriptions
	}
	return g, nil
}

// DeleteEndpointGroup removes an endpoint group.
func (s *Store) DeleteEndpointGroup(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpointGroups[arn]; !ok {
		return service.NewAWSError("EndpointGroupNotFoundException",
			"Endpoint group not found: "+arn, 404)
	}
	delete(s.endpointGroups, arn)
	return nil
}

// ListEndpointGroups returns endpoint groups for a listener.
func (s *Store) ListEndpointGroups(listenerArn string) []*StoredEndpointGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredEndpointGroup, 0)
	for _, g := range s.endpointGroups {
		if listenerArn == "" || g.ListenerArn == listenerArn {
			out = append(out, g)
		}
	}
	return out
}

// AddEndpoints appends endpoints to an endpoint group.
func (s *Store) AddEndpoints(arn string, endpoints []map[string]any) (*StoredEndpointGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.endpointGroups[arn]
	if !ok {
		return nil, service.NewAWSError("EndpointGroupNotFoundException",
			"Endpoint group not found: "+arn, 404)
	}
	for _, e := range endpoints {
		g.EndpointDescriptions = append(g.EndpointDescriptions, endpointConfigToDescription(e))
	}
	return g, nil
}

// RemoveEndpoints removes endpoints from an endpoint group by EndpointId.
func (s *Store) RemoveEndpoints(arn string, identifiers []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.endpointGroups[arn]
	if !ok {
		return service.NewAWSError("EndpointGroupNotFoundException",
			"Endpoint group not found: "+arn, 404)
	}
	ids := make(map[string]struct{}, len(identifiers))
	for _, id := range identifiers {
		if v, ok := id["EndpointId"].(string); ok {
			ids[v] = struct{}{}
		}
	}
	out := g.EndpointDescriptions[:0]
	for _, e := range g.EndpointDescriptions {
		if v, ok := e["EndpointId"].(string); ok {
			if _, drop := ids[v]; drop {
				continue
			}
		}
		out = append(out, e)
	}
	g.EndpointDescriptions = out
	return nil
}

// ── Custom Routing Endpoint Group ────────────────────────────────────────────

// CreateCustomRoutingEndpointGroup stores a new custom routing endpoint group.
func (s *Store) CreateCustomRoutingEndpointGroup(listenerArn, region string, destinations []map[string]any) (*StoredCustomRoutingEndpointGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customListeners[listenerArn]; !ok {
		return nil, service.NewAWSError("ListenerNotFoundException",
			"Custom routing listener not found: "+listenerArn, 404)
	}
	id := generateID()
	arn := s.customEndpointGroupArn(listenerArn, id)
	g := &StoredCustomRoutingEndpointGroup{
		Arn:                     arn,
		ListenerArn:             listenerArn,
		EndpointGroupRegion:     region,
		DestinationDescriptions: destinations,
		EndpointDescriptions:    []map[string]any{},
	}
	s.customEndpointGroups[arn] = g
	return g, nil
}

// GetCustomRoutingEndpointGroup returns a custom routing endpoint group by ARN.
func (s *Store) GetCustomRoutingEndpointGroup(arn string) (*StoredCustomRoutingEndpointGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.customEndpointGroups[arn]
	if !ok {
		return nil, service.NewAWSError("EndpointGroupNotFoundException",
			"Custom routing endpoint group not found: "+arn, 404)
	}
	return g, nil
}

// DeleteCustomRoutingEndpointGroup removes a custom routing endpoint group.
func (s *Store) DeleteCustomRoutingEndpointGroup(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customEndpointGroups[arn]; !ok {
		return service.NewAWSError("EndpointGroupNotFoundException",
			"Custom routing endpoint group not found: "+arn, 404)
	}
	delete(s.customEndpointGroups, arn)
	return nil
}

// ListCustomRoutingEndpointGroups returns custom routing endpoint groups for a listener.
func (s *Store) ListCustomRoutingEndpointGroups(listenerArn string) []*StoredCustomRoutingEndpointGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCustomRoutingEndpointGroup, 0)
	for _, g := range s.customEndpointGroups {
		if listenerArn == "" || g.ListenerArn == listenerArn {
			out = append(out, g)
		}
	}
	return out
}

// AddCustomRoutingEndpoints adds endpoints to a custom routing endpoint group.
func (s *Store) AddCustomRoutingEndpoints(arn string, endpoints []map[string]any) (*StoredCustomRoutingEndpointGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.customEndpointGroups[arn]
	if !ok {
		return nil, service.NewAWSError("EndpointGroupNotFoundException",
			"Custom routing endpoint group not found: "+arn, 404)
	}
	for _, e := range endpoints {
		desc := map[string]any{}
		if v, ok := e["EndpointId"].(string); ok {
			desc["EndpointId"] = v
		}
		g.EndpointDescriptions = append(g.EndpointDescriptions, desc)
	}
	return g, nil
}

// RemoveCustomRoutingEndpoints removes endpoints from a custom routing endpoint group.
func (s *Store) RemoveCustomRoutingEndpoints(arn string, endpointIDs []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.customEndpointGroups[arn]
	if !ok {
		return service.NewAWSError("EndpointGroupNotFoundException",
			"Custom routing endpoint group not found: "+arn, 404)
	}
	ids := make(map[string]struct{}, len(endpointIDs))
	for _, id := range endpointIDs {
		ids[id] = struct{}{}
	}
	out := g.EndpointDescriptions[:0]
	for _, e := range g.EndpointDescriptions {
		if v, ok := e["EndpointId"].(string); ok {
			if _, drop := ids[v]; drop {
				continue
			}
		}
		out = append(out, e)
	}
	g.EndpointDescriptions = out
	return nil
}

// ── BYOIP CIDR ───────────────────────────────────────────────────────────────

// ProvisionByoipCidr provisions a new CIDR.
func (s *Store) ProvisionByoipCidr(cidr string) (*StoredByoipCidr, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byoipCidrs[cidr]; ok {
		return nil, service.NewAWSError("ByoipCidrAlreadyExistsException",
			"CIDR already provisioned: "+cidr, 409)
	}
	c := &StoredByoipCidr{
		Cidr:  cidr,
		State: "READY",
		Events: []map[string]any{
			{"Message": "Provisioned", "Timestamp": time.Now().UTC().Format(time.RFC3339)},
		},
	}
	s.byoipCidrs[cidr] = c
	return c, nil
}

// GetByoipCidr returns a BYOIP CIDR.
func (s *Store) GetByoipCidr(cidr string) (*StoredByoipCidr, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byoipCidrs[cidr]
	if !ok {
		return nil, service.NewAWSError("ByoipCidrNotFoundException",
			"BYOIP CIDR not found: "+cidr, 404)
	}
	return c, nil
}

// SetByoipCidrState updates the state of a BYOIP CIDR.
func (s *Store) SetByoipCidrState(cidr, state string) (*StoredByoipCidr, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byoipCidrs[cidr]
	if !ok {
		return nil, service.NewAWSError("ByoipCidrNotFoundException",
			"BYOIP CIDR not found: "+cidr, 404)
	}
	c.State = state
	c.Events = append(c.Events, map[string]any{
		"Message":   state,
		"Timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	return c, nil
}

// DeleteByoipCidr removes a BYOIP CIDR.
func (s *Store) DeleteByoipCidr(cidr string) (*StoredByoipCidr, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byoipCidrs[cidr]
	if !ok {
		return nil, service.NewAWSError("ByoipCidrNotFoundException",
			"BYOIP CIDR not found: "+cidr, 404)
	}
	delete(s.byoipCidrs, cidr)
	c.State = "DEPROVISIONED"
	return c, nil
}

// ListByoipCidrs returns all BYOIP CIDRs.
func (s *Store) ListByoipCidrs() []*StoredByoipCidr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredByoipCidr, 0, len(s.byoipCidrs))
	for _, c := range s.byoipCidrs {
		out = append(out, c)
	}
	return out
}

// ── Cross Account Attachment ─────────────────────────────────────────────────

// CreateCrossAccountAttachment stores a new cross-account attachment.
func (s *Store) CreateCrossAccountAttachment(name string, principals []string, resources []map[string]any) *StoredCrossAccountAttachment {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	arn := s.attachmentArn(id)
	now := time.Now().UTC()
	a := &StoredCrossAccountAttachment{
		Arn:              arn,
		Name:             name,
		Principals:       principals,
		Resources:        resources,
		CreatedTime:      now,
		LastModifiedTime: now,
	}
	s.attachments[arn] = a
	return a
}

// GetCrossAccountAttachment returns an attachment by ARN.
func (s *Store) GetCrossAccountAttachment(arn string) (*StoredCrossAccountAttachment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.attachments[arn]
	if !ok {
		return nil, service.NewAWSError("AttachmentNotFoundException",
			"Cross-account attachment not found: "+arn, 404)
	}
	return a, nil
}

// UpdateCrossAccountAttachment updates fields on an attachment.
func (s *Store) UpdateCrossAccountAttachment(arn string, name *string, addPrincipals, removePrincipals []string, addResources, removeResources []map[string]any) (*StoredCrossAccountAttachment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attachments[arn]
	if !ok {
		return nil, service.NewAWSError("AttachmentNotFoundException",
			"Cross-account attachment not found: "+arn, 404)
	}
	if name != nil && *name != "" {
		a.Name = *name
	}
	if len(addPrincipals) > 0 {
		a.Principals = append(a.Principals, addPrincipals...)
	}
	if len(removePrincipals) > 0 {
		removed := make(map[string]struct{}, len(removePrincipals))
		for _, p := range removePrincipals {
			removed[p] = struct{}{}
		}
		out := a.Principals[:0]
		for _, p := range a.Principals {
			if _, drop := removed[p]; drop {
				continue
			}
			out = append(out, p)
		}
		a.Principals = out
	}
	if len(addResources) > 0 {
		a.Resources = append(a.Resources, addResources...)
	}
	if len(removeResources) > 0 {
		out := a.Resources[:0]
		for _, r := range a.Resources {
			if !resourceInList(r, removeResources) {
				out = append(out, r)
			}
		}
		a.Resources = out
	}
	a.LastModifiedTime = time.Now().UTC()
	return a, nil
}

// DeleteCrossAccountAttachment removes an attachment.
func (s *Store) DeleteCrossAccountAttachment(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.attachments[arn]; !ok {
		return service.NewAWSError("AttachmentNotFoundException",
			"Cross-account attachment not found: "+arn, 404)
	}
	delete(s.attachments, arn)
	return nil
}

// ListCrossAccountAttachments returns all attachments.
func (s *Store) ListCrossAccountAttachments() []*StoredCrossAccountAttachment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCrossAccountAttachment, 0, len(s.attachments))
	for _, a := range s.attachments {
		out = append(out, a)
	}
	return out
}

// ListCrossAccountResourceAccounts returns the unique account IDs from all attachment principals.
func (s *Store) ListCrossAccountResourceAccounts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, a := range s.attachments {
		for _, p := range a.Principals {
			seen[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// ListCrossAccountResources returns resources across all attachments matching the given account.
func (s *Store) ListCrossAccountResources(ownerAccountID string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0)
	for _, a := range s.attachments {
		matched := ownerAccountID == ""
		for _, p := range a.Principals {
			if p == ownerAccountID {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for _, r := range a.Resources {
			entry := map[string]any{
				"AttachmentArn": a.Arn,
			}
			if v, ok := r["EndpointId"]; ok {
				entry["EndpointId"] = v
			}
			if v, ok := r["Cidr"]; ok {
				entry["Cidr"] = v
			}
			out = append(out, entry)
		}
	}
	return out
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// TagResource adds tags to an ARN.
func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

// UntagResource removes tag keys from an ARN.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns all tags for an ARN.
func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.tags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (s *Store) acceleratorArn(id string) string {
	return fmt.Sprintf("arn:aws:globalaccelerator::%s:accelerator/%s", s.accountID, id)
}

func (s *Store) customAcceleratorArn(id string) string {
	return fmt.Sprintf("arn:aws:globalaccelerator::%s:accelerator/%s", s.accountID, id)
}

func (s *Store) listenerArn(acceleratorArn, id string) string {
	return fmt.Sprintf("%s/listener/%s", acceleratorArn, id[:8])
}

func (s *Store) customListenerArn(acceleratorArn, id string) string {
	return fmt.Sprintf("%s/listener/%s", acceleratorArn, id[:8])
}

func (s *Store) endpointGroupArn(listenerArn, id string) string {
	return fmt.Sprintf("%s/endpoint-group/%s", listenerArn, id[:8])
}

func (s *Store) customEndpointGroupArn(listenerArn, id string) string {
	return fmt.Sprintf("%s/endpoint-group/%s", listenerArn, id[:8])
}

func (s *Store) attachmentArn(id string) string {
	return fmt.Sprintf("arn:aws:globalaccelerator::%s:attachment/%s", s.accountID, id)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func defaultIPv4Addresses() []string {
	return []string{"75.2.0.1", "99.83.0.1"}
}

func buildIpSets(ipAddressType string, addresses []string) []map[string]any {
	if ipAddressType == "DUAL_STACK" {
		return []map[string]any{
			{
				"IpAddressFamily": "IPv4",
				"IpFamily":        "IPv4",
				"IpAddresses":     addresses,
			},
			{
				"IpAddressFamily": "IPv6",
				"IpFamily":        "IPv6",
				"IpAddresses":     []string{"2600:9000:1234::1", "2600:9000:1234::2"},
			},
		}
	}
	return []map[string]any{
		{
			"IpAddressFamily": "IPv4",
			"IpFamily":        "IPv4",
			"IpAddresses":     addresses,
		},
	}
}

func endpointConfigToDescription(cfg map[string]any) map[string]any {
	desc := map[string]any{
		"HealthState":  "HEALTHY",
		"HealthReason": "",
	}
	if v, ok := cfg["EndpointId"].(string); ok {
		desc["EndpointId"] = v
	}
	if v, ok := cfg["Weight"].(float64); ok {
		desc["Weight"] = int(v)
	} else if v, ok := cfg["Weight"].(int); ok {
		desc["Weight"] = v
	} else {
		desc["Weight"] = 128
	}
	if v, ok := cfg["ClientIPPreservationEnabled"].(bool); ok {
		desc["ClientIPPreservationEnabled"] = v
	}
	return desc
}

func resourceInList(r map[string]any, list []map[string]any) bool {
	rid, _ := r["EndpointId"].(string)
	rcidr, _ := r["Cidr"].(string)
	for _, x := range list {
		xid, _ := x["EndpointId"].(string)
		xcidr, _ := x["Cidr"].(string)
		if rid != "" && rid == xid {
			return true
		}
		if rcidr != "" && rcidr == xcidr {
			return true
		}
	}
	return false
}
