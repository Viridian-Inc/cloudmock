package admin

import (
	"net/http"
	"strings"

	errs "github.com/Viridian-Inc/cloudmock/pkg/errors"
)

// ErrorDeployLink links an error group to the deploy(s) it was first seen in,
// correlating ErrorGroup.Release against known deploy records.
type ErrorDeployLink struct {
	GroupID string        `json:"group_id"`
	Message string        `json:"message"`
	Release string        `json:"release"`
	Status  string        `json:"status"`
	Count   int           `json:"count"`
	Deploys []DeployEvent `json:"deploys"`
}

// handleErrorDeployCorrelation handles GET /api/errors/deploys — for each error
// group that carries a Release matching a known deploy, return the group with
// its matching deploys.
func (a *API) handleErrorDeployCorrelation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.errorStore == nil {
		writeError(w, http.StatusServiceUnavailable, "error tracking not enabled")
		return
	}

	groups, err := a.errorStore.GetGroups("", 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.deploysMu.RLock()
	deploys := make([]DeployEvent, len(a.deploys))
	copy(deploys, a.deploys)
	a.deploysMu.RUnlock()

	links := correlateErrorDeploys(groups, deploys)
	if links == nil {
		links = []ErrorDeployLink{}
	}
	writeJSON(w, http.StatusOK, links)
}

// correlateErrorDeploys joins error groups to deploys by ErrorGroup.Release.
func correlateErrorDeploys(groups []errs.ErrorGroup, deploys []DeployEvent) []ErrorDeployLink {
	var links []ErrorDeployLink
	for _, g := range groups {
		if g.Release == "" {
			continue
		}
		var matched []DeployEvent
		for _, d := range deploys {
			if deployMatchesRelease(d, g.Release) {
				matched = append(matched, d)
			}
		}
		if len(matched) > 0 {
			links = append(links, ErrorDeployLink{
				GroupID: g.ID,
				Message: g.Message,
				Release: g.Release,
				Status:  g.Status,
				Count:   g.Count,
				Deploys: matched,
			})
		}
	}
	return links
}

// deployMatchesRelease reports whether a deploy corresponds to an error group's
// release tag. A release is commonly a commit SHA (full or short), a branch, or
// a deploy ID, so we match on any of those — including a short/full commit
// prefix relationship.
func deployMatchesRelease(d DeployEvent, release string) bool {
	if release == "" {
		return false
	}
	if release == d.Commit || release == d.Branch || release == d.ID {
		return true
	}
	if d.Commit != "" && len(release) >= 7 {
		if strings.HasPrefix(d.Commit, release) || strings.HasPrefix(release, d.Commit) {
			return true
		}
	}
	return false
}
