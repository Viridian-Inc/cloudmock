package ecr

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type tagEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type imageScanningConfigurationJSON struct {
	ScanOnPush bool `json:"scanOnPush"`
}

type createRepositoryRequest struct {
	RepositoryName             string                          `json:"repositoryName"`
	Tags                       []tagEntry                      `json:"tags"`
	ImageTagMutability         string                          `json:"imageTagMutability"`
	ImageScanningConfiguration *imageScanningConfigurationJSON `json:"imageScanningConfiguration"`
}

type repositoryJSON struct {
	RepositoryArn              string                          `json:"repositoryArn"`
	RegistryId                 string                          `json:"registryId"`
	RepositoryName             string                          `json:"repositoryName"`
	RepositoryUri              string                          `json:"repositoryUri"`
	CreatedAt                  float64                         `json:"createdAt"`
	ImageTagMutability         string                          `json:"imageTagMutability"`
	ImageScanningConfiguration *imageScanningConfigurationJSON `json:"imageScanningConfiguration"`
}

type createRepositoryResponse struct {
	Repository repositoryJSON `json:"repository"`
}

type deleteRepositoryRequest struct {
	RepositoryName string `json:"repositoryName"`
	Force          bool   `json:"force"`
}

type deleteRepositoryResponse struct {
	Repository repositoryJSON `json:"repository"`
}

type describeRepositoriesRequest struct {
	RepositoryNames []string `json:"repositoryNames"`
}

type describeRepositoriesResponse struct {
	Repositories []repositoryJSON `json:"repositories"`
	NextToken    string           `json:"nextToken,omitempty"`
}

type imageIDJSON struct {
	ImageDigest string `json:"imageDigest"`
	ImageTag    string `json:"imageTag"`
}

type listImagesRequest struct {
	RepositoryName string `json:"repositoryName"`
}

type listImagesResponse struct {
	ImageIds  []imageIDJSON `json:"imageIds"`
	NextToken string        `json:"nextToken,omitempty"`
}

type batchGetImageRequest struct {
	RepositoryName string        `json:"repositoryName"`
	ImageIds       []imageIDJSON `json:"imageIds"`
}

type imageJSON struct {
	RegistryId     string      `json:"registryId"`
	RepositoryName string      `json:"repositoryName"`
	ImageId        imageIDJSON `json:"imageId"`
	ImageManifest  string      `json:"imageManifest"`
}

type imageFailureJSON struct {
	ImageId       imageIDJSON `json:"imageId"`
	FailureCode   string      `json:"failureCode"`
	FailureReason string      `json:"failureReason"`
}

type batchGetImageResponse struct {
	Images  []imageJSON        `json:"images"`
	Failures []imageFailureJSON `json:"failures"`
}

type putImageRequest struct {
	RepositoryName string `json:"repositoryName"`
	ImageManifest  string `json:"imageManifest"`
	ImageTag       string `json:"imageTag"`
}

type putImageResponse struct {
	Image imageJSON `json:"image"`
}

type batchDeleteImageRequest struct {
	RepositoryName string        `json:"repositoryName"`
	ImageIds       []imageIDJSON `json:"imageIds"`
}

type batchDeleteImageResponse struct {
	ImageIds []imageIDJSON      `json:"imageIds"`
	Failures []imageFailureJSON `json:"failures"`
}

type authorizationData struct {
	AuthorizationToken string  `json:"authorizationToken"`
	ExpiresAt          float64 `json:"expiresAt"`
	ProxyEndpoint      string  `json:"proxyEndpoint"`
}

type getAuthorizationTokenResponse struct {
	AuthorizationData []authorizationData `json:"authorizationData"`
}

type describeImageScanFindingsRequest struct {
	RepositoryName string      `json:"repositoryName"`
	ImageId        imageIDJSON `json:"imageId"`
}

type describeImageScanFindingsResponse struct {
	RegistryId     string      `json:"registryId"`
	RepositoryName string      `json:"repositoryName"`
	ImageId        imageIDJSON `json:"imageId"`
	ImageScanStatus struct {
		Status      string `json:"status"`
		Description string `json:"description"`
	} `json:"imageScanStatus"`
	ImageScanFindings struct {
		Findings           []any `json:"findings"`
		FindingSeverityCounts map[string]int `json:"findingSeverityCounts"`
	} `json:"imageScanFindings"`
}

type tagResourceRequest struct {
	ResourceArn string     `json:"resourceArn"`
	Tags        []tagEntry `json:"tags"`
}

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

type listTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsForResourceResponse struct {
	Tags []tagEntry `json:"tags"`
}

// ---- helpers ----

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
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func repoToJSON(r *Repository) repositoryJSON {
	scanCfg := &imageScanningConfigurationJSON{
		ScanOnPush: r.ImageScanningConfig.ScanOnPush,
	}
	return repositoryJSON{
		RepositoryArn:              r.ARN,
		RegistryId:                 r.RegistryId,
		RepositoryName:             r.Name,
		RepositoryUri:              r.URI,
		CreatedAt:                  float64(r.CreatedAt.Unix()),
		ImageTagMutability:         r.ImageTagMutability,
		ImageScanningConfiguration: scanCfg,
	}
}

func tagsFromList(entries []tagEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, t := range entries {
		m[t.Key] = t.Value
	}
	return m
}

func tagsToList(m map[string]string) []tagEntry {
	out := make([]tagEntry, 0, len(m))
	for k, v := range m {
		out = append(out, tagEntry{Key: k, Value: v})
	}
	return out
}

func refsFromJSON(ids []imageIDJSON) []imageIDRef {
	refs := make([]imageIDRef, len(ids))
	for i, id := range ids {
		refs[i] = imageIDRef{Digest: id.ImageDigest, Tag: id.ImageTag}
	}
	return refs
}

func failuresToJSON(failures []imageFailure) []imageFailureJSON {
	out := make([]imageFailureJSON, len(failures))
	for i, f := range failures {
		out[i] = imageFailureJSON{
			ImageId:       imageIDJSON{ImageDigest: f.ImageID.Digest, ImageTag: f.ImageID.Tag},
			FailureCode:   f.FailureCode,
			FailureReason: f.FailureReason,
		}
	}
	return out
}

// ---- handlers ----

func handleCreateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createRepositoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}

	scanOnPush := false
	if req.ImageScanningConfiguration != nil {
		scanOnPush = req.ImageScanningConfiguration.ScanOnPush
	}

	repo, awsErr := store.CreateRepository(req.RepositoryName, req.ImageTagMutability, scanOnPush, tagsFromList(req.Tags))
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(createRepositoryResponse{Repository: repoToJSON(repo)})
}

func handleDeleteRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteRepositoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}

	repo, awsErr := store.DeleteRepository(req.RepositoryName, req.Force)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(deleteRepositoryResponse{Repository: repoToJSON(repo)})
}

func handleDescribeRepositories(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeRepositoriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	repos, awsErr := store.ListRepositories(req.RepositoryNames)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	out := make([]repositoryJSON, len(repos))
	for i, r := range repos {
		out[i] = repoToJSON(r)
	}

	return jsonOK(describeRepositoriesResponse{Repositories: out})
}

func handleListImages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listImagesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}

	images, awsErr := store.ListImages(req.RepositoryName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	ids := make([]imageIDJSON, 0, len(images))
	for _, img := range images {
		ids = append(ids, imageIDJSON{ImageDigest: img.Digest, ImageTag: img.Tag})
	}

	return jsonOK(listImagesResponse{ImageIds: ids})
}

func handleBatchGetImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchGetImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}

	refs := refsFromJSON(req.ImageIds)
	images, failures := store.BatchGetImage(req.RepositoryName, refs)

	out := make([]imageJSON, 0, len(images))
	for _, img := range images {
		out = append(out, imageJSON{
			RegistryId:     store.accountID,
			RepositoryName: req.RepositoryName,
			ImageId:        imageIDJSON{ImageDigest: img.Digest, ImageTag: img.Tag},
			ImageManifest:  img.Manifest,
		})
	}

	return jsonOK(batchGetImageResponse{
		Images:   out,
		Failures: failuresToJSON(failures),
	})
}

func handlePutImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}
	if req.ImageManifest == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"imageManifest is required.", http.StatusBadRequest))
	}

	img, awsErr := store.PutImage(req.RepositoryName, req.ImageManifest, req.ImageTag)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(putImageResponse{
		Image: imageJSON{
			RegistryId:     store.accountID,
			RepositoryName: req.RepositoryName,
			ImageId:        imageIDJSON{ImageDigest: img.Digest, ImageTag: img.Tag},
			ImageManifest:  img.Manifest,
		},
	})
}

func handleBatchDeleteImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchDeleteImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RepositoryName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"repositoryName is required.", http.StatusBadRequest))
	}

	refs := refsFromJSON(req.ImageIds)
	deleted, failures := store.BatchDeleteImage(req.RepositoryName, refs)

	ids := make([]imageIDJSON, 0, len(deleted))
	for _, img := range deleted {
		ids = append(ids, imageIDJSON{ImageDigest: img.Digest, ImageTag: img.Tag})
	}

	return jsonOK(batchDeleteImageResponse{
		ImageIds: ids,
		Failures: failuresToJSON(failures),
	})
}

func handleGetAuthorizationToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	// Return a single authorization token for the registry.
	token := base64.StdEncoding.EncodeToString([]byte("AWS:password"))
	expiresAt := time.Now().UTC().Add(12 * time.Hour)
	proxyEndpoint := fmt.Sprintf("https://%s.dkr.ecr.%s.amazonaws.com",
		store.accountID, store.region)

	return jsonOK(getAuthorizationTokenResponse{
		AuthorizationData: []authorizationData{
			{
				AuthorizationToken: token,
				ExpiresAt:          float64(expiresAt.Unix()),
				ProxyEndpoint:      proxyEndpoint,
			},
		},
	})
}

func handleDescribeImageScanFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeImageScanFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	var resp describeImageScanFindingsResponse
	resp.RegistryId = store.accountID
	resp.RepositoryName = req.RepositoryName
	resp.ImageId = req.ImageId
	resp.ImageScanStatus.Status = "COMPLETE"
	resp.ImageScanStatus.Description = "The scan was completed successfully."
	resp.ImageScanFindings.Findings = []any{}
	resp.ImageScanFindings.FindingSeverityCounts = map[string]int{}

	return jsonOK(resp)
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	if awsErr := store.TagResource(req.ResourceArn, tagsFromList(req.Tags)); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	if awsErr := store.UntagResource(req.ResourceArn, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	tags, awsErr := store.ListTagsForResource(req.ResourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(listTagsForResourceResponse{Tags: tagsToList(tags)})
}
