package kinesis_test

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
	kinesissvc "github.com/neureaux/cloudmock/services/kinesis"
)

// newKinesisGateway builds a full gateway stack with the Kinesis service registered and IAM disabled.
func newKinesisGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(kinesissvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// kinesisReq builds a JSON POST request targeting the Kinesis service via X-Amz-Target.
func kinesisReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("kinesisReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Kinesis_20131202."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/kinesis/aws4_request, SignedHeaders=host, Signature=abc123")
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

// ---- Test 1: CreateStream + DescribeStream + ListStreams ----

func TestKinesis_CreateDescribeList(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "my-stream",
		"ShardCount": 2,
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("CreateStream: expected 200, got %d\nbody: %s", wCreate.Code, wCreate.Body.String())
	}

	// DescribeStream
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, kinesisReq(t, "DescribeStream", map[string]string{
		"StreamName": "my-stream",
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeStream: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	desc, ok := mDesc["StreamDescription"].(map[string]any)
	if !ok {
		t.Fatalf("DescribeStream: missing StreamDescription\nbody: %s", wDesc.Body.String())
	}
	if desc["StreamName"] != "my-stream" {
		t.Errorf("DescribeStream: expected StreamName=my-stream, got %q", desc["StreamName"])
	}
	if desc["StreamStatus"] != "ACTIVE" {
		t.Errorf("DescribeStream: expected StreamStatus=ACTIVE, got %q", desc["StreamStatus"])
	}
	arn, _ := desc["StreamARN"].(string)
	if arn == "" {
		t.Error("DescribeStream: StreamARN is empty")
	}
	shards, _ := desc["Shards"].([]any)
	if len(shards) != 2 {
		t.Errorf("DescribeStream: expected 2 shards, got %d", len(shards))
	}

	// Verify each shard has a ShardId and HashKeyRange.
	for i, sh := range shards {
		s := sh.(map[string]any)
		if s["ShardId"] == "" {
			t.Errorf("DescribeStream: shard %d missing ShardId", i)
		}
		hkr, ok := s["HashKeyRange"].(map[string]any)
		if !ok {
			t.Errorf("DescribeStream: shard %d missing HashKeyRange", i)
		} else {
			if hkr["StartingHashKey"] == "" || hkr["EndingHashKey"] == "" {
				t.Errorf("DescribeStream: shard %d has empty HashKeyRange", i)
			}
		}
	}

	// ListStreams
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, kinesisReq(t, "ListStreams", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("ListStreams: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}

	mList := decodeJSON(t, wList.Body.String())
	streamNames, _ := mList["StreamNames"].([]any)
	found := false
	for _, n := range streamNames {
		if n.(string) == "my-stream" {
			found = true
		}
	}
	if !found {
		t.Errorf("ListStreams: my-stream not found in %v", streamNames)
	}
}

// ---- Test 2: PutRecord + GetShardIterator(TRIM_HORIZON) + GetRecords round-trip ----

func TestKinesis_PutRecord_GetRecords_RoundTrip(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream with 1 shard.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "records-stream",
		"ShardCount": 1,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateStream: %d %s", wc.Code, wc.Body.String())
	}

	// PutRecord.
	payload := []byte("hello kinesis")
	payloadB64 := base64.StdEncoding.EncodeToString(payload)

	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, kinesisReq(t, "PutRecord", map[string]string{
		"StreamName":   "records-stream",
		"Data":         payloadB64,
		"PartitionKey": "pk-1",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutRecord: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mPut := decodeJSON(t, wp.Body.String())
	shardID, _ := mPut["ShardId"].(string)
	seqNum, _ := mPut["SequenceNumber"].(string)
	if shardID == "" {
		t.Fatal("PutRecord: missing ShardId")
	}
	if seqNum == "" {
		t.Fatal("PutRecord: missing SequenceNumber")
	}

	// GetShardIterator with TRIM_HORIZON.
	wi := httptest.NewRecorder()
	handler.ServeHTTP(wi, kinesisReq(t, "GetShardIterator", map[string]string{
		"StreamName":        "records-stream",
		"ShardId":           shardID,
		"ShardIteratorType": "TRIM_HORIZON",
	}))
	if wi.Code != http.StatusOK {
		t.Fatalf("GetShardIterator: expected 200, got %d\nbody: %s", wi.Code, wi.Body.String())
	}

	mIter := decodeJSON(t, wi.Body.String())
	iterToken, _ := mIter["ShardIterator"].(string)
	if iterToken == "" {
		t.Fatal("GetShardIterator: missing ShardIterator")
	}

	// GetRecords.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": iterToken,
		"Limit":         100,
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetRecords: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mGet := decodeJSON(t, wg.Body.String())
	records, _ := mGet["Records"].([]any)
	if len(records) != 1 {
		t.Fatalf("GetRecords: expected 1 record, got %d\nbody: %s", len(records), wg.Body.String())
	}

	rec := records[0].(map[string]any)
	dataB64, _ := rec["Data"].(string)
	if dataB64 == "" {
		t.Fatal("GetRecords: record missing Data")
	}
	decoded, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		t.Fatalf("GetRecords: failed to decode Data: %v", err)
	}
	if string(decoded) != string(payload) {
		t.Errorf("GetRecords: expected data %q, got %q", string(payload), string(decoded))
	}

	partitionKey, _ := rec["PartitionKey"].(string)
	if partitionKey != "pk-1" {
		t.Errorf("GetRecords: expected PartitionKey=pk-1, got %q", partitionKey)
	}

	nextIter, _ := mGet["NextShardIterator"].(string)
	if nextIter == "" {
		t.Fatal("GetRecords: missing NextShardIterator")
	}

	// Call GetRecords again with NextShardIterator — should return no new records.
	wg2 := httptest.NewRecorder()
	handler.ServeHTTP(wg2, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": nextIter,
	}))
	if wg2.Code != http.StatusOK {
		t.Fatalf("GetRecords (next): expected 200, got %d\nbody: %s", wg2.Code, wg2.Body.String())
	}
	mGet2 := decodeJSON(t, wg2.Body.String())
	records2, _ := mGet2["Records"].([]any)
	if len(records2) != 0 {
		t.Errorf("GetRecords (next): expected 0 records after consuming all, got %d", len(records2))
	}
}

// ---- Test 3: PutRecords batch ----

func TestKinesis_PutRecords(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "batch-stream",
		"ShardCount": 2,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateStream: %d %s", wc.Code, wc.Body.String())
	}

	records := []map[string]string{
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-A")), "PartitionKey": "pk-a"},
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-B")), "PartitionKey": "pk-b"},
		{"Data": base64.StdEncoding.EncodeToString([]byte("record-C")), "PartitionKey": "pk-c"},
	}

	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, kinesisReq(t, "PutRecords", map[string]any{
		"StreamName": "batch-stream",
		"Records":    records,
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutRecords: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mPut := decodeJSON(t, wp.Body.String())
	failedCount, _ := mPut["FailedRecordCount"].(float64)
	if failedCount != 0 {
		t.Errorf("PutRecords: expected FailedRecordCount=0, got %v", failedCount)
	}

	results, _ := mPut["Records"].([]any)
	if len(results) != 3 {
		t.Fatalf("PutRecords: expected 3 result records, got %d", len(results))
	}

	for i, r := range results {
		rec := r.(map[string]any)
		if rec["ShardId"] == "" {
			t.Errorf("PutRecords: record %d missing ShardId", i)
		}
		if rec["SequenceNumber"] == "" {
			t.Errorf("PutRecords: record %d missing SequenceNumber", i)
		}
	}
}

// ---- Test 4: DeleteStream ----

func TestKinesis_DeleteStream(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "delete-stream",
		"ShardCount": 1,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateStream: %d %s", wc.Code, wc.Body.String())
	}

	// DeleteStream.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kinesisReq(t, "DeleteStream", map[string]string{
		"StreamName": "delete-stream",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteStream: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// DescribeStream after delete should return 400.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, kinesisReq(t, "DescribeStream", map[string]string{
		"StreamName": "delete-stream",
	}))
	if wdesc.Code != http.StatusBadRequest {
		t.Fatalf("DescribeStream after delete: expected 400, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}

	// DeleteStream again should return 400.
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, kinesisReq(t, "DeleteStream", map[string]string{
		"StreamName": "delete-stream",
	}))
	if wd2.Code != http.StatusBadRequest {
		t.Fatalf("DeleteStream (second): expected 400, got %d\nbody: %s", wd2.Code, wd2.Body.String())
	}
}

// ---- Test 5: Tags (AddTagsToStream / RemoveTagsFromStream / ListTagsForStream) ----

func TestKinesis_Tags(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "tag-stream",
		"ShardCount": 1,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateStream: %d %s", wc.Code, wc.Body.String())
	}

	// AddTagsToStream.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kinesisReq(t, "AddTagsToStream", map[string]any{
		"StreamName": "tag-stream",
		"Tags":       map[string]string{"env": "test", "team": "data"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("AddTagsToStream: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}

	// ListTagsForStream.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, kinesisReq(t, "ListTagsForStream", map[string]string{
		"StreamName": "tag-stream",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForStream: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	mList := decodeJSON(t, wl.Body.String())
	tags, _ := mList["Tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("ListTagsForStream: expected 2 tags, got %d\nbody: %s", len(tags), wl.Body.String())
	}

	tagMap := make(map[string]string)
	for _, tg := range tags {
		entry := tg.(map[string]any)
		tagMap[entry["Key"].(string)] = entry["Value"].(string)
	}
	if tagMap["env"] != "test" {
		t.Errorf("ListTagsForStream: expected env=test, got %q", tagMap["env"])
	}
	if tagMap["team"] != "data" {
		t.Errorf("ListTagsForStream: expected team=data, got %q", tagMap["team"])
	}

	// RemoveTagsFromStream.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, kinesisReq(t, "RemoveTagsFromStream", map[string]any{
		"StreamName": "tag-stream",
		"TagKeys":    []string{"env"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RemoveTagsFromStream: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}

	// ListTagsForStream after removal — should have only "team".
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, kinesisReq(t, "ListTagsForStream", map[string]string{
		"StreamName": "tag-stream",
	}))
	mList2 := decodeJSON(t, wl2.Body.String())
	tags2, _ := mList2["Tags"].([]any)
	if len(tags2) != 1 {
		t.Fatalf("ListTagsForStream after remove: expected 1 tag, got %d", len(tags2))
	}
	entry := tags2[0].(map[string]any)
	if entry["Key"].(string) != "team" {
		t.Errorf("ListTagsForStream after remove: expected remaining tag key=team, got %q", entry["Key"])
	}
}

// ---- Test 6: GetShardIterator types (LATEST vs TRIM_HORIZON) ----

func TestKinesis_ShardIteratorTypes(t *testing.T) {
	handler := newKinesisGateway(t)

	// CreateStream.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "iter-stream",
		"ShardCount": 1,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateStream: %d %s", wc.Code, wc.Body.String())
	}

	// PutRecord before obtaining LATEST iterator.
	wp1 := httptest.NewRecorder()
	handler.ServeHTTP(wp1, kinesisReq(t, "PutRecord", map[string]string{
		"StreamName":   "iter-stream",
		"Data":         base64.StdEncoding.EncodeToString([]byte("before-latest")),
		"PartitionKey": "pk1",
	}))
	if wp1.Code != http.StatusOK {
		t.Fatalf("PutRecord (before latest): %d %s", wp1.Code, wp1.Body.String())
	}
	mPut1 := decodeJSON(t, wp1.Body.String())
	shardID := mPut1["ShardId"].(string)

	// Get LATEST iterator (after the record we just wrote).
	wLatest := httptest.NewRecorder()
	handler.ServeHTTP(wLatest, kinesisReq(t, "GetShardIterator", map[string]string{
		"StreamName":        "iter-stream",
		"ShardId":           shardID,
		"ShardIteratorType": "LATEST",
	}))
	if wLatest.Code != http.StatusOK {
		t.Fatalf("GetShardIterator LATEST: %d %s", wLatest.Code, wLatest.Body.String())
	}
	mLatest := decodeJSON(t, wLatest.Body.String())
	latestToken := mLatest["ShardIterator"].(string)

	// PutRecord after obtaining LATEST iterator.
	wp2 := httptest.NewRecorder()
	handler.ServeHTTP(wp2, kinesisReq(t, "PutRecord", map[string]string{
		"StreamName":   "iter-stream",
		"Data":         base64.StdEncoding.EncodeToString([]byte("after-latest")),
		"PartitionKey": "pk1",
	}))
	if wp2.Code != http.StatusOK {
		t.Fatalf("PutRecord (after latest): %d %s", wp2.Code, wp2.Body.String())
	}

	// GetRecords with LATEST iterator should only see the second record.
	wgLatest := httptest.NewRecorder()
	handler.ServeHTTP(wgLatest, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": latestToken,
		"Limit":         100,
	}))
	if wgLatest.Code != http.StatusOK {
		t.Fatalf("GetRecords LATEST: %d %s", wgLatest.Code, wgLatest.Body.String())
	}
	mGLatest := decodeJSON(t, wgLatest.Body.String())
	rLatest, _ := mGLatest["Records"].([]any)
	if len(rLatest) != 1 {
		t.Fatalf("GetRecords LATEST: expected 1 record, got %d", len(rLatest))
	}
	latestData, _ := base64.StdEncoding.DecodeString(rLatest[0].(map[string]any)["Data"].(string))
	if string(latestData) != "after-latest" {
		t.Errorf("GetRecords LATEST: expected 'after-latest', got %q", string(latestData))
	}

	// Get TRIM_HORIZON iterator — should see both records.
	wTH := httptest.NewRecorder()
	handler.ServeHTTP(wTH, kinesisReq(t, "GetShardIterator", map[string]string{
		"StreamName":        "iter-stream",
		"ShardId":           shardID,
		"ShardIteratorType": "TRIM_HORIZON",
	}))
	if wTH.Code != http.StatusOK {
		t.Fatalf("GetShardIterator TRIM_HORIZON: %d %s", wTH.Code, wTH.Body.String())
	}
	mTH := decodeJSON(t, wTH.Body.String())
	thToken := mTH["ShardIterator"].(string)

	wgTH := httptest.NewRecorder()
	handler.ServeHTTP(wgTH, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": thToken,
		"Limit":         100,
	}))
	if wgTH.Code != http.StatusOK {
		t.Fatalf("GetRecords TRIM_HORIZON: %d %s", wgTH.Code, wgTH.Body.String())
	}
	mGTH := decodeJSON(t, wgTH.Body.String())
	rTH, _ := mGTH["Records"].([]any)
	if len(rTH) != 2 {
		t.Fatalf("GetRecords TRIM_HORIZON: expected 2 records, got %d", len(rTH))
	}

	// Verify AT_SEQUENCE_NUMBER iterator.
	seqNum1 := mPut1["SequenceNumber"].(string)
	wAT := httptest.NewRecorder()
	handler.ServeHTTP(wAT, kinesisReq(t, "GetShardIterator", map[string]string{
		"StreamName":             "iter-stream",
		"ShardId":                shardID,
		"ShardIteratorType":      "AT_SEQUENCE_NUMBER",
		"StartingSequenceNumber": seqNum1,
	}))
	if wAT.Code != http.StatusOK {
		t.Fatalf("GetShardIterator AT_SEQUENCE_NUMBER: %d %s", wAT.Code, wAT.Body.String())
	}
	mAT := decodeJSON(t, wAT.Body.String())
	atToken := mAT["ShardIterator"].(string)

	wgAT := httptest.NewRecorder()
	handler.ServeHTTP(wgAT, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": atToken,
		"Limit":         100,
	}))
	mGAT := decodeJSON(t, wgAT.Body.String())
	rAT, _ := mGAT["Records"].([]any)
	// AT_SEQUENCE_NUMBER should include the record at that sequence number onwards.
	if len(rAT) != 2 {
		t.Errorf("GetRecords AT_SEQUENCE_NUMBER: expected 2 records from first seq, got %d", len(rAT))
	}

	// Verify AFTER_SEQUENCE_NUMBER iterator (skips first record).
	wAfter := httptest.NewRecorder()
	handler.ServeHTTP(wAfter, kinesisReq(t, "GetShardIterator", map[string]string{
		"StreamName":             "iter-stream",
		"ShardId":                shardID,
		"ShardIteratorType":      "AFTER_SEQUENCE_NUMBER",
		"StartingSequenceNumber": seqNum1,
	}))
	if wAfter.Code != http.StatusOK {
		t.Fatalf("GetShardIterator AFTER_SEQUENCE_NUMBER: %d %s", wAfter.Code, wAfter.Body.String())
	}
	mAfter := decodeJSON(t, wAfter.Body.String())
	afterToken := mAfter["ShardIterator"].(string)

	wgAfter := httptest.NewRecorder()
	handler.ServeHTTP(wgAfter, kinesisReq(t, "GetRecords", map[string]any{
		"ShardIterator": afterToken,
		"Limit":         100,
	}))
	mGAfter := decodeJSON(t, wgAfter.Body.String())
	rAfter, _ := mGAfter["Records"].([]any)
	if len(rAfter) != 1 {
		t.Errorf("GetRecords AFTER_SEQUENCE_NUMBER: expected 1 record after first seq, got %d", len(rAfter))
	}
}

// ---- Additional: CreateStream duplicate returns error ----

func TestKinesis_CreateStream_Duplicate(t *testing.T) {
	handler := newKinesisGateway(t)

	wc1 := httptest.NewRecorder()
	handler.ServeHTTP(wc1, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "dup-stream",
		"ShardCount": 1,
	}))
	if wc1.Code != http.StatusOK {
		t.Fatalf("CreateStream first: %d %s", wc1.Code, wc1.Body.String())
	}

	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, kinesisReq(t, "CreateStream", map[string]any{
		"StreamName": "dup-stream",
		"ShardCount": 1,
	}))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateStream duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
}

// ---- Additional: Unknown action ----

func TestKinesis_UnknownAction(t *testing.T) {
	handler := newKinesisGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kinesisReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
