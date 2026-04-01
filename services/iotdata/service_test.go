package iotdata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/iotdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.IoTDataService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     make(map[string]string),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// ---- Test 1: UpdateThingShadow (create) ----

func TestUpdateThingShadow(t *testing.T) {
	s := newService()
	body := map[string]any{
		"state": map[string]any{
			"desired":  map[string]any{"temperature": 72},
			"reported": map[string]any{"temperature": 70},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	ctx := &service.RequestContext{
		Action:     "UpdateThingShadow",
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     map[string]string{"thingName": "sensor-1"},
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}

	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	rBody := respBody(t, resp)
	assert.NotNil(t, rBody["state"])
	assert.Equal(t, float64(1), rBody["version"])
}

// ---- Test 2: GetThingShadow ----

func TestGetThingShadow(t *testing.T) {
	s := newService()
	// First create a shadow
	body := map[string]any{
		"state": map[string]any{"desired": map[string]any{"temp": 75}},
	}
	bodyBytes, _ := json.Marshal(body)
	ctx := &service.RequestContext{
		Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: bodyBytes, Params: map[string]string{"thingName": "sensor-get"},
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := s.HandleRequest(ctx)
	require.NoError(t, err)

	// Now get it
	resp, err := s.HandleRequest(jsonCtx("GetThingShadow", map[string]any{"thingName": "sensor-get"}))
	require.NoError(t, err)
	rBody := respBody(t, resp)
	assert.NotNil(t, rBody["state"])
}

// ---- Test 3: DeleteThingShadow ----

func TestDeleteThingShadow(t *testing.T) {
	s := newService()
	// Create shadow
	body := map[string]any{"state": map[string]any{"desired": map[string]any{"x": 1}}}
	bodyBytes, _ := json.Marshal(body)
	ctx := &service.RequestContext{
		Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: bodyBytes, Params: map[string]string{"thingName": "sensor-del"},
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := s.HandleRequest(ctx)
	require.NoError(t, err)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteThingShadow", map[string]any{"thingName": "sensor-del"}))
	require.NoError(t, err)

	// Verify gone
	_, err = s.HandleRequest(jsonCtx("GetThingShadow", map[string]any{"thingName": "sensor-del"}))
	require.Error(t, err)
}

// ---- Test 4: GetThingShadow NotFound ----

func TestGetThingShadowNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetThingShadow", map[string]any{"thingName": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// ---- Test 5: ListNamedShadowsForThing ----

func TestListNamedShadowsForThing(t *testing.T) {
	s := newService()
	// Create named shadows
	for _, shadowName := range []string{"shadow-a", "shadow-b"} {
		body := map[string]any{"state": map[string]any{"desired": map[string]any{"x": 1}}}
		bodyBytes, _ := json.Marshal(body)
		ctx := &service.RequestContext{
			Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
			Body: bodyBytes, Params: map[string]string{"thingName": "sensor-named", "shadowName": shadowName},
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
		}
		_, err := s.HandleRequest(ctx)
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListNamedShadowsForThing", map[string]any{"thingName": "sensor-named"}))
	require.NoError(t, err)
	rBody := respBody(t, resp)
	results := rBody["results"].([]any)
	assert.Len(t, results, 2)
}

// ---- Test 6: Publish ----

func TestPublish(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("Publish", map[string]any{
		"topic": "sensors/temperature",
		"qos":   float64(1),
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ---- Test 7: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 8: UpdateThingShadow increments version ----

func TestShadowVersionIncrement(t *testing.T) {
	s := newService()
	for i := 0; i < 3; i++ {
		body := map[string]any{"state": map[string]any{"desired": map[string]any{"count": i}}}
		bodyBytes, _ := json.Marshal(body)
		ctx := &service.RequestContext{
			Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
			Body: bodyBytes, Params: map[string]string{"thingName": "sensor-ver"},
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
		}
		_, err := s.HandleRequest(ctx)
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("GetThingShadow", map[string]any{"thingName": "sensor-ver"}))
	require.NoError(t, err)
	rBody := respBody(t, resp)
	assert.Equal(t, float64(3), rBody["version"])
}

// ---- Test 9: Named shadow get/update/delete ----

func TestNamedShadow(t *testing.T) {
	s := newService()
	// Create named shadow
	body := map[string]any{"state": map[string]any{"desired": map[string]any{"x": 1}}}
	bodyBytes, _ := json.Marshal(body)
	ctx := &service.RequestContext{
		Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: bodyBytes, Params: map[string]string{"thingName": "sensor-ns", "shadowName": "myNamedShadow"},
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := s.HandleRequest(ctx)
	require.NoError(t, err)

	// Get named shadow
	getCtx := &service.RequestContext{
		Action: "GetThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: []byte("{}"), Params: map[string]string{"thingName": "sensor-ns", "shadowName": "myNamedShadow"},
		RawRequest: httptest.NewRequest(http.MethodGet, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	resp, err := s.HandleRequest(getCtx)
	require.NoError(t, err)
	assert.NotNil(t, respBody(t, resp)["state"])

	// Delete named shadow
	delCtx := &service.RequestContext{
		Action: "DeleteThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: []byte("{}"), Params: map[string]string{"thingName": "sensor-ns", "shadowName": "myNamedShadow"},
		RawRequest: httptest.NewRequest(http.MethodDelete, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err = s.HandleRequest(delCtx)
	require.NoError(t, err)
}

// ---- Test 10: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "iot-data", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 11: DeleteThingShadow NotFound ----

func TestDeleteThingShadowNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteThingShadow", map[string]any{"thingName": "nonexistent"}))
	require.Error(t, err)
}

// ---- Test 12: Shadow delta computation ----

func TestShadowDeltaComputation(t *testing.T) {
	s := newService()
	// Create shadow with desired state
	body := map[string]any{"state": map[string]any{
		"desired":  map[string]any{"temperature": 72, "mode": "cool"},
		"reported": map[string]any{"temperature": 70, "mode": "cool"},
	}}
	bodyBytes, _ := json.Marshal(body)
	ctx := &service.RequestContext{
		Action: "UpdateThingShadow", Region: "us-east-1", AccountID: "123456789012",
		Body: bodyBytes, Params: map[string]string{"thingName": "sensor-delta"},
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := s.HandleRequest(ctx)
	require.NoError(t, err)

	// Get shadow and check delta
	resp, err := s.HandleRequest(jsonCtx("GetThingShadow", map[string]any{"thingName": "sensor-delta"}))
	require.NoError(t, err)
	rBody := respBody(t, resp)
	state := rBody["state"].(map[string]any)
	delta, ok := state["delta"].(map[string]any)
	require.True(t, ok, "expected delta to exist when desired != reported")
	// temperature differs: desired=72, reported=70
	assert.Equal(t, float64(72), delta["temperature"])
	// mode is same, should not be in delta
	_, hasModeInDelta := delta["mode"]
	assert.False(t, hasModeInDelta)
}

// ---- Test 13: ListNamedShadows empty ----

func TestListNamedShadowsEmpty(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListNamedShadowsForThing", map[string]any{"thingName": "no-shadows"}))
	require.NoError(t, err)
	rBody := respBody(t, resp)
	results := rBody["results"].([]any)
	assert.Len(t, results, 0)
}
