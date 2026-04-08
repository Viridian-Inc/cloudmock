package wafregional

import (
	gojson "github.com/goccy/go-json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// WAF Regional uses JSON protocol despite historical query protocol.
// Modern AWS SDKs send JSON to waf-regional.

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
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

func handleCreateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	metricName, _ := params["MetricName"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	defaultAction := "ALLOW"
	if da, ok := params["DefaultAction"].(map[string]any); ok {
		if v, ok := da["Type"].(string); ok {
			defaultAction = v
		}
	}
	acl, awsErr := store.CreateWebACL(name, metricName, defaultAction, changeToken)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"WebACL": map[string]any{
			"WebACLId":   acl.WebACLId,
			"Name":       acl.Name,
			"MetricName": acl.MetricName,
			"DefaultAction": map[string]any{"Type": acl.DefaultAction},
			"Rules":      []any{},
			"WebACLArn":  acl.WebACLArn,
		},
		"ChangeToken": acl.ChangeToken,
	})
}

func handleGetWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["WebACLId"].(string)
	acl, awsErr := store.GetWebACL(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	rules := make([]map[string]any, 0, len(acl.Rules))
	for _, r := range acl.Rules {
		rules = append(rules, map[string]any{
			"Priority": r.Priority,
			"RuleId":   r.RuleId,
			"Action":   map[string]any{"Type": r.Action},
			"Type":     r.Type,
		})
	}
	return jsonOK(map[string]any{
		"WebACL": map[string]any{
			"WebACLId":      acl.WebACLId,
			"Name":          acl.Name,
			"MetricName":    acl.MetricName,
			"DefaultAction": map[string]any{"Type": acl.DefaultAction},
			"Rules":         rules,
			"WebACLArn":     acl.WebACLArn,
		},
	})
}

func handleListWebACLs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	acls := store.ListWebACLs()
	items := make([]map[string]any, 0, len(acls))
	for _, acl := range acls {
		items = append(items, map[string]any{
			"WebACLId": acl.WebACLId,
			"Name":     acl.Name,
		})
	}
	return jsonOK(map[string]any{"WebACLs": items})
}

func handleUpdateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["WebACLId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	var updates []map[string]any
	if raw, ok := params["Updates"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				updates = append(updates, m)
			}
		}
	}
	if awsErr := store.UpdateWebACL(id, changeToken, updates); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleDeleteWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["WebACLId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	if awsErr := store.DeleteWebACL(id, changeToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleCreateRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	metricName, _ := params["MetricName"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	rule, awsErr := store.CreateRule(name, metricName, changeToken)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Rule": map[string]any{
			"RuleId":     rule.RuleId,
			"Name":       rule.Name,
			"MetricName": rule.MetricName,
			"Predicates": []any{},
		},
		"ChangeToken": newUUID(),
	})
}

func handleGetRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["RuleId"].(string)
	rule, awsErr := store.GetRule(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	predicates := make([]map[string]any, 0, len(rule.Predicates))
	for _, p := range rule.Predicates {
		predicates = append(predicates, map[string]any{
			"Negated": p.Negated, "Type": p.Type, "DataId": p.DataId,
		})
	}
	return jsonOK(map[string]any{
		"Rule": map[string]any{
			"RuleId": rule.RuleId, "Name": rule.Name,
			"MetricName": rule.MetricName, "Predicates": predicates,
		},
	})
}

func handleListRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	rules := store.ListRules()
	items := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		items = append(items, map[string]any{"RuleId": r.RuleId, "Name": r.Name})
	}
	return jsonOK(map[string]any{"Rules": items})
}

func handleUpdateRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["RuleId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	var updates []map[string]any
	if raw, ok := params["Updates"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				updates = append(updates, m)
			}
		}
	}
	if awsErr := store.UpdateRule(id, changeToken, updates); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleDeleteRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["RuleId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	if awsErr := store.DeleteRule(id, changeToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleCreateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	ipSet, awsErr := store.CreateIPSet(name, changeToken)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"IPSet": map[string]any{
			"IPSetId":          ipSet.IPSetId,
			"Name":             ipSet.Name,
			"IPSetDescriptors": []any{},
		},
		"ChangeToken": newUUID(),
	})
}

func handleGetIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["IPSetId"].(string)
	ipSet, awsErr := store.GetIPSet(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	descriptors := make([]map[string]any, 0, len(ipSet.IPSetDescriptors))
	for _, d := range ipSet.IPSetDescriptors {
		descriptors = append(descriptors, map[string]any{"Type": d.Type, "Value": d.Value})
	}
	return jsonOK(map[string]any{
		"IPSet": map[string]any{
			"IPSetId":          ipSet.IPSetId,
			"Name":             ipSet.Name,
			"IPSetDescriptors": descriptors,
		},
	})
}

func handleListIPSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	ipSets := store.ListIPSets()
	items := make([]map[string]any, 0, len(ipSets))
	for _, ipSet := range ipSets {
		items = append(items, map[string]any{"IPSetId": ipSet.IPSetId, "Name": ipSet.Name})
	}
	return jsonOK(map[string]any{"IPSets": items})
}

func handleUpdateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["IPSetId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	var updates []map[string]any
	if raw, ok := params["Updates"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				updates = append(updates, m)
			}
		}
	}
	if awsErr := store.UpdateIPSet(id, changeToken, updates); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleDeleteIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["IPSetId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	if awsErr := store.DeleteIPSet(id, changeToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleCreateByteMatchSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	bms, awsErr := store.CreateByteMatchSet(name, changeToken)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ByteMatchSet": map[string]any{
			"ByteMatchSetId":  bms.ByteMatchSetId,
			"Name":            bms.Name,
			"ByteMatchTuples": []any{},
		},
		"ChangeToken": newUUID(),
	})
}

func handleGetByteMatchSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ByteMatchSetId"].(string)
	bms, awsErr := store.GetByteMatchSet(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	tuples := make([]map[string]any, 0, len(bms.ByteMatchTuples))
	for _, t := range bms.ByteMatchTuples {
		tuples = append(tuples, map[string]any{
			"FieldToMatch": map[string]any{"Type": t.FieldToMatch.Type, "Data": t.FieldToMatch.Data},
			"TargetString": t.TargetString,
			"TextTransformation": t.TextTransformation,
			"PositionalConstraint": t.PositionalConstraint,
		})
	}
	return jsonOK(map[string]any{
		"ByteMatchSet": map[string]any{
			"ByteMatchSetId":  bms.ByteMatchSetId,
			"Name":            bms.Name,
			"ByteMatchTuples": tuples,
		},
	})
}

func handleListByteMatchSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	sets := store.ListByteMatchSets()
	items := make([]map[string]any, 0, len(sets))
	for _, bms := range sets {
		items = append(items, map[string]any{
			"ByteMatchSetId": bms.ByteMatchSetId, "Name": bms.Name,
		})
	}
	return jsonOK(map[string]any{"ByteMatchSets": items})
}

func handleDeleteByteMatchSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ByteMatchSetId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	if awsErr := store.DeleteByteMatchSet(id, changeToken); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleUpdateByteMatchSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ByteMatchSetId"].(string)
	changeToken, _ := params["ChangeToken"].(string)
	var updates []map[string]any
	if raw, ok := params["Updates"].([]any); ok {
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				updates = append(updates, m)
			}
		}
	}
	if awsErr := store.UpdateByteMatchSet(id, changeToken, updates); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ChangeToken": newUUID()})
}

func handleGetChangeToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	token := store.GetChangeToken()
	return jsonOK(map[string]any{"ChangeToken": token})
}

func handleGetChangeTokenStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	token, _ := params["ChangeToken"].(string)
	status := store.GetChangeTokenStatus(token)
	return jsonOK(map[string]any{"ChangeTokenStatus": status})
}

func handleAssociateWebACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	webACLId, _ := params["WebACLId"].(string)
	resourceArn, _ := params["ResourceArn"].(string)
	if awsErr := store.AssociateWebACL(webACLId, resourceArn); awsErr != nil {
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
	return jsonOK(map[string]any{
		"WebACLSummary": map[string]any{
			"WebACLId": acl.WebACLId,
			"Name":     acl.Name,
		},
	})
}
