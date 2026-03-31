package textract

import (
	"encoding/json"
	"net/http"

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

// ---- async job handlers ----

func handleStartDocumentTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	docLoc := getMap(params, "DocumentLocation")
	if docLoc == nil {
		return jsonErr(service.ErrValidation("DocumentLocation is required."))
	}
	j := store.StartJob(JobTextDetection, docLoc, nil, getMap(params, "NotificationChannel"), nil)
	return jsonOK(map[string]any{"JobId": j.JobId})
}

func handleGetDocumentTextDetection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	jobId := getStr(params, "JobId")
	if jobId == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	j, awsErr := store.GetJob(jobId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"JobStatus":        string(j.Status),
		"StatusMessage":    j.StatusMessage,
		"DocumentMetadata": map[string]any{"Pages": 1},
	}
	if j.Status == JobSucceeded {
		blockMaps := make([]map[string]any, 0, len(j.Blocks))
		for _, b := range j.Blocks {
			blockMaps = append(blockMaps, map[string]any{
				"BlockType":  b.BlockType,
				"Id":         b.Id,
				"Text":       b.Text,
				"Confidence": b.Confidence,
				"Page":       b.Page,
			})
		}
		resp["Blocks"] = blockMaps
		resp["DetectDocumentTextModelVersion"] = "1.0"
	}
	return jsonOK(resp)
}

func handleStartDocumentAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	docLoc := getMap(params, "DocumentLocation")
	if docLoc == nil {
		return jsonErr(service.ErrValidation("DocumentLocation is required."))
	}
	features := getStringSlice(params, "FeatureTypes")
	j := store.StartJob(JobDocAnalysis, docLoc, features, getMap(params, "NotificationChannel"), nil)
	return jsonOK(map[string]any{"JobId": j.JobId})
}

func handleGetDocumentAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	jobId := getStr(params, "JobId")
	if jobId == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	j, awsErr := store.GetJob(jobId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"JobStatus":        string(j.Status),
		"StatusMessage":    j.StatusMessage,
		"DocumentMetadata": map[string]any{"Pages": 1},
	}
	if j.Status == JobSucceeded {
		blockMaps := make([]map[string]any, 0, len(j.Blocks))
		for _, b := range j.Blocks {
			blockMaps = append(blockMaps, map[string]any{
				"BlockType":  b.BlockType,
				"Id":         b.Id,
				"Text":       b.Text,
				"Confidence": b.Confidence,
				"Page":       b.Page,
			})
		}
		resp["Blocks"] = blockMaps
		resp["AnalyzeDocumentModelVersion"] = "1.0"
	}
	return jsonOK(resp)
}

func handleStartExpenseAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	docLoc := getMap(params, "DocumentLocation")
	if docLoc == nil {
		return jsonErr(service.ErrValidation("DocumentLocation is required."))
	}
	j := store.StartJob(JobExpenseAnalysis, docLoc, nil, getMap(params, "NotificationChannel"), nil)
	return jsonOK(map[string]any{"JobId": j.JobId})
}

func handleGetExpenseAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	jobId := getStr(params, "JobId")
	if jobId == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	j, awsErr := store.GetJob(jobId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"JobStatus":        string(j.Status),
		"StatusMessage":    j.StatusMessage,
		"DocumentMetadata": map[string]any{"Pages": 1},
	}
	if j.Status == JobSucceeded {
		resp["ExpenseDocuments"] = j.ExpenseDocuments
	}
	return jsonOK(resp)
}

// ---- sync handlers ----

func handleAnalyzeDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	doc := getMap(params, "Document")
	if doc == nil {
		return jsonErr(service.ErrValidation("Document is required."))
	}
	features := getStringSlice(params, "FeatureTypes")
	result := store.AnalyzeDocumentSync(doc, features)
	return jsonOK(result)
}

func handleDetectDocumentText(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	doc := getMap(params, "Document")
	if doc == nil {
		return jsonErr(service.ErrValidation("Document is required."))
	}
	result := store.DetectDocumentTextSync(doc)
	return jsonOK(result)
}

// ---- tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
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
	arn := getStr(params, "ResourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
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
	arn := getStr(params, "ResourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": entriesToTags(tags)})
}
