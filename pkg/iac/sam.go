package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ImportSAMTemplate parses a SAM/CloudFormation template.yaml and extracts
// resource definitions into an IaCImportResult + DependencyGraph.
//
// Supported resource types:
//
//	AWS::DynamoDB::Table
//	AWS::Serverless::Function / AWS::Lambda::Function
//	AWS::SQS::Queue
//	AWS::SNS::Topic
//	AWS::S3::Bucket
//	AWS::Serverless::Api
func ImportSAMTemplate(path string, environment string, logger *slog.Logger) (*IaCImportResult, *DependencyGraph, error) {
	if environment == "" {
		environment = "dev"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var tpl samTemplate
	if err := yaml.Unmarshal(data, &tpl); err != nil {
		return nil, nil, err
	}

	result := &IaCImportResult{}
	graph := NewDependencyGraph()

	for logicalID, res := range tpl.Resources {
		props := res.Properties

		switch res.Type {
		case "AWS::DynamoDB::Table":
			t := parseSAMDynamoTable(logicalID, props)
			if t != nil {
				result.Tables = append(result.Tables, *t)
				graph.AddNode(DependencyNode{
					ID: "dynamodb." + t.Name, Label: t.Name,
					Type: "table", Service: "dynamodb",
				})
				logger.Info("  sam dynamodb table", "name", t.Name)
			}

		case "AWS::Serverless::Function", "AWS::Lambda::Function":
			l := parseSAMLambda(logicalID, props)
			if l != nil {
				result.Lambdas = append(result.Lambdas, *l)
				graph.AddNode(DependencyNode{
					ID: "lambda." + l.Name, Label: l.Name,
					Type: "function", Service: "lambda",
				})
				logger.Info("  sam lambda function", "name", l.Name)
			}

		case "AWS::SQS::Queue":
			name := stringProp(props, "QueueName")
			if name == "" {
				name = logicalID
			}
			result.SQSQueues = append(result.SQSQueues, SQSQueueDef{Name: name})
			graph.AddNode(DependencyNode{
				ID: "sqs." + name, Label: name,
				Type: "queue", Service: "sqs",
			})

		case "AWS::SNS::Topic":
			name := stringProp(props, "TopicName")
			if name == "" {
				name = logicalID
			}
			result.SNSTopics = append(result.SNSTopics, SNSTopicDef{Name: name})
			graph.AddNode(DependencyNode{
				ID: "sns." + name, Label: name,
				Type: "topic", Service: "sns",
			})

		case "AWS::S3::Bucket":
			name := stringProp(props, "BucketName")
			if name == "" {
				name = strings.ToLower(logicalID)
			}
			result.S3Buckets = append(result.S3Buckets, S3BucketDef{Name: name})
			graph.AddNode(DependencyNode{
				ID: "s3." + name, Label: name,
				Type: "bucket", Service: "s3",
			})
		}

		// Extract DependsOn edges.
		for _, dep := range res.DependsOn {
			graph.AddEdge(DependencyEdge{
				Source: logicalID,
				Target: dep,
				Type:   "dependsOn",
			})
		}
	}

	if len(graph.Nodes) > 0 {
		logger.Info("SAM import complete",
			"tables", len(result.Tables), "lambdas", len(result.Lambdas),
			"queues", len(result.SQSQueues), "topics", len(result.SNSTopics),
			"buckets", len(result.S3Buckets))
	}

	return result, graph, nil
}

// --- SAM/CloudFormation template types ---

type samTemplate struct {
	Resources map[string]samResource `yaml:"Resources"`
}

type samResource struct {
	Type       string         `yaml:"Type"`
	Properties map[string]any `yaml:"Properties"`
	DependsOn  samDependsOn   `yaml:"DependsOn"`
}

// samDependsOn handles both string and []string forms of DependsOn.
type samDependsOn []string

func (d *samDependsOn) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*d = []string{value.Value}
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*d = items
	}
	return nil
}

// --- Resource-specific parsers ---

func parseSAMDynamoTable(logicalID string, props map[string]any) *DynamoTableDef {
	name := stringProp(props, "TableName")
	if name == "" {
		name = logicalID
	}

	// KeySchema: [{AttributeName: pk, KeyType: HASH}, ...]
	hashKey, rangeKey := extractKeySchema(props)

	// AttributeDefinitions: [{AttributeName: pk, AttributeType: S}, ...]
	attrs := extractAttributeDefs(props)

	return &DynamoTableDef{
		Name:       name,
		HashKey:    hashKey,
		RangeKey:   rangeKey,
		Attributes: attrs,
	}
}

func parseSAMLambda(logicalID string, props map[string]any) *LambdaDef {
	name := stringProp(props, "FunctionName")
	if name == "" {
		name = logicalID
	}
	return &LambdaDef{
		Name:    name,
		Runtime: stringProp(props, "Runtime"),
		Handler: stringProp(props, "Handler"),
		Timeout: intProp(props, "Timeout"),
		Memory:  intProp(props, "MemorySize"),
	}
}

// --- YAML property helpers ---

func stringProp(props map[string]any, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intProp(props map[string]any, key string) int {
	if v, ok := props[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func extractKeySchema(props map[string]any) (hashKey, rangeKey string) {
	ks, ok := props["KeySchema"]
	if !ok {
		return
	}
	items, ok := ks.([]any)
	if !ok {
		return
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["AttributeName"].(string)
		keyType, _ := m["KeyType"].(string)
		switch keyType {
		case "HASH":
			hashKey = name
		case "RANGE":
			rangeKey = name
		}
	}
	return
}

func extractAttributeDefs(props map[string]any) []AttributeDef {
	ad, ok := props["AttributeDefinitions"]
	if !ok {
		return nil
	}
	items, ok := ad.([]any)
	if !ok {
		return nil
	}
	var attrs []AttributeDef
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["AttributeName"].(string)
		attrType, _ := m["AttributeType"].(string)
		if name != "" {
			attrs = append(attrs, AttributeDef{Name: name, Type: attrType})
		}
	}
	return attrs
}

// --- Auto-detect SAM/CFN ---

// IsSAMTemplate checks if a file looks like a SAM or CloudFormation template.
func IsSAMTemplate(path string) bool {
	base := filepath.Base(path)
	switch base {
	case "template.yaml", "template.yml", "template.json",
		"samconfig.toml", "sam.yaml", "sam.yml":
		return true
	}
	return false
}

// FindSAMTemplate searches a directory for a SAM/CloudFormation template file.
func FindSAMTemplate(dir string) string {
	candidates := []string{
		"template.yaml", "template.yml",
		"sam.yaml", "sam.yml",
	}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
