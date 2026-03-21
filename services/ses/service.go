package ses

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SESService is the cloudmock implementation of the AWS Simple Email Service API.
type SESService struct {
	store *Store
}

// New returns a new SESService for the given AWS account ID and region.
func New(accountID, region string) *SESService {
	return &SESService{
		store: NewStore(),
	}
}

// Name returns the AWS service name used for routing.
func (s *SESService) Name() string { return "ses" }

// Actions returns the list of SES API actions supported by this service.
func (s *SESService) Actions() []service.Action {
	return []service.Action{
		{Name: "SendEmail", Method: http.MethodPost, IAMAction: "ses:SendEmail"},
		{Name: "SendRawEmail", Method: http.MethodPost, IAMAction: "ses:SendRawEmail"},
		{Name: "VerifyEmailIdentity", Method: http.MethodPost, IAMAction: "ses:VerifyEmailIdentity"},
		{Name: "ListIdentities", Method: http.MethodPost, IAMAction: "ses:ListIdentities"},
		{Name: "DeleteIdentity", Method: http.MethodPost, IAMAction: "ses:DeleteIdentity"},
		{Name: "GetIdentityVerificationAttributes", Method: http.MethodPost, IAMAction: "ses:GetIdentityVerificationAttributes"},
		{Name: "ListVerifiedEmailAddresses", Method: http.MethodPost, IAMAction: "ses:ListVerifiedEmailAddresses"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *SESService) HealthCheck() error { return nil }

// HandleRequest routes an incoming SES request to the appropriate handler.
// SES uses form-encoded POST bodies; the Action may appear in the query string
// (already parsed into ctx.Params) or in the form-encoded body.
func (s *SESService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	case "SendEmail":
		return handleSendEmail(ctx, s.store)
	case "SendRawEmail":
		return handleSendRawEmail(ctx, s.store)
	case "VerifyEmailIdentity":
		return handleVerifyEmailIdentity(ctx, s.store)
	case "ListIdentities":
		return handleListIdentities(ctx, s.store)
	case "DeleteIdentity":
		return handleDeleteIdentity(ctx, s.store)
	case "GetIdentityVerificationAttributes":
		return handleGetIdentityVerificationAttributes(ctx, s.store)
	case "ListVerifiedEmailAddresses":
		return handleListVerifiedEmailAddresses(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
