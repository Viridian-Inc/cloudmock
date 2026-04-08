package codebuild

import (
	gojson "github.com/goccy/go-json"
	"net/http"

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

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInputException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- Project handlers ----

func handleCreateProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	p := &Project{
		Name:        getStr(req, "name"),
		Description: getStr(req, "description"),
		ServiceRole: getStr(req, "serviceRole"),
	}

	if src, ok := req["source"].(map[string]any); ok {
		p.Source = ProjectSource{
			Type:     getStr(src, "type"),
			Location: getStr(src, "location"),
		}
	}
	if art, ok := req["artifacts"].(map[string]any); ok {
		p.Artifacts = ProjectArtifacts{
			Type:     getStr(art, "type"),
			Location: getStr(art, "location"),
		}
	}
	if env, ok := req["environment"].(map[string]any); ok {
		p.Environment = parseEnvironment(env)
	}
	if timeout, ok := req["timeoutInMinutes"].(float64); ok {
		p.TimeoutInMins = int(timeout)
	}
	if tags, ok := req["tags"].([]any); ok {
		p.Tags = parseTagsList(tags)
	}

	created, awsErr := store.CreateProject(p)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"project": projectToMap(created)})
}

func handleBatchGetProjects(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	names := getStrSlice(req, "names")
	found, notFound := store.BatchGetProjects(names)

	projects := make([]map[string]any, len(found))
	for i, p := range found {
		projects[i] = projectToMap(p)
	}

	return jsonOK(map[string]any{
		"projects":         projects,
		"projectsNotFound": notFound,
	})
}

func handleListProjects(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	names := store.ListProjects()
	if names == nil {
		names = []string{}
	}
	return jsonOK(map[string]any{"projects": names})
}

func handleUpdateProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Project name is required."))
	}

	updated, awsErr := store.UpdateProject(name, req)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"project": projectToMap(updated)})
}

func handleDeleteProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("Project name is required."))
	}

	if awsErr := store.DeleteProject(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Build handlers ----

func handleStartBuild(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	projectName := getStr(req, "projectName")
	if projectName == "" {
		return jsonErr(service.ErrValidation("projectName is required."))
	}

	var envOverrides *ProjectEnvironment
	if envMap, ok := req["environmentVariablesOverride"].([]any); ok && len(envMap) > 0 {
		envOverrides = &ProjectEnvironment{}
		for _, ev := range envMap {
			if m, ok := ev.(map[string]any); ok {
				envOverrides.EnvironmentVars = append(envOverrides.EnvironmentVars, EnvironmentVariable{
					Name:  getStr(m, "name"),
					Value: getStr(m, "value"),
					Type:  getStr(m, "type"),
				})
			}
		}
	}

	b, awsErr := store.StartBuild(projectName, envOverrides)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"build": buildToMap(b)})
}

func handleBatchGetBuilds(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	ids := getStrSlice(req, "ids")
	found, notFound := store.BatchGetBuilds(ids)

	builds := make([]map[string]any, len(found))
	for i, b := range found {
		builds[i] = buildToMap(b)
	}

	return jsonOK(map[string]any{
		"builds":         builds,
		"buildsNotFound": notFound,
	})
}

func handleListBuilds(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	ids := store.ListBuilds()
	if ids == nil {
		ids = []string{}
	}
	return jsonOK(map[string]any{"ids": ids})
}

func handleListBuildsForProject(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	projectName := getStr(req, "projectName")
	if projectName == "" {
		return jsonErr(service.ErrValidation("projectName is required."))
	}

	ids := store.ListBuildsForProject(projectName)
	return jsonOK(map[string]any{"ids": ids})
}

func handleStopBuild(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	id := getStr(req, "id")
	if id == "" {
		return jsonErr(service.ErrValidation("Build ID is required."))
	}

	b, awsErr := store.StopBuild(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"build": buildToMap(b)})
}

// ---- Report Group handlers ----

func handleCreateReportGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "name")
	reportType := getStr(req, "type")
	if reportType == "" {
		reportType = "TEST"
	}

	var exportCfg ReportExportConfig
	if ec, ok := req["exportConfig"].(map[string]any); ok {
		exportCfg.ExportConfigType = getStr(ec, "exportConfigType")
		if s3, ok := ec["s3Destination"].(map[string]any); ok {
			exportCfg.S3Destination = &S3ReportExportConfig{
				Bucket: getStr(s3, "bucket"),
				Path:   getStr(s3, "path"),
			}
		}
	}

	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	rg, awsErr := store.CreateReportGroup(name, reportType, exportCfg, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"reportGroup": reportGroupToMap(rg)})
}

func handleBatchGetReportGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arns := getStrSlice(req, "reportGroupArns")
	found, notFound := store.BatchGetReportGroups(arns)

	groups := make([]map[string]any, len(found))
	for i, rg := range found {
		groups[i] = reportGroupToMap(rg)
	}

	return jsonOK(map[string]any{
		"reportGroups":         groups,
		"reportGroupsNotFound": notFound,
	})
}

func handleListReportGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	arns := store.ListReportGroups()
	if arns == nil {
		arns = []string{}
	}
	return jsonOK(map[string]any{"reportGroups": arns})
}

func handleDeleteReportGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("Report group ARN is required."))
	}

	// Extract name from ARN for lookup
	name := arn
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == '/' {
			name = arn[i+1:]
			break
		}
	}

	if awsErr := store.DeleteReportGroup(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- conversion helpers ----

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

func parseEnvironment(env map[string]any) ProjectEnvironment {
	pe := ProjectEnvironment{
		Type:        getStr(env, "type"),
		Image:       getStr(env, "image"),
		ComputeType: getStr(env, "computeType"),
	}
	if vars, ok := env["environmentVariables"].([]any); ok {
		for _, v := range vars {
			if m, ok := v.(map[string]any); ok {
				pe.EnvironmentVars = append(pe.EnvironmentVars, EnvironmentVariable{
					Name:  getStr(m, "name"),
					Value: getStr(m, "value"),
					Type:  getStr(m, "type"),
				})
			}
		}
	}
	return pe
}

func projectToMap(p *Project) map[string]any {
	envVars := make([]map[string]any, len(p.Environment.EnvironmentVars))
	for i, ev := range p.Environment.EnvironmentVars {
		envVars[i] = map[string]any{"name": ev.Name, "value": ev.Value, "type": ev.Type}
	}

	return map[string]any{
		"name":        p.Name,
		"arn":         p.ARN,
		"description": p.Description,
		"source": map[string]any{
			"type":     p.Source.Type,
			"location": p.Source.Location,
		},
		"artifacts": map[string]any{
			"type":     p.Artifacts.Type,
			"location": p.Artifacts.Location,
		},
		"environment": map[string]any{
			"type":                 p.Environment.Type,
			"image":                p.Environment.Image,
			"computeType":          p.Environment.ComputeType,
			"environmentVariables": envVars,
		},
		"serviceRole":      p.ServiceRole,
		"timeoutInMinutes": p.TimeoutInMins,
		"created":          float64(p.CreatedAt.Unix()),
		"lastModified":     float64(p.LastModified.Unix()),
		"tags":             tagsToList(p.Tags),
	}
}

func buildToMap(b *Build) map[string]any {
	m := map[string]any{
		"id":           b.ID,
		"arn":          b.ARN,
		"projectName":  b.ProjectName,
		"buildNumber":  b.BuildNumber,
		"buildStatus":  b.BuildStatus,
		"currentPhase": b.CurrentPhase,
		"startTime":    float64(b.StartTime.Unix()),
		"source": map[string]any{
			"type":     b.Source.Type,
			"location": b.Source.Location,
		},
		"artifacts": map[string]any{
			"type":     b.Artifacts.Type,
			"location": b.Artifacts.Location,
		},
		"environment": map[string]any{
			"type":        b.Environment.Type,
			"image":       b.Environment.Image,
			"computeType": b.Environment.ComputeType,
		},
		"serviceRole":      b.ServiceRole,
		"timeoutInMinutes": b.TimeoutInMins,
		"logs": map[string]any{
			"groupName":  b.Logs.GroupName,
			"streamName": b.Logs.StreamName,
			"deepLink":   b.Logs.DeepLink,
		},
	}
	if b.EndTime != nil {
		m["endTime"] = float64(b.EndTime.Unix())
	}

	// Include build phases
	if len(b.Phases) > 0 {
		phases := make([]map[string]any, len(b.Phases))
		for i, p := range b.Phases {
			pm := map[string]any{
				"phaseType":   string(p.PhaseType),
				"phaseStatus": p.PhaseStatus,
				"startTime":   float64(p.StartTime.Unix()),
			}
			if p.EndTime != nil {
				pm["endTime"] = float64(p.EndTime.Unix())
			}
			if p.DurationInSec > 0 {
				pm["durationInSeconds"] = p.DurationInSec
			}
			phases[i] = pm
		}
		m["phases"] = phases
	}
	return m
}

func reportGroupToMap(rg *ReportGroup) map[string]any {
	m := map[string]any{
		"arn":       rg.ARN,
		"name":      rg.Name,
		"type":      rg.Type,
		"created":   float64(rg.CreatedAt.Unix()),
		"tags":      tagsToList(rg.Tags),
	}
	ec := map[string]any{
		"exportConfigType": rg.ExportConfig.ExportConfigType,
	}
	if rg.ExportConfig.S3Destination != nil {
		ec["s3Destination"] = map[string]any{
			"bucket": rg.ExportConfig.S3Destination.Bucket,
			"path":   rg.ExportConfig.S3Destination.Path,
		}
	}
	m["exportConfig"] = ec
	return m
}
