package iotdata

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// IoTDataService is the cloudmock implementation of the AWS IoT Data Plane API.
type IoTDataService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new IoTDataService for the given AWS account ID and region.
func New(accountID, region string) *IoTDataService {
	return &IoTDataService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *IoTDataService) Name() string { return "iot-data" }

// Actions returns the list of IoT Data API actions supported by this service.
func (s *IoTDataService) Actions() []service.Action {
	return []service.Action{
		{Name: "GetThingShadow", Method: http.MethodGet, IAMAction: "iot:GetThingShadow"},
		{Name: "UpdateThingShadow", Method: http.MethodPost, IAMAction: "iot:UpdateThingShadow"},
		{Name: "DeleteThingShadow", Method: http.MethodDelete, IAMAction: "iot:DeleteThingShadow"},
		{Name: "ListNamedShadowsForThing", Method: http.MethodGet, IAMAction: "iot:ListNamedShadowsForThing"},
		{Name: "Publish", Method: http.MethodPost, IAMAction: "iot:Publish"},
	}
}

// HealthCheck always returns nil.
func (s *IoTDataService) HealthCheck() error { return nil }

// HandleRequest routes an incoming IoT Data request to the appropriate handler.
func (s *IoTDataService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
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
	case "GetThingShadow":
		return handleGetThingShadow(params, s.store)
	case "UpdateThingShadow":
		return handleUpdateThingShadow(params, ctx.Body, s.store)
	case "DeleteThingShadow":
		return handleDeleteThingShadow(params, s.store)
	case "ListNamedShadowsForThing":
		return handleListNamedShadowsForThing(params, s.store)
	case "Publish":
		return handlePublish(params, ctx.Body, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
