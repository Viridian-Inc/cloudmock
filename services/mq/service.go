package mq

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// MQService is the cloudmock implementation of the Amazon MQ API.
type MQService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new MQService for the given AWS account ID and region.
func New(accountID, region string) *MQService {
	return &MQService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *MQService) Name() string { return "mq" }

// Actions returns the list of MQ API actions supported by this service.
func (s *MQService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateBroker", Method: http.MethodPost, IAMAction: "mq:CreateBroker"},
		{Name: "DescribeBroker", Method: http.MethodGet, IAMAction: "mq:DescribeBroker"},
		{Name: "ListBrokers", Method: http.MethodGet, IAMAction: "mq:ListBrokers"},
		{Name: "DeleteBroker", Method: http.MethodDelete, IAMAction: "mq:DeleteBroker"},
		{Name: "UpdateBroker", Method: http.MethodPut, IAMAction: "mq:UpdateBroker"},
		{Name: "RebootBroker", Method: http.MethodPost, IAMAction: "mq:RebootBroker"},
		{Name: "CreateConfiguration", Method: http.MethodPost, IAMAction: "mq:CreateConfiguration"},
		{Name: "DescribeConfiguration", Method: http.MethodGet, IAMAction: "mq:DescribeConfiguration"},
		{Name: "ListConfigurations", Method: http.MethodGet, IAMAction: "mq:ListConfigurations"},
		{Name: "UpdateConfiguration", Method: http.MethodPut, IAMAction: "mq:UpdateConfiguration"},
		{Name: "DescribeConfigurationRevision", Method: http.MethodGet, IAMAction: "mq:DescribeConfigurationRevision"},
		{Name: "ListConfigurationRevisions", Method: http.MethodGet, IAMAction: "mq:ListConfigurationRevisions"},
		{Name: "CreateUser", Method: http.MethodPost, IAMAction: "mq:CreateUser"},
		{Name: "DescribeUser", Method: http.MethodGet, IAMAction: "mq:DescribeUser"},
		{Name: "ListUsers", Method: http.MethodGet, IAMAction: "mq:ListUsers"},
		{Name: "UpdateUser", Method: http.MethodPut, IAMAction: "mq:UpdateUser"},
		{Name: "DeleteUser", Method: http.MethodDelete, IAMAction: "mq:DeleteUser"},
		{Name: "CreateTags", Method: http.MethodPost, IAMAction: "mq:CreateTags"},
		{Name: "DeleteTags", Method: http.MethodDelete, IAMAction: "mq:DeleteTags"},
		{Name: "ListTags", Method: http.MethodGet, IAMAction: "mq:ListTags"},
	}
}

// HealthCheck always returns nil.
func (s *MQService) HealthCheck() error { return nil }

// HandleRequest routes an incoming MQ request to the appropriate handler.
func (s *MQService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}
	for k, v := range ctx.Params {
		if _, exists := params[k]; !exists {
			params[k] = v
		}
	}

	switch ctx.Action {
	case "CreateBroker":
		return handleCreateBroker(params, s.store)
	case "DescribeBroker":
		return handleDescribeBroker(params, s.store)
	case "ListBrokers":
		return handleListBrokers(s.store)
	case "DeleteBroker":
		return handleDeleteBroker(params, s.store)
	case "UpdateBroker":
		return handleUpdateBroker(params, s.store)
	case "RebootBroker":
		return handleRebootBroker(params, s.store)
	case "CreateConfiguration":
		return handleCreateConfiguration(params, s.store)
	case "DescribeConfiguration":
		return handleDescribeConfiguration(params, s.store)
	case "ListConfigurations":
		return handleListConfigurations(s.store)
	case "UpdateConfiguration":
		return handleUpdateConfiguration(params, s.store)
	case "DescribeConfigurationRevision":
		return handleDescribeConfigurationRevision(params, s.store)
	case "ListConfigurationRevisions":
		return handleListConfigurationRevisions(params, s.store)
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
	case "CreateTags":
		return handleCreateTags(params, s.store)
	case "DeleteTags":
		return handleDeleteTags(params, s.store)
	case "ListTags":
		return handleListTags(params, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
