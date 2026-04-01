package iot

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

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func getStr(p map[string]any, k string) string {
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(p map[string]any, k string) bool {
	if v, ok := p[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
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

func getMap(p map[string]any, k string) map[string]any {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func getStringSlice(p map[string]any, k string) []string {
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

func getSliceOfMaps(p map[string]any, k string) []map[string]any {
	if v, ok := p[k]; ok {
		if arr, ok := v.([]any); ok {
			r := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					r = append(r, m)
				}
			}
			return r
		}
	}
	return nil
}

type tagEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func parseTags(p map[string]any) map[string]string {
	if v, ok := p["tags"]; ok {
		if arr, ok := v.([]any); ok {
			m := make(map[string]string, len(arr))
			for _, item := range arr {
				if t, ok := item.(map[string]any); ok {
					m[getStr(t, "Key")] = getStr(t, "Value")
				}
			}
			return m
		}
	}
	return nil
}

func tagsToEntries(m map[string]string) []tagEntry {
	entries := make([]tagEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, tagEntry{Key: k, Value: v})
	}
	return entries
}

// ---- Thing handlers ----

func handleCreateThing(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	t, awsErr := store.CreateThing(name, getStr(p, "thingTypeName"), getStrMap(p, "attributePayload"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"thingName": t.ThingName, "thingArn": t.ThingArn, "thingId": newUUID()})
}

func handleDescribeThing(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	t, awsErr := store.DescribeThing(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"thingName":     t.ThingName,
		"thingArn":      t.ThingArn,
		"thingTypeName": t.ThingTypeName,
		"attributes":    t.Attributes,
		"version":       t.Version,
	})
}

func handleListThings(store *Store) (*service.Response, error) {
	things := store.ListThings()
	entries := make([]map[string]any, 0, len(things))
	for _, t := range things {
		entries = append(entries, map[string]any{
			"thingName":     t.ThingName,
			"thingArn":      t.ThingArn,
			"thingTypeName": t.ThingTypeName,
			"attributes":    t.Attributes,
			"version":       t.Version,
		})
	}
	return jsonOK(map[string]any{"things": entries})
}

func handleUpdateThing(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	if awsErr := store.UpdateThing(name, getStr(p, "thingTypeName"), getStrMap(p, "attributePayload"), getBool(p, "removeThingType")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteThing(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	if awsErr := store.DeleteThing(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Thing type handlers ----

func handleCreateThingType(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingTypeName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingTypeName is required."))
	}
	tt, awsErr := store.CreateThingType(name, getMap(p, "thingTypeProperties"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"thingTypeName": tt.ThingTypeName, "thingTypeArn": tt.ThingTypeArn, "thingTypeId": newUUID()})
}

func handleDescribeThingType(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingTypeName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingTypeName is required."))
	}
	tt, awsErr := store.DescribeThingType(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"thingTypeName":       tt.ThingTypeName,
		"thingTypeArn":        tt.ThingTypeArn,
		"thingTypeProperties": tt.Properties,
		"thingTypeMetadata":   map[string]any{"deprecated": tt.Deprecated, "creationDate": tt.CreationDate.Format(time.RFC3339)},
	})
}

func handleListThingTypes(store *Store) (*service.Response, error) {
	types := store.ListThingTypes()
	entries := make([]map[string]any, 0, len(types))
	for _, tt := range types {
		entries = append(entries, map[string]any{
			"thingTypeName": tt.ThingTypeName,
			"thingTypeArn":  tt.ThingTypeArn,
		})
	}
	return jsonOK(map[string]any{"thingTypes": entries})
}

func handleDeleteThingType(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingTypeName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingTypeName is required."))
	}
	if awsErr := store.DeleteThingType(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Thing group handlers ----

func handleCreateThingGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingGroupName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingGroupName is required."))
	}
	tg, awsErr := store.CreateThingGroup(name, getStr(p, "parentGroupName"), getMap(p, "thingGroupProperties"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"thingGroupName": tg.ThingGroupName, "thingGroupArn": tg.ThingGroupArn, "thingGroupId": newUUID()})
}

func handleDescribeThingGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingGroupName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingGroupName is required."))
	}
	tg, awsErr := store.DescribeThingGroup(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"thingGroupName":       tg.ThingGroupName,
		"thingGroupArn":        tg.ThingGroupArn,
		"thingGroupProperties": tg.Properties,
		"thingGroupMetadata":   map[string]any{"parentGroupName": tg.ParentGroupName, "creationDate": tg.CreationDate.Format(time.RFC3339)},
		"version":              1,
	})
}

func handleListThingGroups(store *Store) (*service.Response, error) {
	groups := store.ListThingGroups()
	entries := make([]map[string]any, 0, len(groups))
	for _, tg := range groups {
		entries = append(entries, map[string]any{
			"groupName": tg.ThingGroupName,
			"groupArn":  tg.ThingGroupArn,
		})
	}
	return jsonOK(map[string]any{"thingGroups": entries})
}

func handleDeleteThingGroup(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "thingGroupName")
	if name == "" {
		return jsonErr(service.ErrValidation("thingGroupName is required."))
	}
	if awsErr := store.DeleteThingGroup(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAddThingToThingGroup(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	groupName := getStr(p, "thingGroupName")
	if thingName == "" || groupName == "" {
		return jsonErr(service.ErrValidation("thingName and thingGroupName are required."))
	}
	if awsErr := store.AddThingToThingGroup(thingName, groupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleRemoveThingFromThingGroup(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	groupName := getStr(p, "thingGroupName")
	if thingName == "" || groupName == "" {
		return jsonErr(service.ErrValidation("thingName and thingGroupName are required."))
	}
	if awsErr := store.RemoveThingFromThingGroup(thingName, groupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Policy handlers ----

func handleCreatePolicy(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "policyName")
	if name == "" {
		return jsonErr(service.ErrValidation("policyName is required."))
	}
	pol, awsErr := store.CreatePolicy(name, getStr(p, "policyDocument"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyName":     pol.PolicyName,
		"policyArn":      pol.PolicyArn,
		"policyDocument": pol.PolicyDocument,
		"policyVersionId": pol.VersionId,
	})
}

func handleGetPolicy(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "policyName")
	if name == "" {
		return jsonErr(service.ErrValidation("policyName is required."))
	}
	pol, awsErr := store.GetPolicy(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyName":      pol.PolicyName,
		"policyArn":       pol.PolicyArn,
		"policyDocument":  pol.PolicyDocument,
		"defaultVersionId": pol.VersionId,
		"creationDate":    pol.CreationDate.Format(time.RFC3339),
	})
}

func handleListPolicies(store *Store) (*service.Response, error) {
	pols := store.ListPolicies()
	entries := make([]map[string]any, 0, len(pols))
	for _, pol := range pols {
		entries = append(entries, map[string]any{"policyName": pol.PolicyName, "policyArn": pol.PolicyArn})
	}
	return jsonOK(map[string]any{"policies": entries})
}

func handleDeletePolicy(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "policyName")
	if name == "" {
		return jsonErr(service.ErrValidation("policyName is required."))
	}
	if awsErr := store.DeletePolicy(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAttachPolicy(p map[string]any, store *Store) (*service.Response, error) {
	policyName := getStr(p, "policyName")
	target := getStr(p, "target")
	if policyName == "" || target == "" {
		return jsonErr(service.ErrValidation("policyName and target are required."))
	}
	if awsErr := store.AttachPolicy(policyName, target); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDetachPolicy(p map[string]any, store *Store) (*service.Response, error) {
	policyName := getStr(p, "policyName")
	target := getStr(p, "target")
	if policyName == "" || target == "" {
		return jsonErr(service.ErrValidation("policyName and target are required."))
	}
	if awsErr := store.DetachPolicy(policyName, target); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListAttachedPolicies(p map[string]any, store *Store) (*service.Response, error) {
	target := getStr(p, "target")
	if target == "" {
		return jsonErr(service.ErrValidation("target is required."))
	}
	pols := store.ListAttachedPolicies(target)
	entries := make([]map[string]any, 0, len(pols))
	for _, pol := range pols {
		entries = append(entries, map[string]any{"policyName": pol.PolicyName, "policyArn": pol.PolicyArn})
	}
	return jsonOK(map[string]any{"policies": entries})
}

// ---- Certificate handlers ----

func handleCreateKeysAndCertificate(p map[string]any, store *Store) (*service.Response, error) {
	setAsActive := getBool(p, "setAsActive")
	cert, awsErr := store.CreateKeysAndCertificate(setAsActive)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"certificateId":  cert.CertificateId,
		"certificateArn": cert.CertificateArn,
		"certificatePem": cert.CertificatePem,
		"keyPair":        cert.KeyPair,
	})
}

func handleDescribeCertificate(p map[string]any, store *Store) (*service.Response, error) {
	certId := getStr(p, "certificateId")
	if certId == "" {
		return jsonErr(service.ErrValidation("certificateId is required."))
	}
	cert, awsErr := store.DescribeCertificate(certId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"certificateDescription": map[string]any{
			"certificateId":  cert.CertificateId,
			"certificateArn": cert.CertificateArn,
			"certificatePem": cert.CertificatePem,
			"status":         cert.Status,
			"creationDate":   cert.CreationDate.Format(time.RFC3339),
		},
	})
}

func handleListCertificates(store *Store) (*service.Response, error) {
	certs := store.ListCertificates()
	entries := make([]map[string]any, 0, len(certs))
	for _, cert := range certs {
		entries = append(entries, map[string]any{
			"certificateId":  cert.CertificateId,
			"certificateArn": cert.CertificateArn,
			"status":         cert.Status,
			"creationDate":   cert.CreationDate.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"certificates": entries})
}

func handleDeleteCertificate(p map[string]any, store *Store) (*service.Response, error) {
	certId := getStr(p, "certificateId")
	if certId == "" {
		return jsonErr(service.ErrValidation("certificateId is required."))
	}
	if awsErr := store.DeleteCertificate(certId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAttachThingPrincipal(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	principal := getStr(p, "principal")
	if thingName == "" || principal == "" {
		return jsonErr(service.ErrValidation("thingName and principal are required."))
	}
	if awsErr := store.AttachThingPrincipal(thingName, principal); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDetachThingPrincipal(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	principal := getStr(p, "principal")
	if thingName == "" || principal == "" {
		return jsonErr(service.ErrValidation("thingName and principal are required."))
	}
	if awsErr := store.DetachThingPrincipal(thingName, principal); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListThingPrincipals(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	if thingName == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	principals, awsErr := store.ListThingPrincipals(thingName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"principals": principals})
}

// ---- Topic rule handlers ----

func handleCreateTopicRule(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "ruleName")
	if name == "" {
		return jsonErr(service.ErrValidation("ruleName is required."))
	}
	rulePayload := getMap(p, "topicRulePayload")
	sql := ""
	description := ""
	var actions []map[string]any
	ruleDisabled := false
	if rulePayload != nil {
		sql = getStr(rulePayload, "sql")
		description = getStr(rulePayload, "description")
		actions = getSliceOfMaps(rulePayload, "actions")
		ruleDisabled = getBool(rulePayload, "ruleDisabled")
	}
	_, awsErr := store.CreateTopicRule(name, sql, description, actions, ruleDisabled)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleGetTopicRule(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "ruleName")
	if name == "" {
		return jsonErr(service.ErrValidation("ruleName is required."))
	}
	tr, awsErr := store.GetTopicRule(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ruleArn": tr.RuleArn,
		"rule": map[string]any{
			"ruleName":     tr.RuleName,
			"sql":          tr.Sql,
			"description":  tr.Description,
			"actions":      tr.Actions,
			"ruleDisabled": tr.RuleDisabled,
			"createdAt":    tr.CreationDate.Format(time.RFC3339),
		},
	})
}

func handleListTopicRules(store *Store) (*service.Response, error) {
	rules := store.ListTopicRules()
	entries := make([]map[string]any, 0, len(rules))
	for _, tr := range rules {
		entries = append(entries, map[string]any{
			"ruleName":     tr.RuleName,
			"ruleArn":      tr.RuleArn,
			"ruleDisabled": tr.RuleDisabled,
			"createdAt":    tr.CreationDate.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"rules": entries})
}

func handleDeleteTopicRule(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "ruleName")
	if name == "" {
		return jsonErr(service.ErrValidation("ruleName is required."))
	}
	if awsErr := store.DeleteTopicRule(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Tag handlers ----

func handleTagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := parseTags(p)
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tagKeys := getStringSlice(p, "tagKeys")
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"tags": tagsToEntries(tags)})
}
