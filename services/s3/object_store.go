package s3

import (
	"crypto/md5" //nolint:gosec // MD5 is used for ETags per the S3 specification, not security
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Object holds the data and metadata for a single S3 object.
type Object struct {
	Key          string
	Body         []byte
	ContentType  string
	ETag         string // MD5 hex of body, wrapped in quotes per S3 convention
	Size         int64
	LastModified time.Time
	Metadata     map[string]string
}

// ObjectStore is an in-memory store for objects within a single S3 bucket.
type ObjectStore struct {
	mu      sync.RWMutex
	objects map[string]*Object // key → object
}

// NewObjectStore returns an empty ObjectStore.
func NewObjectStore() *ObjectStore {
	return &ObjectStore{
		objects: make(map[string]*Object),
	}
}

// computeETag returns a quoted MD5 hex string for the given data.
func computeETag(data []byte) string {
	sum := md5.Sum(data) //nolint:gosec
	return fmt.Sprintf(`"%x"`, sum)
}

// PutObject stores an object under the given key, replacing any existing value.
func (os *ObjectStore) PutObject(key string, body []byte, contentType string, metadata map[string]string) *Object {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	obj := &Object{
		Key:          key,
		Body:         body,
		ContentType:  contentType,
		ETag:         computeETag(body),
		Size:         int64(len(body)),
		LastModified: time.Now().UTC(),
		Metadata:     metadata,
	}
	os.mu.Lock()
	os.objects[key] = obj
	os.mu.Unlock()
	return obj
}

// GetObject returns the object for the given key, or an AWSError if not found.
func (os *ObjectStore) GetObject(key string) (*Object, error) {
	os.mu.RLock()
	obj, ok := os.objects[key]
	os.mu.RUnlock()
	if !ok {
		return nil, service.NewAWSError("NoSuchKey",
			"The specified key does not exist.", http.StatusNotFound)
	}
	return obj, nil
}

// DeleteObject removes the object for the given key. It is a no-op if the key
// does not exist (S3 DELETE is idempotent).
func (os *ObjectStore) DeleteObject(key string) {
	os.mu.Lock()
	delete(os.objects, key)
	os.mu.Unlock()
}

// HeadObject returns the object metadata for the given key, or an AWSError if
// not found.
func (os *ObjectStore) HeadObject(key string) (*Object, error) {
	return os.GetObject(key)
}

// ListObjectsOutput holds the result of a ListObjects call.
type ListObjectsOutput struct {
	Objects               []*Object
	CommonPrefixes        []string
	IsTruncated           bool
	NextContinuationToken string
}

// Len returns the number of objects currently in the store (lock-free for
// bucket-level empty checks — callers must hold the bucket lock or accept a
// snapshot).
func (os *ObjectStore) Len() int {
	os.mu.RLock()
	n := len(os.objects)
	os.mu.RUnlock()
	return n
}

// ListObjects returns objects matching the given parameters.
//
//   - prefix: only keys with this prefix are returned
//   - delimiter: collapse keys that share a sub-prefix after the prefix into CommonPrefixes
//   - maxKeys: maximum number of keys to return (0 → default 1000)
//   - continuationToken: resume listing after this key (exclusive)
func (os *ObjectStore) ListObjects(prefix, delimiter string, maxKeys int, continuationToken string) *ListObjectsOutput {
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	os.mu.RLock()
	keys := make([]string, 0, len(os.objects))
	for k := range os.objects {
		keys = append(keys, k)
	}
	os.mu.RUnlock()

	sort.Strings(keys)

	out := &ListObjectsOutput{}
	commonPrefixSet := make(map[string]struct{})
	count := 0

	for _, k := range keys {
		// Apply continuation token (start after this key).
		if continuationToken != "" && k <= continuationToken {
			continue
		}

		// Apply prefix filter.
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}

		// Apply delimiter grouping.
		if delimiter != "" {
			// Find the delimiter after the prefix portion.
			rest := strings.TrimPrefix(k, prefix)
			idx := strings.Index(rest, delimiter)
			if idx >= 0 {
				cp := prefix + rest[:idx+len(delimiter)]
				if _, seen := commonPrefixSet[cp]; !seen {
					commonPrefixSet[cp] = struct{}{}
					out.CommonPrefixes = append(out.CommonPrefixes, cp)
					count++
					if count >= maxKeys {
						out.IsTruncated = true
						// Use the key that caused truncation as next token.
						out.NextContinuationToken = k
						break
					}
				}
				continue
			}
		}

		if count >= maxKeys {
			out.IsTruncated = true
			out.NextContinuationToken = k
			break
		}

		os.mu.RLock()
		obj := os.objects[k]
		os.mu.RUnlock()
		if obj != nil {
			out.Objects = append(out.Objects, obj)
		}
		count++
	}

	sort.Strings(out.CommonPrefixes)
	return out
}
