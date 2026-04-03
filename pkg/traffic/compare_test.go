package traffic

import (
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
