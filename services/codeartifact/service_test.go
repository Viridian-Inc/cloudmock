package codeartifact_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/codeartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.CodeArtifactService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     make(map[string]string),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func createDomain(t *testing.T, s *svc.CodeArtifactService, name string) map[string]any {
	t.Helper()
	ctx := jsonCtx("CreateDomain", map[string]any{"domain": name})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return respBody(t, resp)
}

func createRepo(t *testing.T, s *svc.CodeArtifactService, domain, repo string) map[string]any {
	t.Helper()
	ctx := jsonCtx("CreateRepository", map[string]any{
		"domain":      domain,
		"repository":  repo,
		"description": "test repo",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	return respBody(t, resp)
}

// --- Domain Tests ---

func TestCreateDomain(t *testing.T) {
	s := newService()
	body := createDomain(t, s, "my-domain")
	domain := body["domain"].(map[string]any)
	assert.Equal(t, "my-domain", domain["name"])
	assert.Equal(t, "Active", domain["status"])
	assert.Equal(t, "123456789012", domain["owner"])
	assert.Contains(t, domain["arn"], "domain/my-domain")
	assert.NotEmpty(t, domain["encryptionKey"])
}

func TestCreateDomainDuplicate(t *testing.T) {
	s := newService()
	createDomain(t, s, "dup-domain")
	ctx := jsonCtx("CreateDomain", map[string]any{"domain": "dup-domain"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ConflictException")
}

func TestCreateDomainMissingName(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateDomain", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestDescribeDomain(t *testing.T) {
	s := newService()
	createDomain(t, s, "desc-domain")

	ctx := jsonCtx("DescribeDomain", map[string]any{"domain": "desc-domain"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "desc-domain", body["domain"].(map[string]any)["name"])
}

func TestDescribeDomainNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DescribeDomain", map[string]any{"domain": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestListDomains(t *testing.T) {
	s := newService()
	createDomain(t, s, "dom-1")
	createDomain(t, s, "dom-2")

	resp, err := s.HandleRequest(jsonCtx("ListDomains", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	domains := body["domains"].([]any)
	assert.Len(t, domains, 2)
}

func TestDeleteDomain(t *testing.T) {
	s := newService()
	createDomain(t, s, "del-domain")

	ctx := jsonCtx("DeleteDomain", map[string]any{"domain": "del-domain"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "del-domain", body["domain"].(map[string]any)["name"])

	// Verify gone
	resp2, _ := s.HandleRequest(jsonCtx("ListDomains", map[string]any{}))
	body2 := respBody(t, resp2)
	assert.Len(t, body2["domains"].([]any), 0)
}

func TestDeleteDomainNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("DeleteDomain", map[string]any{"domain": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestDeleteDomainWithRepositories(t *testing.T) {
	s := newService()
	createDomain(t, s, "busy-domain")
	createRepo(t, s, "busy-domain", "repo-1")

	ctx := jsonCtx("DeleteDomain", map[string]any{"domain": "busy-domain"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ConflictException")
}

// --- Repository Tests ---

func TestCreateRepository(t *testing.T) {
	s := newService()
	createDomain(t, s, "repo-domain")
	body := createRepo(t, s, "repo-domain", "my-repo")
	repo := body["repository"].(map[string]any)
	assert.Equal(t, "my-repo", repo["name"])
	assert.Equal(t, "repo-domain", repo["domainName"])
	assert.Equal(t, "123456789012", repo["domainOwner"])
	assert.Contains(t, repo["arn"], "repository/repo-domain/my-repo")
}

func TestCreateRepositoryDuplicate(t *testing.T) {
	s := newService()
	createDomain(t, s, "duprel-domain")
	createRepo(t, s, "duprel-domain", "dup-repo")
	ctx := jsonCtx("CreateRepository", map[string]any{
		"domain":     "duprel-domain",
		"repository": "dup-repo",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ConflictException")
}

func TestCreateRepositoryDomainNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("CreateRepository", map[string]any{
		"domain":     "nope",
		"repository": "repo",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestCreateRepositoryMissingName(t *testing.T) {
	s := newService()
	createDomain(t, s, "noname-domain")
	ctx := jsonCtx("CreateRepository", map[string]any{
		"domain": "noname-domain",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestDescribeRepository(t *testing.T) {
	s := newService()
	createDomain(t, s, "descr-domain")
	createRepo(t, s, "descr-domain", "descr-repo")

	ctx := jsonCtx("DescribeRepository", map[string]any{
		"domain":     "descr-domain",
		"repository": "descr-repo",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "descr-repo", body["repository"].(map[string]any)["name"])
}

func TestDescribeRepositoryNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "norepo-domain")
	ctx := jsonCtx("DescribeRepository", map[string]any{
		"domain":     "norepo-domain",
		"repository": "nope",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestListRepositories(t *testing.T) {
	s := newService()
	createDomain(t, s, "listrepo-domain")
	createRepo(t, s, "listrepo-domain", "repo-a")
	createRepo(t, s, "listrepo-domain", "repo-b")

	ctx := jsonCtx("ListRepositories", map[string]any{"domain": "listrepo-domain"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	repos := body["repositories"].([]any)
	assert.Len(t, repos, 2)
}

func TestDeleteRepository(t *testing.T) {
	s := newService()
	createDomain(t, s, "delrepo-domain")
	createRepo(t, s, "delrepo-domain", "del-repo")

	ctx := jsonCtx("DeleteRepository", map[string]any{
		"domain":     "delrepo-domain",
		"repository": "del-repo",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "del-repo", body["repository"].(map[string]any)["name"])

	// Verify domain repo count decremented
	resp2, _ := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"domain": "delrepo-domain"}))
	body2 := respBody(t, resp2)
	assert.Equal(t, float64(0), body2["domain"].(map[string]any)["repositoryCount"])
}

func TestDeleteRepositoryNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "delrepo2-domain")
	ctx := jsonCtx("DeleteRepository", map[string]any{
		"domain":     "delrepo2-domain",
		"repository": "nope",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestCreateRepositoryWithUpstreams(t *testing.T) {
	s := newService()
	createDomain(t, s, "ups-domain")
	createRepo(t, s, "ups-domain", "upstream-repo")

	ctx := jsonCtx("CreateRepository", map[string]any{
		"domain":     "ups-domain",
		"repository": "downstream-repo",
		"upstreams":  []any{map[string]any{"repositoryName": "upstream-repo"}},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	repo := body["repository"].(map[string]any)
	upstreams := repo["upstreams"].([]any)
	assert.Len(t, upstreams, 1)
	assert.Equal(t, "upstream-repo", upstreams[0].(map[string]any)["repositoryName"])
}

func TestDomainRepoCount(t *testing.T) {
	s := newService()
	createDomain(t, s, "count-domain")
	createRepo(t, s, "count-domain", "repo-1")
	createRepo(t, s, "count-domain", "repo-2")

	resp, _ := s.HandleRequest(jsonCtx("DescribeDomain", map[string]any{"domain": "count-domain"}))
	body := respBody(t, resp)
	assert.Equal(t, float64(2), body["domain"].(map[string]any)["repositoryCount"])
}

// --- Endpoint & Auth ---

func TestGetRepositoryEndpoint(t *testing.T) {
	s := newService()
	createDomain(t, s, "ep-domain")
	createRepo(t, s, "ep-domain", "ep-repo")

	ctx := jsonCtx("GetRepositoryEndpoint", map[string]any{
		"domain":     "ep-domain",
		"repository": "ep-repo",
		"format":     "npm",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	endpoint := body["repositoryEndpoint"].(string)
	assert.Contains(t, endpoint, "ep-domain")
	assert.Contains(t, endpoint, "npm")
	assert.Contains(t, endpoint, "ep-repo")
	// Validate realistic URL format: https://{domain}-{account}.d.codeartifact.{region}.amazonaws.com/{format}/{repo}/
	assert.Contains(t, endpoint, "https://ep-domain-123456789012.d.codeartifact.us-east-1.amazonaws.com/npm/ep-repo/")
}

func TestGetRepositoryEndpointMaven(t *testing.T) {
	s := newService()
	createDomain(t, s, "mvn-domain")
	createRepo(t, s, "mvn-domain", "mvn-repo")

	ctx := jsonCtx("GetRepositoryEndpoint", map[string]any{
		"domain":     "mvn-domain",
		"repository": "mvn-repo",
		"format":     "maven",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	endpoint := body["repositoryEndpoint"].(string)
	assert.Contains(t, endpoint, "maven/mvn-repo/")
}

func TestGetRepositoryEndpointNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetRepositoryEndpoint", map[string]any{
		"domain":     "nope",
		"repository": "nope",
		"format":     "npm",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestGetAuthorizationToken(t *testing.T) {
	s := newService()
	createDomain(t, s, "auth-domain")

	ctx := jsonCtx("GetAuthorizationToken", map[string]any{"domain": "auth-domain"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.NotEmpty(t, body["authorizationToken"])
	assert.NotZero(t, body["expiration"])
}

func TestGetAuthorizationTokenDomainNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("GetAuthorizationToken", map[string]any{"domain": "nope"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

// --- Package Tests (using EnsurePackage) ---

func TestDescribePackageNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "pkg-domain")
	createRepo(t, s, "pkg-domain", "pkg-repo")

	ctx := jsonCtx("DescribePackage", map[string]any{
		"domain":     "pkg-domain",
		"repository": "pkg-repo",
		"format":     "npm",
		"package":    "nonexistent",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestListPackageVersionsNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "pkgv-domain")
	createRepo(t, s, "pkgv-domain", "pkgv-repo")

	ctx := jsonCtx("ListPackageVersions", map[string]any{
		"domain":     "pkgv-domain",
		"repository": "pkgv-repo",
		"format":     "npm",
		"package":    "nonexistent",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestDescribePackageVersionNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "pkgvd-domain")
	createRepo(t, s, "pkgvd-domain", "pkgvd-repo")

	ctx := jsonCtx("DescribePackageVersion", map[string]any{
		"domain":         "pkgvd-domain",
		"repository":     "pkgvd-repo",
		"format":         "npm",
		"package":        "nonexistent",
		"packageVersion": "1.0.0",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

// --- Tags ---

func TestTagging(t *testing.T) {
	s := newService()
	body := createDomain(t, s, "tag-domain")
	arn := body["domain"].(map[string]any)["arn"].(string)

	// Tag
	ctx := jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        []any{map[string]any{"key": "env", "value": "prod"}},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List tags
	ctx2 := jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	tags := body2["tags"].([]any)
	assert.Len(t, tags, 1)

	// Untag
	ctx3 := jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	})
	resp3, err3 := s.HandleRequest(ctx3)
	require.NoError(t, err3)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	// Verify removed
	resp4, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	body4 := respBody(t, resp4)
	assert.Len(t, body4["tags"].([]any), 0)
}

func TestTagResourceMissingArn(t *testing.T) {
	s := newService()
	ctx := jsonCtx("TagResource", map[string]any{
		"tags": []any{map[string]any{"key": "env", "value": "prod"}},
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

// --- UpdateRepository ---

func TestUpdateRepository(t *testing.T) {
	s := newService()
	createDomain(t, s, "upd-domain")
	createRepo(t, s, "upd-domain", "upd-repo")

	ctx := jsonCtx("UpdateRepository", map[string]any{
		"domain":      "upd-domain",
		"repository":  "upd-repo",
		"description": "Updated description",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	repo := body["repository"].(map[string]any)
	assert.Equal(t, "Updated description", repo["description"])
}

func TestUpdateRepositoryWithUpstreams(t *testing.T) {
	s := newService()
	createDomain(t, s, "updups-domain")
	createRepo(t, s, "updups-domain", "upstream-repo")
	createRepo(t, s, "updups-domain", "downstream-repo")

	ctx := jsonCtx("UpdateRepository", map[string]any{
		"domain":     "updups-domain",
		"repository": "downstream-repo",
		"upstreams":  []any{map[string]any{"repositoryName": "upstream-repo"}},
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	repo := body["repository"].(map[string]any)
	upstreams := repo["upstreams"].([]any)
	assert.Len(t, upstreams, 1)
}

func TestUpdateRepositoryNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "updnf-domain")
	ctx := jsonCtx("UpdateRepository", map[string]any{
		"domain":     "updnf-domain",
		"repository": "nope",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

// --- Domain Permissions Policy ---

func TestPutAndGetDomainPermissionsPolicy(t *testing.T) {
	s := newService()
	createDomain(t, s, "policy-domain")

	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"codeartifact:*"}]}`
	ctx := jsonCtx("PutDomainPermissionsPolicy", map[string]any{
		"domain":         "policy-domain",
		"policyDocument": doc,
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	policy := body["policy"].(map[string]any)
	assert.Equal(t, doc, policy["document"])
	assert.NotEmpty(t, policy["revision"])

	ctx2 := jsonCtx("GetDomainPermissionsPolicy", map[string]any{"domain": "policy-domain"})
	resp2, err2 := s.HandleRequest(ctx2)
	require.NoError(t, err2)
	body2 := respBody(t, resp2)
	assert.Equal(t, doc, body2["policy"].(map[string]any)["document"])
}

func TestGetDomainPermissionsPolicyNotSet(t *testing.T) {
	s := newService()
	createDomain(t, s, "nopolicy-domain")
	ctx := jsonCtx("GetDomainPermissionsPolicy", map[string]any{"domain": "nopolicy-domain"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestDeleteDomainPermissionsPolicy(t *testing.T) {
	s := newService()
	createDomain(t, s, "delpolicy-domain")

	s.HandleRequest(jsonCtx("PutDomainPermissionsPolicy", map[string]any{
		"domain":         "delpolicy-domain",
		"policyDocument": `{"Version":"2012-10-17"}`,
	}))

	ctx := jsonCtx("DeleteDomainPermissionsPolicy", map[string]any{"domain": "delpolicy-domain"})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	_, err2 := s.HandleRequest(jsonCtx("GetDomainPermissionsPolicy", map[string]any{"domain": "delpolicy-domain"}))
	require.Error(t, err2)
}

func TestPutDomainPermissionsPolicyDomainNotFound(t *testing.T) {
	s := newService()
	ctx := jsonCtx("PutDomainPermissionsPolicy", map[string]any{
		"domain":         "nope",
		"policyDocument": `{}`,
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestPutDomainPermissionsPolicyMissingDocument(t *testing.T) {
	s := newService()
	createDomain(t, s, "nodoc-domain")
	ctx := jsonCtx("PutDomainPermissionsPolicy", map[string]any{"domain": "nodoc-domain"})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

// --- GetPackageVersionReadme ---

func TestGetPackageVersionReadme(t *testing.T) {
	s := newService()
	createDomain(t, s, "readme-domain")
	createRepo(t, s, "readme-domain", "readme-repo")

	// Ensure a package version exists
	store := s.Store()
	store.EnsurePackage("readme-domain", "readme-repo", "npm", "", "my-pkg", "1.0.0")

	ctx := jsonCtx("GetPackageVersionReadme", map[string]any{
		"domain":         "readme-domain",
		"repository":     "readme-repo",
		"format":         "npm",
		"package":        "my-pkg",
		"packageVersion": "1.0.0",
	})
	resp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Contains(t, body["readme"].(string), "my-pkg")
	assert.Equal(t, "1.0.0", body["version"])
}

func TestGetPackageVersionReadmeNotFound(t *testing.T) {
	s := newService()
	createDomain(t, s, "readme2-domain")
	createRepo(t, s, "readme2-domain", "readme2-repo")

	ctx := jsonCtx("GetPackageVersionReadme", map[string]any{
		"domain":         "readme2-domain",
		"repository":     "readme2-repo",
		"format":         "npm",
		"package":        "nonexistent",
		"packageVersion": "1.0.0",
	})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

// --- Invalid Action ---

func TestInvalidAction(t *testing.T) {
	s := newService()
	ctx := jsonCtx("BogusAction", map[string]any{})
	_, err := s.HandleRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

// --- Service Metadata ---

func TestServiceName(t *testing.T) {
	s := newService()
	assert.Equal(t, "codeartifact", s.Name())
}

func TestHealthCheck(t *testing.T) {
	s := newService()
	assert.NoError(t, s.HealthCheck())
}
