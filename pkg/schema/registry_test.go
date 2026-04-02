package schema

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Helpers ─────────────────────────────────────────────────────────────────

// fakeSchemaService is a mock service that implements both Service and SchemaProvider.
type fakeSchemaService struct {
	name    string
	schemas []ResourceSchema
}

func (f *fakeSchemaService) Name() string                                                 { return f.name }
func (f *fakeSchemaService) Actions() []service.Action                                    { return nil }
func (f *fakeSchemaService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) { return nil, nil }
func (f *fakeSchemaService) HealthCheck() error                                           { return nil }
func (f *fakeSchemaService) ResourceSchemas() []ResourceSchema                            { return f.schemas }

// fakeBasicService implements Service but NOT SchemaProvider.
type fakeBasicService struct {
	name    string
	actions []service.Action
}

func (f *fakeBasicService) Name() string                 { return f.name }
func (f *fakeBasicService) Actions() []service.Action    { return f.actions }
func (f *fakeBasicService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) { return nil, nil }
func (f *fakeBasicService) HealthCheck() error           { return nil }

// ── Tests ───────────────────────────────────────────────────────────────────

func TestExtractFromServices(t *testing.T) {
	svc := &fakeSchemaService{
		name: "s3",
		schemas: []ResourceSchema{
			{
				ServiceName:   "s3",
				TerraformType: "cloudmock_s3_bucket",
				AWSType:       "AWS::S3::Bucket",
			},
		},
	}

	schemas := ExtractFromServices([]service.Service{svc})
	require.Len(t, schemas, 1)
	assert.Equal(t, "cloudmock_s3_bucket", schemas[0].TerraformType)
}

func TestExtractFromServicesSkipsNonProvider(t *testing.T) {
	basic := &fakeBasicService{name: "sts"}
	schemas := ExtractFromServices([]service.Service{basic})
	assert.Empty(t, schemas)
}

func TestExtractFromActions(t *testing.T) {
	svc := &fakeBasicService{
		name: "sqs",
		actions: []service.Action{
			{Name: "CreateQueue"},
			{Name: "DeleteQueue"},
			{Name: "ListQueues"},
		},
	}

	schemas := ExtractFromActions(svc, "123456789012", "us-east-1")
	require.Len(t, schemas, 1)

	s := schemas[0]
	assert.Equal(t, "sqs", s.ServiceName)
	assert.Equal(t, "cloudmock_sqs_queue", s.TerraformType)
	assert.Equal(t, "AWS::SQS::Queue", s.AWSType)
	assert.Equal(t, "CreateQueue", s.CreateAction)
	assert.Equal(t, "DeleteQueue", s.DeleteAction)
	assert.Equal(t, "ListQueues", s.ListAction)
}

func TestExtractFromActionsMultipleResources(t *testing.T) {
	svc := &fakeBasicService{
		name: "ec2",
		actions: []service.Action{
			{Name: "CreateVpc"},
			{Name: "DescribeVpcs"},
			{Name: "DeleteVpc"},
			{Name: "CreateSubnet"},
			{Name: "DescribeSubnets"},
			{Name: "DeleteSubnet"},
		},
	}

	schemas := ExtractFromActions(svc, "123456789012", "us-east-1")
	assert.Len(t, schemas, 2)

	// Collect by type.
	byType := map[string]ResourceSchema{}
	for _, s := range schemas {
		byType[s.TerraformType] = s
	}

	vpc, ok := byType["cloudmock_ec2_vpc"]
	require.True(t, ok)
	assert.Equal(t, "CreateVpc", vpc.CreateAction)
	assert.Equal(t, "DescribeVpcs", vpc.ReadAction)

	subnet, ok := byType["cloudmock_ec2_subnet"]
	require.True(t, ok)
	assert.Equal(t, "CreateSubnet", subnet.CreateAction)
}

func TestBuildRegistryTier1Wins(t *testing.T) {
	tier1 := []ResourceSchema{
		{
			ServiceName:   "s3",
			TerraformType: "cloudmock_s3_bucket",
			AWSType:       "AWS::S3::Bucket",
			CreateAction:  "CreateBucket",
			ImportID:      "bucket",
		},
	}
	tier2 := []ResourceSchema{
		{
			ServiceName:   "s3",
			TerraformType: "cloudmock_s3_bucket",
			AWSType:       "AWS::S3::Bucket",
			CreateAction:  "StubCreateBucket",
			ImportID:      "BucketName",
		},
	}

	reg := BuildRegistry(tier1, tier2)
	s, ok := reg.Get("cloudmock_s3_bucket")
	require.True(t, ok)
	// Tier 1 should win.
	assert.Equal(t, "CreateBucket", s.CreateAction)
	assert.Equal(t, "bucket", s.ImportID)
}

func TestRegistryAllSorted(t *testing.T) {
	reg := NewRegistry()
	reg.Add(
		ResourceSchema{TerraformType: "cloudmock_sqs_queue", ServiceName: "sqs"},
		ResourceSchema{TerraformType: "cloudmock_ec2_vpc", ServiceName: "ec2"},
		ResourceSchema{TerraformType: "cloudmock_dynamodb_table", ServiceName: "dynamodb"},
		ResourceSchema{TerraformType: "cloudmock_s3_bucket", ServiceName: "s3"},
	)

	all := reg.All()
	require.Len(t, all, 4)
	assert.Equal(t, "cloudmock_dynamodb_table", all[0].TerraformType)
	assert.Equal(t, "cloudmock_ec2_vpc", all[1].TerraformType)
	assert.Equal(t, "cloudmock_s3_bucket", all[2].TerraformType)
	assert.Equal(t, "cloudmock_sqs_queue", all[3].TerraformType)
}

func TestRegistryByService(t *testing.T) {
	reg := NewRegistry()
	reg.Add(
		ResourceSchema{TerraformType: "cloudmock_ec2_vpc", ServiceName: "ec2"},
		ResourceSchema{TerraformType: "cloudmock_ec2_subnet", ServiceName: "ec2"},
		ResourceSchema{TerraformType: "cloudmock_s3_bucket", ServiceName: "s3"},
		ResourceSchema{TerraformType: "cloudmock_ec2_instance", ServiceName: "ec2"},
	)

	ec2 := reg.ByService("ec2")
	require.Len(t, ec2, 3)
	// Should be sorted.
	assert.Equal(t, "cloudmock_ec2_instance", ec2[0].TerraformType)
	assert.Equal(t, "cloudmock_ec2_subnet", ec2[1].TerraformType)
	assert.Equal(t, "cloudmock_ec2_vpc", ec2[2].TerraformType)

	s3 := reg.ByService("s3")
	require.Len(t, s3, 1)
	assert.Equal(t, "cloudmock_s3_bucket", s3[0].TerraformType)

	empty := reg.ByService("nonexistent")
	assert.Empty(t, empty)
}

func TestRegistryLen(t *testing.T) {
	reg := NewRegistry()
	assert.Equal(t, 0, reg.Len())

	reg.Add(ResourceSchema{TerraformType: "cloudmock_s3_bucket"})
	assert.Equal(t, 1, reg.Len())

	// Adding same type overwrites, not duplicates.
	reg.Add(ResourceSchema{TerraformType: "cloudmock_s3_bucket"})
	assert.Equal(t, 1, reg.Len())
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AutoScalingGroup", "auto_scaling_group"},
		{"VPC", "vpc"},
		{"DBInstance", "db_instance"},
		{"SimpleName", "simple_name"},
		{"already_snake", "already_snake"},
		{"CacheClusterId", "cache_cluster_id"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, toSnakeCase(tt.input), "toSnakeCase(%q)", tt.input)
	}
}

func TestToPascalCase(t *testing.T) {
	assert.Equal(t, "DynamoDB", toPascalCase("dynamodb"))
	assert.Equal(t, "EC2", toPascalCase("ec2"))
	assert.Equal(t, "S3", toPascalCase("s3"))
	assert.Equal(t, "AutoScaling", toPascalCase("autoscaling"))
	assert.Equal(t, "Unknownservice", toPascalCase("unknownservice"))
}

func TestIsComputedField(t *testing.T) {
	assert.True(t, isComputedField("VpcArn"))
	assert.True(t, isComputedField("arn"))
	assert.True(t, isComputedField("VpcId"))
	assert.True(t, isComputedField("CreatedAt"))
	assert.False(t, isComputedField("Name"))
	assert.False(t, isComputedField("CidrBlock"))
}
