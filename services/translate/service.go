package translate

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS translate service.
type Service struct {
	store *Store
}

// New returns a new translate Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "translate" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateParallelData", Method: http.MethodPost, IAMAction: "translate:CreateParallelData"},
		{Name: "DeleteParallelData", Method: http.MethodPost, IAMAction: "translate:DeleteParallelData"},
		{Name: "DeleteTerminology", Method: http.MethodPost, IAMAction: "translate:DeleteTerminology"},
		{Name: "DescribeTextTranslationJob", Method: http.MethodPost, IAMAction: "translate:DescribeTextTranslationJob"},
		{Name: "GetParallelData", Method: http.MethodPost, IAMAction: "translate:GetParallelData"},
		{Name: "GetTerminology", Method: http.MethodPost, IAMAction: "translate:GetTerminology"},
		{Name: "ImportTerminology", Method: http.MethodPost, IAMAction: "translate:ImportTerminology"},
		{Name: "ListLanguages", Method: http.MethodPost, IAMAction: "translate:ListLanguages"},
		{Name: "ListParallelData", Method: http.MethodPost, IAMAction: "translate:ListParallelData"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "translate:ListTagsForResource"},
		{Name: "ListTerminologies", Method: http.MethodPost, IAMAction: "translate:ListTerminologies"},
		{Name: "ListTextTranslationJobs", Method: http.MethodPost, IAMAction: "translate:ListTextTranslationJobs"},
		{Name: "StartTextTranslationJob", Method: http.MethodPost, IAMAction: "translate:StartTextTranslationJob"},
		{Name: "StopTextTranslationJob", Method: http.MethodPost, IAMAction: "translate:StopTextTranslationJob"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "translate:TagResource"},
		{Name: "TranslateDocument", Method: http.MethodPost, IAMAction: "translate:TranslateDocument"},
		{Name: "TranslateText", Method: http.MethodPost, IAMAction: "translate:TranslateText"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "translate:UntagResource"},
		{Name: "UpdateParallelData", Method: http.MethodPost, IAMAction: "translate:UpdateParallelData"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateParallelData":
		return handleCreateParallelData(ctx, s.store)
	case "DeleteParallelData":
		return handleDeleteParallelData(ctx, s.store)
	case "DeleteTerminology":
		return handleDeleteTerminology(ctx, s.store)
	case "DescribeTextTranslationJob":
		return handleDescribeTextTranslationJob(ctx, s.store)
	case "GetParallelData":
		return handleGetParallelData(ctx, s.store)
	case "GetTerminology":
		return handleGetTerminology(ctx, s.store)
	case "ImportTerminology":
		return handleImportTerminology(ctx, s.store)
	case "ListLanguages":
		return handleListLanguages(ctx, s.store)
	case "ListParallelData":
		return handleListParallelData(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListTerminologies":
		return handleListTerminologies(ctx, s.store)
	case "ListTextTranslationJobs":
		return handleListTextTranslationJobs(ctx, s.store)
	case "StartTextTranslationJob":
		return handleStartTextTranslationJob(ctx, s.store)
	case "StopTextTranslationJob":
		return handleStopTextTranslationJob(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "TranslateDocument":
		return handleTranslateDocument(ctx, s.store)
	case "TranslateText":
		return handleTranslateText(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateParallelData":
		return handleUpdateParallelData(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
