package transfer

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// TransferService is the cloudmock implementation of the AWS Transfer Family API.
type TransferService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new TransferService for the given AWS account ID and region.
func New(accountID, region string) *TransferService {
	return &TransferService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *TransferService) Name() string { return "transfer" }

// Actions returns the list of Transfer Family API actions supported by this service.
func (s *TransferService) Actions() []service.Action {
	return []service.Action{
		// Servers
		{Name: "CreateServer", Method: http.MethodPost, IAMAction: "transfer:CreateServer"},
		{Name: "DescribeServer", Method: http.MethodPost, IAMAction: "transfer:DescribeServer"},
		{Name: "ListServers", Method: http.MethodPost, IAMAction: "transfer:ListServers"},
		{Name: "UpdateServer", Method: http.MethodPost, IAMAction: "transfer:UpdateServer"},
		{Name: "StartServer", Method: http.MethodPost, IAMAction: "transfer:StartServer"},
		{Name: "StopServer", Method: http.MethodPost, IAMAction: "transfer:StopServer"},
		{Name: "DeleteServer", Method: http.MethodPost, IAMAction: "transfer:DeleteServer"},
		// Users
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "transfer:CreateUser"},
		{Name: "DescribeUser", Method: http.MethodPost, IAMAction: "transfer:DescribeUser"},
		{Name: "ListUsers", Method: http.MethodPost, IAMAction: "transfer:ListUsers"},
		{Name: "UpdateUser", Method: http.MethodPost, IAMAction: "transfer:UpdateUser"},
		{Name: "DeleteUser", Method: http.MethodPost, IAMAction: "transfer:DeleteUser"},
		{Name: "ImportSshPublicKey", Method: http.MethodPost, IAMAction: "transfer:ImportSshPublicKey"},
		{Name: "DeleteSshPublicKey", Method: http.MethodPost, IAMAction: "transfer:DeleteSshPublicKey"},
		// Workflows
		{Name: "CreateWorkflow", Method: http.MethodPost, IAMAction: "transfer:CreateWorkflow"},
		{Name: "DescribeWorkflow", Method: http.MethodPost, IAMAction: "transfer:DescribeWorkflow"},
		{Name: "ListWorkflows", Method: http.MethodPost, IAMAction: "transfer:ListWorkflows"},
		{Name: "DeleteWorkflow", Method: http.MethodPost, IAMAction: "transfer:DeleteWorkflow"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "transfer:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "transfer:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "transfer:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *TransferService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Transfer request to the appropriate handler.
func (s *TransferService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}

	switch ctx.Action {
	case "CreateServer":
		return handleCreateServer(params, s.store)
	case "DescribeServer":
		return handleDescribeServer(params, s.store)
	case "ListServers":
		return handleListServers(s.store)
	case "UpdateServer":
		return handleUpdateServer(params, s.store)
	case "StartServer":
		return handleStartServer(params, s.store)
	case "StopServer":
		return handleStopServer(params, s.store)
	case "DeleteServer":
		return handleDeleteServer(params, s.store)
	case "CreateUser":
		return handleCreateUser(params, s.store)
	case "DescribeUser":
		return handleDescribeUser(params, s.store)
	case "ListUsers":
		return handleListUsers(params, s.store)
	case "UpdateUser":
		return handleUpdateUser(params, s.store)
	case "DeleteUser":
		return handleDeleteUser(params, s.store)
	case "ImportSshPublicKey":
		return handleImportSSHPublicKey(params, s.store)
	case "DeleteSshPublicKey":
		return handleDeleteSSHPublicKey(params, s.store)
	case "CreateWorkflow":
		return handleCreateWorkflow(params, s.store)
	case "DescribeWorkflow":
		return handleDescribeWorkflow(params, s.store)
	case "ListWorkflows":
		return handleListWorkflows(s.store)
	case "DeleteWorkflow":
		return handleDeleteWorkflow(params, s.store)
	case "TagResource":
		return handleTagResource(params, s.store)
	case "UntagResource":
		return handleUntagResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
