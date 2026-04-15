package ecrpublic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StoredRepository models an ECR Public repository.
type StoredRepository struct {
	Name        string
	Arn         string
	RegistryID  string
	URI         string
	CreatedAt   time.Time
	CatalogData map[string]any
	Policy      string
	Tags        map[string]string
	Images      map[string]*StoredImage
}

// StoredImage models a single image manifest.
type StoredImage struct {
	Digest      string
	Tags        []string
	Manifest    string
	MediaType   string
	PushedAt    time.Time
	SizeInBytes int64
}

// StoredLayerUpload tracks an in-progress layer upload.
type StoredLayerUpload struct {
	RepositoryName string
	UploadID       string
	PartSize       int64
	Layers         [][]byte
	StartedAt      time.Time
}

// Store holds all ECR Public state.
type Store struct {
	mu           sync.RWMutex
	accountID    string
	region       string
	repositories map[string]*StoredRepository
	uploads      map[string]*StoredLayerUpload
	registryData map[string]any
	resourceTags map[string]map[string]string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:    accountID,
		region:       region,
		repositories: make(map[string]*StoredRepository),
		uploads:      make(map[string]*StoredLayerUpload),
		registryData: map[string]any{"displayName": "cloudmock registry"},
		resourceTags: make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repositories = make(map[string]*StoredRepository)
	s.uploads = make(map[string]*StoredLayerUpload)
	s.registryData = map[string]any{"displayName": "cloudmock registry"}
	s.resourceTags = make(map[string]map[string]string)
}

// ── Repositories ────────────────────────────────────────────────────────────

func (s *Store) CreateRepository(name string, catalog map[string]any, tags map[string]string) (*StoredRepository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repositories[name]; ok {
		return nil, service.NewAWSError("RepositoryAlreadyExistsException",
			"Repository already exists: "+name, 400)
	}
	arn := fmt.Sprintf("arn:aws:ecr-public::%s:repository/%s", s.accountID, name)
	repo := &StoredRepository{
		Name:        name,
		Arn:         arn,
		RegistryID:  s.accountID,
		URI:         fmt.Sprintf("public.ecr.aws/%s/%s", s.accountID, name),
		CreatedAt:   time.Now().UTC(),
		CatalogData: catalog,
		Tags:        copyStringMap(tags),
		Images:      make(map[string]*StoredImage),
	}
	s.repositories[name] = repo
	return repo, nil
}

func (s *Store) GetRepository(name string) (*StoredRepository, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+name, 404)
	}
	return repo, nil
}

func (s *Store) DeleteRepository(name string, force bool) (*StoredRepository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+name, 404)
	}
	if !force && len(repo.Images) > 0 {
		return nil, service.NewAWSError("RepositoryNotEmptyException",
			"Repository not empty: "+name, 400)
	}
	delete(s.repositories, name)
	return repo, nil
}

func (s *Store) ListRepositories() []*StoredRepository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredRepository, 0, len(s.repositories))
	for _, r := range s.repositories {
		out = append(out, r)
	}
	return out
}

func (s *Store) SetRepositoryCatalog(name string, catalog map[string]any) (*StoredRepository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+name, 404)
	}
	repo.CatalogData = catalog
	return repo, nil
}

func (s *Store) SetRepositoryPolicy(name, policy string) (*StoredRepository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+name, 404)
	}
	repo.Policy = policy
	return repo, nil
}

func (s *Store) DeleteRepositoryPolicy(name string) (*StoredRepository, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo, ok := s.repositories[name]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+name, 404)
	}
	repo.Policy = ""
	return repo, nil
}

// ── Images ──────────────────────────────────────────────────────────────────

func (s *Store) PutImage(repoName, manifest, mediaType, tag string) (*StoredImage, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo, ok := s.repositories[repoName]
	if !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+repoName, 404)
	}
	sum := sha256.Sum256([]byte(manifest))
	digest := "sha256:" + hex.EncodeToString(sum[:])
	img, exists := repo.Images[digest]
	if !exists {
		img = &StoredImage{
			Digest:      digest,
			Manifest:    manifest,
			MediaType:   mediaType,
			PushedAt:    time.Now().UTC(),
			SizeInBytes: int64(len(manifest)),
		}
		repo.Images[digest] = img
	}
	if tag != "" {
		for _, other := range repo.Images {
			other.Tags = removeTag(other.Tags, tag)
		}
		img.Tags = append(img.Tags, tag)
	}
	return img, nil
}

func (s *Store) BatchDeleteImage(repoName string, imageIDs []map[string]string) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := make([]map[string]any, 0, len(imageIDs))
	repo, ok := s.repositories[repoName]
	if !ok {
		return results
	}
	for _, id := range imageIDs {
		digest := id["imageDigest"]
		tag := id["imageTag"]
		var target *StoredImage
		if digest != "" {
			target = repo.Images[digest]
		}
		if target == nil && tag != "" {
			for _, img := range repo.Images {
				if containsStr(img.Tags, tag) {
					target = img
					break
				}
			}
		}
		if target == nil {
			results = append(results, map[string]any{
				"imageId": map[string]any{"imageDigest": digest, "imageTag": tag},
				"reason":  "ImageNotFound",
			})
			continue
		}
		if tag != "" {
			target.Tags = removeTag(target.Tags, tag)
			if len(target.Tags) == 0 {
				delete(repo.Images, target.Digest)
			}
		} else {
			delete(repo.Images, target.Digest)
		}
	}
	return results
}

func (s *Store) ListImages(repoName string) []*StoredImage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repo, ok := s.repositories[repoName]
	if !ok {
		return nil
	}
	out := make([]*StoredImage, 0, len(repo.Images))
	for _, img := range repo.Images {
		out = append(out, img)
	}
	return out
}

// ── Layer uploads ───────────────────────────────────────────────────────────

func (s *Store) InitiateUpload(repoName string) (*StoredLayerUpload, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repositories[repoName]; !ok {
		return nil, service.NewAWSError("RepositoryNotFoundException",
			"Repository not found: "+repoName, 404)
	}
	id := newUUID()
	u := &StoredLayerUpload{
		RepositoryName: repoName,
		UploadID:       id,
		PartSize:       8 * 1024 * 1024,
		StartedAt:      time.Now().UTC(),
	}
	s.uploads[id] = u
	return u, nil
}

func (s *Store) UploadPart(uploadID string, data []byte) (*StoredLayerUpload, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.uploads[uploadID]
	if !ok {
		return nil, service.NewAWSError("UploadNotFoundException",
			"Upload not found: "+uploadID, 404)
	}
	u.Layers = append(u.Layers, data)
	return u, nil
}

func (s *Store) CompleteUpload(uploadID string) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.uploads[uploadID]
	if !ok {
		return "", service.NewAWSError("UploadNotFoundException",
			"Upload not found: "+uploadID, 404)
	}
	h := sha256.New()
	for _, part := range u.Layers {
		h.Write(part)
	}
	digest := "sha256:" + hex.EncodeToString(h.Sum(nil))
	delete(s.uploads, uploadID)
	return digest, nil
}

func (s *Store) CheckLayerAvailability(digests []string) []map[string]any {
	out := make([]map[string]any, 0, len(digests))
	for _, d := range digests {
		out = append(out, map[string]any{
			"layerDigest":       d,
			"layerAvailability": "AVAILABLE",
		})
	}
	return out
}

// ── Registry catalog ─────────────────────────────────────────────────────────

func (s *Store) GetRegistryCatalog() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyAnyMap(s.registryData)
}

func (s *Store) PutRegistryCatalog(data map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registryData = data
	return copyAnyMap(s.registryData)
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.resourceTags[arn][k] = v
	}
}

func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.resourceTags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.resourceTags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newUUID() string {
	b := make([]byte, 16)
	n := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(n >> (i * 4))
	}
	return hex.EncodeToString(b)
}

func removeTag(tags []string, tag string) []string {
	out := tags[:0]
	for _, t := range tags {
		if t != tag {
			out = append(out, t)
		}
	}
	return out
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyAnyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
