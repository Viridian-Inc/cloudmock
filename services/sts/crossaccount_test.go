package sts_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/account"
	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	stssvc "github.com/neureaux/cloudmock/services/sts"
)

// newCrossAccountGateway builds a gateway with multi-account support enabled.
func newCrossAccountGateway(t *testing.T) (http.Handler, *account.Registry) {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "none"

	acctReg := account.NewRegistry(cfg.AccountID, cfg.Region)
	acctReg.CreateAccount("999999999999", "Target Account")

	stsService := stssvc.New(cfg.AccountID)
	stsService.SetCredentialMapper(acctReg)

	reg := routing.NewRegistry()
	reg.Register(stsService)

	gw := gateway.New(cfg, reg)
	gw.SetAccountRegistry(acctReg)
	return gw, acctReg
}

func TestSTS_AssumeRole_SameAccount(t *testing.T) {
	handler, acctReg := newCrossAccountGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::000000000000:role/MyRole")
	extra.Set("RoleSessionName", "same-acct")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Extract the access key and verify it maps to the same account.
	akRe := regexp.MustCompile(`<AccessKeyId>([^<]+)</AccessKeyId>`)
	m := akRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract AccessKeyId")
	}

	acctID, ok := acctReg.ResolveCredential(m[1])
	if !ok {
		t.Fatal("credential should be mapped")
	}
	if acctID != "000000000000" {
		t.Errorf("credential should map to same account 000000000000, got %q", acctID)
	}
}

func TestSTS_AssumeRole_CrossAccount(t *testing.T) {
	handler, acctReg := newCrossAccountGateway(t)

	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::999999999999:role/CrossRole")
	extra.Set("RoleSessionName", "cross-acct")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Verify the assumed role ARN references the target account.
	if !strings.Contains(body, "999999999999") {
		t.Errorf("response should contain target account ID 999999999999\nbody: %s", body)
	}

	// Extract the access key and verify it maps to the target account.
	akRe := regexp.MustCompile(`<AccessKeyId>([^<]+)</AccessKeyId>`)
	m := akRe.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("could not extract AccessKeyId")
	}

	acctID, ok := acctReg.ResolveCredential(m[1])
	if !ok {
		t.Fatal("credential should be mapped for cross-account assume")
	}
	if acctID != "999999999999" {
		t.Errorf("credential should map to target account 999999999999, got %q", acctID)
	}
}

func TestSTS_GetCallerIdentity_WithAssumedRole(t *testing.T) {
	handler, acctReg := newCrossAccountGateway(t)

	// First, do a cross-account AssumeRole to get credentials.
	extra := url.Values{}
	extra.Set("RoleArn", "arn:aws:iam::999999999999:role/TestRole")
	extra.Set("RoleSessionName", "identity-test")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, stsReq(t, "AssumeRole", extra))

	if w.Code != http.StatusOK {
		t.Fatalf("AssumeRole: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Extract the temporary access key.
	akRe := regexp.MustCompile(`<AccessKeyId>([^<]+)</AccessKeyId>`)
	m := akRe.FindStringSubmatch(w.Body.String())
	if len(m) < 2 {
		t.Fatal("could not extract AccessKeyId from AssumeRole response")
	}
	tempKeyID := m[1]

	// Verify the credential is mapped to the target account in the registry.
	acctID, ok := acctReg.ResolveCredential(tempKeyID)
	if !ok {
		t.Fatal("temporary credential should be mapped")
	}
	if acctID != "999999999999" {
		t.Errorf("temporary credential maps to %q, want 999999999999", acctID)
	}
}
