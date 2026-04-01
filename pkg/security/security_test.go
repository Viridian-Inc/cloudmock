package security

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
)

// mockService satisfies the service.Service interface for testing.
type mockService struct {
	name string
}

func (m *mockService) Name() string                   { return m.name }
func (m *mockService) Actions() []service.Action       { return nil }
func (m *mockService) HealthCheck() error              { return nil }
func (m *mockService) HandleRequest(_ *service.RequestContext) (*service.Response, error) {
	return nil, nil
}

func newTestRegistry(names ...string) *routing.Registry {
	reg := routing.NewRegistry()
	for _, name := range names {
		reg.Register(&mockService{name: name})
	}
	return reg
}

func TestScannerChecks(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3", "iam"))
	checks := scanner.Checks()
	if len(checks) != 10 {
		t.Errorf("expected 10 built-in checks, got %d", len(checks))
	}
}

func TestScanReturnsFindings(t *testing.T) {
	reg := newTestRegistry("s3", "iam", "dynamodb", "secretsmanager", "kms", "lambda", "sqs", "sns")
	scanner := NewScanner(reg)
	result := scanner.Scan()

	if result == nil {
		t.Fatal("expected scan result")
	}
	if result.ScanID == "" {
		t.Error("expected scan ID")
	}
	if len(result.Findings) == 0 {
		t.Error("expected findings")
	}
	if result.Summary.Total != len(result.Findings) {
		t.Errorf("summary total %d != findings count %d", result.Summary.Total, len(result.Findings))
	}
}

func TestScanCachesLastResult(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3"))
	if scanner.LastScan() != nil {
		t.Error("expected no last scan initially")
	}

	scanner.Scan()
	if scanner.LastScan() == nil {
		t.Error("expected cached last scan")
	}
}

func TestCloudTrailNotEnabled(t *testing.T) {
	// Registry without cloudtrail service.
	scanner := NewScanner(newTestRegistry("s3"))
	result := scanner.Scan()

	found := false
	for _, f := range result.Findings {
		if f.CheckID == "cloudtrail-enabled" && f.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cloudtrail-enabled to fail when service is not registered")
	}
}

func TestCloudTrailEnabled(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3", "cloudtrail"))
	result := scanner.Scan()

	found := false
	for _, f := range result.Findings {
		if f.CheckID == "cloudtrail-enabled" && f.Status == "pass" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cloudtrail-enabled to pass when service is registered")
	}
}

func TestConfigRecorderNotEnabled(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3"))
	result := scanner.Scan()

	found := false
	for _, f := range result.Findings {
		if f.CheckID == "config-recorder" && f.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected config-recorder to fail when service is not registered")
	}
}

func TestFindingsByCategory(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3", "iam", "sqs", "sns", "kms"))
	result := scanner.Scan()

	encFindings := FindingsByCategory(result.Findings, scanner.Checks(), "encryption")
	if len(encFindings) == 0 {
		t.Error("expected encryption findings")
	}
	for _, f := range encFindings {
		// All encryption findings should be from encryption-category checks.
		isEnc := false
		for _, c := range scanner.Checks() {
			if c.ID == f.CheckID && c.Category == "encryption" {
				isEnc = true
				break
			}
		}
		if !isEnc {
			t.Errorf("finding %s is not from an encryption check", f.CheckID)
		}
	}
}

func TestSummaryCountsCorrect(t *testing.T) {
	scanner := NewScanner(newTestRegistry("s3", "iam"))
	result := scanner.Scan()

	total := result.Summary.Pass + result.Summary.Fail + result.Summary.Warning
	// Info findings are not counted in pass/fail/warning, so total may be <= Summary.Total
	if total > result.Summary.Total {
		t.Errorf("pass(%d)+fail(%d)+warning(%d) > total(%d)",
			result.Summary.Pass, result.Summary.Fail, result.Summary.Warning, result.Summary.Total)
	}
}
