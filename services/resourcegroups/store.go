package resourcegroups

import (
	"fmt"
	"sync"
	"time"
)

// Group represents a Resource Groups group.
type Group struct {
	GroupArn    string
	Name        string
	Description string
	ResourceQuery *ResourceQuery
	Tags        map[string]string
	Resources   []string // resource ARNs
	CreatedAt   time.Time
}

// ResourceQuery describes how to find group members.
type ResourceQuery struct {
	Type  string // TAG_FILTERS_1_0, CLOUDFORMATION_STACK_1_0
	Query string // JSON query string
}

// Store manages Resource Groups resources in memory.
type Store struct {
	mu        sync.RWMutex
	groups    map[string]*Group // name -> Group
	accountID string
	region    string
}

// NewStore returns a new empty Resource Groups Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		groups:    make(map[string]*Group),
		accountID: accountID,
		region:    region,
	}
}

// CreateGroup creates a new resource group.
func (s *Store) CreateGroup(name, description string, query *ResourceQuery, tags map[string]string) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[name]; ok {
		return nil, fmt.Errorf("group already exists: %s", name)
	}

	group := &Group{
		GroupArn:      fmt.Sprintf("arn:aws:resource-groups:%s:%s:group/%s", s.region, s.accountID, name),
		Name:          name,
		Description:   description,
		ResourceQuery: query,
		Tags:          tags,
		CreatedAt:     time.Now().UTC(),
	}
	s.groups[name] = group
	return group, nil
}

// GetGroup retrieves a group by name.
func (s *Store) GetGroup(name string) (*Group, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[name]
	return g, ok
}

// ListGroups returns all groups.
func (s *Store) ListGroups() []*Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		out = append(out, g)
	}
	return out
}

// UpdateGroup updates a group's description.
func (s *Store) UpdateGroup(name, description string) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	if description != "" {
		g.Description = description
	}
	return g, nil
}

// DeleteGroup removes a group.
func (s *Store) DeleteGroup(name string) (*Group, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, false
	}
	delete(s.groups, name)
	return g, true
}

// GroupResources adds resources to a group.
func (s *Store) GroupResources(name string, arns []string) ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, arns
	}

	existing := make(map[string]bool)
	for _, a := range g.Resources {
		existing[a] = true
	}

	var succeeded, failed []string
	for _, arn := range arns {
		if existing[arn] {
			failed = append(failed, arn)
		} else {
			g.Resources = append(g.Resources, arn)
			existing[arn] = true
			succeeded = append(succeeded, arn)
		}
	}
	return succeeded, failed
}

// UngroupResources removes resources from a group.
func (s *Store) UngroupResources(name string, arns []string) ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, arns
	}

	removeSet := make(map[string]bool)
	for _, a := range arns {
		removeSet[a] = true
	}

	var succeeded []string
	var remaining []string
	for _, a := range g.Resources {
		if removeSet[a] {
			succeeded = append(succeeded, a)
		} else {
			remaining = append(remaining, a)
		}
	}
	g.Resources = remaining

	// Find failed (not in group)
	successSet := make(map[string]bool)
	for _, a := range succeeded {
		successSet[a] = true
	}
	var failed []string
	for _, a := range arns {
		if !successSet[a] {
			failed = append(failed, a)
		}
	}
	return succeeded, failed
}

// ListGroupResources returns resources in a group.
func (s *Store) ListGroupResources(name string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[name]
	if !ok {
		return nil, false
	}
	result := make([]string, len(g.Resources))
	copy(result, g.Resources)
	return result, true
}

// GetTags returns tags for a group.
func (s *Store) GetTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.groups {
		if g.GroupArn == arn {
			tags := make(map[string]string)
			for k, v := range g.Tags {
				tags[k] = v
			}
			return tags, true
		}
	}
	return nil, false
}

// TagResource adds tags to a group.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.groups {
		if g.GroupArn == arn {
			if g.Tags == nil {
				g.Tags = make(map[string]string)
			}
			for k, v := range tags {
				g.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tags from a group.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.groups {
		if g.GroupArn == arn {
			for _, k := range keys {
				delete(g.Tags, k)
			}
			return true
		}
	}
	return false
}
