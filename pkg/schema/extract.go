package schema

import (
	"strings"
	"unicode"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/stub"
)

// ExtractFromServices takes a list of service.Service instances and extracts
// ResourceSchemas from those that implement SchemaProvider. Services that do
// not implement SchemaProvider are silently skipped.
func ExtractFromServices(services []service.Service) []ResourceSchema {
	var schemas []ResourceSchema
	for _, svc := range services {
		if sp, ok := svc.(SchemaProvider); ok {
			schemas = append(schemas, sp.ResourceSchemas()...)
		}
	}
	return schemas
}

// ExtractFromStubModels converts stub ServiceModel definitions to ResourceSchemas.
func ExtractFromStubModels(models []*stub.ServiceModel, accountID, region string) []ResourceSchema {
	var schemas []ResourceSchema

	for _, model := range models {
		for rtKey, resType := range model.ResourceTypes {
			rs := ResourceSchema{
				ServiceName:   model.ServiceName,
				ResourceType:  "aws_" + model.ServiceName + "_" + toSnakeCase(resType.Name),
				TerraformType: "cloudmock_" + model.ServiceName + "_" + toSnakeCase(resType.Name),
				AWSType:       "AWS::" + toPascalCase(model.ServiceName) + "::" + resType.Name,
				ImportID:      resType.IdField,
			}

			// Map fields to attributes.
			for _, field := range resType.Fields {
				attr := AttributeSchema{
					Name:     toSnakeCase(field.Name),
					Type:     mapStubFieldType(field.Type),
					Required: field.Required,
					Computed: isComputedField(field.Name),
				}
				rs.Attributes = append(rs.Attributes, attr)
			}

			// Map actions to CRUD operations.
			for _, action := range model.Actions {
				if action.ResourceType != rtKey {
					continue
				}
				switch action.Type {
				case "create":
					rs.CreateAction = action.Name
				case "describe":
					rs.ReadAction = action.Name
				case "list":
					rs.ListAction = action.Name
				case "update":
					rs.UpdateAction = action.Name
				case "delete":
					rs.DeleteAction = action.Name
				}
			}

			schemas = append(schemas, rs)
		}
	}

	return schemas
}

// ExtractFromActions creates basic schemas from a service's Actions() method
// for Tier 1 services that haven't implemented SchemaProvider yet.
// It infers one resource per service by looking for Create+Describe+Delete action patterns.
func ExtractFromActions(svc service.Service, accountID, region string) []ResourceSchema {
	actions := svc.Actions()
	name := svc.Name()

	// Group actions by inferred resource type based on action name prefixes.
	// e.g., "CreateVpc" / "DescribeVpcs" / "DeleteVpc" -> "Vpc"
	type resourceActions struct {
		create   string
		read     string
		update   string
		delete   string
		list     string
	}

	resources := map[string]*resourceActions{}

	for _, a := range actions {
		resName, verb := parseActionName(a.Name)
		if resName == "" {
			continue
		}
		ra, ok := resources[resName]
		if !ok {
			ra = &resourceActions{}
			resources[resName] = ra
		}
		switch verb {
		case "create":
			ra.create = a.Name
		case "describe":
			ra.read = a.Name
		case "list":
			ra.list = a.Name
		case "update", "modify":
			ra.update = a.Name
		case "delete", "terminate":
			ra.delete = a.Name
		}
	}

	var schemas []ResourceSchema
	for resName, ra := range resources {
		// Only emit a schema if we have at least create and delete (or read).
		if ra.create == "" {
			continue
		}
		rs := ResourceSchema{
			ServiceName:   name,
			ResourceType:  "aws_" + name + "_" + toSnakeCase(resName),
			TerraformType: "cloudmock_" + name + "_" + toSnakeCase(resName),
			AWSType:       "AWS::" + toPascalCase(name) + "::" + resName,
			CreateAction:  ra.create,
			ReadAction:    ra.read,
			UpdateAction:  ra.update,
			DeleteAction:  ra.delete,
			ListAction:    ra.list,
		}
		schemas = append(schemas, rs)
	}

	return schemas
}

// parseActionName splits an AWS action name like "CreateVpc" into ("Vpc", "create"),
// "DescribeVpcs" into ("Vpc", "describe"), "RunInstances" into ("Instance", "create"), etc.
func parseActionName(name string) (resource string, verb string) {
	prefixes := []struct {
		prefix string
		verb   string
	}{
		{"Create", "create"},
		{"Run", "create"},       // EC2 RunInstances
		{"Allocate", "create"},  // EC2 AllocateAddress
		{"Describe", "describe"},
		{"List", "list"},
		{"Delete", "delete"},
		{"Terminate", "delete"}, // EC2 TerminateInstances
		{"Release", "delete"},   // EC2 ReleaseAddress
		{"Update", "update"},
		{"Modify", "update"},
		{"Put", "create"},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(name, p.prefix) {
			res := name[len(p.prefix):]
			// Strip trailing "s" for plural describe/list actions (e.g., DescribeVpcs -> Vpc).
			if (p.verb == "describe" || p.verb == "list") && len(res) > 1 && strings.HasSuffix(res, "s") && !strings.HasSuffix(res, "ss") && !strings.HasSuffix(res, "us") {
				res = res[:len(res)-1]
			}
			// Strip trailing "es" that remains (e.g., "Addresses" -> "Addresse" -> "Address" already handled,
			// but "Instances" -> "Instance" is handled above).
			return res, p.verb
		}
	}

	return "", ""
}

// isComputedField returns true if the field name suggests a server-generated value.
func isComputedField(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "arn") ||
		strings.HasSuffix(lower, "id") ||
		strings.Contains(lower, "created")
}

// mapStubFieldType converts stub field types to schema attribute types.
func mapStubFieldType(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "timestamp":
		return "string"
	case "list":
		return "list"
	case "map":
		return "map"
	default:
		return "string"
	}
}

// toSnakeCase converts PascalCase or camelCase to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				// Insert underscore before uppercase if preceded by lowercase,
				// or if preceded by uppercase followed by lowercase (e.g., "DBInstance" -> "db_instance").
				if unicode.IsLower(prev) {
					result = append(result, '_')
				} else if unicode.IsUpper(prev) && i+1 < len(s) && unicode.IsLower(rune(s[i+1])) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// toPascalCase converts a lowercase service name to PascalCase.
// It handles common AWS service name patterns.
func toPascalCase(s string) string {
	// Handle known multi-word service names.
	known := map[string]string{
		"autoscaling":          "AutoScaling",
		"cloudformation":       "CloudFormation",
		"cloudwatch":           "CloudWatch",
		"cloudwatchlogs":       "CloudWatchLogs",
		"dynamodb":             "DynamoDB",
		"ec2":                  "EC2",
		"ecr":                  "ECR",
		"ecs":                  "ECS",
		"elasticache":          "ElastiCache",
		"elasticbeanstalk":     "ElasticBeanstalk",
		"elasticloadbalancing": "ElasticLoadBalancing",
		"eventbridge":          "EventBridge",
		"firehose":             "Firehose",
		"kinesis":              "Kinesis",
		"kms":                  "KMS",
		"lambda":               "Lambda",
		"rds":                  "RDS",
		"route53":              "Route53",
		"s3":                   "S3",
		"secretsmanager":       "SecretsManager",
		"ses":                  "SES",
		"sns":                  "SNS",
		"sqs":                  "SQS",
		"ssm":                  "SSM",
		"stepfunctions":        "StepFunctions",
		"sts":                  "STS",
		"cognito":              "Cognito",
		"apigateway":           "ApiGateway",
		"neptune":              "Neptune",
		"redshift":             "Redshift",
	}
	if v, ok := known[s]; ok {
		return v
	}
	// Fallback: capitalize first letter.
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
