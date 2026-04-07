package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	tfschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cmschema "github.com/Viridian-Inc/cloudmock/pkg/schema"
)

// serviceProtocols maps service names to their wire protocol.
// This determines how the Terraform provider sends requests to cloudmock.
var serviceProtocols = map[string]string{
	// JSON protocol (X-Amz-Target + JSON body)
	"dynamodb":       "json",
	"kms":            "json",
	"kinesis":        "json",
	"firehose":       "json",
	"stepfunctions":  "json",
	"secretsmanager": "json",
	"logs":           "json",
	"events":         "json",
	"ecr":            "json",
	"ecs":            "json",
	"cognito":        "json",

	// Query protocol (form-encoded Action parameter)
	"ec2":              "query",
	"sqs":              "query",
	"sns":              "query",
	"rds":              "query",
	"cloudformation":   "query",
	"cloudwatch":       "query",
	"autoscaling":      "query",
	"elasticache":      "query",
	"route53":          "query",
	"ssm":              "query",
	"ses":              "query",
	"sts":              "query",

	// REST protocol (S3, API Gateway)
	"s3":           "rest",
	"apigateway":   "rest",
	"lambda":       "rest",
}

// targetPrefixes maps service names to their X-Amz-Target prefix (for JSON protocol).
var targetPrefixes = map[string]string{
	"dynamodb":       "DynamoDB_20120810",
	"kms":            "TrentService",
	"kinesis":        "Kinesis_20131202",
	"firehose":       "Firehose_20150804",
	"stepfunctions":  "AWSStepFunctions",
	"secretsmanager": "secretsmanager",
	"logs":           "Logs_20140328",
	"events":         "AWSEvents",
	"ecr":            "AmazonEC2ContainerRegistry_V20150921",
	"ecs":            "AmazonEC2ContainerServiceV20141113",
	"cognito":        "AWSCognitoIdentityProviderService",
}

// buildResource creates a Terraform resource definition from a schema registry entry.
func buildResource(rs cmschema.ResourceSchema) *tfschema.Resource {
	hasUpdate := rs.UpdateAction != ""

	return &tfschema.Resource{
		CreateContext: makeCreate(rs),
		ReadContext:   makeRead(rs),
		UpdateContext: makeUpdate(rs),
		DeleteContext: makeDelete(rs),
		Schema:        buildTFSchema(rs, hasUpdate),
		Importer: &tfschema.ResourceImporter{
			StateContext: tfschema.ImportStatePassthroughContext,
		},
	}
}

// buildTFSchema converts a list of AttributeSchemas to a Terraform schema map.
// When hasUpdate is false, all non-computed attributes must have ForceNew set,
// because Terraform requires it when no UpdateContext is defined.
func buildTFSchema(rs cmschema.ResourceSchema, hasUpdate bool) map[string]*tfschema.Schema {
	tfSchemaMap := make(map[string]*tfschema.Schema)

	for _, attr := range rs.Attributes {
		s := &tfschema.Schema{
			Type:        mapAttrType(attr.Type),
			Description: fmt.Sprintf("The %s of the %s.", attr.Name, rs.ResourceType),
		}

		switch {
		case attr.Computed && !attr.Required:
			s.Computed = true
		case attr.Required:
			s.Required = true
		default:
			s.Optional = true
		}

		if attr.ForceNew {
			s.ForceNew = true
		}

		// When no update action is defined, Terraform requires all mutable
		// (non-computed) attributes to have ForceNew set.
		if !hasUpdate && !attr.Computed {
			s.ForceNew = true
		}

		if attr.Default != nil && !attr.Required && !attr.Computed {
			s.Default = attr.Default
		}

		// For list and set types, define the element schema.
		if attr.Type == "list" || attr.Type == "set" {
			s.Elem = &tfschema.Schema{Type: tfschema.TypeString}
		}
		if attr.Type == "map" {
			s.Elem = &tfschema.Schema{Type: tfschema.TypeString}
		}

		tfSchemaMap[attr.Name] = s
	}

	return tfSchemaMap
}

// mapAttrType converts a schema attribute type string to a Terraform schema type.
func mapAttrType(t string) tfschema.ValueType {
	switch t {
	case "string":
		return tfschema.TypeString
	case "int":
		return tfschema.TypeInt
	case "bool":
		return tfschema.TypeBool
	case "float":
		return tfschema.TypeFloat
	case "list":
		return tfschema.TypeList
	case "set":
		return tfschema.TypeSet
	case "map":
		return tfschema.TypeMap
	default:
		return tfschema.TypeString
	}
}

// makeCreate returns a CreateContext function for the given resource schema.
func makeCreate(rs cmschema.ResourceSchema) tfschema.CreateContextFunc {
	return func(ctx context.Context, d *tfschema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*APIClient)

		protocol := serviceProtocols[rs.ServiceName]
		if protocol == "" {
			protocol = "json" // default to JSON
		}

		var result map[string]any
		var err error

		switch protocol {
		case "json":
			params := readJSONParams(rs, d)
			prefix := targetPrefixes[rs.ServiceName]
			result, err = client.DoJSON(rs.ServiceName, prefix, rs.CreateAction, params)

		case "query":
			params := readQueryParams(rs, d)
			result, err = client.DoQuery(rs.ServiceName, rs.CreateAction, params)

		case "rest":
			result, err = doRESTCreate(client, rs, d)
		}

		if err != nil {
			return diag.FromErr(fmt.Errorf("creating %s: %w", rs.TerraformType, err))
		}

		// Set the resource ID from the response or from the import ID attribute.
		id := extractResourceID(rs, result, d)
		if id == "" {
			return diag.Errorf("could not determine resource ID after creating %s", rs.TerraformType)
		}
		d.SetId(id)

		// Set computed attributes from the response.
		setComputedAttrs(rs, d, result)

		return nil
	}
}

// makeRead returns a ReadContext function for the given resource schema.
func makeRead(rs cmschema.ResourceSchema) tfschema.ReadContextFunc {
	return func(ctx context.Context, d *tfschema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*APIClient)

		if rs.ReadAction == "" {
			// No read action defined — skip refresh.
			return nil
		}

		protocol := serviceProtocols[rs.ServiceName]
		if protocol == "" {
			protocol = "json"
		}

		var result map[string]any
		var err error

		switch protocol {
		case "json":
			params := readIDParam(rs, d)
			prefix := targetPrefixes[rs.ServiceName]
			result, err = client.DoJSON(rs.ServiceName, prefix, rs.ReadAction, params)

		case "query":
			params := readIDParamQuery(rs, d)
			result, err = client.DoQuery(rs.ServiceName, rs.ReadAction, params)

		case "rest":
			result, err = doRESTRead(client, rs, d)
		}

		if err != nil {
			// If the resource is not found, remove it from state.
			if isNotFoundError(err) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("reading %s (%s): %w", rs.TerraformType, d.Id(), err))
		}

		setComputedAttrs(rs, d, result)
		return nil
	}
}

// makeUpdate returns an UpdateContext function for the given resource schema.
func makeUpdate(rs cmschema.ResourceSchema) tfschema.UpdateContextFunc {
	if rs.UpdateAction == "" {
		return nil // no update support — forces replacement
	}

	return func(ctx context.Context, d *tfschema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*APIClient)

		protocol := serviceProtocols[rs.ServiceName]
		if protocol == "" {
			protocol = "json"
		}

		var err error

		switch protocol {
		case "json":
			params := readJSONParams(rs, d)
			prefix := targetPrefixes[rs.ServiceName]
			_, err = client.DoJSON(rs.ServiceName, prefix, rs.UpdateAction, params)

		case "query":
			params := readQueryParams(rs, d)
			_, err = client.DoQuery(rs.ServiceName, rs.UpdateAction, params)

		case "rest":
			err = fmt.Errorf("REST update not yet implemented for %s", rs.TerraformType)
		}

		if err != nil {
			return diag.FromErr(fmt.Errorf("updating %s (%s): %w", rs.TerraformType, d.Id(), err))
		}

		// Re-read to sync state.
		return makeRead(rs)(ctx, d, meta)
	}
}

// makeDelete returns a DeleteContext function for the given resource schema.
func makeDelete(rs cmschema.ResourceSchema) tfschema.DeleteContextFunc {
	return func(ctx context.Context, d *tfschema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*APIClient)

		if rs.DeleteAction == "" {
			// No delete action — just remove from state.
			d.SetId("")
			return nil
		}

		protocol := serviceProtocols[rs.ServiceName]
		if protocol == "" {
			protocol = "json"
		}

		var err error

		switch protocol {
		case "json":
			params := readIDParam(rs, d)
			prefix := targetPrefixes[rs.ServiceName]
			_, err = client.DoJSON(rs.ServiceName, prefix, rs.DeleteAction, params)

		case "query":
			params := readIDParamQuery(rs, d)
			_, err = client.DoQuery(rs.ServiceName, rs.DeleteAction, params)

		case "rest":
			err = doRESTDelete(client, rs, d)
		}

		if err != nil {
			if isNotFoundError(err) {
				// Already deleted.
				return nil
			}
			return diag.FromErr(fmt.Errorf("deleting %s (%s): %w", rs.TerraformType, d.Id(), err))
		}

		d.SetId("")
		return nil
	}
}

// readJSONParams reads all non-computed attributes from ResourceData into a JSON-compatible map.
func readJSONParams(rs cmschema.ResourceSchema, d *tfschema.ResourceData) map[string]any {
	params := make(map[string]any)
	for _, attr := range rs.Attributes {
		if attr.Computed && !attr.Required {
			continue
		}
		val := d.Get(attr.Name)
		if val == nil {
			continue
		}
		// Convert attribute name from snake_case to PascalCase for the API.
		apiKey := snakeToPascal(attr.Name)
		params[apiKey] = coerceValue(val)
	}
	return params
}

// coerceValue converts Terraform SDK types into plain Go types that can be JSON-serialized.
func coerceValue(val any) any {
	switch v := val.(type) {
	case *tfschema.Set:
		list := v.List()
		out := make([]any, len(list))
		for i, item := range list {
			out[i] = coerceValue(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = coerceValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = coerceValue(item)
		}
		return out
	default:
		return val
	}
}

// readQueryParams reads all non-computed attributes from ResourceData into a string map for query protocol.
func readQueryParams(rs cmschema.ResourceSchema, d *tfschema.ResourceData) map[string]string {
	params := make(map[string]string)
	for _, attr := range rs.Attributes {
		if attr.Computed && !attr.Required {
			continue
		}
		val := d.Get(attr.Name)
		if val == nil {
			continue
		}
		apiKey := snakeToPascal(attr.Name)
		params[apiKey] = fmt.Sprintf("%v", val)
	}
	return params
}

// readIDParam builds a JSON params map containing only the resource identifier.
func readIDParam(rs cmschema.ResourceSchema, d *tfschema.ResourceData) map[string]any {
	params := make(map[string]any)
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		params[apiKey] = d.Id()
	}
	return params
}

// readIDParamQuery builds a query params map containing only the resource identifier.
func readIDParamQuery(rs cmschema.ResourceSchema, d *tfschema.ResourceData) map[string]string {
	params := make(map[string]string)
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		params[apiKey] = d.Id()
	}
	return params
}

// doRESTCreate handles REST-protocol create operations (S3 buckets, etc.).
func doRESTCreate(client *APIClient, rs cmschema.ResourceSchema, d *tfschema.ResourceData) (map[string]any, error) {
	switch rs.ServiceName {
	case "s3":
		bucket := d.Get("bucket").(string)
		resp, err := client.DoRESTRaw("s3", "PUT", "/"+bucket, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := readResponseBody(resp)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		return map[string]any{
			"bucket": bucket,
		}, nil
	default:
		return nil, fmt.Errorf("REST create not implemented for service %s", rs.ServiceName)
	}
}

// doRESTRead handles REST-protocol read operations.
func doRESTRead(client *APIClient, rs cmschema.ResourceSchema, d *tfschema.ResourceData) (map[string]any, error) {
	switch rs.ServiceName {
	case "s3":
		bucket := d.Id()
		resp, err := client.DoRESTRaw("s3", "HEAD", "/"+bucket, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("not found")
		}
		if resp.StatusCode >= 400 {
			body, _ := readResponseBody(resp)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		region := resp.Header.Get("X-Amz-Bucket-Region")
		if region == "" {
			region = client.region
		}
		return map[string]any{
			"bucket": bucket,
			"region": region,
			"arn":    fmt.Sprintf("arn:aws:s3:::%s", bucket),
		}, nil
	default:
		return nil, fmt.Errorf("REST read not implemented for service %s", rs.ServiceName)
	}
}

// doRESTDelete handles REST-protocol delete operations.
func doRESTDelete(client *APIClient, rs cmschema.ResourceSchema, d *tfschema.ResourceData) error {
	switch rs.ServiceName {
	case "s3":
		bucket := d.Id()
		resp, err := client.DoRESTRaw("s3", "DELETE", "/"+bucket, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil // already gone
		}
		if resp.StatusCode >= 400 {
			body, _ := readResponseBody(resp)
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		return nil
	default:
		return fmt.Errorf("REST delete not implemented for service %s", rs.ServiceName)
	}
}

// extractResourceID determines the resource ID from the API response or ResourceData.
func extractResourceID(rs cmschema.ResourceSchema, result map[string]any, d *tfschema.ResourceData) string {
	// Try to get the ID from the import ID attribute in ResourceData first.
	if rs.ImportID != "" {
		if val, ok := d.GetOk(rs.ImportID); ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}

	// Try the PascalCase version of ImportID from the API response.
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		if val, ok := result[apiKey]; ok {
			return fmt.Sprintf("%v", val)
		}
		// Also try the raw ImportID.
		if val, ok := result[rs.ImportID]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	// Look for common ID fields in the response.
	for _, key := range []string{"Id", "ID", "id", "Arn", "ARN", "arn"} {
		if val, ok := result[key]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	return ""
}

// setComputedAttrs sets computed attribute values from the API response into ResourceData.
func setComputedAttrs(rs cmschema.ResourceSchema, d *tfschema.ResourceData, result map[string]any) {
	for _, attr := range rs.Attributes {
		if !attr.Computed {
			continue
		}
		apiKey := snakeToPascal(attr.Name)
		// Try PascalCase key first, then snake_case.
		if val, ok := result[apiKey]; ok {
			_ = d.Set(attr.Name, val)
		} else if val, ok := result[attr.Name]; ok {
			_ = d.Set(attr.Name, val)
		}
	}
}

// snakeToPascal converts snake_case to PascalCase.
// e.g., "table_name" -> "TableName", "vpc_id" -> "VpcId"
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// isNotFoundError checks if an error indicates a resource was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "NotFound") ||
		strings.Contains(msg, "NoSuchKey") ||
		strings.Contains(msg, "NoSuchBucket") ||
		strings.Contains(msg, "ResourceNotFoundException") ||
		strings.Contains(msg, "HTTP 404")
}

// readResponseBody reads the full body from an HTTP response.
func readResponseBody(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
