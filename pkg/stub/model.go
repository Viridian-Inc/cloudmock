package stub

// ServiceModel describes an AWS service's API in a simplified form that the
// stub engine can use to generate a working mock at runtime.
type ServiceModel struct {
	ServiceName   string                  `json:"serviceName"`   // e.g., "ec2"
	Protocol      string                  `json:"protocol"`      // "json", "query", "rest-json", "rest-xml"
	TargetPrefix  string                  `json:"targetPrefix"`  // e.g., "AmazonEC2" for X-Amz-Target
	Actions       map[string]Action       `json:"actions"`       // action name -> definition
	ResourceTypes map[string]ResourceType `json:"resourceTypes"` // resource type name -> definition
}

// Action describes a single AWS API action.
type Action struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`         // "create", "delete", "describe", "list", "update", "tag", "untag", "listTags", "other"
	ResourceType string  `json:"resourceType"` // which resource type this acts on
	InputFields  []Field `json:"inputFields"`  // required/optional input fields
	OutputFields []Field `json:"outputFields"` // fields in the response
	IdField      string  `json:"idField"`      // which input field is the resource identifier
}

// Field describes a single field in an action's input or output.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`     // "string", "integer", "boolean", "timestamp", "list", "map"
	Required bool   `json:"required"`
}

// ResourceType describes a type of resource managed by the service.
type ResourceType struct {
	Name       string  `json:"name"`
	IdField    string  `json:"idField"`    // e.g., "VpcId"
	ArnPattern string  `json:"arnPattern"` // e.g., "arn:aws:ec2:{region}:{account}:vpc/{id}"
	Fields     []Field `json:"fields"`     // fields stored on the resource
}
