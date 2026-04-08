package polly

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AudioEvent struct {
	AudioChunk []byte `json:"AudioChunk,omitempty"`
}

type CloseStreamEvent struct {
}

type DeleteLexiconInput struct {
	Name string `json:"LexiconName,omitempty"`
}

type DeleteLexiconOutput struct {
}

type DescribeVoicesInput struct {
	Engine *string `json:"Engine,omitempty"`
	IncludeAdditionalLanguageCodes bool `json:"IncludeAdditionalLanguageCodes,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeVoicesOutput struct {
	NextToken *string `json:"NextToken,omitempty"`
	Voices []Voice `json:"Voices,omitempty"`
}

type FlushStreamConfiguration struct {
	Force bool `json:"Force,omitempty"`
}

type GetLexiconInput struct {
	Name string `json:"LexiconName,omitempty"`
}

type GetLexiconOutput struct {
	Lexicon *Lexicon `json:"Lexicon,omitempty"`
	LexiconAttributes *LexiconAttributes `json:"LexiconAttributes,omitempty"`
}

type GetSpeechSynthesisTaskInput struct {
	TaskId string `json:"TaskId,omitempty"`
}

type GetSpeechSynthesisTaskOutput struct {
	SynthesisTask *SynthesisTask `json:"SynthesisTask,omitempty"`
}

type Lexicon struct {
	Content *string `json:"Content,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type LexiconAttributes struct {
	Alphabet *string `json:"Alphabet,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	LastModified *time.Time `json:"LastModified,omitempty"`
	LexemesCount int `json:"LexemesCount,omitempty"`
	LexiconArn *string `json:"LexiconArn,omitempty"`
	Size int `json:"Size,omitempty"`
}

type LexiconDescription struct {
	Attributes *LexiconAttributes `json:"Attributes,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ListLexiconsInput struct {
	NextToken *string `json:"NextToken,omitempty"`
}

type ListLexiconsOutput struct {
	Lexicons []LexiconDescription `json:"Lexicons,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListSpeechSynthesisTasksInput struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type ListSpeechSynthesisTasksOutput struct {
	NextToken *string `json:"NextToken,omitempty"`
	SynthesisTasks []SynthesisTask `json:"SynthesisTasks,omitempty"`
}

type PutLexiconInput struct {
	Content string `json:"Content,omitempty"`
	Name string `json:"LexiconName,omitempty"`
}

type PutLexiconOutput struct {
}

type ServiceFailureException struct {
	Message *string `json:"message,omitempty"`
}

type ServiceQuotaExceededException struct {
	Message string `json:"message,omitempty"`
	QuotaCode string `json:"quotaCode,omitempty"`
	ServiceCode string `json:"serviceCode,omitempty"`
}

type StartSpeechSynthesisStreamActionStream struct {
	CloseStreamEvent *CloseStreamEvent `json:"CloseStreamEvent,omitempty"`
	TextEvent *TextEvent `json:"TextEvent,omitempty"`
}

type StartSpeechSynthesisStreamEventStream struct {
	AudioEvent *AudioEvent `json:"AudioEvent,omitempty"`
	ServiceFailureException *ServiceFailureException `json:"ServiceFailureException,omitempty"`
	ServiceQuotaExceededException *ServiceQuotaExceededException `json:"ServiceQuotaExceededException,omitempty"`
	StreamClosedEvent *StreamClosedEvent `json:"StreamClosedEvent,omitempty"`
	ThrottlingException *ThrottlingException `json:"ThrottlingException,omitempty"`
	ValidationException *ValidationException `json:"ValidationException,omitempty"`
}

type StartSpeechSynthesisStreamInput struct {
	ActionStream *StartSpeechSynthesisStreamActionStream `json:"ActionStream,omitempty"`
	Engine string `json:"x-amzn-Engine,omitempty"`
	LanguageCode *string `json:"x-amzn-LanguageCode,omitempty"`
	LexiconNames []string `json:"x-amzn-LexiconNames,omitempty"`
	OutputFormat string `json:"x-amzn-OutputFormat,omitempty"`
	SampleRate *string `json:"x-amzn-SampleRate,omitempty"`
	VoiceId string `json:"x-amzn-VoiceId,omitempty"`
}

type StartSpeechSynthesisStreamOutput struct {
	EventStream *StartSpeechSynthesisStreamEventStream `json:"EventStream,omitempty"`
}

type StartSpeechSynthesisTaskInput struct {
	Engine *string `json:"Engine,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	LexiconNames []string `json:"LexiconNames,omitempty"`
	OutputFormat string `json:"OutputFormat,omitempty"`
	OutputS3BucketName string `json:"OutputS3BucketName,omitempty"`
	OutputS3KeyPrefix *string `json:"OutputS3KeyPrefix,omitempty"`
	SampleRate *string `json:"SampleRate,omitempty"`
	SnsTopicArn *string `json:"SnsTopicArn,omitempty"`
	SpeechMarkTypes []string `json:"SpeechMarkTypes,omitempty"`
	Text string `json:"Text,omitempty"`
	TextType *string `json:"TextType,omitempty"`
	VoiceId string `json:"VoiceId,omitempty"`
}

type StartSpeechSynthesisTaskOutput struct {
	SynthesisTask *SynthesisTask `json:"SynthesisTask,omitempty"`
}

type StreamClosedEvent struct {
	RequestCharacters int `json:"RequestCharacters,omitempty"`
}

type SynthesisTask struct {
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	Engine *string `json:"Engine,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	LexiconNames []string `json:"LexiconNames,omitempty"`
	OutputFormat *string `json:"OutputFormat,omitempty"`
	OutputUri *string `json:"OutputUri,omitempty"`
	RequestCharacters int `json:"RequestCharacters,omitempty"`
	SampleRate *string `json:"SampleRate,omitempty"`
	SnsTopicArn *string `json:"SnsTopicArn,omitempty"`
	SpeechMarkTypes []string `json:"SpeechMarkTypes,omitempty"`
	TaskId *string `json:"TaskId,omitempty"`
	TaskStatus *string `json:"TaskStatus,omitempty"`
	TaskStatusReason *string `json:"TaskStatusReason,omitempty"`
	TextType *string `json:"TextType,omitempty"`
	VoiceId *string `json:"VoiceId,omitempty"`
}

type SynthesizeSpeechInput struct {
	Engine *string `json:"Engine,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	LexiconNames []string `json:"LexiconNames,omitempty"`
	OutputFormat string `json:"OutputFormat,omitempty"`
	SampleRate *string `json:"SampleRate,omitempty"`
	SpeechMarkTypes []string `json:"SpeechMarkTypes,omitempty"`
	Text string `json:"Text,omitempty"`
	TextType *string `json:"TextType,omitempty"`
	VoiceId string `json:"VoiceId,omitempty"`
}

type SynthesizeSpeechOutput struct {
	AudioStream []byte `json:"AudioStream,omitempty"`
	ContentType *string `json:"Content-Type,omitempty"`
	RequestCharacters int `json:"x-amzn-RequestCharacters,omitempty"`
}

type TextEvent struct {
	FlushStreamConfiguration *FlushStreamConfiguration `json:"FlushStreamConfiguration,omitempty"`
	Text string `json:"Text,omitempty"`
	TextType *string `json:"TextType,omitempty"`
}

type ThrottlingException struct {
	Message *string `json:"message,omitempty"`
	ThrottlingReasons []ThrottlingReason `json:"throttlingReasons,omitempty"`
}

type ThrottlingReason struct {
	Reason *string `json:"reason,omitempty"`
	Resource *string `json:"resource,omitempty"`
}

type ValidationException struct {
	Fields []ValidationExceptionField `json:"fields,omitempty"`
	Message string `json:"message,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ValidationExceptionField struct {
	Message string `json:"message,omitempty"`
	Name string `json:"name,omitempty"`
}

type Voice struct {
	AdditionalLanguageCodes []string `json:"AdditionalLanguageCodes,omitempty"`
	Gender *string `json:"Gender,omitempty"`
	Id *string `json:"Id,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	LanguageName *string `json:"LanguageName,omitempty"`
	Name *string `json:"Name,omitempty"`
	SupportedEngines []string `json:"SupportedEngines,omitempty"`
}



// ── Handler helpers ──────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleDeleteLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteLexiconInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteLexicon business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteLexicon"})
}

func handleDescribeVoices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeVoicesInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeVoices business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeVoices"})
}

func handleGetLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetLexiconInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetLexicon business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetLexicon"})
}

func handleGetSpeechSynthesisTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSpeechSynthesisTaskInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSpeechSynthesisTask business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSpeechSynthesisTask"})
}

func handleListLexicons(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListLexiconsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListLexicons business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListLexicons"})
}

func handleListSpeechSynthesisTasks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListSpeechSynthesisTasksInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListSpeechSynthesisTasks business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListSpeechSynthesisTasks"})
}

func handlePutLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutLexiconInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutLexicon business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutLexicon"})
}

func handleStartSpeechSynthesisStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartSpeechSynthesisStreamInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartSpeechSynthesisStream business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartSpeechSynthesisStream"})
}

func handleStartSpeechSynthesisTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartSpeechSynthesisTaskInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartSpeechSynthesisTask business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartSpeechSynthesisTask"})
}

func handleSynthesizeSpeech(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SynthesizeSpeechInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SynthesizeSpeech business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SynthesizeSpeech"})
}

