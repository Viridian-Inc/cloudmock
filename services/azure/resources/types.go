package resources

// ResourceGroup is the Azure Resource Manager resource group shape.
type ResourceGroup struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Type       string                  `json:"type"`
	Location   string                  `json:"location"`
	ManagedBy  string                  `json:"managedBy,omitempty"`
	Tags       map[string]string       `json:"tags,omitempty"`
	Properties ResourceGroupProperties `json:"properties"`
}

type ResourceGroupProperties struct {
	ProvisioningState string `json:"provisioningState"`
}

// TagsResource is the wrapper shape returned by the Azure tags-at-scope APIs.
type TagsResource struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties TagsProperties `json:"properties"`
}

type TagsProperties struct {
	Tags map[string]string `json:"tags"`
}

// GenericResource is the shared ARM resource shape returned by the generic
// Microsoft.Resources resource APIs.
type GenericResource struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	Location          string            `json:"location,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	Kind              string            `json:"kind,omitempty"`
	ManagedBy         string            `json:"managedBy,omitempty"`
	ExtendedLocation  map[string]any    `json:"extendedLocation,omitempty"`
	Identity          map[string]any    `json:"identity,omitempty"`
	Plan              map[string]any    `json:"plan,omitempty"`
	SKU               map[string]any    `json:"sku,omitempty"`
	Properties        map[string]any    `json:"properties,omitempty"`
	CreatedTime       string            `json:"createdTime,omitempty"`
	ChangedTime       string            `json:"changedTime,omitempty"`
	ProvisioningState string            `json:"provisioningState,omitempty"`
	createdTime       string
	changedTime       string
}

// TemplateProvisioner creates provider resources from resolved ARM template resources.
type TemplateProvisioner interface {
	SupportsTemplateResource(resource map[string]any) bool
	ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error)
}

// TemplateListOperationProvider optionally exposes ARM list* function results
// backed by a provisioner's runtime state.
type TemplateListOperationProvider interface {
	TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool)
}

// Deployment is the stored Azure Resource Manager deployment shape.
type Deployment struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Type       string               `json:"type"`
	Properties DeploymentProperties `json:"properties"`
}

type DeploymentProperties struct {
	ProvisioningState string                      `json:"provisioningState"`
	Mode              string                      `json:"mode"`
	Timestamp         string                      `json:"timestamp"`
	Parameters        map[string]any              `json:"parameters,omitempty"`
	Outputs           map[string]DeploymentOutput `json:"outputs,omitempty"`
	OutputResources   []any                       `json:"outputResources,omitempty"`
}

type DeploymentOutput struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// ProviderManifest describes an Azure resource provider namespace.
type ProviderManifest struct {
	ID                 string                 `json:"id"`
	Namespace          string                 `json:"namespace"`
	RegistrationPolicy string                 `json:"registrationPolicy"`
	RegistrationState  string                 `json:"registrationState"`
	ResourceTypes      []ProviderResourceType `json:"resourceTypes"`
}

type ProviderResourceType struct {
	ResourceType string   `json:"resourceType"`
	Locations    []string `json:"locations,omitempty"`
	APIVersions  []string `json:"apiVersions"`
	Capabilities string   `json:"capabilities,omitempty"`
}

func defaultProviders() map[string]ProviderManifest {
	providers := []ProviderManifest{
		{
			Namespace: "Microsoft.Resources",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "resourceGroups", Locations: []string{"global"}, APIVersions: []string{apiVersion20210401}, Capabilities: "None"},
				{ResourceType: "deployments", Locations: []string{"global"}, APIVersions: []string{"2021-04-01"}, Capabilities: "None"},
				{ResourceType: "resources", Locations: []string{"global"}, APIVersions: []string{"2021-04-01"}, Capabilities: "None"},
				{ResourceType: "tags", Locations: []string{"global"}, APIVersions: []string{"2021-04-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Authorization",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "locks", Locations: []string{"global"}, APIVersions: []string{"2020-05-01"}, Capabilities: "None"},
				{ResourceType: "roleAssignments", Locations: []string{"global"}, APIVersions: []string{"2022-04-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Storage",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "storageAccounts", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01", "2023-01-01", "2022-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "storageAccounts/blobServices", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01", "2023-01-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.KeyVault",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "vaults", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-11-01", "2023-07-01", "2022-07-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.Compute",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "virtualMachines", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-11-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "disks", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-01-02"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.Web",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "serverfarms", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-04-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "sites", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-04-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "sites/slots", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-04-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.Network",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "virtualNetworks", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "virtualNetworks/subnets", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "None"},
				{ResourceType: "networkSecurityGroups", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "networkSecurityGroups/securityRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "None"},
				{ResourceType: "publicIPAddresses", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "networkInterfaces", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "loadBalancers", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "applicationGateways", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "privateEndpoints", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "privateEndpoints/privateDnsZoneGroups", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2023-09-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "dnsZones/A", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/AAAA", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/CAA", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/CNAME", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/MX", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/NS", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/PTR", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/SOA", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/SRV", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
				{ResourceType: "dnsZones/TXT", Locations: []string{"global"}, APIVersions: []string{"2018-05-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Insights",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "actionGroups", Locations: []string{"global"}, APIVersions: []string{"2021-09-01", "2019-06-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "metricAlerts", Locations: []string{"global"}, APIVersions: []string{"2024-03-01-preview", "2018-03-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "diagnosticSettings", Locations: []string{"global"}, APIVersions: []string{"2021-05-01-preview"}, Capabilities: "None"},
				{ResourceType: "metrics", Locations: []string{"global"}, APIVersions: []string{"2023-10-01"}, Capabilities: "None"},
				{ResourceType: "metricDefinitions", Locations: []string{"global"}, APIVersions: []string{"2023-10-01"}, Capabilities: "None"},
				{ResourceType: "eventtypes", Locations: []string{"global"}, APIVersions: []string{"2015-04-01"}, Capabilities: "None"},
				{ResourceType: "components", Locations: []string{"eastus", "westus2", "global"}, APIVersions: []string{"2015-05-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.OperationalInsights",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "workspaces", Locations: []string{"eastus", "westus2", "global"}, APIVersions: []string{"2025-02-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.ServiceBus",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "namespaces", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "namespaces/authorizationRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/queues", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/queues/authorizationRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/topics", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/topics/authorizationRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/topics/subscriptions", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/topics/subscriptions/rules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-01-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.EventGrid",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "topics", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-02-15"}, Capabilities: "SupportsTags"},
				{ResourceType: "topics/eventSubscriptions", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-02-15"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.EventHub",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "namespaces", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-01-01", "2024-01-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "namespaces/authorizationRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-01-01", "2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/eventhubs", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-01-01", "2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/eventhubs/authorizationRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-01-01", "2024-01-01"}, Capabilities: "None"},
				{ResourceType: "namespaces/eventhubs/consumergroups", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-01-01", "2024-01-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Cdn",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "profiles", Locations: []string{"global"}, APIVersions: []string{"2025-04-15"}, Capabilities: "SupportsTags"},
				{ResourceType: "profiles/endpoints", Locations: []string{"global"}, APIVersions: []string{"2025-04-15"}, Capabilities: "SupportsTags"},
				{ResourceType: "profiles/endpoints/originGroups", Locations: []string{"global"}, APIVersions: []string{"2025-04-15"}, Capabilities: "None"},
				{ResourceType: "profiles/endpoints/origins", Locations: []string{"global"}, APIVersions: []string{"2025-04-15"}, Capabilities: "None"},
				{ResourceType: "profiles/endpoints/customDomains", Locations: []string{"global"}, APIVersions: []string{"2025-04-15"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.ContainerRegistry",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "registries", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-11-01", "2023-07-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.ContainerInstance",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "containerGroups", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-09-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.ContainerService",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "managedClusters", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-02-01", "2026-03-01", "2026-04-01", "2024-04-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "locations/kubernetesVersions", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2026-03-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.App",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "managedEnvironments", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-07-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "containerApps", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-07-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.DocumentDB",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "databaseAccounts", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2024-05-15"}, Capabilities: "SupportsTags"},
				{ResourceType: "databaseAccounts/sqlDatabases", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2024-05-15"}, Capabilities: "None"},
				{ResourceType: "databaseAccounts/sqlDatabases/containers", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-05-01", "2024-05-15"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Sql",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "servers", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-01-01", "2023-08-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "servers/databases", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-01-01", "2023-08-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "servers/firewallRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-01-01", "2023-08-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.DBforPostgreSQL",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "flexibleServers", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-08-01", "2024-08-01"}, Capabilities: "SupportsTags"},
				{ResourceType: "flexibleServers/databases", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-08-01", "2024-08-01"}, Capabilities: "None"},
				{ResourceType: "flexibleServers/firewallRules", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2025-08-01", "2024-08-01"}, Capabilities: "None"},
			},
		},
		{
			Namespace: "Microsoft.Cache",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "Redis", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-11-01", "2023-08-01"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.ManagedIdentity",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "userAssignedIdentities", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2023-01-31", "2018-11-30"}, Capabilities: "SupportsTags"},
			},
		},
		{
			Namespace: "Microsoft.AppConfiguration",
			ResourceTypes: []ProviderResourceType{
				{ResourceType: "configurationStores", Locations: []string{"eastus", "westus2"}, APIVersions: []string{"2024-06-01"}, Capabilities: "SupportsTags"},
			},
		},
	}

	out := make(map[string]ProviderManifest, len(providers))
	for _, provider := range providers {
		out[normalizeProviderNamespace(provider.Namespace)] = provider
	}
	return out
}

func normalizeProviderNamespace(namespace string) string {
	for _, provider := range []string{"Microsoft.Resources", "Microsoft.Authorization", "Microsoft.Storage", "Microsoft.KeyVault", "Microsoft.Compute", "Microsoft.Web", "Microsoft.Network", "Microsoft.Insights", "Microsoft.OperationalInsights", "Microsoft.ServiceBus", "Microsoft.EventGrid", "Microsoft.EventHub", "Microsoft.Cdn", "Microsoft.ContainerRegistry", "Microsoft.ContainerInstance", "Microsoft.ContainerService", "Microsoft.App", "Microsoft.DocumentDB", "Microsoft.Sql", "Microsoft.DBforPostgreSQL", "Microsoft.Cache", "Microsoft.ManagedIdentity", "Microsoft.AppConfiguration"} {
		if len(namespace) == len(provider) {
			match := true
			for i := range namespace {
				a := namespace[i]
				b := provider[i]
				if a >= 'A' && a <= 'Z' {
					a += 'a' - 'A'
				}
				if b >= 'A' && b <= 'Z' {
					b += 'a' - 'A'
				}
				if a != b {
					match = false
					break
				}
			}
			if match {
				return providerNamespaceKey(provider)
			}
		}
	}
	return providerNamespaceKey(namespace)
}

func providerNamespaceKey(namespace string) string {
	out := make([]byte, len(namespace))
	for i := range namespace {
		c := namespace[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
