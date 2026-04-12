package iac

import (
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportTerraformDir scans a Terraform project directory for .tf files and
// extracts resource definitions into an IaCImportResult. It also builds a
// DependencyGraph from explicit depends_on and implicit reference patterns.
func ImportTerraformDir(dir string, environment string, logger *slog.Logger) (*IaCImportResult, *DependencyGraph, error) {
	if environment == "" {
		environment = "dev"
	}

	result := &IaCImportResult{}
	graph := NewDependencyGraph()

	var blocks []tfResourceBlock

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".tf") {
			return nil
		}
		if strings.Contains(path, ".terraform") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		src := string(content)
		parsed := extractResourceBlocks(src)
		for _, rb := range parsed {
			rb.file = path
			blocks = append(blocks, rb)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	if len(blocks) == 0 {
		return result, graph, nil
	}

	logger.Info("found Terraform resources", "dir", dir, "blocks", len(blocks))

	// Build a map of resource ID → node for dependency resolution.
	nodeIDs := make(map[string]string) // "aws_dynamodb_table.users" → node ID

	for _, rb := range blocks {
		nodeID := rb.tfType + "." + rb.name
		service := tfTypeToService(rb.tfType)
		nodeType := tfTypeToNodeType(rb.tfType)
		label := extractTFStringAttr(rb.body, "name")
		if label == "" {
			label = rb.name
		}

		graph.AddNode(DependencyNode{
			ID:      nodeID,
			Label:   label,
			Type:    nodeType,
			Service: service,
		})
		nodeIDs[nodeID] = nodeID

		// Parse resource-specific definitions.
		switch rb.tfType {
		case "aws_dynamodb_table":
			if t := parseTFDynamoTable(rb.body, environment); t != nil {
				result.Tables = append(result.Tables, *t)
				logger.Info("  dynamodb table", "name", t.Name)
			}
		case "aws_lambda_function":
			if l := parseTFLambda(rb.body, environment); l != nil {
				result.Lambdas = append(result.Lambdas, *l)
				logger.Info("  lambda function", "name", l.Name)
			}
		case "aws_sqs_queue":
			name := extractTFStringAttr(rb.body, "name")
			if name == "" {
				name = rb.name
			}
			result.SQSQueues = append(result.SQSQueues, SQSQueueDef{Name: name})
			logger.Info("  sqs queue", "name", name)
		case "aws_sns_topic":
			name := extractTFStringAttr(rb.body, "name")
			if name == "" {
				name = rb.name
			}
			result.SNSTopics = append(result.SNSTopics, SNSTopicDef{Name: name})
			logger.Info("  sns topic", "name", name)
		case "aws_s3_bucket":
			name := extractTFStringAttr(rb.body, "bucket")
			if name == "" {
				name = rb.name
			}
			result.S3Buckets = append(result.S3Buckets, S3BucketDef{Name: name})
			logger.Info("  s3 bucket", "name", name)
		}

		// --- Dependency extraction ---

		// Explicit depends_on.
		for _, dep := range extractDependsOn(rb.body) {
			graph.AddEdge(DependencyEdge{
				Source: nodeID,
				Target: dep,
				Type:   "dependsOn",
			})
		}

		// Implicit references: scan for aws_TYPE.NAME patterns.
		for _, ref := range extractImplicitRefs(rb.body) {
			if ref != nodeID { // skip self-references
				graph.AddEdge(DependencyEdge{
					Source: nodeID,
					Target: ref,
					Type:   "reference",
				})
			}
		}
	}

	return result, graph, nil
}

// --- Resource block extraction ---

// tfResourceBlockRe matches `resource "TYPE" "NAME" { ... }` blocks.
// Uses a simple brace-counting approach for nested blocks.
var tfResourceBlockRe = regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)

type tfResourceBlock struct {
	tfType string
	name   string
	body   string
	file   string
}

func extractResourceBlocks(src string) []tfResourceBlock {
	matches := tfResourceBlockRe.FindAllStringSubmatchIndex(src, -1)
	var blocks []tfResourceBlock

	for _, loc := range matches {
		tfType := src[loc[2]:loc[3]]
		name := src[loc[4]:loc[5]]

		// Find the matching closing brace by counting.
		bodyStart := loc[1] // position after the opening {
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
			continue // malformed block
		}
		body := src[bodyStart : i-1]

		blocks = append(blocks, tfResourceBlock{
			tfType: tfType,
			name:   name,
			body:   body,
		})
	}
	return blocks
}

// --- Terraform attribute extractors ---

// extractTFStringAttr extracts a top-level string attribute: `key = "value"`.
func extractTFStringAttr(body, key string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"`)
	m := re.FindStringSubmatch(body)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// extractTFIntAttr extracts a top-level integer attribute: `key = 123`.
func extractTFIntAttr(body, key string) int {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*(\d+)`)
	m := re.FindStringSubmatch(body)
	if len(m) >= 2 {
		var v int
		for _, c := range m[1] {
			v = v*10 + int(c-'0')
		}
		return v
	}
	return 0
}

// --- DynamoDB table parsing ---

func parseTFDynamoTable(body, env string) *DynamoTableDef {
	name := extractTFStringAttr(body, "name")
	if name == "" {
		return nil
	}

	hashKey := extractTFStringAttr(body, "hash_key")
	rangeKey := extractTFStringAttr(body, "range_key")

	// Parse attribute blocks: attribute { name = "..." type = "..." }
	attrRe := regexp.MustCompile(`attribute\s*\{[^}]*name\s*=\s*"([^"]*)"[^}]*type\s*=\s*"([^"]*)"[^}]*\}`)
	attrMatches := attrRe.FindAllStringSubmatch(body, -1)
	var attrs []AttributeDef
	for _, m := range attrMatches {
		attrs = append(attrs, AttributeDef{Name: m[1], Type: m[2]})
	}

	// Parse GSIs: global_secondary_index { name = "..." hash_key = "..." ... }
	gsiRe := regexp.MustCompile(`global_secondary_index\s*\{([^}]*(?:\{[^}]*\}[^}]*)*)\}`)
	gsiMatches := gsiRe.FindAllStringSubmatch(body, -1)
	var gsis []GSIDef
	for _, m := range gsiMatches {
		gsiBody := m[1]
		gsi := GSIDef{
			Name:       extractTFStringAttr(gsiBody, "name"),
			HashKey:    extractTFStringAttr(gsiBody, "hash_key"),
			RangeKey:   extractTFStringAttr(gsiBody, "range_key"),
			Projection: extractTFStringAttr(gsiBody, "projection_type"),
		}
		if gsi.Projection == "" {
			gsi.Projection = "ALL"
		}
		if gsi.Name != "" {
			gsis = append(gsis, gsi)
		}
	}

	return &DynamoTableDef{
		Name:       name,
		HashKey:    hashKey,
		RangeKey:   rangeKey,
		Attributes: attrs,
		GSIs:       gsis,
	}
}

// --- Lambda function parsing ---

func parseTFLambda(body, env string) *LambdaDef {
	name := extractTFStringAttr(body, "function_name")
	if name == "" {
		return nil
	}
	return &LambdaDef{
		Name:    name,
		Runtime: extractTFStringAttr(body, "runtime"),
		Handler: extractTFStringAttr(body, "handler"),
		Timeout: extractTFIntAttr(body, "timeout"),
		Memory:  extractTFIntAttr(body, "memory_size"),
	}
}

// --- Dependency extraction ---

// tfDependsOnRe matches depends_on = [aws_TYPE.NAME, ...].
var tfDependsOnRe = regexp.MustCompile(`depends_on\s*=\s*\[([^\]]*)\]`)

// refTokenRe matches aws_TYPE.NAME references like aws_dynamodb_table.users.arn.
var refTokenRe = regexp.MustCompile(`(aws_[a-z_]+\.[a-z_][a-z0-9_]*)`)

func extractDependsOn(body string) []string {
	m := tfDependsOnRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return nil
	}
	items := strings.Split(m[1], ",")
	var deps []string
	for _, item := range items {
		dep := strings.TrimSpace(item)
		dep = strings.Trim(dep, `"`)
		if dep != "" {
			deps = append(deps, dep)
		}
	}
	return deps
}

func extractImplicitRefs(body string) []string {
	matches := refTokenRe.FindAllString(body, -1)
	seen := make(map[string]bool)
	var refs []string
	for _, m := range matches {
		// Strip any trailing .attribute (take only TYPE.NAME)
		parts := strings.SplitN(m, ".", 3)
		if len(parts) >= 2 {
			ref := parts[0] + "." + parts[1]
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// --- Type mapping ---

var tfServiceMap = map[string]string{
	"aws_dynamodb_table":       "dynamodb",
	"aws_lambda_function":      "lambda",
	"aws_sqs_queue":            "sqs",
	"aws_sns_topic":            "sns",
	"aws_s3_bucket":            "s3",
	"aws_iam_role":             "iam",
	"aws_iam_policy":           "iam",
	"aws_api_gateway_rest_api": "apigateway",
	"aws_apigatewayv2_api":     "apigateway",
	"aws_ecs_cluster":          "ecs",
	"aws_ecs_service":          "ecs",
	"aws_ecs_task_definition":  "ecs",
	"aws_kinesis_stream":       "kinesis",
	"aws_kms_key":              "kms",
	"aws_cognito_user_pool":    "cognito",
	"aws_route53_zone":         "route53",
	"aws_cloudwatch_log_group": "cloudwatchlogs",
	"aws_cloudwatch_metric_alarm": "cloudwatch",
	"aws_secretsmanager_secret":   "secretsmanager",
	"aws_ssm_parameter":           "ssm",
	"aws_stepfunctions_state_machine": "stepfunctions",
	"aws_eventbridge_rule":        "events",
}

var tfNodeTypeMap = map[string]string{
	"aws_dynamodb_table":       "table",
	"aws_lambda_function":      "function",
	"aws_sqs_queue":            "queue",
	"aws_sns_topic":            "topic",
	"aws_s3_bucket":            "bucket",
	"aws_iam_role":             "role",
	"aws_iam_policy":           "policy",
	"aws_api_gateway_rest_api": "api",
	"aws_apigatewayv2_api":     "api",
	"aws_ecs_cluster":          "cluster",
	"aws_ecs_service":          "service",
	"aws_kinesis_stream":       "stream",
	"aws_kms_key":              "key",
	"aws_cognito_user_pool":    "pool",
	"aws_route53_zone":         "zone",
}

func tfTypeToService(tfType string) string {
	if s, ok := tfServiceMap[tfType]; ok {
		return s
	}
	// Fallback: extract from prefix (aws_dynamodb_xxx → dynamodb)
	parts := strings.SplitN(strings.TrimPrefix(tfType, "aws_"), "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func tfTypeToNodeType(tfType string) string {
	if t, ok := tfNodeTypeMap[tfType]; ok {
		return t
	}
	return "resource"
}

// --- Auto-detection ---

// DetectIaCType returns "terraform", "pulumi", or "" based on the files
// present in the given directory.
func DetectIaCType(dir string) string {
	// Check for Terraform
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.tf")); len(matches) > 0 {
		return "terraform"
	}
	// Check for Pulumi
	if _, err := os.Stat(filepath.Join(dir, "Pulumi.yaml")); err == nil {
		return "pulumi"
	}
	if _, err := os.Stat(filepath.Join(dir, "Pulumi.yml")); err == nil {
		return "pulumi"
	}
	// Check for .ts files with aws imports (Pulumi TypeScript)
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.ts")); len(matches) > 0 {
		return "pulumi"
	}
	return ""
}

// ImportDir auto-detects the IaC type and imports resources.
func ImportDir(dir string, environment string, logger *slog.Logger) (*IaCImportResult, *DependencyGraph, error) {
	iacType := DetectIaCType(dir)
	switch iacType {
	case "terraform":
		logger.Info("detected Terraform project", "dir", dir)
		return ImportTerraformDir(dir, environment, logger)
	case "pulumi":
		logger.Info("detected Pulumi project", "dir", dir)
		result, err := ImportPulumiDir(dir, environment, logger)
		return result, nil, err // Pulumi parser doesn't build DependencyGraph yet
	default:
		logger.Warn("no IaC project detected", "dir", dir)
		return &IaCImportResult{}, NewDependencyGraph(), nil
	}
}
