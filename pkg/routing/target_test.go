package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectTarget_AWSIncludesProviderServiceActionAndAPIVersion(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20260101/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc")
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAWS, target.Provider)
	assert.Equal(t, "dynamodb", target.Service)
	assert.Equal(t, "PutItem", target.Action)
	assert.Equal(t, "20120810", target.APIVersion)
}

func TestDetectTarget_AzureResourceGroupCreateOrUpdate(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourcegroups/rg-a?api-version=2021-04-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Resources/resourceGroups", target.Service)
	assert.Equal(t, "CreateOrUpdate", target.Action)
	assert.Equal(t, "2021-04-01", target.APIVersion)
}

func TestDetectTarget_AzureProviderResourceCreateOrUpdate(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct?api-version=2023-01-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Storage/storageAccounts", target.Service)
	assert.Equal(t, "CreateOrUpdate", target.Action)
	assert.Equal(t, "2023-01-01", target.APIVersion)
}

func TestDetectTarget_AzureCDNProfileActions(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string
		action      string
	}{
		{
			name:   "create or update Front Door profile",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15",
			action: "CreateOrUpdate",
		},
		{
			name:   "get Front Door profile",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15",
			action: "Get",
		},
		{
			name:   "list Front Door profiles by resource group",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles?api-version=2025-04-15",
			action: "List",
		},
		{
			name:   "list Front Door profiles by subscription",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Cdn/profiles?api-version=2025-04-15",
			action: "List",
		},
		{
			name:   "delete Front Door profile",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15",
			action: "Delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCDNEndpointActions(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string
		action      string
	}{
		{
			name:   "create or update CDN endpoint",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15",
			action: "CreateOrUpdateEndpoint",
		},
		{
			name:   "get CDN endpoint",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15",
			action: "GetEndpoint",
		},
		{
			name:   "list CDN endpoints under profile",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints?api-version=2025-04-15",
			action: "ListEndpoints",
		},
		{
			name:   "delete CDN endpoint",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15",
			action: "DeleteEndpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCDNOriginGroupActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update CDN origin group",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-a?api-version=2025-04-15",
			action: "CreateOrUpdateOriginGroup",
		},
		{
			name:   "get CDN origin group",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-a?api-version=2025-04-15",
			action: "GetOriginGroup",
		},
		{
			name:   "list CDN origin groups under endpoint",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups?api-version=2025-04-15",
			action: "ListOriginGroups",
		},
		{
			name:   "delete CDN origin group",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-a?api-version=2025-04-15",
			action: "DeleteOriginGroup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCDNOriginActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update CDN origin",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-a?api-version=2025-04-15",
			action: "CreateOrUpdateOrigin",
		},
		{
			name:   "get CDN origin",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-a?api-version=2025-04-15",
			action: "GetOrigin",
		},
		{
			name:   "list CDN origins under endpoint",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins?api-version=2025-04-15",
			action: "ListOrigins",
		},
		{
			name:   "delete CDN origin",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-a?api-version=2025-04-15",
			action: "DeleteOrigin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCDNCustomDomainActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update CDN custom domain",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com?api-version=2025-04-15",
			action: "CreateOrUpdateCustomDomain",
		},
		{
			name:   "get CDN custom domain",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com?api-version=2025-04-15",
			action: "GetCustomDomain",
		},
		{
			name:   "list CDN custom domains under endpoint",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains?api-version=2025-04-15",
			action: "ListCustomDomains",
		},
		{
			name:   "delete CDN custom domain",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com?api-version=2025-04-15",
			action: "DeleteCustomDomain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCDNCustomDomainHTTPSActions(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		action string
	}{
		{
			name:   "enable CDN custom HTTPS",
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com/enableCustomHttps?api-version=2025-04-15",
			action: "EnableCustomHttps",
		},
		{
			name:   "disable CDN custom HTTPS",
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com/disableCustomHttps?api-version=2025-04-15",
			action: "DisableCustomHttps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Cdn/profiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-04-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureStorageAccountListCollection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts?api-version=2024-01-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Storage/storageAccounts", target.Service)
	assert.Equal(t, "List", target.Action)
	assert.Equal(t, "2024-01-01", target.APIVersion)
}

func TestDetectTarget_AzureActivityLogsList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights/eventtypes/management/values?api-version=2015-04-01&$filter=eventTimestamp%20ge%20'2026-06-16T00:00:00Z'", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Insights/eventtypes", target.Service)
	assert.Equal(t, "ListActivityLogs", target.Action)
	assert.Equal(t, "2015-04-01", target.APIVersion)
}

func TestDetectTarget_AzureLogAnalyticsWorkspaceActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update workspace",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourcegroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a?api-version=2025-02-01",
			action: "CreateOrUpdateWorkspace",
		},
		{
			name:   "get workspace",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a?api-version=2025-02-01",
			action: "GetWorkspace",
		},
		{
			name:   "list workspaces by resource group",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces?api-version=2025-02-01",
			action: "ListWorkspacesByResourceGroup",
		},
		{
			name:   "delete workspace",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a?api-version=2025-02-01",
			action: "DeleteWorkspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.OperationalInsights/workspaces", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-02-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureKeyVaultControlPlane(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults/vault-a?api-version=2024-11-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.KeyVault/vaults", target.Service)
	assert.Equal(t, "CreateOrUpdate", target.Action)
	assert.Equal(t, "2024-11-01", target.APIVersion)
}

func TestDetectTarget_AzureKeyVaultSecretDataPlane(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "set secret",
			method: http.MethodPut,
			url:    "https://vault-a.vault.azure.net/secrets/db-password?api-version=2025-07-01",
			action: "SetSecret",
		},
		{
			name:   "get secret version",
			method: http.MethodGet,
			url:    "https://vault-a.vault.azure.net/secrets/db-password/0123456789abcdef0123456789abcdef?api-version=2025-07-01",
			action: "GetSecret",
		},
		{
			name:   "list secret versions",
			method: http.MethodGet,
			url:    "https://vault-a.vault.azure.net/secrets/db-password/versions?api-version=2025-07-01",
			action: "ListSecretVersions",
		},
		{
			name:   "update secret properties",
			method: http.MethodPatch,
			url:    "https://vault-a.vault.azure.net/secrets/db-password/0123456789abcdef0123456789abcdef?api-version=2025-07-01",
			action: "UpdateSecret",
		},
		{
			name:   "backup secret",
			method: http.MethodPost,
			url:    "https://vault-a.vault.azure.net/secrets/db-password/backup?api-version=2025-07-01",
			action: "BackupSecret",
		},
		{
			name:   "list deleted secrets",
			method: http.MethodGet,
			url:    "https://vault-a.vault.azure.net/deletedsecrets?api-version=2025-07-01",
			action: "ListDeletedSecrets",
		},
		{
			name:   "get deleted secret",
			method: http.MethodGet,
			url:    "https://vault-a.vault.azure.net/deletedsecrets/db-password?api-version=2025-07-01",
			action: "GetDeletedSecret",
		},
		{
			name:   "recover deleted secret",
			method: http.MethodPost,
			url:    "https://vault-a.vault.azure.net/deletedsecrets/db-password/recover?api-version=2025-07-01",
			action: "RecoverDeletedSecret",
		},
		{
			name:   "purge deleted secret",
			method: http.MethodDelete,
			url:    "https://vault-a.vault.azure.net/deletedsecrets/db-password?api-version=2025-07-01",
			action: "PurgeDeletedSecret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.KeyVault/secrets", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-07-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureKeyVaultKeyDataPlane(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create key",
			method: http.MethodPost,
			url:    "https://vault-a.vault.azure.net/keys/signing-key/create?api-version=2025-07-01",
			action: "CreateKey",
		},
		{
			name:   "encrypt",
			method: http.MethodPost,
			url:    "https://vault-a.vault.azure.net/keys/signing-key/version-1/encrypt?api-version=2025-07-01",
			action: "Encrypt",
		},
		{
			name:   "decrypt",
			method: http.MethodPost,
			url:    "https://vault-a.vault.azure.net/keys/signing-key/version-1/decrypt?api-version=2025-07-01",
			action: "Decrypt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.KeyVault/keys", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-07-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureStorageDataPlaneHosts(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		service string
	}{
		{
			name:    "blob host",
			url:     "https://acct.blob.core.windows.net/container/blob.txt",
			service: "Microsoft.Storage/blobServices",
		},
		{
			name:    "queue host",
			url:     "https://acct.queue.core.windows.net/work/messages",
			service: "Microsoft.Storage/queueServices",
		},
		{
			name:    "table host",
			url:     "https://acct.table.core.windows.net/Tables",
			service: "Microsoft.Storage/tableServices",
		},
		{
			name:    "file host",
			url:     "https://acct.file.core.windows.net/reports?restype=share",
			service: "Microsoft.Storage/fileServices",
		},
		{
			name:    "floci-style table local path",
			url:     "http://localhost:4577/devstoreaccount1-table/Tables",
			service: "Microsoft.Storage/tableServices",
		},
		{
			name:    "floci-style file local path",
			url:     "http://localhost:4577/devstoreaccount1-file/reports?restype=share",
			service: "Microsoft.Storage/fileServices",
		},
		{
			name:    "floci-style queue local path",
			url:     "http://localhost:4577/devstoreaccount1-queue/work",
			service: "Microsoft.Storage/queueServices",
		},
		{
			name:    "floci-style blob local path",
			url:     "http://localhost:4577/devstoreaccount1/docs?restype=container",
			service: "Microsoft.Storage/blobServices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, tt.url, nil)
			req.Header.Set("x-ms-version", "2023-11-03")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, tt.service, target.Service)
			assert.Equal(t, "CreateOrUpdate", target.Action)
			assert.Equal(t, "2023-11-03", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureTableDataPlaneMerge(t *testing.T) {
	req := httptest.NewRequest("MERGE", "http://localhost:4577/devstoreaccount1-table/Tasks(PartitionKey='p1',RowKey='r1')", nil)
	req.Header.Set("x-ms-version", "2023-11-03")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Storage/tableServices", target.Service)
	assert.Equal(t, "Merge", target.Action)
	assert.Equal(t, "2023-11-03", target.APIVersion)
}

func TestDetectTarget_AzureStorageLocalBlobListContainers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:4577/devstoreaccount1?comp=list", nil)
	req.Header.Set("x-ms-version", "2023-11-03")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Storage/blobServices", target.Service)
	assert.Equal(t, "List", target.Action)
	assert.Equal(t, "2023-11-03", target.APIVersion)
}

func TestDetectTarget_AzureAppConfigurationDataPlane(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "get key-value from Azure host",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/kv/app:message?api-version=2024-09-01&label=prod",
			action: "GetKeyValue",
		},
		{
			name:   "list key-values from floci-style local path",
			method: http.MethodGet,
			url:    "http://localhost:4577/devstoreaccount1-appconfig/kv?api-version=2024-09-01&key=app:*",
			action: "ListKeyValues",
		},
		{
			name:   "set key-value",
			method: http.MethodPut,
			url:    "https://cfgstore.azconfig.io/kv/app:message?api-version=2024-09-01",
			action: "SetKeyValue",
		},
		{
			name:   "delete key-value",
			method: http.MethodDelete,
			url:    "https://cfgstore.azconfig.io/kv/app:message?api-version=2024-09-01",
			action: "DeleteKeyValue",
		},
		{
			name:   "list keys",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&name=app:*",
			action: "ListKeys",
		},
		{
			name:   "list labels",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/labels?api-version=2024-09-01&name=prod*",
			action: "ListLabels",
		},
		{
			name:   "lock key-value",
			method: http.MethodPut,
			url:    "https://cfgstore.azconfig.io/locks/app:message?api-version=2024-09-01&label=prod",
			action: "LockKeyValue",
		},
		{
			name:   "list revisions",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:*",
			action: "ListRevisions",
		},
		{
			name:   "create snapshot",
			method: http.MethodPut,
			url:    "https://cfgstore.azconfig.io/snapshots/snap-a?api-version=2024-09-01",
			action: "CreateSnapshot",
		},
		{
			name:   "list snapshots",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&name=snap*",
			action: "ListSnapshots",
		},
		{
			name:   "poll snapshot operation",
			method: http.MethodGet,
			url:    "https://cfgstore.azconfig.io/operations?snapshot=snap-a&api-version=2024-09-01",
			action: "GetSnapshotOperation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.AppConfiguration/kv", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-09-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureCosmosDBDataPlane(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create database from Azure host",
			method: http.MethodPost,
			url:    "https://acct.documents.azure.com/dbs",
			action: "CreateDatabase",
		},
		{
			name:   "query documents from Azure host",
			method: http.MethodPost,
			url:    "https://acct.documents.azure.com/dbs/app/colls/items/docs",
			action: "QueryDocuments",
		},
		{
			name:   "list partition key ranges from Azure host",
			method: http.MethodGet,
			url:    "https://acct.documents.azure.com/dbs/app/colls/items/pkranges",
			action: "ListPartitionKeyRanges",
		},
		{
			name:   "query plan from Azure host",
			method: http.MethodPost,
			url:    "https://acct.documents.azure.com/dbs/app/colls/items/docs",
			action: "GetQueryPlan",
		},
		{
			name:   "transactional batch from Azure host",
			method: http.MethodPost,
			url:    "https://acct.documents.azure.com/dbs/app/colls/items/docs",
			action: "ExecuteTransactionalBatch",
		},
		{
			name:   "read document from floci-style local path",
			method: http.MethodGet,
			url:    "http://localhost:4577/devstoreaccount1-cosmos/dbs/app/colls/items/docs/item-1",
			action: "ReadDocument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "type=master&ver=1.0&sig=fake")
			req.Header.Set("x-ms-version", "2018-12-31")
			if tt.action == "QueryDocuments" {
				req.Header.Set("x-ms-documentdb-isquery", "True")
			}
			if tt.action == "GetQueryPlan" {
				req.Header.Set("x-ms-cosmos-is-query-plan-request", "True")
			}
			if tt.action == "ExecuteTransactionalBatch" {
				req.Header.Set("x-ms-cosmos-is-batch-request", "True")
			}

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.DocumentDB/sqlApi", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2018-12-31", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureServiceBusRuntimeDataPlane(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string
		action      string
	}{
		{
			name:   "send queue message",
			method: http.MethodPost,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages",
			action: "SendMessage",
		},
		{
			name:        "send queue message batch",
			method:      http.MethodPost,
			url:         "https://ns-a.servicebus.windows.net/queue-a/messages",
			contentType: "application/vnd.microsoft.servicebus.json",
			action:      "SendMessageBatch",
		},
		{
			name:   "peek-lock queue message",
			method: http.MethodPost,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60",
			action: "PeekLockMessage",
		},
		{
			name:   "receive-delete queue message",
			method: http.MethodDelete,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60",
			action: "ReceiveAndDeleteMessage",
		},
		{
			name:   "complete locked queue message",
			method: http.MethodDelete,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages/msg-1/lock-1",
			action: "CompleteMessage",
		},
		{
			name:   "unlock locked queue message",
			method: http.MethodPut,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages/msg-1/lock-1",
			action: "UnlockMessage",
		},
		{
			name:   "renew locked queue message",
			method: http.MethodPost,
			url:    "https://ns-a.servicebus.windows.net/queue-a/messages/msg-1/lock-1",
			action: "RenewLockMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ServiceBus/runtime", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Empty(t, target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureServiceBusControlPlaneActions(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		contentType string
		action      string
	}{
		{
			name:   "create or update namespace",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01",
			action: "CreateOrUpdateNamespace",
		},
		{
			name:   "list queues",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues?api-version=2024-01-01",
			action: "ListQueues",
		},
		{
			name:   "get topic",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a?api-version=2024-01-01",
			action: "GetTopic",
		},
		{
			name:   "create subscription",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a?api-version=2024-01-01",
			action: "CreateOrUpdateSubscription",
		},
		{
			name:   "create rule",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/rule-a?api-version=2024-01-01",
			action: "CreateOrUpdateRule",
		},
		{
			name:   "get rule",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/rule-a?api-version=2024-01-01",
			action: "GetRule",
		},
		{
			name:   "list rules",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules?api-version=2024-01-01",
			action: "ListRules",
		},
		{
			name:   "delete rule",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/rule-a?api-version=2024-01-01",
			action: "DeleteRule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ServiceBus/namespaces", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-01-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureEventGridPublishDataPlane(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", nil)
	req.Header.Set("aeg-sas-key", "local-key")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.EventGrid/publish", target.Service)
	assert.Equal(t, "PublishEvents", target.Action)
	assert.Equal(t, "2018-01-01", target.APIVersion)
}

func TestDetectTarget_AzureEventGridPublishRequiresAPIVersion(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events", nil)
	req.Header.Set("aeg-sas-key", "local-key")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "", target.Service)
	assert.Equal(t, "", target.Action)
	assert.Equal(t, "", target.APIVersion)
}

func TestDetectTarget_AzureEventGridTopicKeyActions(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		action string
	}{
		{
			name:   "list keys",
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15",
			action: "ListSharedAccessKeys",
		},
		{
			name:   "regenerate key",
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/regenerateKey?api-version=2025-02-15",
			action: "RegenerateKey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.EventGrid/topics", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-02-15", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureEventGridListTopicsBySubscription(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.EventGrid/topics?api-version=2025-02-15", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.EventGrid/topics", target.Service)
	assert.Equal(t, "ListTopics", target.Action)
	assert.Equal(t, "2025-02-15", target.APIVersion)
}

func TestDetectTarget_AzureEventHubConsumerGroupActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create consumer group",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/consumergroups/cg-a?api-version=2026-01-01",
			action: "CreateOrUpdateConsumerGroup",
		},
		{
			name:   "get consumer group",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/consumergroups/cg-a?api-version=2026-01-01",
			action: "GetConsumerGroup",
		},
		{
			name:   "list consumer groups",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/consumergroups?api-version=2026-01-01",
			action: "ListConsumerGroups",
		},
		{
			name:   "delete consumer group",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/consumergroups/cg-a?api-version=2026-01-01",
			action: "DeleteConsumerGroup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.EventHub/namespaces", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2026-01-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureEventHubNamespaceAuthorizationRuleActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create authorization rule",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "CreateOrUpdateNamespaceAuthorizationRule",
		},
		{
			name:   "get authorization rule",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "GetNamespaceAuthorizationRule",
		},
		{
			name:   "list authorization rules",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules?api-version=2026-01-01",
			action: "ListNamespaceAuthorizationRules",
		},
		{
			name:   "delete authorization rule",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "DeleteNamespaceAuthorizationRule",
		},
		{
			name:   "list keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules/rule-a/listKeys?api-version=2026-01-01",
			action: "ListNamespaceKeys",
		},
		{
			name:   "regenerate keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/authorizationRules/rule-a/regenerateKeys?api-version=2026-01-01",
			action: "RegenerateNamespaceKeys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.EventHub/namespaces", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2026-01-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureEventHubAuthorizationRuleActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create event hub authorization rule",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "CreateOrUpdateEventHubAuthorizationRule",
		},
		{
			name:   "get event hub authorization rule",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "GetEventHubAuthorizationRule",
		},
		{
			name:   "list event hub authorization rules",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules?api-version=2026-01-01",
			action: "ListEventHubAuthorizationRules",
		},
		{
			name:   "delete event hub authorization rule",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules/rule-a?api-version=2026-01-01",
			action: "DeleteEventHubAuthorizationRule",
		},
		{
			name:   "list event hub keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules/rule-a/listKeys?api-version=2026-01-01",
			action: "ListEventHubKeys",
		},
		{
			name:   "regenerate event hub keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/ns-a/eventhubs/hub-a/authorizationRules/rule-a/regenerateKeys?api-version=2026-01-01",
			action: "RegenerateEventHubKeys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.EventHub/namespaces", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2026-01-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureEventHubRuntimeDataPlane(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://namespace-a.servicebus.windows.net/hub-a/messages?timeout=60&api-version=2014-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.EventHub/runtime", target.Service)
	assert.Equal(t, "SendEvent", target.Action)
	assert.Equal(t, "2014-01", target.APIVersion)
}

func TestDetectTarget_AzureApplicationInsightsComponentActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update component",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-a?api-version=2015-05-01",
			action: "CreateOrUpdateComponent",
		},
		{
			name:   "get component",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-a?api-version=2015-05-01",
			action: "GetComponent",
		},
		{
			name:   "list subscription components",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights/components?api-version=2015-05-01",
			action: "ListComponents",
		},
		{
			name:   "delete component",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-a?api-version=2015-05-01",
			action: "DeleteComponent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Insights/components", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2015-05-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureApplicationInsightsQueryDataPlane(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.applicationinsights.io/v1/apps/app-123/query?query=requests%20%7C%20take%2010&timespan=PT12H", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Insights/query", target.Service)
	assert.Equal(t, "QueryGet", target.Action)
	assert.Equal(t, "v1", target.APIVersion)
}

func TestDetectTarget_AzureLogAnalyticsQueryDataPlane(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "current host",
			url:  "https://api.loganalytics.azure.com/v1/workspaces/ws-123/query",
		},
		{
			name: "legacy host",
			url:  "https://api.loganalytics.io/v1/workspaces/ws-123/query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.OperationalInsights/query", target.Service)
			assert.Equal(t, "QueryWorkspace", target.Action)
			assert.Equal(t, "v1", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerRegistryActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "check name availability",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerRegistry/checkNameAvailability?api-version=2025-11-01",
			action: "CheckNameAvailability",
		},
		{
			name:   "list replications",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/replications?api-version=2025-11-01",
			action: "ListReplications",
		},
		{
			name:   "update registry",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra?api-version=2025-11-01",
			action: "UpdateRegistry",
		},
		{
			name:   "list subscription registries",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerRegistry/registries?api-version=2025-11-01",
			action: "ListRegistries",
		},
		{
			name:   "list usages",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/listUsages?api-version=2025-11-01",
			action: "ListUsages",
		},
		{
			name:   "import image",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/importImage?api-version=2025-11-01",
			action: "ImportImage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerRegistry/registries", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-11-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerRegistryDataPlaneActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "hosted registry ping",
			method: http.MethodGet,
			url:    "https://acra.azurecr.io/v2/",
			action: "RegistryPing",
		},
		{
			name:   "hosted catalog",
			method: http.MethodGet,
			url:    "https://acra.azurecr.io/v2/_catalog",
			action: "ListRepositories",
		},
		{
			name:   "hosted tag list",
			method: http.MethodGet,
			url:    "https://acra.azurecr.io/v2/library/alpine/tags/list",
			action: "ListTags",
		},
		{
			name:   "hosted put manifest",
			method: http.MethodPut,
			url:    "https://acra.azurecr.io/v2/library/alpine/manifests/latest",
			action: "PutManifest",
		},
		{
			name:   "hosted get manifest",
			method: http.MethodGet,
			url:    "https://acra.azurecr.io/v2/library/alpine/manifests/latest",
			action: "GetManifest",
		},
		{
			name:   "hosted delete manifest",
			method: http.MethodDelete,
			url:    "https://acra.azurecr.io/v2/library/alpine/manifests/latest",
			action: "DeleteManifest",
		},
		{
			name:   "hosted start blob upload",
			method: http.MethodPost,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/uploads/",
			action: "StartBlobUpload",
		},
		{
			name:   "hosted upload blob chunk",
			method: http.MethodPatch,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/uploads/upload-1",
			action: "UploadBlobChunk",
		},
		{
			name:   "hosted complete blob upload",
			method: http.MethodPut,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/uploads/upload-1?digest=sha256:abc123",
			action: "CompleteBlobUpload",
		},
		{
			name:   "hosted get blob",
			method: http.MethodGet,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/sha256:abc123",
			action: "GetBlob",
		},
		{
			name:   "hosted delete blob",
			method: http.MethodDelete,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/sha256:abc123",
			action: "DeleteBlob",
		},
		{
			name:   "hosted cancel blob upload",
			method: http.MethodDelete,
			url:    "https://acra.azurecr.io/v2/library/alpine/blobs/uploads/upload-1",
			action: "CancelBlobUpload",
		},
		{
			name:   "floci-style local manifest",
			method: http.MethodGet,
			url:    "http://localhost:4577/acra-acr/v2/library/alpine/manifests/latest",
			action: "GetManifest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerRegistry/registry", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerInstanceContainerGroupActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create container group",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a?api-version=2025-09-01",
			action: "CreateOrUpdateContainerGroup",
		},
		{
			name:   "get container group",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a?api-version=2025-09-01",
			action: "GetContainerGroup",
		},
		{
			name:   "list resource group container groups",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups?api-version=2025-09-01",
			action: "ListContainerGroups",
		},
		{
			name:   "list subscription container groups",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/containerGroups?api-version=2025-09-01",
			action: "ListContainerGroups",
		},
		{
			name:   "delete container group",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a?api-version=2025-09-01",
			action: "DeleteContainerGroup",
		},
		{
			name:   "start container group",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/start?api-version=2025-09-01",
			action: "StartContainerGroup",
		},
		{
			name:   "stop container group",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/stop?api-version=2025-09-01",
			action: "StopContainerGroup",
		},
		{
			name:   "restart container group",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/restart?api-version=2025-09-01",
			action: "RestartContainerGroup",
		},
		{
			name:   "get outbound network dependencies endpoints",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/outboundNetworkDependenciesEndpoints?api-version=2025-09-01",
			action: "GetOutboundNetworkDependenciesEndpoints",
		},
		{
			name:   "list container logs",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/containers/web/logs?api-version=2025-09-01",
			action: "ListContainerLogs",
		},
		{
			name:   "execute container command",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/containers/web/exec?api-version=2025-09-01",
			action: "ExecuteContainerCommand",
		},
		{
			name:   "attach container",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/containers/web/attach?api-version=2025-09-01",
			action: "AttachContainer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerInstance/containerGroups", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-09-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerInstanceContainerGroupProfileActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create container group profile",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a?api-version=2025-09-01",
			action: "CreateOrUpdateContainerGroupProfile",
		},
		{
			name:   "get container group profile",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a?api-version=2025-09-01",
			action: "GetContainerGroupProfile",
		},
		{
			name:   "patch container group profile tags",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a?api-version=2025-09-01",
			action: "UpdateContainerGroupProfile",
		},
		{
			name:   "delete container group profile",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a?api-version=2025-09-01",
			action: "DeleteContainerGroupProfile",
		},
		{
			name:   "list resource group container group profiles",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles?api-version=2025-09-01",
			action: "ListContainerGroupProfiles",
		},
		{
			name:   "list subscription container group profiles",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/containerGroupProfiles?api-version=2025-09-01",
			action: "ListContainerGroupProfiles",
		},
		{
			name:   "list all profile revisions",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a/revisions?api-version=2025-09-01",
			action: "ListContainerGroupProfileRevisions",
		},
		{
			name:   "get profile by revision number",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a/revisions/1?api-version=2025-09-01",
			action: "GetContainerGroupProfileRevision",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerInstance/containerGroupProfiles", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-09-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerInstanceLocationActions(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		action string
	}{
		{
			name:   "list cached images",
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westcentralus/cachedImages?api-version=2025-09-01",
			action: "ListCachedImages",
		},
		{
			name:   "list capabilities",
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westus/capabilities?api-version=2025-09-01",
			action: "ListCapabilities",
		},
		{
			name:   "list usage",
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westcentralus/usages?api-version=2025-09-01",
			action: "ListUsage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerInstance/locations", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-09-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureContainerInstanceOperationsList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://management.azure.com/providers/Microsoft.ContainerInstance/operations?api-version=2025-09-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.ContainerInstance/operations", target.Service)
	assert.Equal(t, "ListOperations", target.Action)
	assert.Equal(t, "2025-09-01", target.APIVersion)
}

func TestDetectTarget_AzureContainerAppsActions(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		url     string
		service string
		action  string
	}{
		{
			name:    "create managed environment",
			method:  http.MethodPut,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a?api-version=2025-07-01",
			service: "Microsoft.App/managedEnvironments",
			action:  "CreateOrUpdateManagedEnvironment",
		},
		{
			name:    "list managed environments",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments?api-version=2025-07-01",
			service: "Microsoft.App/managedEnvironments",
			action:  "ListManagedEnvironments",
		},
		{
			name:    "list managed environments by subscription",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.App/managedEnvironments?api-version=2025-07-01",
			service: "Microsoft.App/managedEnvironments",
			action:  "ListManagedEnvironments",
		},
		{
			name:    "delete managed environment",
			method:  http.MethodDelete,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a?api-version=2025-07-01",
			service: "Microsoft.App/managedEnvironments",
			action:  "DeleteManagedEnvironment",
		},
		{
			name:    "create container app",
			method:  http.MethodPut,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "CreateOrUpdateContainerApp",
		},
		{
			name:    "get container app",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "GetContainerApp",
		},
		{
			name:    "list container app secrets",
			method:  http.MethodPost,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/listSecrets?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "ListContainerAppSecrets",
		},
		{
			name:    "list container app revisions",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/revisions?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "ListContainerAppRevisions",
		},
		{
			name:    "get container app revision",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/revisions/app-a--000001?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "GetContainerAppRevision",
		},
		{
			name:    "activate container app revision",
			method:  http.MethodPost,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/revisions/app-a--000001/activate?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "ActivateContainerAppRevision",
		},
		{
			name:    "deactivate container app revision",
			method:  http.MethodPost,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/revisions/app-a--000001/deactivate?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "DeactivateContainerAppRevision",
		},
		{
			name:    "restart container app revision",
			method:  http.MethodPost,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a/revisions/app-a--000001/restart?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "RestartContainerAppRevision",
		},
		{
			name:    "list container apps",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "ListContainerApps",
		},
		{
			name:    "list container apps by subscription",
			method:  http.MethodGet,
			url:     "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.App/containerApps?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "ListContainerApps",
		},
		{
			name:    "delete container app",
			method:  http.MethodDelete,
			url:     "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a?api-version=2025-07-01",
			service: "Microsoft.App/containerApps",
			action:  "DeleteContainerApp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, tt.service, target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-07-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureComputeVirtualMachineActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create virtual machine",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01",
			action: "CreateOrUpdateVirtualMachine",
		},
		{
			name:   "get virtual machine",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01",
			action: "GetVirtualMachine",
		},
		{
			name:   "patch virtual machine",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01",
			action: "UpdateVirtualMachine",
		},
		{
			name:   "delete virtual machine",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01",
			action: "DeleteVirtualMachine",
		},
		{
			name:   "list resource group virtual machines",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines?api-version=2025-11-01",
			action: "ListVirtualMachines",
		},
		{
			name:   "list subscription virtual machines",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Compute/virtualMachines?api-version=2025-11-01",
			action: "ListVirtualMachines",
		},
		{
			name:   "get virtual machine instance view",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/instanceView?api-version=2025-11-01",
			action: "GetVirtualMachineInstanceView",
		},
		{
			name:   "start virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/start?api-version=2025-11-01",
			action: "StartVirtualMachine",
		},
		{
			name:   "power off virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/powerOff?api-version=2025-11-01",
			action: "PowerOffVirtualMachine",
		},
		{
			name:   "deallocate virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/deallocate?api-version=2025-11-01",
			action: "DeallocateVirtualMachine",
		},
		{
			name:   "restart virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/restart?api-version=2025-11-01",
			action: "RestartVirtualMachine",
		},
		{
			name:   "redeploy virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/redeploy?api-version=2025-11-01",
			action: "RedeployVirtualMachine",
		},
		{
			name:   "reapply virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/reapply?api-version=2025-11-01",
			action: "ReapplyVirtualMachine",
		},
		{
			name:   "run command on virtual machine",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/runCommand?api-version=2025-11-01",
			action: "RunCommandVirtualMachine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Compute/virtualMachines", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-11-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureComputeDiskActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create disk",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a?api-version=2025-01-02",
			action: "CreateOrUpdateDisk",
		},
		{
			name:   "get disk",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a?api-version=2025-01-02",
			action: "GetDisk",
		},
		{
			name:   "patch disk",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a?api-version=2025-01-02",
			action: "UpdateDisk",
		},
		{
			name:   "delete disk",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a?api-version=2025-01-02",
			action: "DeleteDisk",
		},
		{
			name:   "list resource group disks",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks?api-version=2025-01-02",
			action: "ListDisks",
		},
		{
			name:   "list subscription disks",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Compute/disks?api-version=2025-01-02",
			action: "ListDisks",
		},
		{
			name:   "grant disk access",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a/beginGetAccess?api-version=2025-01-02",
			action: "GrantAccessDisk",
		},
		{
			name:   "revoke disk access",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a/endGetAccess?api-version=2025-01-02",
			action: "RevokeAccessDisk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Compute/disks", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2025-01-02", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAKSManagedClusterActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create cluster",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a?api-version=2026-02-01",
			action: "CreateOrUpdateManagedCluster",
		},
		{
			name:   "list resource group clusters",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters?api-version=2026-02-01",
			action: "ListManagedClusters",
		},
		{
			name:   "list subscription clusters",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerService/managedClusters?api-version=2026-02-01",
			action: "ListManagedClusters",
		},
		{
			name:   "patch cluster tags",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a?api-version=2026-02-01",
			action: "UpdateManagedClusterTags",
		},
		{
			name:   "list admin credentials",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/listClusterAdminCredential?api-version=2026-02-01",
			action: "ListClusterAdminCredentials",
		},
		{
			name:   "list user credentials",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/listClusterUserCredential?api-version=2026-02-01",
			action: "ListClusterUserCredentials",
		},
		{
			name:   "list monitoring user credentials",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/listClusterMonitoringUserCredential?api-version=2026-03-01&server-fqdn=private",
			action: "ListClusterMonitoringUserCredentials",
		},
		{
			name:   "stop cluster",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/stop?api-version=2026-03-01",
			action: "StopManagedCluster",
		},
		{
			name:   "start cluster",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/start?api-version=2026-03-01",
			action: "StartManagedCluster",
		},
		{
			name:   "rotate cluster certificates",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/rotateClusterCertificates?api-version=2026-03-01",
			action: "RotateClusterCertificates",
		},
		{
			name:   "rotate service account signing keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/rotateServiceAccountSigningKeys?api-version=2026-03-01",
			action: "RotateServiceAccountSigningKeys",
		},
		{
			name:   "get upgrade profile",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/upgradeProfiles/default?api-version=2026-03-01",
			action: "GetManagedClusterUpgradeProfile",
		},
		{
			name:   "run command",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/runCommand?api-version=2026-03-01",
			action: "RunCommand",
		},
		{
			name:   "get command result",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/commandResults/cloudmock-command-1?api-version=2026-03-01",
			action: "GetCommandResult",
		},
		{
			name:   "list agent pools",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools?api-version=2026-04-01",
			action: "ListAgentPools",
		},
		{
			name:   "create agent pool",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools/userpool?api-version=2026-04-01",
			action: "CreateOrUpdateAgentPool",
		},
		{
			name:   "delete agent pool",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools/userpool?api-version=2026-04-01",
			action: "DeleteAgentPool",
		},
		{
			name:   "upgrade agent pool node image version",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools/userpool/upgradeNodeImageVersion?api-version=2026-03-01",
			action: "UpgradeAgentPoolNodeImageVersion",
		},
		{
			name:   "abort agent pool latest operation",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools/userpool/abort?api-version=2026-03-01",
			action: "AbortAgentPoolLatestOperation",
		},
		{
			name:   "get agent pool upgrade profile",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/agentPools/userpool/upgradeProfiles/default?api-version=2026-03-01",
			action: "GetAgentPoolUpgradeProfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ContainerService/managedClusters", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.NotEmpty(t, target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAKSLocationActions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerService/locations/eastus/kubernetesVersions?api-version=2026-03-01", nil)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.ContainerService/locations", target.Service)
	assert.Equal(t, "ListKubernetesVersions", target.Action)
	assert.Equal(t, "2026-03-01", target.APIVersion)
}

func TestDetectTarget_AzureAPIManagementServiceActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create service",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a?api-version=2024-05-01",
			action: "CreateOrUpdateService",
		},
		{
			name:   "list resource group services",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service?api-version=2024-05-01",
			action: "ListServices",
		},
		{
			name:   "list subscription services",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ApiManagement/service?api-version=2024-05-01",
			action: "ListServices",
		},
		{
			name:   "delete service",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a?api-version=2024-05-01",
			action: "DeleteService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ApiManagement/service", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-05-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAPIManagementAPIAndOperationActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create api",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api?api-version=2024-05-01",
			action: "CreateOrUpdateAPI",
		},
		{
			name:   "list apis",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis?api-version=2024-05-01",
			action: "ListAPIs",
		},
		{
			name:   "delete api",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api?api-version=2024-05-01",
			action: "DeleteAPI",
		},
		{
			name:   "create operation",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api/operations/get-item?api-version=2024-05-01",
			action: "CreateOrUpdateOperation",
		},
		{
			name:   "list operations",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api/operations?api-version=2024-05-01",
			action: "ListOperations",
		},
		{
			name:   "delete operation",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api/operations/get-item?api-version=2024-05-01",
			action: "DeleteOperation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ApiManagement/service", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-05-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAPIManagementPolicyActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create service policy",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/policies/policy?api-version=2024-05-01",
			action: "CreateOrUpdatePolicy",
		},
		{
			name:   "list api policies",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api/policies?api-version=2024-05-01",
			action: "ListPolicies",
		},
		{
			name:   "delete operation policy",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/apis/catalog-api/operations/get-item/policies/policy?api-version=2024-05-01",
			action: "DeletePolicy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ApiManagement/service", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-05-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAPIManagementAuxiliaryResourceActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create product",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products/starter?api-version=2024-05-01",
			action: "CreateOrUpdateProduct",
		},
		{
			name:   "list products",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products?api-version=2024-05-01",
			action: "ListProducts",
		},
		{
			name:   "delete product",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products/starter?api-version=2024-05-01",
			action: "DeleteProduct",
		},
		{
			name:   "link product api",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products/starter/apis/catalog-api?api-version=2024-05-01",
			action: "CreateOrUpdateProductAPI",
		},
		{
			name:   "list product apis",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products/starter/apis?api-version=2024-05-01",
			action: "ListProductAPIs",
		},
		{
			name:   "delete product api link",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/products/starter/apis/catalog-api?api-version=2024-05-01",
			action: "DeleteProductAPI",
		},
		{
			name:   "create subscription",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/subscriptions/starter-sub?api-version=2024-05-01",
			action: "CreateOrUpdateSubscription",
		},
		{
			name:   "list subscriptions",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/subscriptions?api-version=2024-05-01",
			action: "ListSubscriptions",
		},
		{
			name:   "delete subscription",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/subscriptions/starter-sub?api-version=2024-05-01",
			action: "DeleteSubscription",
		},
		{
			name:   "create named value",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/namedValues/floci-secret?api-version=2024-05-01",
			action: "CreateOrUpdateNamedValue",
		},
		{
			name:   "list named values",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/namedValues?api-version=2024-05-01",
			action: "ListNamedValues",
		},
		{
			name:   "delete named value",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/namedValues/floci-secret?api-version=2024-05-01",
			action: "DeleteNamedValue",
		},
		{
			name:   "create backend",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/backends/catalog-backend?api-version=2024-05-01",
			action: "CreateOrUpdateBackend",
		},
		{
			name:   "list backends",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/backends?api-version=2024-05-01",
			action: "ListBackends",
		},
		{
			name:   "delete backend",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apim-a/backends/catalog-backend?api-version=2024-05-01",
			action: "DeleteBackend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.ApiManagement/service", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-05-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAPIManagementLocalGatewayAction(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apim-a/catalog/items/42", nil)

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.ApiManagement/service", target.Service)
	assert.Equal(t, "GatewayProxy", target.Action)
	assert.Equal(t, "", target.APIVersion)
}

func TestDetectTarget_AzureFunctionsLocalDataPlane(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create function app",
			method: http.MethodPut,
			url:    "http://localhost:4577/devstoreaccount1-functions/admin/apps/app-a",
			action: "CreateOrUpdateFunctionApp",
		},
		{
			name:   "deploy function",
			method: http.MethodPut,
			url:    "http://localhost:4577/devstoreaccount1-functions/admin/apps/app-a/functions/hello",
			action: "DeployFunction",
		},
		{
			name:   "invoke function",
			method: http.MethodGet,
			url:    "http://localhost:4577/devstoreaccount1-functions/api/app-a/hello?msg=world",
			action: "InvokeFunction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/functions", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServiceSubscriptionLists(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		service string
		action  string
	}{
		{
			name:    "list plans by subscription",
			url:     "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Web/serverfarms?api-version=2024-04-01",
			service: "Microsoft.Web/serverfarms",
			action:  "ListPlans",
		},
		{
			name:    "list sites by subscription",
			url:     "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Web/sites?api-version=2024-04-01",
			service: "Microsoft.Web/sites",
			action:  "ListSites",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, tt.service, target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServicePlanActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create or update plan",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01",
			action: "CreateOrUpdatePlan",
		},
		{
			name:   "get plan",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01",
			action: "GetPlan",
		},
		{
			name:   "patch plan",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01",
			action: "UpdatePlan",
		},
		{
			name:   "delete plan",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01",
			action: "DeletePlan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/serverfarms", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServiceSlotActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "list slots",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots?api-version=2024-04-01",
			action: "ListSlots",
		},
		{
			name:   "create or update slot",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging?api-version=2024-04-01",
			action: "CreateOrUpdateSlot",
		},
		{
			name:   "get slot",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging?api-version=2024-04-01",
			action: "GetSlot",
		},
		{
			name:   "patch slot",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging?api-version=2024-04-01",
			action: "UpdateSlot",
		},
		{
			name:   "get slot web configuration",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging/config/web?api-version=2024-04-01",
			action: "GetSlotConfiguration",
		},
		{
			name:   "create or update slot web configuration",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging/config/web?api-version=2024-04-01",
			action: "CreateOrUpdateSlotConfiguration",
		},
		{
			name:   "patch slot web configuration",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging/config/web?api-version=2024-04-01",
			action: "UpdateSlotConfiguration",
		},
		{
			name:   "delete slot",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging?deleteMetrics=true&api-version=2024-04-01",
			action: "DeleteSlot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/sites", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServiceSiteActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "start site",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/start?api-version=2024-04-01",
			action: "StartSite",
		},
		{
			name:   "stop site",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/stop?api-version=2024-04-01",
			action: "StopSite",
		},
		{
			name:   "restart site",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/restart?softRestart=true&synchronous=true&api-version=2024-04-01",
			action: "RestartSite",
		},
		{
			name:   "sync function triggers",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/syncfunctiontriggers?api-version=2024-04-01",
			action: "SyncFunctionTriggers",
		},
		{
			name:   "patch site",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a?api-version=2024-04-01",
			action: "UpdateSite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/sites", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServiceFunctionActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "create function",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger?api-version=2024-04-01",
			action: "CreateOrUpdateFunction",
		},
		{
			name:   "get function",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger?api-version=2024-04-01",
			action: "GetFunction",
		},
		{
			name:   "list functions",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions?api-version=2024-04-01",
			action: "ListFunctions",
		},
		{
			name:   "delete function",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger?api-version=2024-04-01",
			action: "DeleteFunction",
		},
		{
			name:   "list function keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger/listkeys?api-version=2024-04-01",
			action: "ListFunctionKeys",
		},
		{
			name:   "create or update function secret",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger/keys/client?api-version=2024-04-01",
			action: "CreateOrUpdateFunctionSecret",
		},
		{
			name:   "delete function secret",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/functions/HttpTrigger/keys/client?api-version=2024-04-01",
			action: "DeleteFunctionSecret",
		},
		{
			name:   "list host keys",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/host/default/listkeys?api-version=2024-04-01",
			action: "ListHostKeys",
		},
		{
			name:   "create or update host secret",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/host/default/functionkeys/client?api-version=2024-04-01",
			action: "CreateOrUpdateHostSecret",
		},
		{
			name:   "delete host secret",
			method: http.MethodDelete,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/host/default/functionkeys/client?api-version=2024-04-01",
			action: "DeleteHostSecret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/sites", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAppServiceConfigActions(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "get web configuration",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/web?api-version=2024-04-01",
			action: "GetConfiguration",
		},
		{
			name:   "update web configuration",
			method: http.MethodPatch,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/web?api-version=2024-04-01",
			action: "UpdateConfiguration",
		},
		{
			name:   "update app settings",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/appsettings?api-version=2024-04-01",
			action: "UpdateApplicationSettings",
		},
		{
			name:   "list app settings",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/appsettings/list?api-version=2024-04-01",
			action: "ListApplicationSettings",
		},
		{
			name:   "update connection strings",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/connectionstrings?api-version=2024-04-01",
			action: "UpdateConnectionStrings",
		},
		{
			name:   "list connection strings",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/connectionstrings/list?api-version=2024-04-01",
			action: "ListConnectionStrings",
		},
		{
			name:   "list slot configuration names",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/slotConfigNames?api-version=2024-04-01",
			action: "ListSlotConfigurationNames",
		},
		{
			name:   "update slot configuration names",
			method: http.MethodPut,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/slotConfigNames?api-version=2024-04-01",
			action: "UpdateSlotConfigurationNames",
		},
		{
			name:   "list publishing credentials",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/config/publishingcredentials/list?api-version=2024-04-01",
			action: "ListPublishingCredentials",
		},
		{
			name:   "list publishing profile XML with secrets",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/publishxml?api-version=2024-04-01",
			action: "ListPublishingProfileXMLWithSecrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Web/sites", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2024-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureResourceProviderRoutesUseMicrosoftResources(t *testing.T) {
	tests := []struct {
		name   string
		method string
		url    string
		action string
	}{
		{
			name:   "list providers",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers?api-version=2021-04-01",
			action: "List",
		},
		{
			name:   "get provider",
			method: http.MethodGet,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Storage?api-version=2021-04-01",
			action: "Get",
		},
		{
			name:   "register provider",
			method: http.MethodPost,
			url:    "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Storage/register?api-version=2021-04-01",
			action: "register",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("Authorization", "Bearer token")

			target := DetectTarget(req)

			assert.Equal(t, ProviderAzure, target.Provider)
			assert.Equal(t, "Microsoft.Resources/providers", target.Service)
			assert.Equal(t, tt.action, target.Action)
			assert.Equal(t, "2021-04-01", target.APIVersion)
		})
	}
}

func TestDetectTarget_AzureAuthorizationRoleAssignmentAtNestedScope(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPut,
		"https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct/providers/Microsoft.Authorization/roleAssignments/assignment-a?api-version=2022-04-01",
		nil,
	)
	req.Header.Set("Authorization", "Bearer token")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.Authorization/roleAssignments", target.Service)
	assert.Equal(t, "CreateOrUpdate", target.Action)
	assert.Equal(t, "2022-04-01", target.APIVersion)
}

func TestDetectTarget_HeaderOverridesSupportProviderFixtures(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Cloudmock-Provider", "azure")
	req.Header.Set("X-Cloudmock-Service", "Microsoft.KeyVault/vaults")
	req.Header.Set("X-Cloudmock-Action", "CreateOrUpdate")
	req.Header.Set("X-Cloudmock-API-Version", "2023-07-01")

	target := DetectTarget(req)

	assert.Equal(t, ProviderAzure, target.Provider)
	assert.Equal(t, "Microsoft.KeyVault/vaults", target.Service)
	assert.Equal(t, "CreateOrUpdate", target.Action)
	assert.Equal(t, "2023-07-01", target.APIVersion)
}
