package resourcegroups

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ResourceGroupsService is the cloudmock implementation of the AWS Resource Groups API.
type ResourceGroupsService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ResourceGroupsService for the given AWS account ID and region.
func New(accountID, region string) *ResourceGroupsService {
	return &ResourceGroupsService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *ResourceGroupsService) Name() string { return "resource-groups" }

// Actions returns the list of Resource Groups API actions supported by this service.
func (s *ResourceGroupsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "resource-groups:CreateGroup"},
		{Name: "GetGroup", Method: http.MethodGet, IAMAction: "resource-groups:GetGroup"},
		{Name: "ListGroups", Method: http.MethodGet, IAMAction: "resource-groups:ListGroups"},
		{Name: "DeleteGroup", Method: http.MethodDelete, IAMAction: "resource-groups:DeleteGroup"},
		{Name: "UpdateGroup", Method: http.MethodPut, IAMAction: "resource-groups:UpdateGroup"},
		{Name: "GroupResources", Method: http.MethodPost, IAMAction: "resource-groups:GroupResources"},
		{Name: "UngroupResources", Method: http.MethodPost, IAMAction: "resource-groups:UngroupResources"},
		{Name: "ListGroupResources", Method: http.MethodPost, IAMAction: "resource-groups:ListGroupResources"},
		{Name: "SearchResources", Method: http.MethodPost, IAMAction: "resource-groups:SearchResources"},
		{Name: "Tag", Method: http.MethodPut, IAMAction: "resource-groups:Tag"},
		{Name: "Untag", Method: http.MethodPatch, IAMAction: "resource-groups:Untag"},
		{Name: "GetTags", Method: http.MethodGet, IAMAction: "resource-groups:GetTags"},
	}
}

// HealthCheck always returns nil.
func (s *ResourceGroupsService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Resource Groups request to the appropriate handler.
// Resource Groups uses REST-JSON protocol.
func (s *ResourceGroupsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// POST /groups -> CreateGroup
	// GET /groups -> ListGroups
	if path == "/groups" {
		switch method {
		case http.MethodPost:
			return handleCreateGroup(params, s.store)
		case http.MethodGet:
			return handleListGroups(s.store)
		}
	}

	// POST /group-resources -> GroupResources
	if path == "/group-resources" && method == http.MethodPost {
		return handleGroupResources(params, s.store)
	}
	// POST /ungroup-resources -> UngroupResources
	if path == "/ungroup-resources" && method == http.MethodPost {
		return handleUngroupResources(params, s.store)
	}
	// POST /list-group-resources -> ListGroupResources
	if path == "/list-group-resources" && method == http.MethodPost {
		return handleListGroupResources(params, s.store)
	}
	// POST /resources/search -> SearchResources
	if path == "/resources/search" && method == http.MethodPost {
		return handleSearchResources(params, s.store)
	}

	// /groups/{name}/query -> GetGroupQuery/UpdateGroupQuery (must be before /groups/{name})
	if strings.HasPrefix(path, "/groups/") && strings.HasSuffix(path, "/query") {
		name := strings.TrimPrefix(path, "/groups/")
		name = strings.TrimSuffix(name, "/query")
		switch method {
		case http.MethodGet:
			return handleGetGroupQuery(name, s.store)
		case http.MethodPut:
			return handleUpdateGroupQuery(name, params, s.store)
		}
	}

	// /groups/{name} -> Get/Update/Delete
	if strings.HasPrefix(path, "/groups/") {
		name := strings.TrimPrefix(path, "/groups/")
		switch method {
		case http.MethodGet:
			return handleGetGroup(name, s.store)
		case http.MethodPut:
			return handleUpdateGroup(name, params, s.store)
		case http.MethodDelete:
			return handleDeleteGroup(name, s.store)
		}
	}

	// /resources/{arn}/tags -> Tag/Untag/GetTags
	if strings.HasPrefix(path, "/resources/") && strings.HasSuffix(path, "/tags") {
		arn := strings.TrimPrefix(path, "/resources/")
		arn = strings.TrimSuffix(arn, "/tags")
		switch method {
		case http.MethodGet:
			return handleGetTags(arn, s.store)
		case http.MethodPut:
			return handleTagResource(arn, params, s.store)
		case http.MethodPatch:
			return handleUntagResource(arn, params, s.store)
		case http.MethodDelete:
			return handleUntagResource(arn, params, s.store)
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
