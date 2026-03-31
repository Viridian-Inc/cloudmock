package wafv2

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
		return service.NewAWSError("WAFInvalidParameterException",
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

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
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

func parseStringSlice(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseMapSlice(raw []any) []map[string]any {
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func webACLSummary(acl *WebACL) map[string]any {
	return map[string]any{
		"Name":      acl.Name,
		"Id":        acl.Id,
		"ARN":       acl.ARN,
		"LockToken": acl.LockToken,
	}
}

func webACLDetail(acl *WebACL) map[string]any {
	return map[string]any{
		"Name":             acl.Name,
		"Id":               acl.Id,
		"ARN":              acl.ARN,
		"Description":      acl.Description,
		"DefaultAction":    acl.DefaultAction,
		"Rules":            acl.Rules,
		"VisibilityConfig": acl.VisibilityConfig,
		"Capacity":         acl.Capacity,
	}
}

func handleCreateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	scope, _ := params["Scope"].(string)
	defaultAction, _ := params["DefaultAction"].(map[string]any)
	var rules []map[string]any
	if raw, ok := params["Rules"].([]any); ok {
		rules = parseMapSlice(raw)
	}
	visibilityConfig, _ := params["VisibilityConfig"].(map[string]any)
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	acl, awsErr := store.CreateWebACL(name, description, scope, defaultAction, rules, visibilityConfig, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Summary": webACLSummary(acl)})
}

func handleGetWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	acl, awsErr := store.GetWebACL(name, scope, id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"WebACL": webACLDetail(acl), "LockToken": acl.LockToken})
}

func handleListWebACLs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	scope, _ := params["Scope"].(string)
	acls := store.ListWebACLs(scope)
	items := make([]map[string]any, 0, len(acls))
	for _, acl := range acls {
		items = append(items, webACLSummary(acl))
	}
	return jsonOK(map[string]any{"WebACLs": items})
}

func handleUpdateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	defaultAction, _ := params["DefaultAction"].(map[string]any)
	description, _ := params["Description"].(string)
	var rules []map[string]any
	if raw, ok := params["Rules"].([]any); ok {
		rules = parseMapSlice(raw)
	}
	visibilityConfig, _ := params["VisibilityConfig"].(map[string]any)
	acl, awsErr := store.UpdateWebACL(name, scope, id, lockToken, defaultAction, rules, visibilityConfig, description)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"NextLockToken": acl.LockToken})
}

func handleDeleteWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	if awsErr := store.DeleteWebACL(name, scope, id, lockToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateRuleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	scope, _ := params["Scope"].(string)
	capacity := int64(100)
	if v, ok := params["Capacity"].(float64); ok {
		capacity = int64(v)
	}
	var rules []map[string]any
	if raw, ok := params["Rules"].([]any); ok {
		rules = parseMapSlice(raw)
	}
	visibilityConfig, _ := params["VisibilityConfig"].(map[string]any)
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	rg, awsErr := store.CreateRuleGroup(name, description, scope, capacity, rules, visibilityConfig, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Summary": map[string]any{
		"Name": rg.Name, "Id": rg.Id, "ARN": rg.ARN, "LockToken": rg.LockToken,
	}})
}

func handleGetRuleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	rg, awsErr := store.GetRuleGroup(name, scope, id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"RuleGroup": map[string]any{
			"Name": rg.Name, "Id": rg.Id, "ARN": rg.ARN,
			"Description": rg.Description, "Capacity": rg.Capacity,
			"Rules": rg.Rules, "VisibilityConfig": rg.VisibilityConfig,
		},
		"LockToken": rg.LockToken,
	})
}

func handleListRuleGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	scope, _ := params["Scope"].(string)
	rgs := store.ListRuleGroups(scope)
	items := make([]map[string]any, 0, len(rgs))
	for _, rg := range rgs {
		items = append(items, map[string]any{
			"Name": rg.Name, "Id": rg.Id, "ARN": rg.ARN, "LockToken": rg.LockToken,
		})
	}
	return jsonOK(map[string]any{"RuleGroups": items})
}

func handleUpdateRuleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	description, _ := params["Description"].(string)
	var rules []map[string]any
	if raw, ok := params["Rules"].([]any); ok {
		rules = parseMapSlice(raw)
	}
	visibilityConfig, _ := params["VisibilityConfig"].(map[string]any)
	rg, awsErr := store.UpdateRuleGroup(name, scope, id, lockToken, rules, visibilityConfig, description)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"NextLockToken": rg.LockToken})
}

func handleDeleteRuleGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	if awsErr := store.DeleteRuleGroup(name, scope, id, lockToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	scope, _ := params["Scope"].(string)
	ipVersion, _ := params["IPAddressVersion"].(string)
	var addresses []string
	if raw, ok := params["Addresses"].([]any); ok {
		addresses = parseStringSlice(raw)
	}
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	ipSet, awsErr := store.CreateIPSet(name, description, scope, ipVersion, addresses, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Summary": map[string]any{
		"Name": ipSet.Name, "Id": ipSet.Id, "ARN": ipSet.ARN, "LockToken": ipSet.LockToken,
	}})
}

func handleGetIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	ipSet, awsErr := store.GetIPSet(name, scope, id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"IPSet": map[string]any{
			"Name": ipSet.Name, "Id": ipSet.Id, "ARN": ipSet.ARN,
			"Description": ipSet.Description, "IPAddressVersion": ipSet.IPAddressVersion,
			"Addresses": ipSet.Addresses,
		},
		"LockToken": ipSet.LockToken,
	})
}

func handleListIPSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	scope, _ := params["Scope"].(string)
	ipSets := store.ListIPSets(scope)
	items := make([]map[string]any, 0, len(ipSets))
	for _, ipSet := range ipSets {
		items = append(items, map[string]any{
			"Name": ipSet.Name, "Id": ipSet.Id, "ARN": ipSet.ARN, "LockToken": ipSet.LockToken,
		})
	}
	return jsonOK(map[string]any{"IPSets": items})
}

func handleUpdateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	description, _ := params["Description"].(string)
	var addresses []string
	if raw, ok := params["Addresses"].([]any); ok {
		addresses = parseStringSlice(raw)
	}
	ipSet, awsErr := store.UpdateIPSet(name, scope, id, lockToken, addresses, description)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"NextLockToken": ipSet.LockToken})
}

func handleDeleteIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	if awsErr := store.DeleteIPSet(name, scope, id, lockToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateRegexPatternSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	scope, _ := params["Scope"].(string)
	var patterns []string
	if raw, ok := params["RegularExpressionList"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				if regex, ok := m["RegexString"].(string); ok {
					patterns = append(patterns, regex)
				}
			} else if s, ok := item.(string); ok {
				patterns = append(patterns, s)
			}
		}
	}
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	rps, awsErr := store.CreateRegexPatternSet(name, description, scope, patterns, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Summary": map[string]any{
		"Name": rps.Name, "Id": rps.Id, "ARN": rps.ARN, "LockToken": rps.LockToken,
	}})
}

func handleGetRegexPatternSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	rps, awsErr := store.GetRegexPatternSet(name, scope, id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	regexList := make([]map[string]any, 0, len(rps.RegularExpressionList))
	for _, r := range rps.RegularExpressionList {
		regexList = append(regexList, map[string]any{"RegexString": r})
	}
	return jsonOK(map[string]any{
		"RegexPatternSet": map[string]any{
			"Name": rps.Name, "Id": rps.Id, "ARN": rps.ARN,
			"Description":           rps.Description,
			"RegularExpressionList": regexList,
		},
		"LockToken": rps.LockToken,
	})
}

func handleListRegexPatternSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	scope, _ := params["Scope"].(string)
	sets := store.ListRegexPatternSets(scope)
	items := make([]map[string]any, 0, len(sets))
	for _, rps := range sets {
		items = append(items, map[string]any{
			"Name": rps.Name, "Id": rps.Id, "ARN": rps.ARN, "LockToken": rps.LockToken,
		})
	}
	return jsonOK(map[string]any{"RegexPatternSets": items})
}

func handleDeleteRegexPatternSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	scope, _ := params["Scope"].(string)
	id, _ := params["Id"].(string)
	lockToken, _ := params["LockToken"].(string)
	if awsErr := store.DeleteRegexPatternSet(name, scope, id, lockToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAssociateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	webACLArn, _ := params["WebACLArn"].(string)
	resourceArn, _ := params["ResourceArn"].(string)
	if awsErr := store.AssociateWebACL(webACLArn, resourceArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDisassociateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	if awsErr := store.DisassociateWebACL(resourceArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleGetWebACLForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	acl, awsErr := store.GetWebACLForResource(resourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"WebACL": webACLDetail(acl)})
}

func handlePutLoggingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	loggingConfig, _ := params["LoggingConfiguration"].(map[string]any)
	if loggingConfig == nil {
		return jsonErr(service.ErrValidation("LoggingConfiguration is required."))
	}
	resourceArn, _ := loggingConfig["ResourceArn"].(string)
	if awsErr := store.SetLoggingConfig(resourceArn, loggingConfig); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"LoggingConfiguration": loggingConfig})
}

func handleGetLoggingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	config, awsErr := store.GetLoggingConfig(resourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"LoggingConfiguration": config})
}

func handleDeleteLoggingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	if awsErr := store.DeleteLoggingConfig(resourceArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["ResourceARN"].(string)
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["ResourceARN"].(string)
	var tagKeys []string
	if raw, ok := params["TagKeys"].([]any); ok {
		tagKeys = parseStringSlice(raw)
	}
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["ResourceARN"].(string)
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"TagInfoForResource": map[string]any{
			"ResourceARN": arn,
			"TagList":     tagsToMaps(tags),
		},
	})
}
