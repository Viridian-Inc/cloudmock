package containerservice

// ManagedCluster is the stored ARM model for Microsoft.ContainerService/managedClusters.
type ManagedCluster struct {
	ID                    string             `json:"id"`
	Name                  string             `json:"name"`
	Type                  string             `json:"type"`
	Location              string             `json:"location"`
	Tags                  map[string]string  `json:"tags,omitempty"`
	KubernetesVersion     string             `json:"-"`
	DNSPrefix             string             `json:"-"`
	FQDN                  string             `json:"-"`
	Endpoint              string             `json:"-"`
	Kubeconfig            string             `json:"-"`
	ProvisioningState     string             `json:"-"`
	PowerState            string             `json:"-"`
	AgentPoolProfiles     []AgentPoolProfile `json:"-"`
	SubscriptionID        string             `json:"-"`
	ResourceGroup         string             `json:"-"`
	NodeResourceGroupName string             `json:"-"`
}

// AgentPoolProfile is the stored ARM model for AKS agent pools.
type AgentPoolProfile struct {
	Name              string `json:"name"`
	Count             int    `json:"count"`
	VMSize            string `json:"vmSize"`
	OSType            string `json:"osType"`
	Mode              string `json:"mode"`
	NodeImageVersion  string `json:"nodeImageVersion"`
	ProvisioningState string `json:"provisioningState"`
}

// CommandResult is the stored ARM model for AKS runCommand results.
type CommandResult struct {
	ID                string
	SubscriptionID    string
	ResourceGroup     string
	ClusterName       string
	Command           string
	ExitCode          int
	Logs              string
	ProvisioningState string
	Reason            string
	StartedAt         string
	FinishedAt        string
}
