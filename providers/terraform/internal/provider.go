package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/services/dynamodb"
	"github.com/neureaux/cloudmock/services/ec2"
	"github.com/neureaux/cloudmock/services/s3"
)

// Provider returns the cloudmock Terraform provider.
// It dynamically builds resources from the schema registry.
func Provider() *schema.Provider {
	registry := buildRegistry()

	resourcesMap := map[string]*schema.Resource{}
	for _, rs := range registry.All() {
		resourcesMap[rs.TerraformType] = buildResource(rs)
	}

	return &schema.Provider{
		Schema: providerSchema(),
		ResourcesMap:         resourcesMap,
		ConfigureContextFunc: configureProvider,
	}
}

// providerSchema returns the provider-level configuration schema.
func providerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"endpoint": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The cloudmock gateway endpoint URL (e.g., http://localhost:4566).",
			DefaultFunc: schema.EnvDefaultFunc("CLOUDMOCK_ENDPOINT", "http://localhost:4566"),
		},
		"region": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "us-east-1",
			Description: "The AWS region to use for credential scope.",
		},
		"access_key": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "test",
			Description: "The access key for authenticating with cloudmock.",
		},
		"secret_key": {
			Type:        schema.TypeString,
			Optional:    true,
			Sensitive:   true,
			Default:     "test",
			Description: "The secret key for authenticating with cloudmock.",
		},
	}
}

// configureProvider reads the provider configuration and returns an API client
// that resources use to communicate with the cloudmock gateway.
func configureProvider(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	endpoint := d.Get("endpoint").(string)
	region := d.Get("region").(string)
	accessKey := d.Get("access_key").(string)
	secretKey := d.Get("secret_key").(string)

	if endpoint == "" {
		return nil, diag.Errorf("endpoint must be set")
	}

	client := NewAPIClient(endpoint, region, accessKey, secretKey)
	return client, nil
}

// buildRegistry creates a schema registry populated with all known resource schemas.
// It extracts schemas from Tier 1 services that implement SchemaProvider,
// and from Tier 2 stub models.
func buildRegistry() *cmschema.Registry {
	// Create placeholder services to extract schemas.
	// These are not started — we only need their ResourceSchemas() output.
	s3Svc := s3.New()
	dynamoSvc := dynamodb.New("000000000000", "us-east-1")
	ec2Svc := ec2.New("000000000000", "us-east-1")

	var tier1 []cmschema.ResourceSchema
	for _, schemas := range [][]cmschema.ResourceSchema{
		s3Svc.ResourceSchemas(),
		dynamoSvc.ResourceSchemas(),
		ec2Svc.ResourceSchemas(),
	} {
		tier1 = append(tier1, schemas...)
	}

	// For now we only include Tier 1 schemas with hand-crafted definitions.
	// Tier 2 stub schemas can be added later once the stub catalog is wired in.
	return cmschema.BuildRegistry(tier1, nil)
}

// ProviderWithRegistry creates a provider from a pre-built registry.
// This is useful for testing with custom schemas.
func ProviderWithRegistry(registry *cmschema.Registry) *schema.Provider {
	resourcesMap := map[string]*schema.Resource{}
	for _, rs := range registry.All() {
		resourcesMap[rs.TerraformType] = buildResource(rs)
	}

	return &schema.Provider{
		Schema: providerSchema(),
		ResourcesMap:         resourcesMap,
		ConfigureContextFunc: configureProvider,
	}
}

// ProviderFromSchemas creates a provider from a list of resource schemas.
// This is a convenience helper for tests.
func ProviderFromSchemas(schemas []cmschema.ResourceSchema) *schema.Provider {
	reg := cmschema.NewRegistry()
	reg.Add(schemas...)
	return ProviderWithRegistry(reg)
}

// RegistryResourceCount returns the number of resources the provider manages.
// Useful for smoke tests.
func RegistryResourceCount() int {
	return buildRegistry().Len()
}

func init() {
	// Ensure the provider description is set for documentation.
	_ = fmt.Sprintf("cloudmock Terraform provider")
}
