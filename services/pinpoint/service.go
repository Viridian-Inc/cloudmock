package pinpoint

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// PinpointService is the cloudmock implementation of the Amazon Pinpoint API.
type PinpointService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new PinpointService for the given AWS account ID and region.
func New(accountID, region string) *PinpointService {
	return &PinpointService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *PinpointService) Name() string { return "pinpoint" }

// Actions returns the list of Pinpoint API actions supported by this service.
func (s *PinpointService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApp", Method: http.MethodPost, IAMAction: "mobiletargeting:CreateApp"},
		{Name: "GetApp", Method: http.MethodGet, IAMAction: "mobiletargeting:GetApp"},
		{Name: "GetApps", Method: http.MethodGet, IAMAction: "mobiletargeting:GetApps"},
		{Name: "DeleteApp", Method: http.MethodDelete, IAMAction: "mobiletargeting:DeleteApp"},
		{Name: "CreateSegment", Method: http.MethodPost, IAMAction: "mobiletargeting:CreateSegment"},
		{Name: "GetSegment", Method: http.MethodGet, IAMAction: "mobiletargeting:GetSegment"},
		{Name: "GetSegments", Method: http.MethodGet, IAMAction: "mobiletargeting:GetSegments"},
		{Name: "DeleteSegment", Method: http.MethodDelete, IAMAction: "mobiletargeting:DeleteSegment"},
		{Name: "CreateCampaign", Method: http.MethodPost, IAMAction: "mobiletargeting:CreateCampaign"},
		{Name: "GetCampaign", Method: http.MethodGet, IAMAction: "mobiletargeting:GetCampaign"},
		{Name: "GetCampaigns", Method: http.MethodGet, IAMAction: "mobiletargeting:GetCampaigns"},
		{Name: "DeleteCampaign", Method: http.MethodDelete, IAMAction: "mobiletargeting:DeleteCampaign"},
		{Name: "CreateJourney", Method: http.MethodPost, IAMAction: "mobiletargeting:CreateJourney"},
		{Name: "GetJourney", Method: http.MethodGet, IAMAction: "mobiletargeting:GetJourney"},
		{Name: "ListJourneys", Method: http.MethodGet, IAMAction: "mobiletargeting:ListJourneys"},
		{Name: "UpdateEndpoint", Method: http.MethodPut, IAMAction: "mobiletargeting:UpdateEndpoint"},
		{Name: "GetEndpoint", Method: http.MethodGet, IAMAction: "mobiletargeting:GetEndpoint"},
	}
}

// HealthCheck always returns nil.
func (s *PinpointService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Pinpoint request to the appropriate handler.
func (s *PinpointService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// /v1/apps
	if path == "/v1/apps" {
		switch method {
		case http.MethodPost:
			return handleCreateApp(params, s.store)
		case http.MethodGet:
			return handleGetApps(s.store)
		}
	}

	parts := strings.Split(strings.TrimPrefix(path, "/v1/apps/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
	}

	appID := parts[0]

	// /v1/apps/{appID}
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetApp(appID, s.store)
		case http.MethodDelete:
			return handleDeleteApp(appID, s.store)
		}
	}

	if len(parts) >= 2 {
		subResource := parts[1]
		switch subResource {
		case "segments":
			return s.handleSegmentRoutes(appID, parts[2:], method, params)
		case "campaigns":
			return s.handleCampaignRoutes(appID, parts[2:], method, params)
		case "journeys":
			return s.handleJourneyRoutes(appID, parts[2:], method, params)
		case "endpoints":
			if len(parts) >= 3 {
				endpointID := parts[2]
				switch method {
				case http.MethodPut:
					return handleUpdateEndpoint(appID, endpointID, params, s.store)
				case http.MethodGet:
					return handleGetEndpoint(appID, endpointID, s.store)
				}
			}
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}

func (s *PinpointService) handleSegmentRoutes(appID string, parts []string, method string, params map[string]any) (*service.Response, error) {
	if len(parts) == 0 {
		switch method {
		case http.MethodPost:
			return handleCreateSegment(appID, params, s.store)
		case http.MethodGet:
			return handleGetSegments(appID, s.store)
		}
	}
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetSegment(appID, parts[0], s.store)
		case http.MethodDelete:
			return handleDeleteSegment(appID, parts[0], s.store)
		}
	}
	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}

func (s *PinpointService) handleCampaignRoutes(appID string, parts []string, method string, params map[string]any) (*service.Response, error) {
	if len(parts) == 0 {
		switch method {
		case http.MethodPost:
			return handleCreateCampaign(appID, params, s.store)
		case http.MethodGet:
			return handleGetCampaigns(appID, s.store)
		}
	}
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetCampaign(appID, parts[0], s.store)
		case http.MethodDelete:
			return handleDeleteCampaign(appID, parts[0], s.store)
		}
	}
	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}

func (s *PinpointService) handleJourneyRoutes(appID string, parts []string, method string, params map[string]any) (*service.Response, error) {
	if len(parts) == 0 {
		switch method {
		case http.MethodPost:
			return handleCreateJourney(appID, params, s.store)
		case http.MethodGet:
			return handleListJourneys(appID, s.store)
		}
	}
	if len(parts) == 1 && method == http.MethodGet {
		return handleGetJourney(appID, parts[0], s.store)
	}
	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
