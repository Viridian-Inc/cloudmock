package codeartifact

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStrSlice(m map[string]any, key string) []string {
	arr, ok := m[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseTagsList(tags []any) map[string]string {
	m := make(map[string]string)
	for _, t := range tags {
		if tm, ok := t.(map[string]any); ok {
			k := getStr(tm, "key")
			v := getStr(tm, "value")
			if k != "" {
				m[k] = v
			}
		}
	}
	return m
}

func tagsToList(m map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for k, v := range m {
		out = append(out, map[string]any{"key": k, "value": v})
	}
	return out
}

// getParam extracts a parameter from query params first, then body.
func getParam(ctx *service.RequestContext, m map[string]any, key string) string {
	if v := ctx.Params[key]; v != "" {
		return v
	}
	return getStr(m, key)
}

// ---- Domain handlers ----

func handleCreateDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getParam(ctx, req, "domain")
	encryptionKey := getStr(req, "encryptionKey")
	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	domain, awsErr := store.CreateDomain(name, encryptionKey, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"domain": domainToMap(domain)})
}

func handleDescribeDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getParam(ctx, req, "domain")
	domain, awsErr := store.DescribeDomain(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"domain": domainToMap(domain)})
}

func handleListDomains(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	domains := store.ListDomains()
	result := make([]map[string]any, len(domains))
	for i, d := range domains {
		result[i] = map[string]any{
			"name":          d.Name,
			"owner":         d.Owner,
			"arn":           d.ARN,
			"status":        d.Status,
			"encryptionKey": d.EncryptionKey,
			"createdTime":   float64(d.CreatedTime.Unix()),
		}
	}
	return jsonOK(map[string]any{"domains": result})
}

func handleDeleteDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getParam(ctx, req, "domain")
	domain, awsErr := store.DeleteDomain(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"domain": domainToMap(domain)})
}

// ---- Repository handlers ----

func handleCreateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	description := getStr(req, "description")

	var upstreams []UpstreamRepo
	if ups, ok := req["upstreams"].([]any); ok {
		for _, u := range ups {
			if um, ok := u.(map[string]any); ok {
				upstreams = append(upstreams, UpstreamRepo{
					RepositoryName: getStr(um, "repositoryName"),
				})
			}
		}
	}

	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	repo, awsErr := store.CreateRepository(domainName, repoName, description, upstreams, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repository": repoToMap(repo)})
}

func handleDescribeRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")

	repo, awsErr := store.DescribeRepository(domainName, repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repository": repoToMap(repo)})
}

func handleListRepositories(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repos := store.ListRepositories(domainName)

	result := make([]map[string]any, len(repos))
	for i, r := range repos {
		result[i] = map[string]any{
			"name":        r.Name,
			"arn":         r.ARN,
			"domainName":  r.DomainName,
			"domainOwner": r.DomainOwner,
			"description": r.Description,
		}
	}
	return jsonOK(map[string]any{"repositories": result})
}

func handleUpdateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	description := getStr(req, "description")

	var upstreams []UpstreamRepo
	if ups, ok := req["upstreams"].([]any); ok {
		for _, u := range ups {
			if um, ok := u.(map[string]any); ok {
				upstreams = append(upstreams, UpstreamRepo{
					RepositoryName: getStr(um, "repositoryName"),
				})
			}
		}
	}

	repo, awsErr := store.UpdateRepository(domainName, repoName, description, upstreams)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repository": repoToMap(repo)})
}

func handleDeleteRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")

	repo, awsErr := store.DeleteRepository(domainName, repoName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repository": repoToMap(repo)})
}

// ---- Package handlers ----

func handleDescribePackage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")
	namespace := getParam(ctx, req, "namespace")
	pkgName := getParam(ctx, req, "package")

	pkg, awsErr := store.DescribePackage(domainName, repoName, format, namespace, pkgName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"package": packageToMap(pkg)})
}

func handleListPackages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")
	namespace := getParam(ctx, req, "namespace")

	pkgs := store.ListPackages(domainName, repoName, format, namespace)
	result := make([]map[string]any, len(pkgs))
	for i, p := range pkgs {
		result[i] = packageToMap(p)
	}
	return jsonOK(map[string]any{"packages": result})
}

func handleListPackageVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")
	namespace := getParam(ctx, req, "namespace")
	pkgName := getParam(ctx, req, "package")

	versions, awsErr := store.ListPackageVersions(domainName, repoName, format, namespace, pkgName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	result := make([]map[string]any, len(versions))
	for i, v := range versions {
		result[i] = map[string]any{
			"version":  v.Version,
			"status":   v.Status,
			"revision": v.Revision,
		}
	}
	return jsonOK(map[string]any{
		"package":  pkgName,
		"format":   format,
		"versions": result,
	})
}

func handleDescribePackageVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")
	namespace := getParam(ctx, req, "namespace")
	pkgName := getParam(ctx, req, "package")
	version := getParam(ctx, req, "packageVersion")

	v, awsErr := store.DescribePackageVersion(domainName, repoName, format, namespace, pkgName, version)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"packageVersion": map[string]any{
			"version":       v.Version,
			"status":        v.Status,
			"revision":      v.Revision,
			"displayName":   v.DisplayName,
			"summary":       v.Summary,
			"homePage":      v.HomePage,
			"publishedTime": float64(v.PublishedTime.Unix()),
		},
	})
}

func handleGetPackageVersionReadme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")
	namespace := getParam(ctx, req, "namespace")
	pkgName := getParam(ctx, req, "package")
	version := getParam(ctx, req, "packageVersion")

	readme, resolvedVersion, awsErr := store.GetPackageVersionReadme(domainName, repoName, format, namespace, pkgName, version)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"format":         format,
		"namespace":      namespace,
		"package":        pkgName,
		"version":        resolvedVersion,
		"versionRevision": "",
		"readme":         readme,
	})
}

// ---- Domain Permissions Policy handlers ----

func handlePutDomainPermissionsPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	document := getStr(req, "policyDocument")

	policy, awsErr := store.PutDomainPermissionsPolicy(domainName, document)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policy": map[string]any{
			"document": policy.Document,
			"revision": policy.Revision,
		},
	})
}

func handleGetDomainPermissionsPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	policy, awsErr := store.GetDomainPermissionsPolicy(domainName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policy": map[string]any{
			"document": policy.Document,
			"revision": policy.Revision,
		},
	})
}

func handleDeleteDomainPermissionsPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	if awsErr := store.DeleteDomainPermissionsPolicy(domainName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Endpoint & Auth handlers ----

func handleGetRepositoryEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")
	repoName := getParam(ctx, req, "repository")
	format := getParam(ctx, req, "format")

	endpoint, awsErr := store.GetRepositoryEndpoint(domainName, repoName, format)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"repositoryEndpoint": endpoint})
}

func handleGetAuthorizationToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName := getParam(ctx, req, "domain")

	token, expiration, awsErr := store.GetAuthorizationToken(domainName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"authorizationToken": token,
		"expiration":         float64(expiration.Unix()),
	})
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}

	var tags map[string]string
	if tagList, ok := req["tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	keys := getStrSlice(req, "tagKeys")

	if awsErr := store.UntagResource(arn, keys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "resourceArn")
	tags := store.ListTagsForResource(arn)
	return jsonOK(map[string]any{"tags": tagsToList(tags)})
}

// ---- conversion helpers ----

func domainToMap(d *Domain) map[string]any {
	return map[string]any{
		"name":            d.Name,
		"owner":           d.Owner,
		"arn":             d.ARN,
		"status":          d.Status,
		"encryptionKey":   d.EncryptionKey,
		"repositoryCount": d.RepositoryCount,
		"assetSizeBytes":  d.AssetSizeBytes,
		"createdTime":     float64(d.CreatedTime.Unix()),
	}
}

func repoToMap(r *Repository) map[string]any {
	upstreams := make([]map[string]any, len(r.Upstreams))
	for i, u := range r.Upstreams {
		upstreams[i] = map[string]any{"repositoryName": u.RepositoryName}
	}

	extConns := make([]map[string]any, len(r.ExternalConnections))
	for i, ec := range r.ExternalConnections {
		extConns[i] = map[string]any{
			"externalConnectionName": ec.ExternalConnectionName,
			"packageFormat":          ec.PackageFormat,
			"status":                 ec.Status,
		}
	}

	return map[string]any{
		"name":                r.Name,
		"arn":                 r.ARN,
		"domainName":          r.DomainName,
		"domainOwner":         r.DomainOwner,
		"description":         r.Description,
		"upstreams":           upstreams,
		"externalConnections": extConns,
		"createdTime":         float64(r.CreatedTime.Unix()),
	}
}

func packageToMap(p *Package) map[string]any {
	m := map[string]any{
		"format":  p.Format,
		"package": p.PackageName,
	}
	if p.Namespace != "" {
		m["namespace"] = p.Namespace
	}
	if p.OriginConfig != nil && p.OriginConfig.Restrictions != nil {
		m["originConfiguration"] = map[string]any{
			"restrictions": map[string]any{
				"publish":  p.OriginConfig.Restrictions.Publish,
				"upstream": p.OriginConfig.Restrictions.Upstream,
			},
		}
	}
	return m
}
