package provisioning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const cfBaseURL = "https://api.cloudflare.com/client/v4"

// CloudflareClient talks to the Cloudflare DNS API.
type CloudflareClient struct {
	token      string
	zoneID     string
	httpClient *http.Client
}

// NewCloudflareClient creates a Cloudflare DNS API client.
func NewCloudflareClient(token, zoneID string) *CloudflareClient {
	return &CloudflareClient{
		token:      token,
		zoneID:     zoneID,
		httpClient: &http.Client{},
	}
}

// --- Cloudflare API types ---

type cfCreateRecordRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

type cfAPIResponse struct {
	Success bool            `json:"success"`
	Errors  []cfAPIError    `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type cfAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfDNSRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// --- Public API ---

// AddCNAME creates a proxied CNAME record pointing name to target.
// name is the subdomain (e.g. "acme.cloudmock.io"), target is the
// Fly app hostname (e.g. "cm-acme.fly.dev").
func (c *CloudflareClient) AddCNAME(ctx context.Context, name, target string) error {
	body := cfCreateRecordRequest{
		Type:    "CNAME",
		Name:    name,
		Content: target,
		Proxied: true,
		TTL:     1, // 1 = automatic when proxied
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("cloudflare marshal create record: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", cfBaseURL, c.zoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cloudflare create record request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare add CNAME %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.apiError("add CNAME", name, resp)
	}

	var apiResp cfAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("cloudflare add CNAME decode: %w", err)
	}
	if !apiResp.Success {
		return fmt.Errorf("cloudflare add CNAME %s: API error: %v", name, apiResp.Errors)
	}
	return nil
}

// RemoveCNAME finds and deletes the CNAME record for the given name.
func (c *CloudflareClient) RemoveCNAME(ctx context.Context, name string) error {
	// First, find the record ID by querying for the name.
	recordID, err := c.findRecordID(ctx, name)
	if err != nil {
		return fmt.Errorf("cloudflare remove CNAME %s: %w", name, err)
	}

	// Delete the record.
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cfBaseURL, c.zoneID, recordID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("cloudflare delete record request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare remove CNAME %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.apiError("remove CNAME", name, resp)
	}

	var apiResp cfAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("cloudflare remove CNAME decode: %w", err)
	}
	if !apiResp.Success {
		return fmt.Errorf("cloudflare remove CNAME %s: API error: %v", name, apiResp.Errors)
	}
	return nil
}

// findRecordID looks up the Cloudflare DNS record ID for a given name.
func (c *CloudflareClient) findRecordID(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=CNAME&name=%s", cfBaseURL, c.zoneID, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("cloudflare find record request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudflare find record %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.apiError("find record", name, resp)
	}

	var apiResp cfAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("cloudflare find record decode: %w", err)
	}
	if !apiResp.Success {
		return "", fmt.Errorf("cloudflare find record %s: API error: %v", name, apiResp.Errors)
	}

	var records []cfDNSRecord
	if err := json.Unmarshal(apiResp.Result, &records); err != nil {
		return "", fmt.Errorf("cloudflare find record unmarshal: %w", err)
	}
	if len(records) == 0 {
		return "", fmt.Errorf("cloudflare: no CNAME record found for %s", name)
	}
	return records[0].ID, nil
}

// --- helpers ---

func (c *CloudflareClient) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}

func (c *CloudflareClient) apiError(op, resource string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return &APIError{
		Operation:  op,
		Resource:   resource,
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}
