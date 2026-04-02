package integration_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	ec2svc "github.com/neureaux/cloudmock/services/ec2"
)

// setupEC2Gateway builds a full gateway with the Tier 1 EC2 service registered
// plus all Tier 2 stubs, mirroring the production gateway setup.
func setupEC2Gateway(t *testing.T) *httptest.Server {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none"

	store := iampkg.NewStore(cfg.AccountID)
	_ = store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	registry := routing.NewRegistry()

	// Register Tier 1 EC2 service.
	registry.Register(ec2svc.New(cfg.AccountID, cfg.Region))

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	return httptest.NewServer(gw)
}

// ec2IntegReq builds a form-encoded EC2 POST request for the integration server.
func ec2IntegReq(t *testing.T, srv *httptest.Server, action string, extra url.Values) *http.Response {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ec2/aws4_request, SignedHeaders=host, Signature=abc123")
	req.Header.Set("Host", "ec2.amazonaws.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.Do: %v", err)
	}
	return resp
}

// TestEC2Integration_AccessibleThroughGateway verifies EC2 is reachable through
// the gateway with all services registered and returns sensible responses.
func TestEC2Integration_AccessibleThroughGateway(t *testing.T) {
	srv := setupEC2Gateway(t)
	defer srv.Close()

	resp := ec2IntegReq(t, srv, "DescribeVpcs", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DescribeVpcs: expected 200, got %d", resp.StatusCode)
	}
}

// TestEC2Integration_DefaultVPCPresentOnStartup verifies the default VPC and its
// resources are available immediately after the gateway starts.
func TestEC2Integration_DefaultVPCPresentOnStartup(t *testing.T) {
	srv := setupEC2Gateway(t)
	defer srv.Close()

	// Default VPC should exist.
	resp := ec2IntegReq(t, srv, "DescribeVpcs", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DescribeVpcs: expected 200, got %d", resp.StatusCode)
	}

	buf := new(strings.Builder)
	io2 := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(io2)
		buf.Write(io2[:n])
		if err != nil {
			break
		}
	}
	body := buf.String()

	if !strings.Contains(body, "172.31.0.0/16") {
		t.Errorf("DescribeVpcs: expected default VPC CIDR 172.31.0.0/16\nbody: %s", body)
	}
	if !strings.Contains(body, "<isDefault>true</isDefault>") {
		t.Errorf("DescribeVpcs: expected isDefault=true\nbody: %s", body)
	}
}

// TestEC2Integration_TierOneOverridesStub verifies the Tier 1 EC2 service is
// serving requests (not the Tier 2 stub) by checking for rich XML responses.
func TestEC2Integration_TierOneOverridesStub(t *testing.T) {
	srv := setupEC2Gateway(t)
	defer srv.Close()

	// Create a VPC — the Tier 1 service returns a proper EC2 XML response including
	// dhcpOptionsId, which the stub does not produce.
	resp := ec2IntegReq(t, srv, "CreateVpc", url.Values{
		"CidrBlock": {"10.0.0.0/16"},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("CreateVpc: expected 200, got %d", resp.StatusCode)
	}

	buf := new(strings.Builder)
	io2 := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(io2)
		buf.Write(io2[:n])
		if err != nil {
			break
		}
	}
	body := buf.String()

	// The Tier 1 service produces dhcpOptionsId; the stub does not.
	if !strings.Contains(body, "dopt-") {
		t.Errorf("CreateVpc: expected dhcpOptionsId (Tier 1 response), got\nbody: %s", body)
	}
	if !strings.Contains(body, "<state>available</state>") {
		t.Errorf("CreateVpc: expected state=available\nbody: %s", body)
	}
}
