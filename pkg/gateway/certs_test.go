package gateway

import (
	"testing"
)

func TestBuildSANs_Basic(t *testing.T) {
	sans := buildSANs("example.com", "mock.dev")

	expected := []string{
		"localhost",
		"*.localhost",
		"localhost.example.com",
		"*.localhost.example.com",
		"localhost.mock.dev",
		"*.localhost.mock.dev",
	}

	if len(sans) != len(expected) {
		t.Fatalf("expected %d SANs, got %d: %v", len(expected), len(sans), sans)
	}

	for i, want := range expected {
		if sans[i] != want {
			t.Errorf("SANs[%d] = %q, want %q", i, sans[i], want)
		}
	}
}

func TestBuildSANs_NoDomains(t *testing.T) {
	sans := buildSANs()
	if len(sans) != 2 {
		t.Fatalf("expected 2 base SANs, got %d: %v", len(sans), sans)
	}
	if sans[0] != "localhost" || sans[1] != "*.localhost" {
		t.Errorf("unexpected base SANs: %v", sans)
	}
}

func TestSansMatch_AllPresent(t *testing.T) {
	current := []string{"localhost", "*.localhost", "localhost.example.com", "*.localhost.example.com", "extra.thing"}
	needed := []string{"localhost", "*.localhost", "localhost.example.com"}

	if !sansMatch(current, needed) {
		t.Error("sansMatch should return true when all needed SANs are present")
	}
}

func TestSansMatch_Missing(t *testing.T) {
	current := []string{"localhost", "*.localhost"}
	needed := []string{"localhost", "*.localhost", "localhost.example.com"}

	if sansMatch(current, needed) {
		t.Error("sansMatch should return false when a needed SAN is missing")
	}
}

func TestSansMatch_Empty(t *testing.T) {
	if !sansMatch([]string{"a", "b"}, []string{}) {
		t.Error("sansMatch should return true when needed is empty")
	}
	if !sansMatch([]string{}, []string{}) {
		t.Error("sansMatch should return true when both are empty")
	}
}
