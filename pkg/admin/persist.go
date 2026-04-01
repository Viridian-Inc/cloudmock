package admin

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SetPersistDir configures the directory where dashboards, views, and deploys
// are persisted as JSON files. On startup it loads any existing files.
// The directory structure is:
//
//	{dir}/dashboards/{id}.json
//	{dir}/views/{id}.json
//	{dir}/deploys/{id}.json
func (a *API) SetPersistDir(dir string) {
	a.persistDir = dir

	// Create subdirectories.
	for _, sub := range []string{"dashboards", "views", "deploys"} {
		os.MkdirAll(filepath.Join(dir, sub), 0755)
	}

	// Load existing dashboards.
	a.dashboardsMu.Lock()
	if loaded := loadAll[Dashboard](filepath.Join(dir, "dashboards")); len(loaded) > 0 {
		a.dashboards = loaded
	}
	a.dashboardsMu.Unlock()

	// Load existing views.
	a.viewsMu.Lock()
	if loaded := loadAll[SavedView](filepath.Join(dir, "views")); len(loaded) > 0 {
		a.views = loaded
	}
	a.viewsMu.Unlock()

	// Load existing deploys.
	a.deploysMu.Lock()
	if loaded := loadAll[DeployEvent](filepath.Join(dir, "deploys")); len(loaded) > 0 {
		a.deploys = loaded
	}
	a.deploysMu.Unlock()
}

// persistDashboards saves all dashboards to disk. Caller must hold dashboardsMu.
func (a *API) persistDashboards() {
	if a.persistDir == "" {
		return
	}
	saveAll(filepath.Join(a.persistDir, "dashboards"), a.dashboards, func(d Dashboard) string { return d.ID })
}

// persistViews saves all views to disk. Caller must hold viewsMu.
func (a *API) persistViews() {
	if a.persistDir == "" {
		return
	}
	saveAll(filepath.Join(a.persistDir, "views"), a.views, func(v SavedView) string { return v.ID })
}

// persistDeploys saves all deploys to disk. Caller must hold deploysMu.
func (a *API) persistDeploys() {
	if a.persistDir == "" {
		return
	}
	saveAll(filepath.Join(a.persistDir, "deploys"), a.deploys, func(d DeployEvent) string { return d.ID })
}

// saveAll writes each item as {dir}/{id(item)}.json, then removes stale files.
func saveAll[T any](dir string, items []T, id func(T) string) {
	// Collect the set of current IDs.
	currentIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		itemID := id(item)
		currentIDs[itemID] = struct{}{}
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			slog.Error("persist: marshal", "error", err)
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, itemID+".json"), data, 0644); err != nil {
			slog.Error("persist: write", "error", err)
		}
	}

	// Remove files for items that no longer exist (handles deletes).
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		fileID := strings.TrimSuffix(e.Name(), ".json")
		if _, ok := currentIDs[fileID]; !ok {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// loadAll reads all .json files from dir and returns the deserialized items.
func loadAll[T any](dir string) []T {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var items []T
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}
