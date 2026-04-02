package amplify

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// AmplifyService is the cloudmock implementation of the AWS Amplify API.
type AmplifyService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AmplifyService for the given AWS account ID and region.
func New(accountID, region string) *AmplifyService {
	return &AmplifyService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *AmplifyService) Name() string { return "amplify" }

// Actions returns the list of Amplify API actions supported by this service.
func (s *AmplifyService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApp", Method: http.MethodPost, IAMAction: "amplify:CreateApp"},
		{Name: "GetApp", Method: http.MethodGet, IAMAction: "amplify:GetApp"},
		{Name: "ListApps", Method: http.MethodGet, IAMAction: "amplify:ListApps"},
		{Name: "UpdateApp", Method: http.MethodPost, IAMAction: "amplify:UpdateApp"},
		{Name: "DeleteApp", Method: http.MethodDelete, IAMAction: "amplify:DeleteApp"},
		{Name: "CreateBranch", Method: http.MethodPost, IAMAction: "amplify:CreateBranch"},
		{Name: "GetBranch", Method: http.MethodGet, IAMAction: "amplify:GetBranch"},
		{Name: "ListBranches", Method: http.MethodGet, IAMAction: "amplify:ListBranches"},
		{Name: "UpdateBranch", Method: http.MethodPost, IAMAction: "amplify:UpdateBranch"},
		{Name: "DeleteBranch", Method: http.MethodDelete, IAMAction: "amplify:DeleteBranch"},
		{Name: "CreateDomainAssociation", Method: http.MethodPost, IAMAction: "amplify:CreateDomainAssociation"},
		{Name: "GetDomainAssociation", Method: http.MethodGet, IAMAction: "amplify:GetDomainAssociation"},
		{Name: "ListDomainAssociations", Method: http.MethodGet, IAMAction: "amplify:ListDomainAssociations"},
		{Name: "UpdateDomainAssociation", Method: http.MethodPost, IAMAction: "amplify:UpdateDomainAssociation"},
		{Name: "DeleteDomainAssociation", Method: http.MethodDelete, IAMAction: "amplify:DeleteDomainAssociation"},
		{Name: "CreateWebhook", Method: http.MethodPost, IAMAction: "amplify:CreateWebhook"},
		{Name: "GetWebhook", Method: http.MethodGet, IAMAction: "amplify:GetWebhook"},
		{Name: "ListWebhooks", Method: http.MethodGet, IAMAction: "amplify:ListWebhooks"},
		{Name: "UpdateWebhook", Method: http.MethodPost, IAMAction: "amplify:UpdateWebhook"},
		{Name: "DeleteWebhook", Method: http.MethodDelete, IAMAction: "amplify:DeleteWebhook"},
		{Name: "StartJob", Method: http.MethodPost, IAMAction: "amplify:StartJob"},
		{Name: "GetJob", Method: http.MethodGet, IAMAction: "amplify:GetJob"},
		{Name: "ListJobs", Method: http.MethodGet, IAMAction: "amplify:ListJobs"},
		{Name: "StopJob", Method: http.MethodDelete, IAMAction: "amplify:StopJob"},
		{Name: "CreateBackendEnvironment", Method: http.MethodPost, IAMAction: "amplify:CreateBackendEnvironment"},
		{Name: "GetBackendEnvironment", Method: http.MethodGet, IAMAction: "amplify:GetBackendEnvironment"},
		{Name: "ListBackendEnvironments", Method: http.MethodGet, IAMAction: "amplify:ListBackendEnvironments"},
		{Name: "DeleteBackendEnvironment", Method: http.MethodDelete, IAMAction: "amplify:DeleteBackendEnvironment"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "amplify:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "amplify:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "amplify:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AmplifyService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Amplify request to the appropriate handler.
func (s *AmplifyService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateApp":
		return handleCreateApp(ctx, s.store)
	case "GetApp":
		return handleGetApp(ctx, s.store)
	case "ListApps":
		return handleListApps(ctx, s.store)
	case "UpdateApp":
		return handleUpdateApp(ctx, s.store)
	case "DeleteApp":
		return handleDeleteApp(ctx, s.store)
	case "CreateBranch":
		return handleCreateBranch(ctx, s.store)
	case "GetBranch":
		return handleGetBranch(ctx, s.store)
	case "ListBranches":
		return handleListBranches(ctx, s.store)
	case "UpdateBranch":
		return handleUpdateBranch(ctx, s.store)
	case "DeleteBranch":
		return handleDeleteBranch(ctx, s.store)
	case "CreateDomainAssociation":
		return handleCreateDomainAssociation(ctx, s.store)
	case "GetDomainAssociation":
		return handleGetDomainAssociation(ctx, s.store)
	case "ListDomainAssociations":
		return handleListDomainAssociations(ctx, s.store)
	case "UpdateDomainAssociation":
		return handleUpdateDomainAssociation(ctx, s.store)
	case "DeleteDomainAssociation":
		return handleDeleteDomainAssociation(ctx, s.store)
	case "CreateWebhook":
		return handleCreateWebhook(ctx, s.store)
	case "GetWebhook":
		return handleGetWebhook(ctx, s.store)
	case "ListWebhooks":
		return handleListWebhooks(ctx, s.store)
	case "UpdateWebhook":
		return handleUpdateWebhook(ctx, s.store)
	case "DeleteWebhook":
		return handleDeleteWebhook(ctx, s.store)
	case "StartJob":
		return handleStartJob(ctx, s.store)
	case "GetJob":
		return handleGetJob(ctx, s.store)
	case "ListJobs":
		return handleListJobs(ctx, s.store)
	case "StopJob":
		return handleStopJob(ctx, s.store)
	case "CreateBackendEnvironment":
		return handleCreateBackendEnvironment(ctx, s.store)
	case "GetBackendEnvironment":
		return handleGetBackendEnvironment(ctx, s.store)
	case "ListBackendEnvironments":
		return handleListBackendEnvironments(ctx, s.store)
	case "DeleteBackendEnvironment":
		return handleDeleteBackendEnvironment(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
