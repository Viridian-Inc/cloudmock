// Package provisioning implements Fly Machines and Cloudflare DNS
// integration for per-tenant cloudmock instance management.
package provisioning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const flyBaseURL = "https://api.machines.dev/v1"

// FlyClient talks to the Fly Machines REST API.
type FlyClient struct {
	token      string
	org        string
	region     string
	image      string
	httpClient *http.Client
}

// NewFlyClient creates a Fly Machines API client.
func NewFlyClient(token, org, region, image string) *FlyClient {
	return &FlyClient{
		token:      token,
		org:        org,
		region:     region,
		image:      image,
		httpClient: &http.Client{},
	}
}

// --- Fly API request/response types ---

type flyCreateAppRequest struct {
	AppName string `json:"app_name"`
	OrgSlug string `json:"org_slug"`
}

type flyCreateMachineRequest struct {
	Region string           `json:"region"`
	Config flyMachineConfig `json:"config"`
}

type flyMachineConfig struct {
	Image       string            `json:"image"`
	Env         map[string]string `json:"env,omitempty"`
	Services    []flyService      `json:"services,omitempty"`
	AutoDestroy bool              `json:"auto_destroy"`
	Guest       flyGuest          `json:"guest"`
}

type flyService struct {
	Ports        []flyPort `json:"ports"`
	Protocol     string    `json:"protocol"`
	InternalPort int       `json:"internal_port"`
}

type flyPort struct {
	Port     int    `json:"port"`
	Handlers []string `json:"handlers,omitempty"`
}

type flyGuest struct {
	CPUs   int    `json:"cpus"`
	MemMB  int    `json:"memory_mb"`
	CPUKind string `json:"cpu_kind"`
}

type flyCreateMachineResponse struct {
	ID string `json:"id"`
}

// --- Public API ---

// CreateApp creates a new Fly application.
func (c *FlyClient) CreateApp(ctx context.Context, appName string) error {
	body := flyCreateAppRequest{
		AppName: appName,
		OrgSlug: c.org,
	}
	resp, err := c.doJSON(ctx, http.MethodPost, flyBaseURL+"/apps", body)
	if err != nil {
		return fmt.Errorf("fly create app %s: %w", appName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return c.apiError("create app", appName, resp)
	}
	return nil
}

// DeleteApp removes a Fly application and all its machines.
func (c *FlyClient) DeleteApp(ctx context.Context, appName string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, flyBaseURL+"/apps/"+appName, nil)
	if err != nil {
		return fmt.Errorf("fly delete app request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fly delete app %s: %w", appName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return c.apiError("delete app", appName, resp)
	}
	return nil
}

// CreateMachine provisions a new Fly Machine in the given app.
// It returns the machine ID assigned by the Fly platform.
func (c *FlyClient) CreateMachine(ctx context.Context, appName string, env map[string]string) (string, error) {
	body := flyCreateMachineRequest{
		Region: c.region,
		Config: flyMachineConfig{
			Image:       c.image,
			Env:         env,
			AutoDestroy: false,
			Guest: flyGuest{
				CPUs:    1,
				MemMB:   256,
				CPUKind: "shared",
			},
			Services: []flyService{
				{
					Protocol:     "tcp",
					InternalPort: 4566,
					Ports: []flyPort{
						{Port: 443, Handlers: []string{"tls", "http"}},
						{Port: 80, Handlers: []string{"http"}},
					},
				},
				{
					Protocol:     "tcp",
					InternalPort: 4500,
					Ports: []flyPort{
						{Port: 4500, Handlers: []string{"tls", "http"}},
					},
				},
				{
					Protocol:     "tcp",
					InternalPort: 4599,
					Ports: []flyPort{
						{Port: 4599, Handlers: []string{"tls", "http"}},
					},
				},
			},
		},
	}

	resp, err := c.doJSON(ctx, http.MethodPost, flyBaseURL+"/apps/"+appName+"/machines", body)
	if err != nil {
		return "", fmt.Errorf("fly create machine in %s: %w", appName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", c.apiError("create machine", appName, resp)
	}

	var result flyCreateMachineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("fly create machine decode: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("fly create machine: empty machine ID in response")
	}
	return result.ID, nil
}

// StopMachine sends a stop signal to a running machine.
func (c *FlyClient) StopMachine(ctx context.Context, appName, machineID string) error {
	url := fmt.Sprintf("%s/apps/%s/machines/%s/stop", flyBaseURL, appName, machineID)
	resp, err := c.doJSON(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("fly stop machine %s/%s: %w", appName, machineID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.apiError("stop machine", machineID, resp)
	}
	return nil
}

// DestroyMachine permanently removes a machine.
func (c *FlyClient) DestroyMachine(ctx context.Context, appName, machineID string) error {
	url := fmt.Sprintf("%s/apps/%s/machines/%s?force=true", flyBaseURL, appName, machineID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("fly destroy machine request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fly destroy machine %s/%s: %w", appName, machineID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.apiError("destroy machine", machineID, resp)
	}
	return nil
}

// --- helpers ---

func (c *FlyClient) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}

func (c *FlyClient) doJSON(ctx context.Context, method, url string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	return c.httpClient.Do(req)
}

func (c *FlyClient) apiError(op, resource string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return &APIError{
		Operation:  op,
		Resource:   resource,
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// APIError represents an error returned by an external API.
type APIError struct {
	Operation  string
	Resource   string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Operation, e.Resource, e.StatusCode, e.Body)
}
