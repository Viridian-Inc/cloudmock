package tagging

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

func handleGetResources(params map[string]any, store *Store) (*service.Response, error) {
	var filters []TagFilter
	if tfs, ok := params["TagFilters"].([]any); ok {
		for _, tf := range tfs {
			if fm, ok := tf.(map[string]any); ok {
				f := TagFilter{}
				if k, ok := fm["Key"].(string); ok {
					f.Key = k
				}
				if vals, ok := fm["Values"].([]any); ok {
					for _, v := range vals {
						if sv, ok := v.(string); ok {
							f.Values = append(f.Values, sv)
						}
					}
				}
				filters = append(filters, f)
			}
		}
	}

	resourceTypeFilter := ""
	if v, ok := params["ResourceTypeFilters"].([]any); ok && len(v) > 0 {
		if sv, ok := v[0].(string); ok {
			resourceTypeFilter = sv
		}
	}

	entries := store.GetResources(filters, resourceTypeFilter)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		tags := make([]map[string]any, 0, len(entry.Tags))
		for k, v := range entry.Tags {
			tags = append(tags, map[string]any{"Key": k, "Value": v})
		}
		out = append(out, map[string]any{
			"ResourceARN": entry.ARN,
			"Tags":        tags,
		})
	}

	return jsonOK(map[string]any{
		"ResourceTagMappingList": out,
		"PaginationToken":        "",
	})
}

func handleGetTagKeys(store *Store) (*service.Response, error) {
	keys := store.GetTagKeys()
	return jsonOK(map[string]any{
		"TagKeys":         keys,
		"PaginationToken": "",
	})
}

func handleGetTagValues(params map[string]any, store *Store) (*service.Response, error) {
	key := ""
	if v, ok := params["Key"].(string); ok {
		key = v
	}
	if key == "" {
		return jsonErr(service.ErrValidation("Key is required"))
	}
	values := store.GetTagValues(key)
	return jsonOK(map[string]any{
		"TagValues":       values,
		"PaginationToken": "",
	})
}

func handleTagResources(params map[string]any, store *Store) (*service.Response, error) {
	var arns []string
	if v, ok := params["ResourceARNList"].([]any); ok {
		for _, a := range v {
			if sv, ok := a.(string); ok {
				arns = append(arns, sv)
			}
		}
	}

	tags := make(map[string]string)
	if v, ok := params["Tags"].(map[string]any); ok {
		for k, val := range v {
			if sv, ok := val.(string); ok {
				tags[k] = sv
			}
		}
	}

	failedMap := store.TagResources(arns, tags)
	return jsonOK(map[string]any{
		"FailedResourcesMap": failedMap,
	})
}

func handleUntagResources(params map[string]any, store *Store) (*service.Response, error) {
	var arns []string
	if v, ok := params["ResourceARNList"].([]any); ok {
		for _, a := range v {
			if sv, ok := a.(string); ok {
				arns = append(arns, sv)
			}
		}
	}

	var tagKeys []string
	if v, ok := params["TagKeys"].([]any); ok {
		for _, k := range v {
			if sv, ok := k.(string); ok {
				tagKeys = append(tagKeys, sv)
			}
		}
	}

	failedMap := store.UntagResources(arns, tagKeys)
	return jsonOK(map[string]any{
		"FailedResourcesMap": failedMap,
	})
}
