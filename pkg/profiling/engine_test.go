package profiling

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngine_CaptureHeap(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, err := eng.Capture("my-service", "heap", 0)
	if err != nil {
		t.Fatalf("capture heap: %v", err)
	}
	if p.Type != "heap" {
		t.Errorf("expected type heap, got %s", p.Type)
	}
	if p.Size == 0 {
		t.Error("expected non-zero profile size")
	}
	if _, err := os.Stat(p.FilePath); err != nil {
		t.Errorf("profile file should exist: %v", err)
	}
}

func TestEngine_CaptureGoroutine(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, err := eng.Capture("my-service", "goroutine", 0)
	if err != nil {
		t.Fatalf("capture goroutine: %v", err)
	}
	if p.Type != "goroutine" {
		t.Errorf("expected type goroutine, got %s", p.Type)
	}
	if p.Size == 0 {
		t.Error("expected non-zero profile size")
	}
}

func TestEngine_CaptureCPU(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, err := eng.Capture("my-service", "cpu", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("capture cpu: %v", err)
	}
	if p.Type != "cpu" {
		t.Errorf("expected type cpu, got %s", p.Type)
	}
	if p.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %s", p.Duration)
	}
}

func TestEngine_ListFiltering(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	eng.Capture("svc-a", "heap", 0)
	eng.Capture("svc-b", "heap", 0)
	eng.Capture("svc-a", "goroutine", 0)

	all, _ := eng.List("")
	if len(all) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(all))
	}

	filtered, _ := eng.List("svc-a")
	if len(filtered) != 2 {
		t.Errorf("expected 2 profiles for svc-a, got %d", len(filtered))
	}

	filtered, _ = eng.List("svc-b")
	if len(filtered) != 1 {
		t.Errorf("expected 1 profile for svc-b, got %d", len(filtered))
	}
}

func TestEngine_CircularBufferEviction(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 3)

	// Capture 4 profiles; the first should be evicted.
	var firstPath string
	for i := 0; i < 4; i++ {
		p, err := eng.Capture("svc", "heap", 0)
		if err != nil {
			t.Fatalf("capture %d: %v", i, err)
		}
		if i == 0 {
			firstPath = p.FilePath
		}
	}

	// First profile file should have been deleted.
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Error("expected first profile file to be evicted")
	}

	all, _ := eng.List("")
	if len(all) != 3 {
		t.Errorf("expected 3 profiles after eviction, got %d", len(all))
	}

	// Remaining files should exist.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Errorf("expected 3 files on disk, got %d", len(entries))
	}
}

func TestEngine_GetAndFilePath(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, _ := eng.Capture("svc", "heap", 0)

	got, err := eng.Get(p.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("expected ID %s, got %s", p.ID, got.ID)
	}

	fp, err := eng.FilePath(p.ID)
	if err != nil {
		t.Fatalf("filepath: %v", err)
	}
	if filepath.Base(fp) != p.ID+".pprof" {
		t.Errorf("unexpected file path: %s", fp)
	}

	_, err = eng.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestEngine_FindRelevant(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	eng.Capture("svc-a", "heap", 0)
	eng.Capture("svc-b", "heap", 0)

	now := time.Now()
	relevant := eng.FindRelevant("svc-a", now)
	if len(relevant) != 1 {
		t.Errorf("expected 1 relevant profile, got %d", len(relevant))
	}

	// Far in the future should find nothing.
	far := now.Add(10 * time.Minute)
	relevant = eng.FindRelevant("svc-a", far)
	if len(relevant) != 0 {
		t.Errorf("expected 0 relevant profiles for far future, got %d", len(relevant))
	}
}

func TestEngine_UnsupportedType(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	_, err := eng.Capture("svc", "block", 0)
	if err == nil {
		t.Error("expected error for unsupported profile type")
	}
}
