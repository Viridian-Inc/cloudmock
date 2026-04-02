package shield

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

func handleCreateProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	name, _ := params["Name"].(string)
	resourceArn, _ := params["ResourceArn"].(string)
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	p, awsErr := store.CreateProtection(name, resourceArn, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ProtectionId": p.Id})
}

func handleDescribeProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionId"].(string)
	p, awsErr := store.GetProtection(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Protection": map[string]any{
			"Id": p.Id, "Name": p.Name, "ResourceArn": p.ResourceArn, "ProtectionArn": p.ProtectionArn,
		},
	})
}

func handleListProtections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	protections := store.ListProtections()
	items := make([]map[string]any, 0, len(protections))
	for _, p := range protections {
		items = append(items, map[string]any{
			"Id": p.Id, "Name": p.Name, "ResourceArn": p.ResourceArn, "ProtectionArn": p.ProtectionArn,
		})
	}
	return jsonOK(map[string]any{"Protections": items})
}

func handleDeleteProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionId"].(string)
	if awsErr := store.DeleteProtection(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	_, awsErr := store.CreateSubscription()
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	sub, awsErr := store.GetSubscription()
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	limits := make([]map[string]any, 0, len(sub.Limits))
	for _, l := range sub.Limits {
		limits = append(limits, map[string]any{"Type": l.Type, "Max": l.Max})
	}
	return jsonOK(map[string]any{
		"Subscription": map[string]any{
			"StartTime":                   float64(sub.StartTime.Unix()),
			"EndTime":                     float64(sub.EndTime.Unix()),
			"TimeCommitmentInSeconds":     sub.TimeCommitmentInSeconds,
			"AutoRenew":                   sub.AutoRenew,
			"Limits":                      limits,
			"ProactiveEngagementStatus":   sub.ProactiveEngagementStatus,
			"SubscriptionState":           sub.SubscriptionState,
			"SubscriptionArn":             sub.SubscriptionArn,
		},
	})
}

func handleDescribeAttack(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	attackId, _ := params["AttackId"].(string)
	a, awsErr := store.GetAttack(attackId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	vectors := make([]map[string]any, 0, len(a.AttackVectors))
	for _, v := range a.AttackVectors {
		vectors = append(vectors, map[string]any{"VectorType": v.VectorType})
	}
	result := map[string]any{
		"AttackId":      a.AttackId,
		"ResourceArn":   a.ResourceArn,
		"StartTime":     float64(a.StartTime.Unix()),
		"AttackVectors": vectors,
	}
	if a.EndTime != nil {
		result["EndTime"] = float64(a.EndTime.Unix())
	}
	return jsonOK(map[string]any{"Attack": result})
}

func handleListAttacks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	attacks := store.ListAttacks()
	items := make([]map[string]any, 0, len(attacks))
	for _, a := range attacks {
		m := map[string]any{
			"AttackId":    a.AttackId,
			"ResourceArn": a.ResourceArn,
			"StartTime":   float64(a.StartTime.Unix()),
		}
		if a.EndTime != nil {
			m["EndTime"] = float64(a.EndTime.Unix())
		}
		items = append(items, m)
	}
	return jsonOK(map[string]any{"AttackSummaries": items})
}

func handleCreateProtectionGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionGroupId"].(string)
	aggregation, _ := params["Aggregation"].(string)
	pattern, _ := params["Pattern"].(string)
	resourceType, _ := params["ResourceType"].(string)
	var members []string
	if raw, ok := params["Members"].([]any); ok {
		members = parseStringSlice(raw)
	}
	var tags []Tag
	if raw, ok := params["Tags"].([]any); ok {
		tags = parseTags(raw)
	}
	_, awsErr := store.CreateProtectionGroup(id, aggregation, pattern, resourceType, members, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeProtectionGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionGroupId"].(string)
	pg, awsErr := store.GetProtectionGroup(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ProtectionGroup": map[string]any{
			"ProtectionGroupId": pg.ProtectionGroupId, "Aggregation": pg.Aggregation,
			"Pattern": pg.Pattern, "ResourceType": pg.ResourceType,
			"Members": pg.Members, "ProtectionGroupArn": pg.ProtectionGroupArn,
		},
	})
}

func handleListProtectionGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	pgs := store.ListProtectionGroups()
	items := make([]map[string]any, 0, len(pgs))
	for _, pg := range pgs {
		items = append(items, map[string]any{
			"ProtectionGroupId": pg.ProtectionGroupId, "Aggregation": pg.Aggregation,
			"Pattern": pg.Pattern, "ProtectionGroupArn": pg.ProtectionGroupArn,
		})
	}
	return jsonOK(map[string]any{"ProtectionGroups": items})
}

func handleUpdateProtectionGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionGroupId"].(string)
	aggregation, _ := params["Aggregation"].(string)
	pattern, _ := params["Pattern"].(string)
	resourceType, _ := params["ResourceType"].(string)
	var members []string
	if raw, ok := params["Members"].([]any); ok {
		members = parseStringSlice(raw)
	}
	if awsErr := store.UpdateProtectionGroup(id, aggregation, pattern, resourceType, members); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteProtectionGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["ProtectionGroupId"].(string)
	if awsErr := store.DeleteProtectionGroup(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	autoRenew, _ := params["AutoRenew"].(string)
	if awsErr := store.UpdateSubscription(autoRenew); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeAttackStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	stats := store.DescribeAttackStatistics()
	return jsonOK(map[string]any{
		"TimeRange": map[string]any{
			"FromInclusive": float64(stats.FromInclusive.Unix()),
			"ToExclusive":   float64(stats.ToExclusive.Unix()),
		},
		"DataItems": stats.DataItems,
	})
}

func handleDescribeDRTAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	drt := store.DescribeDRTAccess()
	return jsonOK(map[string]any{
		"RoleArn":        drt.RoleArn,
		"LogBucketList":  drt.LogBucketList,
	})
}

func handleEnableApplicationLayerAutomaticResponse(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	action := map[string]any{}
	if a, ok := params["Action"].(map[string]any); ok {
		action = a
	}
	if awsErr := store.EnableApplicationLayerAutomaticResponse(resourceArn, action); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDisableApplicationLayerAutomaticResponse(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn, _ := params["ResourceArn"].(string)
	if awsErr := store.DisableApplicationLayerAutomaticResponse(resourceArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAssociateHealthCheck(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	protectionID, _ := params["ProtectionId"].(string)
	healthCheckArn, _ := params["HealthCheckArn"].(string)
	if awsErr := store.AssociateHealthCheck(protectionID, healthCheckArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDisassociateHealthCheck(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	protectionID, _ := params["ProtectionId"].(string)
	healthCheckArn, _ := params["HealthCheckArn"].(string)
	if awsErr := store.DisassociateHealthCheck(protectionID, healthCheckArn); awsErr != nil {
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
	return jsonOK(map[string]any{"Tags": tagsToMaps(tags)})
}
