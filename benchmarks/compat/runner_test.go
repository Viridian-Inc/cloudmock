package compat_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/benchmarks/compat"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
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
	report.SourceSHA = "d73dcdebc17df4bbca11e47170337f685c7a1fa9"

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
	if readBack.SourceSHA != report.SourceSHA {
		t.Errorf("SourceSHA mismatch: want %q, got %q", report.SourceSHA, readBack.SourceSHA)
	}
}

func TestWriteReportWithoutSourceSHA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compat.json")
	if err := compat.WriteReport(&compat.CompatReport{}, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["source_sha"]; exists {
		t.Fatal("report without source metadata must omit source_sha")
	}
}

func TestCompareToBaseline(t *testing.T) {
	baseline := &compat.CompatReport{Services: map[string]*compat.ServiceCompat{
		"s3": {Operations: []compat.OpCompat{
			{Name: "Unchanged", Status: "pass"},
			{Name: "NewFailure", Status: "pass"},
			{Name: "NewSkip", Status: "pass"},
			{Name: "MissingOperation", Status: "pass"},
			{Name: "ExistingFailure", Status: "fail"},
			{Name: "ExistingSkip", Status: "skip"},
		}},
		"sqs": {Operations: []compat.OpCompat{{Name: "MissingService", Status: "pass"}}},
	}}
	report := &compat.CompatReport{Services: map[string]*compat.ServiceCompat{
		"s3": {Operations: []compat.OpCompat{
			{Name: "Unchanged", Status: "pass"},
			{Name: "NewFailure", Status: "fail"},
			{Name: "NewSkip", Status: "skip"},
			{Name: "ExistingFailure", Status: "fail"},
			{Name: "ExistingSkip", Status: "fail"},
		}},
	}}
	want := []compat.Regression{
		{Service: "s3", Operation: "MissingOperation", Status: "missing"},
		{Service: "s3", Operation: "NewFailure", Status: "fail"},
		{Service: "s3", Operation: "NewSkip", Status: "skip"},
		{Service: "sqs", Operation: "MissingService", Status: "missing"},
	}
	if got, err := compat.CompareToBaseline(report, baseline); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("regressions = %#v, want %#v; error=%v", got, want, err)
	}
	if got, err := compat.CompareToBaseline(baseline, baseline); err != nil || len(got) != 0 {
		t.Fatalf("unchanged baseline must pass despite existing failures: %#v; error=%v", got, err)
	}
}

func TestCompareToBaselineNewPassCannotMaskRegression(t *testing.T) {
	baseline := &compat.CompatReport{Passed: 1, Failed: 1, Services: map[string]*compat.ServiceCompat{
		"s3": {Operations: []compat.OpCompat{{Name: "GetObject", Status: "pass"}, {Name: "PutObject", Status: "fail"}}},
	}}
	report := &compat.CompatReport{Passed: 1, Failed: 1, Services: map[string]*compat.ServiceCompat{
		"s3": {Operations: []compat.OpCompat{{Name: "GetObject", Status: "fail"}, {Name: "PutObject", Status: "pass"}}},
	}}
	want := []compat.Regression{{Service: "s3", Operation: "GetObject", Status: "fail"}}
	if got, err := compat.CompareToBaseline(report, baseline); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("equal aggregate counts masked regression: %#v; error=%v", got, err)
	}
}

func TestCompareToBaselineRejectsNoPassingOperations(t *testing.T) {
	for _, operations := range [][]compat.OpCompat{
		nil,
		{{Name: "Unavailable", Status: "fail"}, {Name: "Unsupported", Status: "skip"}},
	} {
		baseline := &compat.CompatReport{Passed: 100, Services: map[string]*compat.ServiceCompat{
			"s3": {Operations: operations},
		}}
		// Operation evidence, rather than aggregate counters, determines whether
		// the baseline is capable of detecting a regression.
		if _, err := compat.CompareToBaseline(&compat.CompatReport{}, baseline); err == nil || !strings.Contains(err.Error(), "no passing operations") {
			t.Fatalf("baseline with no passes accepted: %v", err)
		}
	}
}

func TestReportCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "compat-report")
	if output, err := exec.Command("go", "build", "-o", binary, "./cmd").CombinedOutput(); err != nil {
		t.Fatalf("build report CLI: %v\n%s", err, output)
	}
	const sourceSHA = "d73dcdebc17df4bbca11e47170337f685c7a1fa9"
	for _, tc := range []struct {
		name       string
		baseline   string
		wantStatus int
		wantLog    string
	}{
		{name: "report only preserves failures", wantStatus: 0},
		{name: "existing failure is allowed", baseline: "fail", wantStatus: 0, wantLog: "No compatibility regressions"},
		{name: "regression fails after writing report", baseline: "pass", wantStatus: 1, wantLog: "s3.GetObject: pass -> fail"},
		{name: "baseline without passes fails", baseline: "no-pass", wantStatus: 1, wantLog: "baseline contains no passing operations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			benchPath := buildBenchmarkFile(t, map[string]*harness.ServiceResult{"s3": makeService(map[string]harness.Grade{
				"GetObject": harness.GradeFail,
				"PutObject": harness.GradePass,
			})})
			outputPath := filepath.Join(t.TempDir(), "compat.json")
			args := []string{"--input", benchPath, "--output", outputPath, "--source-sha", sourceSHA}
			if tc.baseline != "" {
				baseline := &compat.CompatReport{Services: map[string]*compat.ServiceCompat{
					"s3": {Operations: []compat.OpCompat{{Name: "GetObject", Status: tc.baseline}, {Name: "PutObject", Status: "pass"}}},
				}}
				if tc.baseline == "no-pass" {
					baseline.Services["s3"].Operations = []compat.OpCompat{{Name: "GetObject", Status: "fail"}, {Name: "PutObject", Status: "skip"}}
				}
				if err := compat.WriteReport(baseline, outputPath); err != nil {
					t.Fatal(err)
				}
				// Loading the baseline before replacing it must preserve the gate.
				args = append(args, "--baseline", outputPath)
			}
			output, err := exec.Command(binary, args...).CombinedOutput()
			status := 0
			if err != nil {
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("run report CLI: %v", err)
				}
				status = exit.ExitCode()
			}
			if status != tc.wantStatus {
				t.Fatalf("exit status %d, want %d:\n%s", status, tc.wantStatus, output)
			}
			if !strings.Contains(string(output), sourceSHA) || !strings.Contains(string(output), tc.wantLog) {
				t.Errorf("CLI did not log tested SHA and gate result:\n%s", output)
			}
			data, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatal(err)
			}
			var report compat.CompatReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.SourceSHA != sourceSHA || report.Failed != 1 || report.Passed != 1 {
				t.Fatalf("current report missing after CLI exit: %+v", report)
			}
		})
	}
}
