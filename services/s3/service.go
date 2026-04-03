package s3

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/schema"
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
		{Name: "CreateMultipartUpload", Method: http.MethodPost, IAMAction: "s3:PutObject"},
		{Name: "UploadPart", Method: http.MethodPut, IAMAction: "s3:PutObject"},
		{Name: "CompleteMultipartUpload", Method: http.MethodPost, IAMAction: "s3:PutObject"},
		{Name: "AbortMultipartUpload", Method: http.MethodDelete, IAMAction: "s3:AbortMultipartUpload"},
		{Name: "ListMultipartUploads", Method: http.MethodGet, IAMAction: "s3:ListBucketMultipartUploads"},
		{Name: "ListParts", Method: http.MethodGet, IAMAction: "s3:ListMultipartUploadParts"},
		{Name: "PutBucketVersioning", Method: http.MethodPut, IAMAction: "s3:PutBucketVersioning"},
		{Name: "GetBucketVersioning", Method: http.MethodGet, IAMAction: "s3:GetBucketVersioning"},
		{Name: "ListObjectVersions", Method: http.MethodGet, IAMAction: "s3:ListBucketVersions"},
		{Name: "PutBucketPolicy", Method: http.MethodPut, IAMAction: "s3:PutBucketPolicy"},
		{Name: "GetBucketPolicy", Method: http.MethodGet, IAMAction: "s3:GetBucketPolicy"},
		{Name: "DeleteBucketPolicy", Method: http.MethodDelete, IAMAction: "s3:DeleteBucketPolicy"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *S3Service) HealthCheck() error { return nil }

// GetBucketNames returns all bucket names for topology queries.
func (s *S3Service) GetBucketNames() []string {
	buckets := s.store.ListBuckets()
	names := make([]string, 0, len(buckets))
	for _, b := range buckets {
		names = append(names, b.Name)
	}
	return names
}

// ResourceSchemas returns the schema for S3 bucket resources.
func (s *S3Service) ResourceSchemas() []schema.ResourceSchema {
	return []schema.ResourceSchema{
		{
			ServiceName:   "s3",
			ResourceType:  "aws_s3_bucket",
			TerraformType: "cloudmock_s3_bucket",
			AWSType:       "AWS::S3::Bucket",
			CreateAction:  "CreateBucket",
			ReadAction:    "HeadBucket",
			DeleteAction:  "DeleteBucket",
			ListAction:    "ListBuckets",
			ImportID:      "bucket",
			Attributes: []schema.AttributeSchema{
				{Name: "bucket", Type: "string", Required: true, ForceNew: true},
				{Name: "arn", Type: "string", Computed: true},
				{Name: "region", Type: "string", Computed: true},
				{Name: "acl", Type: "string", Default: "private"},
				{Name: "tags", Type: "map"},
			},
		},
	}
}

// routeKey identifies a unique S3 route by HTTP method and subresource.
type routeKey struct {
	method string
	scope  routeScope
	sub    string // query parameter subresource: "uploads", "versioning", etc.
}

// routeScope distinguishes root, bucket-level, and object-level paths.
type routeScope int

const (
	scopeRoot   routeScope = iota
	scopeBucket routeScope = iota
	scopeObject routeScope = iota
)

// s3Handler is a handler function for an S3 route.
type s3Handler func(store *Store, ctx *service.RequestContext) (*service.Response, error)

// routes maps (method, scope, subresource) → handler for O(1) dispatch.
var routes = map[routeKey]s3Handler{
	// GET routes
	{http.MethodGet, scopeRoot, ""}:            handleListBuckets,
	{http.MethodGet, scopeBucket, ""}:           handleListObjectsV2,
	{http.MethodGet, scopeBucket, "uploads"}:    handleListMultipartUploads,
	{http.MethodGet, scopeBucket, "versioning"}: handleGetBucketVersioning,
	{http.MethodGet, scopeBucket, "versions"}:   handleListObjectVersions,
	{http.MethodGet, scopeBucket, "policy"}:     handleGetBucketPolicy,
	{http.MethodGet, scopeObject, ""}:           handleGetObject,
	{http.MethodGet, scopeObject, "uploadId"}:   handleListParts,
	// POST routes
	{http.MethodPost, scopeObject, "uploads"}:   handleCreateMultipartUpload,
	// PUT routes
	{http.MethodPut, scopeBucket, ""}:            handleCreateBucket,
	{http.MethodPut, scopeBucket, "versioning"}:  handlePutBucketVersioning,
	{http.MethodPut, scopeBucket, "policy"}:      handlePutBucketPolicy,
	// DELETE routes
	{http.MethodDelete, scopeBucket, ""}:                  handleDeleteBucket,
	{http.MethodDelete, scopeBucket, "policy"}:            handleDeleteBucketPolicy,
	{http.MethodDelete, scopeBucket, "tagging"}:           handleNoOpBucket,
	{http.MethodDelete, scopeBucket, "cors"}:              handleNoOpBucket,
	{http.MethodDelete, scopeBucket, "lifecycle"}:         handleNoOpBucket,
	{http.MethodDelete, scopeBucket, "encryption"}:        handleNoOpBucket,
	{http.MethodDelete, scopeBucket, "publicAccessBlock"}: handleNoOpBucket,
	{http.MethodDelete, scopeBucket, "ownershipControls"}: handleNoOpBucket,
	// HEAD routes
	{http.MethodHead, scopeBucket, ""}:  handleHeadBucket,
	{http.MethodHead, scopeObject, ""}: handleHeadObject,
	// GET subresource stubs (Terraform/Pulumi read these after creating a bucket)
	{http.MethodGet, scopeBucket, "tagging"}:           handleGetEmptyXML("Tagging", "TagSet"),
	{http.MethodGet, scopeBucket, "cors"}:              handleGetNoSuchConfig("CORSConfiguration"),
	{http.MethodGet, scopeBucket, "lifecycle"}:         handleGetNoSuchConfig("LifecycleConfiguration"),
	{http.MethodGet, scopeBucket, "encryption"}:        handleGetBucketEncryption,
	{http.MethodGet, scopeBucket, "acl"}:               handleGetBucketACL,
	{http.MethodGet, scopeBucket, "location"}:          handleGetBucketLocation,
	{http.MethodGet, scopeBucket, "logging"}:           handleGetEmptyXML("BucketLoggingStatus", ""),
	{http.MethodGet, scopeBucket, "notification"}:      handleGetEmptyXML("NotificationConfiguration", ""),
	{http.MethodGet, scopeBucket, "website"}:           handleGetNoSuchConfig("WebsiteConfiguration"),
	{http.MethodGet, scopeBucket, "accelerate"}:        handleGetEmptyXML("AccelerateConfiguration", ""),
	{http.MethodGet, scopeBucket, "ownershipControls"}: handleGetOwnershipControls,
	{http.MethodGet, scopeBucket, "publicAccessBlock"}: handleGetPublicAccessBlock,
	// PUT subresource stubs
	{http.MethodPut, scopeBucket, "tagging"}:           handleNoOpBucket,
	{http.MethodPut, scopeBucket, "cors"}:              handleNoOpBucket,
	{http.MethodPut, scopeBucket, "lifecycle"}:         handleNoOpBucket,
	{http.MethodPut, scopeBucket, "encryption"}:        handleNoOpBucket,
	{http.MethodPut, scopeBucket, "acl"}:               handleNoOpBucket,
	{http.MethodPut, scopeBucket, "logging"}:           handleNoOpBucket,
	{http.MethodPut, scopeBucket, "notification"}:      handleNoOpBucket,
	{http.MethodPut, scopeBucket, "website"}:           handleNoOpBucket,
	{http.MethodPut, scopeBucket, "accelerate"}:        handleNoOpBucket,
	{http.MethodPut, scopeBucket, "ownershipControls"}: handleNoOpBucket,
	{http.MethodPut, scopeBucket, "publicAccessBlock"}: handleNoOpBucket,
}

// HandleRequest routes an incoming S3 request to the appropriate handler.
func (s *S3Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest

	bucketName := extractBucketName(ctx)
	objectKey := extractObjectKey(ctx)

	q := r.URL.Query()

	// Determine the scope of the request.
	scope := scopeRoot
	if bucketName != "" {
		if objectKey != "" {
			scope = scopeObject
		} else {
			scope = scopeBucket
		}
	}

	// Determine the subresource for map lookup.
	// Ignore x-id, x-amz-*, and other SDK metadata query params.
	sub := ""
	switch {
	case q.Has("uploads"):
		sub = "uploads"
	case q.Has("versioning"):
		sub = "versioning"
	case q.Has("versions"):
		sub = "versions"
	case q.Has("policy"):
		sub = "policy"
	case q.Has("tagging"):
		sub = "tagging"
	case q.Has("cors"):
		sub = "cors"
	case q.Has("lifecycle"):
		sub = "lifecycle"
	case q.Has("encryption"):
		sub = "encryption"
	case q.Has("acl"):
		sub = "acl"
	case q.Has("location"):
		sub = "location"
	case q.Has("notification"):
		sub = "notification"
	case q.Has("logging"):
		sub = "logging"
	case q.Has("website"):
		sub = "website"
	case q.Has("accelerate"):
		sub = "accelerate"
	case q.Has("ownershipControls"):
		sub = "ownershipControls"
	case q.Has("publicAccessBlock"):
		sub = "publicAccessBlock"
	case q.Get("uploadId") != "":
		sub = "uploadId"
	case q.Get("partNumber") != "":
		sub = "partNumber"
	}

	// --- Special cases that can't be expressed as simple map lookups ---

	// POST /bucket/key?uploadId=X → CompleteMultipartUpload (has event bus side-effect)
	if r.Method == http.MethodPost && scope == scopeObject && sub == "uploadId" {
		resp, err := handleCompleteMultipartUpload(s.store, ctx)
		if err == nil && s.bus != nil {
			s.publishObjectEvent(ctx, bucketName, objectKey, "s3:ObjectCreated:CompleteMultipartUpload")
		}
		return resp, err
	}

	// PUT /bucket/key?partNumber=N&uploadId=X → UploadPart
	if r.Method == http.MethodPut && scope == scopeObject && q.Get("uploadId") != "" && q.Get("partNumber") != "" {
		return handleUploadPart(s.store, ctx)
	}

	// PUT /bucket/key → PutObject or CopyObject (with event bus side-effect)
	if r.Method == http.MethodPut && scope == scopeObject && sub == "" {
		if r.Header.Get("x-amz-copy-source") != "" || r.Header.Get("X-Amz-Copy-Source") != "" {
			return handleCopyObject(s.store, ctx)
		}
		resp, err := handlePutObject(s.store, ctx)
		if err == nil && s.bus != nil {
			s.publishObjectEvent(ctx, bucketName, objectKey, "s3:ObjectCreated:Put")
		}
		return resp, err
	}

	// DELETE /bucket/key?uploadId=X → AbortMultipartUpload
	if r.Method == http.MethodDelete && scope == scopeObject && sub == "uploadId" {
		return handleAbortMultipartUpload(s.store, ctx)
	}

	// DELETE /bucket/key → DeleteObject (with event bus side-effect)
	if r.Method == http.MethodDelete && scope == scopeObject && sub == "" {
		resp, err := handleDeleteObject(s.store, ctx)
		if err == nil && s.bus != nil {
			s.publishObjectEvent(ctx, bucketName, objectKey, "s3:ObjectRemoved:Delete")
		}
		return resp, err
	}

	// --- Map-based dispatch for all other routes ---
	if handler, ok := routes[routeKey{r.Method, scope, sub}]; ok {
		return handler(s.store, ctx)
	}

	awsErr := service.NewAWSError(
		"NotImplemented",
		"This operation is not implemented by cloudmock.",
		http.StatusNotImplemented,
	)
	return &service.Response{Format: service.FormatXML}, awsErr
}

// GetObjectData retrieves the raw bytes of an S3 object by bucket and key.
// This is used for cross-service communication (e.g., Lambda fetching code from S3).
func (s *S3Service) GetObjectData(bucket, key string) ([]byte, error) {
	objs, err := s.store.bucketObjects(bucket)
	if err != nil {
		return nil, err
	}
	obj, err := objs.GetObject(key)
	if err != nil {
		return nil, err
	}
	return obj.Body, nil
}

// publishObjectEvent sends an S3 object event to the event bus asynchronously.
// The S3 API response does not wait for event delivery.
func (s *S3Service) publishObjectEvent(ctx *service.RequestContext, bucket, key, eventType string) {
	// Capture context values before spawning goroutine — ctx must not be
	// accessed after HandleRequest returns.
	accountID := ctx.AccountID
	if accountID == "" {
		accountID = "000000000000"
	}
	region := ctx.Region
	if region == "" {
		region = "us-east-1"
	}

	go func() {
		// Look up object metadata for the event detail.
		detail := map[string]any{
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

		s.bus.Publish(&eventbus.Event{
			Source:    "s3",
			Type:      eventType,
			Detail:    detail,
			Time:      time.Now().UTC(),
			Region:    region,
			AccountID: accountID,
		})
	}()
}
