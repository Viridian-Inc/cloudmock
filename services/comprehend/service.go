package comprehend

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS comprehend service.
type Service struct {
	store *Store
}

// New returns a new comprehend Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "comprehend" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "BatchDetectDominantLanguage", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectDominantLanguage"},
		{Name: "BatchDetectEntities", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectEntities"},
		{Name: "BatchDetectKeyPhrases", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectKeyPhrases"},
		{Name: "BatchDetectSentiment", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectSentiment"},
		{Name: "BatchDetectSyntax", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectSyntax"},
		{Name: "BatchDetectTargetedSentiment", Method: http.MethodPost, IAMAction: "comprehend:BatchDetectTargetedSentiment"},
		{Name: "ClassifyDocument", Method: http.MethodPost, IAMAction: "comprehend:ClassifyDocument"},
		{Name: "ContainsPiiEntities", Method: http.MethodPost, IAMAction: "comprehend:ContainsPiiEntities"},
		{Name: "CreateDataset", Method: http.MethodPost, IAMAction: "comprehend:CreateDataset"},
		{Name: "CreateDocumentClassifier", Method: http.MethodPost, IAMAction: "comprehend:CreateDocumentClassifier"},
		{Name: "CreateEndpoint", Method: http.MethodPost, IAMAction: "comprehend:CreateEndpoint"},
		{Name: "CreateEntityRecognizer", Method: http.MethodPost, IAMAction: "comprehend:CreateEntityRecognizer"},
		{Name: "CreateFlywheel", Method: http.MethodPost, IAMAction: "comprehend:CreateFlywheel"},
		{Name: "DeleteDocumentClassifier", Method: http.MethodPost, IAMAction: "comprehend:DeleteDocumentClassifier"},
		{Name: "DeleteEndpoint", Method: http.MethodPost, IAMAction: "comprehend:DeleteEndpoint"},
		{Name: "DeleteEntityRecognizer", Method: http.MethodPost, IAMAction: "comprehend:DeleteEntityRecognizer"},
		{Name: "DeleteFlywheel", Method: http.MethodPost, IAMAction: "comprehend:DeleteFlywheel"},
		{Name: "DeleteResourcePolicy", Method: http.MethodPost, IAMAction: "comprehend:DeleteResourcePolicy"},
		{Name: "DescribeDataset", Method: http.MethodPost, IAMAction: "comprehend:DescribeDataset"},
		{Name: "DescribeDocumentClassificationJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeDocumentClassificationJob"},
		{Name: "DescribeDocumentClassifier", Method: http.MethodPost, IAMAction: "comprehend:DescribeDocumentClassifier"},
		{Name: "DescribeDominantLanguageDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeDominantLanguageDetectionJob"},
		{Name: "DescribeEndpoint", Method: http.MethodPost, IAMAction: "comprehend:DescribeEndpoint"},
		{Name: "DescribeEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeEntitiesDetectionJob"},
		{Name: "DescribeEntityRecognizer", Method: http.MethodPost, IAMAction: "comprehend:DescribeEntityRecognizer"},
		{Name: "DescribeEventsDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeEventsDetectionJob"},
		{Name: "DescribeFlywheel", Method: http.MethodPost, IAMAction: "comprehend:DescribeFlywheel"},
		{Name: "DescribeFlywheelIteration", Method: http.MethodPost, IAMAction: "comprehend:DescribeFlywheelIteration"},
		{Name: "DescribeKeyPhrasesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeKeyPhrasesDetectionJob"},
		{Name: "DescribePiiEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribePiiEntitiesDetectionJob"},
		{Name: "DescribeResourcePolicy", Method: http.MethodPost, IAMAction: "comprehend:DescribeResourcePolicy"},
		{Name: "DescribeSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeSentimentDetectionJob"},
		{Name: "DescribeTargetedSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeTargetedSentimentDetectionJob"},
		{Name: "DescribeTopicsDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:DescribeTopicsDetectionJob"},
		{Name: "DetectDominantLanguage", Method: http.MethodPost, IAMAction: "comprehend:DetectDominantLanguage"},
		{Name: "DetectEntities", Method: http.MethodPost, IAMAction: "comprehend:DetectEntities"},
		{Name: "DetectKeyPhrases", Method: http.MethodPost, IAMAction: "comprehend:DetectKeyPhrases"},
		{Name: "DetectPiiEntities", Method: http.MethodPost, IAMAction: "comprehend:DetectPiiEntities"},
		{Name: "DetectSentiment", Method: http.MethodPost, IAMAction: "comprehend:DetectSentiment"},
		{Name: "DetectSyntax", Method: http.MethodPost, IAMAction: "comprehend:DetectSyntax"},
		{Name: "DetectTargetedSentiment", Method: http.MethodPost, IAMAction: "comprehend:DetectTargetedSentiment"},
		{Name: "DetectToxicContent", Method: http.MethodPost, IAMAction: "comprehend:DetectToxicContent"},
		{Name: "ImportModel", Method: http.MethodPost, IAMAction: "comprehend:ImportModel"},
		{Name: "ListDatasets", Method: http.MethodPost, IAMAction: "comprehend:ListDatasets"},
		{Name: "ListDocumentClassificationJobs", Method: http.MethodPost, IAMAction: "comprehend:ListDocumentClassificationJobs"},
		{Name: "ListDocumentClassifierSummaries", Method: http.MethodPost, IAMAction: "comprehend:ListDocumentClassifierSummaries"},
		{Name: "ListDocumentClassifiers", Method: http.MethodPost, IAMAction: "comprehend:ListDocumentClassifiers"},
		{Name: "ListDominantLanguageDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListDominantLanguageDetectionJobs"},
		{Name: "ListEndpoints", Method: http.MethodPost, IAMAction: "comprehend:ListEndpoints"},
		{Name: "ListEntitiesDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListEntitiesDetectionJobs"},
		{Name: "ListEntityRecognizerSummaries", Method: http.MethodPost, IAMAction: "comprehend:ListEntityRecognizerSummaries"},
		{Name: "ListEntityRecognizers", Method: http.MethodPost, IAMAction: "comprehend:ListEntityRecognizers"},
		{Name: "ListEventsDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListEventsDetectionJobs"},
		{Name: "ListFlywheelIterationHistory", Method: http.MethodPost, IAMAction: "comprehend:ListFlywheelIterationHistory"},
		{Name: "ListFlywheels", Method: http.MethodPost, IAMAction: "comprehend:ListFlywheels"},
		{Name: "ListKeyPhrasesDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListKeyPhrasesDetectionJobs"},
		{Name: "ListPiiEntitiesDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListPiiEntitiesDetectionJobs"},
		{Name: "ListSentimentDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListSentimentDetectionJobs"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "comprehend:ListTagsForResource"},
		{Name: "ListTargetedSentimentDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListTargetedSentimentDetectionJobs"},
		{Name: "ListTopicsDetectionJobs", Method: http.MethodPost, IAMAction: "comprehend:ListTopicsDetectionJobs"},
		{Name: "PutResourcePolicy", Method: http.MethodPost, IAMAction: "comprehend:PutResourcePolicy"},
		{Name: "StartDocumentClassificationJob", Method: http.MethodPost, IAMAction: "comprehend:StartDocumentClassificationJob"},
		{Name: "StartDominantLanguageDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartDominantLanguageDetectionJob"},
		{Name: "StartEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartEntitiesDetectionJob"},
		{Name: "StartEventsDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartEventsDetectionJob"},
		{Name: "StartFlywheelIteration", Method: http.MethodPost, IAMAction: "comprehend:StartFlywheelIteration"},
		{Name: "StartKeyPhrasesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartKeyPhrasesDetectionJob"},
		{Name: "StartPiiEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartPiiEntitiesDetectionJob"},
		{Name: "StartSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartSentimentDetectionJob"},
		{Name: "StartTargetedSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartTargetedSentimentDetectionJob"},
		{Name: "StartTopicsDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StartTopicsDetectionJob"},
		{Name: "StopDominantLanguageDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopDominantLanguageDetectionJob"},
		{Name: "StopEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopEntitiesDetectionJob"},
		{Name: "StopEventsDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopEventsDetectionJob"},
		{Name: "StopKeyPhrasesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopKeyPhrasesDetectionJob"},
		{Name: "StopPiiEntitiesDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopPiiEntitiesDetectionJob"},
		{Name: "StopSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopSentimentDetectionJob"},
		{Name: "StopTargetedSentimentDetectionJob", Method: http.MethodPost, IAMAction: "comprehend:StopTargetedSentimentDetectionJob"},
		{Name: "StopTrainingDocumentClassifier", Method: http.MethodPost, IAMAction: "comprehend:StopTrainingDocumentClassifier"},
		{Name: "StopTrainingEntityRecognizer", Method: http.MethodPost, IAMAction: "comprehend:StopTrainingEntityRecognizer"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "comprehend:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "comprehend:UntagResource"},
		{Name: "UpdateEndpoint", Method: http.MethodPost, IAMAction: "comprehend:UpdateEndpoint"},
		{Name: "UpdateFlywheel", Method: http.MethodPost, IAMAction: "comprehend:UpdateFlywheel"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "BatchDetectDominantLanguage":
		return handleBatchDetectDominantLanguage(ctx, s.store)
	case "BatchDetectEntities":
		return handleBatchDetectEntities(ctx, s.store)
	case "BatchDetectKeyPhrases":
		return handleBatchDetectKeyPhrases(ctx, s.store)
	case "BatchDetectSentiment":
		return handleBatchDetectSentiment(ctx, s.store)
	case "BatchDetectSyntax":
		return handleBatchDetectSyntax(ctx, s.store)
	case "BatchDetectTargetedSentiment":
		return handleBatchDetectTargetedSentiment(ctx, s.store)
	case "ClassifyDocument":
		return handleClassifyDocument(ctx, s.store)
	case "ContainsPiiEntities":
		return handleContainsPiiEntities(ctx, s.store)
	case "CreateDataset":
		return handleCreateDataset(ctx, s.store)
	case "CreateDocumentClassifier":
		return handleCreateDocumentClassifier(ctx, s.store)
	case "CreateEndpoint":
		return handleCreateEndpoint(ctx, s.store)
	case "CreateEntityRecognizer":
		return handleCreateEntityRecognizer(ctx, s.store)
	case "CreateFlywheel":
		return handleCreateFlywheel(ctx, s.store)
	case "DeleteDocumentClassifier":
		return handleDeleteDocumentClassifier(ctx, s.store)
	case "DeleteEndpoint":
		return handleDeleteEndpoint(ctx, s.store)
	case "DeleteEntityRecognizer":
		return handleDeleteEntityRecognizer(ctx, s.store)
	case "DeleteFlywheel":
		return handleDeleteFlywheel(ctx, s.store)
	case "DeleteResourcePolicy":
		return handleDeleteResourcePolicy(ctx, s.store)
	case "DescribeDataset":
		return handleDescribeDataset(ctx, s.store)
	case "DescribeDocumentClassificationJob":
		return handleDescribeDocumentClassificationJob(ctx, s.store)
	case "DescribeDocumentClassifier":
		return handleDescribeDocumentClassifier(ctx, s.store)
	case "DescribeDominantLanguageDetectionJob":
		return handleDescribeDominantLanguageDetectionJob(ctx, s.store)
	case "DescribeEndpoint":
		return handleDescribeEndpoint(ctx, s.store)
	case "DescribeEntitiesDetectionJob":
		return handleDescribeEntitiesDetectionJob(ctx, s.store)
	case "DescribeEntityRecognizer":
		return handleDescribeEntityRecognizer(ctx, s.store)
	case "DescribeEventsDetectionJob":
		return handleDescribeEventsDetectionJob(ctx, s.store)
	case "DescribeFlywheel":
		return handleDescribeFlywheel(ctx, s.store)
	case "DescribeFlywheelIteration":
		return handleDescribeFlywheelIteration(ctx, s.store)
	case "DescribeKeyPhrasesDetectionJob":
		return handleDescribeKeyPhrasesDetectionJob(ctx, s.store)
	case "DescribePiiEntitiesDetectionJob":
		return handleDescribePiiEntitiesDetectionJob(ctx, s.store)
	case "DescribeResourcePolicy":
		return handleDescribeResourcePolicy(ctx, s.store)
	case "DescribeSentimentDetectionJob":
		return handleDescribeSentimentDetectionJob(ctx, s.store)
	case "DescribeTargetedSentimentDetectionJob":
		return handleDescribeTargetedSentimentDetectionJob(ctx, s.store)
	case "DescribeTopicsDetectionJob":
		return handleDescribeTopicsDetectionJob(ctx, s.store)
	case "DetectDominantLanguage":
		return handleDetectDominantLanguage(ctx, s.store)
	case "DetectEntities":
		return handleDetectEntities(ctx, s.store)
	case "DetectKeyPhrases":
		return handleDetectKeyPhrases(ctx, s.store)
	case "DetectPiiEntities":
		return handleDetectPiiEntities(ctx, s.store)
	case "DetectSentiment":
		return handleDetectSentiment(ctx, s.store)
	case "DetectSyntax":
		return handleDetectSyntax(ctx, s.store)
	case "DetectTargetedSentiment":
		return handleDetectTargetedSentiment(ctx, s.store)
	case "DetectToxicContent":
		return handleDetectToxicContent(ctx, s.store)
	case "ImportModel":
		return handleImportModel(ctx, s.store)
	case "ListDatasets":
		return handleListDatasets(ctx, s.store)
	case "ListDocumentClassificationJobs":
		return handleListDocumentClassificationJobs(ctx, s.store)
	case "ListDocumentClassifierSummaries":
		return handleListDocumentClassifierSummaries(ctx, s.store)
	case "ListDocumentClassifiers":
		return handleListDocumentClassifiers(ctx, s.store)
	case "ListDominantLanguageDetectionJobs":
		return handleListDominantLanguageDetectionJobs(ctx, s.store)
	case "ListEndpoints":
		return handleListEndpoints(ctx, s.store)
	case "ListEntitiesDetectionJobs":
		return handleListEntitiesDetectionJobs(ctx, s.store)
	case "ListEntityRecognizerSummaries":
		return handleListEntityRecognizerSummaries(ctx, s.store)
	case "ListEntityRecognizers":
		return handleListEntityRecognizers(ctx, s.store)
	case "ListEventsDetectionJobs":
		return handleListEventsDetectionJobs(ctx, s.store)
	case "ListFlywheelIterationHistory":
		return handleListFlywheelIterationHistory(ctx, s.store)
	case "ListFlywheels":
		return handleListFlywheels(ctx, s.store)
	case "ListKeyPhrasesDetectionJobs":
		return handleListKeyPhrasesDetectionJobs(ctx, s.store)
	case "ListPiiEntitiesDetectionJobs":
		return handleListPiiEntitiesDetectionJobs(ctx, s.store)
	case "ListSentimentDetectionJobs":
		return handleListSentimentDetectionJobs(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListTargetedSentimentDetectionJobs":
		return handleListTargetedSentimentDetectionJobs(ctx, s.store)
	case "ListTopicsDetectionJobs":
		return handleListTopicsDetectionJobs(ctx, s.store)
	case "PutResourcePolicy":
		return handlePutResourcePolicy(ctx, s.store)
	case "StartDocumentClassificationJob":
		return handleStartDocumentClassificationJob(ctx, s.store)
	case "StartDominantLanguageDetectionJob":
		return handleStartDominantLanguageDetectionJob(ctx, s.store)
	case "StartEntitiesDetectionJob":
		return handleStartEntitiesDetectionJob(ctx, s.store)
	case "StartEventsDetectionJob":
		return handleStartEventsDetectionJob(ctx, s.store)
	case "StartFlywheelIteration":
		return handleStartFlywheelIteration(ctx, s.store)
	case "StartKeyPhrasesDetectionJob":
		return handleStartKeyPhrasesDetectionJob(ctx, s.store)
	case "StartPiiEntitiesDetectionJob":
		return handleStartPiiEntitiesDetectionJob(ctx, s.store)
	case "StartSentimentDetectionJob":
		return handleStartSentimentDetectionJob(ctx, s.store)
	case "StartTargetedSentimentDetectionJob":
		return handleStartTargetedSentimentDetectionJob(ctx, s.store)
	case "StartTopicsDetectionJob":
		return handleStartTopicsDetectionJob(ctx, s.store)
	case "StopDominantLanguageDetectionJob":
		return handleStopDominantLanguageDetectionJob(ctx, s.store)
	case "StopEntitiesDetectionJob":
		return handleStopEntitiesDetectionJob(ctx, s.store)
	case "StopEventsDetectionJob":
		return handleStopEventsDetectionJob(ctx, s.store)
	case "StopKeyPhrasesDetectionJob":
		return handleStopKeyPhrasesDetectionJob(ctx, s.store)
	case "StopPiiEntitiesDetectionJob":
		return handleStopPiiEntitiesDetectionJob(ctx, s.store)
	case "StopSentimentDetectionJob":
		return handleStopSentimentDetectionJob(ctx, s.store)
	case "StopTargetedSentimentDetectionJob":
		return handleStopTargetedSentimentDetectionJob(ctx, s.store)
	case "StopTrainingDocumentClassifier":
		return handleStopTrainingDocumentClassifier(ctx, s.store)
	case "StopTrainingEntityRecognizer":
		return handleStopTrainingEntityRecognizer(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateEndpoint":
		return handleUpdateEndpoint(ctx, s.store)
	case "UpdateFlywheel":
		return handleUpdateFlywheel(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
