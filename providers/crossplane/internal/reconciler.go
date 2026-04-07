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

	cmschema "github.com/Viridian-Inc/cloudmock/pkg/schema"
)

// Reconciler handles CRUD operations for a single cloudmock resource type
// by communicating with the cloudmock HTTP gateway.
type Reconciler struct {
	client    *http.Client
	endpoint  string
	region    string
	accessKey string
	secretKey string
	schema    *cmschema.ResourceSchema
}

// NewReconciler creates a Reconciler for the given resource schema.
func NewReconciler(endpoint, region, accessKey, secretKey string, schema *cmschema.ResourceSchema) *Reconciler {
	return &Reconciler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		endpoint:  strings.TrimRight(endpoint, "/"),
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
		schema:    schema,
	}
}

// Observe reads the external resource state from cloudmock.
// Returns whether the resource exists, its current state, and any error.
func (r *Reconciler) Observe(id string) (bool, map[string]any, error) {
	if r.schema.ReadAction == "" {
		return false, nil, nil
	}

	protocol := serviceProtocols[r.schema.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var result map[string]any
	var err error

	switch protocol {
	case "json":
		params := r.buildIDParam(id)
		prefix := targetPrefixes[r.schema.ServiceName]
		result, err = r.doJSON(r.schema.ServiceName, prefix, r.schema.ReadAction, params)
	case "query":
		params := r.buildIDParamQuery(id)
		result, err = r.doQuery(r.schema.ServiceName, r.schema.ReadAction, params)
	case "rest":
		result, err = r.doRESTRead(id)
	}

	if err != nil {
		if isNotFoundError(err) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("observing %s (%s): %w", r.schema.TerraformType, id, err)
	}

	return true, result, nil
}

// Create creates the resource in cloudmock.
// Returns the resource ID, its state, and any error.
func (r *Reconciler) Create(inputs map[string]any) (string, map[string]any, error) {
	protocol := serviceProtocols[r.schema.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var result map[string]any
	var err error

	switch protocol {
	case "json":
		params := r.buildJSONParams(inputs)
		prefix := targetPrefixes[r.schema.ServiceName]
		result, err = r.doJSON(r.schema.ServiceName, prefix, r.schema.CreateAction, params)
	case "query":
		params := r.buildQueryParams(inputs)
		result, err = r.doQuery(r.schema.ServiceName, r.schema.CreateAction, params)
	case "rest":
		result, err = r.doRESTCreate(inputs)
	}

	if err != nil {
		return "", nil, fmt.Errorf("creating %s: %w", r.schema.TerraformType, err)
	}

	id := extractResourceID(r.schema, result, inputs)
	if id == "" {
		return "", nil, fmt.Errorf("could not determine resource ID after creating %s", r.schema.TerraformType)
	}

	// Merge computed attributes from response into outputs.
	outputs := make(map[string]any)
	for k, v := range inputs {
		outputs[k] = v
	}
	mergeComputedAttrs(r.schema, outputs, result)

	return id, outputs, nil
}

// Update updates the resource in cloudmock.
// Returns the updated state and any error.
func (r *Reconciler) Update(id string, inputs map[string]any) (map[string]any, error) {
	if r.schema.UpdateAction == "" {
		return nil, fmt.Errorf("resource %s does not support updates", r.schema.TerraformType)
	}

	protocol := serviceProtocols[r.schema.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var err error

	switch protocol {
	case "json":
		params := r.buildJSONParams(inputs)
		prefix := targetPrefixes[r.schema.ServiceName]
		_, err = r.doJSON(r.schema.ServiceName, prefix, r.schema.UpdateAction, params)
	case "query":
		params := r.buildQueryParams(inputs)
		_, err = r.doQuery(r.schema.ServiceName, r.schema.UpdateAction, params)
	case "rest":
		return nil, fmt.Errorf("REST update not implemented for %s", r.schema.TerraformType)
	}

	if err != nil {
		return nil, fmt.Errorf("updating %s (%s): %w", r.schema.TerraformType, id, err)
	}

	// Re-read to get updated state.
	exists, state, err := r.Observe(id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("resource %s (%s) not found after update", r.schema.TerraformType, id)
	}
	return state, nil
}

// Delete removes the resource from cloudmock.
func (r *Reconciler) Delete(id string) error {
	if r.schema.DeleteAction == "" {
		return nil
	}

	protocol := serviceProtocols[r.schema.ServiceName]
	if protocol == "" {
		protocol = "json"
	}

	var err error

	switch protocol {
	case "json":
		params := r.buildIDParam(id)
		prefix := targetPrefixes[r.schema.ServiceName]
		_, err = r.doJSON(r.schema.ServiceName, prefix, r.schema.DeleteAction, params)
	case "query":
		params := r.buildIDParamQuery(id)
		_, err = r.doQuery(r.schema.ServiceName, r.schema.DeleteAction, params)
	case "rest":
		err = r.doRESTDelete(id)
	}

	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("deleting %s (%s): %w", r.schema.TerraformType, id, err)
	}

	return nil
}

// ── HTTP transport methods ──────────────────────────────────────────────────

// serviceProtocols maps service names to their wire protocol.
var serviceProtocols = map[string]string{
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

func (r *Reconciler) doJSON(service, targetPrefix, action string, params map[string]any) (map[string]any, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	if targetPrefix != "" {
		req.Header.Set("X-Amz-Target", targetPrefix+"."+action)
	}
	r.setAuthHeader(req, service)

	return r.doAndParse(req)
}

func (r *Reconciler) doQuery(service, action string, params map[string]string) (map[string]any, error) {
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, r.endpoint+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.setAuthHeader(req, service)

	return r.doAndParse(req)
}

func (r *Reconciler) doRESTCreate(inputs map[string]any) (map[string]any, error) {
	switch r.schema.ServiceName {
	case "s3":
		bucket, _ := inputs["bucket"].(string)
		if bucket == "" {
			return nil, fmt.Errorf("bucket name is required")
		}
		resp, err := r.doRESTRaw("s3", "PUT", "/"+bucket, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return map[string]any{"bucket": bucket}, nil
	default:
		return nil, fmt.Errorf("REST create not implemented for service %s", r.schema.ServiceName)
	}
}

func (r *Reconciler) doRESTRead(id string) (map[string]any, error) {
	switch r.schema.ServiceName {
	case "s3":
		resp, err := r.doRESTRaw("s3", "HEAD", "/"+id, nil)
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
			region = r.region
		}
		return map[string]any{
			"bucket": id,
			"region": region,
			"arn":    fmt.Sprintf("arn:aws:s3:::%s", id),
		}, nil
	default:
		return nil, fmt.Errorf("REST read not implemented for service %s", r.schema.ServiceName)
	}
}

func (r *Reconciler) doRESTDelete(id string) error {
	switch r.schema.ServiceName {
	case "s3":
		resp, err := r.doRESTRaw("s3", "DELETE", "/"+id, nil)
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
		return fmt.Errorf("REST delete not implemented for service %s", r.schema.ServiceName)
	}
}

func (r *Reconciler) doRESTRaw(service, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, r.endpoint+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/xml")
	}
	r.setAuthHeader(req, service)

	return r.client.Do(req)
}

func (r *Reconciler) setAuthHeader(req *http.Request, service string) {
	date := time.Now().UTC().Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, r.region, service)
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=host, Signature=fakesig",
			r.accessKey, credentialScope))
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
}

func (r *Reconciler) doAndParse(req *http.Request) (map[string]any, error) {
	resp, err := r.client.Do(req)
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

func (r *Reconciler) buildJSONParams(inputs map[string]any) map[string]any {
	params := make(map[string]any)
	for _, attr := range r.schema.Attributes {
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

func (r *Reconciler) buildQueryParams(inputs map[string]any) map[string]string {
	params := make(map[string]string)
	for _, attr := range r.schema.Attributes {
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

func (r *Reconciler) buildIDParam(id string) map[string]any {
	params := make(map[string]any)
	if r.schema.ImportID != "" {
		apiKey := snakeToPascal(r.schema.ImportID)
		params[apiKey] = id
	}
	return params
}

func (r *Reconciler) buildIDParamQuery(id string) map[string]string {
	params := make(map[string]string)
	if r.schema.ImportID != "" {
		apiKey := snakeToPascal(r.schema.ImportID)
		params[apiKey] = id
	}
	return params
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

// extractResourceID determines the resource ID from the API response or inputs.
func extractResourceID(rs *cmschema.ResourceSchema, result, inputs map[string]any) string {
	if rs.ImportID != "" {
		if val, ok := inputs[rs.ImportID]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}

	if rs.ImportID != "" {
		apiKey := snakeToPascal(rs.ImportID)
		if val, ok := result[apiKey]; ok {
			return fmt.Sprintf("%v", val)
		}
		if val, ok := result[rs.ImportID]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

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
