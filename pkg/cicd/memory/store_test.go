package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/cicd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndGetPipeline(t *testing.T) {
	s := NewStore()

	p := cicd.Pipeline{
		ID:       "pipe-1",
		Provider: "github_actions",
		Repo:     "org/repo",
		Branch:   "main",
		Status:   "success",
		StartedAt: time.Now(),
	}

	err := s.SavePipeline(p)
	require.NoError(t, err)

	got, err := s.GetPipeline("pipe-1")
	require.NoError(t, err)
	assert.Equal(t, "pipe-1", got.ID)
	assert.Equal(t, "github_actions", got.Provider)
}

func TestGetPipelineNotFound(t *testing.T) {
	s := NewStore()
	_, err := s.GetPipeline("nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListPipelinesNewestFirst(t *testing.T) {
	s := NewStore()

	for i := 0; i < 5; i++ {
		s.SavePipeline(cicd.Pipeline{
			ID:        fmt.Sprintf("pipe-%d", i),
			StartedAt: time.Now().Add(time.Duration(i) * time.Minute),
		})
	}

	results := s.ListPipelines(3)
	require.Len(t, results, 3)
	assert.Equal(t, "pipe-4", results[0].ID)
	assert.Equal(t, "pipe-3", results[1].ID)
	assert.Equal(t, "pipe-2", results[2].ID)
}

func TestCircularBuffer(t *testing.T) {
	s := NewStore()

	// Fill beyond max capacity.
	for i := 0; i < maxPipelines+10; i++ {
		s.SavePipeline(cicd.Pipeline{
			ID:     fmt.Sprintf("pipe-%d", i),
			Status: "success",
		})
	}

	all := s.ListPipelines(0)
	assert.Len(t, all, maxPipelines)

	// Oldest should have been evicted.
	_, err := s.GetPipeline("pipe-0")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateExistingPipeline(t *testing.T) {
	s := NewStore()

	s.SavePipeline(cicd.Pipeline{ID: "pipe-1", Status: "running"})
	s.SavePipeline(cicd.Pipeline{ID: "pipe-1", Status: "success"})

	got, _ := s.GetPipeline("pipe-1")
	assert.Equal(t, "success", got.Status)

	all := s.ListPipelines(0)
	assert.Len(t, all, 1)
}

func TestSaveAndGetTestResults(t *testing.T) {
	s := NewStore()

	results := []cicd.TestResult{
		{PipelineID: "pipe-1", Suite: "unit", Name: "TestFoo", Status: "passed"},
		{PipelineID: "pipe-1", Suite: "unit", Name: "TestBar", Status: "failed", Error: "assertion failed"},
	}

	err := s.SaveTestResults("pipe-1", results)
	require.NoError(t, err)

	got, err := s.GetTestResults("pipe-1")
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestSummary(t *testing.T) {
	s := NewStore()

	s.SavePipeline(cicd.Pipeline{ID: "p1", Status: "success", Provider: "github_actions", DurationMs: 1000})
	s.SavePipeline(cicd.Pipeline{ID: "p2", Status: "failure", Provider: "github_actions", DurationMs: 2000})
	s.SavePipeline(cicd.Pipeline{ID: "p3", Status: "success", Provider: "gitlab_ci", DurationMs: 3000})

	summary := s.Summary()
	assert.Equal(t, 3, summary.TotalPipelines)
	assert.InDelta(t, 0.6667, summary.PassRate, 0.01)
	assert.InDelta(t, 2000, summary.AvgDurationMs, 0.01)
	assert.Equal(t, 2, summary.ByProvider["github_actions"])
	assert.Equal(t, 1, summary.ByProvider["gitlab_ci"])
}

func TestFlakyTestDetection(t *testing.T) {
	s := NewStore()

	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("p%d", i)
		s.SavePipeline(cicd.Pipeline{ID: id, Status: "success"})
		status := "passed"
		if i%2 == 0 {
			status = "failed"
		}
		s.SaveTestResults(id, []cicd.TestResult{
			{PipelineID: id, Suite: "integration", Name: "TestFlaky", Status: status},
		})
	}

	summary := s.Summary()
	require.Len(t, summary.FlakyTests, 1)
	assert.Equal(t, "TestFlaky", summary.FlakyTests[0].Name)
	assert.Equal(t, 5, summary.FlakyTests[0].Runs)
}
