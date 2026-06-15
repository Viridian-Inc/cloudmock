package cicd

import "testing"

func TestParseGitLabPipeline_Success(t *testing.T) {
	var ev GitLabPipelineEvent
	ev.ObjectKind = "pipeline"
	ev.ObjectAttributes.ID = 42
	ev.ObjectAttributes.Ref = "main"
	ev.ObjectAttributes.SHA = "abc123"
	ev.ObjectAttributes.Status = "success"
	ev.ObjectAttributes.Duration = 63 // seconds
	ev.ObjectAttributes.CreatedAt = "2026-06-15 12:00:00 UTC"
	ev.ObjectAttributes.FinishedAt = "2026-06-15 12:01:03 UTC"
	ev.ObjectAttributes.URL = "https://gitlab.com/g/p/-/pipelines/42"
	ev.Project.PathWithNamespace = "g/p"
	ev.Builds = []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		StartedAt  string `json:"started_at"`
		FinishedAt string `json:"finished_at"`
	}{
		{Name: "build", Status: "success", StartedAt: "2026-06-15 12:00:00 UTC", FinishedAt: "2026-06-15 12:00:30 UTC"},
	}

	p := ParseGitLabPipeline(ev)

	if p.ID != "gl-42" {
		t.Errorf("ID = %q, want gl-42", p.ID)
	}
	if p.Provider != "gitlab_ci" {
		t.Errorf("Provider = %q, want gitlab_ci", p.Provider)
	}
	if p.Repo != "g/p" || p.Branch != "main" || p.CommitHash != "abc123" {
		t.Errorf("repo/branch/commit = %q/%q/%q", p.Repo, p.Branch, p.CommitHash)
	}
	if p.Status != "success" {
		t.Errorf("Status = %q, want success", p.Status)
	}
	if p.DurationMs != 63000 {
		t.Errorf("DurationMs = %d, want 63000 (seconds→ms)", p.DurationMs)
	}
	if p.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if len(p.Jobs) != 1 || p.Jobs[0].Status != "success" || p.Jobs[0].DurationMs != 30000 {
		t.Errorf("jobs = %+v", p.Jobs)
	}
}

func TestMapGitLabStatus(t *testing.T) {
	cases := map[string]string{
		"success":  "success",
		"failed":   "failure",
		"canceled": "cancelled",
		"running":  "running",
		"pending":  "running",
		"created":  "running",
		"weird":    "weird", // unknown statuses pass through
	}
	for in, want := range cases {
		if got := mapGitLabStatus(in); got != want {
			t.Errorf("mapGitLabStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
