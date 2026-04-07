package ce

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// CostExplorerService is the cloudmock implementation of the AWS Cost Explorer API.
type CostExplorerService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new CostExplorerService for the given AWS account ID and region.
func New(accountID, region string) *CostExplorerService {
	return &CostExplorerService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *CostExplorerService) Name() string { return "ce" }

// Actions returns the list of Cost Explorer API actions supported by this service.
func (s *CostExplorerService) Actions() []service.Action {
	return []service.Action{
		{Name: "GetCostAndUsage", Method: http.MethodPost, IAMAction: "ce:GetCostAndUsage"},
		{Name: "GetCostForecast", Method: http.MethodPost, IAMAction: "ce:GetCostForecast"},
		{Name: "GetDimensionValues", Method: http.MethodPost, IAMAction: "ce:GetDimensionValues"},
		{Name: "GetTags", Method: http.MethodPost, IAMAction: "ce:GetTags"},
		{Name: "GetReservationUtilization", Method: http.MethodPost, IAMAction: "ce:GetReservationUtilization"},
		{Name: "GetSavingsPlansUtilization", Method: http.MethodPost, IAMAction: "ce:GetSavingsPlansUtilization"},
	}
}

// HealthCheck always returns nil.
func (s *CostExplorerService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Cost Explorer request to the appropriate handler.
func (s *CostExplorerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "GetCostAndUsage":
		return handleGetCostAndUsage(params, s.store)
	case "GetCostForecast":
		return handleGetCostForecast(params, s.store)
	case "GetDimensionValues":
		return handleGetDimensionValues(params, s.store)
	case "GetTags":
		return handleGetTags(s.store)
	case "GetReservationUtilization":
		return handleGetReservationUtilization(params, s.store)
	case "GetSavingsPlansUtilization":
		return handleGetSavingsPlansUtilization(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
