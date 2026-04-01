package transcribe_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/transcribe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.TranscribeService {
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

// ---- Test 1: StartTranscriptionJob ----

func TestStartTranscriptionJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "job-1",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
		"LanguageCode":         "en-US",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	job := body["TranscriptionJob"].(map[string]any)
	assert.Equal(t, "job-1", job["TranscriptionJobName"])
}

// ---- Test 2: GetTranscriptionJob ----

func TestGetTranscriptionJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "job-get",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{
		"TranscriptionJobName": "job-get",
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	job := body["TranscriptionJob"].(map[string]any)
	assert.Equal(t, "job-get", job["TranscriptionJobName"])
	assert.NotEmpty(t, job["TranscriptionJobStatus"])
}

// ---- Test 3: ListTranscriptionJobs ----

func TestListTranscriptionJobs(t *testing.T) {
	s := newService()
	for _, name := range []string{"tj-a", "tj-b"} {
		_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
			"TranscriptionJobName": name,
			"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
		}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListTranscriptionJobs", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	summaries := body["TranscriptionJobSummaries"].([]any)
	assert.Len(t, summaries, 2)
}

// ---- Test 4: DeleteTranscriptionJob ----

func TestDeleteTranscriptionJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-del",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-del",
	}))
	require.NoError(t, err)

	// Verify removed
	_, err = s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-del",
	}))
	require.Error(t, err)
}

// ---- Test 5: Transcription job lifecycle QUEUED -> IN_PROGRESS -> COMPLETED ----

func TestTranscriptionJobLifecycle(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-lc",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.NoError(t, err)

	// Initially QUEUED or may have transitioned
	resp, err := s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{"TranscriptionJobName": "tj-lc"}))
	require.NoError(t, err)
	job := respBody(t, resp)["TranscriptionJob"].(map[string]any)
	assert.Contains(t, []string{"QUEUED", "IN_PROGRESS", "COMPLETED"}, job["TranscriptionJobStatus"])

	// Wait for completion
	time.Sleep(3 * time.Second)
	resp2, err := s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{"TranscriptionJobName": "tj-lc"}))
	require.NoError(t, err)
	job2 := respBody(t, resp2)["TranscriptionJob"].(map[string]any)
	assert.Equal(t, "COMPLETED", job2["TranscriptionJobStatus"])
	assert.NotNil(t, job2["Transcript"])
}

// ---- Test 6: CreateVocabulary ----

func TestCreateVocabulary(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{
		"VocabularyName": "vocab-1",
		"LanguageCode":   "en-US",
		"Phrases":        []any{"hello", "world"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "vocab-1", body["VocabularyName"])
	assert.Contains(t, []string{"PENDING", "READY"}, body["VocabularyState"])
}

// ---- Test 7: GetVocabulary ----

func TestGetVocabulary(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{
		"VocabularyName": "vocab-get",
		"LanguageCode":   "en-US",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("GetVocabulary", map[string]any{"VocabularyName": "vocab-get"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "vocab-get", body["VocabularyName"])
}

// ---- Test 8: ListVocabularies ----

func TestListVocabularies(t *testing.T) {
	s := newService()
	for _, name := range []string{"v-1", "v-2"} {
		_, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{"VocabularyName": name}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListVocabularies", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	vocabs := body["Vocabularies"].([]any)
	assert.Len(t, vocabs, 2)
}

// ---- Test 9: DeleteVocabulary ----

func TestDeleteVocabulary(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{"VocabularyName": "v-del"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteVocabulary", map[string]any{"VocabularyName": "v-del"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("GetVocabulary", map[string]any{"VocabularyName": "v-del"}))
	require.Error(t, err)
}

// ---- Test 10: UpdateVocabulary ----

func TestUpdateVocabulary(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{
		"VocabularyName": "v-upd",
		"LanguageCode":   "en-US",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("UpdateVocabulary", map[string]any{
		"VocabularyName": "v-upd",
		"Phrases":        []any{"updated", "phrases"},
	}))
	require.NoError(t, err)
	body := respBody(t, resp)
	// After update, state is forced to PENDING but lifecycle may transition quickly to READY
	assert.Contains(t, []string{"PENDING", "READY"}, body["VocabularyState"])
}

// ---- Test 11: NotFound ----

func TestTranscriptionJobNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{
		"TranscriptionJobName": "nonexistent",
	}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "NotFoundException", awsErr.Code)
}

// ---- Test 12: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 13: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-tag",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.NoError(t, err)

	arn := "arn:aws:transcribe:us-east-1:123456789012:transcription-job/tj-tag"

	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        []any{map[string]any{"Key": "env", "Value": "test"}},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["Tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceArn": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["Tags"].([]any)
	assert.Len(t, tags2, 0)
}

// ---- Test 14: Duplicate job ----

func TestDuplicateTranscriptionJob(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-dup",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "tj-dup",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
	}))
	require.Error(t, err)
}

// ---- Test 15: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "transcribe", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Enrichment tests ----

func TestTranscribeJob_TranscriptText(t *testing.T) {
	s := newService()
	// Start a job
	resp, err := s.HandleRequest(jsonCtx("StartTranscriptionJob", map[string]any{
		"TranscriptionJobName": "text-test",
		"LanguageCode":         "en-US",
		"Media":                map[string]any{"MediaFileUri": "s3://bucket/audio.mp3"},
		"OutputBucketName":     "output-bucket",
	}))
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(3 * time.Second)

	getResp, err := s.HandleRequest(jsonCtx("GetTranscriptionJob", map[string]any{
		"TranscriptionJobName": "text-test",
	}))
	require.NoError(t, err)
	m := respBody(t, getResp)
	job := m["TranscriptionJob"].(map[string]any)
	if job["TranscriptionJobStatus"] == "COMPLETED" {
		transcript := job["Transcript"].(map[string]any)
		assert.NotEmpty(t, transcript["TranscriptFileUri"])
		results := transcript["Results"].(map[string]any)
		transcripts := results["transcripts"].([]any)
		assert.Greater(t, len(transcripts), 0)
		assert.Contains(t, transcripts[0].(map[string]any)["transcript"], "mock transcript")
	}
	_ = resp
}

func TestVocabulary_InvalidLanguageCode(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{
		"VocabularyName": "bad-lang",
		"LanguageCode":   "xx-XX",
		"Phrases":        []string{"hello"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestVocabulary_ValidLanguageCode(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateVocabulary", map[string]any{
		"VocabularyName": "good-lang",
		"LanguageCode":   "fr-FR",
		"Phrases":        []string{"bonjour"},
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, "fr-FR", m["LanguageCode"])
}
