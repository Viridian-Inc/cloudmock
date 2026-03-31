package acmpca

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ACMPCAService is the cloudmock implementation of the AWS Certificate Manager Private CA API.
type ACMPCAService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ACMPCAService for the given AWS account ID and region.
func New(accountID, region string) *ACMPCAService {
	return &ACMPCAService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ACMPCAService) Name() string { return "acm-pca" }

// Actions returns the list of ACM-PCA API actions supported by this service.
func (s *ACMPCAService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:CreateCertificateAuthority"},
		{Name: "DescribeCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:DescribeCertificateAuthority"},
		{Name: "ListCertificateAuthorities", Method: http.MethodPost, IAMAction: "acm-pca:ListCertificateAuthorities"},
		{Name: "DeleteCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:DeleteCertificateAuthority"},
		{Name: "UpdateCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:UpdateCertificateAuthority"},
		{Name: "IssueCertificate", Method: http.MethodPost, IAMAction: "acm-pca:IssueCertificate"},
		{Name: "GetCertificate", Method: http.MethodPost, IAMAction: "acm-pca:GetCertificate"},
		{Name: "RevokeCertificate", Method: http.MethodPost, IAMAction: "acm-pca:RevokeCertificate"},
		{Name: "TagCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:TagCertificateAuthority"},
		{Name: "UntagCertificateAuthority", Method: http.MethodPost, IAMAction: "acm-pca:UntagCertificateAuthority"},
		{Name: "ListTags", Method: http.MethodPost, IAMAction: "acm-pca:ListTags"},
		{Name: "CreatePermission", Method: http.MethodPost, IAMAction: "acm-pca:CreatePermission"},
		{Name: "ListPermissions", Method: http.MethodPost, IAMAction: "acm-pca:ListPermissions"},
		{Name: "DeletePermission", Method: http.MethodPost, IAMAction: "acm-pca:DeletePermission"},
	}
}

// HealthCheck always returns nil.
func (s *ACMPCAService) HealthCheck() error { return nil }

// HandleRequest routes an incoming ACM-PCA request to the appropriate handler.
func (s *ACMPCAService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateCertificateAuthority":
		return handleCreateCertificateAuthority(ctx, s.store)
	case "DescribeCertificateAuthority":
		return handleDescribeCertificateAuthority(ctx, s.store)
	case "ListCertificateAuthorities":
		return handleListCertificateAuthorities(ctx, s.store)
	case "DeleteCertificateAuthority":
		return handleDeleteCertificateAuthority(ctx, s.store)
	case "UpdateCertificateAuthority":
		return handleUpdateCertificateAuthority(ctx, s.store)
	case "IssueCertificate":
		return handleIssueCertificate(ctx, s.store)
	case "GetCertificate":
		return handleGetCertificate(ctx, s.store)
	case "RevokeCertificate":
		return handleRevokeCertificate(ctx, s.store)
	case "TagCertificateAuthority":
		return handleTagCertificateAuthority(ctx, s.store)
	case "UntagCertificateAuthority":
		return handleUntagCertificateAuthority(ctx, s.store)
	case "ListTags":
		return handleListTags(ctx, s.store)
	case "CreatePermission":
		return handleCreatePermission(ctx, s.store)
	case "ListPermissions":
		return handleListPermissions(ctx, s.store)
	case "DeletePermission":
		return handleDeletePermission(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
