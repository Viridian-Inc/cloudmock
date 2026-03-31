package transcribe

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func unixFloat(t time.Time) float64 {
	return float64(t.Unix())
}

func unixFloatPtr(t *time.Time) *float64 {
	if t == nil {
		return nil
	}
	v := float64(t.Unix())
	return &v
}

type tagEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func tagsFromEntries(entries []tagEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m
}

func entriesToTags(m map[string]string) []tagEntry {
	entries := make([]tagEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, tagEntry{Key: k, Value: v})
	}
	return entries
}

func getStr(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMap(params map[string]any, key string) map[string]any {
	if v, ok := params[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func getInt(params map[string]any, key string) int {
	if v, ok := params[key]; ok {
		if n, ok := v.(float64); ok {
			return int(n)
		}
	}
	return 0
}

func getStringSlice(params map[string]any, key string) []string {
	if v, ok := params[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

func parseTags(params map[string]any) []tagEntry {
	v, ok := params["Tags"]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	entries := make([]tagEntry, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, tagEntry{
			Key:   getStr(m, "Key"),
			Value: getStr(m, "Value"),
		})
	}
	return entries
}

// ---- Transcription job handlers ----

func handleStartTranscriptionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TranscriptionJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TranscriptionJobName is required."))
	}
	j, awsErr := store.StartTranscriptionJob(
		name,
		getStr(params, "LanguageCode"),
		getStr(params, "MediaFormat"),
		getInt(params, "MediaSampleRateHertz"),
		getMap(params, "Media"),
		getStr(params, "OutputBucketName"),
		getStr(params, "OutputKey"),
		getMap(params, "Settings"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TranscriptionJob": transcriptionJobResponse(j),
	})
}

func handleGetTranscriptionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TranscriptionJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TranscriptionJobName is required."))
	}
	j, awsErr := store.GetTranscriptionJob(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TranscriptionJob": transcriptionJobResponse(j),
	})
}

func handleListTranscriptionJobs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListTranscriptionJobs()
	summaries := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		entry := map[string]any{
			"TranscriptionJobName":   j.TranscriptionJobName,
			"TranscriptionJobStatus": string(j.TranscriptionJobStatus),
			"LanguageCode":           j.LanguageCode,
			"CreationTime":           unixFloat(j.CreationTime),
		}
		if j.StartTime != nil {
			entry["StartTime"] = unixFloat(*j.StartTime)
		}
		if j.CompletionTime != nil {
			entry["CompletionTime"] = unixFloat(*j.CompletionTime)
		}
		summaries = append(summaries, entry)
	}
	return jsonOK(map[string]any{
		"TranscriptionJobSummaries": summaries,
		"Status":                    "COMPLETED",
	})
}

func handleDeleteTranscriptionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TranscriptionJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TranscriptionJobName is required."))
	}
	if awsErr := store.DeleteTranscriptionJob(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func transcriptionJobResponse(j *TranscriptionJob) map[string]any {
	resp := map[string]any{
		"TranscriptionJobName":   j.TranscriptionJobName,
		"TranscriptionJobStatus": string(j.TranscriptionJobStatus),
		"LanguageCode":           j.LanguageCode,
		"MediaFormat":            j.MediaFormat,
		"Media":                  j.Media,
		"CreationTime":           unixFloat(j.CreationTime),
	}
	if j.MediaSampleRateHertz > 0 {
		resp["MediaSampleRateHertz"] = j.MediaSampleRateHertz
	}
	if j.OutputBucketName != "" {
		resp["OutputBucketName"] = j.OutputBucketName
	}
	if j.Settings != nil {
		resp["Settings"] = j.Settings
	}
	if j.StartTime != nil {
		resp["StartTime"] = unixFloat(*j.StartTime)
	}
	if j.CompletionTime != nil {
		resp["CompletionTime"] = unixFloat(*j.CompletionTime)
	}
	if j.TranscriptionJobStatus == TranscriptionCompleted {
		resp["Transcript"] = j.Transcript
	}
	if j.FailureReason != "" {
		resp["FailureReason"] = j.FailureReason
	}
	return resp
}

// ---- Vocabulary handlers ----

func handleCreateVocabulary(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "VocabularyName")
	if name == "" {
		return jsonErr(service.ErrValidation("VocabularyName is required."))
	}
	v, awsErr := store.CreateVocabulary(
		name,
		getStr(params, "LanguageCode"),
		getStringSlice(params, "Phrases"),
		getStr(params, "VocabularyFileUri"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"VocabularyName":  v.VocabularyName,
		"LanguageCode":    v.LanguageCode,
		"VocabularyState": string(v.VocabularyState),
		"LastModifiedTime": unixFloat(v.LastModifiedTime),
	})
}

func handleGetVocabulary(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "VocabularyName")
	if name == "" {
		return jsonErr(service.ErrValidation("VocabularyName is required."))
	}
	v, awsErr := store.GetVocabulary(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"VocabularyName":   v.VocabularyName,
		"LanguageCode":     v.LanguageCode,
		"VocabularyState":  string(v.VocabularyState),
		"LastModifiedTime": unixFloat(v.LastModifiedTime),
	}
	if v.FailureReason != "" {
		resp["FailureReason"] = v.FailureReason
	}
	return jsonOK(resp)
}

func handleListVocabularies(_ *service.RequestContext, store *Store) (*service.Response, error) {
	vocabs := store.ListVocabularies()
	entries := make([]map[string]any, 0, len(vocabs))
	for _, v := range vocabs {
		entries = append(entries, map[string]any{
			"VocabularyName":   v.VocabularyName,
			"LanguageCode":     v.LanguageCode,
			"VocabularyState":  string(v.VocabularyState),
			"LastModifiedTime": unixFloat(v.LastModifiedTime),
		})
	}
	return jsonOK(map[string]any{
		"Vocabularies": entries,
		"Status":       "COMPLETED",
	})
}

func handleDeleteVocabulary(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "VocabularyName")
	if name == "" {
		return jsonErr(service.ErrValidation("VocabularyName is required."))
	}
	if awsErr := store.DeleteVocabulary(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateVocabulary(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "VocabularyName")
	if name == "" {
		return jsonErr(service.ErrValidation("VocabularyName is required."))
	}
	v, awsErr := store.UpdateVocabulary(
		name,
		getStr(params, "LanguageCode"),
		getStringSlice(params, "Phrases"),
		getStr(params, "VocabularyFileUri"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"VocabularyName":   v.VocabularyName,
		"LanguageCode":     v.LanguageCode,
		"VocabularyState":  string(v.VocabularyState),
		"LastModifiedTime": unixFloat(v.LastModifiedTime),
	})
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := tagsFromEntries(parseTags(params))
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tagKeys := getStringSlice(params, "TagKeys")
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": entriesToTags(tags)})
}
