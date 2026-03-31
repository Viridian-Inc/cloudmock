package codeartifact

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Domain represents a CodeArtifact domain.
type Domain struct {
	Name              string
	ARN               string
	Owner             string
	Status            string
	EncryptionKey     string
	RepositoryCount   int
	AssetSizeBytes    int64
	CreatedTime       time.Time
	Tags              map[string]string
}

// Repository represents a CodeArtifact repository.
type Repository struct {
	Name             string
	ARN              string
	DomainName       string
	DomainOwner      string
	Description      string
	Upstreams        []UpstreamRepo
	ExternalConnections []ExternalConnection
	CreatedTime      time.Time
	Tags             map[string]string
}

// UpstreamRepo describes an upstream repository reference.
type UpstreamRepo struct {
	RepositoryName string
}

// ExternalConnection describes an external package connection.
type ExternalConnection struct {
	ExternalConnectionName string
	PackageFormat          string
	Status                 string
}

// Package represents a package in a repository.
type Package struct {
	Format      string
	Namespace   string
	PackageName string
	OriginConfig *PackageOriginConfig
}

// PackageOriginConfig describes package origin restrictions.
type PackageOriginConfig struct {
	Restrictions *PackageOriginRestrictions
}

// PackageOriginRestrictions describes publish/upstream restrictions.
type PackageOriginRestrictions struct {
	Publish  string
	Upstream string
}

// PackageVersion represents a specific version of a package.
type PackageVersion struct {
	Version     string
	Status      string
	Revision    string
	DisplayName string
	Summary     string
	HomePage    string
	PublishedTime time.Time
}

// Store is the in-memory store for all CodeArtifact resources.
type Store struct {
	mu           sync.RWMutex
	accountID    string
	region       string
	domains      map[string]*Domain
	repositories map[string]map[string]*Repository // domainName -> repoName -> Repository
	packages     map[string]map[string]*Package    // domainName/repoName -> format/namespace/name -> Package
	versions     map[string][]*PackageVersion      // domainName/repoName/packageID -> versions
	tags         map[string]map[string]string
}

// NewStore creates an empty CodeArtifact store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:    accountID,
		region:       region,
		domains:      make(map[string]*Domain),
		repositories: make(map[string]map[string]*Repository),
		packages:     make(map[string]map[string]*Package),
		versions:     make(map[string][]*PackageVersion),
		tags:         make(map[string]map[string]string),
	}
}

// ---- ARN builders ----

func (s *Store) domainARN(name string) string {
	return fmt.Sprintf("arn:aws:codeartifact:%s:%s:domain/%s", s.region, s.accountID, name)
}

func (s *Store) repositoryARN(domainName, repoName string) string {
	return fmt.Sprintf("arn:aws:codeartifact:%s:%s:repository/%s/%s", s.region, s.accountID, domainName, repoName)
}

func packageKey(domainName, repoName string) string {
	return domainName + "/" + repoName
}

func packageID(format, namespace, name string) string {
	if namespace != "" {
		return format + "/" + namespace + "/" + name
	}
	return format + "/" + name
}

// ---- Domain operations ----

func (s *Store) CreateDomain(name, encryptionKey string, tags map[string]string) (*Domain, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Domain name is required.")
	}
	if _, exists := s.domains[name]; exists {
		return nil, service.NewAWSError("ConflictException",
			fmt.Sprintf("Domain already exists: %s", name), http.StatusConflict)
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	if encryptionKey == "" {
		encryptionKey = fmt.Sprintf("arn:aws:kms:%s:%s:key/default", s.region, s.accountID)
	}

	domain := &Domain{
		Name:          name,
		ARN:           s.domainARN(name),
		Owner:         s.accountID,
		Status:        "Active",
		EncryptionKey: encryptionKey,
		CreatedTime:   time.Now().UTC(),
		Tags:          tags,
	}
	s.domains[name] = domain
	return domain, nil
}

func (s *Store) DescribeDomain(name string) (*Domain, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domain, ok := s.domains[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Domain not found: %s", name), http.StatusNotFound)
	}
	return domain, nil
}

func (s *Store) ListDomains() []*Domain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Domain, 0, len(s.domains))
	for _, d := range s.domains {
		result = append(result, d)
	}
	return result
}

func (s *Store) DeleteDomain(name string) (*Domain, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain, ok := s.domains[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Domain not found: %s", name), http.StatusNotFound)
	}

	// Check for repositories
	if repos := s.repositories[name]; len(repos) > 0 {
		return nil, service.NewAWSError("ConflictException",
			"Domain has repositories and cannot be deleted.", http.StatusConflict)
	}

	delete(s.domains, name)
	return domain, nil
}

// ---- Repository operations ----

func (s *Store) CreateRepository(domainName, repoName, description string, upstreams []UpstreamRepo, tags map[string]string) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.domains[domainName]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Domain not found: %s", domainName), http.StatusNotFound)
	}

	if repoName == "" {
		return nil, service.ErrValidation("Repository name is required.")
	}

	if s.repositories[domainName] == nil {
		s.repositories[domainName] = make(map[string]*Repository)
	}

	if _, exists := s.repositories[domainName][repoName]; exists {
		return nil, service.NewAWSError("ConflictException",
			fmt.Sprintf("Repository already exists: %s", repoName), http.StatusConflict)
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	repo := &Repository{
		Name:        repoName,
		ARN:         s.repositoryARN(domainName, repoName),
		DomainName:  domainName,
		DomainOwner: s.accountID,
		Description: description,
		Upstreams:   upstreams,
		CreatedTime: time.Now().UTC(),
		Tags:        tags,
	}
	s.repositories[domainName][repoName] = repo

	// Update domain repository count
	s.domains[domainName].RepositoryCount++

	return repo, nil
}

func (s *Store) DescribeRepository(domainName, repoName string) (*Repository, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := s.repositories[domainName]
	if repos == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Repository not found: %s/%s", domainName, repoName), http.StatusNotFound)
	}

	repo, ok := repos[repoName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Repository not found: %s/%s", domainName, repoName), http.StatusNotFound)
	}
	return repo, nil
}

func (s *Store) ListRepositories(domainName string) []*Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if domainName != "" {
		repos := s.repositories[domainName]
		result := make([]*Repository, 0, len(repos))
		for _, r := range repos {
			result = append(result, r)
		}
		return result
	}

	// List all repos across all domains
	var result []*Repository
	for _, repos := range s.repositories {
		for _, r := range repos {
			result = append(result, r)
		}
	}
	return result
}

func (s *Store) DeleteRepository(domainName, repoName string) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repos := s.repositories[domainName]
	if repos == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Repository not found: %s/%s", domainName, repoName), http.StatusNotFound)
	}

	repo, ok := repos[repoName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Repository not found: %s/%s", domainName, repoName), http.StatusNotFound)
	}

	delete(repos, repoName)
	if d, ok := s.domains[domainName]; ok {
		d.RepositoryCount--
	}
	return repo, nil
}

// ---- Package operations ----

func (s *Store) DescribePackage(domainName, repoName, format, namespace, pkgName string) (*Package, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := packageKey(domainName, repoName)
	pkgs := s.packages[key]
	if pkgs == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Package not found: %s", pkgName), http.StatusNotFound)
	}

	pid := packageID(format, namespace, pkgName)
	pkg, ok := pkgs[pid]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Package not found: %s", pkgName), http.StatusNotFound)
	}
	return pkg, nil
}

func (s *Store) ListPackages(domainName, repoName, format, namespace string) []*Package {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := packageKey(domainName, repoName)
	pkgs := s.packages[key]
	var result []*Package
	for _, pkg := range pkgs {
		if format != "" && pkg.Format != format {
			continue
		}
		if namespace != "" && pkg.Namespace != namespace {
			continue
		}
		result = append(result, pkg)
	}
	return result
}

func (s *Store) ListPackageVersions(domainName, repoName, format, namespace, pkgName string) ([]*PackageVersion, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := packageKey(domainName, repoName)
	pid := packageID(format, namespace, pkgName)
	vKey := key + "/" + pid

	versions := s.versions[vKey]
	if versions == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Package not found: %s", pkgName), http.StatusNotFound)
	}
	return versions, nil
}

func (s *Store) DescribePackageVersion(domainName, repoName, format, namespace, pkgName, version string) (*PackageVersion, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := packageKey(domainName, repoName)
	pid := packageID(format, namespace, pkgName)
	vKey := key + "/" + pid

	versions := s.versions[vKey]
	for _, v := range versions {
		if v.Version == version {
			return v, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Package version not found: %s@%s", pkgName, version), http.StatusNotFound)
}

// EnsurePackage creates a package and version if they don't exist (used for testing).
func (s *Store) EnsurePackage(domainName, repoName, format, namespace, pkgName, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := packageKey(domainName, repoName)
	pid := packageID(format, namespace, pkgName)

	if s.packages[key] == nil {
		s.packages[key] = make(map[string]*Package)
	}
	if _, ok := s.packages[key][pid]; !ok {
		s.packages[key][pid] = &Package{
			Format:      format,
			Namespace:   namespace,
			PackageName: pkgName,
		}
	}

	vKey := key + "/" + pid

	// Check if version already exists
	for _, v := range s.versions[vKey] {
		if v.Version == version {
			return
		}
	}

	s.versions[vKey] = append(s.versions[vKey], &PackageVersion{
		Version:       version,
		Status:        "Published",
		Revision:      newUUID(),
		PublishedTime: time.Now().UTC(),
	})
}

// ---- Endpoint & Auth ----

func (s *Store) GetRepositoryEndpoint(domainName, repoName, format string) (string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := s.repositories[domainName]
	if repos == nil || repos[repoName] == nil {
		return "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Repository not found: %s/%s", domainName, repoName), http.StatusNotFound)
	}

	endpoint := fmt.Sprintf("https://%s-%s.d.codeartifact.%s.amazonaws.com/%s/%s/",
		domainName, s.accountID, s.region, format, repoName)
	return endpoint, nil
}

func (s *Store) GetAuthorizationToken(domainName string) (string, time.Time, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.domains[domainName]; !ok {
		return "", time.Time{}, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Domain not found: %s", domainName), http.StatusNotFound)
	}

	token := newToken()
	expiration := time.Now().UTC().Add(12 * time.Hour)
	return token, expiration, nil
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.tags[arn]
	if m == nil {
		return nil
	}
	for _, k := range keys {
		delete(m, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.tags[arn]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
