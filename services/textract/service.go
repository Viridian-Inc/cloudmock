package textract

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// TextractService is the cloudmock implementation of the AWS Textract API.
type TextractService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new TextractService for the given AWS account ID and region.
func New(accountID, region string) *TextractService {
	return &TextractService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *TextractService) Name() string { return "textract" }

// Actions returns the list of Textract API actions supported by this service.
func (s *TextractService) Actions() []service.Action {
	return []service.Action{
		{Name: "StartDocumentTextDetection", Method: http.MethodPost, IAMAction: "textract:StartDocumentTextDetection"},
		{Name: "GetDocumentTextDetection", Method: http.MethodPost, IAMAction: "textract:GetDocumentTextDetection"},
		{Name: "StartDocumentAnalysis", Method: http.MethodPost, IAMAction: "textract:StartDocumentAnalysis"},
		{Name: "GetDocumentAnalysis", Method: http.MethodPost, IAMAction: "textract:GetDocumentAnalysis"},
		{Name: "StartExpenseAnalysis", Method: http.MethodPost, IAMAction: "textract:StartExpenseAnalysis"},
		{Name: "GetExpenseAnalysis", Method: http.MethodPost, IAMAction: "textract:GetExpenseAnalysis"},
		{Name: "AnalyzeDocument", Method: http.MethodPost, IAMAction: "textract:AnalyzeDocument"},
		{Name: "DetectDocumentText", Method: http.MethodPost, IAMAction: "textract:DetectDocumentText"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "textract:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "textract:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "textract:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *TextractService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Textract request to the appropriate handler.
// Textract uses JSON protocol with TargetPrefix "Textract".
func (s *TextractService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "StartDocumentTextDetection":
		return handleStartDocumentTextDetection(ctx, s.store)
	case "GetDocumentTextDetection":
		return handleGetDocumentTextDetection(ctx, s.store)
	case "StartDocumentAnalysis":
		return handleStartDocumentAnalysis(ctx, s.store)
	case "GetDocumentAnalysis":
		return handleGetDocumentAnalysis(ctx, s.store)
	case "StartExpenseAnalysis":
		return handleStartExpenseAnalysis(ctx, s.store)
	case "GetExpenseAnalysis":
		return handleGetExpenseAnalysis(ctx, s.store)
	case "AnalyzeDocument":
		return handleAnalyzeDocument(ctx, s.store)
	case "DetectDocumentText":
		return handleDetectDocumentText(ctx, s.store)
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
