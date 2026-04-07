package rekognition_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/rekognition"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "RekognitionService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/rekognition/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestAssociateFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCompareFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CompareFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CompareFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCopyProjectVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CopyProjectVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CopyProjectVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCollection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCollection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCollection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateDataset(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDataset", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDataset: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFaceLivenessSession(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFaceLivenessSession", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFaceLivenessSession: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateProject(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateProject", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateProject: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateProjectVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateProjectVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateProjectVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCollection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCollection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCollection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDataset(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDataset", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDataset: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProject(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProject", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProject: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProjectPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProjectPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProjectPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProjectVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProjectVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProjectVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCollection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCollection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCollection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataset(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataset", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataset: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProjectVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProjectVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProjectVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProjects(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProjects", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProjects: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectCustomLabels(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectCustomLabels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectCustomLabels: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectLabels(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectLabels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectLabels: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectModerationLabels(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectModerationLabels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectModerationLabels: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectProtectiveEquipment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectProtectiveEquipment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectProtectiveEquipment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectText(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectText", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectText: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDistributeDatasetEntries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DistributeDatasetEntries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DistributeDatasetEntries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCelebrityInfo(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCelebrityInfo", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCelebrityInfo: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCelebrityRecognition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCelebrityRecognition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCelebrityRecognition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetContentModeration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetContentModeration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetContentModeration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFaceDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFaceDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFaceDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFaceLivenessSessionResults(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFaceLivenessSessionResults", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFaceLivenessSessionResults: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFaceSearch(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFaceSearch", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFaceSearch: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetLabelDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetLabelDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetLabelDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMediaAnalysisJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMediaAnalysisJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMediaAnalysisJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetPersonTracking(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetPersonTracking", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetPersonTracking: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSegmentDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSegmentDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSegmentDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTextDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetTextDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTextDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestIndexFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "IndexFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("IndexFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCollections(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCollections", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCollections: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDatasetEntries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDatasetEntries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDatasetEntries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDatasetLabels(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDatasetLabels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDatasetLabels: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListMediaAnalysisJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListMediaAnalysisJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListMediaAnalysisJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListProjectPolicies(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListProjectPolicies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListProjectPolicies: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListStreamProcessors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListStreamProcessors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListStreamProcessors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListUsers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListUsers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUsers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutProjectPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutProjectPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutProjectPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRecognizeCelebrities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RecognizeCelebrities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RecognizeCelebrities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchFaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchFaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchFaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchFacesByImage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchFacesByImage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchFacesByImage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchUsers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchUsers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchUsers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchUsersByImage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchUsersByImage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchUsersByImage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartCelebrityRecognition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartCelebrityRecognition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartCelebrityRecognition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartContentModeration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartContentModeration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartContentModeration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartFaceDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartFaceDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartFaceDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartFaceSearch(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartFaceSearch", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartFaceSearch: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartLabelDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartLabelDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartLabelDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartMediaAnalysisJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartMediaAnalysisJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartMediaAnalysisJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartPersonTracking(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartPersonTracking", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartPersonTracking: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartProjectVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartProjectVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartProjectVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartSegmentDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartSegmentDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartSegmentDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartTextDetection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartTextDetection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartTextDetection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopProjectVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopProjectVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopProjectVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUntagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UntagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDatasetEntries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDatasetEntries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDatasetEntries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateStreamProcessor(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateStreamProcessor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateStreamProcessor: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

