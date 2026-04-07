package compat

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

// CompatReport is the top-level compatibility report.
type CompatReport struct {
	GeneratedAt  string                    `json:"generated_at"`
	CloudMockVer string                    `json:"cloudmock_version"`
	TotalOps     int                       `json:"total_ops"`
	Passed       int                       `json:"passed"`
	Failed       int                       `json:"failed"`
	Skipped      int                       `json:"skipped"`
	CompatPct    float64                   `json:"compat_pct"`
	Services     map[string]*ServiceCompat `json:"services"`
}

// ServiceCompat holds per-service compatibility data.
type ServiceCompat struct {
	Service    string     `json:"service"`
	TotalOps   int        `json:"total_ops"`
	Passed     int        `json:"passed"`
	Failed     int        `json:"failed"`
	Skipped    int        `json:"skipped"`
	CompatPct  float64    `json:"compat_pct"`
	Operations []OpCompat `json:"operations"`
}

// OpCompat holds per-operation compatibility data.
type OpCompat struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // "pass", "fail", "skip"
	CMStatus    int    `json:"cm_status,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

// GenerateFromBenchmark creates a compat report from a benchmark results JSON file.
func GenerateFromBenchmark(benchPath string) (*CompatReport, error) {
	data, err := os.ReadFile(benchPath)
	if err != nil {
		return nil, fmt.Errorf("reading benchmark file %s: %w", benchPath, err)
	}

	var results harness.BenchmarkResults
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parsing benchmark JSON: %w", err)
	}

	// Find the cloudmock target — fall back to first target if not found.
	var targetResults *harness.TargetResults
	if tr, ok := results.Targets["cloudmock"]; ok {
		targetResults = tr
	} else {
		for _, tr := range results.Targets {
			targetResults = tr
			break
		}
	}
	if targetResults == nil {
		return nil, fmt.Errorf("no targets found in benchmark results")
	}

	report := &CompatReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		CloudMockVer: results.Meta.CloudMockVersion,
		Services:     make(map[string]*ServiceCompat),
	}

	// Sort service names for deterministic output.
	serviceNames := make([]string, 0, len(targetResults.Services))
	for name := range targetResults.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	for _, svcName := range serviceNames {
		svcResult := targetResults.Services[svcName]
		sc := &ServiceCompat{
			Service: svcName,
		}

		// Sort operation names for deterministic output.
		opNames := make([]string, 0, len(svcResult.Operations))
		for name := range svcResult.Operations {
			opNames = append(opNames, name)
		}
		sort.Strings(opNames)

		for _, opName := range opNames {
			opResult := svcResult.Operations[opName]
			oc := opCompatFromResult(opName, opResult)
			sc.Operations = append(sc.Operations, oc)

			sc.TotalOps++
			switch oc.Status {
			case "pass":
				sc.Passed++
			case "fail":
				sc.Failed++
			default:
				sc.Skipped++
			}
		}

		sc.CompatPct = compatPct(sc.Passed, sc.TotalOps-sc.Skipped)
		report.Services[svcName] = sc

		report.TotalOps += sc.TotalOps
		report.Passed += sc.Passed
		report.Failed += sc.Failed
		report.Skipped += sc.Skipped
	}

	report.CompatPct = compatPct(report.Passed, report.TotalOps-report.Skipped)

	return report, nil
}

// WriteReport serialises report as indented JSON and writes it to path.
func WriteReport(report *CompatReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing report to %s: %w", path, err)
	}
	return nil
}

// opCompatFromResult converts a harness.OperationResult to an OpCompat.
func opCompatFromResult(name string, op *harness.OperationResult) OpCompat {
	oc := OpCompat{Name: name}

	switch op.Correctness {
	case harness.GradePass:
		oc.Status = "pass"
	case harness.GradeFail:
		oc.Status = "fail"
		if len(op.Findings) > 0 {
			oc.ErrorDetail = op.Findings[0].Actual
		}
	default:
		// "partial", "unsupported", or anything else → skip
		oc.Status = "skip"
	}

	return oc
}

// compatPct returns the percentage of passed operations out of total non-skipped.
// Returns 100.0 when there are no non-skipped operations.
func compatPct(passed, nonSkipped int) float64 {
	if nonSkipped == 0 {
		return 100.0
	}
	pct := float64(passed) / float64(nonSkipped) * 100.0
	// Round to two decimal places.
	return math.Round(pct*100) / 100
}
