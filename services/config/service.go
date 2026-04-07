package config

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/eventbus"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ConfigService is the cloudmock implementation of the AWS Config API.
type ConfigService struct {
	store     *Store
	accountID string
	region    string
	bus       *eventbus.Bus
}

// New returns a new ConfigService for the given AWS account ID and region.
func New(accountID, region string) *ConfigService {
	return &ConfigService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// NewWithBus returns a new ConfigService wired to an event bus.
func NewWithBus(accountID, region string, bus *eventbus.Bus) *ConfigService {
	return &ConfigService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
		bus:       bus,
	}
}

// Name returns the AWS service name used for routing.
func (s *ConfigService) Name() string { return "config" }

// Actions returns the list of Config API actions supported by this service.
func (s *ConfigService) Actions() []service.Action {
	return []service.Action{
		{Name: "PutConfigRule", Method: http.MethodPost, IAMAction: "config:PutConfigRule"},
		{Name: "DescribeConfigRules", Method: http.MethodPost, IAMAction: "config:DescribeConfigRules"},
		{Name: "DeleteConfigRule", Method: http.MethodPost, IAMAction: "config:DeleteConfigRule"},
		{Name: "PutConfigurationRecorder", Method: http.MethodPost, IAMAction: "config:PutConfigurationRecorder"},
		{Name: "DescribeConfigurationRecorders", Method: http.MethodPost, IAMAction: "config:DescribeConfigurationRecorders"},
		{Name: "DeleteConfigurationRecorder", Method: http.MethodPost, IAMAction: "config:DeleteConfigurationRecorder"},
		{Name: "PutDeliveryChannel", Method: http.MethodPost, IAMAction: "config:PutDeliveryChannel"},
		{Name: "DescribeDeliveryChannels", Method: http.MethodPost, IAMAction: "config:DescribeDeliveryChannels"},
		{Name: "DeleteDeliveryChannel", Method: http.MethodPost, IAMAction: "config:DeleteDeliveryChannel"},
		{Name: "StartConfigurationRecorder", Method: http.MethodPost, IAMAction: "config:StartConfigurationRecorder"},
		{Name: "StopConfigurationRecorder", Method: http.MethodPost, IAMAction: "config:StopConfigurationRecorder"},
		{Name: "GetComplianceDetailsByConfigRule", Method: http.MethodPost, IAMAction: "config:GetComplianceDetailsByConfigRule"},
		{Name: "DescribeComplianceByConfigRule", Method: http.MethodPost, IAMAction: "config:DescribeComplianceByConfigRule"},
		{Name: "PutConformancePack", Method: http.MethodPost, IAMAction: "config:PutConformancePack"},
		{Name: "DescribeConformancePacks", Method: http.MethodPost, IAMAction: "config:DescribeConformancePacks"},
		{Name: "DeleteConformancePack", Method: http.MethodPost, IAMAction: "config:DeleteConformancePack"},
		{Name: "PutEvaluations", Method: http.MethodPost, IAMAction: "config:PutEvaluations"},
		{Name: "GetResourceConfigHistory", Method: http.MethodPost, IAMAction: "config:GetResourceConfigHistory"},
	}
}

// HealthCheck always returns nil.
func (s *ConfigService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Config request to the appropriate handler.
func (s *ConfigService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "PutConfigRule":
		return handlePutConfigRule(ctx, s.store)
	case "DescribeConfigRules":
		return handleDescribeConfigRules(ctx, s.store)
	case "DeleteConfigRule":
		return handleDeleteConfigRule(ctx, s.store)
	case "PutConfigurationRecorder":
		return handlePutConfigurationRecorder(ctx, s.store)
	case "DescribeConfigurationRecorders":
		return handleDescribeConfigurationRecorders(ctx, s.store)
	case "DeleteConfigurationRecorder":
		return handleDeleteConfigurationRecorder(ctx, s.store)
	case "PutDeliveryChannel":
		return handlePutDeliveryChannel(ctx, s.store)
	case "DescribeDeliveryChannels":
		return handleDescribeDeliveryChannels(ctx, s.store)
	case "DeleteDeliveryChannel":
		return handleDeleteDeliveryChannel(ctx, s.store)
	case "StartConfigurationRecorder":
		return s.handleStartRecorderWithBus(ctx)
	case "StopConfigurationRecorder":
		return s.handleStopRecorderWithBus(ctx)
	case "GetComplianceDetailsByConfigRule":
		return handleGetComplianceDetailsByConfigRule(ctx, s.store)
	case "DescribeComplianceByConfigRule":
		return handleDescribeComplianceByConfigRule(ctx, s.store)
	case "PutConformancePack":
		return handlePutConformancePack(ctx, s.store)
	case "DescribeConformancePacks":
		return handleDescribeConformancePacks(ctx, s.store)
	case "DeleteConformancePack":
		return handleDeleteConformancePack(ctx, s.store)
	case "PutEvaluations":
		return handlePutEvaluations(ctx, s.store)
	case "GetResourceConfigHistory":
		return s.handleGetResourceConfigHistoryWithStore(ctx)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}

func (s *ConfigService) handleStartRecorderWithBus(ctx *service.RequestContext) (*service.Response, error) {
	resp, err := handleStartConfigurationRecorder(ctx, s.store)
	if err != nil {
		return resp, err
	}
	// Subscribe to bus
	var params map[string]any
	parseJSON(ctx.Body, &params)
	name, _ := params["ConfigurationRecorderName"].(string)
	s.startBusRecording(name)
	return resp, err
}

func (s *ConfigService) handleStopRecorderWithBus(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	parseJSON(ctx.Body, &params)
	name, _ := params["ConfigurationRecorderName"].(string)
	s.stopBusRecording(name)
	return handleStopConfigurationRecorder(ctx, s.store)
}

func (s *ConfigService) handleGetResourceConfigHistoryWithStore(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	resourceType, _ := params["resourceType"].(string)
	resourceId, _ := params["resourceId"].(string)

	items := s.store.GetConfigHistory(resourceType, resourceId)
	configItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		configItems = append(configItems, map[string]any{
			"resourceType":                item.ResourceType,
			"resourceId":                  item.ResourceId,
			"resourceName":                item.ResourceName,
			"configurationItemCaptureTime": float64(item.ConfigurationItemCaptureTime.Unix()),
			"configurationItemStatus":     item.ConfigurationItemStatus,
			"configuration":               item.Configuration,
			"accountId":                   item.AccountId,
			"awsRegion":                   item.AwsRegion,
		})
	}

	return jsonOK(map[string]any{"configurationItems": configItems})
}
