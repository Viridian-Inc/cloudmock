package polly

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS polly service.
type Service struct {
	store *Store
}

// New returns a new polly Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "polly" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "DeleteLexicon", Method: http.MethodDelete, IAMAction: "polly:DeleteLexicon"},
		{Name: "DescribeVoices", Method: http.MethodGet, IAMAction: "polly:DescribeVoices"},
		{Name: "GetLexicon", Method: http.MethodGet, IAMAction: "polly:GetLexicon"},
		{Name: "GetSpeechSynthesisTask", Method: http.MethodGet, IAMAction: "polly:GetSpeechSynthesisTask"},
		{Name: "ListLexicons", Method: http.MethodGet, IAMAction: "polly:ListLexicons"},
		{Name: "ListSpeechSynthesisTasks", Method: http.MethodGet, IAMAction: "polly:ListSpeechSynthesisTasks"},
		{Name: "PutLexicon", Method: http.MethodPut, IAMAction: "polly:PutLexicon"},
		{Name: "StartSpeechSynthesisStream", Method: http.MethodPost, IAMAction: "polly:StartSpeechSynthesisStream"},
		{Name: "StartSpeechSynthesisTask", Method: http.MethodPost, IAMAction: "polly:StartSpeechSynthesisTask"},
		{Name: "SynthesizeSpeech", Method: http.MethodPost, IAMAction: "polly:SynthesizeSpeech"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "DeleteLexicon":
		return handleDeleteLexicon(ctx, s.store)
	case "DescribeVoices":
		return handleDescribeVoices(ctx, s.store)
	case "GetLexicon":
		return handleGetLexicon(ctx, s.store)
	case "GetSpeechSynthesisTask":
		return handleGetSpeechSynthesisTask(ctx, s.store)
	case "ListLexicons":
		return handleListLexicons(ctx, s.store)
	case "ListSpeechSynthesisTasks":
		return handleListSpeechSynthesisTasks(ctx, s.store)
	case "PutLexicon":
		return handlePutLexicon(ctx, s.store)
	case "StartSpeechSynthesisStream":
		return handleStartSpeechSynthesisStream(ctx, s.store)
	case "StartSpeechSynthesisTask":
		return handleStartSpeechSynthesisTask(ctx, s.store)
	case "SynthesizeSpeech":
		return handleSynthesizeSpeech(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
