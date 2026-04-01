package iotwireless

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

type WirelessDevice struct {
	Id                string
	Arn               string
	Name              string
	Type              string // Sidewalk | LoRaWAN
	DestinationName   string
	Description       string
	LoRaWAN           map[string]any
	Sidewalk          map[string]any
	ThingArn          string
	CreationTime      time.Time
	Tags              map[string]string
}

type WirelessGateway struct {
	Id              string
	Arn             string
	Name            string
	Description     string
	LoRaWAN         map[string]any
	ThingArn        string
	CreationTime    time.Time
	LastUplinkReceivedAt *time.Time
	Tags            map[string]string
}

type DeviceProfile struct {
	Id           string
	Arn          string
	Name         string
	LoRaWAN      map[string]any
	Sidewalk     map[string]any
	CreationTime time.Time
	Tags         map[string]string
}

type ServiceProfile struct {
	Id           string
	Arn          string
	Name         string
	LoRaWAN      map[string]any
	CreationTime time.Time
	Tags         map[string]string
}

type Destination struct {
	Name            string
	Arn             string
	Expression      string
	ExpressionType  string
	Description     string
	RoleArn         string
	CreationTime    time.Time
	Tags            map[string]string
}

type Store struct {
	mu              sync.RWMutex
	wirelessDevices map[string]*WirelessDevice   // keyed by ID
	wirelessGateways map[string]*WirelessGateway // keyed by ID
	deviceProfiles  map[string]*DeviceProfile    // keyed by ID
	serviceProfiles map[string]*ServiceProfile   // keyed by ID
	destinations    map[string]*Destination      // keyed by name
	tagsByArn       map[string]map[string]string
	accountID       string
	region          string
}

func NewStore(accountID, region string) *Store {
	return &Store{
		wirelessDevices:  make(map[string]*WirelessDevice),
		wirelessGateways: make(map[string]*WirelessGateway),
		deviceProfiles:   make(map[string]*DeviceProfile),
		serviceProfiles:  make(map[string]*ServiceProfile),
		destinations:     make(map[string]*Destination),
		tagsByArn:        make(map[string]map[string]string),
		accountID:        accountID,
		region:           region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) deviceARN(id string) string {
	return fmt.Sprintf("arn:aws:iotwireless:%s:%s:WirelessDevice/%s", s.region, s.accountID, id)
}
func (s *Store) gatewayARN(id string) string {
	return fmt.Sprintf("arn:aws:iotwireless:%s:%s:WirelessGateway/%s", s.region, s.accountID, id)
}
func (s *Store) deviceProfileARN(id string) string {
	return fmt.Sprintf("arn:aws:iotwireless:%s:%s:DeviceProfile/%s", s.region, s.accountID, id)
}
func (s *Store) serviceProfileARN(id string) string {
	return fmt.Sprintf("arn:aws:iotwireless:%s:%s:ServiceProfile/%s", s.region, s.accountID, id)
}
func (s *Store) destinationARN(name string) string {
	return fmt.Sprintf("arn:aws:iotwireless:%s:%s:Destination/%s", s.region, s.accountID, name)
}

// Wireless devices.

func (s *Store) CreateWirelessDevice(name, deviceType, destinationName, description string, loRaWAN, sidewalk map[string]any, tags map[string]string) (*WirelessDevice, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if deviceType == "" {
		deviceType = "LoRaWAN"
	}
	validTypes := map[string]bool{"LoRaWAN": true, "Sidewalk": true}
	if !validTypes[deviceType] {
		return nil, service.NewAWSError("ValidationException",
			fmt.Sprintf("Invalid device type: %s. Must be LoRaWAN or Sidewalk.", deviceType), http.StatusBadRequest)
	}
	id := newUUID()
	arn := s.deviceARN(id)
	d := &WirelessDevice{
		Id:              id,
		Arn:             arn,
		Name:            name,
		Type:            deviceType,
		DestinationName: destinationName,
		Description:     description,
		LoRaWAN:         loRaWAN,
		Sidewalk:        sidewalk,
		CreationTime:    time.Now().UTC(),
		Tags:            tags,
	}
	s.wirelessDevices[id] = d
	s.tagsByArn[arn] = tags
	return d, nil
}

func (s *Store) GetWirelessDevice(id string) (*WirelessDevice, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.wirelessDevices[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless device %s not found", id), http.StatusNotFound)
	}
	return d, nil
}

func (s *Store) ListWirelessDevices() []*WirelessDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*WirelessDevice, 0, len(s.wirelessDevices))
	for _, d := range s.wirelessDevices {
		out = append(out, d)
	}
	return out
}

func (s *Store) DeleteWirelessDevice(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.wirelessDevices[id]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless device %s not found", id), http.StatusNotFound)
	}
	delete(s.wirelessDevices, id)
	delete(s.tagsByArn, d.Arn)
	return nil
}

func (s *Store) UpdateWirelessDevice(id, name, destinationName, description string) (*WirelessDevice, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.wirelessDevices[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless device %s not found", id), http.StatusNotFound)
	}
	if name != "" {
		d.Name = name
	}
	if destinationName != "" {
		d.DestinationName = destinationName
	}
	if description != "" {
		d.Description = description
	}
	return d, nil
}

// Wireless gateways.

func (s *Store) CreateWirelessGateway(name, description string, loRaWAN map[string]any, tags map[string]string) (*WirelessGateway, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newUUID()
	arn := s.gatewayARN(id)
	gw := &WirelessGateway{
		Id:           id,
		Arn:          arn,
		Name:         name,
		Description:  description,
		LoRaWAN:      loRaWAN,
		CreationTime: time.Now().UTC(),
		Tags:         tags,
	}
	s.wirelessGateways[id] = gw
	s.tagsByArn[arn] = tags
	return gw, nil
}

func (s *Store) GetWirelessGateway(id string) (*WirelessGateway, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	gw, ok := s.wirelessGateways[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless gateway %s not found", id), http.StatusNotFound)
	}
	return gw, nil
}

func (s *Store) ListWirelessGateways() []*WirelessGateway {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*WirelessGateway, 0, len(s.wirelessGateways))
	for _, gw := range s.wirelessGateways {
		out = append(out, gw)
	}
	return out
}

func (s *Store) DeleteWirelessGateway(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	gw, ok := s.wirelessGateways[id]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless gateway %s not found", id), http.StatusNotFound)
	}
	delete(s.wirelessGateways, id)
	delete(s.tagsByArn, gw.Arn)
	return nil
}

func (s *Store) UpdateWirelessGateway(id, name, description string) (*WirelessGateway, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gw, ok := s.wirelessGateways[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Wireless gateway %s not found", id), http.StatusNotFound)
	}
	if name != "" {
		gw.Name = name
	}
	if description != "" {
		gw.Description = description
	}
	return gw, nil
}

// Device profiles.

func (s *Store) CreateDeviceProfile(name string, loRaWAN, sidewalk map[string]any, tags map[string]string) (*DeviceProfile, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	// Validate frequency band if LoRaWAN config is provided
	if loRaWAN != nil {
		if rfRegion, ok := loRaWAN["RfRegion"].(string); ok && rfRegion != "" {
			validBands := map[string]bool{
				"US915": true, "EU868": true, "AU915": true, "AS923-1": true,
				"AS923-2": true, "AS923-3": true, "AS923-4": true, "CN470": true,
				"CN779": true, "EU433": true, "IN865": true, "KR920": true, "RU864": true,
			}
			if !validBands[rfRegion] {
				return nil, service.NewAWSError("ValidationException",
					fmt.Sprintf("Invalid RfRegion: %s.", rfRegion), http.StatusBadRequest)
			}
		}
	}
	id := newUUID()
	arn := s.deviceProfileARN(id)
	dp := &DeviceProfile{
		Id:           id,
		Arn:          arn,
		Name:         name,
		LoRaWAN:      loRaWAN,
		Sidewalk:     sidewalk,
		CreationTime: time.Now().UTC(),
		Tags:         tags,
	}
	s.deviceProfiles[id] = dp
	s.tagsByArn[arn] = tags
	return dp, nil
}

func (s *Store) GetDeviceProfile(id string) (*DeviceProfile, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dp, ok := s.deviceProfiles[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Device profile %s not found", id), http.StatusNotFound)
	}
	return dp, nil
}

func (s *Store) ListDeviceProfiles() []*DeviceProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*DeviceProfile, 0, len(s.deviceProfiles))
	for _, dp := range s.deviceProfiles {
		out = append(out, dp)
	}
	return out
}

func (s *Store) DeleteDeviceProfile(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	dp, ok := s.deviceProfiles[id]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Device profile %s not found", id), http.StatusNotFound)
	}
	delete(s.deviceProfiles, id)
	delete(s.tagsByArn, dp.Arn)
	return nil
}

// Service profiles.

func (s *Store) CreateServiceProfile(name string, loRaWAN map[string]any, tags map[string]string) (*ServiceProfile, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newUUID()
	arn := s.serviceProfileARN(id)
	sp := &ServiceProfile{
		Id:           id,
		Arn:          arn,
		Name:         name,
		LoRaWAN:      loRaWAN,
		CreationTime: time.Now().UTC(),
		Tags:         tags,
	}
	s.serviceProfiles[id] = sp
	s.tagsByArn[arn] = tags
	return sp, nil
}

func (s *Store) GetServiceProfile(id string) (*ServiceProfile, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sp, ok := s.serviceProfiles[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Service profile %s not found", id), http.StatusNotFound)
	}
	return sp, nil
}

func (s *Store) ListServiceProfiles() []*ServiceProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ServiceProfile, 0, len(s.serviceProfiles))
	for _, sp := range s.serviceProfiles {
		out = append(out, sp)
	}
	return out
}

func (s *Store) DeleteServiceProfile(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.serviceProfiles[id]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Service profile %s not found", id), http.StatusNotFound)
	}
	delete(s.serviceProfiles, id)
	delete(s.tagsByArn, sp.Arn)
	return nil
}

// Destinations.

func (s *Store) CreateDestination(name, expression, expressionType, description, roleArn string, tags map[string]string) (*Destination, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.destinations[name]; exists {
		return nil, service.NewAWSError("ConflictException",
			fmt.Sprintf("Destination %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if expressionType == "" {
		expressionType = "RuleName"
	}
	arn := s.destinationARN(name)
	d := &Destination{
		Name:           name,
		Arn:            arn,
		Expression:     expression,
		ExpressionType: expressionType,
		Description:    description,
		RoleArn:        roleArn,
		CreationTime:   time.Now().UTC(),
		Tags:           tags,
	}
	s.destinations[name] = d
	s.tagsByArn[arn] = tags
	return d, nil
}

func (s *Store) GetDestination(name string) (*Destination, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.destinations[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Destination %s not found", name), http.StatusNotFound)
	}
	return d, nil
}

func (s *Store) ListDestinations() []*Destination {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Destination, 0, len(s.destinations))
	for _, d := range s.destinations {
		out = append(out, d)
	}
	return out
}

func (s *Store) UpdateDestination(name, expression, expressionType, description, roleArn string) (*Destination, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.destinations[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Destination %s not found", name), http.StatusNotFound)
	}
	if expression != "" {
		d.Expression = expression
	}
	if expressionType != "" {
		d.ExpressionType = expressionType
	}
	if description != "" {
		d.Description = description
	}
	if roleArn != "" {
		d.RoleArn = roleArn
	}
	return d, nil
}

func (s *Store) DeleteDestination(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.destinations[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Destination %s not found", name), http.StatusNotFound)
	}
	delete(s.destinations, name)
	delete(s.tagsByArn, d.Arn)
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
