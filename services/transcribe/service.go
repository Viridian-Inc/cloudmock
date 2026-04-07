package transcribe

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// TranscribeService is the cloudmock implementation of the AWS Transcribe API.
type TranscribeService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new TranscribeService for the given AWS account ID and region.
func New(accountID, region string) *TranscribeService {
	return &TranscribeService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *TranscribeService) Name() string { return "transcribe" }

// Actions returns the list of Transcribe API actions supported by this service.
func (s *TranscribeService) Actions() []service.Action {
	return []service.Action{
		{Name: "StartTranscriptionJob", Method: http.MethodPost, IAMAction: "transcribe:StartTranscriptionJob"},
		{Name: "GetTranscriptionJob", Method: http.MethodPost, IAMAction: "transcribe:GetTranscriptionJob"},
		{Name: "ListTranscriptionJobs", Method: http.MethodPost, IAMAction: "transcribe:ListTranscriptionJobs"},
		{Name: "DeleteTranscriptionJob", Method: http.MethodPost, IAMAction: "transcribe:DeleteTranscriptionJob"},
		{Name: "CreateVocabulary", Method: http.MethodPost, IAMAction: "transcribe:CreateVocabulary"},
		{Name: "GetVocabulary", Method: http.MethodPost, IAMAction: "transcribe:GetVocabulary"},
		{Name: "ListVocabularies", Method: http.MethodPost, IAMAction: "transcribe:ListVocabularies"},
		{Name: "DeleteVocabulary", Method: http.MethodPost, IAMAction: "transcribe:DeleteVocabulary"},
		{Name: "UpdateVocabulary", Method: http.MethodPost, IAMAction: "transcribe:UpdateVocabulary"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "transcribe:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "transcribe:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "transcribe:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *TranscribeService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Transcribe request to the appropriate handler.
// Transcribe uses JSON protocol with TargetPrefix "Transcribe".
func (s *TranscribeService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "StartTranscriptionJob":
		return handleStartTranscriptionJob(ctx, s.store)
	case "GetTranscriptionJob":
		return handleGetTranscriptionJob(ctx, s.store)
	case "ListTranscriptionJobs":
		return handleListTranscriptionJobs(ctx, s.store)
	case "DeleteTranscriptionJob":
		return handleDeleteTranscriptionJob(ctx, s.store)
	case "CreateVocabulary":
		return handleCreateVocabulary(ctx, s.store)
	case "GetVocabulary":
		return handleGetVocabulary(ctx, s.store)
	case "ListVocabularies":
		return handleListVocabularies(ctx, s.store)
	case "DeleteVocabulary":
		return handleDeleteVocabulary(ctx, s.store)
	case "UpdateVocabulary":
		return handleUpdateVocabulary(ctx, s.store)
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
