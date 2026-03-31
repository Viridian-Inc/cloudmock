package pinpoint

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Application represents a Pinpoint application.
type Application struct {
	ApplicationID string
	Arn           string
	Name          string
	Tags          map[string]string
	CreationDate  time.Time
}

// Segment represents a Pinpoint segment.
type Segment struct {
	SegmentID     string
	ApplicationID string
	Name          string
	SegmentType   string
	Dimensions    map[string]any
	Version       int
	CreationDate  time.Time
	LastModified  time.Time
}

// Campaign represents a Pinpoint campaign.
type Campaign struct {
	CampaignID    string
	ApplicationID string
	Name          string
	SegmentID     string
	State         string
	Description   string
	Schedule      map[string]any
	CreationDate  time.Time
	LastModified  time.Time
	lifecycle     *lifecycle.Machine
}

// Journey represents a Pinpoint journey.
type Journey struct {
	JourneyID     string
	ApplicationID string
	Name          string
	State         string
	CreationDate  time.Time
	LastModified  time.Time
}

// EndpointItem represents a Pinpoint endpoint.
type EndpointItem struct {
	EndpointID    string
	ApplicationID string
	ChannelType   string
	Address       string
	User          map[string]any
	Attributes    map[string][]string
	CreationDate  time.Time
}

// Store manages Pinpoint resources in memory.
type Store struct {
	mu         sync.RWMutex
	apps       map[string]*Application
	segments   map[string]map[string]*Segment   // appID -> segID -> Segment
	campaigns  map[string]map[string]*Campaign   // appID -> campID -> Campaign
	journeys   map[string]map[string]*Journey    // appID -> journeyID -> Journey
	endpoints  map[string]map[string]*EndpointItem // appID -> endpointID -> Endpoint
	accountID  string
	region     string
	lcConfig   *lifecycle.Config
	appSeq     int
	segSeq     int
	campSeq    int
	journeySeq int
}

// NewStore returns a new empty Pinpoint Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		apps:      make(map[string]*Application),
		segments:  make(map[string]map[string]*Segment),
		campaigns: make(map[string]map[string]*Campaign),
		journeys:  make(map[string]map[string]*Journey),
		endpoints: make(map[string]map[string]*EndpointItem),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

// CreateApp creates a new application.
func (s *Store) CreateApp(name string, tags map[string]string) *Application {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appSeq++
	id := fmt.Sprintf("app-%012d", s.appSeq)
	app := &Application{
		ApplicationID: id,
		Arn:           fmt.Sprintf("arn:aws:mobiletargeting:%s:%s:apps/%s", s.region, s.accountID, id),
		Name:          name,
		Tags:          tags,
		CreationDate:  time.Now().UTC(),
	}
	s.apps[id] = app
	s.segments[id] = make(map[string]*Segment)
	s.campaigns[id] = make(map[string]*Campaign)
	s.journeys[id] = make(map[string]*Journey)
	s.endpoints[id] = make(map[string]*EndpointItem)
	return app
}

// GetApp retrieves an application.
func (s *Store) GetApp(id string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[id]
	return app, ok
}

// ListApps returns all applications.
func (s *Store) ListApps() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Application, 0, len(s.apps))
	for _, a := range s.apps {
		out = append(out, a)
	}
	return out
}

// DeleteApp removes an application and all its resources.
func (s *Store) DeleteApp(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[id]; !ok {
		return false
	}
	delete(s.apps, id)
	delete(s.segments, id)
	delete(s.campaigns, id)
	delete(s.journeys, id)
	delete(s.endpoints, id)
	return true
}

// CreateSegment creates a new segment.
func (s *Store) CreateSegment(appID, name, segType string, dimensions map[string]any) (*Segment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	s.segSeq++
	id := fmt.Sprintf("seg-%012d", s.segSeq)
	now := time.Now().UTC()
	seg := &Segment{
		SegmentID:     id,
		ApplicationID: appID,
		Name:          name,
		SegmentType:   segType,
		Dimensions:    dimensions,
		Version:       1,
		CreationDate:  now,
		LastModified:  now,
	}
	s.segments[appID][id] = seg
	return seg, nil
}

// GetSegment retrieves a segment.
func (s *Store) GetSegment(appID, segID string) (*Segment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	segMap, ok := s.segments[appID]
	if !ok {
		return nil, false
	}
	seg, ok := segMap[segID]
	return seg, ok
}

// ListSegments returns all segments for an app.
func (s *Store) ListSegments(appID string) []*Segment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	segMap := s.segments[appID]
	out := make([]*Segment, 0, len(segMap))
	for _, seg := range segMap {
		out = append(out, seg)
	}
	return out
}

// DeleteSegment removes a segment.
func (s *Store) DeleteSegment(appID, segID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	segMap, ok := s.segments[appID]
	if !ok {
		return false
	}
	if _, ok := segMap[segID]; !ok {
		return false
	}
	delete(segMap, segID)
	return true
}

// CreateCampaign creates a new campaign.
func (s *Store) CreateCampaign(appID, name, segID, description string, schedule map[string]any) (*Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	s.campSeq++
	id := fmt.Sprintf("camp-%012d", s.campSeq)
	now := time.Now().UTC()

	transitions := []lifecycle.Transition{
		{From: "SCHEDULED", To: "EXECUTING", Delay: 3 * time.Second},
		{From: "EXECUTING", To: "COMPLETED", Delay: 5 * time.Second},
	}

	camp := &Campaign{
		CampaignID:    id,
		ApplicationID: appID,
		Name:          name,
		SegmentID:     segID,
		State:         "SCHEDULED",
		Description:   description,
		Schedule:      schedule,
		CreationDate:  now,
		LastModified:  now,
	}
	camp.lifecycle = lifecycle.NewMachine("SCHEDULED", transitions, s.lcConfig)
	camp.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		camp.State = string(to)
		camp.LastModified = time.Now().UTC()
	})

	s.campaigns[appID][id] = camp
	return camp, nil
}

// GetCampaign retrieves a campaign.
func (s *Store) GetCampaign(appID, campID string) (*Campaign, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	campMap, ok := s.campaigns[appID]
	if !ok {
		return nil, false
	}
	camp, ok := campMap[campID]
	return camp, ok
}

// ListCampaigns returns all campaigns for an app.
func (s *Store) ListCampaigns(appID string) []*Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()
	campMap := s.campaigns[appID]
	out := make([]*Campaign, 0, len(campMap))
	for _, camp := range campMap {
		out = append(out, camp)
	}
	return out
}

// DeleteCampaign removes a campaign.
func (s *Store) DeleteCampaign(appID, campID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	campMap, ok := s.campaigns[appID]
	if !ok {
		return false
	}
	camp, ok := campMap[campID]
	if !ok {
		return false
	}
	if camp.lifecycle != nil {
		camp.lifecycle.Stop()
	}
	delete(campMap, campID)
	return true
}

// CreateJourney creates a new journey.
func (s *Store) CreateJourney(appID, name string) (*Journey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	s.journeySeq++
	id := fmt.Sprintf("journey-%012d", s.journeySeq)
	now := time.Now().UTC()
	j := &Journey{
		JourneyID:     id,
		ApplicationID: appID,
		Name:          name,
		State:         "DRAFT",
		CreationDate:  now,
		LastModified:  now,
	}
	s.journeys[appID][id] = j
	return j, nil
}

// GetJourney retrieves a journey.
func (s *Store) GetJourney(appID, journeyID string) (*Journey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jMap, ok := s.journeys[appID]
	if !ok {
		return nil, false
	}
	j, ok := jMap[journeyID]
	return j, ok
}

// ListJourneys returns all journeys for an app.
func (s *Store) ListJourneys(appID string) []*Journey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jMap := s.journeys[appID]
	out := make([]*Journey, 0, len(jMap))
	for _, j := range jMap {
		out = append(out, j)
	}
	return out
}

// UpdateEndpoint creates or updates an endpoint.
func (s *Store) UpdateEndpoint(appID, endpointID, channelType, address string, user map[string]any, attrs map[string][]string) (*EndpointItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appID]; !ok {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	ep := &EndpointItem{
		EndpointID:    endpointID,
		ApplicationID: appID,
		ChannelType:   channelType,
		Address:       address,
		User:          user,
		Attributes:    attrs,
		CreationDate:  time.Now().UTC(),
	}
	s.endpoints[appID][endpointID] = ep
	return ep, nil
}

// GetEndpoint retrieves an endpoint.
func (s *Store) GetEndpoint(appID, endpointID string) (*EndpointItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	epMap, ok := s.endpoints[appID]
	if !ok {
		return nil, false
	}
	ep, ok := epMap[endpointID]
	return ep, ok
}
