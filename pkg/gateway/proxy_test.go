package gateway

import (
	"strings"
	"testing"
)

func TestBuildRoutes_LocalhostRoutesExist(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	// Verify all expected .localhost hosts are present
	expectedHosts := []string{
		"autotend-app.localhost",
		"cloudmock.localhost",
		"bff.localhost",
		"api.localhost",
		"auth.localhost",
		"admin.localhost",
		"graphql.localhost",
	}

	for _, want := range expectedHosts {
		found := false
		for _, r := range routes {
			if r.Host == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected .localhost route for host %q, not found", want)
		}
	}
}

func TestBuildRoutes_CustomDomainRoutesUseDomains(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	// Check autotend custom domain routes
	atDomain := "localhost.example.com"
	cmDomain := "localhost.mock.dev"

	foundAT := false
	foundCM := false
	for _, r := range routes {
		if r.Host == atDomain || strings.HasSuffix(r.Host, "."+atDomain) {
			foundAT = true
		}
		if r.Host == cmDomain {
			foundCM = true
		}
	}

	if !foundAT {
		t.Error("expected custom domain routes for localhost.example.com, not found")
	}
	if !foundCM {
		t.Error("expected custom domain routes for localhost.mock.dev, not found")
	}
}

func TestBuildRoutes_NoHardcodedDomains(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	hardcoded := []string{"autotend.io", "cloudmock.io"}
	for _, r := range routes {
		for _, hc := range hardcoded {
			if strings.Contains(r.Host, hc) {
				t.Errorf("route host %q contains hardcoded domain %q", r.Host, hc)
			}
		}
	}
}

func TestBuildRoutes_PathPrefixOrdering(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	// For cloudmock.localhost, /_cloudmock/ and /api/ must come before /
	var cloudmockPaths []string
	for _, r := range routes {
		if r.Host == "cloudmock.localhost" {
			cloudmockPaths = append(cloudmockPaths, r.Path)
		}
	}

	if len(cloudmockPaths) < 3 {
		t.Fatalf("expected at least 3 cloudmock.localhost routes, got %d", len(cloudmockPaths))
	}

	// The catch-all "/" must be last
	if cloudmockPaths[len(cloudmockPaths)-1] != "/" {
		t.Errorf("expected catch-all '/' to be last for cloudmock.localhost, got %q", cloudmockPaths[len(cloudmockPaths)-1])
	}

	// More specific paths should come before /
	for i, p := range cloudmockPaths[:len(cloudmockPaths)-1] {
		if p == "/" {
			t.Errorf("catch-all '/' at position %d is not last for cloudmock.localhost", i)
		}
	}
}
