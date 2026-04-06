package comprehend_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/comprehend"
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
	req.Header.Set("X-Amz-Target", "Comprehend_20171127."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/comprehend/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestBatchDetectDominantLanguage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectDominantLanguage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectDominantLanguage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDetectEntities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectEntities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectEntities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDetectKeyPhrases(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectKeyPhrases", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectKeyPhrases: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDetectSentiment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectSentiment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectSentiment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDetectSyntax(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectSyntax", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectSyntax: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDetectTargetedSentiment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDetectTargetedSentiment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDetectTargetedSentiment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestClassifyDocument(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ClassifyDocument", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ClassifyDocument: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestContainsPiiEntities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ContainsPiiEntities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ContainsPiiEntities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestCreateDocumentClassifier(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDocumentClassifier", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDocumentClassifier: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateEndpoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateEndpoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateEndpoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateEntityRecognizer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateEntityRecognizer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateEntityRecognizer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFlywheel(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFlywheel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFlywheel: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDocumentClassifier(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDocumentClassifier", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDocumentClassifier: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteEndpoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteEndpoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteEndpoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteEntityRecognizer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteEntityRecognizer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteEntityRecognizer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFlywheel(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFlywheel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFlywheel: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteResourcePolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteResourcePolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteResourcePolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDescribeDocumentClassificationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDocumentClassificationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDocumentClassificationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDocumentClassifier(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDocumentClassifier", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDocumentClassifier: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDominantLanguageDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDominantLanguageDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDominantLanguageDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeEndpoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeEndpoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEndpoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeEntityRecognizer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeEntityRecognizer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEntityRecognizer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeEventsDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeEventsDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEventsDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFlywheel(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFlywheel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFlywheel: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFlywheelIteration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFlywheelIteration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFlywheelIteration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeKeyPhrasesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeKeyPhrasesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeKeyPhrasesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribePiiEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribePiiEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribePiiEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeResourcePolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeResourcePolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeResourcePolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTargetedSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTargetedSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTargetedSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTopicsDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTopicsDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTopicsDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectDominantLanguage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectDominantLanguage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectDominantLanguage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectEntities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectEntities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectEntities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectKeyPhrases(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectKeyPhrases", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectKeyPhrases: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectPiiEntities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectPiiEntities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectPiiEntities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectSentiment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectSentiment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectSentiment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectSyntax(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectSyntax", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectSyntax: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectTargetedSentiment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectTargetedSentiment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectTargetedSentiment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDetectToxicContent(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DetectToxicContent", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DetectToxicContent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestImportModel(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ImportModel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ImportModel: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDatasets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDatasets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDatasets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDocumentClassificationJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDocumentClassificationJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDocumentClassificationJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDocumentClassifierSummaries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDocumentClassifierSummaries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDocumentClassifierSummaries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDocumentClassifiers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDocumentClassifiers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDocumentClassifiers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDominantLanguageDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDominantLanguageDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDominantLanguageDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEndpoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEndpoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEndpoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEntitiesDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEntitiesDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEntitiesDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEntityRecognizerSummaries(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEntityRecognizerSummaries", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEntityRecognizerSummaries: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEntityRecognizers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEntityRecognizers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEntityRecognizers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEventsDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEventsDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEventsDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFlywheelIterationHistory(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFlywheelIterationHistory", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFlywheelIterationHistory: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFlywheels(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFlywheels", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFlywheels: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListKeyPhrasesDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListKeyPhrasesDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListKeyPhrasesDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPiiEntitiesDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPiiEntitiesDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPiiEntitiesDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListSentimentDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListSentimentDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSentimentDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListTargetedSentimentDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTargetedSentimentDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTargetedSentimentDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTopicsDetectionJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTopicsDetectionJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTopicsDetectionJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutResourcePolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutResourcePolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutResourcePolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartDocumentClassificationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartDocumentClassificationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartDocumentClassificationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartDominantLanguageDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartDominantLanguageDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartDominantLanguageDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartEventsDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartEventsDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartEventsDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartFlywheelIteration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartFlywheelIteration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartFlywheelIteration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartKeyPhrasesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartKeyPhrasesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartKeyPhrasesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartPiiEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartPiiEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartPiiEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartTargetedSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartTargetedSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartTargetedSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartTopicsDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartTopicsDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartTopicsDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopDominantLanguageDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopDominantLanguageDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopDominantLanguageDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopEventsDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopEventsDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopEventsDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopKeyPhrasesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopKeyPhrasesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopKeyPhrasesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopPiiEntitiesDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopPiiEntitiesDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopPiiEntitiesDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopTargetedSentimentDetectionJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopTargetedSentimentDetectionJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopTargetedSentimentDetectionJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopTrainingDocumentClassifier(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopTrainingDocumentClassifier", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopTrainingDocumentClassifier: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopTrainingEntityRecognizer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopTrainingEntityRecognizer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopTrainingEntityRecognizer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateEndpoint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateEndpoint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateEndpoint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFlywheel(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFlywheel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFlywheel: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

