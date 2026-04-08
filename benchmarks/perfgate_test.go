//go:build perfgate

// Package benchmarks contains performance gate tests that run on every push.
// These tests start CloudMock in-process, run key operations, and fail if
// latency exceeds the thresholds defined in baseline.json.
//
// Run with: go test ./benchmarks/ -tags perfgate -timeout 120s -v
package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
	"github.com/Viridian-Inc/cloudmock/benchmarks/suites/stress"
	"github.com/Viridian-Inc/cloudmock/benchmarks/suites/tier1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type threshold struct {
	MaxP50Ms float64 `json:"max_p50_ms"`
}

type baseline struct {
	Thresholds map[string]map[string]threshold `json:"thresholds"`
}

func loadBaseline(t *testing.T) baseline {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "baseline.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read baseline.json")
	var b baseline
	require.NoError(t, json.Unmarshal(data, &b), "failed to parse baseline.json")
	return b
}

func runSuiteGate(t *testing.T, suite harness.Suite, endpoint string, b baseline) {
	t.Helper()
	thresholds, ok := b.Thresholds[suite.Name()]
	if !ok {
		t.Skipf("no thresholds defined for %s", suite.Name())
	}

	for _, op := range suite.Operations() {
		th, ok := thresholds[op.Name]
		if !ok {
			continue // no threshold for this op
		}

		t.Run(fmt.Sprintf("%s/%s", suite.Name(), op.Name), func(t *testing.T) {
			ctx := context.Background()

			// Each iteration gets its own Setup/Teardown to avoid resource conflicts
			// (e.g. S3 CreateBucket returns 409 if bucket already exists).
			runOnce := func() (float64, error) {
				if op.Setup != nil {
					if err := op.Setup(ctx, endpoint); err != nil {
						return 0, fmt.Errorf("setup: %w", err)
					}
				}
				start := time.Now()
				resp, err := op.Run(ctx, endpoint)
				elapsed := time.Since(start).Seconds() * 1000
				if op.Teardown != nil {
					op.Teardown(ctx, endpoint)
				}
				if err != nil {
					return elapsed, err
				}
				if op.Validate != nil {
					for _, f := range op.Validate(resp) {
						if f.Grade == harness.GradeFail {
							return elapsed, fmt.Errorf("correctness: %s expected=%s actual=%s", f.Field, f.Expected, f.Actual)
						}
					}
				}
				return elapsed, nil
			}

			// Warm up (3 runs)
			for i := 0; i < 3; i++ {
				if _, err := runOnce(); err != nil {
					t.Fatalf("warmup failed: %v", err)
				}
			}

			// Measure (10 runs)
			times := make([]float64, 10)
			for i := 0; i < 10; i++ {
				elapsed, err := runOnce()
				if err != nil {
					t.Fatalf("run %d failed: %v", i, err)
				}
				times[i] = elapsed
			}

			// Calculate p50
			sorted := make([]float64, len(times))
			copy(sorted, times)
			for i := range sorted {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[j] < sorted[i] {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
			p50 := sorted[len(sorted)/2]

			t.Logf("p50=%.2fms (threshold=%.2fms)", p50, th.MaxP50Ms)

			assert.LessOrEqual(t, p50, th.MaxP50Ms,
				"PERFORMANCE REGRESSION: %s p50=%.2fms exceeds threshold %.2fms",
				op.Name, p50, th.MaxP50Ms)
		})
	}
}

func TestPerfGate_DynamoDB(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewDynamoDBSuite(), endpoint, b)
}

func TestPerfGate_DynamoDB_Stress(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, stress.NewDynamoDBStressSuite(), endpoint, b)
}

func TestPerfGate_S3(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewS3Suite(), endpoint, b)
}

func TestPerfGate_SQS(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewSQSSuite(), endpoint, b)
}

func TestPerfGate_SNS(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewSNSSuite(), endpoint, b)
}

func TestPerfGate_IAM(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewIAMSuite(), endpoint, b)
}

func TestPerfGate_STS(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewSTSSuite(), endpoint, b)
}

func TestPerfGate_Cognito(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewCognitoSuite(), endpoint, b)
}

func TestPerfGate_EC2(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewEC2Suite(), endpoint, b)
}

func TestPerfGate_KMS(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewKMSSuite(), endpoint, b)
}

func TestPerfGate_Lambda(t *testing.T) {
	endpoint := perfGateEndpoint(t)
	b := loadBaseline(t)
	runSuiteGate(t, tier1.NewLambdaSuite(), endpoint, b)
}

// perfGateEndpoint starts CloudMock or uses PERFGATE_ENDPOINT env var.
func perfGateEndpoint(t *testing.T) string {
	t.Helper()
	ep := os.Getenv("PERFGATE_ENDPOINT")
	if ep != "" {
		return ep
	}
	t.Fatal("PERFGATE_ENDPOINT not set. Start CloudMock in test mode and set PERFGATE_ENDPOINT=http://localhost:4566")
	return ""
}
