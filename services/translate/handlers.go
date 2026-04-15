package translate

import (
	"net/http"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

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
		return service.NewAWSError("InvalidParameterValueException",
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

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getBytes(m map[string]any, key string) []byte {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case string:
		return []byte(x)
	case []byte:
		return x
	case []any:
		out := make([]byte, 0, len(x))
		for _, b := range x {
			if n, ok := b.(float64); ok {
				out = append(out, byte(int(n)))
			}
		}
		return out
	}
	return nil
}

func getStringTags(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	arr, ok := m[key].([]any)
	if !ok {
		return out
	}
	for _, t := range arr {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		k := getStr(tm, "Key")
		v := getStr(tm, "Value")
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func rfc3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ── Terminology handlers ─────────────────────────────────────────────────────

func handleImportTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	data := getMap(req, "TerminologyData")
	if data == nil {
		return jsonErr(service.ErrValidation("TerminologyData is required."))
	}
	format := getStr(data, "Format")
	if format == "" {
		return jsonErr(service.ErrValidation("TerminologyData.Format is required."))
	}
	directionality := getStr(data, "Directionality")
	if directionality == "" {
		directionality = "UNI"
	}
	sourceLang := getStr(data, "SourceLanguageCode")
	targetLangs := getStrList(data, "TargetLanguageCodes")
	description := getStr(req, "Description")
	var fileBytes []byte
	if f := getMap(req, "TerminologyData"); f != nil {
		fileBytes = getBytes(f, "File")
	}

	term := store.ImportTerminology(name, description, sourceLang, targetLangs, format, directionality, fileBytes, getStringTags(req, "Tags"))
	return jsonOK(map[string]any{
		"TerminologyProperties": terminologyProps(term),
	})
}

func handleGetTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	t, err := store.GetTerminology(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"TerminologyProperties": terminologyProps(t),
		"TerminologyDataLocation": map[string]any{
			"Location":      "s3://cloudmock-mock/terminology/" + t.Name,
			"RepositoryType": "S3",
		},
	})
}

func handleDeleteTerminology(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.DeleteTerminology(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListTerminologies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	terms := store.ListTerminologies()
	out := make([]map[string]any, 0, len(terms))
	for _, t := range terms {
		out = append(out, terminologyProps(t))
	}
	return jsonOK(map[string]any{"TerminologyPropertiesList": out})
}

func terminologyProps(t *StoredTerminology) map[string]any {
	return map[string]any{
		"Name":                 t.Name,
		"Arn":                  t.Arn,
		"Description":          t.Description,
		"SourceLanguageCode":   t.SourceLanguage,
		"TargetLanguageCodes":  t.TargetLanguages,
		"TermCount":            t.TermCount,
		"Format":               t.Format,
		"Directionality":       t.Directionality,
		"CreatedAt":            rfc3339(t.CreatedAt),
		"LastUpdatedAt":        rfc3339(t.LastUpdatedAt),
	}
}

// ── Parallel data handlers ──────────────────────────────────────────────────

func handleCreateParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	config := getMap(req, "ParallelDataConfig")
	if config == nil {
		return jsonErr(service.ErrValidation("ParallelDataConfig is required."))
	}
	encKey := ""
	if ek := getMap(req, "EncryptionKey"); ek != nil {
		encKey = getStr(ek, "Id")
	}
	pd, err := store.CreateParallelData(name, getStr(req, "Description"), "", nil, config, encKey, getStringTags(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"Name":   pd.Name,
		"Status": pd.Status,
	})
}

func handleGetParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pd, err := store.GetParallelData(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ParallelDataProperties": parallelDataProps(pd),
		"DataLocation": map[string]any{
			"Location":       "s3://cloudmock-mock/parallel-data/" + pd.Name,
			"RepositoryType": "S3",
		},
	})
}

func handleDeleteParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if err := store.DeleteParallelData(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"Name":   name,
		"Status": "DELETING",
	})
}

func handleListParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	pds := store.ListParallelData()
	out := make([]map[string]any, 0, len(pds))
	for _, pd := range pds {
		out = append(out, parallelDataProps(pd))
	}
	return jsonOK(map[string]any{"ParallelDataPropertiesList": out})
}

func handleUpdateParallelData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pd, err := store.UpdateParallelData(name, getStr(req, "Description"), getMap(req, "ParallelDataConfig"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"Name":          pd.Name,
		"Status":        pd.Status,
		"LatestUpdateAttemptStatus": "ACTIVE",
		"LatestUpdateAttemptAt":     rfc3339(pd.LastUpdatedAt),
	})
}

func parallelDataProps(pd *StoredParallelData) map[string]any {
	return map[string]any{
		"Name":                pd.Name,
		"Arn":                 pd.Arn,
		"Description":         pd.Description,
		"Status":              pd.Status,
		"SourceLanguageCode":  pd.SourceLanguageCode,
		"TargetLanguageCodes": pd.TargetLanguageCodes,
		"ParallelDataConfig":  pd.ParallelDataConfig,
		"CreatedAt":           rfc3339(pd.CreatedAt),
		"LastUpdatedAt":       rfc3339(pd.LastUpdatedAt),
	}
}

// ── Jobs ─────────────────────────────────────────────────────────────────────

func handleStartTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	jobName := getStr(req, "JobName")
	if jobName == "" {
		return jsonErr(service.ErrValidation("JobName is required."))
	}
	sourceLang := getStr(req, "SourceLanguageCode")
	if sourceLang == "" {
		return jsonErr(service.ErrValidation("SourceLanguageCode is required."))
	}
	targetLangs := getStrList(req, "TargetLanguageCodes")
	if len(targetLangs) == 0 {
		return jsonErr(service.ErrValidation("TargetLanguageCodes is required."))
	}
	inputCfg := getMap(req, "InputDataConfig")
	if inputCfg == nil {
		return jsonErr(service.ErrValidation("InputDataConfig is required."))
	}
	outputCfg := getMap(req, "OutputDataConfig")
	if outputCfg == nil {
		return jsonErr(service.ErrValidation("OutputDataConfig is required."))
	}
	dataAccessRole := getStr(req, "DataAccessRoleArn")
	if dataAccessRole == "" {
		return jsonErr(service.ErrValidation("DataAccessRoleArn is required."))
	}

	job := store.StartJob(&StoredJob{
		JobName:             jobName,
		SourceLanguageCode:  sourceLang,
		TargetLanguageCodes: targetLangs,
		TerminologyNames:    getStrList(req, "TerminologyNames"),
		ParallelDataNames:   getStrList(req, "ParallelDataNames"),
		InputDataConfig:     inputCfg,
		OutputDataConfig:    outputCfg,
		DataAccessRoleArn:   dataAccessRole,
		Settings:            getMap(req, "Settings"),
	})
	return jsonOK(map[string]any{
		"JobId":     job.JobID,
		"JobStatus": job.JobStatus,
	})
}

func handleStopTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "JobId")
	if id == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	job, err := store.StopJob(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"JobId":     job.JobID,
		"JobStatus": job.JobStatus,
	})
}

func handleDescribeTextTranslationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "JobId")
	if id == "" {
		return jsonErr(service.ErrValidation("JobId is required."))
	}
	job, err := store.GetJob(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"TextTranslationJobProperties": jobProps(job),
	})
}

func handleListTextTranslationJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListJobs()
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, jobProps(j))
	}
	return jsonOK(map[string]any{"TextTranslationJobPropertiesList": out})
}

func jobProps(j *StoredJob) map[string]any {
	return map[string]any{
		"JobId":               j.JobID,
		"JobName":             j.JobName,
		"JobStatus":           j.JobStatus,
		"SubmittedTime":       rfc3339(j.SubmittedTime),
		"EndTime":             rfc3339(j.EndTime),
		"SourceLanguageCode":  j.SourceLanguageCode,
		"TargetLanguageCodes": j.TargetLanguageCodes,
		"TerminologyNames":    j.TerminologyNames,
		"ParallelDataNames":   j.ParallelDataNames,
		"InputDataConfig":     j.InputDataConfig,
		"OutputDataConfig":    j.OutputDataConfig,
		"DataAccessRoleArn":   j.DataAccessRoleArn,
		"Settings":            j.Settings,
		"Message":             j.Message,
		"JobDetails": map[string]any{
			"TranslatedDocumentsCount": 0,
			"DocumentsWithErrorsCount": 0,
			"InputDocumentsCount":      0,
		},
	}
}

// ── Tagging ──────────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.TagResource(arn, getStringTags(req, "Tags"))
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
	tags := store.ListTags(arn)
	tagList := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, map[string]any{"Key": k, "Value": v})
	}
	return jsonOK(map[string]any{"Tags": tagList})
}

// ── Direct translation ──────────────────────────────────────────────────────

// Cloudmock doesn't actually translate. It returns the input text unchanged
// with the requested target language code — enough for SDK smoke tests.

func handleTranslateText(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	text := getStr(req, "Text")
	if text == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	sourceLang := getStr(req, "SourceLanguageCode")
	if sourceLang == "" {
		return jsonErr(service.ErrValidation("SourceLanguageCode is required."))
	}
	targetLang := getStr(req, "TargetLanguageCode")
	if targetLang == "" {
		return jsonErr(service.ErrValidation("TargetLanguageCode is required."))
	}

	detected := sourceLang
	if sourceLang == "auto" {
		detected = "en"
	}
	return jsonOK(map[string]any{
		"TranslatedText":     text,
		"SourceLanguageCode": detected,
		"TargetLanguageCode": targetLang,
		"AppliedTerminologies": []any{},
	})
}

func handleTranslateDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	doc := getMap(req, "Document")
	if doc == nil {
		return jsonErr(service.ErrValidation("Document is required."))
	}
	content := getBytes(doc, "Content")
	if len(content) == 0 {
		return jsonErr(service.ErrValidation("Document.Content is required."))
	}
	sourceLang := getStr(req, "SourceLanguageCode")
	if sourceLang == "" {
		return jsonErr(service.ErrValidation("SourceLanguageCode is required."))
	}
	targetLang := getStr(req, "TargetLanguageCode")
	if targetLang == "" {
		return jsonErr(service.ErrValidation("TargetLanguageCode is required."))
	}

	return jsonOK(map[string]any{
		"TranslatedDocument": map[string]any{
			"Content": content,
		},
		"SourceLanguageCode": sourceLang,
		"TargetLanguageCode": targetLang,
	})
}

// ── Static catalogue ─────────────────────────────────────────────────────────

var supportedLanguages = []map[string]any{
	{"LanguageName": "Afrikaans", "LanguageCode": "af"},
	{"LanguageName": "Arabic", "LanguageCode": "ar"},
	{"LanguageName": "Bengali", "LanguageCode": "bn"},
	{"LanguageName": "Chinese (Simplified)", "LanguageCode": "zh"},
	{"LanguageName": "Chinese (Traditional)", "LanguageCode": "zh-TW"},
	{"LanguageName": "Czech", "LanguageCode": "cs"},
	{"LanguageName": "Danish", "LanguageCode": "da"},
	{"LanguageName": "Dutch", "LanguageCode": "nl"},
	{"LanguageName": "English", "LanguageCode": "en"},
	{"LanguageName": "Finnish", "LanguageCode": "fi"},
	{"LanguageName": "French", "LanguageCode": "fr"},
	{"LanguageName": "German", "LanguageCode": "de"},
	{"LanguageName": "Greek", "LanguageCode": "el"},
	{"LanguageName": "Hebrew", "LanguageCode": "he"},
	{"LanguageName": "Hindi", "LanguageCode": "hi"},
	{"LanguageName": "Italian", "LanguageCode": "it"},
	{"LanguageName": "Japanese", "LanguageCode": "ja"},
	{"LanguageName": "Korean", "LanguageCode": "ko"},
	{"LanguageName": "Norwegian", "LanguageCode": "no"},
	{"LanguageName": "Polish", "LanguageCode": "pl"},
	{"LanguageName": "Portuguese", "LanguageCode": "pt"},
	{"LanguageName": "Russian", "LanguageCode": "ru"},
	{"LanguageName": "Spanish", "LanguageCode": "es"},
	{"LanguageName": "Swedish", "LanguageCode": "sv"},
	{"LanguageName": "Thai", "LanguageCode": "th"},
	{"LanguageName": "Turkish", "LanguageCode": "tr"},
	{"LanguageName": "Ukrainian", "LanguageCode": "uk"},
	{"LanguageName": "Urdu", "LanguageCode": "ur"},
	{"LanguageName": "Vietnamese", "LanguageCode": "vi"},
}

func handleListLanguages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	_ = parseJSON(ctx.Body, &req)
	display := getStr(req, "DisplayLanguageCode")
	if display == "" {
		display = "en"
	}
	return jsonOK(map[string]any{
		"Languages":           supportedLanguages,
		"DisplayLanguageCode": display,
	})
}

// Ensure imports are used even when some sections get trimmed.
var _ = strings.TrimSpace
