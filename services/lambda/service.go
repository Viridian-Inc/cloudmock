package lambda

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
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
}

// New returns a new LambdaService for the given AWS account ID and region.
func New(accountID, region string) *LambdaService {
	return &LambdaService{
		store:    NewStore(accountID, region),
		executor: NewExecutor(),
		esmStore: NewEventSourceMappingStore(),
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
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *LambdaService) HealthCheck() error { return nil }

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

	switch subPath {
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

func lambdaNotImplemented() (*service.Response, error) {
	return jsonErr(service.NewAWSError("NotImplemented",
		"This method and path combination is not implemented by cloudmock.", http.StatusNotImplemented))
}
