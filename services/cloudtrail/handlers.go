package cloudtrail

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

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

func trailToMap(t *Trail) map[string]any {
	m := map[string]any{
		"Name":                       t.Name,
		"TrailARN":                   t.TrailARN,
		"HomeRegion":                 t.HomeRegion,
		"S3BucketName":               t.S3BucketName,
		"IncludeGlobalServiceEvents": t.IncludeGlobalServiceEvents,
		"IsMultiRegionTrail":         t.IsMultiRegionTrail,
		"IsOrganizationTrail":        t.IsOrganizationTrail,
		"LogFileValidationEnabled":   t.LogFileValidationEnabled,
		"HasCustomEventSelectors":    t.HasCustomEventSelectors,
		"HasInsightSelectors":        t.HasInsightSelectors,
	}
	if t.S3KeyPrefix != "" {
		m["S3KeyPrefix"] = t.S3KeyPrefix
	}
	if t.SnsTopicName != "" {
		m["SnsTopicName"] = t.SnsTopicName
	}
	if t.SnsTopicARN != "" {
		m["SnsTopicARN"] = t.SnsTopicARN
	}
	if t.CloudWatchLogsLogGroupArn != "" {
		m["CloudWatchLogsLogGroupArn"] = t.CloudWatchLogsLogGroupArn
	}
	if t.CloudWatchLogsRoleArn != "" {
		m["CloudWatchLogsRoleArn"] = t.CloudWatchLogsRoleArn
	}
	if t.KmsKeyId != "" {
		m["KmsKeyId"] = t.KmsKeyId
	}
	return m
}

func parseTags(raw []any) []Tag {
	tags := make([]Tag, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		t := Tag{}
		if v, ok := m["Key"].(string); ok {
			t.Key = v
		}
		if v, ok := m["Value"].(string); ok {
			t.Value = v
		}
		tags = append(tags, t)
	}
	return tags
}

func tagsToMaps(tags []Tag) []map[string]string {
	out := make([]map[string]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]string{"Key": t.Key, "Value": t.Value})
	}
	return out
}

func eventSelectorsToMaps(selectors []EventSelector) []map[string]any {
	out := make([]map[string]any, 0, len(selectors))
	for _, es := range selectors {
		m := map[string]any{
			"ReadWriteType":           es.ReadWriteType,
			"IncludeManagementEvents": es.IncludeManagementEvents,
		}
		drs := make([]map[string]any, 0, len(es.DataResources))
		for _, dr := range es.DataResources {
			drs = append(drs, map[string]any{
				"Type":   dr.Type,
				"Values": dr.Values,
			})
		}
		m["DataResources"] = drs
		if len(es.ExcludeManagementEventSources) > 0 {
			m["ExcludeManagementEventSources"] = es.ExcludeManagementEventSources
		}
		out = append(out, m)
	}
	return out
}

func parseEventSelectors(raw []any) []EventSelector {
	selectors := make([]EventSelector, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		es := EventSelector{
			ReadWriteType:           "All",
			IncludeManagementEvents: true,
		}
		if v, ok := m["ReadWriteType"].(string); ok {
			es.ReadWriteType = v
		}
		if v, ok := m["IncludeManagementEvents"].(bool); ok {
			es.IncludeManagementEvents = v
		}
		if drs, ok := m["DataResources"].([]any); ok {
			for _, drRaw := range drs {
				drMap, ok := drRaw.(map[string]any)
				if !ok {
					continue
				}
				dr := DataResource{}
				if v, ok := drMap["Type"].(string); ok {
					dr.Type = v
				}
				if vals, ok := drMap["Values"].([]any); ok {
					for _, val := range vals {
						if s, ok := val.(string); ok {
							dr.Values = append(dr.Values, s)
						}
					}
				}
				es.DataResources = append(es.DataResources, dr)
			}
		}
		selectors = append(selectors, es)
	}
	return selectors
}

func handleCreateTrail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	s3Bucket, _ := params["S3BucketName"].(string)
	s3Prefix, _ := params["S3KeyPrefix"].(string)
	isMultiRegion, _ := params["IsMultiRegionTrail"].(bool)
	isOrg, _ := params["IsOrganizationTrail"].(bool)
	logValidation, _ := params["EnableLogFileValidation"].(bool)
	includeGlobal := true
	if v, ok := params["IncludeGlobalServiceEvents"].(bool); ok {
		includeGlobal = v
	}

	var tags []Tag
	if rawTags, ok := params["TagsList"].([]any); ok {
		tags = parseTags(rawTags)
	}

	trail, awsErr := store.CreateTrail(name, s3Bucket, s3Prefix, isMultiRegion, isOrg, logValidation, includeGlobal, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(trailToMap(trail))
}

func handleGetTrail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	trail, awsErr := store.GetTrail(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"Trail": trailToMap(trail)})
}

func handleDescribeTrails(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	trails := store.ListTrails()
	items := make([]map[string]any, 0, len(trails))
	for _, t := range trails {
		items = append(items, trailToMap(t))
	}
	return jsonOK(map[string]any{"trailList": items})
}

func handleDeleteTrail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	if awsErr := store.DeleteTrail(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleUpdateTrail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	trail, awsErr := store.UpdateTrail(name, params)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(trailToMap(trail))
}

func handleStartLogging(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	if awsErr := store.StartLogging(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleStopLogging(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	if awsErr := store.StopLogging(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleGetTrailStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["Name"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}

	trail, awsErr := store.GetTrail(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	status := map[string]any{
		"IsLogging": trail.IsLogging,
	}
	if trail.LatestDeliveryTime != nil {
		status["LatestDeliveryTime"] = float64(trail.LatestDeliveryTime.Unix())
	}
	if trail.StartLoggingTime != nil {
		status["StartLoggingTime"] = float64(trail.StartLoggingTime.Unix())
	}
	if trail.StopLoggingTime != nil {
		status["StopLoggingTime"] = float64(trail.StopLoggingTime.Unix())
	}

	return jsonOK(status)
}

func handlePutEventSelectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	trailName, _ := params["TrailName"].(string)
	if trailName == "" {
		return jsonErr(service.ErrValidation("TrailName is required."))
	}

	var selectors []EventSelector
	if rawSelectors, ok := params["EventSelectors"].([]any); ok {
		selectors = parseEventSelectors(rawSelectors)
	}

	if awsErr := store.PutEventSelectors(trailName, selectors); awsErr != nil {
		return jsonErr(awsErr)
	}

	trail, _ := store.GetTrail(trailName)
	return jsonOK(map[string]any{
		"TrailARN":       trail.TrailARN,
		"EventSelectors": eventSelectorsToMaps(trail.EventSelectors),
	})
}

func handleGetEventSelectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	trailName, _ := params["TrailName"].(string)
	if trailName == "" {
		return jsonErr(service.ErrValidation("TrailName is required."))
	}

	trail, awsErr := store.GetTrail(trailName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"TrailARN":       trail.TrailARN,
		"EventSelectors": eventSelectorsToMaps(trail.EventSelectors),
	})
}

func handlePutInsightSelectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	trailName, _ := params["TrailName"].(string)
	if trailName == "" {
		return jsonErr(service.ErrValidation("TrailName is required."))
	}

	var selectors []InsightSelector
	if rawSelectors, ok := params["InsightSelectors"].([]any); ok {
		for _, item := range rawSelectors {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			is := InsightSelector{}
			if v, ok := m["InsightType"].(string); ok {
				is.InsightType = v
			}
			selectors = append(selectors, is)
		}
	}

	if awsErr := store.PutInsightSelectors(trailName, selectors); awsErr != nil {
		return jsonErr(awsErr)
	}

	trail, _ := store.GetTrail(trailName)
	insightMaps := make([]map[string]any, 0, len(trail.InsightSelectors))
	for _, is := range trail.InsightSelectors {
		insightMaps = append(insightMaps, map[string]any{"InsightType": is.InsightType})
	}

	return jsonOK(map[string]any{
		"TrailARN":         trail.TrailARN,
		"InsightSelectors": insightMaps,
	})
}

func handleGetInsightSelectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	trailName, _ := params["TrailName"].(string)
	if trailName == "" {
		return jsonErr(service.ErrValidation("TrailName is required."))
	}

	trail, awsErr := store.GetTrail(trailName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	insightMaps := make([]map[string]any, 0, len(trail.InsightSelectors))
	for _, is := range trail.InsightSelectors {
		insightMaps = append(insightMaps, map[string]any{"InsightType": is.InsightType})
	}

	return jsonOK(map[string]any{
		"TrailARN":         trail.TrailARN,
		"InsightSelectors": insightMaps,
	})
}

func handleLookupEvents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	maxResults := 50
	if v, ok := params["MaxResults"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	events := store.LookupEvents(maxResults)
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		m := map[string]any{
			"EventId":     e.EventId,
			"EventName":   e.EventName,
			"EventTime":   float64(e.EventTime.Unix()),
			"EventSource": e.EventSource,
			"Username":    e.Username,
			"ReadOnly":    e.ReadOnly,
		}
		if len(e.Resources) > 0 {
			res := make([]map[string]any, 0, len(e.Resources))
			for _, r := range e.Resources {
				res = append(res, map[string]any{
					"ResourceType": r.ResourceType,
					"ResourceName": r.ResourceName,
				})
			}
			m["Resources"] = res
		}
		items = append(items, m)
	}

	return jsonOK(map[string]any{"Events": items})
}

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	resourceId, _ := params["ResourceId"].(string)
	if resourceId == "" {
		return jsonErr(service.ErrValidation("ResourceId is required."))
	}

	var tags []Tag
	if rawTags, ok := params["TagsList"].([]any); ok {
		tags = parseTags(rawTags)
	}

	if awsErr := store.AddTags(resourceId, tags); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleRemoveTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	resourceId, _ := params["ResourceId"].(string)
	if resourceId == "" {
		return jsonErr(service.ErrValidation("ResourceId is required."))
	}

	var tags []Tag
	if rawTags, ok := params["TagsList"].([]any); ok {
		tags = parseTags(rawTags)
	}

	if awsErr := store.RemoveTags(resourceId, tags); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	var arns []string
	if rawArns, ok := params["ResourceIdList"].([]any); ok {
		for _, a := range rawArns {
			if s, ok := a.(string); ok {
				arns = append(arns, s)
			}
		}
	}

	resourceTagList := make([]map[string]any, 0)
	for _, arn := range arns {
		tags, _ := store.ListTagsByARN(arn)
		resourceTagList = append(resourceTagList, map[string]any{
			"ResourceId": arn,
			"TagsList":   tagsToMaps(tags),
		})
	}

	return jsonOK(map[string]any{"ResourceTagList": resourceTagList})
}
