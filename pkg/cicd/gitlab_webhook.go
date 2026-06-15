package cicd

import (
	"fmt"
	"time"
)

// GitLabPipelineEvent represents the relevant fields from a GitLab CI
// "Pipeline Hook" webhook payload (object_kind: "pipeline").
type GitLabPipelineEvent struct {
	ObjectKind       string `json:"object_kind"` // "pipeline"
	ObjectAttributes struct {
		ID         int64  `json:"id"`
		Ref        string `json:"ref"`
		SHA        string `json:"sha"`
		Status     string `json:"status"`   // success, failed, running, pending, canceled, skipped...
		Duration   int64  `json:"duration"` // seconds
		CreatedAt  string `json:"created_at"`
		FinishedAt string `json:"finished_at"`
		URL        string `json:"url"`
	} `json:"object_attributes"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
	} `json:"project"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
	Builds []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
	} `json:"builds"`
}

// ParseGitLabPipeline converts a GitLab Pipeline Hook payload into our Pipeline
// type, mirroring ParseGitHubWorkflowRun.
func ParseGitLabPipeline(ev GitLabPipelineEvent) Pipeline {
	attrs := ev.ObjectAttributes

	var finishedAt *time.Time
	if t := parseGitLabTime(attrs.FinishedAt); !t.IsZero() {
		finishedAt = &t
	}

	var jobs []Job
	for _, b := range ev.Builds {
		job := Job{
			Name:      b.Name,
			Status:    mapGitLabStatus(b.Status),
			StartedAt: parseGitLabTime(b.StartedAt),
		}
		if ft := parseGitLabTime(b.FinishedAt); !ft.IsZero() {
			job.FinishedAt = &ft
			if !job.StartedAt.IsZero() {
				job.DurationMs = ft.Sub(job.StartedAt).Milliseconds()
			}
		}
		jobs = append(jobs, job)
	}

	url := attrs.URL
	if url == "" {
		url = ev.Project.WebURL
	}

	return Pipeline{
		ID:         fmt.Sprintf("gl-%d", attrs.ID),
		Provider:   "gitlab_ci",
		Repo:       ev.Project.PathWithNamespace,
		Branch:     attrs.Ref,
		CommitHash: attrs.SHA,
		Status:     mapGitLabStatus(attrs.Status),
		StartedAt:  parseGitLabTime(attrs.CreatedAt),
		FinishedAt: finishedAt,
		DurationMs: attrs.Duration * 1000, // GitLab reports seconds
		URL:        url,
		Jobs:       jobs,
	}
}

// mapGitLabStatus normalizes GitLab pipeline/job statuses to our Pipeline
// status vocabulary (running/success/failure/cancelled).
func mapGitLabStatus(status string) string {
	switch status {
	case "success":
		return "success"
	case "failed":
		return "failure"
	case "canceled", "cancelled":
		return "cancelled"
	case "created", "waiting_for_resource", "preparing", "pending", "running", "manual", "scheduled":
		return "running"
	default:
		return status
	}
}

// parseGitLabTime leniently parses the timestamp formats GitLab emits in
// webhooks (which are not RFC3339), returning the zero time on failure.
func parseGitLabTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		"2006-01-02T15:04:05.000Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
