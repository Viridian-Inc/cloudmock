package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/cicd"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// handlePipelines handles listing and creating pipelines.
// GET  /api/pipelines — list recent pipelines
// POST /api/pipelines — ingest pipeline data (from webhook)
func (a *API) handlePipelines(w http.ResponseWriter, r *http.Request) {
	if a.cicdStore == nil {
		writeError(w, http.StatusServiceUnavailable, "CI/CD store not available")
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = n
			}
		}
		pipelines := a.cicdStore.ListPipelines(limit)
		writeJSON(w, http.StatusOK, pipelines)

	case http.MethodPost:
		var p cicd.Pipeline
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if p.ID == "" {
			p.ID = fmt.Sprintf("pipe-%d", time.Now().UnixNano())
		}
		if p.StartedAt.IsZero() {
			p.StartedAt = time.Now()
		}

		if err := a.cicdStore.SavePipeline(p); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// If pipeline completed successfully, create a deploy event.
		if p.Status == "success" {
			a.createDeployFromPipeline(r, p)
		}

		a.auditLog(r.Context(), "pipeline.created", "pipeline:"+p.ID, map[string]any{
			"repo":   p.Repo,
			"status": p.Status,
		})
		writeJSON(w, http.StatusCreated, p)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePipelineByID handles GET /api/pipelines/{id}.
func (a *API) handlePipelineByID(w http.ResponseWriter, r *http.Request) {
	if a.cicdStore == nil {
		writeError(w, http.StatusServiceUnavailable, "CI/CD store not available")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/pipelines/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	// Route to sub-resources.
	if len(parts) == 2 && parts[1] == "tests" {
		a.handlePipelineTests(w, r, id)
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	p, err := a.cicdStore.GetPipeline(id)
	if err != nil {
		if err == cicd.ErrNotFound {
			writeError(w, http.StatusNotFound, "pipeline not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handlePipelineTests handles test results for a pipeline.
// GET  /api/pipelines/{id}/tests — get test results
// POST /api/pipelines/{id}/tests — ingest test results
func (a *API) handlePipelineTests(w http.ResponseWriter, r *http.Request, pipelineID string) {
	switch r.Method {
	case http.MethodGet:
		results, err := a.cicdStore.GetTestResults(pipelineID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)

	case http.MethodPost:
		var results []cicd.TestResult
		if err := json.NewDecoder(r.Body).Decode(&results); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		// Set pipeline ID on all results.
		for i := range results {
			results[i].PipelineID = pipelineID
		}
		if err := a.cicdStore.SaveTestResults(pipelineID, results); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "pipeline.tests.ingested", "pipeline:"+pipelineID, map[string]any{
			"count": len(results),
		})
		writeJSON(w, http.StatusCreated, map[string]int{"count": len(results)})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleCISummary returns overall CI health metrics.
// GET /api/ci/summary
func (a *API) handleCISummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.cicdStore == nil {
		writeError(w, http.StatusServiceUnavailable, "CI/CD store not available")
		return
	}

	summary := a.cicdStore.Summary()
	writeJSON(w, http.StatusOK, summary)
}

// handleGitHubWebhook processes GitHub Actions workflow_run webhook events.
// POST /api/webhooks/github
func (a *API) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.cicdStore == nil {
		writeError(w, http.StatusServiceUnavailable, "CI/CD store not available")
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "workflow_run" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "event": eventType})
		return
	}

	var wh cicd.GitHubWorkflowRun
	if err := json.NewDecoder(r.Body).Decode(&wh); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	pipeline := cicd.ParseGitHubWorkflowRun(wh)
	if err := a.cicdStore.SavePipeline(pipeline); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If pipeline completed successfully, create a deploy event.
	if pipeline.Status == "success" {
		a.createDeployFromPipeline(r, pipeline)
	}

	a.auditLog(r.Context(), "github.webhook.processed", "pipeline:"+pipeline.ID, map[string]any{
		"repo":   pipeline.Repo,
		"status": pipeline.Status,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "processed", "pipeline_id": pipeline.ID})
}

// createDeployFromPipeline creates a deploy event when a pipeline succeeds.
func (a *API) createDeployFromPipeline(r *http.Request, p cicd.Pipeline) {
	if a.dp != nil {
		deploy := dataplane.DeployEvent{
			ID:          fmt.Sprintf("deploy-ci-%s", p.ID),
			Service:     p.Repo,
			Version:     p.Branch,
			CommitSHA:   p.CommitHash,
			Author:      p.Provider,
			Description: fmt.Sprintf("CI pipeline %s completed successfully", p.ID),
			DeployedAt:  time.Now(),
			Metadata: map[string]string{
				"pipeline_id": p.ID,
				"provider":    p.Provider,
				"url":         p.URL,
			},
		}
		_ = a.dp.Config.AddDeploy(r.Context(), deploy)
	} else {
		// Fallback to in-memory deploy list.
		deploy := DeployEvent{
			ID:        fmt.Sprintf("deploy-ci-%s", p.ID),
			Service:   p.Repo,
			Commit:    p.CommitHash,
			Branch:    p.Branch,
			Author:    p.Provider,
			Message:   fmt.Sprintf("CI pipeline %s completed successfully", p.ID),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		a.deploysMu.Lock()
		a.deploys = append(a.deploys, deploy)
		a.deploysMu.Unlock()
	}
}
