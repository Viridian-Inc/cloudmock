package containerservice

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var containerServiceManagedClusterAPIVersions = []string{"2026-02-01", "2026-03-01", "2026-04-01", "2024-04-01"}
var containerServiceLocationAPIVersions = []string{"2026-03-01"}

const (
	aksServerApplicationID = "6dae42f8-4368-4678-94ff-3960e28e3630"
	aksClientApplicationID = "80faf920-1908-4b52-b5ef-a8e7bedfc67a"
)

// ContainerService implements mocked-mode Azure Kubernetes Service ARM APIs.
type ContainerService struct {
	mu            sync.RWMutex
	clusters      map[string]ManagedCluster
	commands      map[string]CommandResult
	nextCommandID int
}

func New() *ContainerService {
	return &ContainerService{clusters: make(map[string]ManagedCluster), commands: make(map[string]CommandResult)}
}

func (s *ContainerService) Name() string { return "Microsoft.ContainerService" }

func (s *ContainerService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateManagedCluster", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerService/managedClusters/write"},
		{Name: "GetManagedCluster", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/read"},
		{Name: "ListManagedClusters", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/read"},
		{Name: "ListKubernetesVersions", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/locations/kubernetesVersions/read"},
		{Name: "UpdateManagedClusterTags", Method: http.MethodPatch, IAMAction: "azure:Microsoft.ContainerService/managedClusters/write"},
		{Name: "DeleteManagedCluster", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerService/managedClusters/delete"},
		{Name: "ListClusterAdminCredentials", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/listClusterAdminCredential/action"},
		{Name: "ListClusterUserCredentials", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/listClusterUserCredential/action"},
		{Name: "ListClusterMonitoringUserCredentials", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/listClusterMonitoringUserCredential/action"},
		{Name: "StartManagedCluster", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/start/action"},
		{Name: "StopManagedCluster", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/stop/action"},
		{Name: "RotateClusterCertificates", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/rotateClusterCertificates/action"},
		{Name: "RotateServiceAccountSigningKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/rotateServiceAccountSigningKeys/action"},
		{Name: "RunCommand", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/runCommand/action"},
		{Name: "GetCommandResult", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/commandResults/read"},
		{Name: "GetManagedClusterUpgradeProfile", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/upgradeProfiles/read"},
		{Name: "CreateOrUpdateAgentPool", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/write"},
		{Name: "GetAgentPool", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/read"},
		{Name: "ListAgentPools", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/read"},
		{Name: "DeleteAgentPool", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/delete"},
		{Name: "UpgradeAgentPoolNodeImageVersion", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/upgradeNodeImageVersion/action"},
		{Name: "AbortAgentPoolLatestOperation", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerService/managedClusters/agentPools/abort/action"},
	}
}

func (s *ContainerService) HealthCheck() error { return nil }

func (s *ContainerService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(containerServiceManagedClusterAPIVersions)+len(containerServiceLocationAPIVersions))
	for _, apiVersion := range containerServiceManagedClusterAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerService/managedClusters",
			APIVersion: apiVersion,
		})
	}
	for _, apiVersion := range containerServiceLocationAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerService/locations",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *ContainerService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.ContainerService/managedClusters")
}

func (s *ContainerService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Container Service template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Container Service template resource is missing name")
	}
	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateManagedCluster(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContainerService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if route, ok := parseLocationRoute(ctx.RawRequest.URL.EscapedPath()); ok {
		if route.Operation == "kubernetesVersions" && ctx.RawRequest.Method == http.MethodGet {
			return s.listKubernetesVersions(route.Location)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "managedClusters") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Service route is not implemented.")
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listManagedClusters(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.ChildType != "" {
		return s.handleChildRequest(ctx, route)
	}
	if route.ActionName != "" {
		return s.handleActionRequest(ctx, route)
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateManagedCluster(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getManagedCluster(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodPatch:
		return s.updateManagedClusterTags(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodDelete:
		return s.deleteManagedCluster(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerService) handleActionRequest(ctx *service.RequestContext, route containerServiceRoute) (*service.Response, error) {
	switch {
	case strings.EqualFold(route.ActionName, "listClusterAdminCredential"), strings.EqualFold(route.ActionName, "listClusterUserCredential"), strings.EqualFold(route.ActionName, "listClusterMonitoringUserCredential"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		name := "clusterAdmin"
		if strings.EqualFold(route.ActionName, "listClusterUserCredential") {
			name = "clusterUser"
		}
		if strings.EqualFold(route.ActionName, "listClusterMonitoringUserCredential") {
			name = "clusterMonitoringUser"
		}
		format := ""
		if strings.EqualFold(route.ActionName, "listClusterUserCredential") {
			format = ctx.RawRequest.URL.Query().Get("format")
		}
		return s.listClusterCredentials(route.SubscriptionID, route.ResourceGroup, route.Name, name, ctx.RawRequest.URL.Query().Get("server-fqdn"), format)
	case strings.EqualFold(route.ActionName, "start"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.setManagedClusterPowerState(route.SubscriptionID, route.ResourceGroup, route.Name, "Running", route.ActionName, ctx.RawRequest.URL.Query().Get("api-version"))
	case strings.EqualFold(route.ActionName, "stop"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.setManagedClusterPowerState(route.SubscriptionID, route.ResourceGroup, route.Name, "Stopped", route.ActionName, ctx.RawRequest.URL.Query().Get("api-version"))
	case strings.EqualFold(route.ActionName, "rotateClusterCertificates"), strings.EqualFold(route.ActionName, "rotateServiceAccountSigningKeys"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.acceptManagedClusterOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.ActionName, ctx.RawRequest.URL.Query().Get("api-version"))
	case strings.EqualFold(route.ActionName, "runCommand"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.runCommand(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Service action is not implemented.")
	}
}

func (s *ContainerService) handleChildRequest(ctx *service.RequestContext, route containerServiceRoute) (*service.Response, error) {
	if strings.EqualFold(route.ChildType, "commandResults") {
		if route.ChildName == "" || ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.getCommandResult(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	}
	if strings.EqualFold(route.ChildType, "upgradeProfiles") {
		if route.ChildName == "" || ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.getManagedClusterUpgradeProfile(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	}
	if !strings.EqualFold(route.ChildType, "agentPools") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Service child route is not implemented.")
	}
	if route.GrandchildType != "" {
		if strings.EqualFold(route.GrandchildType, "upgradeProfiles") {
			if route.ChildName == "" || route.GrandchildName == "" || ctx.RawRequest.Method != http.MethodGet {
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
			return s.getAgentPoolUpgradeProfile(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandchildName)
		}
		if strings.EqualFold(route.GrandchildType, "upgradeNodeImageVersion") {
			if route.ChildName == "" || ctx.RawRequest.Method != http.MethodPost {
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
			return s.upgradeAgentPoolNodeImageVersion(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.RawRequest.URL.Query().Get("api-version"))
		}
		if strings.EqualFold(route.GrandchildType, "abort") {
			if route.ChildName == "" || ctx.RawRequest.Method != http.MethodPost {
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
			return s.abortAgentPoolLatestOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.RawRequest.URL.Query().Get("api-version"))
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Service child route is not implemented.")
	}
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listAgentPools(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateAgentPool(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getAgentPool(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteAgentPool(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerService) createOrUpdateManagedCluster(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties struct {
			KubernetesVersion string             `json:"kubernetesVersion"`
			DNSPrefix         string             `json:"dnsPrefix"`
			AgentPoolProfiles []AgentPoolProfile `json:"agentPoolProfiles"`
		} `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Properties.KubernetesVersion == "" {
		input.Properties.KubernetesVersion = "1.29"
	}
	if input.Properties.DNSPrefix == "" {
		input.Properties.DNSPrefix = name + "-dns"
	}
	pools := input.Properties.AgentPoolProfiles
	if len(pools) == 0 {
		pools = []AgentPoolProfile{defaultAgentPool()}
	}
	for i := range pools {
		normalizeAgentPool(&pools[i])
	}

	key := clusterKey(subscriptionID, resourceGroup, name)
	cluster := ManagedCluster{
		ID:                    clusterID(subscriptionID, resourceGroup, name),
		Name:                  name,
		Type:                  "Microsoft.ContainerService/managedClusters",
		Location:              input.Location,
		Tags:                  stringifyTags(input.Tags),
		KubernetesVersion:     input.Properties.KubernetesVersion,
		DNSPrefix:             input.Properties.DNSPrefix,
		FQDN:                  input.Properties.DNSPrefix + ".hcp." + input.Location + ".azmk8s.io",
		Endpoint:              "https://localhost:6443",
		ProvisioningState:     "Succeeded",
		PowerState:            "Running",
		AgentPoolProfiles:     pools,
		SubscriptionID:        subscriptionID,
		ResourceGroup:         resourceGroup,
		NodeResourceGroupName: "MC_" + resourceGroup + "_" + name + "_" + input.Location,
	}
	cluster.Kubeconfig = mockKubeconfig(cluster)

	s.mu.Lock()
	_, existed := s.clusters[key]
	s.clusters[key] = cluster
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, managedClusterResponse(cluster))
}

func (s *ContainerService) getManagedCluster(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, managedClusterResponse(cluster))
}

func (s *ContainerService) updateManagedClusterTags(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Tags map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := clusterKey(subscriptionID, resourceGroup, name)
	cluster, ok := s.clusters[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	cluster.Tags = stringifyTags(input.Tags)
	s.clusters[key] = cluster
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, managedClusterResponse(cluster))
}

func (s *ContainerService) deleteManagedCluster(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := clusterKey(subscriptionID, resourceGroup, name)
	if _, ok := s.clusters[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	delete(s.clusters, key)
	return &service.Response{StatusCode: http.StatusAccepted, RawContentType: "application/json"}, nil
}

func (s *ContainerService) listManagedClusters(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}
	values := make([]map[string]any, 0)
	s.mu.RLock()
	for key, cluster := range s.clusters {
		if strings.HasPrefix(key, prefix) {
			values = append(values, managedClusterResponse(cluster))
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerService) listKubernetesVersions(location string) (*service.Response, error) {
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"values": []map[string]any{
			kubernetesVersionProfile("1.29", true, false, map[string][]string{
				"1.29.7": {"1.30.0", "1.31.0"},
			}),
			kubernetesVersionProfile("1.30", false, false, map[string][]string{
				"1.30.0": {"1.31.0"},
			}),
			kubernetesVersionProfile("1.31", false, true, map[string][]string{
				"1.31.0": {},
			}),
		},
	})
}

func (s *ContainerService) listClusterCredentials(subscriptionID, resourceGroup, name, credentialName, serverFQDN, format string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"kubeconfigs": []map[string]string{
			{"name": credentialName, "value": mockKubeconfigForCredential(cluster, credentialName, serverFQDN, format)},
		},
	})
}

func (s *ContainerService) setManagedClusterPowerState(subscriptionID, resourceGroup, name, state, actionName, apiVersion string) (*service.Response, error) {
	s.mu.Lock()
	key := clusterKey(subscriptionID, resourceGroup, name)
	cluster, ok := s.clusters[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	cluster.PowerState = state
	s.clusters[key] = cluster
	s.mu.Unlock()

	return &service.Response{
		StatusCode:     http.StatusAccepted,
		RawContentType: "application/json",
		Headers: map[string]string{
			"Location":    managedClusterOperationLocation(cluster, actionName, apiVersion),
			"Retry-After": "0",
		},
	}, nil
}

func (s *ContainerService) acceptManagedClusterOperation(subscriptionID, resourceGroup, name, actionName, apiVersion string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	return &service.Response{
		StatusCode:     http.StatusAccepted,
		RawContentType: "application/json",
		Headers: map[string]string{
			"Location":    managedClusterOperationLocation(cluster, actionName, apiVersion),
			"Retry-After": "0",
		},
	}, nil
}

func (s *ContainerService) runCommand(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Command string `json:"command"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if strings.TrimSpace(input.Command) == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The run command request must include a command.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	s.nextCommandID++
	commandID := fmt.Sprintf("cloudmock-command-%d", s.nextCommandID)
	result := CommandResult{
		ID:                commandID,
		SubscriptionID:    subscriptionID,
		ResourceGroup:     resourceGroup,
		ClusterName:       name,
		Command:           input.Command,
		ExitCode:          0,
		Logs:              fmt.Sprintf("cloudmock executed %q on managed cluster %s", input.Command, cluster.Name),
		ProvisioningState: "succeeded",
		StartedAt:         "2026-03-01T00:00:00Z",
		FinishedAt:        "2026-03-01T00:00:01Z",
	}
	s.commands[commandResultKey(subscriptionID, resourceGroup, name, commandID)] = result
	return azurearm.JSONResponse(http.StatusOK, commandResultResponse(result))
}

func (s *ContainerService) getCommandResult(subscriptionID, resourceGroup, name, commandID string) (*service.Response, error) {
	s.mu.RLock()
	_, clusterOK := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	result, resultOK := s.commands[commandResultKey(subscriptionID, resourceGroup, name, commandID)]
	s.mu.RUnlock()
	if !clusterOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	if !resultOK {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Command result %q could not be found.", commandID))
	}
	return azurearm.JSONResponse(http.StatusOK, commandResultResponse(result))
}

func (s *ContainerService) getManagedClusterUpgradeProfile(subscriptionID, resourceGroup, name, profileName string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", name))
	}
	if !strings.EqualFold(profileName, "default") {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster upgrade profile %q could not be found.", profileName))
	}
	return azurearm.JSONResponse(http.StatusOK, managedClusterUpgradeProfileResponse(cluster))
}

func (s *ContainerService) getAgentPoolUpgradeProfile(subscriptionID, resourceGroup, clusterName, poolName, profileName string) (*service.Response, error) {
	if !strings.EqualFold(profileName, "default") {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool upgrade profile %q could not be found.", profileName))
	}
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, clusterName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	for _, pool := range cluster.AgentPoolProfiles {
		if strings.EqualFold(pool.Name, poolName) {
			return azurearm.JSONResponse(http.StatusOK, agentPoolUpgradeProfileResponse(cluster, pool))
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool %q could not be found.", poolName))
}

func (s *ContainerService) listAgentPools(subscriptionID, resourceGroup, clusterName string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, clusterName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	values := make([]map[string]any, 0, len(cluster.AgentPoolProfiles))
	for _, pool := range cluster.AgentPoolProfiles {
		values = append(values, agentPoolResponse(subscriptionID, resourceGroup, clusterName, pool))
	}
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerService) getAgentPool(subscriptionID, resourceGroup, clusterName, poolName string) (*service.Response, error) {
	s.mu.RLock()
	cluster, ok := s.clusters[clusterKey(subscriptionID, resourceGroup, clusterName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	for _, pool := range cluster.AgentPoolProfiles {
		if strings.EqualFold(pool.Name, poolName) {
			return azurearm.JSONResponse(http.StatusOK, agentPoolResponse(subscriptionID, resourceGroup, clusterName, pool))
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool %q could not be found.", poolName))
}

func (s *ContainerService) createOrUpdateAgentPool(subscriptionID, resourceGroup, clusterName, poolName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties AgentPoolProfile `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := clusterKey(subscriptionID, resourceGroup, clusterName)
	cluster, ok := s.clusters[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	pool := input.Properties
	pool.Name = poolName
	created := true
	for i := range cluster.AgentPoolProfiles {
		if strings.EqualFold(cluster.AgentPoolProfiles[i].Name, poolName) {
			created = false
			mergeAgentPool(&cluster.AgentPoolProfiles[i], pool)
			pool = cluster.AgentPoolProfiles[i]
			break
		}
	}
	if created {
		normalizeAgentPool(&pool)
		cluster.AgentPoolProfiles = append(cluster.AgentPoolProfiles, pool)
	}
	s.clusters[key] = cluster
	s.mu.Unlock()

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, agentPoolResponse(subscriptionID, resourceGroup, clusterName, pool))
}

func (s *ContainerService) deleteAgentPool(subscriptionID, resourceGroup, clusterName, poolName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := clusterKey(subscriptionID, resourceGroup, clusterName)
	cluster, ok := s.clusters[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	for i, pool := range cluster.AgentPoolProfiles {
		if strings.EqualFold(pool.Name, poolName) {
			cluster.AgentPoolProfiles = append(cluster.AgentPoolProfiles[:i], cluster.AgentPoolProfiles[i+1:]...)
			s.clusters[key] = cluster
			return &service.Response{StatusCode: http.StatusAccepted, RawContentType: "application/json"}, nil
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool %q could not be found.", poolName))
}

func (s *ContainerService) upgradeAgentPoolNodeImageVersion(subscriptionID, resourceGroup, clusterName, poolName, apiVersion string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clusterKey(subscriptionID, resourceGroup, clusterName)
	cluster, ok := s.clusters[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	for i, pool := range cluster.AgentPoolProfiles {
		if strings.EqualFold(pool.Name, poolName) {
			pool.NodeImageVersion = latestNodeImageVersion(pool.OSType)
			pool.ProvisioningState = "UpgradingNodeImageVersion"
			cluster.AgentPoolProfiles[i] = pool
			s.clusters[key] = cluster
			body := agentPoolResponse(subscriptionID, resourceGroup, clusterName, pool)
			resp, err := azurearm.JSONResponse(http.StatusAccepted, body)
			if err != nil {
				return nil, err
			}
			resp.Headers = map[string]string{
				"Azure-AsyncOperation": agentPoolOperationLocation(cluster, pool.Name, "upgradeNodeImageVersion", apiVersion),
				"Location":             agentPoolOperationLocation(cluster, pool.Name, "upgradeNodeImageVersion", apiVersion),
				"Retry-After":          "0",
			}
			return resp, nil
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool %q could not be found.", poolName))
}

func (s *ContainerService) abortAgentPoolLatestOperation(subscriptionID, resourceGroup, clusterName, poolName, apiVersion string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := clusterKey(subscriptionID, resourceGroup, clusterName)
	cluster, ok := s.clusters[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed cluster %q could not be found.", clusterName))
	}
	for i, pool := range cluster.AgentPoolProfiles {
		if !strings.EqualFold(pool.Name, poolName) {
			continue
		}
		if !agentPoolOperationInProgress(pool.ProvisioningState) {
			return azurearm.ErrorResponse(http.StatusConflict, "OperationNotAllowed", fmt.Sprintf("No running operation could be aborted for agent pool %q.", poolName))
		}
		pool.ProvisioningState = "Canceled"
		cluster.AgentPoolProfiles[i] = pool
		s.clusters[key] = cluster
		return &service.Response{
			StatusCode:     http.StatusAccepted,
			RawContentType: "application/json",
			Headers: map[string]string{
				"Azure-AsyncOperation": agentPoolOperationStatusLocation(cluster, pool.Name, "abort", apiVersion),
				"Location":             agentPoolOperationResourceLocation(cluster, pool.Name, "abort", apiVersion),
				"Retry-After":          "0",
			},
		}, nil
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Agent pool %q could not be found.", poolName))
}

type containerServiceRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	Name           string
	ActionName     string
	ChildType      string
	ChildName      string
	GrandchildType string
	GrandchildName string
}

type containerServiceLocationRoute struct {
	SubscriptionID string
	Location       string
	Operation      string
}

func parseLocationRoute(escapedPath string) (containerServiceLocationRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) != 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "providers") ||
		!strings.EqualFold(parts[3], "Microsoft.ContainerService") ||
		!strings.EqualFold(parts[4], "locations") {
		return containerServiceLocationRoute{}, false
	}
	return containerServiceLocationRoute{
		SubscriptionID: parts[1],
		Location:       parts[5],
		Operation:      parts[6],
	}, true
}

func parseRoute(escapedPath string) (containerServiceRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ContainerService") &&
		strings.EqualFold(parts[4], "managedClusters") {
		return containerServiceRoute{SubscriptionID: parts[1], ResourceType: "managedClusters"}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ContainerService") {
		return containerServiceRoute{}, false
	}
	route := containerServiceRoute{
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
		if strings.EqualFold(parts[8], "agentPools") {
			route.ChildType = parts[8]
		} else {
			route.ActionName = parts[8]
		}
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
		route.GrandchildType = parts[10]
		return route, true
	case 12:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandchildType = parts[10]
		route.GrandchildName = parts[11]
		return route, true
	default:
		return containerServiceRoute{}, false
	}
}

func managedClusterResponse(cluster ManagedCluster) map[string]any {
	pools := make([]map[string]any, 0, len(cluster.AgentPoolProfiles))
	for _, pool := range cluster.AgentPoolProfiles {
		pools = append(pools, map[string]any{
			"name":              pool.Name,
			"count":             pool.Count,
			"vmSize":            pool.VMSize,
			"osType":            pool.OSType,
			"mode":              pool.Mode,
			"nodeImageVersion":  pool.NodeImageVersion,
			"provisioningState": pool.ProvisioningState,
		})
	}
	props := map[string]any{
		"provisioningState":        cluster.ProvisioningState,
		"kubernetesVersion":        cluster.KubernetesVersion,
		"currentKubernetesVersion": cluster.KubernetesVersion,
		"powerState":               map[string]string{"code": managedClusterPowerState(cluster)},
		"dnsPrefix":                cluster.DNSPrefix,
		"fqdn":                     cluster.FQDN,
		"enableRBAC":               true,
		"nodeResourceGroup":        cluster.NodeResourceGroupName,
		"agentPoolProfiles":        pools,
	}
	out := map[string]any{
		"id":         cluster.ID,
		"name":       cluster.Name,
		"type":       cluster.Type,
		"location":   cluster.Location,
		"properties": props,
	}
	if len(cluster.Tags) > 0 {
		out["tags"] = cluster.Tags
	}
	return out
}

func agentPoolResponse(subscriptionID, resourceGroup, clusterName string, pool AgentPoolProfile) map[string]any {
	return map[string]any{
		"id":   clusterID(subscriptionID, resourceGroup, clusterName) + "/agentPools/" + pool.Name,
		"name": pool.Name,
		"type": "Microsoft.ContainerService/managedClusters/agentPools",
		"properties": map[string]any{
			"count":             pool.Count,
			"vmSize":            pool.VMSize,
			"osType":            pool.OSType,
			"mode":              pool.Mode,
			"nodeImageVersion":  pool.NodeImageVersion,
			"provisioningState": pool.ProvisioningState,
		},
	}
}

func commandResultResponse(result CommandResult) map[string]any {
	properties := map[string]any{
		"exitCode":          result.ExitCode,
		"finishedAt":        result.FinishedAt,
		"logs":              result.Logs,
		"provisioningState": result.ProvisioningState,
		"startedAt":         result.StartedAt,
	}
	if result.Reason != "" {
		properties["reason"] = result.Reason
	}
	return map[string]any{
		"id":         result.ID,
		"properties": properties,
	}
}

func kubernetesVersionProfile(version string, isDefault, isPreview bool, patches map[string][]string) map[string]any {
	patchVersions := make(map[string]any, len(patches))
	for patchVersion, upgrades := range patches {
		patchVersions[patchVersion] = map[string]any{"upgrades": upgrades}
	}
	profile := map[string]any{
		"version":       version,
		"capabilities":  map[string]any{"supportPlan": []string{"KubernetesOfficial"}},
		"patchVersions": patchVersions,
	}
	if isDefault {
		profile["isDefault"] = true
	}
	if isPreview {
		profile["isPreview"] = true
	}
	return profile
}

func managedClusterUpgradeProfileResponse(cluster ManagedCluster) map[string]any {
	upgrades := upgradeVersionItems(cluster.KubernetesVersion)
	pools := make([]map[string]any, 0, len(cluster.AgentPoolProfiles))
	for _, pool := range cluster.AgentPoolProfiles {
		pools = append(pools, map[string]any{
			"name":              pool.Name,
			"kubernetesVersion": cluster.KubernetesVersion,
			"osType":            pool.OSType,
			"upgrades":          upgrades,
		})
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i]["name"].(string) < pools[j]["name"].(string) })
	return map[string]any{
		"id":   cluster.ID + "/upgradeprofiles/default",
		"name": "default",
		"type": "Microsoft.ContainerService/managedClusters/upgradeprofiles",
		"properties": map[string]any{
			"agentPoolProfiles": pools,
			"controlPlaneProfile": map[string]any{
				"name":              "master",
				"kubernetesVersion": cluster.KubernetesVersion,
				"osType":            "Linux",
				"upgrades":          upgrades,
			},
		},
	}
}

func agentPoolUpgradeProfileResponse(cluster ManagedCluster, pool AgentPoolProfile) map[string]any {
	return map[string]any{
		"id":   cluster.ID + "/agentPools/" + pool.Name + "/upgradeprofiles/default",
		"name": "default",
		"type": "Microsoft.ContainerService/managedClusters/agentPools/upgradeProfiles",
		"properties": map[string]any{
			"kubernetesVersion":      cluster.KubernetesVersion,
			"latestNodeImageVersion": latestNodeImageVersion(pool.OSType),
			"osType":                 pool.OSType,
			"upgrades":               upgradeVersionItems(cluster.KubernetesVersion),
		},
	}
}

func upgradeVersionItems(current string) []map[string]any {
	major, minor := parseKubernetesMajorMinor(current)
	return []map[string]any{
		{"kubernetesVersion": fmt.Sprintf("%d.%d.0", major, minor+1)},
		{"kubernetesVersion": fmt.Sprintf("%d.%d.0", major, minor+2)},
	}
}

func parseKubernetesMajorMinor(current string) (int, int) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(current), "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) >= 2 {
		major, majorErr := strconv.Atoi(parts[0])
		minor, minorErr := strconv.Atoi(parts[1])
		if majorErr == nil && minorErr == nil && major > 0 && minor >= 0 {
			return major, minor
		}
	}
	return 1, 29
}

func latestNodeImageVersion(osType string) string {
	if strings.EqualFold(osType, "Windows") {
		return "AKSWindows:2022:2026.03.01"
	}
	return "AKSUbuntu:2204:2026.03.01"
}

func initialNodeImageVersion(osType string) string {
	if strings.EqualFold(osType, "Windows") {
		return "AKSWindows:2022:2026.02.01"
	}
	return "AKSUbuntu:2204:2026.02.01"
}

func agentPoolOperationInProgress(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "upgradingnodeimageversion", "creating", "updating", "deleting", "canceling":
		return true
	default:
		return false
	}
}

func managedClusterPowerState(cluster ManagedCluster) string {
	if cluster.PowerState == "" {
		return "Running"
	}
	return cluster.PowerState
}

func defaultAgentPool() AgentPoolProfile {
	return AgentPoolProfile{
		Name:              "nodepool1",
		Count:             1,
		VMSize:            "Standard_DS2_v2",
		OSType:            "Linux",
		Mode:              "System",
		ProvisioningState: "Succeeded",
	}
}

func normalizeAgentPool(pool *AgentPoolProfile) {
	if pool.Name == "" {
		pool.Name = "nodepool1"
	}
	if pool.Count == 0 {
		pool.Count = 1
	}
	if pool.VMSize == "" {
		pool.VMSize = "Standard_DS2_v2"
	}
	if pool.OSType == "" {
		pool.OSType = "Linux"
	}
	if pool.Mode == "" {
		pool.Mode = "System"
	}
	if pool.NodeImageVersion == "" {
		pool.NodeImageVersion = initialNodeImageVersion(pool.OSType)
	}
	pool.ProvisioningState = "Succeeded"
}

func mergeAgentPool(existing *AgentPoolProfile, update AgentPoolProfile) {
	if update.Count != 0 {
		existing.Count = update.Count
	}
	if update.VMSize != "" {
		existing.VMSize = update.VMSize
	}
	if update.OSType != "" {
		existing.OSType = update.OSType
	}
	if update.Mode != "" {
		existing.Mode = update.Mode
	}
	if update.NodeImageVersion != "" {
		existing.NodeImageVersion = update.NodeImageVersion
	}
	if existing.NodeImageVersion == "" {
		existing.NodeImageVersion = initialNodeImageVersion(existing.OSType)
	}
	existing.ProvisioningState = "Succeeded"
}

func mockKubeconfig(cluster ManagedCluster) string {
	return mockKubeconfigForCredential(cluster, "clusterAdmin", "", "")
}

func mockKubeconfigForCredential(cluster ManagedCluster, credentialName, serverFQDN, format string) string {
	server := cluster.Endpoint
	if serverFQDN != "" {
		server = "https://" + serverFQDN + "." + cluster.FQDN
	}
	yaml := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: %s
contexts:
- context:
    cluster: %s
    user: %s_%s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s_%s
  user:
%s`, server, cluster.Name, cluster.Name, credentialName, cluster.Name, cluster.Name, cluster.Name, credentialName, cluster.Name, kubeconfigUserAuthBlock(format))
	return base64.StdEncoding.EncodeToString([]byte(yaml))
}

func kubeconfigUserAuthBlock(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "exec":
		return fmt.Sprintf(`    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: kubelogin
      args:
      - get-token
      - --login
      - azurecli
      - --server-id
      - %s
`, aksServerApplicationID)
	case "azure":
		return fmt.Sprintf(`    auth-provider:
      name: azure
      config:
        apiserver-id: %s
        client-id: %s
        tenant-id: 00000000-0000-0000-0000-000000000000
`, aksServerApplicationID, aksClientApplicationID)
	default:
		return "    token: cloudmock-aks-mock-token\n"
	}
}

func clusterID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerService/managedClusters/" + name
}

func managedClusterOperationLocation(cluster ManagedCluster, actionName, apiVersion string) string {
	if apiVersion == "" {
		apiVersion = "2026-03-01"
	}
	operationID := strings.ToLower(cluster.Name + "-" + actionName)
	return "/subscriptions/" + cluster.SubscriptionID + "/providers/Microsoft.ContainerService/locations/" + cluster.Location + "/operationresults/" + operationID + "?api-version=" + url.QueryEscape(apiVersion)
}

func agentPoolOperationLocation(cluster ManagedCluster, poolName, actionName, apiVersion string) string {
	if apiVersion == "" {
		apiVersion = "2026-03-01"
	}
	operationID := strings.ToLower(cluster.Name + "-" + poolName + "-" + actionName)
	return "/subscriptions/" + cluster.SubscriptionID + "/providers/Microsoft.ContainerService/locations/" + cluster.Location + "/operationresults/" + operationID + "?api-version=" + url.QueryEscape(apiVersion)
}

func agentPoolOperationStatusLocation(cluster ManagedCluster, poolName, actionName, apiVersion string) string {
	return agentPoolOperationLocationWithSegment(cluster, poolName, actionName, apiVersion, "operationStatus")
}

func agentPoolOperationResourceLocation(cluster ManagedCluster, poolName, actionName, apiVersion string) string {
	return agentPoolOperationLocationWithSegment(cluster, poolName, actionName, apiVersion, "operations")
}

func agentPoolOperationLocationWithSegment(cluster ManagedCluster, poolName, actionName, apiVersion, segment string) string {
	if apiVersion == "" {
		apiVersion = "2026-03-01"
	}
	operationID := strings.ToLower(cluster.Name + "-" + poolName + "-" + actionName)
	return "/subscriptions/" + cluster.SubscriptionID + "/providers/Microsoft.ContainerService/locations/" + cluster.Location + "/" + segment + "/" + operationID + "?api-version=" + url.QueryEscape(apiVersion)
}

func clusterKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func commandResultKey(subscriptionID, resourceGroup, clusterName, commandID string) string {
	return clusterKey(subscriptionID, resourceGroup, clusterName) + "/" + strings.ToLower(commandID)
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
