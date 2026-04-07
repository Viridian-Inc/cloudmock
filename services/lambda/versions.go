package lambda

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// FunctionVersion represents a published version of a Lambda function.
type FunctionVersion struct {
	FunctionName string
	FunctionArn  string
	Version      string
	Description  string
	Runtime      string
	Handler      string
	Role         string
	CodeSha256   string
	CodeSize     int64
	Timeout      int
	MemorySize   int
	LastModified string
}

// FunctionAlias maps a name to a function version.
type FunctionAlias struct {
	AliasArn        string
	Name            string
	FunctionVersion string
	Description     string
}

// FunctionURLConfig stores a Lambda function URL configuration.
type FunctionURLConfig struct {
	FunctionArn string
	FunctionUrl string
	AuthType    string // "NONE" or "AWS_IAM"
	Cors        *FunctionURLCors
	CreationTime string
	LastModified string
}

type FunctionURLCors struct {
	AllowOrigins []string `json:"AllowOrigins,omitempty"`
	AllowMethods []string `json:"AllowMethods,omitempty"`
	AllowHeaders []string `json:"AllowHeaders,omitempty"`
	MaxAge       int      `json:"MaxAge,omitempty"`
}

// VersionStore manages versions, aliases, URLs, permissions, concurrency, and tags.
type VersionStore struct {
	mu          sync.RWMutex
	versions    map[string][]*FunctionVersion        // funcName -> versions
	nextVersion map[string]int                        // funcName -> next version number
	aliases     map[string]map[string]*FunctionAlias  // funcName -> aliasName -> alias
	urls        map[string]*FunctionURLConfig          // funcName -> URL config
	permissions map[string][]map[string]any            // funcName -> policy statements
	concurrency map[string]int                         // funcName -> reserved concurrency
	tags        map[string]map[string]string           // funcArn -> tags
	region      string
	acctID      string
}

func NewVersionStore(accountID, region string) *VersionStore {
	return &VersionStore{
		versions:    make(map[string][]*FunctionVersion),
		nextVersion: make(map[string]int),
		aliases:     make(map[string]map[string]*FunctionAlias),
		urls:        make(map[string]*FunctionURLConfig),
		permissions: make(map[string][]map[string]any),
		concurrency: make(map[string]int),
		tags:        make(map[string]map[string]string),
		region:      region,
		acctID:      accountID,
	}
}

// ── Versions ─────────────────────────────────────────────────────────────────

func (s *VersionStore) PublishVersion(f *Function, description string) *FunctionVersion {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextVersion[f.FunctionName]++
	ver := s.nextVersion[f.FunctionName]
	verStr := fmt.Sprintf("%d", ver)

	fv := &FunctionVersion{
		FunctionName: f.FunctionName,
		FunctionArn:  fmt.Sprintf("%s:%s", f.FunctionArn, verStr),
		Version:      verStr,
		Description:  description,
		Runtime:      f.Runtime,
		Handler:      f.Handler,
		Role:         f.Role,
		CodeSha256:   f.CodeSha256,
		CodeSize:     f.CodeSize,
		Timeout:      f.Timeout,
		MemorySize:   f.MemorySize,
		LastModified: time.Now().UTC().Format(time.RFC3339Nano),
	}
	s.versions[f.FunctionName] = append(s.versions[f.FunctionName], fv)
	return fv
}

func (s *VersionStore) ListVersions(funcName string) []*FunctionVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[funcName]
}

// ── Aliases ──────────────────────────────────────────────────────────────────

func (s *VersionStore) CreateAlias(funcName, funcArn, aliasName, funcVersion, description string) *FunctionAlias {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.aliases[funcName] == nil {
		s.aliases[funcName] = make(map[string]*FunctionAlias)
	}
	alias := &FunctionAlias{
		AliasArn:        fmt.Sprintf("%s:%s", funcArn, aliasName),
		Name:            aliasName,
		FunctionVersion: funcVersion,
		Description:     description,
	}
	s.aliases[funcName][aliasName] = alias
	return alias
}

func (s *VersionStore) GetAlias(funcName, aliasName string) (*FunctionAlias, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.aliases[funcName]; ok {
		a, ok := m[aliasName]
		return a, ok
	}
	return nil, false
}

func (s *VersionStore) UpdateAlias(funcName, aliasName, funcVersion, description string) (*FunctionAlias, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.aliases[funcName]; ok {
		if a, ok := m[aliasName]; ok {
			if funcVersion != "" {
				a.FunctionVersion = funcVersion
			}
			if description != "" {
				a.Description = description
			}
			return a, true
		}
	}
	return nil, false
}

func (s *VersionStore) DeleteAlias(funcName, aliasName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.aliases[funcName]; ok {
		if _, ok := m[aliasName]; ok {
			delete(m, aliasName)
			return true
		}
	}
	return false
}

func (s *VersionStore) ListAliases(funcName string) []*FunctionAlias {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*FunctionAlias
	for _, a := range s.aliases[funcName] {
		out = append(out, a)
	}
	return out
}

// ── Function URLs ────────────────────────────────────────────────────────────

func (s *VersionStore) CreateURL(funcName, funcArn, authType string, cors *FunctionURLCors) *FunctionURLConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	urlID := randomHexShort(12)
	cfg := &FunctionURLConfig{
		FunctionArn:  funcArn,
		FunctionUrl:  fmt.Sprintf("https://%s.lambda-url.%s.on.aws/", urlID, s.region),
		AuthType:     authType,
		Cors:         cors,
		CreationTime: now,
		LastModified: now,
	}
	s.urls[funcName] = cfg
	return cfg
}

func (s *VersionStore) GetURL(funcName string) (*FunctionURLConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.urls[funcName]
	return cfg, ok
}

func (s *VersionStore) DeleteURL(funcName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.urls[funcName]; ok {
		delete(s.urls, funcName)
		return true
	}
	return false
}

// ── Permissions ──────────────────────────────────────────────────────────────

func (s *VersionStore) AddPermission(funcName string, statement map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions[funcName] = append(s.permissions[funcName], statement)
}

func (s *VersionStore) GetPolicy(funcName string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.permissions[funcName]
}

func (s *VersionStore) RemovePermission(funcName, statementID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stmts := s.permissions[funcName]
	for i, stmt := range stmts {
		if sid, _ := stmt["Sid"].(string); sid == statementID {
			s.permissions[funcName] = append(stmts[:i], stmts[i+1:]...)
			return true
		}
	}
	return false
}

// ── Concurrency ──────────────────────────────────────────────────────────────

func (s *VersionStore) PutConcurrency(funcName string, reserved int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.concurrency[funcName] = reserved
}

func (s *VersionStore) GetConcurrency(funcName string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.concurrency[funcName]
	return c, ok
}

func (s *VersionStore) DeleteConcurrency(funcName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.concurrency, funcName)
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func (s *VersionStore) SetTags(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

func (s *VersionStore) GetTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tags[arn]
}

func (s *VersionStore) RemoveTags(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.tags[arn], k)
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handlePublishVersion(ctx *service.RequestContext, funcStore *FunctionStore, verStore *VersionStore, funcName string) (*service.Response, error) {
	f, ok := funcStore.Get(funcName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Function not found: "+funcName, http.StatusNotFound))
	}
	var req struct {
		Description string `json:"Description"`
	}
	_ = json.Unmarshal(ctx.Body, &req)
	fv := verStore.PublishVersion(f, req.Description)
	return jsonOK(fv)
}

func handleListVersionsByFunction(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	versions := verStore.ListVersions(funcName)
	return jsonOK(map[string]any{"Versions": versions})
}

func handleCreateAlias(ctx *service.RequestContext, funcStore *FunctionStore, verStore *VersionStore, funcName string) (*service.Response, error) {
	f, ok := funcStore.Get(funcName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Function not found: "+funcName, http.StatusNotFound))
	}
	var req struct {
		Name            string `json:"Name"`
		FunctionVersion string `json:"FunctionVersion"`
		Description     string `json:"Description"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil || req.Name == "" {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Name is required.", http.StatusBadRequest))
	}
	alias := verStore.CreateAlias(funcName, f.FunctionArn, req.Name, req.FunctionVersion, req.Description)
	return jsonOK(alias)
}

func handleGetAlias(_ *service.RequestContext, verStore *VersionStore, funcName, aliasName string) (*service.Response, error) {
	alias, ok := verStore.GetAlias(funcName, aliasName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Alias not found: "+aliasName, http.StatusNotFound))
	}
	return jsonOK(alias)
}

func handleUpdateAlias(ctx *service.RequestContext, verStore *VersionStore, funcName, aliasName string) (*service.Response, error) {
	var req struct {
		FunctionVersion string `json:"FunctionVersion"`
		Description     string `json:"Description"`
	}
	_ = json.Unmarshal(ctx.Body, &req)
	alias, ok := verStore.UpdateAlias(funcName, aliasName, req.FunctionVersion, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Alias not found: "+aliasName, http.StatusNotFound))
	}
	return jsonOK(alias)
}

func handleDeleteAlias(_ *service.RequestContext, verStore *VersionStore, funcName, aliasName string) (*service.Response, error) {
	if !verStore.DeleteAlias(funcName, aliasName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Alias not found: "+aliasName, http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleListAliases(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	aliases := verStore.ListAliases(funcName)
	return jsonOK(map[string]any{"Aliases": aliases})
}

func handleCreateFunctionUrlConfig(ctx *service.RequestContext, funcStore *FunctionStore, verStore *VersionStore, funcName string) (*service.Response, error) {
	f, ok := funcStore.Get(funcName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Function not found: "+funcName, http.StatusNotFound))
	}
	var req struct {
		AuthType string           `json:"AuthType"`
		Cors     *FunctionURLCors `json:"Cors"`
	}
	_ = json.Unmarshal(ctx.Body, &req)
	if req.AuthType == "" {
		req.AuthType = "NONE"
	}
	cfg := verStore.CreateURL(funcName, f.FunctionArn, req.AuthType, req.Cors)
	return jsonOK(cfg)
}

func handleGetFunctionUrlConfig(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	cfg, ok := verStore.GetURL(funcName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Function URL config not found for: "+funcName, http.StatusNotFound))
	}
	return jsonOK(cfg)
}

func handleDeleteFunctionUrlConfig(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	if !verStore.DeleteURL(funcName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Function URL config not found for: "+funcName, http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleAddPermission(ctx *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	var req map[string]any
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid request body.", http.StatusBadRequest))
	}
	verStore.AddPermission(funcName, req)
	return jsonOK(map[string]any{"Statement": string(ctx.Body)})
}

func handleGetPolicy(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	stmts := verStore.GetPolicy(funcName)
	if len(stmts) == 0 {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No policy found for: "+funcName, http.StatusNotFound))
	}
	policy := map[string]any{
		"Version":   "2012-10-17",
		"Statement": stmts,
	}
	policyJSON, _ := json.Marshal(policy)
	return jsonOK(map[string]any{"Policy": string(policyJSON)})
}

func handleRemovePermission(ctx *service.RequestContext, verStore *VersionStore, funcName, statementID string) (*service.Response, error) {
	if !verStore.RemovePermission(funcName, statementID) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Statement not found: "+statementID, http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handlePutFunctionConcurrency(ctx *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	var req struct {
		ReservedConcurrentExecutions int `json:"ReservedConcurrentExecutions"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid request body.", http.StatusBadRequest))
	}
	verStore.PutConcurrency(funcName, req.ReservedConcurrentExecutions)
	return jsonOK(map[string]any{"ReservedConcurrentExecutions": req.ReservedConcurrentExecutions})
}

func handleGetFunctionConcurrency(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	c, ok := verStore.GetConcurrency(funcName)
	if !ok {
		return jsonOK(map[string]any{})
	}
	return jsonOK(map[string]any{"ReservedConcurrentExecutions": c})
}

func handleDeleteFunctionConcurrency(_ *service.RequestContext, verStore *VersionStore, funcName string) (*service.Response, error) {
	verStore.DeleteConcurrency(funcName)
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleTagResource(ctx *service.RequestContext, verStore *VersionStore, arn string) (*service.Response, error) {
	var req struct {
		Tags map[string]string `json:"Tags"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid request body.", http.StatusBadRequest))
	}
	verStore.SetTags(arn, req.Tags)
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleListTags(_ *service.RequestContext, verStore *VersionStore, arn string) (*service.Response, error) {
	tags := verStore.GetTags(arn)
	if tags == nil {
		tags = make(map[string]string)
	}
	return jsonOK(map[string]any{"Tags": tags})
}

func handleUntagResource(ctx *service.RequestContext, verStore *VersionStore, arn string) (*service.Response, error) {
	// Tag keys come from query string
	r := ctx.RawRequest
	keys := r.URL.Query()["tagKeys"]
	verStore.RemoveTags(arn, keys)
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func randomHexShort(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
