package s3

import (
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- XML types ----

// xmlBucket is the XML representation of a single bucket entry.
type xmlBucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// xmlOwner is the XML representation of the bucket owner.
type xmlOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// xmlBuckets wraps the list of bucket entries.
type xmlBuckets struct {
	Buckets []xmlBucket `xml:"Bucket"`
}

// listAllMyBucketsResult is the top-level XML response for ListBuckets.
type listAllMyBucketsResult struct {
	XMLName xml.Name   `xml:"ListAllMyBucketsResult"`
	Xmlns   string     `xml:"xmlns,attr"`
	Owner   xmlOwner   `xml:"Owner"`
	Buckets xmlBuckets `xml:"Buckets"`
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

	xmlBkts := make([]xmlBucket, 0, len(buckets))
	for _, b := range buckets {
		xmlBkts = append(xmlBkts, xmlBucket{
			Name:         b.Name,
			CreationDate: b.CreationDate.Format("2006-01-02T15:04:05.000Z"),
		})
	}

	result := &listAllMyBucketsResult{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: xmlOwner{
			ID:          ctx.AccountID,
			DisplayName: "cloudmock",
		},
		Buckets: xmlBuckets{Buckets: xmlBkts},
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
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
		Body:       nil,
		Format:     service.FormatXML,
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
		Body:       nil,
		Format:     service.FormatXML,
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
		Body:       nil,
		Format:     service.FormatXML,
	}, nil
}
