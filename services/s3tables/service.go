package s3tables

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// S3TablesService is the cloudmock implementation of the Amazon S3 Tables API.
type S3TablesService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new S3TablesService for the given AWS account ID and region.
func New(accountID, region string) *S3TablesService {
	return &S3TablesService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *S3TablesService) Name() string { return "s3tables" }

// Actions returns the list of S3 Tables API actions supported by this service.
func (s *S3TablesService) Actions() []service.Action {
	return []service.Action{
		// Table buckets
		{Name: "CreateTableBucket", Method: http.MethodPut, IAMAction: "s3tables:CreateTableBucket"},
		{Name: "GetTableBucket", Method: http.MethodGet, IAMAction: "s3tables:GetTableBucket"},
		{Name: "ListTableBuckets", Method: http.MethodGet, IAMAction: "s3tables:ListTableBuckets"},
		{Name: "DeleteTableBucket", Method: http.MethodDelete, IAMAction: "s3tables:DeleteTableBucket"},
		// Namespaces
		{Name: "CreateNamespace", Method: http.MethodPut, IAMAction: "s3tables:CreateNamespace"},
		{Name: "GetNamespace", Method: http.MethodGet, IAMAction: "s3tables:GetNamespace"},
		{Name: "ListNamespaces", Method: http.MethodGet, IAMAction: "s3tables:ListNamespaces"},
		{Name: "DeleteNamespace", Method: http.MethodDelete, IAMAction: "s3tables:DeleteNamespace"},
		// Tables
		{Name: "CreateTable", Method: http.MethodPut, IAMAction: "s3tables:CreateTable"},
		{Name: "GetTable", Method: http.MethodGet, IAMAction: "s3tables:GetTable"},
		{Name: "ListTables", Method: http.MethodGet, IAMAction: "s3tables:ListTables"},
		{Name: "UpdateTableMetadataLocation", Method: http.MethodPut, IAMAction: "s3tables:UpdateTableMetadataLocation"},
		{Name: "DeleteTable", Method: http.MethodDelete, IAMAction: "s3tables:DeleteTable"},
		// Table policies
		{Name: "PutTablePolicy", Method: http.MethodPut, IAMAction: "s3tables:PutTablePolicy"},
		{Name: "GetTablePolicy", Method: http.MethodGet, IAMAction: "s3tables:GetTablePolicy"},
		{Name: "DeleteTablePolicy", Method: http.MethodDelete, IAMAction: "s3tables:DeleteTablePolicy"},
	}
}

// HealthCheck always returns nil.
func (s *S3TablesService) HealthCheck() error { return nil }

// HandleRequest routes an incoming S3 Tables request to the appropriate handler.
// S3 Tables uses REST-JSON protocol with path-based routing.
func (s *S3TablesService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}

	// /buckets -> ListTableBuckets
	// /buckets/{arn} -> Get/DeleteTableBucket
	// /namespaces/{bucketARN}/{namespace} -> Namespace CRUD
	// /tables/{bucketARN}/{namespace}/{name} -> CRUD
	// /tables/{bucketARN}/{namespace}/{name}/metadata-location -> UpdateTableMetadataLocation
	// /policy/{tableARN} -> Policy CRUD

	if path == "/buckets" || path == "" {
		switch method {
		case http.MethodGet:
			return handleListTableBuckets(s.store)
		case http.MethodPut, http.MethodPost:
			return handleCreateTableBucket(params, s.store)
		}
	}

	if strings.HasPrefix(path, "/buckets/") {
		bucketARN := strings.TrimPrefix(path, "/buckets/")
		switch method {
		case http.MethodGet:
			return handleGetTableBucket(bucketARN, s.store)
		case http.MethodDelete:
			return handleDeleteTableBucket(bucketARN, s.store)
		}
	}

	if strings.HasPrefix(path, "/namespaces") {
		rest := strings.TrimPrefix(path, "/namespaces")
		if rest == "" || rest == "/" {
			// List namespaces for a bucket (bucketARN in query)
			bucketARN := r.URL.Query().Get("tableBucketARN")
			return handleListNamespaces(bucketARN, s.store)
		}
		rest = strings.TrimPrefix(rest, "/")
		// ARN contains slashes, so namespace is the last segment
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash > 0 {
			bucketARN := rest[:lastSlash]
			namespaceName := rest[lastSlash+1:]
			switch method {
			case http.MethodPut, http.MethodPost:
				return handleCreateNamespace(params, bucketARN, namespaceName, s.store)
			case http.MethodGet:
				return handleGetNamespace(bucketARN, namespaceName, s.store)
			case http.MethodDelete:
				return handleDeleteNamespace(bucketARN, namespaceName, s.store)
			}
		}
	}

	if strings.HasPrefix(path, "/tables") {
		rest := strings.TrimPrefix(path, "/tables")
		if rest == "" {
			// List tables for a bucket (bucketARN in query)
			bucketARN := r.URL.Query().Get("tableBucketARN")
			return handleListTables(bucketARN, s.store)
		}
		rest = strings.TrimPrefix(rest, "/")
		// Tables path: {bucketARN}/{namespace}/{name}[/metadata-location]
		// Since ARN contains slashes, we find namespace and name from the end.
		// ARN format is arn:aws:s3tables:{region}:{account}:bucket/{name} (one slash at end)
		// So structure after the ARN prefix is: /{namespace}/{tableName}[/metadata-location]
		// We split from the right: last 2 or 3 segments are tableName/namespace/optional-subpath
		// The ARN ends at the second-to-last "/" before namespace.
		// Strategy: split on "/" from right, taking the last 3 parts.
		// This works because ARN is everything before the (n-2)th slash from the right.
		segments := strings.Split(rest, "/")
		if len(segments) >= 3 {
			// Check for metadata-location sub-path (4+ segments from right)
			if len(segments) >= 4 && segments[len(segments)-1] == "metadata-location" {
				name := segments[len(segments)-2]
				namespace := segments[len(segments)-3]
				bucketARN := strings.Join(segments[:len(segments)-3], "/")
				if method == http.MethodPut {
					return handleUpdateTableMetadataLocation(params, bucketARN, namespace, name, s.store)
				}
			}
			name := segments[len(segments)-1]
			namespace := segments[len(segments)-2]
			bucketARN := strings.Join(segments[:len(segments)-2], "/")
			switch method {
			case http.MethodPut, http.MethodPost:
				return handleCreateTable(params, bucketARN, namespace, name, s.store)
			case http.MethodGet:
				return handleGetTable(bucketARN, namespace, name, s.store)
			case http.MethodDelete:
				return handleDeleteTable(bucketARN, namespace, name, s.store)
			}
		}
	}

	if strings.HasPrefix(path, "/policy/") {
		tableARN := strings.TrimPrefix(path, "/policy/")
		switch method {
		case http.MethodPut, http.MethodPost:
			return handlePutTablePolicy(params, tableARN, s.store)
		case http.MethodGet:
			return handleGetTablePolicy(tableARN, s.store)
		case http.MethodDelete:
			return handleDeleteTablePolicy(tableARN, s.store)
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
