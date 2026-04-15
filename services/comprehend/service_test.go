package comprehend_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/comprehend"
)

// ── Test plumbing ───────────────────────────────────────────────────────────

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

func doCall(t *testing.T, h http.Handler, action string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: want 200, got %d\nbody: %s", action, w.Code, w.Body.String())
	}
	var out map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s: decode: %v\nbody: %s", action, err, w.Body.String())
		}
	}
	return w, out
}

func doCallExpectStatus(t *testing.T, h http.Handler, action string, body any, want int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	if w.Code != want {
		t.Fatalf("%s: want %d, got %d\nbody: %s", action, want, w.Code, w.Body.String())
	}
	return w
}

func mustStr(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %+v", key, m)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q not a string: %T", key, v)
	}
	return s
}

func sampleInputDataConfig() map[string]any {
	return map[string]any{
		"S3Uri":      "s3://bucket/input/",
		"DataFormat": "COMPREHEND_CSV",
	}
}

func sampleOutputDataConfig() map[string]any {
	return map[string]any{
		"S3Uri": "s3://bucket/output/",
	}
}

// ── Sync detection handlers ─────────────────────────────────────────────────

func TestDetectSyncOps(t *testing.T) {
	h := newGateway(t)

	cases := []struct {
		action   string
		req      map[string]any
		key      string
		expectMap bool
	}{
		{"DetectDominantLanguage", map[string]any{"Text": "Hello world"}, "Languages", false},
		{"DetectEntities", map[string]any{"Text": "John Smith lives in Seattle.", "LanguageCode": "en"}, "Entities", false},
		{"DetectKeyPhrases", map[string]any{"Text": "the quick brown fox", "LanguageCode": "en"}, "KeyPhrases", false},
		{"DetectPiiEntities", map[string]any{"Text": "alice@example.com", "LanguageCode": "en"}, "Entities", false},
		{"DetectSentiment", map[string]any{"Text": "i love it", "LanguageCode": "en"}, "Sentiment", false},
		{"DetectSyntax", map[string]any{"Text": "Hello world", "LanguageCode": "en"}, "SyntaxTokens", false},
		{"DetectTargetedSentiment", map[string]any{"Text": "the service was great", "LanguageCode": "en"}, "Entities", false},
	}
	for _, c := range cases {
		t.Run(c.action, func(t *testing.T) {
			_, out := doCall(t, h, c.action, c.req)
			if _, ok := out[c.key]; !ok {
				t.Fatalf("%s: missing %s in response: %+v", c.action, c.key, out)
			}
		})
	}
}

func TestDetectToxicContent(t *testing.T) {
	h := newGateway(t)
	_, out := doCall(t, h, "DetectToxicContent", map[string]any{
		"LanguageCode": "en",
		"TextSegments": []map[string]any{{"Text": "hello world"}},
	})
	if _, ok := out["ResultList"]; !ok {
		t.Fatalf("missing ResultList: %+v", out)
	}
}

func TestContainsPiiEntities(t *testing.T) {
	h := newGateway(t)
	_, out := doCall(t, h, "ContainsPiiEntities", map[string]any{
		"Text":         "alice@example.com",
		"LanguageCode": "en",
	})
	if _, ok := out["Labels"]; !ok {
		t.Fatalf("missing Labels: %+v", out)
	}
}

func TestDetectMissingArgs(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "DetectSentiment", map[string]any{"LanguageCode": "en"}, http.StatusBadRequest)
}

// ── Batch detection handlers ────────────────────────────────────────────────

func TestBatchDetectAll(t *testing.T) {
	h := newGateway(t)
	texts := []string{"first text", "second text", "third text"}
	cases := []string{
		"BatchDetectDominantLanguage",
		"BatchDetectEntities",
		"BatchDetectKeyPhrases",
		"BatchDetectSentiment",
		"BatchDetectSyntax",
		"BatchDetectTargetedSentiment",
	}
	for _, action := range cases {
		t.Run(action, func(t *testing.T) {
			_, out := doCall(t, h, action, map[string]any{
				"TextList":     texts,
				"LanguageCode": "en",
			})
			rl, ok := out["ResultList"].([]any)
			if !ok {
				t.Fatalf("%s: missing/invalid ResultList: %+v", action, out)
			}
			if len(rl) != len(texts) {
				t.Fatalf("%s: want %d results, got %d", action, len(texts), len(rl))
			}
		})
	}
}

func TestBatchDetectMissingTexts(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "BatchDetectEntities", map[string]any{"LanguageCode": "en"}, http.StatusBadRequest)
}

// ── Document classifier lifecycle ───────────────────────────────────────────

func TestDocumentClassifierLifecycle(t *testing.T) {
	h := newGateway(t)

	_, out := doCall(t, h, "CreateDocumentClassifier", map[string]any{
		"DocumentClassifierName": "my-classifier",
		"DataAccessRoleArn":      "arn:aws:iam::123456789012:role/comprehend",
		"LanguageCode":           "en",
		"InputDataConfig":        sampleInputDataConfig(),
		"OutputDataConfig":       sampleOutputDataConfig(),
		"Tags": []map[string]any{
			{"Key": "team", "Value": "ml"},
		},
	})
	arn := mustStr(t, out, "DocumentClassifierArn")
	if !strings.Contains(arn, "document-classifier/my-classifier") {
		t.Fatalf("unexpected arn: %s", arn)
	}

	_, out = doCall(t, h, "DescribeDocumentClassifier", map[string]any{
		"DocumentClassifierArn": arn,
	})
	props, ok := out["DocumentClassifierProperties"].(map[string]any)
	if !ok {
		t.Fatalf("missing DocumentClassifierProperties: %+v", out)
	}
	if mustStr(t, props, "Status") != "TRAINED" {
		t.Fatalf("unexpected status: %v", props["Status"])
	}

	_, out = doCall(t, h, "ListDocumentClassifiers", nil)
	if list, ok := out["DocumentClassifierPropertiesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}

	_, out = doCall(t, h, "ListDocumentClassifierSummaries", nil)
	if list, ok := out["DocumentClassifierSummariesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected summaries: %+v", out)
	}

	doCall(t, h, "StopTrainingDocumentClassifier", map[string]any{
		"DocumentClassifierArn": arn,
	})

	doCall(t, h, "DeleteDocumentClassifier", map[string]any{
		"DocumentClassifierArn": arn,
	})

	doCallExpectStatus(t, h, "DescribeDocumentClassifier", map[string]any{
		"DocumentClassifierArn": arn,
	}, http.StatusNotFound)
}

func TestCreateDocumentClassifierValidation(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "CreateDocumentClassifier", map[string]any{}, http.StatusBadRequest)
}

// ── Entity recognizer lifecycle ─────────────────────────────────────────────

func TestEntityRecognizerLifecycle(t *testing.T) {
	h := newGateway(t)

	_, out := doCall(t, h, "CreateEntityRecognizer", map[string]any{
		"RecognizerName":    "my-recognizer",
		"DataAccessRoleArn": "arn:aws:iam::123456789012:role/comprehend",
		"LanguageCode":      "en",
		"InputDataConfig":   sampleInputDataConfig(),
	})
	arn := mustStr(t, out, "EntityRecognizerArn")

	_, out = doCall(t, h, "DescribeEntityRecognizer", map[string]any{
		"EntityRecognizerArn": arn,
	})
	if _, ok := out["EntityRecognizerProperties"]; !ok {
		t.Fatalf("missing EntityRecognizerProperties")
	}

	_, out = doCall(t, h, "ListEntityRecognizers", nil)
	if list, ok := out["EntityRecognizerPropertiesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}

	_, out = doCall(t, h, "ListEntityRecognizerSummaries", nil)
	if list, ok := out["EntityRecognizerSummariesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected summaries: %+v", out)
	}

	doCall(t, h, "StopTrainingEntityRecognizer", map[string]any{
		"EntityRecognizerArn": arn,
	})

	doCall(t, h, "DeleteEntityRecognizer", map[string]any{
		"EntityRecognizerArn": arn,
	})

	doCallExpectStatus(t, h, "DescribeEntityRecognizer", map[string]any{
		"EntityRecognizerArn": arn,
	}, http.StatusNotFound)
}

// ── Endpoint lifecycle ──────────────────────────────────────────────────────

func TestEndpointLifecycle(t *testing.T) {
	h := newGateway(t)

	// First create a model the endpoint can reference.
	_, out := doCall(t, h, "CreateDocumentClassifier", map[string]any{
		"DocumentClassifierName": "ep-classifier",
		"DataAccessRoleArn":      "arn:aws:iam::123456789012:role/comprehend",
		"LanguageCode":           "en",
		"InputDataConfig":        sampleInputDataConfig(),
	})
	modelArn := mustStr(t, out, "DocumentClassifierArn")

	_, out = doCall(t, h, "CreateEndpoint", map[string]any{
		"EndpointName":          "my-endpoint",
		"ModelArn":              modelArn,
		"DesiredInferenceUnits": 2,
		"DataAccessRoleArn":     "arn:aws:iam::123456789012:role/comprehend",
	})
	endpointArn := mustStr(t, out, "EndpointArn")

	_, out = doCall(t, h, "DescribeEndpoint", map[string]any{
		"EndpointArn": endpointArn,
	})
	props, ok := out["EndpointProperties"].(map[string]any)
	if !ok {
		t.Fatalf("missing EndpointProperties: %+v", out)
	}
	if mustStr(t, props, "Status") != "IN_SERVICE" {
		t.Fatalf("unexpected status: %v", props["Status"])
	}

	_, out = doCall(t, h, "ListEndpoints", nil)
	if list, ok := out["EndpointPropertiesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}

	doCall(t, h, "UpdateEndpoint", map[string]any{
		"EndpointArn":           endpointArn,
		"DesiredInferenceUnits": 4,
	})

	// Endpoint guards classifier deletion.
	doCallExpectStatus(t, h, "DeleteDocumentClassifier", map[string]any{
		"DocumentClassifierArn": modelArn,
	}, http.StatusBadRequest)

	doCall(t, h, "DeleteEndpoint", map[string]any{
		"EndpointArn": endpointArn,
	})

	doCallExpectStatus(t, h, "DescribeEndpoint", map[string]any{
		"EndpointArn": endpointArn,
	}, http.StatusNotFound)

	doCall(t, h, "DeleteDocumentClassifier", map[string]any{
		"DocumentClassifierArn": modelArn,
	})
}

func TestCreateEndpointRequiresModel(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "CreateEndpoint", map[string]any{
		"EndpointName":          "x",
		"DesiredInferenceUnits": 1,
	}, http.StatusBadRequest)
}

// ── Flywheel lifecycle ──────────────────────────────────────────────────────

func TestFlywheelLifecycle(t *testing.T) {
	h := newGateway(t)

	_, out := doCall(t, h, "CreateFlywheel", map[string]any{
		"FlywheelName":      "my-flywheel",
		"DataAccessRoleArn": "arn:aws:iam::123456789012:role/comprehend",
		"DataLakeS3Uri":     "s3://bucket/lake",
		"ModelType":         "DOCUMENT_CLASSIFIER",
	})
	flywheelArn := mustStr(t, out, "FlywheelArn")

	_, out = doCall(t, h, "DescribeFlywheel", map[string]any{
		"FlywheelArn": flywheelArn,
	})
	if _, ok := out["FlywheelProperties"]; !ok {
		t.Fatalf("missing FlywheelProperties")
	}

	_, out = doCall(t, h, "ListFlywheels", nil)
	if list, ok := out["FlywheelSummaryList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}

	doCall(t, h, "UpdateFlywheel", map[string]any{
		"FlywheelArn":       flywheelArn,
		"DataAccessRoleArn": "arn:aws:iam::123456789012:role/new",
	})

	// Iteration
	_, out = doCall(t, h, "StartFlywheelIteration", map[string]any{
		"FlywheelArn": flywheelArn,
	})
	iterationID := mustStr(t, out, "FlywheelIterationId")

	_, out = doCall(t, h, "DescribeFlywheelIteration", map[string]any{
		"FlywheelArn":         flywheelArn,
		"FlywheelIterationId": iterationID,
	})
	if _, ok := out["FlywheelIterationProperties"]; !ok {
		t.Fatalf("missing FlywheelIterationProperties")
	}

	_, out = doCall(t, h, "ListFlywheelIterationHistory", map[string]any{
		"FlywheelArn": flywheelArn,
	})
	if list, ok := out["FlywheelIterationPropertiesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected iteration list: %+v", out)
	}

	// Dataset
	_, out = doCall(t, h, "CreateDataset", map[string]any{
		"DatasetName":     "ds1",
		"FlywheelArn":     flywheelArn,
		"DatasetType":     "TRAIN",
		"InputDataConfig": sampleInputDataConfig(),
	})
	dsArn := mustStr(t, out, "DatasetArn")

	_, out = doCall(t, h, "DescribeDataset", map[string]any{
		"DatasetArn": dsArn,
	})
	if _, ok := out["DatasetProperties"]; !ok {
		t.Fatalf("missing DatasetProperties")
	}

	_, out = doCall(t, h, "ListDatasets", map[string]any{
		"FlywheelArn": flywheelArn,
	})
	if list, ok := out["DatasetPropertiesList"].([]any); !ok || len(list) != 1 {
		t.Fatalf("unexpected dataset list: %+v", out)
	}

	doCall(t, h, "DeleteFlywheel", map[string]any{
		"FlywheelArn": flywheelArn,
	})

	doCallExpectStatus(t, h, "DescribeFlywheel", map[string]any{
		"FlywheelArn": flywheelArn,
	}, http.StatusNotFound)
}

// ── Async job lifecycles ────────────────────────────────────────────────────

func TestJobLifecycles(t *testing.T) {
	h := newGateway(t)

	jobs := []struct {
		family       string // base name e.g. "DominantLanguage"
		startAction  string
		describeKey  string
		describeProp string
		listAction   string
		listKey      string
		stopAction   string
	}{
		{
			family:       "DocumentClassification",
			startAction:  "StartDocumentClassificationJob",
			describeKey:  "DescribeDocumentClassificationJob",
			describeProp: "DocumentClassificationJobProperties",
			listAction:   "ListDocumentClassificationJobs",
			listKey:      "DocumentClassificationJobPropertiesList",
		},
		{
			family:       "DominantLanguage",
			startAction:  "StartDominantLanguageDetectionJob",
			describeKey:  "DescribeDominantLanguageDetectionJob",
			describeProp: "DominantLanguageDetectionJobProperties",
			listAction:   "ListDominantLanguageDetectionJobs",
			listKey:      "DominantLanguageDetectionJobPropertiesList",
			stopAction:   "StopDominantLanguageDetectionJob",
		},
		{
			family:       "Entities",
			startAction:  "StartEntitiesDetectionJob",
			describeKey:  "DescribeEntitiesDetectionJob",
			describeProp: "EntitiesDetectionJobProperties",
			listAction:   "ListEntitiesDetectionJobs",
			listKey:      "EntitiesDetectionJobPropertiesList",
			stopAction:   "StopEntitiesDetectionJob",
		},
		{
			family:       "Events",
			startAction:  "StartEventsDetectionJob",
			describeKey:  "DescribeEventsDetectionJob",
			describeProp: "EventsDetectionJobProperties",
			listAction:   "ListEventsDetectionJobs",
			listKey:      "EventsDetectionJobPropertiesList",
			stopAction:   "StopEventsDetectionJob",
		},
		{
			family:       "KeyPhrases",
			startAction:  "StartKeyPhrasesDetectionJob",
			describeKey:  "DescribeKeyPhrasesDetectionJob",
			describeProp: "KeyPhrasesDetectionJobProperties",
			listAction:   "ListKeyPhrasesDetectionJobs",
			listKey:      "KeyPhrasesDetectionJobPropertiesList",
			stopAction:   "StopKeyPhrasesDetectionJob",
		},
		{
			family:       "PiiEntities",
			startAction:  "StartPiiEntitiesDetectionJob",
			describeKey:  "DescribePiiEntitiesDetectionJob",
			describeProp: "PiiEntitiesDetectionJobProperties",
			listAction:   "ListPiiEntitiesDetectionJobs",
			listKey:      "PiiEntitiesDetectionJobPropertiesList",
			stopAction:   "StopPiiEntitiesDetectionJob",
		},
		{
			family:       "Sentiment",
			startAction:  "StartSentimentDetectionJob",
			describeKey:  "DescribeSentimentDetectionJob",
			describeProp: "SentimentDetectionJobProperties",
			listAction:   "ListSentimentDetectionJobs",
			listKey:      "SentimentDetectionJobPropertiesList",
			stopAction:   "StopSentimentDetectionJob",
		},
		{
			family:       "TargetedSentiment",
			startAction:  "StartTargetedSentimentDetectionJob",
			describeKey:  "DescribeTargetedSentimentDetectionJob",
			describeProp: "TargetedSentimentDetectionJobProperties",
			listAction:   "ListTargetedSentimentDetectionJobs",
			listKey:      "TargetedSentimentDetectionJobPropertiesList",
			stopAction:   "StopTargetedSentimentDetectionJob",
		},
		{
			family:       "Topics",
			startAction:  "StartTopicsDetectionJob",
			describeKey:  "DescribeTopicsDetectionJob",
			describeProp: "TopicsDetectionJobProperties",
			listAction:   "ListTopicsDetectionJobs",
			listKey:      "TopicsDetectionJobPropertiesList",
		},
	}

	for _, j := range jobs {
		j := j
		t.Run(j.family, func(t *testing.T) {
			_, out := doCall(t, h, j.startAction, map[string]any{
				"JobName":           "job-" + j.family,
				"InputDataConfig":   sampleInputDataConfig(),
				"OutputDataConfig":  sampleOutputDataConfig(),
				"DataAccessRoleArn": "arn:aws:iam::123456789012:role/comprehend",
				"LanguageCode":      "en",
			})
			jobID := mustStr(t, out, "JobId")
			if mustStr(t, out, "JobStatus") != "COMPLETED" {
				t.Fatalf("%s: unexpected status %v", j.startAction, out["JobStatus"])
			}

			_, out = doCall(t, h, j.describeKey, map[string]any{"JobId": jobID})
			if _, ok := out[j.describeProp]; !ok {
				t.Fatalf("%s: missing %s", j.describeKey, j.describeProp)
			}

			_, out = doCall(t, h, j.listAction, nil)
			if list, ok := out[j.listKey].([]any); !ok || len(list) != 1 {
				t.Fatalf("%s: want 1 job, got %+v", j.listAction, out)
			}

			if j.stopAction != "" {
				_, out = doCall(t, h, j.stopAction, map[string]any{"JobId": jobID})
				if mustStr(t, out, "JobStatus") != "STOP_REQUESTED" {
					t.Fatalf("%s: unexpected stopped status %v", j.stopAction, out["JobStatus"])
				}
			}
		})
	}
}

func TestStartJobValidation(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "StartSentimentDetectionJob", map[string]any{}, http.StatusBadRequest)
}

func TestDescribeJobNotFound(t *testing.T) {
	h := newGateway(t)
	doCallExpectStatus(t, h, "DescribeSentimentDetectionJob", map[string]any{
		"JobId": "missing",
	}, http.StatusNotFound)
}

// ── ImportModel ─────────────────────────────────────────────────────────────

func TestImportModelClassifier(t *testing.T) {
	h := newGateway(t)
	_, out := doCall(t, h, "ImportModel", map[string]any{
		"SourceModelArn":    "arn:aws:comprehend:us-east-1:123456789012:document-classifier/source",
		"ModelName":         "imported",
		"DataAccessRoleArn": "arn:aws:iam::123456789012:role/comprehend",
	})
	arn := mustStr(t, out, "ModelArn")
	if !strings.Contains(arn, "document-classifier/imported") {
		t.Fatalf("unexpected imported arn: %s", arn)
	}
	doCall(t, h, "DescribeDocumentClassifier", map[string]any{
		"DocumentClassifierArn": arn,
	})
}

func TestImportModelRecognizer(t *testing.T) {
	h := newGateway(t)
	_, out := doCall(t, h, "ImportModel", map[string]any{
		"SourceModelArn": "arn:aws:comprehend:us-east-1:123456789012:entity-recognizer/source",
		"ModelName":      "imported-er",
	})
	arn := mustStr(t, out, "ModelArn")
	if !strings.Contains(arn, "entity-recognizer/imported-er") {
		t.Fatalf("unexpected imported arn: %s", arn)
	}
	doCall(t, h, "DescribeEntityRecognizer", map[string]any{
		"EntityRecognizerArn": arn,
	})
}

// ── Resource policy lifecycle ───────────────────────────────────────────────

func TestResourcePolicyLifecycle(t *testing.T) {
	h := newGateway(t)

	// Need an endpoint to attach a policy to.
	_, out := doCall(t, h, "CreateDocumentClassifier", map[string]any{
		"DocumentClassifierName": "rp-classifier",
		"DataAccessRoleArn":      "arn:aws:iam::123456789012:role/comprehend",
		"LanguageCode":           "en",
		"InputDataConfig":        sampleInputDataConfig(),
	})
	modelArn := mustStr(t, out, "DocumentClassifierArn")

	_, out = doCall(t, h, "CreateEndpoint", map[string]any{
		"EndpointName":          "rp-endpoint",
		"ModelArn":              modelArn,
		"DesiredInferenceUnits": 1,
	})
	endpointArn := mustStr(t, out, "EndpointArn")

	_, out = doCall(t, h, "PutResourcePolicy", map[string]any{
		"ResourceArn":    endpointArn,
		"ResourcePolicy": `{"Version":"2012-10-17","Statement":[]}`,
	})
	revID := mustStr(t, out, "PolicyRevisionId")
	if revID == "" {
		t.Fatalf("missing PolicyRevisionId")
	}

	_, out = doCall(t, h, "DescribeResourcePolicy", map[string]any{
		"ResourceArn": endpointArn,
	})
	if mustStr(t, out, "ResourcePolicy") == "" {
		t.Fatalf("missing ResourcePolicy text")
	}

	doCall(t, h, "DeleteResourcePolicy", map[string]any{
		"ResourceArn": endpointArn,
	})

	doCallExpectStatus(t, h, "DescribeResourcePolicy", map[string]any{
		"ResourceArn": endpointArn,
	}, http.StatusNotFound)
}

// ── Tagging ─────────────────────────────────────────────────────────────────

func TestTaggingLifecycle(t *testing.T) {
	h := newGateway(t)

	_, out := doCall(t, h, "CreateDocumentClassifier", map[string]any{
		"DocumentClassifierName": "tag-classifier",
		"DataAccessRoleArn":      "arn:aws:iam::123456789012:role/comprehend",
		"LanguageCode":           "en",
		"InputDataConfig":        sampleInputDataConfig(),
	})
	arn := mustStr(t, out, "DocumentClassifierArn")

	doCall(t, h, "TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags": []map[string]any{
			{"Key": "team", "Value": "ml"},
			{"Key": "env", "Value": "prod"},
		},
	})

	_, out = doCall(t, h, "ListTagsForResource", map[string]any{
		"ResourceArn": arn,
	})
	tags, ok := out["Tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("want 2 tags, got %+v", out)
	}

	doCall(t, h, "UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"env"},
	})

	_, out = doCall(t, h, "ListTagsForResource", map[string]any{
		"ResourceArn": arn,
	})
	tags, _ = out["Tags"].([]any)
	if len(tags) != 1 {
		t.Fatalf("want 1 tag after untag, got %+v", out)
	}
}

// ── Reset ───────────────────────────────────────────────────────────────────

func TestStoreReset(t *testing.T) {
	store := svc.NewStore("123456789012", "us-east-1")
	if _, err := store.CreateDocumentClassifier(&svc.StoredDocumentClassifier{
		Name:              "c1",
		DataAccessRoleArn: "role",
		LanguageCode:      "en",
		InputDataConfig:   sampleInputDataConfig(),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := store.ListDocumentClassifiers(); len(got) != 1 {
		t.Fatalf("want 1 classifier, got %d", len(got))
	}
	store.Reset()
	if got := store.ListDocumentClassifiers(); len(got) != 0 {
		t.Fatalf("want 0 classifiers after reset, got %d", len(got))
	}
}

// ── Smoke check that all 85 actions are wired and reachable ────────────────

func TestAllActionsRouted(t *testing.T) {
	h := newGateway(t)
	// We just need to verify the action dispatch table covers each action
	// without returning InvalidAction. Sending an empty body should yield
	// either 200 (for actions with no required fields) or a 4xx
	// ValidationError — but never InvalidAction.
	actions := []string{
		"BatchDetectDominantLanguage", "BatchDetectEntities", "BatchDetectKeyPhrases",
		"BatchDetectSentiment", "BatchDetectSyntax", "BatchDetectTargetedSentiment",
		"ClassifyDocument", "ContainsPiiEntities", "CreateDataset",
		"CreateDocumentClassifier", "CreateEndpoint", "CreateEntityRecognizer",
		"CreateFlywheel", "DeleteDocumentClassifier", "DeleteEndpoint",
		"DeleteEntityRecognizer", "DeleteFlywheel", "DeleteResourcePolicy",
		"DescribeDataset", "DescribeDocumentClassificationJob",
		"DescribeDocumentClassifier", "DescribeDominantLanguageDetectionJob",
		"DescribeEndpoint", "DescribeEntitiesDetectionJob",
		"DescribeEntityRecognizer", "DescribeEventsDetectionJob",
		"DescribeFlywheel", "DescribeFlywheelIteration",
		"DescribeKeyPhrasesDetectionJob", "DescribePiiEntitiesDetectionJob",
		"DescribeResourcePolicy", "DescribeSentimentDetectionJob",
		"DescribeTargetedSentimentDetectionJob", "DescribeTopicsDetectionJob",
		"DetectDominantLanguage", "DetectEntities", "DetectKeyPhrases",
		"DetectPiiEntities", "DetectSentiment", "DetectSyntax",
		"DetectTargetedSentiment", "DetectToxicContent", "ImportModel",
		"ListDatasets", "ListDocumentClassificationJobs",
		"ListDocumentClassifierSummaries", "ListDocumentClassifiers",
		"ListDominantLanguageDetectionJobs", "ListEndpoints",
		"ListEntitiesDetectionJobs", "ListEntityRecognizerSummaries",
		"ListEntityRecognizers", "ListEventsDetectionJobs",
		"ListFlywheelIterationHistory", "ListFlywheels",
		"ListKeyPhrasesDetectionJobs", "ListPiiEntitiesDetectionJobs",
		"ListSentimentDetectionJobs", "ListTagsForResource",
		"ListTargetedSentimentDetectionJobs", "ListTopicsDetectionJobs",
		"PutResourcePolicy", "StartDocumentClassificationJob",
		"StartDominantLanguageDetectionJob", "StartEntitiesDetectionJob",
		"StartEventsDetectionJob", "StartFlywheelIteration",
		"StartKeyPhrasesDetectionJob", "StartPiiEntitiesDetectionJob",
		"StartSentimentDetectionJob", "StartTargetedSentimentDetectionJob",
		"StartTopicsDetectionJob", "StopDominantLanguageDetectionJob",
		"StopEntitiesDetectionJob", "StopEventsDetectionJob",
		"StopKeyPhrasesDetectionJob", "StopPiiEntitiesDetectionJob",
		"StopSentimentDetectionJob", "StopTargetedSentimentDetectionJob",
		"StopTrainingDocumentClassifier", "StopTrainingEntityRecognizer",
		"TagResource", "UntagResource", "UpdateEndpoint", "UpdateFlywheel",
	}
	if len(actions) != 85 {
		t.Fatalf("expected 85 actions, got %d", len(actions))
	}
	for _, a := range actions {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, svcReq(t, a, map[string]any{}))
		if w.Code == http.StatusOK {
			continue
		}
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		// Tolerate validation/not-found errors for actions that need real
		// arguments. What we won't tolerate is "InvalidAction" which means
		// the action is unrouted.
		if code, ok := body["__type"].(string); ok && code == "InvalidAction" {
			t.Errorf("action %s returned InvalidAction", a)
		}
	}
}
