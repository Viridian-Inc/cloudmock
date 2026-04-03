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

// ---- stub handlers for bucket subresources (Terraform/Pulumi compatibility) ----

// handleNoOpBucket accepts the request and returns 200 with no body.
func handleNoOpBucket(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK}, nil
}

// handleGetEmptyXML returns a minimal XML response for a subresource that has no configuration.
func handleGetEmptyXML(root, child string) s3Handler {
	return func(store *Store, ctx *service.RequestContext) (*service.Response, error) {
		xml := `<?xml version="1.0" encoding="UTF-8"?><` + root + ` xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`
		if child != "" {
			xml += `<` + child + `/>`
		}
		xml += `</` + root + `>`
		return &service.Response{
			StatusCode:     http.StatusOK,
			RawBody:        []byte(xml),
			RawContentType: "application/xml",
		}, nil
	}
}

// handleGetNoSuchConfig returns a 404 NoSuchXxxConfiguration error (expected by Terraform when config doesn't exist).
func handleGetNoSuchConfig(configName string) s3Handler {
	return func(store *Store, ctx *service.RequestContext) (*service.Response, error) {
		code := "NoSuch" + configName
		xml := `<?xml version="1.0" encoding="UTF-8"?><Error><Code>` + code + `</Code><Message>The ` + configName + ` does not exist.</Message></Error>`
		return &service.Response{
			StatusCode:     http.StatusNotFound,
			RawBody:        []byte(xml),
			RawContentType: "application/xml",
		}, nil
	}
}

func handleGetBucketEncryption(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><ServerSideEncryptionConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule><ApplyServerSideEncryptionByDefault><SSEAlgorithm>AES256</SSEAlgorithm></ApplyServerSideEncryptionByDefault><BucketKeyEnabled>false</BucketKeyEnabled></Rule></ServerSideEncryptionConfiguration>`
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(xml),
		RawContentType: "application/xml",
	}, nil
}

func handleGetBucketACL(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Owner><ID>` + xmlEscape(ctx.AccountID) + `</ID><DisplayName>cloudmock</DisplayName></Owner><AccessControlList><Grant><Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser"><ID>` + xmlEscape(ctx.AccountID) + `</ID><DisplayName>cloudmock</DisplayName></Grantee><Permission>FULL_CONTROL</Permission></Grant></AccessControlList></AccessControlPolicy>`
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(xml),
		RawContentType: "application/xml",
	}, nil
}

func handleGetBucketLocation(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(xml),
		RawContentType: "application/xml",
	}, nil
}

func handleGetOwnershipControls(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><OwnershipControls xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Rule><ObjectOwnership>BucketOwnerEnforced</ObjectOwnership></Rule></OwnershipControls>`
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(xml),
		RawContentType: "application/xml",
	}, nil
}

func handleGetPublicAccessBlock(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><PublicAccessBlockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><BlockPublicAcls>true</BlockPublicAcls><IgnorePublicAcls>true</IgnorePublicAcls><BlockPublicPolicy>true</BlockPublicPolicy><RestrictPublicBuckets>true</RestrictPublicBuckets></PublicAccessBlockConfiguration>`
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        []byte(xml),
		RawContentType: "application/xml",
	}, nil
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
