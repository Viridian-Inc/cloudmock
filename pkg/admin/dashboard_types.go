package admin

import "time"

// Dashboard represents a user-defined dashboard containing widgets.
type Dashboard struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Widgets     []Widget  `json:"widgets"`
	Owner       string    `json:"owner,omitempty"`
	Visibility  string    `json:"visibility,omitempty"` // "private" or "team"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Widget is a single chart or metric card placed on a dashboard.
type Widget struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Type     string       `json:"type"` // "timeseries", "scalar", "table"
	Query    string       `json:"query"`
	Position GridPosition `json:"position"`
	Size     GridSize     `json:"size"`
}

// GridPosition is the x,y origin of a widget on the dashboard grid.
type GridPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// GridSize is the width and height of a widget in grid units.
type GridSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

// MetricQuery is the parsed representation of a DSL query string such as
// "avg:latency_ms{service:dynamodb}".
type MetricQuery struct {
	Aggregation string            `json:"aggregation"` // avg, sum, max, min, count, p50, p95, p99
	Metric      string            `json:"metric"`      // latency_ms, request_count, error_count, error_rate
	Filters     map[string]string `json:"filters"`     // service, action, method, status
}

// QueryResult is the response from executing a MetricQuery.
type QueryResult struct {
	Query   string        `json:"query"`
	Mode    string        `json:"mode"` // "timeseries" or "scalar"
	Scalar  *float64      `json:"scalar,omitempty"`
	Buckets []QueryBucket `json:"buckets,omitempty"`
}

// QueryBucket is a single time bucket in a timeseries query result.
type QueryBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
