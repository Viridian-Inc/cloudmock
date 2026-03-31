package organizations

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
		return service.NewAWSError("InvalidInputException",
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

func orgToMap(o *Organization) map[string]any {
	pts := make([]map[string]any, 0, len(o.AvailablePolicyTypes))
	for _, pt := range o.AvailablePolicyTypes {
		pts = append(pts, map[string]any{"Type": pt.Type, "Status": pt.Status})
	}
	return map[string]any{
		"Id":                   o.Id,
		"Arn":                  o.Arn,
		"MasterAccountId":      o.MasterAccountId,
		"MasterAccountArn":     o.MasterAccountArn,
		"MasterAccountEmail":   o.MasterAccountEmail,
		"FeatureSet":           o.FeatureSet,
		"AvailablePolicyTypes": pts,
	}
}

func rootToMap(r *Root) map[string]any {
	pts := make([]map[string]any, 0, len(r.PolicyTypes))
	for _, pt := range r.PolicyTypes {
		pts = append(pts, map[string]any{"Type": pt.Type, "Status": pt.Status})
	}
	return map[string]any{
		"Id":          r.Id,
		"Arn":         r.Arn,
		"Name":        r.Name,
		"PolicyTypes": pts,
	}
}

func ouToMap(ou *OrganizationalUnit) map[string]any {
	return map[string]any{
		"Id":   ou.Id,
		"Arn":  ou.Arn,
		"Name": ou.Name,
	}
}

func accountToMap(a *Account) map[string]any {
	return map[string]any{
		"Id":              a.Id,
		"Arn":             a.Arn,
		"Name":            a.Name,
		"Email":           a.Email,
		"Status":          a.Status,
		"JoinedMethod":    a.JoinedMethod,
		"JoinedTimestamp": float64(a.JoinedTimestamp.Unix()),
	}
}

func policyToMap(p *Policy) map[string]any {
	return map[string]any{
		"PolicySummary": map[string]any{
			"Id":          p.PolicySummary.Id,
			"Arn":         p.PolicySummary.Arn,
			"Name":        p.PolicySummary.Name,
			"Description": p.PolicySummary.Description,
			"Type":        p.PolicySummary.Type,
			"AwsManaged":  p.PolicySummary.AwsManaged,
		},
		"Content": p.Content,
	}
}

func handleCreateOrganization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	featureSet, _ := params["FeatureSet"].(string)
	org, awsErr := store.CreateOrganization(featureSet)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Organization": orgToMap(org)})
}

func handleDescribeOrganization(_ *service.RequestContext, store *Store) (*service.Response, error) {
	org, awsErr := store.GetOrganization()
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Organization": orgToMap(org)})
}

func handleDeleteOrganization(_ *service.RequestContext, store *Store) (*service.Response, error) {
	if awsErr := store.DeleteOrganization(); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListRoots(_ *service.RequestContext, store *Store) (*service.Response, error) {
	roots := store.ListRoots()
	items := make([]map[string]any, 0, len(roots))
	for _, r := range roots {
		items = append(items, rootToMap(r))
	}
	return jsonOK(map[string]any{"Roots": items})
}

func handleCreateOrganizationalUnit(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	parentID, _ := params["ParentId"].(string)
	name, _ := params["Name"].(string)
	ou, awsErr := store.CreateOU(parentID, name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"OrganizationalUnit": ouToMap(ou)})
}

func handleDescribeOrganizationalUnit(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	ouID, _ := params["OrganizationalUnitId"].(string)
	ou, awsErr := store.GetOU(ouID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"OrganizationalUnit": ouToMap(ou)})
}

func handleListOrganizationalUnitsForParent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	parentID, _ := params["ParentId"].(string)
	ous := store.ListOUsForParent(parentID)
	items := make([]map[string]any, 0, len(ous))
	for _, ou := range ous {
		items = append(items, ouToMap(ou))
	}
	return jsonOK(map[string]any{"OrganizationalUnits": items})
}

func handleDeleteOrganizationalUnit(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	ouID, _ := params["OrganizationalUnitId"].(string)
	if awsErr := store.DeleteOU(ouID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["AccountName"].(string)
	email, _ := params["Email"].(string)
	acct, awsErr := store.CreateAccount(name, email)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"CreateAccountStatus": map[string]any{
			"Id":              newUUID(),
			"AccountName":     acct.Name,
			"State":           "SUCCEEDED",
			"AccountId":       acct.Id,
			"RequestedTimestamp": float64(acct.JoinedTimestamp.Unix()),
			"CompletedTimestamp": float64(acct.JoinedTimestamp.Unix()),
		},
	})
}

func handleDescribeAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountID, _ := params["AccountId"].(string)
	acct, awsErr := store.GetAccount(accountID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Account": accountToMap(acct)})
}

func handleListAccounts(_ *service.RequestContext, store *Store) (*service.Response, error) {
	accts := store.ListAccounts()
	items := make([]map[string]any, 0, len(accts))
	for _, a := range accts {
		items = append(items, accountToMap(a))
	}
	return jsonOK(map[string]any{"Accounts": items})
}

func handleListAccountsForParent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	parentID, _ := params["ParentId"].(string)
	accts := store.ListAccountsForParent(parentID)
	items := make([]map[string]any, 0, len(accts))
	for _, a := range accts {
		items = append(items, accountToMap(a))
	}
	return jsonOK(map[string]any{"Accounts": items})
}

func handleMoveAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountID, _ := params["AccountId"].(string)
	sourceParentID, _ := params["SourceParentId"].(string)
	destParentID, _ := params["DestinationParentId"].(string)
	if awsErr := store.MoveAccount(accountID, sourceParentID, destParentID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreatePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	content, _ := params["Content"].(string)
	policyType, _ := params["Type"].(string)
	var tags []Tag
	if rawTags, ok := params["Tags"].([]any); ok {
		tags = parseTags(rawTags)
	}
	policy, awsErr := store.CreatePolicy(name, description, content, policyType, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Policy": policyToMap(policy)})
}

func handleDescribePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	policy, awsErr := store.GetPolicy(policyID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Policy": policyToMap(policy)})
}

func handleListPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	filter, _ := params["Filter"].(string)
	policies := store.ListPolicies(filter)
	items := make([]map[string]any, 0, len(policies))
	for _, p := range policies {
		items = append(items, map[string]any{
			"Id":          p.PolicySummary.Id,
			"Arn":         p.PolicySummary.Arn,
			"Name":        p.PolicySummary.Name,
			"Description": p.PolicySummary.Description,
			"Type":        p.PolicySummary.Type,
			"AwsManaged":  p.PolicySummary.AwsManaged,
		})
	}
	return jsonOK(map[string]any{"Policies": items})
}

func handleUpdatePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	name, _ := params["Name"].(string)
	description, _ := params["Description"].(string)
	content, _ := params["Content"].(string)
	policy, awsErr := store.UpdatePolicy(policyID, name, description, content)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Policy": policyToMap(policy)})
}

func handleDeletePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	if awsErr := store.DeletePolicy(policyID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAttachPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	targetID, _ := params["TargetId"].(string)
	if awsErr := store.AttachPolicy(policyID, targetID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDetachPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	targetID, _ := params["TargetId"].(string)
	if awsErr := store.DetachPolicy(policyID, targetID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTargetsForPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	policyID, _ := params["PolicyId"].(string)
	targets, awsErr := store.ListTargetsForPolicy(policyID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		items = append(items, map[string]any{
			"TargetId": t.TargetId,
			"Arn":      t.Arn,
			"Name":     t.Name,
			"Type":     t.Type,
		})
	}
	return jsonOK(map[string]any{"Targets": items})
}

func handleEnablePolicyType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	rootID, _ := params["RootId"].(string)
	policyType, _ := params["PolicyType"].(string)
	root, awsErr := store.EnablePolicyType(rootID, policyType)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Root": rootToMap(root)})
}

func handleDisablePolicyType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	rootID, _ := params["RootId"].(string)
	policyType, _ := params["PolicyType"].(string)
	root, awsErr := store.DisablePolicyType(rootID, policyType)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Root": rootToMap(root)})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID, _ := params["ResourceId"].(string)
	var tags []Tag
	if rawTags, ok := params["Tags"].([]any); ok {
		tags = parseTags(rawTags)
	}
	if awsErr := store.TagResource(resourceID, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID, _ := params["ResourceId"].(string)
	var tagKeys []string
	if rawKeys, ok := params["TagKeys"].([]any); ok {
		for _, k := range rawKeys {
			if s, ok := k.(string); ok {
				tagKeys = append(tagKeys, s)
			}
		}
	}
	if awsErr := store.UntagResource(resourceID, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceID, _ := params["ResourceId"].(string)
	tags, awsErr := store.ListTagsForResource(resourceID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": tagsToMaps(tags)})
}
