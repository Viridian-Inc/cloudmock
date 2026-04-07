package translate

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AppliedTerminology struct {
	Name *string `json:"Name,omitempty"`
	Terms []Term `json:"Terms,omitempty"`
}

type CreateParallelDataRequest struct {
	ClientToken string `json:"ClientToken,omitempty"`
	Description *string `json:"Description,omitempty"`
	EncryptionKey *EncryptionKey `json:"EncryptionKey,omitempty"`
	Name string `json:"Name,omitempty"`
	ParallelDataConfig ParallelDataConfig `json:"ParallelDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateParallelDataResponse struct {
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DeleteParallelDataRequest struct {
	Name string `json:"Name,omitempty"`
}

type DeleteParallelDataResponse struct {
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DeleteTerminologyRequest struct {
	Name string `json:"Name,omitempty"`
}

type DescribeTextTranslationJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeTextTranslationJobResponse struct {
	TextTranslationJobProperties *TextTranslationJobProperties `json:"TextTranslationJobProperties,omitempty"`
}

type Document struct {
	Content []byte `json:"Content,omitempty"`
	ContentType string `json:"ContentType,omitempty"`
}

type EncryptionKey struct {
	Id string `json:"Id,omitempty"`
	Type string `json:"Type,omitempty"`
}

type GetParallelDataRequest struct {
	Name string `json:"Name,omitempty"`
}

type GetParallelDataResponse struct {
	AuxiliaryDataLocation *ParallelDataDataLocation `json:"AuxiliaryDataLocation,omitempty"`
	DataLocation *ParallelDataDataLocation `json:"DataLocation,omitempty"`
	LatestUpdateAttemptAuxiliaryDataLocation *ParallelDataDataLocation `json:"LatestUpdateAttemptAuxiliaryDataLocation,omitempty"`
	ParallelDataProperties *ParallelDataProperties `json:"ParallelDataProperties,omitempty"`
}

type GetTerminologyRequest struct {
	Name string `json:"Name,omitempty"`
	TerminologyDataFormat *string `json:"TerminologyDataFormat,omitempty"`
}

type GetTerminologyResponse struct {
	AuxiliaryDataLocation *TerminologyDataLocation `json:"AuxiliaryDataLocation,omitempty"`
	TerminologyDataLocation *TerminologyDataLocation `json:"TerminologyDataLocation,omitempty"`
	TerminologyProperties *TerminologyProperties `json:"TerminologyProperties,omitempty"`
}

type ImportTerminologyRequest struct {
	Description *string `json:"Description,omitempty"`
	EncryptionKey *EncryptionKey `json:"EncryptionKey,omitempty"`
	MergeStrategy string `json:"MergeStrategy,omitempty"`
	Name string `json:"Name,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	TerminologyData TerminologyData `json:"TerminologyData,omitempty"`
}

type ImportTerminologyResponse struct {
	AuxiliaryDataLocation *TerminologyDataLocation `json:"AuxiliaryDataLocation,omitempty"`
	TerminologyProperties *TerminologyProperties `json:"TerminologyProperties,omitempty"`
}

type InputDataConfig struct {
	ContentType string `json:"ContentType,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type JobDetails struct {
	DocumentsWithErrorsCount int `json:"DocumentsWithErrorsCount,omitempty"`
	InputDocumentsCount int `json:"InputDocumentsCount,omitempty"`
	TranslatedDocumentsCount int `json:"TranslatedDocumentsCount,omitempty"`
}

type Language struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	LanguageName string `json:"LanguageName,omitempty"`
}

type ListLanguagesRequest struct {
	DisplayLanguageCode *string `json:"DisplayLanguageCode,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListLanguagesResponse struct {
	DisplayLanguageCode *string `json:"DisplayLanguageCode,omitempty"`
	Languages []Language `json:"Languages,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListParallelDataRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListParallelDataResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ParallelDataPropertiesList []ParallelDataProperties `json:"ParallelDataPropertiesList,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags []Tag `json:"Tags,omitempty"`
}

type ListTerminologiesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListTerminologiesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	TerminologyPropertiesList []TerminologyProperties `json:"TerminologyPropertiesList,omitempty"`
}

type ListTextTranslationJobsRequest struct {
	Filter *TextTranslationJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListTextTranslationJobsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	TextTranslationJobPropertiesList []TextTranslationJobProperties `json:"TextTranslationJobPropertiesList,omitempty"`
}

type OutputDataConfig struct {
	EncryptionKey *EncryptionKey `json:"EncryptionKey,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type ParallelDataConfig struct {
	Format *string `json:"Format,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type ParallelDataDataLocation struct {
	Location string `json:"Location,omitempty"`
	RepositoryType string `json:"RepositoryType,omitempty"`
}

type ParallelDataProperties struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	EncryptionKey *EncryptionKey `json:"EncryptionKey,omitempty"`
	FailedRecordCount int64 `json:"FailedRecordCount,omitempty"`
	ImportedDataSize int64 `json:"ImportedDataSize,omitempty"`
	ImportedRecordCount int64 `json:"ImportedRecordCount,omitempty"`
	LastUpdatedAt *time.Time `json:"LastUpdatedAt,omitempty"`
	LatestUpdateAttemptAt *time.Time `json:"LatestUpdateAttemptAt,omitempty"`
	LatestUpdateAttemptStatus *string `json:"LatestUpdateAttemptStatus,omitempty"`
	Message *string `json:"Message,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParallelDataConfig *ParallelDataConfig `json:"ParallelDataConfig,omitempty"`
	SkippedRecordCount int64 `json:"SkippedRecordCount,omitempty"`
	SourceLanguageCode *string `json:"SourceLanguageCode,omitempty"`
	Status *string `json:"Status,omitempty"`
	TargetLanguageCodes []string `json:"TargetLanguageCodes,omitempty"`
}

type StartTextTranslationJobRequest struct {
	ClientToken string `json:"ClientToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	ParallelDataNames []string `json:"ParallelDataNames,omitempty"`
	Settings *TranslationSettings `json:"Settings,omitempty"`
	SourceLanguageCode string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCodes []string `json:"TargetLanguageCodes,omitempty"`
	TerminologyNames []string `json:"TerminologyNames,omitempty"`
}

type StartTextTranslationJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopTextTranslationJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopTextTranslationJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
}

type Term struct {
	SourceText *string `json:"SourceText,omitempty"`
	TargetText *string `json:"TargetText,omitempty"`
}

type TerminologyData struct {
	Directionality *string `json:"Directionality,omitempty"`
	File []byte `json:"File,omitempty"`
	Format string `json:"Format,omitempty"`
}

type TerminologyDataLocation struct {
	Location string `json:"Location,omitempty"`
	RepositoryType string `json:"RepositoryType,omitempty"`
}

type TerminologyProperties struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Directionality *string `json:"Directionality,omitempty"`
	EncryptionKey *EncryptionKey `json:"EncryptionKey,omitempty"`
	Format *string `json:"Format,omitempty"`
	LastUpdatedAt *time.Time `json:"LastUpdatedAt,omitempty"`
	Message *string `json:"Message,omitempty"`
	Name *string `json:"Name,omitempty"`
	SizeBytes int `json:"SizeBytes,omitempty"`
	SkippedTermCount int `json:"SkippedTermCount,omitempty"`
	SourceLanguageCode *string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCodes []string `json:"TargetLanguageCodes,omitempty"`
	TermCount int `json:"TermCount,omitempty"`
}

type TextTranslationJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmittedAfterTime *time.Time `json:"SubmittedAfterTime,omitempty"`
	SubmittedBeforeTime *time.Time `json:"SubmittedBeforeTime,omitempty"`
}

type TextTranslationJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobDetails *JobDetails `json:"JobDetails,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	ParallelDataNames []string `json:"ParallelDataNames,omitempty"`
	Settings *TranslationSettings `json:"Settings,omitempty"`
	SourceLanguageCode *string `json:"SourceLanguageCode,omitempty"`
	SubmittedTime *time.Time `json:"SubmittedTime,omitempty"`
	TargetLanguageCodes []string `json:"TargetLanguageCodes,omitempty"`
	TerminologyNames []string `json:"TerminologyNames,omitempty"`
}

type TranslateDocumentRequest struct {
	Document Document `json:"Document,omitempty"`
	Settings *TranslationSettings `json:"Settings,omitempty"`
	SourceLanguageCode string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCode string `json:"TargetLanguageCode,omitempty"`
	TerminologyNames []string `json:"TerminologyNames,omitempty"`
}

type TranslateDocumentResponse struct {
	AppliedSettings *TranslationSettings `json:"AppliedSettings,omitempty"`
	AppliedTerminologies []AppliedTerminology `json:"AppliedTerminologies,omitempty"`
	SourceLanguageCode string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCode string `json:"TargetLanguageCode,omitempty"`
	TranslatedDocument TranslatedDocument `json:"TranslatedDocument,omitempty"`
}

type TranslateTextRequest struct {
	Settings *TranslationSettings `json:"Settings,omitempty"`
	SourceLanguageCode string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCode string `json:"TargetLanguageCode,omitempty"`
	TerminologyNames []string `json:"TerminologyNames,omitempty"`
	Text string `json:"Text,omitempty"`
}

type TranslateTextResponse struct {
	AppliedSettings *TranslationSettings `json:"AppliedSettings,omitempty"`
	AppliedTerminologies []AppliedTerminology `json:"AppliedTerminologies,omitempty"`
	SourceLanguageCode string `json:"SourceLanguageCode,omitempty"`
	TargetLanguageCode string `json:"TargetLanguageCode,omitempty"`
	TranslatedText string `json:"TranslatedText,omitempty"`
}

type TranslatedDocument struct {
	Content []byte `json:"Content,omitempty"`
}

type TranslationSettings struct {
	Brevity *string `json:"Brevity,omitempty"`
	Formality *string `json:"Formality,omitempty"`
	Profanity *string `json:"Profanity,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"TagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateParallelDataRequest struct {
	ClientToken string `json:"ClientToken,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
	ParallelDataConfig ParallelDataConfig `json:"ParallelDataConfig,omitempty"`
}

type UpdateParallelDataResponse struct {
	LatestUpdateAttemptAt *time.Time `json:"LatestUpdateAttemptAt,omitempty"`
	LatestUpdateAttemptStatus *string `json:"LatestUpdateAttemptStatus,omitempty"`
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
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
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateParallelDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateParallelData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateParallelData"})
}

func handleDeleteParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteParallelDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteParallelData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteParallelData"})
}

func handleDeleteTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTerminologyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTerminology business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTerminology"})
}

func handleDescribeTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTextTranslationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTextTranslationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTextTranslationJob"})
}

func handleGetParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetParallelDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetParallelData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetParallelData"})
}

func handleGetTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTerminologyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetTerminology business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetTerminology"})
}

func handleImportTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ImportTerminologyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ImportTerminology business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ImportTerminology"})
}

func handleListLanguages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListLanguagesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListLanguages business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListLanguages"})
}

func handleListParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListParallelDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListParallelData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListParallelData"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListTerminologies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTerminologiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTerminologies business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTerminologies"})
}

func handleListTextTranslationJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTextTranslationJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTextTranslationJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTextTranslationJobs"})
}

func handleStartTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartTextTranslationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartTextTranslationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartTextTranslationJob"})
}

func handleStopTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopTextTranslationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopTextTranslationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopTextTranslationJob"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleTranslateDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TranslateDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TranslateDocument business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TranslateDocument"})
}

func handleTranslateText(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TranslateTextRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TranslateText business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TranslateText"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateParallelDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateParallelData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateParallelData"})
}

