package rekognition

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS rekognition service.
type Service struct {
	store *Store
}

// New returns a new rekognition Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "rekognition" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AssociateFaces", Method: http.MethodPost, IAMAction: "rekognition:AssociateFaces"},
		{Name: "CompareFaces", Method: http.MethodPost, IAMAction: "rekognition:CompareFaces"},
		{Name: "CopyProjectVersion", Method: http.MethodPost, IAMAction: "rekognition:CopyProjectVersion"},
		{Name: "CreateCollection", Method: http.MethodPost, IAMAction: "rekognition:CreateCollection"},
		{Name: "CreateDataset", Method: http.MethodPost, IAMAction: "rekognition:CreateDataset"},
		{Name: "CreateFaceLivenessSession", Method: http.MethodPost, IAMAction: "rekognition:CreateFaceLivenessSession"},
		{Name: "CreateProject", Method: http.MethodPost, IAMAction: "rekognition:CreateProject"},
		{Name: "CreateProjectVersion", Method: http.MethodPost, IAMAction: "rekognition:CreateProjectVersion"},
		{Name: "CreateStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:CreateStreamProcessor"},
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "rekognition:CreateUser"},
		{Name: "DeleteCollection", Method: http.MethodPost, IAMAction: "rekognition:DeleteCollection"},
		{Name: "DeleteDataset", Method: http.MethodPost, IAMAction: "rekognition:DeleteDataset"},
		{Name: "DeleteFaces", Method: http.MethodPost, IAMAction: "rekognition:DeleteFaces"},
		{Name: "DeleteProject", Method: http.MethodPost, IAMAction: "rekognition:DeleteProject"},
		{Name: "DeleteProjectPolicy", Method: http.MethodPost, IAMAction: "rekognition:DeleteProjectPolicy"},
		{Name: "DeleteProjectVersion", Method: http.MethodPost, IAMAction: "rekognition:DeleteProjectVersion"},
		{Name: "DeleteStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:DeleteStreamProcessor"},
		{Name: "DeleteUser", Method: http.MethodPost, IAMAction: "rekognition:DeleteUser"},
		{Name: "DescribeCollection", Method: http.MethodPost, IAMAction: "rekognition:DescribeCollection"},
		{Name: "DescribeDataset", Method: http.MethodPost, IAMAction: "rekognition:DescribeDataset"},
		{Name: "DescribeProjectVersions", Method: http.MethodPost, IAMAction: "rekognition:DescribeProjectVersions"},
		{Name: "DescribeProjects", Method: http.MethodPost, IAMAction: "rekognition:DescribeProjects"},
		{Name: "DescribeStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:DescribeStreamProcessor"},
		{Name: "DetectCustomLabels", Method: http.MethodPost, IAMAction: "rekognition:DetectCustomLabels"},
		{Name: "DetectFaces", Method: http.MethodPost, IAMAction: "rekognition:DetectFaces"},
		{Name: "DetectLabels", Method: http.MethodPost, IAMAction: "rekognition:DetectLabels"},
		{Name: "DetectModerationLabels", Method: http.MethodPost, IAMAction: "rekognition:DetectModerationLabels"},
		{Name: "DetectProtectiveEquipment", Method: http.MethodPost, IAMAction: "rekognition:DetectProtectiveEquipment"},
		{Name: "DetectText", Method: http.MethodPost, IAMAction: "rekognition:DetectText"},
		{Name: "DisassociateFaces", Method: http.MethodPost, IAMAction: "rekognition:DisassociateFaces"},
		{Name: "DistributeDatasetEntries", Method: http.MethodPost, IAMAction: "rekognition:DistributeDatasetEntries"},
		{Name: "GetCelebrityInfo", Method: http.MethodPost, IAMAction: "rekognition:GetCelebrityInfo"},
		{Name: "GetCelebrityRecognition", Method: http.MethodPost, IAMAction: "rekognition:GetCelebrityRecognition"},
		{Name: "GetContentModeration", Method: http.MethodPost, IAMAction: "rekognition:GetContentModeration"},
		{Name: "GetFaceDetection", Method: http.MethodPost, IAMAction: "rekognition:GetFaceDetection"},
		{Name: "GetFaceLivenessSessionResults", Method: http.MethodPost, IAMAction: "rekognition:GetFaceLivenessSessionResults"},
		{Name: "GetFaceSearch", Method: http.MethodPost, IAMAction: "rekognition:GetFaceSearch"},
		{Name: "GetLabelDetection", Method: http.MethodPost, IAMAction: "rekognition:GetLabelDetection"},
		{Name: "GetMediaAnalysisJob", Method: http.MethodPost, IAMAction: "rekognition:GetMediaAnalysisJob"},
		{Name: "GetPersonTracking", Method: http.MethodPost, IAMAction: "rekognition:GetPersonTracking"},
		{Name: "GetSegmentDetection", Method: http.MethodPost, IAMAction: "rekognition:GetSegmentDetection"},
		{Name: "GetTextDetection", Method: http.MethodPost, IAMAction: "rekognition:GetTextDetection"},
		{Name: "IndexFaces", Method: http.MethodPost, IAMAction: "rekognition:IndexFaces"},
		{Name: "ListCollections", Method: http.MethodPost, IAMAction: "rekognition:ListCollections"},
		{Name: "ListDatasetEntries", Method: http.MethodPost, IAMAction: "rekognition:ListDatasetEntries"},
		{Name: "ListDatasetLabels", Method: http.MethodPost, IAMAction: "rekognition:ListDatasetLabels"},
		{Name: "ListFaces", Method: http.MethodPost, IAMAction: "rekognition:ListFaces"},
		{Name: "ListMediaAnalysisJobs", Method: http.MethodPost, IAMAction: "rekognition:ListMediaAnalysisJobs"},
		{Name: "ListProjectPolicies", Method: http.MethodPost, IAMAction: "rekognition:ListProjectPolicies"},
		{Name: "ListStreamProcessors", Method: http.MethodPost, IAMAction: "rekognition:ListStreamProcessors"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "rekognition:ListTagsForResource"},
		{Name: "ListUsers", Method: http.MethodPost, IAMAction: "rekognition:ListUsers"},
		{Name: "PutProjectPolicy", Method: http.MethodPost, IAMAction: "rekognition:PutProjectPolicy"},
		{Name: "RecognizeCelebrities", Method: http.MethodPost, IAMAction: "rekognition:RecognizeCelebrities"},
		{Name: "SearchFaces", Method: http.MethodPost, IAMAction: "rekognition:SearchFaces"},
		{Name: "SearchFacesByImage", Method: http.MethodPost, IAMAction: "rekognition:SearchFacesByImage"},
		{Name: "SearchUsers", Method: http.MethodPost, IAMAction: "rekognition:SearchUsers"},
		{Name: "SearchUsersByImage", Method: http.MethodPost, IAMAction: "rekognition:SearchUsersByImage"},
		{Name: "StartCelebrityRecognition", Method: http.MethodPost, IAMAction: "rekognition:StartCelebrityRecognition"},
		{Name: "StartContentModeration", Method: http.MethodPost, IAMAction: "rekognition:StartContentModeration"},
		{Name: "StartFaceDetection", Method: http.MethodPost, IAMAction: "rekognition:StartFaceDetection"},
		{Name: "StartFaceSearch", Method: http.MethodPost, IAMAction: "rekognition:StartFaceSearch"},
		{Name: "StartLabelDetection", Method: http.MethodPost, IAMAction: "rekognition:StartLabelDetection"},
		{Name: "StartMediaAnalysisJob", Method: http.MethodPost, IAMAction: "rekognition:StartMediaAnalysisJob"},
		{Name: "StartPersonTracking", Method: http.MethodPost, IAMAction: "rekognition:StartPersonTracking"},
		{Name: "StartProjectVersion", Method: http.MethodPost, IAMAction: "rekognition:StartProjectVersion"},
		{Name: "StartSegmentDetection", Method: http.MethodPost, IAMAction: "rekognition:StartSegmentDetection"},
		{Name: "StartStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:StartStreamProcessor"},
		{Name: "StartTextDetection", Method: http.MethodPost, IAMAction: "rekognition:StartTextDetection"},
		{Name: "StopProjectVersion", Method: http.MethodPost, IAMAction: "rekognition:StopProjectVersion"},
		{Name: "StopStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:StopStreamProcessor"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "rekognition:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "rekognition:UntagResource"},
		{Name: "UpdateDatasetEntries", Method: http.MethodPost, IAMAction: "rekognition:UpdateDatasetEntries"},
		{Name: "UpdateStreamProcessor", Method: http.MethodPost, IAMAction: "rekognition:UpdateStreamProcessor"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "AssociateFaces":
		return handleAssociateFaces(ctx, s.store)
	case "CompareFaces":
		return handleCompareFaces(ctx, s.store)
	case "CopyProjectVersion":
		return handleCopyProjectVersion(ctx, s.store)
	case "CreateCollection":
		return handleCreateCollection(ctx, s.store)
	case "CreateDataset":
		return handleCreateDataset(ctx, s.store)
	case "CreateFaceLivenessSession":
		return handleCreateFaceLivenessSession(ctx, s.store)
	case "CreateProject":
		return handleCreateProject(ctx, s.store)
	case "CreateProjectVersion":
		return handleCreateProjectVersion(ctx, s.store)
	case "CreateStreamProcessor":
		return handleCreateStreamProcessor(ctx, s.store)
	case "CreateUser":
		return handleCreateUser(ctx, s.store)
	case "DeleteCollection":
		return handleDeleteCollection(ctx, s.store)
	case "DeleteDataset":
		return handleDeleteDataset(ctx, s.store)
	case "DeleteFaces":
		return handleDeleteFaces(ctx, s.store)
	case "DeleteProject":
		return handleDeleteProject(ctx, s.store)
	case "DeleteProjectPolicy":
		return handleDeleteProjectPolicy(ctx, s.store)
	case "DeleteProjectVersion":
		return handleDeleteProjectVersion(ctx, s.store)
	case "DeleteStreamProcessor":
		return handleDeleteStreamProcessor(ctx, s.store)
	case "DeleteUser":
		return handleDeleteUser(ctx, s.store)
	case "DescribeCollection":
		return handleDescribeCollection(ctx, s.store)
	case "DescribeDataset":
		return handleDescribeDataset(ctx, s.store)
	case "DescribeProjectVersions":
		return handleDescribeProjectVersions(ctx, s.store)
	case "DescribeProjects":
		return handleDescribeProjects(ctx, s.store)
	case "DescribeStreamProcessor":
		return handleDescribeStreamProcessor(ctx, s.store)
	case "DetectCustomLabels":
		return handleDetectCustomLabels(ctx, s.store)
	case "DetectFaces":
		return handleDetectFaces(ctx, s.store)
	case "DetectLabels":
		return handleDetectLabels(ctx, s.store)
	case "DetectModerationLabels":
		return handleDetectModerationLabels(ctx, s.store)
	case "DetectProtectiveEquipment":
		return handleDetectProtectiveEquipment(ctx, s.store)
	case "DetectText":
		return handleDetectText(ctx, s.store)
	case "DisassociateFaces":
		return handleDisassociateFaces(ctx, s.store)
	case "DistributeDatasetEntries":
		return handleDistributeDatasetEntries(ctx, s.store)
	case "GetCelebrityInfo":
		return handleGetCelebrityInfo(ctx, s.store)
	case "GetCelebrityRecognition":
		return handleGetCelebrityRecognition(ctx, s.store)
	case "GetContentModeration":
		return handleGetContentModeration(ctx, s.store)
	case "GetFaceDetection":
		return handleGetFaceDetection(ctx, s.store)
	case "GetFaceLivenessSessionResults":
		return handleGetFaceLivenessSessionResults(ctx, s.store)
	case "GetFaceSearch":
		return handleGetFaceSearch(ctx, s.store)
	case "GetLabelDetection":
		return handleGetLabelDetection(ctx, s.store)
	case "GetMediaAnalysisJob":
		return handleGetMediaAnalysisJob(ctx, s.store)
	case "GetPersonTracking":
		return handleGetPersonTracking(ctx, s.store)
	case "GetSegmentDetection":
		return handleGetSegmentDetection(ctx, s.store)
	case "GetTextDetection":
		return handleGetTextDetection(ctx, s.store)
	case "IndexFaces":
		return handleIndexFaces(ctx, s.store)
	case "ListCollections":
		return handleListCollections(ctx, s.store)
	case "ListDatasetEntries":
		return handleListDatasetEntries(ctx, s.store)
	case "ListDatasetLabels":
		return handleListDatasetLabels(ctx, s.store)
	case "ListFaces":
		return handleListFaces(ctx, s.store)
	case "ListMediaAnalysisJobs":
		return handleListMediaAnalysisJobs(ctx, s.store)
	case "ListProjectPolicies":
		return handleListProjectPolicies(ctx, s.store)
	case "ListStreamProcessors":
		return handleListStreamProcessors(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListUsers":
		return handleListUsers(ctx, s.store)
	case "PutProjectPolicy":
		return handlePutProjectPolicy(ctx, s.store)
	case "RecognizeCelebrities":
		return handleRecognizeCelebrities(ctx, s.store)
	case "SearchFaces":
		return handleSearchFaces(ctx, s.store)
	case "SearchFacesByImage":
		return handleSearchFacesByImage(ctx, s.store)
	case "SearchUsers":
		return handleSearchUsers(ctx, s.store)
	case "SearchUsersByImage":
		return handleSearchUsersByImage(ctx, s.store)
	case "StartCelebrityRecognition":
		return handleStartCelebrityRecognition(ctx, s.store)
	case "StartContentModeration":
		return handleStartContentModeration(ctx, s.store)
	case "StartFaceDetection":
		return handleStartFaceDetection(ctx, s.store)
	case "StartFaceSearch":
		return handleStartFaceSearch(ctx, s.store)
	case "StartLabelDetection":
		return handleStartLabelDetection(ctx, s.store)
	case "StartMediaAnalysisJob":
		return handleStartMediaAnalysisJob(ctx, s.store)
	case "StartPersonTracking":
		return handleStartPersonTracking(ctx, s.store)
	case "StartProjectVersion":
		return handleStartProjectVersion(ctx, s.store)
	case "StartSegmentDetection":
		return handleStartSegmentDetection(ctx, s.store)
	case "StartStreamProcessor":
		return handleStartStreamProcessor(ctx, s.store)
	case "StartTextDetection":
		return handleStartTextDetection(ctx, s.store)
	case "StopProjectVersion":
		return handleStopProjectVersion(ctx, s.store)
	case "StopStreamProcessor":
		return handleStopStreamProcessor(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateDatasetEntries":
		return handleUpdateDatasetEntries(ctx, s.store)
	case "UpdateStreamProcessor":
		return handleUpdateStreamProcessor(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
