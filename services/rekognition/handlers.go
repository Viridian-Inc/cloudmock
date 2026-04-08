package rekognition

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AgeRange struct {
	High int `json:"High,omitempty"`
	Low int `json:"Low,omitempty"`
}

type Asset struct {
	GroundTruthManifest *GroundTruthManifest `json:"GroundTruthManifest,omitempty"`
}

type AssociateFacesRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	CollectionId string `json:"CollectionId,omitempty"`
	FaceIds []string `json:"FaceIds,omitempty"`
	UserId string `json:"UserId,omitempty"`
	UserMatchThreshold float64 `json:"UserMatchThreshold,omitempty"`
}

type AssociateFacesResponse struct {
	AssociatedFaces []AssociatedFace `json:"AssociatedFaces,omitempty"`
	UnsuccessfulFaceAssociations []UnsuccessfulFaceAssociation `json:"UnsuccessfulFaceAssociations,omitempty"`
	UserStatus *string `json:"UserStatus,omitempty"`
}

type AssociatedFace struct {
	FaceId *string `json:"FaceId,omitempty"`
}

type AudioMetadata struct {
	Codec *string `json:"Codec,omitempty"`
	DurationMillis int64 `json:"DurationMillis,omitempty"`
	NumberOfChannels int64 `json:"NumberOfChannels,omitempty"`
	SampleRate int64 `json:"SampleRate,omitempty"`
}

type AuditImage struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Bytes []byte `json:"Bytes,omitempty"`
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type Beard struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type BlackFrame struct {
	MaxPixelThreshold float64 `json:"MaxPixelThreshold,omitempty"`
	MinCoveragePercentage float64 `json:"MinCoveragePercentage,omitempty"`
}

type BoundingBox struct {
	Height float64 `json:"Height,omitempty"`
	Left float64 `json:"Left,omitempty"`
	Top float64 `json:"Top,omitempty"`
	Width float64 `json:"Width,omitempty"`
}

type Celebrity struct {
	Face *ComparedFace `json:"Face,omitempty"`
	Id *string `json:"Id,omitempty"`
	KnownGender *KnownGender `json:"KnownGender,omitempty"`
	MatchConfidence float64 `json:"MatchConfidence,omitempty"`
	Name *string `json:"Name,omitempty"`
	Urls []string `json:"Urls,omitempty"`
}

type CelebrityDetail struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	Face *FaceDetail `json:"Face,omitempty"`
	Id *string `json:"Id,omitempty"`
	KnownGender *KnownGender `json:"KnownGender,omitempty"`
	Name *string `json:"Name,omitempty"`
	Urls []string `json:"Urls,omitempty"`
}

type CelebrityRecognition struct {
	Celebrity *CelebrityDetail `json:"Celebrity,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type Challenge struct {
	Type string `json:"Type,omitempty"`
	Version string `json:"Version,omitempty"`
}

type ChallengePreference struct {
	Type string `json:"Type,omitempty"`
	Versions *Versions `json:"Versions,omitempty"`
}

type CompareFacesMatch struct {
	Face *ComparedFace `json:"Face,omitempty"`
	Similarity float64 `json:"Similarity,omitempty"`
}

type CompareFacesRequest struct {
	QualityFilter *string `json:"QualityFilter,omitempty"`
	SimilarityThreshold float64 `json:"SimilarityThreshold,omitempty"`
	SourceImage Image `json:"SourceImage,omitempty"`
	TargetImage Image `json:"TargetImage,omitempty"`
}

type CompareFacesResponse struct {
	FaceMatches []CompareFacesMatch `json:"FaceMatches,omitempty"`
	SourceImageFace *ComparedSourceImageFace `json:"SourceImageFace,omitempty"`
	SourceImageOrientationCorrection *string `json:"SourceImageOrientationCorrection,omitempty"`
	TargetImageOrientationCorrection *string `json:"TargetImageOrientationCorrection,omitempty"`
	UnmatchedFaces []ComparedFace `json:"UnmatchedFaces,omitempty"`
}

type ComparedFace struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	Emotions []Emotion `json:"Emotions,omitempty"`
	Landmarks []Landmark `json:"Landmarks,omitempty"`
	Pose *Pose `json:"Pose,omitempty"`
	Quality *ImageQuality `json:"Quality,omitempty"`
	Smile *Smile `json:"Smile,omitempty"`
}

type ComparedSourceImageFace struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
}

type ConnectedHomeSettings struct {
	Labels []string `json:"Labels,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
}

type ConnectedHomeSettingsForUpdate struct {
	Labels []string `json:"Labels,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
}

type ContentModerationDetection struct {
	ContentTypes []ContentType `json:"ContentTypes,omitempty"`
	DurationMillis int64 `json:"DurationMillis,omitempty"`
	EndTimestampMillis int64 `json:"EndTimestampMillis,omitempty"`
	ModerationLabel *ModerationLabel `json:"ModerationLabel,omitempty"`
	StartTimestampMillis int64 `json:"StartTimestampMillis,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type ContentType struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type CopyProjectVersionRequest struct {
	DestinationProjectArn string `json:"DestinationProjectArn,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	OutputConfig OutputConfig `json:"OutputConfig,omitempty"`
	SourceProjectArn string `json:"SourceProjectArn,omitempty"`
	SourceProjectVersionArn string `json:"SourceProjectVersionArn,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
	VersionName string `json:"VersionName,omitempty"`
}

type CopyProjectVersionResponse struct {
	ProjectVersionArn *string `json:"ProjectVersionArn,omitempty"`
}

type CoversBodyPart struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type CreateCollectionRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateCollectionResponse struct {
	CollectionArn *string `json:"CollectionArn,omitempty"`
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	StatusCode int `json:"StatusCode,omitempty"`
}

type CreateDatasetRequest struct {
	DatasetSource *DatasetSource `json:"DatasetSource,omitempty"`
	DatasetType string `json:"DatasetType,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateDatasetResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
}

type CreateFaceLivenessSessionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Settings *CreateFaceLivenessSessionRequestSettings `json:"Settings,omitempty"`
}

type CreateFaceLivenessSessionRequestSettings struct {
	AuditImagesLimit int `json:"AuditImagesLimit,omitempty"`
	ChallengePreferences []ChallengePreference `json:"ChallengePreferences,omitempty"`
	OutputConfig *LivenessOutputConfig `json:"OutputConfig,omitempty"`
}

type CreateFaceLivenessSessionResponse struct {
	SessionId string `json:"SessionId,omitempty"`
}

type CreateProjectRequest struct {
	AutoUpdate *string `json:"AutoUpdate,omitempty"`
	Feature *string `json:"Feature,omitempty"`
	ProjectName string `json:"ProjectName,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateProjectResponse struct {
	ProjectArn *string `json:"ProjectArn,omitempty"`
}

type CreateProjectVersionRequest struct {
	FeatureConfig *CustomizationFeatureConfig `json:"FeatureConfig,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	OutputConfig OutputConfig `json:"OutputConfig,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
	TestingData *TestingData `json:"TestingData,omitempty"`
	TrainingData *TrainingData `json:"TrainingData,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
	VersionName string `json:"VersionName,omitempty"`
}

type CreateProjectVersionResponse struct {
	ProjectVersionArn *string `json:"ProjectVersionArn,omitempty"`
}

type CreateStreamProcessorRequest struct {
	DataSharingPreference *StreamProcessorDataSharingPreference `json:"DataSharingPreference,omitempty"`
	Input StreamProcessorInput `json:"Input,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Name string `json:"Name,omitempty"`
	NotificationChannel *StreamProcessorNotificationChannel `json:"NotificationChannel,omitempty"`
	Output StreamProcessorOutput `json:"Output,omitempty"`
	RegionsOfInterest []RegionOfInterest `json:"RegionsOfInterest,omitempty"`
	RoleArn string `json:"RoleArn,omitempty"`
	Settings StreamProcessorSettings `json:"Settings,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateStreamProcessorResponse struct {
	StreamProcessorArn *string `json:"StreamProcessorArn,omitempty"`
}

type CreateUserRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	CollectionId string `json:"CollectionId,omitempty"`
	UserId string `json:"UserId,omitempty"`
}

type CreateUserResponse struct {
}

type CustomLabel struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Geometry *Geometry `json:"Geometry,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type CustomizationFeatureConfig struct {
	ContentModeration *CustomizationFeatureContentModerationConfig `json:"ContentModeration,omitempty"`
}

type CustomizationFeatureContentModerationConfig struct {
	ConfidenceThreshold float64 `json:"ConfidenceThreshold,omitempty"`
}

type DatasetChanges struct {
	GroundTruth []byte `json:"GroundTruth,omitempty"`
}

type DatasetDescription struct {
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	DatasetStats *DatasetStats `json:"DatasetStats,omitempty"`
	LastUpdatedTimestamp *time.Time `json:"LastUpdatedTimestamp,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	StatusMessageCode *string `json:"StatusMessageCode,omitempty"`
}

type DatasetLabelDescription struct {
	LabelName *string `json:"LabelName,omitempty"`
	LabelStats *DatasetLabelStats `json:"LabelStats,omitempty"`
}

type DatasetLabelStats struct {
	BoundingBoxCount int `json:"BoundingBoxCount,omitempty"`
	EntryCount int `json:"EntryCount,omitempty"`
}

type DatasetMetadata struct {
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	DatasetArn *string `json:"DatasetArn,omitempty"`
	DatasetType *string `json:"DatasetType,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	StatusMessageCode *string `json:"StatusMessageCode,omitempty"`
}

type DatasetSource struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	GroundTruthManifest *GroundTruthManifest `json:"GroundTruthManifest,omitempty"`
}

type DatasetStats struct {
	ErrorEntries int `json:"ErrorEntries,omitempty"`
	LabeledEntries int `json:"LabeledEntries,omitempty"`
	TotalEntries int `json:"TotalEntries,omitempty"`
	TotalLabels int `json:"TotalLabels,omitempty"`
}

type DeleteCollectionRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
}

type DeleteCollectionResponse struct {
	StatusCode int `json:"StatusCode,omitempty"`
}

type DeleteDatasetRequest struct {
	DatasetArn string `json:"DatasetArn,omitempty"`
}

type DeleteDatasetResponse struct {
}

type DeleteFacesRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	FaceIds []string `json:"FaceIds,omitempty"`
}

type DeleteFacesResponse struct {
	DeletedFaces []string `json:"DeletedFaces,omitempty"`
	UnsuccessfulFaceDeletions []UnsuccessfulFaceDeletion `json:"UnsuccessfulFaceDeletions,omitempty"`
}

type DeleteProjectPolicyRequest struct {
	PolicyName string `json:"PolicyName,omitempty"`
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
}

type DeleteProjectPolicyResponse struct {
}

type DeleteProjectRequest struct {
	ProjectArn string `json:"ProjectArn,omitempty"`
}

type DeleteProjectResponse struct {
	Status *string `json:"Status,omitempty"`
}

type DeleteProjectVersionRequest struct {
	ProjectVersionArn string `json:"ProjectVersionArn,omitempty"`
}

type DeleteProjectVersionResponse struct {
	Status *string `json:"Status,omitempty"`
}

type DeleteStreamProcessorRequest struct {
	Name string `json:"Name,omitempty"`
}

type DeleteStreamProcessorResponse struct {
}

type DeleteUserRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	CollectionId string `json:"CollectionId,omitempty"`
	UserId string `json:"UserId,omitempty"`
}

type DeleteUserResponse struct {
}

type DescribeCollectionRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
}

type DescribeCollectionResponse struct {
	CollectionARN *string `json:"CollectionARN,omitempty"`
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	FaceCount int64 `json:"FaceCount,omitempty"`
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	UserCount int64 `json:"UserCount,omitempty"`
}

type DescribeDatasetRequest struct {
	DatasetArn string `json:"DatasetArn,omitempty"`
}

type DescribeDatasetResponse struct {
	DatasetDescription *DatasetDescription `json:"DatasetDescription,omitempty"`
}

type DescribeProjectVersionsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
	VersionNames []string `json:"VersionNames,omitempty"`
}

type DescribeProjectVersionsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ProjectVersionDescriptions []ProjectVersionDescription `json:"ProjectVersionDescriptions,omitempty"`
}

type DescribeProjectsRequest struct {
	Features []string `json:"Features,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ProjectNames []string `json:"ProjectNames,omitempty"`
}

type DescribeProjectsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ProjectDescriptions []ProjectDescription `json:"ProjectDescriptions,omitempty"`
}

type DescribeStreamProcessorRequest struct {
	Name string `json:"Name,omitempty"`
}

type DescribeStreamProcessorResponse struct {
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	DataSharingPreference *StreamProcessorDataSharingPreference `json:"DataSharingPreference,omitempty"`
	Input *StreamProcessorInput `json:"Input,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LastUpdateTimestamp *time.Time `json:"LastUpdateTimestamp,omitempty"`
	Name *string `json:"Name,omitempty"`
	NotificationChannel *StreamProcessorNotificationChannel `json:"NotificationChannel,omitempty"`
	Output *StreamProcessorOutput `json:"Output,omitempty"`
	RegionsOfInterest []RegionOfInterest `json:"RegionsOfInterest,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	Settings *StreamProcessorSettings `json:"Settings,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	StreamProcessorArn *string `json:"StreamProcessorArn,omitempty"`
}

type DetectCustomLabelsRequest struct {
	Image Image `json:"Image,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	ProjectVersionArn string `json:"ProjectVersionArn,omitempty"`
}

type DetectCustomLabelsResponse struct {
	CustomLabels []CustomLabel `json:"CustomLabels,omitempty"`
}

type DetectFacesRequest struct {
	Attributes []string `json:"Attributes,omitempty"`
	Image Image `json:"Image,omitempty"`
}

type DetectFacesResponse struct {
	FaceDetails []FaceDetail `json:"FaceDetails,omitempty"`
	OrientationCorrection *string `json:"OrientationCorrection,omitempty"`
}

type DetectLabelsImageBackground struct {
	DominantColors []DominantColor `json:"DominantColors,omitempty"`
	Quality *DetectLabelsImageQuality `json:"Quality,omitempty"`
}

type DetectLabelsImageForeground struct {
	DominantColors []DominantColor `json:"DominantColors,omitempty"`
	Quality *DetectLabelsImageQuality `json:"Quality,omitempty"`
}

type DetectLabelsImageProperties struct {
	Background *DetectLabelsImageBackground `json:"Background,omitempty"`
	DominantColors []DominantColor `json:"DominantColors,omitempty"`
	Foreground *DetectLabelsImageForeground `json:"Foreground,omitempty"`
	Quality *DetectLabelsImageQuality `json:"Quality,omitempty"`
}

type DetectLabelsImagePropertiesSettings struct {
	MaxDominantColors int `json:"MaxDominantColors,omitempty"`
}

type DetectLabelsImageQuality struct {
	Brightness float64 `json:"Brightness,omitempty"`
	Contrast float64 `json:"Contrast,omitempty"`
	Sharpness float64 `json:"Sharpness,omitempty"`
}

type DetectLabelsRequest struct {
	Features []string `json:"Features,omitempty"`
	Image Image `json:"Image,omitempty"`
	MaxLabels int `json:"MaxLabels,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	Settings *DetectLabelsSettings `json:"Settings,omitempty"`
}

type DetectLabelsResponse struct {
	ImageProperties *DetectLabelsImageProperties `json:"ImageProperties,omitempty"`
	LabelModelVersion *string `json:"LabelModelVersion,omitempty"`
	Labels []Label `json:"Labels,omitempty"`
	OrientationCorrection *string `json:"OrientationCorrection,omitempty"`
}

type DetectLabelsSettings struct {
	GeneralLabels *GeneralLabelsSettings `json:"GeneralLabels,omitempty"`
	ImageProperties *DetectLabelsImagePropertiesSettings `json:"ImageProperties,omitempty"`
}

type DetectModerationLabelsRequest struct {
	HumanLoopConfig *HumanLoopConfig `json:"HumanLoopConfig,omitempty"`
	Image Image `json:"Image,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	ProjectVersion *string `json:"ProjectVersion,omitempty"`
}

type DetectModerationLabelsResponse struct {
	ContentTypes []ContentType `json:"ContentTypes,omitempty"`
	HumanLoopActivationOutput *HumanLoopActivationOutput `json:"HumanLoopActivationOutput,omitempty"`
	ModerationLabels []ModerationLabel `json:"ModerationLabels,omitempty"`
	ModerationModelVersion *string `json:"ModerationModelVersion,omitempty"`
	ProjectVersion *string `json:"ProjectVersion,omitempty"`
}

type DetectProtectiveEquipmentRequest struct {
	Image Image `json:"Image,omitempty"`
	SummarizationAttributes *ProtectiveEquipmentSummarizationAttributes `json:"SummarizationAttributes,omitempty"`
}

type DetectProtectiveEquipmentResponse struct {
	Persons []ProtectiveEquipmentPerson `json:"Persons,omitempty"`
	ProtectiveEquipmentModelVersion *string `json:"ProtectiveEquipmentModelVersion,omitempty"`
	Summary *ProtectiveEquipmentSummary `json:"Summary,omitempty"`
}

type DetectTextFilters struct {
	RegionsOfInterest []RegionOfInterest `json:"RegionsOfInterest,omitempty"`
	WordFilter *DetectionFilter `json:"WordFilter,omitempty"`
}

type DetectTextRequest struct {
	Filters *DetectTextFilters `json:"Filters,omitempty"`
	Image Image `json:"Image,omitempty"`
}

type DetectTextResponse struct {
	TextDetections []TextDetection `json:"TextDetections,omitempty"`
	TextModelVersion *string `json:"TextModelVersion,omitempty"`
}

type DetectionFilter struct {
	MinBoundingBoxHeight float64 `json:"MinBoundingBoxHeight,omitempty"`
	MinBoundingBoxWidth float64 `json:"MinBoundingBoxWidth,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
}

type DisassociateFacesRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	CollectionId string `json:"CollectionId,omitempty"`
	FaceIds []string `json:"FaceIds,omitempty"`
	UserId string `json:"UserId,omitempty"`
}

type DisassociateFacesResponse struct {
	DisassociatedFaces []DisassociatedFace `json:"DisassociatedFaces,omitempty"`
	UnsuccessfulFaceDisassociations []UnsuccessfulFaceDisassociation `json:"UnsuccessfulFaceDisassociations,omitempty"`
	UserStatus *string `json:"UserStatus,omitempty"`
}

type DisassociatedFace struct {
	FaceId *string `json:"FaceId,omitempty"`
}

type DistributeDataset struct {
	Arn string `json:"Arn,omitempty"`
}

type DistributeDatasetEntriesRequest struct {
	Datasets []DistributeDataset `json:"Datasets,omitempty"`
}

type DistributeDatasetEntriesResponse struct {
}

type DominantColor struct {
	Blue int `json:"Blue,omitempty"`
	CSSColor *string `json:"CSSColor,omitempty"`
	Green int `json:"Green,omitempty"`
	HexCode *string `json:"HexCode,omitempty"`
	PixelPercent float64 `json:"PixelPercent,omitempty"`
	Red int `json:"Red,omitempty"`
	SimplifiedColor *string `json:"SimplifiedColor,omitempty"`
}

type Emotion struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type EquipmentDetection struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	CoversBodyPart *CoversBodyPart `json:"CoversBodyPart,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type EvaluationResult struct {
	F1Score float64 `json:"F1Score,omitempty"`
	Summary *Summary `json:"Summary,omitempty"`
}

type EyeDirection struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Pitch float64 `json:"Pitch,omitempty"`
	Yaw float64 `json:"Yaw,omitempty"`
}

type EyeOpen struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type Eyeglasses struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type Face struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	ExternalImageId *string `json:"ExternalImageId,omitempty"`
	FaceId *string `json:"FaceId,omitempty"`
	ImageId *string `json:"ImageId,omitempty"`
	IndexFacesModelVersion *string `json:"IndexFacesModelVersion,omitempty"`
	UserId *string `json:"UserId,omitempty"`
}

type FaceDetail struct {
	AgeRange *AgeRange `json:"AgeRange,omitempty"`
	Beard *Beard `json:"Beard,omitempty"`
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	Emotions []Emotion `json:"Emotions,omitempty"`
	EyeDirection *EyeDirection `json:"EyeDirection,omitempty"`
	Eyeglasses *Eyeglasses `json:"Eyeglasses,omitempty"`
	EyesOpen *EyeOpen `json:"EyesOpen,omitempty"`
	FaceOccluded *FaceOccluded `json:"FaceOccluded,omitempty"`
	Gender *Gender `json:"Gender,omitempty"`
	Landmarks []Landmark `json:"Landmarks,omitempty"`
	MouthOpen *MouthOpen `json:"MouthOpen,omitempty"`
	Mustache *Mustache `json:"Mustache,omitempty"`
	Pose *Pose `json:"Pose,omitempty"`
	Quality *ImageQuality `json:"Quality,omitempty"`
	Smile *Smile `json:"Smile,omitempty"`
	Sunglasses *Sunglasses `json:"Sunglasses,omitempty"`
}

type FaceDetection struct {
	Face *FaceDetail `json:"Face,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type FaceMatch struct {
	Face *Face `json:"Face,omitempty"`
	Similarity float64 `json:"Similarity,omitempty"`
}

type FaceOccluded struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type FaceRecord struct {
	Face *Face `json:"Face,omitempty"`
	FaceDetail *FaceDetail `json:"FaceDetail,omitempty"`
}

type FaceSearchSettings struct {
	CollectionId *string `json:"CollectionId,omitempty"`
	FaceMatchThreshold float64 `json:"FaceMatchThreshold,omitempty"`
}

type Gender struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type GeneralLabelsSettings struct {
	LabelCategoryExclusionFilters []string `json:"LabelCategoryExclusionFilters,omitempty"`
	LabelCategoryInclusionFilters []string `json:"LabelCategoryInclusionFilters,omitempty"`
	LabelExclusionFilters []string `json:"LabelExclusionFilters,omitempty"`
	LabelInclusionFilters []string `json:"LabelInclusionFilters,omitempty"`
}

type Geometry struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Polygon []Point `json:"Polygon,omitempty"`
}

type GetCelebrityInfoRequest struct {
	Id string `json:"Id,omitempty"`
}

type GetCelebrityInfoResponse struct {
	KnownGender *KnownGender `json:"KnownGender,omitempty"`
	Name *string `json:"Name,omitempty"`
	Urls []string `json:"Urls,omitempty"`
}

type GetCelebrityRecognitionRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetCelebrityRecognitionResponse struct {
	Celebrities []CelebrityRecognition `json:"Celebrities,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetContentModerationRequest struct {
	AggregateBy *string `json:"AggregateBy,omitempty"`
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetContentModerationRequestMetadata struct {
	AggregateBy *string `json:"AggregateBy,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetContentModerationResponse struct {
	GetRequestMetadata *GetContentModerationRequestMetadata `json:"GetRequestMetadata,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	ModerationLabels []ContentModerationDetection `json:"ModerationLabels,omitempty"`
	ModerationModelVersion *string `json:"ModerationModelVersion,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetFaceDetectionRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetFaceDetectionResponse struct {
	Faces []FaceDetection `json:"Faces,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetFaceLivenessSessionResultsRequest struct {
	SessionId string `json:"SessionId,omitempty"`
}

type GetFaceLivenessSessionResultsResponse struct {
	AuditImages []AuditImage `json:"AuditImages,omitempty"`
	Challenge *Challenge `json:"Challenge,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	ReferenceImage *AuditImage `json:"ReferenceImage,omitempty"`
	SessionId string `json:"SessionId,omitempty"`
	Status string `json:"Status,omitempty"`
}

type GetFaceSearchRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetFaceSearchResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Persons []PersonMatch `json:"Persons,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetLabelDetectionRequest struct {
	AggregateBy *string `json:"AggregateBy,omitempty"`
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetLabelDetectionRequestMetadata struct {
	AggregateBy *string `json:"AggregateBy,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetLabelDetectionResponse struct {
	GetRequestMetadata *GetLabelDetectionRequestMetadata `json:"GetRequestMetadata,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	LabelModelVersion *string `json:"LabelModelVersion,omitempty"`
	Labels []LabelDetection `json:"Labels,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetMediaAnalysisJobRequest struct {
	JobId string `json:"JobId,omitempty"`
}

type GetMediaAnalysisJobResponse struct {
	CompletionTimestamp *time.Time `json:"CompletionTimestamp,omitempty"`
	CreationTimestamp time.Time `json:"CreationTimestamp,omitempty"`
	FailureDetails *MediaAnalysisJobFailureDetails `json:"FailureDetails,omitempty"`
	Input MediaAnalysisInput `json:"Input,omitempty"`
	JobId string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	ManifestSummary *MediaAnalysisManifestSummary `json:"ManifestSummary,omitempty"`
	OperationsConfig MediaAnalysisOperationsConfig `json:"OperationsConfig,omitempty"`
	OutputConfig MediaAnalysisOutputConfig `json:"OutputConfig,omitempty"`
	Results *MediaAnalysisResults `json:"Results,omitempty"`
	Status string `json:"Status,omitempty"`
}

type GetPersonTrackingRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
}

type GetPersonTrackingResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Persons []PersonDetection `json:"Persons,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetSegmentDetectionRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetSegmentDetectionResponse struct {
	AudioMetadata []AudioMetadata `json:"AudioMetadata,omitempty"`
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Segments []SegmentDetection `json:"Segments,omitempty"`
	SelectedSegmentTypes []SegmentTypeInfo `json:"SelectedSegmentTypes,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata []VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GetTextDetectionRequest struct {
	JobId string `json:"JobId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetTextDetectionResponse struct {
	JobId *string `json:"JobId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	TextDetections []TextDetectionResult `json:"TextDetections,omitempty"`
	TextModelVersion *string `json:"TextModelVersion,omitempty"`
	Video *Video `json:"Video,omitempty"`
	VideoMetadata *VideoMetadata `json:"VideoMetadata,omitempty"`
}

type GroundTruthManifest struct {
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type HumanLoopActivationOutput struct {
	HumanLoopActivationConditionsEvaluationResults *string `json:"HumanLoopActivationConditionsEvaluationResults,omitempty"`
	HumanLoopActivationReasons []string `json:"HumanLoopActivationReasons,omitempty"`
	HumanLoopArn *string `json:"HumanLoopArn,omitempty"`
}

type HumanLoopConfig struct {
	DataAttributes *HumanLoopDataAttributes `json:"DataAttributes,omitempty"`
	FlowDefinitionArn string `json:"FlowDefinitionArn,omitempty"`
	HumanLoopName string `json:"HumanLoopName,omitempty"`
}

type HumanLoopDataAttributes struct {
	ContentClassifiers []string `json:"ContentClassifiers,omitempty"`
}

type Image struct {
	Bytes []byte `json:"Bytes,omitempty"`
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type ImageQuality struct {
	Brightness float64 `json:"Brightness,omitempty"`
	Sharpness float64 `json:"Sharpness,omitempty"`
}

type IndexFacesRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	DetectionAttributes []string `json:"DetectionAttributes,omitempty"`
	ExternalImageId *string `json:"ExternalImageId,omitempty"`
	Image Image `json:"Image,omitempty"`
	MaxFaces int `json:"MaxFaces,omitempty"`
	QualityFilter *string `json:"QualityFilter,omitempty"`
}

type IndexFacesResponse struct {
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	FaceRecords []FaceRecord `json:"FaceRecords,omitempty"`
	OrientationCorrection *string `json:"OrientationCorrection,omitempty"`
	UnindexedFaces []UnindexedFace `json:"UnindexedFaces,omitempty"`
}

type Instance struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	DominantColors []DominantColor `json:"DominantColors,omitempty"`
}

type KinesisDataStream struct {
	Arn *string `json:"Arn,omitempty"`
}

type KinesisVideoStream struct {
	Arn *string `json:"Arn,omitempty"`
}

type KinesisVideoStreamStartSelector struct {
	FragmentNumber *string `json:"FragmentNumber,omitempty"`
	ProducerTimestamp int64 `json:"ProducerTimestamp,omitempty"`
}

type KnownGender struct {
	Type *string `json:"Type,omitempty"`
}

type Label struct {
	Aliases []LabelAlias `json:"Aliases,omitempty"`
	Categories []LabelCategory `json:"Categories,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	Instances []Instance `json:"Instances,omitempty"`
	Name *string `json:"Name,omitempty"`
	Parents []Parent `json:"Parents,omitempty"`
}

type LabelAlias struct {
	Name *string `json:"Name,omitempty"`
}

type LabelCategory struct {
	Name *string `json:"Name,omitempty"`
}

type LabelDetection struct {
	DurationMillis int64 `json:"DurationMillis,omitempty"`
	EndTimestampMillis int64 `json:"EndTimestampMillis,omitempty"`
	Label *Label `json:"Label,omitempty"`
	StartTimestampMillis int64 `json:"StartTimestampMillis,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type LabelDetectionSettings struct {
	GeneralLabels *GeneralLabelsSettings `json:"GeneralLabels,omitempty"`
}

type Landmark struct {
	Type *string `json:"Type,omitempty"`
	X float64 `json:"X,omitempty"`
	Y float64 `json:"Y,omitempty"`
}

type ListCollectionsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCollectionsResponse struct {
	CollectionIds []string `json:"CollectionIds,omitempty"`
	FaceModelVersions []string `json:"FaceModelVersions,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDatasetEntriesRequest struct {
	ContainsLabels []string `json:"ContainsLabels,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
	HasErrors bool `json:"HasErrors,omitempty"`
	Labeled bool `json:"Labeled,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SourceRefContains *string `json:"SourceRefContains,omitempty"`
}

type ListDatasetEntriesResponse struct {
	DatasetEntries []string `json:"DatasetEntries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDatasetLabelsRequest struct {
	DatasetArn string `json:"DatasetArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListDatasetLabelsResponse struct {
	DatasetLabelDescriptions []DatasetLabelDescription `json:"DatasetLabelDescriptions,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFacesRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	FaceIds []string `json:"FaceIds,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	UserId *string `json:"UserId,omitempty"`
}

type ListFacesResponse struct {
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	Faces []Face `json:"Faces,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListMediaAnalysisJobsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListMediaAnalysisJobsResponse struct {
	MediaAnalysisJobs []MediaAnalysisJobDescription `json:"MediaAnalysisJobs,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListProjectPoliciesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
}

type ListProjectPoliciesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ProjectPolicies []ProjectPolicy `json:"ProjectPolicies,omitempty"`
}

type ListStreamProcessorsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListStreamProcessorsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	StreamProcessors []StreamProcessor `json:"StreamProcessors,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags map[string]string `json:"Tags,omitempty"`
}

type ListUsersRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListUsersResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Users []User `json:"Users,omitempty"`
}

type LivenessOutputConfig struct {
	S3Bucket string `json:"S3Bucket,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
}

type MatchedUser struct {
	UserId *string `json:"UserId,omitempty"`
	UserStatus *string `json:"UserStatus,omitempty"`
}

type MediaAnalysisDetectModerationLabelsConfig struct {
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	ProjectVersion *string `json:"ProjectVersion,omitempty"`
}

type MediaAnalysisInput struct {
	S3Object S3Object `json:"S3Object,omitempty"`
}

type MediaAnalysisJobDescription struct {
	CompletionTimestamp *time.Time `json:"CompletionTimestamp,omitempty"`
	CreationTimestamp time.Time `json:"CreationTimestamp,omitempty"`
	FailureDetails *MediaAnalysisJobFailureDetails `json:"FailureDetails,omitempty"`
	Input MediaAnalysisInput `json:"Input,omitempty"`
	JobId string `json:"JobId,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	ManifestSummary *MediaAnalysisManifestSummary `json:"ManifestSummary,omitempty"`
	OperationsConfig MediaAnalysisOperationsConfig `json:"OperationsConfig,omitempty"`
	OutputConfig MediaAnalysisOutputConfig `json:"OutputConfig,omitempty"`
	Results *MediaAnalysisResults `json:"Results,omitempty"`
	Status string `json:"Status,omitempty"`
}

type MediaAnalysisJobFailureDetails struct {
	Code *string `json:"Code,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type MediaAnalysisManifestSummary struct {
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type MediaAnalysisModelVersions struct {
	Moderation *string `json:"Moderation,omitempty"`
}

type MediaAnalysisOperationsConfig struct {
	DetectModerationLabels *MediaAnalysisDetectModerationLabelsConfig `json:"DetectModerationLabels,omitempty"`
}

type MediaAnalysisOutputConfig struct {
	S3Bucket string `json:"S3Bucket,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
}

type MediaAnalysisResults struct {
	ModelVersions *MediaAnalysisModelVersions `json:"ModelVersions,omitempty"`
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type ModerationLabel struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParentName *string `json:"ParentName,omitempty"`
	TaxonomyLevel int `json:"TaxonomyLevel,omitempty"`
}

type MouthOpen struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type Mustache struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type NotificationChannel struct {
	RoleArn string `json:"RoleArn,omitempty"`
	SNSTopicArn string `json:"SNSTopicArn,omitempty"`
}

type OutputConfig struct {
	S3Bucket *string `json:"S3Bucket,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
}

type Parent struct {
	Name *string `json:"Name,omitempty"`
}

type PersonDetail struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Face *FaceDetail `json:"Face,omitempty"`
	Index int64 `json:"Index,omitempty"`
}

type PersonDetection struct {
	Person *PersonDetail `json:"Person,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type PersonMatch struct {
	FaceMatches []FaceMatch `json:"FaceMatches,omitempty"`
	Person *PersonDetail `json:"Person,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type Point struct {
	X float64 `json:"X,omitempty"`
	Y float64 `json:"Y,omitempty"`
}

type Pose struct {
	Pitch float64 `json:"Pitch,omitempty"`
	Roll float64 `json:"Roll,omitempty"`
	Yaw float64 `json:"Yaw,omitempty"`
}

type ProjectDescription struct {
	AutoUpdate *string `json:"AutoUpdate,omitempty"`
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	Datasets []DatasetMetadata `json:"Datasets,omitempty"`
	Feature *string `json:"Feature,omitempty"`
	ProjectArn *string `json:"ProjectArn,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type ProjectPolicy struct {
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	LastUpdatedTimestamp *time.Time `json:"LastUpdatedTimestamp,omitempty"`
	PolicyDocument *string `json:"PolicyDocument,omitempty"`
	PolicyName *string `json:"PolicyName,omitempty"`
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ProjectArn *string `json:"ProjectArn,omitempty"`
}

type ProjectVersionDescription struct {
	BaseModelVersion *string `json:"BaseModelVersion,omitempty"`
	BillableTrainingTimeInSeconds int64 `json:"BillableTrainingTimeInSeconds,omitempty"`
	CreationTimestamp *time.Time `json:"CreationTimestamp,omitempty"`
	EvaluationResult *EvaluationResult `json:"EvaluationResult,omitempty"`
	Feature *string `json:"Feature,omitempty"`
	FeatureConfig *CustomizationFeatureConfig `json:"FeatureConfig,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	ManifestSummary *GroundTruthManifest `json:"ManifestSummary,omitempty"`
	MaxInferenceUnits int `json:"MaxInferenceUnits,omitempty"`
	MinInferenceUnits int `json:"MinInferenceUnits,omitempty"`
	OutputConfig *OutputConfig `json:"OutputConfig,omitempty"`
	ProjectVersionArn *string `json:"ProjectVersionArn,omitempty"`
	SourceProjectVersionArn *string `json:"SourceProjectVersionArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	TestingDataResult *TestingDataResult `json:"TestingDataResult,omitempty"`
	TrainingDataResult *TrainingDataResult `json:"TrainingDataResult,omitempty"`
	TrainingEndTimestamp *time.Time `json:"TrainingEndTimestamp,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type ProtectiveEquipmentBodyPart struct {
	Confidence float64 `json:"Confidence,omitempty"`
	EquipmentDetections []EquipmentDetection `json:"EquipmentDetections,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ProtectiveEquipmentPerson struct {
	BodyParts []ProtectiveEquipmentBodyPart `json:"BodyParts,omitempty"`
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Confidence float64 `json:"Confidence,omitempty"`
	Id int `json:"Id,omitempty"`
}

type ProtectiveEquipmentSummarizationAttributes struct {
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	RequiredEquipmentTypes []string `json:"RequiredEquipmentTypes,omitempty"`
}

type ProtectiveEquipmentSummary struct {
	PersonsIndeterminate []int `json:"PersonsIndeterminate,omitempty"`
	PersonsWithRequiredEquipment []int `json:"PersonsWithRequiredEquipment,omitempty"`
	PersonsWithoutRequiredEquipment []int `json:"PersonsWithoutRequiredEquipment,omitempty"`
}

type PutProjectPolicyRequest struct {
	PolicyDocument string `json:"PolicyDocument,omitempty"`
	PolicyName string `json:"PolicyName,omitempty"`
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
	ProjectArn string `json:"ProjectArn,omitempty"`
}

type PutProjectPolicyResponse struct {
	PolicyRevisionId *string `json:"PolicyRevisionId,omitempty"`
}

type RecognizeCelebritiesRequest struct {
	Image Image `json:"Image,omitempty"`
}

type RecognizeCelebritiesResponse struct {
	CelebrityFaces []Celebrity `json:"CelebrityFaces,omitempty"`
	OrientationCorrection *string `json:"OrientationCorrection,omitempty"`
	UnrecognizedFaces []ComparedFace `json:"UnrecognizedFaces,omitempty"`
}

type RegionOfInterest struct {
	BoundingBox *BoundingBox `json:"BoundingBox,omitempty"`
	Polygon []Point `json:"Polygon,omitempty"`
}

type S3Destination struct {
	Bucket *string `json:"Bucket,omitempty"`
	KeyPrefix *string `json:"KeyPrefix,omitempty"`
}

type S3Object struct {
	Bucket *string `json:"Bucket,omitempty"`
	Name *string `json:"Name,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type SearchFacesByImageRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	FaceMatchThreshold float64 `json:"FaceMatchThreshold,omitempty"`
	Image Image `json:"Image,omitempty"`
	MaxFaces int `json:"MaxFaces,omitempty"`
	QualityFilter *string `json:"QualityFilter,omitempty"`
}

type SearchFacesByImageResponse struct {
	FaceMatches []FaceMatch `json:"FaceMatches,omitempty"`
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	SearchedFaceBoundingBox *BoundingBox `json:"SearchedFaceBoundingBox,omitempty"`
	SearchedFaceConfidence float64 `json:"SearchedFaceConfidence,omitempty"`
}

type SearchFacesRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	FaceId string `json:"FaceId,omitempty"`
	FaceMatchThreshold float64 `json:"FaceMatchThreshold,omitempty"`
	MaxFaces int `json:"MaxFaces,omitempty"`
}

type SearchFacesResponse struct {
	FaceMatches []FaceMatch `json:"FaceMatches,omitempty"`
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	SearchedFaceId *string `json:"SearchedFaceId,omitempty"`
}

type SearchUsersByImageRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	Image Image `json:"Image,omitempty"`
	MaxUsers int `json:"MaxUsers,omitempty"`
	QualityFilter *string `json:"QualityFilter,omitempty"`
	UserMatchThreshold float64 `json:"UserMatchThreshold,omitempty"`
}

type SearchUsersByImageResponse struct {
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	SearchedFace *SearchedFaceDetails `json:"SearchedFace,omitempty"`
	UnsearchedFaces []UnsearchedFace `json:"UnsearchedFaces,omitempty"`
	UserMatches []UserMatch `json:"UserMatches,omitempty"`
}

type SearchUsersRequest struct {
	CollectionId string `json:"CollectionId,omitempty"`
	FaceId *string `json:"FaceId,omitempty"`
	MaxUsers int `json:"MaxUsers,omitempty"`
	UserId *string `json:"UserId,omitempty"`
	UserMatchThreshold float64 `json:"UserMatchThreshold,omitempty"`
}

type SearchUsersResponse struct {
	FaceModelVersion *string `json:"FaceModelVersion,omitempty"`
	SearchedFace *SearchedFace `json:"SearchedFace,omitempty"`
	SearchedUser *SearchedUser `json:"SearchedUser,omitempty"`
	UserMatches []UserMatch `json:"UserMatches,omitempty"`
}

type SearchedFace struct {
	FaceId *string `json:"FaceId,omitempty"`
}

type SearchedFaceDetails struct {
	FaceDetail *FaceDetail `json:"FaceDetail,omitempty"`
}

type SearchedUser struct {
	UserId *string `json:"UserId,omitempty"`
}

type SegmentDetection struct {
	DurationFrames int64 `json:"DurationFrames,omitempty"`
	DurationMillis int64 `json:"DurationMillis,omitempty"`
	DurationSMPTE *string `json:"DurationSMPTE,omitempty"`
	EndFrameNumber int64 `json:"EndFrameNumber,omitempty"`
	EndTimecodeSMPTE *string `json:"EndTimecodeSMPTE,omitempty"`
	EndTimestampMillis int64 `json:"EndTimestampMillis,omitempty"`
	ShotSegment *ShotSegment `json:"ShotSegment,omitempty"`
	StartFrameNumber int64 `json:"StartFrameNumber,omitempty"`
	StartTimecodeSMPTE *string `json:"StartTimecodeSMPTE,omitempty"`
	StartTimestampMillis int64 `json:"StartTimestampMillis,omitempty"`
	TechnicalCueSegment *TechnicalCueSegment `json:"TechnicalCueSegment,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type SegmentTypeInfo struct {
	ModelVersion *string `json:"ModelVersion,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ShotSegment struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Index int64 `json:"Index,omitempty"`
}

type Smile struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type StartCelebrityRecognitionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartCelebrityRecognitionResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartContentModerationRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartContentModerationResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartFaceDetectionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	FaceAttributes *string `json:"FaceAttributes,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartFaceDetectionResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartFaceSearchRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	CollectionId string `json:"CollectionId,omitempty"`
	FaceMatchThreshold float64 `json:"FaceMatchThreshold,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartFaceSearchResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartLabelDetectionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	Features []string `json:"Features,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	MinConfidence float64 `json:"MinConfidence,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Settings *LabelDetectionSettings `json:"Settings,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartLabelDetectionResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartMediaAnalysisJobRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	Input MediaAnalysisInput `json:"Input,omitempty"`
	JobName *string `json:"JobName,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	OperationsConfig MediaAnalysisOperationsConfig `json:"OperationsConfig,omitempty"`
	OutputConfig MediaAnalysisOutputConfig `json:"OutputConfig,omitempty"`
}

type StartMediaAnalysisJobResponse struct {
	JobId string `json:"JobId,omitempty"`
}

type StartPersonTrackingRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartPersonTrackingResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartProjectVersionRequest struct {
	MaxInferenceUnits int `json:"MaxInferenceUnits,omitempty"`
	MinInferenceUnits int `json:"MinInferenceUnits,omitempty"`
	ProjectVersionArn string `json:"ProjectVersionArn,omitempty"`
}

type StartProjectVersionResponse struct {
	Status *string `json:"Status,omitempty"`
}

type StartSegmentDetectionFilters struct {
	ShotFilter *StartShotDetectionFilter `json:"ShotFilter,omitempty"`
	TechnicalCueFilter *StartTechnicalCueDetectionFilter `json:"TechnicalCueFilter,omitempty"`
}

type StartSegmentDetectionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	Filters *StartSegmentDetectionFilters `json:"Filters,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	SegmentTypes []string `json:"SegmentTypes,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartSegmentDetectionResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StartShotDetectionFilter struct {
	MinSegmentConfidence float64 `json:"MinSegmentConfidence,omitempty"`
}

type StartStreamProcessorRequest struct {
	Name string `json:"Name,omitempty"`
	StartSelector *StreamProcessingStartSelector `json:"StartSelector,omitempty"`
	StopSelector *StreamProcessingStopSelector `json:"StopSelector,omitempty"`
}

type StartStreamProcessorResponse struct {
	SessionId *string `json:"SessionId,omitempty"`
}

type StartTechnicalCueDetectionFilter struct {
	BlackFrame *BlackFrame `json:"BlackFrame,omitempty"`
	MinSegmentConfidence float64 `json:"MinSegmentConfidence,omitempty"`
}

type StartTextDetectionFilters struct {
	RegionsOfInterest []RegionOfInterest `json:"RegionsOfInterest,omitempty"`
	WordFilter *DetectionFilter `json:"WordFilter,omitempty"`
}

type StartTextDetectionRequest struct {
	ClientRequestToken *string `json:"ClientRequestToken,omitempty"`
	Filters *StartTextDetectionFilters `json:"Filters,omitempty"`
	JobTag *string `json:"JobTag,omitempty"`
	NotificationChannel *NotificationChannel `json:"NotificationChannel,omitempty"`
	Video Video `json:"Video,omitempty"`
}

type StartTextDetectionResponse struct {
	JobId *string `json:"JobId,omitempty"`
}

type StopProjectVersionRequest struct {
	ProjectVersionArn string `json:"ProjectVersionArn,omitempty"`
}

type StopProjectVersionResponse struct {
	Status *string `json:"Status,omitempty"`
}

type StopStreamProcessorRequest struct {
	Name string `json:"Name,omitempty"`
}

type StopStreamProcessorResponse struct {
}

type StreamProcessingStartSelector struct {
	KVSStreamStartSelector *KinesisVideoStreamStartSelector `json:"KVSStreamStartSelector,omitempty"`
}

type StreamProcessingStopSelector struct {
	MaxDurationInSeconds int64 `json:"MaxDurationInSeconds,omitempty"`
}

type StreamProcessor struct {
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type StreamProcessorDataSharingPreference struct {
	OptIn bool `json:"OptIn,omitempty"`
}

type StreamProcessorInput struct {
	KinesisVideoStream *KinesisVideoStream `json:"KinesisVideoStream,omitempty"`
}

type StreamProcessorNotificationChannel struct {
	SNSTopicArn string `json:"SNSTopicArn,omitempty"`
}

type StreamProcessorOutput struct {
	KinesisDataStream *KinesisDataStream `json:"KinesisDataStream,omitempty"`
	S3Destination *S3Destination `json:"S3Destination,omitempty"`
}

type StreamProcessorSettings struct {
	ConnectedHome *ConnectedHomeSettings `json:"ConnectedHome,omitempty"`
	FaceSearch *FaceSearchSettings `json:"FaceSearch,omitempty"`
}

type StreamProcessorSettingsForUpdate struct {
	ConnectedHomeForUpdate *ConnectedHomeSettingsForUpdate `json:"ConnectedHomeForUpdate,omitempty"`
}

type Summary struct {
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type Sunglasses struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Value bool `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
}

type TechnicalCueSegment struct {
	Confidence float64 `json:"Confidence,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type TestingData struct {
	Assets []Asset `json:"Assets,omitempty"`
	AutoCreate bool `json:"AutoCreate,omitempty"`
}

type TestingDataResult struct {
	Input *TestingData `json:"Input,omitempty"`
	Output *TestingData `json:"Output,omitempty"`
	Validation *ValidationData `json:"Validation,omitempty"`
}

type TextDetection struct {
	Confidence float64 `json:"Confidence,omitempty"`
	DetectedText *string `json:"DetectedText,omitempty"`
	Geometry *Geometry `json:"Geometry,omitempty"`
	Id int `json:"Id,omitempty"`
	ParentId int `json:"ParentId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type TextDetectionResult struct {
	TextDetection *TextDetection `json:"TextDetection,omitempty"`
	Timestamp int64 `json:"Timestamp,omitempty"`
}

type TrainingData struct {
	Assets []Asset `json:"Assets,omitempty"`
}

type TrainingDataResult struct {
	Input *TrainingData `json:"Input,omitempty"`
	Output *TrainingData `json:"Output,omitempty"`
	Validation *ValidationData `json:"Validation,omitempty"`
}

type UnindexedFace struct {
	FaceDetail *FaceDetail `json:"FaceDetail,omitempty"`
	Reasons []string `json:"Reasons,omitempty"`
}

type UnsearchedFace struct {
	FaceDetails *FaceDetail `json:"FaceDetails,omitempty"`
	Reasons []string `json:"Reasons,omitempty"`
}

type UnsuccessfulFaceAssociation struct {
	Confidence float64 `json:"Confidence,omitempty"`
	FaceId *string `json:"FaceId,omitempty"`
	Reasons []string `json:"Reasons,omitempty"`
	UserId *string `json:"UserId,omitempty"`
}

type UnsuccessfulFaceDeletion struct {
	FaceId *string `json:"FaceId,omitempty"`
	Reasons []string `json:"Reasons,omitempty"`
	UserId *string `json:"UserId,omitempty"`
}

type UnsuccessfulFaceDisassociation struct {
	FaceId *string `json:"FaceId,omitempty"`
	Reasons []string `json:"Reasons,omitempty"`
	UserId *string `json:"UserId,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"TagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateDatasetEntriesRequest struct {
	Changes DatasetChanges `json:"Changes,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
}

type UpdateDatasetEntriesResponse struct {
}

type UpdateStreamProcessorRequest struct {
	DataSharingPreferenceForUpdate *StreamProcessorDataSharingPreference `json:"DataSharingPreferenceForUpdate,omitempty"`
	Name string `json:"Name,omitempty"`
	ParametersToDelete []string `json:"ParametersToDelete,omitempty"`
	RegionsOfInterestForUpdate []RegionOfInterest `json:"RegionsOfInterestForUpdate,omitempty"`
	SettingsForUpdate *StreamProcessorSettingsForUpdate `json:"SettingsForUpdate,omitempty"`
}

type UpdateStreamProcessorResponse struct {
}

type User struct {
	UserId *string `json:"UserId,omitempty"`
	UserStatus *string `json:"UserStatus,omitempty"`
}

type UserMatch struct {
	Similarity float64 `json:"Similarity,omitempty"`
	User *MatchedUser `json:"User,omitempty"`
}

type ValidationData struct {
	Assets []Asset `json:"Assets,omitempty"`
}

type Versions struct {
	Maximum *string `json:"Maximum,omitempty"`
	Minimum *string `json:"Minimum,omitempty"`
}

type Video struct {
	S3Object *S3Object `json:"S3Object,omitempty"`
}

type VideoMetadata struct {
	Codec *string `json:"Codec,omitempty"`
	ColorRange *string `json:"ColorRange,omitempty"`
	DurationMillis int64 `json:"DurationMillis,omitempty"`
	Format *string `json:"Format,omitempty"`
	FrameHeight int64 `json:"FrameHeight,omitempty"`
	FrameRate float64 `json:"FrameRate,omitempty"`
	FrameWidth int64 `json:"FrameWidth,omitempty"`
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

func handleAssociateFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateFaces"})
}

func handleCompareFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CompareFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CompareFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CompareFaces"})
}

func handleCopyProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CopyProjectVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CopyProjectVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CopyProjectVersion"})
}

func handleCreateCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCollectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCollection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCollection"})
}

func handleCreateDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDatasetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDataset business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDataset"})
}

func handleCreateFaceLivenessSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFaceLivenessSessionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFaceLivenessSession business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFaceLivenessSession"})
}

func handleCreateProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateProjectRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateProject business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateProject"})
}

func handleCreateProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateProjectVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateProjectVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateProjectVersion"})
}

func handleCreateStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateStreamProcessor"})
}

func handleCreateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateUser"})
}

func handleDeleteCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCollectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCollection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCollection"})
}

func handleDeleteDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDatasetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDataset business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDataset"})
}

func handleDeleteFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFaces"})
}

func handleDeleteProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProjectRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProject business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProject"})
}

func handleDeleteProjectPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProjectPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProjectPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProjectPolicy"})
}

func handleDeleteProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProjectVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProjectVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProjectVersion"})
}

func handleDeleteStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteStreamProcessor"})
}

func handleDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteUser"})
}

func handleDescribeCollection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCollectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCollection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCollection"})
}

func handleDescribeDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDatasetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataset business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataset"})
}

func handleDescribeProjectVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProjectVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProjectVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProjectVersions"})
}

func handleDescribeProjects(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProjectsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProjects business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProjects"})
}

func handleDescribeStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeStreamProcessor"})
}

func handleDetectCustomLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectCustomLabelsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectCustomLabels business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectCustomLabels"})
}

func handleDetectFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectFaces"})
}

func handleDetectLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectLabelsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectLabels business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectLabels"})
}

func handleDetectModerationLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectModerationLabelsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectModerationLabels business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectModerationLabels"})
}

func handleDetectProtectiveEquipment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectProtectiveEquipmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectProtectiveEquipment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectProtectiveEquipment"})
}

func handleDetectText(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DetectTextRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DetectText business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DetectText"})
}

func handleDisassociateFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateFaces"})
}

func handleDistributeDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DistributeDatasetEntriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DistributeDatasetEntries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DistributeDatasetEntries"})
}

func handleGetCelebrityInfo(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCelebrityInfoRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCelebrityInfo business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCelebrityInfo"})
}

func handleGetCelebrityRecognition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCelebrityRecognitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCelebrityRecognition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCelebrityRecognition"})
}

func handleGetContentModeration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetContentModerationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetContentModeration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetContentModeration"})
}

func handleGetFaceDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFaceDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFaceDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFaceDetection"})
}

func handleGetFaceLivenessSessionResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFaceLivenessSessionResultsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFaceLivenessSessionResults business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFaceLivenessSessionResults"})
}

func handleGetFaceSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFaceSearchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFaceSearch business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFaceSearch"})
}

func handleGetLabelDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetLabelDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetLabelDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetLabelDetection"})
}

func handleGetMediaAnalysisJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMediaAnalysisJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMediaAnalysisJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMediaAnalysisJob"})
}

func handleGetPersonTracking(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetPersonTrackingRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetPersonTracking business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetPersonTracking"})
}

func handleGetSegmentDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSegmentDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSegmentDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSegmentDetection"})
}

func handleGetTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTextDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetTextDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetTextDetection"})
}

func handleIndexFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req IndexFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement IndexFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "IndexFaces"})
}

func handleListCollections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCollectionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCollections business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCollections"})
}

func handleListDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDatasetEntriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDatasetEntries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDatasetEntries"})
}

func handleListDatasetLabels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDatasetLabelsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDatasetLabels business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDatasetLabels"})
}

func handleListFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFaces"})
}

func handleListMediaAnalysisJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListMediaAnalysisJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListMediaAnalysisJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListMediaAnalysisJobs"})
}

func handleListProjectPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListProjectPoliciesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListProjectPolicies business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListProjectPolicies"})
}

func handleListStreamProcessors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListStreamProcessorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListStreamProcessors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListStreamProcessors"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListUsersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListUsers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListUsers"})
}

func handlePutProjectPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutProjectPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutProjectPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutProjectPolicy"})
}

func handleRecognizeCelebrities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RecognizeCelebritiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RecognizeCelebrities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RecognizeCelebrities"})
}

func handleSearchFaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchFacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchFaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchFaces"})
}

func handleSearchFacesByImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchFacesByImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchFacesByImage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchFacesByImage"})
}

func handleSearchUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchUsersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchUsers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchUsers"})
}

func handleSearchUsersByImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchUsersByImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchUsersByImage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchUsersByImage"})
}

func handleStartCelebrityRecognition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartCelebrityRecognitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartCelebrityRecognition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartCelebrityRecognition"})
}

func handleStartContentModeration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartContentModerationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartContentModeration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartContentModeration"})
}

func handleStartFaceDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartFaceDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartFaceDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartFaceDetection"})
}

func handleStartFaceSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartFaceSearchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartFaceSearch business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartFaceSearch"})
}

func handleStartLabelDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartLabelDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartLabelDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartLabelDetection"})
}

func handleStartMediaAnalysisJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartMediaAnalysisJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartMediaAnalysisJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartMediaAnalysisJob"})
}

func handleStartPersonTracking(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartPersonTrackingRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartPersonTracking business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartPersonTracking"})
}

func handleStartProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartProjectVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartProjectVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartProjectVersion"})
}

func handleStartSegmentDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartSegmentDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartSegmentDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartSegmentDetection"})
}

func handleStartStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartStreamProcessor"})
}

func handleStartTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartTextDetectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartTextDetection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartTextDetection"})
}

func handleStopProjectVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopProjectVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopProjectVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopProjectVersion"})
}

func handleStopStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopStreamProcessor"})
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

func handleUpdateDatasetEntries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDatasetEntriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDatasetEntries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDatasetEntries"})
}

func handleUpdateStreamProcessor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateStreamProcessorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateStreamProcessor business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateStreamProcessor"})
}

