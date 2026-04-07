package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.md")

	results := &harness.BenchmarkResults{
		Meta: harness.Meta{Date: "2026-04-02", Platform: "darwin/arm64", Mode: "docker", Iterations: 100},
		Startup: map[string]*harness.StartupResult{
			"cloudmock_docker":  {MedianMs: 850},
			"localstack_docker": {MedianMs: 4200},
		},
		Resources: map[string]*harness.ResourceStats{
			"cloudmock_docker":  {PeakMemoryMB: 45, AvgMemoryMB: 32, PeakCPUPct: 12, AvgCPUPct: 5},
			"localstack_docker": {PeakMemoryMB: 380, AvgMemoryMB: 290, PeakCPUPct: 65, AvgCPUPct: 30},
		},
		Targets: map[string]*harness.TargetResults{
			"cloudmock_docker": {
				Target: "cloudmock", Mode: "docker",
				Services: map[string]*harness.ServiceResult{
					"s3": {Service: "s3", Tier: 1, Operations: map[string]*harness.OperationResult{
						"PutObject": {Name: "PutObject", ColdMs: 12.0, Warm: harness.LatencyStats{P50: 2.1, P95: 4.5, P99: 8.0}, Correctness: harness.GradePass},
					}},
				},
			},
			"localstack_docker": {
				Target: "localstack", Mode: "docker",
				Services: map[string]*harness.ServiceResult{
					"s3": {Service: "s3", Tier: 1, Operations: map[string]*harness.OperationResult{
						"PutObject": {Name: "PutObject", ColdMs: 85.0, Warm: harness.LatencyStats{P50: 15.3, P95: 42.0, P99: 78.0}, Correctness: harness.GradePass},
					}},
				},
			},
		},
	}

	err := WriteMarkdown(results, path)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	md := string(content)

	assert.Contains(t, md, "# Benchmark Results")
	assert.Contains(t, md, "Startup")
	assert.Contains(t, md, "cloudmock")
	assert.Contains(t, md, "localstack")
	assert.Contains(t, md, "s3")
	assert.Contains(t, md, "PutObject")
}
