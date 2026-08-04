package routing

import (
	"net/http"
	"strings"
)

// Provider identifies the cloud provider protocol that owns a request.
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure"
	ProviderGCP   Provider = "gcp"
)

// RouteTarget is the provider-aware routing identity for an incoming request.
// The legacy AWS router only needed Service and Action; Azure and future
// providers also need an API version because resource-provider versions can
// coexist for the same logical service.
type RouteTarget struct {
	Provider   Provider
	Service    string
	Action     string
	APIVersion string
}

// ServiceKey identifies a registered service implementation.
type ServiceKey struct {
	Provider   Provider
	Service    string
	APIVersion string
}

// DetectTarget returns the provider, service, action, and API version for r.
func DetectTarget(r *http.Request) RouteTarget {
	provider := DetectProvider(r)
	action := DetectAction(r)
	serviceName := ""

	switch provider {
	case ProviderAzure:
		serviceName = detectAzureService(r)
	default:
		serviceName = DetectService(r)
	}

	if override := r.Header.Get("X-Cloudmock-Service"); override != "" {
		serviceName = override
	}
	if provider == ProviderAzure && action == "" {
		action = detectAzureAction(r, serviceName)
	}

	return RouteTarget{
		Provider:   provider,
		Service:    serviceName,
		Action:     action,
		APIVersion: DetectAPIVersion(r, provider),
	}
}

// DetectProvider identifies the cloud provider for r.
func DetectProvider(r *http.Request) Provider {
	if provider := strings.TrimSpace(r.Header.Get("X-Cloudmock-Provider")); provider != "" {
		return Provider(strings.ToLower(provider))
	}

	host := normalizedHost(r)

	if isAzureManagementHost(host) || azureStorageDataPlaneKind(host) != "" || isAzureStorageLocalDataPlaneRequest(r) || isAzureContainerRegistryDataPlaneRequest(r) || isAzureCosmosDBDataPlaneRequest(r) || isAzureServiceBusDataPlaneHost(host) || isAzureEventGridDataPlaneHost(host) || isAzureApplicationInsightsQueryHost(host) || isAzureLogAnalyticsQueryHost(host) || isAzureKeyVaultDataPlaneHost(host) || isAzureAppConfigurationDataPlaneRequest(r) || isAzureAPIManagementLocalDataPlaneRequest(r) || isAzureFunctionsLocalDataPlaneRequest(r) || looksLikeAzureARMRequest(r) {
		return ProviderAzure
	}
	if strings.HasSuffix(host, ".googleapis.com") || strings.HasSuffix(host, ".googleapis.cn") {
		return ProviderGCP
	}
	if DetectService(r) != "" {
		return ProviderAWS
	}
	return ""
}

// DetectAPIVersion extracts the provider-specific API version.
func DetectAPIVersion(r *http.Request, provider Provider) string {
	if version := strings.TrimSpace(r.Header.Get("X-Cloudmock-API-Version")); version != "" {
		return version
	}

	switch provider {
	case ProviderAzure:
		if version := r.URL.Query().Get("api-version"); version != "" {
			return version
		}
		if isAzureApplicationInsightsQueryHost(normalizedHost(r)) || isAzureLogAnalyticsQueryHost(normalizedHost(r)) {
			parts := splitPath(r.URL.EscapedPath())
			if len(parts) > 0 && strings.HasPrefix(strings.ToLower(parts[0]), "v") {
				return parts[0]
			}
		}
		return r.Header.Get("x-ms-version")
	case ProviderAWS:
		if version := awsAPIVersionFromTarget(r.Header.Get("X-Amz-Target")); version != "" {
			return version
		}
		return r.URL.Query().Get("Version")
	default:
		return ""
	}
}

func normalizedHost(r *http.Request) string {
	host := strings.ToLower(r.Host)
	if host == "" {
		host = strings.ToLower(r.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	return host
}

func awsAPIVersionFromTarget(target string) string {
	if target == "" {
		return ""
	}
	servicePart := target
	if dot := strings.IndexByte(servicePart, '.'); dot >= 0 {
		servicePart = servicePart[:dot]
	}
	if under := strings.IndexByte(servicePart, '_'); under >= 0 && under+1 < len(servicePart) {
		return servicePart[under+1:]
	}
	return ""
}

func isAzureManagementHost(host string) bool {
	switch host {
	case "management.azure.com",
		"management.usgovcloudapi.net",
		"management.microsoftazure.de",
		"management.chinacloudapi.cn":
		return true
	default:
		return false
	}
}

func looksLikeAzureARMRequest(r *http.Request) bool {
	path := strings.ToLower(r.URL.EscapedPath())
	if strings.HasPrefix(path, "/subscriptions/") || strings.HasPrefix(path, "/providers/") {
		return true
	}
	if r.URL.Query().Get("api-version") == "" {
		return false
	}
	auth := strings.ToLower(r.Header.Get("Authorization"))
	return strings.HasPrefix(auth, "bearer ")
}

func detectAzureService(r *http.Request) string {
	if isAzureContainerRegistryDataPlaneRequest(r) {
		return "Microsoft.ContainerRegistry/registry"
	}
	if isAzureEventHubRuntimeDataPlaneRequest(r) {
		return "Microsoft.EventHub/runtime"
	}
	if isAzureServiceBusDataPlaneHost(normalizedHost(r)) {
		return "Microsoft.ServiceBus/runtime"
	}
	if isAzureEventGridPublishDataPlaneRequest(r) {
		return "Microsoft.EventGrid/publish"
	}
	if isAzureApplicationInsightsQueryHost(normalizedHost(r)) {
		return "Microsoft.Insights/query"
	}
	if isAzureLogAnalyticsQueryHost(normalizedHost(r)) {
		return "Microsoft.OperationalInsights/query"
	}
	if isAzureAppConfigurationDataPlaneRequest(r) {
		return "Microsoft.AppConfiguration/kv"
	}
	if isAzureAPIManagementLocalDataPlaneRequest(r) {
		return "Microsoft.ApiManagement/service"
	}
	if isAzureFunctionsLocalDataPlaneRequest(r) {
		return "Microsoft.Web/functions"
	}
	if isAzureCosmosDBDataPlaneRequest(r) {
		return "Microsoft.DocumentDB/sqlApi"
	}
	if isAzureKeyVaultDataPlaneHost(normalizedHost(r)) {
		parts := splitPath(r.URL.EscapedPath())
		if len(parts) > 0 && strings.EqualFold(parts[0], "keys") {
			return "Microsoft.KeyVault/keys"
		}
		return "Microsoft.KeyVault/secrets"
	}
	switch azureStorageDataPlaneKind(normalizedHost(r)) {
	case "blob":
		return "Microsoft.Storage/blobServices"
	case "queue":
		return "Microsoft.Storage/queueServices"
	case "table":
		return "Microsoft.Storage/tableServices"
	case "file":
		return "Microsoft.Storage/fileServices"
	}
	switch azureStorageLocalDataPlaneKind(r) {
	case "blob":
		return "Microsoft.Storage/blobServices"
	case "queue":
		return "Microsoft.Storage/queueServices"
	case "table":
		return "Microsoft.Storage/tableServices"
	case "file":
		return "Microsoft.Storage/fileServices"
	}

	parts := splitPath(r.URL.EscapedPath())
	if isAzureContainerRegistrySubscriptionAction(parts) {
		return "Microsoft.ContainerRegistry/registries"
	}
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.EqualFold(parts[i], "providers") {
			if azurePathIsResourceProviderManagement(parts, i) {
				return "Microsoft.Resources/providers"
			}
			if i+2 >= len(parts) {
				return "Microsoft.Resources/providers"
			}
			return parts[i+1] + "/" + parts[i+2]
		}
	}

	for _, part := range parts {
		if strings.EqualFold(part, "resourceGroups") {
			return "Microsoft.Resources/resourceGroups"
		}
	}
	if len(parts) > 0 && strings.EqualFold(parts[0], "subscriptions") {
		return "Microsoft.Resources/subscriptions"
	}
	return ""
}

func detectAzureAction(r *http.Request, serviceName string) string {
	if serviceName == "" {
		return ""
	}
	if serviceName == "Microsoft.KeyVault/secrets" {
		return detectAzureKeyVaultSecretAction(r)
	}
	if serviceName == "Microsoft.KeyVault/keys" {
		return detectAzureKeyVaultKeyAction(r)
	}
	if serviceName == "Microsoft.AppConfiguration/kv" {
		return detectAzureAppConfigurationAction(r)
	}
	if serviceName == "Microsoft.DocumentDB/sqlApi" {
		return detectAzureCosmosDBAction(r)
	}
	if serviceName == "Microsoft.ServiceBus/runtime" {
		return detectAzureServiceBusRuntimeAction(r)
	}
	if serviceName == "Microsoft.EventHub/runtime" {
		return detectAzureEventHubRuntimeAction(r)
	}
	if serviceName == "Microsoft.ServiceBus/namespaces" {
		if action := detectAzureServiceBusControlPlaneAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.EventGrid/publish" {
		return detectAzureEventGridPublishAction(r)
	}
	if serviceName == "Microsoft.Insights/query" {
		return detectAzureApplicationInsightsQueryAction(r)
	}
	if serviceName == "Microsoft.Insights/eventtypes" {
		if action := detectAzureActivityLogsAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.OperationalInsights/query" {
		return detectAzureLogAnalyticsQueryAction(r)
	}
	if serviceName == "Microsoft.EventGrid/topics" {
		if action := detectAzureEventGridTopicAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.EventHub/namespaces" {
		if action := detectAzureEventHubNamespaceAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerRegistry/registries" {
		if action := detectAzureContainerRegistryAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerRegistry/registry" {
		if action := detectAzureContainerRegistryDataPlaneAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerInstance/containerGroups" {
		if action := detectAzureContainerInstanceAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerInstance/containerGroupProfiles" {
		if action := detectAzureContainerInstanceProfileAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerInstance/locations" {
		if action := detectAzureContainerInstanceLocationAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerInstance/operations" {
		if action := detectAzureContainerInstanceOperationsAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.App/containerApps" || serviceName == "Microsoft.App/managedEnvironments" {
		if action := detectAzureContainerAppsAction(r, serviceName); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Compute/virtualMachines" {
		if action := detectAzureComputeVirtualMachineAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Compute/disks" {
		if action := detectAzureComputeDiskAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerService/managedClusters" {
		if action := detectAzureAKSAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ContainerService/locations" {
		if action := detectAzureAKSLocationAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.ApiManagement/service" {
		if action := detectAzureAPIManagementAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Web/sites" {
		if action := detectAzureAppServiceSiteAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Web/serverfarms" {
		if action := detectAzureAppServicePlanAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Web/functions" {
		if action := detectAzureFunctionsAction(r); action != "" {
			return action
		}
	}
	if serviceName == "Microsoft.Insights/components" {
		if action := detectAzureApplicationInsightsComponentAction(r); action != "" {
			return action
		}
	}
	if strings.EqualFold(serviceName, "Microsoft.OperationalInsights/workspaces") {
		if action := detectAzureLogAnalyticsWorkspaceAction(r); action != "" {
			return action
		}
	}
	if strings.EqualFold(serviceName, "Microsoft.Cdn/profiles") {
		if action := detectAzureCDNAction(r); action != "" {
			return action
		}
	}
	switch r.Method {
	case http.MethodPut:
		return "CreateOrUpdate"
	case http.MethodPatch:
		return "Update"
	case "MERGE":
		return "Merge"
	case http.MethodDelete:
		return "Delete"
	case http.MethodPost:
		parts := splitPath(r.URL.EscapedPath())
		if len(parts) == 0 {
			return ""
		}
		return parts[len(parts)-1]
	case http.MethodGet, http.MethodHead:
		if azurePathLooksLikeCollection(r, serviceName) {
			return "List"
		}
		return "Get"
	default:
		return ""
	}
}

func azurePathLooksLikeCollection(r *http.Request, serviceName string) bool {
	parts := splitPath(r.URL.EscapedPath())
	if serviceName == "Microsoft.KeyVault/secrets" {
		return len(parts) == 1 && strings.EqualFold(parts[0], "secrets")
	}
	if serviceName == "Microsoft.KeyVault/keys" {
		return len(parts) == 1 && strings.EqualFold(parts[0], "keys")
	}
	if serviceName == "Microsoft.Resources/resourceGroups" {
		return len(parts) > 0 && strings.EqualFold(parts[len(parts)-1], "resourceGroups")
	}
	if serviceName == "Microsoft.Resources/providers" {
		return len(parts) > 0 && strings.EqualFold(parts[len(parts)-1], "providers")
	}
	if serviceName == "Microsoft.Storage/blobServices" {
		return strings.EqualFold(r.URL.Query().Get("comp"), "list")
	}
	if serviceName == "Microsoft.Storage/fileServices" {
		return strings.EqualFold(r.URL.Query().Get("comp"), "list")
	}
	if serviceName == "" {
		return false
	}
	for i := 0; i+2 < len(parts); i++ {
		if strings.EqualFold(parts[i], "providers") && parts[i+1]+"/"+parts[i+2] == serviceName {
			return i+3 == len(parts)
		}
	}
	return false
}

func detectAzureCDNAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+4 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") ||
			!strings.EqualFold(parts[i+1], "Microsoft.Cdn") ||
			!strings.EqualFold(parts[i+2], "profiles") {
			continue
		}
		if i+4 >= len(parts) || !strings.EqualFold(parts[i+4], "endpoints") {
			return ""
		}
		isOriginGroupCollection := i+7 == len(parts) && strings.EqualFold(parts[i+6], "originGroups")
		isOriginGroupResource := i+8 == len(parts) && strings.EqualFold(parts[i+6], "originGroups")
		isOriginCollection := i+7 == len(parts) && strings.EqualFold(parts[i+6], "origins")
		isOriginResource := i+8 == len(parts) && strings.EqualFold(parts[i+6], "origins")
		isCustomDomainCollection := i+7 == len(parts) && strings.EqualFold(parts[i+6], "customDomains")
		isCustomDomainResource := i+8 == len(parts) && strings.EqualFold(parts[i+6], "customDomains")
		isCustomDomainAction := i+9 == len(parts) && strings.EqualFold(parts[i+6], "customDomains")
		isEndpointCollection := i+5 == len(parts)
		isEndpointResource := i+6 == len(parts)
		switch r.Method {
		case http.MethodPut:
			if isCustomDomainResource {
				return "CreateOrUpdateCustomDomain"
			}
			if isOriginResource {
				return "CreateOrUpdateOrigin"
			}
			if isOriginGroupResource {
				return "CreateOrUpdateOriginGroup"
			}
			if isEndpointResource {
				return "CreateOrUpdateEndpoint"
			}
		case http.MethodGet, http.MethodHead:
			if isCustomDomainCollection {
				return "ListCustomDomains"
			}
			if isCustomDomainResource {
				return "GetCustomDomain"
			}
			if isOriginCollection {
				return "ListOrigins"
			}
			if isOriginResource {
				return "GetOrigin"
			}
			if isOriginGroupCollection {
				return "ListOriginGroups"
			}
			if isOriginGroupResource {
				return "GetOriginGroup"
			}
			if isEndpointCollection {
				return "ListEndpoints"
			}
			if isEndpointResource {
				return "GetEndpoint"
			}
		case http.MethodDelete:
			if isCustomDomainResource {
				return "DeleteCustomDomain"
			}
			if isOriginResource {
				return "DeleteOrigin"
			}
			if isOriginGroupResource {
				return "DeleteOriginGroup"
			}
			if isEndpointResource {
				return "DeleteEndpoint"
			}
		case http.MethodPost:
			if isCustomDomainAction {
				switch {
				case strings.EqualFold(parts[i+8], "enableCustomHttps"):
					return "EnableCustomHttps"
				case strings.EqualFold(parts[i+8], "disableCustomHttps"):
					return "DisableCustomHttps"
				}
			}
		}
	}
	return ""
}

func detectAzureKeyVaultSecretAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	if strings.EqualFold(parts[0], "deletedsecrets") {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			if len(parts) == 1 {
				return "ListDeletedSecrets"
			}
			if len(parts) == 2 {
				return "GetDeletedSecret"
			}
		case http.MethodPost:
			if len(parts) == 3 && strings.EqualFold(parts[2], "recover") {
				return "RecoverDeletedSecret"
			}
		case http.MethodDelete:
			if len(parts) == 2 {
				return "PurgeDeletedSecret"
			}
		}
		return ""
	}
	if !strings.EqualFold(parts[0], "secrets") {
		return ""
	}

	switch r.Method {
	case http.MethodPut:
		if len(parts) == 2 {
			return "SetSecret"
		}
	case http.MethodGet, http.MethodHead:
		if len(parts) == 1 {
			return "ListSecrets"
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "versions") {
			return "ListSecretVersions"
		}
		if len(parts) == 2 || len(parts) == 3 {
			return "GetSecret"
		}
	case http.MethodPatch:
		if len(parts) == 3 && !strings.EqualFold(parts[2], "versions") {
			return "UpdateSecret"
		}
	case http.MethodPost:
		if len(parts) == 3 && strings.EqualFold(parts[2], "backup") {
			return "BackupSecret"
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return "DeleteSecret"
		}
	}
	return ""
}

func detectAzureKeyVaultKeyAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "keys") {
		return ""
	}
	if r.Method != http.MethodPost {
		return ""
	}
	if len(parts) == 3 && strings.EqualFold(parts[2], "create") {
		return "CreateKey"
	}
	if len(parts) == 4 && strings.EqualFold(parts[3], "encrypt") {
		return "Encrypt"
	}
	if len(parts) == 4 && strings.EqualFold(parts[3], "decrypt") {
		return "Decrypt"
	}
	return ""
}

func detectAzureLogAnalyticsWorkspaceAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	providerIndex := -1
	for i, part := range parts {
		if strings.EqualFold(part, "providers") {
			providerIndex = i
			break
		}
	}
	if providerIndex < 4 || providerIndex+2 >= len(parts) ||
		!strings.EqualFold(parts[providerIndex+1], "Microsoft.OperationalInsights") ||
		!strings.EqualFold(parts[providerIndex+2], "workspaces") {
		return ""
	}
	if len(parts) == providerIndex+3 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListWorkspacesByResourceGroup"
	}
	if len(parts) != providerIndex+4 {
		return ""
	}

	switch r.Method {
	case http.MethodPut:
		return "CreateOrUpdateWorkspace"
	case http.MethodGet, http.MethodHead:
		return "GetWorkspace"
	case http.MethodDelete:
		return "DeleteWorkspace"
	default:
		return ""
	}
}

func detectAzureAppConfigurationAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-appconfig") {
		parts = parts[1:]
	}
	if len(parts) == 0 || !strings.EqualFold(parts[0], "kv") {
		if len(parts) == 1 && strings.EqualFold(parts[0], "keys") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListKeys"
		}
		if len(parts) == 1 && strings.EqualFold(parts[0], "labels") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListLabels"
		}
		if len(parts) == 1 && strings.EqualFold(parts[0], "revisions") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListRevisions"
		}
		if len(parts) == 1 && strings.EqualFold(parts[0], "snapshots") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListSnapshots"
		}
		if len(parts) == 1 && strings.EqualFold(parts[0], "operations") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "GetSnapshotOperation"
		}
		if len(parts) == 2 && strings.EqualFold(parts[0], "snapshots") {
			switch r.Method {
			case http.MethodPut:
				return "CreateSnapshot"
			case http.MethodGet, http.MethodHead:
				return "GetSnapshot"
			case http.MethodPatch:
				return "UpdateSnapshot"
			}
		}
		if len(parts) == 2 && strings.EqualFold(parts[0], "locks") {
			switch r.Method {
			case http.MethodPut:
				return "LockKeyValue"
			case http.MethodDelete:
				return "UnlockKeyValue"
			}
		}
		return ""
	}
	switch r.Method {
	case http.MethodPut:
		if len(parts) == 2 {
			return "SetKeyValue"
		}
	case http.MethodGet, http.MethodHead:
		if len(parts) == 1 {
			return "ListKeyValues"
		}
		if len(parts) == 2 {
			return "GetKeyValue"
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return "DeleteKeyValue"
		}
	}
	return ""
}

func azurePathIsResourceProviderManagement(parts []string, providerIndex int) bool {
	if providerIndex+1 >= len(parts) {
		return true
	}
	if providerIndex+2 >= len(parts) {
		return true
	}
	next := parts[providerIndex+2]
	return strings.EqualFold(next, "register") || strings.EqualFold(next, "unregister")
}

func isAzureKeyVaultDataPlaneHost(host string) bool {
	suffixes := []string{
		".vault.azure.net",
		".vault.usgovcloudapi.net",
		".vault.azure.cn",
		".vault.microsoftazure.de",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			return true
		}
	}
	return false
}

func isAzureAppConfigurationDataPlaneRequest(r *http.Request) bool {
	host := normalizedHost(r)
	if strings.HasSuffix(host, ".azconfig.io") && strings.TrimSuffix(host, ".azconfig.io") != "" {
		return true
	}
	parts := splitPath(r.URL.EscapedPath())
	return len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-appconfig") && r.URL.Query().Get("api-version") != ""
}

func isAzureAPIManagementLocalDataPlaneRequest(r *http.Request) bool {
	parts := splitPath(r.URL.EscapedPath())
	return len(parts) >= 2 && strings.HasSuffix(strings.ToLower(parts[0]), "-apim")
}

func isAzureFunctionsLocalDataPlaneRequest(r *http.Request) bool {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) < 2 || !strings.HasSuffix(strings.ToLower(parts[0]), "-functions") {
		return false
	}
	return strings.EqualFold(parts[1], "admin") || strings.EqualFold(parts[1], "api")
}

func detectAzureFunctionsAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) < 2 || !strings.HasSuffix(strings.ToLower(parts[0]), "-functions") {
		return ""
	}
	parts = parts[1:]
	if len(parts) >= 3 && strings.EqualFold(parts[0], "api") {
		return "InvokeFunction"
	}
	if len(parts) < 2 || !strings.EqualFold(parts[0], "admin") || !strings.EqualFold(parts[1], "apps") {
		return ""
	}
	switch len(parts) {
	case 2:
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			return "ListFunctionApps"
		}
	case 3:
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateFunctionApp"
		case http.MethodGet, http.MethodHead:
			return "GetFunctionApp"
		case http.MethodDelete:
			return "DeleteFunctionApp"
		}
	case 4:
		if strings.EqualFold(parts[3], "functions") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListFunctions"
		}
	case 5:
		if !strings.EqualFold(parts[3], "functions") {
			return ""
		}
		switch r.Method {
		case http.MethodPut:
			return "DeployFunction"
		case http.MethodGet, http.MethodHead:
			return "GetFunction"
		case http.MethodDelete:
			return "DeleteFunction"
		}
	}
	return ""
}

func detectAzureAppServiceSiteAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") || !strings.EqualFold(parts[i+1], "Microsoft.Web") || !strings.EqualFold(parts[i+2], "sites") {
			continue
		}
		if i+3 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListSites"
		}
		if i+4 == len(parts) && r.Method == http.MethodPatch {
			return "UpdateSite"
		}
		if i+4 >= len(parts) {
			return ""
		}
		if strings.EqualFold(parts[i+4], "config") {
			if i+5 >= len(parts) {
				return ""
			}
			switch strings.ToLower(parts[i+5]) {
			case "web":
				if i+6 == len(parts) {
					switch r.Method {
					case http.MethodGet, http.MethodHead:
						return "GetConfiguration"
					case http.MethodPatch:
						return "UpdateConfiguration"
					}
				}
			case "appsettings":
				if i+6 < len(parts) && strings.EqualFold(parts[i+6], "list") && r.Method == http.MethodPost {
					return "ListApplicationSettings"
				}
				if i+6 == len(parts) && r.Method == http.MethodPut {
					return "UpdateApplicationSettings"
				}
			case "connectionstrings":
				if i+6 < len(parts) && strings.EqualFold(parts[i+6], "list") && r.Method == http.MethodPost {
					return "ListConnectionStrings"
				}
				if i+6 == len(parts) && r.Method == http.MethodPut {
					return "UpdateConnectionStrings"
				}
			case "slotconfignames":
				if i+6 == len(parts) {
					switch r.Method {
					case http.MethodGet, http.MethodHead:
						return "ListSlotConfigurationNames"
					case http.MethodPut:
						return "UpdateSlotConfigurationNames"
					}
				}
			case "publishingcredentials":
				if i+6 < len(parts) && strings.EqualFold(parts[i+6], "list") && r.Method == http.MethodPost {
					return "ListPublishingCredentials"
				}
			}
			return ""
		}
		if strings.EqualFold(parts[i+4], "publishxml") {
			if i+5 == len(parts) && r.Method == http.MethodPost {
				return "ListPublishingProfileXMLWithSecrets"
			}
			return ""
		}
		if strings.EqualFold(parts[i+4], "slots") {
			if i+5 == len(parts) {
				if r.Method == http.MethodGet || r.Method == http.MethodHead {
					return "ListSlots"
				}
				return ""
			}
			if i+6 == len(parts) {
				switch r.Method {
				case http.MethodPut:
					return "CreateOrUpdateSlot"
				case http.MethodPatch:
					return "UpdateSlot"
				case http.MethodGet, http.MethodHead:
					return "GetSlot"
				case http.MethodDelete:
					return "DeleteSlot"
				default:
					return ""
				}
			}
			if i+8 == len(parts) &&
				strings.EqualFold(parts[i+6], "config") &&
				strings.EqualFold(parts[i+7], "web") {
				switch r.Method {
				case http.MethodGet, http.MethodHead:
					return "GetSlotConfiguration"
				case http.MethodPut:
					return "CreateOrUpdateSlotConfiguration"
				case http.MethodPatch:
					return "UpdateSlotConfiguration"
				default:
					return ""
				}
			}
			return ""
		}
		if strings.EqualFold(parts[i+4], "host") {
			if i+5 >= len(parts) || !strings.EqualFold(parts[i+5], "default") {
				return ""
			}
			if i+7 == len(parts) && strings.EqualFold(parts[i+6], "listkeys") {
				if r.Method == http.MethodPost {
					return "ListHostKeys"
				}
				return ""
			}
			if i+8 == len(parts) {
				switch r.Method {
				case http.MethodPut:
					return "CreateOrUpdateHostSecret"
				case http.MethodDelete:
					return "DeleteHostSecret"
				default:
					return ""
				}
			}
			return ""
		}
		if strings.EqualFold(parts[i+4], "functions") {
			if i+5 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				return "ListFunctions"
			}
			if i+6 == len(parts) {
				switch r.Method {
				case http.MethodPut:
					return "CreateOrUpdateFunction"
				case http.MethodGet, http.MethodHead:
					return "GetFunction"
				case http.MethodDelete:
					return "DeleteFunction"
				default:
					return ""
				}
			}
			if i+7 == len(parts) && strings.EqualFold(parts[i+6], "listkeys") {
				if r.Method == http.MethodPost {
					return "ListFunctionKeys"
				}
				return ""
			}
			if i+8 == len(parts) && strings.EqualFold(parts[i+6], "keys") {
				switch r.Method {
				case http.MethodPut:
					return "CreateOrUpdateFunctionSecret"
				case http.MethodDelete:
					return "DeleteFunctionSecret"
				default:
					return ""
				}
			}
			return ""
		}
		if r.Method != http.MethodPost {
			return ""
		}
		switch strings.ToLower(parts[i+4]) {
		case "start":
			return "StartSite"
		case "stop":
			return "StopSite"
		case "restart":
			return "RestartSite"
		case "syncfunctiontriggers":
			return "SyncFunctionTriggers"
		default:
			return ""
		}
	}
	return ""
}

func detectAzureAppServicePlanAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") || !strings.EqualFold(parts[i+1], "Microsoft.Web") || !strings.EqualFold(parts[i+2], "serverfarms") {
			continue
		}
		if i+3 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListPlans"
		}
		if i+4 == len(parts) {
			switch r.Method {
			case http.MethodPut:
				return "CreateOrUpdatePlan"
			case http.MethodGet, http.MethodHead:
				return "GetPlan"
			case http.MethodPatch:
				return "UpdatePlan"
			case http.MethodDelete:
				return "DeletePlan"
			default:
				return ""
			}
		}
		if i+4 >= len(parts) {
			return ""
		}
	}
	return ""
}

func detectAzureApplicationInsightsComponentAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") || !strings.EqualFold(parts[i+1], "Microsoft.Insights") || !strings.EqualFold(parts[i+2], "components") {
			continue
		}
		if i+3 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListComponents"
		}
		if i+4 != len(parts) {
			return ""
		}
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateComponent"
		case http.MethodGet, http.MethodHead:
			return "GetComponent"
		case http.MethodDelete:
			return "DeleteComponent"
		default:
			return ""
		}
	}
	return ""
}

func isAzureCosmosDBDataPlaneRequest(r *http.Request) bool {
	host := normalizedHost(r)
	if strings.HasSuffix(host, ".documents.azure.com") && strings.TrimSuffix(host, ".documents.azure.com") != "" {
		return true
	}
	if strings.HasSuffix(host, ".documents.azure.us") && strings.TrimSuffix(host, ".documents.azure.us") != "" {
		return true
	}
	if strings.HasSuffix(host, ".documents.azure.cn") && strings.TrimSuffix(host, ".documents.azure.cn") != "" {
		return true
	}
	parts := splitPath(r.URL.EscapedPath())
	return len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-cosmos") && r.Header.Get("x-ms-version") != ""
}

func isAzureServiceBusDataPlaneHost(host string) bool {
	suffixes := []string{
		".servicebus.windows.net",
		".servicebus.usgovcloudapi.net",
		".servicebus.chinacloudapi.cn",
		".servicebus.cloudapi.de",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			return true
		}
	}
	return false
}

func isAzureEventGridDataPlaneHost(host string) bool {
	suffixes := []string{
		".eventgrid.azure.net",
		".eventgrid.azure.us",
		".eventgrid.azure.cn",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			return true
		}
	}
	return false
}

func isAzureEventGridPublishDataPlaneRequest(r *http.Request) bool {
	if !isAzureEventGridDataPlaneHost(normalizedHost(r)) {
		return false
	}
	if r.URL.Query().Get("api-version") != "2018-01-01" {
		return false
	}
	parts := splitPath(r.URL.EscapedPath())
	return r.Method == http.MethodPost && len(parts) == 2 && strings.EqualFold(parts[0], "api") && strings.EqualFold(parts[1], "events")
}

func isAzureApplicationInsightsQueryHost(host string) bool {
	return host == "api.applicationinsights.io"
}

func isAzureLogAnalyticsQueryHost(host string) bool {
	return host == "api.loganalytics.azure.com" || host == "api.loganalytics.io"
}

func detectAzureEventGridPublishAction(r *http.Request) string {
	if isAzureEventGridPublishDataPlaneRequest(r) {
		return "PublishEvents"
	}
	return ""
}

func detectAzureApplicationInsightsQueryAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) != 4 || !strings.EqualFold(parts[1], "apps") || !strings.EqualFold(parts[3], "query") {
		return ""
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return "QueryGet"
	}
	return ""
}

func detectAzureLogAnalyticsQueryAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) != 4 || !strings.EqualFold(parts[1], "workspaces") || !strings.EqualFold(parts[3], "query") {
		return ""
	}
	if r.Method == http.MethodPost {
		return "QueryWorkspace"
	}
	return ""
}

func detectAzureActivityLogsAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) != 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "providers") ||
		!strings.EqualFold(parts[3], "Microsoft.Insights") ||
		!strings.EqualFold(parts[4], "eventtypes") ||
		!strings.EqualFold(parts[5], "management") ||
		!strings.EqualFold(parts[6], "values") {
		return ""
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return "ListActivityLogs"
	}
	return ""
}

func detectAzureEventGridTopicAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") ||
			!strings.EqualFold(parts[i+1], "Microsoft.EventGrid") ||
			!strings.EqualFold(parts[i+2], "topics") {
			continue
		}

		isTopicCollection := i+3 == len(parts)
		isTopicResource := i+4 == len(parts)
		isListKeysAction := i+5 == len(parts) && strings.EqualFold(parts[i+4], "listKeys")
		isRegenerateKeyAction := i+5 == len(parts) && strings.EqualFold(parts[i+4], "regenerateKey")
		isSubscriptionCollection := i+5 == len(parts) && strings.EqualFold(parts[i+4], "eventSubscriptions")
		isSubscriptionResource := i+6 == len(parts) && strings.EqualFold(parts[i+4], "eventSubscriptions")
		isGetDeliveryAttributesAction := i+7 == len(parts) && strings.EqualFold(parts[i+4], "eventSubscriptions") && strings.EqualFold(parts[i+6], "getDeliveryAttributes")

		switch r.Method {
		case http.MethodPut:
			if isSubscriptionResource {
				return "CreateOrUpdateTopicEventSubscription"
			}
			if isTopicResource {
				return "CreateOrUpdateTopic"
			}
		case http.MethodGet, http.MethodHead:
			if isSubscriptionCollection {
				return "ListTopicEventSubscriptions"
			}
			if isSubscriptionResource {
				return "GetTopicEventSubscription"
			}
			if isTopicCollection {
				return "ListTopics"
			}
			if isTopicResource {
				return "GetTopic"
			}
		case http.MethodDelete:
			if isSubscriptionResource {
				return "DeleteTopicEventSubscription"
			}
			if isTopicResource {
				return "DeleteTopic"
			}
		case http.MethodPost:
			if isListKeysAction {
				return "ListSharedAccessKeys"
			}
			if isRegenerateKeyAction {
				return "RegenerateKey"
			}
			if isGetDeliveryAttributesAction {
				return "GetDeliveryAttributes"
			}
		}
		return ""
	}
	return ""
}

func detectAzureEventHubNamespaceAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i < len(parts); i++ {
		if strings.EqualFold(parts[i], "authorizationRules") {
			return detectAzureEventHubAuthorizationRuleAction(r, parts, i)
		}
		if !strings.EqualFold(parts[i], "consumergroups") {
			continue
		}
		switch r.Method {
		case http.MethodPut:
			if i+1 < len(parts) {
				return "CreateOrUpdateConsumerGroup"
			}
		case http.MethodGet, http.MethodHead:
			if i+1 < len(parts) {
				return "GetConsumerGroup"
			}
			return "ListConsumerGroups"
		case http.MethodDelete:
			if i+1 < len(parts) {
				return "DeleteConsumerGroup"
			}
		}
	}
	return ""
}

func detectAzureEventHubAuthorizationRuleAction(r *http.Request, parts []string, index int) string {
	eventHubScoped := false
	for i := 0; i < index; i++ {
		if strings.EqualFold(parts[i], "eventhubs") {
			eventHubScoped = true
			break
		}
	}
	if index+2 < len(parts) {
		switch {
		case r.Method == http.MethodPost && strings.EqualFold(parts[index+2], "listKeys"):
			if eventHubScoped {
				return "ListEventHubKeys"
			}
			return "ListNamespaceKeys"
		case r.Method == http.MethodPost && strings.EqualFold(parts[index+2], "regenerateKeys"):
			if eventHubScoped {
				return "RegenerateEventHubKeys"
			}
			return "RegenerateNamespaceKeys"
		}
	}
	switch r.Method {
	case http.MethodPut:
		if index+1 < len(parts) {
			if eventHubScoped {
				return "CreateOrUpdateEventHubAuthorizationRule"
			}
			return "CreateOrUpdateNamespaceAuthorizationRule"
		}
	case http.MethodGet, http.MethodHead:
		if index+1 < len(parts) {
			if eventHubScoped {
				return "GetEventHubAuthorizationRule"
			}
			return "GetNamespaceAuthorizationRule"
		}
		if eventHubScoped {
			return "ListEventHubAuthorizationRules"
		}
		return "ListNamespaceAuthorizationRules"
	case http.MethodDelete:
		if index+1 < len(parts) {
			if eventHubScoped {
				return "DeleteEventHubAuthorizationRule"
			}
			return "DeleteNamespaceAuthorizationRule"
		}
	}
	return ""
}

func isAzureContainerRegistrySubscriptionAction(parts []string) bool {
	return len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ContainerRegistry") &&
		strings.EqualFold(parts[4], "checkNameAvailability")
}

func isAzureContainerRegistryDataPlaneRequest(r *http.Request) bool {
	host := normalizedHost(r)
	for _, suffix := range []string{".azurecr.io", ".azurecr.us", ".azurecr.cn", ".azurecr.de"} {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			parts := splitPath(r.URL.EscapedPath())
			return len(parts) > 0 && strings.EqualFold(parts[0], "v2")
		}
	}
	parts := splitPath(r.URL.EscapedPath())
	return len(parts) >= 2 && strings.HasSuffix(strings.ToLower(parts[0]), "-acr") && strings.EqualFold(parts[1], "v2")
}

func azureContainerRegistryDataPlaneParts(r *http.Request) []string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-acr") {
		parts = parts[1:]
	}
	return parts
}

func detectAzureContainerRegistryDataPlaneAction(r *http.Request) string {
	parts := azureContainerRegistryDataPlaneParts(r)
	if len(parts) == 0 || !strings.EqualFold(parts[0], "v2") {
		return ""
	}
	if len(parts) == 1 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "RegistryPing"
	}
	if len(parts) == 2 && strings.EqualFold(parts[1], "_catalog") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListRepositories"
	}
	if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-2], "tags") && strings.EqualFold(parts[len(parts)-1], "list") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListTags"
	}
	blobIndex := -1
	for i := 1; i < len(parts); i++ {
		if strings.EqualFold(parts[i], "blobs") {
			blobIndex = i
			break
		}
	}
	if blobIndex >= 2 && blobIndex+1 < len(parts) {
		if strings.EqualFold(parts[blobIndex+1], "uploads") {
			if len(parts) == blobIndex+2 && r.Method == http.MethodPost {
				return "StartBlobUpload"
			}
			if len(parts) == blobIndex+3 {
				switch r.Method {
				case http.MethodPatch:
					return "UploadBlobChunk"
				case http.MethodPut:
					return "CompleteBlobUpload"
				case http.MethodDelete:
					return "CancelBlobUpload"
				}
			}
			return ""
		}
		if len(parts) == blobIndex+2 {
			switch r.Method {
			case http.MethodGet, http.MethodHead:
				return "GetBlob"
			case http.MethodDelete:
				return "DeleteBlob"
			}
		}
		return ""
	}
	manifestIndex := -1
	for i := 1; i < len(parts); i++ {
		if strings.EqualFold(parts[i], "manifests") {
			manifestIndex = i
			break
		}
	}
	if manifestIndex < 2 || manifestIndex+2 != len(parts) {
		return ""
	}
	switch r.Method {
	case http.MethodPut:
		return "PutManifest"
	case http.MethodGet, http.MethodHead:
		return "GetManifest"
	case http.MethodDelete:
		return "DeleteManifest"
	default:
		return ""
	}
}

func detectAzureContainerRegistryAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if r.Method == http.MethodPost && strings.EqualFold(last, "checkNameAvailability") {
		return "CheckNameAvailability"
	}
	if r.Method == http.MethodPost && strings.EqualFold(last, "importImage") {
		return "ImportImage"
	}
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(last, "replications") {
		return "ListReplications"
	}
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(last, "listUsages") {
		return "ListUsages"
	}
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		strings.EqualFold(last, "registries") &&
		((len(parts) == 5 &&
			strings.EqualFold(parts[0], "subscriptions") &&
			strings.EqualFold(parts[2], "providers") &&
			strings.EqualFold(parts[3], "Microsoft.ContainerRegistry")) ||
			(len(parts) == 7 &&
				strings.EqualFold(parts[0], "subscriptions") &&
				strings.EqualFold(parts[2], "resourceGroups") &&
				strings.EqualFold(parts[4], "providers") &&
				strings.EqualFold(parts[5], "Microsoft.ContainerRegistry"))) {
		return "ListRegistries"
	}
	if r.Method == http.MethodPatch &&
		len(parts) == 8 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "resourceGroups") &&
		strings.EqualFold(parts[4], "providers") &&
		strings.EqualFold(parts[5], "Microsoft.ContainerRegistry") &&
		strings.EqualFold(parts[6], "registries") {
		return "UpdateRegistry"
	}
	return ""
}

func detectAzureContainerInstanceAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if strings.EqualFold(last, "containerGroups") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListContainerGroups"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "containerGroups") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateContainerGroup"
		case http.MethodGet, http.MethodHead:
			return "GetContainerGroup"
		case http.MethodDelete:
			return "DeleteContainerGroup"
		}
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-3], "containerGroups") {
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(last, "outboundNetworkDependenciesEndpoints") {
			return "GetOutboundNetworkDependenciesEndpoints"
		}
		if r.Method == http.MethodPost {
			switch {
			case strings.EqualFold(last, "start"):
				return "StartContainerGroup"
			case strings.EqualFold(last, "stop"):
				return "StopContainerGroup"
			case strings.EqualFold(last, "restart"):
				return "RestartContainerGroup"
			}
		}
	}
	if len(parts) >= 5 &&
		strings.EqualFold(parts[len(parts)-5], "containerGroups") &&
		strings.EqualFold(parts[len(parts)-3], "containers") {
		switch {
		case (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(last, "logs"):
			return "ListContainerLogs"
		case r.Method == http.MethodPost && strings.EqualFold(last, "exec"):
			return "ExecuteContainerCommand"
		case r.Method == http.MethodPost && strings.EqualFold(last, "attach"):
			return "AttachContainer"
		}
	}
	return ""
}

func detectAzureContainerInstanceProfileAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if strings.EqualFold(last, "containerGroupProfiles") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListContainerGroupProfiles"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "containerGroupProfiles") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateContainerGroupProfile"
		case http.MethodGet, http.MethodHead:
			return "GetContainerGroupProfile"
		case http.MethodPatch:
			return "UpdateContainerGroupProfile"
		case http.MethodDelete:
			return "DeleteContainerGroupProfile"
		}
	}
	if len(parts) >= 3 &&
		strings.EqualFold(parts[len(parts)-3], "containerGroupProfiles") &&
		strings.EqualFold(last, "revisions") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListContainerGroupProfileRevisions"
	}
	if len(parts) >= 4 &&
		strings.EqualFold(parts[len(parts)-4], "containerGroupProfiles") &&
		strings.EqualFold(parts[len(parts)-2], "revisions") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "GetContainerGroupProfileRevision"
	}
	return ""
}

func detectAzureContainerInstanceLocationAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) < 7 ||
		!strings.EqualFold(parts[len(parts)-3], "locations") ||
		(r.Method != http.MethodGet && r.Method != http.MethodHead) {
		return ""
	}
	switch {
	case strings.EqualFold(parts[len(parts)-1], "cachedImages"):
		return "ListCachedImages"
	case strings.EqualFold(parts[len(parts)-1], "capabilities"):
		return "ListCapabilities"
	case strings.EqualFold(parts[len(parts)-1], "usages"):
		return "ListUsage"
	default:
		return ""
	}
}

func detectAzureContainerInstanceOperationsAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 3 &&
		strings.EqualFold(parts[0], "providers") &&
		strings.EqualFold(parts[1], "Microsoft.ContainerInstance") &&
		strings.EqualFold(parts[2], "operations") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListOperations"
	}
	return ""
}

func detectAzureContainerAppsAction(r *http.Request, serviceName string) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	resourceType := "containerApps"
	singular := "ContainerApp"
	plural := "ContainerApps"
	if serviceName == "Microsoft.App/managedEnvironments" {
		resourceType = "managedEnvironments"
		singular = "ManagedEnvironment"
		plural = "ManagedEnvironments"
	}
	last := parts[len(parts)-1]
	if strings.EqualFold(last, resourceType) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "List" + plural
	}
	if serviceName == "Microsoft.App/containerApps" &&
		len(parts) >= 3 &&
		strings.EqualFold(parts[len(parts)-3], resourceType) &&
		strings.EqualFold(last, "listSecrets") &&
		r.Method == http.MethodPost {
		return "ListContainerAppSecrets"
	}
	if serviceName == "Microsoft.App/containerApps" &&
		len(parts) >= 3 &&
		strings.EqualFold(parts[len(parts)-3], resourceType) &&
		strings.EqualFold(last, "revisions") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListContainerAppRevisions"
	}
	if serviceName == "Microsoft.App/containerApps" &&
		len(parts) >= 4 &&
		strings.EqualFold(parts[len(parts)-4], resourceType) &&
		strings.EqualFold(parts[len(parts)-2], "revisions") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "GetContainerAppRevision"
	}
	if serviceName == "Microsoft.App/containerApps" &&
		len(parts) >= 5 &&
		strings.EqualFold(parts[len(parts)-5], resourceType) &&
		strings.EqualFold(parts[len(parts)-3], "revisions") &&
		r.Method == http.MethodPost {
		switch {
		case strings.EqualFold(last, "activate"):
			return "ActivateContainerAppRevision"
		case strings.EqualFold(last, "deactivate"):
			return "DeactivateContainerAppRevision"
		case strings.EqualFold(last, "restart"):
			return "RestartContainerAppRevision"
		}
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], resourceType) {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdate" + singular
		case http.MethodGet, http.MethodHead:
			return "Get" + singular
		case http.MethodDelete:
			return "Delete" + singular
		}
	}
	return ""
}

func detectAzureComputeVirtualMachineAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if strings.EqualFold(last, "virtualMachines") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListVirtualMachines"
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-3], "virtualMachines") {
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(last, "instanceView") {
			return "GetVirtualMachineInstanceView"
		}
		if r.Method == http.MethodPost {
			switch {
			case strings.EqualFold(last, "start"):
				return "StartVirtualMachine"
			case strings.EqualFold(last, "powerOff"):
				return "PowerOffVirtualMachine"
			case strings.EqualFold(last, "deallocate"):
				return "DeallocateVirtualMachine"
			case strings.EqualFold(last, "restart"):
				return "RestartVirtualMachine"
			case strings.EqualFold(last, "redeploy"):
				return "RedeployVirtualMachine"
			case strings.EqualFold(last, "reapply"):
				return "ReapplyVirtualMachine"
			case strings.EqualFold(last, "runCommand"):
				return "RunCommandVirtualMachine"
			}
		}
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "virtualMachines") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateVirtualMachine"
		case http.MethodGet, http.MethodHead:
			return "GetVirtualMachine"
		case http.MethodPatch:
			return "UpdateVirtualMachine"
		case http.MethodDelete:
			return "DeleteVirtualMachine"
		}
	}
	return ""
}

func detectAzureComputeDiskAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	if strings.EqualFold(parts[len(parts)-1], "disks") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListDisks"
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-3], "disks") && r.Method == http.MethodPost {
		switch {
		case strings.EqualFold(parts[len(parts)-1], "beginGetAccess"):
			return "GrantAccessDisk"
		case strings.EqualFold(parts[len(parts)-1], "endGetAccess"):
			return "RevokeAccessDisk"
		}
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "disks") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateDisk"
		case http.MethodGet, http.MethodHead:
			return "GetDisk"
		case http.MethodPatch:
			return "UpdateDisk"
		case http.MethodDelete:
			return "DeleteDisk"
		}
	}
	return ""
}

func detectAzureAKSAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if len(parts) >= 4 &&
		strings.EqualFold(parts[len(parts)-4], "agentPools") &&
		strings.EqualFold(parts[len(parts)-2], "upgradeProfiles") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "GetAgentPoolUpgradeProfile"
	}
	if len(parts) >= 3 &&
		strings.EqualFold(parts[len(parts)-3], "agentPools") &&
		strings.EqualFold(last, "upgradeNodeImageVersion") &&
		r.Method == http.MethodPost {
		return "UpgradeAgentPoolNodeImageVersion"
	}
	if len(parts) >= 3 &&
		strings.EqualFold(parts[len(parts)-3], "agentPools") &&
		strings.EqualFold(last, "abort") &&
		r.Method == http.MethodPost {
		return "AbortAgentPoolLatestOperation"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "commandResults") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "GetCommandResult"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "agentPools") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateAgentPool"
		case http.MethodGet, http.MethodHead:
			return "GetAgentPool"
		case http.MethodDelete:
			return "DeleteAgentPool"
		}
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "upgradeProfiles") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "GetManagedClusterUpgradeProfile"
	}
	if strings.EqualFold(last, "runCommand") && r.Method == http.MethodPost {
		return "RunCommand"
	}
	if strings.EqualFold(last, "agentPools") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListAgentPools"
	}
	if strings.EqualFold(last, "listClusterAdminCredential") && r.Method == http.MethodPost {
		return "ListClusterAdminCredentials"
	}
	if strings.EqualFold(last, "listClusterUserCredential") && r.Method == http.MethodPost {
		return "ListClusterUserCredentials"
	}
	if strings.EqualFold(last, "listClusterMonitoringUserCredential") && r.Method == http.MethodPost {
		return "ListClusterMonitoringUserCredentials"
	}
	if strings.EqualFold(last, "rotateClusterCertificates") && r.Method == http.MethodPost {
		return "RotateClusterCertificates"
	}
	if strings.EqualFold(last, "rotateServiceAccountSigningKeys") && r.Method == http.MethodPost {
		return "RotateServiceAccountSigningKeys"
	}
	if strings.EqualFold(last, "start") && r.Method == http.MethodPost {
		return "StartManagedCluster"
	}
	if strings.EqualFold(last, "stop") && r.Method == http.MethodPost {
		return "StopManagedCluster"
	}
	if strings.EqualFold(last, "managedClusters") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListManagedClusters"
	}
	if len(parts) >= 8 && strings.EqualFold(parts[len(parts)-2], "managedClusters") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateManagedCluster"
		case http.MethodGet, http.MethodHead:
			return "GetManagedCluster"
		case http.MethodPatch:
			return "UpdateManagedClusterTags"
		case http.MethodDelete:
			return "DeleteManagedCluster"
		}
	}
	return ""
}

func detectAzureAKSLocationAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) >= 7 &&
		strings.EqualFold(parts[len(parts)-3], "locations") &&
		strings.EqualFold(parts[len(parts)-1], "kubernetesVersions") &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListKubernetesVersions"
	}
	return ""
}

func detectAzureAPIManagementAction(r *http.Request) string {
	if isAzureAPIManagementLocalDataPlaneRequest(r) {
		return "GatewayProxy"
	}
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if action := detectAzureAPIManagementProductAction(r, parts); action != "" {
		return action
	}
	if action := detectAzureAPIManagementCollectionAction(r, parts, "subscriptions", "Subscription", "Subscriptions"); action != "" {
		return action
	}
	if action := detectAzureAPIManagementCollectionAction(r, parts, "namedValues", "NamedValue", "NamedValues"); action != "" {
		return action
	}
	if action := detectAzureAPIManagementCollectionAction(r, parts, "backends", "Backend", "Backends"); action != "" {
		return action
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "policies") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdatePolicy"
		case http.MethodGet, http.MethodHead:
			return "GetPolicy"
		case http.MethodDelete:
			return "DeletePolicy"
		}
	}
	if strings.EqualFold(last, "policies") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListPolicies"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "operations") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateOperation"
		case http.MethodGet, http.MethodHead:
			return "GetOperation"
		case http.MethodDelete:
			return "DeleteOperation"
		}
	}
	if strings.EqualFold(last, "operations") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListOperations"
	}
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "apis") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateAPI"
		case http.MethodGet, http.MethodHead:
			return "GetAPI"
		case http.MethodDelete:
			return "DeleteAPI"
		}
	}
	if strings.EqualFold(last, "apis") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListAPIs"
	}
	if strings.EqualFold(last, "service") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return "ListServices"
	}
	if len(parts) >= 8 && strings.EqualFold(parts[len(parts)-2], "service") {
		switch r.Method {
		case http.MethodPut:
			return "CreateOrUpdateService"
		case http.MethodGet, http.MethodHead:
			return "GetService"
		case http.MethodDelete:
			return "DeleteService"
		}
	}
	return ""
}

func detectAzureAPIManagementProductAction(r *http.Request, parts []string) string {
	for i, part := range parts {
		if !strings.EqualFold(part, "products") {
			continue
		}
		if i+2 < len(parts) && strings.EqualFold(parts[i+2], "apis") {
			if i+3 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				return "ListProductAPIs"
			}
			if i+4 == len(parts) {
				switch r.Method {
				case http.MethodPut:
					return "CreateOrUpdateProductAPI"
				case http.MethodGet, http.MethodHead:
					return "GetProductAPI"
				case http.MethodDelete:
					return "DeleteProductAPI"
				}
			}
			return ""
		}
		if i+1 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "ListProducts"
		}
		if i+2 == len(parts) {
			switch r.Method {
			case http.MethodPut:
				return "CreateOrUpdateProduct"
			case http.MethodGet, http.MethodHead:
				return "GetProduct"
			case http.MethodDelete:
				return "DeleteProduct"
			}
		}
		return ""
	}
	return ""
}

func detectAzureAPIManagementCollectionAction(r *http.Request, parts []string, collection, resourceName, listName string) string {
	for i, part := range parts {
		if !strings.EqualFold(part, collection) {
			continue
		}
		if i+1 == len(parts) && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			return "List" + listName
		}
		if i+2 == len(parts) {
			switch r.Method {
			case http.MethodPut:
				return "CreateOrUpdate" + resourceName
			case http.MethodGet, http.MethodHead:
				return "Get" + resourceName
			case http.MethodDelete:
				return "Delete" + resourceName
			}
		}
		continue
	}
	return ""
}

func detectAzureServiceBusControlPlaneAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") ||
			!strings.EqualFold(parts[i+1], "Microsoft.ServiceBus") ||
			!strings.EqualFold(parts[i+2], "namespaces") {
			continue
		}

		isNamespaceCollection := i+3 == len(parts)
		isNamespaceResource := i+4 == len(parts)
		isQueueCollection := i+5 == len(parts) && strings.EqualFold(parts[i+4], "queues")
		isQueueResource := i+6 == len(parts) && strings.EqualFold(parts[i+4], "queues")
		isTopicCollection := i+5 == len(parts) && strings.EqualFold(parts[i+4], "topics")
		isTopicResource := i+6 == len(parts) && strings.EqualFold(parts[i+4], "topics")
		isSubscriptionCollection := i+7 == len(parts) && strings.EqualFold(parts[i+4], "topics") && strings.EqualFold(parts[i+6], "subscriptions")
		isSubscriptionResource := i+8 == len(parts) && strings.EqualFold(parts[i+4], "topics") && strings.EqualFold(parts[i+6], "subscriptions")
		isRuleCollection := i+9 == len(parts) && strings.EqualFold(parts[i+4], "topics") && strings.EqualFold(parts[i+6], "subscriptions") && strings.EqualFold(parts[i+8], "rules")
		isRuleResource := i+10 == len(parts) && strings.EqualFold(parts[i+4], "topics") && strings.EqualFold(parts[i+6], "subscriptions") && strings.EqualFold(parts[i+8], "rules")

		switch r.Method {
		case http.MethodPut:
			if isRuleResource {
				return "CreateOrUpdateRule"
			}
			if isSubscriptionResource {
				return "CreateOrUpdateSubscription"
			}
			if isTopicResource {
				return "CreateOrUpdateTopic"
			}
			if isQueueResource {
				return "CreateOrUpdateQueue"
			}
			if isNamespaceResource {
				return "CreateOrUpdateNamespace"
			}
		case http.MethodGet, http.MethodHead:
			if isRuleCollection {
				return "ListRules"
			}
			if isRuleResource {
				return "GetRule"
			}
			if isSubscriptionCollection {
				return "ListSubscriptions"
			}
			if isSubscriptionResource {
				return "GetSubscription"
			}
			if isTopicCollection {
				return "ListTopics"
			}
			if isTopicResource {
				return "GetTopic"
			}
			if isQueueCollection {
				return "ListQueues"
			}
			if isQueueResource {
				return "GetQueue"
			}
			if isNamespaceCollection {
				return "ListNamespaces"
			}
			if isNamespaceResource {
				return "GetNamespace"
			}
		case http.MethodDelete:
			if isRuleResource {
				return "DeleteRule"
			}
			if isSubscriptionResource {
				return "DeleteSubscription"
			}
			if isTopicResource {
				return "DeleteTopic"
			}
			if isQueueResource {
				return "DeleteQueue"
			}
			if isNamespaceResource {
				return "DeleteNamespace"
			}
		}
	}
	return ""
}

func detectAzureServiceBusRuntimeAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) < 2 {
		return ""
	}
	if strings.EqualFold(parts[len(parts)-1], "messages") && r.Method == http.MethodPost {
		if isAzureServiceBusBatchContentType(r) {
			return "SendMessageBatch"
		}
		return "SendMessage"
	}
	if len(parts) >= 3 && strings.EqualFold(parts[len(parts)-2], "messages") && strings.EqualFold(parts[len(parts)-1], "head") {
		switch r.Method {
		case http.MethodPost:
			return "PeekLockMessage"
		case http.MethodDelete:
			return "ReceiveAndDeleteMessage"
		}
	}
	if len(parts) >= 4 && strings.EqualFold(parts[len(parts)-3], "messages") {
		switch r.Method {
		case http.MethodDelete:
			return "CompleteMessage"
		case http.MethodPut:
			return "UnlockMessage"
		case http.MethodPost:
			return "RenewLockMessage"
		}
	}
	return ""
}

func detectAzureEventHubRuntimeAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) < 2 {
		return ""
	}
	if strings.EqualFold(parts[len(parts)-1], "messages") && r.Method == http.MethodPost {
		return "SendEvent"
	}
	return ""
}

func isAzureEventHubRuntimeDataPlaneRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	return isAzureServiceBusDataPlaneHost(normalizedHost(r)) &&
		strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("api-version")), "2014-01")
}

func isAzureServiceBusBatchContentType(r *http.Request) bool {
	if r == nil {
		return false
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if semicolon := strings.IndexByte(contentType, ';'); semicolon >= 0 {
		contentType = contentType[:semicolon]
	}
	return strings.TrimSpace(contentType) == "application/vnd.microsoft.servicebus.json"
}

func detectAzureCosmosDBAction(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-cosmos") {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		if r.Method == http.MethodGet {
			return "GetAccountInfo"
		}
		return ""
	}
	if !strings.EqualFold(parts[0], "dbs") {
		return ""
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return "ListDatabases"
		case http.MethodPost:
			return "CreateDatabase"
		}
		return ""
	}
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			return "GetDatabase"
		case http.MethodDelete:
			return "DeleteDatabase"
		}
		return ""
	}
	if len(parts) >= 3 && strings.EqualFold(parts[2], "colls") {
		if len(parts) == 3 {
			switch r.Method {
			case http.MethodGet:
				return "ListCollections"
			case http.MethodPost:
				return "CreateCollection"
			}
			return ""
		}
		if len(parts) == 4 {
			switch r.Method {
			case http.MethodGet:
				return "GetCollection"
			case http.MethodDelete:
				return "DeleteCollection"
			}
			return ""
		}
		if len(parts) == 5 && strings.EqualFold(parts[4], "pkranges") {
			if r.Method == http.MethodGet {
				return "ListPartitionKeyRanges"
			}
			return ""
		}
		if len(parts) >= 5 && strings.EqualFold(parts[4], "docs") {
			batch := strings.EqualFold(r.Header.Get("x-ms-cosmos-is-batch-request"), "true")
			query := strings.EqualFold(r.Header.Get("x-ms-documentdb-isquery"), "true") || strings.EqualFold(r.Header.Get("Content-Type"), "application/query+json")
			queryPlan := strings.EqualFold(r.Header.Get("x-ms-cosmos-is-query-plan-request"), "true")
			if len(parts) == 5 {
				switch r.Method {
				case http.MethodGet:
					return "ListDocuments"
				case http.MethodPost:
					if batch {
						return "ExecuteTransactionalBatch"
					}
					if queryPlan {
						return "GetQueryPlan"
					}
					if query {
						return "QueryDocuments"
					}
					return "CreateDocument"
				}
				return ""
			}
			if len(parts) == 6 {
				switch r.Method {
				case http.MethodGet:
					return "ReadDocument"
				case http.MethodPut:
					return "ReplaceDocument"
				case http.MethodPatch:
					return "PatchDocument"
				case http.MethodDelete:
					return "DeleteDocument"
				}
			}
		}
	}
	return ""
}

func azureStorageDataPlaneKind(host string) string {
	switch {
	case hasAzureStorageHostSuffix(host, "blob"):
		return "blob"
	case hasAzureStorageHostSuffix(host, "queue"):
		return "queue"
	case hasAzureStorageHostSuffix(host, "table"):
		return "table"
	case hasAzureStorageHostSuffix(host, "file"):
		return "file"
	default:
		return ""
	}
}

func isAzureStorageLocalDataPlaneRequest(r *http.Request) bool {
	return azureStorageLocalDataPlaneKind(r) != ""
}

func azureStorageLocalDataPlaneKind(r *http.Request) string {
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) == 0 || r.Header.Get("x-ms-version") == "" {
		return ""
	}
	first := strings.ToLower(parts[0])
	if strings.HasSuffix(first, "-queue") {
		return "queue"
	}
	if strings.HasSuffix(first, "-table") {
		return "table"
	}
	if strings.HasSuffix(first, "-file") {
		return "file"
	}
	if isAzureBlobLocalDataPlaneRequest(r, parts, first) {
		return "blob"
	}
	return ""
}

func isAzureBlobLocalDataPlaneRequest(r *http.Request, parts []string, first string) bool {
	if !isLocalStorageHost(normalizedHost(r)) || len(parts) == 0 {
		return false
	}
	if strings.HasSuffix(first, "-blob") {
		return true
	}
	if !strings.EqualFold(first, "devstoreaccount1") {
		return false
	}
	return len(parts) >= 2 || strings.EqualFold(r.URL.Query().Get("comp"), "list")
}

func isLocalStorageHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1":
		return true
	default:
		return false
	}
}

func hasAzureStorageHostSuffix(host, service string) bool {
	suffixes := []string{
		"." + service + ".core.windows.net",
		"." + service + ".core.usgovcloudapi.net",
		"." + service + ".core.chinacloudapi.cn",
		"." + service + ".core.cloudapi.de",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	parts := raw[:0]
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
