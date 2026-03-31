package support

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// SupportService is the cloudmock implementation of the AWS Support API.
type SupportService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new SupportService for the given AWS account ID and region.
func New(accountID, region string) *SupportService {
	return &SupportService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *SupportService) Name() string { return "support" }

// Actions returns the list of Support API actions supported by this service.
func (s *SupportService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateCase", Method: http.MethodPost, IAMAction: "support:CreateCase"},
		{Name: "DescribeCases", Method: http.MethodPost, IAMAction: "support:DescribeCases"},
		{Name: "ResolveCase", Method: http.MethodPost, IAMAction: "support:ResolveCase"},
		{Name: "DescribeTrustedAdvisorChecks", Method: http.MethodPost, IAMAction: "support:DescribeTrustedAdvisorChecks"},
		{Name: "DescribeTrustedAdvisorCheckResult", Method: http.MethodPost, IAMAction: "support:DescribeTrustedAdvisorCheckResult"},
		{Name: "RefreshTrustedAdvisorCheck", Method: http.MethodPost, IAMAction: "support:RefreshTrustedAdvisorCheck"},
		{Name: "DescribeServices", Method: http.MethodPost, IAMAction: "support:DescribeServices"},
		{Name: "DescribeSeverityLevels", Method: http.MethodPost, IAMAction: "support:DescribeSeverityLevels"},
		{Name: "AddCommunicationToCase", Method: http.MethodPost, IAMAction: "support:AddCommunicationToCase"},
		{Name: "DescribeCommunications", Method: http.MethodPost, IAMAction: "support:DescribeCommunications"},
	}
}

// HealthCheck always returns nil.
func (s *SupportService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Support request to the appropriate handler.
func (s *SupportService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateCase":
		return handleCreateCase(params, s.store)
	case "DescribeCases":
		return handleDescribeCases(params, s.store)
	case "ResolveCase":
		return handleResolveCase(params, s.store)
	case "DescribeTrustedAdvisorChecks":
		return handleDescribeTrustedAdvisorChecks(s.store)
	case "DescribeTrustedAdvisorCheckResult":
		return handleDescribeTrustedAdvisorCheckResult(params, s.store)
	case "RefreshTrustedAdvisorCheck":
		return handleRefreshTrustedAdvisorCheck(params, s.store)
	case "DescribeServices":
		return handleDescribeServices(s.store)
	case "DescribeSeverityLevels":
		return handleDescribeSeverityLevels(s.store)
	case "AddCommunicationToCase":
		return handleAddCommunicationToCase(params, s.store)
	case "DescribeCommunications":
		return handleDescribeCommunications(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
