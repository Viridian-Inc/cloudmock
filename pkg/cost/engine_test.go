package cost

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// mockRequestReader implements dataplane.RequestReader with canned data.
type mockRequestReader struct {
	entries []dataplane.RequestEntry
}

func (m *mockRequestReader) Query(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	var result []dataplane.RequestEntry
	for _, e := range m.entries {
		if !filter.From.IsZero() && e.Timestamp.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.Timestamp.After(filter.To) {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

func (m *mockRequestReader) GetByID(ctx context.Context, id string) (*dataplane.RequestEntry, error) {
	for _, e := range m.entries {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, nil
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestRequestCost_Lambda(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service:   "Lambda",
		Method:    "POST",
		LatencyMs: 100,
	}

	got := eng.RequestCost(entry)
	// 100ms / 1000 * 128MB / 1024 * 0.0000166667
	want := (100.0 / 1000.0) * (128.0 / 1024.0) * 0.0000166667
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost Lambda = %e, want %e", got, want)
	}
}

func TestRequestCost_DynamoDB_Read(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service: "dynamodb",
		Method:  "GET",
	}

	got := eng.RequestCost(entry)
	want := pricing.DynamoDB.PerRCU
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost DynamoDB Read = %e, want %e", got, want)
	}
}

func TestRequestCost_DynamoDB_Write(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service: "dynamodb",
		Method:  "PUT",
	}

	got := eng.RequestCost(entry)
	want := pricing.DynamoDB.PerWCU
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost DynamoDB Write = %e, want %e", got, want)
	}
}

func TestRequestCost_S3_Get(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service: "s3",
		Method:  "GET",
	}

	got := eng.RequestCost(entry)
	want := pricing.S3.PerGET
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost S3 GET = %e, want %e", got, want)
	}
}

func TestRequestCost_S3_Put(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service: "s3",
		Method:  "PUT",
	}

	got := eng.RequestCost(entry)
	want := pricing.S3.PerPUT
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost S3 PUT = %e, want %e", got, want)
	}
}

func TestRequestCost_SQS(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service: "sqs",
		Method:  "POST",
	}

	got := eng.RequestCost(entry)
	want := pricing.SQS.PerRequest
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost SQS = %e, want %e", got, want)
	}
}

func TestRequestCost_DataTransfer(t *testing.T) {
	pricing := DefaultPricingConfig()
	eng := New(nil, pricing)

	body := strings.Repeat("x", 2048) // 2 KB
	entry := dataplane.RequestEntry{
		Service:      "sqs",
		Method:       "POST",
		ResponseBody: body,
	}

	got := eng.RequestCost(entry)
	transferCost := (2048.0 / 1024.0) * pricing.DataTransfer.PerKB
	want := pricing.SQS.PerRequest + transferCost
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost DataTransfer = %e, want %e", got, want)
	}
}

func TestRequestCost_CustomPricing(t *testing.T) {
	pricing := PricingConfig{
		Lambda: LambdaPricing{
			PerGBSecond:     0.001,
			DefaultMemoryMB: 256,
		},
	}
	eng := New(nil, pricing)

	entry := dataplane.RequestEntry{
		Service:   "lambda",
		Method:    "POST",
		LatencyMs: 200,
	}

	got := eng.RequestCost(entry)
	want := (200.0 / 1000.0) * (256.0 / 1024.0) * 0.001
	if !almostEqual(got, want, 1e-15) {
		t.Errorf("RequestCost CustomPricing = %e, want %e", got, want)
	}
}

func TestByService(t *testing.T) {
	reader := &mockRequestReader{
		entries: []dataplane.RequestEntry{
			{Service: "lambda", Method: "POST", LatencyMs: 100},
			{Service: "lambda", Method: "POST", LatencyMs: 200},
			{Service: "sqs", Method: "POST"},
			{Service: "s3", Method: "GET"},
		},
	}

	eng := New(reader, DefaultPricingConfig())
	result, err := eng.ByService(context.Background())
	if err != nil {
		t.Fatalf("ByService error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 services, got %d", len(result))
	}

	// Verify lambda has 2 requests
	for _, sc := range result {
		if sc.Service == "lambda" {
			if sc.RequestCount != 2 {
				t.Errorf("lambda RequestCount = %d, want 2", sc.RequestCount)
			}
			// Verify total is sum of both costs
			cost1 := eng.RequestCost(reader.entries[0])
			cost2 := eng.RequestCost(reader.entries[1])
			if !almostEqual(sc.TotalCost, cost1+cost2, 1e-15) {
				t.Errorf("lambda TotalCost = %e, want %e", sc.TotalCost, cost1+cost2)
			}
		}
	}

	// Verify sorted by TotalCost descending
	for i := 1; i < len(result); i++ {
		if result[i].TotalCost > result[i-1].TotalCost {
			t.Errorf("results not sorted by TotalCost desc at index %d", i)
		}
	}
}

func TestByRoute(t *testing.T) {
	reader := &mockRequestReader{
		entries: []dataplane.RequestEntry{
			{Service: "s3", Method: "GET", Path: "/bucket/a"},
			{Service: "s3", Method: "PUT", Path: "/bucket/b"},
			{Service: "s3", Method: "PUT", Path: "/bucket/b"},
			{Service: "s3", Method: "GET", Path: "/bucket/a"},
		},
	}

	eng := New(reader, DefaultPricingConfig())
	result, err := eng.ByRoute(context.Background(), 10)
	if err != nil {
		t.Fatalf("ByRoute error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(result))
	}

	// Sorted by cost desc; PUT costs more than GET
	if result[0].Method != "PUT" {
		t.Errorf("expected PUT route first (higher cost), got %s", result[0].Method)
	}
	if result[0].RequestCount != 2 {
		t.Errorf("PUT route RequestCount = %d, want 2", result[0].RequestCount)
	}

	// Test limit
	limited, err := eng.ByRoute(context.Background(), 1)
	if err != nil {
		t.Fatalf("ByRoute limit error: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 route with limit=1, got %d", len(limited))
	}
}

func TestByTenant(t *testing.T) {
	reader := &mockRequestReader{
		entries: []dataplane.RequestEntry{
			{Service: "sqs", Method: "POST", TenantID: "tenant-a"},
			{Service: "sqs", Method: "POST", TenantID: "tenant-a"},
			{Service: "sqs", Method: "POST", TenantID: "tenant-b"},
			{Service: "sqs", Method: "POST", TenantID: ""}, // should be skipped
		},
	}

	eng := New(reader, DefaultPricingConfig())
	result, err := eng.ByTenant(context.Background(), 10)
	if err != nil {
		t.Fatalf("ByTenant error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(result))
	}

	// tenant-a should be first (2 requests, higher total cost)
	if result[0].TenantID != "tenant-a" {
		t.Errorf("expected tenant-a first, got %s", result[0].TenantID)
	}
	if result[0].RequestCount != 2 {
		t.Errorf("tenant-a RequestCount = %d, want 2", result[0].RequestCount)
	}
	if result[1].TenantID != "tenant-b" {
		t.Errorf("expected tenant-b second, got %s", result[1].TenantID)
	}
}

func TestTrend(t *testing.T) {
	now := time.Now()
	reader := &mockRequestReader{
		entries: []dataplane.RequestEntry{
			{Service: "sqs", Method: "POST", Timestamp: now.Add(-30 * time.Minute)},
			{Service: "sqs", Method: "POST", Timestamp: now.Add(-90 * time.Minute)},
			{Service: "sqs", Method: "POST", Timestamp: now.Add(-150 * time.Minute)},
		},
	}

	eng := New(reader, DefaultPricingConfig())
	result, err := eng.Trend(context.Background(), 4*time.Hour, 1*time.Hour)
	if err != nil {
		t.Fatalf("Trend error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(result))
	}

	// Verify sorted by Start ascending
	for i := 1; i < len(result); i++ {
		if result[i].Start.Before(result[i-1].Start) {
			t.Errorf("buckets not sorted by Start asc at index %d", i)
		}
	}

	// Each bucket should have 1 request
	for i, b := range result {
		if b.RequestCount != 1 {
			t.Errorf("bucket %d RequestCount = %d, want 1", i, b.RequestCount)
		}
		if b.TotalCost <= 0 {
			t.Errorf("bucket %d TotalCost = %e, want > 0", i, b.TotalCost)
		}
	}
}
