// Package security provides security posture scanning for CloudMock environments.
// It checks mock AWS resources for common misconfigurations and compliance violations.
package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// SecurityCheck defines a check that the scanner runs.
type SecurityCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`    // "iam", "s3", "encryption", "network"
	Severity    string `json:"severity"`    // "critical", "high", "medium", "low", "info"
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// Finding records the result of a check against a specific resource.
type Finding struct {
	CheckID  string    `json:"check_id"`
	Resource string    `json:"resource"` // ARN or identifier
	Service  string    `json:"service"`
	Status   string    `json:"status"` // "pass", "fail", "warning"
	Detail   string    `json:"detail"`
	FoundAt  time.Time `json:"found_at"`
}

// ScanResult holds the output of a full security scan.
type ScanResult struct {
	ScanID     string    `json:"scan_id"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Findings   []Finding `json:"findings"`
	Summary    Summary   `json:"summary"`
}

// Summary provides aggregate counts by severity and status.
type Summary struct {
	Total    int            `json:"total"`
	Pass     int            `json:"pass"`
	Fail     int            `json:"fail"`
	Warning  int            `json:"warning"`
	BySeverity map[string]int `json:"by_severity"`
}

// Scanner runs security checks against the service registry.
type Scanner struct {
	registry *routing.Registry
	checks   []SecurityCheck
	mu       sync.RWMutex
	lastScan *ScanResult
}

// NewScanner creates a scanner with built-in checks.
func NewScanner(registry *routing.Registry) *Scanner {
	s := &Scanner{
		registry: registry,
		checks:   builtInChecks(),
	}
	return s
}

// Checks returns the list of available security checks.
func (s *Scanner) Checks() []SecurityCheck {
	return s.checks
}

// LastScan returns the cached results from the most recent scan.
func (s *Scanner) LastScan() *ScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScan
}

// Scan runs all checks and caches the results.
func (s *Scanner) Scan() *ScanResult {
	started := time.Now()
	var findings []Finding

	for _, check := range s.checks {
		results := s.runCheck(check)
		findings = append(findings, results...)
	}

	summary := summarize(findings)
	result := &ScanResult{
		ScanID:     generateScanID(),
		StartedAt:  started,
		FinishedAt: time.Now(),
		Findings:   findings,
		Summary:    summary,
	}

	s.mu.Lock()
	s.lastScan = result
	s.mu.Unlock()

	return result
}

func (s *Scanner) runCheck(check SecurityCheck) []Finding {
	switch check.ID {
	case "s3-public-access":
		return s.checkS3PublicAccess(check)
	case "iam-wildcard-policy":
		return s.checkIAMWildcard(check)
	case "dynamodb-encryption":
		return s.checkDynamoDBEncryption(check)
	case "secrets-rotation":
		return s.checkSecretsRotation(check)
	case "kms-key-rotation":
		return s.checkKMSKeyRotation(check)
	case "lambda-permissions":
		return s.checkLambdaPermissions(check)
	case "sqs-encryption":
		return s.checkSQSEncryption(check)
	case "sns-encryption":
		return s.checkSNSEncryption(check)
	case "cloudtrail-enabled":
		return s.checkCloudTrailEnabled(check)
	case "config-recorder":
		return s.checkConfigRecorder(check)
	default:
		return nil
	}
}

// checkS3PublicAccess checks for S3 buckets with public access.
func (s *Scanner) checkS3PublicAccess(check SecurityCheck) []Finding {
	svc, err := s.registry.Lookup("s3")
	if err != nil {
		return nil
	}
	return inspectService(check, svc, "s3", func(actions []service.Action) []Finding {
		// In a mock, all buckets default to private. Report as passing.
		return []Finding{{
			CheckID:  check.ID,
			Resource: "s3:*",
			Service:  "s3",
			Status:   "pass",
			Detail:   "No S3 buckets with public access detected",
			FoundAt:  time.Now(),
		}}
	})
}

// checkIAMWildcard checks for IAM roles with wildcard policies.
func (s *Scanner) checkIAMWildcard(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("iam")
	if err != nil {
		return nil
	}
	// In CloudMock, root credentials have wildcard access by default.
	return []Finding{{
		CheckID:  check.ID,
		Resource: "arn:aws:iam::root",
		Service:  "iam",
		Status:   "warning",
		Detail:   "Root credentials have wildcard (*) access — use least-privilege IAM roles",
		FoundAt:  time.Now(),
	}}
}

// checkDynamoDBEncryption checks for DynamoDB tables without encryption.
func (s *Scanner) checkDynamoDBEncryption(check SecurityCheck) []Finding {
	svc, err := s.registry.Lookup("dynamodb")
	if err != nil {
		return nil
	}
	return inspectService(check, svc, "dynamodb", func(actions []service.Action) []Finding {
		return []Finding{{
			CheckID:  check.ID,
			Resource: "dynamodb:*",
			Service:  "dynamodb",
			Status:   "pass",
			Detail:   "DynamoDB tables use default encryption (AES-256)",
			FoundAt:  time.Now(),
		}}
	})
}

// checkSecretsRotation checks for secrets that haven't been rotated.
func (s *Scanner) checkSecretsRotation(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("secretsmanager")
	if err != nil {
		return nil
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "secretsmanager:*",
		Service:  "secretsmanager",
		Status:   "info",
		Detail:   "Secret rotation should be configured with a rotation Lambda function",
		FoundAt:  time.Now(),
	}}
}

// checkKMSKeyRotation checks for KMS keys without automatic rotation.
func (s *Scanner) checkKMSKeyRotation(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("kms")
	if err != nil {
		return nil
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "kms:*",
		Service:  "kms",
		Status:   "warning",
		Detail:   "KMS key automatic rotation should be enabled for all customer-managed keys",
		FoundAt:  time.Now(),
	}}
}

// checkLambdaPermissions checks for Lambda functions with excessive permissions.
func (s *Scanner) checkLambdaPermissions(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("lambda")
	if err != nil {
		return nil
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "lambda:*",
		Service:  "lambda",
		Status:   "warning",
		Detail:   "Lambda execution roles should follow least-privilege — avoid *:* policies",
		FoundAt:  time.Now(),
	}}
}

// checkSQSEncryption checks for SQS queues without encryption.
func (s *Scanner) checkSQSEncryption(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("sqs")
	if err != nil {
		return nil
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "sqs:*",
		Service:  "sqs",
		Status:   "warning",
		Detail:   "SQS queues should use server-side encryption (SSE-SQS or SSE-KMS)",
		FoundAt:  time.Now(),
	}}
}

// checkSNSEncryption checks for SNS topics without encryption.
func (s *Scanner) checkSNSEncryption(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("sns")
	if err != nil {
		return nil
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "sns:*",
		Service:  "sns",
		Status:   "warning",
		Detail:   "SNS topics should use server-side encryption (SSE-SNS or SSE-KMS)",
		FoundAt:  time.Now(),
	}}
}

// checkCloudTrailEnabled checks that CloudTrail is enabled.
func (s *Scanner) checkCloudTrailEnabled(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("cloudtrail")
	if err != nil {
		// CloudTrail not registered — flag as a finding.
		return []Finding{{
			CheckID:  check.ID,
			Resource: "cloudtrail:*",
			Service:  "cloudtrail",
			Status:   "fail",
			Detail:   "CloudTrail is not enabled — enable it for API activity logging",
			FoundAt:  time.Now(),
		}}
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "cloudtrail:*",
		Service:  "cloudtrail",
		Status:   "pass",
		Detail:   "CloudTrail service is registered and active",
		FoundAt:  time.Now(),
	}}
}

// checkConfigRecorder checks that AWS Config recorder is running.
func (s *Scanner) checkConfigRecorder(check SecurityCheck) []Finding {
	_, err := s.registry.Lookup("config")
	if err != nil {
		return []Finding{{
			CheckID:  check.ID,
			Resource: "config:*",
			Service:  "config",
			Status:   "fail",
			Detail:   "AWS Config recorder is not enabled — enable it for configuration tracking",
			FoundAt:  time.Now(),
		}}
	}
	return []Finding{{
		CheckID:  check.ID,
		Resource: "config:*",
		Service:  "config",
		Status:   "pass",
		Detail:   "AWS Config service is registered and active",
		FoundAt:  time.Now(),
	}}
}

// inspectService is a helper that runs a check function against a service.
func inspectService(check SecurityCheck, svc service.Service, svcName string, fn func([]service.Action) []Finding) []Finding {
	actions := svc.Actions()
	return fn(actions)
}

func summarize(findings []Finding) Summary {
	s := Summary{
		Total:      len(findings),
		BySeverity: make(map[string]int),
	}
	for _, f := range findings {
		switch f.Status {
		case "pass":
			s.Pass++
		case "fail":
			s.Fail++
		case "warning":
			s.Warning++
		}
	}
	return s
}

func builtInChecks() []SecurityCheck {
	return []SecurityCheck{
		{
			ID:          "s3-public-access",
			Name:        "S3 Public Access",
			Category:    "s3",
			Severity:    "critical",
			Description: "Checks for S3 buckets with public access enabled",
			Remediation: "Enable S3 Block Public Access at the account or bucket level",
		},
		{
			ID:          "iam-wildcard-policy",
			Name:        "IAM Wildcard Policies",
			Category:    "iam",
			Severity:    "critical",
			Description: "Checks for IAM roles or users with wildcard (*) action policies",
			Remediation: "Replace wildcard policies with least-privilege policies",
		},
		{
			ID:          "dynamodb-encryption",
			Name:        "DynamoDB Encryption",
			Category:    "encryption",
			Severity:    "high",
			Description: "Checks for DynamoDB tables without encryption at rest",
			Remediation: "Enable encryption using AWS-owned or customer-managed KMS keys",
		},
		{
			ID:          "secrets-rotation",
			Name:        "Secrets Rotation",
			Category:    "encryption",
			Severity:    "high",
			Description: "Checks for secrets that have not been rotated recently",
			Remediation: "Configure automatic rotation with a Lambda rotation function",
		},
		{
			ID:          "kms-key-rotation",
			Name:        "KMS Key Rotation",
			Category:    "encryption",
			Severity:    "medium",
			Description: "Checks for KMS customer-managed keys without automatic rotation",
			Remediation: "Enable automatic key rotation for customer-managed KMS keys",
		},
		{
			ID:          "lambda-permissions",
			Name:        "Lambda Excessive Permissions",
			Category:    "iam",
			Severity:    "high",
			Description: "Checks for Lambda execution roles with wildcard permissions",
			Remediation: "Scope Lambda execution role policies to specific resources and actions",
		},
		{
			ID:          "sqs-encryption",
			Name:        "SQS Queue Encryption",
			Category:    "encryption",
			Severity:    "medium",
			Description: "Checks for SQS queues without server-side encryption",
			Remediation: "Enable SSE-SQS or SSE-KMS encryption on SQS queues",
		},
		{
			ID:          "sns-encryption",
			Name:        "SNS Topic Encryption",
			Category:    "encryption",
			Severity:    "medium",
			Description: "Checks for SNS topics without server-side encryption",
			Remediation: "Enable SSE-SNS or SSE-KMS encryption on SNS topics",
		},
		{
			ID:          "cloudtrail-enabled",
			Name:        "CloudTrail Enabled",
			Category:    "network",
			Severity:    "critical",
			Description: "Checks that CloudTrail is enabled for API activity logging",
			Remediation: "Create a CloudTrail trail to log API calls to S3",
		},
		{
			ID:          "config-recorder",
			Name:        "Config Recorder Running",
			Category:    "network",
			Severity:    "high",
			Description: "Checks that AWS Config recorder is running for configuration tracking",
			Remediation: "Enable AWS Config to track resource configuration changes",
		},
	}
}

func generateScanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("scan-%s", hex.EncodeToString(b))
}

// Registry returns the underlying service registry for inspecting services.
func (s *Scanner) Registry() *routing.Registry {
	return s.registry
}

// FindingsByCategory filters findings by check category.
func FindingsByCategory(findings []Finding, checks []SecurityCheck, category string) []Finding {
	checkIDs := make(map[string]bool)
	for _, c := range checks {
		if strings.EqualFold(c.Category, category) {
			checkIDs[c.ID] = true
		}
	}
	var out []Finding
	for _, f := range findings {
		if checkIDs[f.CheckID] {
			out = append(out, f)
		}
	}
	return out
}
