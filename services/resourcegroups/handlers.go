package resourcegroups

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

func str(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func strSlice(params map[string]any, key string) []string {
	if v, ok := params[key].([]any); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func tagsMap(params map[string]any, key string) map[string]string {
	tags := make(map[string]string)
	if v, ok := params[key].(map[string]any); ok {
		for k, val := range v {
			if sv, ok := val.(string); ok {
				tags[k] = sv
			}
		}
	}
	return tags
}

func groupResponse(g *Group) map[string]any {
	resp := map[string]any{
		"GroupArn":    g.GroupArn,
		"Name":        g.Name,
		"Description": g.Description,
	}
	return resp
}

func handleCreateGroup(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}

	var query *ResourceQuery
	if rq, ok := params["ResourceQuery"].(map[string]any); ok {
		query = &ResourceQuery{
			Type:  str(rq, "Type"),
			Query: str(rq, "Query"),
		}
	}

	tags := tagsMap(params, "Tags")
	group, err := store.CreateGroup(name, str(params, "Description"), query, tags)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Group", name))
	}

	resp := map[string]any{
		"Group": groupResponse(group),
		"Tags":  group.Tags,
	}
	if group.ResourceQuery != nil {
		resp["ResourceQuery"] = map[string]any{
			"Type":  group.ResourceQuery.Type,
			"Query": group.ResourceQuery.Query,
		}
	}
	return jsonOK(resp)
}

func handleGetGroup(name string, store *Store) (*service.Response, error) {
	group, ok := store.GetGroup(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", name))
	}
	return jsonOK(map[string]any{"Group": groupResponse(group)})
}

func handleListGroups(store *Store) (*service.Response, error) {
	groups := store.ListGroups()
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, map[string]any{
			"GroupArn":    g.GroupArn,
			"Name":        g.Name,
			"Description": g.Description,
		})
	}
	return jsonOK(map[string]any{
		"GroupIdentifiers": out,
		"Groups":           out,
	})
}

func handleUpdateGroup(name string, params map[string]any, store *Store) (*service.Response, error) {
	group, err := store.UpdateGroup(name, str(params, "Description"))
	if err != nil {
		return jsonErr(service.ErrNotFound("Group", name))
	}
	return jsonOK(map[string]any{"Group": groupResponse(group)})
}

func handleDeleteGroup(name string, store *Store) (*service.Response, error) {
	group, ok := store.DeleteGroup(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", name))
	}
	return jsonOK(map[string]any{"Group": groupResponse(group)})
}

func handleGroupResources(params map[string]any, store *Store) (*service.Response, error) {
	groupName := str(params, "Group")
	arns := strSlice(params, "ResourceArns")
	succeeded, failed := store.GroupResources(groupName, arns)

	failedEntries := make([]map[string]any, 0, len(failed))
	for _, a := range failed {
		failedEntries = append(failedEntries, map[string]any{
			"ResourceArn": a,
			"ErrorCode":   "ALREADY_EXISTS",
			"ErrorMessage": "Resource already in group",
		})
	}
	return jsonOK(map[string]any{
		"Succeeded": succeeded,
		"Failed":    failedEntries,
	})
}

func handleUngroupResources(params map[string]any, store *Store) (*service.Response, error) {
	groupName := str(params, "Group")
	arns := strSlice(params, "ResourceArns")
	succeeded, failed := store.UngroupResources(groupName, arns)

	failedEntries := make([]map[string]any, 0, len(failed))
	for _, a := range failed {
		failedEntries = append(failedEntries, map[string]any{
			"ResourceArn": a,
			"ErrorCode":   "NOT_FOUND",
			"ErrorMessage": "Resource not in group",
		})
	}
	return jsonOK(map[string]any{
		"Succeeded": succeeded,
		"Failed":    failedEntries,
	})
}

func handleListGroupResources(params map[string]any, store *Store) (*service.Response, error) {
	groupName := str(params, "Group")
	if groupName == "" {
		groupName = str(params, "GroupName")
	}
	resources, ok := store.ListGroupResources(groupName)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", groupName))
	}

	out := make([]map[string]any, 0, len(resources))
	for _, arn := range resources {
		out = append(out, map[string]any{
			"Identifier": map[string]any{
				"ResourceArn":  arn,
				"ResourceType": "AWS::Unknown::Resource",
			},
			"Status": map[string]any{"Name": "ATTACHED"},
		})
	}
	return jsonOK(map[string]any{"ResourceIdentifiers": out, "Resources": out})
}

func handleSearchResources(params map[string]any, store *Store) (*service.Response, error) {
	// Return all resources from all groups as a simple mock
	groups := store.ListGroups()
	var out []map[string]any
	for _, g := range groups {
		for _, arn := range g.Resources {
			out = append(out, map[string]any{
				"Identifier": map[string]any{
					"ResourceArn":  arn,
					"ResourceType": "AWS::Unknown::Resource",
				},
			})
		}
	}
	return jsonOK(map[string]any{"ResourceIdentifiers": out})
}

func handleGetTags(arn string, store *Store) (*service.Response, error) {
	tags, ok := store.GetTags(arn)
	if !ok {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonOK(map[string]any{
		"Arn":  arn,
		"Tags": tags,
	})
}

func handleTagResource(arn string, params map[string]any, store *Store) (*service.Response, error) {
	tags := tagsMap(params, "Tags")
	if !store.TagResource(arn, tags) {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonOK(map[string]any{
		"Arn":  arn,
		"Tags": tags,
	})
}

func handleUntagResource(arn string, params map[string]any, store *Store) (*service.Response, error) {
	keys := strSlice(params, "Keys")
	if !store.UntagResource(arn, keys) {
		return jsonErr(service.ErrNotFound("Resource", arn))
	}
	return jsonOK(map[string]any{
		"Arn":  arn,
		"Keys": keys,
	})
}

func handleGetGroupQuery(name string, store *Store) (*service.Response, error) {
	g, ok := store.GetGroupQuery(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", name))
	}
	rq := map[string]any{}
	if g.ResourceQuery != nil {
		rq = map[string]any{"Type": g.ResourceQuery.Type, "Query": g.ResourceQuery.Query}
	}
	return jsonOK(map[string]any{
		"GroupName":     g.Name,
		"GroupArn":      g.GroupArn,
		"ResourceQuery": rq,
	})
}

func handleUpdateGroupQuery(name string, params map[string]any, store *Store) (*service.Response, error) {
	var rq *ResourceQuery
	if qm, ok := params["ResourceQuery"].(map[string]any); ok {
		rq = &ResourceQuery{
			Type:  str(qm, "Type"),
			Query: str(qm, "Query"),
		}
	}
	g, ok := store.UpdateGroupQuery(name, rq)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", name))
	}
	rqOut := map[string]any{}
	if g.ResourceQuery != nil {
		rqOut = map[string]any{"Type": g.ResourceQuery.Type, "Query": g.ResourceQuery.Query}
	}
	return jsonOK(map[string]any{
		"GroupName":     g.Name,
		"GroupArn":      g.GroupArn,
		"ResourceQuery": rqOut,
	})
}
