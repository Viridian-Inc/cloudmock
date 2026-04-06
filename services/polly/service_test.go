package polly_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/polly"
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
	req.Header.Set("X-Amz-Target", "polly."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/polly/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestDeleteLexicon(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteLexicon", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteLexicon: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeVoices(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeVoices", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeVoices: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetLexicon(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetLexicon", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetLexicon: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSpeechSynthesisTask(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSpeechSynthesisTask", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSpeechSynthesisTask: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListLexicons(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListLexicons", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListLexicons: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListSpeechSynthesisTasks(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListSpeechSynthesisTasks", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSpeechSynthesisTasks: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutLexicon(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutLexicon", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutLexicon: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartSpeechSynthesisStream(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartSpeechSynthesisStream", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartSpeechSynthesisStream: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartSpeechSynthesisTask(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartSpeechSynthesisTask", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartSpeechSynthesisTask: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSynthesizeSpeech(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SynthesizeSpeech", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SynthesizeSpeech: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

