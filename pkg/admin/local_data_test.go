package admin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/admin"
)

func seedFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalDataInfo(t *testing.T) {
	api, _ := newTestAPI(t)
	dir := t.TempDir()
	api.SetLocalData(admin.LocalData{
		Project:   "myapp",
		Dir:       dir,
		StateFile: filepath.Join(dir, "state.json"),
		Subdirs:   []string{"chaos", "dashboards"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/local-data", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["project"] != "myapp" {
		t.Errorf("project = %v, want myapp", resp["project"])
	}
	if resp["persistent"] != true {
		t.Errorf("persistent = %v, want true", resp["persistent"])
	}
	if resp["dir"] != dir {
		t.Errorf("dir = %v, want %s", resp["dir"], dir)
	}
	if resp["onDisk"] != true {
		t.Errorf("onDisk = %v, want true", resp["onDisk"])
	}
}

func TestLocalDataInfo_InMemory(t *testing.T) {
	api, _ := newTestAPI(t) // no SetLocalData → in-memory
	req := httptest.NewRequest(http.MethodGet, "/api/local-data", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["persistent"] != false {
		t.Errorf("persistent = %v, want false for in-memory mode", resp["persistent"])
	}
}

func TestLocalDataDelete(t *testing.T) {
	api, _ := newTestAPI(t)
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")

	// Seed on-disk data: state file (+ a stray .tmp), two managed subdirs with
	// files, and a plugins/ dir that MUST survive the wipe.
	seedFile(t, stateFile, `{"version":1}`)
	seedFile(t, stateFile+".tmp", `partial`)
	seedFile(t, filepath.Join(dir, "chaos", "r1.json"), `{}`)
	seedFile(t, filepath.Join(dir, "dashboards", "d1.json"), `{}`)
	seedFile(t, filepath.Join(dir, "plugins", "mybin"), `binary`)

	api.SetLocalData(admin.LocalData{
		Project:   "myapp",
		Dir:       dir,
		StateFile: stateFile,
		Subdirs:   []string{"chaos", "dashboards"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/local-data/delete", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	// State file and its .tmp sibling are gone.
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Errorf("state file still exists (err=%v)", err)
	}
	if _, err := os.Stat(stateFile + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("state .tmp still exists (err=%v)", err)
	}

	// Managed subdirs are recreated but empty.
	for _, sub := range []string{"chaos", "dashboards"} {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			t.Errorf("subdir %s missing after wipe: %v", sub, err)
			continue
		}
		if len(entries) != 0 {
			t.Errorf("subdir %s not empty after wipe: %d entries", sub, len(entries))
		}
	}

	// plugins/ is preserved — a wipe must never delete plugin binaries.
	if _, err := os.Stat(filepath.Join(dir, "plugins", "mybin")); err != nil {
		t.Errorf("plugins/mybin was deleted by wipe (err=%v)", err)
	}
}

func TestLocalDataDelete_NotConfigured(t *testing.T) {
	api, _ := newTestAPI(t) // no SetLocalData
	req := httptest.NewRequest(http.MethodPost, "/api/local-data/delete", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (in-memory, nothing to delete)", w.Code)
	}
}

func TestLocalDataDelete_MethodNotAllowed(t *testing.T) {
	api, _ := newTestAPI(t)
	api.SetLocalData(admin.LocalData{Dir: t.TempDir()})
	req := httptest.NewRequest(http.MethodGet, "/api/local-data/delete", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}
