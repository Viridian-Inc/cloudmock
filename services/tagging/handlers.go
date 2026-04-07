package tagging

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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

	var resourceTypeFilters []string
	if v, ok := params["ResourceTypeFilters"].([]any); ok {
		for _, item := range v {
			if sv, ok := item.(string); ok {
				resourceTypeFilters = append(resourceTypeFilters, sv)
			}
		}
	}

	entries := store.GetResources(filters, resourceTypeFilters)
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		tags := make([]map[string]any, 0, len(entry.Tags))
		for k, v := range entry.Tags {
			tags = append(tags, map[string]any{"Key": k, "Value": v})
		}
		out = append(out, map[string]any{
			"ResourceARN":  entry.ARN,
			"Tags":         tags,
			"ComplianceDetails": map[string]any{
				"NoncompliantKeys":    []string{},
				"KeysWithNoncompliantValues": []string{},
				"ComplianceStatus":    true,
			},
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
	if len(arns) == 0 {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"ResourceARNList must contain at least one ARN.", http.StatusBadRequest))
	}

	tags := make(map[string]string)
	if v, ok := params["Tags"].(map[string]any); ok {
		for k, val := range v {
			if sv, ok := val.(string); ok {
				tags[k] = sv
			}
		}
	}
	if len(tags) == 0 {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Tags must contain at least one tag.", http.StatusBadRequest))
	}

	// Validate tag key length (max 128 chars) and value length (max 256 chars).
	for k, v := range tags {
		if len(k) > 128 {
			return jsonErr(service.NewAWSError("InvalidParameterException",
				"Tag key exceeds maximum length of 128 characters.", http.StatusBadRequest))
		}
		if len(v) > 256 {
			return jsonErr(service.NewAWSError("InvalidParameterException",
				"Tag value exceeds maximum length of 256 characters.", http.StatusBadRequest))
		}
		if strings.HasPrefix(k, "aws:") {
			return jsonErr(service.NewAWSError("InvalidParameterException",
				"Tag keys beginning with 'aws:' are reserved.", http.StatusBadRequest))
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

func handleGetComplianceSummary(store *Store) (*service.Response, error) {
	store.mu.RLock()
	totalResources := len(store.resources)
	store.mu.RUnlock()

	return jsonOK(map[string]any{
		"SummaryList": []map[string]any{
			{
				"LastUpdated":        "2024-01-01T00:00:00Z",
				"NonCompliantResources": 0,
				"TargetId":           "",
				"TargetIdType":       "",
				"Region":            store.region,
				"ResourceType":      "",
				"ComplianceStatus": totalResources > 0,
			},
		},
		"PaginationToken": "",
	})
}
