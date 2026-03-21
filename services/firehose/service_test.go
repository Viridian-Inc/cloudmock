package firehose_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	firehosesvc "github.com/neureaux/cloudmock/services/firehose"
)

// newFirehoseGateway builds a full gateway stack with the Firehose service registered and IAM disabled.
func newFirehoseGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(firehosesvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// firehoseReq builds a JSON POST request targeting the Firehose service via X-Amz-Target.
func firehoseReq(t *testing.T, action string, body interface{}) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("firehoseReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Firehose_20150804."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/firehose/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// ---- Test 1: CreateDeliveryStream + DescribeDeliveryStream + ListDeliveryStreams ----

func TestFirehose_CreateDescribeList(t *testing.T) {
	handler := newFirehoseGateway(t)

	// CreateDeliveryStream
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, firehoseReq(t, "CreateDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "my-stream",
		"DeliveryStreamType": "DirectPut",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::my-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/firehose-role",
			"Prefix":    "data/",
			"BufferingHints": map[string]interface{}{
				"IntervalInSeconds": 60,
				"SizeInMBs":         5,
			},
		},
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream: expected 200, got %d\nbody: %s", wCreate.Code, wCreate.Body.String())
	}

	mCreate := decodeJSON(t, wCreate.Body.String())
	arn, _ := mCreate["DeliveryStreamARN"].(string)
	if arn == "" {
		t.Fatal("CreateDeliveryStream: DeliveryStreamARN is empty")
	}
	if !contains(arn, "my-stream") {
		t.Errorf("CreateDeliveryStream: ARN %q does not contain stream name", arn)
	}

	// DescribeDeliveryStream
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, firehoseReq(t, "DescribeDeliveryStream", map[string]string{
		"DeliveryStreamName": "my-stream",
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeDeliveryStream: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	desc, ok := mDesc["DeliveryStreamDescription"].(map[string]interface{})
	if !ok {
		t.Fatalf("DescribeDeliveryStream: missing DeliveryStreamDescription\nbody: %s", wDesc.Body.String())
	}
	if desc["DeliveryStreamName"] != "my-stream" {
		t.Errorf("DescribeDeliveryStream: expected DeliveryStreamName=my-stream, got %q", desc["DeliveryStreamName"])
	}
	if desc["DeliveryStreamStatus"] != "ACTIVE" {
		t.Errorf("DescribeDeliveryStream: expected DeliveryStreamStatus=ACTIVE, got %q", desc["DeliveryStreamStatus"])
	}
	if desc["DeliveryStreamARN"] != arn {
		t.Errorf("DescribeDeliveryStream: ARN mismatch: got %q, want %q", desc["DeliveryStreamARN"], arn)
	}
	dests, _ := desc["Destinations"].([]interface{})
	if len(dests) != 1 {
		t.Errorf("DescribeDeliveryStream: expected 1 destination, got %d", len(dests))
	} else {
		d := dests[0].(map[string]interface{})
		if d["DestinationId"] == "" {
			t.Error("DescribeDeliveryStream: DestinationId is empty")
		}
		s3Desc, _ := d["S3DestinationDescription"].(map[string]interface{})
		if s3Desc["BucketARN"] != "arn:aws:s3:::my-bucket" {
			t.Errorf("DescribeDeliveryStream: BucketARN mismatch: %q", s3Desc["BucketARN"])
		}
	}

	// ListDeliveryStreams
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, firehoseReq(t, "ListDeliveryStreams", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("ListDeliveryStreams: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}

	mList := decodeJSON(t, wList.Body.String())
	streamNames, _ := mList["DeliveryStreamNames"].([]interface{})
	found := false
	for _, n := range streamNames {
		if n.(string) == "my-stream" {
			found = true
		}
	}
	if !found {
		t.Errorf("ListDeliveryStreams: my-stream not found in %v", streamNames)
	}
}

// ---- Test 2: PutRecord ----

func TestFirehose_PutRecord(t *testing.T) {
	handler := newFirehoseGateway(t)

	// CreateDeliveryStream
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, firehoseReq(t, "CreateDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "put-stream",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::put-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/r",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream: %d %s", wc.Code, wc.Body.String())
	}

	// PutRecord
	payload := base64.StdEncoding.EncodeToString([]byte("hello firehose"))
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, firehoseReq(t, "PutRecord", map[string]interface{}{
		"DeliveryStreamName": "put-stream",
		"Record": map[string]string{
			"Data": payload,
		},
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutRecord: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mPut := decodeJSON(t, wp.Body.String())
	recordId, _ := mPut["RecordId"].(string)
	if recordId == "" {
		t.Fatal("PutRecord: RecordId is empty")
	}
}

// ---- Test 3: PutRecordBatch ----

func TestFirehose_PutRecordBatch(t *testing.T) {
	handler := newFirehoseGateway(t)

	// CreateDeliveryStream
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, firehoseReq(t, "CreateDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "batch-stream",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::batch-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/r",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream: %d %s", wc.Code, wc.Body.String())
	}

	records := []map[string]string{
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-A"))},
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-B"))},
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-C"))},
	}

	wb := httptest.NewRecorder()
	handler.ServeHTTP(wb, firehoseReq(t, "PutRecordBatch", map[string]interface{}{
		"DeliveryStreamName": "batch-stream",
		"Records":            records,
	}))
	if wb.Code != http.StatusOK {
		t.Fatalf("PutRecordBatch: expected 200, got %d\nbody: %s", wb.Code, wb.Body.String())
	}

	mBatch := decodeJSON(t, wb.Body.String())
	failedCount, _ := mBatch["FailedPutCount"].(float64)
	if failedCount != 0 {
		t.Errorf("PutRecordBatch: expected FailedPutCount=0, got %v", failedCount)
	}

	responses, _ := mBatch["RequestResponses"].([]interface{})
	if len(responses) != 3 {
		t.Fatalf("PutRecordBatch: expected 3 responses, got %d", len(responses))
	}
	for i, r := range responses {
		rec := r.(map[string]interface{})
		if rec["RecordId"] == "" {
			t.Errorf("PutRecordBatch: response %d missing RecordId", i)
		}
	}
}

// ---- Test 4: DeleteDeliveryStream ----

func TestFirehose_DeleteDeliveryStream(t *testing.T) {
	handler := newFirehoseGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, firehoseReq(t, "CreateDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "delete-stream",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::del-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/r",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream: %d %s", wc.Code, wc.Body.String())
	}

	// Delete
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, firehoseReq(t, "DeleteDeliveryStream", map[string]string{
		"DeliveryStreamName": "delete-stream",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteDeliveryStream: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Describe after delete should return 400
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, firehoseReq(t, "DescribeDeliveryStream", map[string]string{
		"DeliveryStreamName": "delete-stream",
	}))
	if wdesc.Code != http.StatusBadRequest {
		t.Fatalf("DescribeDeliveryStream after delete: expected 400, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}

	// Delete again should return 400
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, firehoseReq(t, "DeleteDeliveryStream", map[string]string{
		"DeliveryStreamName": "delete-stream",
	}))
	if wd2.Code != http.StatusBadRequest {
		t.Fatalf("DeleteDeliveryStream (second): expected 400, got %d\nbody: %s", wd2.Code, wd2.Body.String())
	}
}

// ---- Test 5: Tags (TagDeliveryStream / UntagDeliveryStream / ListTagsForDeliveryStream) ----

func TestFirehose_Tags(t *testing.T) {
	handler := newFirehoseGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, firehoseReq(t, "CreateDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "tag-stream",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::tag-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/r",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream: %d %s", wc.Code, wc.Body.String())
	}

	// TagDeliveryStream
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, firehoseReq(t, "TagDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "tag-stream",
		"Tags": []map[string]string{
			{"Key": "env", "Value": "test"},
			{"Key": "team", "Value": "data"},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagDeliveryStream: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsForDeliveryStream
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, firehoseReq(t, "ListTagsForDeliveryStream", map[string]string{
		"DeliveryStreamName": "tag-stream",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForDeliveryStream: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	mList := decodeJSON(t, wl.Body.String())
	tags, _ := mList["Tags"].([]interface{})
	if len(tags) != 2 {
		t.Fatalf("ListTagsForDeliveryStream: expected 2 tags, got %d\nbody: %s", len(tags), wl.Body.String())
	}

	tagMap := make(map[string]string)
	for _, tg := range tags {
		entry := tg.(map[string]interface{})
		tagMap[entry["Key"].(string)] = entry["Value"].(string)
	}
	if tagMap["env"] != "test" {
		t.Errorf("ListTagsForDeliveryStream: expected env=test, got %q", tagMap["env"])
	}
	if tagMap["team"] != "data" {
		t.Errorf("ListTagsForDeliveryStream: expected team=data, got %q", tagMap["team"])
	}

	// UntagDeliveryStream
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, firehoseReq(t, "UntagDeliveryStream", map[string]interface{}{
		"DeliveryStreamName": "tag-stream",
		"TagKeys":            []string{"env"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagDeliveryStream: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// ListTagsForDeliveryStream after untag — should have only "team"
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, firehoseReq(t, "ListTagsForDeliveryStream", map[string]string{
		"DeliveryStreamName": "tag-stream",
	}))
	if wl2.Code != http.StatusOK {
		t.Fatalf("ListTagsForDeliveryStream (after untag): expected 200, got %d\nbody: %s", wl2.Code, wl2.Body.String())
	}

	mList2 := decodeJSON(t, wl2.Body.String())
	tags2, _ := mList2["Tags"].([]interface{})
	if len(tags2) != 1 {
		t.Fatalf("ListTagsForDeliveryStream after untag: expected 1 tag, got %d", len(tags2))
	}
	entry := tags2[0].(map[string]interface{})
	if entry["Key"].(string) != "team" {
		t.Errorf("ListTagsForDeliveryStream after untag: expected remaining key=team, got %q", entry["Key"])
	}
}

// ---- Additional: CreateDeliveryStream duplicate returns error ----

func TestFirehose_CreateDeliveryStream_Duplicate(t *testing.T) {
	handler := newFirehoseGateway(t)

	body := map[string]interface{}{
		"DeliveryStreamName": "dup-stream",
		"S3DestinationConfiguration": map[string]interface{}{
			"BucketARN": "arn:aws:s3:::dup-bucket",
			"RoleARN":   "arn:aws:iam::123456789012:role/r",
		},
	}

	wc1 := httptest.NewRecorder()
	handler.ServeHTTP(wc1, firehoseReq(t, "CreateDeliveryStream", body))
	if wc1.Code != http.StatusOK {
		t.Fatalf("CreateDeliveryStream first: %d %s", wc1.Code, wc1.Body.String())
	}

	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, firehoseReq(t, "CreateDeliveryStream", body))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateDeliveryStream duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
}

// ---- Additional: Unknown action ----

func TestFirehose_UnknownAction(t *testing.T) {
	handler := newFirehoseGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, firehoseReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// contains is a helper to check if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
