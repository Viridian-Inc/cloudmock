package cloudwatchlogs_test

import (
	"encoding/json"
	"testing"

	logssvc "github.com/neureaux/cloudmock/services/cloudwatchlogs"
)

const (
	cwlTestAccount = "123456789012"
	cwlTestRegion  = "us-east-1"
)

func TestCWL_ExportState_Empty(t *testing.T) {
	svc := logssvc.New(cwlTestAccount, cwlTestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		LogGroups []any `json:"log_groups"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.LogGroups) != 0 {
		t.Errorf("expected empty log_groups, got %d", len(state.LogGroups))
	}
}

func TestCWL_ExportState_WithLogGroups(t *testing.T) {
	svc := logssvc.New(cwlTestAccount, cwlTestRegion)

	seed := json.RawMessage(`{"log_groups":[{"name":"/app/api","retention_days":30,"tags":{"env":"prod"}},{"name":"/app/worker","retention_days":7}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		LogGroups []struct {
			Name          string            `json:"name"`
			RetentionDays int               `json:"retention_days"`
			Tags          map[string]string `json:"tags"`
		} `json:"log_groups"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.LogGroups) != 2 {
		t.Fatalf("expected 2 log groups, got %d", len(state.LogGroups))
	}

	found := make(map[string]bool)
	for _, g := range state.LogGroups {
		found[g.Name] = true
	}
	for _, expected := range []string{"/app/api", "/app/worker"} {
		if !found[expected] {
			t.Errorf("log group %q not found in export", expected)
		}
	}
}

func TestCWL_ExportState_PreservesRetentionDays(t *testing.T) {
	svc := logssvc.New(cwlTestAccount, cwlTestRegion)

	seed := json.RawMessage(`{"log_groups":[{"name":"/audit","retention_days":90}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		LogGroups []struct {
			Name          string `json:"name"`
			RetentionDays int    `json:"retention_days"`
		} `json:"log_groups"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.LogGroups) == 0 {
		t.Fatal("expected log groups in export")
	}
	if state.LogGroups[0].RetentionDays != 90 {
		t.Errorf("expected retention_days=90, got %d", state.LogGroups[0].RetentionDays)
	}
}

func TestCWL_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := logssvc.New(cwlTestAccount, cwlTestRegion)

	if err := svc.ImportState(json.RawMessage(`{"log_groups":[]}`)); err != nil {
		t.Fatalf("ImportState with empty log_groups: %v", err)
	}
}

func TestCWL_RoundTrip_PreservesLogGroups(t *testing.T) {
	svc := logssvc.New(cwlTestAccount, cwlTestRegion)

	seed := json.RawMessage(`{"log_groups":[{"name":"/service/a","retention_days":14},{"name":"/service/b","retention_days":0}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := logssvc.New(cwlTestAccount, cwlTestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	raw2, err := svc2.ExportState()
	if err != nil {
		t.Fatalf("ExportState (svc2): %v", err)
	}

	var state struct {
		LogGroups []struct{ Name string `json:"name"` } `json:"log_groups"`
	}
	json.Unmarshal(raw2, &state)

	names := make(map[string]bool)
	for _, g := range state.LogGroups {
		names[g.Name] = true
	}
	for _, expected := range []string{"/service/a", "/service/b"} {
		if !names[expected] {
			t.Errorf("log group %q not restored", expected)
		}
	}
}
