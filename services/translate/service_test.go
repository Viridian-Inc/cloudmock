package translate_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/translate"
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
	req.Header.Set("X-Amz-Target", "AWSShineFrontendService_20170701."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/translate/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestCreateParallelData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateParallelData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateParallelData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteParallelData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteParallelData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteParallelData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTerminology(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTerminology", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTerminology: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTextTranslationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTextTranslationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTextTranslationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetParallelData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetParallelData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetParallelData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTerminology(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetTerminology", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTerminology: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestImportTerminology(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ImportTerminology", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ImportTerminology: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListLanguages(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListLanguages", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListLanguages: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListParallelData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListParallelData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListParallelData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListTerminologies(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTerminologies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTerminologies: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTextTranslationJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTextTranslationJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTextTranslationJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartTextTranslationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartTextTranslationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartTextTranslationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopTextTranslationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopTextTranslationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopTextTranslationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestTranslateDocument(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TranslateDocument", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TranslateDocument: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTranslateText(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TranslateText", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TranslateText: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateParallelData(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateParallelData", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateParallelData: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

