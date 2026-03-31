package acm

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ACMService is the cloudmock implementation of the AWS Certificate Manager API.
type ACMService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ACMService for the given AWS account ID and region.
func New(accountID, region string) *ACMService {
	return &ACMService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ACMService) Name() string { return "acm" }

// Actions returns the list of ACM API actions supported by this service.
func (s *ACMService) Actions() []service.Action {
	return []service.Action{
		{Name: "RequestCertificate", Method: http.MethodPost, IAMAction: "acm:RequestCertificate"},
		{Name: "DescribeCertificate", Method: http.MethodPost, IAMAction: "acm:DescribeCertificate"},
		{Name: "ListCertificates", Method: http.MethodPost, IAMAction: "acm:ListCertificates"},
		{Name: "DeleteCertificate", Method: http.MethodPost, IAMAction: "acm:DeleteCertificate"},
		{Name: "ImportCertificate", Method: http.MethodPost, IAMAction: "acm:ImportCertificate"},
		{Name: "RenewCertificate", Method: http.MethodPost, IAMAction: "acm:RenewCertificate"},
		{Name: "ExportCertificate", Method: http.MethodPost, IAMAction: "acm:ExportCertificate"},
		{Name: "GetCertificate", Method: http.MethodPost, IAMAction: "acm:GetCertificate"},
		{Name: "AddTagsToCertificate", Method: http.MethodPost, IAMAction: "acm:AddTagsToCertificate"},
		{Name: "RemoveTagsFromCertificate", Method: http.MethodPost, IAMAction: "acm:RemoveTagsFromCertificate"},
		{Name: "ListTagsForCertificate", Method: http.MethodPost, IAMAction: "acm:ListTagsForCertificate"},
	}
}

// HealthCheck always returns nil.
func (s *ACMService) HealthCheck() error { return nil }

// HandleRequest routes an incoming ACM request to the appropriate handler.
func (s *ACMService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "RequestCertificate":
		return handleRequestCertificate(ctx, s.store)
	case "DescribeCertificate":
		return handleDescribeCertificate(ctx, s.store)
	case "ListCertificates":
		return handleListCertificates(ctx, s.store)
	case "DeleteCertificate":
		return handleDeleteCertificate(ctx, s.store)
	case "ImportCertificate":
		return handleImportCertificate(ctx, s.store)
	case "RenewCertificate":
		return handleRenewCertificate(ctx, s.store)
	case "ExportCertificate":
		return handleExportCertificate(ctx, s.store)
	case "GetCertificate":
		return handleGetCertificate(ctx, s.store)
	case "AddTagsToCertificate":
		return handleAddTagsToCertificate(ctx, s.store)
	case "RemoveTagsFromCertificate":
		return handleRemoveTagsFromCertificate(ctx, s.store)
	case "ListTagsForCertificate":
		return handleListTagsForCertificate(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
