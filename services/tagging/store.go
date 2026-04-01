package tagging

import (
	"strings"
	"sync"
)

// TagEntry represents tags for a single resource.
type TagEntry struct {
	ARN  string
	Tags map[string]string
}

// Store manages the cross-service tag registry.
type Store struct {
	mu        sync.RWMutex
	resources map[string]*TagEntry // ARN -> TagEntry
	accountID string
	region    string
}

// NewStore returns a new empty Tagging Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		resources: make(map[string]*TagEntry),
		accountID: accountID,
		region:    region,
	}
}

// TagResources adds or updates tags for the specified resources.
func (s *Store) TagResources(arns []string, tags map[string]string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	failedMap := make(map[string]string)
	for _, arn := range arns {
		entry, ok := s.resources[arn]
		if !ok {
			entry = &TagEntry{ARN: arn, Tags: make(map[string]string)}
			s.resources[arn] = entry
		}
		for k, v := range tags {
			entry.Tags[k] = v
		}
	}
	return failedMap
}

// UntagResources removes tags from the specified resources.
func (s *Store) UntagResources(arns []string, tagKeys []string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	failedMap := make(map[string]string)
	for _, arn := range arns {
		entry, ok := s.resources[arn]
		if !ok {
			continue
		}
		for _, key := range tagKeys {
			delete(entry.Tags, key)
		}
	}
	return failedMap
}

// arnMatchesResourceType checks if an ARN matches a resource type filter.
// Resource type filters use the format "service:resourcetype", e.g., "ec2:instance".
func arnMatchesResourceType(arn string, resourceType string) bool {
	// ARN format: arn:aws:service:region:account:resourcetype/id
	parts := strings.SplitN(arn, ":", 7)
	if len(parts) < 6 {
		return false
	}
	arnService := parts[2]

	// Handle "service" and "service:resourcetype" filters.
	filterParts := strings.SplitN(resourceType, ":", 2)
	filterService := filterParts[0]
	if arnService != filterService {
		return false
	}
	if len(filterParts) == 1 {
		return true // Service-only match.
	}
	// Check resource type portion.
	resourcePart := parts[5]
	if len(parts) == 7 {
		resourcePart = parts[5] + ":" + parts[6]
	}
	return strings.HasPrefix(resourcePart, filterParts[1])
}

// GetResources returns resources matching the given tag filters and resource type filters.
func (s *Store) GetResources(tagFilters []TagFilter, resourceTypeFilters []string) []*TagEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*TagEntry
	for _, entry := range s.resources {
		if len(resourceTypeFilters) > 0 {
			matched := false
			for _, rtf := range resourceTypeFilters {
				if arnMatchesResourceType(entry.ARN, rtf) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if matchesFilters(entry, tagFilters) {
			results = append(results, entry)
		}
	}
	return results
}

// GetTagKeys returns all unique tag keys.
func (s *Store) GetTagKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keySet := make(map[string]bool)
	for _, entry := range s.resources {
		for k := range entry.Tags {
			keySet[k] = true
		}
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	return keys
}

// GetTagValues returns all unique values for a given tag key.
func (s *Store) GetTagValues(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	valueSet := make(map[string]bool)
	for _, entry := range s.resources {
		if v, ok := entry.Tags[key]; ok {
			valueSet[v] = true
		}
	}

	values := make([]string, 0, len(valueSet))
	for v := range valueSet {
		values = append(values, v)
	}
	return values
}

// ResourceCount returns the total number of tagged resources.
func (s *Store) ResourceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.resources)
}

// TagFilter represents a filter for querying tagged resources.
type TagFilter struct {
	Key    string
	Values []string
}

func matchesFilters(entry *TagEntry, filters []TagFilter) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		val, ok := entry.Tags[f.Key]
		if !ok {
			return false
		}
		if len(f.Values) > 0 {
			matched := false
			for _, fv := range f.Values {
				if val == fv {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	return true
}
