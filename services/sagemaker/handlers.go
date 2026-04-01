package sagemaker

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

func getSliceOfMaps(params map[string]any, key string) []map[string]any {
	if v, ok := params[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					result = append(result, m)
				}
			}
			return result
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

// ---- Notebook Instance handlers ----

func handleCreateNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}

	nb, awsErr := store.CreateNotebookInstance(
		name,
		getStr(params, "InstanceType"),
		getStr(params, "RoleArn"),
		getStr(params, "SubnetId"),
		getStr(params, "DirectInternetAccess"),
		getStringSlice(params, "SecurityGroupIds"),
		getInt(params, "VolumeSizeInGB"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"NotebookInstanceArn": nb.NotebookInstanceArn,
	})
}

func handleDescribeNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}
	nb, awsErr := store.GetNotebookInstance(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"NotebookInstanceArn":  nb.NotebookInstanceArn,
		"NotebookInstanceName": nb.NotebookInstanceName,
		"NotebookInstanceStatus": string(nb.Status),
		"InstanceType":         nb.InstanceType,
		"RoleArn":              nb.RoleArn,
		"SubnetId":             nb.SubnetId,
		"SecurityGroupIds":     nb.SecurityGroupIds,
		"DirectInternetAccess": nb.DirectInternetAccess,
		"VolumeSizeInGB":       nb.VolumeSizeInGB,
		"CreationTime":         unixFloat(nb.CreationTime),
		"LastModifiedTime":     unixFloat(nb.LastModifiedTime),
	})
}

func handleListNotebookInstances(_ *service.RequestContext, store *Store) (*service.Response, error) {
	nbs := store.ListNotebookInstances()
	entries := make([]map[string]any, 0, len(nbs))
	for _, nb := range nbs {
		entries = append(entries, map[string]any{
			"NotebookInstanceArn":    nb.NotebookInstanceArn,
			"NotebookInstanceName":   nb.NotebookInstanceName,
			"NotebookInstanceStatus": string(nb.Status),
			"InstanceType":           nb.InstanceType,
			"CreationTime":           unixFloat(nb.CreationTime),
			"LastModifiedTime":       unixFloat(nb.LastModifiedTime),
		})
	}
	return jsonOK(map[string]any{"NotebookInstances": entries})
}

func handleDeleteNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}
	if awsErr := store.DeleteNotebookInstance(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleStartNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}
	if awsErr := store.StartNotebookInstance(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleStopNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}
	if awsErr := store.StopNotebookInstance(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateNotebookInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "NotebookInstanceName")
	if name == "" {
		return jsonErr(service.ErrValidation("NotebookInstanceName is required."))
	}
	_, awsErr := store.UpdateNotebookInstance(name, getStr(params, "InstanceType"), getStr(params, "RoleArn"), getInt(params, "VolumeSizeInGB"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Training Job handlers ----

func handleCreateTrainingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TrainingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TrainingJobName is required."))
	}
	tj, awsErr := store.CreateTrainingJob(
		name,
		getMap(params, "AlgorithmSpecification"),
		getStr(params, "RoleArn"),
		getSliceOfMaps(params, "InputDataConfig"),
		getMap(params, "OutputDataConfig"),
		getMap(params, "ResourceConfig"),
		getMap(params, "StoppingCondition"),
		getStrMap(params, "HyperParameters"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TrainingJobArn": tj.TrainingJobArn,
	})
}

func handleDescribeTrainingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TrainingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TrainingJobName is required."))
	}
	tj, awsErr := store.GetTrainingJob(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"TrainingJobName":        tj.TrainingJobName,
		"TrainingJobArn":         tj.TrainingJobArn,
		"TrainingJobStatus":      string(tj.TrainingJobStatus),
		"SecondaryStatus":        tj.SecondaryStatus,
		"AlgorithmSpecification": tj.AlgorithmSpecification,
		"RoleArn":                tj.RoleArn,
		"InputDataConfig":        tj.InputDataConfig,
		"OutputDataConfig":       tj.OutputDataConfig,
		"ResourceConfig":         tj.ResourceConfig,
		"StoppingCondition":      tj.StoppingCondition,
		"HyperParameters":        tj.HyperParameters,
		"CreationTime":           unixFloat(tj.CreationTime),
		"LastModifiedTime":       unixFloat(tj.LastModifiedTime),
		"ModelArtifacts":         tj.ModelArtifacts,
	}
	if tj.TrainingStartTime != nil {
		resp["TrainingStartTime"] = unixFloat(*tj.TrainingStartTime)
	}
	if tj.TrainingEndTime != nil {
		resp["TrainingEndTime"] = unixFloat(*tj.TrainingEndTime)
	}
	if tj.FailureReason != "" {
		resp["FailureReason"] = tj.FailureReason
	}
	return jsonOK(resp)
}

func handleListTrainingJobs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListTrainingJobs()
	entries := make([]map[string]any, 0, len(jobs))
	for _, tj := range jobs {
		entry := map[string]any{
			"TrainingJobName":   tj.TrainingJobName,
			"TrainingJobArn":    tj.TrainingJobArn,
			"TrainingJobStatus": string(tj.TrainingJobStatus),
			"CreationTime":      unixFloat(tj.CreationTime),
			"LastModifiedTime":  unixFloat(tj.LastModifiedTime),
		}
		if tj.TrainingEndTime != nil {
			entry["TrainingEndTime"] = unixFloat(*tj.TrainingEndTime)
		}
		entries = append(entries, entry)
	}
	return jsonOK(map[string]any{"TrainingJobSummaries": entries})
}

func handleStopTrainingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TrainingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TrainingJobName is required."))
	}
	if awsErr := store.StopTrainingJob(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Model handlers ----

func handleCreateModel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ModelName")
	if name == "" {
		return jsonErr(service.ErrValidation("ModelName is required."))
	}
	m, awsErr := store.CreateModel(name, getMap(params, "PrimaryContainer"), getStr(params, "ExecutionRoleArn"), tagsFromEntries(parseTags(params)))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ModelArn": m.ModelArn})
}

func handleDescribeModel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ModelName")
	if name == "" {
		return jsonErr(service.ErrValidation("ModelName is required."))
	}
	m, awsErr := store.GetModel(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ModelName":        m.ModelName,
		"ModelArn":         m.ModelArn,
		"PrimaryContainer": m.PrimaryContainer,
		"ExecutionRoleArn": m.ExecutionRoleArn,
		"CreationTime":     unixFloat(m.CreationTime),
	})
}

func handleListModels(_ *service.RequestContext, store *Store) (*service.Response, error) {
	models := store.ListModels()
	entries := make([]map[string]any, 0, len(models))
	for _, m := range models {
		entries = append(entries, map[string]any{
			"ModelName":    m.ModelName,
			"ModelArn":     m.ModelArn,
			"CreationTime": unixFloat(m.CreationTime),
		})
	}
	return jsonOK(map[string]any{"Models": entries})
}

func handleDeleteModel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ModelName")
	if name == "" {
		return jsonErr(service.ErrValidation("ModelName is required."))
	}
	if awsErr := store.DeleteModel(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Endpoint Config handlers ----

func handleCreateEndpointConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointConfigName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointConfigName is required."))
	}
	ec, awsErr := store.CreateEndpointConfig(name, getSliceOfMaps(params, "ProductionVariants"), tagsFromEntries(parseTags(params)))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"EndpointConfigArn": ec.EndpointConfigArn})
}

func handleDescribeEndpointConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointConfigName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointConfigName is required."))
	}
	ec, awsErr := store.GetEndpointConfig(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"EndpointConfigName": ec.EndpointConfigName,
		"EndpointConfigArn":  ec.EndpointConfigArn,
		"ProductionVariants": ec.ProductionVariants,
		"CreationTime":       unixFloat(ec.CreationTime),
	})
}

func handleListEndpointConfigs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	configs := store.ListEndpointConfigs()
	entries := make([]map[string]any, 0, len(configs))
	for _, ec := range configs {
		entries = append(entries, map[string]any{
			"EndpointConfigName": ec.EndpointConfigName,
			"EndpointConfigArn":  ec.EndpointConfigArn,
			"CreationTime":       unixFloat(ec.CreationTime),
		})
	}
	return jsonOK(map[string]any{"EndpointConfigs": entries})
}

func handleDeleteEndpointConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointConfigName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointConfigName is required."))
	}
	if awsErr := store.DeleteEndpointConfig(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Endpoint handlers ----

func handleCreateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}
	configName := getStr(params, "EndpointConfigName")
	if configName == "" {
		return jsonErr(service.ErrValidation("EndpointConfigName is required."))
	}
	ep, awsErr := store.CreateEndpoint(name, configName, tagsFromEntries(parseTags(params)))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"EndpointArn": ep.EndpointArn})
}

func handleDescribeEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}
	ep, awsErr := store.GetEndpoint(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"EndpointName":       ep.EndpointName,
		"EndpointArn":        ep.EndpointArn,
		"EndpointConfigName": ep.EndpointConfigName,
		"EndpointStatus":     string(ep.EndpointStatus),
		"CreationTime":       unixFloat(ep.CreationTime),
		"LastModifiedTime":   unixFloat(ep.LastModifiedTime),
	}
	if ep.FailureReason != "" {
		resp["FailureReason"] = ep.FailureReason
	}
	return jsonOK(resp)
}

func handleListEndpoints(_ *service.RequestContext, store *Store) (*service.Response, error) {
	eps := store.ListEndpoints()
	entries := make([]map[string]any, 0, len(eps))
	for _, ep := range eps {
		entries = append(entries, map[string]any{
			"EndpointName":   ep.EndpointName,
			"EndpointArn":    ep.EndpointArn,
			"EndpointStatus": string(ep.EndpointStatus),
			"CreationTime":   unixFloat(ep.CreationTime),
		})
	}
	return jsonOK(map[string]any{"Endpoints": entries})
}

func handleDeleteEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}
	if awsErr := store.DeleteEndpoint(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "EndpointName")
	if name == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}
	configName := getStr(params, "EndpointConfigName")
	if configName == "" {
		return jsonErr(service.ErrValidation("EndpointConfigName is required."))
	}
	ep, awsErr := store.UpdateEndpoint(name, configName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"EndpointArn": ep.EndpointArn})
}

// ---- Processing Job handlers ----

func handleCreateProcessingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ProcessingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("ProcessingJobName is required."))
	}
	pj, awsErr := store.CreateProcessingJob(
		name,
		getStr(params, "RoleArn"),
		getMap(params, "AppSpecification"),
		getMap(params, "ProcessingResources"),
		getSliceOfMaps(params, "ProcessingInputs"),
		getMap(params, "ProcessingOutputConfig"),
		getMap(params, "StoppingCondition"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ProcessingJobArn": pj.ProcessingJobArn})
}

func handleDescribeProcessingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ProcessingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("ProcessingJobName is required."))
	}
	pj, awsErr := store.GetProcessingJob(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"ProcessingJobName":      pj.ProcessingJobName,
		"ProcessingJobArn":       pj.ProcessingJobArn,
		"ProcessingJobStatus":    string(pj.ProcessingJobStatus),
		"RoleArn":                pj.RoleArn,
		"AppSpecification":       pj.AppSpecification,
		"ProcessingResources":    pj.ProcessingResources,
		"ProcessingInputs":       pj.ProcessingInputs,
		"ProcessingOutputConfig": pj.ProcessingOutputConfig,
		"StoppingCondition":      pj.StoppingCondition,
		"CreationTime":           unixFloat(pj.CreationTime),
	}
	if pj.ProcessingStartTime != nil {
		resp["ProcessingStartTime"] = unixFloat(*pj.ProcessingStartTime)
	}
	if pj.ProcessingEndTime != nil {
		resp["ProcessingEndTime"] = unixFloat(*pj.ProcessingEndTime)
	}
	if pj.FailureReason != "" {
		resp["FailureReason"] = pj.FailureReason
	}
	return jsonOK(resp)
}

func handleListProcessingJobs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListProcessingJobs()
	entries := make([]map[string]any, 0, len(jobs))
	for _, pj := range jobs {
		entries = append(entries, map[string]any{
			"ProcessingJobName":   pj.ProcessingJobName,
			"ProcessingJobArn":    pj.ProcessingJobArn,
			"ProcessingJobStatus": string(pj.ProcessingJobStatus),
			"CreationTime":        unixFloat(pj.CreationTime),
		})
	}
	return jsonOK(map[string]any{"ProcessingJobSummaries": entries})
}

func handleStopProcessingJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "ProcessingJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("ProcessingJobName is required."))
	}
	if awsErr := store.StopProcessingJob(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Transform Job handlers ----

func handleCreateTransformJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TransformJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TransformJobName is required."))
	}
	tj, awsErr := store.CreateTransformJob(
		name,
		getStr(params, "ModelName"),
		getMap(params, "TransformInput"),
		getMap(params, "TransformOutput"),
		getMap(params, "TransformResources"),
		tagsFromEntries(parseTags(params)),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"TransformJobArn": tj.TransformJobArn})
}

func handleDescribeTransformJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TransformJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TransformJobName is required."))
	}
	tj, awsErr := store.GetTransformJob(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"TransformJobName":   tj.TransformJobName,
		"TransformJobArn":    tj.TransformJobArn,
		"TransformJobStatus": string(tj.TransformJobStatus),
		"ModelName":          tj.ModelName,
		"TransformInput":     tj.TransformInput,
		"TransformOutput":    tj.TransformOutput,
		"TransformResources": tj.TransformResources,
		"CreationTime":       unixFloat(tj.CreationTime),
	}
	if tj.TransformStartTime != nil {
		resp["TransformStartTime"] = unixFloat(*tj.TransformStartTime)
	}
	if tj.TransformEndTime != nil {
		resp["TransformEndTime"] = unixFloat(*tj.TransformEndTime)
	}
	if tj.FailureReason != "" {
		resp["FailureReason"] = tj.FailureReason
	}
	return jsonOK(resp)
}

func handleListTransformJobs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListTransformJobs()
	entries := make([]map[string]any, 0, len(jobs))
	for _, tj := range jobs {
		entries = append(entries, map[string]any{
			"TransformJobName":   tj.TransformJobName,
			"TransformJobArn":    tj.TransformJobArn,
			"TransformJobStatus": string(tj.TransformJobStatus),
			"CreationTime":       unixFloat(tj.CreationTime),
		})
	}
	return jsonOK(map[string]any{"TransformJobSummaries": entries})
}

func handleStopTransformJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(params, "TransformJobName")
	if name == "" {
		return jsonErr(service.ErrValidation("TransformJobName is required."))
	}
	if awsErr := store.StopTransformJob(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Tag handlers ----

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

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := tagsFromEntries(parseTags(params))
	if awsErr := store.AddTags(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": entriesToTags(tags)})
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tagKeys := getStringSlice(params, "TagKeys")
	if awsErr := store.DeleteTags(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(params, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, awsErr := store.ListTags(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": entriesToTags(tags)})
}

// ---- InvokeEndpoint ----

func handleInvokeEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	endpointName := getStr(params, "EndpointName")
	if endpointName == "" {
		return jsonErr(service.ErrValidation("EndpointName is required."))
	}

	ep, awsErr := store.GetEndpoint(endpointName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	status := EndpointStatus(ep.Lifecycle.State())
	if status != EndpointInService {
		return jsonErr(service.NewAWSError("ValidationError",
			"Endpoint is not InService", http.StatusBadRequest))
	}

	// Return a mock prediction response echoing the input shape with random scores.
	body := params["Body"]
	contentType := getStr(params, "ContentType")
	if contentType == "" {
		contentType = "application/json"
	}

	var responseBody any
	if body != nil {
		// Echo input with prediction scores appended.
		responseBody = map[string]any{
			"predictions": []map[string]any{
				{"score": 0.95, "label": "positive"},
				{"score": 0.05, "label": "negative"},
			},
			"input": body,
		}
	} else {
		responseBody = map[string]any{
			"predictions": []map[string]any{
				{"score": 0.95, "label": "positive"},
				{"score": 0.05, "label": "negative"},
			},
		}
	}

	return jsonOK(map[string]any{
		"Body":        responseBody,
		"ContentType": contentType,
		"InvokedProductionVariant": "AllTraffic",
	})
}
