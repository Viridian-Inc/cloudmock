package authorization

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// AuthorizationService implements first-slice Azure RBAC role assignment APIs.
type AuthorizationService struct {
	mu          sync.RWMutex
	assignments map[string]RoleAssignment
	locks       map[string]ManagementLock
}

func New() *AuthorizationService {
	return &AuthorizationService{
		assignments: make(map[string]RoleAssignment),
		locks:       make(map[string]ManagementLock),
	}
}

func (s *AuthorizationService) Name() string { return "Microsoft.Authorization" }

func (s *AuthorizationService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.Authorization/roleAssignments/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.Authorization/roleAssignments/read"},
		{Name: "List", Method: http.MethodGet, IAMAction: "azure:Microsoft.Authorization/roleAssignments/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Authorization/roleAssignments/delete"},
		{Name: "CreateOrUpdateLock", Method: http.MethodPut, IAMAction: "azure:Microsoft.Authorization/locks/write"},
		{Name: "GetLock", Method: http.MethodGet, IAMAction: "azure:Microsoft.Authorization/locks/read"},
		{Name: "ListLocks", Method: http.MethodGet, IAMAction: "azure:Microsoft.Authorization/locks/read"},
		{Name: "DeleteLock", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Authorization/locks/delete"},
	}
}

func (s *AuthorizationService) HealthCheck() error { return nil }

func (s *AuthorizationService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Authorization/roleAssignments", APIVersion: roleAssignmentsAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Authorization/locks", APIVersion: managementLocksAPIVersion},
	}
}

func (s *AuthorizationService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if providerIndex := authorizationProviderIndex(parts, "roleAssignments"); providerIndex >= 0 {
		return s.handleRoleAssignments(ctx, parts, providerIndex)
	}
	if providerIndex := authorizationProviderIndex(parts, "locks"); providerIndex >= 0 {
		return s.handleManagementLocks(ctx, parts, providerIndex)
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The authorization route is not implemented.")
}

func (s *AuthorizationService) handleRoleAssignments(ctx *service.RequestContext, parts []string, providerIndex int) (*service.Response, error) {
	scope := "/" + strings.Join(parts[:providerIndex], "/")
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listRoleAssignments(scope)
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The role assignment route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateRoleAssignment(scope, name, ctx.Body)
	case http.MethodGet:
		return s.getRoleAssignment(scope, name)
	case http.MethodDelete:
		return s.deleteRoleAssignment(scope, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AuthorizationService) handleManagementLocks(ctx *service.RequestContext, parts []string, providerIndex int) (*service.Response, error) {
	scope := "/" + strings.Join(parts[:providerIndex], "/")
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listManagementLocks(scope)
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The management lock route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateManagementLock(scope, name, ctx.Body)
	case http.MethodGet:
		return s.getManagementLock(scope, name)
	case http.MethodDelete:
		return s.deleteManagementLock(scope, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AuthorizationService) createOrUpdateRoleAssignment(scope, name string, body []byte) (*service.Response, error) {
	var input struct {
		Properties RoleAssignmentProperties `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties.PrincipalID == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "PrincipalIdRequired", "The role assignment principalId is required.")
	}
	if input.Properties.RoleDefinitionID == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "RoleDefinitionIdRequired", "The role assignment roleDefinitionId is required.")
	}

	assignment := RoleAssignment{
		ID:   roleAssignmentID(scope, name),
		Name: name,
		Type: "Microsoft.Authorization/roleAssignments",
		Properties: RoleAssignmentProperties{
			Scope:            scope,
			PrincipalID:      input.Properties.PrincipalID,
			PrincipalType:    input.Properties.PrincipalType,
			RoleDefinitionID: input.Properties.RoleDefinitionID,
			Description:      input.Properties.Description,
			Condition:        input.Properties.Condition,
			ConditionVersion: input.Properties.ConditionVersion,
		},
	}

	key := roleAssignmentKey(scope, name)
	s.mu.Lock()
	_, existed := s.assignments[key]
	s.assignments[key] = assignment
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, assignment)
}

func (s *AuthorizationService) getRoleAssignment(scope, name string) (*service.Response, error) {
	s.mu.RLock()
	assignment, ok := s.assignments[roleAssignmentKey(scope, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "RoleAssignmentNotFound", fmt.Sprintf("Role assignment %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, assignment)
}

func (s *AuthorizationService) listRoleAssignments(scope string) (*service.Response, error) {
	prefix := strings.ToLower(scope) + "/"

	s.mu.RLock()
	values := make([]RoleAssignment, 0)
	for key, assignment := range s.assignments {
		if strings.HasPrefix(key, prefix) {
			values = append(values, assignment)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AuthorizationService) deleteRoleAssignment(scope, name string) (*service.Response, error) {
	key := roleAssignmentKey(scope, name)

	s.mu.Lock()
	assignment, ok := s.assignments[key]
	if ok {
		delete(s.assignments, key)
	}
	s.mu.Unlock()

	if !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	return azurearm.JSONResponse(http.StatusOK, assignment)
}

func (s *AuthorizationService) createOrUpdateManagementLock(scope, name string, body []byte) (*service.Response, error) {
	var input struct {
		Properties ManagementLockProperties `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties.Level == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "LockLevelRequired", "The management lock level is required.")
	}

	lock := ManagementLock{
		ID:   managementLockID(scope, name),
		Name: name,
		Type: "Microsoft.Authorization/locks",
		Properties: ManagementLockProperties{
			Level:  input.Properties.Level,
			Notes:  input.Properties.Notes,
			Owners: input.Properties.Owners,
		},
	}

	key := managementLockKey(scope, name)
	s.mu.Lock()
	_, existed := s.locks[key]
	s.locks[key] = lock
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, lock)
}

func (s *AuthorizationService) getManagementLock(scope, name string) (*service.Response, error) {
	s.mu.RLock()
	lock, ok := s.locks[managementLockKey(scope, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ManagementLockNotFound", fmt.Sprintf("Management lock %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, lock)
}

func (s *AuthorizationService) listManagementLocks(scope string) (*service.Response, error) {
	prefix := strings.ToLower(scope) + "/"

	s.mu.RLock()
	values := make([]ManagementLock, 0)
	for key, lock := range s.locks {
		if strings.HasPrefix(key, prefix) {
			values = append(values, lock)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AuthorizationService) deleteManagementLock(scope, name string) (*service.Response, error) {
	key := managementLockKey(scope, name)

	s.mu.Lock()
	lock, ok := s.locks[key]
	if ok {
		delete(s.locks, key)
	}
	s.mu.Unlock()

	if !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	return azurearm.JSONResponse(http.StatusOK, lock)
}

func authorizationProviderIndex(parts []string, resourceType string) int {
	for i := len(parts) - 3; i >= 0; i-- {
		if strings.EqualFold(parts[i], "providers") &&
			strings.EqualFold(parts[i+1], "Microsoft.Authorization") &&
			strings.EqualFold(parts[i+2], resourceType) {
			return i
		}
	}
	return -1
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	parts := raw[:0]
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func roleAssignmentID(scope, name string) string {
	return scope + "/providers/Microsoft.Authorization/roleAssignments/" + name
}

func roleAssignmentKey(scope, name string) string {
	return strings.ToLower(scope) + "/" + strings.ToLower(name)
}

func managementLockID(scope, name string) string {
	return scope + "/providers/Microsoft.Authorization/locks/" + name
}

func managementLockKey(scope, name string) string {
	return strings.ToLower(scope) + "/" + strings.ToLower(name)
}
