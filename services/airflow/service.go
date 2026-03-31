package airflow

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// AirflowService is the cloudmock implementation of the AWS MWAA (Managed Workflows for Apache Airflow) API.
type AirflowService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new AirflowService for the given AWS account ID and region.
func New(accountID, region string) *AirflowService {
	return &AirflowService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *AirflowService) Name() string { return "airflow" }

// Actions returns the list of MWAA API actions supported by this service.
func (s *AirflowService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateEnvironment", Method: http.MethodPut, IAMAction: "airflow:CreateEnvironment"},
		{Name: "GetEnvironment", Method: http.MethodGet, IAMAction: "airflow:GetEnvironment"},
		{Name: "ListEnvironments", Method: http.MethodGet, IAMAction: "airflow:ListEnvironments"},
		{Name: "UpdateEnvironment", Method: http.MethodPatch, IAMAction: "airflow:UpdateEnvironment"},
		{Name: "DeleteEnvironment", Method: http.MethodDelete, IAMAction: "airflow:DeleteEnvironment"},
		{Name: "CreateCliToken", Method: http.MethodPost, IAMAction: "airflow:CreateCliToken"},
		{Name: "CreateWebLoginToken", Method: http.MethodPost, IAMAction: "airflow:CreateWebLoginToken"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "airflow:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "airflow:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "airflow:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *AirflowService) HealthCheck() error { return nil }

// HandleRequest routes an incoming MWAA request to the appropriate handler.
func (s *AirflowService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}
	for k, v := range ctx.Params {
		if _, exists := params[k]; !exists {
			params[k] = v
		}
	}

	switch ctx.Action {
	case "CreateEnvironment":
		return handleCreateEnvironment(params, s.store)
	case "GetEnvironment":
		return handleGetEnvironment(params, s.store)
	case "ListEnvironments":
		return handleListEnvironments(s.store)
	case "UpdateEnvironment":
		return handleUpdateEnvironment(params, s.store)
	case "DeleteEnvironment":
		return handleDeleteEnvironment(params, s.store)
	case "CreateCliToken":
		return handleCreateCliToken(params, s.store)
	case "CreateWebLoginToken":
		return handleCreateWebLoginToken(params, s.store)
	case "TagResource":
		return handleTagResource(params, s.store)
	case "UntagResource":
		return handleUntagResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
