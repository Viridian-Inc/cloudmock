package s3

import (
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// xmlEscape escapes special XML characters in a string.
func xmlEscape(s string) string {
	// Fast path: most S3 bucket/key names contain no special chars.
	if !strings.ContainsAny(s, "<>&\"'") {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---- helpers ----

// extractBucketName returns the first non-empty path segment from the request URL.
// E.g. "/my-bucket" → "my-bucket".
func extractBucketName(ctx *service.RequestContext) string {
	path := ctx.RawRequest.URL.Path
	path = strings.TrimPrefix(path, "/")
	if idx := strings.Index(path, "/"); idx >= 0 {
		path = path[:idx]
	}
	return path
}

// ---- handlers ----

func handleListBuckets(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	buckets := store.ListBuckets()

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	b.WriteString(`<Owner><ID>`)
	b.WriteString(xmlEscape(ctx.AccountID))
	b.WriteString(`</ID><DisplayName>cloudmock</DisplayName></Owner><Buckets>`)
	for _, bkt := range buckets {
		b.WriteString(`<Bucket><Name>`)
		b.WriteString(xmlEscape(bkt.Name))
		b.WriteString(`</Name><CreationDate>`)
		b.WriteString(bkt.CreationDate.Format("2006-01-02T15:04:05.000Z"))
		b.WriteString(`</CreationDate></Bucket>`)
	}
	b.WriteString(`</Buckets></ListAllMyBucketsResult>`)

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(b.String()),
		RawContentType: "application/xml",
	}, nil
}

func handleCreateBucket(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	name := extractBucketName(ctx)
	if err := store.CreateBucket(name); err != nil {
		if awsErr, ok := err.(*service.AWSError); ok {
			return &service.Response{Format: service.FormatXML}, awsErr
		}
		return nil, err
	}
	return &service.Response{
		StatusCode: http.StatusOK,
	}, nil
}

func handleDeleteBucket(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	name := extractBucketName(ctx)
	if err := store.DeleteBucket(name); err != nil {
		if awsErr, ok := err.(*service.AWSError); ok {
			return &service.Response{Format: service.FormatXML}, awsErr
		}
		return nil, err
	}
	return &service.Response{
		StatusCode: http.StatusNoContent,
	}, nil
}

func handleHeadBucket(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	name := extractBucketName(ctx)
	if err := store.HeadBucket(name); err != nil {
		if awsErr, ok := err.(*service.AWSError); ok {
			return &service.Response{Format: service.FormatXML}, awsErr
		}
		return nil, err
	}
	return &service.Response{
		StatusCode: http.StatusOK,
	}, nil
}
