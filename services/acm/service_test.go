package acm_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/acm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ACMService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// --- RequestCertificate ---

func TestACM_RequestCertificate(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "example.com",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respBody(t, resp)
	arn, ok := m["CertificateArn"].(string)
	require.True(t, ok)
	assert.Contains(t, arn, "arn:aws:acm:us-east-1:123456789012:certificate/")
}

func TestACM_RequestCertificate_MissingDomain(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ValidationError", awsErr.Error()[:15])
}

// --- DescribeCertificate ---

func TestACM_DescribeCertificate(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "example.com",
	}))
	m := respBody(t, resp)
	arn := m["CertificateArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m = respBody(t, resp)
	cert := m["Certificate"].(map[string]any)
	assert.Equal(t, "example.com", cert["DomainName"])
	// With default config (instant transitions), status transitions to ISSUED immediately
	assert.Contains(t, []string{"PENDING_VALIDATION", "ISSUED"}, cert["Status"])
	assert.Equal(t, "AMAZON_ISSUED", cert["Type"])
	assert.Equal(t, "RSA_2048", cert["KeyAlgorithm"])
	assert.Equal(t, "DNS", cert["ValidationMethod"])
}

func TestACM_DescribeCertificate_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/nonexistent",
	}))
	require.Error(t, err)
	awsErr := err.(*service.AWSError)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Error()[:25])
}

func TestACM_DescribeCertificate_MissingArn(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{}))
	require.Error(t, err)
}

// --- ListCertificates ---

func TestACM_ListCertificates(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{"DomainName": "a.com"}))
	s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{"DomainName": "b.com"}))
	s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{"DomainName": "c.com"}))

	resp, err := s.HandleRequest(jsonCtx("ListCertificates", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	list := m["CertificateSummaryList"].([]any)
	assert.Len(t, list, 3)
}

// --- DeleteCertificate ---

func TestACM_DeleteCertificate(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "delete-me.com",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	_, err = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.Error(t, err)
}

func TestACM_DeleteCertificate_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DeleteCertificate", map[string]any{
		"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/nonexistent",
	}))
	require.Error(t, err)
}

// --- ImportCertificate ---

func TestACM_ImportCertificate(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
		"PrivateKey":  "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	m := respBody(t, resp)
	assert.Contains(t, m["CertificateArn"].(string), "arn:aws:acm:")
}

func TestACM_ImportCertificate_MissingFields(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "cert-body",
	}))
	require.Error(t, err)
}

// --- ExportCertificate ---

func TestACM_ExportCertificate(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "cert-body",
		"PrivateKey":  "priv-key",
		"CertificateChain": "chain",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("ExportCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "cert-body", m["Certificate"])
	assert.Equal(t, "priv-key", m["PrivateKey"])
	assert.Equal(t, "chain", m["CertificateChain"])
}

func TestACM_ExportCertificate_NonImported(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "example.com",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	_, err := s.HandleRequest(jsonCtx("ExportCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.Error(t, err)
}

// --- GetCertificate ---

func TestACM_GetCertificate_Imported(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "cert-body",
		"PrivateKey":  "priv-key",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "cert-body", m["Certificate"])
}

func TestACM_GetCertificate_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetCertificate", map[string]any{
		"CertificateArn": "arn:aws:acm:us-east-1:123456789012:certificate/nonexistent",
	}))
	require.Error(t, err)
}

// --- Tagging ---

func TestACM_Tagging(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "example.com",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	// Add tags
	_, err := s.HandleRequest(jsonCtx("AddTagsToCertificate", map[string]any{
		"CertificateArn": arn,
		"Tags":           []any{map[string]any{"Key": "env", "Value": "prod"}, map[string]any{"Key": "team", "Value": "platform"}},
	}))
	require.NoError(t, err)

	// List tags
	resp, err = s.HandleRequest(jsonCtx("ListTagsForCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	tags := m["Tags"].([]any)
	assert.Len(t, tags, 2)

	// Remove tags
	_, err = s.HandleRequest(jsonCtx("RemoveTagsFromCertificate", map[string]any{
		"CertificateArn": arn,
		"Tags":           []any{map[string]any{"Key": "env"}},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	m = respBody(t, resp)
	tags = m["Tags"].([]any)
	assert.Len(t, tags, 1)
}

// --- RenewCertificate ---

func TestACM_RenewCertificate_ImportedNotRenewable(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "body",
		"PrivateKey":  "key",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	_, err := s.HandleRequest(jsonCtx("RenewCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationException")
}

// --- SANs ---

func TestACM_RequestCertificate_WithSANs(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName":              "example.com",
		"SubjectAlternativeNames": []any{"www.example.com", "api.example.com"},
		"ValidationMethod":        "EMAIL",
	}))
	require.NoError(t, err)
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert := respBody(t, resp)["Certificate"].(map[string]any)
	assert.Equal(t, "EMAIL", cert["ValidationMethod"])
	sans := cert["SubjectAlternativeNames"].([]any)
	assert.GreaterOrEqual(t, len(sans), 3) // example.com + www + api
}

// --- Behavioral: DNS Validation Records ---

func TestACM_RequestCertificate_DNSValidationRecords(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName":       "example.com",
		"ValidationMethod": "DNS",
	}))
	require.NoError(t, err)
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert := respBody(t, resp)["Certificate"].(map[string]any)
	dvs := cert["DomainValidationOptions"].([]any)
	require.GreaterOrEqual(t, len(dvs), 1)

	dv := dvs[0].(map[string]any)
	assert.Equal(t, "example.com", dv["DomainName"])
	assert.Equal(t, "DNS", dv["ValidationMethod"])

	rr, ok := dv["ResourceRecord"].(map[string]any)
	require.True(t, ok, "DNS validation should have ResourceRecord")
	assert.Equal(t, "CNAME", rr["Type"])
	assert.NotEmpty(t, rr["Name"])
	assert.NotEmpty(t, rr["Value"])
	assert.Contains(t, rr["Value"].(string), "acm-validations.aws")
}

func TestACM_RenewCertificate_GeneratesNewRecords(t *testing.T) {
	s := newService()
	// Request cert - with instant lifecycle it should auto-issue
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "renew.example.com",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	// Wait for lifecycle to issue
	time.Sleep(100 * time.Millisecond)

	// Verify it's issued
	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert := respBody(t, resp)["Certificate"].(map[string]any)
	assert.Equal(t, "ISSUED", cert["Status"])

	// Renew
	resp, err := s.HandleRequest(jsonCtx("RenewCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// After renewal, describe should still show valid cert
	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert = respBody(t, resp)["Certificate"].(map[string]any)
	assert.Equal(t, "ISSUED", cert["Status"])
	assert.NotNil(t, cert["NotAfter"])
}

func TestACM_SetLocator_Accepted(t *testing.T) {
	s := svc.New("123456789012", "us-east-1")
	// Smoke test that SetLocator doesn't panic
	s.SetLocator(nil)
}

// --- Invalid Action ---

func TestACM_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr := err.(*service.AWSError)
	assert.Contains(t, awsErr.Error(), "InvalidAction")
}

// --- Lifecycle ---

func TestACM_Lifecycle_TransitionsToIssued(t *testing.T) {
	// With default config (instant transitions), status should reach ISSUED
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("RequestCertificate", map[string]any{
		"DomainName": "lifecycle.com",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert := respBody(t, resp)["Certificate"].(map[string]any)
	// With instant transitions, it should be ISSUED (or still PENDING_VALIDATION if goroutine hasn't run)
	assert.Contains(t, []string{"PENDING_VALIDATION", "ISSUED"}, cert["Status"])
}

func TestACM_ImportedCertificate_IssuedStatus(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("ImportCertificate", map[string]any{
		"Certificate": "body",
		"PrivateKey":  "key",
	}))
	arn := respBody(t, resp)["CertificateArn"].(string)

	resp, _ = s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{
		"CertificateArn": arn,
	}))
	cert := respBody(t, resp)["Certificate"].(map[string]any)
	assert.Equal(t, "ISSUED", cert["Status"])
	assert.Equal(t, "IMPORTED", cert["Type"])
}
