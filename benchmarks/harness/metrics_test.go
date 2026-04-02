package harness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputePercentiles(t *testing.T) {
	var latencies []float64
	for i := 1; i <= 100; i++ {
		latencies = append(latencies, float64(i))
	}

	stats := ComputeLatencyStats(latencies)

	assert.InDelta(t, 50.0, stats.P50, 1.0)
	assert.InDelta(t, 95.0, stats.P95, 1.0)
	assert.InDelta(t, 99.0, stats.P99, 1.0)
	assert.Equal(t, 1.0, stats.Min)
	assert.Equal(t, 100.0, stats.Max)
	assert.InDelta(t, 50.5, stats.Mean, 0.1)
}

func TestComputePercentiles_Empty(t *testing.T) {
	stats := ComputeLatencyStats(nil)
	assert.Equal(t, 0.0, stats.P50)
	assert.Equal(t, 0.0, stats.Min)
	assert.Equal(t, 0.0, stats.Max)
}

func TestComputePercentiles_SingleValue(t *testing.T) {
	stats := ComputeLatencyStats([]float64{42.0})
	assert.Equal(t, 42.0, stats.P50)
	assert.Equal(t, 42.0, stats.P95)
	assert.Equal(t, 42.0, stats.P99)
	assert.Equal(t, 42.0, stats.Min)
	assert.Equal(t, 42.0, stats.Max)
	assert.Equal(t, 42.0, stats.Mean)
}
