package dms

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// DMSService is the cloudmock implementation of the AWS Database Migration Service API.
type DMSService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new DMSService for the given AWS account ID and region.
func New(accountID, region string) *DMSService {
	return &DMSService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *DMSService) Name() string { return "dms" }

// Actions returns the list of DMS API actions supported by this service.
func (s *DMSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateReplicationInstance", Method: http.MethodPost, IAMAction: "dms:CreateReplicationInstance"},
		{Name: "DescribeReplicationInstances", Method: http.MethodPost, IAMAction: "dms:DescribeReplicationInstances"},
		{Name: "DeleteReplicationInstance", Method: http.MethodPost, IAMAction: "dms:DeleteReplicationInstance"},
		{Name: "CreateEndpoint", Method: http.MethodPost, IAMAction: "dms:CreateEndpoint"},
		{Name: "DescribeEndpoints", Method: http.MethodPost, IAMAction: "dms:DescribeEndpoints"},
		{Name: "DeleteEndpoint", Method: http.MethodPost, IAMAction: "dms:DeleteEndpoint"},
		{Name: "CreateReplicationTask", Method: http.MethodPost, IAMAction: "dms:CreateReplicationTask"},
		{Name: "DescribeReplicationTasks", Method: http.MethodPost, IAMAction: "dms:DescribeReplicationTasks"},
		{Name: "StartReplicationTask", Method: http.MethodPost, IAMAction: "dms:StartReplicationTask"},
		{Name: "StopReplicationTask", Method: http.MethodPost, IAMAction: "dms:StopReplicationTask"},
		{Name: "DeleteReplicationTask", Method: http.MethodPost, IAMAction: "dms:DeleteReplicationTask"},
		{Name: "CreateEventSubscription", Method: http.MethodPost, IAMAction: "dms:CreateEventSubscription"},
		{Name: "DescribeEventSubscriptions", Method: http.MethodPost, IAMAction: "dms:DescribeEventSubscriptions"},
		{Name: "DeleteEventSubscription", Method: http.MethodPost, IAMAction: "dms:DeleteEventSubscription"},
		{Name: "TestConnection", Method: http.MethodPost, IAMAction: "dms:TestConnection"},
		{Name: "DescribeConnections", Method: http.MethodPost, IAMAction: "dms:DescribeConnections"},
	}
}

// HealthCheck always returns nil.
func (s *DMSService) HealthCheck() error { return nil }

// HandleRequest routes an incoming DMS request to the appropriate handler.
func (s *DMSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateReplicationInstance":
		return handleCreateReplicationInstance(params, s.store)
	case "DescribeReplicationInstances":
		return handleDescribeReplicationInstances(s.store)
	case "DeleteReplicationInstance":
		return handleDeleteReplicationInstance(params, s.store)
	case "CreateEndpoint":
		return handleCreateEndpoint(params, s.store)
	case "DescribeEndpoints":
		return handleDescribeEndpoints(s.store)
	case "DeleteEndpoint":
		return handleDeleteEndpoint(params, s.store)
	case "CreateReplicationTask":
		return handleCreateReplicationTask(params, s.store)
	case "DescribeReplicationTasks":
		return handleDescribeReplicationTasks(s.store)
	case "StartReplicationTask":
		return handleStartReplicationTask(params, s.store)
	case "StopReplicationTask":
		return handleStopReplicationTask(params, s.store)
	case "DeleteReplicationTask":
		return handleDeleteReplicationTask(params, s.store)
	case "CreateEventSubscription":
		return handleCreateEventSubscription(params, s.store)
	case "DescribeEventSubscriptions":
		return handleDescribeEventSubscriptions(s.store)
	case "DeleteEventSubscription":
		return handleDeleteEventSubscription(params, s.store)
	case "TestConnection":
		return handleTestConnection(params, s.store)
	case "DescribeConnections":
		return handleDescribeConnections(s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
