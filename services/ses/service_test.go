package ses_test

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	sessvc "github.com/Viridian-Inc/cloudmock/services/ses"
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

// ---- Test 7: SendEmail — HTML-only body (no text body) ----

func TestSES_SendEmail_HtmlOnlyBody(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":             {"HTML Only"},
		"Message.Body.Html.Data":           {"<h1>Hello</h1><p>HTML only message.</p>"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail HTML-only: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail HTML-only: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendEmail HTML-only: MessageId should not be empty")
	}
}

// ---- Test 8: SendEmail — text-only body (no HTML) ----

func TestSES_SendEmail_TextOnlyBody(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":             {"Text Only"},
		"Message.Body.Text.Data":           {"Plain text body only."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail text-only: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail text-only: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendEmail text-only: MessageId should not be empty")
	}
}

// ---- Test 9: SendEmail — CC and BCC recipients ----

func TestSES_SendEmail_CcAndBcc(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                            {"sender@example.com"},
		"Destination.ToAddresses.member.1":  {"to@example.com"},
		"Destination.CcAddresses.member.1":  {"cc1@example.com"},
		"Destination.CcAddresses.member.2":  {"cc2@example.com"},
		"Destination.BccAddresses.member.1": {"bcc@example.com"},
		"Message.Subject.Data":              {"CC and BCC test"},
		"Message.Body.Text.Data":            {"Body with CC and BCC."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail CC/BCC: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "SendEmailResponse") {
		t.Errorf("SendEmail CC/BCC: expected SendEmailResponse in body\nbody: %s", body)
	}
	if !strings.Contains(body, "MessageId") {
		t.Errorf("SendEmail CC/BCC: expected MessageId in body\nbody: %s", body)
	}
}

// ---- Test 10: SendEmail — CC-only recipient (no To) ----

func TestSES_SendEmail_CcOnly(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.CcAddresses.member.1": {"cc@example.com"},
		"Message.Subject.Data":             {"CC only"},
		"Message.Body.Text.Data":           {"CC-only recipient."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail CC-only: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail CC-only: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendEmail CC-only: MessageId should not be empty")
	}
}

// ---- Test 11: SendEmail — BCC-only recipient (no To or CC) ----

func TestSES_SendEmail_BccOnly(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                            {"sender@example.com"},
		"Destination.BccAddresses.member.1": {"bcc@example.com"},
		"Message.Subject.Data":              {"BCC only"},
		"Message.Body.Text.Data":            {"BCC-only recipient."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail BCC-only: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail BCC-only: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendEmail BCC-only: MessageId should not be empty")
	}
}

// ---- Test 12: SendEmail — multiple To recipients ----

func TestSES_SendEmail_MultipleToRecipients(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"one@example.com"},
		"Destination.ToAddresses.member.2": {"two@example.com"},
		"Destination.ToAddresses.member.3": {"three@example.com"},
		"Message.Subject.Data":             {"Multiple recipients"},
		"Message.Body.Text.Data":           {"Sent to three recipients."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail multiple To: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendEmail multiple To: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendEmail multiple To: MessageId should not be empty")
	}
}

// ---- Test 13: SendEmail — unique MessageId per call ----

func TestSES_SendEmail_UniqueMessageIds(t *testing.T) {
	handler := newSESGateway(t)

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
			"Source":                           {"sender@example.com"},
			"Destination.ToAddresses.member.1": {"recipient@example.com"},
			"Message.Subject.Data":             {"Uniqueness test"},
			"Message.Body.Text.Data":           {"Body."},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("SendEmail iteration %d: expected 200, got %d", i, w.Code)
		}

		var resp struct {
			Result struct {
				MessageId string `xml:"MessageId"`
			} `xml:"SendEmailResult"`
		}
		if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("SendEmail iteration %d: unmarshal: %v", i, err)
		}
		if ids[resp.Result.MessageId] {
			t.Errorf("SendEmail: duplicate MessageId %q on iteration %d", resp.Result.MessageId, i)
		}
		ids[resp.Result.MessageId] = true
	}
}

// ---- Test 14: SendEmail — missing subject succeeds (SES allows it) ----

func TestSES_SendEmail_NoSubject(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Body.Text.Data":           {"No subject provided."},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail no subject: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 15: VerifyEmailIdentity — idempotent re-verify ----

func TestSES_VerifyEmailIdentity_Idempotent(t *testing.T) {
	handler := newSESGateway(t)

	email := "idempotent@example.com"

	// Verify the same identity twice.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "VerifyEmailIdentity", url.Values{
			"EmailAddress": {email},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("VerifyEmailIdentity attempt %d: expected 200, got %d\nbody: %s", i+1, w.Code, w.Body.String())
		}
	}

	// Should appear only once in ListIdentities.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "ListIdentities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIdentities: expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	count := strings.Count(body, email)
	if count != 1 {
		t.Errorf("VerifyEmailIdentity idempotent: expected %q to appear once, appeared %d times\nbody: %s", email, count, body)
	}
}

// ---- Test 16: VerifyEmailIdentity — missing EmailAddress ----

func TestSES_VerifyEmailIdentity_MissingEmail(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "VerifyEmailIdentity", nil))

	if w.Code == http.StatusOK {
		t.Error("VerifyEmailIdentity without EmailAddress: expected error, got 200")
	}
}

// ---- Test 17: VerifyEmailIdentity — verify then delete then re-verify ----

func TestSES_VerifyEmailIdentity_StateTransitions(t *testing.T) {
	handler := newSESGateway(t)

	email := "transition@example.com"

	// Step 1: Verify.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {email},
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("Verify step 1: expected 200, got %d", w1.Code)
	}

	// Step 2: Confirm listed.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, sesReq(t, "ListIdentities", nil))
	if !strings.Contains(wl.Body.String(), email) {
		t.Fatalf("after verify: identity should be listed")
	}

	// Step 3: Delete.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sesReq(t, "DeleteIdentity", url.Values{
		"Identity": {email},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("Delete: expected 200, got %d", wd.Code)
	}

	// Step 4: Confirm not listed.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, sesReq(t, "ListIdentities", nil))
	if strings.Contains(wl2.Body.String(), email) {
		t.Fatalf("after delete: identity should not be listed")
	}

	// Step 5: Re-verify.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {email},
	}))
	if w2.Code != http.StatusOK {
		t.Fatalf("Re-verify: expected 200, got %d", w2.Code)
	}

	// Step 6: Confirm listed again.
	wl3 := httptest.NewRecorder()
	handler.ServeHTTP(wl3, sesReq(t, "ListIdentities", nil))
	if !strings.Contains(wl3.Body.String(), email) {
		t.Errorf("after re-verify: identity should be listed\nbody: %s", wl3.Body.String())
	}
}

// ---- Test 18: ListIdentities — empty store returns empty list ----

func TestSES_ListIdentities_Empty(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "ListIdentities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIdentities empty: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "ListIdentitiesResponse") {
		t.Errorf("ListIdentities empty: expected ListIdentitiesResponse tag\nbody: %s", body)
	}
	if !strings.Contains(body, "ListIdentitiesResult") {
		t.Errorf("ListIdentities empty: expected ListIdentitiesResult tag\nbody: %s", body)
	}
	// Should have no <member> elements.
	if strings.Contains(body, "<member>") {
		t.Errorf("ListIdentities empty: expected no <member> elements\nbody: %s", body)
	}
}

// ---- Test 19: ListIdentities — many identities ----

func TestSES_ListIdentities_ManyIdentities(t *testing.T) {
	handler := newSESGateway(t)

	emails := make([]string, 20)
	for i := range emails {
		emails[i] = fmt.Sprintf("user%d@example.com", i)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "VerifyEmailIdentity", url.Values{
			"EmailAddress": {emails[i]},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("VerifyEmailIdentity %s: expected 200, got %d", emails[i], w.Code)
		}
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "ListIdentities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIdentities: expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	for _, email := range emails {
		if !strings.Contains(body, email) {
			t.Errorf("ListIdentities: expected %q in response\nbody: %s", email, body)
		}
	}
}

// ---- Test 20: ListVerifiedEmailAddresses — empty store ----

func TestSES_ListVerifiedEmailAddresses_Empty(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "ListVerifiedEmailAddresses", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListVerifiedEmailAddresses empty: expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "ListVerifiedEmailAddressesResponse") {
		t.Errorf("expected ListVerifiedEmailAddressesResponse tag\nbody: %s", body)
	}
	if strings.Contains(body, "<member>") {
		t.Errorf("expected no <member> elements in empty list\nbody: %s", body)
	}
}

// ---- Test 21: ListVerifiedEmailAddresses — matches ListIdentities ----

func TestSES_ListVerifiedEmailAddresses_MatchesListIdentities(t *testing.T) {
	handler := newSESGateway(t)

	emails := []string{"alpha@example.com", "beta@example.com", "gamma@example.com"}
	for _, email := range emails {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "VerifyEmailIdentity", url.Values{
			"EmailAddress": {email},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("VerifyEmailIdentity %s: expected 200, got %d", email, w.Code)
		}
	}

	// ListIdentities
	wli := httptest.NewRecorder()
	handler.ServeHTTP(wli, sesReq(t, "ListIdentities", nil))
	if wli.Code != http.StatusOK {
		t.Fatalf("ListIdentities: expected 200, got %d", wli.Code)
	}

	// ListVerifiedEmailAddresses
	wlv := httptest.NewRecorder()
	handler.ServeHTTP(wlv, sesReq(t, "ListVerifiedEmailAddresses", nil))
	if wlv.Code != http.StatusOK {
		t.Fatalf("ListVerifiedEmailAddresses: expected 200, got %d", wlv.Code)
	}

	// Both should contain all three emails.
	for _, email := range emails {
		if !strings.Contains(wli.Body.String(), email) {
			t.Errorf("ListIdentities: missing %q", email)
		}
		if !strings.Contains(wlv.Body.String(), email) {
			t.Errorf("ListVerifiedEmailAddresses: missing %q", email)
		}
	}
}

// ---- Test 22: DeleteIdentity — deleting nonexistent identity still succeeds ----

func TestSES_DeleteIdentity_Nonexistent(t *testing.T) {
	handler := newSESGateway(t)

	// AWS SES DeleteIdentity is idempotent; it succeeds even when the identity does not exist.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "DeleteIdentity", url.Values{
		"Identity": {"does-not-exist@example.com"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIdentity nonexistent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "DeleteIdentityResponse") {
		t.Errorf("DeleteIdentity nonexistent: expected response tag\nbody: %s", w.Body.String())
	}
}

// ---- Test 23: GetIdentityVerificationAttributes — empty request ----

func TestSES_GetIdentityVerificationAttributes_NoIdentities(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "GetIdentityVerificationAttributes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIdentityVerificationAttributes no identities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "GetIdentityVerificationAttributesResponse") {
		t.Errorf("expected response tag\nbody: %s", body)
	}
}

// ---- Test 24: GetIdentityVerificationAttributes — all identities return Success ----

func TestSES_GetIdentityVerificationAttributes_AllReturnSuccess(t *testing.T) {
	handler := newSESGateway(t)

	// Verify one identity.
	handler.ServeHTTP(httptest.NewRecorder(), sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {"verified@example.com"},
	}))

	// Query both a verified and an unregistered identity.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "GetIdentityVerificationAttributes", url.Values{
		"Identities.member.1": {"verified@example.com"},
		"Identities.member.2": {"unknown@example.com"},
		"Identities.member.3": {"another@example.com"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// All three should have "Success" status (cloudmock auto-verifies).
	expectedCount := strings.Count(body, "Success")
	if expectedCount < 3 {
		t.Errorf("expected at least 3 Success entries, found %d\nbody: %s", expectedCount, body)
	}
	for _, addr := range []string{"verified@example.com", "unknown@example.com", "another@example.com"} {
		if !strings.Contains(body, addr) {
			t.Errorf("expected %q in response\nbody: %s", addr, body)
		}
	}
}

// ---- Test 25: SendRawEmail — large payload succeeds ----

func TestSES_SendRawEmail_LargePayload(t *testing.T) {
	handler := newSESGateway(t)

	// Build a raw email with a larger body.
	largeBody := strings.Repeat("X", 10000)
	raw := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Large\r\n\r\n" + largeBody
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendRawEmail", url.Values{
		"RawMessage.Data": {encoded},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendRawEmail large: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendRawEmailResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("SendRawEmail large: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("SendRawEmail large: MessageId should not be empty")
	}
}

// ---- Test 26: SendRawEmail — unique MessageIds ----

func TestSES_SendRawEmail_UniqueMessageIds(t *testing.T) {
	handler := newSESGateway(t)

	raw := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Unique\r\n\r\nBody."
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, "SendRawEmail", url.Values{
			"RawMessage.Data": {encoded},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("SendRawEmail iteration %d: expected 200, got %d", i, w.Code)
		}

		var resp struct {
			Result struct {
				MessageId string `xml:"MessageId"`
			} `xml:"SendRawEmailResult"`
		}
		if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("SendRawEmail iteration %d: unmarshal: %v", i, err)
		}
		if ids[resp.Result.MessageId] {
			t.Errorf("SendRawEmail: duplicate MessageId %q on iteration %d", resp.Result.MessageId, i)
		}
		ids[resp.Result.MessageId] = true
	}
}

// ---- Test 27: SendEmail — missing subject and body still works (source + recipient is minimum) ----

func TestSES_SendEmail_MinimalParams(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"sender@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":             {""},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("SendEmail minimal: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 28: Response contains RequestId metadata ----

func TestSES_ResponseContainsRequestId(t *testing.T) {
	handler := newSESGateway(t)

	actions := []struct {
		name   string
		params url.Values
	}{
		{"ListIdentities", nil},
		{"ListVerifiedEmailAddresses", nil},
		{"VerifyEmailIdentity", url.Values{"EmailAddress": {"reqid@example.com"}}},
		{"SendEmail", url.Values{
			"Source":                           {"sender@example.com"},
			"Destination.ToAddresses.member.1": {"recipient@example.com"},
			"Message.Subject.Data":             {"Test"},
			"Message.Body.Text.Data":           {"Body."},
		}},
	}

	for _, tc := range actions {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sesReq(t, tc.name, tc.params))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.name, w.Code)
		}
		if !strings.Contains(w.Body.String(), "RequestId") {
			t.Errorf("%s: response should contain RequestId\nbody: %s", tc.name, w.Body.String())
		}
	}
}

// ---- Test 29: Full lifecycle — verify, send, list, delete ----

func TestSES_FullLifecycle(t *testing.T) {
	handler := newSESGateway(t)

	// Step 1: Verify identity.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, sesReq(t, "VerifyEmailIdentity", url.Values{
		"EmailAddress": {"lifecycle@example.com"},
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("Verify: expected 200, got %d", w1.Code)
	}

	// Step 2: Send email from verified identity.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, sesReq(t, "SendEmail", url.Values{
		"Source":                           {"lifecycle@example.com"},
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":             {"Lifecycle test"},
		"Message.Body.Text.Data":           {"Full lifecycle email."},
		"Message.Body.Html.Data":           {"<p>Full lifecycle email.</p>"},
	}))
	if w2.Code != http.StatusOK {
		t.Fatalf("SendEmail: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	var sendResp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"SendEmailResult"`
	}
	if err := xml.Unmarshal(w2.Body.Bytes(), &sendResp); err != nil {
		t.Fatalf("SendEmail: unmarshal: %v", err)
	}
	if sendResp.Result.MessageId == "" {
		t.Error("SendEmail: MessageId should not be empty")
	}

	// Step 3: Confirm identity is listed.
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, sesReq(t, "ListIdentities", nil))
	if !strings.Contains(w3.Body.String(), "lifecycle@example.com") {
		t.Error("ListIdentities: identity should be present")
	}

	// Step 4: Check verification attributes.
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, sesReq(t, "GetIdentityVerificationAttributes", url.Values{
		"Identities.member.1": {"lifecycle@example.com"},
	}))
	if w4.Code != http.StatusOK {
		t.Fatalf("GetIdentityVerificationAttributes: expected 200, got %d", w4.Code)
	}
	if !strings.Contains(w4.Body.String(), "Success") {
		t.Error("GetIdentityVerificationAttributes: expected Success status")
	}

	// Step 5: Delete identity.
	w5 := httptest.NewRecorder()
	handler.ServeHTTP(w5, sesReq(t, "DeleteIdentity", url.Values{
		"Identity": {"lifecycle@example.com"},
	}))
	if w5.Code != http.StatusOK {
		t.Fatalf("DeleteIdentity: expected 200, got %d", w5.Code)
	}

	// Step 6: Confirm identity is gone.
	w6 := httptest.NewRecorder()
	handler.ServeHTTP(w6, sesReq(t, "ListIdentities", nil))
	if strings.Contains(w6.Body.String(), "lifecycle@example.com") {
		t.Error("ListIdentities: identity should not be present after deletion")
	}
}

// ---- Test 30: SendEmail — error response contains Error XML structure ----

func TestSES_SendEmail_ErrorResponseFormat(t *testing.T) {
	handler := newSESGateway(t)

	// Missing source should produce a structured error.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "SendEmail", url.Values{
		"Destination.ToAddresses.member.1": {"recipient@example.com"},
		"Message.Subject.Data":             {"No source"},
	}))

	if w.Code == http.StatusOK {
		t.Fatal("SendEmail without Source: expected error, got 200")
	}

	body := w.Body.String()
	// Should contain an error structure.
	if !strings.Contains(body, "Error") {
		t.Errorf("error response should contain Error element\nbody: %s", body)
	}
}

// ---- Test 31: Unknown action returns InvalidAction error code ----

func TestSES_UnknownAction_ErrorCode(t *testing.T) {
	handler := newSESGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sesReq(t, "BogusAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("BogusAction: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "InvalidAction") {
		t.Errorf("expected InvalidAction in error response\nbody: %s", body)
	}
}
