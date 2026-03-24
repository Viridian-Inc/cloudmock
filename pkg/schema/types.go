package schema

// ResourceSchema describes a single cloud resource for provider code generation.
type ResourceSchema struct {
	ServiceName   string
	ResourceType  string // e.g., "aws_s3_bucket"
	TerraformType string // e.g., "cloudmock_s3_bucket"
	AWSType       string // e.g., "AWS::S3::Bucket"
	Attributes    []AttributeSchema
	CreateAction  string
	ReadAction    string
	UpdateAction  string
	DeleteAction  string
	ListAction    string
	ImportID      string // which attribute is the import key
	References    []ResourceRef
}

// AttributeSchema describes a single attribute on a resource.
type AttributeSchema struct {
	Name     string
	Type     string // "string", "int", "bool", "float", "list", "map", "set"
	Required bool
	Computed bool        // server-generated (ARN, ID, timestamps)
	ForceNew bool        // changing requires replacement
	Default  any
	RefTo    string // references another resource (e.g., "cloudmock_vpc.id")
}

// ResourceRef describes a reference from one resource attribute to another.
type ResourceRef struct {
	FromAttr   string
	ToResource string
	ToAttr     string
}

// SchemaProvider is the opt-in interface for Tier 1 services that provide
// hand-crafted resource schemas.
type SchemaProvider interface {
	ResourceSchemas() []ResourceSchema
}
