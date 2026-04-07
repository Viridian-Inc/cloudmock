package fis

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// FISService is the cloudmock implementation of the AWS Fault Injection Simulator API.
type FISService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new FISService for the given AWS account ID and region.
func New(accountID, region string) *FISService {
	return &FISService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *FISService) Name() string { return "fis" }

// Actions returns the list of FIS API actions supported by this service.
func (s *FISService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateExperimentTemplate", Method: http.MethodPost, IAMAction: "fis:CreateExperimentTemplate"},
		{Name: "GetExperimentTemplate", Method: http.MethodGet, IAMAction: "fis:GetExperimentTemplate"},
		{Name: "ListExperimentTemplates", Method: http.MethodGet, IAMAction: "fis:ListExperimentTemplates"},
		{Name: "UpdateExperimentTemplate", Method: http.MethodPatch, IAMAction: "fis:UpdateExperimentTemplate"},
		{Name: "DeleteExperimentTemplate", Method: http.MethodDelete, IAMAction: "fis:DeleteExperimentTemplate"},
		{Name: "StartExperiment", Method: http.MethodPost, IAMAction: "fis:StartExperiment"},
		{Name: "GetExperiment", Method: http.MethodGet, IAMAction: "fis:GetExperiment"},
		{Name: "ListExperiments", Method: http.MethodGet, IAMAction: "fis:ListExperiments"},
		{Name: "StopExperiment", Method: http.MethodDelete, IAMAction: "fis:StopExperiment"},
		{Name: "ListTargetResourceTypes", Method: http.MethodGet, IAMAction: "fis:ListTargetResourceTypes"},
		{Name: "ListActions", Method: http.MethodGet, IAMAction: "fis:ListActions"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "fis:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "fis:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "fis:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *FISService) HealthCheck() error { return nil }

// HandleRequest routes an incoming FIS request to the appropriate handler.
func (s *FISService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// /experimentTemplates
	if path == "/experimentTemplates" {
		switch method {
		case http.MethodPost:
			return handleCreateExperimentTemplate(params, s.store)
		case http.MethodGet:
			return handleListExperimentTemplates(s.store)
		}
	}
	if strings.HasPrefix(path, "/experimentTemplates/") {
		id := strings.TrimPrefix(path, "/experimentTemplates/")
		switch method {
		case http.MethodGet:
			return handleGetExperimentTemplate(id, s.store)
		case http.MethodPatch:
			return handleUpdateExperimentTemplate(id, params, s.store)
		case http.MethodDelete:
			return handleDeleteExperimentTemplate(id, s.store)
		}
	}

	// /experiments
	if path == "/experiments" {
		switch method {
		case http.MethodPost:
			return handleStartExperiment(params, s.store)
		case http.MethodGet:
			return handleListExperiments(s.store)
		}
	}
	if strings.HasPrefix(path, "/experiments/") {
		rest := strings.TrimPrefix(path, "/experiments/")
		if strings.HasSuffix(rest, "/stop") {
			id := strings.TrimSuffix(rest, "/stop")
			return handleStopExperiment(id, s.store)
		}
		if method == http.MethodGet {
			return handleGetExperiment(rest, s.store)
		}
	}

	// /targetResourceTypes
	if path == "/targetResourceTypes" && method == http.MethodGet {
		return handleListTargetResourceTypes(s.store)
	}

	// /actions
	if path == "/actions" && method == http.MethodGet {
		return handleListActions(s.store)
	}

	// /tags/{arn}
	if strings.HasPrefix(path, "/tags/") {
		arn := strings.TrimPrefix(path, "/tags/")
		switch method {
		case http.MethodGet:
			return handleListTagsForResource(arn, s.store)
		case http.MethodPost:
			return handleTagResource(arn, params, s.store)
		case http.MethodDelete:
			var keys []string
			if ks, ok := params["tagKeys"].([]any); ok {
				for _, k := range ks {
					if s, ok := k.(string); ok {
						keys = append(keys, s)
					}
				}
			}
			return handleUntagResource(arn, keys, s.store)
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
