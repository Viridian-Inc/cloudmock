package sts

import (
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

// STSService is the cloudmock implementation of the AWS Security Token Service API.
type STSService struct {
	accountID  string
	credMapper CredentialMapper
}

// New returns a new STSService for the given AWS account ID.
func New(accountID string) *STSService {
	return &STSService{accountID: accountID}
}

// SetCredentialMapper attaches a CredentialMapper for cross-account credential tracking.
// When set, AssumeRole will register temporary credentials against the target account.
func (s *STSService) SetCredentialMapper(cm CredentialMapper) {
	s.credMapper = cm
}

// Name returns the AWS service name used for routing.
func (s *STSService) Name() string { return "sts" }

// Actions returns the list of STS API actions supported by this service.
func (s *STSService) Actions() []service.Action {
	return []service.Action{
		{Name: "GetCallerIdentity", Method: http.MethodPost, IAMAction: "sts:GetCallerIdentity"},
		{Name: "AssumeRole", Method: http.MethodPost, IAMAction: "sts:AssumeRole"},
		{Name: "GetSessionToken", Method: http.MethodPost, IAMAction: "sts:GetSessionToken"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *STSService) HealthCheck() error { return nil }

// HandleRequest routes an incoming STS request to the appropriate handler.
// STS uses form-encoded POST bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *STSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action

	// If Action was not extracted from the query string, try the form body.
	if action == "" {
		if formVals, err := parseFormBody(ctx.Body); err == nil {
			action = formVals.Get("Action")
		}
	}

	switch action {
	case "GetCallerIdentity":
		return handleGetCallerIdentity(ctx)
	case "AssumeRole":
		return handleAssumeRole(ctx, s.accountID, s.credMapper)
	case "GetSessionToken":
		return handleGetSessionToken(ctx)
	default:
		awsErr := service.NewAWSError(
			"InvalidAction",
			"The action "+action+" is not valid for this web service.",
			http.StatusBadRequest,
		)
		return &service.Response{Format: service.FormatXML}, awsErr
	}
}

// parseFormBody parses a URL-encoded (application/x-www-form-urlencoded) body.
func parseFormBody(body []byte) (url.Values, error) {
	return url.ParseQuery(string(body))
}
