// Package pulumi provides the tfbridge configuration for the cloudmock Pulumi provider.
//
// This file maps terraform-provider-cloudmock resources and data sources to
// Pulumi resource types, organized by AWS service module.
//
// To build the full provider, add these dependencies to go.mod:
//
//	go get github.com/pulumi/pulumi-terraform-bridge/v3
//	go get github.com/pulumi/pulumi/sdk/v3
//
// Then uncomment the code below and remove the build constraint.
package pulumi

// Core resource mappings from Terraform types to Pulumi tokens.
// Format: cloudmock:<module>:<Resource>
//
// These are the explicit, stable mappings for core cloudmock resources.
// Any resources not listed here are auto-mapped by convention.
var coreResourceMappings = map[string][2]string{
	"cloudmock_s3_bucket":           {"s3", "Bucket"},
	"cloudmock_dynamodb_table":      {"dynamodb", "Table"},
	"cloudmock_vpc":                 {"ec2", "Vpc"},
	"cloudmock_subnet":              {"ec2", "Subnet"},
	"cloudmock_security_group":      {"ec2", "SecurityGroup"},
	"cloudmock_instance":            {"ec2", "Instance"},
	"cloudmock_eip":                 {"ec2", "Eip"},
	"cloudmock_internet_gateway":    {"ec2", "InternetGateway"},
	"cloudmock_nat_gateway":         {"ec2", "NatGateway"},
	"cloudmock_route_table":         {"ec2", "RouteTable"},
	"cloudmock_sqs_queue":           {"sqs", "Queue"},
	"cloudmock_sns_topic":           {"sns", "Topic"},
	"cloudmock_lambda_function":     {"lambda", "Function"},
	"cloudmock_kms_key":             {"kms", "Key"},
	"cloudmock_secret":              {"secretsmanager", "Secret"},
	"cloudmock_ssm_parameter":       {"ssm", "Parameter"},
	"cloudmock_rds_instance":        {"rds", "Instance"},
	"cloudmock_ecr_repository":      {"ecr", "Repository"},
	"cloudmock_ecs_cluster":         {"ecs", "Cluster"},
	"cloudmock_ecs_service":         {"ecs", "Service"},
	"cloudmock_ecs_task_definition": {"ecs", "TaskDefinition"},
	"cloudmock_cognito_user_pool":   {"cognito", "UserPool"},
}

// moduleMap maps Terraform resource name prefixes to Pulumi module names.
var moduleMap = map[string]string{
	"s3":             "s3",
	"dynamodb":       "dynamodb",
	"vpc":            "ec2",
	"subnet":         "ec2",
	"security":       "ec2",
	"instance":       "ec2",
	"eip":            "ec2",
	"internet":       "ec2",
	"nat":            "ec2",
	"route":          "ec2",
	"sqs":            "sqs",
	"sns":            "sns",
	"lambda":         "lambda",
	"kms":            "kms",
	"secret":         "secretsmanager",
	"ssm":            "ssm",
	"rds":            "rds",
	"ecr":            "ecr",
	"ecs":            "ecs",
	"cognito":        "cognito",
	"cloudwatch":     "cloudwatch",
	"eventbridge":    "eventbridge",
	"stepfunction":   "sfn",
	"apigateway":     "apigateway",
	"cloudformation": "cloudformation",
	"ses":            "ses",
	"kinesis":        "kinesis",
	"firehose":       "firehose",
	"route53":        "route53",
}

/*
import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	shimv2 "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfshim/sdk-v2"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	terraform "github.com/Viridian-Inc/cloudmock/providers/terraform/internal"
)

const (
	mainPkg = "cloudmock"
	mainMod = "index"
)

func makeResource(mod, res string) tokens.Type {
	return tokens.Type(fmt.Sprintf("%s:%s:%s", mainPkg, mod, res))
}

func makeDataSource(mod, res string) tokens.ModuleMember {
	return tokens.ModuleMember(fmt.Sprintf("%s:%s:get%s", mainPkg, mod, res))
}

// ProviderInfo returns the Pulumi bridge configuration for cloudmock.
// It maps terraform-provider-cloudmock resources to Pulumi types.
func ProviderInfo() tfbridge.ProviderInfo {
	p := shimv2.NewProvider(terraform.Provider())

	resources := map[string]*tfbridge.ResourceInfo{}
	for tfType, info := range coreResourceMappings {
		resources[tfType] = &tfbridge.ResourceInfo{
			Tok: makeResource(info[0], info[1]),
		}
	}

	// Auto-map any remaining resources from the Terraform schema.
	tfResources := p.ResourcesMap()
	tfResources.Range(func(key string, _ shim.Resource) bool {
		if _, ok := resources[key]; !ok {
			name := strings.TrimPrefix(key, "cloudmock_")
			parts := strings.SplitN(name, "_", 2)
			mod := mainMod
			if m, ok := moduleMap[parts[0]]; ok {
				mod = m
			}
			resName := pascalCase(name)
			if len(parts) > 1 {
				resName = pascalCase(parts[1])
			}
			resources[key] = &tfbridge.ResourceInfo{
				Tok: makeResource(mod, resName),
			}
		}
		return true
	})

	return tfbridge.ProviderInfo{
		P:           p,
		Name:        "cloudmock",
		DisplayName: "CloudMock",
		Publisher:   "neureaux",
		GitHubOrg:   "neureaux",
		Repository:  "https://github.com/Viridian-Inc/cloudmock",
		Description: "A Pulumi provider for cloudmock — local AWS service emulation.",
		Keywords:    []string{"pulumi", "cloudmock", "aws", "mock", "testing"},
		License:     "Apache-2.0",
		Homepage:    "https://github.com/Viridian-Inc/cloudmock",

		Config: map[string]*tfbridge.SchemaInfo{
			"endpoint": {
				Default: &tfbridge.DefaultInfo{
					EnvVars: []string{"CLOUDMOCK_ENDPOINT"},
					Value:   "http://localhost:4566",
				},
			},
			"region": {
				Default: &tfbridge.DefaultInfo{
					EnvVars: []string{"AWS_REGION", "AWS_DEFAULT_REGION"},
					Value:   "us-east-1",
				},
			},
		},

		Resources: resources,

		JavaScript: &tfbridge.JavaScriptInfo{
			PackageName: "@neureaux/pulumi-cloudmock",
		},
		Python: &tfbridge.PythonInfo{
			PackageName: "neureaux_pulumi_cloudmock",
		},
		Golang: &tfbridge.GolangInfo{
			ImportBasePath: "github.com/Viridian-Inc/cloudmock/providers/pulumi/sdk/go/cloudmock",
		},
		CSharp: &tfbridge.CSharpInfo{
			PackageReferences: map[string]string{
				"Pulumi": "3.*",
			},
			Namespaces: map[string]string{
				mainPkg: "CloudMock",
			},
		},
	}
}

func pascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}
*/
