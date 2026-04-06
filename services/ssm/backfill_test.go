package ssm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- SSM Documents ----

func TestSSM_DocumentLifecycle(t *testing.T) {
	handler := newSSMGateway(t)

	// CreateDocument
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "CreateDocument", map[string]any{
		"Name":           "my-doc",
		"Content":        `{"schemaVersion":"2.2","mainSteps":[]}`,
		"DocumentType":   "Command",
		"DocumentFormat": "JSON",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDocument: %d %s", w.Code, w.Body.String())
	}

	// DescribeDocument
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "DescribeDocument", map[string]string{"Name": "my-doc"}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDocument: %d %s", w.Code, w.Body.String())
	}

	// ListDocuments
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "ListDocuments", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDocuments: %d %s", w.Code, w.Body.String())
	}

	// GetDocument
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "GetDocument", map[string]string{"Name": "my-doc"}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDocument: %d %s", w.Code, w.Body.String())
	}

	// DeleteDocument
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "DeleteDocument", map[string]string{"Name": "my-doc"}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDocument: %d %s", w.Code, w.Body.String())
	}
}

// ---- SSM Automation ----

func TestSSM_AutomationExecution(t *testing.T) {
	handler := newSSMGateway(t)

	// Create a document first
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "CreateDocument", map[string]any{
		"Name":         "auto-doc",
		"Content":      `{"schemaVersion":"0.3","mainSteps":[]}`,
		"DocumentType": "Automation",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDocument: %d %s", w.Code, w.Body.String())
	}

	// StartAutomationExecution
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "StartAutomationExecution", map[string]any{
		"DocumentName": "auto-doc",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("StartAutomationExecution: %d %s", w.Code, w.Body.String())
	}

	// DescribeAutomationExecutions
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "DescribeAutomationExecutions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAutomationExecutions: %d %s", w.Code, w.Body.String())
	}
}
