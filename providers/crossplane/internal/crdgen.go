// Package internal implements CRD generation and reconciliation for the
// native Crossplane provider for cloudmock.
package internal

import (
	"fmt"
	"strings"

	cmschema "github.com/Viridian-Inc/cloudmock/pkg/schema"
)

// CRD represents a Kubernetes CustomResourceDefinition for a Crossplane
// Managed Resource.
type CRD struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   CRDMetadata `yaml:"metadata"`
	Spec       CRDSpec     `yaml:"spec"`
}

// CRDMetadata holds CRD metadata fields.
type CRDMetadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

// CRDSpec holds the CRD spec.
type CRDSpec struct {
	Group    string       `yaml:"group"`
	Names    CRDNames     `yaml:"names"`
	Scope    string       `yaml:"scope"`
	Versions []CRDVersion `yaml:"versions"`
}

// CRDNames holds the CRD naming fields.
type CRDNames struct {
	Kind       string   `yaml:"kind"`
	ListKind   string   `yaml:"listKind"`
	Plural     string   `yaml:"plural"`
	Singular   string   `yaml:"singular"`
	ShortNames []string `yaml:"shortNames,omitempty"`
}

// CRDVersion defines a single version of the CRD.
type CRDVersion struct {
	Name    string            `yaml:"name"`
	Served  bool              `yaml:"served"`
	Storage bool              `yaml:"storage"`
	Schema  *CRDValidation    `yaml:"schema,omitempty"`
	Printer []PrinterColumn   `yaml:"additionalPrinterColumns,omitempty"`
}

// CRDValidation wraps the OpenAPI v3 schema for a CRD version.
type CRDValidation struct {
	OpenAPIV3Schema *JSONSchemaProps `yaml:"openAPIV3Schema"`
}

// PrinterColumn defines an additional column for kubectl output.
type PrinterColumn struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	JSONPath string `yaml:"jsonPath"`
}

// JSONSchemaProps is a simplified OpenAPI v3 schema object.
type JSONSchemaProps struct {
	Type        string                      `yaml:"type"`
	Description string                      `yaml:"description,omitempty"`
	Properties  map[string]*JSONSchemaProps  `yaml:"properties,omitempty"`
	Required    []string                    `yaml:"required,omitempty"`
	Items       *JSONSchemaProps             `yaml:"items,omitempty"`
	AdditionalProperties *JSONSchemaProps    `yaml:"additionalProperties,omitempty"`
	Default     any                 `yaml:"default,omitempty"`
	XPreserveUnknownFields *bool            `yaml:"x-kubernetes-preserve-unknown-fields,omitempty"`
}

// ── CRD Resource Mappings ───────────────────────────────────────────────────

// crdResourceMappings maps TerraformType -> [apiGroup, kind, shortName].
// This is consistent with the provider-metadata.yaml mappings.
var crdResourceMappings = map[string][3]string{
	"cloudmock_s3_bucket":              {"s3", "Bucket", "s3bucket"},
	"cloudmock_dynamodb_table":         {"dynamodb", "Table", "ddbtable"},
	"cloudmock_ec2_vpc":                {"ec2", "VPC", "vpc"},
	"cloudmock_ec2_subnet":             {"ec2", "Subnet", "subnet"},
	"cloudmock_ec2_security_group":     {"ec2", "SecurityGroup", "sg"},
	"cloudmock_ec2_instance":           {"ec2", "Instance", "inst"},
	"cloudmock_ec2_eip":                {"ec2", "EIP", "eip"},
	"cloudmock_ec2_internet_gateway":   {"ec2", "InternetGateway", "igw"},
	"cloudmock_ec2_nat_gateway":        {"ec2", "NATGateway", "natgw"},
	"cloudmock_ec2_route_table":        {"ec2", "RouteTable", "rt"},
	"cloudmock_sqs_queue":              {"sqs", "Queue", "sqsqueue"},
	"cloudmock_sns_topic":              {"sns", "Topic", "snstopic"},
	"cloudmock_lambda_function":        {"lambda", "Function", "fn"},
	"cloudmock_kms_key":                {"kms", "Key", "kmskey"},
	"cloudmock_secret":                 {"secretsmanager", "Secret", "secret"},
	"cloudmock_ssm_parameter":          {"ssm", "Parameter", "param"},
	"cloudmock_rds_instance":           {"rds", "DBInstance", "rds"},
	"cloudmock_ecr_repository":         {"ecr", "Repository", "ecr"},
	"cloudmock_ecs_cluster":            {"ecs", "Cluster", "ecscluster"},
	"cloudmock_ecs_service":            {"ecs", "Service", "ecssvc"},
	"cloudmock_ecs_task_definition":    {"ecs", "TaskDefinition", "taskdef"},
	"cloudmock_cognito_user_pool":      {"cognito", "UserPool", "userpool"},
}

// GenerateCRDs produces Crossplane Managed Resource CRDs from the schema registry.
func GenerateCRDs(reg *cmschema.Registry) []CRD {
	var crds []CRD
	for _, rs := range reg.All() {
		crd := generateCRD(rs)
		crds = append(crds, crd)
	}
	return crds
}

// GenerateCRD produces a single CRD for a resource schema.
func GenerateCRD(rs cmschema.ResourceSchema) CRD {
	return generateCRD(rs)
}

func generateCRD(rs cmschema.ResourceSchema) CRD {
	group, kind, shortName := resolveCRDMapping(rs)
	plural := strings.ToLower(kind) + "s"
	singular := strings.ToLower(kind)

	// Build forProvider schema (input fields).
	forProviderProps := map[string]*JSONSchemaProps{}
	var forProviderRequired []string
	for _, attr := range rs.Attributes {
		if attr.Computed && !attr.Required {
			continue // computed-only fields go in atProvider
		}
		forProviderProps[attr.Name] = attrToJSONSchema(attr)
		if attr.Required {
			forProviderRequired = append(forProviderRequired, attr.Name)
		}
	}

	// Build atProvider schema (computed/output fields).
	atProviderProps := map[string]*JSONSchemaProps{}
	for _, attr := range rs.Attributes {
		if attr.Computed {
			atProviderProps[attr.Name] = attrToJSONSchema(attr)
		}
	}
	// Always include an id field.
	atProviderProps["id"] = &JSONSchemaProps{
		Type:        "string",
		Description: "The provider-assigned unique ID for this resource.",
	}

	// Build the spec schema.
	preserveTrue := true
	specProperties := map[string]*JSONSchemaProps{
		"forProvider": {
			Type:        "object",
			Description: "Input parameters for the resource.",
			Properties:  forProviderProps,
			Required:    forProviderRequired,
		},
		"providerConfigRef": {
			Type:        "object",
			Description: "Reference to the ProviderConfig.",
			Properties: map[string]*JSONSchemaProps{
				"name": {Type: "string", Description: "Name of the ProviderConfig."},
			},
			Required: []string{"name"},
		},
		"deletionPolicy": {
			Type:        "string",
			Description: "DeletionPolicy specifies what happens to the resource when the CR is deleted.",
			Default:     "Delete",
		},
	}

	statusProperties := map[string]*JSONSchemaProps{
		"atProvider": {
			Type:        "object",
			Description: "Observed state from the external resource.",
			Properties:  atProviderProps,
		},
		"conditions": {
			Type:        "array",
			Description: "Conditions of the resource.",
			Items: &JSONSchemaProps{
				Type:                       "object",
				XPreserveUnknownFields:     &preserveTrue,
			},
		},
	}

	openAPISchema := &JSONSchemaProps{
		Type:        "object",
		Description: fmt.Sprintf("A %s is a managed resource that represents a %s.", kind, rs.AWSType),
		Properties: map[string]*JSONSchemaProps{
			"apiVersion": {Type: "string"},
			"kind":       {Type: "string"},
			"metadata":   {Type: "object"},
			"spec": {
				Type:       "object",
				Properties: specProperties,
				Required:   []string{"forProvider"},
			},
			"status": {
				Type:       "object",
				Properties: statusProperties,
			},
		},
	}

	var shortNames []string
	if shortName != "" {
		shortNames = []string{shortName}
	}

	fullGroup := group + ".cloudmock.app"

	return CRD{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Metadata: CRDMetadata{
			Name: plural + "." + fullGroup,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "cloudmock-crossplane",
			},
		},
		Spec: CRDSpec{
			Group: fullGroup,
			Names: CRDNames{
				Kind:       kind,
				ListKind:   kind + "List",
				Plural:     plural,
				Singular:   singular,
				ShortNames: shortNames,
			},
			Scope: "Cluster",
			Versions: []CRDVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema: &CRDValidation{
						OpenAPIV3Schema: openAPISchema,
					},
					Printer: []PrinterColumn{
						{Name: "READY", Type: "string", JSONPath: ".status.conditions[?(@.type=='Ready')].status"},
						{Name: "SYNCED", Type: "string", JSONPath: ".status.conditions[?(@.type=='Synced')].status"},
						{Name: "AGE", Type: "date", JSONPath: ".metadata.creationTimestamp"},
					},
				},
			},
		},
	}
}

// resolveCRDMapping returns (apiGroup, kind, shortName) for a resource schema.
func resolveCRDMapping(rs cmschema.ResourceSchema) (string, string, string) {
	if mapping, ok := crdResourceMappings[rs.TerraformType]; ok {
		return mapping[0], mapping[1], mapping[2]
	}

	// Derive from TerraformType: cloudmock_<service>_<resource> -> service, PascalCase(resource)
	name := strings.TrimPrefix(rs.TerraformType, "cloudmock_")
	group := rs.ServiceName
	resName := name
	if strings.HasPrefix(name, rs.ServiceName+"_") {
		resName = name[len(rs.ServiceName)+1:]
	}
	kind := pascalCase(resName)
	return group, kind, ""
}

// CRDGroup returns the API group for a resource schema.
func CRDGroup(rs cmschema.ResourceSchema) string {
	group, _, _ := resolveCRDMapping(rs)
	return group + ".cloudmock.app"
}

// CRDKind returns the Kind for a resource schema.
func CRDKind(rs cmschema.ResourceSchema) string {
	_, kind, _ := resolveCRDMapping(rs)
	return kind
}

// attrToJSONSchema converts a cloudmock attribute to a JSON Schema property.
func attrToJSONSchema(attr cmschema.AttributeSchema) *JSONSchemaProps {
	prop := &JSONSchemaProps{
		Description: fmt.Sprintf("The %s attribute.", attr.Name),
	}

	switch attr.Type {
	case "string":
		prop.Type = "string"
	case "int":
		prop.Type = "integer"
	case "bool":
		prop.Type = "boolean"
	case "float":
		prop.Type = "number"
	case "list", "set":
		prop.Type = "array"
		prop.Items = &JSONSchemaProps{Type: "string"}
	case "map":
		prop.Type = "object"
		prop.AdditionalProperties = &JSONSchemaProps{Type: "string"}
	default:
		prop.Type = "string"
	}

	if attr.Default != nil {
		prop.Default = attr.Default
	}

	return prop
}

// pascalCase converts snake_case to PascalCase.
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
