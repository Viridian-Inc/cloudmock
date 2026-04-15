package comprehend

import (
	"net/http"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ─────────────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidRequestException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

// parseTagList accepts AWS-style [{Key, Value}] tag arrays.
func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		if k == "" {
			k = getStr(t, "key")
		}
		v := getStr(t, "Value")
		if v == "" {
			v = getStr(t, "value")
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// extractNameFromArn pulls the trailing name component out of a
// classifier/recognizer/endpoint/flywheel ARN.
func extractNameFromArn(arn string) string {
	if arn == "" {
		return ""
	}
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}

// ── Mapping helpers ─────────────────────────────────────────────────────────

func tagsToList(m map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for k, v := range m {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

func classifierProps(c *StoredDocumentClassifier) map[string]any {
	out := map[string]any{
		"DocumentClassifierArn": c.Arn,
		"Status":                c.Status,
		"SubmitTime":            rfc3339(c.SubmitTime),
		"LanguageCode":          c.LanguageCode,
		"DataAccessRoleArn":     c.DataAccessRoleArn,
		"InputDataConfig":       c.InputDataConfig,
		"OutputDataConfig":      c.OutputDataConfig,
		"ClassifierMetadata":    c.ClassifierMetadata,
	}
	if !c.EndTime.IsZero() {
		out["EndTime"] = rfc3339(c.EndTime)
	}
	if !c.TrainingStartTime.IsZero() {
		out["TrainingStartTime"] = rfc3339(c.TrainingStartTime)
	}
	if !c.TrainingEndTime.IsZero() {
		out["TrainingEndTime"] = rfc3339(c.TrainingEndTime)
	}
	if c.VersionName != "" {
		out["VersionName"] = c.VersionName
	}
	if c.Mode != "" {
		out["Mode"] = c.Mode
	}
	if c.VolumeKmsKeyId != "" {
		out["VolumeKmsKeyId"] = c.VolumeKmsKeyId
	}
	if c.ModelKmsKeyId != "" {
		out["ModelKmsKeyId"] = c.ModelKmsKeyId
	}
	if c.VpcConfig != nil {
		out["VpcConfig"] = c.VpcConfig
	}
	if c.SourceModelArn != "" {
		out["SourceModelArn"] = c.SourceModelArn
	}
	if c.FlywheelArn != "" {
		out["FlywheelArn"] = c.FlywheelArn
	}
	if c.Message != "" {
		out["Message"] = c.Message
	}
	return out
}

func recognizerProps(r *StoredEntityRecognizer) map[string]any {
	out := map[string]any{
		"EntityRecognizerArn": r.Arn,
		"Status":              r.Status,
		"SubmitTime":          rfc3339(r.SubmitTime),
		"LanguageCode":        r.LanguageCode,
		"DataAccessRoleArn":   r.DataAccessRoleArn,
		"InputDataConfig":     r.InputDataConfig,
		"RecognizerMetadata":  r.RecognizerMetadata,
	}
	if !r.EndTime.IsZero() {
		out["EndTime"] = rfc3339(r.EndTime)
	}
	if !r.TrainingStartTime.IsZero() {
		out["TrainingStartTime"] = rfc3339(r.TrainingStartTime)
	}
	if !r.TrainingEndTime.IsZero() {
		out["TrainingEndTime"] = rfc3339(r.TrainingEndTime)
	}
	if r.VersionName != "" {
		out["VersionName"] = r.VersionName
	}
	if r.VolumeKmsKeyId != "" {
		out["VolumeKmsKeyId"] = r.VolumeKmsKeyId
	}
	if r.ModelKmsKeyId != "" {
		out["ModelKmsKeyId"] = r.ModelKmsKeyId
	}
	if r.VpcConfig != nil {
		out["VpcConfig"] = r.VpcConfig
	}
	if r.SourceModelArn != "" {
		out["SourceModelArn"] = r.SourceModelArn
	}
	if r.FlywheelArn != "" {
		out["FlywheelArn"] = r.FlywheelArn
	}
	if r.Message != "" {
		out["Message"] = r.Message
	}
	return out
}

func endpointProps(e *StoredEndpoint) map[string]any {
	out := map[string]any{
		"EndpointArn":           e.Arn,
		"Status":                e.Status,
		"ModelArn":              e.ModelArn,
		"DesiredModelArn":       e.DesiredModelArn,
		"DesiredInferenceUnits": e.DesiredInferenceUnits,
		"CurrentInferenceUnits": e.CurrentInferenceUnits,
		"CreationTime":          rfc3339(e.CreationTime),
		"LastModifiedTime":      rfc3339(e.LastModifiedTime),
	}
	if e.DataAccessRoleArn != "" {
		out["DataAccessRoleArn"] = e.DataAccessRoleArn
	}
	if e.DesiredDataAccessRoleArn != "" {
		out["DesiredDataAccessRoleArn"] = e.DesiredDataAccessRoleArn
	}
	if e.FlywheelArn != "" {
		out["FlywheelArn"] = e.FlywheelArn
	}
	if e.Message != "" {
		out["Message"] = e.Message
	}
	return out
}

func flywheelProps(f *StoredFlywheel) map[string]any {
	out := map[string]any{
		"FlywheelArn":             f.Arn,
		"Status":                  f.Status,
		"DataAccessRoleArn":       f.DataAccessRoleArn,
		"DataLakeS3Uri":           f.DataLakeS3Uri,
		"CreationTime":            rfc3339(f.CreationTime),
		"LastModifiedTime":        rfc3339(f.LastModifiedTime),
		"LatestFlywheelIteration": f.LatestFlywheelIteration,
	}
	if f.ActiveModelArn != "" {
		out["ActiveModelArn"] = f.ActiveModelArn
	}
	if f.TaskConfig != nil {
		out["TaskConfig"] = f.TaskConfig
	}
	if f.DataSecurityConfig != nil {
		out["DataSecurityConfig"] = f.DataSecurityConfig
	}
	if f.ModelType != "" {
		out["ModelType"] = f.ModelType
	}
	if f.Message != "" {
		out["Message"] = f.Message
	}
	return out
}

func flywheelIterationProps(it *StoredFlywheelIteration) map[string]any {
	out := map[string]any{
		"FlywheelArn":                it.FlywheelArn,
		"WorkflowArn":                it.WorkflowArn,
		"FlywheelIterationId":        it.IterationId,
		"CreationTime":               rfc3339(it.CreationTime),
		"EndTime":                    rfc3339(it.EndTime),
		"Status":                     it.Status,
		"EvaluatedModelArn":          it.EvaluatedModelArn,
		"EvaluatedModelMetrics":      it.EvaluatedModelMetrics,
		"TrainedModelArn":            it.TrainedModelArn,
		"TrainedModelMetrics":        it.TrainedModelMetrics,
		"EvaluationManifestS3Prefix": it.EvaluationManifestS3Prefix,
	}
	if it.Message != "" {
		out["Message"] = it.Message
	}
	return out
}

func datasetProps(d *StoredFlywheelDataset) map[string]any {
	out := map[string]any{
		"DatasetArn":        d.Arn,
		"DatasetName":       d.Name,
		"DatasetType":       d.Type,
		"Status":            d.Status,
		"NumberOfDocuments": d.NumberOfDocuments,
		"CreationTime":      rfc3339(d.CreationTime),
		"EndTime":           rfc3339(d.EndTime),
	}
	if d.Description != "" {
		out["Description"] = d.Description
	}
	if d.Message != "" {
		out["Message"] = d.Message
	}
	return out
}

func jobProps(j *StoredJob) map[string]any {
	out := map[string]any{
		"JobId":             j.JobId,
		"JobArn":            j.JobArn,
		"JobName":           j.JobName,
		"JobStatus":         j.JobStatus,
		"SubmitTime":        rfc3339(j.SubmitTime),
		"InputDataConfig":   j.InputDataConfig,
		"OutputDataConfig":  j.OutputDataConfig,
		"DataAccessRoleArn": j.DataAccessRoleArn,
	}
	if !j.EndTime.IsZero() {
		out["EndTime"] = rfc3339(j.EndTime)
	}
	if j.Message != "" {
		out["Message"] = j.Message
	}
	if j.LanguageCode != "" {
		out["LanguageCode"] = j.LanguageCode
	}
	if j.VolumeKmsKeyId != "" {
		out["VolumeKmsKeyId"] = j.VolumeKmsKeyId
	}
	if j.VpcConfig != nil {
		out["VpcConfig"] = j.VpcConfig
	}
	if j.FlywheelArn != "" {
		out["FlywheelArn"] = j.FlywheelArn
	}
	if j.NumberOfTopics > 0 {
		out["NumberOfTopics"] = j.NumberOfTopics
	}
	if len(j.TargetEventTypes) > 0 {
		out["TargetEventTypes"] = j.TargetEventTypes
	}
	if j.Mode != "" {
		out["Mode"] = j.Mode
	}
	if j.RedactionConfig != nil {
		out["RedactionConfig"] = j.RedactionConfig
	}
	if j.EntityRecognizerArn != "" {
		out["EntityRecognizerArn"] = j.EntityRecognizerArn
	}
	if j.DocumentClassifierArn != "" {
		out["DocumentClassifierArn"] = j.DocumentClassifierArn
	}
	return out
}

// ── Sync detection canned responses ─────────────────────────────────────────

func cannedDominantLanguages() []map[string]any {
	return []map[string]any{
		{"LanguageCode": "en", "Score": 0.99},
	}
}

func cannedEntities() []map[string]any {
	return []map[string]any{
		{"Score": 0.95, "Type": "PERSON", "Text": "John Smith", "BeginOffset": 0, "EndOffset": 10},
		{"Score": 0.92, "Type": "LOCATION", "Text": "Seattle", "BeginOffset": 20, "EndOffset": 27},
	}
}

func cannedKeyPhrases() []map[string]any {
	return []map[string]any{
		{"Score": 0.99, "Text": "first phrase", "BeginOffset": 0, "EndOffset": 12},
		{"Score": 0.95, "Text": "second phrase", "BeginOffset": 14, "EndOffset": 27},
	}
}

func cannedSentiment() (string, map[string]any) {
	return "NEUTRAL", map[string]any{
		"Positive": 0.20,
		"Negative": 0.10,
		"Neutral":  0.60,
		"Mixed":    0.10,
	}
}

func cannedSyntaxTokens() []map[string]any {
	return []map[string]any{
		{"TokenId": 1, "Text": "Hello", "BeginOffset": 0, "EndOffset": 5,
			"PartOfSpeech": map[string]any{"Tag": "INTJ", "Score": 0.99}},
		{"TokenId": 2, "Text": "world", "BeginOffset": 6, "EndOffset": 11,
			"PartOfSpeech": map[string]any{"Tag": "NOUN", "Score": 0.99}},
	}
}

func cannedPiiEntities() []map[string]any {
	return []map[string]any{
		{"Score": 0.99, "Type": "EMAIL", "BeginOffset": 0, "EndOffset": 10},
	}
}

func cannedTargetedSentiment() []map[string]any {
	return []map[string]any{
		{
			"DescriptiveMentionIndex": []int{0},
			"Mentions": []map[string]any{
				{
					"Score":          0.95,
					"GroupScore":     0.95,
					"Text":           "service",
					"Type":           "OTHER",
					"MentionSentiment": map[string]any{
						"Sentiment": "POSITIVE",
						"SentimentScore": map[string]any{
							"Positive": 0.92, "Negative": 0.02, "Neutral": 0.05, "Mixed": 0.01,
						},
					},
					"BeginOffset": 0,
					"EndOffset":   7,
				},
			},
		},
	}
}

func cannedToxicLabels() []map[string]any {
	return []map[string]any{
		{
			"Labels": []map[string]any{
				{"Name": "PROFANITY", "Score": 0.05},
				{"Name": "HATE_SPEECH", "Score": 0.02},
			},
			"Toxicity": 0.05,
		},
	}
}

// ── Sync detection handlers ─────────────────────────────────────────────────

func handleDetectDominantLanguage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{"Languages": cannedDominantLanguages()})
}

func handleDetectEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" && len(getStrList(req, "Bytes")) == 0 {
		// Bytes can also be a base64 string; we tolerate either being absent in tests
		if _, ok := req["Bytes"]; !ok {
			return jsonErr(service.ErrValidation("Text or Bytes is required."))
		}
	}
	return jsonOK(map[string]any{"Entities": cannedEntities()})
}

func handleDetectKeyPhrases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{"KeyPhrases": cannedKeyPhrases()})
}

func handleDetectPiiEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{"Entities": cannedPiiEntities()})
}

func handleDetectSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	sent, score := cannedSentiment()
	return jsonOK(map[string]any{
		"Sentiment":      sent,
		"SentimentScore": score,
	})
}

func handleDetectSyntax(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{"SyntaxTokens": cannedSyntaxTokens()})
}

func handleDetectTargetedSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{"Entities": cannedTargetedSentiment()})
}

func handleDetectToxicContent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getMapList(req, "TextSegments")) == 0 {
		return jsonErr(service.ErrValidation("TextSegments is required."))
	}
	return jsonOK(map[string]any{"ResultList": cannedToxicLabels()})
}

func handleClassifyDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "EndpointArn") == "" {
		return jsonErr(service.ErrValidation("EndpointArn is required."))
	}
	return jsonOK(map[string]any{
		"Classes": []map[string]any{
			{"Name": "POSITIVE", "Score": 0.92},
			{"Name": "NEGATIVE", "Score": 0.08},
		},
		"Labels": []map[string]any{},
	})
}

func handleContainsPiiEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "Text") == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	return jsonOK(map[string]any{
		"Labels": []map[string]any{
			{"Name": "EMAIL", "Score": 0.95},
		},
	})
}

// ── Batch detection handlers ────────────────────────────────────────────────

func batchResults(req map[string]any, build func(i int) map[string]any) map[string]any {
	texts := getStrList(req, "TextList")
	results := make([]map[string]any, 0, len(texts))
	for i := range texts {
		entry := build(i)
		entry["Index"] = i
		results = append(results, entry)
	}
	return map[string]any{
		"ResultList": results,
		"ErrorList":  []any{},
	}
}

func handleBatchDetectDominantLanguage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"Languages": cannedDominantLanguages()}
	}))
}

func handleBatchDetectEntities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"Entities": cannedEntities()}
	}))
}

func handleBatchDetectKeyPhrases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"KeyPhrases": cannedKeyPhrases()}
	}))
}

func handleBatchDetectSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	sent, score := cannedSentiment()
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"Sentiment": sent, "SentimentScore": score}
	}))
}

func handleBatchDetectSyntax(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"SyntaxTokens": cannedSyntaxTokens()}
	}))
}

func handleBatchDetectTargetedSentiment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(getStrList(req, "TextList")) == 0 {
		return jsonErr(service.ErrValidation("TextList is required."))
	}
	return jsonOK(batchResults(req, func(i int) map[string]any {
		return map[string]any{"Entities": cannedTargetedSentiment()}
	}))
}

// ── Document classifiers ────────────────────────────────────────────────────

func handleCreateDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "DocumentClassifierName")
	if name == "" {
		return jsonErr(service.ErrValidation("DocumentClassifierName is required."))
	}
	if getStr(req, "DataAccessRoleArn") == "" {
		return jsonErr(service.ErrValidation("DataAccessRoleArn is required."))
	}
	if getStr(req, "LanguageCode") == "" {
		return jsonErr(service.ErrValidation("LanguageCode is required."))
	}
	if getMap(req, "InputDataConfig") == nil {
		return jsonErr(service.ErrValidation("InputDataConfig is required."))
	}
	c := &StoredDocumentClassifier{
		Name:              name,
		VersionName:       getStr(req, "VersionName"),
		LanguageCode:      getStr(req, "LanguageCode"),
		DataAccessRoleArn: getStr(req, "DataAccessRoleArn"),
		InputDataConfig:   getMap(req, "InputDataConfig"),
		OutputDataConfig:  getMap(req, "OutputDataConfig"),
		VolumeKmsKeyId:    getStr(req, "VolumeKmsKeyId"),
		ModelKmsKeyId:     getStr(req, "ModelKmsKeyId"),
		VpcConfig:         getMap(req, "VpcConfig"),
		Mode:              getStr(req, "Mode"),
		Tags:              parseTagList(req, "Tags"),
	}
	saved, err := store.CreateDocumentClassifier(c)
	if err != nil {
		return jsonErr(err)
	}
	if len(saved.Tags) > 0 {
		store.TagResource(saved.Arn, saved.Tags)
	}
	return jsonOK(map[string]any{"DocumentClassifierArn": saved.Arn})
}

func handleDescribeDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DocumentClassifierArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DocumentClassifierArn is required."))
	}
	c, err := store.GetDocumentClassifier(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"DocumentClassifierProperties": classifierProps(c)})
}

func handleDeleteDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DocumentClassifierArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DocumentClassifierArn is required."))
	}
	if err := store.DeleteDocumentClassifier(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListDocumentClassifiers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListDocumentClassifiers()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, classifierProps(c))
	}
	return jsonOK(map[string]any{"DocumentClassifierPropertiesList": out})
}

func handleListDocumentClassifierSummaries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// Group classifiers by name to produce summaries.
	list := store.ListDocumentClassifiers()
	byName := make(map[string]*StoredDocumentClassifier)
	counts := make(map[string]int)
	for _, c := range list {
		counts[c.Name]++
		if existing, ok := byName[c.Name]; !ok || c.SubmitTime.After(existing.SubmitTime) {
			byName[c.Name] = c
		}
	}
	out := make([]map[string]any, 0, len(byName))
	for name, c := range byName {
		out = append(out, map[string]any{
			"DocumentClassifierName": name,
			"NumberOfVersions":       counts[name],
			"LatestVersionCreatedAt": rfc3339(c.SubmitTime),
			"LatestVersionName":      c.VersionName,
			"LatestVersionStatus":    c.Status,
		})
	}
	return jsonOK(map[string]any{"DocumentClassifierSummariesList": out})
}

func handleStopTrainingDocumentClassifier(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DocumentClassifierArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DocumentClassifierArn is required."))
	}
	if err := store.StopTrainingDocumentClassifier(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Entity recognizers ──────────────────────────────────────────────────────

func handleCreateEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "RecognizerName")
	if name == "" {
		return jsonErr(service.ErrValidation("RecognizerName is required."))
	}
	if getStr(req, "DataAccessRoleArn") == "" {
		return jsonErr(service.ErrValidation("DataAccessRoleArn is required."))
	}
	if getStr(req, "LanguageCode") == "" {
		return jsonErr(service.ErrValidation("LanguageCode is required."))
	}
	if getMap(req, "InputDataConfig") == nil {
		return jsonErr(service.ErrValidation("InputDataConfig is required."))
	}
	r := &StoredEntityRecognizer{
		Name:              name,
		VersionName:       getStr(req, "VersionName"),
		LanguageCode:      getStr(req, "LanguageCode"),
		DataAccessRoleArn: getStr(req, "DataAccessRoleArn"),
		InputDataConfig:   getMap(req, "InputDataConfig"),
		VolumeKmsKeyId:    getStr(req, "VolumeKmsKeyId"),
		ModelKmsKeyId:     getStr(req, "ModelKmsKeyId"),
		VpcConfig:         getMap(req, "VpcConfig"),
		Tags:              parseTagList(req, "Tags"),
	}
	saved, err := store.CreateEntityRecognizer(r)
	if err != nil {
		return jsonErr(err)
	}
	if len(saved.Tags) > 0 {
		store.TagResource(saved.Arn, saved.Tags)
	}
	return jsonOK(map[string]any{"EntityRecognizerArn": saved.Arn})
}

func handleDescribeEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EntityRecognizerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EntityRecognizerArn is required."))
	}
	r, err := store.GetEntityRecognizer(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EntityRecognizerProperties": recognizerProps(r)})
}

func handleDeleteEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EntityRecognizerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EntityRecognizerArn is required."))
	}
	if err := store.DeleteEntityRecognizer(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListEntityRecognizers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListEntityRecognizers()
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, recognizerProps(r))
	}
	return jsonOK(map[string]any{"EntityRecognizerPropertiesList": out})
}

func handleListEntityRecognizerSummaries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListEntityRecognizers()
	byName := make(map[string]*StoredEntityRecognizer)
	counts := make(map[string]int)
	for _, r := range list {
		counts[r.Name]++
		if existing, ok := byName[r.Name]; !ok || r.SubmitTime.After(existing.SubmitTime) {
			byName[r.Name] = r
		}
	}
	out := make([]map[string]any, 0, len(byName))
	for name, r := range byName {
		out = append(out, map[string]any{
			"RecognizerName":         name,
			"NumberOfVersions":       counts[name],
			"LatestVersionCreatedAt": rfc3339(r.SubmitTime),
			"LatestVersionName":      r.VersionName,
			"LatestVersionStatus":    r.Status,
		})
	}
	return jsonOK(map[string]any{"EntityRecognizerSummariesList": out})
}

func handleStopTrainingEntityRecognizer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EntityRecognizerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EntityRecognizerArn is required."))
	}
	if err := store.StopTrainingEntityRecognizer(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Endpoints ───────────────────────────────────────────────────────────────

func handleCreateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "EndpointName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}
	desired := getInt(req, "DesiredInferenceUnits")
	if desired <= 0 {
		return jsonErr(service.ErrValidation("DesiredInferenceUnits must be positive."))
	}
	modelArn := getStr(req, "ModelArn")
	flywheelArn := getStr(req, "FlywheelArn")
	if modelArn == "" && flywheelArn == "" {
		return jsonErr(service.ErrValidation("Either ModelArn or FlywheelArn is required."))
	}
	e := &StoredEndpoint{
		Name:                  name,
		ModelArn:              modelArn,
		DesiredModelArn:       modelArn,
		DesiredInferenceUnits: desired,
		DataAccessRoleArn:     getStr(req, "DataAccessRoleArn"),
		FlywheelArn:           flywheelArn,
		Tags:                  parseTagList(req, "Tags"),
	}
	saved, err := store.CreateEndpoint(e)
	if err != nil {
		return jsonErr(err)
	}
	if len(saved.Tags) > 0 {
		store.TagResource(saved.Arn, saved.Tags)
	}
	return jsonOK(map[string]any{
		"EndpointArn": saved.Arn,
		"ModelArn":    saved.ModelArn,
	})
}

func handleDescribeEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointArn is required."))
	}
	e, err := store.GetEndpoint(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointProperties": endpointProps(e)})
}

func handleDeleteEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointArn is required."))
	}
	if err := store.DeleteEndpoint(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListEndpoints()
	out := make([]map[string]any, 0, len(list))
	for _, e := range list {
		out = append(out, endpointProps(e))
	}
	return jsonOK(map[string]any{"EndpointPropertiesList": out})
}

func handleUpdateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointArn is required."))
	}
	e, err := store.UpdateEndpoint(
		arn,
		getStr(req, "DesiredModelArn"),
		getStr(req, "DesiredDataAccessRoleArn"),
		getStr(req, "FlywheelArn"),
		getInt(req, "DesiredInferenceUnits"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"DesiredModelArn": e.DesiredModelArn})
}

// ── Flywheels ───────────────────────────────────────────────────────────────

func handleCreateFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "FlywheelName")
	if name == "" {
		return jsonErr(service.ErrValidation("FlywheelName is required."))
	}
	if getStr(req, "DataAccessRoleArn") == "" {
		return jsonErr(service.ErrValidation("DataAccessRoleArn is required."))
	}
	if getStr(req, "DataLakeS3Uri") == "" {
		return jsonErr(service.ErrValidation("DataLakeS3Uri is required."))
	}
	f := &StoredFlywheel{
		Name:               name,
		ActiveModelArn:     getStr(req, "ActiveModelArn"),
		DataAccessRoleArn:  getStr(req, "DataAccessRoleArn"),
		TaskConfig:         getMap(req, "TaskConfig"),
		DataLakeS3Uri:      getStr(req, "DataLakeS3Uri"),
		DataSecurityConfig: getMap(req, "DataSecurityConfig"),
		ModelType:          getStr(req, "ModelType"),
		Tags:               parseTagList(req, "Tags"),
	}
	saved, err := store.CreateFlywheel(f)
	if err != nil {
		return jsonErr(err)
	}
	if len(saved.Tags) > 0 {
		store.TagResource(saved.Arn, saved.Tags)
	}
	return jsonOK(map[string]any{
		"FlywheelArn":    saved.Arn,
		"ActiveModelArn": saved.ActiveModelArn,
	})
}

func handleDescribeFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	f, err := store.GetFlywheel(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"FlywheelProperties": flywheelProps(f)})
}

func handleDeleteFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	if err := store.DeleteFlywheel(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListFlywheels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListFlywheels()
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, map[string]any{
			"FlywheelArn":      f.Arn,
			"ActiveModelArn":   f.ActiveModelArn,
			"DataLakeS3Uri":    f.DataLakeS3Uri,
			"Status":           f.Status,
			"ModelType":        f.ModelType,
			"Message":          f.Message,
			"CreationTime":     rfc3339(f.CreationTime),
			"LastModifiedTime": rfc3339(f.LastModifiedTime),
		})
	}
	return jsonOK(map[string]any{"FlywheelSummaryList": out})
}

func handleUpdateFlywheel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	f, err := store.UpdateFlywheel(
		arn,
		getStr(req, "ActiveModelArn"),
		getStr(req, "DataAccessRoleArn"),
		getMap(req, "DataSecurityConfig"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"FlywheelProperties": flywheelProps(f)})
}

func handleStartFlywheelIteration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	it, err := store.StartFlywheelIteration(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"FlywheelArn":         arn,
		"FlywheelIterationId": it.IterationId,
	})
}

func handleDescribeFlywheelIteration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	id := getStr(req, "FlywheelIterationId")
	if arn == "" || id == "" {
		return jsonErr(service.ErrValidation("FlywheelArn and FlywheelIterationId are required."))
	}
	it, err := store.GetFlywheelIteration(arn, id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"FlywheelIterationProperties": flywheelIterationProps(it)})
}

func handleListFlywheelIterationHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "FlywheelArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	list, err := store.ListFlywheelIterations(arn)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(list))
	for _, it := range list {
		out = append(out, flywheelIterationProps(it))
	}
	return jsonOK(map[string]any{"FlywheelIterationPropertiesList": out})
}

// ── Datasets ────────────────────────────────────────────────────────────────

func handleCreateDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "DatasetName")
	if name == "" {
		return jsonErr(service.ErrValidation("DatasetName is required."))
	}
	flywheelArn := getStr(req, "FlywheelArn")
	if flywheelArn == "" {
		return jsonErr(service.ErrValidation("FlywheelArn is required."))
	}
	d := &StoredFlywheelDataset{
		Name:            name,
		FlywheelArn:     flywheelArn,
		Type:            getStr(req, "DatasetType"),
		Description:     getStr(req, "Description"),
		InputDataConfig: getMap(req, "InputDataConfig"),
	}
	saved, err := store.CreateDataset(d)
	if err != nil {
		return jsonErr(err)
	}
	if tags := parseTagList(req, "Tags"); len(tags) > 0 {
		store.TagResource(saved.Arn, tags)
	}
	return jsonOK(map[string]any{"DatasetArn": saved.Arn})
}

func handleDescribeDataset(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "DatasetArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("DatasetArn is required."))
	}
	d, err := store.GetDataset(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"DatasetProperties": datasetProps(d)})
}

func handleListDatasets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListDatasets(getStr(req, "FlywheelArn"))
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, datasetProps(d))
	}
	return jsonOK(map[string]any{"DatasetPropertiesList": out})
}

// ── Async job kinds ─────────────────────────────────────────────────────────

const (
	jobKindDocClassification     = "document-classification-job"
	jobKindDominantLanguage      = "dominant-language-detection-job"
	jobKindEntities              = "entities-detection-job"
	jobKindEvents                = "events-detection-job"
	jobKindKeyPhrases            = "key-phrases-detection-job"
	jobKindPiiEntities           = "pii-entities-detection-job"
	jobKindSentiment             = "sentiment-detection-job"
	jobKindTargetedSentiment     = "targeted-sentiment-detection-job"
	jobKindTopics                = "topics-detection-job"
)

// startJobFromReq builds a StoredJob from the request and persists it.
func startJobFromReq(store *Store, kind string, req map[string]any) *StoredJob {
	j := &StoredJob{
		JobName:               getStr(req, "JobName"),
		InputDataConfig:       getMap(req, "InputDataConfig"),
		OutputDataConfig:      getMap(req, "OutputDataConfig"),
		DataAccessRoleArn:     getStr(req, "DataAccessRoleArn"),
		LanguageCode:          getStr(req, "LanguageCode"),
		VolumeKmsKeyId:        getStr(req, "VolumeKmsKeyId"),
		VpcConfig:             getMap(req, "VpcConfig"),
		ClientRequestToken:    getStr(req, "ClientRequestToken"),
		FlywheelArn:           getStr(req, "FlywheelArn"),
		ModelKmsKeyId:         getStr(req, "ModelKmsKeyId"),
		NumberOfTopics:        getInt(req, "NumberOfTopics"),
		TargetEventTypes:      getStrList(req, "TargetEventTypes"),
		Mode:                  getStr(req, "Mode"),
		RedactionConfig:       getMap(req, "RedactionConfig"),
		EntityRecognizerArn:   getStr(req, "EntityRecognizerArn"),
		DocumentClassifierArn: getStr(req, "DocumentClassifierArn"),
	}
	return store.CreateJob(kind, j)
}

func startJobResponse(j *StoredJob) map[string]any {
	return map[string]any{
		"JobId":     j.JobId,
		"JobArn":    j.JobArn,
		"JobStatus": j.JobStatus,
	}
}

func startJobHandler(kind string) func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
		var req map[string]any
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		if getMap(req, "InputDataConfig") == nil {
			return jsonErr(service.ErrValidation("InputDataConfig is required."))
		}
		if getMap(req, "OutputDataConfig") == nil {
			return jsonErr(service.ErrValidation("OutputDataConfig is required."))
		}
		if getStr(req, "DataAccessRoleArn") == "" {
			return jsonErr(service.ErrValidation("DataAccessRoleArn is required."))
		}
		j := startJobFromReq(store, kind, req)
		return jsonOK(startJobResponse(j))
	}
}

func describeJobHandler(kind, propsKey string) func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
		var req map[string]any
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		id := getStr(req, "JobId")
		if id == "" {
			return jsonErr(service.ErrValidation("JobId is required."))
		}
		j, err := store.GetJob(kind, id)
		if err != nil {
			return jsonErr(err)
		}
		return jsonOK(map[string]any{propsKey: jobProps(j)})
	}
}

func listJobsHandler(kind, listKey string) func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
		var req map[string]any
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		list := store.ListJobs(kind)
		out := make([]map[string]any, 0, len(list))
		for _, j := range list {
			out = append(out, jobProps(j))
		}
		return jsonOK(map[string]any{listKey: out})
	}
}

func stopJobHandler(kind string) func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return func(ctx *service.RequestContext, store *Store) (*service.Response, error) {
		var req map[string]any
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		id := getStr(req, "JobId")
		if id == "" {
			return jsonErr(service.ErrValidation("JobId is required."))
		}
		j, err := store.StopJob(kind, id)
		if err != nil {
			return jsonErr(err)
		}
		return jsonOK(map[string]any{
			"JobId":     j.JobId,
			"JobStatus": j.JobStatus,
		})
	}
}

// ── Document classification job ─────────────────────────────────────────────

func handleStartDocumentClassificationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindDocClassification)(ctx, store)
}

func handleDescribeDocumentClassificationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindDocClassification, "DocumentClassificationJobProperties")(ctx, store)
}

func handleListDocumentClassificationJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindDocClassification, "DocumentClassificationJobPropertiesList")(ctx, store)
}

// ── Dominant language detection job ─────────────────────────────────────────

func handleStartDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindDominantLanguage)(ctx, store)
}

func handleDescribeDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindDominantLanguage, "DominantLanguageDetectionJobProperties")(ctx, store)
}

func handleListDominantLanguageDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindDominantLanguage, "DominantLanguageDetectionJobPropertiesList")(ctx, store)
}

func handleStopDominantLanguageDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindDominantLanguage)(ctx, store)
}

// ── Entities detection job ──────────────────────────────────────────────────

func handleStartEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindEntities)(ctx, store)
}

func handleDescribeEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindEntities, "EntitiesDetectionJobProperties")(ctx, store)
}

func handleListEntitiesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindEntities, "EntitiesDetectionJobPropertiesList")(ctx, store)
}

func handleStopEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindEntities)(ctx, store)
}

// ── Events detection job ────────────────────────────────────────────────────

func handleStartEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindEvents)(ctx, store)
}

func handleDescribeEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindEvents, "EventsDetectionJobProperties")(ctx, store)
}

func handleListEventsDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindEvents, "EventsDetectionJobPropertiesList")(ctx, store)
}

func handleStopEventsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindEvents)(ctx, store)
}

// ── Key phrases detection job ───────────────────────────────────────────────

func handleStartKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindKeyPhrases)(ctx, store)
}

func handleDescribeKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindKeyPhrases, "KeyPhrasesDetectionJobProperties")(ctx, store)
}

func handleListKeyPhrasesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindKeyPhrases, "KeyPhrasesDetectionJobPropertiesList")(ctx, store)
}

func handleStopKeyPhrasesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindKeyPhrases)(ctx, store)
}

// ── PII entities detection job ──────────────────────────────────────────────

func handleStartPiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindPiiEntities)(ctx, store)
}

func handleDescribePiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindPiiEntities, "PiiEntitiesDetectionJobProperties")(ctx, store)
}

func handleListPiiEntitiesDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindPiiEntities, "PiiEntitiesDetectionJobPropertiesList")(ctx, store)
}

func handleStopPiiEntitiesDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindPiiEntities)(ctx, store)
}

// ── Sentiment detection job ─────────────────────────────────────────────────

func handleStartSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindSentiment)(ctx, store)
}

func handleDescribeSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindSentiment, "SentimentDetectionJobProperties")(ctx, store)
}

func handleListSentimentDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindSentiment, "SentimentDetectionJobPropertiesList")(ctx, store)
}

func handleStopSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindSentiment)(ctx, store)
}

// ── Targeted sentiment detection job ────────────────────────────────────────

func handleStartTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindTargetedSentiment)(ctx, store)
}

func handleDescribeTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindTargetedSentiment, "TargetedSentimentDetectionJobProperties")(ctx, store)
}

func handleListTargetedSentimentDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindTargetedSentiment, "TargetedSentimentDetectionJobPropertiesList")(ctx, store)
}

func handleStopTargetedSentimentDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return stopJobHandler(jobKindTargetedSentiment)(ctx, store)
}

// ── Topics detection job ────────────────────────────────────────────────────

func handleStartTopicsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return startJobHandler(jobKindTopics)(ctx, store)
}

func handleDescribeTopicsDetectionJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return describeJobHandler(jobKindTopics, "TopicsDetectionJobProperties")(ctx, store)
}

func handleListTopicsDetectionJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return listJobsHandler(jobKindTopics, "TopicsDetectionJobPropertiesList")(ctx, store)
}

// ── Resource policies ───────────────────────────────────────────────────────

func handlePutResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	policy := getStr(req, "ResourcePolicy")
	if policy == "" {
		return jsonErr(service.ErrValidation("ResourcePolicy is required."))
	}
	p, err := store.PutResourcePolicy(arn, policy, getStr(req, "PolicyRevisionId"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"PolicyRevisionId": p.PolicyRevisionId})
}

func handleDescribeResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	p, err := store.GetResourcePolicy(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ResourcePolicy":   p.PolicyText,
		"CreationTime":     rfc3339(p.CreationTime),
		"LastModifiedTime": rfc3339(p.LastModifiedTime),
		"PolicyRevisionId": p.PolicyRevisionId,
	})
}

func handleDeleteResourcePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if err := store.DeleteResourcePolicy(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── ImportModel ─────────────────────────────────────────────────────────────

func handleImportModel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sourceArn := getStr(req, "SourceModelArn")
	if sourceArn == "" {
		return jsonErr(service.ErrValidation("SourceModelArn is required."))
	}
	modelName := getStr(req, "ModelName")
	if modelName == "" {
		modelName = extractNameFromArn(sourceArn) + "-imported"
	}
	versionName := getStr(req, "VersionName")

	// Source ARN tells us whether this is a classifier or recognizer.
	if strings.Contains(sourceArn, ":document-classifier/") {
		c := &StoredDocumentClassifier{
			Name:           modelName,
			VersionName:    versionName,
			Status:         "TRAINED",
			SourceModelArn: sourceArn,
			ModelKmsKeyId:  getStr(req, "ModelKmsKeyId"),
			DataAccessRoleArn: getStr(req, "DataAccessRoleArn"),
			Tags:           parseTagList(req, "Tags"),
		}
		saved, err := store.CreateDocumentClassifier(c)
		if err != nil {
			return jsonErr(err)
		}
		if len(saved.Tags) > 0 {
			store.TagResource(saved.Arn, saved.Tags)
		}
		return jsonOK(map[string]any{"ModelArn": saved.Arn})
	}
	if strings.Contains(sourceArn, ":entity-recognizer/") {
		r := &StoredEntityRecognizer{
			Name:           modelName,
			VersionName:    versionName,
			Status:         "TRAINED",
			SourceModelArn: sourceArn,
			ModelKmsKeyId:  getStr(req, "ModelKmsKeyId"),
			DataAccessRoleArn: getStr(req, "DataAccessRoleArn"),
			Tags:           parseTagList(req, "Tags"),
		}
		saved, err := store.CreateEntityRecognizer(r)
		if err != nil {
			return jsonErr(err)
		}
		if len(saved.Tags) > 0 {
			store.TagResource(saved.Arn, saved.Tags)
		}
		return jsonOK(map[string]any{"ModelArn": saved.Arn})
	}
	// Default: treat as classifier import for unknown ARN shapes.
	c := &StoredDocumentClassifier{
		Name:           modelName,
		VersionName:    versionName,
		Status:         "TRAINED",
		SourceModelArn: sourceArn,
		Tags:           parseTagList(req, "Tags"),
	}
	saved, err := store.CreateDocumentClassifier(c)
	if err != nil {
		return jsonErr(err)
	}
	if len(saved.Tags) > 0 {
		store.TagResource(saved.Arn, saved.Tags)
	}
	return jsonOK(map[string]any{"ModelArn": saved.Arn})
}

// ── Tagging ─────────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.TagResource(arn, parseTagList(req, "Tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.UntagResource(arn, getStrList(req, "TagKeys"))
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	return jsonOK(map[string]any{
		"ResourceArn": arn,
		"Tags":        tagsToList(store.ListTags(arn)),
	})
}
