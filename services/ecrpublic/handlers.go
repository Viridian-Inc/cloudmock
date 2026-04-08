package ecrpublic

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AuthorizationData struct {
	AuthorizationToken *string `json:"authorizationToken,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type BatchCheckLayerAvailabilityRequest struct {
	LayerDigests []string `json:"layerDigests,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type BatchCheckLayerAvailabilityResponse struct {
	Failures []LayerFailure `json:"failures,omitempty"`
	Layers []Layer `json:"layers,omitempty"`
}

type BatchDeleteImageRequest struct {
	ImageIds []ImageIdentifier `json:"imageIds,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type BatchDeleteImageResponse struct {
	Failures []ImageFailure `json:"failures,omitempty"`
	ImageIds []ImageIdentifier `json:"imageIds,omitempty"`
}

type CompleteLayerUploadRequest struct {
	LayerDigests []string `json:"layerDigests,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
	UploadId string `json:"uploadId,omitempty"`
}

type CompleteLayerUploadResponse struct {
	LayerDigest *string `json:"layerDigest,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
	UploadId *string `json:"uploadId,omitempty"`
}

type CreateRepositoryRequest struct {
	CatalogData *RepositoryCatalogDataInput `json:"catalogData,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type CreateRepositoryResponse struct {
	CatalogData *RepositoryCatalogData `json:"catalogData,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
}

type DeleteRepositoryPolicyRequest struct {
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type DeleteRepositoryPolicyResponse struct {
	PolicyText *string `json:"policyText,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
}

type DeleteRepositoryRequest struct {
	Force bool `json:"force,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type DeleteRepositoryResponse struct {
	Repository *Repository `json:"repository,omitempty"`
}

type DescribeImageTagsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type DescribeImageTagsResponse struct {
	ImageTagDetails []ImageTagDetail `json:"imageTagDetails,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type DescribeImagesRequest struct {
	ImageIds []ImageIdentifier `json:"imageIds,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type DescribeImagesResponse struct {
	ImageDetails []ImageDetail `json:"imageDetails,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type DescribeRegistriesRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type DescribeRegistriesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Registries []Registry `json:"registries,omitempty"`
}

type DescribeRepositoriesRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryNames []string `json:"repositoryNames,omitempty"`
}

type DescribeRepositoriesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Repositories []Repository `json:"repositories,omitempty"`
}

type GetAuthorizationTokenRequest struct {
}

type GetAuthorizationTokenResponse struct {
	AuthorizationData *AuthorizationData `json:"authorizationData,omitempty"`
}

type GetRegistryCatalogDataRequest struct {
}

type GetRegistryCatalogDataResponse struct {
	RegistryCatalogData RegistryCatalogData `json:"registryCatalogData,omitempty"`
}

type GetRepositoryCatalogDataRequest struct {
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type GetRepositoryCatalogDataResponse struct {
	CatalogData *RepositoryCatalogData `json:"catalogData,omitempty"`
}

type GetRepositoryPolicyRequest struct {
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type GetRepositoryPolicyResponse struct {
	PolicyText *string `json:"policyText,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
}

type Image struct {
	ImageId *ImageIdentifier `json:"imageId,omitempty"`
	ImageManifest *string `json:"imageManifest,omitempty"`
	ImageManifestMediaType *string `json:"imageManifestMediaType,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
}

type ImageDetail struct {
	ArtifactMediaType *string `json:"artifactMediaType,omitempty"`
	ImageDigest *string `json:"imageDigest,omitempty"`
	ImageManifestMediaType *string `json:"imageManifestMediaType,omitempty"`
	ImagePushedAt *time.Time `json:"imagePushedAt,omitempty"`
	ImageSizeInBytes int64 `json:"imageSizeInBytes,omitempty"`
	ImageTags []string `json:"imageTags,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
}

type ImageFailure struct {
	FailureCode *string `json:"failureCode,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	ImageId *ImageIdentifier `json:"imageId,omitempty"`
}

type ImageIdentifier struct {
	ImageDigest *string `json:"imageDigest,omitempty"`
	ImageTag *string `json:"imageTag,omitempty"`
}

type ImageTagDetail struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	ImageDetail *ReferencedImageDetail `json:"imageDetail,omitempty"`
	ImageTag *string `json:"imageTag,omitempty"`
}

type InitiateLayerUploadRequest struct {
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type InitiateLayerUploadResponse struct {
	PartSize int64 `json:"partSize,omitempty"`
	UploadId *string `json:"uploadId,omitempty"`
}

type Layer struct {
	LayerAvailability *string `json:"layerAvailability,omitempty"`
	LayerDigest *string `json:"layerDigest,omitempty"`
	LayerSize int64 `json:"layerSize,omitempty"`
	MediaType *string `json:"mediaType,omitempty"`
}

type LayerFailure struct {
	FailureCode *string `json:"failureCode,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	LayerDigest *string `json:"layerDigest,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags []Tag `json:"tags,omitempty"`
}

type PutImageRequest struct {
	ImageDigest *string `json:"imageDigest,omitempty"`
	ImageManifest string `json:"imageManifest,omitempty"`
	ImageManifestMediaType *string `json:"imageManifestMediaType,omitempty"`
	ImageTag *string `json:"imageTag,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type PutImageResponse struct {
	Image *Image `json:"image,omitempty"`
}

type PutRegistryCatalogDataRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
}

type PutRegistryCatalogDataResponse struct {
	RegistryCatalogData RegistryCatalogData `json:"registryCatalogData,omitempty"`
}

type PutRepositoryCatalogDataRequest struct {
	CatalogData RepositoryCatalogDataInput `json:"catalogData,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type PutRepositoryCatalogDataResponse struct {
	CatalogData *RepositoryCatalogData `json:"catalogData,omitempty"`
}

type ReferencedImageDetail struct {
	ArtifactMediaType *string `json:"artifactMediaType,omitempty"`
	ImageDigest *string `json:"imageDigest,omitempty"`
	ImageManifestMediaType *string `json:"imageManifestMediaType,omitempty"`
	ImagePushedAt *time.Time `json:"imagePushedAt,omitempty"`
	ImageSizeInBytes int64 `json:"imageSizeInBytes,omitempty"`
}

type Registry struct {
	Aliases []RegistryAlias `json:"aliases,omitempty"`
	RegistryArn string `json:"registryArn,omitempty"`
	RegistryId string `json:"registryId,omitempty"`
	RegistryUri string `json:"registryUri,omitempty"`
	Verified bool `json:"verified,omitempty"`
}

type RegistryAlias struct {
	DefaultRegistryAlias bool `json:"defaultRegistryAlias,omitempty"`
	Name string `json:"name,omitempty"`
	PrimaryRegistryAlias bool `json:"primaryRegistryAlias,omitempty"`
	Status string `json:"status,omitempty"`
}

type RegistryCatalogData struct {
	DisplayName *string `json:"displayName,omitempty"`
}

type Repository struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryArn *string `json:"repositoryArn,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
	RepositoryUri *string `json:"repositoryUri,omitempty"`
}

type RepositoryCatalogData struct {
	AboutText *string `json:"aboutText,omitempty"`
	Architectures []string `json:"architectures,omitempty"`
	Description *string `json:"description,omitempty"`
	LogoUrl *string `json:"logoUrl,omitempty"`
	MarketplaceCertified bool `json:"marketplaceCertified,omitempty"`
	OperatingSystems []string `json:"operatingSystems,omitempty"`
	UsageText *string `json:"usageText,omitempty"`
}

type RepositoryCatalogDataInput struct {
	AboutText *string `json:"aboutText,omitempty"`
	Architectures []string `json:"architectures,omitempty"`
	Description *string `json:"description,omitempty"`
	LogoImageBlob []byte `json:"logoImageBlob,omitempty"`
	OperatingSystems []string `json:"operatingSystems,omitempty"`
	UsageText *string `json:"usageText,omitempty"`
}

type SetRepositoryPolicyRequest struct {
	Force bool `json:"force,omitempty"`
	PolicyText string `json:"policyText,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type SetRepositoryPolicyResponse struct {
	PolicyText *string `json:"policyText,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
}

type Tag struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type TagResourceResponse struct {
}

type UntagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UploadLayerPartRequest struct {
	LayerPartBlob []byte `json:"layerPartBlob,omitempty"`
	PartFirstByte int64 `json:"partFirstByte,omitempty"`
	PartLastByte int64 `json:"partLastByte,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
	UploadId string `json:"uploadId,omitempty"`
}

type UploadLayerPartResponse struct {
	LastByteReceived int64 `json:"lastByteReceived,omitempty"`
	RegistryId *string `json:"registryId,omitempty"`
	RepositoryName *string `json:"repositoryName,omitempty"`
	UploadId *string `json:"uploadId,omitempty"`
}



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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleBatchCheckLayerAvailability(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchCheckLayerAvailabilityRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchCheckLayerAvailability business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchCheckLayerAvailability"})
}

func handleBatchDeleteImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDeleteImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDeleteImage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDeleteImage"})
}

func handleCompleteLayerUpload(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CompleteLayerUploadRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CompleteLayerUpload business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CompleteLayerUpload"})
}

func handleCreateRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateRepositoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateRepository business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateRepository"})
}

func handleDeleteRepository(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteRepositoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteRepository business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteRepository"})
}

func handleDeleteRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteRepositoryPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteRepositoryPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteRepositoryPolicy"})
}

func handleDescribeImageTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeImageTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeImageTags business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeImageTags"})
}

func handleDescribeImages(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeImagesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeImages business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeImages"})
}

func handleDescribeRegistries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeRegistriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeRegistries business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeRegistries"})
}

func handleDescribeRepositories(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeRepositoriesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeRepositories business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeRepositories"})
}

func handleGetAuthorizationToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAuthorizationTokenRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAuthorizationToken business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAuthorizationToken"})
}

func handleGetRegistryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetRegistryCatalogDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetRegistryCatalogData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetRegistryCatalogData"})
}

func handleGetRepositoryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetRepositoryCatalogDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetRepositoryCatalogData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetRepositoryCatalogData"})
}

func handleGetRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetRepositoryPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetRepositoryPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetRepositoryPolicy"})
}

func handleInitiateLayerUpload(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req InitiateLayerUploadRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement InitiateLayerUpload business logic
	return jsonOK(map[string]any{"status": "ok", "action": "InitiateLayerUpload"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handlePutImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutImage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutImage"})
}

func handlePutRegistryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutRegistryCatalogDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutRegistryCatalogData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutRegistryCatalogData"})
}

func handlePutRepositoryCatalogData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutRepositoryCatalogDataRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutRepositoryCatalogData business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutRepositoryCatalogData"})
}

func handleSetRepositoryPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SetRepositoryPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SetRepositoryPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SetRepositoryPolicy"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUploadLayerPart(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UploadLayerPartRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UploadLayerPart business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UploadLayerPart"})
}

