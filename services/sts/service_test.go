package sts_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	stssvc "github.com/Viridian-Inc/cloudmock/services/sts"
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

// newSTSGatewayWithIAM builds a gateway with IAM enforce mode for AccessDenied tests.
func newSTSGatewayWithIAM(t *testing.T) (*gateway.Gateway, *iampkg.Store, *iampkg.Engine) {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "enforce"

	store := iampkg.NewStore(cfg.AccountID)
	if err := store.InitRoot("ROOTKEY", "ROOTSECRET"); err != nil {
		t.Fatalf("InitRoot: %v", err)
	}

	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(stssvc.New(cfg.AccountID))

	gw := gateway.NewWithIAM(cfg, reg, store, engine)
	return gw, store, engine
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

// stsReqWithKey builds a form-encoded POST request with a custom access key ID.
func stsReqWithKey(t *testing.T, action string, extra url.Values, accessKeyID string) *http.Request {
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
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+accessKeyID+"/20240101/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc123")
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

func TestSTS_GetCallerIdentity_AccountIDFormat(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Account ID must be a 12-digit number.
	accountRe := regexp.MustCompile(`<Account>(\d{12})</Account>`)
	if !accountRe.MatchString(body) {
		t.Errorf("Account ID should be a 12-digit number in <Account> element\nbody: %s", body)
	}
}

func TestSTS_GetCallerIdentity_ARNFormat(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// ARN should match the standard AWS ARN format for IAM.
	arnRe := regexp.MustCompile(`<Arn>arn:aws:iam::\d{12}:(root|user/\w+)</Arn>`)
	if !arnRe.MatchString(body) {
		t.Errorf("ARN should match arn:aws:iam::<account>:(root|user/<name>) format\nbody: %s", body)
	}
}

func TestSTS_GetCallerIdentity_XMLNamespace(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "https://sts.amazonaws.com/doc/2011-06-15/") {
		t.Errorf("response should contain the STS XML namespace\nbody: %s", body)
	}
}

func TestSTS_GetCallerIdentity_ContentType(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "xml") {
		t.Errorf("Content-Type should contain 'xml', got %q", ct)
	}
}

func TestSTS_GetCallerIdentity_RequestIDFormat(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetCallerIdentity", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// RequestId should be UUID-shaped (8-4-4-4-12 hex).
	uuidRe := regexp.MustCompile(`<RequestId>[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}</RequestId>`)
	if !uuidRe.MatchString(body) {
		t.Errorf("RequestId should be UUID-shaped\nbody: %s", body)
	}
}

func TestSTS_GetCallerIdentity_UniqueRequestIDs(t *testing.T) {
	handler := newSTSGateway(t)

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, stsReq(t, "GetCallerIdentity", nil))

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, stsReq(t, "GetCallerIdentity", nil))

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for both requests, got %d and %d", w1.Code, w2.Code)
	}

	// Extract RequestId from both responses and verify they differ.
	re := regexp.MustCompile(`<RequestId>([^<]+)</RequestId>`)
	m1 := re.FindStringSubmatch(w1.Body.String())
	m2 := re.FindStringSubmatch(w2.Body.String())

	if len(m1) < 2 || len(m2) < 2 {
		t.Fatal("could not extract RequestId from one or both responses")
	}

	if m1[1] == m2[1] {
		t.Errorf("two GetCallerIdentity calls returned the same RequestId: %s", m1[1])
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

func TestSTS_AssumeRole_AssumedRoleArnDerived(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/TestRole")
	extra.Set("RoleSessionName", "test-sess")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// The assumed role ARN should be RoleArn + "/" + RoleSessionName.
	want := "arn:aws:iam::000000000000:role/TestRole/test-sess"
	if !strings.Contains(body, want) {
		t.Errorf("AssumedRoleUser ARN should be %q\nbody: %s", want, body)
	}
}

func TestSTS_AssumeRole_AssumedRoleIDPrefix(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")
	extra.Set("RoleSessionName", "sess1")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// AssumedRoleId should start with AROA prefix and include the session name after ":".
	aroaRe := regexp.MustCompile(`<AssumedRoleId>AROA[0-9a-f]+:sess1</AssumedRoleId>`)
	if !aroaRe.MatchString(body) {
		t.Errorf("AssumedRoleId should start with AROA and end with :sess1\nbody: %s", body)
	}
}

func TestSTS_AssumeRole_CredentialExpiration(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")
	extra.Set("RoleSessionName", "expire-test")

	before := time.Now().UTC()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Extract expiration and verify it is approximately 1 hour in the future.
	expRe := regexp.MustCompile(`<Expiration>([^<]+)</Expiration>`)
	m := expRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract Expiration from response")
	}

	exp, err := time.Parse("2006-01-02T15:04:05Z", m[1])
	if err != nil {
		t.Fatalf("Expiration %q is not valid RFC3339: %v", m[1], err)
	}

	expectedMin := before.Add(55 * time.Minute)
	expectedMax := before.Add(65 * time.Minute)

	if exp.Before(expectedMin) || exp.After(expectedMax) {
		t.Errorf("Expiration %v should be ~1 hour after request time %v", exp, before)
	}
}

func TestSTS_AssumeRole_UniqueCredentials(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")
	extra.Set("RoleSessionName", "unique-test")

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, stsReq(t, "AssumeRole", extra))

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, stsReq(t, "AssumeRole", extra))

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for both, got %d and %d", w1.Code, w2.Code)
	}

	// Each call should produce different AccessKeyId values.
	akRe := regexp.MustCompile(`<AccessKeyId>([^<]+)</AccessKeyId>`)
	m1 := akRe.FindStringSubmatch(w1.Body.String())
	m2 := akRe.FindStringSubmatch(w2.Body.String())

	if len(m1) < 2 || len(m2) < 2 {
		t.Fatal("could not extract AccessKeyId from one or both responses")
	}

	if m1[1] == m2[1] {
		t.Errorf("two AssumeRole calls returned the same AccessKeyId: %s", m1[1])
	}
}

func TestSTS_AssumeRole_MissingBothParams(t *testing.T) {
	handler := newSTSGateway(t)

	// Send AssumeRole with neither RoleArn nor RoleSessionName.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("AssumeRole missing both params: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSTS_AssumeRole_SessionNameInResponse(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/DevOps")
	extra.Set("RoleSessionName", "deploy-session-42")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// The session name should appear in both the assumed role ARN and the assumed role ID.
	if !strings.Contains(body, "deploy-session-42") {
		t.Errorf("response should contain the session name 'deploy-session-42'\nbody: %s", body)
	}
}

func TestSTS_AssumeRole_XMLNamespace(t *testing.T) {
	handler := newSTSGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/R")
	extra.Set("RoleSessionName", "s")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "https://sts.amazonaws.com/doc/2011-06-15/") {
		t.Errorf("AssumeRole response should contain the STS XML namespace\nbody: %s", w.Body.String())
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

func TestSTS_GetSessionToken_CredentialASIAPrefix(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// AccessKeyId must start with ASIA (STS-issued temporary credentials).
	akRe := regexp.MustCompile(`<AccessKeyId>(ASIA[0-9a-f]+)</AccessKeyId>`)
	if !akRe.MatchString(body) {
		t.Errorf("AccessKeyId should start with 'ASIA'\nbody: %s", body)
	}
}

func TestSTS_GetSessionToken_ExpirationInFuture(t *testing.T) {
	handler := newSTSGateway(t)

	before := time.Now().UTC()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	expRe := regexp.MustCompile(`<Expiration>([^<]+)</Expiration>`)
	m := expRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract Expiration from response")
	}

	exp, err := time.Parse("2006-01-02T15:04:05Z", m[1])
	if err != nil {
		t.Fatalf("Expiration %q is not valid RFC3339: %v", m[1], err)
	}

	if !exp.After(before) {
		t.Errorf("Expiration %v should be after request time %v", exp, before)
	}
}

func TestSTS_GetSessionToken_UniqueCredentials(t *testing.T) {
	handler := newSTSGateway(t)

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, stsReq(t, "GetSessionToken", nil))

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, stsReq(t, "GetSessionToken", nil))

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for both, got %d and %d", w1.Code, w2.Code)
	}

	akRe := regexp.MustCompile(`<AccessKeyId>([^<]+)</AccessKeyId>`)
	m1 := akRe.FindStringSubmatch(w1.Body.String())
	m2 := akRe.FindStringSubmatch(w2.Body.String())

	if len(m1) < 2 || len(m2) < 2 {
		t.Fatal("could not extract AccessKeyId from one or both responses")
	}

	if m1[1] == m2[1] {
		t.Errorf("two GetSessionToken calls returned the same AccessKeyId: %s", m1[1])
	}
}

func TestSTS_GetSessionToken_SessionTokenNonEmpty(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// SessionToken must not be empty.
	stRe := regexp.MustCompile(`<SessionToken>([^<]+)</SessionToken>`)
	m := stRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract SessionToken from response")
	}
	if len(m[1]) < 10 {
		t.Errorf("SessionToken should be a substantial random string, got length %d: %s", len(m[1]), m[1])
	}
}

func TestSTS_GetSessionToken_SecretAccessKeyNonEmpty(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	sakRe := regexp.MustCompile(`<SecretAccessKey>([^<]+)</SecretAccessKey>`)
	m := sakRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract SecretAccessKey from response")
	}
	if len(m[1]) < 10 {
		t.Errorf("SecretAccessKey should be a substantial random string, got length %d: %s", len(m[1]), m[1])
	}
}

func TestSTS_GetSessionToken_XMLNamespace(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "GetSessionToken", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "https://sts.amazonaws.com/doc/2011-06-15/") {
		t.Errorf("GetSessionToken response should contain the STS XML namespace\nbody: %s", w.Body.String())
	}
}

// ---- Error cases ----

func TestSTS_UnknownAction(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSTS_UnknownAction_ErrorCodeInBody(t *testing.T) {
	handler := newSTSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "DescribeFoo", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "InvalidAction") {
		t.Errorf("error response should contain 'InvalidAction' code\nbody: %s", body)
	}
}

func TestSTS_EmptyAction(t *testing.T) {
	handler := newSTSGateway(t)

	// Send a request with no Action field at all.
	form := url.Values{}
	form.Set("Version", "2011-06-15")
	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSTS_AssumeRole_ErrorResponseFormat(t *testing.T) {
	handler := newSTSGateway(t)

	// Missing RoleArn should produce a well-formed XML error with proper error code.
	extra := url.Values{}
	extra.Set("RoleSessionName", "my-session")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Error XML should contain Code and Message elements.
	for _, want := range []string{"<Code>", "<Message>", "ValidationError"} {
		if !strings.Contains(body, want) {
			t.Errorf("error body should contain %q\nbody: %s", want, body)
		}
	}
}

func TestSTS_AccessDenied_WithIAMEnforcement(t *testing.T) {
	gw, store, _ := newSTSGatewayWithIAM(t)

	// Create a user with no policies -- should be denied.
	if _, err := store.CreateUser("noperm"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	key, err := store.CreateAccessKey("noperm")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsReqWithKey(t, "GetCallerIdentity", nil, key.AccessKeyID))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AccessDenied, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "AccessDenied") {
		t.Errorf("error response should contain 'AccessDenied'\nbody: %s", body)
	}
}

func TestSTS_AccessDenied_InvalidCredential(t *testing.T) {
	gw, _, _ := newSTSGatewayWithIAM(t)

	// Use a completely unknown access key.
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsReqWithKey(t, "GetCallerIdentity", nil, "AKIAFAKEUNKNOWNKEY"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSTS_RootCanCall_WithIAMEnforcement(t *testing.T) {
	gw, _, _ := newSTSGatewayWithIAM(t)

	// Root credential should always be allowed.
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsReqWithKey(t, "GetCallerIdentity", nil, "ROOTKEY"))

	if w.Code != http.StatusOK {
		t.Fatalf("root should be allowed, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "GetCallerIdentityResponse") {
		t.Errorf("response should be a valid GetCallerIdentityResponse\nbody: %s", body)
	}
}

func TestSTS_AssumeRole_ErrorXMLContainsCode(t *testing.T) {
	handler := newSTSGateway(t)

	// Missing RoleArn.
	extra := url.Values{}
	extra.Set("RoleSessionName", "s")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Verify the error XML contains both <Code> and <Message> elements.
	codeRe := regexp.MustCompile(`<Code>\w+</Code>`)
	msgRe := regexp.MustCompile(`<Message>[^<]+</Message>`)

	if !codeRe.MatchString(body) {
		t.Errorf("error XML should contain a <Code> element\nbody: %s", body)
	}
	if !msgRe.MatchString(body) {
		t.Errorf("error XML should contain a <Message> element\nbody: %s", body)
	}
}

func TestSTS_MultipleActionsSequential(t *testing.T) {
	handler := newSTSGateway(t)

	// Call all three operations sequentially on the same gateway to verify statefulness.
	for _, action := range []string{"GetCallerIdentity", "GetSessionToken"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, stsReq(t, action, nil))

		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d\nbody: %s", action, w.Code, w.Body.String())
		}
	}

	// AssumeRole requires params.
	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/SeqTest")
	extra.Set("RoleSessionName", "seq")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("AssumeRole: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
