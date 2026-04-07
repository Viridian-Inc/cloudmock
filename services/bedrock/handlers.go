package bedrock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusCreated,
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
	Key   string `json:"key"`
	Value string `json:"value"`
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

func getStrMap(params map[string]any, key string) map[string]string {
	if v, ok := params[key]; ok {
		if m, ok := v.(map[string]any); ok {
			result := make(map[string]string, len(m))
			for k, val := range m {
				if s, ok := val.(string); ok {
					result[k] = s
				}
			}
			return result
		}
	}
	return nil
}

func getInt(params map[string]any, key string) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
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
	v, ok := params["tags"]
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
			Key:   getStr(m, "key"),
			Value: getStr(m, "value"),
		})
	}
	return entries
}

// ---- Model customization job handlers ----

func handleCreateModelCustomizationJob(params map[string]any, store *Store) (*service.Response, error) {
	jobName := getStr(params, "jobName")
	if jobName == "" {
		return jsonErr(service.ErrValidation("jobName is required."))
	}
	job, awsErr := store.CreateModelCustomizationJob(
		jobName,
		getStr(params, "baseModelIdentifier"),
		getStr(params, "customModelName"),
		getStr(params, "roleArn"),
		getStr(params, "customizationType"),
		getStrMap(params, "hyperParameters"),
		getMap(params, "trainingDataConfig"),
		getMap(params, "validationDataConfig"),
		getMap(params, "outputDataConfig"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"jobArn": job.JobArn})
}

func handleGetModelCustomizationJob(params map[string]any, store *Store) (*service.Response, error) {
	jobName := getStr(params, "jobIdentifier")
	if jobName == "" {
		return jsonErr(service.ErrValidation("jobIdentifier is required."))
	}
	job, awsErr := store.GetModelCustomizationJob(jobName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"jobName":             job.JobName,
		"jobArn":              job.JobArn,
		"status":              string(job.Status),
		"baseModelIdentifier": job.BaseModelIdentifier,
		"customModelName":     job.CustomModelName,
		"customModelArn":      job.CustomModelArn,
		"roleArn":             job.RoleArn,
		"customizationType":   job.CustomizationType,
		"hyperParameters":     job.HyperParameters,
		"trainingDataConfig":  job.TrainingDataConfig,
		"outputDataConfig":    job.OutputDataConfig,
		"creationTime":        job.CreationTime.Format(time.RFC3339),
	}
	if job.ValidationDataConfig != nil {
		resp["validationDataConfig"] = job.ValidationDataConfig
	}
	if job.EndTime != nil {
		resp["endTime"] = job.EndTime.Format(time.RFC3339)
	}
	if job.FailureMessage != "" {
		resp["failureMessage"] = job.FailureMessage
	}
	return jsonOK(resp)
}

func handleListModelCustomizationJobs(store *Store) (*service.Response, error) {
	jobs := store.ListModelCustomizationJobs()
	summaries := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		entry := map[string]any{
			"jobName":             job.JobName,
			"jobArn":              job.JobArn,
			"status":              string(job.Status),
			"baseModelIdentifier": job.BaseModelIdentifier,
			"customModelName":     job.CustomModelName,
			"customModelArn":      job.CustomModelArn,
			"creationTime":        job.CreationTime.Format(time.RFC3339),
		}
		if job.EndTime != nil {
			entry["endTime"] = job.EndTime.Format(time.RFC3339)
		}
		summaries = append(summaries, entry)
	}
	return jsonOK(map[string]any{"modelCustomizationJobSummaries": summaries})
}

func handleStopModelCustomizationJob(params map[string]any, store *Store) (*service.Response, error) {
	jobName := getStr(params, "jobIdentifier")
	if jobName == "" {
		return jsonErr(service.ErrValidation("jobIdentifier is required."))
	}
	if awsErr := store.StopModelCustomizationJob(jobName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Provisioned model throughput handlers ----

func handleCreateProvisionedModelThroughput(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "provisionedModelName")
	if name == "" {
		return jsonErr(service.ErrValidation("provisionedModelName is required."))
	}
	pm, awsErr := store.CreateProvisionedModelThroughput(
		name,
		getStr(params, "modelId"),
		getInt(params, "modelUnits"),
		getStr(params, "commitmentDuration"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"provisionedModelArn": pm.ProvisionedModelArn})
}

func handleGetProvisionedModelThroughput(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "provisionedModelId")
	if name == "" {
		return jsonErr(service.ErrValidation("provisionedModelId is required."))
	}
	pm, awsErr := store.GetProvisionedModelThroughput(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"provisionedModelId":   pm.ProvisionedModelId,
		"provisionedModelArn":  pm.ProvisionedModelArn,
		"provisionedModelName": pm.ProvisionedModelName,
		"modelArn":             pm.ModelArn,
		"modelUnits":           pm.ModelUnits,
		"desiredModelUnits":    pm.DesiredModelUnits,
		"status":               string(pm.Status),
		"commitmentDuration":   pm.CommitmentDuration,
		"creationTime":         pm.CreationTime.Format(time.RFC3339),
		"lastModifiedTime":     pm.LastModifiedTime.Format(time.RFC3339),
	})
}

func handleListProvisionedModelThroughputs(store *Store) (*service.Response, error) {
	pms := store.ListProvisionedModelThroughputs()
	summaries := make([]map[string]any, 0, len(pms))
	for _, pm := range pms {
		summaries = append(summaries, map[string]any{
			"provisionedModelArn":  pm.ProvisionedModelArn,
			"provisionedModelName": pm.ProvisionedModelName,
			"modelArn":             pm.ModelArn,
			"modelUnits":           pm.ModelUnits,
			"desiredModelUnits":    pm.DesiredModelUnits,
			"status":               string(pm.Status),
			"creationTime":         pm.CreationTime.Format(time.RFC3339),
			"lastModifiedTime":     pm.LastModifiedTime.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"provisionedModelSummaries": summaries})
}

func handleUpdateProvisionedModelThroughput(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "provisionedModelId")
	if name == "" {
		return jsonErr(service.ErrValidation("provisionedModelId is required."))
	}
	_, awsErr := store.UpdateProvisionedModelThroughput(
		name,
		getInt(params, "desiredModelUnits"),
		getStr(params, "desiredModelId"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteProvisionedModelThroughput(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "provisionedModelId")
	if name == "" {
		return jsonErr(service.ErrValidation("provisionedModelId is required."))
	}
	if awsErr := store.DeleteProvisionedModelThroughput(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Foundation model handlers ----

func handleGetFoundationModel(params map[string]any, store *Store) (*service.Response, error) {
	modelId := getStr(params, "modelIdentifier")
	if modelId == "" {
		return jsonErr(service.ErrValidation("modelIdentifier is required."))
	}
	fm, awsErr := store.GetFoundationModel(modelId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"modelDetails": map[string]any{
			"modelId":                    fm.ModelId,
			"modelArn":                   fm.ModelArn,
			"modelName":                  fm.ModelName,
			"provider":                   fm.Provider,
			"inputModalities":            fm.InputModalities,
			"outputModalities":           fm.OutputModalities,
			"customizationsSupported":    fm.CustomizationsSupported,
			"inferenceTypesSupported":    fm.InferenceTypesSupported,
			"responseStreamingSupported": fm.ResponseStreamingSupported,
		},
	})
}

func handleListFoundationModels(store *Store) (*service.Response, error) {
	models := store.ListFoundationModels()
	summaries := make([]map[string]any, 0, len(models))
	for _, fm := range models {
		summaries = append(summaries, map[string]any{
			"modelId":                    fm.ModelId,
			"modelArn":                   fm.ModelArn,
			"modelName":                  fm.ModelName,
			"provider":                   fm.Provider,
			"inputModalities":            fm.InputModalities,
			"outputModalities":           fm.OutputModalities,
			"customizationsSupported":    fm.CustomizationsSupported,
			"inferenceTypesSupported":    fm.InferenceTypesSupported,
			"responseStreamingSupported": fm.ResponseStreamingSupported,
		})
	}
	return jsonOK(map[string]any{"modelSummaries": summaries})
}

// ---- Tag handlers ----

func handleTagResource(params map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(params, "resourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceARN is required."))
	}
	tags := tagsFromEntries(parseTags(params))
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(params map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(params, "resourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceARN is required."))
	}
	tagKeys := getStringSlice(params, "tagKeys")
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(params map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(params, "resourceARN")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceARN is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"tags": entriesToTags(tags)})
}

// ---- InvokeModel handler ----

func handleInvokeModel(params map[string]any, store *Store) (*service.Response, error) {
	modelID := getStr(params, "modelId")
	if modelID == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"modelId is required.", http.StatusBadRequest))
	}

	// Validate the model exists in foundation models.
	found := false
	for _, fm := range store.ListFoundationModels() {
		if fm.ModelId == modelID || fm.ModelName == modelID {
			found = true
			break
		}
	}
	if !found {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Could not resolve the foundation model from model identifier: "+modelID,
			http.StatusNotFound))
	}

	// Parse the body/prompt from the request.
	body := getStr(params, "body")
	prompt := "Hello"
	if body != "" {
		var bodyMap map[string]any
		if json.Unmarshal([]byte(body), &bodyMap) == nil {
			if p, ok := bodyMap["prompt"].(string); ok {
				prompt = p
			} else if msgs, ok := bodyMap["messages"].([]any); ok && len(msgs) > 0 {
				if msg, ok := msgs[0].(map[string]any); ok {
					if c, ok := msg["content"].(string); ok {
						prompt = c
					}
				}
			}
		}
	}

	// Generate a mock response.
	mockResponse := map[string]any{
		"generated_text": fmt.Sprintf("This is a mock response from %s. You asked: %s", modelID, prompt),
		"stop_reason":    "end_turn",
		"usage": map[string]any{
			"input_tokens":  len(prompt) / 4,
			"output_tokens": 25,
		},
	}

	responseBytes, _ := json.Marshal(mockResponse)
	return jsonOK(map[string]any{
		"body":        string(responseBytes),
		"contentType": "application/json",
	})
}

// ---- Guardrail handlers ----

func handleCreateGuardrail(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "name")
	if name == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"name is required.", http.StatusBadRequest))
	}

	guardrail := store.CreateGuardrail(name,
		getStr(params, "description"),
		getStr(params, "blockedInputMessaging"),
		getStr(params, "blockedOutputsMessaging"),
	)

	return jsonCreated(map[string]any{
		"guardrailId":  guardrail.ID,
		"guardrailArn": guardrail.ARN,
		"name":         guardrail.Name,
		"version":      guardrail.Version,
		"createdAt":    guardrail.CreatedAt.Format(time.RFC3339),
	})
}

func handleApplyGuardrail(params map[string]any, store *Store) (*service.Response, error) {
	guardrailID := getStr(params, "guardrailIdentifier")
	if guardrailID == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"guardrailIdentifier is required.", http.StatusBadRequest))
	}
	source := getStr(params, "source")
	if source == "" {
		source = "INPUT"
	}

	guardrail, ok := store.GetGuardrail(guardrailID)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Guardrail not found: "+guardrailID, http.StatusNotFound))
	}

	// Mock content analysis result.
	content := getStr(params, "content")
	action := "NONE"
	if content != "" && len(content) > 500 {
		action = "GUARDRAIL_INTERVENED"
	}

	return jsonOK(map[string]any{
		"action": action,
		"output": map[string]any{
			"text": content,
		},
		"assessments": []map[string]any{
			{
				"guardrailId": guardrail.ID,
				"topicPolicy": map[string]any{
					"topics": []any{},
				},
				"contentPolicy": map[string]any{
					"filters": []any{},
				},
			},
		},
		"guardrailCoverage": map[string]any{
			"textCharacters": map[string]any{
				"guarded": len(content),
				"total":   len(content),
			},
		},
	})
}

// ---- GetGuardrail ----

func handleGetGuardrail(params map[string]any, store *Store) (*service.Response, error) {
	id := getStr(params, "guardrailIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("guardrailIdentifier is required."))
	}
	g, ok := store.GetGuardrail(id)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Guardrail %s not found", id), http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"guardrailId":  g.ID,
		"guardrailArn": g.ARN,
		"name":         g.Name,
		"description":  g.Description,
		"version":      g.Version,
		"status":       g.Status,
		"createdAt":    unixFloat(g.CreatedAt),
		"updatedAt":    unixFloat(g.UpdatedAt),
	})
}

// ---- ListGuardrails ----

func handleListGuardrails(store *Store) (*service.Response, error) {
	guardrails := store.ListGuardrails()
	summaries := make([]map[string]any, 0, len(guardrails))
	for _, g := range guardrails {
		summaries = append(summaries, map[string]any{
			"guardrailId":  g.ID,
			"guardrailArn": g.ARN,
			"name":         g.Name,
			"version":      g.Version,
			"status":       g.Status,
			"createdAt":    unixFloat(g.CreatedAt),
			"updatedAt":    unixFloat(g.UpdatedAt),
		})
	}
	return jsonOK(map[string]any{"guardrails": summaries})
}

// ---- UpdateGuardrail ----

func handleUpdateGuardrail(params map[string]any, store *Store) (*service.Response, error) {
	id := getStr(params, "guardrailIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("guardrailIdentifier is required."))
	}
	name := getStr(params, "name")
	description := getStr(params, "description")
	if !store.UpdateGuardrail(id, name, description) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Guardrail %s not found", id), http.StatusNotFound))
	}
	g, _ := store.GetGuardrail(id)
	return jsonOK(map[string]any{
		"guardrailId":  g.ID,
		"guardrailArn": g.ARN,
		"version":      g.Version,
		"updatedAt":    unixFloat(g.UpdatedAt),
	})
}

// ---- DeleteGuardrail ----

func handleDeleteGuardrail(params map[string]any, store *Store) (*service.Response, error) {
	id := getStr(params, "guardrailIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("guardrailIdentifier is required."))
	}
	if !store.DeleteGuardrail(id) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Guardrail %s not found", id), http.StatusNotFound))
	}
	return emptyOK()
}

// ---- CreateModelEvaluationJob ----

func handleCreateModelEvaluationJob(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "jobName")
	if name == "" {
		return jsonErr(service.ErrValidation("jobName is required."))
	}
	roleArn := getStr(params, "roleArn")
	modelId := getStr(params, "modelIdentifier")
	evalConfig := getMap(params, "evaluationConfig")

	job, ok := store.CreateModelEvaluationJob(name, roleArn, modelId, evalConfig)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Model evaluation job %s already exists", name), http.StatusConflict))
	}
	return jsonCreated(map[string]any{"jobArn": job.JobArn})
}

// ---- GetModelEvaluationJob ----

func handleGetModelEvaluationJob(params map[string]any, store *Store) (*service.Response, error) {
	name := getStr(params, "jobIdentifier")
	if name == "" {
		return jsonErr(service.ErrValidation("jobIdentifier is required."))
	}
	job, ok := store.GetModelEvaluationJob(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Model evaluation job %s not found", name), http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"jobArn":          job.JobArn,
		"jobName":         job.JobName,
		"status":          job.Status,
		"modelIdentifier": job.ModelIdentifier,
		"roleArn":         job.RoleArn,
		"creationTime":    unixFloat(job.CreationTime),
		"endTime":         unixFloatPtr(job.EndTime),
	})
}

// ---- ListModelEvaluationJobs ----

func handleListModelEvaluationJobs(store *Store) (*service.Response, error) {
	jobs := store.ListModelEvaluationJobs()
	summaries := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		summaries = append(summaries, map[string]any{
			"jobArn":       j.JobArn,
			"jobName":      j.JobName,
			"status":       j.Status,
			"creationTime": unixFloat(j.CreationTime),
		})
	}
	return jsonOK(map[string]any{"jobSummaries": summaries})
}
