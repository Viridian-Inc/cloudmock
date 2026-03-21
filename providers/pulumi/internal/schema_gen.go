package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
)

// GeneratePulumiSchema builds a Pulumi package schema JSON structure from
// the cloudmock schema registry. The resulting map can be marshaled to JSON
// and returned by the provider's GetSchema RPC.
func GeneratePulumiSchema(reg *cmschema.Registry) map[string]interface{} {
	resources := map[string]interface{}{}

	for _, rs := range reg.All() {
		token := resourceToken(rs)

		inputProps := map[string]interface{}{}
		outputProps := map[string]interface{}{}
		var requiredInputs []interface{}
		var requiredOutputs []interface{}

		for _, attr := range rs.Attributes {
			propSchema := attrToPulumiProperty(attr)

			// All attributes appear in output properties.
			outputProps[attr.Name] = propSchema

			// Non-computed attributes are input properties.
			if !attr.Computed || attr.Required {
				inputProps[attr.Name] = propSchema
			}

			if attr.Required {
				requiredInputs = append(requiredInputs, attr.Name)
				requiredOutputs = append(requiredOutputs, attr.Name)
			}
		}

		// The "id" property is always present on outputs.
		outputProps["id"] = map[string]interface{}{
			"type":        "string",
			"description": "The provider-assigned unique ID for this resource.",
		}

		resDef := map[string]interface{}{
			"description":     fmt.Sprintf("Manages a %s resource in cloudmock.", rs.AWSType),
			"inputProperties": inputProps,
			"properties":      outputProps,
		}
		if len(requiredInputs) > 0 {
			resDef["requiredInputs"] = requiredInputs
		}
		if len(requiredOutputs) > 0 {
			resDef["required"] = requiredOutputs
		}

		resources[token] = resDef
	}

	configProps := map[string]interface{}{
		"endpoint": map[string]interface{}{
			"type":        "string",
			"description": "The cloudmock gateway endpoint URL.",
			"default":     "http://localhost:4566",
		},
		"region": map[string]interface{}{
			"type":        "string",
			"description": "The AWS region for credential scope.",
			"default":     "us-east-1",
		},
		"accessKey": map[string]interface{}{
			"type":        "string",
			"description": "The access key for authenticating with cloudmock.",
			"default":     "test",
		},
		"secretKey": map[string]interface{}{
			"type":        "string",
			"description": "The secret key for authenticating with cloudmock.",
			"default":     "test",
			"secret":      true,
		},
	}

	schema := map[string]interface{}{
		"name":        "cloudmock",
		"displayName": "CloudMock",
		"version":     "0.1.0",
		"description": "A Pulumi provider for cloudmock — local AWS service emulation.",
		"keywords":    []interface{}{"pulumi", "cloudmock", "aws", "mock", "testing"},
		"homepage":    "https://github.com/neureaux/cloudmock",
		"publisher":   "neureaux",
		"license":     "Apache-2.0",
		"repository":  "https://github.com/neureaux/cloudmock",
		"config": map[string]interface{}{
			"variables": configProps,
		},
		"provider": map[string]interface{}{
			"inputProperties": configProps,
		},
		"resources": resources,
		"language": map[string]interface{}{
			"nodejs": map[string]interface{}{
				"packageName": "@neureaux/pulumi-cloudmock",
			},
			"python": map[string]interface{}{
				"packageName": "neureaux_pulumi_cloudmock",
			},
			"go": map[string]interface{}{
				"importBasePath": "github.com/neureaux/cloudmock/providers/pulumi/sdk/go/cloudmock",
			},
		},
	}

	return schema
}

// GeneratePulumiSchemaJSON returns the Pulumi schema as pretty-printed JSON bytes.
func GeneratePulumiSchemaJSON(reg *cmschema.Registry) ([]byte, error) {
	schema := GeneratePulumiSchema(reg)
	return json.MarshalIndent(schema, "", "  ")
}

// resourceToken computes the Pulumi resource token for a schema entry.
// Format: cloudmock:<service>:<PascalResource>
//
// It uses the coreResourceMappings for known resources and falls back
// to deriving the token from the TerraformType.
func resourceToken(rs cmschema.ResourceSchema) string {
	// Check core mappings first.
	if info, ok := coreResourceMappings[rs.TerraformType]; ok {
		return fmt.Sprintf("cloudmock:%s:%s", info[0], info[1])
	}

	// Derive from TerraformType: strip "cloudmock_" prefix.
	name := strings.TrimPrefix(rs.TerraformType, "cloudmock_")

	// Use the service name as the module.
	module := rs.ServiceName
	if m, ok := moduleMap[module]; ok {
		module = m
	}

	// The resource name is the TerraformType without the service prefix.
	resName := name
	if strings.HasPrefix(name, rs.ServiceName+"_") {
		resName = name[len(rs.ServiceName)+1:]
	}

	return fmt.Sprintf("cloudmock:%s:%s", module, pascalCase(resName))
}

// ResourceToken is the exported version of resourceToken, for use in tests.
func ResourceToken(rs cmschema.ResourceSchema) string {
	return resourceToken(rs)
}

// attrToPulumiProperty converts a cloudmock attribute schema to a Pulumi property definition.
func attrToPulumiProperty(attr cmschema.AttributeSchema) map[string]interface{} {
	prop := map[string]interface{}{
		"description": fmt.Sprintf("The %s attribute.", attr.Name),
	}

	switch attr.Type {
	case "string":
		prop["type"] = "string"
	case "int":
		prop["type"] = "integer"
	case "bool":
		prop["type"] = "boolean"
	case "float":
		prop["type"] = "number"
	case "list", "set":
		prop["type"] = "array"
		prop["items"] = map[string]interface{}{
			"type": "string",
		}
	case "map":
		prop["type"] = "object"
		prop["additionalProperties"] = map[string]interface{}{
			"$ref": "pulumi.json#/Any",
		}
	default:
		prop["type"] = "string"
	}

	if attr.Default != nil {
		prop["default"] = attr.Default
	}

	return prop
}

// coreResourceMappings maps Terraform resource types to [module, PascalResource] pairs.
var coreResourceMappings = map[string][2]string{
	"cloudmock_s3_bucket":           {"s3", "Bucket"},
	"cloudmock_dynamodb_table":      {"dynamodb", "Table"},
	"cloudmock_ec2_vpc":             {"ec2", "Vpc"},
	"cloudmock_ec2_subnet":          {"ec2", "Subnet"},
	"cloudmock_ec2_security_group":  {"ec2", "SecurityGroup"},
	"cloudmock_ec2_instance":        {"ec2", "Instance"},
	"cloudmock_ec2_eip":             {"ec2", "Eip"},
	"cloudmock_ec2_internet_gateway": {"ec2", "InternetGateway"},
	"cloudmock_ec2_nat_gateway":     {"ec2", "NatGateway"},
	"cloudmock_ec2_route_table":     {"ec2", "RouteTable"},
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

// moduleMap maps service name prefixes to Pulumi module names.
var moduleMap = map[string]string{
	"s3":             "s3",
	"dynamodb":       "dynamodb",
	"ec2":            "ec2",
	"sqs":            "sqs",
	"sns":            "sns",
	"lambda":         "lambda",
	"kms":            "kms",
	"secretsmanager": "secretsmanager",
	"ssm":            "ssm",
	"rds":            "rds",
	"ecr":            "ecr",
	"ecs":            "ecs",
	"cognito":        "cognito",
	"cloudwatch":     "cloudwatch",
	"eventbridge":    "eventbridge",
	"stepfunctions":  "sfn",
	"apigateway":     "apigateway",
	"cloudformation": "cloudformation",
	"ses":            "ses",
	"kinesis":        "kinesis",
	"firehose":       "firehose",
	"route53":        "route53",
}

// pascalCase converts a snake_case string to PascalCase.
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
