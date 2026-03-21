package s3

import (
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Bucket holds metadata for a single S3 bucket.
type Bucket struct {
	Name         string
	CreationDate time.Time
}

// Store is an in-memory store for S3 buckets.
type Store struct {
	mu      sync.RWMutex
	buckets map[string]*Bucket
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		buckets: make(map[string]*Bucket),
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
	}
	return nil
}

// DeleteBucket removes the bucket with the given name.
// Returns an AWSError if the bucket does not exist.
func (s *Store) DeleteBucket(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.buckets[name]; !ok {
		return service.NewAWSError("NoSuchBucket",
			"The specified bucket does not exist.", http.StatusNotFound)
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
