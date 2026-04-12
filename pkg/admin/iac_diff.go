package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/iac"
)

// SetIaCResult stores the latest IaC scan result for the diff endpoint.
// Called by the gateway's watcher callback after each re-scan.
func (a *API) SetIaCResult(result *iac.IaCImportResult) {
	a.iacResultMu.Lock()
	defer a.iacResultMu.Unlock()
	a.iacResult = result
}

// handleIaCDiff computes and returns the diff between IaC-declared resources
// and what's currently provisioned in CloudMock.
//
// GET /api/iac/diff
//
// Response: { entries: [...], summary: { total, synced, missing, orphaned, drift } }
func (a *API) handleIaCDiff(w http.ResponseWriter, r *http.Request) {
	a.iacResultMu.RLock()
	iacResult := a.iacResult
	a.iacResultMu.RUnlock()

	if iacResult == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"entries": []any{},
			"summary": map[string]int{
				"total": 0, "synced": 0, "missing": 0, "orphaned": 0, "drift": 0,
			},
			"message": "No IaC project detected. Use --iac <path> to enable.",
		})
		return
	}

	diff := iac.ComputeDiff(iacResult, a.registry, slog.Default())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}
