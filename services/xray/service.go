package xray

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// XRayService implements the AWS X-Ray API.
type XRayService struct {
	store *Store
}

// New returns a new XRayService for the given AWS account ID and region.
func New(accountID, region string) *XRayService {
	return &XRayService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *XRayService) Name() string { return "xray" }

// Actions returns the list of X-Ray API actions supported by this service.
func (s *XRayService) Actions() []service.Action {
	return []service.Action{
		{Name: "PutTraceSegments", Method: http.MethodPost, IAMAction: "xray:PutTraceSegments"},
		{Name: "BatchGetTraces", Method: http.MethodPost, IAMAction: "xray:BatchGetTraces"},
		{Name: "GetTraceSummaries", Method: http.MethodPost, IAMAction: "xray:GetTraceSummaries"},
		{Name: "GetTraceGraph", Method: http.MethodPost, IAMAction: "xray:GetTraceGraph"},
		{Name: "GetSamplingRules", Method: http.MethodPost, IAMAction: "xray:GetSamplingRules"},
		{Name: "CreateSamplingRule", Method: http.MethodPost, IAMAction: "xray:CreateSamplingRule"},
		{Name: "UpdateSamplingRule", Method: http.MethodPost, IAMAction: "xray:UpdateSamplingRule"},
		{Name: "DeleteSamplingRule", Method: http.MethodPost, IAMAction: "xray:DeleteSamplingRule"},
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "xray:CreateGroup"},
		{Name: "GetGroup", Method: http.MethodPost, IAMAction: "xray:GetGroup"},
		{Name: "GetGroups", Method: http.MethodPost, IAMAction: "xray:GetGroups"},
		{Name: "UpdateGroup", Method: http.MethodPost, IAMAction: "xray:UpdateGroup"},
		{Name: "DeleteGroup", Method: http.MethodPost, IAMAction: "xray:DeleteGroup"},
		{Name: "PutEncryptionConfig", Method: http.MethodPost, IAMAction: "xray:PutEncryptionConfig"},
		{Name: "GetEncryptionConfig", Method: http.MethodPost, IAMAction: "xray:GetEncryptionConfig"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "xray:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "xray:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "xray:ListTagsForResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *XRayService) HealthCheck() error { return nil }

// HandleRequest routes an incoming X-Ray request to the appropriate handler.
// X-Ray uses the JSON protocol; the action is parsed from X-Amz-Target or the URL path.
func (s *XRayService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
	case "PutTraceSegments":
		return handlePutTraceSegments(params, s.store)
	case "BatchGetTraces":
		return handleBatchGetTraces(params, s.store)
	case "GetTraceSummaries":
		return handleGetTraceSummaries(params, s.store)
	case "GetTraceGraph":
		return handleGetTraceGraph(params, s.store)
	case "GetSamplingRules":
		return handleGetSamplingRules(params, s.store)
	case "CreateSamplingRule":
		return handleCreateSamplingRule(params, s.store)
	case "UpdateSamplingRule":
		return handleUpdateSamplingRule(params, s.store)
	case "DeleteSamplingRule":
		return handleDeleteSamplingRule(params, s.store)
	case "CreateGroup":
		return handleCreateGroup(params, s.store)
	case "GetGroup":
		return handleGetGroup(params, s.store)
	case "GetGroups":
		return handleGetGroups(params, s.store)
	case "UpdateGroup":
		return handleUpdateGroup(params, s.store)
	case "DeleteGroup":
		return handleDeleteGroup(params, s.store)
	case "PutEncryptionConfig":
		return handlePutEncryptionConfig(params, s.store)
	case "GetEncryptionConfig":
		return handleGetEncryptionConfig(params, s.store)
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
