package account

import (
	"sync"
)

// ContactInformation represents account contact information.
type ContactInformation struct {
	FullName          string
	AddressLine1      string
	AddressLine2      string
	AddressLine3      string
	City              string
	StateOrRegion     string
	PostalCode        string
	CountryCode       string
	PhoneNumber       string
	CompanyName       string
	DistrictOrCounty  string
	WebsiteURL        string
}

// AlternateContact represents an alternate contact.
type AlternateContact struct {
	AlternateContactType string // BILLING, OPERATIONS, SECURITY
	Name                 string
	Title                string
	EmailAddress         string
	PhoneNumber          string
}

// RegionInfo represents a region's opt-in status.
type RegionInfo struct {
	RegionName      string
	RegionOptStatus string // ENABLED, ENABLING, DISABLING, DISABLED
	Description     string
}

// Store manages Account resources in memory.
type Store struct {
	mu                sync.RWMutex
	contactInfo       *ContactInformation
	alternateContacts map[string]*AlternateContact // type -> contact
	regions           map[string]*RegionInfo       // regionName -> info
	accountID         string
	region            string
}

// NewStore returns a new empty Account Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		contactInfo: &ContactInformation{
			FullName:     "CloudMock Admin",
			AddressLine1: "123 Mock Street",
			City:         "Seattle",
			StateOrRegion: "WA",
			PostalCode:   "98101",
			CountryCode:  "US",
			PhoneNumber:  "+1-555-0100",
			CompanyName:  "CloudMock Inc.",
		},
		alternateContacts: make(map[string]*AlternateContact),
		regions:           make(map[string]*RegionInfo),
		accountID:         accountID,
		region:            region,
	}
	s.initRegions()
	return s
}

// regionDescriptions provides realistic region descriptions.
var regionDescriptions = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"eu-west-1":      "Europe (Ireland)",
	"eu-west-2":      "Europe (London)",
	"eu-west-3":      "Europe (Paris)",
	"eu-central-1":   "Europe (Frankfurt)",
	"eu-north-1":     "Europe (Stockholm)",
	"eu-south-1":     "Europe (Milan)",
	"eu-south-2":     "Europe (Spain)",
	"eu-central-2":   "Europe (Zurich)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-southeast-3": "Asia Pacific (Jakarta)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-south-2":     "Asia Pacific (Hyderabad)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"sa-east-1":      "South America (Sao Paulo)",
	"ca-central-1":   "Canada (Central)",
	"af-south-1":     "Africa (Cape Town)",
	"me-south-1":     "Middle East (Bahrain)",
	"me-central-1":   "Middle East (UAE)",
	"il-central-1":   "Israel (Tel Aviv)",
}

func (s *Store) initRegions() {
	enabledRegions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
		"sa-east-1", "ca-central-1",
	}
	optInRegions := []string{
		"af-south-1", "ap-east-1", "ap-south-2", "ap-southeast-3",
		"eu-south-1", "eu-south-2", "eu-central-2", "me-south-1", "me-central-1",
		"il-central-1",
	}
	for _, r := range enabledRegions {
		s.regions[r] = &RegionInfo{RegionName: r, RegionOptStatus: "ENABLED", Description: regionDescriptions[r]}
	}
	for _, r := range optInRegions {
		s.regions[r] = &RegionInfo{RegionName: r, RegionOptStatus: "DISABLED", Description: regionDescriptions[r]}
	}
}

// GetContactInformation returns the account contact information.
func (s *Store) GetContactInformation() *ContactInformation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.contactInfo
}

// PutContactInformation updates the account contact information.
func (s *Store) PutContactInformation(info *ContactInformation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contactInfo = info
}

// GetAlternateContact retrieves an alternate contact by type.
func (s *Store) GetAlternateContact(contactType string) (*AlternateContact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.alternateContacts[contactType]
	return c, ok
}

// PutAlternateContact creates or updates an alternate contact.
func (s *Store) PutAlternateContact(contact *AlternateContact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alternateContacts[contact.AlternateContactType] = contact
}

// DeleteAlternateContact removes an alternate contact.
func (s *Store) DeleteAlternateContact(contactType string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.alternateContacts[contactType]; !ok {
		return false
	}
	delete(s.alternateContacts, contactType)
	return true
}

// GetRegionOptStatus returns the opt-in status for a region.
func (s *Store) GetRegionOptStatus(regionName string) (*RegionInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.regions[regionName]
	return r, ok
}

// ListRegions returns all regions.
func (s *Store) ListRegions() []*RegionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RegionInfo, 0, len(s.regions))
	for _, r := range s.regions {
		out = append(out, r)
	}
	return out
}

// EnableRegion enables an opt-in region.
func (s *Store) EnableRegion(regionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.regions[regionName]
	if !ok {
		return nil
	}
	r.RegionOptStatus = "ENABLING"
	// In real AWS this would be async, for mock we set it directly
	r.RegionOptStatus = "ENABLED"
	return nil
}

// DisableRegion disables an opt-in region.
func (s *Store) DisableRegion(regionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.regions[regionName]
	if !ok {
		return nil
	}
	r.RegionOptStatus = "DISABLING"
	r.RegionOptStatus = "DISABLED"
	return nil
}
