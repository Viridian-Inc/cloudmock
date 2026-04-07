package cloudtrail_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/eventbus"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/cloudtrail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CloudTrailService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func createTrail(t *testing.T, s *svc.CloudTrailService, name string) map[string]any {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateTrail", map[string]any{
		"Name":         name,
		"S3BucketName": "my-trail-bucket",
	}))
	require.NoError(t, err)
	return respBody(t, resp)
}

func TestCloudTrail_CreateTrail(t *testing.T) {
	s := newService()
	m := createTrail(t, s, "test-trail")
	assert.Equal(t, "test-trail", m["Name"])
	assert.Contains(t, m["TrailARN"].(string), "arn:aws:cloudtrail:")
	assert.Equal(t, "my-trail-bucket", m["S3BucketName"])
}

func TestCloudTrail_CreateTrail_MissingBucket(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateTrail", map[string]any{
		"Name": "no-bucket-trail",
	}))
	require.Error(t, err)
}

func TestCloudTrail_CreateTrail_Duplicate(t *testing.T) {
	s := newService()
	createTrail(t, s, "dup-trail")
	_, err := s.HandleRequest(jsonCtx("CreateTrail", map[string]any{
		"Name":         "dup-trail",
		"S3BucketName": "bucket",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TrailAlreadyExistsException")
}

func TestCloudTrail_GetTrail(t *testing.T) {
	s := newService()
	createTrail(t, s, "my-trail")
	resp, err := s.HandleRequest(jsonCtx("GetTrail", map[string]any{"Name": "my-trail"}))
	require.NoError(t, err)
	trail := respBody(t, resp)["Trail"].(map[string]any)
	assert.Equal(t, "my-trail", trail["Name"])
}

func TestCloudTrail_GetTrail_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetTrail", map[string]any{"Name": "nonexistent"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TrailNotFoundException")
}

func TestCloudTrail_DescribeTrails(t *testing.T) {
	s := newService()
	createTrail(t, s, "trail-1")
	createTrail(t, s, "trail-2")
	resp, err := s.HandleRequest(jsonCtx("DescribeTrails", map[string]any{}))
	require.NoError(t, err)
	list := respBody(t, resp)["trailList"].([]any)
	assert.Len(t, list, 2)
}

func TestCloudTrail_DeleteTrail(t *testing.T) {
	s := newService()
	createTrail(t, s, "delete-me")
	_, err := s.HandleRequest(jsonCtx("DeleteTrail", map[string]any{"Name": "delete-me"}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("GetTrail", map[string]any{"Name": "delete-me"}))
	require.Error(t, err)
}

func TestCloudTrail_UpdateTrail(t *testing.T) {
	s := newService()
	createTrail(t, s, "update-trail")
	resp, err := s.HandleRequest(jsonCtx("UpdateTrail", map[string]any{
		"Name":               "update-trail",
		"S3BucketName":       "new-bucket",
		"IsMultiRegionTrail": true,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "new-bucket", m["S3BucketName"])
	assert.Equal(t, true, m["IsMultiRegionTrail"])
}

func TestCloudTrail_StartStopLogging(t *testing.T) {
	s := newService()
	createTrail(t, s, "logging-trail")

	_, err := s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "logging-trail"}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("GetTrailStatus", map[string]any{"Name": "logging-trail"}))
	m := respBody(t, resp)
	assert.Equal(t, true, m["IsLogging"])
	assert.NotNil(t, m["StartLoggingTime"])

	_, err = s.HandleRequest(jsonCtx("StopLogging", map[string]any{"Name": "logging-trail"}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("GetTrailStatus", map[string]any{"Name": "logging-trail"}))
	m = respBody(t, resp)
	assert.Equal(t, false, m["IsLogging"])
	assert.NotNil(t, m["StopLoggingTime"])
}

func TestCloudTrail_EventSelectors(t *testing.T) {
	s := newService()
	createTrail(t, s, "es-trail")
	resp, err := s.HandleRequest(jsonCtx("PutEventSelectors", map[string]any{
		"TrailName": "es-trail",
		"EventSelectors": []any{
			map[string]any{
				"ReadWriteType":           "ReadOnly",
				"IncludeManagementEvents": true,
				"DataResources": []any{
					map[string]any{"Type": "AWS::S3::Object", "Values": []any{"arn:aws:s3:::my-bucket/"}},
				},
			},
		},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	selectors := m["EventSelectors"].([]any)
	assert.Len(t, selectors, 1)
	sel := selectors[0].(map[string]any)
	assert.Equal(t, "ReadOnly", sel["ReadWriteType"])

	resp, _ = s.HandleRequest(jsonCtx("GetEventSelectors", map[string]any{"TrailName": "es-trail"}))
	m = respBody(t, resp)
	selectors = m["EventSelectors"].([]any)
	assert.Len(t, selectors, 1)
}

func TestCloudTrail_InsightSelectors(t *testing.T) {
	s := newService()
	createTrail(t, s, "insight-trail")
	resp, err := s.HandleRequest(jsonCtx("PutInsightSelectors", map[string]any{
		"TrailName":        "insight-trail",
		"InsightSelectors": []any{map[string]any{"InsightType": "ApiCallRateInsight"}},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Len(t, m["InsightSelectors"].([]any), 1)

	resp, _ = s.HandleRequest(jsonCtx("GetInsightSelectors", map[string]any{"TrailName": "insight-trail"}))
	m = respBody(t, resp)
	assert.Len(t, m["InsightSelectors"].([]any), 1)
}

func TestCloudTrail_LookupEvents(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("LookupEvents", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotNil(t, m["Events"])
}

func TestCloudTrail_Tagging(t *testing.T) {
	s := newService()
	m := createTrail(t, s, "tag-trail")
	arn := m["TrailARN"].(string)

	_, err := s.HandleRequest(jsonCtx("AddTags", map[string]any{
		"ResourceId": arn,
		"TagsList":   []any{map[string]any{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("ListTags", map[string]any{
		"ResourceIdList": []any{arn},
	}))
	require.NoError(t, err)
	rtl := respBody(t, resp)["ResourceTagList"].([]any)
	assert.Len(t, rtl, 1)
	entry := rtl[0].(map[string]any)
	tags := entry["TagsList"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("RemoveTags", map[string]any{
		"ResourceId": arn,
		"TagsList":   []any{map[string]any{"Key": "env"}},
	}))
	require.NoError(t, err)
}

func TestCloudTrail_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestCloudTrail_MissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateTrail", map[string]any{
		"S3BucketName": "bucket",
	}))
	require.Error(t, err)
}

// --- Behavioral Tests ---

func newServiceWithBus() (*svc.CloudTrailService, *eventbus.Bus) {
	bus := eventbus.NewBus()
	s := svc.NewWithBus("123456789012", "us-east-1", bus)
	return s, bus
}

func TestCloudTrail_EventBus_RecordsEvents(t *testing.T) {
	s, bus := newServiceWithBus()
	createTrail(t, s, "bus-trail")

	// Start logging
	_, err := s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "bus-trail"}))
	require.NoError(t, err)

	// Publish an event via bus (synchronous for test)
	bus.PublishSync(&eventbus.Event{
		Source:    "s3",
		Type:      "s3:ApiCall:CreateBucket",
		Detail:    map[string]any{"username": "testuser", "resourceType": "AWS::S3::Bucket", "resourceName": "my-bucket"},
		Time:      time.Now().UTC(),
		Region:    "us-east-1",
		AccountID: "123456789012",
	})

	// Lookup events should return the recorded event
	resp, err := s.HandleRequest(jsonCtx("LookupEvents", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	events := m["Events"].([]any)
	assert.GreaterOrEqual(t, len(events), 1)

	// Find our event
	found := false
	for _, e := range events {
		em := e.(map[string]any)
		if em["EventName"] == "CreateBucket" {
			found = true
			assert.Equal(t, "s3.amazonaws.com", em["EventSource"])
			assert.Equal(t, "testuser", em["Username"])
			break
		}
	}
	assert.True(t, found, "CreateBucket event not found in LookupEvents")
}

func TestCloudTrail_EventBus_StopLoggingUnsubscribes(t *testing.T) {
	s, bus := newServiceWithBus()
	createTrail(t, s, "stop-trail")

	_, err := s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "stop-trail"}))
	require.NoError(t, err)

	// Stop logging
	_, err = s.HandleRequest(jsonCtx("StopLogging", map[string]any{"Name": "stop-trail"}))
	require.NoError(t, err)

	// Publish an event - should NOT be recorded
	bus.PublishSync(&eventbus.Event{
		Source: "ec2",
		Type:   "ec2:ApiCall:RunInstances",
		Detail: map[string]any{"username": "user2"},
		Time:   time.Now().UTC(),
	})

	resp, err := s.HandleRequest(jsonCtx("LookupEvents", map[string]any{}))
	require.NoError(t, err)
	events := respBody(t, resp)["Events"].([]any)
	// Should have no events since logging was stopped before the event
	for _, e := range events {
		em := e.(map[string]any)
		assert.NotEqual(t, "RunInstances", em["EventName"], "Event should not be recorded after StopLogging")
	}
}

func TestCloudTrail_LookupEvents_FilterByEventSource(t *testing.T) {
	s, bus := newServiceWithBus()
	createTrail(t, s, "filter-trail")

	_, _ = s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "filter-trail"}))

	bus.PublishSync(&eventbus.Event{
		Source: "s3", Type: "s3:ApiCall:PutObject",
		Detail: map[string]any{}, Time: time.Now().UTC(),
	})
	bus.PublishSync(&eventbus.Event{
		Source: "ec2", Type: "ec2:ApiCall:DescribeInstances",
		Detail: map[string]any{}, Time: time.Now().UTC(),
	})

	// Filter by EventSource
	resp, err := s.HandleRequest(jsonCtx("LookupEvents", map[string]any{
		"LookupAttributes": []any{
			map[string]any{"AttributeKey": "EventSource", "AttributeValue": "s3.amazonaws.com"},
		},
	}))
	require.NoError(t, err)
	events := respBody(t, resp)["Events"].([]any)
	for _, e := range events {
		em := e.(map[string]any)
		assert.Equal(t, "s3.amazonaws.com", em["EventSource"])
	}
}

func TestCloudTrail_LookupEvents_FilterByTimeRange(t *testing.T) {
	s, bus := newServiceWithBus()
	createTrail(t, s, "time-trail")

	_, _ = s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "time-trail"}))

	now := time.Now().UTC()
	bus.PublishSync(&eventbus.Event{
		Source: "iam", Type: "iam:ApiCall:CreateUser",
		Detail: map[string]any{}, Time: now,
	})

	// Query with a time range that excludes the event
	future := now.Add(1 * time.Hour)
	resp, err := s.HandleRequest(jsonCtx("LookupEvents", map[string]any{
		"StartTime": float64(future.Unix()),
	}))
	require.NoError(t, err)
	events := respBody(t, resp)["Events"].([]any)
	assert.Len(t, events, 0)
}

func TestCloudTrail_GetTrailStatus_LatestDeliveryTime(t *testing.T) {
	s, bus := newServiceWithBus()
	createTrail(t, s, "status-trail")

	_, _ = s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "status-trail"}))

	bus.PublishSync(&eventbus.Event{
		Source: "lambda", Type: "lambda:ApiCall:Invoke",
		Detail: map[string]any{}, Time: time.Now().UTC(),
	})

	resp, err := s.HandleRequest(jsonCtx("GetTrailStatus", map[string]any{"Name": "status-trail"}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, true, m["IsLogging"])
	assert.NotNil(t, m["LatestDeliveryTime"])
}

func TestCloudTrail_NoBus_DegradeGracefully(t *testing.T) {
	// Without a bus, StartLogging should still work fine
	s := newService()
	createTrail(t, s, "no-bus-trail")
	_, err := s.HandleRequest(jsonCtx("StartLogging", map[string]any{"Name": "no-bus-trail"}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("StopLogging", map[string]any{"Name": "no-bus-trail"}))
	require.NoError(t, err)
}
