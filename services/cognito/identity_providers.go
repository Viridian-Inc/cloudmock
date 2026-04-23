package cognito

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// IdentityProvider represents a Cognito User Pool IdentityProvider —
// the federation bridge between a Cognito User Pool and an external
// IdP (SAML 2.0 or OIDC). ProviderType values match AWS's enum:
// SAML, OIDC, Google, Facebook, LoginWithAmazon, SignInWithApple.
//
// ProviderDetails is a flat string→string map (AWS's actual shape);
// the SAML path typically has MetadataURL / MetadataFile / IDPSignout,
// and the OIDC path typically has client_id / client_secret / attributes_request_method
// / oidc_issuer / authorize_scopes / attributes_url / attributes_url_add_attributes
// / authorize_url / token_url / jwks_uri.
type IdentityProvider struct {
	UserPoolId       string
	ProviderName     string
	ProviderType     string
	ProviderDetails  map[string]string
	AttributeMapping map[string]string
	IdpIdentifiers   []string
	CreationDate     time.Time
	LastModifiedDate time.Time
}

// ── Store methods ────────────────────────────────────────────────────────────

// CreateIdentityProvider registers a new federated IdP on the user pool.
// Returns DuplicateProviderException if a provider with the same name
// already exists on that pool — matching AWS's error shape.
func (s *Store) CreateIdentityProvider(
	userPoolID string,
	name string,
	pType string,
	details map[string]string,
	attrMap map[string]string,
	idpIdentifiers []string,
) (*IdentityProvider, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return nil, poolNotFound(userPoolID)
	}
	if _, exists := pool.IdentityProviders[name]; exists {
		return nil, service.NewAWSError("DuplicateProviderException",
			fmt.Sprintf("Provider %s already exists.", name), http.StatusBadRequest)
	}

	now := time.Now().UTC()
	idp := &IdentityProvider{
		UserPoolId:       pool.Id,
		ProviderName:     name,
		ProviderType:     pType,
		ProviderDetails:  copyStringMap(details),
		AttributeMapping: copyStringMap(attrMap),
		IdpIdentifiers:   append([]string(nil), idpIdentifiers...),
		CreationDate:     now,
		LastModifiedDate: now,
	}
	pool.IdentityProviders[name] = idp
	return idp, nil
}

// DescribeIdentityProvider returns the full IdP record, including
// ProviderDetails which is omitted from ListIdentityProviders summaries.
func (s *Store) DescribeIdentityProvider(
	userPoolID, providerName string,
) (*IdentityProvider, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return nil, poolNotFound(userPoolID)
	}
	idp, ok := pool.IdentityProviders[providerName]
	if !ok {
		return nil, idpNotFound(providerName)
	}
	return idp, nil
}

// UpdateIdentityProvider partially updates the IdP — any nil / empty
// input map is treated as "leave unchanged" (matching AWS's behavior
// where omitted fields are not cleared). Use a non-nil empty map to
// explicitly clear a field.
func (s *Store) UpdateIdentityProvider(
	userPoolID string,
	name string,
	details map[string]string,
	attrMap map[string]string,
	idpIdentifiers []string,
) (*IdentityProvider, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return nil, poolNotFound(userPoolID)
	}
	idp, ok := pool.IdentityProviders[name]
	if !ok {
		return nil, idpNotFound(name)
	}
	if details != nil {
		idp.ProviderDetails = copyStringMap(details)
	}
	if attrMap != nil {
		idp.AttributeMapping = copyStringMap(attrMap)
	}
	if idpIdentifiers != nil {
		idp.IdpIdentifiers = append([]string(nil), idpIdentifiers...)
	}
	idp.LastModifiedDate = time.Now().UTC()
	return idp, nil
}

// DeleteIdentityProvider removes the IdP from the pool. Returns
// ResourceNotFoundException when missing — AWS's actual behavior.
func (s *Store) DeleteIdentityProvider(userPoolID, providerName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return poolNotFound(userPoolID)
	}
	if _, ok := pool.IdentityProviders[providerName]; !ok {
		return idpNotFound(providerName)
	}
	delete(pool.IdentityProviders, providerName)
	return nil
}

// ListIdentityProviders returns summaries (no ProviderDetails) sorted
// by ProviderName. maxResults == 0 means unlimited.
func (s *Store) ListIdentityProviders(
	userPoolID string, maxResults int,
) ([]*IdentityProvider, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return nil, poolNotFound(userPoolID)
	}
	out := make([]*IdentityProvider, 0, len(pool.IdentityProviders))
	for _, idp := range pool.IdentityProviders {
		out = append(out, idp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ProviderName < out[j].ProviderName
	})
	if maxResults > 0 && len(out) > maxResults {
		out = out[:maxResults]
	}
	return out, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func idpNotFound(name string) *service.AWSError {
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Identity provider %s not found.", name), http.StatusBadRequest)
}

func copyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// idpToJSON matches AWS's IdentityProviderType response shape. Dates
// are returned as Unix seconds (float), same as groups + pools.
func idpToJSON(idp *IdentityProvider, includeDetails bool) map[string]any {
	out := map[string]any{
		"UserPoolId":       idp.UserPoolId,
		"ProviderName":     idp.ProviderName,
		"ProviderType":     idp.ProviderType,
		"CreationDate":     float64(idp.CreationDate.Unix()),
		"LastModifiedDate": float64(idp.LastModifiedDate.Unix()),
	}
	if includeDetails {
		// Describe / Create / Update include the full detail map +
		// attribute mapping + identifiers. List returns summaries.
		out["ProviderDetails"] = idp.ProviderDetails
		out["AttributeMapping"] = idp.AttributeMapping
		out["IdpIdentifiers"] = idp.IdpIdentifiers
	}
	return out
}

// ── Request types ────────────────────────────────────────────────────────────

type createIdpRequest struct {
	UserPoolId       string            `json:"UserPoolId"`
	ProviderName     string            `json:"ProviderName"`
	ProviderType     string            `json:"ProviderType"`
	ProviderDetails  map[string]string `json:"ProviderDetails"`
	AttributeMapping map[string]string `json:"AttributeMapping"`
	IdpIdentifiers   []string          `json:"IdpIdentifiers"`
}

type describeIdpRequest struct {
	UserPoolId   string `json:"UserPoolId"`
	ProviderName string `json:"ProviderName"`
}

type updateIdpRequest struct {
	UserPoolId       string            `json:"UserPoolId"`
	ProviderName     string            `json:"ProviderName"`
	ProviderDetails  map[string]string `json:"ProviderDetails"`
	AttributeMapping map[string]string `json:"AttributeMapping"`
	IdpIdentifiers   []string          `json:"IdpIdentifiers"`
}

type deleteIdpRequest struct {
	UserPoolId   string `json:"UserPoolId"`
	ProviderName string `json:"ProviderName"`
}

type listIdpRequest struct {
	UserPoolId string `json:"UserPoolId"`
	MaxResults int    `json:"MaxResults"`
	NextToken  string `json:"NextToken"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateIdentityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createIdpRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.UserPoolId == "" || req.ProviderName == "" || req.ProviderType == "" {
		return cognitoErr("InvalidParameterException",
			"UserPoolId, ProviderName, and ProviderType are required.")
	}
	if !isValidProviderType(req.ProviderType) {
		return cognitoErr("InvalidParameterException",
			fmt.Sprintf("ProviderType %s is not a valid Cognito IdP type.", req.ProviderType))
	}
	idp, awsErr := store.CreateIdentityProvider(
		req.UserPoolId, req.ProviderName, req.ProviderType,
		req.ProviderDetails, req.AttributeMapping, req.IdpIdentifiers,
	)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(map[string]any{"IdentityProvider": idpToJSON(idp, true)})
}

func handleDescribeIdentityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeIdpRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.UserPoolId == "" || req.ProviderName == "" {
		return cognitoErr("InvalidParameterException",
			"UserPoolId and ProviderName are required.")
	}
	idp, awsErr := store.DescribeIdentityProvider(req.UserPoolId, req.ProviderName)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(map[string]any{"IdentityProvider": idpToJSON(idp, true)})
}

func handleUpdateIdentityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateIdpRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.UserPoolId == "" || req.ProviderName == "" {
		return cognitoErr("InvalidParameterException",
			"UserPoolId and ProviderName are required.")
	}
	idp, awsErr := store.UpdateIdentityProvider(
		req.UserPoolId, req.ProviderName,
		req.ProviderDetails, req.AttributeMapping, req.IdpIdentifiers,
	)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(map[string]any{"IdentityProvider": idpToJSON(idp, true)})
}

func handleDeleteIdentityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteIdpRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.UserPoolId == "" || req.ProviderName == "" {
		return cognitoErr("InvalidParameterException",
			"UserPoolId and ProviderName are required.")
	}
	if awsErr := store.DeleteIdentityProvider(req.UserPoolId, req.ProviderName); awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(struct{}{})
}

func handleListIdentityProviders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listIdpRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.UserPoolId == "" {
		return cognitoErr("InvalidParameterException", "UserPoolId is required.")
	}
	idps, awsErr := store.ListIdentityProviders(req.UserPoolId, req.MaxResults)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	// ListIdentityProviders returns ProviderDescription entries — a
	// subset of the full record (no ProviderDetails or AttributeMapping).
	items := make([]map[string]any, 0, len(idps))
	for _, idp := range idps {
		items = append(items, idpToJSON(idp, false))
	}
	return cognitoOK(map[string]any{"Providers": items})
}

// ── Validation ───────────────────────────────────────────────────────────────

// Allowed AWS ProviderType values. Kept in lockstep with the Cognito
// SDK's enum — any value not in this list is rejected at Create time
// with InvalidParameterException (AWS's actual behavior).
var validProviderTypes = map[string]bool{
	"SAML":              true,
	"OIDC":              true,
	"Google":            true,
	"Facebook":          true,
	"LoginWithAmazon":   true,
	"SignInWithApple":   true,
}

func isValidProviderType(t string) bool {
	return validProviderTypes[t]
}
