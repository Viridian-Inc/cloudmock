package cicd

import (
	"fmt"
	"time"
)

// GitHubWorkflowRun represents the relevant fields from a GitHub Actions
// workflow_run webhook payload.
type GitHubWorkflowRun struct {
	Action      string `json:"action"` // "completed", "requested", etc.
	WorkflowRun struct {
		ID         int64      `json:"id"`
		Name       string     `json:"name"`
		HeadBranch string     `json:"head_branch"`
		HeadSHA    string     `json:"head_sha"`
		Status     string     `json:"status"`     // "completed", "in_progress", "queued"
		Conclusion string     `json:"conclusion"` // "success", "failure", "cancelled", "skipped"
		HTMLURL    string     `json:"html_url"`
		CreatedAt  time.Time  `json:"created_at"`
		UpdatedAt  time.Time  `json:"updated_at"`
		RunStarted time.Time  `json:"run_started_at"`
		Jobs       []struct { // Not in webhook payload; used when enriched via API
			Name       string     `json:"name"`
			Status     string     `json:"status"`
			Conclusion string     `json:"conclusion"`
			StartedAt  time.Time  `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at"`
		} `json:"jobs,omitempty"`
	} `json:"workflow_run"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// ParseGitHubWorkflowRun converts a GitHub Actions workflow_run webhook
// payload into our Pipeline type.
func ParseGitHubWorkflowRun(wh GitHubWorkflowRun) Pipeline {
	run := wh.WorkflowRun

	status := mapGitHubStatus(run.Status, run.Conclusion)

	var finishedAt *time.Time
	var durationMs int64
	if run.Status == "completed" {
		t := run.UpdatedAt
		finishedAt = &t
		durationMs = t.Sub(run.RunStarted).Milliseconds()
	}

	var jobs []Job
	for _, j := range run.Jobs {
		jStatus := mapGitHubStatus(j.Status, j.Conclusion)
		job := Job{
			Name:      j.Name,
			Status:    jStatus,
			StartedAt: j.StartedAt,
		}
		if j.CompletedAt != nil {
			job.FinishedAt = j.CompletedAt
			job.DurationMs = j.CompletedAt.Sub(j.StartedAt).Milliseconds()
		}
		jobs = append(jobs, job)
	}

	return Pipeline{
		ID:         fmt.Sprintf("gh-%d", run.ID),
		Provider:   "github_actions",
		Repo:       wh.Repository.FullName,
		Branch:     run.HeadBranch,
		CommitHash: run.HeadSHA,
		Status:     status,
		StartedAt:  run.RunStarted,
		FinishedAt: finishedAt,
		DurationMs: durationMs,
		URL:        run.HTMLURL,
		Jobs:       jobs,
	}
}

func mapGitHubStatus(status, conclusion string) string {
	if status != "completed" {
		return "running"
	}
	switch conclusion {
	case "success":
		return "success"
	case "failure":
		return "failure"
	case "cancelled":
		return "cancelled"
	default:
		return conclusion
	}
}
