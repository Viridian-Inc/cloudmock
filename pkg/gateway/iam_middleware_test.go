package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// allowAllService is a service.Service that always returns 200.
type allowAllService struct{}

func (a *allowAllService) Name() string              { return "s3" }
func (a *allowAllService) Actions() []service.Action { return nil }
func (a *allowAllService) HealthCheck() error        { return nil }
func (a *allowAllService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{}`),
		Format:     service.FormatJSON,
	}, nil
}

func newIAMGateway(t *testing.T, iamMode string, store *iampkg.Store, engine *iampkg.Engine, svcs ...service.Service) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = iamMode

	reg := routing.NewRegistry()
	for _, svc := range svcs {
		reg.Register(svc)
	}

	gw := gateway.NewWithIAM(cfg, reg, store, engine)
	return gw
}

func s3AuthHeader(keyID string) string {
	return "AWS4-HMAC-SHA256 Credential=" + keyID + "/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc123"
}

func TestIAMEnforce_RootAllowed(t *testing.T) {
	store := iampkg.NewStore("123456789012")
	if err := store.InitRoot("ROOTKEY", "ROOTSECRET"); err != nil {
		t.Fatalf("InitRoot: %v", err)
	}
	engine := iampkg.NewEngine()

	handler := newIAMGateway(t, "enforce", store, engine, &allowAllService{})

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", s3AuthHeader("ROOTKEY"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestIAMEnforce_UnknownKeyDenied(t *testing.T) {
	store := iampkg.NewStore("123456789012")
	if err := store.InitRoot("ROOTKEY", "ROOTSECRET"); err != nil {
		t.Fatalf("InitRoot: %v", err)
	}
	engine := iampkg.NewEngine()

	handler := newIAMGateway(t, "enforce", store, engine, &allowAllService{})

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", s3AuthHeader("UNKNOWN"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestIAMNone_BypassesAuth(t *testing.T) {
	handler := newTestGateway(t, "none", &allowAllService{})

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", s3AuthHeader("anything"))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}
