package glacier

import (
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// VaultLock represents a vault lock policy.
type VaultLock struct {
	Policy    string
	State     string // InProgress, Locked
	LockID    string
	CreatedAt time.Time
}

// VaultNotification represents vault notification configuration.
type VaultNotification struct {
	SNSTopic string
	Events   []string
}

// Vault represents a Glacier vault.
type Vault struct {
	VaultName         string
	VaultARN          string
	CreationDate      time.Time
	LastInventoryDate *time.Time
	NumberOfArchives  int64
	SizeInBytes       int64
	Lock              *VaultLock
	Notification      *VaultNotification
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
	tags      map[string]map[string]string // vaultARN -> tags
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
		tags:      make(map[string]map[string]string),
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
	s.tags[vault.VaultARN] = make(map[string]string)
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

// DeleteVault removes a vault. Returns false, reason if it fails.
func (s *Store) DeleteVault(name string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[name]
	if !ok {
		return false, "not_found"
	}
	if vault.NumberOfArchives > 0 {
		return false, "not_empty"
	}
	delete(s.vaults, name)
	delete(s.archives, name)
	return true, ""
}

// InitiateVaultLock starts the vault lock process (step 1 of 2).
func (s *Store) InitiateVaultLock(vaultName, policy string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return "", fmt.Errorf("vault not found: %s", vaultName)
	}
	if vault.Lock != nil && vault.Lock.State == "Locked" {
		return "", fmt.Errorf("vault is already locked: %s", vaultName)
	}
	s.jobSeq++
	lockID := fmt.Sprintf("lock-%012d", s.jobSeq)
	vault.Lock = &VaultLock{
		Policy:    policy,
		State:     "InProgress",
		LockID:    lockID,
		CreatedAt: time.Now().UTC(),
	}
	return lockID, nil
}

// CompleteVaultLock completes the vault lock (step 2 of 2).
func (s *Store) CompleteVaultLock(vaultName, lockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	if vault.Lock == nil || vault.Lock.State != "InProgress" {
		return fmt.Errorf("no in-progress vault lock for: %s", vaultName)
	}
	if vault.Lock.LockID != lockID {
		return fmt.Errorf("lock ID does not match")
	}
	vault.Lock.State = "Locked"
	return nil
}

// SetVaultNotifications sets notification config on a vault.
func (s *Store) SetVaultNotifications(vaultName, snsTopic string, events []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	vault.Notification = &VaultNotification{SNSTopic: snsTopic, Events: events}
	return nil
}

// GetVaultNotifications returns vault notification config.
func (s *Store) GetVaultNotifications(vaultName string) (*VaultNotification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultName)
	}
	if vault.Notification == nil {
		return nil, fmt.Errorf("no notification configuration for vault: %s", vaultName)
	}
	return vault.Notification, nil
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

// AbortVaultLock cancels an in-progress vault lock.
func (s *Store) AbortVaultLock(vaultName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	if vault.Lock == nil {
		return fmt.Errorf("no vault lock in progress for vault: %s", vaultName)
	}
	if vault.Lock.State == "Locked" {
		return fmt.Errorf("vault lock is already completed and cannot be aborted")
	}
	vault.Lock = nil
	return nil
}

// GetVaultLock retrieves the vault lock for a vault.
func (s *Store) GetVaultLock(vaultName string) (*VaultLock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultName)
	}
	if vault.Lock == nil {
		return nil, fmt.Errorf("no vault lock for vault: %s", vaultName)
	}
	return vault.Lock, nil
}

// GetJobOutput returns mock output for a completed job.
func (s *Store) GetJobOutput(vaultName, jobID string) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok || job.VaultName != vaultName {
		return nil, "", fmt.Errorf("job not found: %s", jobID)
	}
	if job.StatusCode == "InProgress" {
		return nil, "", fmt.Errorf("job %s has not completed yet", jobID)
	}
	if job.Action == "InventoryRetrieval" {
		// Return a mock inventory JSON
		output := []byte(`{"VaultARN":"` + s.vaultARN(vaultName) + `","InventoryDate":"` + time.Now().UTC().Format(time.RFC3339) + `","ArchiveList":[]}`)
		return output, "application/json", nil
	}
	// ArchiveRetrieval: return dummy bytes
	return []byte("MOCK_ARCHIVE_CONTENT"), "application/octet-stream", nil
}

// AddTagsToVault adds tags to a vault.
func (s *Store) AddTagsToVault(vaultName string, tags map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	existing := s.tags[vault.VaultARN]
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

// RemoveTagsFromVault removes tags from a vault.
func (s *Store) RemoveTagsFromVault(vaultName string, keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return fmt.Errorf("vault not found: %s", vaultName)
	}
	existing := s.tags[vault.VaultARN]
	for _, k := range keys {
		delete(existing, k)
	}
	return nil
}

// ListTagsForVault returns tags for a vault.
func (s *Store) ListTagsForVault(vaultName string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vault, ok := s.vaults[vaultName]
	if !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultName)
	}
	tags := s.tags[vault.VaultARN]
	cp := make(map[string]string, len(tags))
	for k, v := range tags {
		cp[k] = v
	}
	return cp, nil
}
