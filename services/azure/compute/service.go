package compute

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const computeAPIVersion = "2025-11-01"

// ComputeService implements first-slice Azure Compute control-plane APIs.
type ComputeService struct {
	mu    sync.RWMutex
	vm    map[string]VirtualMachine
	disk  map[string]Disk
	power map[string]string
}

type runCommandInputParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func New() *ComputeService {
	return &ComputeService{
		vm:    make(map[string]VirtualMachine),
		disk:  make(map[string]Disk),
		power: make(map[string]string),
	}
}

func (s *ComputeService) Name() string { return "Microsoft.Compute" }

func (s *ComputeService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateVirtualMachine", Method: http.MethodPut, IAMAction: "azure:Microsoft.Compute/virtualMachines/write"},
		{Name: "GetVirtualMachine", Method: http.MethodGet, IAMAction: "azure:Microsoft.Compute/virtualMachines/read"},
		{Name: "ListVirtualMachines", Method: http.MethodGet, IAMAction: "azure:Microsoft.Compute/virtualMachines/read"},
		{Name: "UpdateVirtualMachine", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Compute/virtualMachines/write"},
		{Name: "DeleteVirtualMachine", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Compute/virtualMachines/delete"},
		{Name: "GetVirtualMachineInstanceView", Method: http.MethodGet, IAMAction: "azure:Microsoft.Compute/virtualMachines/instanceView/read"},
		{Name: "StartVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/start/action"},
		{Name: "PowerOffVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/powerOff/action"},
		{Name: "DeallocateVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/deallocate/action"},
		{Name: "RestartVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/restart/action"},
		{Name: "RedeployVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/redeploy/action"},
		{Name: "ReapplyVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/reapply/action"},
		{Name: "RunCommandVirtualMachine", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/virtualMachines/runCommand/action"},
		{Name: "CreateOrUpdateDisk", Method: http.MethodPut, IAMAction: "azure:Microsoft.Compute/disks/write"},
		{Name: "GetDisk", Method: http.MethodGet, IAMAction: "azure:Microsoft.Compute/disks/read"},
		{Name: "ListDisks", Method: http.MethodGet, IAMAction: "azure:Microsoft.Compute/disks/read"},
		{Name: "UpdateDisk", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Compute/disks/write"},
		{Name: "DeleteDisk", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Compute/disks/delete"},
		{Name: "GrantAccessDisk", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/disks/beginGetAccess/action"},
		{Name: "RevokeAccessDisk", Method: http.MethodPost, IAMAction: "azure:Microsoft.Compute/disks/endGetAccess/action"},
	}
}

func (s *ComputeService) HealthCheck() error { return nil }

func (s *ComputeService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Compute/virtualMachines", APIVersion: computeAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Compute/disks", APIVersion: "2025-01-02"},
	}
}

func (s *ComputeService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.Compute/virtualMachines") ||
		strings.EqualFold(resourceType, "Microsoft.Compute/disks")
}

func (s *ComputeService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("compute template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"identity":   resource["identity"],
		"plan":       resource["plan"],
		"tags":       resource["tags"],
		"zones":      resource["zones"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	var resp *service.Response
	switch resourceType := stringValue(resource["type"]); {
	case strings.EqualFold(resourceType, "Microsoft.Compute/virtualMachines"):
		resp, err = s.createOrUpdateVirtualMachine(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Compute/disks"):
		resp, err = s.createOrUpdateDisk(subscriptionID, resourceGroup, name, data)
	default:
		err = fmt.Errorf("unsupported template resource type %q", resourceType)
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

func (s *ComputeService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if route, ok := parseDiskRoute(ctx.RawRequest.URL.EscapedPath()); ok {
		return s.handleDiskRequest(ctx, route)
	}

	route, ok := parseVMRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The virtual machine route is not implemented.")
	}

	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listVirtualMachines(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.Operation != "" {
		if strings.EqualFold(route.Operation, "instanceView") {
			if ctx.RawRequest.Method != http.MethodGet {
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
			return s.getVirtualMachineInstanceView(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if strings.EqualFold(route.Operation, "runCommand") {
			return s.runVirtualMachineCommand(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		}
		return s.handleVirtualMachineOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.Operation)
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateVirtualMachine(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getVirtualMachine(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.RawRequest.URL.Query())
	case http.MethodPatch:
		return s.updateVirtualMachine(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodDelete:
		return s.deleteVirtualMachine(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ComputeService) handleDiskRequest(ctx *service.RequestContext, route diskRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listDisks(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.Operation != "" {
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		switch {
		case strings.EqualFold(route.Operation, "beginGetAccess"):
			return s.grantDiskAccess(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
		case strings.EqualFold(route.Operation, "endGetAccess"):
			return s.revokeDiskAccess(route.SubscriptionID, route.ResourceGroup, route.Name)
		default:
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", fmt.Sprintf("The disk operation %q is not implemented.", route.Operation))
		}
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateDisk(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getDisk(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodPatch:
		return s.updateDisk(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodDelete:
		return s.deleteDisk(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ComputeService) createOrUpdateVirtualMachine(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Identity   map[string]any `json:"identity"`
		Plan       map[string]any `json:"plan"`
		Tags       map[string]any `json:"tags"`
		Zones      []string       `json:"zones"`
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

	vm := VirtualMachine{
		ID:         virtualMachineID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Compute/virtualMachines",
		Location:   input.Location,
		Identity:   input.Identity,
		Plan:       input.Plan,
		Tags:       stringifyTags(input.Tags),
		Zones:      input.Zones,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := virtualMachineKey(subscriptionID, resourceGroup, name)
	_, existed := s.vm[key]
	s.vm[key] = vm
	if _, ok := s.power[key]; !ok {
		s.power[key] = "running"
	}
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, vm)
}

func (s *ComputeService) getVirtualMachine(subscriptionID, resourceGroup, name string, query url.Values) (*service.Response, error) {
	s.mu.RLock()
	vm, ok := s.vm[virtualMachineKey(subscriptionID, resourceGroup, name)]
	powerState := s.power[virtualMachineKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}
	if powerState == "" {
		powerState = "running"
	}
	if strings.EqualFold(query.Get("$expand"), "instanceView") {
		vm.Properties = cloneProperties(vm.Properties)
		vm.Properties["instanceView"] = instanceViewForPowerState(powerState)
	}
	return azurearm.JSONResponse(http.StatusOK, vm)
}

func (s *ComputeService) getVirtualMachineInstanceView(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.vm[virtualMachineKey(subscriptionID, resourceGroup, name)]
	powerState := s.power[virtualMachineKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}
	if powerState == "" {
		powerState = "running"
	}
	return azurearm.JSONResponse(http.StatusOK, instanceViewForPowerState(powerState))
}

func (s *ComputeService) updateVirtualMachine(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := virtualMachineKey(subscriptionID, resourceGroup, name)
	vm, ok := s.vm[key]
	if ok {
		if tags, present := input["tags"]; present {
			vm.Tags = stringifyTags(mapValue(tags))
		}
		if identity, present := input["identity"].(map[string]any); present {
			vm.Identity = identity
		}
		if plan, present := input["plan"].(map[string]any); present {
			vm.Plan = plan
		}
		if properties, present := input["properties"]; present {
			merged := cloneProperties(vm.Properties)
			for propKey, propValue := range mapValue(properties) {
				merged[propKey] = propValue
			}
			merged["provisioningState"] = "Succeeded"
			vm.Properties = merged
		}
		s.vm[key] = vm
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, vm)
}

func (s *ComputeService) listVirtualMachines(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]VirtualMachine, 0)
	for key, vm := range s.vm {
		if strings.HasPrefix(key, prefix) {
			values = append(values, vm)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ComputeService) deleteVirtualMachine(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := virtualMachineKey(subscriptionID, resourceGroup, name)
	if _, ok := s.vm[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}
	delete(s.vm, key)
	delete(s.power, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ComputeService) handleVirtualMachineOperation(subscriptionID, resourceGroup, name, operation string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := virtualMachineKey(subscriptionID, resourceGroup, name)
	if _, ok := s.vm[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}
	switch {
	case strings.EqualFold(operation, "start"):
		s.power[key] = "running"
	case strings.EqualFold(operation, "powerOff"):
		s.power[key] = "stopped"
	case strings.EqualFold(operation, "deallocate"):
		s.power[key] = "deallocated"
	case strings.EqualFold(operation, "restart"), strings.EqualFold(operation, "redeploy"), strings.EqualFold(operation, "reapply"):
		s.power[key] = "running"
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", fmt.Sprintf("The virtual machine operation %q is not implemented.", operation))
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Azure-AsyncOperation": virtualMachineID(subscriptionID, resourceGroup, name) + "/operationStatuses/" + strings.ToLower(operation),
			"Location":             virtualMachineID(subscriptionID, resourceGroup, name) + "/operationResults/" + strings.ToLower(operation),
			"Retry-After":          "0",
		},
	}, nil
}

func (s *ComputeService) runVirtualMachineCommand(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		CommandID  string                     `json:"commandId"`
		Script     []string                   `json:"script"`
		Parameters []runCommandInputParameter `json:"parameters"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if strings.TrimSpace(input.CommandID) == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingRunCommandId", "Run command request body must include commandId.")
	}

	s.mu.RLock()
	_, ok := s.vm[virtualMachineKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual machine %q could not be found.", name))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"code":          "ComponentStatus/StdOut/succeeded",
				"level":         "Info",
				"displayStatus": "Provisioning succeeded",
				"message":       runCommandStdoutMessage(name, input.CommandID, input.Script, input.Parameters),
			},
			{
				"code":          "ComponentStatus/StdErr/succeeded",
				"level":         "Info",
				"displayStatus": "Provisioning succeeded",
				"message":       "",
			},
		},
	})
}

func (s *ComputeService) createOrUpdateDisk(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
		Zones      []string       `json:"zones"`
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
	if input.SKU == nil {
		input.SKU = map[string]any{"name": "Standard_LRS"}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["diskState"]; !ok {
		input.Properties["diskState"] = "Unattached"
	}

	disk := Disk{
		ID:         diskID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Compute/disks",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Zones:      input.Zones,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := diskKey(subscriptionID, resourceGroup, name)
	_, existed := s.disk[key]
	s.disk[key] = disk
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, disk)
}

func (s *ComputeService) getDisk(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	disk, ok := s.disk[diskKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Disk %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, disk)
}

func (s *ComputeService) updateDisk(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := diskKey(subscriptionID, resourceGroup, name)
	disk, ok := s.disk[key]
	if ok {
		if tags, present := input["tags"]; present {
			disk.Tags = stringifyTags(mapValue(tags))
		}
		if sku, present := input["sku"].(map[string]any); present {
			disk.SKU = sku
		}
		if properties, present := input["properties"]; present {
			merged := cloneProperties(disk.Properties)
			for propKey, propValue := range mapValue(properties) {
				merged[propKey] = propValue
			}
			merged["provisioningState"] = "Succeeded"
			if _, ok := merged["diskState"]; !ok {
				merged["diskState"] = "Unattached"
			}
			disk.Properties = merged
		}
		s.disk[key] = disk
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Disk %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, disk)
}

func (s *ComputeService) listDisks(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]Disk, 0)
	for key, disk := range s.disk {
		if strings.HasPrefix(key, prefix) {
			values = append(values, disk)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ComputeService) deleteDisk(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := diskKey(subscriptionID, resourceGroup, name)
	if _, ok := s.disk[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Disk %q could not be found.", name))
	}
	delete(s.disk, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ComputeService) grantDiskAccess(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Access                   string `json:"access"`
		DurationInSeconds        int    `json:"durationInSeconds"`
		FileFormat               string `json:"fileFormat"`
		GetSecureVMGuestStateSAS bool   `json:"getSecureVMGuestStateSAS"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	permission, ok := diskAccessPermission(input.Access)
	if strings.TrimSpace(input.Access) == "" || !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidGrantAccessRequest", "Grant access request body must include access with one of None, Read, or Write.")
	}
	if input.DurationInSeconds <= 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidGrantAccessRequest", "Grant access request body must include a positive durationInSeconds.")
	}

	s.mu.RLock()
	_, exists := s.disk[diskKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Disk %q could not be found.", name))
	}

	response := map[string]any{
		"accessSAS": diskAccessSAS(subscriptionID, resourceGroup, name, "", permission, input.DurationInSeconds),
	}
	if input.GetSecureVMGuestStateSAS {
		response["securityDataAccessSAS"] = diskAccessSAS(subscriptionID, resourceGroup, name, "_vmgs", permission, input.DurationInSeconds)
		response["securityMetadataAccessSAS"] = diskAccessSAS(subscriptionID, resourceGroup, name, "_vmmd", permission, input.DurationInSeconds)
	}
	return azurearm.JSONResponse(http.StatusOK, response)
}

func (s *ComputeService) revokeDiskAccess(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, exists := s.disk[diskKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Disk %q could not be found.", name))
	}
	return &service.Response{StatusCode: http.StatusOK}, nil
}

type vmRoute struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Operation      string
}

type diskRoute struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Operation      string
}

func parseVMRoute(escapedPath string) (vmRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Compute") &&
		strings.EqualFold(parts[4], "virtualMachines") {
		return vmRoute{SubscriptionID: parts[1]}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Compute") ||
		!strings.EqualFold(parts[6], "virtualMachines") {
		return vmRoute{}, false
	}
	route := vmRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		route.Operation = parts[8]
		return route, true
	default:
		return vmRoute{}, false
	}
}

func parseDiskRoute(escapedPath string) (diskRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Compute") &&
		strings.EqualFold(parts[4], "disks") {
		return diskRoute{SubscriptionID: parts[1]}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Compute") ||
		!strings.EqualFold(parts[6], "disks") {
		return diskRoute{}, false
	}
	route := diskRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		route.Operation = parts[8]
		return route, true
	default:
		return diskRoute{}, false
	}
}

func virtualMachineID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + name
}

func virtualMachineKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func diskID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/" + name
}

func diskKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func diskAccessPermission(access string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(access)) {
	case "none":
		return "", true
	case "read":
		return "r", true
	case "write":
		return "w", true
	default:
		return "", false
	}
}

func diskAccessSAS(subscriptionID, resourceGroup, name, suffix, permission string, durationSeconds int) string {
	return fmt.Sprintf(
		"https://cloudmock.azure.local/disks/%s/%s/%s%s?sv=cloudmock&sr=b&sp=%s&se=%d&sig=cloudmock",
		url.PathEscape(subscriptionID),
		url.PathEscape(resourceGroup),
		url.PathEscape(name),
		suffix,
		url.QueryEscape(permission),
		durationSeconds,
	)
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

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return nil
	}
}

func runCommandStdoutMessage(vmName, commandID string, script []string, parameters []runCommandInputParameter) string {
	var builder strings.Builder
	builder.WriteString(commandID)
	builder.WriteString(" executed on ")
	builder.WriteString(vmName)
	if len(script) > 0 {
		builder.WriteString("\nscript:")
		for _, line := range script {
			builder.WriteString("\n")
			builder.WriteString(line)
		}
	}
	if len(parameters) > 0 {
		builder.WriteString("\nparameters:")
		for _, parameter := range parameters {
			builder.WriteString("\n")
			builder.WriteString(parameter.Name)
			builder.WriteString("=")
			builder.WriteString(parameter.Value)
		}
	}
	return builder.String()
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func cloneProperties(properties map[string]any) map[string]any {
	out := make(map[string]any, len(properties)+1)
	for key, value := range properties {
		out[key] = value
	}
	return out
}

func instanceViewForPowerState(powerState string) VirtualMachineInstanceView {
	powerCode := "PowerState/running"
	powerDisplay := "VM running"
	switch powerState {
	case "stopped":
		powerCode = "PowerState/stopped"
		powerDisplay = "VM stopped"
	case "deallocated":
		powerCode = "PowerState/deallocated"
		powerDisplay = "VM deallocated"
	}
	return VirtualMachineInstanceView{
		Statuses: []InstanceViewStatus{
			{Code: "ProvisioningState/succeeded", Level: "Info", DisplayStatus: "Provisioning succeeded"},
			{Code: powerCode, Level: "Info", DisplayStatus: powerDisplay},
		},
	}
}
