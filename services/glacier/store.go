package glacier

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Vault represents a Glacier vault.
type Vault struct {
	VaultName         string
	VaultARN          string
	CreationDate      time.Time
	LastInventoryDate *time.Time
	NumberOfArchives  int64
	SizeInBytes       int64
}

// Archive represents a Glacier archive.
type Archive struct {
	ArchiveID          string
	VaultName          string
	ArchiveDescription string
	CreationDate       time.Time
	Size               int64
	SHA256TreeHash     string
}

// Job represents a Glacier job.
type Job struct {
	JobID              string
	VaultName          string
	Action             string // ArchiveRetrieval or InventoryRetrieval
	ArchiveID          string
	StatusCode         string
	StatusMessage      string
	CreationDate       time.Time
	CompletionDate     *time.Time
	ArchiveSizeInBytes int64
	SNSTopic           string
	lifecycle          *lifecycle.Machine
}

// Store manages Glacier resources in memory.
type Store struct {
	mu        sync.RWMutex
	vaults    map[string]*Vault
	archives  map[string]map[string]*Archive // vaultName -> archiveID -> Archive
	jobs      map[string]*Job
	accountID string
	region    string
	lcConfig  *lifecycle.Config
	archSeq   int
	jobSeq    int
}

// NewStore returns a new empty Glacier Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		vaults:    make(map[string]*Vault),
		archives:  make(map[string]map[string]*Archive),
		jobs:      make(map[string]*Job),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func (s *Store) vaultARN(name string) string {
	return fmt.Sprintf("arn:aws:glacier:%s:%s:vaults/%s", s.region, s.accountID, name)
}

// CreateVault creates a new vault.
func (s *Store) CreateVault(name string) (*Vault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vaults[name]; ok {
		return nil, fmt.Errorf("vault already exists: %s", name)
	}

	vault := &Vault{
		VaultName:    name,
		VaultARN:     s.vaultARN(name),
		CreationDate: time.Now().UTC(),
	}
	s.vaults[name] = vault
	s.archives[name] = make(map[string]*Archive)
	return vault, nil
}

// GetVault retrieves a vault by name.
func (s *Store) GetVault(name string) (*Vault, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.vaults[name]
	return v, ok
}

// ListVaults returns all vaults.
func (s *Store) ListVaults() []*Vault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Vault, 0, len(s.vaults))
	for _, v := range s.vaults {
		out = append(out, v)
	}
	return out
}

// DeleteVault removes a vault.
func (s *Store) DeleteVault(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.vaults[name]; !ok {
		return false
	}
	delete(s.vaults, name)
	delete(s.archives, name)
	return true
}

// UploadArchive uploads an archive to a vault.
func (s *Store) UploadArchive(vaultName, description, treeHash string, size int64) (*Archive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vault, ok := s.vaults[vaultName]
	if !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultName)
	}

	s.archSeq++
	archiveID := fmt.Sprintf("archive-%012d", s.archSeq)

	archive := &Archive{
		ArchiveID:          archiveID,
		VaultName:          vaultName,
		ArchiveDescription: description,
		CreationDate:       time.Now().UTC(),
		Size:               size,
		SHA256TreeHash:     treeHash,
	}
	s.archives[vaultName][archiveID] = archive
	vault.NumberOfArchives++
	vault.SizeInBytes += size
	now := time.Now().UTC()
	vault.LastInventoryDate = &now
	return archive, nil
}

// DeleteArchive removes an archive.
func (s *Store) DeleteArchive(vaultName, archiveID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	arch, ok := s.archives[vaultName][archiveID]
	if !ok {
		return fmt.Errorf("archive not found: %s", archiveID)
	}
	vault.NumberOfArchives--
	vault.SizeInBytes -= arch.Size
	delete(s.archives[vaultName], archiveID)
	return nil
}

// InitiateJob starts a new job.
func (s *Store) InitiateJob(vaultName, action, archiveID, snsTopic string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vaults[vaultName]; !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultName)
	}

	s.jobSeq++
	jobID := fmt.Sprintf("job-%012d", s.jobSeq)

	transitions := []lifecycle.Transition{
		{From: "InProgress", To: "Succeeded", Delay: 3 * time.Second},
	}

	job := &Job{
		JobID:         jobID,
		VaultName:     vaultName,
		Action:        action,
		ArchiveID:     archiveID,
		StatusCode:    "InProgress",
		StatusMessage: "In progress",
		CreationDate:  time.Now().UTC(),
		SNSTopic:      snsTopic,
	}
	job.lifecycle = lifecycle.NewMachine("InProgress", transitions, s.lcConfig)
	job.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		job.StatusCode = string(to)
		job.StatusMessage = string(to)
		now := time.Now().UTC()
		job.CompletionDate = &now
	})

	s.jobs[jobID] = job
	return job, nil
}

// GetJob retrieves a job by ID.
func (s *Store) GetJob(vaultName, jobID string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok || job.VaultName != vaultName {
		return nil, false
	}
	return job, true
}

// ListJobs returns all jobs for a vault.
func (s *Store) ListJobs(vaultName string) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0)
	for _, job := range s.jobs {
		if job.VaultName == vaultName {
			out = append(out, job)
		}
	}
	return out
}
