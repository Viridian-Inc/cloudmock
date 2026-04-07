package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/scm"
)

// scmState holds the current SCM configuration and provider.
type scmState struct {
	config   scm.Config
	provider scm.SCMProvider
}

// SetSCMConfig wires a pre-loaded SCM config to the admin API.
func (a *API) SetSCMConfig(cfg scm.Config) {
	token := cfg.Token
	if token == "" {
		token = os.Getenv("CLOUDMOCK_GITHUB_TOKEN")
	}

	if token == "" || cfg.Provider != "github" {
		return
	}

	a.scm = &scmState{
		config:   cfg,
		provider: scm.NewGitHubProvider(token),
	}
}

// handleSourceContext handles GET /api/source/context — get source code around a file:line.
func (a *API) handleSourceContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.scm == nil {
		writeError(w, http.StatusServiceUnavailable, "SCM integration not configured")
		return
	}

	repo := r.URL.Query().Get("repo")
	filePath := r.URL.Query().Get("file")
	lineStr := r.URL.Query().Get("line")
	contextStr := r.URL.Query().Get("context")

	if repo == "" || filePath == "" || lineStr == "" {
		writeError(w, http.StatusBadRequest, "repo, file, and line parameters are required")
		return
	}

	line, err := strconv.Atoi(lineStr)
	if err != nil || line < 1 {
		writeError(w, http.StatusBadRequest, "line must be a positive integer")
		return
	}

	contextLines := 10
	if contextStr != "" {
		if n, err := strconv.Atoi(contextStr); err == nil && n > 0 {
			contextLines = n
		}
	}

	ctx, err := scm.GetSourceContext(a.scm.provider, repo, filePath, line, contextLines)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

// handleSourceSuspects handles GET /api/source/suspects — get suspect commits for a stack trace.
func (a *API) handleSourceSuspects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.scm == nil {
		writeError(w, http.StatusServiceUnavailable, "SCM integration not configured")
		return
	}

	// Get the error group by ID, then use its stack trace
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "group_id parameter is required")
		return
	}

	if a.errorStore == nil {
		writeError(w, http.StatusServiceUnavailable, "error tracking not enabled")
		return
	}

	group, err := a.errorStore.GetGroup(groupID)
	if err != nil {
		writeError(w, http.StatusNotFound, "error group not found")
		return
	}

	suspects, err := scm.FindSuspectCommits(a.scm.provider, group.Stack, a.scm.config.Repos)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, suspects)
}

// handleSCMConfig handles GET and POST /api/scm/config.
func (a *API) handleSCMConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleGetSCMConfig(w, r)
	case http.MethodPost:
		a.handlePostSCMConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handleGetSCMConfig(w http.ResponseWriter, _ *http.Request) {
	if a.scm == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"configured": false,
			"provider":   "",
			"repos":      []scm.RepoMapping{},
		})
		return
	}

	// Mask the token
	maskedToken := ""
	if a.scm.config.Token != "" {
		t := a.scm.config.Token
		if len(t) > 8 {
			maskedToken = t[:4] + strings.Repeat("*", len(t)-8) + t[len(t)-4:]
		} else {
			maskedToken = "****"
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"provider":   a.scm.config.Provider,
		"token":      maskedToken,
		"repos":      a.scm.config.Repos,
	})
}

func (a *API) handlePostSCMConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var cfg scm.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if cfg.Provider != "github" {
		writeError(w, http.StatusBadRequest, "only 'github' provider is supported")
		return
	}

	token := cfg.Token
	if token == "" {
		token = os.Getenv("CLOUDMOCK_GITHUB_TOKEN")
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required (or set CLOUDMOCK_GITHUB_TOKEN)")
		return
	}

	a.scm = &scmState{
		config:   cfg,
		provider: scm.NewGitHubProvider(token),
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "configured"})
}
