package polly

import (
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Input types (what SDK clients send) ─────────────────────────────────────

type deleteLexiconInput struct {
	Name string `json:"LexiconName,omitempty"`
}

type describeVoicesInput struct {
	Engine                         string `json:"Engine,omitempty"`
	IncludeAdditionalLanguageCodes bool   `json:"IncludeAdditionalLanguageCodes,omitempty"`
	LanguageCode                   string `json:"LanguageCode,omitempty"`
	NextToken                      string `json:"NextToken,omitempty"`
}

type getLexiconInput struct {
	Name string `json:"LexiconName,omitempty"`
}

type getTaskInput struct {
	TaskID string `json:"TaskId,omitempty"`
}

type listLexiconsInput struct {
	NextToken string `json:"NextToken,omitempty"`
}

type listTasksInput struct {
	MaxResults int    `json:"MaxResults,omitempty"`
	NextToken  string `json:"NextToken,omitempty"`
	Status     string `json:"Status,omitempty"`
}

type putLexiconInput struct {
	Content string `json:"Content,omitempty"`
	Name    string `json:"LexiconName,omitempty"`
}

type startTaskInput struct {
	Engine             string   `json:"Engine,omitempty"`
	LanguageCode       string   `json:"LanguageCode,omitempty"`
	LexiconNames       []string `json:"LexiconNames,omitempty"`
	OutputFormat       string   `json:"OutputFormat,omitempty"`
	OutputS3BucketName string   `json:"OutputS3BucketName,omitempty"`
	OutputS3KeyPrefix  string   `json:"OutputS3KeyPrefix,omitempty"`
	SampleRate         string   `json:"SampleRate,omitempty"`
	SnsTopicArn        string   `json:"SnsTopicArn,omitempty"`
	SpeechMarkTypes    []string `json:"SpeechMarkTypes,omitempty"`
	Text               string   `json:"Text,omitempty"`
	TextType           string   `json:"TextType,omitempty"`
	VoiceID            string   `json:"VoiceId,omitempty"`
}

type synthesizeInput struct {
	Engine          string   `json:"Engine,omitempty"`
	LanguageCode    string   `json:"LanguageCode,omitempty"`
	LexiconNames    []string `json:"LexiconNames,omitempty"`
	OutputFormat    string   `json:"OutputFormat,omitempty"`
	SampleRate      string   `json:"SampleRate,omitempty"`
	SpeechMarkTypes []string `json:"SpeechMarkTypes,omitempty"`
	Text            string   `json:"Text,omitempty"`
	TextType        string   `json:"TextType,omitempty"`
	VoiceID         string   `json:"VoiceId,omitempty"`
}

// ── Handler helpers ──────────────────────────────────────────────────────────

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
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// Static catalogue of Polly voices. Mirrors a representative slice of the
// real service so SDK consumers see deterministic results.
var staticVoices = []map[string]any{
	{"Id": "Joanna", "Name": "Joanna", "Gender": "Female", "LanguageCode": "en-US", "LanguageName": "US English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Matthew", "Name": "Matthew", "Gender": "Male", "LanguageCode": "en-US", "LanguageName": "US English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Ivy", "Name": "Ivy", "Gender": "Female", "LanguageCode": "en-US", "LanguageName": "US English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Kendra", "Name": "Kendra", "Gender": "Female", "LanguageCode": "en-US", "LanguageName": "US English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Amy", "Name": "Amy", "Gender": "Female", "LanguageCode": "en-GB", "LanguageName": "British English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Brian", "Name": "Brian", "Gender": "Male", "LanguageCode": "en-GB", "LanguageName": "British English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Emma", "Name": "Emma", "Gender": "Female", "LanguageCode": "en-GB", "LanguageName": "British English", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Celine", "Name": "Celine", "Gender": "Female", "LanguageCode": "fr-FR", "LanguageName": "French", "SupportedEngines": []string{"standard"}},
	{"Id": "Mathieu", "Name": "Mathieu", "Gender": "Male", "LanguageCode": "fr-FR", "LanguageName": "French", "SupportedEngines": []string{"standard"}},
	{"Id": "Lupe", "Name": "Lupe", "Gender": "Female", "LanguageCode": "es-US", "LanguageName": "US Spanish", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Penelope", "Name": "Penelope", "Gender": "Female", "LanguageCode": "es-US", "LanguageName": "US Spanish", "SupportedEngines": []string{"standard"}},
	{"Id": "Miguel", "Name": "Miguel", "Gender": "Male", "LanguageCode": "es-US", "LanguageName": "US Spanish", "SupportedEngines": []string{"standard"}},
	{"Id": "Takumi", "Name": "Takumi", "Gender": "Male", "LanguageCode": "ja-JP", "LanguageName": "Japanese", "SupportedEngines": []string{"neural", "standard"}},
	{"Id": "Mizuki", "Name": "Mizuki", "Gender": "Female", "LanguageCode": "ja-JP", "LanguageName": "Japanese", "SupportedEngines": []string{"standard"}},
}

func isKnownVoice(id string) bool {
	for _, v := range staticVoices {
		if v["Id"] == id {
			return true
		}
	}
	return false
}

func lexiconToMap(l *StoredLexicon) map[string]any {
	return map[string]any{
		"Name": l.Name,
		"Attributes": map[string]any{
			"Alphabet":     l.Alphabet,
			"LanguageCode": l.LanguageCode,
			"LastModified": l.LastModified.Format(time.RFC3339),
			"LexemesCount": l.LexemesCount,
			"LexiconArn":   l.Arn,
			"Size":         l.Size,
		},
	}
}

func taskToMap(t *StoredTask) map[string]any {
	out := map[string]any{
		"TaskId":            t.TaskID,
		"TaskStatus":        t.TaskStatus,
		"CreationTime":      t.CreationTime.Format(time.RFC3339),
		"RequestCharacters": t.RequestCharacters,
	}
	if t.TaskStatusReason != "" {
		out["TaskStatusReason"] = t.TaskStatusReason
	}
	if t.Engine != "" {
		out["Engine"] = t.Engine
	}
	if t.LanguageCode != "" {
		out["LanguageCode"] = t.LanguageCode
	}
	if t.OutputFormat != "" {
		out["OutputFormat"] = t.OutputFormat
	}
	if t.OutputURI != "" {
		out["OutputUri"] = t.OutputURI
	}
	if t.SampleRate != "" {
		out["SampleRate"] = t.SampleRate
	}
	if t.SnsTopicArn != "" {
		out["SnsTopicArn"] = t.SnsTopicArn
	}
	if t.TextType != "" {
		out["TextType"] = t.TextType
	}
	if t.VoiceID != "" {
		out["VoiceId"] = t.VoiceID
	}
	if len(t.LexiconNames) > 0 {
		out["LexiconNames"] = t.LexiconNames
	}
	if len(t.SpeechMarkTypes) > 0 {
		out["SpeechMarkTypes"] = t.SpeechMarkTypes
	}
	return out
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleDeleteLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	name := lexiconNameFromPath(ctx)
	if name == "" {
		var req deleteLexiconInput
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		name = req.Name
	}
	if name == "" {
		return jsonErr(service.ErrValidation("LexiconName is required."))
	}
	if err := store.DeleteLexicon(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeVoices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeVoicesInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if lang := ctx.Params["LanguageCode"]; lang != "" {
		req.LanguageCode = lang
	}
	if eng := ctx.Params["Engine"]; eng != "" {
		req.Engine = eng
	}

	out := make([]map[string]any, 0, len(staticVoices))
	for _, v := range staticVoices {
		if req.LanguageCode != "" && v["LanguageCode"] != req.LanguageCode {
			continue
		}
		if req.Engine != "" {
			engines, _ := v["SupportedEngines"].([]string)
			if !containsStr(engines, req.Engine) {
				continue
			}
		}
		out = append(out, v)
	}
	return jsonOK(map[string]any{"Voices": out})
}

func handleGetLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	name := lexiconNameFromPath(ctx)
	if name == "" {
		var req getLexiconInput
		if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
			return jsonErr(awsErr)
		}
		name = req.Name
	}
	if name == "" {
		return jsonErr(service.ErrValidation("LexiconName is required."))
	}
	lex, err := store.GetLexicon(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"Lexicon": map[string]any{
			"Name":    lex.Name,
			"Content": lex.Content,
		},
		"LexiconAttributes": map[string]any{
			"Alphabet":     lex.Alphabet,
			"LanguageCode": lex.LanguageCode,
			"LastModified": lex.LastModified.Format(time.RFC3339),
			"LexemesCount": lex.LexemesCount,
			"LexiconArn":   lex.Arn,
			"Size":         lex.Size,
		},
	})
}

func handleGetSpeechSynthesisTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	id := ctx.Params["TaskId"]
	if id == "" {
		var req getTaskInput
		_ = parseJSON(ctx.Body, &req)
		id = req.TaskID
	}
	if id == "" {
		return jsonErr(service.ErrValidation("TaskId is required."))
	}
	task, err := store.GetTask(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"SynthesisTask": taskToMap(task)})
}

func handleListLexicons(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	lexicons := store.ListLexicons()
	out := make([]map[string]any, 0, len(lexicons))
	for _, l := range lexicons {
		out = append(out, lexiconToMap(l))
	}
	return jsonOK(map[string]any{"Lexicons": out})
}

func handleListSpeechSynthesisTasks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTasksInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	status := req.Status
	if s := ctx.Params["Status"]; s != "" {
		status = s
	}

	tasks := store.ListTasks(status)
	out := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, taskToMap(t))
	}
	return jsonOK(map[string]any{"SynthesisTasks": out})
}

func handlePutLexicon(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putLexiconInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		req.Name = lexiconNameFromPath(ctx)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("LexiconName is required."))
	}
	if req.Content == "" {
		return jsonErr(service.ErrValidation("Content is required."))
	}
	store.PutLexicon(req.Name, req.Content)
	return jsonOK(map[string]any{})
}

// StartSpeechSynthesisStream returns an empty acknowledgement. Real Polly
// uses a bidirectional HTTP/2 event stream which is out of scope for the mock.
func handleStartSpeechSynthesisStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{})
}

func handleStartSpeechSynthesisTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startTaskInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.OutputFormat == "" {
		return jsonErr(service.ErrValidation("OutputFormat is required."))
	}
	if req.OutputS3BucketName == "" {
		return jsonErr(service.ErrValidation("OutputS3BucketName is required."))
	}
	if req.Text == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	if req.VoiceID == "" {
		return jsonErr(service.ErrValidation("VoiceId is required."))
	}
	if !isKnownVoice(req.VoiceID) {
		return jsonErr(service.NewAWSError("InvalidVoiceIdException",
			"Voice not found: "+req.VoiceID, http.StatusBadRequest))
	}

	id := newTaskID()
	engine := "standard"
	if req.Engine != "" {
		engine = req.Engine
	}
	textType := "text"
	if req.TextType != "" {
		textType = req.TextType
	}

	task := store.CreateTask(&StoredTask{
		TaskID:            id,
		TaskStatus:        "completed",
		CreationTime:      time.Now().UTC(),
		RequestCharacters: len(req.Text),
		Engine:            engine,
		LanguageCode:      req.LanguageCode,
		LexiconNames:      req.LexiconNames,
		OutputFormat:      req.OutputFormat,
		OutputURI:         "s3://" + req.OutputS3BucketName + "/" + req.OutputS3KeyPrefix + id,
		SampleRate:        req.SampleRate,
		SnsTopicArn:       req.SnsTopicArn,
		SpeechMarkTypes:   req.SpeechMarkTypes,
		TextType:          textType,
		VoiceID:           req.VoiceID,
	})

	return jsonOK(map[string]any{"SynthesisTask": taskToMap(task)})
}

func handleSynthesizeSpeech(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req synthesizeInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.OutputFormat == "" {
		return jsonErr(service.ErrValidation("OutputFormat is required."))
	}
	if req.Text == "" {
		return jsonErr(service.ErrValidation("Text is required."))
	}
	if req.VoiceID == "" {
		return jsonErr(service.ErrValidation("VoiceId is required."))
	}
	if !isKnownVoice(req.VoiceID) {
		return jsonErr(service.NewAWSError("InvalidVoiceIdException",
			"Voice not found: "+req.VoiceID, http.StatusBadRequest))
	}

	contentType := "audio/mpeg"
	switch req.OutputFormat {
	case "ogg_vorbis":
		contentType = "audio/ogg"
	case "pcm":
		contentType = "audio/pcm"
	case "json":
		contentType = "application/x-json-stream"
	}

	return jsonOK(map[string]any{
		"AudioStream":              []byte{},
		"Content-Type":             contentType,
		"x-amzn-RequestCharacters": len(req.Text),
	})
}

// lexiconNameFromPath pulls /v1/lexicons/{name} out of the request path for
// REST bindings, falling back to Params["LexiconName"].
func lexiconNameFromPath(ctx *service.RequestContext) string {
	if n := ctx.Params["LexiconName"]; n != "" {
		return n
	}
	if ctx.RawRequest == nil {
		return ""
	}
	path := ctx.RawRequest.URL.Path
	const prefix = "/v1/lexicons/"
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix):]
	}
	return ""
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
