//go:build smoke

package benchmarks

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/benchmarks/harness"
	"github.com/neureaux/cloudmock/benchmarks/suites"
	"github.com/neureaux/cloudmock/benchmarks/suites/tier1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmark_S3_Quick(t *testing.T) {
	endpoint := "http://localhost:4566"

	suite := tier1.NewS3Suite()
	for _, op := range suite.Operations() {
		t.Run(op.Name, func(t *testing.T) {
			result, err := harness.RunOperation(context.Background(), op, endpoint, 1, 0)
			require.NoError(t, err)
			assert.NotEqual(t, harness.GradeUnsupported, result.Correctness,
				"operation %s should be supported", op.Name)
			t.Logf("%s: cold=%.1fms correctness=%s", op.Name, result.ColdMs, result.Correctness)
		})
	}
}

func TestBenchmark_Registry_Count(t *testing.T) {
	r := suites.NewRegistry()
	r.Register(tier1.NewS3Suite())
	r.Register(tier1.NewDynamoDBSuite())
	r.Register(tier1.NewSQSSuite())

	assert.GreaterOrEqual(t, len(r.List()), 3)
}
