package guardduty

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredDetector models a GuardDuty detector.
type StoredDetector struct {
	DetectorID                 string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	FindingPublishingFrequency string
	ServiceRole                string
	Status                     string
	Tags                       map[string]string
	DataSources                map[string]any
	Features                   []map[string]any
}

// StoredFilter models a GuardDuty findings filter.
type StoredFilter struct {
	DetectorID      string
	Name            string
	Description     string
	Action          string
	Rank            int
	FindingCriteria map[string]any
	Tags            map[string]string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StoredIPSet models an IP set used to allow-list traffic.
type StoredIPSet struct {
	DetectorID          string
	IPSetID             string
	Name                string
	Format              string
	Location            string
	Status              string
	ExpectedBucketOwner string
	Tags                map[string]string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// StoredThreatIntelSet models a threat intelligence set.
type StoredThreatIntelSet struct {
	DetectorID          string
	ThreatIntelSetID    string
	Name                string
	Format              string
	Location            string
	Status              string
	ExpectedBucketOwner string
	Tags                map[string]string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// StoredEntitySet models a threat or trusted entity set.
type StoredEntitySet struct {
	DetectorID          string
	EntitySetID         string
	Kind                string // "threat" or "trusted"
	Name                string
	Format              string
	Location            string
	Status              string
	ExpectedBucketOwner string
	Tags                map[string]string
	ErrorDetails        string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// StoredMember models a GuardDuty member account.
type StoredMember struct {
	DetectorID         string
	AccountID          string
	Email              string
	RelationshipStatus string
	MasterID           string
	AdministratorID    string
	InvitedAt          time.Time
	UpdatedAt          time.Time
	Tags               map[string]string
}

// StoredInvitation models an invitation to a member account.
type StoredInvitation struct {
	AccountID          string
	InvitationID       string
	RelationshipStatus string
	InvitedAt          time.Time
	AdministratorID    string
}

// StoredPublishingDestination models a destination for findings publishing.
type StoredPublishingDestination struct {
	DetectorID                      string
	DestinationID                   string
	DestinationType                 string
	Status                          string
	DestinationProperties           map[string]any
	PublishingFailureStartTimestamp int64
	Tags                            map[string]string
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
}

// StoredFinding models a GuardDuty finding.
type StoredFinding struct {
	DetectorID    string
	ID            string
	Arn           string
	AccountID     string
	SchemaVersion string
	Type          string
	Title         string
	Description   string
	Severity      float64
	Confidence    float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Region        string
	Partition     string
	Service       map[string]any
	Resource      map[string]any
	Archived      bool
	Feedback      string
}

// StoredMalwareScan models a malware scan.
type StoredMalwareScan struct {
	ScanID         string
	DetectorID     string
	ResourceArn    string
	ResourceType   string
	ScanType       string
	ScanStatus     string
	ScanStartedAt  time.Time
	ScanEndedAt    time.Time
	TriggerDetails map[string]any
	ResourceDetails map[string]any
}

// StoredMalwareProtectionPlan models a malware protection plan.
type StoredMalwareProtectionPlan struct {
	PlanID            string
	Arn               string
	Role              string
	ProtectedResource map[string]any
	Actions           map[string]any
	Status            string
	StatusReasons     []map[string]any
	Tags              map[string]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// StoredMalwareScanSettings models per-detector malware scan settings.
type StoredMalwareScanSettings struct {
	EbsSnapshotPreservation string
	ScanResourceCriteria    map[string]any
}

// StoredOrganizationConfig models the org-level configuration.
type StoredOrganizationConfig struct {
	AutoEnable                    bool
	AutoEnableOrganizationMembers string
	MemberAccountLimitReached     bool
	DataSources                   map[string]any
	Features                      []map[string]any
}

// ── Store ────────────────────────────────────────────────────────────────────

// Store is the in-memory data store for GuardDuty resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	detectors             map[string]*StoredDetector
	filters               map[string]map[string]*StoredFilter        // detectorID -> name -> filter
	ipSets                map[string]map[string]*StoredIPSet         // detectorID -> id -> ipset
	threatIntelSets       map[string]map[string]*StoredThreatIntelSet
	threatEntitySets      map[string]map[string]*StoredEntitySet
	trustedEntitySets     map[string]map[string]*StoredEntitySet
	members               map[string]map[string]*StoredMember // detectorID -> accountID -> member
	memberDetectors       map[string]map[string]map[string]any // detectorID -> accountID -> data sources
	invitations           map[string]*StoredInvitation         // accountID -> invitation
	publishingDest        map[string]map[string]*StoredPublishingDestination
	findings              map[string]map[string]*StoredFinding // detectorID -> findingID -> finding
	malwareScans          map[string]*StoredMalwareScan        // scanID -> scan
	malwareProtectionPlan map[string]*StoredMalwareProtectionPlan
	malwareScanSettings   map[string]*StoredMalwareScanSettings // detectorID -> settings
	orgConfig             map[string]*StoredOrganizationConfig  // detectorID -> org config
	orgAdminAccounts      map[string]string                     // accountID -> status
	tags                  map[string]map[string]string          // arn -> tags
	masterAccount         *StoredMember                         // simulated master/administrator
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:             accountID,
		region:                region,
		detectors:             make(map[string]*StoredDetector),
		filters:               make(map[string]map[string]*StoredFilter),
		ipSets:                make(map[string]map[string]*StoredIPSet),
		threatIntelSets:       make(map[string]map[string]*StoredThreatIntelSet),
		threatEntitySets:      make(map[string]map[string]*StoredEntitySet),
		trustedEntitySets:     make(map[string]map[string]*StoredEntitySet),
		members:               make(map[string]map[string]*StoredMember),
		memberDetectors:       make(map[string]map[string]map[string]any),
		invitations:           make(map[string]*StoredInvitation),
		publishingDest:        make(map[string]map[string]*StoredPublishingDestination),
		findings:              make(map[string]map[string]*StoredFinding),
		malwareScans:          make(map[string]*StoredMalwareScan),
		malwareProtectionPlan: make(map[string]*StoredMalwareProtectionPlan),
		malwareScanSettings:   make(map[string]*StoredMalwareScanSettings),
		orgConfig:             make(map[string]*StoredOrganizationConfig),
		orgAdminAccounts:      make(map[string]string),
		tags:                  make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state. Implements the admin Resettable contract.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detectors = make(map[string]*StoredDetector)
	s.filters = make(map[string]map[string]*StoredFilter)
	s.ipSets = make(map[string]map[string]*StoredIPSet)
	s.threatIntelSets = make(map[string]map[string]*StoredThreatIntelSet)
	s.threatEntitySets = make(map[string]map[string]*StoredEntitySet)
	s.trustedEntitySets = make(map[string]map[string]*StoredEntitySet)
	s.members = make(map[string]map[string]*StoredMember)
	s.memberDetectors = make(map[string]map[string]map[string]any)
	s.invitations = make(map[string]*StoredInvitation)
	s.publishingDest = make(map[string]map[string]*StoredPublishingDestination)
	s.findings = make(map[string]map[string]*StoredFinding)
	s.malwareScans = make(map[string]*StoredMalwareScan)
	s.malwareProtectionPlan = make(map[string]*StoredMalwareProtectionPlan)
	s.malwareScanSettings = make(map[string]*StoredMalwareScanSettings)
	s.orgConfig = make(map[string]*StoredOrganizationConfig)
	s.orgAdminAccounts = make(map[string]string)
	s.tags = make(map[string]map[string]string)
	s.masterAccount = nil
}

// AccountID exposes the configured account ID for handlers.
func (s *Store) AccountID() string { return s.accountID }

// Region exposes the configured region for handlers.
func (s *Store) Region() string { return s.region }

// ── Helpers ──────────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func errNotFound(format string, args ...any) *service.AWSError {
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf(format, args...), http.StatusBadRequest)
}

func errBadRequest(format string, args ...any) *service.AWSError {
	return service.NewAWSError("BadRequestException",
		fmt.Sprintf(format, args...), http.StatusBadRequest)
}

func errInternal(format string, args ...any) *service.AWSError {
	return service.NewAWSError("InternalServerErrorException",
		fmt.Sprintf(format, args...), http.StatusInternalServerError)
}

func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *Store) buildArn(parts ...string) string {
	return fmt.Sprintf("arn:aws:guardduty:%s:%s:%s", s.region, s.accountID, strings.Join(parts, "/"))
}

// ── Detectors ────────────────────────────────────────────────────────────────

// CreateDetector creates and stores a detector.
func (s *Store) CreateDetector(enable bool, freq string, tags map[string]string, dataSources map[string]any, features []map[string]any) *StoredDetector {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := newID()
	now := time.Now().UTC()
	if freq == "" {
		freq = "SIX_HOURS"
	}
	status := "ENABLED"
	if !enable {
		status = "DISABLED"
	}
	d := &StoredDetector{
		DetectorID:                 id,
		CreatedAt:                  now,
		UpdatedAt:                  now,
		FindingPublishingFrequency: freq,
		ServiceRole:                fmt.Sprintf("arn:aws:iam::%s:role/aws-service-role/guardduty.amazonaws.com/AWSServiceRoleForAmazonGuardDuty", s.accountID),
		Status:                     status,
		Tags:                       cloneStringMap(tags),
		DataSources:                dataSources,
		Features:                   features,
	}
	s.detectors[id] = d
	s.filters[id] = make(map[string]*StoredFilter)
	s.ipSets[id] = make(map[string]*StoredIPSet)
	s.threatIntelSets[id] = make(map[string]*StoredThreatIntelSet)
	s.threatEntitySets[id] = make(map[string]*StoredEntitySet)
	s.trustedEntitySets[id] = make(map[string]*StoredEntitySet)
	s.members[id] = make(map[string]*StoredMember)
	s.memberDetectors[id] = make(map[string]map[string]any)
	s.publishingDest[id] = make(map[string]*StoredPublishingDestination)
	s.findings[id] = make(map[string]*StoredFinding)
	s.orgConfig[id] = &StoredOrganizationConfig{}
	s.malwareScanSettings[id] = &StoredMalwareScanSettings{
		EbsSnapshotPreservation: "NO_RETENTION",
		ScanResourceCriteria: map[string]any{
			"include": map[string]any{},
			"exclude": map[string]any{},
		},
	}
	s.tags[s.detectorArn(id)] = cloneStringMap(tags)
	return d
}

// GetDetector returns the detector with the given ID.
func (s *Store) GetDetector(id string) (*StoredDetector, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.detectors[id]
	if !ok {
		return nil, errNotFound("Detector not found: %s", id)
	}
	return d, nil
}

// DeleteDetector removes a detector and all associated state.
func (s *Store) DeleteDetector(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[id]; !ok {
		return errNotFound("Detector not found: %s", id)
	}
	delete(s.detectors, id)
	delete(s.filters, id)
	delete(s.ipSets, id)
	delete(s.threatIntelSets, id)
	delete(s.threatEntitySets, id)
	delete(s.trustedEntitySets, id)
	delete(s.members, id)
	delete(s.memberDetectors, id)
	delete(s.publishingDest, id)
	delete(s.findings, id)
	delete(s.orgConfig, id)
	delete(s.malwareScanSettings, id)
	delete(s.tags, s.detectorArn(id))
	return nil
}

// ListDetectors returns sorted detector IDs.
func (s *Store) ListDetectors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.detectors))
	for id := range s.detectors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// UpdateDetector updates mutable detector properties.
func (s *Store) UpdateDetector(id string, enable *bool, freq string, dataSources map[string]any, features []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.detectors[id]
	if !ok {
		return errNotFound("Detector not found: %s", id)
	}
	if enable != nil {
		if *enable {
			d.Status = "ENABLED"
		} else {
			d.Status = "DISABLED"
		}
	}
	if freq != "" {
		d.FindingPublishingFrequency = freq
	}
	if dataSources != nil {
		d.DataSources = dataSources
	}
	if features != nil {
		d.Features = features
	}
	d.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) detectorArn(id string) string {
	return s.buildArn("detector", id)
}

// ── Filters ──────────────────────────────────────────────────────────────────

// CreateFilter creates a filter associated with a detector.
func (s *Store) CreateFilter(detectorID, name, description, action string, rank int, criteria map[string]any, tags map[string]string) (*StoredFilter, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	if _, ok := s.filters[detectorID][name]; ok {
		return nil, errBadRequest("Filter already exists: %s", name)
	}
	if action == "" {
		action = "NOOP"
	}
	now := time.Now().UTC()
	f := &StoredFilter{
		DetectorID:      detectorID,
		Name:            name,
		Description:     description,
		Action:          action,
		Rank:            rank,
		FindingCriteria: criteria,
		Tags:            cloneStringMap(tags),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.filters[detectorID][name] = f
	s.tags[s.filterArn(detectorID, name)] = cloneStringMap(tags)
	return f, nil
}

// GetFilter returns a stored filter.
func (s *Store) GetFilter(detectorID, name string) (*StoredFilter, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	f, ok := s.filters[detectorID][name]
	if !ok {
		return nil, errNotFound("Filter not found: %s", name)
	}
	return f, nil
}

// UpdateFilter updates an existing filter.
func (s *Store) UpdateFilter(detectorID, name, description, action string, rank int, criteria map[string]any) (*StoredFilter, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	f, ok := s.filters[detectorID][name]
	if !ok {
		return nil, errNotFound("Filter not found: %s", name)
	}
	if description != "" {
		f.Description = description
	}
	if action != "" {
		f.Action = action
	}
	if rank > 0 {
		f.Rank = rank
	}
	if criteria != nil {
		f.FindingCriteria = criteria
	}
	f.UpdatedAt = time.Now().UTC()
	return f, nil
}

// DeleteFilter removes a filter.
func (s *Store) DeleteFilter(detectorID, name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if _, ok := s.filters[detectorID][name]; !ok {
		return errNotFound("Filter not found: %s", name)
	}
	delete(s.filters[detectorID], name)
	delete(s.tags, s.filterArn(detectorID, name))
	return nil
}

// ListFilters returns the filter names for a detector.
func (s *Store) ListFilters(detectorID string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]string, 0, len(s.filters[detectorID]))
	for name := range s.filters[detectorID] {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) filterArn(detectorID, name string) string {
	return s.buildArn("detector", detectorID, "filter", name)
}

// ── IPSets ───────────────────────────────────────────────────────────────────

// CreateIPSet creates a new IP set.
func (s *Store) CreateIPSet(detectorID, name, format, location string, activate bool, expectedOwner string, tags map[string]string) (*StoredIPSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	id := newID()
	status := "INACTIVE"
	if activate {
		status = "ACTIVE"
	}
	now := time.Now().UTC()
	set := &StoredIPSet{
		DetectorID:          detectorID,
		IPSetID:             id,
		Name:                name,
		Format:              format,
		Location:            location,
		Status:              status,
		ExpectedBucketOwner: expectedOwner,
		Tags:                cloneStringMap(tags),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.ipSets[detectorID][id] = set
	s.tags[s.ipSetArn(detectorID, id)] = cloneStringMap(tags)
	return set, nil
}

// GetIPSet returns a stored IP set.
func (s *Store) GetIPSet(detectorID, id string) (*StoredIPSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	set, ok := s.ipSets[detectorID][id]
	if !ok {
		return nil, errNotFound("IPSet not found: %s", id)
	}
	return set, nil
}

// UpdateIPSet updates an existing IP set.
func (s *Store) UpdateIPSet(detectorID, id, name, location string, activate *bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	set, ok := s.ipSets[detectorID][id]
	if !ok {
		return errNotFound("IPSet not found: %s", id)
	}
	if name != "" {
		set.Name = name
	}
	if location != "" {
		set.Location = location
	}
	if activate != nil {
		if *activate {
			set.Status = "ACTIVE"
		} else {
			set.Status = "INACTIVE"
		}
	}
	set.UpdatedAt = time.Now().UTC()
	return nil
}

// DeleteIPSet removes an IP set.
func (s *Store) DeleteIPSet(detectorID, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if _, ok := s.ipSets[detectorID][id]; !ok {
		return errNotFound("IPSet not found: %s", id)
	}
	delete(s.ipSets[detectorID], id)
	delete(s.tags, s.ipSetArn(detectorID, id))
	return nil
}

// ListIPSets returns the IDs of all IP sets for a detector.
func (s *Store) ListIPSets(detectorID string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]string, 0, len(s.ipSets[detectorID]))
	for id := range s.ipSets[detectorID] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) ipSetArn(detectorID, id string) string {
	return s.buildArn("detector", detectorID, "ipset", id)
}

// ── ThreatIntelSets ──────────────────────────────────────────────────────────

// CreateThreatIntelSet creates a threat intel set.
func (s *Store) CreateThreatIntelSet(detectorID, name, format, location string, activate bool, expectedOwner string, tags map[string]string) (*StoredThreatIntelSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	id := newID()
	status := "INACTIVE"
	if activate {
		status = "ACTIVE"
	}
	now := time.Now().UTC()
	set := &StoredThreatIntelSet{
		DetectorID:          detectorID,
		ThreatIntelSetID:    id,
		Name:                name,
		Format:              format,
		Location:            location,
		Status:              status,
		ExpectedBucketOwner: expectedOwner,
		Tags:                cloneStringMap(tags),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.threatIntelSets[detectorID][id] = set
	s.tags[s.threatIntelSetArn(detectorID, id)] = cloneStringMap(tags)
	return set, nil
}

// GetThreatIntelSet returns a stored threat intel set.
func (s *Store) GetThreatIntelSet(detectorID, id string) (*StoredThreatIntelSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	set, ok := s.threatIntelSets[detectorID][id]
	if !ok {
		return nil, errNotFound("ThreatIntelSet not found: %s", id)
	}
	return set, nil
}

// UpdateThreatIntelSet updates an existing set.
func (s *Store) UpdateThreatIntelSet(detectorID, id, name, location string, activate *bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	set, ok := s.threatIntelSets[detectorID][id]
	if !ok {
		return errNotFound("ThreatIntelSet not found: %s", id)
	}
	if name != "" {
		set.Name = name
	}
	if location != "" {
		set.Location = location
	}
	if activate != nil {
		if *activate {
			set.Status = "ACTIVE"
		} else {
			set.Status = "INACTIVE"
		}
	}
	set.UpdatedAt = time.Now().UTC()
	return nil
}

// DeleteThreatIntelSet removes a threat intel set.
func (s *Store) DeleteThreatIntelSet(detectorID, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if _, ok := s.threatIntelSets[detectorID][id]; !ok {
		return errNotFound("ThreatIntelSet not found: %s", id)
	}
	delete(s.threatIntelSets[detectorID], id)
	delete(s.tags, s.threatIntelSetArn(detectorID, id))
	return nil
}

// ListThreatIntelSets returns the IDs of all threat intel sets.
func (s *Store) ListThreatIntelSets(detectorID string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]string, 0, len(s.threatIntelSets[detectorID]))
	for id := range s.threatIntelSets[detectorID] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) threatIntelSetArn(detectorID, id string) string {
	return s.buildArn("detector", detectorID, "threatintelset", id)
}

// ── Threat / Trusted entity sets ─────────────────────────────────────────────

// CreateEntitySet creates a threat or trusted entity set, depending on kind.
func (s *Store) CreateEntitySet(kind, detectorID, name, format, location string, activate bool, expectedOwner string, tags map[string]string) (*StoredEntitySet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	bucket := s.entitySetBucket(kind)
	if bucket == nil {
		return nil, errBadRequest("invalid entity set kind: %s", kind)
	}
	id := newID()
	status := "INACTIVE"
	if activate {
		status = "ACTIVE"
	}
	now := time.Now().UTC()
	set := &StoredEntitySet{
		DetectorID:          detectorID,
		EntitySetID:         id,
		Kind:                kind,
		Name:                name,
		Format:              format,
		Location:            location,
		Status:              status,
		ExpectedBucketOwner: expectedOwner,
		Tags:                cloneStringMap(tags),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	bucket[detectorID][id] = set
	s.tags[s.entitySetArn(kind, detectorID, id)] = cloneStringMap(tags)
	return set, nil
}

// GetEntitySet returns a stored entity set.
func (s *Store) GetEntitySet(kind, detectorID, id string) (*StoredEntitySet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	bucket := s.entitySetBucket(kind)
	if bucket == nil {
		return nil, errBadRequest("invalid entity set kind: %s", kind)
	}
	set, ok := bucket[detectorID][id]
	if !ok {
		return nil, errNotFound("%s entity set not found: %s", kind, id)
	}
	return set, nil
}

// UpdateEntitySet updates an existing set.
func (s *Store) UpdateEntitySet(kind, detectorID, id, name, location string, activate *bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	bucket := s.entitySetBucket(kind)
	if bucket == nil {
		return errBadRequest("invalid entity set kind: %s", kind)
	}
	set, ok := bucket[detectorID][id]
	if !ok {
		return errNotFound("%s entity set not found: %s", kind, id)
	}
	if name != "" {
		set.Name = name
	}
	if location != "" {
		set.Location = location
	}
	if activate != nil {
		if *activate {
			set.Status = "ACTIVE"
		} else {
			set.Status = "INACTIVE"
		}
	}
	set.UpdatedAt = time.Now().UTC()
	return nil
}

// DeleteEntitySet removes a stored entity set.
func (s *Store) DeleteEntitySet(kind, detectorID, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	bucket := s.entitySetBucket(kind)
	if bucket == nil {
		return errBadRequest("invalid entity set kind: %s", kind)
	}
	if _, ok := bucket[detectorID][id]; !ok {
		return errNotFound("%s entity set not found: %s", kind, id)
	}
	delete(bucket[detectorID], id)
	delete(s.tags, s.entitySetArn(kind, detectorID, id))
	return nil
}

// ListEntitySets returns the IDs of stored entity sets.
func (s *Store) ListEntitySets(kind, detectorID string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	bucket := s.entitySetBucket(kind)
	if bucket == nil {
		return nil, errBadRequest("invalid entity set kind: %s", kind)
	}
	out := make([]string, 0, len(bucket[detectorID]))
	for id := range bucket[detectorID] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) entitySetBucket(kind string) map[string]map[string]*StoredEntitySet {
	switch kind {
	case "threat":
		return s.threatEntitySets
	case "trusted":
		return s.trustedEntitySets
	}
	return nil
}

func (s *Store) entitySetArn(kind, detectorID, id string) string {
	return s.buildArn("detector", detectorID, kind+"entityset", id)
}

// ── Members ──────────────────────────────────────────────────────────────────

// CreateMember adds a member account under a detector.
func (s *Store) CreateMember(detectorID, accountID, email string) (*StoredMember, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	if accountID == "" || email == "" {
		return nil, errBadRequest("accountId and email are required")
	}
	if existing, ok := s.members[detectorID][accountID]; ok {
		return existing, nil
	}
	now := time.Now().UTC()
	m := &StoredMember{
		DetectorID:         detectorID,
		AccountID:          accountID,
		Email:              email,
		RelationshipStatus: "CREATED",
		MasterID:           s.accountID,
		AdministratorID:    s.accountID,
		InvitedAt:          time.Time{},
		UpdatedAt:          now,
	}
	s.members[detectorID][accountID] = m
	s.memberDetectors[detectorID][accountID] = map[string]any{
		"detectorId": detectorID,
		"dataSources": map[string]any{
			"cloudTrail": map[string]any{"status": "ENABLED"},
		},
	}
	return m, nil
}

// GetMembers returns the members for the given account IDs.
func (s *Store) GetMembers(detectorID string, accountIDs []string) ([]*StoredMember, []string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, nil, errNotFound("Detector not found: %s", detectorID)
	}
	found := make([]*StoredMember, 0, len(accountIDs))
	missing := make([]string, 0)
	for _, id := range accountIDs {
		if m, ok := s.members[detectorID][id]; ok {
			found = append(found, m)
		} else {
			missing = append(missing, id)
		}
	}
	return found, missing, nil
}

// ListMembers returns all members for a detector.
func (s *Store) ListMembers(detectorID string) ([]*StoredMember, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]*StoredMember, 0, len(s.members[detectorID]))
	for _, m := range s.members[detectorID] {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountID < out[j].AccountID })
	return out, nil
}

// DeleteMembers removes members.
func (s *Store) DeleteMembers(detectorID string, accountIDs []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	for _, id := range accountIDs {
		delete(s.members[detectorID], id)
		delete(s.memberDetectors[detectorID], id)
	}
	return nil
}

// InviteMembers issues invitations to members.
func (s *Store) InviteMembers(detectorID string, accountIDs []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	now := time.Now().UTC()
	for _, id := range accountIDs {
		if m, ok := s.members[detectorID][id]; ok {
			m.RelationshipStatus = "INVITED"
			m.InvitedAt = now
			m.UpdatedAt = now
		}
		s.invitations[id] = &StoredInvitation{
			AccountID:          id,
			InvitationID:       newID(),
			RelationshipStatus: "INVITED",
			InvitedAt:          now,
			AdministratorID:    s.accountID,
		}
	}
	return nil
}

// SetMemberStatus mass-updates the relationship status for a list of members.
func (s *Store) SetMemberStatus(detectorID string, accountIDs []string, status string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	now := time.Now().UTC()
	for _, id := range accountIDs {
		if m, ok := s.members[detectorID][id]; ok {
			m.RelationshipStatus = status
			m.UpdatedAt = now
		}
	}
	return nil
}

// UpdateMemberDetectors records a per-member detector configuration update.
func (s *Store) UpdateMemberDetectors(detectorID string, accountIDs []string, dataSources map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	for _, id := range accountIDs {
		if _, ok := s.members[detectorID][id]; !ok {
			continue
		}
		s.memberDetectors[detectorID][id] = map[string]any{
			"detectorId":  detectorID,
			"dataSources": dataSources,
		}
	}
	return nil
}

// MemberDetectors returns the per-member detector configurations.
func (s *Store) MemberDetectors(detectorID string, accountIDs []string) ([]map[string]any, []string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]map[string]any, 0, len(accountIDs))
	missing := make([]string, 0)
	for _, id := range accountIDs {
		if cfg, ok := s.memberDetectors[detectorID][id]; ok {
			cp := map[string]any{
				"accountId":   id,
				"detectorId":  detectorID,
				"dataSources": cfg["dataSources"],
			}
			out = append(out, cp)
		} else {
			missing = append(missing, id)
		}
	}
	return out, missing, nil
}

// ── Invitations ──────────────────────────────────────────────────────────────

// ListInvitations returns the queued invitations.
func (s *Store) ListInvitations() []*StoredInvitation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredInvitation, 0, len(s.invitations))
	for _, inv := range s.invitations {
		out = append(out, inv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountID < out[j].AccountID })
	return out
}

// DeleteInvitations removes invitations for the given account IDs.
func (s *Store) DeleteInvitations(accountIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range accountIDs {
		delete(s.invitations, id)
	}
}

// AcceptInvitation marks an invitation as accepted and creates a master record.
func (s *Store) AcceptInvitation(detectorID, invitationID, masterID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	now := time.Now().UTC()
	s.masterAccount = &StoredMember{
		DetectorID:         detectorID,
		AccountID:          masterID,
		MasterID:           masterID,
		AdministratorID:    masterID,
		RelationshipStatus: "ENABLED",
		InvitedAt:          now,
		UpdatedAt:          now,
	}
	return nil
}

// DisassociateMaster removes the master/administrator relationship.
func (s *Store) DisassociateMaster() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.masterAccount = nil
}

// MasterAccount returns the master/administrator record (or nil).
func (s *Store) MasterAccount() *StoredMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.masterAccount
}

// ── Publishing destinations ─────────────────────────────────────────────────

// CreatePublishingDestination registers a new publishing destination.
func (s *Store) CreatePublishingDestination(detectorID, destType string, props map[string]any) (*StoredPublishingDestination, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	if destType == "" {
		destType = "S3"
	}
	id := newID()
	now := time.Now().UTC()
	d := &StoredPublishingDestination{
		DetectorID:            detectorID,
		DestinationID:         id,
		DestinationType:       destType,
		Status:                "PUBLISHING",
		DestinationProperties: props,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.publishingDest[detectorID][id] = d
	return d, nil
}

// GetPublishingDestination returns a destination by ID.
func (s *Store) GetPublishingDestination(detectorID, id string) (*StoredPublishingDestination, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	d, ok := s.publishingDest[detectorID][id]
	if !ok {
		return nil, errNotFound("PublishingDestination not found: %s", id)
	}
	return d, nil
}

// UpdatePublishingDestination updates a destination's properties.
func (s *Store) UpdatePublishingDestination(detectorID, id string, props map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	d, ok := s.publishingDest[detectorID][id]
	if !ok {
		return errNotFound("PublishingDestination not found: %s", id)
	}
	if props != nil {
		d.DestinationProperties = props
	}
	d.UpdatedAt = time.Now().UTC()
	return nil
}

// DeletePublishingDestination removes a destination.
func (s *Store) DeletePublishingDestination(detectorID, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if _, ok := s.publishingDest[detectorID][id]; !ok {
		return errNotFound("PublishingDestination not found: %s", id)
	}
	delete(s.publishingDest[detectorID], id)
	return nil
}

// ListPublishingDestinations returns all destinations for a detector.
func (s *Store) ListPublishingDestinations(detectorID string) ([]*StoredPublishingDestination, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]*StoredPublishingDestination, 0, len(s.publishingDest[detectorID]))
	for _, d := range s.publishingDest[detectorID] {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DestinationID < out[j].DestinationID })
	return out, nil
}

// ── Findings ─────────────────────────────────────────────────────────────────

// CreateSampleFindings prepopulates a small set of sample findings.
func (s *Store) CreateSampleFindings(detectorID string, findingTypes []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if len(findingTypes) == 0 {
		findingTypes = []string{
			"Recon:EC2/PortProbeUnprotectedPort",
			"UnauthorizedAccess:EC2/SSHBruteForce",
			"Trojan:EC2/BlackholeTraffic",
			"CryptoCurrency:EC2/BitcoinTool.B!DNS",
		}
	}
	now := time.Now().UTC()
	for i, ftype := range findingTypes {
		id := newID()
		f := &StoredFinding{
			DetectorID:    detectorID,
			ID:            id,
			Arn:           s.buildArn("detector", detectorID, "finding", id),
			AccountID:     s.accountID,
			SchemaVersion: "2.0",
			Type:          ftype,
			Title:         fmt.Sprintf("Sample finding %d: %s", i+1, ftype),
			Description:   fmt.Sprintf("Synthetic sample finding for type %s.", ftype),
			Severity:      5.0,
			Confidence:    8.0,
			CreatedAt:     now,
			UpdatedAt:     now,
			Region:        s.region,
			Partition:     "aws",
			Service: map[string]any{
				"serviceName": "guardduty",
				"detectorId":  detectorID,
				"count":       1,
			},
			Resource: map[string]any{
				"resourceType": "Instance",
				"instanceDetails": map[string]any{
					"instanceId": "i-99999999",
				},
			},
		}
		s.findings[detectorID][id] = f
	}
	return nil
}

// AddFinding inserts a finding directly (used in tests).
func (s *Store) AddFinding(detectorID string, f *StoredFinding) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	if f.ID == "" {
		f.ID = newID()
	}
	if f.Arn == "" {
		f.Arn = s.buildArn("detector", detectorID, "finding", f.ID)
	}
	if f.AccountID == "" {
		f.AccountID = s.accountID
	}
	if f.SchemaVersion == "" {
		f.SchemaVersion = "2.0"
	}
	if f.Region == "" {
		f.Region = s.region
	}
	if f.Partition == "" {
		f.Partition = "aws"
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = f.CreatedAt
	}
	f.DetectorID = detectorID
	s.findings[detectorID][f.ID] = f
	return nil
}

// GetFindings returns the findings for the given IDs.
func (s *Store) GetFindings(detectorID string, ids []string) ([]*StoredFinding, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]*StoredFinding, 0, len(ids))
	for _, id := range ids {
		if f, ok := s.findings[detectorID][id]; ok {
			out = append(out, f)
		}
	}
	return out, nil
}

// ListFindings returns finding IDs for a detector.
func (s *Store) ListFindings(detectorID string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	out := make([]string, 0, len(s.findings[detectorID]))
	for id := range s.findings[detectorID] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// SetFindingsArchived updates the archived flag for a list of findings.
func (s *Store) SetFindingsArchived(detectorID string, ids []string, archived bool) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	for _, id := range ids {
		if f, ok := s.findings[detectorID][id]; ok {
			f.Archived = archived
			f.UpdatedAt = time.Now().UTC()
		}
	}
	return nil
}

// SetFindingsFeedback records a feedback string for findings.
func (s *Store) SetFindingsFeedback(detectorID string, ids []string, feedback string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	for _, id := range ids {
		if f, ok := s.findings[detectorID][id]; ok {
			f.Feedback = feedback
			f.UpdatedAt = time.Now().UTC()
		}
	}
	return nil
}

// FindingsStatistics returns aggregated counts by severity.
func (s *Store) FindingsStatistics(detectorID string) (map[string]int, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	bySeverity := make(map[string]int)
	for _, f := range s.findings[detectorID] {
		key := fmt.Sprintf("%g", f.Severity)
		bySeverity[key]++
	}
	return bySeverity, nil
}

// ── Malware scans ────────────────────────────────────────────────────────────

// StartMalwareScan creates a new malware scan record.
func (s *Store) StartMalwareScan(resourceArn string) (*StoredMalwareScan, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if resourceArn == "" {
		return nil, errBadRequest("resourceArn is required")
	}
	id := newID()
	now := time.Now().UTC()
	scan := &StoredMalwareScan{
		ScanID:         id,
		ResourceArn:    resourceArn,
		ResourceType:   "EC2_INSTANCE",
		ScanType:       "GUARDDUTY_INITIATED",
		ScanStatus:     "RUNNING",
		ScanStartedAt:  now,
		TriggerDetails: map[string]any{},
	}
	s.malwareScans[id] = scan
	return scan, nil
}

// ListMalwareScans returns all stored scans.
func (s *Store) ListMalwareScans() []*StoredMalwareScan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMalwareScan, 0, len(s.malwareScans))
	for _, scan := range s.malwareScans {
		out = append(out, scan)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ScanID < out[j].ScanID })
	return out
}

// MalwareScanSettings returns per-detector scan settings.
func (s *Store) MalwareScanSettings(detectorID string) (*StoredMalwareScanSettings, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	cfg, ok := s.malwareScanSettings[detectorID]
	if !ok {
		return &StoredMalwareScanSettings{
			EbsSnapshotPreservation: "NO_RETENTION",
			ScanResourceCriteria:    map[string]any{},
		}, nil
	}
	return cfg, nil
}

// UpdateMalwareScanSettings updates the per-detector scan settings.
func (s *Store) UpdateMalwareScanSettings(detectorID, preservation string, criteria map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	cfg, ok := s.malwareScanSettings[detectorID]
	if !ok {
		cfg = &StoredMalwareScanSettings{}
		s.malwareScanSettings[detectorID] = cfg
	}
	if preservation != "" {
		cfg.EbsSnapshotPreservation = preservation
	}
	if criteria != nil {
		cfg.ScanResourceCriteria = criteria
	}
	return nil
}

// ── Malware protection plans ─────────────────────────────────────────────────

// CreateMalwareProtectionPlan stores a new protection plan.
func (s *Store) CreateMalwareProtectionPlan(role string, protected, actions map[string]any, tags map[string]string) *StoredMalwareProtectionPlan {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	now := time.Now().UTC()
	plan := &StoredMalwareProtectionPlan{
		PlanID:            id,
		Arn:               s.buildArn("malware-protection-plan", id),
		Role:              role,
		ProtectedResource: protected,
		Actions:           actions,
		Status:            "ACTIVE",
		Tags:              cloneStringMap(tags),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.malwareProtectionPlan[id] = plan
	s.tags[plan.Arn] = cloneStringMap(tags)
	return plan
}

// GetMalwareProtectionPlan returns a stored plan by ID.
func (s *Store) GetMalwareProtectionPlan(id string) (*StoredMalwareProtectionPlan, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.malwareProtectionPlan[id]
	if !ok {
		return nil, errNotFound("MalwareProtectionPlan not found: %s", id)
	}
	return plan, nil
}

// ListMalwareProtectionPlans returns all stored plans.
func (s *Store) ListMalwareProtectionPlans() []*StoredMalwareProtectionPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMalwareProtectionPlan, 0, len(s.malwareProtectionPlan))
	for _, p := range s.malwareProtectionPlan {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlanID < out[j].PlanID })
	return out
}

// UpdateMalwareProtectionPlan updates a stored plan.
func (s *Store) UpdateMalwareProtectionPlan(id, role string, protected, actions map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.malwareProtectionPlan[id]
	if !ok {
		return errNotFound("MalwareProtectionPlan not found: %s", id)
	}
	if role != "" {
		plan.Role = role
	}
	if protected != nil {
		plan.ProtectedResource = protected
	}
	if actions != nil {
		plan.Actions = actions
	}
	plan.UpdatedAt = time.Now().UTC()
	return nil
}

// DeleteMalwareProtectionPlan removes a plan.
func (s *Store) DeleteMalwareProtectionPlan(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.malwareProtectionPlan[id]
	if !ok {
		return errNotFound("MalwareProtectionPlan not found: %s", id)
	}
	delete(s.malwareProtectionPlan, id)
	delete(s.tags, plan.Arn)
	return nil
}

// ── Organization ─────────────────────────────────────────────────────────────

// OrgConfig returns the per-detector organization config.
func (s *Store) OrgConfig(detectorID string) (*StoredOrganizationConfig, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return nil, errNotFound("Detector not found: %s", detectorID)
	}
	cfg, ok := s.orgConfig[detectorID]
	if !ok {
		return &StoredOrganizationConfig{}, nil
	}
	return cfg, nil
}

// UpdateOrgConfig replaces the per-detector organization config.
func (s *Store) UpdateOrgConfig(detectorID string, autoEnable bool, autoEnableMembers string, dataSources map[string]any, features []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.detectors[detectorID]; !ok {
		return errNotFound("Detector not found: %s", detectorID)
	}
	cfg := s.orgConfig[detectorID]
	if cfg == nil {
		cfg = &StoredOrganizationConfig{}
	}
	cfg.AutoEnable = autoEnable
	if autoEnableMembers != "" {
		cfg.AutoEnableOrganizationMembers = autoEnableMembers
	}
	if dataSources != nil {
		cfg.DataSources = dataSources
	}
	if features != nil {
		cfg.Features = features
	}
	s.orgConfig[detectorID] = cfg
	return nil
}

// EnableOrgAdminAccount registers an organization admin account.
func (s *Store) EnableOrgAdminAccount(accountID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if accountID == "" {
		return errBadRequest("adminAccountId is required")
	}
	s.orgAdminAccounts[accountID] = "ENABLED"
	return nil
}

// DisableOrgAdminAccount removes an organization admin account registration.
func (s *Store) DisableOrgAdminAccount(accountID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orgAdminAccounts[accountID]; !ok {
		return errNotFound("Organization admin account not found: %s", accountID)
	}
	delete(s.orgAdminAccounts, accountID)
	return nil
}

// OrgAdminAccounts returns the registered admin accounts.
func (s *Store) OrgAdminAccounts() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.orgAdminAccounts))
	for k, v := range s.orgAdminAccounts {
		out[k] = v
	}
	return out
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// TagResource attaches tags to a resource ARN.
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

// UntagResource removes tags from a resource ARN.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns the tags for a resource ARN.
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

// CountFindings returns the total number of findings across detectors.
func (s *Store) CountFindings() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, m := range s.findings {
		total += len(m)
	}
	return total
}
