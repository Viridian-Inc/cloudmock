package s3

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Bucket holds metadata for a single S3 bucket plus its object store.
type Bucket struct {
	Name             string
	CreationDate     time.Time
	Objects          *ObjectStore
	VersioningStatus string // "", "Enabled", or "Suspended"
	Policy           []byte // raw JSON policy, nil if unset
	Configs          map[string][]byte            // bucket-level configs: "tagging", "cors", "lifecycle", etc.
	ObjectConfigs    map[string]map[string][]byte // objectKey -> configType -> data
}

// Store is an in-memory store for S3 buckets and multipart uploads.
type Store struct {
	buckets          sync.Map                    // string -> *Bucket
	mu               sync.Mutex                  // only for multipart uploads
	multipartUploads map[string]*MultipartUpload // uploadId -> upload
	nextUploadID     int
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		multipartUploads: make(map[string]*MultipartUpload),
	}
}

// CreateBucket creates a bucket with the given name.
// Returns an AWSError if the bucket already exists.
func (s *Store) CreateBucket(name string) error {
	newBucket := &Bucket{
		Name:          name,
		CreationDate:  time.Now().UTC(),
		Objects:       NewObjectStore(),
		Configs:       make(map[string][]byte),
		ObjectConfigs: make(map[string]map[string][]byte),
	}
	_, loaded := s.buckets.LoadOrStore(name, newBucket)
	if loaded {
		return service.NewAWSError("BucketAlreadyExists",
			"The requested bucket name is not available.", http.StatusConflict)
	}
	return nil
}

// DeleteBucket removes the bucket with the given name.
// Returns an AWSError if the bucket does not exist or is not empty.
func (s *Store) DeleteBucket(name string) error {
	value, ok := s.buckets.Load(name)
	if !ok {
		return service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	b := value.(*Bucket)
	if b.Objects.Len() > 0 {
		return service.NewAWSError("BucketNotEmpty",
			"The bucket you tried to delete is not empty.", http.StatusConflict)
	}
	s.buckets.Delete(name)
	return nil
}

// HeadBucket returns an error if the bucket does not exist.
func (s *Store) HeadBucket(name string) error {
	if _, ok := s.buckets.Load(name); !ok {
		return service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	return nil
}

// ListBuckets returns all buckets in the store.
func (s *Store) ListBuckets() []*Bucket {
	var out []*Bucket
	s.buckets.Range(func(key, value any) bool {
		out = append(out, value.(*Bucket))
		return true
	})
	return out
}

// bucketObjects returns the ObjectStore for the named bucket, or an AWSError
// if the bucket does not exist.
func (s *Store) bucketObjects(name string) (*ObjectStore, error) {
	value, ok := s.buckets.Load(name)
	if !ok {
		return nil, service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	return value.(*Bucket).Objects, nil
}

// getBucket returns the bucket for the given name, or an AWSError if not found.
func (s *Store) getBucket(name string) (*Bucket, error) {
	value, ok := s.buckets.Load(name)
	if !ok {
		return nil, service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
	}
	return value.(*Bucket), nil
}

// SetVersioning sets the versioning status for the named bucket.
func (s *Store) SetVersioning(name, status string) error {
	b, err := s.getBucket(name)
	if err != nil {
		return err
	}
	b.VersioningStatus = status
	return nil
}

// GetVersioning returns the versioning status for the named bucket.
func (s *Store) GetVersioning(name string) (string, error) {
	b, err := s.getBucket(name)
	if err != nil {
		return "", err
	}
	return b.VersioningStatus, nil
}

// SetBucketPolicy stores a raw JSON policy for the named bucket.
func (s *Store) SetBucketPolicy(name string, policy []byte) error {
	b, err := s.getBucket(name)
	if err != nil {
		return err
	}
	b.Policy = policy
	return nil
}

// GetBucketPolicy returns the raw JSON policy for the named bucket.
func (s *Store) GetBucketPolicy(name string) ([]byte, error) {
	b, err := s.getBucket(name)
	if err != nil {
		return nil, err
	}
	if b.Policy == nil {
		return nil, service.NewAWSError("NoSuchBucketPolicy",
			"The bucket policy does not exist.", http.StatusNotFound)
	}
	return b.Policy, nil
}

// DeleteBucketPolicy removes the policy from the named bucket.
func (s *Store) DeleteBucketPolicy(name string) error {
	b, err := s.getBucket(name)
	if err != nil {
		return err
	}
	b.Policy = nil
	return nil
}

// IsVersioningEnabled returns true if the named bucket has versioning enabled.
func (s *Store) IsVersioningEnabled(name string) bool {
	b, err := s.getBucket(name)
	if err != nil {
		return false
	}
	return b.VersioningStatus == "Enabled"
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
	s.mu.Lock()
	defer s.mu.Unlock()

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

	etag := computeETag(body)

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

// ── Bucket & Object Config Storage ────────────────────────────────────────────

func (s *Store) setBucketConfig(bucket, configType string, data []byte) {
	b, err := s.getBucket(bucket)
	if err != nil {
		return
	}
	if b.Configs == nil {
		b.Configs = make(map[string][]byte)
	}
	b.Configs[configType] = data
}

func (s *Store) getBucketConfig(bucket, configType string) []byte {
	b, err := s.getBucket(bucket)
	if err != nil {
		return nil
	}
	return b.Configs[configType]
}

func (s *Store) deleteBucketConfig(bucket, configType string) {
	b, err := s.getBucket(bucket)
	if err != nil {
		return
	}
	delete(b.Configs, configType)
}

func (s *Store) setObjectConfig(bucket, key, configType string, data []byte) {
	b, err := s.getBucket(bucket)
	if err != nil {
		return
	}
	if b.ObjectConfigs == nil {
		b.ObjectConfigs = make(map[string]map[string][]byte)
	}
	if b.ObjectConfigs[key] == nil {
		b.ObjectConfigs[key] = make(map[string][]byte)
	}
	b.ObjectConfigs[key][configType] = data
}

func (s *Store) getObjectConfig(bucket, key, configType string) []byte {
	b, err := s.getBucket(bucket)
	if err != nil {
		return nil
	}
	if b.ObjectConfigs != nil {
		return b.ObjectConfigs[key][configType]
	}
	return nil
}

func (s *Store) deleteObjectConfig(bucket, key, configType string) {
	b, err := s.getBucket(bucket)
	if err != nil {
		return
	}
	if b.ObjectConfigs != nil {
		delete(b.ObjectConfigs[key], configType)
	}
}

// ListMultipartUploads returns all pending multipart uploads for the given bucket.
func (s *Store) ListMultipartUploads(bucket string) []*MultipartUpload {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []*MultipartUpload
	for _, u := range s.multipartUploads {
		if u.Bucket == bucket {
			out = append(out, u)
		}
	}
	return out
}
