package internal

import (
	"os"
	"path/filepath"
	"testing"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
	"github.com/neureaux/cloudmock/services/dynamodb"
	"github.com/neureaux/cloudmock/services/ec2"
	"github.com/neureaux/cloudmock/services/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func defaultRegistry() *cmschema.Registry {
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

	return cmschema.BuildRegistry(tier1, nil)
}

func TestGenerateCRDs_ProducesCorrectCount(t *testing.T) {
	reg := defaultRegistry()
	crds := GenerateCRDs(reg)
	assert.GreaterOrEqual(t, len(crds), 3, "expected at least S3, DynamoDB, and EC2 CRDs")
}

func TestGenerateCRD_S3Bucket_Structure(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		CreateAction:  "CreateBucket",
		ReadAction:    "HeadBucket",
		DeleteAction:  "DeleteBucket",
		ImportID:      "bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true, ForceNew: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "region", Type: "string", Computed: true},
			{Name: "acl", Type: "string", Default: "private"},
			{Name: "tags", Type: "map"},
		},
	}

	crd := GenerateCRD(rs)

	// Verify top-level structure.
	assert.Equal(t, "apiextensions.k8s.io/v1", crd.APIVersion)
	assert.Equal(t, "CustomResourceDefinition", crd.Kind)

	// Verify group/version/kind.
	assert.Equal(t, "s3.cloudmock.app", crd.Spec.Group)
	assert.Equal(t, "Bucket", crd.Spec.Names.Kind)
	assert.Equal(t, "BucketList", crd.Spec.Names.ListKind)
	assert.Equal(t, "buckets", crd.Spec.Names.Plural)
	assert.Equal(t, "bucket", crd.Spec.Names.Singular)
	assert.Equal(t, []string{"s3bucket"}, crd.Spec.Names.ShortNames)
	assert.Equal(t, "Cluster", crd.Spec.Scope)

	// Verify metadata.
	assert.Equal(t, "buckets.s3.cloudmock.app", crd.Metadata.Name)
	assert.Equal(t, "cloudmock-crossplane", crd.Metadata.Labels["app.kubernetes.io/managed-by"])

	// Verify version.
	require.Len(t, crd.Spec.Versions, 1)
	v := crd.Spec.Versions[0]
	assert.Equal(t, "v1alpha1", v.Name)
	assert.True(t, v.Served)
	assert.True(t, v.Storage)

	// Verify printer columns.
	require.Len(t, v.Printer, 3)
	assert.Equal(t, "READY", v.Printer[0].Name)
	assert.Equal(t, "SYNCED", v.Printer[1].Name)
	assert.Equal(t, "AGE", v.Printer[2].Name)
}

func TestGenerateCRD_ForProviderHasInputFields(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true},
			{Name: "acl", Type: "string"},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "tags", Type: "map"},
		},
	}

	crd := GenerateCRD(rs)
	schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	spec := schema.Properties["spec"]
	forProvider := spec.Properties["forProvider"]

	// Input fields should be in forProvider.
	assert.Contains(t, forProvider.Properties, "bucket")
	assert.Contains(t, forProvider.Properties, "acl")
	assert.Contains(t, forProvider.Properties, "tags")

	// Computed-only fields should NOT be in forProvider.
	assert.NotContains(t, forProvider.Properties, "arn")

	// Required fields.
	assert.Contains(t, forProvider.Required, "bucket")
}

func TestGenerateCRD_AtProviderHasComputedFields(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "region", Type: "string", Computed: true},
		},
	}

	crd := GenerateCRD(rs)
	schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	status := schema.Properties["status"]
	atProvider := status.Properties["atProvider"]

	// Computed fields should be in atProvider.
	assert.Contains(t, atProvider.Properties, "arn")
	assert.Contains(t, atProvider.Properties, "region")
	assert.Contains(t, atProvider.Properties, "id")

	// Non-computed input fields should NOT be in atProvider.
	assert.NotContains(t, atProvider.Properties, "bucket")
}

func TestGenerateCRD_EC2VPC(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "ec2",
		TerraformType: "cloudmock_ec2_vpc",
		AWSType:       "AWS::EC2::VPC",
		Attributes: []cmschema.AttributeSchema{
			{Name: "cidr_block", Type: "string", Required: true, ForceNew: true},
			{Name: "vpc_id", Type: "string", Computed: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "enable_dns_support", Type: "bool", Default: true},
		},
	}

	crd := GenerateCRD(rs)
	assert.Equal(t, "ec2.cloudmock.app", crd.Spec.Group)
	assert.Equal(t, "VPC", crd.Spec.Names.Kind)
	assert.Equal(t, []string{"vpc"}, crd.Spec.Names.ShortNames)
}

func TestGenerateCRD_DynamoDBTable(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "dynamodb",
		TerraformType: "cloudmock_dynamodb_table",
		AWSType:       "AWS::DynamoDB::Table",
		Attributes: []cmschema.AttributeSchema{
			{Name: "table_name", Type: "string", Required: true},
			{Name: "hash_key", Type: "string", Required: true},
			{Name: "arn", Type: "string", Computed: true},
			{Name: "billing_mode", Type: "string", Default: "PROVISIONED"},
			{Name: "read_capacity", Type: "int"},
			{Name: "tags", Type: "map"},
		},
	}

	crd := GenerateCRD(rs)
	assert.Equal(t, "dynamodb.cloudmock.app", crd.Spec.Group)
	assert.Equal(t, "Table", crd.Spec.Names.Kind)

	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	forProvider := spec.Properties["forProvider"]

	// Type mappings.
	assert.Equal(t, "string", forProvider.Properties["table_name"].Type)
	assert.Equal(t, "integer", forProvider.Properties["read_capacity"].Type)
	assert.Equal(t, "object", forProvider.Properties["tags"].Type)
}

func TestGenerateCRD_TypeMappings(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "test",
		TerraformType: "cloudmock_test_widget",
		AWSType:       "AWS::Test::Widget",
		Attributes: []cmschema.AttributeSchema{
			{Name: "name", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "enabled", Type: "bool"},
			{Name: "ratio", Type: "float"},
			{Name: "items", Type: "list"},
			{Name: "rules", Type: "set"},
			{Name: "tags", Type: "map"},
		},
	}

	crd := GenerateCRD(rs)
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	fp := spec.Properties["forProvider"]

	assert.Equal(t, "string", fp.Properties["name"].Type)
	assert.Equal(t, "integer", fp.Properties["count"].Type)
	assert.Equal(t, "boolean", fp.Properties["enabled"].Type)
	assert.Equal(t, "number", fp.Properties["ratio"].Type)
	assert.Equal(t, "array", fp.Properties["items"].Type)
	assert.Equal(t, "array", fp.Properties["rules"].Type)
	assert.Equal(t, "object", fp.Properties["tags"].Type)
}

func TestGenerateCRD_UnknownResourceDerivesMapping(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "newservice",
		TerraformType: "cloudmock_newservice_fancy_thing",
		AWSType:       "AWS::NewService::FancyThing",
		Attributes: []cmschema.AttributeSchema{
			{Name: "name", Type: "string", Required: true},
		},
	}

	crd := GenerateCRD(rs)
	assert.Equal(t, "newservice.cloudmock.app", crd.Spec.Group)
	assert.Equal(t, "FancyThing", crd.Spec.Names.Kind)
	// No short name for unknown resources.
	assert.Empty(t, crd.Spec.Names.ShortNames)
}

func TestCRDGroup_And_CRDKind(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
	}
	assert.Equal(t, "s3.cloudmock.app", CRDGroup(rs))
	assert.Equal(t, "Bucket", CRDKind(rs))
}

func TestGenerateCRD_ValidYAML(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true},
			{Name: "arn", Type: "string", Computed: true},
		},
	}

	crd := GenerateCRD(rs)
	data, err := yaml.Marshal(crd)
	require.NoError(t, err)
	assert.Contains(t, string(data), "apiextensions.k8s.io/v1")
	assert.Contains(t, string(data), "Bucket")
	assert.Contains(t, string(data), "s3.cloudmock.app")

	// Verify it round-trips.
	var parsed CRD
	err = yaml.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "Bucket", parsed.Spec.Names.Kind)
}

func TestGenerateCRDs_WriteToDisk(t *testing.T) {
	if os.Getenv("WRITE_CRDS") == "" {
		t.Skip("set WRITE_CRDS=1 to write CRD YAML files")
	}

	reg := defaultRegistry()
	crds := GenerateCRDs(reg)

	outDir := filepath.Join("..", "crds")
	err := os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	for _, crd := range crds {
		data, err := yaml.Marshal(crd)
		require.NoError(t, err)

		filename := crd.Spec.Group + "_" + crd.Spec.Names.Plural + ".yaml"
		path := filepath.Join(outDir, filename)
		err = os.WriteFile(path, data, 0644)
		require.NoError(t, err)
		t.Logf("wrote %s (%d bytes)", path, len(data))
	}
}

func TestProviderConfigRef(t *testing.T) {
	rs := cmschema.ResourceSchema{
		ServiceName:   "s3",
		TerraformType: "cloudmock_s3_bucket",
		AWSType:       "AWS::S3::Bucket",
		Attributes: []cmschema.AttributeSchema{
			{Name: "bucket", Type: "string", Required: true},
		},
	}

	crd := GenerateCRD(rs)
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]

	// providerConfigRef should be present.
	assert.Contains(t, spec.Properties, "providerConfigRef")
	pcr := spec.Properties["providerConfigRef"]
	assert.Contains(t, pcr.Properties, "name")
	assert.Contains(t, pcr.Required, "name")

	// deletionPolicy should be present.
	assert.Contains(t, spec.Properties, "deletionPolicy")
	assert.Equal(t, "string", spec.Properties["deletionPolicy"].Type)
}
