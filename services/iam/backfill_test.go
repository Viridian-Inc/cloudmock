package iam_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ---- OIDC Providers ----

func TestIAM_OIDCProviderLifecycle(t *testing.T) {
	handler := newIAMGateway(t)

	// CreateOpenIDConnectProvider
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "CreateOpenIDConnectProvider", url.Values{
		"Url":                    {"https://accounts.google.com"},
		"ThumbprintList.member.1": {"abc123def456"},
		"ClientIDList.member.1":   {"my-client-id"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateOpenIDConnectProvider: %d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "OpenIDConnectProviderArn") {
		t.Fatalf("expected ARN in response, got: %s", body)
	}

	// ListOpenIDConnectProviders
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "ListOpenIDConnectProviders", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListOpenIDConnectProviders: %d %s", w.Code, w.Body.String())
	}

	// GetOpenIDConnectProvider
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "GetOpenIDConnectProvider", url.Values{
		"OpenIDConnectProviderArn": {"arn:aws:iam::000000000000:oidc-provider/accounts.google.com"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetOpenIDConnectProvider: %d %s", w.Code, w.Body.String())
	}

	// DeleteOpenIDConnectProvider
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "DeleteOpenIDConnectProvider", url.Values{
		"OpenIDConnectProviderArn": {"arn:aws:iam::000000000000:oidc-provider/accounts.google.com"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteOpenIDConnectProvider: %d %s", w.Code, w.Body.String())
	}
}

// ---- SAML Providers ----

func TestIAM_SAMLProviderLifecycle(t *testing.T) {
	handler := newIAMGateway(t)

	// CreateSAMLProvider
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "CreateSAMLProvider", url.Values{
		"Name":                 {"my-saml"},
		"SAMLMetadataDocument": {"<xml>saml-metadata</xml>"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSAMLProvider: %d %s", w.Code, w.Body.String())
	}

	// ListSAMLProviders
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "ListSAMLProviders", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSAMLProviders: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "my-saml") {
		t.Errorf("expected my-saml in list, got: %s", w.Body.String())
	}

	// DeleteSAMLProvider
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, "DeleteSAMLProvider", url.Values{
		"SAMLProviderArn": {"arn:aws:iam::000000000000:saml-provider/my-saml"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteSAMLProvider: %d %s", w.Code, w.Body.String())
	}
}
