package lakeformation

import (
	"sync"
	"time"
)

// DataLakeSettings holds the data lake configuration.
type DataLakeSettings struct {
	DataLakeAdmins                []DataLakePrincipal
	CreateDatabaseDefaultPermissions []PrincipalPermissions
	CreateTableDefaultPermissions    []PrincipalPermissions
}

// DataLakePrincipal identifies a principal.
type DataLakePrincipal struct {
	DataLakePrincipalIdentifier string
}

// PrincipalPermissions maps a principal to their permissions.
type PrincipalPermissions struct {
	Principal   DataLakePrincipal
	Permissions []string
}

// Resource represents a registered data lake resource.
type Resource struct {
	ResourceArn string
	RoleArn     string
	RegisteredTime time.Time
}

// Permission represents a granted permission.
type Permission struct {
	Principal   DataLakePrincipal
	Resource    PermissionResource
	Permissions []string
	PermissionsWithGrantOption []string
}

// PermissionResource describes the resource a permission applies to.
type PermissionResource struct {
	Database *DatabaseResource
	Table    *TableResource
}

// DatabaseResource identifies a database.
type DatabaseResource struct {
	Name      string
	CatalogId string
}

// TableResource identifies a table.
type TableResource struct {
	DatabaseName string
	Name         string
	CatalogId    string
}

// LFTag represents a Lake Formation tag.
type LFTag struct {
	TagKey    string
	TagValues []string
}

// Store manages all Lake Formation resources.
type Store struct {
	mu          sync.RWMutex
	settings    DataLakeSettings
	resources   map[string]*Resource // arn -> resource
	permissions []*Permission
	lfTags      map[string]*LFTag
	resourceTags map[string][]LFTagPair // resourceKey -> tags
	accountID   string
	region      string
}

// LFTagPair holds a tag key-value association on a resource.
type LFTagPair struct {
	TagKey    string
	TagValues []string
}

// NewStore creates a new Lake Formation store.
func NewStore(accountID, region string) *Store {
	return &Store{
		settings: DataLakeSettings{
			DataLakeAdmins: make([]DataLakePrincipal, 0),
		},
		resources:    make(map[string]*Resource),
		permissions:  make([]*Permission, 0),
		lfTags:       make(map[string]*LFTag),
		resourceTags: make(map[string][]LFTagPair),
		accountID:    accountID,
		region:       region,
	}
}

// ---- Data Lake Settings ----

func (s *Store) GetDataLakeSettings() DataLakeSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) PutDataLakeSettings(settings DataLakeSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
}

// ---- Resource operations ----

func (s *Store) RegisterResource(arn, roleArn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if arn == "" {
		return false
	}
	if _, ok := s.resources[arn]; ok {
		return false
	}
	s.resources[arn] = &Resource{ResourceArn: arn, RoleArn: roleArn, RegisteredTime: time.Now().UTC()}
	return true
}

func (s *Store) DeregisterResource(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resources[arn]; !ok {
		return false
	}
	delete(s.resources, arn)
	return true
}

func (s *Store) ListResources() []*Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Resource, 0, len(s.resources))
	for _, r := range s.resources {
		result = append(result, r)
	}
	return result
}

// ---- Permission operations ----

func (s *Store) GrantPermissions(perm *Permission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = append(s.permissions, perm)
}

func (s *Store) RevokePermissions(principal string, resource PermissionResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]*Permission, 0)
	for _, p := range s.permissions {
		if p.Principal.DataLakePrincipalIdentifier == principal && matchResource(p.Resource, resource) {
			continue
		}
		filtered = append(filtered, p)
	}
	s.permissions = filtered
}

func matchResource(a, b PermissionResource) bool {
	if a.Database != nil && b.Database != nil {
		return a.Database.Name == b.Database.Name
	}
	if a.Table != nil && b.Table != nil {
		return a.Table.DatabaseName == b.Table.DatabaseName && a.Table.Name == b.Table.Name
	}
	return false
}

func (s *Store) ListPermissions(principal string) []*Permission {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Permission, 0)
	for _, p := range s.permissions {
		if principal == "" || p.Principal.DataLakePrincipalIdentifier == principal {
			result = append(result, p)
		}
	}
	return result
}

func (s *Store) GetEffectivePermissionsForPath(resourceArn string) []*Permission {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return all permissions as effective for mock purposes.
	return s.permissions
}

// BatchGrantPermissions grants permissions for multiple entries.
func (s *Store) BatchGrantPermissions(entries []*Permission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = append(s.permissions, entries...)
}

// BatchRevokePermissions revokes permissions for multiple entries.
func (s *Store) BatchRevokePermissions(entries []*Permission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range entries {
		filtered := make([]*Permission, 0)
		for _, p := range s.permissions {
			if p.Principal.DataLakePrincipalIdentifier == entry.Principal.DataLakePrincipalIdentifier &&
				matchResource(p.Resource, entry.Resource) {
				continue
			}
			filtered = append(filtered, p)
		}
		s.permissions = filtered
	}
}

// DescribeResource returns resource info by ARN.
func (s *Store) DescribeResource(arn string) (*Resource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.resources[arn]
	return r, ok
}

// ---- LFTag operations ----

func (s *Store) CreateLFTag(key string, values []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lfTags[key]; ok {
		return false
	}
	s.lfTags[key] = &LFTag{TagKey: key, TagValues: values}
	return true
}

func (s *Store) GetLFTag(key string) (*LFTag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.lfTags[key]
	return t, ok
}

func (s *Store) ListLFTags() []*LFTag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*LFTag, 0, len(s.lfTags))
	for _, t := range s.lfTags {
		result = append(result, t)
	}
	return result
}

func (s *Store) UpdateLFTag(key string, addValues, removeValues []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.lfTags[key]
	if !ok {
		return false
	}
	// Remove values
	removeSet := make(map[string]bool)
	for _, v := range removeValues {
		removeSet[v] = true
	}
	filtered := make([]string, 0)
	for _, v := range t.TagValues {
		if !removeSet[v] {
			filtered = append(filtered, v)
		}
	}
	// Add values
	filtered = append(filtered, addValues...)
	t.TagValues = filtered
	return true
}

func (s *Store) DeleteLFTag(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lfTags[key]; !ok {
		return false
	}
	delete(s.lfTags, key)
	return true
}

// ---- Resource LFTag operations ----

func (s *Store) AddLFTagsToResource(resourceKey string, tags []LFTagPair) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resourceTags[resourceKey] = append(s.resourceTags[resourceKey], tags...)
}

func (s *Store) RemoveLFTagsFromResource(resourceKey string, tagKeys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	removeSet := make(map[string]bool)
	for _, k := range tagKeys {
		removeSet[k] = true
	}
	filtered := make([]LFTagPair, 0)
	for _, t := range s.resourceTags[resourceKey] {
		if !removeSet[t.TagKey] {
			filtered = append(filtered, t)
		}
	}
	s.resourceTags[resourceKey] = filtered
}

func (s *Store) GetResourceLFTags(resourceKey string) []LFTagPair {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resourceTags[resourceKey]
}
