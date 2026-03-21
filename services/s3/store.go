package s3

import (
	"crypto/md5" //nolint:gosec // MD5 is used for ETags per the S3 specification, not security
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Bucket holds metadata for a single S3 bucket plus its object store.
type Bucket struct {
	Name         string
	CreationDate time.Time
	Objects      *ObjectStore
}

// Store is an in-memory store for S3 buckets and multipart uploads.
type Store struct {
	mu               sync.RWMutex
	buckets          map[string]*Bucket
	multipartUploads map[string]*MultipartUpload // uploadId -> upload
	nextUploadID     int
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		buckets:          make(map[string]*Bucket),
		multipartUploads: make(map[string]*MultipartUpload),
	}
}

// CreateBucket creates a bucket with the given name.
// Returns an AWSError if the bucket already exists.
func (s *Store) CreateBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[name]; ok {
		return service.NewAWSError("BucketAlreadyExists",
			"The requested bucket name is not available.", http.StatusConflict)
	}
	s.buckets[name] = &Bucket{
		Name:         name,
		CreationDate: time.Now().UTC(),
		Objects:      NewObjectStore(),
	}
	return nil
}

// DeleteBucket removes the bucket with the given name.
// Returns an AWSError if the bucket does not exist or is not empty.
func (s *Store) DeleteBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[name]
	if !ok {
		return service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	if b.Objects.Len() > 0 {
		return service.NewAWSError("BucketNotEmpty",
			"The bucket you tried to delete is not empty.", http.StatusConflict)
	}
	delete(s.buckets, name)
	return nil
}

// HeadBucket returns an error if the bucket does not exist.
func (s *Store) HeadBucket(name string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.buckets[name]; !ok {
		return service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	return nil
}

// ListBuckets returns all buckets in the store.
func (s *Store) ListBuckets() []*Bucket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Bucket, 0, len(s.buckets))
	for _, b := range s.buckets {
		out = append(out, b)
	}
	return out
}

// bucketObjects returns the ObjectStore for the named bucket, or an AWSError
// if the bucket does not exist.
func (s *Store) bucketObjects(name string) (*ObjectStore, error) {
	s.mu.RLock()
	b, ok := s.buckets[name]
	s.mu.RUnlock()
	if !ok {
		return nil, service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	return b.Objects, nil
}

// generateUploadID creates a unique upload ID.
func (s *Store) generateUploadID() string {
	s.nextUploadID++
	return fmt.Sprintf("upload-%d-%x", s.nextUploadID, time.Now().UnixNano())
}

// CreateMultipartUpload starts a new multipart upload and returns it.
func (s *Store) CreateMultipartUpload(bucket, key string) *MultipartUpload {
	s.mu.Lock()
	defer s.mu.Unlock()

	uploadId := s.generateUploadID()
	upload := &MultipartUpload{
		UploadId:  uploadId,
		Bucket:    bucket,
		Key:       key,
		Parts:     make(map[int]*Part),
		CreatedAt: time.Now().UTC(),
	}
	s.multipartUploads[uploadId] = upload
	return upload
}

// GetMultipartUpload returns the multipart upload for the given ID, or an error if not found.
func (s *Store) GetMultipartUpload(uploadId string) (*MultipartUpload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, ok := s.multipartUploads[uploadId]
	if !ok {
		return nil, service.NewAWSError("NoSuchUpload",
			"The specified multipart upload does not exist.", http.StatusNotFound)
	}
	return upload, nil
}

// UploadPart stores a part in the given multipart upload.
func (s *Store) UploadPart(uploadId string, partNumber int, body []byte) (*Part, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.multipartUploads[uploadId]
	if !ok {
		return nil, service.NewAWSError("NoSuchUpload",
			"The specified multipart upload does not exist.", http.StatusNotFound)
	}

	sum := md5.Sum(body) //nolint:gosec
	etag := fmt.Sprintf(`"%x"`, sum)

	part := &Part{
		PartNumber: partNumber,
		Body:       body,
		ETag:       etag,
		Size:       int64(len(body)),
	}
	upload.Parts[partNumber] = part
	return part, nil
}

// DeleteMultipartUpload removes a completed multipart upload.
func (s *Store) DeleteMultipartUpload(uploadId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.multipartUploads, uploadId)
}

// AbortMultipartUpload cancels and removes a multipart upload and all its parts.
func (s *Store) AbortMultipartUpload(uploadId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.multipartUploads[uploadId]; !ok {
		return service.NewAWSError("NoSuchUpload",
			"The specified multipart upload does not exist.", http.StatusNotFound)
	}
	delete(s.multipartUploads, uploadId)
	return nil
}

// ListMultipartUploads returns all pending multipart uploads for the given bucket.
func (s *Store) ListMultipartUploads(bucket string) []*MultipartUpload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*MultipartUpload
	for _, u := range s.multipartUploads {
		if u.Bucket == bucket {
			out = append(out, u)
		}
	}
	return out
}
