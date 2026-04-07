package iotwireless

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// IoTWirelessService is the cloudmock implementation of the AWS IoT Wireless API.
type IoTWirelessService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new IoTWirelessService for the given AWS account ID and region.
func New(accountID, region string) *IoTWirelessService {
	return &IoTWirelessService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *IoTWirelessService) Name() string { return "iot-wireless" }

// Actions returns the list of IoT Wireless API actions supported by this service.
func (s *IoTWirelessService) Actions() []service.Action {
	return []service.Action{
		// Wireless devices
		{Name: "CreateWirelessDevice", Method: http.MethodPost, IAMAction: "iotwireless:CreateWirelessDevice"},
		{Name: "GetWirelessDevice", Method: http.MethodGet, IAMAction: "iotwireless:GetWirelessDevice"},
		{Name: "ListWirelessDevices", Method: http.MethodGet, IAMAction: "iotwireless:ListWirelessDevices"},
		{Name: "DeleteWirelessDevice", Method: http.MethodDelete, IAMAction: "iotwireless:DeleteWirelessDevice"},
		{Name: "UpdateWirelessDevice", Method: http.MethodPatch, IAMAction: "iotwireless:UpdateWirelessDevice"},
		// Wireless gateways
		{Name: "CreateWirelessGateway", Method: http.MethodPost, IAMAction: "iotwireless:CreateWirelessGateway"},
		{Name: "GetWirelessGateway", Method: http.MethodGet, IAMAction: "iotwireless:GetWirelessGateway"},
		{Name: "ListWirelessGateways", Method: http.MethodGet, IAMAction: "iotwireless:ListWirelessGateways"},
		{Name: "DeleteWirelessGateway", Method: http.MethodDelete, IAMAction: "iotwireless:DeleteWirelessGateway"},
		{Name: "UpdateWirelessGateway", Method: http.MethodPatch, IAMAction: "iotwireless:UpdateWirelessGateway"},
		// Device profiles
		{Name: "CreateDeviceProfile", Method: http.MethodPost, IAMAction: "iotwireless:CreateDeviceProfile"},
		{Name: "GetDeviceProfile", Method: http.MethodGet, IAMAction: "iotwireless:GetDeviceProfile"},
		{Name: "ListDeviceProfiles", Method: http.MethodGet, IAMAction: "iotwireless:ListDeviceProfiles"},
		{Name: "DeleteDeviceProfile", Method: http.MethodDelete, IAMAction: "iotwireless:DeleteDeviceProfile"},
		// Service profiles
		{Name: "CreateServiceProfile", Method: http.MethodPost, IAMAction: "iotwireless:CreateServiceProfile"},
		{Name: "GetServiceProfile", Method: http.MethodGet, IAMAction: "iotwireless:GetServiceProfile"},
		{Name: "ListServiceProfiles", Method: http.MethodGet, IAMAction: "iotwireless:ListServiceProfiles"},
		{Name: "DeleteServiceProfile", Method: http.MethodDelete, IAMAction: "iotwireless:DeleteServiceProfile"},
		// Destinations
		{Name: "CreateDestination", Method: http.MethodPost, IAMAction: "iotwireless:CreateDestination"},
		{Name: "GetDestination", Method: http.MethodGet, IAMAction: "iotwireless:GetDestination"},
		{Name: "ListDestinations", Method: http.MethodGet, IAMAction: "iotwireless:ListDestinations"},
		{Name: "UpdateDestination", Method: http.MethodPatch, IAMAction: "iotwireless:UpdateDestination"},
		{Name: "DeleteDestination", Method: http.MethodDelete, IAMAction: "iotwireless:DeleteDestination"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "iotwireless:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "iotwireless:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "iotwireless:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *IoTWirelessService) HealthCheck() error { return nil }

// HandleRequest routes an incoming IoT Wireless request to the appropriate handler.
func (s *IoTWirelessService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
	case "CreateWirelessDevice":
		return handleCreateWirelessDevice(params, s.store)
	case "GetWirelessDevice":
		return handleGetWirelessDevice(params, s.store)
	case "ListWirelessDevices":
		return handleListWirelessDevices(s.store)
	case "DeleteWirelessDevice":
		return handleDeleteWirelessDevice(params, s.store)
	case "UpdateWirelessDevice":
		return handleUpdateWirelessDevice(params, s.store)
	case "CreateWirelessGateway":
		return handleCreateWirelessGateway(params, s.store)
	case "GetWirelessGateway":
		return handleGetWirelessGateway(params, s.store)
	case "ListWirelessGateways":
		return handleListWirelessGateways(s.store)
	case "DeleteWirelessGateway":
		return handleDeleteWirelessGateway(params, s.store)
	case "UpdateWirelessGateway":
		return handleUpdateWirelessGateway(params, s.store)
	case "CreateDeviceProfile":
		return handleCreateDeviceProfile(params, s.store)
	case "GetDeviceProfile":
		return handleGetDeviceProfile(params, s.store)
	case "ListDeviceProfiles":
		return handleListDeviceProfiles(s.store)
	case "DeleteDeviceProfile":
		return handleDeleteDeviceProfile(params, s.store)
	case "CreateServiceProfile":
		return handleCreateServiceProfile(params, s.store)
	case "GetServiceProfile":
		return handleGetServiceProfile(params, s.store)
	case "ListServiceProfiles":
		return handleListServiceProfiles(s.store)
	case "DeleteServiceProfile":
		return handleDeleteServiceProfile(params, s.store)
	case "CreateDestination":
		return handleCreateDestination(params, s.store)
	case "GetDestination":
		return handleGetDestination(params, s.store)
	case "ListDestinations":
		return handleListDestinations(s.store)
	case "UpdateDestination":
		return handleUpdateDestination(params, s.store)
	case "DeleteDestination":
		return handleDeleteDestination(params, s.store)
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
