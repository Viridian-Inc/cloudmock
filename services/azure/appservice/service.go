package appservice

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const appServiceAPIVersion = "2024-04-01"
const defaultFunctionsAccountName = "devstoreaccount1"

type hostKeysState struct {
	FunctionKeys map[string]string
	SystemKeys   map[string]string
	MasterKey    string
}

// AppService implements first-slice Azure App Service control-plane APIs.
type AppService struct {
	mu                sync.RWMutex
	plans             map[string]AppServicePlan
	sites             map[string]Site
	slots             map[string]Site
	functions         map[string][]Function
	functionKeys      map[string]map[string]string
	hostKeys          map[string]hostKeysState
	appSettings       map[string]map[string]string
	connectionStrings map[string]map[string]map[string]any
	slotConfigNames   map[string]SlotConfigNamesResource
	localFunctionApps map[string]LocalFunctionApp
	localFunctions    map[string]LocalFunction
}

func New() *AppService {
	return &AppService{
		plans:             make(map[string]AppServicePlan),
		sites:             make(map[string]Site),
		slots:             make(map[string]Site),
		functions:         make(map[string][]Function),
		functionKeys:      make(map[string]map[string]string),
		hostKeys:          make(map[string]hostKeysState),
		appSettings:       make(map[string]map[string]string),
		connectionStrings: make(map[string]map[string]map[string]any),
		slotConfigNames:   make(map[string]SlotConfigNamesResource),
		localFunctionApps: make(map[string]LocalFunctionApp),
		localFunctions:    make(map[string]LocalFunction),
	}
}

func (s *AppService) Name() string { return "Microsoft.Web" }

func (s *AppService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdatePlan", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/serverfarms/write"},
		{Name: "UpdatePlan", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Web/serverfarms/write"},
		{Name: "GetPlan", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/serverfarms/read"},
		{Name: "ListPlans", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/serverfarms/read"},
		{Name: "DeletePlan", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/serverfarms/delete"},
		{Name: "CreateOrUpdateSite", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/write"},
		{Name: "UpdateSite", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Web/sites/write"},
		{Name: "GetSite", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/read"},
		{Name: "ListSites", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/read"},
		{Name: "DeleteSite", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/sites/delete"},
		{Name: "CreateOrUpdateSlot", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/slots/write"},
		{Name: "UpdateSlot", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Web/sites/slots/write"},
		{Name: "GetSlot", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/slots/read"},
		{Name: "ListSlots", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/slots/read"},
		{Name: "DeleteSlot", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/sites/slots/delete"},
		{Name: "GetSlotConfiguration", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/slots/config/read"},
		{Name: "CreateOrUpdateSlotConfiguration", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/slots/config/write"},
		{Name: "UpdateSlotConfiguration", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Web/sites/slots/config/write"},
		{Name: "StartSite", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/start/action"},
		{Name: "StopSite", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/stop/action"},
		{Name: "RestartSite", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/restart/action"},
		{Name: "SyncFunctionTriggers", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/syncfunctiontriggers/action"},
		{Name: "GetConfiguration", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/config/read"},
		{Name: "UpdateConfiguration", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Web/sites/config/write"},
		{Name: "UpdateApplicationSettings", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/config/write"},
		{Name: "ListApplicationSettings", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/config/read"},
		{Name: "UpdateConnectionStrings", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/config/write"},
		{Name: "ListConnectionStrings", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/config/read"},
		{Name: "ListSlotConfigurationNames", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/config/read"},
		{Name: "UpdateSlotConfigurationNames", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/config/write"},
		{Name: "ListPublishingCredentials", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/config/list/action"},
		{Name: "ListPublishingProfileXMLWithSecrets", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/publishxml/action"},
		{Name: "ListFunctions", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/sites/functions/read"},
		{Name: "CreateOrUpdateFunction", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/functions/write"},
		{Name: "ListFunctionKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/functions/listkeys/action"},
		{Name: "CreateOrUpdateFunctionSecret", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/functions/keys/write"},
		{Name: "DeleteFunctionSecret", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/sites/functions/keys/delete"},
		{Name: "ListHostKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/sites/host/listkeys/action"},
		{Name: "CreateOrUpdateHostSecret", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/sites/host/keys/write"},
		{Name: "DeleteHostSecret", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/sites/host/keys/delete"},
		{Name: "CreateOrUpdateFunctionApp", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/functions/apps/write"},
		{Name: "GetFunctionApp", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/functions/apps/read"},
		{Name: "ListFunctionApps", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/functions/apps/read"},
		{Name: "DeleteFunctionApp", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/functions/apps/delete"},
		{Name: "DeployFunction", Method: http.MethodPut, IAMAction: "azure:Microsoft.Web/functions/write"},
		{Name: "GetFunction", Method: http.MethodGet, IAMAction: "azure:Microsoft.Web/functions/read"},
		{Name: "DeleteFunction", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Web/functions/delete"},
		{Name: "InvokeFunction", Method: http.MethodPost, IAMAction: "azure:Microsoft.Web/functions/action"},
	}
}

func (s *AppService) HealthCheck() error { return nil }

func (s *AppService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Web/serverfarms", APIVersion: appServiceAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Web/sites", APIVersion: appServiceAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Web/functions", APIVersion: appServiceAPIVersion},
	}
}

func (s *AppService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.Web/serverfarms") ||
		strings.EqualFold(resourceType, "Microsoft.Web/sites") ||
		strings.EqualFold(resourceType, "Microsoft.Web/sites/slots")
}

func (s *AppService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported App Service template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("App Service template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"kind":       resource["kind"],
		"identity":   resource["identity"],
		"tags":       resource["tags"],
		"sku":        resource["sku"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	resourceType := stringValue(resource["type"])
	switch {
	case strings.EqualFold(resourceType, "Microsoft.Web/serverfarms"):
		resp, err = s.createOrUpdatePlan(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Web/sites"):
		resp, err = s.createOrUpdateSite(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Web/sites/slots"):
		siteName, slotName, ok := splitSlotTemplateName(name)
		if !ok {
			err = fmt.Errorf("App Service slot template resource name must be formatted as '<site>/<slot>'")
			break
		}
		resp, err = s.createOrUpdateSlot(subscriptionID, resourceGroup, siteName, slotName, data)
	default:
		err = fmt.Errorf("unsupported App Service template resource type %q", resourceType)
	}
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AppService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if accountName, parts, ok := functionsLocalPath(ctx.RawRequest); ok {
		return s.handleLocalFunctionsRequest(ctx, accountName, parts)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service route is not implemented.")
	}

	switch {
	case strings.EqualFold(route.ResourceType, "serverfarms"):
		return s.handlePlanRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "sites"):
		return s.handleSiteRequest(ctx, route)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service route is not implemented.")
	}
}

func (s *AppService) handleLocalFunctionsRequest(ctx *service.RequestContext, accountName string, parts []string) (*service.Response, error) {
	if len(parts) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Functions route is not implemented.")
	}
	switch {
	case strings.EqualFold(parts[0], "admin"):
		return s.handleLocalFunctionsAdminRequest(ctx, accountName, parts[1:])
	case strings.EqualFold(parts[0], "api"):
		return s.invokeLocalFunction(ctx, accountName, parts[1:])
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Functions route is not implemented.")
	}
}

func (s *AppService) handleLocalFunctionsAdminRequest(ctx *service.RequestContext, accountName string, parts []string) (*service.Response, error) {
	if len(parts) == 0 || !strings.EqualFold(parts[0], "apps") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "Only /admin/apps/... is supported.")
	}
	if len(parts) == 1 {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listLocalFunctionApps(accountName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	appName := parts[1]
	if len(parts) == 2 {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createOrUpdateLocalFunctionApp(accountName, appName, ctx.Body)
		case http.MethodGet:
			return s.getLocalFunctionApp(accountName, appName)
		case http.MethodDelete:
			return s.deleteLocalFunctionApp(accountName, appName)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}

	if len(parts) == 3 && strings.EqualFold(parts[2], "functions") {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listLocalFunctions(accountName, appName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if len(parts) == 4 && strings.EqualFold(parts[2], "functions") {
		functionName := parts[3]
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.deployLocalFunction(accountName, appName, functionName, ctx.Body)
		case http.MethodGet:
			return s.getLocalFunction(accountName, appName, functionName)
		case http.MethodDelete:
			return s.deleteLocalFunction(accountName, appName, functionName)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Functions admin route is not implemented.")
}

func (s *AppService) createOrUpdateLocalFunctionApp(accountName, appName string, body []byte) (*service.Response, error) {
	var input struct {
		Runtime        string            `json:"runtime"`
		LinuxFxVersion string            `json:"linuxFxVersion"`
		Environment    map[string]string `json:"environment"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "Invalid request body.")
		}
	}
	runtime := firstNonBlank(input.Runtime, runtimeFromLinuxFxVersion(input.LinuxFxVersion))
	if runtime == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "runtime is required.")
	}

	app := LocalFunctionApp{
		Name:           appName,
		Runtime:        runtime,
		LinuxFxVersion: input.LinuxFxVersion,
		Environment:    input.Environment,
		Status:         "Running",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}

	s.mu.Lock()
	s.localFunctionApps[localFunctionAppKey(accountName, appName)] = app
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusCreated, app)
}

func (s *AppService) getLocalFunctionApp(accountName, appName string) (*service.Response, error) {
	s.mu.RLock()
	app, ok := s.localFunctionApps[localFunctionAppKey(accountName, appName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "AppNotFound", fmt.Sprintf("Function app %q not found.", appName))
	}
	return azurearm.JSONResponse(http.StatusOK, app)
}

func (s *AppService) listLocalFunctionApps(accountName string) (*service.Response, error) {
	prefix := localFunctionAppKey(accountName, "")
	s.mu.RLock()
	values := make([]LocalFunctionApp, 0)
	for key, app := range s.localFunctionApps {
		if strings.HasPrefix(key, prefix) {
			values = append(values, app)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) deleteLocalFunctionApp(accountName, appName string) (*service.Response, error) {
	s.mu.Lock()
	delete(s.localFunctionApps, localFunctionAppKey(accountName, appName))
	prefix := localFunctionPrefix(accountName, appName)
	for key := range s.localFunctions {
		if strings.HasPrefix(key, prefix) {
			delete(s.localFunctions, key)
		}
	}
	s.mu.Unlock()
	return noContentResponse(), nil
}

func (s *AppService) deployLocalFunction(accountName, appName, functionName string, body []byte) (*service.Response, error) {
	var input struct {
		Handler        string            `json:"handler"`
		TimeoutSeconds int               `json:"timeoutSeconds"`
		Environment    map[string]string `json:"environment"`
		ZipBase64      string            `json:"zipBase64"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "Invalid request body.")
		}
	}

	s.mu.RLock()
	app, ok := s.localFunctionApps[localFunctionAppKey(accountName, appName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "AppNotFound", fmt.Sprintf("Function app %q not found.", appName))
	}

	environment := make(map[string]string)
	for key, value := range app.Environment {
		environment[key] = value
	}
	for key, value := range input.Environment {
		environment[key] = value
	}
	if len(environment) == 0 {
		environment = nil
	}

	status := "AwaitingDeploy"
	if strings.TrimSpace(input.ZipBase64) != "" {
		status = "Ready"
	}
	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 230
	}
	function := LocalFunction{
		Name:           functionName,
		AppName:        appName,
		Runtime:        app.Runtime,
		LinuxFxVersion: app.LinuxFxVersion,
		Handler:        firstNonBlank(input.Handler, "index.handler"),
		TimeoutSeconds: timeoutSeconds,
		Environment:    environment,
		InvokeURL:      localFunctionInvokeURL(accountName, appName, functionName),
		Status:         status,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}

	s.mu.Lock()
	s.localFunctions[localFunctionKey(accountName, appName, functionName)] = function
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusCreated, function)
}

func (s *AppService) getLocalFunction(accountName, appName, functionName string) (*service.Response, error) {
	s.mu.RLock()
	function, ok := s.localFunctions[localFunctionKey(accountName, appName, functionName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q not found in app %q.", functionName, appName))
	}
	return azurearm.JSONResponse(http.StatusOK, function)
}

func (s *AppService) listLocalFunctions(accountName, appName string) (*service.Response, error) {
	prefix := localFunctionPrefix(accountName, appName)
	s.mu.RLock()
	values := make([]LocalFunction, 0)
	for key, function := range s.localFunctions {
		if strings.HasPrefix(key, prefix) {
			values = append(values, function)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) deleteLocalFunction(accountName, appName, functionName string) (*service.Response, error) {
	s.mu.Lock()
	delete(s.localFunctions, localFunctionKey(accountName, appName, functionName))
	s.mu.Unlock()
	return noContentResponse(), nil
}

func (s *AppService) invokeLocalFunction(ctx *service.RequestContext, accountName string, parts []string) (*service.Response, error) {
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidPath", "Expected api/{appName}/{functionName}.")
	}
	appName := parts[0]
	functionName := parts[1]

	s.mu.RLock()
	function, ok := s.localFunctions[localFunctionKey(accountName, appName, functionName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q not found in app %q.", functionName, appName))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"appName":      function.AppName,
		"functionName": function.Name,
		"runtime":      function.Runtime,
		"method":       ctx.RawRequest.Method,
		"path":         ctx.RawRequest.URL.Path,
		"queryParams":  singleValueQuery(ctx.RawRequest.URL.Query()),
		"body":         string(ctx.Body),
	})
}

func (s *AppService) handlePlanRequest(ctx *service.RequestContext, route appServiceRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listPlans(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdatePlan(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodPatch:
		return s.updatePlan(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getPlan(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deletePlan(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AppService) handleSiteRequest(ctx *service.RequestContext, route appServiceRoute) (*service.Response, error) {
	if isSiteAction(route.ChildType) {
		if ctx.RawRequest.Method == http.MethodPost {
			return s.handleSiteAction(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildType)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.ChildType, "config") {
		return s.handleSiteConfigRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "publishxml") {
		if ctx.RawRequest.Method == http.MethodPost {
			return s.listPublishingProfileXMLWithSecrets(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.ChildType, "host") {
		return s.handleSiteHostRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "slots") {
		return s.handleSiteSlotRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "functions") {
		if route.ChildName == "" && ctx.RawRequest.Method == http.MethodGet {
			return s.listFunctions(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		if route.ChildName == "" {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if route.ChildAction != "" {
			switch strings.ToLower(route.ChildAction) {
			case "listkeys":
				if ctx.RawRequest.Method == http.MethodPost {
					return s.listFunctionKeys(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
				}
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			case "keys":
				if route.ChildActionName == "" {
					return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service function key route is not implemented.")
				}
				switch ctx.RawRequest.Method {
				case http.MethodPut:
					return s.createOrUpdateFunctionSecret(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.ChildActionName, ctx.Body)
				case http.MethodDelete:
					return s.deleteFunctionSecret(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.ChildActionName)
				default:
					return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
				}
			}
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service function route is not implemented.")
		}
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createOrUpdateFunction(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
		case http.MethodGet:
			return s.getFunction(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		case http.MethodDelete:
			return s.deleteFunction(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSites(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSite(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodPatch:
		return s.updateSite(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getSite(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteSite(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AppService) handleSiteSlotRequest(ctx *service.RequestContext, route appServiceRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSlots(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.ChildAction, "config") && strings.EqualFold(route.ChildActionName, "web") {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getSlotConfig(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		case http.MethodPut:
			return s.createOrUpdateSlotConfig(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
		case http.MethodPatch:
			return s.updateSlotConfig(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if route.ChildAction != "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service slot route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSlot(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodPatch:
		return s.updateSlot(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getSlot(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteSlot(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AppService) handleSiteHostRequest(ctx *service.RequestContext, route appServiceRoute) (*service.Response, error) {
	if !strings.EqualFold(route.ChildName, "default") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service host route is not implemented.")
	}
	if strings.EqualFold(route.ChildAction, "listkeys") {
		if ctx.RawRequest.Method == http.MethodPost {
			return s.listHostKeys(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.ChildAction == "" || route.ChildActionName == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service host route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateHostSecret(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildAction, route.ChildActionName, ctx.Body)
	case http.MethodDelete:
		return s.deleteHostSecret(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildAction, route.ChildActionName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AppService) createOrUpdatePlan(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Kind       string         `json:"kind"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["status"]; !ok {
		input.Properties["status"] = "Ready"
	}

	plan := AppServicePlan{
		ID:         planID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Web/serverfarms",
		Kind:       input.Kind,
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.plans[key]
	s.plans[key] = plan
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, plan)
}

func (s *AppService) updatePlan(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	plan, ok := s.plans[key]
	if ok {
		if kind := strings.TrimSpace(stringValue(input["kind"])); kind != "" {
			plan.Kind = kind
		}
		if sku, present := input["sku"].(map[string]any); present {
			plan.SKU = sku
		}
		if tags, present := input["tags"]; present {
			plan.Tags = stringifyTags(mapValue(tags))
		}
		if _, present := input["properties"]; present {
			merged := cloneMap(plan.Properties)
			for propKey, propValue := range mapValue(input["properties"]) {
				merged[propKey] = propValue
			}
			merged["provisioningState"] = "Succeeded"
			if _, hasStatus := merged["status"]; !hasStatus {
				merged["status"] = "Ready"
			}
			plan.Properties = merged
		}
		s.plans[key] = plan
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("App Service plan %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, plan)
}

func (s *AppService) handleSiteAction(subscriptionID, resourceGroup, name, action string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	site, ok := s.sites[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", name))
	}
	if site.Properties == nil {
		site.Properties = make(map[string]any)
	}

	switch strings.ToLower(action) {
	case "start":
		site.Properties["state"] = "Running"
		s.syncLocalFunctionAppStatusLocked(defaultFunctionsAccountName, name, "Running")
	case "stop":
		site.Properties["state"] = "Stopped"
		s.syncLocalFunctionAppStatusLocked(defaultFunctionsAccountName, name, "Stopped")
	case "restart":
		site.Properties["state"] = "Running"
		site.Properties["lastRestartedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
		site.Properties["restartCount"] = intValue(site.Properties["restartCount"]) + 1
		s.syncLocalFunctionAppStatusLocked(defaultFunctionsAccountName, name, "Running")
	case "syncfunctiontriggers":
		site.Properties["triggerMetadataSynced"] = true
		s.sites[key] = site
		return noContentResponse(), nil
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service action is not implemented.")
	}

	s.sites[key] = site
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *AppService) getPlan(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	plan, ok := s.plans[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("App Service plan %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, plan)
}

func (s *AppService) listPlans(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := resourcePrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]AppServicePlan, 0)
	for key, plan := range s.plans {
		if strings.HasPrefix(key, prefix) {
			values = append(values, plan)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) deletePlan(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.plans[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("App Service plan %q could not be found.", name))
	}
	delete(s.plans, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *AppService) createOrUpdateSite(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Kind       string         `json:"kind"`
		Identity   map[string]any `json:"identity"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Kind == "" {
		input.Kind = "app"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	siteConfig := mapValue(input.Properties["siteConfig"])
	if err := validateLinuxFxVersion(siteConfig); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}
	appSettings, hasAppSettings := appSettingsFromSiteConfig(siteConfig)
	connectionStrings, hasConnectionStrings := connectionStringsFromSiteConfig(siteConfig)
	functions := functionsFromInput(subscriptionID, resourceGroup, name, input.Location, input.Properties["functions"])
	delete(input.Properties, "functions")
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["state"]; !ok {
		input.Properties["state"] = "Running"
	}
	if _, ok := input.Properties["defaultHostName"]; !ok {
		input.Properties["defaultHostName"] = name + ".azurewebsites.net"
	}

	site := Site{
		ID:         siteID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Web/sites",
		Kind:       input.Kind,
		Location:   input.Location,
		Identity:   input.Identity,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.sites[key]
	s.sites[key] = site
	s.functions[key] = functions
	if hasAppSettings {
		s.appSettings[key] = appSettings
	}
	if hasConnectionStrings {
		s.connectionStrings[key] = connectionStrings
	}
	s.syncLocalFunctionAppLocked(defaultFunctionsAccountName, name, siteConfig)
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, site)
}

func (s *AppService) updateSite(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	patchProperties := mapValue(input["properties"])
	siteConfig := mapValue(patchProperties["siteConfig"])
	if err := validateLinuxFxVersion(siteConfig); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	site, ok := s.sites[key]
	if ok {
		if location := strings.TrimSpace(stringValue(input["location"])); location != "" {
			site.Location = location
		}
		if kind := strings.TrimSpace(stringValue(input["kind"])); kind != "" {
			site.Kind = kind
		}
		if identity, present := input["identity"].(map[string]any); present {
			site.Identity = identity
		}
		if tags, present := input["tags"]; present {
			site.Tags = stringifyTags(mapValue(tags))
		}
		if _, present := input["properties"]; present {
			merged := cloneMap(site.Properties)
			for propKey, propValue := range patchProperties {
				merged[propKey] = propValue
			}
			merged["provisioningState"] = "Succeeded"
			if _, hasState := merged["state"]; !hasState {
				merged["state"] = "Running"
			}
			if _, hasHostName := merged["defaultHostName"]; !hasHostName {
				merged["defaultHostName"] = name + ".azurewebsites.net"
			}
			site.Properties = merged
		}
		s.sites[key] = site
		if len(siteConfig) > 0 {
			s.syncLocalFunctionAppLocked(defaultFunctionsAccountName, name, siteConfig)
		}
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, site)
}

func (s *AppService) getSite(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	site, ok := s.sites[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, site)
}

func (s *AppService) listSites(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := resourcePrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]Site, 0)
	for key, site := range s.sites {
		if strings.HasPrefix(key, prefix) {
			values = append(values, site)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) handleSiteConfigRequest(ctx *service.RequestContext, route appServiceRoute) (*service.Response, error) {
	switch strings.ToLower(route.ChildName) {
	case "web":
		if route.ChildAction != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
		}
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getSiteConfig(route.SubscriptionID, route.ResourceGroup, route.Name)
		case http.MethodPut:
			return s.createOrUpdateSiteConfig(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		case http.MethodPatch:
			return s.updateSiteConfig(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	case "appsettings":
		if strings.EqualFold(route.ChildAction, "list") {
			if ctx.RawRequest.Method == http.MethodPost {
				return s.listSiteAppSettings(route.SubscriptionID, route.ResourceGroup, route.Name)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if route.ChildAction != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
		}
		if ctx.RawRequest.Method == http.MethodPut {
			return s.updateSiteAppSettings(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	case "connectionstrings":
		if strings.EqualFold(route.ChildAction, "list") {
			if ctx.RawRequest.Method == http.MethodPost {
				return s.listSiteConnectionStrings(route.SubscriptionID, route.ResourceGroup, route.Name)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if route.ChildAction != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
		}
		if ctx.RawRequest.Method == http.MethodPut {
			return s.updateSiteConnectionStrings(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	case "slotconfignames":
		if route.ChildAction != "" {
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
		}
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.listSlotConfigurationNames(route.SubscriptionID, route.ResourceGroup, route.Name)
		case http.MethodPut:
			return s.updateSlotConfigurationNames(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	case "publishingcredentials":
		if strings.EqualFold(route.ChildAction, "list") {
			if ctx.RawRequest.Method == http.MethodPost {
				return s.listPublishingCredentials(route.SubscriptionID, route.ResourceGroup, route.Name)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Service config route is not implemented.")
	}
}

func (s *AppService) getSiteConfig(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	site, ok := s.sites[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	return azurearm.JSONResponse(http.StatusOK, siteConfigResource(subscriptionID, resourceGroup, siteName, mapValue(site.Properties["siteConfig"])))
}

func (s *AppService) createOrUpdateSiteConfig(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	config := mapValue(input["properties"])
	if config == nil {
		config = input
	}
	if config == nil {
		config = make(map[string]any)
	}
	if err := validateLinuxFxVersion(config); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if site.Properties == nil {
		site.Properties = make(map[string]any)
	}
	site.Properties["siteConfig"] = config
	s.sites[key] = site
	s.syncLocalFunctionAppLocked(defaultFunctionsAccountName, siteName, config)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, siteConfigResource(subscriptionID, resourceGroup, siteName, config))
}

func (s *AppService) updateSiteConfig(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	patchConfig := mapValue(input["properties"])
	if patchConfig == nil {
		patchConfig = input
	}
	if patchConfig == nil {
		patchConfig = make(map[string]any)
	}
	if err := validateLinuxFxVersion(patchConfig); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if site.Properties == nil {
		site.Properties = make(map[string]any)
	}
	config := cloneMap(mapValue(site.Properties["siteConfig"]))
	for configKey, configValue := range patchConfig {
		config[configKey] = configValue
	}
	site.Properties["siteConfig"] = config
	s.sites[key] = site
	if appSettings, hasAppSettings := appSettingsFromSiteConfig(config); hasAppSettings {
		s.appSettings[key] = appSettings
	}
	if connectionStrings, hasConnectionStrings := connectionStringsFromSiteConfig(config); hasConnectionStrings {
		s.connectionStrings[key] = connectionStrings
	}
	s.syncLocalFunctionAppLocked(defaultFunctionsAccountName, siteName, config)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, siteConfigResource(subscriptionID, resourceGroup, siteName, config))
}

func (s *AppService) listSiteAppSettings(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	_, ok := s.sites[key]
	settings := cloneStringMap(s.appSettings[key])
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	return azurearm.JSONResponse(http.StatusOK, siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "appsettings", settings))
}

func (s *AppService) updateSiteAppSettings(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	settings, err := appSettingsFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	s.appSettings[key] = settings
	if site.Properties == nil {
		site.Properties = make(map[string]any)
	}
	siteConfig := mapValue(site.Properties["siteConfig"])
	if siteConfig == nil {
		siteConfig = make(map[string]any)
	}
	siteConfig["appSettings"] = appSettingsArray(settings)
	site.Properties["siteConfig"] = siteConfig
	s.sites[key] = site
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "appsettings", settings))
}

func (s *AppService) listSiteConnectionStrings(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	_, ok := s.sites[key]
	connections := cloneConnectionStringSet(s.connectionStrings[key])
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	return azurearm.JSONResponse(http.StatusOK, siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "connectionstrings", connections))
}

func (s *AppService) updateSiteConnectionStrings(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	connections, err := connectionStringsFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	s.connectionStrings[key] = connections
	if site.Properties == nil {
		site.Properties = make(map[string]any)
	}
	siteConfig := mapValue(site.Properties["siteConfig"])
	if siteConfig == nil {
		siteConfig = make(map[string]any)
	}
	siteConfig["connectionStrings"] = connectionStringArray(connections)
	site.Properties["siteConfig"] = siteConfig
	s.sites[key] = site
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "connectionstrings", connections))
}

func (s *AppService) listSlotConfigurationNames(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	site, ok := s.sites[key]
	resource, exists := s.slotConfigNames[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !exists {
		resource = slotConfigNamesResource(subscriptionID, resourceGroup, siteName, site.Kind, SlotConfigNameProperties{})
	}
	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *AppService) updateSlotConfigurationNames(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	var input struct {
		Kind       string                   `json:"kind"`
		Properties SlotConfigNameProperties `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = site.Kind
	}
	resource := slotConfigNamesResource(subscriptionID, resourceGroup, siteName, kind, input.Properties)
	s.slotConfigNames[key] = resource
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *AppService) listPublishingCredentials(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	_, ok := s.sites[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	return azurearm.JSONResponse(http.StatusOK, siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "publishingcredentials", publishingCredentialProperties(siteName)))
}

func (s *AppService) listPublishingProfileXMLWithSecrets(subscriptionID, resourceGroup, siteName string, body []byte) (*service.Response, error) {
	var input struct {
		Format string `json:"format"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	_, ok := s.sites[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(publishingProfileXML(siteName, input.Format)),
		RawContentType: "application/xml",
	}, nil
}

func (s *AppService) deleteSite(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.sites[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", name))
	}
	delete(s.sites, key)
	slotPrefix := key + "/slots/"
	for slotKey := range s.slots {
		if strings.HasPrefix(slotKey, slotPrefix) {
			delete(s.slots, slotKey)
		}
	}
	delete(s.functions, key)
	for functionKey := range s.functionKeys {
		if strings.HasPrefix(functionKey, key+"/") {
			delete(s.functionKeys, functionKey)
		}
	}
	delete(s.hostKeys, key)
	delete(s.appSettings, key)
	delete(s.connectionStrings, key)
	delete(s.slotConfigNames, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *AppService) createOrUpdateSlot(subscriptionID, resourceGroup, siteName, slotName string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Kind       string         `json:"kind"`
		Identity   map[string]any `json:"identity"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	siteConfig := mapValue(input.Properties["siteConfig"])
	if err := validateLinuxFxVersion(siteConfig); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	parent, parentExists := s.sites[parentKey]
	if !parentExists {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if input.Location == "" {
		input.Location = parent.Location
		if input.Location == "" {
			input.Location = "eastus"
		}
	}
	if input.Kind == "" {
		input.Kind = parent.Kind
		if input.Kind == "" {
			input.Kind = "app"
		}
	}
	defaultHostName := siteName + "-" + slotName + ".azurewebsites.net"
	if _, ok := input.Properties["provisioningState"]; !ok {
		input.Properties["provisioningState"] = "Succeeded"
	}
	if _, ok := input.Properties["state"]; !ok {
		input.Properties["state"] = "Running"
	}
	input.Properties["repositorySiteName"] = siteName
	if _, ok := input.Properties["defaultHostName"]; !ok {
		input.Properties["defaultHostName"] = defaultHostName
	}
	if _, ok := input.Properties["hostNames"]; !ok {
		input.Properties["hostNames"] = []string{defaultHostName}
	}
	if _, ok := input.Properties["enabledHostNames"]; !ok {
		input.Properties["enabledHostNames"] = []string{defaultHostName, siteName + "-" + slotName + ".scm.azurewebsites.net"}
	}

	slot := Site{
		ID:         slotID(subscriptionID, resourceGroup, siteName, slotName),
		Name:       siteName + "/" + slotName,
		Type:       "Microsoft.Web/sites/slots",
		Kind:       input.Kind,
		Location:   input.Location,
		Identity:   input.Identity,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}
	s.slots[slotKey(subscriptionID, resourceGroup, siteName, slotName)] = slot
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, slot)
}

func (s *AppService) updateSlot(subscriptionID, resourceGroup, siteName, slotName string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	patchProperties := mapValue(input["properties"])
	siteConfig := mapValue(patchProperties["siteConfig"])
	if err := validateLinuxFxVersion(siteConfig); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.Lock()
	_, parentExists := s.sites[parentKey]
	slot, slotExists := s.slots[key]
	if parentExists && slotExists {
		if location := strings.TrimSpace(stringValue(input["location"])); location != "" {
			slot.Location = location
		}
		if kind := strings.TrimSpace(stringValue(input["kind"])); kind != "" {
			slot.Kind = kind
		}
		if identity, present := input["identity"].(map[string]any); present {
			slot.Identity = identity
		}
		if tags, present := input["tags"]; present {
			slot.Tags = stringifyTags(mapValue(tags))
		}
		if _, present := input["properties"]; present {
			merged := cloneMap(slot.Properties)
			for propKey, propValue := range patchProperties {
				merged[propKey] = propValue
			}
			defaultHostName := siteName + "-" + slotName + ".azurewebsites.net"
			merged["provisioningState"] = "Succeeded"
			if _, hasState := merged["state"]; !hasState {
				merged["state"] = "Running"
			}
			merged["repositorySiteName"] = siteName
			if _, hasHostName := merged["defaultHostName"]; !hasHostName {
				merged["defaultHostName"] = defaultHostName
			}
			if _, hasHostNames := merged["hostNames"]; !hasHostNames {
				merged["hostNames"] = []string{defaultHostName}
			}
			if _, hasEnabledHostNames := merged["enabledHostNames"]; !hasEnabledHostNames {
				merged["enabledHostNames"] = []string{defaultHostName, siteName + "-" + slotName + ".scm.azurewebsites.net"}
			}
			slot.Properties = merged
		}
		s.slots[key] = slot
	}
	s.mu.Unlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return azurearm.JSONResponse(http.StatusOK, slot)
}

func (s *AppService) getSlot(subscriptionID, resourceGroup, siteName, slotName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.RLock()
	_, parentExists := s.sites[parentKey]
	slot, slotExists := s.slots[key]
	s.mu.RUnlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return azurearm.JSONResponse(http.StatusOK, slot)
}

func (s *AppService) listSlots(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	prefix := parentKey + "/slots/"
	s.mu.RLock()
	_, parentExists := s.sites[parentKey]
	values := make([]Site, 0)
	if parentExists {
		for key, slot := range s.slots {
			if strings.HasPrefix(key, prefix) {
				values = append(values, slot)
			}
		}
	}
	s.mu.RUnlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) deleteSlot(subscriptionID, resourceGroup, siteName, slotName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.Lock()
	_, parentExists := s.sites[parentKey]
	_, slotExists := s.slots[key]
	if slotExists {
		delete(s.slots, key)
	}
	s.mu.Unlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return noContentResponse(), nil
}

func (s *AppService) getSlotConfig(subscriptionID, resourceGroup, siteName, slotName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.RLock()
	_, parentExists := s.sites[parentKey]
	slot, slotExists := s.slots[key]
	s.mu.RUnlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return azurearm.JSONResponse(http.StatusOK, slotConfigResource(subscriptionID, resourceGroup, siteName, slotName, mapValue(slot.Properties["siteConfig"])))
}

func (s *AppService) createOrUpdateSlotConfig(subscriptionID, resourceGroup, siteName, slotName string, body []byte) (*service.Response, error) {
	config, err := slotConfigFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if err := validateLinuxFxVersion(config); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.Lock()
	_, parentExists := s.sites[parentKey]
	slot, slotExists := s.slots[key]
	if parentExists && slotExists {
		if slot.Properties == nil {
			slot.Properties = make(map[string]any)
		}
		slot.Properties["siteConfig"] = config
		s.slots[key] = slot
	}
	s.mu.Unlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return azurearm.JSONResponse(http.StatusOK, slotConfigResource(subscriptionID, resourceGroup, siteName, slotName, config))
}

func (s *AppService) updateSlotConfig(subscriptionID, resourceGroup, siteName, slotName string, body []byte) (*service.Response, error) {
	patch, err := slotConfigFromBody(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if err := validateLinuxFxVersion(patch); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidLinuxFxVersion", err.Error())
	}

	parentKey := resourceKey(subscriptionID, resourceGroup, siteName)
	key := slotKey(subscriptionID, resourceGroup, siteName, slotName)
	s.mu.Lock()
	_, parentExists := s.sites[parentKey]
	slot, slotExists := s.slots[key]
	var config map[string]any
	if parentExists && slotExists {
		config = cloneMap(mapValue(slot.Properties["siteConfig"]))
		for name, value := range patch {
			config[name] = value
		}
		if slot.Properties == nil {
			slot.Properties = make(map[string]any)
		}
		slot.Properties["siteConfig"] = config
		s.slots[key] = slot
	}
	s.mu.Unlock()
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !slotExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Deployment slot %q could not be found.", slotName))
	}
	return azurearm.JSONResponse(http.StatusOK, slotConfigResource(subscriptionID, resourceGroup, siteName, slotName, config))
}

func slotConfigFromBody(body []byte) (map[string]any, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return nil, err
		}
	}
	config := mapValue(input["properties"])
	if config == nil {
		config = input
	}
	if config == nil {
		config = make(map[string]any)
	}
	return config, nil
}

func (s *AppService) listFunctions(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)

	s.mu.RLock()
	_, siteExists := s.sites[key]
	values := append([]Function(nil), s.functions[key]...)
	s.mu.RUnlock()

	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppService) createOrUpdateFunction(subscriptionID, resourceGroup, siteName, functionName string, body []byte) (*service.Response, error) {
	var input struct {
		Kind       string         `json:"kind"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Kind == "" {
		input.Kind = "function"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	site, ok := s.sites[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	function := functionEnvelopeFromProperties(subscriptionID, resourceGroup, siteName, functionName, site.Location, input.Kind, input.Properties)
	s.functions[key] = upsertFunction(s.functions[key], function)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusCreated, function)
}

func (s *AppService) getFunction(subscriptionID, resourceGroup, siteName, functionName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.RLock()
	_, siteExists := s.sites[key]
	function, functionExists := findFunction(s.functions[key], functionName)
	s.mu.RUnlock()
	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !functionExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q does not exist.", functionName))
	}
	return azurearm.JSONResponse(http.StatusOK, function)
}

func (s *AppService) deleteFunction(subscriptionID, resourceGroup, siteName, functionName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	s.mu.Lock()
	_, siteExists := s.sites[key]
	functions, removed := deleteFunctionFromList(s.functions[key], functionName)
	if removed {
		s.functions[key] = functions
		delete(s.functionKeys, functionKeyStateKey(subscriptionID, resourceGroup, siteName, functionName))
	}
	s.mu.Unlock()
	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !removed {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q does not exist.", functionName))
	}
	return noContentResponse(), nil
}

func (s *AppService) listFunctionKeys(subscriptionID, resourceGroup, siteName, functionName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	stateKey := functionKeyStateKey(subscriptionID, resourceGroup, siteName, functionName)

	s.mu.RLock()
	_, siteExists := s.sites[key]
	_, functionExists := findFunction(s.functions[key], functionName)
	storedKeys := cloneStringMap(s.functionKeys[stateKey])
	s.mu.RUnlock()

	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !functionExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q does not exist.", functionName))
	}

	properties := map[string]string{
		"default": defaultFunctionKeyValue(siteName, functionName),
	}
	for name, value := range storedKeys {
		properties[name] = value
	}
	return azurearm.JSONResponse(http.StatusOK, siteFunctionKeysResource(subscriptionID, resourceGroup, siteName, functionName, properties))
}

func (s *AppService) createOrUpdateFunctionSecret(subscriptionID, resourceGroup, siteName, functionName, keyName string, body []byte) (*service.Response, error) {
	var input struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = keyName
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	stateKey := functionKeyStateKey(subscriptionID, resourceGroup, siteName, functionName)

	s.mu.Lock()
	_, siteExists := s.sites[key]
	_, functionExists := findFunction(s.functions[key], functionName)
	if !siteExists {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !functionExists {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q does not exist.", functionName))
	}
	if s.functionKeys[stateKey] == nil {
		s.functionKeys[stateKey] = make(map[string]string)
	}
	_, existed := s.functionKeys[stateKey][keyName]
	s.functionKeys[stateKey][keyName] = input.Value
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, map[string]string{
		"name":  input.Name,
		"value": input.Value,
	})
}

func (s *AppService) deleteFunctionSecret(subscriptionID, resourceGroup, siteName, functionName, keyName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	stateKey := functionKeyStateKey(subscriptionID, resourceGroup, siteName, functionName)

	s.mu.Lock()
	_, siteExists := s.sites[key]
	_, functionExists := findFunction(s.functions[key], functionName)
	_, keyExists := s.functionKeys[stateKey][keyName]
	if keyExists {
		delete(s.functionKeys[stateKey], keyName)
		if len(s.functionKeys[stateKey]) == 0 {
			delete(s.functionKeys, stateKey)
		}
	}
	s.mu.Unlock()

	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !functionExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionNotFound", fmt.Sprintf("Function %q does not exist.", functionName))
	}
	if !keyExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "FunctionKeyNotFound", fmt.Sprintf("Function key %q does not exist.", keyName))
	}
	return noContentResponse(), nil
}

func (s *AppService) listHostKeys(subscriptionID, resourceGroup, siteName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)

	s.mu.RLock()
	_, siteExists := s.sites[key]
	state := s.hostKeys[key]
	s.mu.RUnlock()

	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}

	functionKeys := map[string]string{
		"default": defaultHostFunctionKeyValue(siteName),
	}
	for name, value := range state.FunctionKeys {
		functionKeys[name] = value
	}
	masterKey := state.MasterKey
	if masterKey == "" {
		masterKey = defaultHostMasterKeyValue(siteName)
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"masterKey":    masterKey,
		"functionKeys": functionKeys,
		"systemKeys":   cloneStringMap(state.SystemKeys),
	})
}

func (s *AppService) createOrUpdateHostSecret(subscriptionID, resourceGroup, siteName, keyType, keyName string, body []byte) (*service.Response, error) {
	var input struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = keyName
	}

	key := resourceKey(subscriptionID, resourceGroup, siteName)
	normalizedKeyType := normalizeHostKeyType(keyType)
	if normalizedKeyType == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "HostKeyTypeNotFound", fmt.Sprintf("Host key type %q is not supported.", keyType))
	}

	s.mu.Lock()
	_, siteExists := s.sites[key]
	if !siteExists {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	state := s.hostKeys[key]
	existed := hostSecretExists(state, normalizedKeyType, keyName)
	state = upsertHostSecret(state, normalizedKeyType, keyName, input.Value)
	s.hostKeys[key] = state
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, map[string]string{
		"name":  input.Name,
		"value": input.Value,
	})
}

func (s *AppService) deleteHostSecret(subscriptionID, resourceGroup, siteName, keyType, keyName string) (*service.Response, error) {
	key := resourceKey(subscriptionID, resourceGroup, siteName)
	normalizedKeyType := normalizeHostKeyType(keyType)
	if normalizedKeyType == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "HostKeyTypeNotFound", fmt.Sprintf("Host key type %q is not supported.", keyType))
	}

	s.mu.Lock()
	_, siteExists := s.sites[key]
	state := s.hostKeys[key]
	existed := hostSecretExists(state, normalizedKeyType, keyName)
	if existed {
		state = removeHostSecret(state, normalizedKeyType, keyName)
		if hostKeysStateIsEmpty(state) {
			delete(s.hostKeys, key)
		} else {
			s.hostKeys[key] = state
		}
	}
	s.mu.Unlock()

	if !siteExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Web app %q could not be found.", siteName))
	}
	if !existed {
		return azurearm.ErrorResponse(http.StatusNotFound, "HostKeyNotFound", fmt.Sprintf("Host key %q does not exist.", keyName))
	}
	return noContentResponse(), nil
}

type appServiceRoute struct {
	SubscriptionID  string
	ResourceGroup   string
	ResourceType    string
	Name            string
	ChildType       string
	ChildName       string
	ChildAction     string
	ChildActionName string
}

func parseRoute(escapedPath string) (appServiceRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Web") {
		return appServiceRoute{
			SubscriptionID: parts[1],
			ResourceType:   parts[4],
		}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Web") {
		return appServiceRoute{}, false
	}
	route := appServiceRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	case 11:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.ChildAction = parts[10]
		return route, true
	case 12:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.ChildAction = parts[10]
		route.ChildActionName = parts[11]
		return route, true
	default:
		return appServiceRoute{}, false
	}
}

func functionsFromInput(subscriptionID, resourceGroup, siteName, location string, raw any) []Function {
	rawList, ok := raw.([]any)
	if !ok || len(rawList) == 0 {
		return nil
	}

	out := make([]Function, 0, len(rawList))
	for _, item := range rawList {
		input, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringValue(input["name"])
		if name == "" {
			continue
		}

		props := cloneMap(input)
		if nested, ok := input["properties"].(map[string]any); ok {
			props = cloneMap(nested)
		}
		delete(props, "properties")
		props["name"] = name

		out = append(out, functionEnvelopeFromProperties(subscriptionID, resourceGroup, siteName, name, location, stringValue(input["kind"]), props))
	}
	return out
}

func functionEnvelopeFromProperties(subscriptionID, resourceGroup, siteName, functionName, location, kind string, properties map[string]any) Function {
	if kind == "" {
		kind = "function"
	}
	props := cloneMap(properties)
	siteResourceID := siteID(subscriptionID, resourceGroup, siteName)
	functionResourceID := functionID(subscriptionID, resourceGroup, siteName, functionName)
	href := siteResourceID + "/functions/" + functionName
	if _, ok := props["name"]; !ok {
		props["name"] = functionName
	}
	if _, ok := props["function_app_id"]; !ok {
		props["function_app_id"] = siteResourceID
	}
	if _, ok := props["href"]; !ok {
		props["href"] = href
	}
	if _, ok := props["config_href"]; !ok {
		props["config_href"] = href + "/config.json"
	}
	if _, ok := props["script_href"]; !ok {
		props["script_href"] = href + "/files/index"
	}
	if _, ok := props["script_root_path_href"]; !ok {
		props["script_root_path_href"] = href + "/files"
	}
	if _, ok := props["secrets_file_href"]; !ok {
		props["secrets_file_href"] = href + "/secrets"
	}
	if _, ok := props["test_data_href"]; !ok {
		props["test_data_href"] = href + "/testdata"
	}
	if _, ok := props["invoke_url_template"]; !ok {
		props["invoke_url_template"] = "https://" + siteName + ".azurewebsites.net/api/" + functionName
	}
	return Function{
		ID:         functionResourceID,
		Name:       siteName + "/" + functionName,
		Type:       "Microsoft.Web/sites/functions",
		Kind:       kind,
		Location:   location,
		Properties: props,
	}
}

func upsertFunction(functions []Function, function Function) []Function {
	for i, existing := range functions {
		if functionNameMatches(existing.Name, function.Name) {
			functions[i] = function
			return functions
		}
	}
	return append(functions, function)
}

func findFunction(functions []Function, functionName string) (Function, bool) {
	for _, function := range functions {
		if functionNameMatches(function.Name, functionName) {
			return function, true
		}
	}
	return Function{}, false
}

func deleteFunctionFromList(functions []Function, functionName string) ([]Function, bool) {
	for i, function := range functions {
		if functionNameMatches(function.Name, functionName) {
			return append(functions[:i], functions[i+1:]...), true
		}
	}
	return functions, false
}

func functionNameMatches(storedName, requestedName string) bool {
	return strings.EqualFold(functionLeafName(storedName), functionLeafName(requestedName))
}

func functionLeafName(name string) string {
	if index := strings.LastIndex(name, "/"); index >= 0 {
		return name[index+1:]
	}
	return name
}

func planID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Web/serverfarms/" + name
}

func siteID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Web/sites/" + name
}

func slotID(subscriptionID, resourceGroup, siteName, slotName string) string {
	return siteID(subscriptionID, resourceGroup, siteName) + "/slots/" + slotName
}

func slotKey(subscriptionID, resourceGroup, siteName, slotName string) string {
	return resourceKey(subscriptionID, resourceGroup, siteName) + "/slots/" + strings.ToLower(slotName)
}

func splitSlotTemplateName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func functionID(subscriptionID, resourceGroup, siteName, functionName string) string {
	return siteID(subscriptionID, resourceGroup, siteName) + "/functions/" + functionName
}

func siteFunctionKeysResource(subscriptionID, resourceGroup, siteName, functionName string, properties any) map[string]any {
	return map[string]any{
		"id":         functionID(subscriptionID, resourceGroup, siteName, functionName) + "/listkeys",
		"name":       siteName + "/" + functionName + "/listkeys",
		"type":       "Microsoft.Web/sites/functions/listkeys",
		"kind":       "function",
		"properties": properties,
	}
}

func siteConfigResource(subscriptionID, resourceGroup, siteName string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = make(map[string]any)
	}
	return siteNamedConfigResource(subscriptionID, resourceGroup, siteName, "web", properties)
}

func slotConfigResource(subscriptionID, resourceGroup, siteName, slotName string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = make(map[string]any)
	}
	return map[string]any{
		"id":         slotID(subscriptionID, resourceGroup, siteName, slotName) + "/config/web",
		"name":       "web",
		"type":       "Microsoft.Web/sites/slots/config",
		"properties": properties,
	}
}

func siteNamedConfigResource(subscriptionID, resourceGroup, siteName, configName string, properties any) map[string]any {
	if properties == nil {
		properties = make(map[string]any)
	}
	return map[string]any{
		"id":         siteID(subscriptionID, resourceGroup, siteName) + "/config/" + configName,
		"name":       configName,
		"type":       "Microsoft.Web/sites/config",
		"properties": properties,
	}
}

func slotConfigNamesResource(subscriptionID, resourceGroup, siteName, kind string, properties SlotConfigNameProperties) SlotConfigNamesResource {
	properties.AppSettingNames = cloneStringSlice(properties.AppSettingNames)
	properties.ConnectionStringNames = cloneStringSlice(properties.ConnectionStringNames)
	properties.AzureStorageConfigNames = cloneStringSlice(properties.AzureStorageConfigNames)
	return SlotConfigNamesResource{
		ID:         siteID(subscriptionID, resourceGroup, siteName) + "/config/slotConfigNames",
		Name:       "slotConfigNames",
		Type:       "Microsoft.Web/sites/config",
		Kind:       kind,
		Properties: properties,
	}
}

func publishingCredentialProperties(siteName string) map[string]any {
	userName := "$" + siteName
	password := publishingPassword(siteName)
	return map[string]any{
		"publishingUserName":         userName,
		"publishingPassword":         password,
		"publishingPasswordHash":     "",
		"publishingPasswordHashSalt": "",
		"scmUri":                     "https://" + userName + ":" + password + "@" + siteName + ".scm.azurewebsites.net",
	}
}

func publishingProfileXML(siteName, format string) string {
	userName := "$" + siteName
	password := publishingPassword(siteName)
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	if normalizedFormat == "" {
		normalizedFormat = "ftp"
	}

	var profiles []string
	if normalizedFormat == "webdeploy" {
		profiles = append(profiles, `<publishProfile profileName="`+xmlAttributeEscape(siteName+` - Web Deploy`)+`" publishMethod="MSDeploy" publishUrl="`+xmlAttributeEscape(siteName+`.scm.azurewebsites.net:443`)+`" msdeploySite="`+xmlAttributeEscape(siteName)+`" userName="`+xmlAttributeEscape(userName)+`" userPWD="`+xmlAttributeEscape(password)+`" destinationAppUrl="`+xmlAttributeEscape(`https://`+siteName+`.azurewebsites.net`)+`" SQLServerDBConnectionString="" mySQLDBConnectionString="" hostingProviderForumLink="" controlPanelLink="https://portal.azure.com" webSystem="WebSites" />`)
	} else {
		profiles = append(profiles, `<publishProfile profileName="`+xmlAttributeEscape(siteName+` - FTP`)+`" publishMethod="FTP" publishUrl="`+xmlAttributeEscape(`ftp://`+siteName+`.ftp.azurewebsites.windows.net/site/wwwroot`)+`" ftpPassiveMode="True" userName="`+xmlAttributeEscape(userName)+`" userPWD="`+xmlAttributeEscape(password)+`" destinationAppUrl="`+xmlAttributeEscape(`https://`+siteName+`.azurewebsites.net`)+`" SQLServerDBConnectionString="" mySQLDBConnectionString="" hostingProviderForumLink="" controlPanelLink="https://portal.azure.com" />`)
	}

	return `<?xml version="1.0" encoding="utf-8"?><publishData>` + strings.Join(profiles, "") + `</publishData>`
}

func publishingPassword(siteName string) string {
	return "cloudmock-" + strings.ToLower(siteName) + "-publishing-password"
}

func xmlAttributeEscape(value string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`"`, `&quot;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
		`'`, `&apos;`,
	)
	return replacer.Replace(value)
}

func resourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func functionKeyStateKey(subscriptionID, resourceGroup, siteName, functionName string) string {
	return resourceKey(subscriptionID, resourceGroup, siteName) + "/" + strings.ToLower(functionLeafName(functionName))
}

func defaultFunctionKeyValue(siteName, functionName string) string {
	return "cloudmock-" + strings.ToLower(siteName) + "-" + strings.ToLower(functionLeafName(functionName)) + "-default-key"
}

func defaultHostFunctionKeyValue(siteName string) string {
	return "cloudmock-" + strings.ToLower(siteName) + "-host-default-key"
}

func defaultHostMasterKeyValue(siteName string) string {
	return "cloudmock-" + strings.ToLower(siteName) + "-master-key"
}

func normalizeHostKeyType(keyType string) string {
	switch strings.ToLower(keyType) {
	case "functionkeys", "function_keys", "function-keys":
		return "functionkeys"
	case "systemkeys", "system_keys", "system-keys":
		return "systemkeys"
	case "masterkey", "master_key", "master-key":
		return "masterkey"
	default:
		return ""
	}
}

func hostSecretExists(state hostKeysState, keyType, keyName string) bool {
	switch keyType {
	case "functionkeys":
		_, ok := state.FunctionKeys[keyName]
		return ok
	case "systemkeys":
		_, ok := state.SystemKeys[keyName]
		return ok
	case "masterkey":
		return state.MasterKey != ""
	default:
		return false
	}
}

func upsertHostSecret(state hostKeysState, keyType, keyName, value string) hostKeysState {
	switch keyType {
	case "functionkeys":
		if state.FunctionKeys == nil {
			state.FunctionKeys = make(map[string]string)
		}
		state.FunctionKeys[keyName] = value
	case "systemkeys":
		if state.SystemKeys == nil {
			state.SystemKeys = make(map[string]string)
		}
		state.SystemKeys[keyName] = value
	case "masterkey":
		state.MasterKey = value
	}
	return state
}

func removeHostSecret(state hostKeysState, keyType, keyName string) hostKeysState {
	switch keyType {
	case "functionkeys":
		delete(state.FunctionKeys, keyName)
		if len(state.FunctionKeys) == 0 {
			state.FunctionKeys = nil
		}
	case "systemkeys":
		delete(state.SystemKeys, keyName)
		if len(state.SystemKeys) == 0 {
			state.SystemKeys = nil
		}
	case "masterkey":
		state.MasterKey = ""
	}
	return state
}

func hostKeysStateIsEmpty(state hostKeysState) bool {
	return len(state.FunctionKeys) == 0 && len(state.SystemKeys) == 0 && state.MasterKey == ""
}

func resourcePrefix(subscriptionID, resourceGroup string) string {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}
	return prefix
}

func splitPath(escapedPath string) []string {
	trimmed := strings.Trim(escapedPath, "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		parts = append(parts, decoded)
	}
	return parts
}

func stringifyTags(tags map[string]any) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringSlice(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneConnectionStringSet(in map[string]map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(in))
	for name, value := range in {
		out[name] = cloneMap(value)
	}
	return out
}

func appSettingsFromBody(body []byte) (map[string]string, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return nil, err
		}
	}
	properties := mapValue(input["properties"])
	if properties == nil {
		properties = input
	}
	return appSettingsFromMap(properties), nil
}

func appSettingsFromSiteConfig(siteConfig map[string]any) (map[string]string, bool) {
	if siteConfig == nil {
		return nil, false
	}
	raw, ok := siteConfig["appSettings"]
	if !ok {
		return nil, false
	}
	return appSettingsFromRaw(raw), true
}

func appSettingsFromRaw(raw any) map[string]string {
	switch typed := raw.(type) {
	case []any:
		out := make(map[string]string, len(typed))
		for _, item := range typed {
			setting, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := stringValue(setting["name"])
			if name == "" {
				continue
			}
			out[name] = fmt.Sprint(setting["value"])
		}
		return out
	case map[string]any:
		return appSettingsFromMap(typed)
	case nil:
		return make(map[string]string)
	default:
		return make(map[string]string)
	}
}

func appSettingsFromMap(in map[string]any) map[string]string {
	out := make(map[string]string, len(in))
	for name, value := range in {
		out[name] = fmt.Sprint(value)
	}
	return out
}

func appSettingsArray(settings map[string]string) []map[string]any {
	names := make([]string, 0, len(settings))
	for name := range settings {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]any{"name": name, "value": settings[name]})
	}
	return out
}

func connectionStringsFromBody(body []byte) (map[string]map[string]any, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return nil, err
		}
	}
	properties := mapValue(input["properties"])
	if properties == nil {
		properties = input
	}
	return connectionStringsFromMap(properties), nil
}

func connectionStringsFromSiteConfig(siteConfig map[string]any) (map[string]map[string]any, bool) {
	if siteConfig == nil {
		return nil, false
	}
	raw, ok := siteConfig["connectionStrings"]
	if !ok {
		return nil, false
	}
	return connectionStringsFromRaw(raw), true
}

func connectionStringsFromRaw(raw any) map[string]map[string]any {
	switch typed := raw.(type) {
	case []any:
		out := make(map[string]map[string]any, len(typed))
		for _, item := range typed {
			connection, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := stringValue(connection["name"])
			if name == "" {
				continue
			}
			value := stringValue(connection["value"])
			if value == "" {
				value = stringValue(connection["connectionString"])
			}
			props := map[string]any{"value": value}
			if connectionType := stringValue(connection["type"]); connectionType != "" {
				props["type"] = connectionType
			}
			if slotSetting, ok := connection["slotSetting"]; ok {
				props["slotSetting"] = slotSetting
			}
			out[name] = props
		}
		return out
	case map[string]any:
		return connectionStringsFromMap(typed)
	case nil:
		return make(map[string]map[string]any)
	default:
		return make(map[string]map[string]any)
	}
}

func connectionStringsFromMap(in map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(in))
	for name, value := range in {
		if nested := mapValue(value); nested != nil {
			out[name] = cloneMap(nested)
			continue
		}
		out[name] = map[string]any{"value": fmt.Sprint(value), "type": "Custom"}
	}
	return out
}

func connectionStringArray(connections map[string]map[string]any) []map[string]any {
	names := make([]string, 0, len(connections))
	for name := range connections {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		entry := cloneMap(connections[name])
		entry["name"] = name
		out = append(out, entry)
	}
	return out
}

func functionsLocalPath(req *http.Request) (string, []string, bool) {
	parts := splitPath(req.URL.EscapedPath())
	if len(parts) < 2 {
		return "", nil, false
	}
	first := parts[0]
	if !strings.HasSuffix(strings.ToLower(first), "-functions") {
		return "", nil, false
	}
	accountName := first[:len(first)-len("-functions")]
	if accountName == "" {
		return "", nil, false
	}
	return accountName, parts[1:], true
}

func localFunctionAppKey(accountName, appName string) string {
	key := strings.ToLower(accountName) + "/"
	if appName != "" {
		key += strings.ToLower(appName)
	}
	return key
}

func localFunctionPrefix(accountName, appName string) string {
	return localFunctionAppKey(accountName, appName) + "/"
}

func localFunctionKey(accountName, appName, functionName string) string {
	return localFunctionPrefix(accountName, appName) + strings.ToLower(functionName)
}

func localFunctionInvokeURL(accountName, appName, functionName string) string {
	return "http://localhost:4577/" + accountName + "-functions/api/" + appName + "/" + functionName
}

func runtimeFromLinuxFxVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "|")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(parts[0])) {
	case "node":
		return "node"
	case "python":
		return "python"
	case "java":
		return "java"
	case "dotnet", "dotnet-isolated":
		return "dotnet"
	default:
		return ""
	}
}

func validateLinuxFxVersion(siteConfig map[string]any) error {
	if siteConfig == nil {
		return nil
	}
	linuxFxVersion := strings.TrimSpace(stringValue(siteConfig["linuxFxVersion"]))
	if linuxFxVersion == "" {
		return nil
	}
	if runtimeFromLinuxFxVersion(linuxFxVersion) == "" {
		return fmt.Errorf("linuxFxVersion must use '<runtime>|<version>' format for a supported Azure Functions runtime")
	}
	return nil
}

func isSiteAction(value string) bool {
	switch strings.ToLower(value) {
	case "start", "stop", "restart", "syncfunctiontriggers":
		return true
	default:
		return false
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var out int
		if _, err := fmt.Sscanf(typed, "%d", &out); err == nil {
			return out
		}
	}
	return 0
}

func (s *AppService) syncLocalFunctionAppLocked(accountName, appName string, siteConfig map[string]any) {
	linuxFxVersion := strings.TrimSpace(stringValue(siteConfig["linuxFxVersion"]))
	if linuxFxVersion == "" {
		return
	}
	runtime := runtimeFromLinuxFxVersion(linuxFxVersion)
	if runtime == "" {
		return
	}
	key := localFunctionAppKey(accountName, appName)
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	if existing, ok := s.localFunctionApps[key]; ok {
		createdAt = existing.CreatedAt
	}
	s.localFunctionApps[key] = LocalFunctionApp{
		Name:           appName,
		Runtime:        runtime,
		LinuxFxVersion: linuxFxVersion,
		Status:         "Running",
		CreatedAt:      createdAt,
	}
}

func (s *AppService) syncLocalFunctionAppStatusLocked(accountName, appName, status string) {
	key := localFunctionAppKey(accountName, appName)
	app, ok := s.localFunctionApps[key]
	if !ok {
		return
	}
	app.Status = status
	s.localFunctionApps[key] = app
}

func firstNonBlank(first, second string) string {
	if strings.TrimSpace(first) != "" {
		return first
	}
	return second
}

func singleValueQuery(values url.Values) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			out[key] = value[0]
		}
	}
	return out
}

func noContentResponse() *service.Response {
	return &service.Response{StatusCode: http.StatusNoContent, RawContentType: "application/json"}
}
