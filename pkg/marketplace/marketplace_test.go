package marketplace

import (
	"testing"
)

func TestNewRegistryHasSeededListings(t *testing.T) {
	r := NewRegistry()
	listings := r.List()
	if len(listings) == 0 {
		t.Fatal("expected seeded listings")
	}
	// Should have at least the 3 built-in + community plugins
	if len(listings) < 3 {
		t.Errorf("expected at least 3 listings, got %d", len(listings))
	}
}

func TestGetByID(t *testing.T) {
	r := NewRegistry()

	k8s, ok := r.Get("kubernetes")
	if !ok {
		t.Fatal("expected to find kubernetes")
	}
	if k8s.Name != "Kubernetes" {
		t.Errorf("name = %q, want %q", k8s.Name, "Kubernetes")
	}
	if !k8s.Installed {
		t.Error("kubernetes should be installed by default")
	}
}

func TestGetNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestSearchByQuery(t *testing.T) {
	r := NewRegistry()

	results := r.Search("kubernetes", "")
	if len(results) == 0 {
		t.Fatal("expected results for 'kubernetes'")
	}
	if results[0].ID != "kubernetes" {
		t.Errorf("first result ID = %q, want %q", results[0].ID, "kubernetes")
	}
}

func TestSearchByCategory(t *testing.T) {
	r := NewRegistry()

	results := r.Search("", "integration")
	if len(results) == 0 {
		t.Fatal("expected integration plugins")
	}
	for _, l := range results {
		if l.Category != "integration" {
			t.Errorf("expected category 'integration', got %q for %q", l.Category, l.Name)
		}
	}
}

func TestSearchByQueryAndCategory(t *testing.T) {
	r := NewRegistry()

	results := r.Search("argo", "integration")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "argocd" {
		t.Errorf("expected argocd, got %s", results[0].ID)
	}
}

func TestSearchNoResults(t *testing.T) {
	r := NewRegistry()
	results := r.Search("zzzznonexistent", "")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestInstall(t *testing.T) {
	r := NewRegistry()

	// terraform-state should not be installed by default
	tf, ok := r.Get("terraform-state")
	if !ok {
		t.Fatal("expected terraform-state listing")
	}
	if tf.Installed {
		t.Error("terraform-state should not be installed by default")
	}

	prevDownloads := tf.Downloads

	err := r.Install("terraform-state")
	if err != nil {
		t.Fatalf("install error: %v", err)
	}

	tf, _ = r.Get("terraform-state")
	if !tf.Installed {
		t.Error("should be installed after Install()")
	}
	if tf.Downloads != prevDownloads+1 {
		t.Errorf("downloads = %d, want %d", tf.Downloads, prevDownloads+1)
	}
}

func TestInstallNotFound(t *testing.T) {
	r := NewRegistry()
	err := r.Install("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUninstall(t *testing.T) {
	r := NewRegistry()

	err := r.Uninstall("kubernetes")
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	k8s, _ := r.Get("kubernetes")
	if k8s.Installed {
		t.Error("should not be installed after Uninstall()")
	}
}

func TestUninstallNotFound(t *testing.T) {
	r := NewRegistry()
	err := r.Uninstall("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
