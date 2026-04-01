package admin

import (
	"testing"
)

func TestParseMetricQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MetricQuery
		wantErr bool
	}{
		{
			name:  "simple avg latency",
			input: "avg:latency_ms",
			want: MetricQuery{
				Aggregation: "avg",
				Metric:      "latency_ms",
				Filters:     map[string]string{},
			},
		},
		{
			name:  "avg latency with service filter",
			input: "avg:latency_ms{service:dynamodb}",
			want: MetricQuery{
				Aggregation: "avg",
				Metric:      "latency_ms",
				Filters:     map[string]string{"service": "dynamodb"},
			},
		},
		{
			name:  "p99 latency with multiple filters",
			input: "p99:latency_ms{service:s3,method:PUT}",
			want: MetricQuery{
				Aggregation: "p99",
				Metric:      "latency_ms",
				Filters:     map[string]string{"service": "s3", "method": "PUT"},
			},
		},
		{
			name:  "count request_count",
			input: "count:request_count{service:sqs}",
			want: MetricQuery{
				Aggregation: "count",
				Metric:      "request_count",
				Filters:     map[string]string{"service": "sqs"},
			},
		},
		{
			name:  "sum error_count with action filter",
			input: "sum:error_count{action:PutItem}",
			want: MetricQuery{
				Aggregation: "sum",
				Metric:      "error_count",
				Filters:     map[string]string{"action": "PutItem"},
			},
		},
		{
			name:  "max error_rate with status filter",
			input: "max:error_rate{status:500}",
			want: MetricQuery{
				Aggregation: "max",
				Metric:      "error_rate",
				Filters:     map[string]string{"status": "500"},
			},
		},
		{
			name:  "p50 no filter",
			input: "p50:latency_ms",
			want: MetricQuery{
				Aggregation: "p50",
				Metric:      "latency_ms",
				Filters:     map[string]string{},
			},
		},
		{
			name:  "min with whitespace",
			input: "  min:latency_ms{service:dynamodb}  ",
			want: MetricQuery{
				Aggregation: "min",
				Metric:      "latency_ms",
				Filters:     map[string]string{"service": "dynamodb"},
			},
		},
		{
			name:  "p95 with all filters",
			input: "p95:latency_ms{service:dynamodb,action:Query,method:POST,status:200}",
			want: MetricQuery{
				Aggregation: "p95",
				Metric:      "latency_ms",
				Filters: map[string]string{
					"service": "dynamodb",
					"action":  "Query",
					"method":  "POST",
					"status":  "200",
				},
			},
		},
		// Error cases
		{
			name:    "empty query",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "avglatency_ms",
			wantErr: true,
		},
		{
			name:    "unsupported aggregation",
			input:   "median:latency_ms",
			wantErr: true,
		},
		{
			name:    "unsupported metric",
			input:   "avg:cpu_usage",
			wantErr: true,
		},
		{
			name:    "unclosed filter block",
			input:   "avg:latency_ms{service:s3",
			wantErr: true,
		},
		{
			name:    "invalid filter pair",
			input:   "avg:latency_ms{service}",
			wantErr: true,
		},
		{
			name:    "unsupported filter key",
			input:   "avg:latency_ms{region:us-east-1}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMetricQuery(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMetricQuery(%q) expected error, got %+v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMetricQuery(%q) unexpected error: %v", tt.input, err)
			}
			if got.Aggregation != tt.want.Aggregation {
				t.Errorf("aggregation: got %q, want %q", got.Aggregation, tt.want.Aggregation)
			}
			if got.Metric != tt.want.Metric {
				t.Errorf("metric: got %q, want %q", got.Metric, tt.want.Metric)
			}
			if len(got.Filters) != len(tt.want.Filters) {
				t.Errorf("filters length: got %d, want %d", len(got.Filters), len(tt.want.Filters))
			}
			for k, v := range tt.want.Filters {
				if got.Filters[k] != v {
					t.Errorf("filter %q: got %q, want %q", k, got.Filters[k], v)
				}
			}
		})
	}
}
