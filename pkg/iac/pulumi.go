// Package iac extracts resource definitions from Infrastructure-as-Code sources
// (Pulumi TypeScript, Terraform HCL) and provisions them in CloudMock.
//
// This enables CloudMock to auto-provision DynamoDB tables, API Gateway routes,
// and other resources directly from IaC source code — no seed scripts needed.
package iac

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// iacCtx creates a RequestContext for IaC provisioning calls.
func iacCtx(action, svcName string, body []byte) *service.RequestContext {
	return &service.RequestContext{
		Action:     action,
		Service:    svcName,
		Body:       body,
		Region:     "us-east-1",
		AccountID:  "000000000000",
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "000000000000", ARN: "arn:aws:iam::000000000000:root", IsRoot: true},
	}
}

// DynamoTableDef holds a parsed DynamoDB table definition from IaC source.
type DynamoTableDef struct {
	Name       string            `json:"name"`
	HashKey    string            `json:"hashKey"`
	RangeKey   string            `json:"rangeKey,omitempty"`
	Attributes []AttributeDef    `json:"attributes"`
	GSIs       []GSIDef          `json:"globalSecondaryIndexes,omitempty"`
	LSIs       []LSIDef          `json:"localSecondaryIndexes,omitempty"`
	StreamEnabled bool           `json:"streamEnabled,omitempty"`
	TTLAttribute  string         `json:"ttlAttribute,omitempty"`
}

type AttributeDef struct {
	Name string `json:"name"`
	Type string `json:"type"` // S, N, B
}

type GSIDef struct {
	Name      string `json:"name"`
	HashKey   string `json:"hashKey"`
	RangeKey  string `json:"rangeKey,omitempty"`
	Projection string `json:"projectionType"`
}

type LSIDef struct {
	Name      string `json:"name"`
	RangeKey  string `json:"rangeKey"`
	Projection string `json:"projectionType"`
}

// LambdaDef holds a parsed Lambda function definition.
type LambdaDef struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	Handler string `json:"handler"`
	Timeout int    `json:"timeout"`
	Memory  int    `json:"memory"`
}

// CognitoDef holds a parsed Cognito User Pool definition.
type CognitoDef struct {
	Name string `json:"name"`
}

// SQSQueueDef holds a parsed SQS queue definition.
type SQSQueueDef struct {
	Name string `json:"name"`
}

// SNSTopicDef holds a parsed SNS topic definition.
type SNSTopicDef struct {
	Name string `json:"name"`
}

// S3BucketDef holds a parsed S3 bucket definition.
type S3BucketDef struct {
	Name string `json:"name"`
}

// APIGatewayDef holds a parsed API Gateway definition.
type APIGatewayDef struct {
	Name string `json:"name"`
}

// MicroserviceDef holds a parsed Lambda-backed microservice with its API routes.
type MicroserviceDef struct {
	Name   string          `json:"name"`
	Routes []MicroserviceRoute `json:"routes"`
	Tables []string        `json:"tables,omitempty"` // DynamoDB tables this service accesses
}

// MicroserviceRoute is an API route (method + path).
type MicroserviceRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// IaCImportResult holds all resources extracted from IaC source.
type IaCImportResult struct {
	Tables        []DynamoTableDef  `json:"tables"`
	Lambdas       []LambdaDef       `json:"lambdas"`
	CognitoPools  []CognitoDef      `json:"cognito_pools"`
	SQSQueues     []SQSQueueDef     `json:"sqs_queues"`
	SNSTopics     []SNSTopicDef     `json:"sns_topics"`
	S3Buckets     []S3BucketDef     `json:"s3_buckets"`
	APIGateways   []APIGatewayDef   `json:"api_gateways"`
	Microservices []MicroserviceDef `json:"microservices"`
}

// ImportPulumiDir scans a Pulumi project directory for resource definitions.
// It looks for TypeScript files containing aws.dynamodb.Table constructors
// and extracts the table schemas.
func ImportPulumiDir(dir string, environment string, logger *slog.Logger) (*IaCImportResult, error) {
	if environment == "" {
		environment = "dev"
	}

	result := &IaCImportResult{}

	// Find all TypeScript files in the directory tree
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".ts") {
			return nil
		}
		// Skip node_modules and test files
		if strings.Contains(path, "node_modules") || strings.Contains(path, ".test.") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		src := string(content)

		// DynamoDB tables
		if strings.Contains(src, "aws.dynamodb.Table") {
			tables := parseDynamoTables(src, environment)
			if len(tables) > 0 {
				logger.Info("found DynamoDB tables in IaC", "file", path, "count", len(tables))
				result.Tables = append(result.Tables, tables...)
			}
		}

		// Lambda functions
		if strings.Contains(src, "aws.lambda.Function") {
			lambdas := parseLambdaFunctions(src, environment)
			if len(lambdas) > 0 {
				logger.Info("found Lambda functions in IaC", "file", path, "count", len(lambdas))
				result.Lambdas = append(result.Lambdas, lambdas...)
			}
		}

		// Cognito User Pools
		if strings.Contains(src, "aws.cognito.UserPool") {
			pools := parseCognitoPools(src, environment)
			if len(pools) > 0 {
				logger.Info("found Cognito User Pools in IaC", "file", path, "count", len(pools))
				result.CognitoPools = append(result.CognitoPools, pools...)
			}
		}

		// SQS Queues
		if strings.Contains(src, "aws.sqs.Queue") {
			queues := parseSQSQueues(src, environment)
			if len(queues) > 0 {
				logger.Info("found SQS queues in IaC", "file", path, "count", len(queues))
				result.SQSQueues = append(result.SQSQueues, queues...)
			}
		}

		// SNS Topics
		if strings.Contains(src, "aws.sns.Topic") {
			topics := parseSNSTopics(src, environment)
			if len(topics) > 0 {
				logger.Info("found SNS topics in IaC", "file", path, "count", len(topics))
				result.SNSTopics = append(result.SNSTopics, topics...)
			}
		}

		// S3 Buckets
		if strings.Contains(src, "aws.s3.Bucket") || strings.Contains(src, "aws.s3.BucketV2") {
			buckets := parseS3Buckets(src, environment)
			if len(buckets) > 0 {
				logger.Info("found S3 buckets in IaC", "file", path, "count", len(buckets))
				result.S3Buckets = append(result.S3Buckets, buckets...)
			}
		}

		// API Gateway
		if strings.Contains(src, "aws.apigateway.RestApi") || strings.Contains(src, "aws.apigatewayv2.Api") {
			apis := parseAPIGateways(src, environment)
			if len(apis) > 0 {
				logger.Info("found API Gateways in IaC", "file", path, "count", len(apis))
				result.APIGateways = append(result.APIGateways, apis...)
			}
		}

		// Lambda endpoint microservices (AutotendLambdaEndpointModuleResource)
		if strings.Contains(src, "AutotendLambdaEndpointModuleResource") && strings.Contains(src, "name:") {
			microservices := parseLambdaEndpoints(src, environment)
			if len(microservices) > 0 {
				logger.Info("found Lambda endpoint microservices in IaC", "file", path, "count", len(microservices))
				result.Microservices = append(result.Microservices, microservices...)
			}
		}

		return nil
	})

	// Also look for extractedRoutes.json in the data/ directory
	routesPath := filepath.Join(dir, "data", "extractedRoutes.json")
	if routesData, err := os.ReadFile(routesPath); err == nil {
		routes := parseExtractedRoutes(routesData)
		if len(routes) > 0 {
			logger.Info("found extracted API routes", "file", routesPath, "services", len(routes))
			// Merge routes into existing microservices, matching by normalized name
			// (handles camelCase vs snake_case: accessControl == access_control)
			existingNormalized := make(map[string]int)
			for i, ms := range result.Microservices {
				existingNormalized[normalizeName(ms.Name)] = i
			}
			for _, ms := range routes {
				norm := normalizeName(ms.Name)
				if idx, ok := existingNormalized[norm]; ok {
					result.Microservices[idx].Routes = ms.Routes
				} else {
					result.Microservices = append(result.Microservices, ms)
					existingNormalized[norm] = len(result.Microservices) - 1
				}
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("walk pulumi dir: %w", err)
	}

	// Filter out resources with unresolved template variables (${...} in name).
	// These are phantom entries from IaC patterns we can't fully resolve.
	result.Tables = filterSlice(result.Tables, func(t DynamoTableDef) bool { return !strings.Contains(t.Name, "${") })
	result.Lambdas = filterSlice(result.Lambdas, func(l LambdaDef) bool { return !strings.Contains(l.Name, "${") })
	result.CognitoPools = filterSlice(result.CognitoPools, func(c CognitoDef) bool { return !strings.Contains(c.Name, "${") })
	result.SQSQueues = filterSlice(result.SQSQueues, func(q SQSQueueDef) bool { return !strings.Contains(q.Name, "${") })
	result.SNSTopics = filterSlice(result.SNSTopics, func(t SNSTopicDef) bool { return !strings.Contains(t.Name, "${") })
	result.S3Buckets = filterSlice(result.S3Buckets, func(b S3BucketDef) bool { return !strings.Contains(b.Name, "${") })
	result.APIGateways = filterSlice(result.APIGateways, func(a APIGatewayDef) bool { return !strings.Contains(a.Name, "${") })
	result.Microservices = filterSlice(result.Microservices, func(m MicroserviceDef) bool { return !strings.Contains(m.Name, "${") })

	return result, nil
}

// filterSlice returns only the elements of s for which keep returns true.
func filterSlice[T any](s []T, keep func(T) bool) []T {
	out := s[:0]
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// parseDynamoTables extracts DynamoDB table definitions from Pulumi TypeScript source.
func parseDynamoTables(src string, environment string) []DynamoTableDef {
	var tables []DynamoTableDef

	// Match: new aws.dynamodb.Table(`name`, { ... }, { parent: this })
	// We need to extract the name and the config object.
	tablePattern := regexp.MustCompile(`new\s+aws\.dynamodb\.Table\s*\(\s*` + "`" + `([^` + "`" + `]+)` + "`" + `\s*,\s*\{`)
	matches := tablePattern.FindAllStringSubmatchIndex(src, -1)

	for _, match := range matches {
		nameStart, nameEnd := match[2], match[3]
		rawName := src[nameStart:nameEnd]

		// Resolve template literals like `membership${environmentSuffix}`
		tableName := resolveTemplateName(rawName, environment)

		// Extract the config block (everything between the outer braces)
		configStart := match[1] - 1 // The opening {
		configEnd := findMatchingBrace(src, configStart)
		if configEnd < 0 {
			continue
		}
		configBlock := src[configStart : configEnd+1]

		table := parseTableConfig(tableName, configBlock)
		if table != nil {
			tables = append(tables, *table)
		}
	}

	return tables
}

// normalizeName converts camelCase/snake_case to a canonical lowercase form for dedup.
// accessControl → accesscontrol, access_control → accesscontrol, stripeWebhook → stripewebhook
func normalizeName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", ""))
}

// extractLocalVars scans TypeScript source for const/let variable assignments
// and resolves their template literal values. Handles patterns like:
//   const resourceName = `autotend-bff-${environment}`;
func extractLocalVars(src string, env string) map[string]string {
	vars := make(map[string]string)
	// Match: const/let varName = `template-${var}`;
	pattern := regexp.MustCompile("(?:const|let)\\s+(\\w+)\\s*=\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		varName := match[1]
		value := resolveTemplateName(match[2], env)
		// Also resolve already-found vars
		for k, v := range vars {
			value = strings.ReplaceAll(value, "${"+k+"}", v)
		}
		vars[varName] = value
	}
	// Also match string assignments: const x = "value";
	strPattern := regexp.MustCompile("(?:const|let)\\s+(\\w+)\\s*=\\s*\"([^\"]+)\"")
	for _, match := range strPattern.FindAllStringSubmatch(src, -1) {
		vars[match[1]] = match[2]
	}
	return vars
}

// resolveTemplateName replaces template literals with environment-based values.
// Returns empty string if unresolvable template variables remain.
func resolveTemplateName(raw string, env string) string {
	raw = strings.ReplaceAll(raw, "${environmentSuffix}", "-"+env)
	raw = strings.ReplaceAll(raw, "${environment}", env)
	raw = strings.ReplaceAll(raw, "${env}", env)
	// Common Pulumi patterns: ${name}, ${resourceName}, ${nameSanitized}, ${bucketName}
	// If any ${...} remain, they're unresolvable — return as-is (callers filter with strings.Contains)
	return raw
}

// parseTableConfig parses a DynamoDB table config block.
func parseTableConfig(name string, block string) *DynamoTableDef {
	table := &DynamoTableDef{Name: name}

	// Extract hashKey
	table.HashKey = extractStringField(block, "hashKey")
	table.RangeKey = extractStringField(block, "rangeKey")

	// Extract attributes
	table.Attributes = extractAttributes(block)

	// Extract GSIs
	table.GSIs = extractGSIs(block)

	// Extract LSIs
	table.LSIs = extractLSIs(block)

	// Stream
	if strings.Contains(block, "streamEnabled: true") {
		table.StreamEnabled = true
	}

	// TTL
	ttlAttr := extractNestedStringField(block, "ttl", "attributeName")
	if ttlAttr != "" {
		table.TTLAttribute = ttlAttr
	}

	if table.HashKey == "" {
		return nil
	}

	return table
}

// extractStringField extracts a simple string value like: hashKey: "pk"
func extractStringField(block, field string) string {
	pattern := regexp.MustCompile(field + `:\s*"([^"]+)"`)
	match := pattern.FindStringSubmatch(block)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// extractNestedStringField extracts a field nested inside another, e.g., ttl: { attributeName: "ttl" }
func extractNestedStringField(block, outer, inner string) string {
	// Find the outer block
	outerPattern := regexp.MustCompile(outer + `:\s*\{`)
	loc := outerPattern.FindStringIndex(block)
	if loc == nil {
		return ""
	}
	braceStart := loc[1] - 1
	braceEnd := findMatchingBrace(block, braceStart)
	if braceEnd < 0 {
		return ""
	}
	innerBlock := block[braceStart : braceEnd+1]
	return extractStringField(innerBlock, inner)
}

// extractAttributes parses: attributes: [ { name: "pk", type: "S" }, ... ]
func extractAttributes(block string) []AttributeDef {
	var attrs []AttributeDef
	attrPattern := regexp.MustCompile(`\{\s*name:\s*"([^"]+)"\s*,\s*type:\s*"([^"]+)"\s*\}`)
	// Find the attributes array region
	attrStart := strings.Index(block, "attributes:")
	if attrStart < 0 {
		return nil
	}
	attrRegion := block[attrStart:]
	bracketEnd := strings.Index(attrRegion, "],")
	if bracketEnd > 0 {
		attrRegion = attrRegion[:bracketEnd+1]
	}
	matches := attrPattern.FindAllStringSubmatch(attrRegion, -1)
	for _, m := range matches {
		attrs = append(attrs, AttributeDef{Name: m[1], Type: m[2]})
	}
	return attrs
}

// extractGSIs parses: globalSecondaryIndexes: [ { name: ..., hashKey: ..., rangeKey: ..., projectionType: ... } ]
func extractGSIs(block string) []GSIDef {
	gsiStart := strings.Index(block, "globalSecondaryIndexes:")
	if gsiStart < 0 {
		return nil
	}
	// Find the opening [ after globalSecondaryIndexes:
	rest := block[gsiStart:]
	bracketStart := strings.Index(rest, "[")
	if bracketStart < 0 {
		return nil
	}
	bracketEnd := findMatchingBracket(rest, bracketStart)
	if bracketEnd < 0 {
		return nil
	}
	arrayBlock := rest[bracketStart : bracketEnd+1]

	var gsis []GSIDef
	// Find each { ... } block in the array
	for i := 0; i < len(arrayBlock); {
		braceStart := strings.Index(arrayBlock[i:], "{")
		if braceStart < 0 {
			break
		}
		braceStart += i
		braceEnd := findMatchingBrace(arrayBlock, braceStart)
		if braceEnd < 0 {
			break
		}
		gsiBlock := arrayBlock[braceStart : braceEnd+1]
		gsi := GSIDef{
			Name:       extractStringField(gsiBlock, "name"),
			HashKey:    extractStringField(gsiBlock, "hashKey"),
			RangeKey:   extractStringField(gsiBlock, "rangeKey"),
			Projection: extractStringField(gsiBlock, "projectionType"),
		}
		if gsi.Name != "" && gsi.HashKey != "" {
			gsis = append(gsis, gsi)
		}
		i = braceEnd + 1
	}
	return gsis
}

// extractLSIs parses: localSecondaryIndexes: [ { name: ..., rangeKey: ..., projectionType: ... } ]
func extractLSIs(block string) []LSIDef {
	lsiStart := strings.Index(block, "localSecondaryIndexes:")
	if lsiStart < 0 {
		return nil
	}
	rest := block[lsiStart:]
	bracketStart := strings.Index(rest, "[")
	if bracketStart < 0 {
		return nil
	}
	bracketEnd := findMatchingBracket(rest, bracketStart)
	if bracketEnd < 0 {
		return nil
	}
	arrayBlock := rest[bracketStart : bracketEnd+1]

	var lsis []LSIDef
	for i := 0; i < len(arrayBlock); {
		braceStart := strings.Index(arrayBlock[i:], "{")
		if braceStart < 0 {
			break
		}
		braceStart += i
		braceEnd := findMatchingBrace(arrayBlock, braceStart)
		if braceEnd < 0 {
			break
		}
		lsiBlock := arrayBlock[braceStart : braceEnd+1]
		lsi := LSIDef{
			Name:       extractStringField(lsiBlock, "name"),
			RangeKey:   extractStringField(lsiBlock, "rangeKey"),
			Projection: extractStringField(lsiBlock, "projectionType"),
		}
		if lsi.Name != "" && lsi.RangeKey != "" {
			lsis = append(lsis, lsi)
		}
		i = braceEnd + 1
	}
	return lsis
}

// parseLambdaEndpoints extracts AutotendLambdaEndpointModuleResource definitions.
// These are the high-level microservice definitions with name and table dependencies.
func parseLambdaEndpoints(src string, env string) []MicroserviceDef {
	var services []MicroserviceDef
	// Match: new AutotendLambdaEndpointModuleResource(`autotend-lep-NAME-env`, { name: "NAME", ...
	pattern := regexp.MustCompile(`new\s+AutotendLambdaEndpointModuleResource\s*\(\s*` + "`" + `[^` + "`" + `]+` + "`" + `\s*,\s*\{`)
	matches := pattern.FindAllStringIndex(src, -1)

	for _, match := range matches {
		blockStart := match[1] - 1
		blockEnd := findMatchingBrace(src, blockStart)
		if blockEnd < 0 {
			continue
		}
		block := src[blockStart : blockEnd+1]

		// Extract name
		name := extractStringField(block, "name")
		if name == "" {
			continue
		}

		// Extract table dependencies from allowedTables array
		var tables []string
		tablePattern := regexp.MustCompile(`tables\.(\w+)`)
		if atStart := strings.Index(block, "allowedTables:"); atStart >= 0 {
			atEnd := strings.Index(block[atStart:], "]")
			if atEnd > 0 {
				atBlock := block[atStart : atStart+atEnd]
				for _, tm := range tablePattern.FindAllStringSubmatch(atBlock, -1) {
					tables = append(tables, tm[1]+"-"+env)
				}
			}
		}

		services = append(services, MicroserviceDef{
			Name:   name,
			Tables: tables,
		})
	}
	return services
}

// parseExtractedRoutes parses the data/extractedRoutes.json file.
func parseExtractedRoutes(data []byte) []MicroserviceDef {
	var raw map[string][]struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var services []MicroserviceDef
	for name, routes := range raw {
		ms := MicroserviceDef{Name: name}
		for _, r := range routes {
			ms.Routes = append(ms.Routes, MicroserviceRoute{
				Method: r.Method,
				Path:   r.Path,
			})
		}
		services = append(services, ms)
	}
	return services
}

// parseLambdaFunctions extracts Lambda function definitions from Pulumi TypeScript.
func parseLambdaFunctions(src string, env string) []LambdaDef {
	// Pre-scan for local variable assignments like: const resourceName = `autotend-bff-${environment}`;
	localVars := extractLocalVars(src, env)

	var lambdas []LambdaDef
	pattern := regexp.MustCompile("new\\s+aws\\.lambda\\.Function\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		// Resolve local variable references like ${resourceName}
		for varName, varVal := range localVars {
			name = strings.ReplaceAll(name, "${"+varName+"}", varVal)
		}
		lambdas = append(lambdas, LambdaDef{
			Name:    name,
			Runtime: "nodejs20.x",
			Handler: "index.handler",
			Timeout: 30,
			Memory:  128,
		})
	}
	return lambdas
}

// parseCognitoPools extracts Cognito User Pool definitions.
func parseCognitoPools(src string, env string) []CognitoDef {
	var pools []CognitoDef
	pattern := regexp.MustCompile("new\\s+aws\\.cognito\\.UserPool\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		pools = append(pools, CognitoDef{Name: name})
	}
	return pools
}

// parseSQSQueues extracts SQS queue definitions.
func parseSQSQueues(src string, env string) []SQSQueueDef {
	var queues []SQSQueueDef
	pattern := regexp.MustCompile("new\\s+aws\\.sqs\\.Queue\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		queues = append(queues, SQSQueueDef{Name: name})
	}
	return queues
}

// parseSNSTopics extracts SNS topic definitions.
func parseSNSTopics(src string, env string) []SNSTopicDef {
	var topics []SNSTopicDef
	pattern := regexp.MustCompile("new\\s+aws\\.sns\\.Topic\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		topics = append(topics, SNSTopicDef{Name: name})
	}
	return topics
}

// parseS3Buckets extracts S3 bucket definitions.
func parseS3Buckets(src string, env string) []S3BucketDef {
	var buckets []S3BucketDef
	pattern := regexp.MustCompile("new\\s+aws\\.s3\\.(?:Bucket|BucketV2)\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		buckets = append(buckets, S3BucketDef{Name: name})
	}
	return buckets
}

// parseAPIGateways extracts API Gateway definitions.
func parseAPIGateways(src string, env string) []APIGatewayDef {
	var apis []APIGatewayDef
	pattern := regexp.MustCompile("new\\s+aws\\.(?:apigateway\\.RestApi|apigatewayv2\\.Api)\\s*\\(\\s*`([^`]+)`")
	for _, match := range pattern.FindAllStringSubmatch(src, -1) {
		name := resolveTemplateName(match[1], env)
		apis = append(apis, APIGatewayDef{Name: name})
	}
	return apis
}

func findMatchingBrace(s string, start int) int {
	if start >= len(s) || s[start] != '{' {
		return -1
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		case '"':
			// Skip string contents
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case '`':
			// Skip template literal
			for i++; i < len(s) && s[i] != '`'; i++ {
			}
		}
	}
	return -1
}

func findMatchingBracket(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		case '{':
			end := findMatchingBrace(s, i)
			if end < 0 {
				return -1
			}
			i = end
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		}
	}
	return -1
}

// ProvisionDynamoTables creates the parsed tables in CloudMock via its DynamoDB service.
func ProvisionDynamoTables(tables []DynamoTableDef, dynamoSvc service.Service, logger *slog.Logger) error {
	for _, table := range tables {
		if err := provisionTable(table, dynamoSvc, logger); err != nil {
			logger.Warn("failed to provision table", "table", table.Name, "error", err)
		}
	}
	return nil
}

func provisionTable(table DynamoTableDef, dynamoSvc service.Service, logger *slog.Logger) error {
	// Build the CreateTable request body matching AWS DynamoDB API format.
	req := map[string]interface{}{
		"TableName":            table.Name,
		"BillingMode":          "PAY_PER_REQUEST",
	}

	// Key schema
	keySchema := []map[string]string{
		{"AttributeName": table.HashKey, "KeyType": "HASH"},
	}
	if table.RangeKey != "" {
		keySchema = append(keySchema, map[string]string{"AttributeName": table.RangeKey, "KeyType": "RANGE"})
	}
	req["KeySchema"] = keySchema

	// Attribute definitions
	attrDefs := make([]map[string]string, len(table.Attributes))
	for i, attr := range table.Attributes {
		attrDefs[i] = map[string]string{"AttributeName": attr.Name, "AttributeType": attr.Type}
	}
	req["AttributeDefinitions"] = attrDefs

	// GSIs
	if len(table.GSIs) > 0 {
		gsis := make([]map[string]interface{}, len(table.GSIs))
		for i, gsi := range table.GSIs {
			ks := []map[string]string{{"AttributeName": gsi.HashKey, "KeyType": "HASH"}}
			if gsi.RangeKey != "" {
				ks = append(ks, map[string]string{"AttributeName": gsi.RangeKey, "KeyType": "RANGE"})
			}
			gsis[i] = map[string]interface{}{
				"IndexName": gsi.Name,
				"KeySchema": ks,
				"Projection": map[string]string{"ProjectionType": gsi.Projection},
			}
		}
		req["GlobalSecondaryIndexes"] = gsis
	}

	// LSIs
	if len(table.LSIs) > 0 {
		lsis := make([]map[string]interface{}, len(table.LSIs))
		for i, lsi := range table.LSIs {
			ks := []map[string]string{
				{"AttributeName": table.HashKey, "KeyType": "HASH"},
				{"AttributeName": lsi.RangeKey, "KeyType": "RANGE"},
			}
			lsis[i] = map[string]interface{}{
				"IndexName": lsi.Name,
				"KeySchema": ks,
				"Projection": map[string]string{"ProjectionType": lsi.Projection},
			}
		}
		req["LocalSecondaryIndexes"] = lsis
	}

	body, _ := json.Marshal(req)

	_, err := dynamoSvc.HandleRequest(iacCtx("CreateTable", "dynamodb", body))
	if err != nil {
		// Ignore "already exists" errors
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "ResourceInUseException") {
			return nil
		}
		return err
	}

	logger.Info("provisioned table from IaC", "table", table.Name, "hashKey", table.HashKey, "rangeKey", table.RangeKey, "gsis", len(table.GSIs))
	return nil
}

// ProvisionLambdas creates Lambda functions in CloudMock.
func ProvisionLambdas(lambdas []LambdaDef, lambdaSvc service.Service, accountID, region string, logger *slog.Logger) {
	for _, fn := range lambdas {
		if strings.Contains(fn.Name, "${") {
			continue // skip unresolved template literals
		}
		body, _ := json.Marshal(map[string]any{
			"FunctionName": fn.Name,
			"Runtime":      fn.Runtime,
			"Handler":      fn.Handler,
			"Role":         fmt.Sprintf("arn:aws:iam::%s:role/%s-role", accountID, fn.Name),
			"Code":         map[string]any{"ZipFile": "UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA=="},
			"Timeout":      fn.Timeout,
			"MemorySize":   fn.Memory,
		})
		// Lambda uses REST path routing: POST /2015-03-31/functions
		req := httptest.NewRequest(http.MethodPost, "/2015-03-31/functions", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		ctx := &service.RequestContext{
			Action: "CreateFunction", Service: "lambda", Body: body,
			Region: region, AccountID: accountID, RawRequest: req,
			Identity: &service.CallerIdentity{AccountID: accountID, ARN: "arn:aws:iam::" + accountID + ":root", IsRoot: true},
		}
		if _, err := lambdaSvc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision Lambda", "name", fn.Name, "error", err)
		} else {
			logger.Info("provisioned Lambda from IaC", "name", fn.Name)
		}
	}
}

// ProvisionCognitoPools creates Cognito User Pools in CloudMock.
func ProvisionCognitoPools(pools []CognitoDef, cognitoSvc service.Service, logger *slog.Logger) {
	for _, pool := range pools {
		body, _ := json.Marshal(map[string]any{"PoolName": pool.Name})
		ctx := iacCtx("CreateUserPool", "cognito-idp", body)
		if _, err := cognitoSvc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision Cognito pool", "name", pool.Name, "error", err)
		} else {
			logger.Info("provisioned Cognito pool from IaC", "name", pool.Name)
		}
	}
}

// ProvisionSQSQueues creates SQS queues in CloudMock.
func ProvisionSQSQueues(queues []SQSQueueDef, sqsSvc service.Service, logger *slog.Logger) {
	for _, q := range queues {
		if strings.Contains(q.Name, "${") {
			continue
		}
		// SQS uses JSON with QueueName
		body, _ := json.Marshal(map[string]any{"QueueName": q.Name})
		ctx := iacCtx("CreateQueue", "sqs", body)
		if _, err := sqsSvc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision SQS queue", "name", q.Name, "error", err)
		} else {
			logger.Info("provisioned SQS queue from IaC", "name", q.Name)
		}
	}
}

// ProvisionSNSTopics creates SNS topics in CloudMock.
func ProvisionSNSTopics(topics []SNSTopicDef, snsSvc service.Service, logger *slog.Logger) {
	for _, t := range topics {
		if strings.Contains(t.Name, "${") {
			continue
		}
		// SNS CreateTopic uses form-encoded: Action=CreateTopic&Name=xxx
		formBody := "Action=CreateTopic&Name=" + t.Name
		ctx := iacCtx("CreateTopic", "sns", []byte(formBody))
		if _, err := snsSvc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision SNS topic", "name", t.Name, "error", err)
		} else {
			logger.Info("provisioned SNS topic from IaC", "name", t.Name)
		}
	}
}

// ProvisionS3Buckets creates S3 buckets in CloudMock.
func ProvisionS3Buckets(buckets []S3BucketDef, s3Svc service.Service, logger *slog.Logger) {
	for _, b := range buckets {
		if strings.Contains(b.Name, "${") {
			continue
		}
		// S3 CreateBucket uses PUT /{bucket} with empty body
		req := httptest.NewRequest(http.MethodPut, "/"+b.Name, nil)
		ctx := &service.RequestContext{
			Action: "CreateBucket", Service: "s3", Body: nil,
			RawRequest: req, Params: map[string]string{"bucket": b.Name},
			Identity: &service.CallerIdentity{AccountID: "000000000000", ARN: "arn:aws:iam::000000000000:root", IsRoot: true},
		}
		if _, err := s3Svc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision S3 bucket", "name", b.Name, "error", err)
		} else {
			logger.Info("provisioned S3 bucket from IaC", "name", b.Name)
		}
	}
}

// ProvisionAPIGateways creates API Gateway REST APIs in CloudMock.
func ProvisionAPIGateways(apis []APIGatewayDef, apigwSvc service.Service, logger *slog.Logger) {
	for _, api := range apis {
		if strings.Contains(api.Name, "${") {
			continue
		}
		body, _ := json.Marshal(map[string]any{"name": api.Name})
		req := httptest.NewRequest(http.MethodPost, "/restapis", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		ctx := &service.RequestContext{
			Action: "CreateRestApi", Service: "apigateway", Body: body,
			RawRequest: req,
			Identity: &service.CallerIdentity{AccountID: "000000000000", ARN: "arn:aws:iam::000000000000:root", IsRoot: true},
		}
		if _, err := apigwSvc.HandleRequest(ctx); err != nil && !isAlreadyExists(err) {
			logger.Warn("failed to provision API Gateway", "name", api.Name, "error", err)
		} else {
			logger.Info("provisioned API Gateway from IaC", "name", api.Name)
		}
	}
}

func isAlreadyExists(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "ResourceInUseException") ||
		strings.Contains(msg, "ConflictException") ||
		strings.Contains(msg, "AlreadyExists") ||
		strings.Contains(msg, "BucketAlreadyOwnedByYou")
}

// mapResourceType converts an AWS service name and resource class to a simplified type string.
func mapResourceType(svc, resClass string) string {
	switch strings.ToLower(svc) {
	case "dynamodb":
		return "table"
	case "lambda":
		return "function"
	case "sqs":
		return "queue"
	case "sns":
		return "topic"
	case "s3":
		return "bucket"
	case "cognito":
		return "userpool"
	case "apigateway", "apigatewayv2":
		return "api"
	}
	// Fallback: use the resource class name lowercased
	return strings.ToLower(resClass)
}

// componentClassRe matches: class X extends pulumi.ComponentResource {
var componentClassRe = regexp.MustCompile(`class\s+(\w+)\s+extends\s+pulumi\.ComponentResource`)

// awsResourceRe matches: new aws.SERVICE.RESOURCE("name", {...}, { ... })
// It captures service, resource class, and the resource name (double-quoted or backtick).
var awsResourceRe = regexp.MustCompile("new\\s+aws\\.(\\w+)\\.(\\w+)\\s*\\(\\s*(?:`([^`]+)`|\"([^\"]+)\")\\s*,")

// dependsOnRe matches: dependsOn: [this.foo] inside an options block.
var dependsOnRe = regexp.MustCompile(`dependsOn:\s*\[([^\]]+)\]`)

// dependsOnRefRe matches references like this.fieldName inside a dependsOn list.
var dependsOnRefRe = regexp.MustCompile(`this\.(\w+)`)

// ExtractDependencyGraph parses Pulumi TypeScript source and returns a DependencyGraph
// with module nodes (ComponentResource classes) and resource nodes (aws.* constructors),
// plus parent and dependsOn edges.
func ExtractDependencyGraph(src string, env string) *DependencyGraph {
	g := NewDependencyGraph()

	// --- Pass 1: Find ComponentResource classes and their boundaries ---
	type moduleBlock struct {
		id    string
		label string
		start int
		end   int
	}
	var modules []moduleBlock

	classMatches := componentClassRe.FindAllStringSubmatchIndex(src, -1)
	for _, cm := range classMatches {
		className := src[cm[2]:cm[3]]
		moduleID := "module:" + className

		// Find the opening brace of the class body.
		braceStart := strings.Index(src[cm[1]:], "{")
		if braceStart < 0 {
			continue
		}
		braceStart += cm[1]
		braceEnd := findMatchingBrace(src, braceStart)
		if braceEnd < 0 {
			continue
		}

		g.AddNode(DependencyNode{
			ID:      moduleID,
			Label:   className,
			Type:    "module",
			Service: "pulumi",
		})

		modules = append(modules, moduleBlock{
			id:    moduleID,
			label: className,
			start: braceStart,
			end:   braceEnd,
		})
	}

	// --- Pass 2: Find aws.* resource constructors ---
	// Track field → nodeID mapping per module for dependsOn resolution.
	// fieldAssignRe matches: this.FIELD = new aws...
	fieldAssignRe := regexp.MustCompile(`this\.(\w+)\s*=\s*new\s+aws\.(\w+)\.(\w+)\s*\(`)

	// nodesByField[moduleID][fieldName] = nodeID
	nodesByField := make(map[string]map[string]string)

	allResourceMatches := awsResourceRe.FindAllStringSubmatchIndex(src, -1)
	for _, rm := range allResourceMatches {
		svc := src[rm[2]:rm[3]]
		resClass := src[rm[4]:rm[5]]

		// Determine the resource name (backtick or double-quoted).
		var rawName string
		if rm[6] >= 0 {
			rawName = src[rm[6]:rm[7]] // backtick
		} else if rm[8] >= 0 {
			rawName = src[rm[8]:rm[9]] // double-quoted
		}
		if rawName == "" {
			continue
		}
		resName := resolveTemplateName(rawName, env)
		if strings.Contains(resName, "${") {
			continue // unresolvable template variable
		}

		resType := mapResourceType(svc, resClass)
		nodeID := resType + ":" + resName

		// Find which module contains this resource (if any).
		matchPos := rm[0]
		parentModuleID := ""
		for _, mod := range modules {
			if matchPos > mod.start && matchPos < mod.end {
				parentModuleID = mod.id
				break
			}
		}

		g.AddNode(DependencyNode{
			ID:      nodeID,
			Label:   resName,
			Type:    resType,
			Service: svc,
			Parent:  parentModuleID,
		})

		// Record field assignment for dependsOn resolution.
		if parentModuleID != "" {
			// Search backwards from the resource constructor for a this.FIELD = assignment.
			// Look within a small window before the match.
			windowStart := matchPos
			if windowStart > 200 {
				windowStart = matchPos - 200
			}
			window := src[windowStart:matchPos]
			if fa := fieldAssignRe.FindStringSubmatch(window); fa != nil {
				fieldName := fa[1]
				if nodesByField[parentModuleID] == nil {
					nodesByField[parentModuleID] = make(map[string]string)
				}
				nodesByField[parentModuleID][fieldName] = nodeID
			}
		}
	}

	// --- Pass 3: Extract dependsOn edges ---
	// Find the options block (3rd argument) of each aws.* constructor.
	// Pattern: new aws.SVC.RES(`name`, { config }, { options })
	// We look for dependsOn: [...] inside the options block.
	fullConstructorRe := regexp.MustCompile("new\\s+aws\\.(\\w+)\\.(\\w+)\\s*\\(\\s*(?:`[^`]+`|\"[^\"]+\")\\s*,\\s*\\{")
	constructorMatches := fullConstructorRe.FindAllStringSubmatchIndex(src, -1)
	for _, cm := range constructorMatches {
		svc := src[cm[2]:cm[3]]
		resClass := src[cm[4]:cm[5]]

		// The config block starts at the last {
		configStart := cm[1] - 1
		configEnd := findMatchingBrace(src, configStart)
		if configEnd < 0 {
			continue
		}

		// After configEnd, look for an options block: , { ... }
		afterConfig := src[configEnd+1:]
		commaIdx := strings.Index(afterConfig, ",")
		if commaIdx < 0 {
			continue
		}
		braceIdx := strings.Index(afterConfig[commaIdx:], "{")
		if braceIdx < 0 {
			continue
		}
		optStart := configEnd + 1 + commaIdx + braceIdx
		optEnd := findMatchingBrace(src, optStart)
		if optEnd < 0 {
			continue
		}
		optBlock := src[optStart : optEnd+1]

		// Check for dependsOn.
		doMatch := dependsOnRe.FindStringSubmatch(optBlock)
		if doMatch == nil {
			continue
		}
		depList := doMatch[1]

		// Determine the resource name to find the source node ID.
		// Re-parse from the constructor match start.
		constructorSnippet := src[cm[0] : cm[1]+1]
		nameRe := regexp.MustCompile("(?:`([^`]+)`|\"([^\"]+)\")")
		nameMatch := nameRe.FindStringSubmatch(constructorSnippet)
		if nameMatch == nil {
			continue
		}
		rawName := nameMatch[1]
		if rawName == "" {
			rawName = nameMatch[2]
		}
		resName := resolveTemplateName(rawName, env)
		resType := mapResourceType(svc, resClass)
		sourceID := resType + ":" + resName

		// Find parent module to resolve field names.
		matchPos := cm[0]
		parentModuleID := ""
		for _, mod := range modules {
			if matchPos > mod.start && matchPos < mod.end {
				parentModuleID = mod.id
				break
			}
		}

		// Extract this.FIELD references.
		refs := dependsOnRefRe.FindAllStringSubmatch(depList, -1)
		for _, ref := range refs {
			fieldName := ref[1]
			targetID := ""
			if parentModuleID != "" {
				if nf, ok := nodesByField[parentModuleID]; ok {
					targetID = nf[fieldName]
				}
			}
			if targetID == "" {
				// Fallback: use field name as a best-effort target.
				targetID = "unknown:" + fieldName
			}
			g.AddEdge(DependencyEdge{
				Source: sourceID,
				Target: targetID,
				Type:   "dependsOn",
			})
		}
	}

	// --- Fallback: flat resources (no ComponentResource) ---
	// If no modules were found, resources are already added without a parent above.

	if len(g.Nodes) == 0 {
		return nil
	}
	return g
}

// ExtractDependencyGraphFromDir scans a Pulumi project directory and builds a DependencyGraph.
func ExtractDependencyGraphFromDir(dir string, environment string) *DependencyGraph {
	merged := NewDependencyGraph()

	// Check modules subdirectory first, then root
	patterns := []string{
		filepath.Join(dir, "modules", "*.ts"),
		filepath.Join(dir, "*.ts"),
	}

	var allFiles []string
	for _, pattern := range patterns {
		files, _ := filepath.Glob(pattern)
		allFiles = append(allFiles, files...)
	}

	for _, f := range allFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		g := ExtractDependencyGraph(string(data), environment)
		if g != nil {
			merged.Nodes = append(merged.Nodes, g.Nodes...)
			merged.Edges = append(merged.Edges, g.Edges...)
		}
	}

	if len(merged.Nodes) == 0 {
		return nil
	}
	return merged
}
