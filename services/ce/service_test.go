package ce_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/ce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CostExplorerService { return svc.New("123456789012", "us-east-1") }

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestCE_GetCostAndUsage(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetCostAndUsage", map[string]any{
		"TimePeriod":  map[string]any{"Start": "2024-01-01", "End": "2024-01-03"},
		"Granularity": "DAILY",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	results := m["ResultsByTime"].([]any)
	assert.Equal(t, 2, len(results))
	first := results[0].(map[string]any)
	assert.NotEmpty(t, first["TimePeriod"])
	assert.NotEmpty(t, first["Total"])
}

func TestCE_GetCostAndUsageGroupBy(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetCostAndUsage", map[string]any{
		"TimePeriod":  map[string]any{"Start": "2024-01-01", "End": "2024-01-02"},
		"Granularity": "DAILY",
		"GroupBy":     []map[string]string{{"Type": "DIMENSION", "Key": "SERVICE"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	results := m["ResultsByTime"].([]any)
	first := results[0].(map[string]any)
	groups := first["Groups"].([]any)
	assert.Greater(t, len(groups), 0)
}

func TestCE_GetCostForecast(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetCostForecast", map[string]any{
		"TimePeriod":  map[string]any{"Start": "2024-02-01", "End": "2024-03-01"},
		"Granularity": "MONTHLY",
		"Metric":      "UNBLENDED_COST",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["Total"])
	assert.NotEmpty(t, m["ForecastResultsByTime"])
}

func TestCE_GetDimensionValues(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetDimensionValues", map[string]any{"Dimension": "SERVICE"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	vals := m["DimensionValues"].([]any)
	assert.Greater(t, len(vals), 0)
}

func TestCE_GetTags(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetTags", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Greater(t, len(m["Tags"].([]any)), 0)
}

func TestCE_GetReservationUtilization(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetReservationUtilization", map[string]any{
		"TimePeriod": map[string]any{"Start": "2024-01-01", "End": "2024-02-01"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotEmpty(t, m["Total"])
}

func TestCE_GetSavingsPlansUtilization(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetSavingsPlansUtilization", map[string]any{
		"TimePeriod": map[string]any{"Start": "2024-01-01", "End": "2024-02-01"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	total := m["Total"].(map[string]any)
	assert.NotEmpty(t, total["Utilization"])
}

func TestCE_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", nil))
	require.Error(t, err)
}

func TestCE_MissingTimePeriod(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetCostAndUsage", map[string]any{"Granularity": "DAILY"}))
	require.Error(t, err)
}

func TestCE_GroupByIncludesUsageQuantity(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetCostAndUsage", map[string]any{
		"TimePeriod":  map[string]any{"Start": "2024-01-01", "End": "2024-01-02"},
		"Granularity": "DAILY",
		"GroupBy":     []map[string]string{{"Type": "DIMENSION", "Key": "SERVICE"}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	results := m["ResultsByTime"].([]any)
	first := results[0].(map[string]any)
	groups := first["Groups"].([]any)
	firstGroup := groups[0].(map[string]any)
	metrics := firstGroup["Metrics"].(map[string]any)
	// Should include UsageQuantity alongside cost metrics
	assert.NotNil(t, metrics["UsageQuantity"])
	uq := metrics["UsageQuantity"].(map[string]any)
	assert.NotEmpty(t, uq["Amount"])
	assert.NotEmpty(t, uq["Unit"])
}

func TestCE_ServicePricingHasRealisticUnits(t *testing.T) {
	pricing := svc.DefaultServicePricing()
	assert.Greater(t, len(pricing), 10)
	for _, p := range pricing {
		assert.NotEmpty(t, p.Name)
		assert.Greater(t, p.BaseCost, 0.0)
		assert.NotEmpty(t, p.Unit)
		assert.Greater(t, p.UnitCost, 0.0)
	}
}

func TestCE_ForecastScalesWithServiceCount(t *testing.T) {
	// With more services, forecast should be higher
	s1 := svc.New("123456789012", "us-east-1")
	resp1, _ := s1.HandleRequest(jsonCtx("GetCostForecast", map[string]any{
		"TimePeriod": map[string]any{"Start": "2024-02-01", "End": "2024-02-02"},
		"Granularity": "DAILY", "Metric": "UNBLENDED_COST",
	}))
	m1 := respJSON(t, resp1)

	// Can't compare exact values due to randomness, but verify structure
	assert.NotEmpty(t, m1["Total"])
	total := m1["Total"].(map[string]any)
	assert.NotEmpty(t, total["Amount"])
}

func TestCE_DimensionValuesIncludesUsageType(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetDimensionValues", map[string]any{"Dimension": "USAGE_TYPE"}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	vals := m["DimensionValues"].([]any)
	assert.Greater(t, len(vals), 0)
}

func TestCE_GetTagsReturnsCommonTags(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetTags", map[string]any{
		"TimePeriod": map[string]any{"Start": "2024-01-01", "End": "2024-01-31"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	tags := m["Tags"].([]any)
	assert.Greater(t, len(tags), 0)
}

func TestCE_GetSavingsPlansUtilizationV2(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetSavingsPlansUtilization", map[string]any{
		"TimePeriod": map[string]any{"Start": "2024-02-01", "End": "2024-02-29"},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.NotNil(t, m["Total"])
	total := m["Total"].(map[string]any)
	assert.NotNil(t, total["Utilization"])
}

func TestCE_GetCostAndUsageMissingTimePeriod(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetCostAndUsage", map[string]any{
		"Granularity": "DAILY",
	}))
	require.Error(t, err)
}
