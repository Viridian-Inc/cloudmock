package codepipeline

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

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

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInputException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStrSlice(m map[string]any, key string) []string {
	arr, ok := m[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseTagsList(tags []any) map[string]string {
	m := make(map[string]string)
	for _, t := range tags {
		if tm, ok := t.(map[string]any); ok {
			k := getStr(tm, "key")
			v := getStr(tm, "value")
			if k != "" {
				m[k] = v
			}
		}
	}
	return m
}

func tagsToList(m map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for k, v := range m {
		out = append(out, map[string]any{"key": k, "value": v})
	}
	return out
}

// ---- Pipeline parsing helpers ----

func parsePipeline(m map[string]any) *Pipeline {
	p := &Pipeline{
		Name:    getStr(m, "name"),
		RoleARN: getStr(m, "roleArn"),
	}

	if stages, ok := m["stages"].([]any); ok {
		for _, s := range stages {
			if sm, ok := s.(map[string]any); ok {
				stage := StageDeclaration{
					Name: getStr(sm, "name"),
				}
				if actions, ok := sm["actions"].([]any); ok {
					for _, a := range actions {
						if am, ok := a.(map[string]any); ok {
							action := ActionDeclaration{
								Name: getStr(am, "name"),
							}
							if atid, ok := am["actionTypeId"].(map[string]any); ok {
								action.ActionTypeID = ActionTypeID{
									Category: getStr(atid, "category"),
									Owner:    getStr(atid, "owner"),
									Provider: getStr(atid, "provider"),
									Version:  getStr(atid, "version"),
								}
							}
							if cfg, ok := am["configuration"].(map[string]any); ok {
								action.Configuration = make(map[string]string)
								for k, v := range cfg {
									if sv, ok := v.(string); ok {
										action.Configuration[k] = sv
									}
								}
							}
							if inputs, ok := am["inputArtifacts"].([]any); ok {
								for _, inp := range inputs {
									if im, ok := inp.(map[string]any); ok {
										action.InputArtifacts = append(action.InputArtifacts, ArtifactRef{Name: getStr(im, "name")})
									}
								}
							}
							if outputs, ok := am["outputArtifacts"].([]any); ok {
								for _, out := range outputs {
									if om, ok := out.(map[string]any); ok {
										action.OutputArtifacts = append(action.OutputArtifacts, ArtifactRef{Name: getStr(om, "name")})
									}
								}
							}
							if ro, ok := am["runOrder"].(float64); ok {
								action.RunOrder = int(ro)
							}
							stage.Actions = append(stage.Actions, action)
						}
					}
				}
				p.Stages = append(p.Stages, stage)
			}
		}
	}
	return p
}

func pipelineToMap(p *Pipeline) map[string]any {
	stages := make([]map[string]any, len(p.Stages))
	for i, stage := range p.Stages {
		actions := make([]map[string]any, len(stage.Actions))
		for j, action := range stage.Actions {
			am := map[string]any{
				"name": action.Name,
				"actionTypeId": map[string]any{
					"category": action.ActionTypeID.Category,
					"owner":    action.ActionTypeID.Owner,
					"provider": action.ActionTypeID.Provider,
					"version":  action.ActionTypeID.Version,
				},
				"runOrder": action.RunOrder,
			}
			if action.Configuration != nil {
				am["configuration"] = action.Configuration
			}
			if len(action.InputArtifacts) > 0 {
				inputs := make([]map[string]any, len(action.InputArtifacts))
				for k, inp := range action.InputArtifacts {
					inputs[k] = map[string]any{"name": inp.Name}
				}
				am["inputArtifacts"] = inputs
			}
			if len(action.OutputArtifacts) > 0 {
				outputs := make([]map[string]any, len(action.OutputArtifacts))
				for k, out := range action.OutputArtifacts {
					outputs[k] = map[string]any{"name": out.Name}
				}
				am["outputArtifacts"] = outputs
			}
			actions[j] = am
		}
		stages[i] = map[string]any{
			"name":    stage.Name,
			"actions": actions,
		}
	}

	return map[string]any{
		"name":    p.Name,
		"roleArn": p.RoleARN,
		"version": p.Version,
		"stages":  stages,
	}
}

func executionToMap(e *PipelineExecution) map[string]any {
	m := map[string]any{
		"pipelineExecutionId": e.ID,
		"pipelineName":        e.PipelineName,
		"pipelineVersion":     e.PipelineVersion,
		"status":              e.Status,
		"startTime":           float64(e.StartTime.Unix()),
	}
	if e.StatusSummary != "" {
		m["statusSummary"] = e.StatusSummary
	}
	if e.EndTime != nil {
		m["lastUpdateTime"] = float64(e.EndTime.Unix())
	}
	return m
}

// ---- Pipeline handlers ----

func handleCreatePipeline(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineMap, ok := req["pipeline"].(map[string]any)
	if !ok {
		return jsonErr(service.ErrValidation("pipeline is required."))
	}

	p := parsePipeline(pipelineMap)

	if tags, ok := req["tags"].([]any); ok {
		p.Tags = parseTagsList(tags)
	}

	created, awsErr := store.CreatePipeline(p)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"pipeline": pipelineToMap(created),
		"tags":     tagsToList(created.Tags),
	})
}

func handleGetPipeline(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Pipeline name is required."))
	}

	p, awsErr := store.GetPipeline(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"pipeline": pipelineToMap(p),
		"metadata": map[string]any{
			"pipelineArn": p.ARN,
			"created":     float64(p.CreatedAt.Unix()),
			"updated":     float64(p.UpdatedAt.Unix()),
		},
	})
}

func handleListPipelines(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	pipelines := store.ListPipelines()

	result := make([]map[string]any, len(pipelines))
	for i, p := range pipelines {
		result[i] = map[string]any{
			"name":    p.Name,
			"version": p.Version,
			"created": float64(p.CreatedAt.Unix()),
			"updated": float64(p.UpdatedAt.Unix()),
		}
	}

	return jsonOK(map[string]any{"pipelines": result})
}

func handleUpdatePipeline(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineMap, ok := req["pipeline"].(map[string]any)
	if !ok {
		return jsonErr(service.ErrValidation("pipeline is required."))
	}

	p := parsePipeline(pipelineMap)
	updated, awsErr := store.UpdatePipeline(p)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"pipeline": pipelineToMap(updated)})
}

func handleDeletePipeline(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Pipeline name is required."))
	}

	if awsErr := store.DeletePipeline(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Execution handlers ----

func handleGetPipelineState(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	p, execs, awsErr := store.GetPipelineState(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	stageStates := make([]map[string]any, len(p.Stages))
	for i, stage := range p.Stages {
		ss := map[string]any{
			"stageName": stage.Name,
		}
		// Find latest execution stage state
		if len(execs) > 0 {
			latest := execs[len(execs)-1]
			for _, es := range latest.StageStates {
				if es.StageName == stage.Name {
					ss["latestExecution"] = map[string]any{
						"pipelineExecutionId": latest.ID,
						"status":              es.Status,
					}
					actionStates := make([]map[string]any, len(es.ActionStates))
					for j, as := range es.ActionStates {
						actionStates[j] = map[string]any{
							"actionName": as.ActionName,
							"latestExecution": map[string]any{
								"status":     as.Status,
								"summary":    as.Summary,
								"lastUpdate": float64(as.LastUpdate.Unix()),
							},
						}
					}
					ss["actionStates"] = actionStates
					break
				}
			}
		}
		stageStates[i] = ss
	}

	return jsonOK(map[string]any{
		"pipelineName":    p.Name,
		"pipelineVersion": p.Version,
		"stageStates":     stageStates,
		"created":         float64(p.CreatedAt.Unix()),
		"updated":         float64(p.UpdatedAt.Unix()),
	})
}

func handleGetPipelineExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineName := getStr(req, "pipelineName")
	executionID := getStr(req, "pipelineExecutionId")

	exec, awsErr := store.GetPipelineExecution(pipelineName, executionID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"pipelineExecution": executionToMap(exec)})
}

func handleListPipelineExecutions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineName := getStr(req, "pipelineName")
	execs, awsErr := store.ListPipelineExecutions(pipelineName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(execs))
	for i, e := range execs {
		result[i] = executionToMap(e)
	}

	return jsonOK(map[string]any{"pipelineExecutionSummaries": result})
}

func handleStartPipelineExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Pipeline name is required."))
	}

	exec, awsErr := store.StartPipelineExecution(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"pipelineExecutionId": exec.ID})
}

func handleStopPipelineExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineName := getStr(req, "pipelineName")
	executionID := getStr(req, "pipelineExecutionId")
	reason := getStr(req, "reason")
	abandon, _ := req["abandon"].(bool)

	exec, awsErr := store.StopPipelineExecution(pipelineName, executionID, reason, abandon)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"pipelineExecutionId": exec.ID})
}

func handlePutApprovalResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineName := getStr(req, "pipelineName")
	stageName := getStr(req, "stageName")
	actionName := getStr(req, "actionName")

	var result ApprovalResult
	if r, ok := req["result"].(map[string]any); ok {
		result.Summary = getStr(r, "summary")
		result.Status = getStr(r, "status")
	}

	if awsErr := store.PutApprovalResult(pipelineName, stageName, actionName, result); awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"approvedAt": float64(0)})
}

func handleRetryStageExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pipelineName := getStr(req, "pipelineName")
	stageName := getStr(req, "stageName")
	executionID := getStr(req, "pipelineExecutionId")

	exec, awsErr := store.RetryStageExecution(pipelineName, stageName, executionID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"pipelineExecutionId": exec.ID})
}

// ---- Webhook handlers ----

func handlePutWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	whMap, ok := req["webhook"].(map[string]any)
	if !ok {
		return jsonErr(service.ErrValidation("webhook is required."))
	}

	name := getStr(whMap, "name")
	targetPipeline := getStr(whMap, "targetPipeline")
	targetAction := getStr(whMap, "targetAction")
	authentication := getStr(whMap, "authentication")

	var filters []WebhookFilter
	if filterList, ok := whMap["filters"].([]any); ok {
		for _, f := range filterList {
			if fm, ok := f.(map[string]any); ok {
				filters = append(filters, WebhookFilter{
					JSONPath:    getStr(fm, "jsonPath"),
					MatchEquals: getStr(fm, "matchEquals"),
				})
			}
		}
	}

	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	wh, awsErr := store.PutWebhook(name, targetPipeline, targetAction, authentication, filters, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"webhook": webhookToMap(wh)})
}

func handleListWebhooks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	webhooks := store.ListWebhooks()
	result := make([]map[string]any, len(webhooks))
	for i, wh := range webhooks {
		result[i] = map[string]any{
			"definition": webhookToMap(wh),
			"url":        wh.URL,
		}
	}
	return jsonOK(map[string]any{"webhooks": result})
}

func handleDeleteWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Webhook name is required."))
	}

	if awsErr := store.DeleteWebhook(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func webhookToMap(wh *Webhook) map[string]any {
	filters := make([]map[string]any, len(wh.Filters))
	for i, f := range wh.Filters {
		filters[i] = map[string]any{
			"jsonPath":    f.JSONPath,
			"matchEquals": f.MatchEquals,
		}
	}
	return map[string]any{
		"name":           wh.Name,
		"targetPipeline": wh.TargetPipeline,
		"targetAction":   wh.TargetAction,
		"authentication": wh.Authentication,
		"filters":        filters,
		"url":            wh.URL,
		"arn":            wh.ARN,
	}
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}

	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	keys := getStrSlice(req, "tagKeys")

	if awsErr := store.UntagResource(arn, keys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	tags := store.ListTagsForResource(arn)

	return jsonOK(map[string]any{"tags": tagsToList(tags)})
}
