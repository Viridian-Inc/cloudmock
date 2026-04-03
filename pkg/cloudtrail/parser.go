// Package cloudtrail provides CloudTrail event parsing, conversion, and replay
// for recreating AWS resource state from audit logs.
package cloudtrail

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
)

// CloudTrailLog is the top-level wrapper for a CloudTrail JSON file.
type CloudTrailLog struct {
	Records []CloudTrailEvent `json:"Records"`
}

// CloudTrailEvent represents a single CloudTrail event record.
type CloudTrailEvent struct {
	EventVersion      string         `json:"eventVersion"`
	EventSource       string         `json:"eventSource"`
	EventName         string         `json:"eventName"`
	AWSRegion         string         `json:"awsRegion"`
	EventTime         string         `json:"eventTime"`
	SourceIPAddress   string         `json:"sourceIPAddress"`
	UserIdentity      map[string]any `json:"userIdentity"`
	RequestParameters map[string]any `json:"requestParameters"`
	ResponseElements  map[string]any `json:"responseElements"`
	ErrorCode         string         `json:"errorCode"`
	ErrorMessage      string         `json:"errorMessage"`
	ReadOnly          bool           `json:"readOnly"`
}

// ServiceName extracts the service from eventSource (e.g., "s3.amazonaws.com" -> "s3").
func (e *CloudTrailEvent) ServiceName() string {
	s := e.EventSource
	if idx := strings.Index(s, "."); idx > 0 {
		return s[:idx]
	}
	return s
}

// ParsedTime returns the eventTime parsed as time.Time.
func (e *CloudTrailEvent) ParsedTime() time.Time {
	t, err := time.Parse(time.RFC3339, e.EventTime)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseFile reads a CloudTrail JSON file and returns the events.
func ParseFile(path string) ([]CloudTrailEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseJSON(data)
}

// ParseJSON parses CloudTrail JSON data (a Records array) and returns the events.
func ParseJSON(data []byte) ([]CloudTrailEvent, error) {
	var log CloudTrailLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}
	return log.Records, nil
}

// FilterWriteEvents returns only events that modify state (not read-only).
func FilterWriteEvents(events []CloudTrailEvent) []CloudTrailEvent {
	var out []CloudTrailEvent
	for _, e := range events {
		if !e.ReadOnly {
			out = append(out, e)
		}
	}
	return out
}

// FilterByServices returns only events matching the given service names.
func FilterByServices(events []CloudTrailEvent, services []string) []CloudTrailEvent {
	if len(services) == 0 {
		return events
	}
	allowed := make(map[string]bool, len(services))
	for _, s := range services {
		allowed[strings.ToLower(s)] = true
	}
	var out []CloudTrailEvent
	for _, e := range events {
		if allowed[e.ServiceName()] {
			out = append(out, e)
		}
	}
	return out
}

// SortByTime sorts events chronologically by eventTime.
func SortByTime(events []CloudTrailEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].EventTime < events[j].EventTime
	})
}
