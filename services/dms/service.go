package dms

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
		// Replication instances
		{Name: "CreateReplicationInstance", Method: http.MethodPost, IAMAction: "dms:CreateReplicationInstance"},
		{Name: "DescribeReplicationInstances", Method: http.MethodPost, IAMAction: "dms:DescribeReplicationInstances"},
		{Name: "ModifyReplicationInstance", Method: http.MethodPost, IAMAction: "dms:ModifyReplicationInstance"},
		{Name: "DeleteReplicationInstance", Method: http.MethodPost, IAMAction: "dms:DeleteReplicationInstance"},
		// Endpoints
		{Name: "CreateEndpoint", Method: http.MethodPost, IAMAction: "dms:CreateEndpoint"},
		{Name: "DescribeEndpoints", Method: http.MethodPost, IAMAction: "dms:DescribeEndpoints"},
		{Name: "ModifyEndpoint", Method: http.MethodPost, IAMAction: "dms:ModifyEndpoint"},
		{Name: "DeleteEndpoint", Method: http.MethodPost, IAMAction: "dms:DeleteEndpoint"},
		{Name: "TestConnection", Method: http.MethodPost, IAMAction: "dms:TestConnection"},
		// Replication tasks
		{Name: "CreateReplicationTask", Method: http.MethodPost, IAMAction: "dms:CreateReplicationTask"},
		{Name: "DescribeReplicationTasks", Method: http.MethodPost, IAMAction: "dms:DescribeReplicationTasks"},
		{Name: "StartReplicationTask", Method: http.MethodPost, IAMAction: "dms:StartReplicationTask"},
		{Name: "StopReplicationTask", Method: http.MethodPost, IAMAction: "dms:StopReplicationTask"},
		{Name: "DeleteReplicationTask", Method: http.MethodPost, IAMAction: "dms:DeleteReplicationTask"},
		// Subnet groups
		{Name: "CreateReplicationSubnetGroup", Method: http.MethodPost, IAMAction: "dms:CreateReplicationSubnetGroup"},
		{Name: "DescribeReplicationSubnetGroups", Method: http.MethodPost, IAMAction: "dms:DescribeReplicationSubnetGroups"},
		{Name: "ModifyReplicationSubnetGroup", Method: http.MethodPost, IAMAction: "dms:ModifyReplicationSubnetGroup"},
		{Name: "DeleteReplicationSubnetGroup", Method: http.MethodPost, IAMAction: "dms:DeleteReplicationSubnetGroup"},
		// Certificates
		{Name: "CreateCertificate", Method: http.MethodPost, IAMAction: "dms:CreateCertificate"},
		{Name: "DescribeCertificates", Method: http.MethodPost, IAMAction: "dms:DescribeCertificates"},
		{Name: "DeleteCertificate", Method: http.MethodPost, IAMAction: "dms:DeleteCertificate"},
		// Tags
		{Name: "AddTagsToResource", Method: http.MethodPost, IAMAction: "dms:AddTagsToResource"},
		{Name: "RemoveTagsFromResource", Method: http.MethodPost, IAMAction: "dms:RemoveTagsFromResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "dms:ListTagsForResource"},
		// Events
		{Name: "CreateEventSubscription", Method: http.MethodPost, IAMAction: "dms:CreateEventSubscription"},
		{Name: "DescribeEventSubscriptions", Method: http.MethodPost, IAMAction: "dms:DescribeEventSubscriptions"},
		{Name: "DeleteEventSubscription", Method: http.MethodPost, IAMAction: "dms:DeleteEventSubscription"},
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
	if params == nil {
		params = make(map[string]any)
	}

	switch ctx.Action {
	case "CreateReplicationInstance":
		return handleCreateReplicationInstance(params, s.store)
	case "DescribeReplicationInstances":
		return handleDescribeReplicationInstances(s.store)
	case "ModifyReplicationInstance":
		return handleModifyReplicationInstance(params, s.store)
	case "DeleteReplicationInstance":
		return handleDeleteReplicationInstance(params, s.store)
	case "CreateEndpoint":
		return handleCreateEndpoint(params, s.store)
	case "DescribeEndpoints":
		return handleDescribeEndpoints(s.store)
	case "ModifyEndpoint":
		return handleModifyEndpoint(params, s.store)
	case "DeleteEndpoint":
		return handleDeleteEndpoint(params, s.store)
	case "TestConnection":
		return handleTestConnection(params, s.store)
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
	case "CreateReplicationSubnetGroup":
		return handleCreateReplicationSubnetGroup(params, s.store)
	case "DescribeReplicationSubnetGroups":
		return handleDescribeReplicationSubnetGroups(s.store)
	case "ModifyReplicationSubnetGroup":
		return handleModifyReplicationSubnetGroup(params, s.store)
	case "DeleteReplicationSubnetGroup":
		return handleDeleteReplicationSubnetGroup(params, s.store)
	case "CreateCertificate":
		return handleCreateCertificate(params, s.store)
	case "DescribeCertificates":
		return handleDescribeCertificates(s.store)
	case "DeleteCertificate":
		return handleDeleteCertificate(params, s.store)
	case "AddTagsToResource":
		return handleAddTagsToResource(params, s.store)
	case "RemoveTagsFromResource":
		return handleRemoveTagsFromResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	case "CreateEventSubscription":
		return handleCreateEventSubscription(params, s.store)
	case "DescribeEventSubscriptions":
		return handleDescribeEventSubscriptions(s.store)
	case "DeleteEventSubscription":
		return handleDeleteEventSubscription(params, s.store)
	case "DescribeConnections":
		return handleDescribeConnections(s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
