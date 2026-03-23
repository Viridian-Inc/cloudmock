package profiling

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFoldedStacks_Heap(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, err := eng.Capture("svc", "heap", 0)
	if err != nil {
		t.Fatalf("capture heap: %v", err)
	}

	folded, err := eng.FoldedStacks(p.ID)
	if err != nil {
		t.Fatalf("folded stacks: %v", err)
	}

	if folded == "" {
		t.Fatal("expected non-empty folded stacks output")
	}

	// Each line should match "func;func... count" format.
	for _, line := range strings.Split(strings.TrimSpace(folded), "\n") {
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			t.Errorf("unexpected folded line format: %q", line)
			continue
		}
		stack := parts[0]
		if !strings.Contains(stack, ";") && !strings.Contains(stack, ".") {
			t.Errorf("expected function names in stack: %q", stack)
		}
	}
}

func TestFoldedStacks_Goroutine(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	p, err := eng.Capture("svc", "goroutine", 0)
	if err != nil {
		t.Fatalf("capture goroutine: %v", err)
	}

	folded, err := eng.FoldedStacks(p.ID)
	if err != nil {
		t.Fatalf("folded stacks: %v", err)
	}

	if folded == "" {
		t.Fatal("expected non-empty folded stacks for goroutine profile")
	}
}

func TestFoldedStacks_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	// Create a fake profile entry with an invalid file.
	badPath := filepath.Join(dir, "bad.pprof")
	os.WriteFile(badPath, []byte("not a real pprof"), 0644)

	eng.mu.Lock()
	eng.profiles = append(eng.profiles, Profile{
		ID:       "bad-profile",
		FilePath: badPath,
	})
	eng.mu.Unlock()

	_, err := eng.FoldedStacks("bad-profile")
	if err == nil {
		t.Error("expected error for invalid pprof file")
	}
}

func TestFoldedStacks_NotFound(t *testing.T) {
	dir := t.TempDir()
	eng := New(dir, 10)

	_, err := eng.FoldedStacks("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}
