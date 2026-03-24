package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	cmschema "github.com/neureaux/cloudmock/pkg/schema"
)

// apiClient communicates with the cloudmock gateway HTTP API.
// This is the same pattern as the Terraform provider's client.go.
type apiClient struct {
	endpoint  string
	region    string
	accessKey string
	secretKey string
	http      *http.Client
}

// newAPIClient creates a new API client configured for the cloudmock gateway.
func newAPIClient(endpoint, region, accessKey, secretKey string) *apiClient {
	return &apiClient{
		endpoint:  strings.TrimRight(endpoint, "/"),
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// serviceProtocols maps service names to their wire protocol.
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

	// REST protocol (S3, API Gateway, Lambda)
	"s3":         "rest",
	"apigateway": "rest",
	"lambda":     "rest",
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

// createResource sends a create request to cloudmock and returns (id, outputs, error).
func (c *apiClient) createResource(rs *cmschema.ResourceSchema, inputs map[string]any) (string, map[string]any, error) {
	protocol := serviceProtocols[rs.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var result map[string]any
	var err error

	switch protocol {
	case "json":
		params := buildJSONParams(rs, inputs)
		prefix := targetPrefixes[rs.ServiceName]
		result, err = c.doJSON(rs.ServiceName, prefix, rs.CreateAction, params)

	case "query":
		params := buildQueryParams(rs, inputs)
		result, err = c.doQuery(rs.ServiceName, rs.CreateAction, params)

	case "rest":
		result, err = c.doRESTCreate(rs, inputs)
	}

	if err != nil {
		return "", nil, fmt.Errorf("creating %s: %w", rs.TerraformType, err)
	}

	id := extractResourceID(rs, result, inputs)
	if id == "" {
		return "", nil, fmt.Errorf("could not determine resource ID after creating %s", rs.TerraformType)
	}

	// Merge computed attributes from response into outputs.
	outputs := make(map[string]any)
	for k, v := range inputs {
		outputs[k] = v
	}
	mergeComputedAttrs(rs, outputs, result)

	return id, outputs, nil
}

// readResource reads the current state of a resource from cloudmock.
func (c *apiClient) readResource(rs *cmschema.ResourceSchema, id string) (map[string]any, error) {
	if rs.ReadAction == "" {
		return nil, nil
	}

	protocol := serviceProtocols[rs.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var result map[string]any
	var err error

	switch protocol {
	case "json":
		params := buildIDParam(rs, id)
		prefix := targetPrefixes[rs.ServiceName]
		result, err = c.doJSON(rs.ServiceName, prefix, rs.ReadAction, params)

	case "query":
		params := buildIDParamQuery(rs, id)
		result, err = c.doQuery(rs.ServiceName, rs.ReadAction, params)

	case "rest":
		result, err = c.doRESTRead(rs, id)
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s (%s): %w", rs.TerraformType, id, err)
	}

	return result, nil
}

// updateResource updates a resource in cloudmock.
func (c *apiClient) updateResource(rs *cmschema.ResourceSchema, id string, inputs map[string]any) (map[string]any, error) {
	if rs.UpdateAction == "" {
		return nil, fmt.Errorf("resource %s does not support updates", rs.TerraformType)
	}

	protocol := serviceProtocols[rs.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var err error

	switch protocol {
	case "json":
		params := buildJSONParams(rs, inputs)
		prefix := targetPrefixes[rs.ServiceName]
		_, err = c.doJSON(rs.ServiceName, prefix, rs.UpdateAction, params)

	case "query":
		params := buildQueryParams(rs, inputs)
		_, err = c.doQuery(rs.ServiceName, rs.UpdateAction, params)

	case "rest":
		return nil, fmt.Errorf("REST update not implemented for %s", rs.TerraformType)
	}

	if err != nil {
		return nil, fmt.Errorf("updating %s (%s): %w", rs.TerraformType, id, err)
	}

	// Re-read to get the updated state.
	return c.readResource(rs, id)
}

// deleteResource deletes a resource from cloudmock.
func (c *apiClient) deleteResource(rs *cmschema.ResourceSchema, id string) error {
	if rs.DeleteAction == "" {
		return nil // no delete action, just remove from state
	}

	protocol := serviceProtocols[rs.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var err error

	switch protocol {
	case "json":
		params := buildIDParam(rs, id)
		prefix := targetPrefixes[rs.ServiceName]
		_, err = c.doJSON(rs.ServiceName, prefix, rs.DeleteAction, params)

	case "query":
		params := buildIDParamQuery(rs, id)
		_, err = c.doQuery(rs.ServiceName, rs.DeleteAction, params)

	case "rest":
		err = c.doRESTDelete(rs, id)
	}

	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("deleting %s (%s): %w", rs.TerraformType, id, err)
	}

	return nil
}

// ── HTTP transport methods ──────────────────────────────────────────────────

func (c *apiClient) doJSON(service, targetPrefix, action string, params map[string]any) (map[string]any, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	if targetPrefix != "" {
		req.Header.Set("X-Amz-Target", targetPrefix+"."+action)
	}
	c.setAuthHeader(req, service)

	return c.doAndParse(req)
}

func (c *apiClient) doQuery(service, action string, params map[string]string) (map[string]any, error) {
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuthHeader(req, service)

	return c.doAndParse(req)
}

func (c *apiClient) doRESTCreate(rs *cmschema.ResourceSchema, inputs map[string]any) (map[string]any, error) {
	switch rs.ServiceName {
	case "s3":
		bucket, _ := inputs["bucket"].(string)
		if bucket == "" {
			return nil, fmt.Errorf("bucket name is required")
		}
		resp, err := c.doRESTRaw("s3", "PUT", "/"+bucket, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return map[string]any{
			"bucket": bucket,
		}, nil
	default:
		return nil, fmt.Errorf("REST create not implemented for service %s", rs.ServiceName)
	}
}

func (c *apiClient) doRESTRead(rs *cmschema.ResourceSchema, id string) (map[string]any, error) {
	switch rs.ServiceName {
	case "s3":
		resp, err := c.doRESTRaw("s3", "HEAD", "/"+id, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("not found")
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		region := resp.Header.Get("X-Amz-Bucket-Region")
		if region == "" {
			region = c.region
		}
		return map[string]any{
			"bucket": id,
			"region": region,
			"arn":    fmt.Sprintf("arn:aws:s3:::%s", id),
		}, nil
	default:
		return nil, fmt.Errorf("REST read not implemented for service %s", rs.ServiceName)
	}
}

func (c *apiClient) doRESTDelete(rs *cmschema.ResourceSchema, id string) error {
	switch rs.ServiceName {
	case "s3":
		resp, err := c.doRESTRaw("s3", "DELETE", "/"+id, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return nil
	default:
		return fmt.Errorf("REST delete not implemented for service %s", rs.ServiceName)
	}
}

func (c *apiClient) doRESTRaw(service, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.endpoint+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/xml")
	}
	c.setAuthHeader(req, service)

	return c.http.Do(req)
}

func (c *apiClient) setAuthHeader(req *http.Request, service string) {
	date := time.Now().UTC().Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, c.region, service)
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=host, Signature=fakesig",
			c.accessKey, credentialScope))
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
}

func (c *apiClient) doAndParse(req *http.Request) (map[string]any, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return map[string]any{"_raw": string(respBody)}, nil
	}

	return result, nil
}

// ── Helper functions ────────────────────────────────────────────────────────

// buildJSONParams converts inputs into PascalCase keyed params for JSON protocol.
func buildJSONParams(rs *cmschema.ResourceSchema, inputs map[string]any) map[string]any {
	params := make(map[string]any)
	for _, attr := range rs.Attributes {
		if attr.Computed && !attr.Required {
			continue
		}
		if val, ok := inputs[attr.Name]; ok && val != nil {
			apiKey := snakeToPascal(attr.Name)
			params[apiKey] = val
		}
	}
	return params
}

// buildQueryParams converts inputs into PascalCase keyed string params for query protocol.
func buildQueryParams(rs *cmschema.ResourceSchema, inputs map[string]any) map[string]string {
	params := make(map[string]string)
	for _, attr := range rs.Attributes {
		if attr.Computed && !attr.Required {
			continue
		}
		if val, ok := inputs[attr.Name]; ok && val != nil {
			apiKey := snakeToPascal(attr.Name)
			params[apiKey] = fmt.Sprintf("%v", val)
		}
	}
	return params
}

// buildIDParam builds a JSON params map containing only the resource identifier.
func buildIDParam(rs *cmschema.ResourceSchema, id string) map[string]any {
	params := make(map[string]any)
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		params[apiKey] = id
	}
	return params
}

// buildIDParamQuery builds a query params map containing only the resource identifier.
func buildIDParamQuery(rs *cmschema.ResourceSchema, id string) map[string]string {
	params := make(map[string]string)
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		params[apiKey] = id
	}
	return params
}

// extractResourceID determines the resource ID from the API response or inputs.
func extractResourceID(rs *cmschema.ResourceSchema, result, inputs map[string]any) string {
	// Try the import ID attribute from inputs first.
	if rs.ImportID != "" {
		if val, ok := inputs[rs.ImportID]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}

	// Try the PascalCase version from the API response.
	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		if val, ok := result[apiKey]; ok {
			return fmt.Sprintf("%v", val)
		}
		if val, ok := result[rs.ImportID]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	// Look for common ID fields.
	for _, key := range []string{"Id", "ID", "id", "Arn", "ARN", "arn"} {
		if val, ok := result[key]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	return ""
}

// mergeComputedAttrs sets computed attributes from an API response into the outputs map.
func mergeComputedAttrs(rs *cmschema.ResourceSchema, outputs, result map[string]any) {
	for _, attr := range rs.Attributes {
		if !attr.Computed {
			continue
		}
		apiKey := snakeToPascal(attr.Name)
		if val, ok := result[apiKey]; ok {
			outputs[attr.Name] = val
		} else if val, ok := result[attr.Name]; ok {
			outputs[attr.Name] = val
		}
	}
}

// snakeToPascal converts snake_case to PascalCase.
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
