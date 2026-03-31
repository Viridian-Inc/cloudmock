package fis

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func str(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func tagsMap(params map[string]any) map[string]string {
	tags := make(map[string]string)
	if v, ok := params["tags"].(map[string]any); ok {
		for k, val := range v {
			if sv, ok := val.(string); ok {
				tags[k] = sv
			}
		}
	}
	return tags
}

func templateResponse(t *ExperimentTemplate) map[string]any {
	resp := map[string]any{
		"id":             t.ID,
		"description":    t.Description,
		"roleArn":        t.RoleArn,
		"tags":           t.Tags,
		"creationTime":   t.CreationTime.Format(time.RFC3339),
		"lastUpdateTime": t.LastUpdateTime.Format(time.RFC3339),
	}

	targets := make(map[string]any)
	for k, v := range t.Targets {
		targets[k] = map[string]any{
			"resourceType":  v.ResourceType,
			"resourceArns":  v.ResourceArns,
			"selectionMode": v.SelectionMode,
		}
	}
	resp["targets"] = targets

	actions := make(map[string]any)
	for k, v := range t.Actions {
		actions[k] = map[string]any{
			"actionId":    v.ActionID,
			"description": v.Description,
			"parameters":  v.Parameters,
			"targets":     v.Targets,
			"startAfter":  v.StartAfter,
		}
	}
	resp["actions"] = actions

	return resp
}

func experimentResponse(e *Experiment) map[string]any {
	resp := map[string]any{
		"id":           e.ID,
		"experimentTemplateId": e.TemplateID,
		"roleArn":      e.RoleArn,
		"state": map[string]any{
			"status": e.State,
			"reason": e.StateReason,
		},
		"tags":         e.Tags,
		"creationTime": e.CreationTime.Format(time.RFC3339),
	}
	if e.StartTime != nil {
		resp["startTime"] = e.StartTime.Format(time.RFC3339)
	}
	if e.EndTime != nil {
		resp["endTime"] = e.EndTime.Format(time.RFC3339)
	}
	return resp
}

func handleCreateExperimentTemplate(params map[string]any, store *Store) (*service.Response, error) {
	desc := str(params, "description")
	roleArn := str(params, "roleArn")

	targets := make(map[string]ExperimentTarget)
	if tMap, ok := params["targets"].(map[string]any); ok {
		for k, v := range tMap {
			if tm, ok := v.(map[string]any); ok {
				t := ExperimentTarget{
					ResourceType:  str(tm, "resourceType"),
					SelectionMode: str(tm, "selectionMode"),
				}
				if arns, ok := tm["resourceArns"].([]any); ok {
					for _, a := range arns {
						if s, ok := a.(string); ok {
							t.ResourceArns = append(t.ResourceArns, s)
						}
					}
				}
				targets[k] = t
			}
		}
	}

	actions := make(map[string]ExperimentAction)
	if aMap, ok := params["actions"].(map[string]any); ok {
		for k, v := range aMap {
			if am, ok := v.(map[string]any); ok {
				a := ExperimentAction{
					ActionID:    str(am, "actionId"),
					Description: str(am, "description"),
				}
				if p, ok := am["parameters"].(map[string]any); ok {
					a.Parameters = make(map[string]string)
					for pk, pv := range p {
						if sv, ok := pv.(string); ok {
							a.Parameters[pk] = sv
						}
					}
				}
				if t, ok := am["targets"].(map[string]any); ok {
					a.Targets = make(map[string]string)
					for tk, tv := range t {
						if sv, ok := tv.(string); ok {
							a.Targets[tk] = sv
						}
					}
				}
				actions[k] = a
			}
		}
	}

	var stopConditions []StopCondition
	if scs, ok := params["stopConditions"].([]any); ok {
		for _, sc := range scs {
			if sm, ok := sc.(map[string]any); ok {
				stopConditions = append(stopConditions, StopCondition{
					Source: str(sm, "source"),
					Value:  str(sm, "value"),
				})
			}
		}
	}

	tmpl, _ := store.CreateExperimentTemplate(desc, roleArn, targets, actions, stopConditions, tagsMap(params))
	return jsonOK(map[string]any{"experimentTemplate": templateResponse(tmpl)})
}

func handleGetExperimentTemplate(id string, store *Store) (*service.Response, error) {
	tmpl, ok := store.GetExperimentTemplate(id)
	if !ok {
		return jsonErr(service.ErrNotFound("ExperimentTemplate", id))
	}
	return jsonOK(map[string]any{"experimentTemplate": templateResponse(tmpl)})
}

func handleListExperimentTemplates(store *Store) (*service.Response, error) {
	templates := store.ListExperimentTemplates()
	out := make([]map[string]any, 0, len(templates))
	for _, t := range templates {
		out = append(out, map[string]any{
			"id":          t.ID,
			"description": t.Description,
			"tags":        t.Tags,
		})
	}
	return jsonOK(map[string]any{"experimentTemplates": out})
}

func handleDeleteExperimentTemplate(id string, store *Store) (*service.Response, error) {
	if !store.DeleteExperimentTemplate(id) {
		return jsonErr(service.ErrNotFound("ExperimentTemplate", id))
	}
	return jsonNoContent()
}

func handleStartExperiment(params map[string]any, store *Store) (*service.Response, error) {
	templateID := str(params, "experimentTemplateId")
	if templateID == "" {
		return jsonErr(service.ErrValidation("experimentTemplateId is required"))
	}
	exp, err := store.StartExperiment(templateID, tagsMap(params))
	if err != nil {
		return jsonErr(service.ErrNotFound("ExperimentTemplate", templateID))
	}
	return jsonOK(map[string]any{"experiment": experimentResponse(exp)})
}

func handleGetExperiment(id string, store *Store) (*service.Response, error) {
	exp, ok := store.GetExperiment(id)
	if !ok {
		return jsonErr(service.ErrNotFound("Experiment", id))
	}
	return jsonOK(map[string]any{"experiment": experimentResponse(exp)})
}

func handleListExperiments(store *Store) (*service.Response, error) {
	experiments := store.ListExperiments()
	out := make([]map[string]any, 0, len(experiments))
	for _, e := range experiments {
		out = append(out, experimentResponse(e))
	}
	return jsonOK(map[string]any{"experiments": out})
}

func handleStopExperiment(id string, store *Store) (*service.Response, error) {
	exp, err := store.StopExperiment(id)
	if err != nil {
		return jsonErr(service.ErrNotFound("Experiment", id))
	}
	return jsonOK(map[string]any{"experiment": experimentResponse(exp)})
}
