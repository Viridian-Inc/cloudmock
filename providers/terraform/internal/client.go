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
)

// APIClient communicates with the cloudmock gateway HTTP API.
type APIClient struct {
	endpoint  string
	region    string
	accessKey string
	secretKey string
	http      *http.Client
}

// NewAPIClient creates a new API client configured for the cloudmock gateway.
func NewAPIClient(endpoint, region, accessKey, secretKey string) *APIClient {
	return &APIClient{
		endpoint:  strings.TrimRight(endpoint, "/"),
		region:    region,
		accessKey: accessKey,
		secretKey: secretKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DoJSON sends a JSON-protocol request (used by DynamoDB, KMS, etc.).
// The request is sent as POST with X-Amz-Target header and JSON body.
func (c *APIClient) DoJSON(service, targetPrefix, action string, params map[string]any) (map[string]any, error) {
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

// DoQuery sends a query-protocol request (used by EC2, SQS, etc.).
// The request is sent as POST with form-encoded Action parameter.
func (c *APIClient) DoQuery(service, action string, params map[string]string) (map[string]any, error) {
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15") // default version
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

// DoREST sends a REST-style request (used by S3, API Gateway, etc.).
// For S3, bucket operations use path-style URLs.
func (c *APIClient) DoREST(service, method, path string, body []byte) (map[string]any, error) {
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

	return c.doAndParse(req)
}

// DoRESTRaw sends a REST-style request and returns the raw response.
func (c *APIClient) DoRESTRaw(service, method, path string, body []byte) (*http.Response, error) {
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

// setAuthHeader adds an AWS SigV4-style Authorization header.
// This is simplified — cloudmock only needs the credential scope for routing.
func (c *APIClient) setAuthHeader(req *http.Request, service string) {
	date := time.Now().UTC().Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", date, c.region, service)
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=host, Signature=fakesig",
			c.accessKey, credentialScope))
	req.Header.Set("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
}

// doAndParse executes the request and parses the JSON response body.
func (c *APIClient) doAndParse(req *http.Request) (map[string]any, error) {
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

	// If response body is empty, return empty map.
	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	// Try JSON first.
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Response might be XML — return raw body as a string value.
		return map[string]any{"_raw": string(respBody)}, nil
	}

	return result, nil
}
