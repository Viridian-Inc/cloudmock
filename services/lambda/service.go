package lambda

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/schema"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// LambdaService is the cloudmock implementation of the AWS Lambda API.
type LambdaService struct {
	store    *FunctionStore
	executor *Executor
	locator  ServiceLocator
	esmStore *EventSourceMappingStore
	layers   *LayerStore
	versions *VersionStore
}

// New returns a new LambdaService for the given AWS account ID and region.
func New(accountID, region string) *LambdaService {
	return &LambdaService{
		store:    NewStore(accountID, region),
		executor: NewExecutor(),
		esmStore: NewEventSourceMappingStore(),
		layers:   NewLayerStore(accountID, region),
		versions: NewVersionStore(accountID, region),
	}
}

// SetLocator sets the service locator for cross-service communication (S3 code source).
func (s *LambdaService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *LambdaService) Name() string { return "lambda" }

// Actions returns the list of Lambda API actions supported by this service.
// Lambda uses REST path-based routing, so these are descriptive only.
func (s *LambdaService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateFunction", Method: http.MethodPost, IAMAction: "lambda:CreateFunction"},
		{Name: "ListFunctions", Method: http.MethodGet, IAMAction: "lambda:ListFunctions"},
		{Name: "GetFunction", Method: http.MethodGet, IAMAction: "lambda:GetFunction"},
		{Name: "DeleteFunction", Method: http.MethodDelete, IAMAction: "lambda:DeleteFunction"},
		{Name: "UpdateFunctionCode", Method: http.MethodPut, IAMAction: "lambda:UpdateFunctionCode"},
		{Name: "Invoke", Method: http.MethodPost, IAMAction: "lambda:InvokeFunction"},
		{Name: "GetFunctionConfiguration", Method: http.MethodGet, IAMAction: "lambda:GetFunctionConfiguration"},
		{Name: "UpdateFunctionConfiguration", Method: http.MethodPut, IAMAction: "lambda:UpdateFunctionConfiguration"},
		{Name: "CreateEventSourceMapping", Method: http.MethodPost, IAMAction: "lambda:CreateEventSourceMapping"},
		{Name: "ListEventSourceMappings", Method: http.MethodGet, IAMAction: "lambda:ListEventSourceMappings"},
		{Name: "GetEventSourceMapping", Method: http.MethodGet, IAMAction: "lambda:GetEventSourceMapping"},
		{Name: "DeleteEventSourceMapping", Method: http.MethodDelete, IAMAction: "lambda:DeleteEventSourceMapping"},
		// Layers
		{Name: "PublishLayerVersion", Method: http.MethodPost, IAMAction: "lambda:PublishLayerVersion"},
		{Name: "GetLayerVersion", Method: http.MethodGet, IAMAction: "lambda:GetLayerVersion"},
		{Name: "ListLayers", Method: http.MethodGet, IAMAction: "lambda:ListLayers"},
		{Name: "ListLayerVersions", Method: http.MethodGet, IAMAction: "lambda:ListLayerVersions"},
		{Name: "DeleteLayerVersion", Method: http.MethodDelete, IAMAction: "lambda:DeleteLayerVersion"},
		// Versions and Aliases
		{Name: "PublishVersion", Method: http.MethodPost, IAMAction: "lambda:PublishVersion"},
		{Name: "ListVersionsByFunction", Method: http.MethodGet, IAMAction: "lambda:ListVersionsByFunction"},
		{Name: "CreateAlias", Method: http.MethodPost, IAMAction: "lambda:CreateAlias"},
		{Name: "GetAlias", Method: http.MethodGet, IAMAction: "lambda:GetAlias"},
		{Name: "UpdateAlias", Method: http.MethodPut, IAMAction: "lambda:UpdateAlias"},
		{Name: "DeleteAlias", Method: http.MethodDelete, IAMAction: "lambda:DeleteAlias"},
		{Name: "ListAliases", Method: http.MethodGet, IAMAction: "lambda:ListAliases"},
		// Function URLs
		{Name: "CreateFunctionUrlConfig", Method: http.MethodPost, IAMAction: "lambda:CreateFunctionUrlConfig"},
		{Name: "GetFunctionUrlConfig", Method: http.MethodGet, IAMAction: "lambda:GetFunctionUrlConfig"},
		{Name: "DeleteFunctionUrlConfig", Method: http.MethodDelete, IAMAction: "lambda:DeleteFunctionUrlConfig"},
		// Permissions
		{Name: "AddPermission", Method: http.MethodPost, IAMAction: "lambda:AddPermission"},
		{Name: "GetPolicy", Method: http.MethodGet, IAMAction: "lambda:GetPolicy"},
		{Name: "RemovePermission", Method: http.MethodDelete, IAMAction: "lambda:RemovePermission"},
		// Concurrency
		{Name: "PutFunctionConcurrency", Method: http.MethodPut, IAMAction: "lambda:PutFunctionConcurrency"},
		{Name: "GetFunctionConcurrency", Method: http.MethodGet, IAMAction: "lambda:GetFunctionConcurrency"},
		{Name: "DeleteFunctionConcurrency", Method: http.MethodDelete, IAMAction: "lambda:DeleteFunctionConcurrency"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "lambda:TagResource"},
		{Name: "ListTags", Method: http.MethodGet, IAMAction: "lambda:ListTags"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "lambda:UntagResource"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *LambdaService) HealthCheck() error { return nil }

// ResourceSchemas returns the schema for Lambda function resources.
func (s *LambdaService) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "lambda",
			ResourceType:  "aws_lambda_function",
			TerraformType: "cloudmock_lambda_function",
			AWSType:       "AWS::Lambda::Function",
			CreateAction:  "CreateFunction",
			ReadAction:    "GetFunction",
			UpdateAction:  "UpdateFunctionCode",
			DeleteAction:  "DeleteFunction",
			ListAction:    "ListFunctions",
			ImportID:      "function_name",
			Attributes: []schema.AttributeSchema{
				{Name: "function_name", Type: "string", Required: true, ForceNew: true},
				{Name: "runtime", Type: "string", Required: true},
				{Name: "role", Type: "string", Required: true},
				{Name: "handler", Type: "string", Required: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "invoke_arn", Type: "string", Computed: true},
				{Name: "last_modified", Type: "string", Computed: true},
				{Name: "memory_size", Type: "int", Default: 128},
				{Name: "timeout", Type: "int", Default: 3},
				{Name: "environment", Type: "map"},
				{Name: "tags", Type: "map"},
			},
		},
		{
			ServiceName:   "lambda",
			ResourceType:  "aws_lambda_event_source_mapping",
			TerraformType: "cloudmock_lambda_event_source_mapping",
			AWSType:       "AWS::Lambda::EventSourceMapping",
			CreateAction:  "CreateEventSourceMapping",
			ReadAction:    "GetEventSourceMapping",
			DeleteAction:  "DeleteEventSourceMapping",
			ListAction:    "ListEventSourceMappings",
			ImportID:      "uuid",
			Attributes: []schema.AttributeSchema{
				{Name: "function_name", Type: "string", Required: true},
				{Name: "event_source_arn", Type: "string", Required: true, ForceNew: true},
				{Name: "uuid", Type: "string", Computed: true},
				{Name: "batch_size", Type: "int", Default: 10},
				{Name: "enabled", Type: "bool", Default: true},
			},
		},
	}
}

// Logs returns the Lambda execution log buffer.
func (s *LambdaService) Logs() *LogBuffer {
	return s.executor.Logs()
}

// InvokeDirect invokes a Lambda function by name with the given event payload.
// This is used for cross-service delivery (SNS → Lambda, EventBridge → Lambda,
// SQS → Lambda event source mappings). Returns the result bytes, or an error.
func (s *LambdaService) InvokeDirect(functionName string, event []byte) ([]byte, error) {
	fn, ok := s.store.Get(functionName)
	if !ok {
		return nil, fmt.Errorf("function not found: %s", functionName)
	}
	return s.executor.Invoke(fn, event)
}

// Store returns the function store (used by event source mapping pollers).
func (s *LambdaService) Store() *FunctionStore {
	return s.store
}

// GetEventSourceMappings returns all event source mappings for topology queries.
func (s *LambdaService) GetEventSourceMappings() []*EventSourceMapping {
	return s.esmStore.List("", "")
}

// GetFunctionNames returns all Lambda function names for topology queries.
func (s *LambdaService) GetFunctionNames() []string {
	fns := s.store.List()
	names := make([]string, 0, len(fns))
	for _, fn := range fns {
		names = append(names, fn.FunctionName)
	}
	return names
}

// GetEventSourceMappingsSummary returns parallel slices of event source ARNs and function names
// for topology building without exposing internal types.
func (s *LambdaService) GetEventSourceMappingsSummary() (arns []string, funcNames []string) {
	mappings := s.esmStore.List("", "")
	arns = make([]string, 0, len(mappings))
	funcNames = make([]string, 0, len(mappings))
	for _, m := range mappings {
		arns = append(arns, m.EventSourceArn)
		funcNames = append(funcNames, m.FunctionName)
	}
	return arns, funcNames
}

// HandleRequest routes an incoming Lambda request to the appropriate handler.
// Lambda uses REST path-based routing.
//
// Routes:
//
//	POST   /2015-03-31/functions                          -> CreateFunction
//	GET    /2015-03-31/functions                          -> ListFunctions
//	GET    /2015-03-31/functions/{name}                   -> GetFunction
//	DELETE /2015-03-31/functions/{name}                   -> DeleteFunction
//	PUT    /2015-03-31/functions/{name}/code              -> UpdateFunctionCode
//	POST   /2015-03-31/functions/{name}/invocations       -> Invoke
//	GET    /2015-03-31/functions/{name}/configuration     -> GetFunctionConfiguration
//	PUT    /2015-03-31/functions/{name}/configuration     -> UpdateFunctionConfiguration
func (s *LambdaService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	// Event source mapping routes: /2015-03-31/event-source-mappings
	const esmPrefix = "/2015-03-31/event-source-mappings"
	if strings.HasPrefix(path, esmPrefix) {
		return s.handleESMRequest(ctx, method, path, esmPrefix)
	}

	// Layer routes: /2018-10-31/layers
	const layersPrefix = "/2018-10-31/layers"
	if strings.HasPrefix(path, layersPrefix) {
		return s.handleLayerRequest(ctx, method, path, layersPrefix)
	}

	// Tag routes: /2017-03-31/tags
	const tagsPrefix = "/2017-03-31/tags"
	if strings.HasPrefix(path, tagsPrefix) {
		arn := strings.TrimPrefix(path, tagsPrefix+"/")
		switch method {
		case http.MethodGet:
			return handleListTags(ctx, s.versions, arn)
		case http.MethodPost:
			return handleTagResource(ctx, s.versions, arn)
		case http.MethodDelete:
			return handleUntagResource(ctx, s.versions, arn)
		}
		return lambdaNotImplemented()
	}

	const basePrefix = "/2015-03-31/functions"

	if !strings.HasPrefix(path, basePrefix) {
		return jsonErr(service.NewAWSError("NotImplemented",
			"Route not implemented by cloudmock.", http.StatusNotImplemented))
	}

	rest := path[len(basePrefix):]

	// POST /2015-03-31/functions -> CreateFunction
	// GET  /2015-03-31/functions -> ListFunctions
	if rest == "" {
		switch method {
		case http.MethodPost:
			return handleCreateFunction(ctx, s.store, s.executor, s.locator)
		case http.MethodGet:
			return handleListFunctions(ctx, s.store)
		}
		return lambdaNotImplemented()
	}

	// rest starts with "/"
	parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", 2)
	if len(parts) == 0 {
		return lambdaNotImplemented()
	}

	funcName := parts[0]

	// /2015-03-31/functions/{name}
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetFunction(ctx, s.store, funcName)
		case http.MethodDelete:
			return handleDeleteFunction(ctx, s.store, s.executor, funcName)
		}
		return lambdaNotImplemented()
	}

	subPath := parts[1]

	// Split subPath further for nested resources (e.g., "aliases/myAlias")
	subParts := strings.SplitN(subPath, "/", 2)
	subResource := subParts[0]
	subID := ""
	if len(subParts) > 1 {
		subID = subParts[1]
	}

	switch subResource {
	case "code":
		if method == http.MethodPut {
			return handleUpdateFunctionCode(ctx, s.store, s.executor, s.locator, funcName)
		}
	case "invocations":
		if method == http.MethodPost {
			return handleInvoke(ctx, s.store, s.executor, funcName)
		}
	case "configuration":
		switch method {
		case http.MethodGet:
			return handleGetFunctionConfiguration(ctx, s.store, funcName)
		case http.MethodPut:
			return handleUpdateFunctionConfiguration(ctx, s.store, funcName)
		}
	case "versions":
		switch method {
		case http.MethodPost:
			return handlePublishVersion(ctx, s.store, s.versions, funcName)
		case http.MethodGet:
			return handleListVersionsByFunction(ctx, s.versions, funcName)
		}
	case "aliases":
		if subID == "" {
			switch method {
			case http.MethodPost:
				return handleCreateAlias(ctx, s.store, s.versions, funcName)
			case http.MethodGet:
				return handleListAliases(ctx, s.versions, funcName)
			}
		} else {
			switch method {
			case http.MethodGet:
				return handleGetAlias(ctx, s.versions, funcName, subID)
			case http.MethodPut:
				return handleUpdateAlias(ctx, s.versions, funcName, subID)
			case http.MethodDelete:
				return handleDeleteAlias(ctx, s.versions, funcName, subID)
			}
		}
	case "url":
		switch method {
		case http.MethodPost:
			return handleCreateFunctionUrlConfig(ctx, s.store, s.versions, funcName)
		case http.MethodGet:
			return handleGetFunctionUrlConfig(ctx, s.versions, funcName)
		case http.MethodDelete:
			return handleDeleteFunctionUrlConfig(ctx, s.versions, funcName)
		}
	case "policy":
		if subID == "" {
			switch method {
			case http.MethodPost:
				return handleAddPermission(ctx, s.versions, funcName)
			case http.MethodGet:
				return handleGetPolicy(ctx, s.versions, funcName)
			}
		} else {
			if method == http.MethodDelete {
				return handleRemovePermission(ctx, s.versions, funcName, subID)
			}
		}
	case "concurrency":
		switch method {
		case http.MethodPut:
			return handlePutFunctionConcurrency(ctx, s.versions, funcName)
		case http.MethodGet:
			return handleGetFunctionConcurrency(ctx, s.versions, funcName)
		case http.MethodDelete:
			return handleDeleteFunctionConcurrency(ctx, s.versions, funcName)
		}
	}

	return lambdaNotImplemented()
}

// handleESMRequest routes event source mapping requests.
func (s *LambdaService) handleESMRequest(ctx *service.RequestContext, method, path, prefix string) (*service.Response, error) {
	rest := path[len(prefix):]

	// POST /2015-03-31/event-source-mappings -> CreateEventSourceMapping
	// GET  /2015-03-31/event-source-mappings -> ListEventSourceMappings
	if rest == "" {
		switch method {
		case http.MethodPost:
			return handleCreateEventSourceMapping(ctx, s)
		case http.MethodGet:
			return handleListEventSourceMappings(ctx, s)
		}
		return lambdaNotImplemented()
	}

	// GET    /2015-03-31/event-source-mappings/{uuid} -> GetEventSourceMapping
	// DELETE /2015-03-31/event-source-mappings/{uuid} -> DeleteEventSourceMapping
	uuid := strings.TrimPrefix(rest, "/")
	switch method {
	case http.MethodGet:
		return handleGetEventSourceMapping(ctx, s, uuid)
	case http.MethodDelete:
		return handleDeleteEventSourceMapping(ctx, s, uuid)
	}
	return lambdaNotImplemented()
}

// handleLayerRequest routes layer-related requests.
func (s *LambdaService) handleLayerRequest(ctx *service.RequestContext, method, path, prefix string) (*service.Response, error) {
	rest := strings.TrimPrefix(path, prefix)

	// GET /2018-10-31/layers -> ListLayers
	if rest == "" && method == http.MethodGet {
		return handleListLayers(ctx, s.layers)
	}

	// /2018-10-31/layers/{layerName}/versions[/{versionNumber}]
	parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", 3)
	if len(parts) < 1 {
		return lambdaNotImplemented()
	}
	layerName := parts[0]

	if len(parts) == 1 {
		return lambdaNotImplemented()
	}

	if parts[1] == "versions" {
		if len(parts) == 2 {
			// POST /layers/{name}/versions -> PublishLayerVersion
			// GET  /layers/{name}/versions -> ListLayerVersions
			switch method {
			case http.MethodPost:
				return handlePublishLayerVersion(ctx, s.layers, layerName)
			case http.MethodGet:
				return handleListLayerVersions(ctx, s.layers, layerName)
			}
		}
		if len(parts) == 3 {
			// GET    /layers/{name}/versions/{num} -> GetLayerVersion
			// DELETE /layers/{name}/versions/{num} -> DeleteLayerVersion
			versionStr := parts[2]
			switch method {
			case http.MethodGet:
				return handleGetLayerVersion(ctx, s.layers, layerName, versionStr)
			case http.MethodDelete:
				return handleDeleteLayerVersion(ctx, s.layers, layerName, versionStr)
			}
		}
	}

	return lambdaNotImplemented()
}

func lambdaNotImplemented() (*service.Response, error) {
	return jsonErr(service.NewAWSError("NotImplemented",
		"This method and path combination is not implemented by cloudmock.", http.StatusNotImplemented))
}
