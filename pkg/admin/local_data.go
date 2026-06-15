package admin

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// LocalData describes the on-disk persistence footprint of the running
// cloudmock instance: its named project, the root data directory, the snapshot
// state file, and the data subdirectories cloudmock manages under the data
// directory. It is set from the gateway at startup via SetLocalData and powers
// the /api/local-data endpoints (report + wipe). The shared plugins/ directory
// is deliberately NOT included so a wipe never deletes plugin binaries.
type LocalData struct {
	Project   string   // project/database name; "" when unnamed (default ~/.cloudmock)
	Dir       string   // root data directory (baseDir)
	StateFile string   // snapshot file path; "" when disk persistence is off
	Subdirs   []string // data subdirectories under Dir that a wipe may remove
}

// SetLocalData records this instance's on-disk persistence footprint for the
// /api/local-data endpoints.
func (a *API) SetLocalData(ld LocalData) {
	a.localData = &ld
}

// handleLocalDataInfo reports the current project/database name, data
// directory, and whether disk persistence is active. GET /api/local-data.
func (a *API) handleLocalDataInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ld := a.localData
	if ld == nil || ld.Dir == "" {
		// No data directory configured → fully in-memory.
		writeJSON(w, http.StatusOK, map[string]any{
			"project":    "",
			"dir":        "",
			"stateFile":  "",
			"persistent": false,
			"onDisk":     false,
		})
		return
	}
	onDisk := false
	if _, err := os.Stat(ld.Dir); err == nil {
		onDisk = true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":    ld.Project,
		"dir":        ld.Dir,
		"stateFile":  ld.StateFile,
		"persistent": ld.StateFile != "",
		"onDisk":     onDisk,
	})
}

// handleLocalDataDelete wipes all locally-stored on-disk data for the current
// project/instance: it resets in-memory service state, clears admin
// dashboards/views/deploys, removes the snapshot state file, and deletes the
// managed data subdirectories (recreating them empty so the running stores
// keep working). The shared plugins/ directory is never touched, and other
// named projects on disk are left intact. POST /api/local-data/delete.
func (a *API) handleLocalDataDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ld := a.localData
	if ld == nil || ld.Dir == "" {
		writeError(w, http.StatusConflict, "no local data directory configured (running in-memory)")
		return
	}

	// 1. Reset in-memory service state (snapshotable services), mirroring
	//    /api/state/reset so the live server is empty after the wipe.
	var resetServices []string
	for _, svc := range a.registry.All() {
		if snap, ok := svc.(service.Snapshotable); ok {
			_ = snap.ImportState([]byte("{}"))
			resetServices = append(resetServices, svc.Name())
		}
	}

	// 2. Clear admin in-memory dashboards/views/deploys.
	a.dashboardsMu.Lock()
	a.dashboards = nil
	a.dashboardsMu.Unlock()
	a.viewsMu.Lock()
	a.views = nil
	a.viewsMu.Unlock()
	a.deploysMu.Lock()
	a.deploys = nil
	a.deploysMu.Unlock()

	// 3. Remove on-disk data: the snapshot file (+ its atomic .tmp sibling)
	//    and each managed subdirectory. plugins/ is never in Subdirs.
	var removed, failures []string
	if ld.StateFile != "" {
		for _, f := range []string{ld.StateFile, ld.StateFile + ".tmp"} {
			if err := os.Remove(f); err == nil {
				removed = append(removed, f)
			} else if !os.IsNotExist(err) {
				failures = append(failures, f+": "+err.Error())
			}
		}
	}
	for _, sub := range ld.Subdirs {
		if sub == "" {
			continue
		}
		dir := filepath.Join(ld.Dir, sub)
		if err := os.RemoveAll(dir); err != nil {
			failures = append(failures, dir+": "+err.Error())
			continue
		}
		removed = append(removed, dir)
		// Recreate empty so the running file-backed stores keep working.
		_ = os.MkdirAll(dir, 0o755)
	}

	a.auditLog(r.Context(), "local_data.deleted", "state:local-data", map[string]any{
		"project":        ld.Project,
		"dir":            ld.Dir,
		"removed":        len(removed),
		"reset_services": len(resetServices),
	})

	resp := map[string]any{
		"status":         "deleted",
		"project":        ld.Project,
		"dir":            ld.Dir,
		"removed":        removed,
		"reset_services": resetServices,
	}
	if len(failures) > 0 {
		resp["failures"] = failures
		writeJSON(w, http.StatusInternalServerError, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
