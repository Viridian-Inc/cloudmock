package s3

import (
	"encoding/xml"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Bucket Tagging ───────────────────────────────────────────────────────────

type TagSet struct {
	XMLName xml.Name `xml:"Tagging"`
	Tags    []Tag    `xml:"TagSet>Tag"`
}

type Tag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

func handlePutBucketTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	body := ctx.Body

	var tagging TagSet
	if err := xml.Unmarshal(body, &tagging); err != nil {
		return s3Err("MalformedXML", "The XML you provided was not well-formed.")
	}

	store.setBucketConfig(bucket, "tagging", body)
	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatXML}, nil
}

func handleGetBucketTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	data := store.getBucketConfig(bucket, "tagging")
	if data == nil {
		return xmlResponse(http.StatusOK, `<Tagging><TagSet></TagSet></Tagging>`)
	}
	return xmlResponseRaw(http.StatusOK, data)
}

func handleDeleteBucketTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	store.deleteBucketConfig(bucket, "tagging")
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatXML}, nil
}

// ── Object Tagging ───────────────────────────────────────────────────────────

func handlePutObjectTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)
	body := ctx.Body

	store.setObjectConfig(bucket, key, "tagging", body)
	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatXML}, nil
}

func handleGetObjectTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)
	data := store.getObjectConfig(bucket, key, "tagging")
	if data == nil {
		return xmlResponse(http.StatusOK, `<Tagging><TagSet></TagSet></Tagging>`)
	}
	return xmlResponseRaw(http.StatusOK, data)
}

func handleDeleteObjectTagging(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)
	store.deleteObjectConfig(bucket, key, "tagging")
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatXML}, nil
}

// ── Bucket CORS ──────────────────────────────────────────────────────────────

func handlePutBucketCORS(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	body := ctx.Body
	store.setBucketConfig(bucket, "cors", body)
	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatXML}, nil
}

func handleGetBucketCORS(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	data := store.getBucketConfig(bucket, "cors")
	if data == nil {
		return s3Err("NoSuchCORSConfiguration", "The CORS configuration does not exist.")
	}
	return xmlResponseRaw(http.StatusOK, data)
}

func handleDeleteBucketCORS(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	store.deleteBucketConfig(bucket, "cors")
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatXML}, nil
}

// ── Bucket Lifecycle ─────────────────────────────────────────────────────────

func handlePutBucketLifecycle(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	body := ctx.Body
	store.setBucketConfig(bucket, "lifecycle", body)
	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatXML}, nil
}

func handleGetBucketLifecycle(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	data := store.getBucketConfig(bucket, "lifecycle")
	if data == nil {
		return s3Err("NoSuchLifecycleConfiguration", "The lifecycle configuration does not exist.")
	}
	return xmlResponseRaw(http.StatusOK, data)
}

// ── Bucket Notification ──────────────────────────────────────────────────────

func handlePutBucketNotification(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	body := ctx.Body
	store.setBucketConfig(bucket, "notification", body)
	return &service.Response{StatusCode: http.StatusOK, Format: service.FormatXML}, nil
}

func handleGetBucketNotification(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	data := store.getBucketConfig(bucket, "notification")
	if data == nil {
		return xmlResponse(http.StatusOK, `<NotificationConfiguration></NotificationConfiguration>`)
	}
	return xmlResponseRaw(http.StatusOK, data)
}

// ── Bucket ACL (real response, not stub) ─────────────────────────────────────

func handleGetBucketACLReal(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	data := store.getBucketConfig(extractBucketName(ctx), "acl")
	if data != nil {
		return xmlResponseRaw(http.StatusOK, data)
	}
	// Default ACL
	return xmlResponse(http.StatusOK, `<AccessControlPolicy>
  <Owner><ID>000000000000</ID><DisplayName>cloudmock</DisplayName></Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser">
        <ID>000000000000</ID><DisplayName>cloudmock</DisplayName>
      </Grantee>
      <Permission>FULL_CONTROL</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func s3Err(code, msg string) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML},
		service.NewAWSError(code, msg, http.StatusNotFound)
}

func xmlResponse(code int, body string) (*service.Response, error) {
	return &service.Response{
		StatusCode: code,
		RawBody:    []byte(xml.Header + body),
		Headers:    map[string]string{"Content-Type": "application/xml"},
		Format:     service.FormatXML,
	}, nil
}

func xmlResponseRaw(code int, data []byte) (*service.Response, error) {
	return &service.Response{
		StatusCode: code,
		RawBody:    data,
		Headers:    map[string]string{"Content-Type": "application/xml"},
		Format:     service.FormatXML,
	}, nil
}
