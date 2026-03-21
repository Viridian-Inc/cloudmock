package s3

import (
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/service"
)

// S3Service is the cloudmock implementation of the Amazon S3 API.
type S3Service struct {
	store *Store
	bus   *eventbus.Bus
}

// New returns a new S3Service with an empty bucket store.
func New() *S3Service {
	return &S3Service{
		store: NewStore(),
	}
}

// NewWithBus returns a new S3Service that publishes events to the given bus.
func NewWithBus(bus *eventbus.Bus) *S3Service {
	return &S3Service{
		store: NewStore(),
		bus:   bus,
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
			resp, err := handlePutObject(s.store, ctx)
			if err == nil && s.bus != nil {
				s.publishObjectEvent(ctx, bucketName, objectKey, "s3:ObjectCreated:Put")
			}
			return resp, err
		}
		if isBucketPath {
			// PUT /bucket → CreateBucket
			return handleCreateBucket(s.store, ctx)
		}

	case http.MethodDelete:
		if isBucketPath && isObjectPath {
			// DELETE /bucket/key → DeleteObject
			resp, err := handleDeleteObject(s.store, ctx)
			if err == nil && s.bus != nil {
				s.publishObjectEvent(ctx, bucketName, objectKey, "s3:ObjectRemoved:Delete")
			}
			return resp, err
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

// publishObjectEvent sends an S3 object event to the event bus.
func (s *S3Service) publishObjectEvent(ctx *service.RequestContext, bucket, key, eventType string) {
	// Look up object metadata for the event detail.
	detail := map[string]interface{}{
		"bucket": bucket,
		"key":    key,
	}

	// Try to include size and etag from the object store.
	if objs, err := s.store.bucketObjects(bucket); err == nil {
		if obj, err := objs.GetObject(key); err == nil {
			detail["size"] = obj.Size
			detail["etag"] = obj.ETag
		}
	}

	accountID := ctx.AccountID
	if accountID == "" {
		accountID = "000000000000"
	}
	region := ctx.Region
	if region == "" {
		region = "us-east-1"
	}

	s.bus.Publish(&eventbus.Event{
		Source:    "s3",
		Type:      eventType,
		Detail:    detail,
		Time:      time.Now().UTC(),
		Region:    region,
		AccountID: accountID,
	})
}
