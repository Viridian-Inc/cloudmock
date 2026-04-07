package comprehend

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AugmentedManifestsListItem struct {
	AnnotationDataS3Uri *string `json:"AnnotationDataS3Uri,omitempty"`
	AttributeNames []string `json:"AttributeNames,omitempty"`
	DocumentType *string `json:"DocumentType,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
	SourceDocumentsS3Uri *string `json:"SourceDocumentsS3Uri,omitempty"`
	Split *string `json:"Split,omitempty"`
}

type BatchDetectDominantLanguageItemResult struct {
	Index int `json:"Index,omitempty"`
	Languages []DominantLanguage `json:"Languages,omitempty"`
}

type BatchDetectDominantLanguageRequest struct {
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectDominantLanguageResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectDominantLanguageItemResult `json:"ResultList,omitempty"`
}

type BatchDetectEntitiesItemResult struct {
	Entities []Entity `json:"Entities,omitempty"`
	Index int `json:"Index,omitempty"`
}

type BatchDetectEntitiesRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectEntitiesResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectEntitiesItemResult `json:"ResultList,omitempty"`
}

type BatchDetectKeyPhrasesItemResult struct {
	Index int `json:"Index,omitempty"`
	KeyPhrases []KeyPhrase `json:"KeyPhrases,omitempty"`
}

type BatchDetectKeyPhrasesRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectKeyPhrasesResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectKeyPhrasesItemResult `json:"ResultList,omitempty"`
}

type BatchDetectSentimentItemResult struct {
	Index int `json:"Index,omitempty"`
	Sentiment *string `json:"Sentiment,omitempty"`
	SentimentScore *SentimentScore `json:"SentimentScore,omitempty"`
}

type BatchDetectSentimentRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectSentimentResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectSentimentItemResult `json:"ResultList,omitempty"`
}

type BatchDetectSyntaxItemResult struct {
	Index int `json:"Index,omitempty"`
	SyntaxTokens []SyntaxToken `json:"SyntaxTokens,omitempty"`
}

type BatchDetectSyntaxRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectSyntaxResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectSyntaxItemResult `json:"ResultList,omitempty"`
}

type BatchDetectTargetedSentimentItemResult struct {
	Entities []TargetedSentimentEntity `json:"Entities,omitempty"`
	Index int `json:"Index,omitempty"`
}

type BatchDetectTargetedSentimentRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextList []string `json:"TextList,omitempty"`
}

type BatchDetectTargetedSentimentResponse struct {
	ErrorList []BatchItemError `json:"ErrorList,omitempty"`
	ResultList []BatchDetectTargetedSentimentItemResult `json:"ResultList,omitempty"`
}

type BatchItemError struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	Index int `json:"Index,omitempty"`
}

type Block struct {
	BlockType *string `json:"BlockType,omitempty"`
	Geometry *Geometry `json:"Geometry,omitempty"`
	Id *string `json:"Id,omitempty"`
	Page int `json:"Page,omitempty"`
	Relationships []RelationshipsListItem `json:"Relationships,omitempty"`
	Text *string `json:"Text,omitempty"`
}

type BlockReference struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	BlockId *string `json:"BlockId,omitempty"`
	ChildBlocks []ChildBlock `json:"ChildBlocks,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
}

type BoundingBox struct {
	Height float64 `json:"Height,omitempty"`
	Left float64 `json:"Left,omitempty"`
	Top float64 `json:"Top,omitempty"`
	Width float64 `json:"Width,omitempty"`
}

type ChildBlock struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	ChildBlockId *string `json:"ChildBlockId,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
}

type ClassifierEvaluationMetrics struct {
	Accuracy float64 `json:"Accuracy,omitempty"`
	F1Score float64 `json:"F1Score,omitempty"`
	HammingLoss float64 `json:"HammingLoss,omitempty"`
	MicroF1Score float64 `json:"MicroF1Score,omitempty"`
	MicroPrecision float64 `json:"MicroPrecision,omitempty"`
	MicroRecall float64 `json:"MicroRecall,omitempty"`
	Precision float64 `json:"Precision,omitempty"`
	Recall float64 `json:"Recall,omitempty"`
}

type ClassifierMetadata struct {
	EvaluationMetrics *ClassifierEvaluationMetrics `json:"EvaluationMetrics,omitempty"`
	NumberOfLabels int `json:"NumberOfLabels,omitempty"`
	NumberOfTestDocuments int `json:"NumberOfTestDocuments,omitempty"`
	NumberOfTrainedDocuments int `json:"NumberOfTrainedDocuments,omitempty"`
}

type ClassifyDocumentRequest struct {
	Bytes []byte `json:"Bytes,omitempty"`
	DocumentReaderConfig *DocumentReaderConfig `json:"DocumentReaderConfig,omitempty"`
	EndpointArn string `json:"EndpointArn,omitempty"`
	Text *string `json:"Text,omitempty"`
}

type ClassifyDocumentResponse struct {
	Classes []DocumentClass `json:"Classes,omitempty"`
	DocumentMetadata *DocumentMetadata `json:"DocumentMetadata,omitempty"`
	DocumentType []DocumentTypeListItem `json:"DocumentType,omitempty"`
	Errors []ErrorsListItem `json:"Errors,omitempty"`
	Labels []DocumentLabel `json:"Labels,omitempty"`
	Warnings []WarningsListItem `json:"Warnings,omitempty"`
}

type ContainsPiiEntitiesRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type ContainsPiiEntitiesResponse struct {
	Labels []EntityLabel `json:"Labels,omitempty"`
}

type CreateDatasetRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DatasetName string `json:"DatasetName,omitempty"`
	DatasetType *string `json:"DatasetType,omitempty"`
	Description *string `json:"Description,omitempty"`
	FlywheelArn string `json:"FlywheelArn,omitempty"`
	InputDataConfig DatasetInputDataConfig `json:"InputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateDatasetResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
}

type CreateDocumentClassifierRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	DocumentClassifierName string `json:"DocumentClassifierName,omitempty"`
	InputDataConfig DocumentClassifierInputDataConfig `json:"InputDataConfig,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	Mode *string `json:"Mode,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	ModelPolicy *string `json:"ModelPolicy,omitempty"`
	OutputDataConfig *DocumentClassifierOutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VersionName *string `json:"VersionName,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type CreateDocumentClassifierResponse struct {
	DocumentClassifierArn *string `json:"DocumentClassifierArn,omitempty"`
}

type CreateEndpointRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DesiredInferenceUnits int `json:"DesiredInferenceUnits,omitempty"`
	EndpointName string `json:"EndpointName,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	ModelArn *string `json:"ModelArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateEndpointResponse struct {
	EndpointArn *string `json:"EndpointArn,omitempty"`
	ModelArn *string `json:"ModelArn,omitempty"`
}

type CreateEntityRecognizerRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig EntityRecognizerInputDataConfig `json:"InputDataConfig,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	ModelPolicy *string `json:"ModelPolicy,omitempty"`
	RecognizerName string `json:"RecognizerName,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VersionName *string `json:"VersionName,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type CreateEntityRecognizerResponse struct {
	EntityRecognizerArn *string `json:"EntityRecognizerArn,omitempty"`
}

type CreateFlywheelRequest struct {
	ActiveModelArn *string `json:"ActiveModelArn,omitempty"`
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	DataLakeS3Uri string `json:"DataLakeS3Uri,omitempty"`
	DataSecurityConfig *DataSecurityConfig `json:"DataSecurityConfig,omitempty"`
	FlywheelName string `json:"FlywheelName,omitempty"`
	ModelType *string `json:"ModelType,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	TaskConfig *TaskConfig `json:"TaskConfig,omitempty"`
}

type CreateFlywheelResponse struct {
	ActiveModelArn *string `json:"ActiveModelArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
}

type DataSecurityConfig struct {
	DataLakeKmsKeyId *string `json:"DataLakeKmsKeyId,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type DatasetAugmentedManifestsListItem struct {
	AnnotationDataS3Uri *string `json:"AnnotationDataS3Uri,omitempty"`
	AttributeNames []string `json:"AttributeNames,omitempty"`
	DocumentType *string `json:"DocumentType,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
	SourceDocumentsS3Uri *string `json:"SourceDocumentsS3Uri,omitempty"`
}

type DatasetDocumentClassifierInputDataConfig struct {
	LabelDelimiter *string `json:"LabelDelimiter,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type DatasetEntityRecognizerAnnotations struct {
	S3Uri string `json:"S3Uri,omitempty"`
}

type DatasetEntityRecognizerDocuments struct {
	InputFormat *string `json:"InputFormat,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type DatasetEntityRecognizerEntityList struct {
	S3Uri string `json:"S3Uri,omitempty"`
}

type DatasetEntityRecognizerInputDataConfig struct {
	Annotations *DatasetEntityRecognizerAnnotations `json:"Annotations,omitempty"`
	Documents DatasetEntityRecognizerDocuments `json:"Documents,omitempty"`
	EntityList *DatasetEntityRecognizerEntityList `json:"EntityList,omitempty"`
}

type DatasetFilter struct {
	CreationTimeAfter *time.Time `json:"CreationTimeAfter,omitempty"`
	CreationTimeBefore *time.Time `json:"CreationTimeBefore,omitempty"`
	DatasetType *string `json:"DatasetType,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DatasetInputDataConfig struct {
	AugmentedManifests []DatasetAugmentedManifestsListItem `json:"AugmentedManifests,omitempty"`
	DataFormat *string `json:"DataFormat,omitempty"`
	DocumentClassifierInputDataConfig *DatasetDocumentClassifierInputDataConfig `json:"DocumentClassifierInputDataConfig,omitempty"`
	EntityRecognizerInputDataConfig *DatasetEntityRecognizerInputDataConfig `json:"EntityRecognizerInputDataConfig,omitempty"`
}

type DatasetProperties struct {
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	DatasetArn *string `json:"DatasetArn,omitempty"`
	DatasetName *string `json:"DatasetName,omitempty"`
	DatasetS3Uri *string `json:"DatasetS3Uri,omitempty"`
	DatasetType *string `json:"DatasetType,omitempty"`
	Description *string `json:"Description,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	Message *string `json:"Message,omitempty"`
	NumberOfDocuments int64 `json:"NumberOfDocuments,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DeleteDocumentClassifierRequest struct {
	DocumentClassifierArn string `json:"DocumentClassifierArn,omitempty"`
}

type DeleteDocumentClassifierResponse struct {
}

type DeleteEndpointRequest struct {
	EndpointArn string `json:"EndpointArn,omitempty"`
}

type DeleteEndpointResponse struct {
}

type DeleteEntityRecognizerRequest struct {
	EntityRecognizerArn string `json:"EntityRecognizerArn,omitempty"`
}

type DeleteEntityRecognizerResponse struct {
}

type DeleteFlywheelRequest struct {
	FlywheelArn string `json:"FlywheelArn,omitempty"`
}

type DeleteFlywheelResponse struct {
}

type DeleteResourcePolicyRequest struct {
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type DeleteResourcePolicyResponse struct {
}

type DescribeDatasetRequest struct {
	DatasetArn string `json:"DatasetArn,omitempty"`
}

type DescribeDatasetResponse struct {
	DatasetProperties *DatasetProperties `json:"DatasetProperties,omitempty"`
}

type DescribeDocumentClassificationJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeDocumentClassificationJobResponse struct {
	DocumentClassificationJobProperties *DocumentClassificationJobProperties `json:"DocumentClassificationJobProperties,omitempty"`
}

type DescribeDocumentClassifierRequest struct {
	DocumentClassifierArn string `json:"DocumentClassifierArn,omitempty"`
}

type DescribeDocumentClassifierResponse struct {
	DocumentClassifierProperties *DocumentClassifierProperties `json:"DocumentClassifierProperties,omitempty"`
}

type DescribeDominantLanguageDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeDominantLanguageDetectionJobResponse struct {
	DominantLanguageDetectionJobProperties *DominantLanguageDetectionJobProperties `json:"DominantLanguageDetectionJobProperties,omitempty"`
}

type DescribeEndpointRequest struct {
	EndpointArn string `json:"EndpointArn,omitempty"`
}

type DescribeEndpointResponse struct {
	EndpointProperties *EndpointProperties `json:"EndpointProperties,omitempty"`
}

type DescribeEntitiesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeEntitiesDetectionJobResponse struct {
	EntitiesDetectionJobProperties *EntitiesDetectionJobProperties `json:"EntitiesDetectionJobProperties,omitempty"`
}

type DescribeEntityRecognizerRequest struct {
	EntityRecognizerArn string `json:"EntityRecognizerArn,omitempty"`
}

type DescribeEntityRecognizerResponse struct {
	EntityRecognizerProperties *EntityRecognizerProperties `json:"EntityRecognizerProperties,omitempty"`
}

type DescribeEventsDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeEventsDetectionJobResponse struct {
	EventsDetectionJobProperties *EventsDetectionJobProperties `json:"EventsDetectionJobProperties,omitempty"`
}

type DescribeFlywheelIterationRequest struct {
	FlywheelArn string `json:"FlywheelArn,omitempty"`
	FlywheelIterationId string `json:"FlywheelIterationId,omitempty"`
}

type DescribeFlywheelIterationResponse struct {
	FlywheelIterationProperties *FlywheelIterationProperties `json:"FlywheelIterationProperties,omitempty"`
}

type DescribeFlywheelRequest struct {
	FlywheelArn string `json:"FlywheelArn,omitempty"`
}

type DescribeFlywheelResponse struct {
	FlywheelProperties *FlywheelProperties `json:"FlywheelProperties,omitempty"`
}

type DescribeKeyPhrasesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeKeyPhrasesDetectionJobResponse struct {
	KeyPhrasesDetectionJobProperties *KeyPhrasesDetectionJobProperties `json:"KeyPhrasesDetectionJobProperties,omitempty"`
}

type DescribePiiEntitiesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribePiiEntitiesDetectionJobResponse struct {
	PiiEntitiesDetectionJobProperties *PiiEntitiesDetectionJobProperties `json:"PiiEntitiesDetectionJobProperties,omitempty"`
}

type DescribeResourcePolicyRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type DescribeResourcePolicyResponse struct {
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ResourcePolicy *string `json:"ResourcePolicy,omitempty"`
}

type DescribeSentimentDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeSentimentDetectionJobResponse struct {
	SentimentDetectionJobProperties *SentimentDetectionJobProperties `json:"SentimentDetectionJobProperties,omitempty"`
}

type DescribeTargetedSentimentDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeTargetedSentimentDetectionJobResponse struct {
	TargetedSentimentDetectionJobProperties *TargetedSentimentDetectionJobProperties `json:"TargetedSentimentDetectionJobProperties,omitempty"`
}

type DescribeTopicsDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type DescribeTopicsDetectionJobResponse struct {
	TopicsDetectionJobProperties *TopicsDetectionJobProperties `json:"TopicsDetectionJobProperties,omitempty"`
}

type DetectDominantLanguageRequest struct {
	Text string `json:"Text,omitempty"`
}

type DetectDominantLanguageResponse struct {
	Languages []DominantLanguage `json:"Languages,omitempty"`
}

type DetectEntitiesRequest struct {
	Bytes []byte `json:"Bytes,omitempty"`
	DocumentReaderConfig *DocumentReaderConfig `json:"DocumentReaderConfig,omitempty"`
	EndpointArn *string `json:"EndpointArn,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Text *string `json:"Text,omitempty"`
}

type DetectEntitiesResponse struct {
	Blocks []Block `json:"Blocks,omitempty"`
	DocumentMetadata *DocumentMetadata `json:"DocumentMetadata,omitempty"`
	DocumentType []DocumentTypeListItem `json:"DocumentType,omitempty"`
	Entities []Entity `json:"Entities,omitempty"`
	Errors []ErrorsListItem `json:"Errors,omitempty"`
}

type DetectKeyPhrasesRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type DetectKeyPhrasesResponse struct {
	KeyPhrases []KeyPhrase `json:"KeyPhrases,omitempty"`
}

type DetectPiiEntitiesRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type DetectPiiEntitiesResponse struct {
	Entities []PiiEntity `json:"Entities,omitempty"`
}

type DetectSentimentRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type DetectSentimentResponse struct {
	Sentiment *string `json:"Sentiment,omitempty"`
	SentimentScore *SentimentScore `json:"SentimentScore,omitempty"`
}

type DetectSyntaxRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type DetectSyntaxResponse struct {
	SyntaxTokens []SyntaxToken `json:"SyntaxTokens,omitempty"`
}

type DetectTargetedSentimentRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	Text string `json:"Text,omitempty"`
}

type DetectTargetedSentimentResponse struct {
	Entities []TargetedSentimentEntity `json:"Entities,omitempty"`
}

type DetectToxicContentRequest struct {
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextSegments []TextSegment `json:"TextSegments,omitempty"`
}

type DetectToxicContentResponse struct {
	ResultList []ToxicLabels `json:"ResultList,omitempty"`
}

type DocumentClass struct {
	Name *string `json:"Name,omitempty"`
	Page int `json:"Page,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

type DocumentClassificationConfig struct {
	Labels []string `json:"Labels,omitempty"`
	Mode string `json:"Mode,omitempty"`
}

type DocumentClassificationJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type DocumentClassificationJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DocumentClassifierArn *string `json:"DocumentClassifierArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type DocumentClassifierDocuments struct {
	S3Uri string `json:"S3Uri,omitempty"`
	TestS3Uri *string `json:"TestS3Uri,omitempty"`
}

type DocumentClassifierFilter struct {
	DocumentClassifierName *string `json:"DocumentClassifierName,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type DocumentClassifierInputDataConfig struct {
	AugmentedManifests []AugmentedManifestsListItem `json:"AugmentedManifests,omitempty"`
	DataFormat *string `json:"DataFormat,omitempty"`
	DocumentReaderConfig *DocumentReaderConfig `json:"DocumentReaderConfig,omitempty"`
	DocumentType *string `json:"DocumentType,omitempty"`
	Documents *DocumentClassifierDocuments `json:"Documents,omitempty"`
	LabelDelimiter *string `json:"LabelDelimiter,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
	TestS3Uri *string `json:"TestS3Uri,omitempty"`
}

type DocumentClassifierOutputDataConfig struct {
	FlywheelStatsS3Prefix *string `json:"FlywheelStatsS3Prefix,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type DocumentClassifierProperties struct {
	ClassifierMetadata *ClassifierMetadata `json:"ClassifierMetadata,omitempty"`
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DocumentClassifierArn *string `json:"DocumentClassifierArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig *DocumentClassifierInputDataConfig `json:"InputDataConfig,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	Mode *string `json:"Mode,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	OutputDataConfig *DocumentClassifierOutputDataConfig `json:"OutputDataConfig,omitempty"`
	SourceModelArn *string `json:"SourceModelArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	TrainingEndTime *time.Time `json:"TrainingEndTime,omitempty"`
	TrainingStartTime *time.Time `json:"TrainingStartTime,omitempty"`
	VersionName *string `json:"VersionName,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type DocumentClassifierSummary struct {
	DocumentClassifierName *string `json:"DocumentClassifierName,omitempty"`
	LatestVersionCreatedAt *time.Time `json:"LatestVersionCreatedAt,omitempty"`
	LatestVersionName *string `json:"LatestVersionName,omitempty"`
	LatestVersionStatus *string `json:"LatestVersionStatus,omitempty"`
	NumberOfVersions int `json:"NumberOfVersions,omitempty"`
}

type DocumentLabel struct {
	Name *string `json:"Name,omitempty"`
	Page int `json:"Page,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

type DocumentMetadata struct {
	ExtractedCharacters []ExtractedCharactersListItem `json:"ExtractedCharacters,omitempty"`
	Pages int `json:"Pages,omitempty"`
}

type DocumentReaderConfig struct {
	DocumentReadAction string `json:"DocumentReadAction,omitempty"`
	DocumentReadMode *string `json:"DocumentReadMode,omitempty"`
	FeatureTypes []string `json:"FeatureTypes,omitempty"`
}

type DocumentTypeListItem struct {
	Page int `json:"Page,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DominantLanguage struct {
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

type DominantLanguageDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type DominantLanguageDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type EndpointFilter struct {
	CreationTimeAfter *time.Time `json:"CreationTimeAfter,omitempty"`
	CreationTimeBefore *time.Time `json:"CreationTimeBefore,omitempty"`
	ModelArn *string `json:"ModelArn,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type EndpointProperties struct {
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	CurrentInferenceUnits int `json:"CurrentInferenceUnits,omitempty"`
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DesiredDataAccessRoleArn *string `json:"DesiredDataAccessRoleArn,omitempty"`
	DesiredInferenceUnits int `json:"DesiredInferenceUnits,omitempty"`
	DesiredModelArn *string `json:"DesiredModelArn,omitempty"`
	EndpointArn *string `json:"EndpointArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	Message *string `json:"Message,omitempty"`
	ModelArn *string `json:"ModelArn,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type EntitiesDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type EntitiesDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	EntityRecognizerArn *string `json:"EntityRecognizerArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type Entity struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	BlockReferences []BlockReference `json:"BlockReferences,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
	Score float64 `json:"Score,omitempty"`
	Text *string `json:"Text,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type EntityLabel struct {
	Name *string `json:"Name,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

type EntityRecognitionConfig struct {
	EntityTypes []EntityTypesListItem `json:"EntityTypes,omitempty"`
}

type EntityRecognizerAnnotations struct {
	S3Uri string `json:"S3Uri,omitempty"`
	TestS3Uri *string `json:"TestS3Uri,omitempty"`
}

type EntityRecognizerDocuments struct {
	InputFormat *string `json:"InputFormat,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
	TestS3Uri *string `json:"TestS3Uri,omitempty"`
}

type EntityRecognizerEntityList struct {
	S3Uri string `json:"S3Uri,omitempty"`
}

type EntityRecognizerEvaluationMetrics struct {
	F1Score float64 `json:"F1Score,omitempty"`
	Precision float64 `json:"Precision,omitempty"`
	Recall float64 `json:"Recall,omitempty"`
}

type EntityRecognizerFilter struct {
	RecognizerName *string `json:"RecognizerName,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type EntityRecognizerInputDataConfig struct {
	Annotations *EntityRecognizerAnnotations `json:"Annotations,omitempty"`
	AugmentedManifests []AugmentedManifestsListItem `json:"AugmentedManifests,omitempty"`
	DataFormat *string `json:"DataFormat,omitempty"`
	Documents *EntityRecognizerDocuments `json:"Documents,omitempty"`
	EntityList *EntityRecognizerEntityList `json:"EntityList,omitempty"`
	EntityTypes []EntityTypesListItem `json:"EntityTypes,omitempty"`
}

type EntityRecognizerMetadata struct {
	EntityTypes []EntityRecognizerMetadataEntityTypesListItem `json:"EntityTypes,omitempty"`
	EvaluationMetrics *EntityRecognizerEvaluationMetrics `json:"EvaluationMetrics,omitempty"`
	NumberOfTestDocuments int `json:"NumberOfTestDocuments,omitempty"`
	NumberOfTrainedDocuments int `json:"NumberOfTrainedDocuments,omitempty"`
}

type EntityRecognizerMetadataEntityTypesListItem struct {
	EvaluationMetrics *EntityTypesEvaluationMetrics `json:"EvaluationMetrics,omitempty"`
	NumberOfTrainMentions int `json:"NumberOfTrainMentions,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type EntityRecognizerOutputDataConfig struct {
	FlywheelStatsS3Prefix *string `json:"FlywheelStatsS3Prefix,omitempty"`
}

type EntityRecognizerProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	EntityRecognizerArn *string `json:"EntityRecognizerArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig *EntityRecognizerInputDataConfig `json:"InputDataConfig,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	OutputDataConfig *EntityRecognizerOutputDataConfig `json:"OutputDataConfig,omitempty"`
	RecognizerMetadata *EntityRecognizerMetadata `json:"RecognizerMetadata,omitempty"`
	SourceModelArn *string `json:"SourceModelArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	TrainingEndTime *time.Time `json:"TrainingEndTime,omitempty"`
	TrainingStartTime *time.Time `json:"TrainingStartTime,omitempty"`
	VersionName *string `json:"VersionName,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type EntityRecognizerSummary struct {
	LatestVersionCreatedAt *time.Time `json:"LatestVersionCreatedAt,omitempty"`
	LatestVersionName *string `json:"LatestVersionName,omitempty"`
	LatestVersionStatus *string `json:"LatestVersionStatus,omitempty"`
	NumberOfVersions int `json:"NumberOfVersions,omitempty"`
	RecognizerName *string `json:"RecognizerName,omitempty"`
}

type EntityTypesEvaluationMetrics struct {
	F1Score float64 `json:"F1Score,omitempty"`
	Precision float64 `json:"Precision,omitempty"`
	Recall float64 `json:"Recall,omitempty"`
}

type EntityTypesListItem struct {
	Type string `json:"Type,omitempty"`
}

type ErrorsListItem struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	Page int `json:"Page,omitempty"`
}

type EventsDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type EventsDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	TargetEventTypes []string `json:"TargetEventTypes,omitempty"`
}

type ExtractedCharactersListItem struct {
	Count int `json:"Count,omitempty"`
	Page int `json:"Page,omitempty"`
}

type FlywheelFilter struct {
	CreationTimeAfter *time.Time `json:"CreationTimeAfter,omitempty"`
	CreationTimeBefore *time.Time `json:"CreationTimeBefore,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type FlywheelIterationFilter struct {
	CreationTimeAfter *time.Time `json:"CreationTimeAfter,omitempty"`
	CreationTimeBefore *time.Time `json:"CreationTimeBefore,omitempty"`
}

type FlywheelIterationProperties struct {
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	EvaluatedModelArn *string `json:"EvaluatedModelArn,omitempty"`
	EvaluatedModelMetrics *FlywheelModelEvaluationMetrics `json:"EvaluatedModelMetrics,omitempty"`
	EvaluationManifestS3Prefix *string `json:"EvaluationManifestS3Prefix,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	FlywheelIterationId *string `json:"FlywheelIterationId,omitempty"`
	Message *string `json:"Message,omitempty"`
	Status *string `json:"Status,omitempty"`
	TrainedModelArn *string `json:"TrainedModelArn,omitempty"`
	TrainedModelMetrics *FlywheelModelEvaluationMetrics `json:"TrainedModelMetrics,omitempty"`
}

type FlywheelModelEvaluationMetrics struct {
	AverageAccuracy float64 `json:"AverageAccuracy,omitempty"`
	AverageF1Score float64 `json:"AverageF1Score,omitempty"`
	AveragePrecision float64 `json:"AveragePrecision,omitempty"`
	AverageRecall float64 `json:"AverageRecall,omitempty"`
}

type FlywheelProperties struct {
	ActiveModelArn *string `json:"ActiveModelArn,omitempty"`
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DataLakeS3Uri *string `json:"DataLakeS3Uri,omitempty"`
	DataSecurityConfig *DataSecurityConfig `json:"DataSecurityConfig,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	LatestFlywheelIteration *string `json:"LatestFlywheelIteration,omitempty"`
	Message *string `json:"Message,omitempty"`
	ModelType *string `json:"ModelType,omitempty"`
	Status *string `json:"Status,omitempty"`
	TaskConfig *TaskConfig `json:"TaskConfig,omitempty"`
}

type FlywheelSummary struct {
	ActiveModelArn *string `json:"ActiveModelArn,omitempty"`
	CreationTime *time.Time `json:"CreationTime,omitempty"`
	DataLakeS3Uri *string `json:"DataLakeS3Uri,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	LatestFlywheelIteration *string `json:"LatestFlywheelIteration,omitempty"`
	Message *string `json:"Message,omitempty"`
	ModelType *string `json:"ModelType,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type Geometry struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Polygon []Point `json:"Polygon,omitempty"`
}

type ImportModelRequest struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	ModelName *string `json:"ModelName,omitempty"`
	SourceModelArn string `json:"SourceModelArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VersionName *string `json:"VersionName,omitempty"`
}

type ImportModelResponse struct {
	ModelArn *string `json:"ModelArn,omitempty"`
}

type InputDataConfig struct {
	DocumentReaderConfig *DocumentReaderConfig `json:"DocumentReaderConfig,omitempty"`
	InputFormat *string `json:"InputFormat,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type KeyPhrase struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
	Score float64 `json:"Score,omitempty"`
	Text *string `json:"Text,omitempty"`
}

type KeyPhrasesDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type KeyPhrasesDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type ListDatasetsRequest struct {
	Filter *DatasetFilter `json:"Filter,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDatasetsResponse struct {
	DatasetPropertiesList []DatasetProperties `json:"DatasetPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassificationJobsRequest struct {
	Filter *DocumentClassificationJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassificationJobsResponse struct {
	DocumentClassificationJobPropertiesList []DocumentClassificationJobProperties `json:"DocumentClassificationJobPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassifierSummariesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassifierSummariesResponse struct {
	DocumentClassifierSummariesList []DocumentClassifierSummary `json:"DocumentClassifierSummariesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassifiersRequest struct {
	Filter *DocumentClassifierFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDocumentClassifiersResponse struct {
	DocumentClassifierPropertiesList []DocumentClassifierProperties `json:"DocumentClassifierPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDominantLanguageDetectionJobsRequest struct {
	Filter *DominantLanguageDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDominantLanguageDetectionJobsResponse struct {
	DominantLanguageDetectionJobPropertiesList []DominantLanguageDetectionJobProperties `json:"DominantLanguageDetectionJobPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEndpointsRequest struct {
	Filter *EndpointFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEndpointsResponse struct {
	EndpointPropertiesList []EndpointProperties `json:"EndpointPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntitiesDetectionJobsRequest struct {
	Filter *EntitiesDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntitiesDetectionJobsResponse struct {
	EntitiesDetectionJobPropertiesList []EntitiesDetectionJobProperties `json:"EntitiesDetectionJobPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntityRecognizerSummariesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntityRecognizerSummariesResponse struct {
	EntityRecognizerSummariesList []EntityRecognizerSummary `json:"EntityRecognizerSummariesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntityRecognizersRequest struct {
	Filter *EntityRecognizerFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEntityRecognizersResponse struct {
	EntityRecognizerPropertiesList []EntityRecognizerProperties `json:"EntityRecognizerPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEventsDetectionJobsRequest struct {
	Filter *EventsDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEventsDetectionJobsResponse struct {
	EventsDetectionJobPropertiesList []EventsDetectionJobProperties `json:"EventsDetectionJobPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFlywheelIterationHistoryRequest struct {
	Filter *FlywheelIterationFilter `json:"Filter,omitempty"`
	FlywheelArn string `json:"FlywheelArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFlywheelIterationHistoryResponse struct {
	FlywheelIterationPropertiesList []FlywheelIterationProperties `json:"FlywheelIterationPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFlywheelsRequest struct {
	Filter *FlywheelFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFlywheelsResponse struct {
	FlywheelSummaryList []FlywheelSummary `json:"FlywheelSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListKeyPhrasesDetectionJobsRequest struct {
	Filter *KeyPhrasesDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListKeyPhrasesDetectionJobsResponse struct {
	KeyPhrasesDetectionJobPropertiesList []KeyPhrasesDetectionJobProperties `json:"KeyPhrasesDetectionJobPropertiesList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListPiiEntitiesDetectionJobsRequest struct {
	Filter *PiiEntitiesDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListPiiEntitiesDetectionJobsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	PiiEntitiesDetectionJobPropertiesList []PiiEntitiesDetectionJobProperties `json:"PiiEntitiesDetectionJobPropertiesList,omitempty"`
}

type ListSentimentDetectionJobsRequest struct {
	Filter *SentimentDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListSentimentDetectionJobsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	SentimentDetectionJobPropertiesList []SentimentDetectionJobProperties `json:"SentimentDetectionJobPropertiesList,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	ResourceArn *string `json:"ResourceArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type ListTargetedSentimentDetectionJobsRequest struct {
	Filter *TargetedSentimentDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListTargetedSentimentDetectionJobsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	TargetedSentimentDetectionJobPropertiesList []TargetedSentimentDetectionJobProperties `json:"TargetedSentimentDetectionJobPropertiesList,omitempty"`
}

type ListTopicsDetectionJobsRequest struct {
	Filter *TopicsDetectionJobFilter `json:"Filter,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListTopicsDetectionJobsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	TopicsDetectionJobPropertiesList []TopicsDetectionJobProperties `json:"TopicsDetectionJobPropertiesList,omitempty"`
}

type MentionSentiment struct {
	Sentiment *string `json:"Sentiment,omitempty"`
	SentimentScore *SentimentScore `json:"SentimentScore,omitempty"`
}

type OutputDataConfig struct {
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type PartOfSpeechTag struct {
	Score float64 `json:"Score,omitempty"`
	Tag *string `json:"Tag,omitempty"`
}

type PiiEntitiesDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type PiiEntitiesDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	Mode *string `json:"Mode,omitempty"`
	OutputDataConfig *PiiOutputDataConfig `json:"OutputDataConfig,omitempty"`
	RedactionConfig *RedactionConfig `json:"RedactionConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
}

type PiiEntity struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
	Score float64 `json:"Score,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type PiiOutputDataConfig struct {
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	S3Uri string `json:"S3Uri,omitempty"`
}

type Point struct {
	X float64 `json:"X,omitempty"`
	Y float64 `json:"Y,omitempty"`
}

type PutResourcePolicyRequest struct {
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ResourceArn string `json:"ResourceArn,omitempty"`
	ResourcePolicy string `json:"ResourcePolicy,omitempty"`
}

type PutResourcePolicyResponse struct {
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
}

type RedactionConfig struct {
	MaskCharacter *string `json:"MaskCharacter,omitempty"`
	MaskMode *string `json:"MaskMode,omitempty"`
	PiiEntityTypes []string `json:"PiiEntityTypes,omitempty"`
}

type RelationshipsListItem struct {
	Ids []string `json:"Ids,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type SentimentDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type SentimentDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type SentimentScore struct {
	Mixed float64 `json:"Mixed,omitempty"`
	Negative float64 `json:"Negative,omitempty"`
	Neutral float64 `json:"Neutral,omitempty"`
	Positive float64 `json:"Positive,omitempty"`
}

type StartDocumentClassificationJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	DocumentClassifierArn *string `json:"DocumentClassifierArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartDocumentClassificationJobResponse struct {
	DocumentClassifierArn *string `json:"DocumentClassifierArn,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartDominantLanguageDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartDominantLanguageDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartEntitiesDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	EntityRecognizerArn *string `json:"EntityRecognizerArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartEntitiesDetectionJobResponse struct {
	EntityRecognizerArn *string `json:"EntityRecognizerArn,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartEventsDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	TargetEventTypes []string `json:"TargetEventTypes,omitempty"`
}

type StartEventsDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartFlywheelIterationRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	FlywheelArn string `json:"FlywheelArn,omitempty"`
}

type StartFlywheelIterationResponse struct {
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
	FlywheelIterationId *string `json:"FlywheelIterationId,omitempty"`
}

type StartKeyPhrasesDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartKeyPhrasesDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartPiiEntitiesDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	Mode string `json:"Mode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	RedactionConfig *RedactionConfig `json:"RedactionConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type StartPiiEntitiesDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartSentimentDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartSentimentDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartTargetedSentimentDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartTargetedSentimentDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StartTopicsDetectionJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	DataAccessRoleArn string `json:"DataAccessRoleArn,omitempty"`
	InputDataConfig InputDataConfig `json:"InputDataConfig,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	NumberOfTopics int `json:"NumberOfTopics,omitempty"`
	OutputDataConfig OutputDataConfig `json:"OutputDataConfig,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type StartTopicsDetectionJobResponse struct {
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopDominantLanguageDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopDominantLanguageDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopEntitiesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopEntitiesDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopEventsDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopEventsDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopKeyPhrasesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopKeyPhrasesDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopPiiEntitiesDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopPiiEntitiesDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopSentimentDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopSentimentDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopTargetedSentimentDetectionJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type StopTargetedSentimentDetectionJobResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type StopTrainingDocumentClassifierRequest struct {
	DocumentClassifierArn string `json:"DocumentClassifierArn,omitempty"`
}

type StopTrainingDocumentClassifierResponse struct {
}

type StopTrainingEntityRecognizerRequest struct {
	EntityRecognizerArn string `json:"EntityRecognizerArn,omitempty"`
}

type StopTrainingEntityRecognizerResponse struct {
}

type SyntaxToken struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
	PartOfSpeech *PartOfSpeechTag `json:"PartOfSpeech,omitempty"`
	Text *string `json:"Text,omitempty"`
	TokenId int `json:"TokenId,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
}

type TargetedSentimentDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type TargetedSentimentDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LanguageCode *string `json:"LanguageCode,omitempty"`
	Message *string `json:"Message,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type TargetedSentimentEntity struct {
	DescriptiveMentionIndex []int `json:"DescriptiveMentionIndex,omitempty"`
	Mentions []TargetedSentimentMention `json:"Mentions,omitempty"`
}

type TargetedSentimentMention struct {
	BeginOffset int `json:"BeginOffset,omitempty"`
	EndOffset int `json:"EndOffset,omitempty"`
	GroupScore float64 `json:"GroupScore,omitempty"`
	MentionSentiment *MentionSentiment `json:"MentionSentiment,omitempty"`
	Score float64 `json:"Score,omitempty"`
	Text *string `json:"Text,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type TaskConfig struct {
	DocumentClassificationConfig *DocumentClassificationConfig `json:"DocumentClassificationConfig,omitempty"`
	EntityRecognitionConfig *EntityRecognitionConfig `json:"EntityRecognitionConfig,omitempty"`
	LanguageCode string `json:"LanguageCode,omitempty"`
}

type TextSegment struct {
	Text string `json:"Text,omitempty"`
}

type TopicsDetectionJobFilter struct {
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	SubmitTimeAfter *time.Time `json:"SubmitTimeAfter,omitempty"`
	SubmitTimeBefore *time.Time `json:"SubmitTimeBefore,omitempty"`
}

type TopicsDetectionJobProperties struct {
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	EndTime *time.Time `json:"EndTime,omitempty"`
	InputDataConfig *InputDataConfig `json:"InputDataConfig,omitempty"`
	JobArn *string `json:"JobArn,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	Message *string `json:"Message,omitempty"`
	NumberOfTopics int `json:"NumberOfTopics,omitempty"`
	OutputDataConfig *OutputDataConfig `json:"OutputDataConfig,omitempty"`
	SubmitTime *time.Time `json:"SubmitTime,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type ToxicContent struct {
	Name *string `json:"Name,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

type ToxicLabels struct {
	Labels []ToxicContent `json:"Labels,omitempty"`
	Toxicity float64 `json:"Toxicity,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"TagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateDataSecurityConfig struct {
	ModelKmsKeyId *string `json:"ModelKmsKeyId,omitempty"`
	VolumeKmsKeyId *string `json:"VolumeKmsKeyId,omitempty"`
	VpcConfig *VpcConfig `json:"VpcConfig,omitempty"`
}

type UpdateEndpointRequest struct {
	DesiredDataAccessRoleArn *string `json:"DesiredDataAccessRoleArn,omitempty"`
	DesiredInferenceUnits int `json:"DesiredInferenceUnits,omitempty"`
	DesiredModelArn *string `json:"DesiredModelArn,omitempty"`
	EndpointArn string `json:"EndpointArn,omitempty"`
	FlywheelArn *string `json:"FlywheelArn,omitempty"`
}

type UpdateEndpointResponse struct {
	DesiredModelArn *string `json:"DesiredModelArn,omitempty"`
}

type UpdateFlywheelRequest struct {
	ActiveModelArn *string `json:"ActiveModelArn,omitempty"`
	DataAccessRoleArn *string `json:"DataAccessRoleArn,omitempty"`
	DataSecurityConfig *UpdateDataSecurityConfig `json:"DataSecurityConfig,omitempty"`
	FlywheelArn string `json:"FlywheelArn,omitempty"`
}

type UpdateFlywheelResponse struct {
	FlywheelProperties *FlywheelProperties `json:"FlywheelProperties,omitempty"`
}

type VpcConfig struct {
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	Subnets []string `json:"Subnets,omitempty"`
}

type WarningsListItem struct {
	Page int `json:"Page,omitempty"`
	WarnCode *string `json:"WarnCode,omitempty"`
	WarnMessage *string `json:"WarnMessage,omitempty"`
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

func handleBatchDetectDominantLanguage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectDominantLanguageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectDominantLanguage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectDominantLanguage"})
}

func handleBatchDetectEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectEntitiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectEntities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectEntities"})
}

func handleBatchDetectKeyPhrases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectKeyPhrasesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectKeyPhrases business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectKeyPhrases"})
}

func handleBatchDetectSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectSentimentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectSentiment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectSentiment"})
}

func handleBatchDetectSyntax(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectSyntaxRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectSyntax business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectSyntax"})
}

func handleBatchDetectTargetedSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDetectTargetedSentimentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDetectTargetedSentiment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDetectTargetedSentiment"})
}

func handleClassifyDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ClassifyDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ClassifyDocument business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ClassifyDocument"})
}

func handleContainsPiiEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ContainsPiiEntitiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ContainsPiiEntities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ContainsPiiEntities"})
}

func handleCreateDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDatasetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDataset business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDataset"})
}

func handleCreateDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDocumentClassifierRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDocumentClassifier business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDocumentClassifier"})
}

func handleCreateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateEndpoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateEndpoint"})
}

func handleCreateEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateEntityRecognizerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateEntityRecognizer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateEntityRecognizer"})
}

func handleCreateFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFlywheelRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFlywheel business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFlywheel"})
}

func handleDeleteDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDocumentClassifierRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDocumentClassifier business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDocumentClassifier"})
}

func handleDeleteEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteEndpoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteEndpoint"})
}

func handleDeleteEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteEntityRecognizerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteEntityRecognizer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteEntityRecognizer"})
}

func handleDeleteFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFlywheelRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFlywheel business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFlywheel"})
}

func handleDeleteResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteResourcePolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteResourcePolicy"})
}

func handleDescribeDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDatasetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataset business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataset"})
}

func handleDescribeDocumentClassificationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDocumentClassificationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDocumentClassificationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDocumentClassificationJob"})
}

func handleDescribeDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDocumentClassifierRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDocumentClassifier business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDocumentClassifier"})
}

func handleDescribeDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDominantLanguageDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDominantLanguageDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDominantLanguageDetectionJob"})
}

func handleDescribeEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeEndpoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeEndpoint"})
}

func handleDescribeEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeEntitiesDetectionJob"})
}

func handleDescribeEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeEntityRecognizerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeEntityRecognizer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeEntityRecognizer"})
}

func handleDescribeEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeEventsDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeEventsDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeEventsDetectionJob"})
}

func handleDescribeFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFlywheelRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFlywheel business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFlywheel"})
}

func handleDescribeFlywheelIteration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFlywheelIterationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFlywheelIteration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFlywheelIteration"})
}

func handleDescribeKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeKeyPhrasesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeKeyPhrasesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeKeyPhrasesDetectionJob"})
}

func handleDescribePiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribePiiEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribePiiEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribePiiEntitiesDetectionJob"})
}

func handleDescribeResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeResourcePolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeResourcePolicy"})
}

func handleDescribeSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeSentimentDetectionJob"})
}

func handleDescribeTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTargetedSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTargetedSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTargetedSentimentDetectionJob"})
}

func handleDescribeTopicsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTopicsDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTopicsDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTopicsDetectionJob"})
}

func handleDetectDominantLanguage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectDominantLanguageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectDominantLanguage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectDominantLanguage"})
}

func handleDetectEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectEntitiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectEntities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectEntities"})
}

func handleDetectKeyPhrases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectKeyPhrasesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectKeyPhrases business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectKeyPhrases"})
}

func handleDetectPiiEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectPiiEntitiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectPiiEntities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectPiiEntities"})
}

func handleDetectSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectSentimentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectSentiment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectSentiment"})
}

func handleDetectSyntax(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectSyntaxRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectSyntax business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectSyntax"})
}

func handleDetectTargetedSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectTargetedSentimentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectTargetedSentiment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectTargetedSentiment"})
}

func handleDetectToxicContent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectToxicContentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectToxicContent business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectToxicContent"})
}

func handleImportModel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ImportModelRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ImportModel business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ImportModel"})
}

func handleListDatasets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDatasetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDatasets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDatasets"})
}

func handleListDocumentClassificationJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDocumentClassificationJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDocumentClassificationJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDocumentClassificationJobs"})
}

func handleListDocumentClassifierSummaries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDocumentClassifierSummariesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDocumentClassifierSummaries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDocumentClassifierSummaries"})
}

func handleListDocumentClassifiers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDocumentClassifiersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDocumentClassifiers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDocumentClassifiers"})
}

func handleListDominantLanguageDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDominantLanguageDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDominantLanguageDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDominantLanguageDetectionJobs"})
}

func handleListEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEndpoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEndpoints"})
}

func handleListEntitiesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEntitiesDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEntitiesDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEntitiesDetectionJobs"})
}

func handleListEntityRecognizerSummaries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEntityRecognizerSummariesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEntityRecognizerSummaries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEntityRecognizerSummaries"})
}

func handleListEntityRecognizers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEntityRecognizersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEntityRecognizers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEntityRecognizers"})
}

func handleListEventsDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEventsDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEventsDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEventsDetectionJobs"})
}

func handleListFlywheelIterationHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFlywheelIterationHistoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFlywheelIterationHistory business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFlywheelIterationHistory"})
}

func handleListFlywheels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFlywheelsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFlywheels business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFlywheels"})
}

func handleListKeyPhrasesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListKeyPhrasesDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListKeyPhrasesDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListKeyPhrasesDetectionJobs"})
}

func handleListPiiEntitiesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPiiEntitiesDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPiiEntitiesDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPiiEntitiesDetectionJobs"})
}

func handleListSentimentDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListSentimentDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListSentimentDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListSentimentDetectionJobs"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListTargetedSentimentDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTargetedSentimentDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTargetedSentimentDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTargetedSentimentDetectionJobs"})
}

func handleListTopicsDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTopicsDetectionJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTopicsDetectionJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTopicsDetectionJobs"})
}

func handlePutResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutResourcePolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutResourcePolicy"})
}

func handleStartDocumentClassificationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartDocumentClassificationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartDocumentClassificationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartDocumentClassificationJob"})
}

func handleStartDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartDominantLanguageDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartDominantLanguageDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartDominantLanguageDetectionJob"})
}

func handleStartEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartEntitiesDetectionJob"})
}

func handleStartEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartEventsDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartEventsDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartEventsDetectionJob"})
}

func handleStartFlywheelIteration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartFlywheelIterationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartFlywheelIteration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartFlywheelIteration"})
}

func handleStartKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartKeyPhrasesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartKeyPhrasesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartKeyPhrasesDetectionJob"})
}

func handleStartPiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartPiiEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartPiiEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartPiiEntitiesDetectionJob"})
}

func handleStartSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartSentimentDetectionJob"})
}

func handleStartTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartTargetedSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartTargetedSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartTargetedSentimentDetectionJob"})
}

func handleStartTopicsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartTopicsDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartTopicsDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartTopicsDetectionJob"})
}

func handleStopDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopDominantLanguageDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopDominantLanguageDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopDominantLanguageDetectionJob"})
}

func handleStopEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopEntitiesDetectionJob"})
}

func handleStopEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopEventsDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopEventsDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopEventsDetectionJob"})
}

func handleStopKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopKeyPhrasesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopKeyPhrasesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopKeyPhrasesDetectionJob"})
}

func handleStopPiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopPiiEntitiesDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopPiiEntitiesDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopPiiEntitiesDetectionJob"})
}

func handleStopSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopSentimentDetectionJob"})
}

func handleStopTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopTargetedSentimentDetectionJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopTargetedSentimentDetectionJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopTargetedSentimentDetectionJob"})
}

func handleStopTrainingDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopTrainingDocumentClassifierRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopTrainingDocumentClassifier business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopTrainingDocumentClassifier"})
}

func handleStopTrainingEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopTrainingEntityRecognizerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopTrainingEntityRecognizer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopTrainingEntityRecognizer"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateEndpoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateEndpoint"})
}

func handleUpdateFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFlywheelRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFlywheel business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFlywheel"})
}

