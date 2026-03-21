package ecr

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Image holds the data for a container image stored in a repository.
type Image struct {
	Digest   string // sha256:xxx
	Tag      string
	Manifest string
	PushedAt time.Time
}

// Repository holds all metadata and images for an ECR repository.
type Repository struct {
	Name                   string
	ARN                    string
	RegistryId             string
	URI                    string
	CreatedAt              time.Time
	ImageTagMutability     string
	ImageScanningConfig    imageScanningConfiguration
	Tags                   map[string]string
	Images                 []*Image
}

type imageScanningConfiguration struct {
	ScanOnPush bool
}

// Store is the in-memory store for ECR repositories and images.
type Store struct {
	mu        sync.RWMutex
	repos     map[string]*Repository // keyed by repository name
	accountID string
	region    string
}

// NewStore creates an empty ECR Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		repos:     make(map[string]*Repository),
		accountID: accountID,
		region:    region,
	}
}

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// digestManifest returns a sha256 digest string for the given manifest.
func digestManifest(manifest string) string {
	sum := sha256.Sum256([]byte(manifest))
	return fmt.Sprintf("sha256:%x", sum)
}

// buildARN constructs an ARN for a repository.
func (s *Store) buildARN(repoName string) string {
	return fmt.Sprintf("arn:aws:ecr:%s:%s:repository/%s", s.region, s.accountID, repoName)
}

// buildURI constructs the repository URI.
func (s *Store) buildURI(repoName string) string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", s.accountID, s.region, repoName)
}

// CreateRepository adds a new repository to the store.
func (s *Store) CreateRepository(name, imageTagMutability string, scanOnPush bool, tags map[string]string) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repos[name]; ok {
		return nil, service.NewAWSError("RepositoryAlreadyExistsException",
			fmt.Sprintf("The repository with name '%s' already exists in the registry with id '%s'", name, s.accountID),
			http.StatusConflict)
	}

	if imageTagMutability == "" {
		imageTagMutability = "MUTABLE"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	repo := &Repository{
		Name:       name,
		ARN:        s.buildARN(name),
		RegistryId: s.accountID,
		URI:        s.buildURI(name),
		CreatedAt:  time.Now().UTC(),
		ImageTagMutability: imageTagMutability,
		ImageScanningConfig: imageScanningConfiguration{ScanOnPush: scanOnPush},
		Tags:   tags,
		Images: []*Image{},
	}
	s.repos[name] = repo
	return repo, nil
}

// GetRepository retrieves a repository by name.
func (s *Store) GetRepository(name string) (*Repository, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.repos[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with name '%s' does not exist in the registry with id '%s'", name, s.accountID),
			http.StatusBadRequest)
	}
	return repo, nil
}

// DeleteRepository removes a repository from the store.
// If force is false and the repository has images, an error is returned.
func (s *Store) DeleteRepository(name string, force bool) (*Repository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repos[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with name '%s' does not exist in the registry with id '%s'", name, s.accountID),
			http.StatusBadRequest)
	}

	if !force && len(repo.Images) > 0 {
		return nil, service.NewAWSError("RepositoryNotEmptyException",
			fmt.Sprintf("The repository with name '%s' is not empty.", name),
			http.StatusConflict)
	}

	delete(s.repos, name)
	return repo, nil
}

// ListRepositories returns all repositories, optionally filtered by names.
func (s *Store) ListRepositories(names []string) ([]*Repository, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		// Return all.
		out := make([]*Repository, 0, len(s.repos))
		for _, r := range s.repos {
			out = append(out, r)
		}
		return out, nil
	}

	// Return specified names.
	out := make([]*Repository, 0, len(names))
	for _, name := range names {
		repo, ok := s.repos[name]
		if !ok {
			return nil, service.NewAWSError("RepositoryNotFoundException",
				fmt.Sprintf("The repository with name '%s' does not exist in the registry with id '%s'", name, s.accountID),
				http.StatusBadRequest)
		}
		out = append(out, repo)
	}
	return out, nil
}

// PutImage adds an image to a repository. Returns the image and any error.
func (s *Store) PutImage(repoName, manifest, imageTag string) (*Image, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repos[repoName]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with name '%s' does not exist in the registry with id '%s'", repoName, s.accountID),
			http.StatusBadRequest)
	}

	digest := digestManifest(manifest)

	// If IMMUTABLE and tag already exists, reject.
	if repo.ImageTagMutability == "IMMUTABLE" && imageTag != "" {
		for _, img := range repo.Images {
			if img.Tag == imageTag {
				return nil, service.NewAWSError("ImageTagAlreadyExistsException",
					fmt.Sprintf("The image tag '%s' already exists in the repository.", imageTag),
					http.StatusConflict)
			}
		}
	}

	// Find existing image with same digest — update tag if needed.
	for _, img := range repo.Images {
		if img.Digest == digest {
			if imageTag != "" {
				img.Tag = imageTag
			}
			return img, nil
		}
	}

	// New image.
	img := &Image{
		Digest:   digest,
		Tag:      imageTag,
		Manifest: manifest,
		PushedAt: time.Now().UTC(),
	}
	repo.Images = append(repo.Images, img)
	return img, nil
}

// ListImages returns all imageIds in a repository.
func (s *Store) ListImages(repoName string) ([]*Image, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.repos[repoName]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with name '%s' does not exist in the registry with id '%s'", repoName, s.accountID),
			http.StatusBadRequest)
	}

	out := make([]*Image, len(repo.Images))
	copy(out, repo.Images)
	return out, nil
}

// imageIDRef describes an image by digest and/or tag.
type imageIDRef struct {
	Digest string
	Tag    string
}

// matchImage returns true if the image matches the given ref.
func matchImage(img *Image, ref imageIDRef) bool {
	if ref.Digest != "" && img.Digest == ref.Digest {
		return true
	}
	if ref.Tag != "" && img.Tag == ref.Tag {
		return true
	}
	return false
}

// BatchGetImage retrieves full image records by imageIds.
func (s *Store) BatchGetImage(repoName string, refs []imageIDRef) ([]*Image, []imageFailure) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, ok := s.repos[repoName]
	if !ok {
		// Return all as failures.
		failures := make([]imageFailure, 0, len(refs))
		for _, ref := range refs {
			failures = append(failures, imageFailure{
				ImageID:     ref,
				FailureCode: "RepositoryNotFoundException",
				FailureReason: fmt.Sprintf("The repository with name '%s' does not exist.", repoName),
			})
		}
		return nil, failures
	}

	found := make([]*Image, 0)
	failures := make([]imageFailure, 0)

	for _, ref := range refs {
		var matched *Image
		for _, img := range repo.Images {
			if matchImage(img, ref) {
				matched = img
				break
			}
		}
		if matched != nil {
			found = append(found, matched)
		} else {
			failures = append(failures, imageFailure{
				ImageID:       ref,
				FailureCode:   "ImageNotFoundException",
				FailureReason: "The image requested does not exist in the specified repository.",
			})
		}
	}

	return found, failures
}

// BatchDeleteImage removes images from a repository.
func (s *Store) BatchDeleteImage(repoName string, refs []imageIDRef) ([]*Image, []imageFailure) {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo, ok := s.repos[repoName]
	if !ok {
		failures := make([]imageFailure, 0, len(refs))
		for _, ref := range refs {
			failures = append(failures, imageFailure{
				ImageID:       ref,
				FailureCode:   "RepositoryNotFoundException",
				FailureReason: fmt.Sprintf("The repository with name '%s' does not exist.", repoName),
			})
		}
		return nil, failures
	}

	deleted := make([]*Image, 0)
	failures := make([]imageFailure, 0)

	for _, ref := range refs {
		found := false
		newImages := make([]*Image, 0, len(repo.Images))
		for _, img := range repo.Images {
			if matchImage(img, ref) {
				found = true
				deleted = append(deleted, img)
			} else {
				newImages = append(newImages, img)
			}
		}
		if !found {
			failures = append(failures, imageFailure{
				ImageID:       ref,
				FailureCode:   "ImageNotFoundException",
				FailureReason: "The image requested does not exist in the specified repository.",
			})
		} else {
			repo.Images = newImages
		}
	}

	return deleted, failures
}

// TagResource adds or replaces tags on a repository.
func (s *Store) TagResource(repoARN string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo := s.repoByARNLocked(repoARN)
	if repo == nil {
		return service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with ARN '%s' does not exist.", repoARN),
			http.StatusBadRequest)
	}
	if repo.Tags == nil {
		repo.Tags = make(map[string]string)
	}
	for k, v := range tags {
		repo.Tags[k] = v
	}
	return nil
}

// UntagResource removes tags from a repository.
func (s *Store) UntagResource(repoARN string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	repo := s.repoByARNLocked(repoARN)
	if repo == nil {
		return service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with ARN '%s' does not exist.", repoARN),
			http.StatusBadRequest)
	}
	for _, k := range tagKeys {
		delete(repo.Tags, k)
	}
	return nil
}

// ListTagsForResource returns tags for a repository by ARN.
func (s *Store) ListTagsForResource(repoARN string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repo := s.repoByARNLocked(repoARN)
	if repo == nil {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			fmt.Sprintf("The repository with ARN '%s' does not exist.", repoARN),
			http.StatusBadRequest)
	}
	// Return a copy of the tags.
	out := make(map[string]string, len(repo.Tags))
	for k, v := range repo.Tags {
		out[k] = v
	}
	return out, nil
}

// repoByARNLocked finds a repository by ARN (caller must hold at least read lock).
func (s *Store) repoByARNLocked(arn string) *Repository {
	for _, r := range s.repos {
		if r.ARN == arn {
			return r
		}
	}
	return nil
}

// imageFailure describes a failed image operation.
type imageFailure struct {
	ImageID       imageIDRef
	FailureCode   string
	FailureReason string
}

