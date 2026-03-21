package sts_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	stssvc "github.com/neureaux/cloudmock/services/sts"
)

// newSTSGateway builds a full gateway stack with the STS service registered and IAM disabled.
func newSTSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(stssvc.New(cfg.AccountID))

	return gateway.New(cfg, reg)
}

// stsReq builds a form-encoded POST request targeting the STS service.
func stsReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2011-06-15")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Authorization header places the service in the credential scope so the
	// gateway router can detect "sts" as the target service.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// ---- GetCallerIdentity ----

func TestSTS_GetCallerIdentity(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GetCallerIdentity: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"GetCallerIdentityResponse",
		"GetCallerIdentityResult",
		"<Arn>",
		"<UserId>",
		"<Account>",
		"000000000000",
		"RequestId",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GetCallerIdentity: expected body to contain %q\nbody: %s", want, body)
		}
	}
}

// ---- AssumeRole ----

func TestSTS_AssumeRole(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")
	extra.Set("RoleSessionName", "my-session")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("AssumeRole: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"AssumeRoleResponse",
		"AssumeRoleResult",
		"<Credentials>",
		"<AccessKeyId>",
		"<SecretAccessKey>",
		"<SessionToken>",
		"<Expiration>",
		"<AssumedRoleUser>",
		"<Arn>",
		"my-session",
		"RequestId",
		"ASIA", // ASIA prefix on generated access key
	} {
		if !strings.Contains(body, want) {
			t.Errorf("AssumeRole: expected body to contain %q\nbody: %s", want, body)
		}
	}
}

func TestSTS_AssumeRole_MissingRoleArn(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleSessionName", "my-session")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("AssumeRole missing RoleArn: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSTS_AssumeRole_MissingSessionName(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("AssumeRole missing RoleSessionName: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- GetSessionToken ----

func TestSTS_GetSessionToken(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GetSessionToken: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"GetSessionTokenResponse",
		"GetSessionTokenResult",
		"<Credentials>",
		"<AccessKeyId>",
		"<SecretAccessKey>",
		"<SessionToken>",
		"<Expiration>",
		"RequestId",
		"ASIA",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GetSessionToken: expected body to contain %q\nbody: %s", want, body)
		}
	}
}

// ---- unknown action ----

func TestSTS_UnknownAction(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
