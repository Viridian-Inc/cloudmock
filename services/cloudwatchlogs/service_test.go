package cloudwatchlogs_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	logssvc "github.com/neureaux/cloudmock/services/cloudwatchlogs"
)

// newLogsGateway builds a full gateway stack with the CloudWatch Logs service registered and IAM disabled.
func newLogsGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(logssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// logsReq builds a JSON POST request targeting the CloudWatch Logs service via X-Amz-Target.
func logsReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("logsReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328."+action)
	// Authorization header places "logs" as the service in the credential scope.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/logs/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// nowMs returns the current time in milliseconds since epoch.
func nowMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ---- Test 1: CreateLogGroup + DescribeLogGroups ----

func TestLogs_CreateLogGroup_DescribeLogGroups(t *testing.T) {
	handler := newLogsGateway(t)

	// CreateLogGroup
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/app/my-service",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateLogGroup: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	// Create a second group for prefix filtering
	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/other/service",
	}))
	if wc2.Code != http.StatusOK {
		t.Fatalf("CreateLogGroup second: expected 200, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}

	// DescribeLogGroups — all
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, logsReq(t, "DescribeLogGroups", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeLogGroups: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	groups, ok := md["logGroups"].([]any)
	if !ok {
		t.Fatalf("DescribeLogGroups: missing logGroups\nbody: %s", wd.Body.String())
	}
	if len(groups) < 2 {
		t.Errorf("DescribeLogGroups: expected at least 2 groups, got %d", len(groups))
	}

	// Verify fields on the first group.
	var appGroup map[string]any
	for _, g := range groups {
		entry := g.(map[string]any)
		if entry["logGroupName"].(string) == "/app/my-service" {
			appGroup = entry
			break
		}
	}
	if appGroup == nil {
		t.Fatalf("DescribeLogGroups: /app/my-service not found\nbody: %s", wd.Body.String())
	}
	arn, _ := appGroup["arn"].(string)
	if !strings.Contains(arn, "/app/my-service") {
		t.Errorf("DescribeLogGroups: ARN %q does not contain group name", arn)
	}
	if appGroup["creationTime"] == nil {
		t.Error("DescribeLogGroups: missing creationTime")
	}

	// DescribeLogGroups — prefix filter
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, logsReq(t, "DescribeLogGroups", map[string]string{
		"logGroupNamePrefix": "/app/",
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeLogGroups prefix: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	mf := decodeJSON(t, wf.Body.String())
	filtered, _ := mf["logGroups"].([]any)
	if len(filtered) != 1 {
		t.Errorf("DescribeLogGroups prefix: expected 1 group, got %d", len(filtered))
	}

	// AlreadyExists check
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/app/my-service",
	}))
	if we.Code != http.StatusBadRequest {
		t.Fatalf("CreateLogGroup duplicate: expected 400, got %d\nbody: %s", we.Code, we.Body.String())
	}
	mErr := decodeJSON(t, we.Body.String())
	errType, _ := mErr["__type"].(string)
	if errType != "ResourceAlreadyExistsException" {
		t.Errorf("CreateLogGroup duplicate: expected ResourceAlreadyExistsException, got %q", errType)
	}
}

// ---- Test 2: CreateLogStream + DescribeLogStreams ----

func TestLogs_CreateLogStream_DescribeLogStreams(t *testing.T) {
	handler := newLogsGateway(t)

	// Create group first.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/stream-test/group",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("setup CreateLogGroup: %d %s", wg.Code, wg.Body.String())
	}

	// CreateLogStream
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/stream-test/group",
		"logStreamName": "stream-alpha",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("CreateLogStream: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}

	// Create a second stream.
	ws2 := httptest.NewRecorder()
	handler.ServeHTTP(ws2, logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/stream-test/group",
		"logStreamName": "stream-beta",
	}))
	if ws2.Code != http.StatusOK {
		t.Fatalf("CreateLogStream second: expected 200, got %d\nbody: %s", ws2.Code, ws2.Body.String())
	}

	// DescribeLogStreams — all streams
	wds := httptest.NewRecorder()
	handler.ServeHTTP(wds, logsReq(t, "DescribeLogStreams", map[string]string{
		"logGroupName": "/stream-test/group",
	}))
	if wds.Code != http.StatusOK {
		t.Fatalf("DescribeLogStreams: expected 200, got %d\nbody: %s", wds.Code, wds.Body.String())
	}

	mds := decodeJSON(t, wds.Body.String())
	streams, ok := mds["logStreams"].([]any)
	if !ok {
		t.Fatalf("DescribeLogStreams: missing logStreams\nbody: %s", wds.Body.String())
	}
	if len(streams) != 2 {
		t.Errorf("DescribeLogStreams: expected 2 streams, got %d", len(streams))
	}

	// DescribeLogStreams — prefix filter
	wpf := httptest.NewRecorder()
	handler.ServeHTTP(wpf, logsReq(t, "DescribeLogStreams", map[string]string{
		"logGroupName":        "/stream-test/group",
		"logStreamNamePrefix": "stream-a",
	}))
	if wpf.Code != http.StatusOK {
		t.Fatalf("DescribeLogStreams prefix: expected 200, got %d\nbody: %s", wpf.Code, wpf.Body.String())
	}
	mpf := decodeJSON(t, wpf.Body.String())
	filtered, _ := mpf["logStreams"].([]any)
	if len(filtered) != 1 {
		t.Errorf("DescribeLogStreams prefix: expected 1 stream, got %d", len(filtered))
	}
	firstName := filtered[0].(map[string]any)["logStreamName"].(string)
	if firstName != "stream-alpha" {
		t.Errorf("DescribeLogStreams prefix: expected stream-alpha, got %q", firstName)
	}

	// CreateLogStream on non-existent group
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/does-not-exist",
		"logStreamName": "s",
	}))
	if wne.Code != http.StatusBadRequest {
		t.Fatalf("CreateLogStream no group: expected 400, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 3: PutLogEvents + GetLogEvents round-trip ----

func TestLogs_PutLogEvents_GetLogEvents(t *testing.T) {
	handler := newLogsGateway(t)

	// Setup group and stream.
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/events/group",
	}))
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/events/group",
		"logStreamName": "event-stream",
	}))

	ts := nowMs()
	events := []map[string]any{
		{"timestamp": ts, "message": "first log line"},
		{"timestamp": ts + 1, "message": "second log line"},
		{"timestamp": ts + 2, "message": "third log line"},
	}

	// PutLogEvents
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, logsReq(t, "PutLogEvents", map[string]any{
		"logGroupName":  "/events/group",
		"logStreamName": "event-stream",
		"logEvents":     events,
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutLogEvents: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mp := decodeJSON(t, wp.Body.String())
	nextToken, _ := mp["nextSequenceToken"].(string)
	if nextToken == "" {
		t.Error("PutLogEvents: missing nextSequenceToken in response")
	}

	// GetLogEvents
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, logsReq(t, "GetLogEvents", map[string]any{
		"logGroupName":  "/events/group",
		"logStreamName": "event-stream",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetLogEvents: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())
	evList, ok := mg["events"].([]any)
	if !ok {
		t.Fatalf("GetLogEvents: missing events\nbody: %s", wg.Body.String())
	}
	if len(evList) != 3 {
		t.Errorf("GetLogEvents: expected 3 events, got %d", len(evList))
	}

	// Verify event fields.
	first := evList[0].(map[string]any)
	if first["message"].(string) != "first log line" {
		t.Errorf("GetLogEvents: first message = %q, want %q", first["message"], "first log line")
	}
	if first["ingestionTime"] == nil {
		t.Error("GetLogEvents: missing ingestionTime on event")
	}

	// Verify pagination tokens are present.
	if mg["nextForwardToken"] == nil {
		t.Error("GetLogEvents: missing nextForwardToken")
	}
	if mg["nextBackwardToken"] == nil {
		t.Error("GetLogEvents: missing nextBackwardToken")
	}

	// GetLogEvents with time filter — only middle event
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, logsReq(t, "GetLogEvents", map[string]any{
		"logGroupName":  "/events/group",
		"logStreamName": "event-stream",
		"startTime":     ts + 1,
		"endTime":       ts + 1,
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("GetLogEvents filtered: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	mf := decodeJSON(t, wf.Body.String())
	filteredEvs, _ := mf["events"].([]any)
	if len(filteredEvs) != 1 {
		t.Errorf("GetLogEvents filtered: expected 1 event, got %d", len(filteredEvs))
	}

	// GetLogEvents with limit
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, logsReq(t, "GetLogEvents", map[string]any{
		"logGroupName":  "/events/group",
		"logStreamName": "event-stream",
		"limit":         2,
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("GetLogEvents limit: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	limitedEvs, _ := ml["events"].([]any)
	if len(limitedEvs) != 2 {
		t.Errorf("GetLogEvents limit: expected 2 events, got %d", len(limitedEvs))
	}
}

// ---- Test 4: FilterLogEvents with substring pattern ----

func TestLogs_FilterLogEvents(t *testing.T) {
	handler := newLogsGateway(t)

	// Setup
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/filter/group",
	}))
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/filter/group",
		"logStreamName": "filter-stream-1",
	}))
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/filter/group",
		"logStreamName": "filter-stream-2",
	}))

	ts := nowMs()

	// Put events into stream 1
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "PutLogEvents", map[string]any{
		"logGroupName":  "/filter/group",
		"logStreamName": "filter-stream-1",
		"logEvents": []map[string]any{
			{"timestamp": ts, "message": "ERROR: disk full"},
			{"timestamp": ts + 1, "message": "INFO: service started"},
		},
	}))

	// Put events into stream 2
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "PutLogEvents", map[string]any{
		"logGroupName":  "/filter/group",
		"logStreamName": "filter-stream-2",
		"logEvents": []map[string]any{
			{"timestamp": ts + 2, "message": "ERROR: out of memory"},
			{"timestamp": ts + 3, "message": "DEBUG: checkpoint"},
		},
	}))

	// FilterLogEvents — match "ERROR" across all streams
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, logsReq(t, "FilterLogEvents", map[string]any{
		"logGroupName":  "/filter/group",
		"filterPattern": "ERROR",
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("FilterLogEvents: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}

	mf := decodeJSON(t, wf.Body.String())
	evs, ok := mf["events"].([]any)
	if !ok {
		t.Fatalf("FilterLogEvents: missing events\nbody: %s", wf.Body.String())
	}
	if len(evs) != 2 {
		t.Errorf("FilterLogEvents ERROR: expected 2 events, got %d", len(evs))
	}
	for _, e := range evs {
		msg := e.(map[string]any)["message"].(string)
		if !strings.Contains(msg, "ERROR") {
			t.Errorf("FilterLogEvents: unexpected event message %q (expected ERROR match)", msg)
		}
	}

	// FilterLogEvents — match nothing
	wn := httptest.NewRecorder()
	handler.ServeHTTP(wn, logsReq(t, "FilterLogEvents", map[string]any{
		"logGroupName":  "/filter/group",
		"filterPattern": "CRITICAL",
	}))
	if wn.Code != http.StatusOK {
		t.Fatalf("FilterLogEvents no match: expected 200, got %d\nbody: %s", wn.Code, wn.Body.String())
	}
	mn := decodeJSON(t, wn.Body.String())
	noEvs, _ := mn["events"].([]any)
	if len(noEvs) != 0 {
		t.Errorf("FilterLogEvents no match: expected 0 events, got %d", len(noEvs))
	}

	// FilterLogEvents — specific stream only
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, logsReq(t, "FilterLogEvents", map[string]any{
		"logGroupName":   "/filter/group",
		"filterPattern":  "ERROR",
		"logStreamNames": []string{"filter-stream-1"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("FilterLogEvents stream filter: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	ms := decodeJSON(t, ws.Body.String())
	streamEvs, _ := ms["events"].([]any)
	if len(streamEvs) != 1 {
		t.Errorf("FilterLogEvents stream filter: expected 1 event, got %d", len(streamEvs))
	}

	// FilterLogEvents — no pattern (returns all events)
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, logsReq(t, "FilterLogEvents", map[string]any{
		"logGroupName": "/filter/group",
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("FilterLogEvents all: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}
	ma := decodeJSON(t, wa.Body.String())
	allEvs, _ := ma["events"].([]any)
	if len(allEvs) != 4 {
		t.Errorf("FilterLogEvents all: expected 4 events, got %d", len(allEvs))
	}
}

// ---- Test 5: DeleteLogStream + DeleteLogGroup ----

func TestLogs_DeleteStreamAndGroup(t *testing.T) {
	handler := newLogsGateway(t)

	// Setup group + stream + events
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/delete-test/group",
	}))
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/delete-test/group",
		"logStreamName": "to-delete",
	}))
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "PutLogEvents", map[string]any{
		"logGroupName":  "/delete-test/group",
		"logStreamName": "to-delete",
		"logEvents": []map[string]any{
			{"timestamp": nowMs(), "message": "some event"},
		},
	}))

	// DeleteLogStream
	wds := httptest.NewRecorder()
	handler.ServeHTTP(wds, logsReq(t, "DeleteLogStream", map[string]string{
		"logGroupName":  "/delete-test/group",
		"logStreamName": "to-delete",
	}))
	if wds.Code != http.StatusOK {
		t.Fatalf("DeleteLogStream: expected 200, got %d\nbody: %s", wds.Code, wds.Body.String())
	}

	// Stream should no longer appear in DescribeLogStreams
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, logsReq(t, "DescribeLogStreams", map[string]string{
		"logGroupName": "/delete-test/group",
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeLogStreams after delete: %d %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	remainingStreams, _ := mdesc["logStreams"].([]any)
	if len(remainingStreams) != 0 {
		t.Errorf("DeleteLogStream: stream still appears in DescribeLogStreams")
	}

	// DeleteLogStream — not found
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, logsReq(t, "DeleteLogStream", map[string]string{
		"logGroupName":  "/delete-test/group",
		"logStreamName": "to-delete",
	}))
	if wne.Code != http.StatusBadRequest {
		t.Fatalf("DeleteLogStream not found: expected 400, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
	mne := decodeJSON(t, wne.Body.String())
	errType, _ := mne["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("DeleteLogStream not found: expected ResourceNotFoundException, got %q", errType)
	}

	// DeleteLogGroup
	wdg := httptest.NewRecorder()
	handler.ServeHTTP(wdg, logsReq(t, "DeleteLogGroup", map[string]any{
		"logGroupName": "/delete-test/group",
	}))
	if wdg.Code != http.StatusOK {
		t.Fatalf("DeleteLogGroup: expected 200, got %d\nbody: %s", wdg.Code, wdg.Body.String())
	}

	// Group should not appear in DescribeLogGroups
	wdg2 := httptest.NewRecorder()
	handler.ServeHTTP(wdg2, logsReq(t, "DescribeLogGroups", map[string]string{
		"logGroupNamePrefix": "/delete-test/",
	}))
	if wdg2.Code != http.StatusOK {
		t.Fatalf("DescribeLogGroups after delete: %d %s", wdg2.Code, wdg2.Body.String())
	}
	mdg2 := decodeJSON(t, wdg2.Body.String())
	remainingGroups, _ := mdg2["logGroups"].([]any)
	if len(remainingGroups) != 0 {
		t.Errorf("DeleteLogGroup: group still appears in DescribeLogGroups")
	}

	// DeleteLogGroup — not found
	wne2 := httptest.NewRecorder()
	handler.ServeHTTP(wne2, logsReq(t, "DeleteLogGroup", map[string]any{
		"logGroupName": "/delete-test/group",
	}))
	if wne2.Code != http.StatusBadRequest {
		t.Fatalf("DeleteLogGroup not found: expected 400, got %d\nbody: %s", wne2.Code, wne2.Body.String())
	}
}

// ---- Test 6: PutRetentionPolicy + DeleteRetentionPolicy ----

func TestLogs_RetentionPolicy(t *testing.T) {
	handler := newLogsGateway(t)

	// Create group
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/retention/group",
	}))

	// PutRetentionPolicy
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, logsReq(t, "PutRetentionPolicy", map[string]any{
		"logGroupName":    "/retention/group",
		"retentionInDays": 30,
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutRetentionPolicy: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	// Verify retention in DescribeLogGroups
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, logsReq(t, "DescribeLogGroups", map[string]string{
		"logGroupNamePrefix": "/retention/",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeLogGroups: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	md := decodeJSON(t, wd.Body.String())
	groups, _ := md["logGroups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("DescribeLogGroups: expected 1 group, got %d", len(groups))
	}
	retentionVal := groups[0].(map[string]any)["retentionInDays"]
	// JSON numbers decode as float64
	if retention, ok := retentionVal.(float64); !ok || int(retention) != 30 {
		t.Errorf("PutRetentionPolicy: expected retentionInDays=30, got %v", retentionVal)
	}

	// PutRetentionPolicy — invalid value
	winv := httptest.NewRecorder()
	handler.ServeHTTP(winv, logsReq(t, "PutRetentionPolicy", map[string]any{
		"logGroupName":    "/retention/group",
		"retentionInDays": 0,
	}))
	if winv.Code != http.StatusBadRequest {
		t.Fatalf("PutRetentionPolicy invalid: expected 400, got %d\nbody: %s", winv.Code, winv.Body.String())
	}

	// DeleteRetentionPolicy
	wdr := httptest.NewRecorder()
	handler.ServeHTTP(wdr, logsReq(t, "DeleteRetentionPolicy", map[string]any{
		"logGroupName": "/retention/group",
	}))
	if wdr.Code != http.StatusOK {
		t.Fatalf("DeleteRetentionPolicy: expected 200, got %d\nbody: %s", wdr.Code, wdr.Body.String())
	}

	// Verify retention removed
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, logsReq(t, "DescribeLogGroups", map[string]string{
		"logGroupNamePrefix": "/retention/",
	}))
	md2 := decodeJSON(t, wd2.Body.String())
	groups2, _ := md2["logGroups"].([]any)
	if len(groups2) != 1 {
		t.Fatalf("DescribeLogGroups after delete retention: expected 1 group, got %d", len(groups2))
	}
	// retentionInDays is omitempty — should be absent or 0 after deletion
	retAfterDelete := groups2[0].(map[string]any)["retentionInDays"]
	if retAfterDelete != nil {
		if r, ok := retAfterDelete.(float64); ok && int(r) != 0 {
			t.Errorf("DeleteRetentionPolicy: expected retentionInDays to be 0 or absent, got %v", r)
		}
	}
}

// ---- Test 7: TagLogGroup / UntagLogGroup / ListTagsLogGroup ----

func TestLogs_TagOperations(t *testing.T) {
	handler := newLogsGateway(t)

	// Create group with initial tags
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/tags/group",
		"tags": map[string]string{
			"Env": "production",
		},
	}))

	// ListTagsLogGroup — should have initial tag
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, logsReq(t, "ListTagsLogGroup", map[string]string{
		"logGroupName": "/tags/group",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsLogGroup: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	tags, ok := ml["tags"].(map[string]any)
	if !ok {
		t.Fatalf("ListTagsLogGroup: missing tags\nbody: %s", wl.Body.String())
	}
	if tags["Env"] != "production" {
		t.Errorf("ListTagsLogGroup: expected Env=production, got %v", tags["Env"])
	}

	// TagLogGroup — add more tags
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, logsReq(t, "TagLogGroup", map[string]any{
		"logGroupName": "/tags/group",
		"tags": map[string]string{
			"Team":    "platform",
			"Version": "v2",
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagLogGroup: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsLogGroup — should have 3 tags now
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, logsReq(t, "ListTagsLogGroup", map[string]string{
		"logGroupName": "/tags/group",
	}))
	ml2 := decodeJSON(t, wl2.Body.String())
	tags2 := ml2["tags"].(map[string]any)
	if len(tags2) != 3 {
		t.Errorf("TagLogGroup: expected 3 tags, got %d", len(tags2))
	}
	if tags2["Team"] != "platform" {
		t.Errorf("TagLogGroup: expected Team=platform, got %v", tags2["Team"])
	}

	// UntagLogGroup — remove Version
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, logsReq(t, "UntagLogGroup", map[string]any{
		"logGroupName": "/tags/group",
		"tags":         []string{"Version"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagLogGroup: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// ListTagsLogGroup — should have 2 tags
	wl3 := httptest.NewRecorder()
	handler.ServeHTTP(wl3, logsReq(t, "ListTagsLogGroup", map[string]string{
		"logGroupName": "/tags/group",
	}))
	ml3 := decodeJSON(t, wl3.Body.String())
	tags3 := ml3["tags"].(map[string]any)
	if len(tags3) != 2 {
		t.Errorf("UntagLogGroup: expected 2 tags, got %d", len(tags3))
	}
	if _, found := tags3["Version"]; found {
		t.Error("UntagLogGroup: Version tag should have been removed")
	}
	if tags3["Env"] != "production" {
		t.Errorf("UntagLogGroup: Env tag should still be present")
	}
}

// ---- Test 8: CreateLogStream duplicate returns ResourceAlreadyExistsException ----

func TestLogs_CreateLogStream_AlreadyExists(t *testing.T) {
	handler := newLogsGateway(t)

	// Setup group
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/dup-stream/group",
	}))

	// Create stream
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/dup-stream/group",
		"logStreamName": "my-stream",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("CreateLogStream first: %d %s", ws.Code, ws.Body.String())
	}

	// Duplicate
	ws2 := httptest.NewRecorder()
	handler.ServeHTTP(ws2, logsReq(t, "CreateLogStream", map[string]string{
		"logGroupName":  "/dup-stream/group",
		"logStreamName": "my-stream",
	}))
	if ws2.Code != http.StatusBadRequest {
		t.Fatalf("CreateLogStream duplicate: expected 400, got %d\nbody: %s", ws2.Code, ws2.Body.String())
	}
	errBody := decodeJSON(t, ws2.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceAlreadyExistsException" {
		t.Errorf("CreateLogStream duplicate: expected ResourceAlreadyExistsException, got %q", errType)
	}
}

// ---- Test 9: PutLogEvents ResourceNotFoundException on nonexistent stream ----

func TestLogs_PutLogEvents_ResourceNotFoundException(t *testing.T) {
	handler := newLogsGateway(t)

	// Create group only, no stream
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/put-events-err/group",
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "PutLogEvents", map[string]any{
		"logGroupName":  "/put-events-err/group",
		"logStreamName": "nonexistent-stream",
		"logEvents": []map[string]any{
			{"timestamp": nowMs(), "message": "test"},
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutLogEvents nonexistent stream: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("PutLogEvents nonexistent stream: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Test 10: GetLogEvents ResourceNotFoundException on nonexistent stream ----

func TestLogs_GetLogEvents_ResourceNotFoundException(t *testing.T) {
	handler := newLogsGateway(t)

	// Create group only, no stream
	handler.ServeHTTP(httptest.NewRecorder(), logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/get-events-err/group",
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "GetLogEvents", map[string]any{
		"logGroupName":  "/get-events-err/group",
		"logStreamName": "nonexistent-stream",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GetLogEvents nonexistent stream: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("GetLogEvents nonexistent stream: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Test 11: FilterLogEvents on nonexistent group returns ResourceNotFoundException ----

func TestLogs_FilterLogEvents_ResourceNotFoundException(t *testing.T) {
	handler := newLogsGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "FilterLogEvents", map[string]any{
		"logGroupName":  "/nonexistent/group",
		"filterPattern": "ERROR",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("FilterLogEvents nonexistent group: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("FilterLogEvents nonexistent group: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Test 12: DeleteLogGroup ResourceNotFoundException ----

func TestLogs_DeleteLogGroup_ResourceNotFoundException(t *testing.T) {
	handler := newLogsGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "DeleteLogGroup", map[string]any{
		"logGroupName": "/nonexistent/group",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteLogGroup nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("DeleteLogGroup nonexistent: expected ResourceNotFoundException, got %q", errType)
	}
}

// ---- Test 13: DescribeLogGroups empty ----

func TestLogs_DescribeLogGroups_Empty(t *testing.T) {
	handler := newLogsGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "DescribeLogGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeLogGroups empty: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	groups, _ := m["logGroups"].([]any)
	if len(groups) != 0 {
		t.Errorf("DescribeLogGroups empty: expected 0, got %d", len(groups))
	}
}

// ---- Test 14: Unknown action returns 400 ----

func TestLogs_UnknownAction(t *testing.T) {
	handler := newLogsGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, logsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	errType, _ := m["__type"].(string)
	if errType != "InvalidAction" {
		t.Errorf("unknown action: expected __type=InvalidAction, got %q", errType)
	}
}

// ---- Test 15: CreateLogGroup duplicate returns ResourceAlreadyExistsException ----

func TestLogs_CreateLogGroup_Duplicate(t *testing.T) {
	handler := newLogsGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/dup/group",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateLogGroup first: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, logsReq(t, "CreateLogGroup", map[string]any{
		"logGroupName": "/dup/group",
	}))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateLogGroup duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
	m := decodeJSON(t, wc2.Body.String())
	errType, _ := m["__type"].(string)
	if !strings.Contains(errType, "AlreadyExists") && errType != "ResourceAlreadyExistsException" {
		t.Errorf("CreateLogGroup duplicate: expected AlreadyExists error, got %q", errType)
	}
}
