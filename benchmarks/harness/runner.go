package harness

import (
	"context"
	"sync"
	"time"
)

// RunOperation executes an operation through cold, warm, and load phases.
func RunOperation(ctx context.Context, op Operation, endpoint string, iterations int, concurrency int) (*OperationResult, error) {
	result := &OperationResult{Name: op.Name}

	if op.Setup != nil {
		if err := op.Setup(ctx, endpoint); err != nil {
			result.Correctness = GradeFail
			result.Findings = []Finding{{Field: "setup", Expected: "no error", Actual: err.Error(), Grade: GradeFail}}
			return result, nil
		}
	}

	defer func() {
		if op.Teardown != nil {
			op.Teardown(ctx, endpoint)
		}
	}()

	// Cold phase
	coldStart := time.Now()
	coldResp, coldErr := op.Run(ctx, endpoint)
	coldNs := time.Since(coldStart).Nanoseconds()
	if coldNs < 1 {
		coldNs = 1
	}
	result.ColdMs = float64(coldNs) / 1e6

	if coldErr != nil {
		result.Correctness = GradeFail
		result.Findings = []Finding{{Field: "cold_run", Expected: "no error", Actual: coldErr.Error(), Grade: GradeFail}}
		return result, nil
	}

	// Validate
	if op.Validate != nil {
		result.Findings = op.Validate(coldResp)
		result.Correctness = worstGrade(result.Findings)
	} else {
		result.Correctness = GradePass
	}

	// Warm phase
	if iterations > 0 {
		warmLatencies := make([]float64, 0, iterations)
		for i := 0; i < iterations; i++ {
			start := time.Now()
			op.Run(ctx, endpoint)
			ms := float64(time.Since(start).Nanoseconds()) / 1e6
			warmLatencies = append(warmLatencies, ms)
		}
		result.Warm = ComputeLatencyStats(warmLatencies)
	}

	// Load phase
	if concurrency > 0 {
		iterPerGoroutine := iterations / concurrency
		if iterPerGoroutine < 1 {
			iterPerGoroutine = 1
		}
		var mu sync.Mutex
		var loadLatencies []float64
		var wg sync.WaitGroup

		loadStart := time.Now()
		for g := 0; g < concurrency; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < iterPerGoroutine; i++ {
					start := time.Now()
					op.Run(ctx, endpoint)
					ms := float64(time.Since(start).Nanoseconds()) / 1e6
					mu.Lock()
					loadLatencies = append(loadLatencies, ms)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		loadDuration := time.Since(loadStart)

		stats := ComputeLatencyStats(loadLatencies)
		result.Load = LoadStats{
			LatencyStats:        stats,
			ThroughputOpsPerSec: float64(len(loadLatencies)) / loadDuration.Seconds(),
		}
	}

	return result, nil
}

func worstGrade(findings []Finding) Grade {
	if len(findings) == 0 {
		return GradePass
	}
	worst := GradePass
	for _, f := range findings {
		if gradeRank(f.Grade) > gradeRank(worst) {
			worst = f.Grade
		}
	}
	return worst
}

func gradeRank(g Grade) int {
	switch g {
	case GradePass:
		return 0
	case GradePartial:
		return 1
	case GradeFail:
		return 2
	case GradeUnsupported:
		return 3
	default:
		return 0
	}
}
