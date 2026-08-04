package gateway_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// echoService is a minimal service.Service that returns the action name in JSON.
type echoService struct{}

func (e *echoService) Name() string              { return "s3" }
func (e *echoService) Actions() []service.Action { return nil }
func (e *echoService) HealthCheck() error        { return nil }
func (e *echoService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	body := map[string]string{"action": ctx.Action, "service": ctx.Service}
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func newTestGateway(t *testing.T, iamMode string, svcs ...service.Service) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = iamMode

	reg := routing.NewRegistry()
	for _, svc := range svcs {
		reg.Register(svc)
	}

	gw := gateway.New(cfg, reg)
	return gw
}

func newTestGatewayWithRegistry(t *testing.T, iamMode string, reg *routing.Registry) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = iamMode

	gw := gateway.New(cfg, reg)
	return gw
}

func TestGateway_RoutesToService(t *testing.T) {
	handler := newTestGateway(t, "none", &echoService{})

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	// Authorization header with s3 credential scope
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["action"] != "ListBuckets" {
		t.Errorf("expected action=ListBuckets, got action=%q", body["action"])
	}
}

func TestGateway_RoutesAzureARMRequestToVersionedService(t *testing.T) {
	reg := routing.NewRegistry()
	reg.RegisterVersioned(routing.ServiceKey{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.Resources/resourceGroups",
		APIVersion: "2021-04-01",
	}, &echoService{})
	handler := newTestGatewayWithRegistry(t, "none", reg)

	req := httptest.NewRequest(
		http.MethodPut,
		"https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01",
		nil,
	)
	req.Header.Set("Authorization", "Bearer azure-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", resp.StatusCode, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["action"] != "CreateOrUpdate" {
		t.Errorf("expected action=CreateOrUpdate, got action=%q", body["action"])
	}
	if body["service"] != "Microsoft.Resources/resourceGroups" {
		t.Errorf("expected service=Microsoft.Resources/resourceGroups, got service=%q", body["service"])
	}
}

func TestGateway_UnknownService_Returns503(t *testing.T) {
	// No services registered
	handler := newTestGateway(t, "none")

	req := httptest.NewRequest(http.MethodGet, "/?Action=DescribeInstances", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ec2/aws4_request, SignedHeaders=host, Signature=abc123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestGateway_NoService_Returns400(t *testing.T) {
	handler := newTestGateway(t, "none")

	// No Authorization header
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGateway_HealthEndpoint(t *testing.T) {
	handler := newTestGateway(t, "none")

	req := httptest.NewRequest(http.MethodGet, "/_cloudmock/health", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
