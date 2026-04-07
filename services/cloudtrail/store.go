package cloudtrail

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// Trail holds all metadata for a CloudTrail trail.
type Trail struct {
	Name                       string
	TrailARN                   string
	HomeRegion                 string
	S3BucketName               string
	S3KeyPrefix                string
	SnsTopicName               string
	SnsTopicARN                string
	IncludeGlobalServiceEvents bool
	IsMultiRegionTrail         bool
	IsOrganizationTrail        bool
	LogFileValidationEnabled   bool
	CloudWatchLogsLogGroupArn  string
	CloudWatchLogsRoleArn      string
	KmsKeyId                   string
	HasCustomEventSelectors    bool
	HasInsightSelectors        bool
	Tags                       []Tag
	IsLogging                  bool
	LatestDeliveryTime         *time.Time
	LatestNotificationTime     *time.Time
	StartLoggingTime           *time.Time
	StopLoggingTime            *time.Time
	EventSelectors             []EventSelector
	InsightSelectors           []InsightSelector
	subscriptionID             string // eventbus subscription ID (unexported)
}

// EventSelector configures event filtering for a trail.
type EventSelector struct {
	ReadWriteType           string
	IncludeManagementEvents bool
	DataResources           []DataResource
	ExcludeManagementEventSources []string
}

// DataResource specifies data event logging.
type DataResource struct {
	Type   string
	Values []string
}

// InsightSelector configures insight event collection.
type InsightSelector struct {
	InsightType string
}

// Event represents a CloudTrail event.
type Event struct {
	EventId         string
	EventName       string
	EventTime       time.Time
	EventSource     string
	Username        string
	Resources       []EventResource
	CloudTrailEvent string
	ReadOnly        string
	AccessKeyId     string
}

// EventResource identifies an AWS resource in an event.
type EventResource struct {
	ResourceType string
	ResourceName string
}

// Store is the in-memory store for CloudTrail resources.
type Store struct {
	mu          sync.RWMutex
	trails      map[string]*Trail  // keyed by trail name
	events      []Event            // global event log (RecordEvent)
	trailEvents map[string][]Event // per-trail event log (from bus)
	accountID   string
	region      string
}

// NewStore creates an empty CloudTrail Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		trails:      make(map[string]*Trail),
		events:      make([]Event, 0),
		trailEvents: make(map[string][]Event),
		accountID:   accountID,
		region:      region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) buildTrailARN(name string) string {
	return fmt.Sprintf("arn:aws:cloudtrail:%s:%s:trail/%s", s.region, s.accountID, name)
}

// CreateTrail creates a new trail.
func (s *Store) CreateTrail(name, s3Bucket, s3Prefix string, isMultiRegion, isOrg, logValidation, includeGlobal bool, tags []Tag) (*Trail, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.trails[name]; ok {
		return nil, service.NewAWSError("TrailAlreadyExistsException",
			fmt.Sprintf("Trail %s already exists.", name), http.StatusConflict)
	}

	if s3Bucket == "" {
		return nil, service.ErrValidation("S3BucketName is required.")
	}

	trail := &Trail{
		Name:                       name,
		TrailARN:                   s.buildTrailARN(name),
		HomeRegion:                 s.region,
		S3BucketName:               s3Bucket,
		S3KeyPrefix:                s3Prefix,
		IncludeGlobalServiceEvents: includeGlobal,
		IsMultiRegionTrail:         isMultiRegion,
		IsOrganizationTrail:        isOrg,
		LogFileValidationEnabled:   logValidation,
		Tags:                       tags,
		IsLogging:                  false,
		EventSelectors: []EventSelector{
			{
				ReadWriteType:           "All",
				IncludeManagementEvents: true,
			},
		},
	}

	s.trails[name] = trail
	return trail, nil
}

// GetTrail returns a trail by name.
func (s *Store) GetTrail(name string) (*Trail, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trail, ok := s.trails[name]
	if !ok {
		return nil, service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	return trail, nil
}

// ListTrails returns all trails.
func (s *Store) ListTrails() []*Trail {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Trail, 0, len(s.trails))
	for _, t := range s.trails {
		out = append(out, t)
	}
	return out
}

// DeleteTrail removes a trail by name.
func (s *Store) DeleteTrail(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.trails[name]; !ok {
		return service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	delete(s.trails, name)
	return nil
}

// UpdateTrail updates a trail's configuration.
func (s *Store) UpdateTrail(name string, updates map[string]any) (*Trail, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trail, ok := s.trails[name]
	if !ok {
		return nil, service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}

	if v, ok := updates["S3BucketName"].(string); ok && v != "" {
		trail.S3BucketName = v
	}
	if v, ok := updates["S3KeyPrefix"].(string); ok {
		trail.S3KeyPrefix = v
	}
	if v, ok := updates["SnsTopicName"].(string); ok {
		trail.SnsTopicName = v
	}
	if v, ok := updates["IsMultiRegionTrail"].(bool); ok {
		trail.IsMultiRegionTrail = v
	}
	if v, ok := updates["IsOrganizationTrail"].(bool); ok {
		trail.IsOrganizationTrail = v
	}
	if v, ok := updates["EnableLogFileValidation"].(bool); ok {
		trail.LogFileValidationEnabled = v
	}
	if v, ok := updates["IncludeGlobalServiceEvents"].(bool); ok {
		trail.IncludeGlobalServiceEvents = v
	}
	if v, ok := updates["CloudWatchLogsLogGroupArn"].(string); ok {
		trail.CloudWatchLogsLogGroupArn = v
	}
	if v, ok := updates["CloudWatchLogsRoleArn"].(string); ok {
		trail.CloudWatchLogsRoleArn = v
	}
	if v, ok := updates["KmsKeyId"].(string); ok {
		trail.KmsKeyId = v
	}

	return trail, nil
}

// StartLogging enables logging for a trail.
func (s *Store) StartLogging(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	trail, ok := s.trails[name]
	if !ok {
		return service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	trail.IsLogging = true
	now := time.Now().UTC()
	trail.StartLoggingTime = &now
	return nil
}

// StopLogging disables logging for a trail.
func (s *Store) StopLogging(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	trail, ok := s.trails[name]
	if !ok {
		return service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	trail.IsLogging = false
	now := time.Now().UTC()
	trail.StopLoggingTime = &now
	return nil
}

// PutEventSelectors sets event selectors for a trail.
func (s *Store) PutEventSelectors(name string, selectors []EventSelector) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	trail, ok := s.trails[name]
	if !ok {
		return service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	trail.EventSelectors = selectors
	trail.HasCustomEventSelectors = true
	return nil
}

// PutInsightSelectors sets insight selectors for a trail.
func (s *Store) PutInsightSelectors(name string, selectors []InsightSelector) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	trail, ok := s.trails[name]
	if !ok {
		return service.NewAWSError("TrailNotFoundException",
			fmt.Sprintf("Trail %s not found.", name), http.StatusNotFound)
	}
	trail.InsightSelectors = selectors
	trail.HasInsightSelectors = len(selectors) > 0
	return nil
}

// RecordEvent adds an event to the store (for cross-service logging).
func (s *Store) RecordEvent(eventName, eventSource, username string, resources []EventResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := Event{
		EventId:     newUUID(),
		EventName:   eventName,
		EventTime:   time.Now().UTC(),
		EventSource: eventSource,
		Username:    username,
		Resources:   resources,
		ReadOnly:    "false",
	}
	s.events = append(s.events, event)
}

// LookupAttribute is a filter criterion for LookupEvents.
type LookupAttribute struct {
	AttributeKey   string
	AttributeValue string
}

// LookupEvents returns events matching the given lookup attributes.
func (s *Store) LookupEvents(maxResults int, startTime, endTime *time.Time, attributes []LookupAttribute) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Merge global events and all trail events
	allEvents := make([]Event, 0, len(s.events))
	allEvents = append(allEvents, s.events...)
	for _, te := range s.trailEvents {
		allEvents = append(allEvents, te...)
	}

	// Deduplicate by EventId
	seen := make(map[string]bool, len(allEvents))
	deduped := make([]Event, 0, len(allEvents))
	for _, e := range allEvents {
		if !seen[e.EventId] {
			seen[e.EventId] = true
			deduped = append(deduped, e)
		}
	}

	// Filter
	var filtered []Event
	for _, e := range deduped {
		if startTime != nil && e.EventTime.Before(*startTime) {
			continue
		}
		if endTime != nil && e.EventTime.After(*endTime) {
			continue
		}
		if !matchesAttributes(e, attributes) {
			continue
		}
		filtered = append(filtered, e)
	}

	if maxResults <= 0 || maxResults > len(filtered) {
		maxResults = len(filtered)
	}

	// Sort by time descending (most recent first)
	// Simple insertion sort since event lists are typically small
	for i := 1; i < len(filtered); i++ {
		for j := i; j > 0 && filtered[j].EventTime.After(filtered[j-1].EventTime); j-- {
			filtered[j], filtered[j-1] = filtered[j-1], filtered[j]
		}
	}

	if maxResults > len(filtered) {
		maxResults = len(filtered)
	}
	return filtered[:maxResults]
}

func matchesAttributes(e Event, attrs []LookupAttribute) bool {
	for _, attr := range attrs {
		switch attr.AttributeKey {
		case "EventSource":
			if e.EventSource != attr.AttributeValue {
				return false
			}
		case "EventName":
			if e.EventName != attr.AttributeValue {
				return false
			}
		case "ResourceType":
			found := false
			for _, r := range e.Resources {
				if r.ResourceType == attr.AttributeValue {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case "ResourceName":
			found := false
			for _, r := range e.Resources {
				if r.ResourceName == attr.AttributeValue {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case "Username":
			if e.Username != attr.AttributeValue {
				return false
			}
		case "EventId":
			if e.EventId != attr.AttributeValue {
				return false
			}
		case "ReadOnly":
			if e.ReadOnly != attr.AttributeValue {
				return false
			}
		case "AccessKeyId":
			if e.AccessKeyId != attr.AttributeValue {
				return false
			}
		}
	}
	return true
}

// AddTags adds tags to a trail.
func (s *Store) AddTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, trail := range s.trails {
		if trail.TrailARN == arn {
			for _, newTag := range tags {
				found := false
				for i, existing := range trail.Tags {
					if existing.Key == newTag.Key {
						trail.Tags[i].Value = newTag.Value
						found = true
						break
					}
				}
				if !found {
					trail.Tags = append(trail.Tags, newTag)
				}
			}
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Resource %s not found.", arn), http.StatusNotFound)
}

// RemoveTags removes tags from a trail.
func (s *Store) RemoveTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, trail := range s.trails {
		if trail.TrailARN == arn {
			for _, removeTag := range tags {
				for i, existing := range trail.Tags {
					if existing.Key == removeTag.Key {
						trail.Tags = append(trail.Tags[:i], trail.Tags[i+1:]...)
						break
					}
				}
			}
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Resource %s not found.", arn), http.StatusNotFound)
}

// ListTagsByARN returns tags for a trail by ARN.
func (s *Store) ListTagsByARN(arn string) ([]Tag, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, trail := range s.trails {
		if trail.TrailARN == arn {
			return trail.Tags, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Resource %s not found.", arn), http.StatusNotFound)
}
