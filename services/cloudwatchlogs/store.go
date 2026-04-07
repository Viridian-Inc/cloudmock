package cloudwatchlogs

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// LogEvent is a single log event with timestamp (epoch millis) and message.
type LogEvent struct {
	Timestamp     int64
	Message       string
	IngestionTime int64
}

// LogStream holds metadata and events for a single log stream.
type LogStream struct {
	Name                string
	ARN                 string
	CreationTime        int64
	FirstEventTimestamp int64
	LastEventTimestamp  int64
	LastIngestionTime   int64
	UploadSequenceToken string
	Events              []LogEvent
}

// LogGroup holds metadata and streams for a single log group.
type LogGroup struct {
	Name          string
	ARN           string
	CreationTime  int64
	RetentionDays int
	StoredBytes   int64
	Tags          map[string]string
	Streams       map[string]*LogStream
}

// Store is the in-memory store for CloudWatch Logs resources.
type Store struct {
	mu        sync.RWMutex
	groups    map[string]*LogGroup // keyed by log group name
	accountID string
	region    string
}

// NewStore creates an empty CloudWatch Logs Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		groups:    make(map[string]*LogGroup),
		accountID: accountID,
		region:    region,
	}
}

// logGroupARN constructs an ARN for a log group.
func (s *Store) logGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s", s.region, s.accountID, name)
}

// logStreamARN constructs an ARN for a log stream.
func (s *Store) logStreamARN(groupName, streamName string) string {
	return fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s:log-stream:%s",
		s.region, s.accountID, groupName, streamName)
}

// newSequenceToken generates a random sequence token string.
func newSequenceToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// nowMillis returns the current UTC time in milliseconds since epoch.
func nowMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ---- Log Group operations ----

// CreateLogGroup creates a new log group. Returns AlreadyExistsException if it exists.
func (s *Store) CreateLogGroup(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[name]; ok {
		return service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("The specified log group already exists: %s", name),
			http.StatusBadRequest)
	}

	s.groups[name] = &LogGroup{
		Name:         name,
		ARN:          s.logGroupARN(name),
		CreationTime: nowMillis(),
		Tags:         make(map[string]string),
		Streams:      make(map[string]*LogStream),
	}
	return nil
}

// DeleteLogGroup removes a log group and all its streams.
func (s *Store) DeleteLogGroup(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("The specified log group does not exist: %s", name),
			http.StatusBadRequest)
	}
	delete(s.groups, name)
	return nil
}

// DescribeLogGroups returns log groups, optionally filtered by prefix.
func (s *Store) DescribeLogGroups(prefix string) []*LogGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*LogGroup
	for _, g := range s.groups {
		if prefix == "" || strings.HasPrefix(g.Name, prefix) {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// getGroupLocked returns the group if it exists; caller must hold at least read lock.
func (s *Store) getGroupLocked(name string) (*LogGroup, *service.AWSError) {
	g, ok := s.groups[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("The specified log group does not exist: %s", name),
			http.StatusBadRequest)
	}
	return g, nil
}

// ---- Log Stream operations ----

// CreateLogStream creates a new stream inside a group.
func (s *Store) CreateLogStream(groupName, streamName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}

	if _, ok := g.Streams[streamName]; ok {
		return service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("The specified log stream already exists: %s", streamName),
			http.StatusBadRequest)
	}

	g.Streams[streamName] = &LogStream{
		Name:                streamName,
		ARN:                 s.logStreamARN(groupName, streamName),
		CreationTime:        nowMillis(),
		UploadSequenceToken: newSequenceToken(),
	}
	return nil
}

// DeleteLogStream removes a stream from a group.
func (s *Store) DeleteLogStream(groupName, streamName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}

	if _, ok := g.Streams[streamName]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("The specified log stream does not exist: %s", streamName),
			http.StatusBadRequest)
	}
	delete(g.Streams, streamName)
	return nil
}

// DescribeLogStreams returns streams for a group, optionally filtered by prefix.
func (s *Store) DescribeLogStreams(groupName, prefix string) ([]*LogStream, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return nil, awsErr
	}

	var out []*LogStream
	for _, st := range g.Streams {
		if prefix == "" || strings.HasPrefix(st.Name, prefix) {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ---- Event operations ----

// PutLogEvents appends events to a stream and returns the next sequence token.
func (s *Store) PutLogEvents(groupName, streamName string, events []LogEvent) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return "", awsErr
	}

	st, ok := g.Streams[streamName]
	if !ok {
		return "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("The specified log stream does not exist: %s", streamName),
			http.StatusBadRequest)
	}

	ingestionTime := nowMillis()
	for i := range events {
		events[i].IngestionTime = ingestionTime
		st.Events = append(st.Events, events[i])

		if st.FirstEventTimestamp == 0 || events[i].Timestamp < st.FirstEventTimestamp {
			st.FirstEventTimestamp = events[i].Timestamp
		}
		if events[i].Timestamp > st.LastEventTimestamp {
			st.LastEventTimestamp = events[i].Timestamp
		}
	}
	st.LastIngestionTime = ingestionTime
	st.UploadSequenceToken = newSequenceToken()

	// Update group stored bytes (approximate: sum of message lengths).
	for _, ev := range events {
		g.StoredBytes += int64(len(ev.Message))
	}

	return st.UploadSequenceToken, nil
}

// GetLogEvents retrieves events from a stream with optional time filtering and limit.
func (s *Store) GetLogEvents(groupName, streamName string, startTime, endTime int64, limit int) ([]LogEvent, string, string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return nil, "", "", awsErr
	}

	st, ok := g.Streams[streamName]
	if !ok {
		return nil, "", "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("The specified log stream does not exist: %s", streamName),
			http.StatusBadRequest)
	}

	var filtered []LogEvent
	for _, ev := range st.Events {
		if startTime > 0 && ev.Timestamp < startTime {
			continue
		}
		if endTime > 0 && ev.Timestamp > endTime {
			continue
		}
		filtered = append(filtered, ev)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// Produce simple forward/backward tokens (opaque strings).
	nextForward := fmt.Sprintf("f/%s/%d", streamName, len(filtered))
	nextBackward := fmt.Sprintf("b/%s/%d", streamName, 0)

	return filtered, nextForward, nextBackward, nil
}

// FilterLogEvents searches across one or more streams for events matching a substring pattern.
func (s *Store) FilterLogEvents(groupName string, streamNames []string, filterPattern string, startTime, endTime int64) ([]LogEvent, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return nil, awsErr
	}

	// Determine which streams to search.
	var targets []*LogStream
	if len(streamNames) > 0 {
		for _, name := range streamNames {
			if st, ok := g.Streams[name]; ok {
				targets = append(targets, st)
			}
		}
	} else {
		for _, st := range g.Streams {
			targets = append(targets, st)
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })

	var out []LogEvent
	for _, st := range targets {
		for _, ev := range st.Events {
			if startTime > 0 && ev.Timestamp < startTime {
				continue
			}
			if endTime > 0 && ev.Timestamp > endTime {
				continue
			}
			if filterPattern != "" && !strings.Contains(ev.Message, filterPattern) {
				continue
			}
			out = append(out, ev)
		}
	}
	return out, nil
}

// ---- Retention Policy ----

// PutRetentionPolicy sets retentionInDays on a log group.
func (s *Store) PutRetentionPolicy(groupName string, retentionDays int) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}
	g.RetentionDays = retentionDays
	return nil
}

// DeleteRetentionPolicy removes the retention policy from a log group.
func (s *Store) DeleteRetentionPolicy(groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}
	g.RetentionDays = 0
	return nil
}

// ---- Tag operations ----

// TagLogGroup adds or updates tags on a log group.
func (s *Store) TagLogGroup(groupName string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}
	for k, v := range tags {
		g.Tags[k] = v
	}
	return nil
}

// UntagLogGroup removes the specified tag keys from a log group.
func (s *Store) UntagLogGroup(groupName string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return awsErr
	}
	for _, k := range tagKeys {
		delete(g.Tags, k)
	}
	return nil
}

// ListTagsLogGroup returns the tags for a log group.
func (s *Store) ListTagsLogGroup(groupName string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, awsErr := s.getGroupLocked(groupName)
	if awsErr != nil {
		return nil, awsErr
	}

	// Return a copy.
	out := make(map[string]string, len(g.Tags))
	for k, v := range g.Tags {
		out[k] = v
	}
	return out, nil
}
