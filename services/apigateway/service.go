package apigateway

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// APIGatewayService is the cloudmock implementation of the AWS API Gateway REST API.
type APIGatewayService struct {
	store *Store
}

// New returns a new APIGatewayService for the given AWS account ID and region.
func New(accountID, region string) *APIGatewayService {
	return &APIGatewayService{
		store: NewStore(accountID, region),
	}
}

// Name returns the AWS service name used for routing.
func (s *APIGatewayService) Name() string { return "apigateway" }

// BrowserInspect returns API Gateway REST APIs in the shape the devtools
// browser view expects: list of {id, name, routes[]}.
func (s *APIGatewayService) BrowserInspect() []map[string]any {
	apis := s.store.ListRestApis()
	out := make([]map[string]any, 0, len(apis))
	for _, api := range apis {
		routes := make([]map[string]any, 0)
		for _, res := range api.Resources {
			for method, m := range res.Methods {
				integration := ""
				if m.Integration != nil {
					integration = m.Integration.Type
					if m.Integration.Uri != "" {
						integration = integration + " " + m.Integration.Uri
					}
				}
				routes = append(routes, map[string]any{
					"method":      method,
					"path":        res.Path,
					"integration": integration,
				})
			}
		}
		out = append(out, map[string]any{
			"id":    api.Id,
			"name":  api.Name,
			"routes": routes,
		})
	}
	return out
}

// Actions returns the list of API Gateway actions supported by this service.
// API Gateway uses path-based REST routing, so these are descriptive only.
func (s *APIGatewayService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateRestApi", Method: http.MethodPost, IAMAction: "apigateway:POST"},
		{Name: "GetRestApis", Method: http.MethodGet, IAMAction: "apigateway:GET"},
		{Name: "GetRestApi", Method: http.MethodGet, IAMAction: "apigateway:GET"},
		{Name: "DeleteRestApi", Method: http.MethodDelete, IAMAction: "apigateway:DELETE"},
		{Name: "CreateResource", Method: http.MethodPost, IAMAction: "apigateway:POST"},
		{Name: "GetResources", Method: http.MethodGet, IAMAction: "apigateway:GET"},
		{Name: "DeleteResource", Method: http.MethodDelete, IAMAction: "apigateway:DELETE"},
		{Name: "PutMethod", Method: http.MethodPut, IAMAction: "apigateway:PUT"},
		{Name: "GetMethod", Method: http.MethodGet, IAMAction: "apigateway:GET"},
		{Name: "PutIntegration", Method: http.MethodPut, IAMAction: "apigateway:PUT"},
		{Name: "CreateDeployment", Method: http.MethodPost, IAMAction: "apigateway:POST"},
		{Name: "GetDeployments", Method: http.MethodGet, IAMAction: "apigateway:GET"},
		{Name: "CreateStage", Method: http.MethodPost, IAMAction: "apigateway:POST"},
		{Name: "GetStages", Method: http.MethodGet, IAMAction: "apigateway:GET"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *APIGatewayService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for API Gateway resource types.
func (s *APIGatewayService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "apigateway",
			ResourceType:  "aws_api_gateway_rest_api",
			TerraformType: "cloudmock_api_gateway_rest_api",
			AWSType:       "AWS::ApiGateway::RestApi",
			CreateAction:  "CreateRestApi",
			ReadAction:    "GetRestApi",
			DeleteAction:  "DeleteRestApi",
			ListAction:    "GetRestApis",
			ImportID:      "id",
			Attributes: []schema.AttributeSchema{
				{Name: "name", Type: "string", Required: true},
				{Name: "description", Type: "string"},
				{Name: "endpoint_configuration", Type: "map"},
				{Name: "id", Type: "string", Computed: true},
				{Name: "root_resource_id", Type: "string", Computed: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "execution_arn", Type: "string", Computed: true},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "apigateway",
			ResourceType:  "aws_api_gateway_deployment",
			TerraformType: "cloudmock_api_gateway_deployment",
			AWSType:       "AWS::ApiGateway::Deployment",
			CreateAction:  "CreateDeployment",
			ReadAction:    "GetDeployments",
			ImportID:      "rest_api_id/id",
			Attributes: []schema.AttributeSchema{
				{Name: "rest_api_id", Type: "string", Required: true, ForceNew: true},
				{Name: "stage_name", Type: "string"},
				{Name: "description", Type: "string"},
				{Name: "id", Type: "string", Computed: true},
				{Name: "invoke_url", Type: "string", Computed: true},
			},
		},
	}
}

// HandleRequest routes an incoming API Gateway request to the appropriate handler.
// API Gateway uses REST path-based routing.
//
// Routes:
//
//	POST   /restapis                                                          → CreateRestApi
//	GET    /restapis                                                          → GetRestApis
//	GET    /restapis/{id}                                                     → GetRestApi
//	DELETE /restapis/{id}                                                     → DeleteRestApi
//	GET    /restapis/{id}/resources                                           → GetResources
//	POST   /restapis/{id}/resources/{parentId}                               → CreateResource
//	DELETE /restapis/{id}/resources/{resourceId}                             → DeleteResource
//	PUT    /restapis/{id}/resources/{resourceId}/methods/{httpMethod}        → PutMethod
//	GET    /restapis/{id}/resources/{resourceId}/methods/{httpMethod}        → GetMethod
//	PUT    /restapis/{id}/resources/{resourceId}/methods/{httpMethod}/integration → PutIntegration
//	POST   /restapis/{id}/deployments                                        → CreateDeployment
//	GET    /restapis/{id}/deployments                                        → GetDeployments
//	GET    /restapis/{id}/stages                                             → GetStages
//	POST   /restapis/{id}/stages                                             → CreateStage
func (s *APIGatewayService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	const basePrefix = "/restapis"

	if !strings.HasPrefix(path, basePrefix) {
		return jsonErr(service.NewAWSError("NotFoundException",
			"Route not found.", http.StatusNotFound))
	}

	// Strip /restapis prefix.
	rest := path[len(basePrefix):]

	// /restapis
	if rest == "" {
		switch method {
		case http.MethodPost:
			return handleCreateRestApi(ctx, s.store)
		case http.MethodGet:
			return handleGetRestApis(ctx, s.store)
		}
		return notImplemented()
	}

	// /restapis/{id}/...
	// rest starts with "/"
	parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", -1)
	if len(parts) == 0 {
		return notImplemented()
	}

	apiID := parts[0]

	// /restapis/{id}
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetRestApi(ctx, s.store, apiID)
		case http.MethodDelete:
			return handleDeleteRestApi(ctx, s.store, apiID)
		}
		return notImplemented()
	}

	section := parts[1]

	switch section {
	case "resources":
		return s.routeResources(ctx, method, parts, apiID)
	case "deployments":
		return s.routeDeployments(ctx, method, parts, apiID)
	case "stages":
		return s.routeStages(ctx, method, parts, apiID)
	}

	return notImplemented()
}

// routeResources handles all /restapis/{id}/resources/... routes.
func (s *APIGatewayService) routeResources(ctx *service.RequestContext, method string, parts []string, apiID string) (*service.Response, error) {
	// /restapis/{id}/resources
	if len(parts) == 2 {
		if method == http.MethodGet {
			return handleGetResources(ctx, s.store, apiID)
		}
		return notImplemented()
	}

	resourceID := parts[2]

	// /restapis/{id}/resources/{resourceId}
	if len(parts) == 3 {
		switch method {
		case http.MethodPost:
			// CreateResource: POST /restapis/{id}/resources/{parentId}
			return handleCreateResource(ctx, s.store, apiID, resourceID)
		case http.MethodDelete:
			return handleDeleteResource(ctx, s.store, apiID, resourceID)
		}
		return notImplemented()
	}

	// /restapis/{id}/resources/{resourceId}/methods/...
	if len(parts) >= 4 && parts[3] == "methods" {
		return s.routeMethods(ctx, method, parts, apiID, resourceID)
	}

	return notImplemented()
}

// routeMethods handles all /restapis/{id}/resources/{resourceId}/methods/... routes.
func (s *APIGatewayService) routeMethods(ctx *service.RequestContext, method string, parts []string, apiID, resourceID string) (*service.Response, error) {
	// /restapis/{id}/resources/{resourceId}/methods
	if len(parts) == 4 {
		return notImplemented()
	}

	httpMethod := parts[4]

	// /restapis/{id}/resources/{resourceId}/methods/{httpMethod}
	if len(parts) == 5 {
		switch method {
		case http.MethodPut:
			return handlePutMethod(ctx, s.store, apiID, resourceID, httpMethod)
		case http.MethodGet:
			return handleGetMethod(ctx, s.store, apiID, resourceID, httpMethod)
		}
		return notImplemented()
	}

	// /restapis/{id}/resources/{resourceId}/methods/{httpMethod}/integration
	if len(parts) >= 6 && parts[5] == "integration" {
		if method == http.MethodPut {
			return handlePutIntegration(ctx, s.store, apiID, resourceID, httpMethod)
		}
		return notImplemented()
	}

	return notImplemented()
}

// routeDeployments handles all /restapis/{id}/deployments routes.
func (s *APIGatewayService) routeDeployments(ctx *service.RequestContext, method string, parts []string, apiID string) (*service.Response, error) {
	// /restapis/{id}/deployments
	if len(parts) == 2 {
		switch method {
		case http.MethodPost:
			return handleCreateDeployment(ctx, s.store, apiID)
		case http.MethodGet:
			return handleGetDeployments(ctx, s.store, apiID)
		}
	}
	return notImplemented()
}

// routeStages handles all /restapis/{id}/stages routes.
func (s *APIGatewayService) routeStages(ctx *service.RequestContext, method string, parts []string, apiID string) (*service.Response, error) {
	// /restapis/{id}/stages
	if len(parts) == 2 {
		switch method {
		case http.MethodPost:
			return handleCreateStage(ctx, s.store, apiID)
		case http.MethodGet:
			return handleGetStages(ctx, s.store, apiID)
		}
	}
	return notImplemented()
}

func notImplemented() (*service.Response, error) {
	return jsonErr(service.NewAWSError("NotFoundException",
		"This route is not implemented by cloudmock.", http.StatusNotFound))
}
