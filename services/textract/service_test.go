package textract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/textract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.TextractService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// ---- Test 1: StartDocumentTextDetection ----

func TestStartDocumentTextDetection(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartDocumentTextDetection", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "my-bucket", "Name": "doc.pdf"},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["JobId"])
}

// ---- Test 2: GetDocumentTextDetection ----

func TestGetDocumentTextDetection(t *testing.T) {
	s := newService()
	startResp, err := s.HandleRequest(jsonCtx("StartDocumentTextDetection", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "doc.pdf"},
		},
	}))
	require.NoError(t, err)
	jobId := respBody(t, startResp)["JobId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetDocumentTextDetection", map[string]any{"JobId": jobId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["JobStatus"])
}

// ---- Test 3: StartDocumentAnalysis ----

func TestStartDocumentAnalysis(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartDocumentAnalysis", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "doc.pdf"},
		},
		"FeatureTypes": []any{"TABLES", "FORMS"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["JobId"])
}

// ---- Test 4: GetDocumentAnalysis ----

func TestGetDocumentAnalysis(t *testing.T) {
	s := newService()
	startResp, err := s.HandleRequest(jsonCtx("StartDocumentAnalysis", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "doc.pdf"},
		},
		"FeatureTypes": []any{"TABLES"},
	}))
	require.NoError(t, err)
	jobId := respBody(t, startResp)["JobId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetDocumentAnalysis", map[string]any{"JobId": jobId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, []string{"IN_PROGRESS", "SUCCEEDED"}, body["JobStatus"])
}

// ---- Test 5: StartExpenseAnalysis ----

func TestStartExpenseAnalysis(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartExpenseAnalysis", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "receipt.jpg"},
		},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["JobId"])
}

// ---- Test 6: GetExpenseAnalysis after completion ----

func TestGetExpenseAnalysisCompleted(t *testing.T) {
	s := newService()
	startResp, err := s.HandleRequest(jsonCtx("StartExpenseAnalysis", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "receipt.jpg"},
		},
	}))
	require.NoError(t, err)
	jobId := respBody(t, startResp)["JobId"].(string)

	time.Sleep(2 * time.Second)

	resp, err := s.HandleRequest(jsonCtx("GetExpenseAnalysis", map[string]any{"JobId": jobId}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "SUCCEEDED", body["JobStatus"])
	assert.NotNil(t, body["ExpenseDocuments"])
}

// ---- Test 7: Job lifecycle IN_PROGRESS -> SUCCEEDED ----

func TestJobLifecycle(t *testing.T) {
	s := newService()
	startResp, err := s.HandleRequest(jsonCtx("StartDocumentTextDetection", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "doc.pdf"},
		},
	}))
	require.NoError(t, err)
	jobId := respBody(t, startResp)["JobId"].(string)

	// Initially IN_PROGRESS or may have already completed
	getResp, err := s.HandleRequest(jsonCtx("GetDocumentTextDetection", map[string]any{"JobId": jobId}))
	require.NoError(t, err)
	assert.Contains(t, []string{"IN_PROGRESS", "SUCCEEDED"}, respBody(t, getResp)["JobStatus"])

	// Wait for completion
	time.Sleep(2 * time.Second)
	getResp2, err := s.HandleRequest(jsonCtx("GetDocumentTextDetection", map[string]any{"JobId": jobId}))
	require.NoError(t, err)
	body := respBody(t, getResp2)
	assert.Equal(t, "SUCCEEDED", body["JobStatus"])
	assert.NotNil(t, body["Blocks"])
}

// ---- Test 8: AnalyzeDocument (sync) ----

func TestAnalyzeDocument(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("AnalyzeDocument", map[string]any{
		"Document": map[string]any{
			"Bytes": "base64data",
		},
		"FeatureTypes": []any{"TABLES"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotNil(t, body["Blocks"])
	assert.NotNil(t, body["DocumentMetadata"])
}

// ---- Test 9: DetectDocumentText (sync) ----

func TestDetectDocumentText(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DetectDocumentText", map[string]any{
		"Document": map[string]any{
			"Bytes": "base64data",
		},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotNil(t, body["Blocks"])
	assert.NotNil(t, body["DocumentMetadata"])
}

// ---- Test 10: Job NotFound ----

func TestJobNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetDocumentTextDetection", map[string]any{
		"JobId": "nonexistent-job-id",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidJobIdException", awsErr.Code)
}

// ---- Test 11: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 12: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	// Start a job to get an ARN
	startResp, err := s.HandleRequest(jsonCtx("StartDocumentTextDetection", map[string]any{
		"DocumentLocation": map[string]any{
			"S3Object": map[string]any{"Bucket": "bucket", "Name": "doc.pdf"},
		},
	}))
	require.NoError(t, err)
	jobId := respBody(t, startResp)["JobId"].(string)
	arn := "arn:aws:textract:us-east-1:123456789012:job/" + jobId

	// TagResource
	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags": []any{
			map[string]any{"Key": "env", "Value": "prod"},
			map[string]any{"Key": "team", "Value": "ml"},
		},
	}))
	require.NoError(t, err)

	// ListTagsForResource
	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["Tags"].([]any)
	assert.Len(t, tags, 2)

	// UntagResource
	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["Tags"].([]any)
	assert.Len(t, tags2, 1)
}

// ---- Test 13: Validation - missing DocumentLocation ----

func TestValidationMissingDocumentLocation(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartDocumentTextDetection", map[string]any{}))
	require.Error(t, err)
}

// ---- Test 14: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "textract", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Enrichment tests ----

func TestAnalyzeDocument_BoundingBoxes(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("AnalyzeDocument", map[string]any{
		"Document":     map[string]any{"Bytes": "dGVzdA=="},
		"FeatureTypes": []string{"TABLES", "FORMS"},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	blocks := m["Blocks"].([]any)
	assert.Greater(t, len(blocks), 3) // PAGE + LINE + WORD + KEY_VALUE_SET + TABLE + CELL

	// Verify bounding box data on first block.
	first := blocks[0].(map[string]any)
	geom := first["Geometry"].(map[string]any)
	assert.NotNil(t, geom["BoundingBox"])
	assert.NotNil(t, geom["Polygon"])
	bb := geom["BoundingBox"].(map[string]any)
	assert.NotNil(t, bb["Top"])
	assert.NotNil(t, bb["Left"])
}

func TestAnalyzeDocument_KeyValuePairs(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("AnalyzeDocument", map[string]any{
		"Document":     map[string]any{"Bytes": "dGVzdA=="},
		"FeatureTypes": []string{"FORMS"},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	blocks := m["Blocks"].([]any)

	// Find KEY_VALUE_SET blocks.
	var kvBlocks []map[string]any
	for _, b := range blocks {
		bm := b.(map[string]any)
		if bm["BlockType"] == "KEY_VALUE_SET" {
			kvBlocks = append(kvBlocks, bm)
		}
	}
	assert.GreaterOrEqual(t, len(kvBlocks), 2) // At least one KEY and one VALUE
}

func TestAnalyzeDocument_TableBlocks(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("AnalyzeDocument", map[string]any{
		"Document":     map[string]any{"Bytes": "dGVzdA=="},
		"FeatureTypes": []string{"TABLES"},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	blocks := m["Blocks"].([]any)

	var tableFound, cellFound bool
	for _, b := range blocks {
		bm := b.(map[string]any)
		if bm["BlockType"] == "TABLE" {
			tableFound = true
		}
		if bm["BlockType"] == "CELL" {
			cellFound = true
			assert.NotNil(t, bm["RowIndex"])
			assert.NotNil(t, bm["ColumnIndex"])
		}
	}
	assert.True(t, tableFound)
	assert.True(t, cellFound)
}
