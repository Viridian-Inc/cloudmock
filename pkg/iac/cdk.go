package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportCDKDir scans a CDK project directory for TypeScript files containing
// AWS CDK construct instantiations (new dynamodb.Table, new lambda.Function,
// etc.) and extracts resource definitions + a dependency graph.
//
// CDK patterns detected:
//
//	new dynamodb.Table(this, 'Id', { tableName: '...', partitionKey: { name: 'pk', type: ... } })
//	new lambda.Function(this, 'Id', { functionName: '...', runtime: ... })
//	new sqs.Queue(this, 'Id', { queueName: '...' })
//	new sns.Topic(this, 'Id', { topicName: '...' })
//	new s3.Bucket(this, 'Id', { bucketName: '...' })
func ImportCDKDir(dir string, environment string, logger *slog.Logger) (*IaCImportResult, *DependencyGraph, error) {
	if environment == "" {
		environment = "dev"
	}

	result := &IaCImportResult{}
	graph := NewDependencyGraph()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".ts") {
			return nil
		}
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".d.ts") ||
			strings.Contains(path, "cdk.out") || strings.Contains(path, ".test.") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		src := string(content)

		// Skip files that don't import CDK constructs.
		if !strings.Contains(src, "aws-cdk-lib") && !strings.Contains(src, "@aws-cdk/") {
			return nil
		}

		// DynamoDB Tables
		tables := parseCDKDynamoTables(src, environment)
		for _, t := range tables {
			result.Tables = append(result.Tables, t)
			graph.AddNode(DependencyNode{
				ID: "dynamodb." + t.Name, Label: t.Name,
				Type: "table", Service: "dynamodb",
			})
			logger.Info("  cdk dynamodb table", "name", t.Name, "file", path)
		}

		// Lambda Functions
		lambdas := parseCDKLambdas(src, environment)
		for _, l := range lambdas {
			result.Lambdas = append(result.Lambdas, l)
			graph.AddNode(DependencyNode{
				ID: "lambda." + l.Name, Label: l.Name,
				Type: "function", Service: "lambda",
			})
			logger.Info("  cdk lambda function", "name", l.Name, "file", path)
		}

		// SQS Queues
		queues := parseCDKQueues(src, environment)
		for _, q := range queues {
			result.SQSQueues = append(result.SQSQueues, q)
			graph.AddNode(DependencyNode{
				ID: "sqs." + q.Name, Label: q.Name,
				Type: "queue", Service: "sqs",
			})
		}

		// SNS Topics
		topics := parseCDKTopics(src, environment)
		for _, t := range topics {
			result.SNSTopics = append(result.SNSTopics, t)
			graph.AddNode(DependencyNode{
				ID: "sns." + t.Name, Label: t.Name,
				Type: "topic", Service: "sns",
			})
		}

		// S3 Buckets
		buckets := parseCDKBuckets(src, environment)
		for _, b := range buckets {
			result.S3Buckets = append(result.S3Buckets, b)
			graph.AddNode(DependencyNode{
				ID: "s3." + b.Name, Label: b.Name,
				Type: "bucket", Service: "s3",
			})
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	if len(result.Tables)+len(result.Lambdas)+len(result.SQSQueues)+len(result.SNSTopics)+len(result.S3Buckets) > 0 {
		logger.Info("CDK import complete",
			"tables", len(result.Tables), "lambdas", len(result.Lambdas),
			"queues", len(result.SQSQueues), "topics", len(result.SNSTopics),
			"buckets", len(result.S3Buckets))
	}

	return result, graph, nil
}

// --- CDK construct extraction ---

// cdkConstructRe matches `new MODULE.Constructor(scope, 'ID'` and captures
// the module (e.g. "dynamodb"), constructor (e.g. "Table"), and ID.
var cdkConstructRe = regexp.MustCompile(`new\s+(?:aws_)?(\w+)\.(\w+)\s*\(\s*\w+\s*,\s*['"]([^'"]+)['"]`)

// cdkPropRe helpers.
var cdkPartitionKeyRe = regexp.MustCompile(`partitionKey\s*:\s*\{\s*name\s*:\s*['"]([^'"]+)['"]`)
var cdkSortKeyRe = regexp.MustCompile(`sortKey\s*:\s*\{\s*name\s*:\s*['"]([^'"]+)['"]`)

// extractCDKConstructs finds all CDK construct instantiations and extracts
// the props body using brace-counting (handles nested objects correctly).
type cdkConstruct struct {
	module string // e.g. "dynamodb"
	class  string // e.g. "Table"
	id     string // e.g. "UsersTable"
	body   string // everything between the props { }
}

func extractCDKConstructs(src string) []cdkConstruct {
	locs := cdkConstructRe.FindAllStringSubmatchIndex(src, -1)
	var constructs []cdkConstruct

	for _, loc := range locs {
		module := src[loc[2]:loc[3]]
		class := src[loc[4]:loc[5]]
		id := src[loc[6]:loc[7]]

		// Find the props opening brace after the ID.
		rest := src[loc[7]:]
		braceIdx := strings.Index(rest, "{")
		if braceIdx < 0 {
			continue
		}
		bodyStart := loc[7] + braceIdx + 1

		// Brace-count to find the matching close.
		depth := 1
		i := bodyStart
		for i < len(src) && depth > 0 {
			switch src[i] {
			case '{':
				depth++
			case '}':
				depth--
			}
			i++
		}
		if depth != 0 {
			continue
		}
		body := src[bodyStart : i-1]

		constructs = append(constructs, cdkConstruct{
			module: strings.ToLower(module),
			class:  class,
			id:     id,
			body:   body,
		})
	}
	return constructs
}

func parseCDKDynamoTables(src, env string) []DynamoTableDef {
	var tables []DynamoTableDef
	for _, c := range extractCDKConstructs(src) {
		if c.module != "dynamodb" || c.class != "Table" {
			continue
		}
		name := extractCDKStringProp(c.body, "tableName")
		if name == "" {
			name = strings.ToLower(c.id)
		}
		hashKey := ""
		if m := cdkPartitionKeyRe.FindStringSubmatch(c.body); len(m) >= 2 {
			hashKey = m[1]
		}
		rangeKey := ""
		if m := cdkSortKeyRe.FindStringSubmatch(c.body); len(m) >= 2 {
			rangeKey = m[1]
		}
		attrs := []AttributeDef{}
		if hashKey != "" {
			attrs = append(attrs, AttributeDef{Name: hashKey, Type: "S"})
		}
		if rangeKey != "" {
			attrs = append(attrs, AttributeDef{Name: rangeKey, Type: "S"})
		}
		tables = append(tables, DynamoTableDef{
			Name: name, HashKey: hashKey, RangeKey: rangeKey, Attributes: attrs,
		})
	}
	return tables
}

func parseCDKLambdas(src, env string) []LambdaDef {
	var lambdas []LambdaDef
	for _, c := range extractCDKConstructs(src) {
		if c.module != "lambda" || c.class != "Function" {
			continue
		}
		name := extractCDKStringProp(c.body, "functionName")
		if name == "" {
			name = strings.ToLower(c.id)
		}
		runtime := extractCDKStringProp(c.body, "runtime")
		if runtime == "" {
			runtimeRe := regexp.MustCompile(`runtime\s*:\s*\w+\.Runtime\.(\w+)`)
			if m := runtimeRe.FindStringSubmatch(c.body); len(m) >= 2 {
				runtime = strings.ToLower(strings.ReplaceAll(m[1], "_", "."))
			}
		}
		handler := extractCDKStringProp(c.body, "handler")
		lambdas = append(lambdas, LambdaDef{Name: name, Runtime: runtime, Handler: handler})
	}
	return lambdas
}

func parseCDKQueues(src, env string) []SQSQueueDef {
	var queues []SQSQueueDef
	for _, c := range extractCDKConstructs(src) {
		if c.module != "sqs" || c.class != "Queue" {
			continue
		}
		name := extractCDKStringProp(c.body, "queueName")
		if name == "" {
			name = strings.ToLower(c.id)
		}
		queues = append(queues, SQSQueueDef{Name: name})
	}
	return queues
}

func parseCDKTopics(src, env string) []SNSTopicDef {
	var topics []SNSTopicDef
	for _, c := range extractCDKConstructs(src) {
		if c.module != "sns" || c.class != "Topic" {
			continue
		}
		name := extractCDKStringProp(c.body, "topicName")
		if name == "" {
			name = strings.ToLower(c.id)
		}
		topics = append(topics, SNSTopicDef{Name: name})
	}
	return topics
}

func parseCDKBuckets(src, env string) []S3BucketDef {
	var buckets []S3BucketDef
	for _, c := range extractCDKConstructs(src) {
		if c.module != "s3" || c.class != "Bucket" {
			continue
		}
		name := extractCDKStringProp(c.body, "bucketName")
		if name == "" {
			name = strings.ToLower(c.id)
		}
		buckets = append(buckets, S3BucketDef{Name: name})
	}
	return buckets
}

// extractCDKStringProp extracts a string property from a CDK props object body.
func extractCDKStringProp(body, key string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(key) + `\s*:\s*['"]([^'"]+)['"]`)
	m := re.FindStringSubmatch(body)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}
