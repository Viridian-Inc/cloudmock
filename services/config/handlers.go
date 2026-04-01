package config

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
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

func ruleToMap(r *ConfigRule) map[string]any {
	m := map[string]any{
		"ConfigRuleName":  r.ConfigRuleName,
		"ConfigRuleArn":   r.ConfigRuleArn,
		"ConfigRuleId":    r.ConfigRuleId,
		"ConfigRuleState": r.ConfigRuleState,
		"Source": map[string]any{
			"Owner":            r.Source.Owner,
			"SourceIdentifier": r.Source.SourceIdentifier,
		},
	}
	if r.Description != "" {
		m["Description"] = r.Description
	}
	if r.InputParameters != "" {
		m["InputParameters"] = r.InputParameters
	}
	if r.MaximumExecutionFrequency != "" {
		m["MaximumExecutionFrequency"] = r.MaximumExecutionFrequency
	}
	if r.Scope != nil {
		scope := map[string]any{}
		if len(r.Scope.ComplianceResourceTypes) > 0 {
			scope["ComplianceResourceTypes"] = r.Scope.ComplianceResourceTypes
		}
		if r.Scope.TagKey != "" {
			scope["TagKey"] = r.Scope.TagKey
		}
		if r.Scope.TagValue != "" {
			scope["TagValue"] = r.Scope.TagValue
		}
		m["Scope"] = scope
	}
	if r.CreatedBy != "" {
		m["CreatedBy"] = r.CreatedBy
	}
	return m
}

func recorderToMap(r *ConfigurationRecorder) map[string]any {
	m := map[string]any{
		"name":    r.Name,
		"roleARN": r.RoleARN,
	}
	if r.RecordingGroup != nil {
		rg := map[string]any{
			"allSupported":               r.RecordingGroup.AllSupported,
			"includeGlobalResourceTypes": r.RecordingGroup.IncludeGlobalResourceTypes,
		}
		if len(r.RecordingGroup.ResourceTypes) > 0 {
			rg["resourceTypes"] = r.RecordingGroup.ResourceTypes
		}
		m["recordingGroup"] = rg
	}
	return m
}

func channelToMap(c *DeliveryChannel) map[string]any {
	m := map[string]any{
		"name":         c.Name,
		"s3BucketName": c.S3BucketName,
	}
	if c.S3KeyPrefix != "" {
		m["s3KeyPrefix"] = c.S3KeyPrefix
	}
	if c.SnsTopicARN != "" {
		m["snsTopicARN"] = c.SnsTopicARN
	}
	if c.ConfigSnapshotDeliveryProperties != nil {
		m["configSnapshotDeliveryProperties"] = map[string]any{
			"deliveryFrequency": c.ConfigSnapshotDeliveryProperties.DeliveryFrequency,
		}
	}
	return m
}

func conformancePackToMap(p *ConformancePack) map[string]any {
	m := map[string]any{
		"ConformancePackName": p.ConformancePackName,
		"ConformancePackArn":  p.ConformancePackArn,
		"ConformancePackId":   p.ConformancePackId,
		"ConformancePackState": p.ConformancePackState,
	}
	if p.DeliveryS3Bucket != "" {
		m["DeliveryS3Bucket"] = p.DeliveryS3Bucket
	}
	if p.DeliveryS3KeyPrefix != "" {
		m["DeliveryS3KeyPrefix"] = p.DeliveryS3KeyPrefix
	}
	return m
}

func parseRuleFromParams(params map[string]any) *ConfigRule {
	ruleMap, ok := params["ConfigRule"].(map[string]any)
	if !ok {
		ruleMap = params
	}

	rule := &ConfigRule{}
	if v, ok := ruleMap["ConfigRuleName"].(string); ok {
		rule.ConfigRuleName = v
	}
	if v, ok := ruleMap["Description"].(string); ok {
		rule.Description = v
	}
	if v, ok := ruleMap["InputParameters"].(string); ok {
		rule.InputParameters = v
	}
	if v, ok := ruleMap["MaximumExecutionFrequency"].(string); ok {
		rule.MaximumExecutionFrequency = v
	}
	if v, ok := ruleMap["ConfigRuleState"].(string); ok {
		rule.ConfigRuleState = v
	}

	if sourceMap, ok := ruleMap["Source"].(map[string]any); ok {
		if v, ok := sourceMap["Owner"].(string); ok {
			rule.Source.Owner = v
		}
		if v, ok := sourceMap["SourceIdentifier"].(string); ok {
			rule.Source.SourceIdentifier = v
		}
	}

	if scopeMap, ok := ruleMap["Scope"].(map[string]any); ok {
		scope := &RuleScope{}
		if v, ok := scopeMap["ComplianceResourceTypes"].([]any); ok {
			for _, item := range v {
				if s, ok := item.(string); ok {
					scope.ComplianceResourceTypes = append(scope.ComplianceResourceTypes, s)
				}
			}
		}
		if v, ok := scopeMap["TagKey"].(string); ok {
			scope.TagKey = v
		}
		if v, ok := scopeMap["TagValue"].(string); ok {
			scope.TagValue = v
		}
		rule.Scope = scope
	}

	return rule
}

func handlePutConfigRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	rule := parseRuleFromParams(params)
	result, awsErr := store.PutConfigRule(rule)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(ruleToMap(result))
}

func handleDescribeConfigRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	var names []string
	if rawNames, ok := params["ConfigRuleNames"].([]any); ok {
		for _, n := range rawNames {
			if s, ok := n.(string); ok {
				names = append(names, s)
			}
		}
	}

	rules := store.GetConfigRules(names)
	items := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		items = append(items, ruleToMap(r))
	}

	return jsonOK(map[string]any{"ConfigRules": items})
}

func handleDeleteConfigRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConfigRuleName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("ConfigRuleName is required."))
	}

	if awsErr := store.DeleteConfigRule(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handlePutConfigurationRecorder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	recMap, _ := params["ConfigurationRecorder"].(map[string]any)
	if recMap == nil {
		recMap = params
	}

	recorder := &ConfigurationRecorder{}
	if v, ok := recMap["name"].(string); ok {
		recorder.Name = v
	}
	if v, ok := recMap["roleARN"].(string); ok {
		recorder.RoleARN = v
	}
	if rgMap, ok := recMap["recordingGroup"].(map[string]any); ok {
		rg := &RecordingGroup{}
		if v, ok := rgMap["allSupported"].(bool); ok {
			rg.AllSupported = v
		}
		if v, ok := rgMap["includeGlobalResourceTypes"].(bool); ok {
			rg.IncludeGlobalResourceTypes = v
		}
		if v, ok := rgMap["resourceTypes"].([]any); ok {
			for _, item := range v {
				if s, ok := item.(string); ok {
					rg.ResourceTypes = append(rg.ResourceTypes, s)
				}
			}
		}
		recorder.RecordingGroup = rg
	}

	if awsErr := store.PutConfigurationRecorder(recorder); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleDescribeConfigurationRecorders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	recorders := store.GetConfigurationRecorders()
	items := make([]map[string]any, 0, len(recorders))
	for _, r := range recorders {
		items = append(items, recorderToMap(r))
	}
	return jsonOK(map[string]any{"ConfigurationRecorders": items})
}

func handleDeleteConfigurationRecorder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConfigurationRecorderName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("ConfigurationRecorderName is required."))
	}

	if awsErr := store.DeleteConfigurationRecorder(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handlePutDeliveryChannel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	chMap, _ := params["DeliveryChannel"].(map[string]any)
	if chMap == nil {
		chMap = params
	}

	channel := &DeliveryChannel{}
	if v, ok := chMap["name"].(string); ok {
		channel.Name = v
	}
	if v, ok := chMap["s3BucketName"].(string); ok {
		channel.S3BucketName = v
	}
	if v, ok := chMap["s3KeyPrefix"].(string); ok {
		channel.S3KeyPrefix = v
	}
	if v, ok := chMap["snsTopicARN"].(string); ok {
		channel.SnsTopicARN = v
	}

	if awsErr := store.PutDeliveryChannel(channel); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleDescribeDeliveryChannels(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	channels := store.GetDeliveryChannels()
	items := make([]map[string]any, 0, len(channels))
	for _, c := range channels {
		items = append(items, channelToMap(c))
	}
	return jsonOK(map[string]any{"DeliveryChannels": items})
}

func handleDeleteDeliveryChannel(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["DeliveryChannelName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("DeliveryChannelName is required."))
	}

	if awsErr := store.DeleteDeliveryChannel(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleStartConfigurationRecorder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConfigurationRecorderName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("ConfigurationRecorderName is required."))
	}

	if awsErr := store.StartRecorder(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleStopConfigurationRecorder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConfigurationRecorderName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("ConfigurationRecorderName is required."))
	}

	if awsErr := store.StopRecorder(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleGetComplianceDetailsByConfigRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	ruleName, _ := params["ConfigRuleName"].(string)
	if ruleName == "" {
		return jsonErr(service.ErrValidation("ConfigRuleName is required."))
	}

	results := store.GetComplianceByRule(ruleName)
	items := make([]map[string]any, 0, len(results))
	for _, r := range results {
		items = append(items, map[string]any{
			"EvaluationResultIdentifier": map[string]any{
				"EvaluationResultQualifier": map[string]any{
					"ConfigRuleName": r.EvaluationResultIdentifier.EvaluationResultQualifier.ConfigRuleName,
					"ResourceType":   r.EvaluationResultIdentifier.EvaluationResultQualifier.ResourceType,
					"ResourceId":     r.EvaluationResultIdentifier.EvaluationResultQualifier.ResourceId,
				},
				"OrderingTimestamp": float64(r.EvaluationResultIdentifier.OrderingTimestamp.Unix()),
			},
			"ComplianceType":        string(r.ComplianceType),
			"ResultRecordedTime":    float64(r.ResultRecordedTime.Unix()),
			"ConfigRuleInvokedTime": float64(r.ConfigRuleInvokedTime.Unix()),
			"Annotation":            r.Annotation,
		})
	}

	return jsonOK(map[string]any{"EvaluationResults": items})
}

func handleDescribeComplianceByConfigRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	rules := store.GetConfigRules(nil)
	items := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		results := store.GetComplianceByRule(r.ConfigRuleName)
		complianceType := "INSUFFICIENT_DATA"
		if len(results) > 0 {
			hasNonCompliant := false
			for _, res := range results {
				if res.ComplianceType == ComplianceNonCompliant {
					hasNonCompliant = true
					break
				}
			}
			if hasNonCompliant {
				complianceType = "NON_COMPLIANT"
			} else {
				complianceType = "COMPLIANT"
			}
		}
		items = append(items, map[string]any{
			"ConfigRuleName": r.ConfigRuleName,
			"Compliance": map[string]any{
				"ComplianceType": complianceType,
			},
		})
	}

	return jsonOK(map[string]any{"ComplianceByConfigRules": items})
}

func handlePutConformancePack(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConformancePackName"].(string)
	s3Bucket, _ := params["DeliveryS3Bucket"].(string)
	s3Prefix, _ := params["DeliveryS3KeyPrefix"].(string)
	templateBody, _ := params["TemplateBody"].(string)

	pack, awsErr := store.PutConformancePack(name, s3Bucket, s3Prefix, templateBody)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"ConformancePackArn": pack.ConformancePackArn})
}

func handleDescribeConformancePacks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	var names []string
	if rawNames, ok := params["ConformancePackNames"].([]any); ok {
		for _, n := range rawNames {
			if s, ok := n.(string); ok {
				names = append(names, s)
			}
		}
	}

	packs := store.GetConformancePacks(names)
	items := make([]map[string]any, 0, len(packs))
	for _, p := range packs {
		items = append(items, conformancePackToMap(p))
	}

	return jsonOK(map[string]any{"ConformancePackDetails": items})
}

func handleDeleteConformancePack(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	name, _ := params["ConformancePackName"].(string)
	if name == "" {
		return jsonErr(service.ErrValidation("ConformancePackName is required."))
	}

	if awsErr := store.DeleteConformancePack(name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handlePutEvaluations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	resultToken, _ := params["ResultToken"].(string)
	_ = resultToken // Used by real AWS to identify the rule

	var evaluations []EvaluationResult
	if rawEvals, ok := params["Evaluations"].([]any); ok {
		now := time.Now().UTC()
		for _, item := range rawEvals {
			evalMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			eval := EvaluationResult{
				ResultRecordedTime:    now,
				ConfigRuleInvokedTime: now,
			}
			if v, ok := evalMap["ComplianceResourceType"].(string); ok {
				eval.EvaluationResultIdentifier.EvaluationResultQualifier.ResourceType = v
			}
			if v, ok := evalMap["ComplianceResourceId"].(string); ok {
				eval.EvaluationResultIdentifier.EvaluationResultQualifier.ResourceId = v
			}
			if v, ok := evalMap["ComplianceType"].(string); ok {
				eval.ComplianceType = ComplianceType(v)
			}
			if v, ok := evalMap["Annotation"].(string); ok {
				eval.Annotation = v
			}
			eval.EvaluationResultIdentifier.OrderingTimestamp = now
			evaluations = append(evaluations, eval)
		}
	}

	// For mock, we key by a placeholder rule name from token
	ruleName := resultToken
	if ruleName == "" {
		ruleName = "unknown"
	}

	store.PutEvaluations(ruleName, evaluations)
	return jsonOK(map[string]any{"FailedEvaluations": []any{}})
}

// handleGetResourceConfigHistory is now implemented in service.go as handleGetResourceConfigHistoryWithStore.
