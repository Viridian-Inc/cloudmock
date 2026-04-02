package s3

import (
	"crypto/md5"  //nolint:gosec // MD5 is used for ETags per the S3 specification, not security
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	VersionId    string // empty when versioning is not enabled
}

// ObjectVersion holds a single version of an object (including delete markers).
type ObjectVersion struct {
	VersionId      string
	Key            string
	Body           []byte
	ContentType    string
	ETag           string
	Size           int64
	LastModified   time.Time
	IsLatest       bool
	IsDeleteMarker bool
}

// ObjectStore is an in-memory store for objects within a single S3 bucket.
type ObjectStore struct {
	mu       sync.RWMutex
	objects  map[string]*Object           // key → current (latest non-deleted) object
	versions map[string][]*ObjectVersion  // key → ordered list of versions (index 0 = newest)
}

// NewObjectStore returns an empty ObjectStore.
func NewObjectStore() *ObjectStore {
	return &ObjectStore{
		objects:  make(map[string]*Object),
		versions: make(map[string][]*ObjectVersion),
	}
}

// computeETag returns a quoted MD5 hex string for the given data.
func computeETag(data []byte) string {
	sum := md5.Sum(data) //nolint:gosec
	// Avoid fmt.Sprintf: pre-allocate for `"` + 32 hex chars + `"` = 34 bytes.
	buf := make([]byte, 0, 34)
	buf = append(buf, '"')
	buf = append(buf, hex.EncodeToString(sum[:])...)
	buf = append(buf, '"')
	return string(buf)
}

// PutObject stores an object under the given key, replacing any existing value.
// This is the non-versioned path (used when versioning is off).
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

// PutObjectVersioned stores a new version of an object and returns it.
// The new version becomes the latest; previous versions are preserved.
func (os *ObjectStore) PutObjectVersioned(key string, body []byte, contentType string, metadata map[string]string) *Object {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	versionId := uuid.New().String()
	now := time.Now().UTC()
	etag := computeETag(body)

	obj := &Object{
		Key:          key,
		Body:         body,
		ContentType:  contentType,
		ETag:         etag,
		Size:         int64(len(body)),
		LastModified: now,
		Metadata:     metadata,
		VersionId:    versionId,
	}

	ver := &ObjectVersion{
		VersionId:    versionId,
		Key:          key,
		Body:         body,
		ContentType:  contentType,
		ETag:         etag,
		Size:         int64(len(body)),
		LastModified: now,
		IsLatest:     true,
	}

	os.mu.Lock()
	// Mark all existing versions as not latest.
	for _, v := range os.versions[key] {
		v.IsLatest = false
	}
	// Prepend new version (index 0 = newest).
	os.versions[key] = append([]*ObjectVersion{ver}, os.versions[key]...)
	// Update the current object reference.
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

// GetObjectVersion returns a specific version of an object.
func (os *ObjectStore) GetObjectVersion(key, versionId string) (*Object, error) {
	os.mu.RLock()
	defer os.mu.RUnlock()

	versions, ok := os.versions[key]
	if !ok {
		return nil, service.NewAWSError("NoSuchKey",
			"The specified key does not exist.", http.StatusNotFound)
	}
	for _, v := range versions {
		if v.VersionId == versionId {
			if v.IsDeleteMarker {
				return nil, service.NewAWSError("NoSuchKey",
					"The specified key does not exist.", http.StatusNotFound)
			}
			return &Object{
				Key:          v.Key,
				Body:         v.Body,
				ContentType:  v.ContentType,
				ETag:         v.ETag,
				Size:         v.Size,
				LastModified: v.LastModified,
				VersionId:    v.VersionId,
			}, nil
		}
	}
	return nil, service.NewAWSError("NoSuchVersion",
		"The specified version does not exist.", http.StatusNotFound)
}

// DeleteObject removes the object for the given key. It is a no-op if the key
// does not exist (S3 DELETE is idempotent).
func (os *ObjectStore) DeleteObject(key string) {
	os.mu.Lock()
	delete(os.objects, key)
	os.mu.Unlock()
}

// DeleteObjectVersioned creates a delete marker instead of actually removing data.
// Returns the versionId of the delete marker.
func (os *ObjectStore) DeleteObjectVersioned(key string) string {
	versionId := uuid.New().String()
	now := time.Now().UTC()

	ver := &ObjectVersion{
		VersionId:      versionId,
		Key:            key,
		LastModified:   now,
		IsLatest:       true,
		IsDeleteMarker: true,
	}

	os.mu.Lock()
	// Mark all existing versions as not latest.
	for _, v := range os.versions[key] {
		v.IsLatest = false
	}
	os.versions[key] = append([]*ObjectVersion{ver}, os.versions[key]...)
	// Remove from current objects so GetObject returns 404.
	delete(os.objects, key)
	os.mu.Unlock()

	return versionId
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

// VersionCount returns the total number of versions across all keys.
func (os *ObjectStore) VersionCount() int {
	os.mu.RLock()
	total := 0
	for _, vs := range os.versions {
		total += len(vs)
	}
	os.mu.RUnlock()
	return total
}

// ListObjectVersions returns all versions and delete markers for objects matching the prefix.
func (os *ObjectStore) ListObjectVersions(prefix string) []*ObjectVersion {
	os.mu.RLock()
	defer os.mu.RUnlock()

	var all []*ObjectVersion
	for key, versions := range os.versions {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		all = append(all, versions...)
	}

	// Sort by key then by LastModified descending.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Key != all[j].Key {
			return all[i].Key < all[j].Key
		}
		return all[i].LastModified.After(all[j].LastModified)
	})

	return all
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

	// Acquire the lock once for the entire operation: collect keys and object
	// references in a single critical section instead of locking per-object.
	os.mu.RLock()
	keys := make([]string, 0, len(os.objects))
	for k := range os.objects {
		keys = append(keys, k)
	}
	// Sort while still holding the read lock so the map snapshot is consistent.
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

		// Direct map access — no per-object lock needed.
		obj := os.objects[k]
		if obj != nil {
			out.Objects = append(out.Objects, obj)
		}
		count++
	}
	os.mu.RUnlock()

	sort.Strings(out.CommonPrefixes)
	return out
}
