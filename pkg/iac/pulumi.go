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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

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

// IaCImportResult holds all resources extracted from IaC source.
type IaCImportResult struct {
	Tables []DynamoTableDef `json:"tables"`
	// Future: Routes, Lambdas, etc.
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

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		src := string(content)
		if !strings.Contains(src, "aws.dynamodb.Table") {
			return nil
		}

		tables := parseDynamoTables(src, environment)
		if len(tables) > 0 {
			logger.Info("found DynamoDB tables in IaC", "file", path, "count", len(tables))
			result.Tables = append(result.Tables, tables...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk pulumi dir: %w", err)
	}

	return result, nil
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

// resolveTemplateName replaces ${environmentSuffix} with -environment.
func resolveTemplateName(raw string, env string) string {
	raw = strings.ReplaceAll(raw, "${environmentSuffix}", "-"+env)
	raw = strings.ReplaceAll(raw, "${environment}", env)
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

	ctx := &service.RequestContext{
		Action:  "CreateTable",
		Service: "dynamodb",
		Body:    body,
	}

	_, err := dynamoSvc.HandleRequest(ctx)
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
