package mediaconvert

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// MediaConvertService is the cloudmock implementation of the AWS Elemental MediaConvert API.
type MediaConvertService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new MediaConvertService for the given AWS account ID and region.
func New(accountID, region string) *MediaConvertService {
	return &MediaConvertService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *MediaConvertService) Name() string { return "mediaconvert" }

// Actions returns the list of MediaConvert API actions supported by this service.
func (s *MediaConvertService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateJob", Method: http.MethodPost, IAMAction: "mediaconvert:CreateJob"},
		{Name: "GetJob", Method: http.MethodGet, IAMAction: "mediaconvert:GetJob"},
		{Name: "ListJobs", Method: http.MethodGet, IAMAction: "mediaconvert:ListJobs"},
		{Name: "CancelJob", Method: http.MethodDelete, IAMAction: "mediaconvert:CancelJob"},
		{Name: "CreateJobTemplate", Method: http.MethodPost, IAMAction: "mediaconvert:CreateJobTemplate"},
		{Name: "GetJobTemplate", Method: http.MethodGet, IAMAction: "mediaconvert:GetJobTemplate"},
		{Name: "ListJobTemplates", Method: http.MethodGet, IAMAction: "mediaconvert:ListJobTemplates"},
		{Name: "DeleteJobTemplate", Method: http.MethodDelete, IAMAction: "mediaconvert:DeleteJobTemplate"},
		{Name: "CreatePreset", Method: http.MethodPost, IAMAction: "mediaconvert:CreatePreset"},
		{Name: "GetPreset", Method: http.MethodGet, IAMAction: "mediaconvert:GetPreset"},
		{Name: "ListPresets", Method: http.MethodGet, IAMAction: "mediaconvert:ListPresets"},
		{Name: "DeletePreset", Method: http.MethodDelete, IAMAction: "mediaconvert:DeletePreset"},
		{Name: "CreateQueue", Method: http.MethodPost, IAMAction: "mediaconvert:CreateQueue"},
		{Name: "GetQueue", Method: http.MethodGet, IAMAction: "mediaconvert:GetQueue"},
		{Name: "ListQueues", Method: http.MethodGet, IAMAction: "mediaconvert:ListQueues"},
		{Name: "DeleteQueue", Method: http.MethodDelete, IAMAction: "mediaconvert:DeleteQueue"},
	}
}

// HealthCheck always returns nil.
func (s *MediaConvertService) HealthCheck() error { return nil }

// HandleRequest routes an incoming MediaConvert request to the appropriate handler.
func (s *MediaConvertService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// Strip API version prefix
	for _, prefix := range []string{"/2017-08-29", ""} {
		p := strings.TrimPrefix(path, prefix)

		if p == "/jobs" {
			switch method {
			case http.MethodPost:
				return handleCreateJob(params, s.store)
			case http.MethodGet:
				return handleListJobs(s.store)
			}
		}
		if strings.HasPrefix(p, "/jobs/") {
			id := strings.TrimPrefix(p, "/jobs/")
			switch method {
			case http.MethodGet:
				return handleGetJob(id, s.store)
			case http.MethodDelete:
				return handleCancelJob(id, s.store)
			}
		}

		if p == "/jobTemplates" {
			switch method {
			case http.MethodPost:
				return handleCreateJobTemplate(params, s.store)
			case http.MethodGet:
				return handleListJobTemplates(s.store)
			}
		}
		if strings.HasPrefix(p, "/jobTemplates/") {
			name := strings.TrimPrefix(p, "/jobTemplates/")
			switch method {
			case http.MethodGet:
				return handleGetJobTemplate(name, s.store)
			case http.MethodDelete:
				return handleDeleteJobTemplate(name, s.store)
			}
		}

		if p == "/presets" {
			switch method {
			case http.MethodPost:
				return handleCreatePreset(params, s.store)
			case http.MethodGet:
				return handleListPresets(s.store)
			}
		}
		if strings.HasPrefix(p, "/presets/") {
			name := strings.TrimPrefix(p, "/presets/")
			switch method {
			case http.MethodGet:
				return handleGetPreset(name, s.store)
			case http.MethodDelete:
				return handleDeletePreset(name, s.store)
			}
		}

		if p == "/queues" {
			switch method {
			case http.MethodPost:
				return handleCreateQueue(params, s.store)
			case http.MethodGet:
				return handleListQueues(s.store)
			}
		}
		if strings.HasPrefix(p, "/queues/") {
			name := strings.TrimPrefix(p, "/queues/")
			switch method {
			case http.MethodGet:
				return handleGetQueue(name, s.store)
			case http.MethodPut:
				return handleUpdateQueue(name, params, s.store)
			case http.MethodDelete:
				return handleDeleteQueue(name, s.store)
			}
		}

		if prefix != "" {
			break // Only try with prefix once
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
