package duckdb

import (
	"encoding/json"
	"time"
)

// spanRow is the internal representation matching the DuckDB spans table.
type spanRow struct {
	TraceID        string
	SpanID         string
	ParentSpanID   string
	StartTime      time.Time
	EndTime        time.Time
	DurationNs     int64
	ServiceName    string
	Action         string
	Method         string
	Path           string
	StatusCode     int
	Error          string
	TenantID       string
	OrgID          string
	UserID         string
	MemAllocKB     float64
	Goroutines     int
	Metadata       map[string]string
	RequestHeaders map[string]string
	RequestBody    string
	ResponseBody   string
}

// marshalMap serializes a map to a JSON string, returning "" for nil maps.
func marshalMap(m map[string]string) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalMap deserializes a JSON string to a map, returning nil on empty/error.
func unmarshalMap(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
