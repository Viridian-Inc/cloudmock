package gateway_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func testModeIdentity() *service.CallerIdentity {
	return &service.CallerIdentity{
		AccountID:   "123456789012",
		ARN:         "arn:aws:iam::123456789012:root",
		UserID:      "123456789012",
		AccessKeyID: "test",
		IsRoot:      true,
	}
}

func azureEchoRegistry() *routing.Registry {
	reg := routing.NewRegistry()
	reg.RegisterVersioned(routing.ServiceKey{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.Resources/resourceGroups",
		APIVersion: "2021-04-01",
	}, &echoService{})
	return reg
}

func TestModeHandlerRoutesAzureARMRequest(t *testing.T) {
	handler := gateway.TestModeHandler(testModeIdentity(), "us-east-1", "123456789012", azureEchoRegistry())

	req := httptest.NewRequest(http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil)
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
	if body["service"] != "Microsoft.Resources/resourceGroups" {
		t.Errorf("expected Azure service in context, got %q", body["service"])
	}
}

func TestFastTestModeServerRoutesAzureARMRequest(t *testing.T) {
	server := gateway.FastTestModeServer(testModeIdentity(), "us-east-1", "123456789012", azureEchoRegistry())
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ln)
	}()
	defer func() {
		_ = server.Shutdown()
		if err := <-errCh; err != nil {
			t.Fatalf("fast server returned error: %v", err)
		}
	}()

	client := &fasthttp.Client{
		Dial: func(_ string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(http.MethodPut)
	req.SetRequestURI("http://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01")
	req.Header.Set("Authorization", "Bearer azure-token")

	if err := client.Do(req, resp); err != nil {
		t.Fatalf("fast client request failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", resp.StatusCode(), string(resp.Body()))
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["service"] != "Microsoft.Resources/resourceGroups" {
		t.Errorf("expected Azure service in context, got %q", body["service"])
	}
}
