package compat_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/neureaux/cloudmock/benchmarks/compat"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

// buildBenchmarkFile writes a minimal BenchmarkResults JSON to a temp file and
// returns the path. The caller is responsible for cleanup via t.TempDir().
func buildBenchmarkFile(t *testing.T, services map[string]*harness.ServiceResult) string {
	t.Helper()
	results := harness.BenchmarkResults{
		Meta: harness.Meta{
			Date:             "2026-04-01T00:00:00Z",
			CloudMockVersion: "v1.2.3",
		},
		Startup:   map[string]*harness.StartupResult{},
		Resources: map[string]*harness.ResourceStats{},
		Targets: map[string]*harness.TargetResults{
			"cloudmock": {
				Target:   "cloudmock",
				Mode:     "docker",
				Services: services,
			},
		},
	}
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal benchmark: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write benchmark file: %v", err)
	}
	return path
}

func makeService(ops map[string]harness.Grade) *harness.ServiceResult {
	operations := make(map[string]*harness.OperationResult, len(ops))
	for name, grade := range ops {
		op := &harness.OperationResult{
			Name:        name,
			Correctness: grade,
		}
		if grade == harness.GradeFail {
			op.Findings = []harness.Finding{
				{Field: "cold_run", Expected: "no error", Actual: "HTTP 501: Not Implemented", Grade: harness.GradeFail},
			}
		}
		operations[name] = op
	}
	return &harness.ServiceResult{
		Service:    "test-service",
		Tier:       1,
		Operations: operations,
	}
}

func TestGenerateReport_FromBenchmark(t *testing.T) {
	services := map[string]*harness.ServiceResult{
		"s3": {
			Service: "s3",
			Tier:    1,
			Operations: map[string]*harness.OperationResult{
				"PutObject": {Name: "PutObject", Correctness: harness.GradePass},
				"GetObject": {Name: "GetObject", Correctness: harness.GradePass},
			},
		},
		"sqs": {
			Service: "sqs",
			Tier:    1,
			Operations: map[string]*harness.OperationResult{
				"SendMessage":    {Name: "SendMessage", Correctness: harness.GradePass},
				"ReceiveMessage": {Name: "ReceiveMessage", Correctness: harness.GradeFail},
			},
		},
	}
	path := buildBenchmarkFile(t, services)

	report, err := compat.GenerateFromBenchmark(path)
	if err != nil {
		t.Fatalf("GenerateFromBenchmark: %v", err)
	}

	if len(report.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(report.Services))
	}

	s3, ok := report.Services["s3"]
	if !ok {
		t.Fatal("s3 service missing from report")
	}
	if s3.TotalOps != 2 {
		t.Errorf("s3: expected 2 ops, got %d", s3.TotalOps)
	}
	if s3.Passed != 2 {
		t.Errorf("s3: expected 2 passed, got %d", s3.Passed)
	}
	if len(s3.Operations) != 2 {
		t.Errorf("s3: expected 2 operations, got %d", len(s3.Operations))
	}

	if report.CloudMockVer != "v1.2.3" {
		t.Errorf("expected version v1.2.3, got %q", report.CloudMockVer)
	}
	if report.GeneratedAt == "" {
		t.Error("GeneratedAt should not be empty")
	}
}

func TestReport_PassOnCorrectness(t *testing.T) {
	svc := makeService(map[string]harness.Grade{
		"GetObject": harness.GradePass,
	})
	path := buildBenchmarkFile(t, map[string]*harness.ServiceResult{"s3": svc})

	report, err := compat.GenerateFromBenchmark(path)
	if err != nil {
		t.Fatalf("GenerateFromBenchmark: %v", err)
	}

	op := report.Services["s3"].Operations[0]
	if op.Status != "pass" {
		t.Errorf("correctness=pass should map to Status=pass, got %q", op.Status)
	}
}

func TestReport_FailOnCorrectness(t *testing.T) {
	svc := makeService(map[string]harness.Grade{
		"CreateBucket": harness.GradeFail,
	})
	path := buildBenchmarkFile(t, map[string]*harness.ServiceResult{"s3": svc})

	report, err := compat.GenerateFromBenchmark(path)
	if err != nil {
		t.Fatalf("GenerateFromBenchmark: %v", err)
	}

	op := report.Services["s3"].Operations[0]
	if op.Status != "fail" {
		t.Errorf("correctness=fail should map to Status=fail, got %q", op.Status)
	}
	if op.ErrorDetail == "" {
		t.Error("ErrorDetail should be populated for a fail result with findings")
	}
}

func TestReport_CompatPercentage(t *testing.T) {
	// 8 pass + 2 fail = 80%
	ops := map[string]harness.Grade{}
	for i := 0; i < 8; i++ {
		ops[string(rune('A'+i))] = harness.GradePass
	}
	ops["Y"] = harness.GradeFail
	ops["Z"] = harness.GradeFail

	svc := makeService(ops)
	path := buildBenchmarkFile(t, map[string]*harness.ServiceResult{"svc": svc})

	report, err := compat.GenerateFromBenchmark(path)
	if err != nil {
		t.Fatalf("GenerateFromBenchmark: %v", err)
	}

	if report.Passed != 8 {
		t.Errorf("expected 8 passed, got %d", report.Passed)
	}
	if report.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", report.Failed)
	}
	if report.CompatPct != 80.0 {
		t.Errorf("expected CompatPct=80.0, got %f", report.CompatPct)
	}
}

func TestWriteReport(t *testing.T) {
	svc := makeService(map[string]harness.Grade{
		"PutObject": harness.GradePass,
		"GetObject": harness.GradeFail,
	})
	benchPath := buildBenchmarkFile(t, map[string]*harness.ServiceResult{"s3": svc})

	report, err := compat.GenerateFromBenchmark(benchPath)
	if err != nil {
		t.Fatalf("GenerateFromBenchmark: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "compat.json")
	if err := compat.WriteReport(report, outPath); err != nil {
		t.Fatalf("WriteReport: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var readBack compat.CompatReport
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("unmarshalling output: %v", err)
	}

	if readBack.TotalOps != report.TotalOps {
		t.Errorf("TotalOps mismatch: want %d, got %d", report.TotalOps, readBack.TotalOps)
	}
	if readBack.CompatPct != report.CompatPct {
		t.Errorf("CompatPct mismatch: want %f, got %f", report.CompatPct, readBack.CompatPct)
	}
	if len(readBack.Services) != len(report.Services) {
		t.Errorf("Services count mismatch: want %d, got %d", len(report.Services), len(readBack.Services))
	}
}
