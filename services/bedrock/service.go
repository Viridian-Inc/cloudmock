package bedrock

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// BedrockService is the cloudmock implementation of the AWS Bedrock API.
type BedrockService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new BedrockService for the given AWS account ID and region.
func New(accountID, region string) *BedrockService {
	return &BedrockService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *BedrockService) Name() string { return "bedrock" }

// Actions returns the list of Bedrock API actions supported by this service.
func (s *BedrockService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateModelCustomizationJob", Method: http.MethodPost, IAMAction: "bedrock:CreateModelCustomizationJob"},
		{Name: "GetModelCustomizationJob", Method: http.MethodGet, IAMAction: "bedrock:GetModelCustomizationJob"},
		{Name: "ListModelCustomizationJobs", Method: http.MethodGet, IAMAction: "bedrock:ListModelCustomizationJobs"},
		{Name: "StopModelCustomizationJob", Method: http.MethodPost, IAMAction: "bedrock:StopModelCustomizationJob"},
		{Name: "CreateProvisionedModelThroughput", Method: http.MethodPost, IAMAction: "bedrock:CreateProvisionedModelThroughput"},
		{Name: "GetProvisionedModelThroughput", Method: http.MethodGet, IAMAction: "bedrock:GetProvisionedModelThroughput"},
		{Name: "ListProvisionedModelThroughputs", Method: http.MethodGet, IAMAction: "bedrock:ListProvisionedModelThroughputs"},
		{Name: "UpdateProvisionedModelThroughput", Method: http.MethodPatch, IAMAction: "bedrock:UpdateProvisionedModelThroughput"},
		{Name: "DeleteProvisionedModelThroughput", Method: http.MethodDelete, IAMAction: "bedrock:DeleteProvisionedModelThroughput"},
		{Name: "GetFoundationModel", Method: http.MethodGet, IAMAction: "bedrock:GetFoundationModel"},
		{Name: "ListFoundationModels", Method: http.MethodGet, IAMAction: "bedrock:ListFoundationModels"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "bedrock:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "bedrock:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "bedrock:ListTagsForResource"},
		{Name: "InvokeModel", Method: http.MethodPost, IAMAction: "bedrock:InvokeModel"},
		{Name: "CreateGuardrail", Method: http.MethodPost, IAMAction: "bedrock:CreateGuardrail"},
		{Name: "ApplyGuardrail", Method: http.MethodPost, IAMAction: "bedrock:ApplyGuardrail"},
	}
}

// HealthCheck always returns nil.
func (s *BedrockService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Bedrock request to the appropriate handler.
// Bedrock uses rest-json protocol.
func (s *BedrockService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}

	// Merge URL params into body params for rest-json.
	for k, v := range ctx.Params {
		if _, exists := params[k]; !exists {
			params[k] = v
		}
	}

	switch ctx.Action {
	case "CreateModelCustomizationJob":
		return handleCreateModelCustomizationJob(params, s.store)
	case "GetModelCustomizationJob":
		return handleGetModelCustomizationJob(params, s.store)
	case "ListModelCustomizationJobs":
		return handleListModelCustomizationJobs(s.store)
	case "StopModelCustomizationJob":
		return handleStopModelCustomizationJob(params, s.store)
	case "CreateProvisionedModelThroughput":
		return handleCreateProvisionedModelThroughput(params, s.store)
	case "GetProvisionedModelThroughput":
		return handleGetProvisionedModelThroughput(params, s.store)
	case "ListProvisionedModelThroughputs":
		return handleListProvisionedModelThroughputs(s.store)
	case "UpdateProvisionedModelThroughput":
		return handleUpdateProvisionedModelThroughput(params, s.store)
	case "DeleteProvisionedModelThroughput":
		return handleDeleteProvisionedModelThroughput(params, s.store)
	case "GetFoundationModel":
		return handleGetFoundationModel(params, s.store)
	case "ListFoundationModels":
		return handleListFoundationModels(s.store)
	case "TagResource":
		return handleTagResource(params, s.store)
	case "UntagResource":
		return handleUntagResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	case "InvokeModel":
		return handleInvokeModel(params, s.store)
	case "CreateGuardrail":
		return handleCreateGuardrail(params, s.store)
	case "ApplyGuardrail":
		return handleApplyGuardrail(params, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
