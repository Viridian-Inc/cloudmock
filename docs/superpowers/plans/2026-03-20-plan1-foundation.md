# cloudmock Foundation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the core cloudmock infrastructure — Go project scaffolding, shared service framework, gateway with AWS request routing, IAM engine with policy evaluation, Docker Compose orchestration, and a minimal S3 stub to validate the full request lifecycle.

**Architecture:** Single gateway HTTP server routes AWS API requests to service containers over a Docker bridge network. Each service implements a shared `Service` interface. IAM service provides gRPC auth checks on every request (configurable: enforce/authenticate/none). Services are discovered via Docker labels and health checks.

**Tech Stack:** Go 1.26, gRPC (google.golang.org/grpc), Protocol Buffers, Docker, Docker Compose, SQLite (mattn/go-sqlite3), chi router, testify

**Spec:** `docs/superpowers/specs/2026-03-20-cloudmock-design.md`

---

## Chunk 1: Project Scaffolding & Service Framework

### Task 1: Initialize Go Module and Directory Structure

**Files:**
- Modify: `go.mod`
- Create: `cmd/gateway/main.go`
- Create: `cmd/cloudmock/main.go`
- Create: `pkg/service/service.go`
- Create: `pkg/service/request.go`
- Create: `pkg/service/response.go`
- Create: `pkg/service/errors.go`
- Create: `pkg/config/config.go`
- Create: `Makefile`
- Create: `.gitignore`
- Create: `LICENSE`

- [ ] **Step 1: Create .gitignore**

```gitignore
# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
*.out
coverage.html

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Build
dist/
vendor/
```

- [ ] **Step 2: Create Apache 2.0 LICENSE file**

Create `LICENSE` with standard Apache 2.0 text.

- [ ] **Step 3: Create Makefile**

```makefile
.PHONY: build test lint clean proto docker

BINARY_GATEWAY=bin/gateway
BINARY_CLI=bin/cloudmock

build: build-gateway build-cli

build-gateway:
	go build -o $(BINARY_GATEWAY) ./cmd/gateway

build-cli:
	go build -o $(BINARY_CLI) ./cmd/cloudmock

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

proto:
	protoc --go_out=. --go-grpc_out=. proto/iam.proto

docker:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
```

- [ ] **Step 4: Create placeholder main files**

`cmd/gateway/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("cloudmock gateway")
}
```

`cmd/cloudmock/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("cloudmock cli")
}
```

- [ ] **Step 5: Run `go mod tidy` and verify build**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 6: Commit**

```bash
git add .gitignore LICENSE Makefile cmd/ pkg/ go.mod go.sum
git commit -m "feat: initialize project scaffolding and directory structure"
```

---

### Task 2: Service Framework — Core Types

**Files:**
- Create: `pkg/service/service.go`
- Create: `pkg/service/request.go`
- Create: `pkg/service/response.go`
- Create: `pkg/service/errors.go`
- Create: `pkg/service/errors_test.go`
- Create: `pkg/service/response_test.go`

- [ ] **Step 1: Write test for AWS error formatting**

`pkg/service/errors_test.go`:
```go
package service_test

import (
	"encoding/xml"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSError_XML(t *testing.T) {
	err := service.NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
	xmlBytes, xmlErr := xml.Marshal(err)
	require.NoError(t, xmlErr)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, "<Code>NoSuchBucket</Code>")
	assert.Contains(t, xmlStr, "<Message>The specified bucket does not exist</Message>")
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
}

func TestAWSError_JSON(t *testing.T) {
	err := service.NewAWSError("ResourceNotFoundException", "Table not found", http.StatusBadRequest)
	jsonBytes, jsonErr := json.Marshal(err)
	require.NoError(t, jsonErr)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &result))
	assert.Equal(t, "ResourceNotFoundException", result["__type"])
	assert.Equal(t, "Table not found", result["Message"])
}

func TestCommonErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *service.AWSError
		code       string
		statusCode int
	}{
		{"not found", service.ErrNotFound("MyBucket"), "ResourceNotFoundException", http.StatusNotFound},
		{"already exists", service.ErrAlreadyExists("MyBucket"), "ResourceAlreadyExistsException", http.StatusConflict},
		{"access denied", service.ErrAccessDenied("s3:GetObject"), "AccessDeniedException", http.StatusForbidden},
		{"validation", service.ErrValidation("BucketName is required"), "ValidationException", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.Equal(t, tt.statusCode, tt.err.StatusCode())
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/service/... -v`
Expected: FAIL — types not defined yet

- [ ] **Step 3: Implement error types**

`pkg/service/errors.go`:
```go
package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
)

// AWSError represents an AWS-style API error response.
type AWSError struct {
	XMLName    xml.Name `xml:"Error" json:"-"`
	Code       string   `xml:"Code" json:"__type"`
	Message    string   `xml:"Message" json:"Message"`
	statusCode int
}

func NewAWSError(code, message string, statusCode int) *AWSError {
	return &AWSError{Code: code, Message: message, statusCode: statusCode}
}

func (e *AWSError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AWSError) StatusCode() int {
	return e.statusCode
}

func (e *AWSError) MarshalJSON() ([]byte, error) {
	type alias struct {
		Type    string `json:"__type"`
		Message string `json:"Message"`
	}
	return json.Marshal(&alias{Type: e.Code, Message: e.Message})
}

func ErrNotFound(resource string) *AWSError {
	return NewAWSError("ResourceNotFoundException", fmt.Sprintf("Resource not found: %s", resource), http.StatusNotFound)
}

func ErrAlreadyExists(resource string) *AWSError {
	return NewAWSError("ResourceAlreadyExistsException", fmt.Sprintf("Resource already exists: %s", resource), http.StatusConflict)
}

func ErrAccessDenied(action string) *AWSError {
	return NewAWSError("AccessDeniedException", fmt.Sprintf("User is not authorized to perform: %s", action), http.StatusForbidden)
}

func ErrValidation(message string) *AWSError {
	return NewAWSError("ValidationException", message, http.StatusBadRequest)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/service/... -v`
Expected: PASS

- [ ] **Step 5: Write test for response formatting**

`pkg/service/response_test.go`:
```go
package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
)

func TestWriteXMLResponse(t *testing.T) {
	type ListBucketsOutput struct {
		Buckets []string `xml:"Buckets>Bucket>Name"`
	}
	output := ListBucketsOutput{Buckets: []string{"my-bucket"}}
	w := httptest.NewRecorder()
	service.WriteXMLResponse(w, http.StatusOK, output)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/xml")
	assert.Contains(t, w.Body.String(), "<Name>my-bucket</Name>")
}

func TestWriteJSONResponse(t *testing.T) {
	output := map[string]interface{}{"TableName": "my-table"}
	w := httptest.NewRecorder()
	service.WriteJSONResponse(w, http.StatusOK, output)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/x-amz-json")
	assert.Contains(t, w.Body.String(), `"TableName":"my-table"`)
}

func TestWriteErrorResponse_XML(t *testing.T) {
	err := service.ErrNotFound("my-bucket")
	w := httptest.NewRecorder()
	service.WriteErrorResponse(w, err, "xml")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "<Code>ResourceNotFoundException</Code>")
}

func TestWriteErrorResponse_JSON(t *testing.T) {
	err := service.ErrNotFound("my-table")
	w := httptest.NewRecorder()
	service.WriteErrorResponse(w, err, "json")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"__type":"ResourceNotFoundException"`)
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./pkg/service/... -v`
Expected: FAIL — WriteXMLResponse etc not defined

- [ ] **Step 7: Implement response helpers**

`pkg/service/response.go`:
```go
package service

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
)

// Response wraps a service action result.
type Response struct {
	StatusCode int
	Body       interface{}
	Format     string // "xml" or "json"
	Headers    map[string]string
}

func WriteXMLResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(statusCode)
	xmlBytes, _ := xml.MarshalIndent(body, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(xmlBytes)
}

func WriteJSONResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}

func WriteErrorResponse(w http.ResponseWriter, awsErr *AWSError, format string) {
	if format == "json" {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(awsErr.StatusCode())
		json.NewEncoder(w).Encode(awsErr)
	} else {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(awsErr.StatusCode())
		xmlBytes, _ := xml.MarshalIndent(awsErr, "", "  ")
		w.Write([]byte(xml.Header))
		w.Write(xmlBytes)
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./pkg/service/... -v`
Expected: PASS

- [ ] **Step 9: Implement Service interface and RequestContext**

`pkg/service/service.go`:
```go
package service

import "net/http"

// Service is the interface every cloudmock AWS service must implement.
type Service interface {
	// Name returns the AWS service identifier (e.g., "s3", "dynamodb").
	Name() string

	// Actions returns the list of API actions this service supports.
	Actions() []Action

	// HandleRequest processes an AWS API request and returns a response.
	HandleRequest(ctx *RequestContext) (*Response, error)

	// HealthCheck returns nil if the service is ready to handle requests.
	HealthCheck() error
}

// Action describes a single AWS API action supported by a service.
type Action struct {
	Name      string           // AWS action name, e.g. "CreateBucket"
	Method    string           // HTTP method, e.g. "PUT"
	IAMAction string           // IAM action string, e.g. "s3:CreateBucket"
	Validator RequestValidator // optional request validator
}

// RequestValidator validates an incoming request before the handler runs.
type RequestValidator func(ctx *RequestContext) *AWSError

// CallerIdentity represents the authenticated AWS caller.
type CallerIdentity struct {
	AccountID   string
	ARN         string
	UserID      string
	AccessKeyID string
	IsRoot      bool
}

// RequestContext carries all parsed information about an incoming AWS request.
type RequestContext struct {
	Action     string
	Region     string
	AccountID  string
	Identity   *CallerIdentity
	RawRequest *http.Request
	Body       []byte
	Params     map[string]string
	Service    string // target service name
}
```

`pkg/service/request.go`:
```go
package service

import (
	"io"
	"net/http"
	"strings"
)

// ParseRequestBody reads and returns the full request body.
func ParseRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// DetectService determines the target AWS service from the request.
// It checks the Authorization header's credential scope first,
// then falls back to X-Amz-Target and Host headers.
func DetectService(r *http.Request) string {
	// Check Authorization header credential scope:
	// AWS4-HMAC-SHA256 Credential=AKID/20260320/us-east-1/s3/aws4_request
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if idx := strings.Index(auth, "Credential="); idx != -1 {
			parts := strings.Split(auth[idx+11:], "/")
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	}

	// Check X-Amz-Target header: DynamoDB_20120810.CreateTable
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) >= 1 {
			svc := strings.ToLower(parts[0])
			// Strip version suffix: dynamodb_20120810 -> dynamodb
			if idx := strings.Index(svc, "_"); idx != -1 {
				svc = svc[:idx]
			}
			return svc
		}
	}

	// Check Host header: s3.localhost.localstack.cloud
	host := r.Host
	if host != "" {
		parts := strings.Split(host, ".")
		if len(parts) > 1 {
			return parts[0]
		}
	}

	return ""
}

// DetectAction determines the AWS API action from the request.
func DetectAction(r *http.Request, serviceName string) string {
	// X-Amz-Target: DynamoDB_20120810.CreateTable -> CreateTable
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// Query string: ?Action=CreateBucket
	action := r.URL.Query().Get("Action")
	if action != "" {
		return action
	}

	return ""
}
```

- [ ] **Step 10: Run all tests**

Run: `go test ./pkg/service/... -v`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add pkg/service/
git commit -m "feat: add service framework — core types, errors, request parsing, response formatting"
```

---

### Task 3: Configuration System

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/config_test.go`
- Create: `cloudmock.yml`

- [ ] **Step 1: Write test for config loading**

`pkg/config/config_test.go`:
```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "000000000000", cfg.AccountID)
	assert.Equal(t, "test", cfg.IAM.RootAccessKey)
	assert.Equal(t, "test", cfg.IAM.RootSecretKey)
	assert.Equal(t, "enforce", cfg.IAM.Mode)
	assert.False(t, cfg.Persistence.Enabled)
	assert.True(t, cfg.Dashboard.Enabled)
	assert.Equal(t, 4566, cfg.Gateway.Port)
	assert.Equal(t, 4500, cfg.Dashboard.Port)
	assert.Equal(t, 4599, cfg.Admin.Port)
	assert.Equal(t, "minimal", cfg.Profile)
}

func TestLoadFromFile(t *testing.T) {
	yaml := `
region: us-west-2
account_id: "123456789012"
profile: standard
iam:
  mode: none
  root_access_key: mykey
  root_secret_key: mysecret
persistence:
  enabled: true
  path: /tmp/cloudmock-data
`
	tmpFile := filepath.Join(t.TempDir(), "cloudmock.yml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(yaml), 0644))

	cfg, err := config.LoadFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", cfg.Region)
	assert.Equal(t, "123456789012", cfg.AccountID)
	assert.Equal(t, "standard", cfg.Profile)
	assert.Equal(t, "none", cfg.IAM.Mode)
	assert.Equal(t, "mykey", cfg.IAM.RootAccessKey)
	assert.True(t, cfg.Persistence.Enabled)
	assert.Equal(t, "/tmp/cloudmock-data", cfg.Persistence.Path)
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("CLOUDMOCK_REGION", "eu-west-1")
	t.Setenv("CLOUDMOCK_IAM_MODE", "authenticate")
	t.Setenv("CLOUDMOCK_PERSIST", "true")
	t.Setenv("CLOUDMOCK_LOG_LEVEL", "debug")

	cfg := config.Default()
	cfg.ApplyEnv()

	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "authenticate", cfg.IAM.Mode)
	assert.True(t, cfg.Persistence.Enabled)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestProfileServices(t *testing.T) {
	cfg := config.Default()
	cfg.Profile = "minimal"
	services := cfg.EnabledServices()
	assert.Contains(t, services, "iam")
	assert.Contains(t, services, "s3")
	assert.Contains(t, services, "sqs")
	assert.Contains(t, services, "lambda")
	assert.NotContains(t, services, "rds")

	cfg.Profile = "standard"
	services = cfg.EnabledServices()
	assert.Contains(t, services, "rds")
	assert.Contains(t, services, "cloudformation")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/... -v`
Expected: FAIL — config package not implemented

- [ ] **Step 3: Implement config**

`pkg/config/config.go`:
```go
package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Region    string `yaml:"region"`
	AccountID string `yaml:"account_id"`
	Profile   string `yaml:"profile"`

	IAM struct {
		Mode          string `yaml:"mode"`
		RootAccessKey string `yaml:"root_access_key"`
		RootSecretKey string `yaml:"root_secret_key"`
		SeedFile      string `yaml:"seed_file"`
	} `yaml:"iam"`

	Persistence struct {
		Enabled bool   `yaml:"enabled"`
		Path    string `yaml:"path"`
	} `yaml:"persistence"`

	Gateway struct {
		Port int `yaml:"port"`
	} `yaml:"gateway"`

	Dashboard struct {
		Enabled bool `yaml:"enabled"`
		Port    int  `yaml:"port"`
	} `yaml:"dashboard"`

	Admin struct {
		Port int `yaml:"port"`
	} `yaml:"admin"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`

	Services map[string]ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
	Enabled  *bool    `yaml:"enabled"`
	Port     int      `yaml:"port,omitempty"`
	Runtimes []string `yaml:"runtimes,omitempty"`
}

var minimalServices = []string{
	"iam", "sts", "s3", "dynamodb", "sqs", "sns", "lambda", "cloudwatch-logs",
}

var standardServices = []string{
	"iam", "sts", "s3", "dynamodb", "sqs", "sns", "lambda", "cloudwatch-logs",
	"cognito", "apigateway", "cloudformation", "cloudwatch", "eventbridge",
	"stepfunctions", "secretsmanager", "kms", "ssm", "route53", "ecr", "ecs",
	"rds", "ses", "kinesis", "firehose",
}

func Default() *Config {
	cfg := &Config{}
	cfg.Region = "us-east-1"
	cfg.AccountID = "000000000000"
	cfg.Profile = "minimal"
	cfg.IAM.Mode = "enforce"
	cfg.IAM.RootAccessKey = "test"
	cfg.IAM.RootSecretKey = "test"
	cfg.Persistence.Enabled = false
	cfg.Persistence.Path = "/data"
	cfg.Gateway.Port = 4566
	cfg.Dashboard.Enabled = true
	cfg.Dashboard.Port = 4500
	cfg.Admin.Port = 4599
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Services = make(map[string]ServiceConfig)
	return cfg
}

func LoadFromFile(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) ApplyEnv() {
	if v := os.Getenv("CLOUDMOCK_REGION"); v != "" {
		c.Region = v
	}
	if v := os.Getenv("CLOUDMOCK_IAM_MODE"); v != "" {
		c.IAM.Mode = v
	}
	if v := os.Getenv("CLOUDMOCK_PERSIST"); v != "" {
		c.Persistence.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("CLOUDMOCK_PERSIST_PATH"); v != "" {
		c.Persistence.Path = v
	}
	if v := os.Getenv("CLOUDMOCK_LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("CLOUDMOCK_SERVICES"); v != "" {
		c.Profile = "custom"
		for _, svc := range strings.Split(v, ",") {
			enabled := true
			c.Services[strings.TrimSpace(svc)] = ServiceConfig{Enabled: &enabled}
		}
	}
}

func (c *Config) EnabledServices() []string {
	switch c.Profile {
	case "minimal":
		return minimalServices
	case "standard":
		return standardServices
	case "full":
		return nil // nil means all services
	case "custom":
		var services []string
		for name, sc := range c.Services {
			if sc.Enabled == nil || *sc.Enabled {
				services = append(services, name)
			}
		}
		return services
	default:
		return minimalServices
	}
}
```

- [ ] **Step 4: Add yaml dependency**

Run: `cd /Users/megan/work/neureaux/cloudmock && go get gopkg.in/yaml.v3`

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/config/... -v`
Expected: PASS

- [ ] **Step 6: Create default cloudmock.yml**

`cloudmock.yml`:
```yaml
region: us-east-1
account_id: "000000000000"
profile: minimal

iam:
  mode: enforce
  root_access_key: test
  root_secret_key: test

persistence:
  enabled: false
  path: /data

gateway:
  port: 4566

dashboard:
  enabled: true
  port: 4500

admin:
  port: 4599

logging:
  level: info
  format: json
```

- [ ] **Step 7: Commit**

```bash
git add pkg/config/ cloudmock.yml go.mod go.sum
git commit -m "feat: add configuration system with profiles, env overrides, YAML loading"
```

---

## Chunk 2: Gateway & AWS Request Routing

### Task 4: AWS Service Router

**Files:**
- Create: `pkg/routing/router.go`
- Create: `pkg/routing/router_test.go`
- Create: `pkg/routing/registry.go`
- Create: `pkg/routing/registry_test.go`

- [ ] **Step 1: Write test for service detection from Authorization header**

`pkg/routing/router_test.go`:
```go
package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/stretchr/testify/assert"
)

func TestDetectService_AuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		auth     string
		expected string
	}{
		{
			"S3 from credential scope",
			"AWS4-HMAC-SHA256 Credential=AKID/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc",
			"s3",
		},
		{
			"DynamoDB from credential scope",
			"AWS4-HMAC-SHA256 Credential=AKID/20260320/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=abc",
			"dynamodb",
		},
		{
			"STS from credential scope",
			"AWS4-HMAC-SHA256 Credential=AKID/20260320/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc",
			"sts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Authorization", tt.auth)
			assert.Equal(t, tt.expected, routing.DetectService(r))
		})
	}
}

func TestDetectService_XAmzTarget(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	assert.Equal(t, "dynamodb", routing.DetectService(r))
}

func TestDetectAction_XAmzTarget(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Amz-Target", "DynamoDB_20120810.CreateTable")
	assert.Equal(t, "CreateTable", routing.DetectAction(r))
}

func TestDetectAction_QueryString(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?Action=DescribeInstances&Version=2016-11-15", nil)
	assert.Equal(t, "DescribeInstances", routing.DetectAction(r))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/routing/... -v`
Expected: FAIL — routing package not found

- [ ] **Step 3: Implement router**

`pkg/routing/router.go`:
```go
package routing

import (
	"net/http"
	"strings"
)

// DetectService determines the target AWS service from the request.
func DetectService(r *http.Request) string {
	// 1. Authorization header credential scope
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if idx := strings.Index(auth, "Credential="); idx != -1 {
			credPart := auth[idx+11:]
			// Trim at comma if present
			if commaIdx := strings.Index(credPart, ","); commaIdx != -1 {
				credPart = credPart[:commaIdx]
			}
			parts := strings.Split(credPart, "/")
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	}

	// 2. X-Amz-Target header
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) >= 1 {
			svc := strings.ToLower(parts[0])
			if idx := strings.Index(svc, "_"); idx != -1 {
				svc = svc[:idx]
			}
			return svc
		}
	}

	return ""
}

// DetectAction determines the AWS API action from the request.
func DetectAction(r *http.Request) string {
	// 1. X-Amz-Target: DynamoDB_20120810.CreateTable -> CreateTable
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// 2. Query string: ?Action=CreateBucket
	action := r.URL.Query().Get("Action")
	if action != "" {
		return action
	}

	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/routing/... -v`
Expected: PASS

- [ ] **Step 5: Write test for service registry**

`pkg/routing/registry_test.go`:
```go
package routing_test

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	name    string
	actions []service.Action
}

func (m *mockService) Name() string                                          { return m.name }
func (m *mockService) Actions() []service.Action                             { return m.actions }
func (m *mockService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) { return nil, nil }
func (m *mockService) HealthCheck() error                                    { return nil }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	reg := routing.NewRegistry()
	svc := &mockService{name: "s3", actions: []service.Action{{Name: "CreateBucket"}}}
	reg.Register(svc)

	found, err := reg.Lookup("s3")
	require.NoError(t, err)
	assert.Equal(t, "s3", found.Name())

	_, err = reg.Lookup("nonexistent")
	assert.Error(t, err)
}

func TestRegistry_ListServices(t *testing.T) {
	reg := routing.NewRegistry()
	reg.Register(&mockService{name: "s3"})
	reg.Register(&mockService{name: "dynamodb"})

	services := reg.List()
	assert.Len(t, services, 2)
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./pkg/routing/... -v`
Expected: FAIL — NewRegistry not defined

- [ ] **Step 7: Implement registry**

`pkg/routing/registry.go`:
```go
package routing

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Registry holds all registered services and provides lookup by name.
type Registry struct {
	mu       sync.RWMutex
	services map[string]service.Service
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]service.Service),
	}
}

func (r *Registry) Register(svc service.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[svc.Name()] = svc
}

func (r *Registry) Lookup(name string) (service.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", name)
	}
	return svc, nil
}

func (r *Registry) List() []service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]service.Service, 0, len(r.services))
	for _, svc := range r.services {
		result = append(result, svc)
	}
	return result
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./pkg/routing/... -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add pkg/routing/
git commit -m "feat: add AWS service detection, action parsing, and service registry"
```

---

### Task 5: Gateway HTTP Server

**Files:**
- Create: `pkg/gateway/gateway.go`
- Create: `pkg/gateway/gateway_test.go`
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Write test for gateway routing**

`pkg/gateway/gateway_test.go`:
```go
package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
)

type echoService struct{}

func (e *echoService) Name() string              { return "s3" }
func (e *echoService) Actions() []service.Action  { return []service.Action{{Name: "ListBuckets", IAMAction: "s3:ListBuckets"}} }
func (e *echoService) HealthCheck() error          { return nil }
func (e *echoService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       map[string]string{"action": ctx.Action},
		Format:     "json",
	}, nil
}

func newTestGateway(t *testing.T) *gateway.Gateway {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none" // skip IAM for gateway routing tests
	reg := routing.NewRegistry()
	reg.Register(&echoService{})
	gw := gateway.New(cfg, reg)
	return gw
}

func TestGateway_RoutesToService(t *testing.T) {
	gw := newTestGateway(t)
	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ListBuckets")
}

func TestGateway_UnknownService_Returns503(t *testing.T) {
	gw := newTestGateway(t)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260320/us-east-1/unknown/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGateway_NoService_Returns400(t *testing.T) {
	gw := newTestGateway(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGateway_HealthEndpoint(t *testing.T) {
	gw := newTestGateway(t)
	req := httptest.NewRequest(http.MethodGet, "/_cloudmock/health", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/gateway/... -v`
Expected: FAIL — gateway package not found

- [ ] **Step 3: Implement gateway**

`pkg/gateway/gateway.go`:
```go
package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
)

// Gateway is the main HTTP server that routes AWS API requests to services.
type Gateway struct {
	cfg      *config.Config
	registry *routing.Registry
	mux      *http.ServeMux
}

func New(cfg *config.Config, registry *routing.Registry) *Gateway {
	gw := &Gateway{
		cfg:      cfg,
		registry: registry,
		mux:      http.NewServeMux(),
	}
	gw.mux.HandleFunc("/_cloudmock/health", gw.handleHealth)
	gw.mux.HandleFunc("/_cloudmock/services", gw.handleListServices)
	gw.mux.HandleFunc("/", gw.handleAWSRequest)
	return gw
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (g *Gateway) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := g.registry.List()
	names := make([]string, len(services))
	for i, svc := range services {
		names[i] = svc.Name()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"services": names})
}

func (g *Gateway) handleAWSRequest(w http.ResponseWriter, r *http.Request) {
	// Detect target service
	svcName := routing.DetectService(r)
	if svcName == "" {
		service.WriteErrorResponse(w, service.NewAWSError(
			"MissingAuthenticationToken",
			"Could not determine target AWS service from request",
			http.StatusBadRequest,
		), "json")
		return
	}

	// Look up service
	svc, err := g.registry.Lookup(svcName)
	if err != nil {
		service.WriteErrorResponse(w, service.NewAWSError(
			"ServiceUnavailable",
			"Service '"+svcName+"' is not enabled. Start it with: cloudmock start --services "+svcName,
			http.StatusServiceUnavailable,
		), "json")
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		service.WriteErrorResponse(w, service.NewAWSError(
			"InternalError",
			"Failed to read request body",
			http.StatusInternalServerError,
		), "json")
		return
	}

	// Build request context
	action := routing.DetectAction(r)
	ctx := &service.RequestContext{
		Action:     action,
		Region:     g.cfg.Region,
		AccountID:  g.cfg.AccountID,
		RawRequest: r,
		Body:       body,
		Service:    svcName,
		Params:     make(map[string]string),
	}

	// In "none" IAM mode, assign root identity
	if g.cfg.IAM.Mode == "none" {
		ctx.Identity = &service.CallerIdentity{
			AccountID:   g.cfg.AccountID,
			ARN:         "arn:aws:iam::" + g.cfg.AccountID + ":root",
			UserID:      g.cfg.AccountID,
			AccessKeyID: g.cfg.IAM.RootAccessKey,
			IsRoot:      true,
		}
	}

	// Parse query params
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			ctx.Params[key] = values[0]
		}
	}

	// Dispatch to service
	resp, handleErr := svc.HandleRequest(ctx)
	if handleErr != nil {
		if awsErr, ok := handleErr.(*service.AWSError); ok {
			service.WriteErrorResponse(w, awsErr, "json")
			return
		}
		slog.Error("service handler error", "service", svcName, "action", action, "error", handleErr)
		service.WriteErrorResponse(w, service.NewAWSError(
			"InternalError",
			"Internal service error",
			http.StatusInternalServerError,
		), "json")
		return
	}

	// Write response
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if resp.Format == "xml" {
		service.WriteXMLResponse(w, resp.StatusCode, resp.Body)
	} else {
		service.WriteJSONResponse(w, resp.StatusCode, resp.Body)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/gateway/... -v`
Expected: PASS

- [ ] **Step 5: Wire up cmd/gateway/main.go**

`cmd/gateway/main.go`:
```go
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
)

func main() {
	configPath := flag.String("config", "", "path to cloudmock.yml")
	flag.Parse()

	var cfg *config.Config
	var err error
	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath)
		if err != nil {
			slog.Error("failed to load config", "path", *configPath, "error", err)
			os.Exit(1)
		}
	} else {
		cfg = config.Default()
	}
	cfg.ApplyEnv()

	registry := routing.NewRegistry()
	gw := gateway.New(cfg, registry)

	addr := fmt.Sprintf(":%d", cfg.Gateway.Port)
	slog.Info("cloudmock gateway starting", "addr", addr, "region", cfg.Region, "iam_mode", cfg.IAM.Mode, "profile", cfg.Profile)

	if err := http.ListenAndServe(addr, gw); err != nil {
		slog.Error("gateway failed", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Verify build**

Run: `go build ./cmd/gateway/`
Expected: clean build

- [ ] **Step 7: Commit**

```bash
git add pkg/gateway/ cmd/gateway/
git commit -m "feat: add gateway HTTP server with AWS request routing and service dispatch"
```

---

## Chunk 3: IAM Engine

### Task 6: IAM Policy Types and Evaluation Engine

**Files:**
- Create: `pkg/iam/types.go`
- Create: `pkg/iam/engine.go`
- Create: `pkg/iam/engine_test.go`
- Create: `pkg/iam/matcher.go`
- Create: `pkg/iam/matcher_test.go`

- [ ] **Step 1: Write test for wildcard and ARN matching**

`pkg/iam/matcher_test.go`:
```go
package iam_test

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/stretchr/testify/assert"
)

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		match   bool
	}{
		{"*", "anything", true},
		{"s3:*", "s3:GetObject", true},
		{"s3:Get*", "s3:GetObject", true},
		{"s3:Get*", "s3:PutObject", false},
		{"s3:GetObject", "s3:GetObject", true},
		{"s3:GetObject", "s3:PutObject", false},
		{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::my-bucket/key", true},
		{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::other-bucket/key", false},
		{"arn:aws:s3:::*", "arn:aws:s3:::my-bucket", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			assert.Equal(t, tt.match, iam.WildcardMatch(tt.pattern, tt.value))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/iam/... -v`
Expected: FAIL

- [ ] **Step 3: Implement wildcard matcher**

`pkg/iam/matcher.go`:
```go
package iam

// WildcardMatch matches a pattern with * wildcards against a value.
// Supports * (matches any sequence of characters) and ? (matches single char).
func WildcardMatch(pattern, value string) bool {
	return wildcardMatchDP(pattern, value)
}

func wildcardMatchDP(pattern, str string) bool {
	p, s := len(pattern), len(str)
	dp := make([][]bool, p+1)
	for i := range dp {
		dp[i] = make([]bool, s+1)
	}
	dp[0][0] = true

	for i := 1; i <= p; i++ {
		if pattern[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		}
	}

	for i := 1; i <= p; i++ {
		for j := 1; j <= s; j++ {
			if pattern[i-1] == '*' {
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			} else if pattern[i-1] == '?' || pattern[i-1] == str[j-1] {
				dp[i][j] = dp[i-1][j-1]
			}
		}
	}
	return dp[p][s]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/iam/... -v`
Expected: PASS

- [ ] **Step 5: Write test for policy evaluation**

`pkg/iam/engine_test.go`:
```go
package iam_test

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/stretchr/testify/assert"
)

func TestEvaluate_AllowAll(t *testing.T) {
	engine := iam.NewEngine()
	engine.AddPolicy("user1", &iam.Policy{
		Version: "2012-10-17",
		Statements: []iam.Statement{
			{
				Effect:   "Allow",
				Actions:  []string{"*"},
				Resources: []string{"*"},
			},
		},
	})

	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::my-bucket/key",
	})
	assert.Equal(t, iam.Allow, result.Decision)
}

func TestEvaluate_ExplicitDeny(t *testing.T) {
	engine := iam.NewEngine()
	engine.AddPolicy("user1", &iam.Policy{
		Version: "2012-10-17",
		Statements: []iam.Statement{
			{
				Effect:   "Allow",
				Actions:  []string{"s3:*"},
				Resources: []string{"*"},
			},
			{
				Effect:   "Deny",
				Actions:  []string{"s3:DeleteBucket"},
				Resources: []string{"*"},
			},
		},
	})

	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:DeleteBucket",
		Resource:  "arn:aws:s3:::my-bucket",
	})
	assert.Equal(t, iam.Deny, result.Decision)
	assert.Equal(t, "explicit deny", result.Reason)
}

func TestEvaluate_ImplicitDeny(t *testing.T) {
	engine := iam.NewEngine()
	engine.AddPolicy("user1", &iam.Policy{
		Version: "2012-10-17",
		Statements: []iam.Statement{
			{
				Effect:   "Allow",
				Actions:  []string{"s3:GetObject"},
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
			},
		},
	})

	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:PutObject",
		Resource:  "arn:aws:s3:::my-bucket/key",
	})
	assert.Equal(t, iam.Deny, result.Decision)
	assert.Equal(t, "implicit deny", result.Reason)
}

func TestEvaluate_ResourceScoping(t *testing.T) {
	engine := iam.NewEngine()
	engine.AddPolicy("user1", &iam.Policy{
		Version: "2012-10-17",
		Statements: []iam.Statement{
			{
				Effect:   "Allow",
				Actions:  []string{"s3:GetObject"},
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
			},
		},
	})

	// Allowed: resource matches
	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::my-bucket/key.txt",
	})
	assert.Equal(t, iam.Allow, result.Decision)

	// Denied: different bucket
	result = engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::other-bucket/key.txt",
	})
	assert.Equal(t, iam.Deny, result.Decision)
}

func TestEvaluate_NoPolicies(t *testing.T) {
	engine := iam.NewEngine()
	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::my-bucket/key",
	})
	assert.Equal(t, iam.Deny, result.Decision)
	assert.Equal(t, "implicit deny", result.Reason)
}

func TestEvaluate_RootBypass(t *testing.T) {
	engine := iam.NewEngine()
	result := engine.Evaluate(&iam.EvalRequest{
		Principal: "root",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::my-bucket/key",
		IsRoot:    true,
	})
	assert.Equal(t, iam.Allow, result.Decision)
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./pkg/iam/... -v`
Expected: FAIL — types not defined

- [ ] **Step 7: Implement IAM types**

`pkg/iam/types.go`:
```go
package iam

// Decision represents an IAM evaluation result.
type Decision int

const (
	Deny  Decision = iota
	Allow
)

// Policy represents an IAM policy document.
type Policy struct {
	Version    string      `json:"Version"`
	Statements []Statement `json:"Statement"`
}

// Statement is a single policy statement.
type Statement struct {
	SID        string   `json:"Sid,omitempty"`
	Effect     string   `json:"Effect"`
	Actions    []string `json:"Action"`
	Resources  []string `json:"Resource"`
	Conditions map[string]map[string]string `json:"Condition,omitempty"`
}

// EvalRequest is the input to policy evaluation.
type EvalRequest struct {
	Principal string
	Action    string
	Resource  string
	IsRoot    bool
}

// EvalResult is the output of policy evaluation.
type EvalResult struct {
	Decision        Decision
	Reason          string
	MatchedStatement *Statement
}
```

- [ ] **Step 8: Implement policy evaluation engine**

`pkg/iam/engine.go`:
```go
package iam

import "sync"

// Engine evaluates IAM policies.
type Engine struct {
	mu       sync.RWMutex
	policies map[string][]*Policy // principal -> policies
}

func NewEngine() *Engine {
	return &Engine{
		policies: make(map[string][]*Policy),
	}
}

func (e *Engine) AddPolicy(principal string, policy *Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[principal] = append(e.policies[principal], policy)
}

func (e *Engine) RemovePolicies(principal string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.policies, principal)
}

// Evaluate implements the AWS IAM policy evaluation logic:
// 1. Root always allowed
// 2. Explicit deny takes priority
// 3. Explicit allow
// 4. Implicit deny (default)
func (e *Engine) Evaluate(req *EvalRequest) *EvalResult {
	if req.IsRoot {
		return &EvalResult{Decision: Allow, Reason: "root account"}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	policies := e.policies[req.Principal]

	var explicitAllow *Statement

	for _, policy := range policies {
		for i := range policy.Statements {
			stmt := &policy.Statements[i]
			if !matchesAction(stmt, req.Action) {
				continue
			}
			if !matchesResource(stmt, req.Resource) {
				continue
			}

			if stmt.Effect == "Deny" {
				return &EvalResult{
					Decision:        Deny,
					Reason:          "explicit deny",
					MatchedStatement: stmt,
				}
			}
			if stmt.Effect == "Allow" && explicitAllow == nil {
				explicitAllow = stmt
			}
		}
	}

	if explicitAllow != nil {
		return &EvalResult{
			Decision:        Allow,
			Reason:          "explicit allow",
			MatchedStatement: explicitAllow,
		}
	}

	return &EvalResult{Decision: Deny, Reason: "implicit deny"}
}

func matchesAction(stmt *Statement, action string) bool {
	for _, a := range stmt.Actions {
		if WildcardMatch(a, action) {
			return true
		}
	}
	return false
}

func matchesResource(stmt *Statement, resource string) bool {
	for _, r := range stmt.Resources {
		if WildcardMatch(r, resource) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./pkg/iam/... -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add pkg/iam/
git commit -m "feat: add IAM policy evaluation engine with wildcard matching"
```

---

### Task 7: IAM Credential Store and Authentication

**Files:**
- Create: `pkg/iam/store.go`
- Create: `pkg/iam/store_test.go`
- Create: `pkg/iam/auth.go`
- Create: `pkg/iam/auth_test.go`

- [ ] **Step 1: Write test for credential store**

`pkg/iam/store_test.go`:
```go
package iam_test

import (
	"testing"

	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_CreateAndLookupUser(t *testing.T) {
	store := iam.NewStore("000000000000")

	user, err := store.CreateUser("testuser")
	require.NoError(t, err)
	assert.Equal(t, "testuser", user.Name)
	assert.Contains(t, user.ARN, "arn:aws:iam::000000000000:user/testuser")

	found, err := store.GetUser("testuser")
	require.NoError(t, err)
	assert.Equal(t, user.ARN, found.ARN)
}

func TestStore_CreateAccessKey(t *testing.T) {
	store := iam.NewStore("000000000000")
	_, err := store.CreateUser("testuser")
	require.NoError(t, err)

	key, err := store.CreateAccessKey("testuser")
	require.NoError(t, err)
	assert.NotEmpty(t, key.AccessKeyID)
	assert.NotEmpty(t, key.SecretAccessKey)

	identity, err := store.LookupAccessKey(key.AccessKeyID)
	require.NoError(t, err)
	assert.Equal(t, "testuser", identity.UserName)
	assert.Equal(t, "000000000000", identity.AccountID)
}

func TestStore_RootAccount(t *testing.T) {
	store := iam.NewStore("000000000000")
	store.InitRoot("AKID_ROOT", "SECRET_ROOT")

	identity, err := store.LookupAccessKey("AKID_ROOT")
	require.NoError(t, err)
	assert.True(t, identity.IsRoot)
	assert.Equal(t, "000000000000", identity.AccountID)
}

func TestStore_AttachPolicy(t *testing.T) {
	store := iam.NewStore("000000000000")
	_, _ = store.CreateUser("testuser")

	policy := &iam.Policy{
		Version: "2012-10-17",
		Statements: []iam.Statement{
			{Effect: "Allow", Actions: []string{"s3:*"}, Resources: []string{"*"}},
		},
	}
	err := store.AttachUserPolicy("testuser", "s3-full", policy)
	require.NoError(t, err)

	policies := store.GetUserPolicies("testuser")
	assert.Len(t, policies, 1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/iam/... -v -run TestStore`
Expected: FAIL

- [ ] **Step 3: Implement credential store**

`pkg/iam/store.go`:
```go
package iam

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

type User struct {
	Name     string
	ARN      string
	UserID   string
	Policies map[string]*Policy
}

type AccessKey struct {
	AccessKeyID     string
	SecretAccessKey  string
	UserName        string
	AccountID       string
	IsRoot          bool
}

type Store struct {
	mu         sync.RWMutex
	accountID  string
	users      map[string]*User
	accessKeys map[string]*AccessKey // accessKeyID -> AccessKey
}

func NewStore(accountID string) *Store {
	return &Store{
		accountID:  accountID,
		users:      make(map[string]*User),
		accessKeys: make(map[string]*AccessKey),
	}
}

func (s *Store) InitRoot(accessKeyID, secretKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessKeys[accessKeyID] = &AccessKey{
		AccessKeyID:    accessKeyID,
		SecretAccessKey: secretKey,
		UserName:       "root",
		AccountID:      s.accountID,
		IsRoot:         true,
	}
}

func (s *Store) CreateUser(name string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[name]; exists {
		return nil, fmt.Errorf("user already exists: %s", name)
	}
	user := &User{
		Name:     name,
		ARN:      fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, name),
		UserID:   generateID("AIDA"),
		Policies: make(map[string]*Policy),
	}
	s.users[name] = user
	return user, nil
}

func (s *Store) GetUser(name string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[name]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", name)
	}
	return user, nil
}

func (s *Store) CreateAccessKey(userName string) (*AccessKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userName)
	}
	key := &AccessKey{
		AccessKeyID:    generateID("AKIA"),
		SecretAccessKey: generateSecret(),
		UserName:       user.Name,
		AccountID:      s.accountID,
		IsRoot:         false,
	}
	s.accessKeys[key.AccessKeyID] = key
	return key, nil
}

func (s *Store) LookupAccessKey(accessKeyID string) (*AccessKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.accessKeys[accessKeyID]
	if !ok {
		return nil, fmt.Errorf("access key not found: %s", accessKeyID)
	}
	return key, nil
}

func (s *Store) AttachUserPolicy(userName, policyName string, policy *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userName]
	if !ok {
		return fmt.Errorf("user not found: %s", userName)
	}
	user.Policies[policyName] = policy
	return nil
}

func (s *Store) GetUserPolicies(userName string) []*Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userName]
	if !ok {
		return nil
	}
	policies := make([]*Policy, 0, len(user.Policies))
	for _, p := range user.Policies {
		policies = append(policies, p)
	}
	return policies
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func generateSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/iam/... -v -run TestStore`
Expected: PASS

- [ ] **Step 5: Write test for auth middleware extraction**

`pkg/iam/auth_test.go`:
```go
package iam_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAccessKeyID(t *testing.T) {
	tests := []struct {
		name     string
		auth     string
		expected string
		err      bool
	}{
		{
			"valid sig v4",
			"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc",
			"AKIAIOSFODNN7EXAMPLE",
			false,
		},
		{
			"empty header",
			"",
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.auth != "" {
				r.Header.Set("Authorization", tt.auth)
			}
			keyID, err := iam.ExtractAccessKeyID(r)
			if tt.err {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyID)
			}
		})
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./pkg/iam/... -v -run TestExtract`
Expected: FAIL

- [ ] **Step 7: Implement auth extraction**

`pkg/iam/auth.go`:
```go
package iam

import (
	"fmt"
	"net/http"
	"strings"
)

// ExtractAccessKeyID extracts the AWS access key ID from the Authorization header.
func ExtractAccessKeyID(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	idx := strings.Index(auth, "Credential=")
	if idx == -1 {
		return "", fmt.Errorf("malformed Authorization header: no Credential")
	}

	credPart := auth[idx+11:]
	if commaIdx := strings.Index(credPart, ","); commaIdx != -1 {
		credPart = credPart[:commaIdx]
	}

	parts := strings.Split(credPart, "/")
	if len(parts) < 1 || parts[0] == "" {
		return "", fmt.Errorf("malformed Authorization header: empty access key")
	}

	return parts[0], nil
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./pkg/iam/... -v`
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git add pkg/iam/
git commit -m "feat: add IAM credential store, access key management, and auth extraction"
```

---

## Chunk 4: Docker & Minimal Service Integration

### Task 8: Minimal S3 Service (Validate Full Lifecycle)

**Files:**
- Create: `services/s3/service.go`
- Create: `services/s3/service_test.go`
- Create: `services/s3/handlers.go`
- Create: `services/s3/store.go`

- [ ] **Step 1: Write test for S3 bucket CRUD**

`services/s3/service_test.go`:
```go
package s3_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStack(t *testing.T) *gateway.Gateway {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(s3svc.New())
	return gateway.New(cfg, reg)
}

func s3Request(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=test/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	return r
}

func TestS3_CreateAndListBuckets(t *testing.T) {
	gw := newTestStack(t)

	// Create bucket
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodPut, "/my-test-bucket"))
	assert.Equal(t, http.StatusOK, w.Code)

	// List buckets
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodGet, "/"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "my-test-bucket")
}

func TestS3_DeleteBucket(t *testing.T) {
	gw := newTestStack(t)

	// Create
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodPut, "/delete-me"))
	require.Equal(t, http.StatusOK, w.Code)

	// Delete
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodDelete, "/delete-me"))
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify gone
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodGet, "/"))
	assert.NotContains(t, w.Body.String(), "delete-me")
}

func TestS3_HeadBucket(t *testing.T) {
	gw := newTestStack(t)

	// 404 for nonexistent
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodHead, "/nope"))
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Create then head
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodPut, "/exists"))
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	gw.ServeHTTP(w, s3Request(http.MethodHead, "/exists"))
	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./services/s3/... -v`
Expected: FAIL — package not found

- [ ] **Step 3: Implement S3 store**

`services/s3/store.go`:
```go
package s3

import (
	"sync"
	"time"
)

type Bucket struct {
	Name         string
	CreationDate time.Time
}

type Store struct {
	mu      sync.RWMutex
	buckets map[string]*Bucket
}

func NewStore() *Store {
	return &Store{
		buckets: make(map[string]*Bucket),
	}
}

func (s *Store) CreateBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.buckets[name]; exists {
		return &bucketError{code: "BucketAlreadyOwnedByYou", msg: "Your previous request to create the named bucket succeeded"}
	}
	s.buckets[name] = &Bucket{
		Name:         name,
		CreationDate: time.Now(),
	}
	return nil
}

func (s *Store) DeleteBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.buckets[name]; !exists {
		return &bucketError{code: "NoSuchBucket", msg: "The specified bucket does not exist"}
	}
	delete(s.buckets, name)
	return nil
}

func (s *Store) HeadBucket(name string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, exists := s.buckets[name]; !exists {
		return &bucketError{code: "NoSuchBucket", msg: "The specified bucket does not exist"}
	}
	return nil
}

func (s *Store) ListBuckets() []*Bucket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buckets := make([]*Bucket, 0, len(s.buckets))
	for _, b := range s.buckets {
		buckets = append(buckets, b)
	}
	return buckets
}

type bucketError struct {
	code string
	msg  string
}

func (e *bucketError) Error() string { return e.code + ": " + e.msg }
```

- [ ] **Step 4: Implement S3 handlers**

`services/s3/handlers.go`:
```go
package s3

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

type ListAllMyBucketsResult struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	XMLNS   string         `xml:"xmlns,attr"`
	Owner   Owner          `xml:"Owner"`
	Buckets BucketList     `xml:"Buckets"`
}

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type BucketList struct {
	Bucket []BucketEntry `xml:"Bucket"`
}

type BucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

func (s *S3Service) handleListBuckets(ctx *service.RequestContext) (*service.Response, error) {
	buckets := s.store.ListBuckets()
	entries := make([]BucketEntry, len(buckets))
	for i, b := range buckets {
		entries[i] = BucketEntry{
			Name:         b.Name,
			CreationDate: b.CreationDate.Format(time.RFC3339),
		}
	}
	result := ListAllMyBucketsResult{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: Owner{
			ID:          ctx.AccountID,
			DisplayName: "cloudmock",
		},
		Buckets: BucketList{Bucket: entries},
	}
	return &service.Response{StatusCode: http.StatusOK, Body: result, Format: "xml"}, nil
}

func (s *S3Service) handleCreateBucket(ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	if bucket == "" {
		return nil, service.ErrValidation("Bucket name is required")
	}
	if err := s.store.CreateBucket(bucket); err != nil {
		if be, ok := err.(*bucketError); ok {
			return nil, service.NewAWSError(be.code, be.msg, http.StatusConflict)
		}
		return nil, err
	}
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       nil,
		Format:     "xml",
		Headers:    map[string]string{"Location": "/" + bucket},
	}, nil
}

func (s *S3Service) handleDeleteBucket(ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	if err := s.store.DeleteBucket(bucket); err != nil {
		if be, ok := err.(*bucketError); ok {
			return nil, service.NewAWSError(be.code, be.msg, http.StatusNotFound)
		}
		return nil, err
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: "xml"}, nil
}

func (s *S3Service) handleHeadBucket(ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	if err := s.store.HeadBucket(bucket); err != nil {
		return nil, service.NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
	}
	return &service.Response{StatusCode: http.StatusOK, Format: "xml"}, nil
}

func extractBucketName(ctx *service.RequestContext) string {
	path := ctx.RawRequest.URL.Path
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
```

- [ ] **Step 5: Implement S3 service**

`services/s3/service.go`:
```go
package s3

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

type S3Service struct {
	store *Store
}

func New() *S3Service {
	return &S3Service{store: NewStore()}
}

func (s *S3Service) Name() string { return "s3" }

func (s *S3Service) Actions() []service.Action {
	return []service.Action{
		{Name: "ListBuckets", Method: http.MethodGet, IAMAction: "s3:ListAllMyBuckets"},
		{Name: "CreateBucket", Method: http.MethodPut, IAMAction: "s3:CreateBucket"},
		{Name: "DeleteBucket", Method: http.MethodDelete, IAMAction: "s3:DeleteBucket"},
		{Name: "HeadBucket", Method: http.MethodHead, IAMAction: "s3:HeadBucket"},
	}
}

func (s *S3Service) HealthCheck() error { return nil }

func (s *S3Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	path := ctx.RawRequest.URL.Path
	method := ctx.RawRequest.Method

	// Route based on method and path
	bucket := extractBucketName(ctx)

	switch {
	case method == http.MethodGet && bucket == "":
		return s.handleListBuckets(ctx)
	case method == http.MethodPut && bucket != "" && !hasObjectKey(ctx):
		return s.handleCreateBucket(ctx)
	case method == http.MethodDelete && bucket != "" && !hasObjectKey(ctx):
		return s.handleDeleteBucket(ctx)
	case method == http.MethodHead && bucket != "":
		return s.handleHeadBucket(ctx)
	default:
		return nil, service.NewAWSError("NotImplemented",
			"The requested action is not yet implemented in cloudmock S3",
			http.StatusNotImplemented)
	}
}

func hasObjectKey(ctx *service.RequestContext) bool {
	path := ctx.RawRequest.URL.Path
	parts := splitPath(path)
	return len(parts) > 1
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range splitTrimmed(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitTrimmed(s, sep string) []string {
	result := []string{}
	for _, part := range split(s, sep) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func split(s, sep string) []string {
	var result []string
	for s != "" {
		i := indexOf(s, sep)
		if i < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:i])
		s = s[i+len(sep):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./services/s3/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add services/s3/
git commit -m "feat: add minimal S3 service with bucket CRUD operations"
```

---

### Task 9: Dockerfiles and Docker Compose

**Files:**
- Create: `docker/base.Dockerfile`
- Create: `docker/gateway.Dockerfile`
- Create: `services/s3/Dockerfile`
- Create: `docker-compose.yml`

- [ ] **Step 1: Create base Dockerfile**

`docker/base.Dockerfile`:
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/_health || exit 1
```

- [ ] **Step 2: Create gateway Dockerfile**

`docker/gateway.Dockerfile`:
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./cmd/gateway

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /gateway /app/gateway
COPY cloudmock.yml /app/cloudmock.yml
EXPOSE 4566 4500 4599
ENTRYPOINT ["/app/gateway"]
CMD ["--config", "/app/cloudmock.yml"]
```

- [ ] **Step 3: Create S3 service Dockerfile**

`services/s3/Dockerfile`:
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /s3-service ./services/s3/cmd

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /s3-service /app/s3-service
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/_health || exit 1
ENTRYPOINT ["/app/s3-service"]
```

- [ ] **Step 4: Create S3 standalone service entrypoint**

Create `services/s3/cmd/main.go`:
```go
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	s3svc "github.com/neureaux/cloudmock/services/s3"
	"github.com/neureaux/cloudmock/pkg/service"
)

func main() {
	s3 := s3svc.New()
	mux := http.NewServeMux()

	mux.HandleFunc("/_health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": s3.Name(),
			"status":  "ok",
			"actions": s3.Actions(),
		})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := service.ParseRequestBody(r)
		ctx := &service.RequestContext{
			RawRequest: r,
			Body:       body,
			Service:    "s3",
			Region:     "us-east-1",
			AccountID:  "000000000000",
			Params:     make(map[string]string),
			Identity: &service.CallerIdentity{
				AccountID: "000000000000",
				IsRoot:    true,
			},
		}
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				ctx.Params[key] = values[0]
			}
		}

		resp, err := s3.HandleRequest(ctx)
		if err != nil {
			if awsErr, ok := err.(*service.AWSError); ok {
				service.WriteErrorResponse(w, awsErr, "xml")
				return
			}
			service.WriteErrorResponse(w, service.NewAWSError("InternalError", err.Error(), 500), "xml")
			return
		}

		if resp.Format == "xml" {
			service.WriteXMLResponse(w, resp.StatusCode, resp.Body)
		} else {
			service.WriteJSONResponse(w, resp.StatusCode, resp.Body)
		}
	})

	slog.Info("S3 service starting", "port", 8080)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("S3 service failed", "error", err)
	}
}
```

- [ ] **Step 5: Create docker-compose.yml**

`docker-compose.yml`:
```yaml
version: "3.8"

networks:
  cloudmock:
    driver: bridge

services:
  gateway:
    build:
      context: .
      dockerfile: docker/gateway.Dockerfile
    container_name: cloudmock-gateway
    ports:
      - "4566:4566"
      - "127.0.0.1:4500:4500"
      - "127.0.0.1:4599:4599"
    environment:
      - CLOUDMOCK_REGION=${CLOUDMOCK_REGION:-us-east-1}
      - CLOUDMOCK_IAM_MODE=${CLOUDMOCK_IAM_MODE:-enforce}
      - CLOUDMOCK_LOG_LEVEL=${CLOUDMOCK_LOG_LEVEL:-info}
    networks:
      - cloudmock
    labels:
      cloudmock.component: gateway
    depends_on:
      s3:
        condition: service_healthy

  s3:
    build:
      context: .
      dockerfile: services/s3/Dockerfile
    container_name: cloudmock-s3
    networks:
      - cloudmock
    labels:
      cloudmock.service: s3
      cloudmock.tier: "1"
    expose:
      - "8080"
```

- [ ] **Step 6: Verify Docker build**

Run: `cd /Users/megan/work/neureaux/cloudmock && docker compose build`
Expected: successful build of both images

- [ ] **Step 7: Commit**

```bash
git add docker/ services/s3/cmd/ services/s3/Dockerfile docker-compose.yml
git commit -m "feat: add Dockerfiles and Docker Compose for gateway and S3 service"
```

---

### Task 10: End-to-End Smoke Test

**Files:**
- Create: `tests/smoke/smoke_test.go`

- [ ] **Step 1: Write smoke test using AWS SDK**

`tests/smoke/smoke_test.go`:
```go
//go:build smoke

package smoke_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cloudmockConfig(t *testing.T) aws.Config {
	t.Helper()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	require.NoError(t, err)
	return cfg
}

func TestSmoke_S3_BucketCRUD(t *testing.T) {
	cfg := cloudmockConfig(t)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
		o.UsePathStyle = true
	})
	ctx := context.TODO()

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String("smoke-test-bucket"),
	})
	require.NoError(t, err)

	// List buckets
	listOut, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	found := false
	for _, b := range listOut.Buckets {
		if *b.Name == "smoke-test-bucket" {
			found = true
		}
	}
	assert.True(t, found, "smoke-test-bucket should be in bucket list")

	// Delete bucket
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String("smoke-test-bucket"),
	})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Add AWS SDK dependency**

Run: `cd /Users/megan/work/neureaux/cloudmock && go get github.com/aws/aws-sdk-go-v2 github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/credentials github.com/aws/aws-sdk-go-v2/service/s3`

- [ ] **Step 3: Run smoke test (requires Docker Compose running)**

Run: `docker compose up -d && sleep 5 && go test ./tests/smoke/... -tags smoke -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add tests/smoke/ go.mod go.sum
git commit -m "feat: add end-to-end smoke test for S3 bucket CRUD via AWS SDK"
```

---

### Task 11: IAM Integration in Gateway

**Files:**
- Modify: `pkg/gateway/gateway.go`
- Create: `pkg/gateway/iam_middleware.go`
- Create: `pkg/gateway/iam_middleware_test.go`

- [ ] **Step 1: Write test for IAM middleware in enforce mode**

`pkg/gateway/iam_middleware_test.go`:
```go
package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
)

type allowAllService struct{}

func (a *allowAllService) Name() string              { return "s3" }
func (a *allowAllService) Actions() []service.Action  { return []service.Action{{Name: "ListBuckets", IAMAction: "s3:ListAllMyBuckets"}} }
func (a *allowAllService) HealthCheck() error          { return nil }
func (a *allowAllService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: map[string]string{"ok": "true"}, Format: "json"}, nil
}

func TestIAMEnforce_RootAllowed(t *testing.T) {
	cfg := config.Default()
	cfg.IAM.Mode = "enforce"
	cfg.IAM.RootAccessKey = "ROOTKEY"
	cfg.IAM.RootSecretKey = "ROOTSECRET"

	store := iampkg.NewStore(cfg.AccountID)
	store.InitRoot(cfg.IAM.RootAccessKey, cfg.IAM.RootSecretKey)
	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(&allowAllService{})
	gw := gateway.NewWithIAM(cfg, reg, store, engine)

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIAMEnforce_UnknownKeyDenied(t *testing.T) {
	cfg := config.Default()
	cfg.IAM.Mode = "enforce"

	store := iampkg.NewStore(cfg.AccountID)
	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(&allowAllService{})
	gw := gateway.NewWithIAM(cfg, reg, store, engine)

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=UNKNOWN/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestIAMNone_BypassesAuth(t *testing.T) {
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(&allowAllService{})
	gw := gateway.New(cfg, reg)

	req := httptest.NewRequest(http.MethodGet, "/?Action=ListBuckets", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=anything/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/gateway/... -v -run TestIAM`
Expected: FAIL — NewWithIAM not defined

- [ ] **Step 3: Implement IAM middleware and NewWithIAM**

`pkg/gateway/iam_middleware.go`:
```go
package gateway

import (
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/service"
	"net/http"
)

// authenticateRequest extracts credentials and resolves identity.
func (g *Gateway) authenticateRequest(r *http.Request) (*service.CallerIdentity, *service.AWSError) {
	if g.cfg.IAM.Mode == "none" {
		return &service.CallerIdentity{
			AccountID:   g.cfg.AccountID,
			ARN:         "arn:aws:iam::" + g.cfg.AccountID + ":root",
			UserID:      g.cfg.AccountID,
			AccessKeyID: "anonymous",
			IsRoot:      true,
		}, nil
	}

	keyID, err := iampkg.ExtractAccessKeyID(r)
	if err != nil {
		return nil, service.NewAWSError("MissingAuthenticationToken",
			"Missing or malformed Authorization header", http.StatusForbidden)
	}

	accessKey, err := g.store.LookupAccessKey(keyID)
	if err != nil {
		return nil, service.NewAWSError("InvalidClientTokenId",
			"The security token included in the request is invalid", http.StatusForbidden)
	}

	return &service.CallerIdentity{
		AccountID:   accessKey.AccountID,
		ARN:         "arn:aws:iam::" + accessKey.AccountID + ":user/" + accessKey.UserName,
		UserID:      accessKey.UserName,
		AccessKeyID: accessKey.AccessKeyID,
		IsRoot:      accessKey.IsRoot,
	}, nil
}

// authorizeRequest checks if the caller has permission for the action.
func (g *Gateway) authorizeRequest(identity *service.CallerIdentity, iamAction, resource string) *service.AWSError {
	if g.cfg.IAM.Mode != "enforce" {
		return nil
	}

	result := g.engine.Evaluate(&iampkg.EvalRequest{
		Principal: identity.UserID,
		Action:    iamAction,
		Resource:  resource,
		IsRoot:    identity.IsRoot,
	})

	if result.Decision == iampkg.Deny {
		return service.ErrAccessDenied(iamAction)
	}
	return nil
}
```

Then update `gateway.go` to add `NewWithIAM` and integrate auth into request handling:

Add to `pkg/gateway/gateway.go` — add `store` and `engine` fields to Gateway struct, add `NewWithIAM` constructor, and update `handleAWSRequest` to call `authenticateRequest` and `authorizeRequest`.

```go
// Add to Gateway struct:
store  *iampkg.Store
engine *iampkg.Engine

// Add constructor:
func NewWithIAM(cfg *config.Config, registry *routing.Registry, store *iampkg.Store, engine *iampkg.Engine) *Gateway {
	gw := New(cfg, registry)
	gw.store = store
	gw.engine = engine
	return gw
}
```

Update `handleAWSRequest` to use `authenticateRequest` and `authorizeRequest` before dispatching.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/gateway/... -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/gateway/
git commit -m "feat: add IAM authentication and authorization middleware to gateway"
```

---

### Task 12: Integration Test — Full Stack with IAM

**Files:**
- Create: `tests/integration/iam_integration_test.go`

- [ ] **Step 1: Write integration test**

`tests/integration/iam_integration_test.go`:
```go
package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFullStack(t *testing.T, iamMode string) (*gateway.Gateway, *iampkg.Store, *iampkg.Engine) {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = iamMode
	cfg.IAM.RootAccessKey = "ROOTKEY"

	store := iampkg.NewStore(cfg.AccountID)
	store.InitRoot("ROOTKEY", "ROOTSECRET")
	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(s3svc.New())
	gw := gateway.NewWithIAM(cfg, reg, store, engine)

	return gw, store, engine
}

func TestFullStack_RootCanCreateBucket(t *testing.T) {
	gw, _, _ := setupFullStack(t, "enforce")

	req := httptest.NewRequest(http.MethodPut, "/test-bucket", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=ROOTKEY/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFullStack_UserWithPolicyCanListBuckets(t *testing.T) {
	gw, store, engine := setupFullStack(t, "enforce")

	user, err := store.CreateUser("reader")
	require.NoError(t, err)
	key, err := store.CreateAccessKey("reader")
	require.NoError(t, err)

	engine.AddPolicy(user.Name, &iampkg.Policy{
		Version: "2012-10-17",
		Statements: []iampkg.Statement{
			{Effect: "Allow", Actions: []string{"s3:ListAllMyBuckets"}, Resources: []string{"*"}},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+key.AccessKeyID+"/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFullStack_UserWithoutPolicyDenied(t *testing.T) {
	gw, _, _ := setupFullStack(t, "enforce")
	_, _ = gw, nil // gateway created with root only

	store := iampkg.NewStore("000000000000")
	store.InitRoot("ROOTKEY", "ROOTSECRET")

	// Create user with no policies via the already-initialized stack
	// (we need to use the stack's store — re-setup)
	gw2, store2, _ := setupFullStack(t, "enforce")
	_, err := store2.CreateUser("noperms")
	require.NoError(t, err)
	key, err := store2.CreateAccessKey("noperms")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+key.AccessKeyID+"/20260320/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw2.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./tests/integration/... -v`
Expected: PASS

- [ ] **Step 3: Run all tests**

Run: `go test ./... -v -count=1`
Expected: ALL PASS (excluding smoke tests which need Docker)

- [ ] **Step 4: Commit**

```bash
git add tests/integration/
git commit -m "feat: add integration tests for full stack — gateway + IAM + S3"
```

---

## Summary

This foundation plan produces:
- Go project with `pkg/service` framework (types, errors, request/response)
- Configuration system with profiles, env overrides, YAML loading
- Gateway HTTP server with AWS request routing and service dispatch
- IAM policy evaluation engine with wildcard matching
- IAM credential store with access key management
- IAM middleware in gateway (enforce/authenticate/none modes)
- Minimal S3 service with bucket CRUD
- Dockerfiles and Docker Compose
- Unit tests, integration tests, and smoke tests

**Next plan:** Plan 2 — Core Tier 1 Services (full S3 with objects, DynamoDB, SQS, SNS, Lambda, STS, KMS, Secrets Manager, SSM)
