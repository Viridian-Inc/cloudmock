package iam

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// OIDCProvider represents an IAM OpenID Connect provider.
type OIDCProvider struct {
	Arn          string
	Url          string
	ClientIDs    []string
	Thumbprints  []string
	CreateDate   time.Time
}

// SAMLProvider represents an IAM SAML provider.
type SAMLProvider struct {
	Arn              string
	Name             string
	MetadataDocument string
	CreateDate       time.Time
}

var (
	oidcMu    sync.RWMutex
	oidcProviders = make(map[string]*OIDCProvider) // ARN -> provider

	samlMu    sync.RWMutex
	samlProviders = make(map[string]*SAMLProvider) // ARN -> provider
)

// ── OIDC Providers ───────────────────────────────────────────────────────────

func handleCreateOpenIDConnectProvider(store *Store, form url.Values) (*service.Response, error) {
	providerUrl := form.Get("Url")
	if providerUrl == "" {
		return iamErr("ValidationError", "Url is required.", http.StatusBadRequest)
	}

	// Collect thumbprints and client IDs from form
	thumbprints := collectListParam(form, "ThumbprintList.member")
	clientIDs := collectListParam(form, "ClientIDList.member")

	// Build ARN from URL host
	host := strings.TrimPrefix(providerUrl, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	arn := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", store.accountID, host)

	provider := &OIDCProvider{
		Arn:         arn,
		Url:         providerUrl,
		ClientIDs:   clientIDs,
		Thumbprints: thumbprints,
		CreateDate:  time.Now().UTC(),
	}

	oidcMu.Lock()
	oidcProviders[arn] = provider
	oidcMu.Unlock()

	return iamXMLResponse("CreateOpenIDConnectProvider", fmt.Sprintf(
		`<OpenIDConnectProviderArn>%s</OpenIDConnectProviderArn>`, arn))
}

func handleGetOpenIDConnectProvider(store *Store, form url.Values) (*service.Response, error) {
	arn := form.Get("OpenIDConnectProviderArn")
	if arn == "" {
		return iamErr("ValidationError", "OpenIDConnectProviderArn is required.", http.StatusBadRequest)
	}

	oidcMu.RLock()
	provider, ok := oidcProviders[arn]
	oidcMu.RUnlock()

	if !ok {
		return iamErr("NoSuchEntity", "OIDC provider not found.", http.StatusNotFound)
	}

	var clientIDsXML string
	for _, id := range provider.ClientIDs {
		clientIDsXML += fmt.Sprintf("<member>%s</member>", id)
	}
	var thumbprintsXML string
	for _, tp := range provider.Thumbprints {
		thumbprintsXML += fmt.Sprintf("<member>%s</member>", tp)
	}

	return iamXMLResponse("GetOpenIDConnectProvider", fmt.Sprintf(
		`<Url>%s</Url>
		<CreateDate>%s</CreateDate>
		<ClientIDList>%s</ClientIDList>
		<ThumbprintList>%s</ThumbprintList>`,
		provider.Url,
		provider.CreateDate.Format(time.RFC3339),
		clientIDsXML,
		thumbprintsXML,
	))
}

func handleListOpenIDConnectProviders(store *Store, form url.Values) (*service.Response, error) {
	oidcMu.RLock()
	defer oidcMu.RUnlock()

	var membersXML string
	for arn := range oidcProviders {
		membersXML += fmt.Sprintf("<member><Arn>%s</Arn></member>", arn)
	}

	return iamXMLResponse("ListOpenIDConnectProviders", fmt.Sprintf(
		`<OpenIDConnectProviderList>%s</OpenIDConnectProviderList>`, membersXML))
}

func handleDeleteOpenIDConnectProvider(store *Store, form url.Values) (*service.Response, error) {
	arn := form.Get("OpenIDConnectProviderArn")
	if arn == "" {
		return iamErr("ValidationError", "OpenIDConnectProviderArn is required.", http.StatusBadRequest)
	}

	oidcMu.Lock()
	delete(oidcProviders, arn)
	oidcMu.Unlock()

	return iamXMLResponse("DeleteOpenIDConnectProvider", "")
}

// ── SAML Providers ───────────────────────────────────────────────────────────

func handleCreateSAMLProvider(store *Store, form url.Values) (*service.Response, error) {
	name := form.Get("Name")
	metadata := form.Get("SAMLMetadataDocument")
	if name == "" {
		return iamErr("ValidationError", "Name is required.", http.StatusBadRequest)
	}

	arn := fmt.Sprintf("arn:aws:iam::%s:saml-provider/%s", store.accountID, name)

	provider := &SAMLProvider{
		Arn:              arn,
		Name:             name,
		MetadataDocument: metadata,
		CreateDate:       time.Now().UTC(),
	}

	samlMu.Lock()
	samlProviders[arn] = provider
	samlMu.Unlock()

	return iamXMLResponse("CreateSAMLProvider", fmt.Sprintf(
		`<SAMLProviderArn>%s</SAMLProviderArn>`, arn))
}

func handleListSAMLProviders(store *Store, form url.Values) (*service.Response, error) {
	samlMu.RLock()
	defer samlMu.RUnlock()

	var membersXML string
	for _, p := range samlProviders {
		membersXML += fmt.Sprintf(
			`<member><Arn>%s</Arn><ValidUntil>%s</ValidUntil><CreateDate>%s</CreateDate></member>`,
			p.Arn, p.CreateDate.Add(365*24*time.Hour).Format(time.RFC3339), p.CreateDate.Format(time.RFC3339))
	}

	return iamXMLResponse("ListSAMLProviders", fmt.Sprintf(
		`<SAMLProviderList>%s</SAMLProviderList>`, membersXML))
}

func handleDeleteSAMLProvider(store *Store, form url.Values) (*service.Response, error) {
	arn := form.Get("SAMLProviderArn")
	if arn == "" {
		return iamErr("ValidationError", "SAMLProviderArn is required.", http.StatusBadRequest)
	}

	samlMu.Lock()
	delete(samlProviders, arn)
	samlMu.Unlock()

	return iamXMLResponse("DeleteSAMLProvider", "")
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func collectListParam(form url.Values, prefix string) []string {
	var items []string
	for i := 1; i <= 20; i++ {
		key := fmt.Sprintf("%s.%d", prefix, i)
		if v := form.Get(key); v != "" {
			items = append(items, v)
		} else {
			break
		}
	}
	return items
}


func iamXMLResponse(action, innerXML string) (*service.Response, error) {
	xml := fmt.Sprintf(
		`<%sResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/"><%sResult>%s</%sResult></%sResponse>`,
		action, action, innerXML, action, action)
	return &service.Response{
		StatusCode: http.StatusOK,
		RawBody:    []byte(xml),
		Format:     service.FormatXML,
	}, nil
}
