package ram

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
		if v, ok := m["key"].(string); ok {
			t.Key = v
		} else if v, ok := m["Key"].(string); ok {
			t.Key = v
		}
		if v, ok := m["value"].(string); ok {
			t.Value = v
		} else if v, ok := m["Value"].(string); ok {
			t.Value = v
		}
		tags = append(tags, t)
	}
	return tags
}

func tagsToMaps(tags []Tag) []map[string]string {
	out := make([]map[string]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]string{"key": t.Key, "value": t.Value})
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

func shareToMap(share *ResourceShare) map[string]any {
	return map[string]any{
		"resourceShareArn":        share.ResourceShareArn,
		"name":                    share.Name,
		"owningAccountId":         share.OwningAccountId,
		"status":                  share.Status,
		"allowExternalPrincipals": share.AllowExternalPrincipals,
		"creationTime":            float64(share.CreationTime.Unix()),
		"lastUpdatedTime":         float64(share.LastUpdatedTime.Unix()),
		"featureSet":              share.FeatureSet,
		"tags":                    tagsToMaps(share.Tags),
	}
}

func assocToMap(a ResourceShareAssociation) map[string]any {
	return map[string]any{
		"resourceShareArn": a.ResourceShareArn,
		"associatedEntity": a.AssociatedEntity,
		"associationType":  a.AssociationType,
		"status":           a.Status,
		"creationTime":     float64(a.CreationTime.Unix()),
		"lastUpdatedTime":  float64(a.LastUpdatedTime.Unix()),
		"external":         a.External,
	}
}

func invToMap(inv *ResourceShareInvitation) map[string]any {
	return map[string]any{
		"resourceShareInvitationArn": inv.ResourceShareInvitationArn,
		"resourceShareArn":           inv.ResourceShareArn,
		"resourceShareName":          inv.ResourceShareName,
		"senderAccountId":            inv.SenderAccountId,
		"receiverAccountId":          inv.ReceiverAccountId,
		"status":                     inv.Status,
		"invitationTimestamp":        float64(inv.InvitationTimestamp.Unix()),
	}
}

func handleCreateResourceShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["name"].(string)
	allowExternal := true
	if v, ok := params["allowExternalPrincipals"].(bool); ok {
		allowExternal = v
	}
	var principals []string
	if raw, ok := params["principals"].([]any); ok {
		principals = parseStringSlice(raw)
	}
	var resourceArns []string
	if raw, ok := params["resourceArns"].([]any); ok {
		resourceArns = parseStringSlice(raw)
	}
	var tags []Tag
	if raw, ok := params["tags"].([]any); ok {
		tags = parseTags(raw)
	}
	share, awsErr := store.CreateResourceShare(name, allowExternal, principals, resourceArns, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"resourceShare": shareToMap(share)})
}

func handleGetResourceShares(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	resourceOwner, _ := params["resourceOwner"].(string)
	shares := store.GetResourceShares(resourceOwner)
	items := make([]map[string]any, 0, len(shares))
	for _, share := range shares {
		items = append(items, shareToMap(share))
	}
	return jsonOK(map[string]any{"resourceShares": items})
}

func handleUpdateResourceShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["resourceShareArn"].(string)
	name, _ := params["name"].(string)
	var allowExternal *bool
	if v, ok := params["allowExternalPrincipals"].(bool); ok {
		allowExternal = &v
	}
	share, awsErr := store.UpdateResourceShare(arn, name, allowExternal)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"resourceShare": shareToMap(share)})
}

func handleDeleteResourceShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["resourceShareArn"].(string)
	if awsErr := store.DeleteResourceShare(arn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"returnValue": true})
}

func handleAssociateResourceShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["resourceShareArn"].(string)
	var principals []string
	if raw, ok := params["principals"].([]any); ok {
		principals = parseStringSlice(raw)
	}
	var resourceArns []string
	if raw, ok := params["resourceArns"].([]any); ok {
		resourceArns = parseStringSlice(raw)
	}
	assocs, awsErr := store.AssociateResourceShare(arn, principals, resourceArns)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]map[string]any, 0, len(assocs))
	for _, a := range assocs {
		items = append(items, assocToMap(a))
	}
	return jsonOK(map[string]any{"resourceShareAssociations": items})
}

func handleDisassociateResourceShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["resourceShareArn"].(string)
	var principals []string
	if raw, ok := params["principals"].([]any); ok {
		principals = parseStringSlice(raw)
	}
	var resourceArns []string
	if raw, ok := params["resourceArns"].([]any); ok {
		resourceArns = parseStringSlice(raw)
	}
	assocs, awsErr := store.DisassociateResourceShare(arn, principals, resourceArns)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]map[string]any, 0, len(assocs))
	for _, a := range assocs {
		items = append(items, assocToMap(a))
	}
	return jsonOK(map[string]any{"resourceShareAssociations": items})
}

func handleGetResourceShareAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	associationType, _ := params["associationType"].(string)
	var shareArns []string
	if raw, ok := params["resourceShareArns"].([]any); ok {
		shareArns = parseStringSlice(raw)
	}
	assocs := store.GetResourceShareAssociations(associationType, shareArns)
	items := make([]map[string]any, 0, len(assocs))
	for _, a := range assocs {
		items = append(items, assocToMap(a))
	}
	return jsonOK(map[string]any{"resourceShareAssociations": items})
}

func handleGetResourceShareInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	invitations := store.GetResourceShareInvitations()
	items := make([]map[string]any, 0, len(invitations))
	for _, inv := range invitations {
		items = append(items, invToMap(inv))
	}
	return jsonOK(map[string]any{"resourceShareInvitations": items})
}

func handleAcceptResourceShareInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	invArn, _ := params["resourceShareInvitationArn"].(string)
	inv, awsErr := store.AcceptResourceShareInvitation(invArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"resourceShareInvitation": invToMap(inv)})
}

func handleRejectResourceShareInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	invArn, _ := params["resourceShareInvitationArn"].(string)
	inv, awsErr := store.RejectResourceShareInvitation(invArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"resourceShareInvitation": invToMap(inv)})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn, _ := params["resourceShareArn"].(string)
	var tags []Tag
	if raw, ok := params["tags"].([]any); ok {
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
	arn, _ := params["resourceShareArn"].(string)
	var tagKeys []string
	if raw, ok := params["tagKeys"].([]any); ok {
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
	arn, _ := params["resourceArn"].(string)
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"tags": tagsToMaps(tags)})
}

func handleListResources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceOwner, _ := params["resourceOwner"].(string)
	if resourceOwner == "" {
		resourceOwner = "SELF"
	}
	var shareArns []string
	if raw, ok := params["resourceShareArns"].([]any); ok {
		shareArns = parseStringSlice(raw)
	}
	resources := store.ListResources(resourceOwner, shareArns)
	items := make([]map[string]any, 0, len(resources))
	for _, r := range resources {
		items = append(items, assocToMap(r))
	}
	return jsonOK(map[string]any{"resources": items})
}

func handleListPrincipals(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceOwner, _ := params["resourceOwner"].(string)
	if resourceOwner == "" {
		resourceOwner = "SELF"
	}
	var shareArns []string
	if raw, ok := params["resourceShareArns"].([]any); ok {
		shareArns = parseStringSlice(raw)
	}
	principals := store.ListPrincipals(resourceOwner, shareArns)
	items := make([]map[string]any, 0, len(principals))
	for _, p := range principals {
		items = append(items, map[string]any{
			"resourceShareArn": p.ResourceShareArn,
			"id":               p.AssociatedEntity,
			"status":           p.Status,
			"creationTime":     float64(p.CreationTime.Unix()),
			"lastUpdatedTime":  float64(p.LastUpdatedTime.Unix()),
			"external":         p.External,
		})
	}
	return jsonOK(map[string]any{"principals": items})
}

func handleEnableSharingWithAwsOrganization(_ *service.RequestContext, _ *Store) (*service.Response, error) {
	// In mock context, this is always a no-op success.
	return jsonOK(map[string]any{"returnValue": true})
}
