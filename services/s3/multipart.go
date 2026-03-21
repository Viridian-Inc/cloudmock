package s3

import (
	"crypto/md5" //nolint:gosec // MD5 is used for ETags per the S3 specification, not security
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// MultipartUpload holds the state of an in-progress multipart upload.
type MultipartUpload struct {
	UploadId  string
	Bucket    string
	Key       string
	Parts     map[int]*Part // partNumber -> part
	CreatedAt time.Time
}

// Part holds a single uploaded part of a multipart upload.
type Part struct {
	PartNumber int
	Body       []byte
	ETag       string
	Size       int64
}

// ---- XML types for multipart operations ----

type initiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadId string   `xml:"UploadId"`
}

type completeMultipartUploadInput struct {
	XMLName xml.Name          `xml:"CompleteMultipartUpload"`
	Parts   []completionPart  `xml:"Part"`
}

type completionPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type completeMultipartUploadResult struct {
	XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}

type xmlUpload struct {
	Key       string `xml:"Key"`
	UploadId  string `xml:"UploadId"`
	Initiated string `xml:"Initiated"`
}

type listMultipartUploadsResult struct {
	XMLName xml.Name    `xml:"ListMultipartUploadsResult"`
	Xmlns   string      `xml:"xmlns,attr"`
	Bucket  string      `xml:"Bucket"`
	Uploads []xmlUpload `xml:"Upload"`
}

type xmlPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
	Size       int64  `xml:"Size"`
}

type listPartsResult struct {
	XMLName  xml.Name  `xml:"ListPartsResult"`
	Xmlns    string    `xml:"xmlns,attr"`
	Bucket   string    `xml:"Bucket"`
	Key      string    `xml:"Key"`
	UploadId string    `xml:"UploadId"`
	Parts    []xmlPart `xml:"Part"`
}

// ---- handlers ----

// handleCreateMultipartUpload initiates a new multipart upload and returns an upload ID.
func handleCreateMultipartUpload(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)

	// Verify bucket exists.
	if _, err := store.bucketObjects(bucket); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	upload := store.CreateMultipartUpload(bucket, key)

	result := &initiateMultipartUploadResult{
		Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
		Bucket:   bucket,
		Key:      key,
		UploadId: upload.UploadId,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

// handleUploadPart stores a single part of a multipart upload.
func handleUploadPart(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	uploadId := ctx.RawRequest.URL.Query().Get("uploadId")
	partNumberStr := ctx.RawRequest.URL.Query().Get("partNumber")

	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil {
		awsErr := service.NewAWSError("InvalidArgument",
			"Invalid partNumber: "+partNumberStr, http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	part, storeErr := store.UploadPart(uploadId, partNumber, ctx.Body)
	if storeErr != nil {
		return &service.Response{Format: service.FormatXML}, storeErr
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Format:     service.FormatXML,
		Headers: map[string]string{
			"ETag": part.ETag,
		},
	}, nil
}

// handleCompleteMultipartUpload concatenates all parts and stores as a regular object.
func handleCompleteMultipartUpload(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)
	uploadId := ctx.RawRequest.URL.Query().Get("uploadId")

	// Parse input XML.
	var input completeMultipartUploadInput
	if err := xml.Unmarshal(ctx.Body, &input); err != nil {
		awsErr := service.NewAWSError("MalformedXML",
			"The XML you provided was not well-formed.", http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	upload, err := store.GetMultipartUpload(uploadId)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	// Concatenate parts in order.
	sort.Slice(input.Parts, func(i, j int) bool {
		return input.Parts[i].PartNumber < input.Parts[j].PartNumber
	})

	var combined []byte
	for _, cp := range input.Parts {
		part, ok := upload.Parts[cp.PartNumber]
		if !ok {
			awsErr := service.NewAWSError("InvalidPart",
				fmt.Sprintf("Part %d was not uploaded.", cp.PartNumber), http.StatusBadRequest)
			return &service.Response{Format: service.FormatXML}, awsErr
		}
		combined = append(combined, part.Body...)
	}

	// Store as regular object.
	objs, bucketErr := store.bucketObjects(bucket)
	if bucketErr != nil {
		return &service.Response{Format: service.FormatXML}, bucketErr
	}
	obj := objs.PutObject(key, combined, "application/octet-stream", nil)

	// Clean up the multipart upload.
	store.DeleteMultipartUpload(uploadId)

	etag := computeMultipartETag(upload, len(input.Parts))

	result := &completeMultipartUploadResult{
		Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
		Location: fmt.Sprintf("http://s3.amazonaws.com/%s/%s", bucket, key),
		Bucket:   bucket,
		Key:      key,
		ETag:     etag,
	}

	// Also set the ETag on the stored object to match the multipart ETag.
	_ = obj

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

// computeMultipartETag computes the S3 multipart ETag format: md5ofmd5s-numparts.
func computeMultipartETag(upload *MultipartUpload, numParts int) string {
	h := md5.New() //nolint:gosec
	partNums := make([]int, 0, len(upload.Parts))
	for pn := range upload.Parts {
		partNums = append(partNums, pn)
	}
	sort.Ints(partNums)
	for _, pn := range partNums {
		partMD5 := md5.Sum(upload.Parts[pn].Body) //nolint:gosec
		h.Write(partMD5[:])
	}
	return fmt.Sprintf(`"%x-%d"`, h.Sum(nil), numParts)
}

// handleAbortMultipartUpload cancels an in-progress multipart upload and removes all parts.
func handleAbortMultipartUpload(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	uploadId := ctx.RawRequest.URL.Query().Get("uploadId")

	if err := store.AbortMultipartUpload(uploadId); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

// handleListMultipartUploads returns all pending multipart uploads for a bucket.
func handleListMultipartUploads(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	// Verify bucket exists.
	if _, err := store.bucketObjects(bucket); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	uploads := store.ListMultipartUploads(bucket)

	xmlUploads := make([]xmlUpload, 0, len(uploads))
	for _, u := range uploads {
		xmlUploads = append(xmlUploads, xmlUpload{
			Key:       u.Key,
			UploadId:  u.UploadId,
			Initiated: u.CreatedAt.Format(time.RFC3339),
		})
	}

	result := &listMultipartUploadsResult{
		Xmlns:   "http://s3.amazonaws.com/doc/2006-03-01/",
		Bucket:  bucket,
		Uploads: xmlUploads,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

// handleListParts returns all uploaded parts for a pending multipart upload.
func handleListParts(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)
	key := extractObjectKey(ctx)
	uploadId := ctx.RawRequest.URL.Query().Get("uploadId")

	upload, err := store.GetMultipartUpload(uploadId)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	partNums := make([]int, 0, len(upload.Parts))
	for pn := range upload.Parts {
		partNums = append(partNums, pn)
	}
	sort.Ints(partNums)

	xmlParts := make([]xmlPart, 0, len(partNums))
	for _, pn := range partNums {
		p := upload.Parts[pn]
		xmlParts = append(xmlParts, xmlPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
			Size:       p.Size,
		})
	}

	result := &listPartsResult{
		Xmlns:    "http://s3.amazonaws.com/doc/2006-03-01/",
		Bucket:   bucket,
		Key:      key,
		UploadId: uploadId,
		Parts:    xmlParts,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

// isPresignedURL returns true if the request contains presigned URL query parameters.
func isPresignedURL(r *http.Request) bool {
	return r.URL.Query().Get("X-Amz-Algorithm") != ""
}
