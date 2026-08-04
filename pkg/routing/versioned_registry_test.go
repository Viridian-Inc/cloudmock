package routing

import (
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_LegacyRegisterAndLookupRemainAWSCompatible(t *testing.T) {
	reg := NewRegistry()
	svc := &mockService{name: "s3"}

	reg.Register(svc)

	got, err := reg.LookupTarget(RouteTarget{Provider: ProviderAWS, Service: "s3"})
	require.NoError(t, err)
	assert.Equal(t, svc, got)
}

func TestRegistry_VersionedServicesCoexist(t *testing.T) {
	reg := NewRegistry()
	v2023 := &mockService{name: "storageAccounts-2023"}
	v2024 := &mockService{name: "storageAccounts-2024"}

	reg.RegisterVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Storage/storageAccounts", APIVersion: "2023-01-01"}, v2023)
	reg.RegisterVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Storage/storageAccounts", APIVersion: "2024-01-01"}, v2024)

	got2023, err := reg.LookupVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Storage/storageAccounts", APIVersion: "2023-01-01"})
	require.NoError(t, err)
	got2024, err := reg.LookupVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Storage/storageAccounts", APIVersion: "2024-01-01"})
	require.NoError(t, err)

	assert.Equal(t, v2023, got2023)
	assert.Equal(t, v2024, got2024)
}

func TestRegistry_DefaultAPIVersionResolvesWhenRequestOmitsVersion(t *testing.T) {
	reg := NewRegistry()
	svc := &mockService{name: "resources-2021"}

	reg.RegisterVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Resources/resourceGroups", APIVersion: "2021-04-01"}, svc)
	reg.SetDefaultVersion(ProviderAzure, "Microsoft.Resources/resourceGroups", "2021-04-01")

	got, err := reg.LookupTarget(RouteTarget{Provider: ProviderAzure, Service: "Microsoft.Resources/resourceGroups"})
	require.NoError(t, err)
	assert.Equal(t, svc, got)
}

func TestRegistry_AzureGenericResourceFallbackHandlesUnregisteredProviderResources(t *testing.T) {
	reg := NewRegistry()
	generic := &mockService{name: "generic-resources"}

	reg.RegisterVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.Resources/resources", APIVersion: "2021-04-01"}, generic)

	got, err := reg.LookupTarget(RouteTarget{Provider: ProviderAzure, Service: "Microsoft.Network/virtualNetworks", APIVersion: "2021-04-01"})
	require.NoError(t, err)
	assert.Equal(t, generic, got)
}

func TestRegistry_RegisterLazyVersionedInitializesOnce(t *testing.T) {
	reg := NewRegistry()
	calls := 0

	reg.RegisterLazyVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.KeyVault/vaults", APIVersion: "2023-07-01"}, func() service.Service {
		calls++
		return &mockService{name: "keyvault"}
	})

	first, err := reg.LookupVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.KeyVault/vaults", APIVersion: "2023-07-01"})
	require.NoError(t, err)
	second, err := reg.LookupVersioned(ServiceKey{Provider: ProviderAzure, Service: "Microsoft.KeyVault/vaults", APIVersion: "2023-07-01"})
	require.NoError(t, err)

	assert.Equal(t, 1, calls)
	assert.Equal(t, first, second)
}
