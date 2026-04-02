package kafka

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// KafkaService is the cloudmock implementation of the AWS MSK (Managed Streaming for Apache Kafka) API.
type KafkaService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new KafkaService for the given AWS account ID and region.
func New(accountID, region string) *KafkaService {
	return &KafkaService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *KafkaService) Name() string { return "kafka" }

// Actions returns the list of MSK API actions supported by this service.
func (s *KafkaService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateCluster", Method: http.MethodPost, IAMAction: "kafka:CreateCluster"},
		{Name: "DescribeCluster", Method: http.MethodGet, IAMAction: "kafka:DescribeCluster"},
		{Name: "ListClusters", Method: http.MethodGet, IAMAction: "kafka:ListClusters"},
		{Name: "DeleteCluster", Method: http.MethodDelete, IAMAction: "kafka:DeleteCluster"},
		{Name: "UpdateBrokerCount", Method: http.MethodPut, IAMAction: "kafka:UpdateBrokerCount"},
		{Name: "UpdateBrokerStorage", Method: http.MethodPut, IAMAction: "kafka:UpdateBrokerStorage"},
		{Name: "UpdateClusterConfiguration", Method: http.MethodPut, IAMAction: "kafka:UpdateClusterConfiguration"},
		{Name: "RebootBroker", Method: http.MethodPut, IAMAction: "kafka:RebootBroker"},
		{Name: "CreateConfiguration", Method: http.MethodPost, IAMAction: "kafka:CreateConfiguration"},
		{Name: "DescribeConfiguration", Method: http.MethodGet, IAMAction: "kafka:DescribeConfiguration"},
		{Name: "ListConfigurations", Method: http.MethodGet, IAMAction: "kafka:ListConfigurations"},
		{Name: "UpdateConfiguration", Method: http.MethodPut, IAMAction: "kafka:UpdateConfiguration"},
		{Name: "DeleteConfiguration", Method: http.MethodDelete, IAMAction: "kafka:DeleteConfiguration"},
		{Name: "DescribeConfigurationRevision", Method: http.MethodGet, IAMAction: "kafka:DescribeConfigurationRevision"},
		{Name: "ListConfigurationRevisions", Method: http.MethodGet, IAMAction: "kafka:ListConfigurationRevisions"},
		{Name: "ListClusterOperations", Method: http.MethodGet, IAMAction: "kafka:ListClusterOperations"},
		{Name: "DescribeClusterOperation", Method: http.MethodGet, IAMAction: "kafka:DescribeClusterOperation"},
		{Name: "GetBootstrapBrokers", Method: http.MethodGet, IAMAction: "kafka:GetBootstrapBrokers"},
		{Name: "ListNodes", Method: http.MethodGet, IAMAction: "kafka:ListNodes"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "kafka:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "kafka:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "kafka:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *KafkaService) HealthCheck() error { return nil }

// HandleRequest routes an incoming MSK request to the appropriate handler.
// MSK uses rest-json protocol.
func (s *KafkaService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
	case "CreateCluster":
		return handleCreateCluster(params, s.store)
	case "DescribeCluster":
		return handleDescribeCluster(params, s.store)
	case "ListClusters":
		return handleListClusters(s.store)
	case "DeleteCluster":
		return handleDeleteCluster(params, s.store)
	case "UpdateBrokerCount":
		return handleUpdateBrokerCount(params, s.store)
	case "UpdateBrokerStorage":
		return handleUpdateBrokerStorage(params, s.store)
	case "UpdateClusterConfiguration":
		return handleUpdateClusterConfiguration(params, s.store)
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
	case "DeleteConfiguration":
		return handleDeleteConfiguration(params, s.store)
	case "DescribeConfigurationRevision":
		return handleDescribeConfigurationRevision(params, s.store)
	case "ListConfigurationRevisions":
		return handleListConfigurationRevisions(params, s.store)
	case "ListClusterOperations":
		return handleListClusterOperations(params, s.store)
	case "DescribeClusterOperation":
		return handleDescribeClusterOperation(params, s.store)
	case "GetBootstrapBrokers":
		return handleGetBootstrapBrokers(params, s.store)
	case "ListNodes":
		return handleListNodes(params, s.store)
	case "TagResource":
		return handleTagResource(params, s.store)
	case "UntagResource":
		return handleUntagResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
