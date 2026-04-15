package ecrpublic

import (
	"encoding/base64"
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		v := getStr(t, "Value")
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Repository handlers ─────────────────────────────────────────────────────

func handleCreateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	if name == "" {
		return jsonErr(service.ErrValidation("repositoryName is required."))
	}
	repo, err := store.CreateRepository(name, getMap(req, "catalogData"), parseTagList(req, "tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"repository":  repositoryProps(repo),
		"catalogData": repo.CatalogData,
	})
}

func handleDeleteRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	if name == "" {
		return jsonErr(service.ErrValidation("repositoryName is required."))
	}
	repo, err := store.DeleteRepository(name, getBool(req, "force"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"repository": repositoryProps(repo)})
}

func handleDescribeRepositories(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	names := getStrList(req, "repositoryNames")
	var repos []*StoredRepository
	if len(names) > 0 {
		for _, n := range names {
			r, err := store.GetRepository(n)
			if err != nil {
				return jsonErr(err)
			}
			repos = append(repos, r)
		}
	} else {
		repos = store.ListRepositories()
	}
	out := make([]map[string]any, 0, len(repos))
	for _, r := range repos {
		out = append(out, repositoryProps(r))
	}
	return jsonOK(map[string]any{"repositories": out})
}

func handleDescribeRegistries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"registries": []map[string]any{
			{
				"registryId":          store.accountID,
				"registryArn":         "arn:aws:ecr-public::" + store.accountID + ":registry/" + store.accountID,
				"registryUri":         "public.ecr.aws/" + store.accountID,
				"verified":            false,
				"aliases":             []any{},
				"registryCatalogData": store.GetRegistryCatalog(),
			},
		},
	})
}

func handleGetAuthorizationToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	token := base64.StdEncoding.EncodeToString([]byte("AWS:cloudmock"))
	return jsonOK(map[string]any{
		"authorizationData": map[string]any{
			"authorizationToken": token,
			"expiresAt":          rfc3339(time.Now().Add(12 * time.Hour).UTC()),
		},
	})
}

func handleGetRegistryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"registryCatalogData": store.GetRegistryCatalog()})
}

func handleGetRepositoryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	repo, err := store.GetRepository(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"catalogData": repo.CatalogData})
}

func handleGetRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	repo, err := store.GetRepository(name)
	if err != nil {
		return jsonErr(err)
	}
	if repo.Policy == "" {
		return jsonErr(service.NewAWSError("RepositoryPolicyNotFoundException",
			"No policy set", 404))
	}
	return jsonOK(map[string]any{
		"repositoryName": repo.Name,
		"registryId":     repo.RegistryID,
		"policyText":     repo.Policy,
	})
}

func handlePutRegistryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	data := getMap(req, "displayName")
	display := getStr(req, "displayName")
	if display != "" {
		data = map[string]any{"displayName": display}
	}
	store.PutRegistryCatalog(data)
	return jsonOK(map[string]any{"registryCatalogData": store.GetRegistryCatalog()})
}

func handlePutRepositoryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	catalog := getMap(req, "catalogData")
	if catalog == nil {
		catalog = map[string]any{}
	}
	repo, err := store.SetRepositoryCatalog(name, catalog)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"catalogData": repo.CatalogData})
}

func handleSetRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	policy := getStr(req, "policyText")
	if policy == "" {
		return jsonErr(service.ErrValidation("policyText is required."))
	}
	repo, err := store.SetRepositoryPolicy(name, policy)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"repositoryName": repo.Name,
		"registryId":     repo.RegistryID,
		"policyText":     repo.Policy,
	})
}

func handleDeleteRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	repo, err := store.DeleteRepositoryPolicy(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"repositoryName": repo.Name,
		"registryId":     repo.RegistryID,
		"policyText":     "",
	})
}

func repositoryProps(r *StoredRepository) map[string]any {
	return map[string]any{
		"repositoryArn":  r.Arn,
		"registryId":     r.RegistryID,
		"repositoryName": r.Name,
		"repositoryUri":  r.URI,
		"createdAt":      rfc3339(r.CreatedAt),
	}
}

// ── Image handlers ──────────────────────────────────────────────────────────

func handlePutImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	manifest := getStr(req, "imageManifest")
	if manifest == "" {
		return jsonErr(service.ErrValidation("imageManifest is required."))
	}
	mediaType := getStr(req, "imageManifestMediaType")
	if mediaType == "" {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}
	tag := getStr(req, "imageTag")
	img, err := store.PutImage(name, manifest, mediaType, tag)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"image": map[string]any{
			"registryId":             store.accountID,
			"repositoryName":         name,
			"imageId":                imageID(img, tag),
			"imageManifest":          img.Manifest,
			"imageManifestMediaType": img.MediaType,
		},
	})
}

func handleBatchDeleteImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	ids := make([]map[string]string, 0)
	for _, item := range getMapList(req, "imageIds") {
		ids = append(ids, map[string]string{
			"imageDigest": getStr(item, "imageDigest"),
			"imageTag":    getStr(item, "imageTag"),
		})
	}
	failures := store.BatchDeleteImage(name, ids)
	return jsonOK(map[string]any{
		"imageIds":          []any{},
		"failures":          failures,
	})
}

func handleDescribeImages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	images := store.ListImages(name)
	out := make([]map[string]any, 0, len(images))
	for _, img := range images {
		out = append(out, map[string]any{
			"registryId":       store.accountID,
			"repositoryName":   name,
			"imageDigest":      img.Digest,
			"imageTags":        img.Tags,
			"imageSizeInBytes": img.SizeInBytes,
			"imagePushedAt":    rfc3339(img.PushedAt),
			"imageManifestMediaType": img.MediaType,
		})
	}
	return jsonOK(map[string]any{"imageDetails": out})
}

func handleDescribeImageTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	images := store.ListImages(name)
	out := make([]map[string]any, 0)
	for _, img := range images {
		for _, t := range img.Tags {
			out = append(out, map[string]any{
				"imageTag": t,
				"imageDetail": map[string]any{
					"imageDigest":      img.Digest,
					"imageSizeInBytes": img.SizeInBytes,
					"imagePushedAt":    rfc3339(img.PushedAt),
					"imageManifestMediaType": img.MediaType,
				},
			})
		}
	}
	return jsonOK(map[string]any{"imageTagDetails": out})
}

func imageID(img *StoredImage, tag string) map[string]any {
	out := map[string]any{"imageDigest": img.Digest}
	if tag != "" {
		out["imageTag"] = tag
	} else if len(img.Tags) > 0 {
		out["imageTag"] = img.Tags[0]
	}
	return out
}

// ── Layer upload handlers ───────────────────────────────────────────────────

func handleBatchCheckLayerAvailability(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	digests := getStrList(req, "layerDigests")
	return jsonOK(map[string]any{
		"layers":   store.CheckLayerAvailability(digests),
		"failures": []any{},
	})
}

func handleInitiateLayerUpload(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "repositoryName")
	upload, err := store.InitiateUpload(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"uploadId": upload.UploadID,
		"partSize": upload.PartSize,
	})
}

func handleUploadLayerPart(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	uploadID := getStr(req, "uploadId")
	b64 := getStr(req, "layerPartBlob")
	data, _ := base64.StdEncoding.DecodeString(b64)
	u, err := store.UploadPart(uploadID, data)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"registryId":     store.accountID,
		"repositoryName": u.RepositoryName,
		"uploadId":       u.UploadID,
		"lastByteReceived": int64(len(data)),
	})
}

func handleCompleteLayerUpload(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	uploadID := getStr(req, "uploadId")
	digest, err := store.CompleteUpload(uploadID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"registryId":     store.accountID,
		"repositoryName": getStr(req, "repositoryName"),
		"uploadId":       uploadID,
		"layerDigest":    digest,
	})
}

// ── Tags ────────────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	store.TagResource(arn, parseTagList(req, "tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	store.UntagResource(arn, getStrList(req, "tagKeys"))
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := store.ListTags(arn)
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return jsonOK(map[string]any{"tags": out})
}
