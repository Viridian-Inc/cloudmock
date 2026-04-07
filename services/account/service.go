package account

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// AccountService is the cloudmock implementation of the AWS Account API.
type AccountService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AccountService for the given AWS account ID and region.
func New(accountID, region string) *AccountService {
	return &AccountService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *AccountService) Name() string { return "account" }

// Actions returns the list of Account API actions supported by this service.
func (s *AccountService) Actions() []service.Action {
	return []service.Action{
		{Name: "GetContactInformation", Method: http.MethodGet, IAMAction: "account:GetContactInformation"},
		{Name: "PutContactInformation", Method: http.MethodPut, IAMAction: "account:PutContactInformation"},
		{Name: "GetAlternateContact", Method: http.MethodPost, IAMAction: "account:GetAlternateContact"},
		{Name: "PutAlternateContact", Method: http.MethodPut, IAMAction: "account:PutAlternateContact"},
		{Name: "DeleteAlternateContact", Method: http.MethodPost, IAMAction: "account:DeleteAlternateContact"},
		{Name: "GetRegionOptStatus", Method: http.MethodPost, IAMAction: "account:GetRegionOptStatus"},
		{Name: "ListRegions", Method: http.MethodGet, IAMAction: "account:ListRegions"},
		{Name: "EnableRegion", Method: http.MethodPost, IAMAction: "account:EnableRegion"},
		{Name: "DisableRegion", Method: http.MethodPost, IAMAction: "account:DisableRegion"},
	}
}

// HealthCheck always returns nil.
func (s *AccountService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Account request to the appropriate handler.
// Account uses REST-JSON protocol.
func (s *AccountService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch {
	case strings.HasSuffix(path, "/contact-information") && method == http.MethodGet:
		return handleGetContactInformation(s.store)
	case strings.HasSuffix(path, "/contact-information") && method == http.MethodPut:
		return handlePutContactInformation(params, s.store)
	case strings.HasSuffix(path, "/getAlternateContact") || strings.HasSuffix(path, "/alternate-contact"):
		if method == http.MethodPost || method == http.MethodGet {
			return handleGetAlternateContact(params, s.store)
		}
	case strings.HasSuffix(path, "/putAlternateContact"):
		return handlePutAlternateContact(params, s.store)
	case strings.HasSuffix(path, "/deleteAlternateContact"):
		return handleDeleteAlternateContact(params, s.store)
	case strings.HasSuffix(path, "/getRegionOptStatus"):
		return handleGetRegionOptStatus(params, s.store)
	case strings.HasSuffix(path, "/regions") && method == http.MethodGet:
		return handleListRegions(s.store)
	case strings.HasSuffix(path, "/enableRegion"):
		return handleEnableRegion(params, s.store)
	case strings.HasSuffix(path, "/disableRegion"):
		return handleDisableRegion(params, s.store)
	}

	// Fallback: use ctx.Action for JSON protocol routing
	switch ctx.Action {
	case "GetContactInformation":
		return handleGetContactInformation(s.store)
	case "PutContactInformation":
		return handlePutContactInformation(params, s.store)
	case "GetAlternateContact":
		return handleGetAlternateContact(params, s.store)
	case "PutAlternateContact":
		return handlePutAlternateContact(params, s.store)
	case "DeleteAlternateContact":
		return handleDeleteAlternateContact(params, s.store)
	case "GetRegionOptStatus":
		return handleGetRegionOptStatus(params, s.store)
	case "ListRegions":
		return handleListRegions(s.store)
	case "EnableRegion":
		return handleEnableRegion(params, s.store)
	case "DisableRegion":
		return handleDisableRegion(params, s.store)
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
