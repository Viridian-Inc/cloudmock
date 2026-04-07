package ssoadmin

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
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

func psToMap(ps *PermissionSet) map[string]any {
	return map[string]any{
		"PermissionSetArn": ps.PermissionSetArn,
		"Name":             ps.Name,
		"Description":      ps.Description,
		"SessionDuration":  ps.SessionDuration,
		"RelayState":       ps.RelayState,
		"CreatedDate":      float64(ps.CreatedDate.Unix()),
	}
}

func handleCreateInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	inst, awsErr := store.CreateInstance(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"InstanceArn": inst.InstanceArn})
}

func handleProvisionPermissionSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	targetType, _ := params["TargetType"].(string)
	targetID, _ := params["TargetId"].(string)
	if instanceArn == "" || permissionSetArn == "" {
		return jsonErr(service.ErrValidation("InstanceArn and PermissionSetArn are required."))
	}
	result, awsErr := store.ProvisionPermissionSet(instanceArn, permissionSetArn, targetType, targetID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(result)
}

func handleListInstances(_ *service.RequestContext, store *Store) (*service.Response, error) {
	instances := store.ListInstances()
	items := make([]map[string]any, 0, len(instances))
	for _, inst := range instances {
		items = append(items, map[string]any{
			"InstanceArn":     inst.InstanceArn,
			"IdentityStoreId": inst.IdentityStoreId,
			"Name":            inst.Name,
			"Status":          inst.Status,
		})
	}
	return jsonOK(map[string]any{"Instances": items})
}

func handleDescribeInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["InstanceArn"].(string)
	inst, awsErr := store.DescribeInstance(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"InstanceArn":     inst.InstanceArn,
		"IdentityStoreId": inst.IdentityStoreId,
		"Name":            inst.Name,
		"Status":          inst.Status,
		"CreatedDate":     float64(inst.CreatedDate.Unix()),
	})
}

func handleCreatePermissionSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	sessionDuration, _ := params["SessionDuration"].(string)
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	ps, awsErr := store.CreatePermissionSet(instanceArn, name, description, sessionDuration, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"PermissionSet": psToMap(ps)})
}

func handleDescribePermissionSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	ps, awsErr := store.DescribePermissionSet(instanceArn, permissionSetArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"PermissionSet": psToMap(ps)})
}

func handleListPermissionSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	instanceArn, _ := params["InstanceArn"].(string)
	psets := store.ListPermissionSets(instanceArn)
	arns := make([]string, 0, len(psets))
	for _, ps := range psets {
		arns = append(arns, ps.PermissionSetArn)
	}
	return jsonOK(map[string]any{"PermissionSets": arns})
}

func handleUpdatePermissionSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	description, _ := params["Description"].(string)
	sessionDuration, _ := params["SessionDuration"].(string)
	relayState, _ := params["RelayState"].(string)
	if awsErr := store.UpdatePermissionSet(instanceArn, permissionSetArn, description, sessionDuration, relayState); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeletePermissionSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	if awsErr := store.DeletePermissionSet(instanceArn, permissionSetArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateAccountAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	targetId, _ := params["TargetId"].(string)
	targetType, _ := params["TargetType"].(string)
	principalId, _ := params["PrincipalId"].(string)
	principalType, _ := params["PrincipalType"].(string)
	if awsErr := store.CreateAccountAssignment(instanceArn, permissionSetArn, targetId, targetType, principalId, principalType); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"AccountAssignmentCreationStatus": map[string]any{
			"Status":           "SUCCEEDED",
			"PermissionSetArn": permissionSetArn,
			"TargetId":         targetId,
			"TargetType":       targetType,
			"PrincipalId":      principalId,
			"PrincipalType":    principalType,
		},
	})
}

func handleListAccountAssignments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	accountId, _ := params["AccountId"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	assignments := store.ListAccountAssignments(instanceArn, accountId, permissionSetArn)
	items := make([]map[string]any, 0, len(assignments))
	for _, a := range assignments {
		items = append(items, map[string]any{
			"PermissionSetArn": a.PermissionSetArn,
			"AccountId":        a.TargetId,
			"PrincipalId":      a.PrincipalId,
			"PrincipalType":    a.PrincipalType,
		})
	}
	return jsonOK(map[string]any{"AccountAssignments": items})
}

func handleDeleteAccountAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	targetId, _ := params["TargetId"].(string)
	targetType, _ := params["TargetType"].(string)
	principalId, _ := params["PrincipalId"].(string)
	principalType, _ := params["PrincipalType"].(string)
	if awsErr := store.DeleteAccountAssignment(instanceArn, permissionSetArn, targetId, targetType, principalId, principalType); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"AccountAssignmentDeletionStatus": map[string]any{
			"Status": "SUCCEEDED",
		},
	})
}

func handleAttachManagedPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	policyArn, _ := params["ManagedPolicyArn"].(string)
	if awsErr := store.AttachManagedPolicy(instanceArn, permissionSetArn, policyArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDetachManagedPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	policyArn, _ := params["ManagedPolicyArn"].(string)
	if awsErr := store.DetachManagedPolicy(instanceArn, permissionSetArn, policyArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListManagedPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	policies, awsErr := store.ListManagedPolicies(instanceArn, permissionSetArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]map[string]any, 0, len(policies))
	for _, mp := range policies {
		items = append(items, map[string]any{"Arn": mp.Arn, "Name": mp.Name})
	}
	return jsonOK(map[string]any{"AttachedManagedPolicies": items})
}

func handlePutInlinePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	policy, _ := params["InlinePolicy"].(string)
	if awsErr := store.PutInlinePolicy(instanceArn, permissionSetArn, policy); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleGetInlinePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	policy, awsErr := store.GetInlinePolicy(instanceArn, permissionSetArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"InlinePolicy": policy})
}

func handleDeleteInlinePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	instanceArn, _ := params["InstanceArn"].(string)
	permissionSetArn, _ := params["PermissionSetArn"].(string)
	if awsErr := store.DeleteInlinePolicy(instanceArn, permissionSetArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["ResourceArn"].(string)
	if arn == "" {
		arn, _ = params["InstanceArn"].(string)
	}
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
	arn, _ := params["ResourceArn"].(string)
	if arn == "" {
		arn, _ = params["InstanceArn"].(string)
	}
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
	arn, _ := params["ResourceArn"].(string)
	if arn == "" {
		arn, _ = params["InstanceArn"].(string)
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": tagsToMaps(tags)})
}
