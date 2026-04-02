package s3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// xmlHeader is the standard XML declaration prepended to all XML responses.
const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`

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
// If versioning is enabled on the bucket, a new version is created.
func handlePutObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	contentType := ctx.RawRequest.Header.Get("Content-Type")

	resp := &service.Response{
		StatusCode: http.StatusOK,
	}

	if store.IsVersioningEnabled(bucket) {
		obj := objs.PutObjectVersioned(key, ctx.Body, contentType, nil)
		resp.Headers = map[string]string{
			"x-amz-version-id": obj.VersionId,
		}
	} else {
		objs.PutObject(key, ctx.Body, contentType, nil)
	}

	return resp, nil
}

// handleGetObject writes the object body directly to the ResponseWriter via
// RawBody so that the gateway skips XML/JSON marshaling.
// Supports ?versionId=X to retrieve a specific version.
func handleGetObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	versionId := ctx.RawRequest.URL.Query().Get("versionId")

	var obj *Object
	if versionId != "" {
		obj, err = objs.GetObjectVersion(key, versionId)
	} else {
		obj, err = objs.GetObject(key)
	}
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	headers := map[string]string{
		"ETag":           obj.ETag,
		"Last-Modified":  obj.LastModified.UTC().Format(http.TimeFormat),
		"Content-Length": strconv.FormatInt(obj.Size, 10),
	}
	if obj.VersionId != "" {
		headers["x-amz-version-id"] = obj.VersionId
	}

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        obj.Body,
		RawContentType: obj.ContentType,
		Headers:        headers,
	}, nil
}

// handleDeleteObject removes an object and returns 204.
// If versioning is enabled, creates a delete marker instead of actually deleting.
func handleDeleteObject(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	resp := &service.Response{
		StatusCode: http.StatusNoContent,
	}

	if store.IsVersioningEnabled(bucket) {
		versionId := objs.DeleteObjectVersioned(key)
		resp.Headers = map[string]string{
			"x-amz-version-id":    versionId,
			"x-amz-delete-marker": "true",
		}
	} else {
		objs.DeleteObject(key)
	}

	return resp, nil
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
		Headers: map[string]string{
			"Content-Type":   obj.ContentType,
			"Content-Length": strconv.FormatInt(obj.Size, 10),
			"ETag":           obj.ETag,
			"Last-Modified":  obj.LastModified.UTC().Format(http.TimeFormat),
		},
	}, nil
}

// handleListObjectsV2 lists objects in a bucket.
// Uses direct string building to avoid xml.Marshal reflection overhead.
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

	keyCount := len(result.Objects) + len(result.CommonPrefixes)

	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	b.WriteString(`<Name>`)
	b.WriteString(xmlEscape(bucket))
	b.WriteString(`</Name><Prefix>`)
	b.WriteString(xmlEscape(prefix))
	b.WriteString(`</Prefix>`)
	if delimiter != "" {
		b.WriteString(`<Delimiter>`)
		b.WriteString(xmlEscape(delimiter))
		b.WriteString(`</Delimiter>`)
	}
	b.WriteString(`<MaxKeys>`)
	b.WriteString(strconv.Itoa(maxKeys))
	b.WriteString(`</MaxKeys><KeyCount>`)
	b.WriteString(strconv.Itoa(keyCount))
	b.WriteString(`</KeyCount><IsTruncated>`)
	if result.IsTruncated {
		b.WriteString(`true`)
	} else {
		b.WriteString(`false`)
	}
	b.WriteString(`</IsTruncated>`)
	if result.NextContinuationToken != "" {
		b.WriteString(`<NextContinuationToken>`)
		b.WriteString(xmlEscape(result.NextContinuationToken))
		b.WriteString(`</NextContinuationToken>`)
	}
	if continuationToken != "" {
		b.WriteString(`<ContinuationToken>`)
		b.WriteString(xmlEscape(continuationToken))
		b.WriteString(`</ContinuationToken>`)
	}
	for _, o := range result.Objects {
		b.WriteString(`<Contents><Key>`)
		b.WriteString(xmlEscape(o.Key))
		b.WriteString(`</Key><Size>`)
		b.WriteString(strconv.FormatInt(o.Size, 10))
		b.WriteString(`</Size><ETag>`)
		b.WriteString(xmlEscape(o.ETag))
		b.WriteString(`</ETag><LastModified>`)
		b.WriteString(o.LastModified.UTC().Format(time.RFC3339))
		b.WriteString(`</LastModified><StorageClass>STANDARD</StorageClass></Contents>`)
	}
	for _, cp := range result.CommonPrefixes {
		b.WriteString(`<CommonPrefixes><Prefix>`)
		b.WriteString(xmlEscape(cp))
		b.WriteString(`</Prefix></CommonPrefixes>`)
	}
	b.WriteString(`</ListBucketResult>`)

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(b.String()),
		RawContentType: "application/xml",
	}, nil
}

// ---- CopyObject ----

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

	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString(`<CopyObjectResult><ETag>`)
	b.WriteString(xmlEscape(newObj.ETag))
	b.WriteString(`</ETag><LastModified>`)
	b.WriteString(newObj.LastModified.UTC().Format(time.RFC3339))
	b.WriteString(`</LastModified></CopyObjectResult>`)

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(b.String()),
		RawContentType: "application/xml",
	}, nil
}
