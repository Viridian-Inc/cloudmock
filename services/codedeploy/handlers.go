package codedeploy

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
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
			k := getStr(tm, "Key")
			v := getStr(tm, "Value")
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
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

// ---- Application handlers ----

func handleCreateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "applicationName")
	platform := getStr(req, "computePlatform")
	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	app, awsErr := store.CreateApplication(name, platform, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"applicationId": app.ID})
}

func handleGetApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "applicationName")
	if name == "" {
		return jsonErr(service.ErrValidation("applicationName is required."))
	}

	app, awsErr := store.GetApplication(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"application": map[string]any{
			"applicationId":   app.ID,
			"applicationName": app.Name,
			"computePlatform": app.ComputePlatform,
			"createTime":      float64(app.CreatedAt.Unix()),
		},
	})
}

func handleListApplications(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	names := store.ListApplications()
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"applications": names})
}

func handleDeleteApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "applicationName")
	if name == "" {
		return jsonErr(service.ErrValidation("applicationName is required."))
	}

	if awsErr := store.DeleteApplication(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Deployment Group handlers ----

func handleCreateDeploymentGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	group := &DeploymentGroup{
		Name:                 getStr(req, "deploymentGroupName"),
		ServiceRoleARN:       getStr(req, "serviceRoleArn"),
		DeploymentConfigName: getStr(req, "deploymentConfigName"),
	}

	if style, ok := req["deploymentStyle"].(map[string]any); ok {
		group.DeploymentStyle = DeploymentStyle{
			DeploymentType:   getStr(style, "deploymentType"),
			DeploymentOption: getStr(style, "deploymentOption"),
		}
	}

	if filters, ok := req["ec2TagFilters"].([]any); ok {
		for _, f := range filters {
			if fm, ok := f.(map[string]any); ok {
				group.Ec2TagFilters = append(group.Ec2TagFilters, EC2TagFilter{
					Key:   getStr(fm, "Key"),
					Value: getStr(fm, "Value"),
					Type:  getStr(fm, "Type"),
				})
			}
		}
	}

	created, awsErr := store.CreateDeploymentGroup(appName, group)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"deploymentGroupId": created.ID})
}

func handleGetDeploymentGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	groupName := getStr(req, "deploymentGroupName")

	group, awsErr := store.GetDeploymentGroup(appName, groupName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	filters := make([]map[string]any, len(group.Ec2TagFilters))
	for i, f := range group.Ec2TagFilters {
		filters[i] = map[string]any{"Key": f.Key, "Value": f.Value, "Type": f.Type}
	}

	return jsonOK(map[string]any{
		"deploymentGroupInfo": map[string]any{
			"deploymentGroupId":    group.ID,
			"deploymentGroupName":  group.Name,
			"applicationName":      group.ApplicationName,
			"serviceRoleArn":       group.ServiceRoleARN,
			"deploymentConfigName": group.DeploymentConfigName,
			"ec2TagFilters":        filters,
			"deploymentStyle": map[string]any{
				"deploymentType":   group.DeploymentStyle.DeploymentType,
				"deploymentOption": group.DeploymentStyle.DeploymentOption,
			},
		},
	})
}

func handleListDeploymentGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	names, awsErr := store.ListDeploymentGroups(appName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{
		"applicationName":  appName,
		"deploymentGroups": names,
	})
}

func handleDeleteDeploymentGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	groupName := getStr(req, "deploymentGroupName")

	if awsErr := store.DeleteDeploymentGroup(appName, groupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUpdateDeploymentGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	groupName := getStr(req, "currentDeploymentGroupName")

	group, awsErr := store.UpdateDeploymentGroup(appName, groupName, req)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"hooksNotCleanedUp": []any{},
		"deploymentGroupId": group.ID,
	})
}

// ---- Deployment handlers ----

func handleCreateDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	groupName := getStr(req, "deploymentGroupName")
	configName := getStr(req, "deploymentConfigName")
	description := getStr(req, "description")

	var revision RevisionLocation
	if rev, ok := req["revision"].(map[string]any); ok {
		revision.RevisionType = getStr(rev, "revisionType")
		if s3, ok := rev["s3Location"].(map[string]any); ok {
			revision.S3Location = &S3Location{
				Bucket:     getStr(s3, "bucket"),
				Key:        getStr(s3, "key"),
				BundleType: getStr(s3, "bundleType"),
				Version:    getStr(s3, "version"),
			}
		}
		if gh, ok := rev["gitHubLocation"].(map[string]any); ok {
			revision.GitHubLocation = &GitHubLocation{
				Repository: getStr(gh, "repository"),
				CommitID:   getStr(gh, "commitId"),
			}
		}
	}

	d, awsErr := store.CreateDeployment(appName, groupName, configName, description, revision)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"deploymentId": d.ID})
}

func handleGetDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	id := getStr(req, "deploymentId")
	d, awsErr := store.GetDeployment(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"deploymentInfo": deploymentToMap(d)})
}

func handleListDeployments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	appName := getStr(req, "applicationName")
	groupName := getStr(req, "deploymentGroupName")

	var status string
	if filters, ok := req["includeOnlyStatuses"].([]any); ok && len(filters) > 0 {
		if s, ok := filters[0].(string); ok {
			status = s
		}
	}

	ids := store.ListDeployments(appName, groupName, status)
	if ids == nil {
		ids = []string{}
	}
	return jsonOK(map[string]any{"deployments": ids})
}

func handleStopDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	id := getStr(req, "deploymentId")
	d, awsErr := store.StopDeployment(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"status":        d.Status,
		"statusMessage": "Deployment stopped.",
	})
}

func handleBatchGetDeployments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	ids := getStrSlice(req, "deploymentIds")
	deployments, awsErr := store.BatchGetDeployments(ids)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(deployments))
	for i, d := range deployments {
		result[i] = deploymentToMap(d)
	}
	return jsonOK(map[string]any{"deploymentsInfo": result})
}

func handleBatchGetDeploymentTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	deploymentID := getStr(req, "deploymentId")
	targetIDs := getStrSlice(req, "targetIds")

	targets, awsErr := store.BatchGetDeploymentTargets(deploymentID, targetIDs)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(targets))
	for i, t := range targets {
		m := map[string]any{
			"deploymentTargetType": t.DeploymentTargetType,
		}
		if t.InstanceTarget != nil {
			m["instanceTarget"] = map[string]any{
				"deploymentId":  t.InstanceTarget.DeploymentID,
				"targetId":      t.InstanceTarget.TargetID,
				"targetArn":     t.InstanceTarget.TargetARN,
				"status":        t.InstanceTarget.Status,
				"lastUpdatedAt": float64(t.InstanceTarget.LastUpdatedAt.Unix()),
			}
		}
		result[i] = m
	}
	return jsonOK(map[string]any{"deploymentTargets": result})
}

// ---- On-Premises Instance Tag handlers ----

func handleAddTagsToOnPremisesInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	names := getStrSlice(req, "instanceNames")
	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	if awsErr := store.AddTagsToOnPremisesInstances(names, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleRemoveTagsFromOnPremisesInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	names := getStrSlice(req, "instanceNames")
	var tagKeys []string
	if tagList, ok := req["tags"].([]any); ok {
		for _, t := range tagList {
			if tm, ok := t.(map[string]any); ok {
				if k, ok := tm["Key"].(string); ok {
					tagKeys = append(tagKeys, k)
				}
			}
		}
	}

	if awsErr := store.RemoveTagsFromOnPremisesInstances(names, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- conversion helpers ----

func deploymentToMap(d *Deployment) map[string]any {
	m := map[string]any{
		"deploymentId":         d.ID,
		"applicationName":      d.ApplicationName,
		"deploymentGroupName":  d.DeploymentGroupName,
		"deploymentConfigName": d.DeploymentConfigName,
		"status":               d.Status,
		"description":          d.Description,
		"creator":              d.Creator,
		"createTime":           float64(d.CreateTime.Unix()),
	}

	rev := map[string]any{"revisionType": d.Revision.RevisionType}
	if d.Revision.S3Location != nil {
		rev["s3Location"] = map[string]any{
			"bucket":     d.Revision.S3Location.Bucket,
			"key":        d.Revision.S3Location.Key,
			"bundleType": d.Revision.S3Location.BundleType,
			"version":    d.Revision.S3Location.Version,
		}
	}
	if d.Revision.GitHubLocation != nil {
		rev["gitHubLocation"] = map[string]any{
			"repository": d.Revision.GitHubLocation.Repository,
			"commitId":   d.Revision.GitHubLocation.CommitID,
		}
	}
	m["revision"] = rev

	if d.StartTime != nil {
		m["startTime"] = float64(d.StartTime.Unix())
	}
	if d.CompleteTime != nil {
		m["completeTime"] = float64(d.CompleteTime.Unix())
	}
	return m
}
