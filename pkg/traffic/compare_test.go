package traffic

import (
	"fmt"
	"testing"
)

func TestCompareJSON_Identical(t *testing.T) {
	a := []byte(`{"name":"test","count":42,"nested":{"key":"val"}}`)
	b := []byte(`{"name":"test","count":42,"nested":{"key":"val"}}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for identical JSON, got %v", diffs)
	}
}

func TestCompareJSON_DifferentValues(t *testing.T) {
	a := []byte(`{"name":"alice","count":10}`)
	b := []byte(`{"name":"bob","count":20}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d: %v", len(diffs), diffs)
	}

	foundName := false
	foundCount := false
	for _, d := range diffs {
		if contains(d, "name") {
			foundName = true
		}
		if contains(d, "count") {
			foundCount = true
		}
	}
	if !foundName {
		t.Error("expected diff for 'name'")
	}
	if !foundCount {
		t.Error("expected diff for 'count'")
	}
}

func TestCompareJSON_IgnorePaths(t *testing.T) {
	a := []byte(`{"RequestId":"abc-123","Status":"ok","Data":"hello"}`)
	b := []byte(`{"RequestId":"xyz-789","Status":"ok","Data":"hello"}`)

	diffs := CompareJSON(a, b, []string{"RequestId"})
	if len(diffs) != 0 {
		t.Errorf("expected no diffs after ignoring RequestId, got %v", diffs)
	}
}

func TestCompareJSON_MissingKeys(t *testing.T) {
	a := []byte(`{"name":"test","extra_field":"value"}`)
	b := []byte(`{"name":"test","new_field":"other"}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for missing/extra keys")
	}

	foundMissing := false
	foundExtra := false
	for _, d := range diffs {
		if contains(d, "missing key") {
			foundMissing = true
		}
		if contains(d, "extra key") {
			foundExtra = true
		}
	}
	if !foundMissing {
		t.Error("expected 'missing key' diff for extra_field")
	}
	if !foundExtra {
		t.Error("expected 'extra key' diff for new_field")
	}
}

func TestCompareJSON_NestedIgnorePaths(t *testing.T) {
	a := []byte(`{"ResponseMetadata":{"RequestId":"aaa"},"Data":"same"}`)
	b := []byte(`{"ResponseMetadata":{"RequestId":"bbb"},"Data":"same"}`)

	diffs := CompareJSON(a, b, []string{"ResponseMetadata"})
	if len(diffs) != 0 {
		t.Errorf("expected no diffs after ignoring ResponseMetadata, got %v", diffs)
	}
}

func TestCompareJSON_ArrayDifferences(t *testing.T) {
	a := []byte(`{"items":[1,2,3]}`)
	b := []byte(`{"items":[1,2]}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for different array lengths")
	}
}

func TestCompareJSON_NonJSON(t *testing.T) {
	a := []byte(`plain text`)
	b := []byte(`different text`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for non-JSON strings")
	}
}

func TestCompareRecordings_AllMatch(t *testing.T) {
	entries := []CapturedEntry{
		{ID: "1", Service: "s3", Action: "GetObject", StatusCode: 200, ResponseBody: `{"key":"val"}`},
		{ID: "2", Service: "sqs", Action: "SendMessage", StatusCode: 200, ResponseBody: `{"ok":true}`},
	}

	original := &Recording{Entries: entries}
	replay := &Recording{Entries: entries}

	report := CompareRecordings(original, replay, ComparisonConfig{})
	if report.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", report.TotalRequests)
	}
	if report.Matched != 2 {
		t.Errorf("expected 2 matched, got %d", report.Matched)
	}
	if report.CompatibilityPct != 100 {
		t.Errorf("expected 100%% compatibility, got %f", report.CompatibilityPct)
	}
	if len(report.Mismatches) != 0 {
		t.Errorf("expected no mismatches, got %d", len(report.Mismatches))
	}
}

func TestCompareRecordings_StatusMismatch(t *testing.T) {
	original := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "s3", Action: "GetObject", StatusCode: 200, ResponseBody: `{}`},
		},
	}
	replay := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "s3", Action: "GetObject", StatusCode: 404, ResponseBody: `{}`},
		},
	}

	report := CompareRecordings(original, replay, ComparisonConfig{})
	if report.Mismatched != 1 {
		t.Errorf("expected 1 mismatch, got %d", report.Mismatched)
	}
	if len(report.Mismatches) != 1 {
		t.Fatalf("expected 1 mismatch entry, got %d", len(report.Mismatches))
	}
	m := report.Mismatches[0]
	if m.Severity != "status" {
		t.Errorf("expected severity 'status', got %q", m.Severity)
	}
	if m.OriginalStatus != 200 {
		t.Errorf("expected original status 200, got %d", m.OriginalStatus)
	}
	if m.ReplayStatus != 404 {
		t.Errorf("expected replay status 404, got %d", m.ReplayStatus)
	}
}

func TestCompareRecordings_BodyMismatch(t *testing.T) {
	original := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "dynamodb", Action: "GetItem", StatusCode: 200, ResponseBody: `{"Item":{"id":"1","name":"alice"}}`},
		},
	}
	replay := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "dynamodb", Action: "GetItem", StatusCode: 200, ResponseBody: `{"Item":{"id":"1","name":"bob"}}`},
		},
	}

	report := CompareRecordings(original, replay, ComparisonConfig{StrictMode: true})
	if report.Mismatched != 1 {
		t.Errorf("expected 1 mismatch, got %d", report.Mismatched)
	}
	if len(report.Mismatches) != 1 {
		t.Fatalf("expected 1 mismatch entry, got %d", len(report.Mismatches))
	}
	m := report.Mismatches[0]
	if m.Severity != "data" {
		t.Errorf("expected severity 'data', got %q", m.Severity)
	}
}

func TestCompareRecordings_DifferentLengths(t *testing.T) {
	original := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", StatusCode: 200},
			{ID: "2", StatusCode: 200},
			{ID: "3", StatusCode: 200},
		},
	}
	replay := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", StatusCode: 200},
		},
	}

	report := CompareRecordings(original, replay, ComparisonConfig{})
	if report.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", report.TotalRequests)
	}
	if report.Errors != 2 {
		t.Errorf("expected 2 errors for missing entries, got %d", report.Errors)
	}
}

func TestCompareJSON_EmptyObjects(t *testing.T) {
	diffs := CompareJSON([]byte(`{}`), []byte(`{}`), nil)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for two empty objects, got %v", diffs)
	}
}

func TestCompareJSON_DeepNested(t *testing.T) {
	a := []byte(`{"level1":{"level2":{"level3":{"value":"original"}}}}`)
	b := []byte(`{"level1":{"level2":{"level3":{"value":"changed"}}}}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) == 0 {
		t.Fatal("expected diffs for deep nested difference")
	}
	found := false
	for _, d := range diffs {
		if contains(d, "level1.level2.level3.value") || contains(d, "value") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected diff mentioning nested path, got %v", diffs)
	}
}

func TestCompareJSON_NullValues(t *testing.T) {
	// null value vs missing key — these should differ.
	a := []byte(`{"key":null}`)
	b := []byte(`{}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) == 0 {
		t.Fatal("expected diffs: null value vs missing key")
	}
}

func TestCompareJSON_NumberTypes(t *testing.T) {
	// JSON numbers: Go's json.Unmarshal decodes all numbers as float64,
	// so 1 and 1.0 should decode identically.
	a := []byte(`{"count":1}`)
	b := []byte(`{"count":1.0}`)

	diffs := CompareJSON(a, b, nil)
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for int 1 vs float 1.0, got %v", diffs)
	}
}

func TestCompareRecordings_EmptyRecordings(t *testing.T) {
	original := &Recording{Entries: []CapturedEntry{}}
	replay := &Recording{Entries: []CapturedEntry{}}

	report := CompareRecordings(original, replay, ComparisonConfig{})
	if report.TotalRequests != 0 {
		t.Errorf("expected 0 total requests, got %d", report.TotalRequests)
	}
	if report.CompatibilityPct != 0 {
		// With 0 total requests the pct is undefined; the implementation returns 0.
		// Accept 0 or 100 as valid sentinel values.
		if report.CompatibilityPct != 100 {
			t.Errorf("unexpected compatibility pct for empty recordings: %f", report.CompatibilityPct)
		}
	}
	if len(report.Mismatches) != 0 {
		t.Errorf("expected no mismatches for empty recordings, got %v", report.Mismatches)
	}
}

func TestCompareRecordings_IgnoreRequestId(t *testing.T) {
	original := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "s3", Action: "GetObject", StatusCode: 200, ResponseBody: `{"RequestId":"aaa-111","Data":"hello"}`},
		},
	}
	replay := &Recording{
		Entries: []CapturedEntry{
			{ID: "1", Service: "s3", Action: "GetObject", StatusCode: 200, ResponseBody: `{"RequestId":"bbb-222","Data":"hello"}`},
		},
	}

	cfg := ComparisonConfig{
		StrictMode:  true,
		IgnorePaths: []string{"RequestId"},
	}
	report := CompareRecordings(original, replay, cfg)
	if report.Mismatched != 0 {
		t.Errorf("expected 0 mismatches after ignoring RequestId, got %d: %v", report.Mismatched, report.Mismatches)
	}
	if report.Matched != 1 {
		t.Errorf("expected 1 matched, got %d", report.Matched)
	}
}

func TestCompareRecordings_CompatibilityScore(t *testing.T) {
	// 8 matched, 2 mismatched → 80% compatibility.
	var origEntries, replayEntries []CapturedEntry
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("%d", i)
		origEntries = append(origEntries, CapturedEntry{ID: id, StatusCode: 200, ResponseBody: `{"ok":true}`})
		replayEntries = append(replayEntries, CapturedEntry{ID: id, StatusCode: 200, ResponseBody: `{"ok":true}`})
	}
	for i := 8; i < 10; i++ {
		id := fmt.Sprintf("%d", i)
		origEntries = append(origEntries, CapturedEntry{ID: id, StatusCode: 200})
		replayEntries = append(replayEntries, CapturedEntry{ID: id, StatusCode: 404})
	}

	original := &Recording{Entries: origEntries}
	replay := &Recording{Entries: replayEntries}

	report := CompareRecordings(original, replay, ComparisonConfig{})
	if report.TotalRequests != 10 {
		t.Errorf("expected 10 total requests, got %d", report.TotalRequests)
	}
	if report.Matched != 8 {
		t.Errorf("expected 8 matched, got %d", report.Matched)
	}
	if report.Mismatched != 2 {
		t.Errorf("expected 2 mismatched, got %d", report.Mismatched)
	}
	if report.CompatibilityPct != 80 {
		t.Errorf("expected 80%% compatibility, got %f", report.CompatibilityPct)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
