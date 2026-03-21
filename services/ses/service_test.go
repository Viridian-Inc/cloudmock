package ses_test

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	sessvc "github.com/neureaux/cloudmock/services/ses"
)

// newSESGateway builds a full gateway stack with the SES service registered and IAM disabled.
func newSESGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(sessvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// sesReq builds a form-encoded POST request targeting the SES service.
func sesReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-12-01")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ses/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// ---- Test 1: VerifyEmailIdentity + ListIdentities ----

func TestSES_VerifyEmailIdentityAndListIdentities(t *testing.T) {
	handler := newSESGateway(t)

	// Verify two identities.
	for _, email := range []string{"alice@example.com", "bob@example.com"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "VerifyEmailIdentity", url.Values{
			"EmailAddress": {email},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("VerifyEmailIdentity %s: expected 200, got %d\nbody: %s", email, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "VerifyEmailIdentityResponse") {
			t.Errorf("VerifyEmailIdentity: expected response tag\nbody: %s", w.Body.String())
		}
	}

	// ListIdentities should include both.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "ListIdentities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIdentities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"ListIdentitiesResponse",
		"ListIdentitiesResult",
		"alice@example.com",
		"bob@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ListIdentities: expected %q in response\nbody: %s", want, body)
		}
	}

	// ListVerifiedEmailAddresses should return the same set.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, sesReq(t, "ListVerifiedEmailAddresses", nil))
	if wv.Code != http.StatusOK {
		t.Fatalf("ListVerifiedEmailAddresses: expected 200, got %d\nbody: %s", wv.Code, wv.Body.String())
	}
	bodyV := wv.Body.String()
	if !strings.Contains(bodyV, "alice@example.com") {
		t.Errorf("ListVerifiedEmailAddresses: expected alice@example.com\nbody: %s", bodyV)
	}
}

// ---- Test 2: SendEmail + verify MessageId returned ----

func TestSES_SendEmail(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                          {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":            {"Hello from cloudmock"},
		"Message.Body.Text.Data":          {"This is the text body."},
		"Message.Body.Html.Data":          {"<p>This is the HTML body.</p>"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"SendEmailResponse",
		"SendEmailResult",
		"MessageId",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("SendEmail: expected %q in response\nbody: %s", want, body)
		}
	}

	// Extract and verify MessageId is non-empty.
	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Errorf("SendEmail: MessageId is empty\nbody: %s", body)
	}

	// SendEmail without Source should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, sesReq(t, "SendEmail", url.Values{
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":            {"No source"},
	}))
	if wf.Code == http.StatusOK {
		t.Error("SendEmail without Source: expected error, got 200")
	}

	// SendEmail without recipients should fail.
	wf2 := httptest.NewRecorder()
	handler.ServeHTTP(wf2, sesReq(t, "SendEmail", url.Values{
		"Source":               {"sender@example.com"},
		"Message.Subject.Data": {"No recipients"},
	}))
	if wf2.Code == http.StatusOK {
		t.Error("SendEmail without recipients: expected error, got 200")
	}
}

// ---- Test 3: DeleteIdentity ----

func TestSES_DeleteIdentity(t *testing.T) {
	handler := newSESGateway(t)

	// Add an identity.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {"delete-me@example.com"},
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("VerifyEmailIdentity: expected 200, got %d", wv.Code)
	}

	// Confirm it is listed.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, sesReq(t, "ListIdentities", nil))
	if !strings.Contains(wl.Body.String(), "delete-me@example.com") {
		t.Fatalf("ListIdentities before delete: identity not found\nbody: %s", wl.Body.String())
	}

	// Delete it.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sesReq(t, "DeleteIdentity", url.Values{
		"Identity": {"delete-me@example.com"},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteIdentity: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "DeleteIdentityResponse") {
		t.Errorf("DeleteIdentity: expected response tag\nbody: %s", wd.Body.String())
	}

	// Should no longer appear in the list.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, sesReq(t, "ListIdentities", nil))
	if strings.Contains(wl2.Body.String(), "delete-me@example.com") {
		t.Errorf("ListIdentities after delete: identity should not appear\nbody: %s", wl2.Body.String())
	}

	// DeleteIdentity with missing param should fail.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, sesReq(t, "DeleteIdentity", nil))
	if we.Code == http.StatusOK {
		t.Error("DeleteIdentity without Identity: expected error, got 200")
	}
}

// ---- Test 4: GetIdentityVerificationAttributes ----

func TestSES_GetIdentityVerificationAttributes(t *testing.T) {
	handler := newSESGateway(t)

	// Verify an identity first.
	handler.ServeHTTP(httptest.NewRecorder(), sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {"verify-attr@example.com"},
	}))

	// GetIdentityVerificationAttributes for two addresses (one verified, one not — both return Success in cloudmock).
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "GetIdentityVerificationAttributes", url.Values{
		"Identities.member.1": {"verify-attr@example.com"},
		"Identities.member.2": {"unverified@example.com"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIdentityVerificationAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"GetIdentityVerificationAttributesResponse",
		"GetIdentityVerificationAttributesResult",
		"VerificationAttributes",
		"verify-attr@example.com",
		"unverified@example.com",
		"Success",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GetIdentityVerificationAttributes: expected %q in response\nbody: %s", want, body)
		}
	}
}

// ---- Test 5: SendRawEmail ----

func TestSES_SendRawEmail(t *testing.T) {
	handler := newSESGateway(t)

	raw := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Raw test\r\n\r\nThis is the raw body."
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendRawEmail", url.Values{
		"RawMessage.Data": {encoded},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendRawEmail: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"SendRawEmailResponse",
		"SendRawEmailResult",
		"MessageId",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("SendRawEmail: expected %q in response\nbody: %s", want, body)
		}
	}

	// Extract and verify MessageId is non-empty.
	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendRawEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendRawEmail: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Errorf("SendRawEmail: MessageId is empty\nbody: %s", body)
	}

	// SendRawEmail without data should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, sesReq(t, "SendRawEmail", nil))
	if wf.Code == http.StatusOK {
		t.Error("SendRawEmail without data: expected error, got 200")
	}

	// SendRawEmail with invalid base64 should fail.
	wi := httptest.NewRecorder()
	handler.ServeHTTP(wi, sesReq(t, "SendRawEmail", url.Values{
		"RawMessage.Data": {"not-valid-base64!!!"},
	}))
	if wi.Code == http.StatusOK {
		t.Error("SendRawEmail with invalid base64: expected error, got 200")
	}
}

// ---- Test 6: Unknown action ----

func TestSES_UnknownAction(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
