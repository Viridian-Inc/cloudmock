package xray

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func getStr(p map[string]any, k string) string {
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(p map[string]any, k string) float64 {
	if v, ok := p[k]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return -1
}

func getInt(p map[string]any, k string) int {
	v := getFloat(p, k)
	if v < 0 {
		return -1
	}
	return int(v)
}

func getBool(p map[string]any, k string) *bool {
	if v, ok := p[k]; ok {
		if b, ok := v.(bool); ok {
			return &b
		}
	}
	return nil
}

func getStrMap(p map[string]any, k string) map[string]string {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			r := make(map[string]string, len(m))
			for key, val := range m {
				if s, ok := val.(string); ok {
					r[key] = s
				}
			}
			return r
		}
	}
	return nil
}

func getStrSlice(p map[string]any, k string) []string {
	if v, ok := p[k]; ok {
		if arr, ok := v.([]any); ok {
			r := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					r = append(r, s)
				}
			}
			return r
		}
	}
	return nil
}

// ---- Trace Segments ----

func handlePutTraceSegments(p map[string]any, store *Store) (*service.Response, error) {
	docsRaw, _ := p["TraceSegmentDocuments"].([]any)
	docs := make([]string, 0, len(docsRaw))
	for _, d := range docsRaw {
		if s, ok := d.(string); ok {
			docs = append(docs, s)
		}
	}
	unprocessed := store.PutTraceSegments(docs)
	return jsonOK(map[string]any{
		"UnprocessedTraceSegments": unprocessed,
	})
}

func handleBatchGetTraces(p map[string]any, store *Store) (*service.Response, error) {
	ids := getStrSlice(p, "TraceIds")
	traces := store.BatchGetTraces(ids)
	unprocessed := make([]string, 0)
	return jsonOK(map[string]any{
		"Traces":              traces,
		"UnprocessedTraceIds": unprocessed,
	})
}

func handleGetTraceSummaries(p map[string]any, store *Store) (*service.Response, error) {
	summaries := store.GetTraceSummaries()
	return jsonOK(map[string]any{
		"TraceSummaries":       summaries,
		"TracesProcessedCount": len(summaries),
	})
}

func handleGetTraceGraph(p map[string]any, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"Services": []any{},
	})
}

// ---- Sampling Rules ----

func handleGetSamplingRules(p map[string]any, store *Store) (*service.Response, error) {
	rules := store.GetSamplingRules()
	items := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		items = append(items, samplingRuleToMap(r))
	}
	return jsonOK(map[string]any{
		"SamplingRuleRecords": items,
	})
}

func handleCreateSamplingRule(p map[string]any, store *Store) (*service.Response, error) {
	ruleRaw, _ := p["SamplingRule"].(map[string]any)
	if ruleRaw == nil {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"SamplingRule is required", http.StatusBadRequest))
	}
	name := getStr(ruleRaw, "RuleName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"RuleName is required", http.StatusBadRequest))
	}
	rule := &SamplingRule{
		RuleName:      name,
		Priority:      getInt(ruleRaw, "Priority"),
		FixedRate:     getFloat(ruleRaw, "FixedRate"),
		ReservoirSize: getInt(ruleRaw, "ReservoirSize"),
		ServiceName:   getStr(ruleRaw, "ServiceName"),
		ServiceType:   getStr(ruleRaw, "ServiceType"),
		Host:          getStr(ruleRaw, "Host"),
		HTTPMethod:    getStr(ruleRaw, "HTTPMethod"),
		URLPath:       getStr(ruleRaw, "URLPath"),
		Tags:          getStrMap(p, "Tags"),
	}
	if rule.ServiceName == "" {
		rule.ServiceName = "*"
	}
	if rule.ServiceType == "" {
		rule.ServiceType = "*"
	}
	if rule.Host == "" {
		rule.Host = "*"
	}
	if rule.HTTPMethod == "" {
		rule.HTTPMethod = "*"
	}
	if rule.URLPath == "" {
		rule.URLPath = "*"
	}
	created, awsErr := store.CreateSamplingRule(rule)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"SamplingRuleRecord": samplingRuleToMap(created),
	})
}

func handleUpdateSamplingRule(p map[string]any, store *Store) (*service.Response, error) {
	updateRaw, _ := p["SamplingRuleUpdate"].(map[string]any)
	if updateRaw == nil {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"SamplingRuleUpdate is required", http.StatusBadRequest))
	}
	name := getStr(updateRaw, "RuleName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"RuleName is required", http.StatusBadRequest))
	}
	fixedRate := getFloat(updateRaw, "FixedRate")
	reservoirSize := getInt(updateRaw, "ReservoirSize")
	priority := getInt(updateRaw, "Priority")

	updated, awsErr := store.UpdateSamplingRule(name, fixedRate, reservoirSize, priority)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"SamplingRuleRecord": samplingRuleToMap(updated),
	})
}

func handleDeleteSamplingRule(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "RuleARN")
	name := getStr(p, "RuleName")
	if name == "" && arn == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"RuleName or RuleARN is required", http.StatusBadRequest))
	}
	// resolve name from ARN if needed — store keys by name, so use name directly
	deleted, awsErr := store.DeleteSamplingRule(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"SamplingRuleRecord": samplingRuleToMap(deleted),
	})
}

func samplingRuleToMap(r *SamplingRule) map[string]any {
	return map[string]any{
		"SamplingRule": map[string]any{
			"RuleName":      r.RuleName,
			"RuleARN":       r.RuleARN,
			"Priority":      r.Priority,
			"FixedRate":     r.FixedRate,
			"ReservoirSize": r.ReservoirSize,
			"ServiceName":   r.ServiceName,
			"ServiceType":   r.ServiceType,
			"Host":          r.Host,
			"HTTPMethod":    r.HTTPMethod,
			"URLPath":       r.URLPath,
			"Version":       r.Version,
		},
		"CreatedAt":  r.CreatedAt.Unix(),
		"ModifiedAt": r.ModifiedAt.Unix(),
	}
}

// ---- Groups ----

func handleCreateGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "GroupName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"GroupName is required", http.StatusBadRequest))
	}
	filterExpr := getStr(p, "FilterExpression")
	insightEnabled := false
	if b := getBool(p, "InsightsConfiguration"); b != nil {
		insightEnabled = *b
	}
	tags := getStrMap(p, "Tags")

	g, awsErr := store.CreateGroup(name, filterExpr, insightEnabled, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Group": groupToMap(g)})
}

func handleGetGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "GroupName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"GroupName is required", http.StatusBadRequest))
	}
	g, awsErr := store.GetGroup(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Group": groupToMap(g)})
}

func handleGetGroups(_ map[string]any, store *Store) (*service.Response, error) {
	groups := store.GetGroups()
	items := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		items = append(items, groupToMap(g))
	}
	return jsonOK(map[string]any{"Groups": items})
}

func handleUpdateGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "GroupName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"GroupName is required", http.StatusBadRequest))
	}
	filterExpr := getStr(p, "FilterExpression")
	insightEnabled := getBool(p, "InsightsEnabled")

	g, awsErr := store.UpdateGroup(name, filterExpr, insightEnabled)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Group": groupToMap(g)})
}

func handleDeleteGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "GroupName")
	if name == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"GroupName is required", http.StatusBadRequest))
	}
	if awsErr := store.DeleteGroup(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func groupToMap(g *Group) map[string]any {
	return map[string]any{
		"GroupName":        g.GroupName,
		"GroupARN":         g.GroupARN,
		"FilterExpression": g.FilterExpression,
		"InsightsConfiguration": map[string]any{
			"InsightsEnabled": g.InsightEnabled,
		},
	}
}

// ---- Encryption Config ----

func handlePutEncryptionConfig(p map[string]any, store *Store) (*service.Response, error) {
	keyID := getStr(p, "KeyId")
	encType := getStr(p, "Type")
	if encType == "" {
		encType = "NONE"
	}
	cfg := store.PutEncryptionConfig(keyID, encType)
	return jsonOK(map[string]any{
		"EncryptionConfig": encryptionConfigToMap(cfg),
	})
}

func handleGetEncryptionConfig(_ map[string]any, store *Store) (*service.Response, error) {
	cfg := store.GetEncryptionConfig()
	return jsonOK(map[string]any{
		"EncryptionConfig": encryptionConfigToMap(cfg),
	})
}

func encryptionConfigToMap(cfg EncryptionConfig) map[string]any {
	return map[string]any{
		"KeyId":  cfg.KeyID,
		"Status": cfg.Status,
		"Type":   cfg.Type,
	}
}

// ---- Tags ----

func handleTagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceARN")
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"ResourceARN is required", http.StatusBadRequest))
	}
	tags := getStrMap(p, "Tags")
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceARN")
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"ResourceARN is required", http.StatusBadRequest))
	}
	keys := getStrSlice(p, "TagKeys")
	if awsErr := store.UntagResource(arn, keys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceARN")
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"ResourceARN is required", http.StatusBadRequest))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	tagList := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, map[string]any{"Key": k, "Value": v})
	}
	return jsonOK(map[string]any{"Tags": tagList})
}
