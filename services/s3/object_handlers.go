package s3

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// extractObjectKey returns the object key portion of the request path — i.e.
// everything after the first path segment (bucket name).
// E.g. "/my-bucket/some/nested/key" → "some/nested/key".
func extractObjectKey(ctx *service.RequestContext) string {
	path := ctx.RawRequest.URL.Path
	path = strings.TrimPrefix(path, "/")
	idx := strings.Index(path, "/")
	if idx < 0 {
		return ""
	}
	return path[idx+1:]
}

// objectHeaders sets standard S3 object response headers on w.
func objectHeaders(w http.ResponseWriter, obj *Object) {
	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(obj.Size, 10))
	w.Header().Set("ETag", obj.ETag)
	w.Header().Set("Last-Modified", obj.LastModified.UTC().Format(http.TimeFormat))
}

// handlePutObject reads the request body and stores an object.
func handlePutObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	contentType := ctx.RawRequest.Header.Get("Content-Type")
	// ctx.Body was read by the gateway before calling HandleRequest.
	objs.PutObject(key, ctx.Body, contentType, nil)

	return &service.Response{
		StatusCode: http.StatusOK,
		Format:     service.FormatXML,
	}, nil
}

// handleGetObject writes the object body directly to the ResponseWriter via
// RawBody so that the gateway skips XML/JSON marshaling.
func handleGetObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	obj, err := objs.GetObject(key)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	headers := map[string]string{
		"ETag":          obj.ETag,
		"Last-Modified": obj.LastModified.UTC().Format(http.TimeFormat),
		"Content-Length": strconv.FormatInt(obj.Size, 10),
	}

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        obj.Body,
		RawContentType: obj.ContentType,
		Headers:        headers,
	}, nil
}

// handleDeleteObject removes an object and returns 204.
func handleDeleteObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	objs.DeleteObject(key)

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

// handleHeadObject returns object headers with no body.
func handleHeadObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	obj, err := objs.HeadObject(key)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Format:     service.FormatXML,
		Headers: map[string]string{
			"Content-Type":   obj.ContentType,
			"Content-Length": strconv.FormatInt(obj.Size, 10),
			"ETag":           obj.ETag,
			"Last-Modified":  obj.LastModified.UTC().Format(http.TimeFormat),
		},
	}, nil
}

// ---- ListObjectsV2 XML types ----

type xmlContent struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	ETag         string `xml:"ETag"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

type xmlCommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type listBucketResult struct {
	XMLName               xml.Name          `xml:"ListBucketResult"`
	Xmlns                 string            `xml:"xmlns,attr"`
	Name                  string            `xml:"Name"`
	Prefix                string            `xml:"Prefix"`
	Delimiter             string            `xml:"Delimiter,omitempty"`
	MaxKeys               int               `xml:"MaxKeys"`
	KeyCount              int               `xml:"KeyCount"`
	IsTruncated           bool              `xml:"IsTruncated"`
	NextContinuationToken string            `xml:"NextContinuationToken,omitempty"`
	ContinuationToken     string            `xml:"ContinuationToken,omitempty"`
	Contents              []xmlContent      `xml:"Contents"`
	CommonPrefixes        []xmlCommonPrefix `xml:"CommonPrefixes"`
}

// handleListObjectsV2 lists objects in a bucket.
func handleListObjectsV2(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	q := ctx.RawRequest.URL.Query()
	prefix := q.Get("prefix")
	delimiter := q.Get("delimiter")
	continuationToken := q.Get("continuation-token")

	maxKeys := 1000
	if mk := q.Get("max-keys"); mk != "" {
		if n, parseErr := strconv.Atoi(mk); parseErr == nil && n > 0 {
			maxKeys = n
		}
	}

	result := objs.ListObjects(prefix, delimiter, maxKeys, continuationToken)

	contents := make([]xmlContent, 0, len(result.Objects))
	for _, o := range result.Objects {
		contents = append(contents, xmlContent{
			Key:          o.Key,
			Size:         o.Size,
			ETag:         o.ETag,
			LastModified: o.LastModified.UTC().Format(time.RFC3339),
			StorageClass: "STANDARD",
		})
	}

	commonPrefixes := make([]xmlCommonPrefix, 0, len(result.CommonPrefixes))
	for _, cp := range result.CommonPrefixes {
		commonPrefixes = append(commonPrefixes, xmlCommonPrefix{Prefix: cp})
	}

	keyCount := len(contents) + len(commonPrefixes)

	body := &listBucketResult{
		Xmlns:                 "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:                  bucket,
		Prefix:                prefix,
		Delimiter:             delimiter,
		MaxKeys:               maxKeys,
		KeyCount:              keyCount,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
		ContinuationToken:     continuationToken,
		Contents:              contents,
		CommonPrefixes:        commonPrefixes,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// ---- CopyObject ----

type copyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

// handleCopyObject copies an object from a source bucket/key to a destination.
func handleCopyObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	destBucket := extractBucketName(ctx)
	destKey := extractObjectKey(ctx)

	// x-amz-copy-source may be URL-encoded; strip leading slash.
	copySource := ctx.RawRequest.Header.Get("x-amz-copy-source")
	if copySource == "" {
		copySource = ctx.RawRequest.Header.Get("X-Amz-Copy-Source")
	}
	copySource, _ = url.QueryUnescape(copySource)
	copySource = strings.TrimPrefix(copySource, "/")

	slashIdx := strings.Index(copySource, "/")
	if slashIdx < 0 {
		awsErr := service.NewAWSError("InvalidArgument",
			fmt.Sprintf("Invalid copy source: %q", copySource), http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	srcBucket := copySource[:slashIdx]
	srcKey := copySource[slashIdx+1:]

	srcObjs, err := store.bucketObjects(srcBucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	srcObj, err := srcObjs.GetObject(srcKey)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	destObjs, err := store.bucketObjects(destBucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	newObj := destObjs.PutObject(destKey, srcObj.Body, srcObj.ContentType, nil)

	body := &copyObjectResult{
		ETag:         newObj.ETag,
		LastModified: newObj.LastModified.UTC().Format(time.RFC3339),
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}
