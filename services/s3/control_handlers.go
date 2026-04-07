package s3

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// handleS3Control routes S3 Control API requests (/v20180820/...).
// These are used by Terraform and Pulumi AWS providers for bucket tagging
// via the s3control signing scope instead of the regular S3 tagging API.
func (s *S3Service) handleS3Control(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	path := r.URL.Path

	switch {
	case strings.Contains(path, "/tags"):
		return s.handleS3ControlTags(ctx)
	default:
		return &service.Response{StatusCode: http.StatusOK}, nil
	}
}

// handleS3ControlTags handles ListTagsForResource, TagResource, and
// UntagResource from the S3 Control API. The bucket is identified by the
// s3:ResourceArn query parameter (e.g. "arn:aws:s3:::my-bucket").
func (s *S3Service) handleS3ControlTags(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest

	// Extract bucket name from ResourceArn query param.
	// Format: arn:aws:s3:::bucket-name
	arn := r.URL.Query().Get("s3:ResourceArn")
	if arn == "" {
		arn = r.URL.Query().Get("resourceArn")
	}
	_ = extractBucketFromARN(arn) // reserved for future tag storage

	switch r.Method {
	case http.MethodGet:
		// ListTagsForResource -- return empty TagSet.
		xml := `<?xml version="1.0" encoding="UTF-8"?>` +
			`<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<TagSet/>` +
			`</Tagging>`
		return &service.Response{
			StatusCode:     http.StatusOK,
			RawBody:        []byte(xml),
			RawContentType: "application/xml",
		}, nil
	case http.MethodPut:
		// TagResource -- accept and ignore.
		return &service.Response{StatusCode: http.StatusOK}, nil
	case http.MethodDelete:
		// UntagResource -- accept and ignore.
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	default:
		return &service.Response{StatusCode: http.StatusOK}, nil
	}
}

// extractBucketFromARN extracts the bucket name from an S3 ARN.
// Expected format: arn:aws:s3:::bucket-name
func extractBucketFromARN(arn string) string {
	if parts := strings.Split(arn, ":::"); len(parts) == 2 {
		return parts[1]
	}
	return ""
}
