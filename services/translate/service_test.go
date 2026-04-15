package translate_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/translate"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body == nil {
		data = []byte("{}")
	} else {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSShineFrontendService_20170701."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/translate/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func TestTranslateText(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "TranslateText", map[string]any{
		"Text":               "hello",
		"SourceLanguageCode": "en",
		"TargetLanguageCode": "es",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("TranslateText: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		TranslatedText     string
		SourceLanguageCode string
		TargetLanguageCode string
	}
	decodeBody(t, w, &out)
	if out.TranslatedText != "hello" {
		t.Fatalf("unexpected text: %s", out.TranslatedText)
	}
	if out.TargetLanguageCode != "es" {
		t.Fatalf("unexpected target: %s", out.TargetLanguageCode)
	}
}

func TestTranslateTextValidates(t *testing.T) {
	h := newGateway(t)
	if w := doCall(t, h, "TranslateText", map[string]any{"Text": "hi"}); w.Code != http.StatusBadRequest {
		t.Fatalf("missing source: want 400, got %d", w.Code)
	}
	if w := doCall(t, h, "TranslateText", map[string]any{"Text": "hi", "SourceLanguageCode": "en"}); w.Code != http.StatusBadRequest {
		t.Fatalf("missing target: want 400, got %d", w.Code)
	}
}

func TestTerminologyLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "ImportTerminology", map[string]any{
		"Name":          "MyTerms",
		"MergeStrategy": "OVERWRITE",
		"Description":   "Custom terms",
		"TerminologyData": map[string]any{
			"Format":              "CSV",
			"Directionality":      "UNI",
			"SourceLanguageCode":  "en",
			"TargetLanguageCodes": []string{"es"},
			"File":                "en,es\nhello,hola\n",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("ImportTerminology: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var imported struct {
		TerminologyProperties struct {
			Name      string
			TermCount int
		}
	}
	decodeBody(t, w, &imported)
	if imported.TerminologyProperties.Name != "MyTerms" {
		t.Fatalf("unexpected name: %s", imported.TerminologyProperties.Name)
	}
	if imported.TerminologyProperties.TermCount != 2 {
		t.Fatalf("expected 2 terms, got %d", imported.TerminologyProperties.TermCount)
	}

	w = doCall(t, h, "ListTerminologies", nil)
	var listed struct {
		TerminologyPropertiesList []struct {
			Name string
		}
	}
	decodeBody(t, w, &listed)
	if len(listed.TerminologyPropertiesList) != 1 {
		t.Fatalf("expected 1, got %d", len(listed.TerminologyPropertiesList))
	}

	w = doCall(t, h, "GetTerminology", map[string]any{"Name": "MyTerms"})
	if w.Code != http.StatusOK {
		t.Fatalf("GetTerminology: want 200, got %d", w.Code)
	}

	if w := doCall(t, h, "DeleteTerminology", map[string]any{"Name": "MyTerms"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteTerminology: want 200, got %d", w.Code)
	}
	if w := doCall(t, h, "GetTerminology", map[string]any{"Name": "MyTerms"}); w.Code != http.StatusNotFound {
		t.Fatalf("GetTerminology after delete: want 404, got %d", w.Code)
	}
}

func TestParallelDataLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateParallelData", map[string]any{
		"Name":        "pd1",
		"Description": "test",
		"ParallelDataConfig": map[string]any{
			"S3Uri":  "s3://foo/bar",
			"Format": "TSV",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("CreateParallelData: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w = doCall(t, h, "CreateParallelData", map[string]any{
		"Name": "pd1",
		"ParallelDataConfig": map[string]any{
			"S3Uri":  "s3://foo/bar",
			"Format": "TSV",
		},
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate CreateParallelData: want 409, got %d", w.Code)
	}

	w = doCall(t, h, "GetParallelData", map[string]any{"Name": "pd1"})
	if w.Code != http.StatusOK {
		t.Fatalf("GetParallelData: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "UpdateParallelData", map[string]any{
		"Name":        "pd1",
		"Description": "updated",
		"ParallelDataConfig": map[string]any{
			"S3Uri":  "s3://foo/bar2",
			"Format": "TSV",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateParallelData: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListParallelData", nil)
	var listed struct {
		ParallelDataPropertiesList []struct {
			Name        string
			Description string
		}
	}
	decodeBody(t, w, &listed)
	if len(listed.ParallelDataPropertiesList) != 1 ||
		listed.ParallelDataPropertiesList[0].Description != "updated" {
		t.Fatalf("unexpected list: %+v", listed)
	}

	if w := doCall(t, h, "DeleteParallelData", map[string]any{"Name": "pd1"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteParallelData: want 200, got %d", w.Code)
	}
}

func TestJobLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "StartTextTranslationJob", map[string]any{
		"JobName":             "myjob",
		"SourceLanguageCode":  "en",
		"TargetLanguageCodes": []string{"es", "fr"},
		"InputDataConfig":     map[string]any{"S3Uri": "s3://input/", "ContentType": "text/plain"},
		"OutputDataConfig":    map[string]any{"S3Uri": "s3://output/"},
		"DataAccessRoleArn":   "arn:aws:iam::000000000000:role/r",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("StartTextTranslationJob: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var started struct {
		JobID     string `json:"JobId"`
		JobStatus string
	}
	decodeBody(t, w, &started)
	if started.JobID == "" || started.JobStatus == "" {
		t.Fatalf("expected job id + status: %+v", started)
	}

	w = doCall(t, h, "DescribeTextTranslationJob", map[string]any{"JobId": started.JobID})
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTextTranslationJob: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListTextTranslationJobs", nil)
	var listed struct {
		TextTranslationJobPropertiesList []struct {
			JobID string `json:"JobId"`
		}
	}
	decodeBody(t, w, &listed)
	if len(listed.TextTranslationJobPropertiesList) != 1 {
		t.Fatalf("expected 1 job, got %d", len(listed.TextTranslationJobPropertiesList))
	}

	w = doCall(t, h, "StopTextTranslationJob", map[string]any{"JobId": started.JobID})
	if w.Code != http.StatusOK {
		t.Fatalf("StopTextTranslationJob: want 200, got %d", w.Code)
	}
}

func TestTagging(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:translate:us-east-1:000000000000:terminology/foo"

	if w := doCall(t, h, "TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags": []map[string]any{
			{"Key": "env", "Value": "dev"},
		},
	}); w.Code != http.StatusOK {
		t.Fatalf("TagResource: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w := doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	var listed struct {
		Tags []struct {
			Key   string
			Value string
		}
	}
	decodeBody(t, w, &listed)
	if len(listed.Tags) != 1 || listed.Tags[0].Value != "dev" {
		t.Fatalf("unexpected tags: %+v", listed.Tags)
	}

	if w := doCall(t, h, "UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"env"},
	}); w.Code != http.StatusOK {
		t.Fatalf("UntagResource: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	decodeBody(t, w, &listed)
	if len(listed.Tags) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(listed.Tags))
	}
}

func TestListLanguages(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "ListLanguages", nil)
	var out struct {
		Languages []struct{ LanguageCode string }
	}
	decodeBody(t, w, &out)
	if len(out.Languages) == 0 {
		t.Fatalf("expected languages")
	}
}

func TestTranslateDocument(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "TranslateDocument", map[string]any{
		"Document": map[string]any{
			"Content":     "hello",
			"ContentType": "text/plain",
		},
		"SourceLanguageCode": "en",
		"TargetLanguageCode": "es",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("TranslateDocument: want 200, got %d: %s", w.Code, w.Body.String())
	}
}
