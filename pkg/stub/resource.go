package stub

import (
	"fmt"
	"strings"
	"sync"
)

// ResourceStore is a thread-safe in-memory store for generic AWS resources.
type ResourceStore struct {
	mu        sync.RWMutex
	resources map[string]map[string]map[string]interface{} // resourceType -> id -> fields
	tags      map[string]map[string]string                  // resourceArn -> tags
	counters  map[string]int                                // for generating sequential IDs
}

// NewResourceStore creates an empty ResourceStore.
func NewResourceStore() *ResourceStore {
	return &ResourceStore{
		resources: make(map[string]map[string]map[string]interface{}),
		tags:      make(map[string]map[string]string),
		counters:  make(map[string]int),
	}
}

// nextID generates a sequential hex ID for the given resource type prefix.
// For example, prefix "vpc" produces "vpc-00000001", "vpc-00000002", etc.
func (s *ResourceStore) nextID(prefix string) string {
	s.counters[prefix]++
	return fmt.Sprintf("%s-%08x", prefix, s.counters[prefix])
}

// Create stores a new resource and returns the generated ID.
// The idPrefix is used for ID generation (e.g., "vpc" produces "vpc-00000001").
func (s *ResourceStore) Create(resourceType, idPrefix string, fields map[string]interface{}) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID(idPrefix)
	if s.resources[resourceType] == nil {
		s.resources[resourceType] = make(map[string]map[string]interface{})
	}
	stored := make(map[string]interface{}, len(fields)+1)
	for k, v := range fields {
		stored[k] = v
	}
	s.resources[resourceType][id] = stored
	return id
}

// Get returns the fields for a resource, or an error if it does not exist.
func (s *ResourceStore) Get(resourceType, id string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byType := s.resources[resourceType]
	if byType == nil {
		return nil, fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	fields, ok := byType[id]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	// Return a copy
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		out[k] = v
	}
	return out, nil
}

// Delete removes a resource. Returns an error if not found.
func (s *ResourceStore) Delete(resourceType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byType := s.resources[resourceType]
	if byType == nil {
		return fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	if _, ok := byType[id]; !ok {
		return fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	delete(byType, id)
	return nil
}

// List returns all resources of the given type.
func (s *ResourceStore) List(resourceType string) []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byType := s.resources[resourceType]
	result := make([]map[string]interface{}, 0, len(byType))
	for _, fields := range byType {
		cp := make(map[string]interface{}, len(fields))
		for k, v := range fields {
			cp[k] = v
		}
		result = append(result, cp)
	}
	return result
}

// Update merges the given updates into an existing resource's fields.
func (s *ResourceStore) Update(resourceType, id string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	byType := s.resources[resourceType]
	if byType == nil {
		return fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	fields, ok := byType[id]
	if !ok {
		return fmt.Errorf("resource not found: %s/%s", resourceType, id)
	}
	for k, v := range updates {
		fields[k] = v
	}
	return nil
}

// Tag adds or overwrites tags on a resource identified by its ARN.
func (s *ResourceStore) Tag(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

// Untag removes the specified tag keys from a resource identified by its ARN.
func (s *ResourceStore) Untag(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tagMap := s.tags[arn]
	if tagMap == nil {
		return
	}
	for _, k := range keys {
		delete(tagMap, k)
	}
}

// ListTags returns the tags for a resource identified by its ARN.
func (s *ResourceStore) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagMap := s.tags[arn]
	out := make(map[string]string, len(tagMap))
	for k, v := range tagMap {
		out[k] = v
	}
	return out
}

// BuildARN generates an ARN from a pattern by substituting {region}, {account}, and {id}.
func BuildARN(pattern, region, account, id string) string {
	r := strings.NewReplacer("{region}", region, "{account}", account, "{id}", id)
	return r.Replace(pattern)
}
