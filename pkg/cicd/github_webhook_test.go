package cicd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseGitHubWorkflowRun_Success(t *testing.T) {
	now := time.Now()
	finished := now.Add(5 * time.Minute)

	wh := GitHubWorkflowRun{
		Action: "completed",
	}
	wh.WorkflowRun.ID = 12345
	wh.WorkflowRun.Name = "CI"
	wh.WorkflowRun.HeadBranch = "main"
	wh.WorkflowRun.HeadSHA = "abc123"
	wh.WorkflowRun.Status = "completed"
	wh.WorkflowRun.Conclusion = "success"
	wh.WorkflowRun.HTMLURL = "https://github.com/org/repo/actions/runs/12345"
	wh.WorkflowRun.CreatedAt = now
	wh.WorkflowRun.UpdatedAt = finished
	wh.WorkflowRun.RunStarted = now
	wh.Repository.FullName = "org/repo"

	pipeline := ParseGitHubWorkflowRun(wh)

	assert.Equal(t, "gh-12345", pipeline.ID)
	assert.Equal(t, "github_actions", pipeline.Provider)
	assert.Equal(t, "org/repo", pipeline.Repo)
	assert.Equal(t, "main", pipeline.Branch)
	assert.Equal(t, "abc123", pipeline.CommitHash)
	assert.Equal(t, "success", pipeline.Status)
	assert.NotNil(t, pipeline.FinishedAt)
	assert.True(t, pipeline.DurationMs > 0)
}

func TestParseGitHubWorkflowRun_Running(t *testing.T) {
	now := time.Now()

	wh := GitHubWorkflowRun{
		Action: "requested",
	}
	wh.WorkflowRun.ID = 99999
	wh.WorkflowRun.Status = "in_progress"
	wh.WorkflowRun.RunStarted = now
	wh.Repository.FullName = "org/repo"

	pipeline := ParseGitHubWorkflowRun(wh)

	assert.Equal(t, "running", pipeline.Status)
	assert.Nil(t, pipeline.FinishedAt)
	assert.Equal(t, int64(0), pipeline.DurationMs)
}

func TestParseGitHubWorkflowRun_Failure(t *testing.T) {
	wh := GitHubWorkflowRun{}
	wh.WorkflowRun.ID = 11111
	wh.WorkflowRun.Status = "completed"
	wh.WorkflowRun.Conclusion = "failure"
	wh.Repository.FullName = "org/repo"

	pipeline := ParseGitHubWorkflowRun(wh)
	assert.Equal(t, "failure", pipeline.Status)
}
