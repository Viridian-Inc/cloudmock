package s3

import (
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// S3Service is the cloudmock implementation of the Amazon S3 API.
type S3Service struct {
	store *Store
}

// New returns a new S3Service with an empty bucket store.
func New() *S3Service {
	return &S3Service{
		store: NewStore(),
	}
}

// Name returns the AWS service name used for routing.
func (s *S3Service) Name() string { return "s3" }

// Actions returns the list of S3 API actions supported by this service.
func (s *S3Service) Actions() []service.Action {
	return []service.Action{
		{Name: "ListBuckets", Method: http.MethodGet, IAMAction: "s3:ListAllMyBuckets"},
		{Name: "CreateBucket", Method: http.MethodPut, IAMAction: "s3:CreateBucket"},
		{Name: "DeleteBucket", Method: http.MethodDelete, IAMAction: "s3:DeleteBucket"},
		{Name: "HeadBucket", Method: http.MethodHead, IAMAction: "s3:ListBucket"},
		{Name: "PutObject", Method: http.MethodPut, IAMAction: "s3:PutObject"},
		{Name: "GetObject", Method: http.MethodGet, IAMAction: "s3:GetObject"},
		{Name: "DeleteObject", Method: http.MethodDelete, IAMAction: "s3:DeleteObject"},
		{Name: "HeadObject", Method: http.MethodHead, IAMAction: "s3:GetObject"},
		{Name: "ListObjectsV2", Method: http.MethodGet, IAMAction: "s3:ListBucket"},
		{Name: "CopyObject", Method: http.MethodPut, IAMAction: "s3:PutObject"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *S3Service) HealthCheck() error { return nil }

// HandleRequest routes an incoming S3 request to the appropriate handler.
func (s *S3Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	path := r.URL.Path

	// Normalise path: strip trailing slash for non-root paths.
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}

	// Determine path segments to distinguish bucket vs object paths.
	bucketName := extractBucketName(ctx)
	isBucketPath := bucketName != ""
	objectKey := extractObjectKey(ctx)
	isObjectPath := objectKey != ""

	switch r.Method {
	case http.MethodGet:
		if !isBucketPath {
			// GET / → ListBuckets
			return handleListBuckets(s.store, ctx)
		}
		if isObjectPath {
			// GET /bucket/key → GetObject
			return handleGetObject(s.store, ctx)
		}
		// GET /bucket or GET /bucket?list-type=2 → ListObjectsV2
		return handleListObjectsV2(s.store, ctx)

	case http.MethodPut:
		if isBucketPath && isObjectPath {
			// PUT /bucket/key with copy-source → CopyObject
			if r.Header.Get("x-amz-copy-source") != "" || r.Header.Get("X-Amz-Copy-Source") != "" {
				return handleCopyObject(s.store, ctx)
			}
			// PUT /bucket/key → PutObject
			return handlePutObject(s.store, ctx)
		}
		if isBucketPath {
			// PUT /bucket → CreateBucket
			return handleCreateBucket(s.store, ctx)
		}

	case http.MethodDelete:
		if isBucketPath && isObjectPath {
			// DELETE /bucket/key → DeleteObject
			return handleDeleteObject(s.store, ctx)
		}
		if isBucketPath {
			// DELETE /bucket → DeleteBucket
			return handleDeleteBucket(s.store, ctx)
		}

	case http.MethodHead:
		if isBucketPath && isObjectPath {
			// HEAD /bucket/key → HeadObject
			return handleHeadObject(s.store, ctx)
		}
		if isBucketPath {
			// HEAD /bucket → HeadBucket
			return handleHeadBucket(s.store, ctx)
		}
	}

	// Anything else is not implemented.
	awsErr := service.NewAWSError(
		"NotImplemented",
		"This operation is not implemented by cloudmock.",
		http.StatusNotImplemented,
	)
	return &service.Response{Format: service.FormatXML}, awsErr
}
