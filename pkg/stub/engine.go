package stub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// StubService is a generic AWS service implementation driven by a ServiceModel.
// It can emulate any AWS service by interpreting its model definition at runtime.
type StubService struct {
	model     *ServiceModel
	store     *ResourceStore
	accountID string
	region    string
}

// NewStubService creates a StubService from a model definition.
func NewStubService(model *ServiceModel, accountID, region string) *StubService {
	return &StubService{
		model:     model,
		store:     NewResourceStore(),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *StubService) Name() string {
	return s.model.ServiceName
}

// Actions returns the list of actions this stub service supports.
func (s *StubService) Actions() []service.Action {
	actions := make([]service.Action, 0, len(s.model.Actions))
	for _, a := range s.model.Actions {
		actions = append(actions, service.Action{
			Name:      a.Name,
			Method:    http.MethodPost,
			IAMAction: s.model.ServiceName + ":" + a.Name,
		})
	}
	return actions
}

// HealthCheck always returns nil.
func (s *StubService) HealthCheck() error { return nil }

// HandleRequest routes an incoming request to a generic handler based on the
// action's Type field in the service model.
func (s *StubService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action, ok := s.model.Actions[ctx.Action]
	if !ok {
		return s.errorResponse(),
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}

	params, err := s.parseInput(ctx)
	if err != nil {
		return s.errorResponse(),
			service.NewAWSError("InvalidParameterValue", err.Error(), http.StatusBadRequest)
	}

	// Validate required fields.
	for _, f := range action.InputFields {
		if f.Required {
			if _, exists := params[f.Name]; !exists {
				return s.errorResponse(),
					service.NewAWSError("MissingParameter",
						fmt.Sprintf("The request must contain the parameter %s", f.Name),
						http.StatusBadRequest)
			}
		}
	}

	switch action.Type {
	case "create":
		return s.handleCreate(action, params)
	case "describe":
		return s.handleDescribe(action, params)
	case "list":
		return s.handleList(action)
	case "delete":
		return s.handleDelete(action, params)
	case "update":
		return s.handleUpdate(action, params)
	case "tag":
		return s.handleTag(action, params)
	case "untag":
		return s.handleUntag(action, params)
	case "listTags":
		return s.handleListTags(action, params)
	case "other":
		return s.successResponse(map[string]interface{}{}), nil
	default:
		return s.successResponse(map[string]interface{}{}), nil
	}
}

// handleCreate generates an ID, stores the resource, and returns the ID and ARN.
func (s *StubService) handleCreate(action Action, params map[string]interface{}) (*service.Response, error) {
	rt, ok := s.model.ResourceTypes[action.ResourceType]
	if !ok {
		return s.successResponse(map[string]interface{}{}), nil
	}

	// Determine ID prefix from the resource type name (lowercase).
	prefix := strings.ToLower(rt.Name)
	id := s.store.Create(action.ResourceType, prefix, params)

	arn := BuildARN(rt.ArnPattern, s.region, s.accountID, id)

	// Store ID and ARN inside the resource.
	_ = s.store.Update(action.ResourceType, id, map[string]interface{}{
		rt.IdField:  id,
		"Arn":       arn,
		"CreatedAt": time.Now().UTC().Format(time.RFC3339),
	})

	result := map[string]interface{}{
		rt.IdField: id,
		"Arn":      arn,
	}
	// Echo back output fields.
	for _, f := range action.OutputFields {
		if v, exists := params[f.Name]; exists {
			result[f.Name] = v
		}
	}

	return s.successResponse(result), nil
}

// handleDescribe looks up a resource by its ID field.
func (s *StubService) handleDescribe(action Action, params map[string]interface{}) (*service.Response, error) {
	id, err := s.extractID(action, params)
	if err != nil {
		return s.errorResponse(), err
	}

	fields, lookupErr := s.store.Get(action.ResourceType, id)
	if lookupErr != nil {
		return s.errorResponse(),
			service.NewAWSError("ResourceNotFoundException",
				fmt.Sprintf("%s not found: %s", action.ResourceType, id),
				http.StatusNotFound)
	}

	return s.successResponse(fields), nil
}

// handleList returns all resources of the action's resource type.
func (s *StubService) handleList(action Action) (*service.Response, error) {
	items := s.store.List(action.ResourceType)
	rt := s.model.ResourceTypes[action.ResourceType]
	listKey := rt.Name + "s" // simple pluralization
	return s.successResponse(map[string]interface{}{
		listKey: items,
	}), nil
}

// handleDelete removes a resource by ID.
func (s *StubService) handleDelete(action Action, params map[string]interface{}) (*service.Response, error) {
	id, err := s.extractID(action, params)
	if err != nil {
		return s.errorResponse(), err
	}

	if deleteErr := s.store.Delete(action.ResourceType, id); deleteErr != nil {
		return s.errorResponse(),
			service.NewAWSError("ResourceNotFoundException",
				fmt.Sprintf("%s not found: %s", action.ResourceType, id),
				http.StatusNotFound)
	}

	return s.successResponse(map[string]interface{}{}), nil
}

// handleUpdate merges fields into an existing resource.
func (s *StubService) handleUpdate(action Action, params map[string]interface{}) (*service.Response, error) {
	id, err := s.extractID(action, params)
	if err != nil {
		return s.errorResponse(), err
	}

	// Remove the ID field from the updates.
	updates := make(map[string]interface{}, len(params))
	for k, v := range params {
		if k != action.IdField {
			updates[k] = v
		}
	}

	if updateErr := s.store.Update(action.ResourceType, id, updates); updateErr != nil {
		return s.errorResponse(),
			service.NewAWSError("ResourceNotFoundException",
				fmt.Sprintf("%s not found: %s", action.ResourceType, id),
				http.StatusNotFound)
	}

	// Return the updated resource.
	fields, _ := s.store.Get(action.ResourceType, id)
	return s.successResponse(fields), nil
}

// handleTag adds tags to a resource.
func (s *StubService) handleTag(action Action, params map[string]interface{}) (*service.Response, error) {
	arnStr, _ := params["ResourceArn"].(string)
	if arnStr == "" {
		// Try Arn field as well
		arnStr, _ = params["Arn"].(string)
	}
	if arnStr == "" {
		return s.errorResponse(),
			service.NewAWSError("InvalidParameterValue", "ResourceArn is required", http.StatusBadRequest)
	}

	tags := extractTags(params)
	s.store.Tag(arnStr, tags)

	return s.successResponse(map[string]interface{}{}), nil
}

// handleUntag removes tags from a resource.
func (s *StubService) handleUntag(action Action, params map[string]interface{}) (*service.Response, error) {
	arnStr, _ := params["ResourceArn"].(string)
	if arnStr == "" {
		arnStr, _ = params["Arn"].(string)
	}
	if arnStr == "" {
		return s.errorResponse(),
			service.NewAWSError("InvalidParameterValue", "ResourceArn is required", http.StatusBadRequest)
	}

	keys := extractTagKeys(params)
	s.store.Untag(arnStr, keys)

	return s.successResponse(map[string]interface{}{}), nil
}

// handleListTags returns tags for a resource.
func (s *StubService) handleListTags(action Action, params map[string]interface{}) (*service.Response, error) {
	arnStr, _ := params["ResourceArn"].(string)
	if arnStr == "" {
		arnStr, _ = params["Arn"].(string)
	}
	if arnStr == "" {
		return s.errorResponse(),
			service.NewAWSError("InvalidParameterValue", "ResourceArn is required", http.StatusBadRequest)
	}

	tags := s.store.ListTags(arnStr)
	tagList := make([]map[string]string, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, map[string]string{"Key": k, "Value": v})
	}

	return s.successResponse(map[string]interface{}{
		"Tags": tagList,
	}), nil
}

// extractID pulls the resource ID from the request params using the action's IdField.
func (s *StubService) extractID(action Action, params map[string]interface{}) (string, error) {
	idField := action.IdField
	if idField == "" {
		rt := s.model.ResourceTypes[action.ResourceType]
		idField = rt.IdField
	}
	id, ok := params[idField].(string)
	if !ok || id == "" {
		return "", service.NewAWSError("MissingParameter",
			fmt.Sprintf("The request must contain the parameter %s", idField),
			http.StatusBadRequest)
	}
	return id, nil
}

// parseInput extracts input parameters from the request body.
// It supports JSON and form-encoded (query protocol) bodies.
func (s *StubService) parseInput(ctx *service.RequestContext) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if len(ctx.Body) == 0 {
		return params, nil
	}

	switch s.model.Protocol {
	case "json", "rest-json":
		if err := json.Unmarshal(ctx.Body, &params); err != nil {
			return nil, fmt.Errorf("invalid JSON body: %w", err)
		}
	case "query", "rest-xml":
		values, err := url.ParseQuery(string(ctx.Body))
		if err != nil {
			return nil, fmt.Errorf("invalid form body: %w", err)
		}
		for k, v := range values {
			if len(v) == 1 {
				params[k] = v[0]
			} else {
				params[k] = v
			}
		}
	default:
		// Try JSON first, fall back to form.
		if err := json.Unmarshal(ctx.Body, &params); err != nil {
			values, parseErr := url.ParseQuery(string(ctx.Body))
			if parseErr != nil {
				return nil, fmt.Errorf("unable to parse request body")
			}
			for k, v := range values {
				if len(v) == 1 {
					params[k] = v[0]
				} else {
					params[k] = v
				}
			}
		}
	}

	return params, nil
}

// successResponse builds a Response with the correct format for this service's protocol.
func (s *StubService) successResponse(body interface{}) *service.Response {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     s.responseFormat(),
	}
}

// errorResponse returns an empty response shell with the correct format.
func (s *StubService) errorResponse() *service.Response {
	return &service.Response{
		Format: s.responseFormat(),
	}
}

// responseFormat returns the appropriate ResponseFormat for this model's protocol.
func (s *StubService) responseFormat() service.ResponseFormat {
	switch s.model.Protocol {
	case "json", "rest-json":
		return service.FormatJSON
	case "query", "rest-xml":
		return service.FormatXML
	default:
		return service.FormatJSON
	}
}

// extractTags pulls a tag map from the "Tags" parameter in the request.
// It handles both []interface{} (list of {Key,Value} maps) and map[string]interface{}.
func extractTags(params map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	raw, ok := params["Tags"]
	if !ok {
		return tags
	}

	switch t := raw.(type) {
	case []interface{}:
		for _, item := range t {
			if m, ok := item.(map[string]interface{}); ok {
				key, _ := m["Key"].(string)
				value, _ := m["Value"].(string)
				if key != "" {
					tags[key] = value
				}
			}
		}
	case map[string]interface{}:
		for k, v := range t {
			if vs, ok := v.(string); ok {
				tags[k] = vs
			}
		}
	}
	return tags
}

// extractTagKeys pulls a list of tag keys from the "TagKeys" parameter.
func extractTagKeys(params map[string]interface{}) []string {
	raw, ok := params["TagKeys"]
	if !ok {
		return nil
	}
	switch t := raw.(type) {
	case []interface{}:
		keys := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				keys = append(keys, s)
			}
		}
		return keys
	}
	return nil
}
