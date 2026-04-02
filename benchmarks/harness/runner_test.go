package harness

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunOperation_CapturesLatency(t *testing.T) {
	op := Operation{
		Name: "TestOp",
		Run: func(ctx context.Context, endpoint string) (any, error) {
			time.Sleep(1 * time.Millisecond)
			return map[string]string{"status": "ok"}, nil
		},
		Validate: func(resp any) []Finding {
			return nil
		},
	}

	result, err := RunOperation(context.Background(), op, "http://localhost:4566", 10, 2)
	require.NoError(t, err)

	assert.Equal(t, "TestOp", result.Name)
	assert.Greater(t, result.ColdMs, 0.0)
	assert.Greater(t, result.Warm.P50, 0.0)
	assert.Greater(t, result.Load.ThroughputOpsPerSec, 0.0)
	assert.Equal(t, GradePass, result.Correctness)
}

func TestRunOperation_FailingValidation(t *testing.T) {
	op := Operation{
		Name: "FailOp",
		Run: func(ctx context.Context, endpoint string) (any, error) {
			return nil, nil
		},
		Validate: func(resp any) []Finding {
			return []Finding{{Field: "body", Expected: "data", Actual: "nil", Grade: GradeFail}}
		},
	}

	result, err := RunOperation(context.Background(), op, "http://localhost:4566", 5, 1)
	require.NoError(t, err)

	assert.Equal(t, GradeFail, result.Correctness)
	assert.Len(t, result.Findings, 1)
}

func TestRunOperation_Quick(t *testing.T) {
	callCount := 0
	op := Operation{
		Name: "QuickOp",
		Run: func(ctx context.Context, endpoint string) (any, error) {
			callCount++
			return "ok", nil
		},
		Validate: func(resp any) []Finding { return nil },
	}

	result, err := RunOperation(context.Background(), op, "http://localhost:4566", 1, 0)
	require.NoError(t, err)
	assert.Greater(t, result.ColdMs, 0.0)
	assert.Equal(t, GradePass, result.Correctness)
}
