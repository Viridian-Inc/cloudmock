package fis

import (
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
		"id":                   e.ID,
		"experimentTemplateId": e.TemplateID,
		"roleArn":              e.RoleArn,
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

	// Include action states.
	if len(e.Actions) > 0 {
		actions := make(map[string]any)
		for name, as := range e.Actions {
			entry := map[string]any{
				"actionId": as.ActionID,
				"state":    map[string]any{"status": as.State},
			}
			if as.Description != "" {
				entry["description"] = as.Description
			}
			if as.StartTime != nil {
				entry["startTime"] = as.StartTime.Format(time.RFC3339)
			}
			if as.EndTime != nil {
				entry["endTime"] = as.EndTime.Format(time.RFC3339)
			}
			actions[name] = entry
		}
		resp["actions"] = actions
	}

	// Include targets.
	if len(e.Targets) > 0 {
		targets := make(map[string]any)
		for k, v := range e.Targets {
			targets[k] = map[string]any{
				"resourceType":  v.ResourceType,
				"resourceArns":  v.ResourceArns,
				"selectionMode": v.SelectionMode,
			}
		}
		resp["targets"] = targets
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

func handleUpdateExperimentTemplate(id string, params map[string]any, store *Store) (*service.Response, error) {
	description := str(params, "description")
	roleArn := str(params, "roleArn")
	tags := tagsMap(params)
	tmpl, ok := store.UpdateExperimentTemplate(id, description, roleArn, tags)
	if !ok {
		return jsonErr(service.ErrNotFound("ExperimentTemplate", id))
	}
	return jsonOK(map[string]any{"experimentTemplate": templateResponse(tmpl)})
}

func handleDeleteExperimentTemplate(id string, store *Store) (*service.Response, error) {
	if !store.DeleteExperimentTemplate(id) {
		return jsonErr(service.ErrNotFound("ExperimentTemplate", id))
	}
	return jsonNoContent()
}

func handleTagResource(arn string, params map[string]any, store *Store) (*service.Response, error) {
	tags := tagsMap(params)
	if !store.TagResource(arn, tags) {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(arn string, keys []string, store *Store) (*service.Response, error) {
	if !store.UntagResource(arn, keys) {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonNoContent()
}

func handleListTagsForResource(arn string, store *Store) (*service.Response, error) {
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonOK(map[string]any{"tags": tags})
}

func handleListTargetResourceTypes(store *Store) (*service.Response, error) {
	// Return well-known FIS target resource types.
	types := []map[string]any{
		{"resourceType": "aws:ec2:instance", "description": "AWS EC2 instances"},
		{"resourceType": "aws:ec2:spot-instance", "description": "AWS EC2 Spot instances"},
		{"resourceType": "aws:ecs:task", "description": "AWS ECS tasks"},
		{"resourceType": "aws:eks:nodegroup", "description": "AWS EKS node groups"},
		{"resourceType": "aws:rds:cluster", "description": "AWS RDS clusters"},
		{"resourceType": "aws:rds:db", "description": "AWS RDS DB instances"},
		{"resourceType": "aws:ssm:managed-instance", "description": "AWS SSM managed instances"},
	}
	return jsonOK(map[string]any{"targetResourceTypes": types})
}

func handleListActions(store *Store) (*service.Response, error) {
	// Return well-known FIS actions.
	actions := []map[string]any{
		{"id": "aws:ec2:stop-instances", "description": "Stop EC2 instances"},
		{"id": "aws:ec2:terminate-instances", "description": "Terminate EC2 instances"},
		{"id": "aws:ec2:reboot-instances", "description": "Reboot EC2 instances"},
		{"id": "aws:ecs:drain-container-instances", "description": "Drain ECS container instances"},
		{"id": "aws:eks:terminate-nodegroup-instances", "description": "Terminate EKS nodegroup instances"},
		{"id": "aws:rds:failover-db-cluster", "description": "Failover RDS cluster"},
		{"id": "aws:ssm:send-command", "description": "Send SSM command"},
	}
	return jsonOK(map[string]any{"actions": actions})
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
