package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.json")

	results := &harness.BenchmarkResults{
		Meta: harness.Meta{Date: "2026-04-02", Platform: "darwin/arm64", Mode: "docker"},
		Startup: map[string]*harness.StartupResult{
			"cloudmock": {MedianMs: 850, Runs: []float64{800, 850, 900}},
		},
		Resources: map[string]*harness.ResourceStats{
			"cloudmock": {PeakMemoryMB: 45, AvgMemoryMB: 32},
		},
		Targets: map[string]*harness.TargetResults{
			"cloudmock_docker": {
				Target: "cloudmock", Mode: "docker",
				Services: map[string]*harness.ServiceResult{
					"s3": {
						Service: "s3", Tier: 1,
						Operations: map[string]*harness.OperationResult{
							"PutObject": {Name: "PutObject", ColdMs: 12.0, Correctness: harness.GradePass},
						},
					},
				},
			},
		},
	}

	err := WriteJSON(results, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed harness.BenchmarkResults
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "2026-04-02", parsed.Meta.Date)
	assert.Equal(t, 850.0, parsed.Startup["cloudmock"].MedianMs)
	assert.Equal(t, harness.GradePass, parsed.Targets["cloudmock_docker"].Services["s3"].Operations["PutObject"].Correctness)
}
