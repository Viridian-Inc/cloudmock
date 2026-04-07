package serverlessrepo

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServerlessRepoService is the cloudmock implementation of the AWS Serverless Application Repository API.
type ServerlessRepoService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ServerlessRepoService for the given AWS account ID and region.
func New(accountID, region string) *ServerlessRepoService {
	return &ServerlessRepoService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *ServerlessRepoService) Name() string { return "serverlessrepo" }

// Actions returns the list of Serverless Application Repository API actions supported by this service.
func (s *ServerlessRepoService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateApplication", Method: http.MethodPost, IAMAction: "serverlessrepo:CreateApplication"},
		{Name: "GetApplication", Method: http.MethodGet, IAMAction: "serverlessrepo:GetApplication"},
		{Name: "ListApplications", Method: http.MethodGet, IAMAction: "serverlessrepo:ListApplications"},
		{Name: "UpdateApplication", Method: http.MethodPatch, IAMAction: "serverlessrepo:UpdateApplication"},
		{Name: "DeleteApplication", Method: http.MethodDelete, IAMAction: "serverlessrepo:DeleteApplication"},
		{Name: "CreateApplicationVersion", Method: http.MethodPut, IAMAction: "serverlessrepo:CreateApplicationVersion"},
		{Name: "ListApplicationVersions", Method: http.MethodGet, IAMAction: "serverlessrepo:ListApplicationVersions"},
		{Name: "CreateCloudFormationChangeSet", Method: http.MethodPost, IAMAction: "serverlessrepo:CreateCloudFormationChangeSet"},
	}
}

// HealthCheck always returns nil.
func (s *ServerlessRepoService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Serverless Repo request to the appropriate handler.
func (s *ServerlessRepoService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	// /applications
	if path == "/applications" {
		switch method {
		case http.MethodPost:
			return handleCreateApplication(params, s.store)
		case http.MethodGet:
			return handleListApplications(s.store)
		}
	}

	if strings.HasPrefix(path, "/applications/") {
		rest := strings.TrimPrefix(path, "/applications/")
		parts := strings.SplitN(rest, "/", 2)
		appID := parts[0]

		if len(parts) == 1 {
			switch method {
			case http.MethodGet:
				return handleGetApplication(appID, s.store)
			case http.MethodPatch:
				return handleUpdateApplication(appID, params, s.store)
			case http.MethodDelete:
				return handleDeleteApplication(appID, s.store)
			}
		}

		if len(parts) == 2 {
			sub := parts[1]
			if sub == "versions" && method == http.MethodGet {
				return handleListApplicationVersions(appID, s.store)
			}
			if strings.HasPrefix(sub, "versions/") {
				version := strings.TrimPrefix(sub, "versions/")
				if method == http.MethodPut {
					return handleCreateApplicationVersion(appID, version, params, s.store)
				}
			}
			if sub == "changesets" && method == http.MethodPost {
				return handleCreateChangeSet(appID, params, s.store)
			}
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
