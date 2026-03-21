package route53

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// HostedZone represents an AWS Route 53 hosted zone.
type HostedZone struct {
	Id              string // e.g. /hostedzone/Z1234567890ABC
	Name            string
	CallerReference string
	Config          ZoneConfig
	RecordSets      []ResourceRecordSet
}

// ZoneConfig holds optional config for a hosted zone.
type ZoneConfig struct {
	Comment     string
	PrivateZone bool
}

// ResourceRecordSet is a DNS record set in a hosted zone.
type ResourceRecordSet struct {
	Name            string
	Type            string // A, AAAA, CNAME, MX, TXT, NS, SOA, etc.
	TTL             int64
	ResourceRecords []ResourceRecord
}

// ResourceRecord holds a single DNS record value.
type ResourceRecord struct {
	Value string
}

// ZoneStore manages Route 53 hosted zones, keyed by zone ID (short form, e.g. "Z1234567890ABC").
type ZoneStore struct {
	mu    sync.RWMutex
	zones map[string]*HostedZone // key is short ID like "Z1234567890ABC"
}

// NewStore returns an empty ZoneStore.
func NewStore() *ZoneStore {
	return &ZoneStore{
		zones: make(map[string]*HostedZone),
	}
}

// generateZoneID generates a random Route 53-style zone ID.
func generateZoneID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 14)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return "Z" + string(b)
}

// defaultNameservers returns a set of AWS-style NS records for a zone.
func defaultNameservers(zoneName string) []ResourceRecord {
	return []ResourceRecord{
		{Value: fmt.Sprintf("ns-1.awsdns-1.org")},
		{Value: fmt.Sprintf("ns-2.awsdns-2.net")},
		{Value: fmt.Sprintf("ns-3.awsdns-3.co.uk")},
		{Value: fmt.Sprintf("ns-4.awsdns-4.com")},
	}
}

// defaultSOA returns a SOA record for a zone.
func defaultSOA(zoneName string) ResourceRecord {
	return ResourceRecord{
		Value: fmt.Sprintf("ns-1.awsdns-1.org. awsdns-hostmaster.amazon.com. 1 7200 900 1209600 86400"),
	}
}

// CreateZone creates a new hosted zone with auto-generated NS and SOA records.
func (s *ZoneStore) CreateZone(name, callerRef string, config ZoneConfig) (*HostedZone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure name ends with a dot (canonical form).
	if len(name) > 0 && name[len(name)-1] != '.' {
		name = name + "."
	}

	id := generateZoneID()
	zone := &HostedZone{
		Id:              "/hostedzone/" + id,
		Name:            name,
		CallerReference: callerRef,
		Config:          config,
		RecordSets: []ResourceRecordSet{
			{
				Name:            name,
				Type:            "NS",
				TTL:             172800,
				ResourceRecords: defaultNameservers(name),
			},
			{
				Name:            name,
				Type:            "SOA",
				TTL:             900,
				ResourceRecords: []ResourceRecord{defaultSOA(name)},
			},
		},
	}
	s.zones[id] = zone
	return zone, nil
}

// GetZone retrieves a hosted zone by short ID.
func (s *ZoneStore) GetZone(id string) (*HostedZone, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	z, ok := s.zones[id]
	return z, ok
}

// DeleteZone removes a hosted zone by short ID. Returns false if not found.
func (s *ZoneStore) DeleteZone(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.zones[id]; !ok {
		return false
	}
	delete(s.zones, id)
	return true
}

// ListZones returns all hosted zones.
func (s *ZoneStore) ListZones() []*HostedZone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*HostedZone, 0, len(s.zones))
	for _, z := range s.zones {
		out = append(out, z)
	}
	return out
}

// ChangeRecords applies a list of changes (CREATE/DELETE/UPSERT) to a zone's record sets.
// Returns an error if a CREATE conflicts or a DELETE targets a non-existent record.
func (s *ZoneStore) ChangeRecords(id string, changes []Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	zone, ok := s.zones[id]
	if !ok {
		return fmt.Errorf("no such hosted zone: %s", id)
	}

	for _, ch := range changes {
		switch ch.Action {
		case "CREATE":
			if err := applyCreate(zone, ch.RRSet); err != nil {
				return err
			}
		case "DELETE":
			if err := applyDelete(zone, ch.RRSet); err != nil {
				return err
			}
		case "UPSERT":
			applyUpsert(zone, ch.RRSet)
		default:
			return fmt.Errorf("unknown action: %s", ch.Action)
		}
	}
	return nil
}

// Change represents a single DNS change operation.
type Change struct {
	Action string
	RRSet  ResourceRecordSet
}

func findRRSet(zone *HostedZone, name, rrType string) int {
	for i, rs := range zone.RecordSets {
		if rs.Name == name && rs.Type == rrType {
			return i
		}
	}
	return -1
}

func applyCreate(zone *HostedZone, rrs ResourceRecordSet) error {
	if idx := findRRSet(zone, rrs.Name, rrs.Type); idx >= 0 {
		return fmt.Errorf("record set already exists: %s %s", rrs.Name, rrs.Type)
	}
	zone.RecordSets = append(zone.RecordSets, rrs)
	return nil
}

func applyDelete(zone *HostedZone, rrs ResourceRecordSet) error {
	idx := findRRSet(zone, rrs.Name, rrs.Type)
	if idx < 0 {
		return fmt.Errorf("record set not found: %s %s", rrs.Name, rrs.Type)
	}
	zone.RecordSets = append(zone.RecordSets[:idx], zone.RecordSets[idx+1:]...)
	return nil
}

func applyUpsert(zone *HostedZone, rrs ResourceRecordSet) {
	idx := findRRSet(zone, rrs.Name, rrs.Type)
	if idx >= 0 {
		zone.RecordSets[idx] = rrs
	} else {
		zone.RecordSets = append(zone.RecordSets, rrs)
	}
}

// ListRecords returns all resource record sets for a zone.
func (s *ZoneStore) ListRecords(id string) ([]ResourceRecordSet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	zone, ok := s.zones[id]
	if !ok {
		return nil, false
	}
	out := make([]ResourceRecordSet, len(zone.RecordSets))
	copy(out, zone.RecordSets)
	return out, true
}
