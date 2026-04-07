// Package main is the CloudMock service code generator.
// It reads AWS service models from botocore and generates the 4-file service
// scaffold: service.go, store.go, handlers.go, service_test.go.
//
// Usage:
//
//	# Generate a complete new service
//	go run ./codegen --service=efs --botocore=/path/to/botocore/data
//
//	# Generate only specific operations (for backfilling existing services)
//	go run ./codegen --service=lambda --operations=PublishLayerVersion,GetLayerVersion --append
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

func main() {
	serviceName := flag.String("service", "", "botocore service name (e.g., efs, guardduty)")
	botocorePath := flag.String("botocore", findBotocore(), "path to botocore/data directory")
	outputDir := flag.String("output", "", "output directory (default: services/<package>)")
	operations := flag.String("operations", "", "comma-separated operation names to generate (default: all)")
	appendMode := flag.Bool("append", false, "append to existing service files instead of creating new")
	flag.Parse()

	if *serviceName == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run ./codegen --service=<name>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	model, err := LoadModel(*botocorePath, *serviceName)
	if err != nil {
		log.Fatalf("load model: %v", err)
	}

	pkgName := goPackageName(*serviceName)
	if *outputDir == "" {
		*outputDir = filepath.Join("services", pkgName)
	}

	// Filter operations if specified
	ops := model.Operations
	if *operations != "" {
		opList := strings.Split(*operations, ",")
		ops = model.FilterOperations(opList)
		if len(ops) == 0 {
			log.Fatalf("no matching operations found for: %s", *operations)
		}
	}

	fmt.Printf("cloudmock codegen: %s (%s protocol, %d operations, %d shapes)\n",
		*serviceName, model.Metadata.Protocol, len(ops), len(model.Shapes))

	if *appendMode {
		fmt.Printf("  append mode: generating %d operations\n", len(ops))
		generateAppend(*outputDir, pkgName, model, ops)
	} else {
		if err := os.MkdirAll(*outputDir, 0o755); err != nil {
			log.Fatalf("create output dir: %v", err)
		}
		generateFull(*outputDir, pkgName, model, ops)
	}

	fmt.Printf("  output: %s/\n", *outputDir)
	fmt.Println("  next: add import + registration to cmd/gateway/main.go")
}

// templateData holds all data needed by the templates.
type templateData struct {
	Package      string
	ServiceName  string // botocore name
	SigningName  string // SigV4 signing name
	Protocol     string // "json", "rest-json", "rest-xml", "query"
	TargetPrefix string // for json protocol X-Amz-Target
	Operations   []opData
	Structs      string // generated Go struct definitions
	NeedsTime    bool   // whether time.Time is used in structs
}

type opData struct {
	Name       string
	HTTPMethod string
	HTTPPath   string
	IAMAction  string
	InputName  string // Go struct name for request
	OutputName string // Go struct name for response
	HasInput   bool
	HasOutput  bool
}

func buildTemplateData(pkgName string, model *BotocoreModel, ops map[string]Operation) templateData {
	data := templateData{
		Package:      pkgName,
		ServiceName:  model.Metadata.EndpointPrefix,
		SigningName:  model.SigningName(),
		Protocol:     model.Metadata.Protocol,
		TargetPrefix: model.Metadata.TargetPrefix,
	}

	opNames := make([]string, 0, len(ops))
	for name := range ops {
		opNames = append(opNames, name)
	}
	sort.Strings(opNames)

	for _, name := range opNames {
		op := ops[name]
		od := opData{
			Name:       name,
			HTTPMethod: op.HTTP.Method,
			HTTPPath:   op.HTTP.RequestURI,
			IAMAction:  data.SigningName + ":" + name,
		}
		if op.Input != nil {
			od.InputName = exportedName(op.Input.Shape)
			od.HasInput = true
		}
		if op.Output != nil {
			od.OutputName = exportedName(op.Output.Shape)
			od.HasOutput = true
		}
		data.Operations = append(data.Operations, od)
	}

	data.Structs = GenerateStructs(model, ops)
	data.NeedsTime = strings.Contains(data.Structs, "time.Time")

	return data
}

func generateFull(outDir, pkgName string, model *BotocoreModel, ops map[string]Operation) {
	data := buildTemplateData(pkgName, model, ops)

	writeTemplate(filepath.Join(outDir, "service.go"), serviceTemplate, data)
	writeTemplate(filepath.Join(outDir, "store.go"), storeTemplate, data)
	writeTemplate(filepath.Join(outDir, "handlers.go"), handlersTemplate, data)
	writeTemplate(filepath.Join(outDir, "service_test.go"), testTemplate, data)
}

func generateAppend(outDir, pkgName string, model *BotocoreModel, ops map[string]Operation) {
	data := buildTemplateData(pkgName, model, ops)

	// In append mode, write handler stubs and action list to separate files
	stubFile := filepath.Join(outDir, "generated_handlers.go")
	writeTemplate(stubFile, appendHandlersTemplate, data)
	fmt.Printf("  generated handlers: %s\n", stubFile)
	fmt.Println("  manual step: add Actions and HandleRequest cases to service.go")
}

func writeTemplate(path string, tmplText string, data templateData) {
	funcMap := template.FuncMap{
		"title": httpMethodGoName,
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("execute template for %s: %v", path, err)
	}
}

// findBotocore attempts to find the botocore data directory.
func findBotocore() string {
	candidates := []string{
		"/opt/homebrew/lib/python3.14/site-packages/botocore/data",
		"/opt/homebrew/lib/python3.13/site-packages/botocore/data",
		"/opt/homebrew/lib/python3.12/site-packages/botocore/data",
		"/usr/lib/python3/dist-packages/botocore/data",
		"/usr/local/lib/python3.12/dist-packages/botocore/data",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Try pip show
	return "/opt/homebrew/lib/python3.14/site-packages/botocore/data"
}

// ── Templates ────────────────────────────────────────────────────────────────

var serviceTemplate = `package {{.Package}}

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS {{.SigningName}} service.
type Service struct {
	store *Store
}

// New returns a new {{.Package}} Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "{{.SigningName}}" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
{{- range .Operations}}
		{Name: "{{.Name}}", Method: http.Method{{.HTTPMethod | title}}, IAMAction: "{{.IAMAction}}"},
{{- end}}
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
{{- range .Operations}}
	case "{{.Name}}":
		return handle{{.Name}}(ctx, s.store)
{{- end}}
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
`

var storeTemplate = `package {{.Package}}

import (
	"fmt"
	"crypto/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Store is the in-memory data store for {{.Package}} resources.
type Store struct {
	mu        sync.RWMutex
	resources map[string]map[string]any // resourceType -> id -> resource
	accountID string
	region    string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		resources: make(map[string]map[string]any),
		accountID: accountID,
		region:    region,
	}
}

func (s *Store) put(resourceType, id string, resource any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resources[resourceType] == nil {
		s.resources[resourceType] = make(map[string]any)
	}
	s.resources[resourceType][id] = resource
}

func (s *Store) get(resourceType, id string) (any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.resources[resourceType]; ok {
		if r, ok := m[id]; ok {
			return r, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %s not found", resourceType, id), http.StatusBadRequest)
}

func (s *Store) list(resourceType string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []any
	for _, r := range s.resources[resourceType] {
		out = append(out, r)
	}
	return out
}

func (s *Store) delete(resourceType, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.resources[resourceType]; ok {
		if _, ok := m[id]; ok {
			delete(m, id)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %s not found", resourceType, id), http.StatusBadRequest)
}

func (s *Store) buildArn(resourceType, id string) string {
	return fmt.Sprintf("arn:aws:{{.SigningName}}:%s:%s:%s/%s", s.region, s.accountID, resourceType, id)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var _ = time.Now // ensure time is used
`

var handlersTemplate = `package {{.Package}}

import (
	"encoding/json"
	"net/http"
{{- if .NeedsTime}}
	"time"
{{- end}}

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

{{.Structs}}

// ── Handler helpers ──────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────
{{range .Operations}}
func handle{{.Name}}(ctx *service.RequestContext, store *Store) (*service.Response, error) {
{{- if .HasInput}}
	var req {{.InputName}}
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
{{- end}}
	// TODO: implement {{.Name}} business logic
	return jsonOK(map[string]any{"status": "ok", "action": "{{.Name}}"})
}
{{end}}
`

var testTemplate = `package {{.Package}}_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/{{.Package}}"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
{{- if .TargetPrefix}}
	req.Header.Set("X-Amz-Target", "{{.TargetPrefix}}."+action)
{{- else}}
	req.Header.Set("X-Amz-Target", "{{.SigningName}}."+action)
{{- end}}
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/{{.SigningName}}/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

{{range .Operations}}
func Test{{.Name}}(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "{{.Name}}", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("{{.Name}}: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
{{end}}
`

var appendHandlersTemplate = `package {{.Package}}

// Generated handler stubs for backfill operations.
// Add these operations to Actions() and HandleRequest() in service.go.

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

/*
Actions to add to service.go Actions():
{{- range .Operations}}
{Name: "{{.Name}}", Method: http.Method{{.HTTPMethod | title}}, IAMAction: "{{.IAMAction}}"},
{{- end}}

Cases to add to HandleRequest():
{{- range .Operations}}
case "{{.Name}}":
	return handle{{.Name}}(ctx, s.store)
{{- end}}
*/
{{range .Operations}}
func handle{{.Name}}(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	// TODO: implement {{.Name}} business logic
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       map[string]any{"status": "ok"},
		Format:     service.FormatJSON,
	}, nil
}
{{end}}

var _ = http.StatusOK // ensure import is used
var _ service.Action  // ensure import is used
`

// httpMethodGoName converts "GET" to "Get", "POST" to "Post", etc.
func httpMethodGoName(method string) string {
	if method == "" {
		return "Post"
	}
	return strings.ToUpper(method[:1]) + strings.ToLower(method[1:])
}
