package backup

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// BackupPlan represents an AWS Backup plan.
type BackupPlan struct {
	BackupPlanID   string
	BackupPlanArn  string
	BackupPlanName string
	Rules          []BackupRule
	CreatedAt      time.Time
	VersionID      string
}

// BackupRule represents a rule in a backup plan.
type BackupRule struct {
	RuleName              string
	TargetBackupVaultName string
	ScheduleExpression    string
	StartWindowMinutes    int64
	CompletionWindowMinutes int64
	Lifecycle             *BackupLifecycle
}

// BackupLifecycle defines retention for a backup rule.
type BackupLifecycle struct {
	DeleteAfterDays            int64
	MoveToColdStorageAfterDays int64
}

// BackupVault represents a backup vault.
type BackupVault struct {
	BackupVaultName        string
	BackupVaultArn         string
	CreationDate           time.Time
	NumberOfRecoveryPoints int64
	EncryptionKeyArn       string
	Locked                 bool
	LockDate               *time.Time
	MinRetentionDays       int64
	MaxRetentionDays       int64
}

// BackupJob represents a backup job.
type BackupJob struct {
	BackupJobID        string
	BackupVaultName    string
	BackupVaultArn     string
	ResourceArn        string
	ResourceType       string
	State              string
	StatusMessage      string
	CreationDate       time.Time
	CompletionDate     *time.Time
	BackupSizeInBytes  int64
	RecoveryPointArn   string
	lifecycle          *lifecycle.Machine
}

// RecoveryPoint represents a recovery point.
type RecoveryPoint struct {
	RecoveryPointArn    string
	BackupVaultName     string
	BackupVaultArn      string
	ResourceArn         string
	ResourceType        string
	CreationDate        time.Time
	Status              string
	BackupSizeInBytes   int64
	IsEncrypted         bool
}

// BackupSelection represents a backup selection (resources to back up).
type BackupSelection struct {
	SelectionID   string
	SelectionName string
	BackupPlanID  string
	IamRoleArn    string
	Resources     []string
	CreationDate  time.Time
}

// Store manages Backup resources in memory.
type Store struct {
	mu          sync.RWMutex
	plans       map[string]*BackupPlan
	vaults      map[string]*BackupVault
	jobs        map[string]*BackupJob
	recoveryPts map[string]*RecoveryPoint
	selections  map[string]map[string]*BackupSelection // planID -> selectionID -> selection
	accountID   string
	region      string
	lcConfig    *lifecycle.Config
	planSeq     int
	jobSeq      int
	selSeq      int
	rpSeq       int
}

// NewStore returns a new empty Backup Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		plans:       make(map[string]*BackupPlan),
		vaults:      make(map[string]*BackupVault),
		jobs:        make(map[string]*BackupJob),
		recoveryPts: make(map[string]*RecoveryPoint),
		selections:  make(map[string]map[string]*BackupSelection),
		accountID:   accountID,
		region:      region,
		lcConfig:    lifecycle.DefaultConfig(),
	}
	// Create default vault
	s.vaults["Default"] = &BackupVault{
		BackupVaultName: "Default",
		BackupVaultArn:  fmt.Sprintf("arn:aws:backup:%s:%s:backup-vault:Default", region, accountID),
		CreationDate:    time.Now().UTC(),
	}
	return s
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:backup:%s:%s:", s.region, s.accountID)
}

// CreateBackupPlan creates a new backup plan.
func (s *Store) CreateBackupPlan(name string, rules []BackupRule) (*BackupPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.planSeq++
	planID := fmt.Sprintf("plan-%012d", s.planSeq)

	plan := &BackupPlan{
		BackupPlanID:   planID,
		BackupPlanArn:  s.arnPrefix() + "backup-plan:" + planID,
		BackupPlanName: name,
		Rules:          rules,
		CreatedAt:      time.Now().UTC(),
		VersionID:      fmt.Sprintf("version-%d", time.Now().UnixNano()),
	}
	s.plans[planID] = plan
	s.selections[planID] = make(map[string]*BackupSelection)
	return plan, nil
}

// GetBackupPlan retrieves a backup plan by ID.
func (s *Store) GetBackupPlan(planID string) (*BackupPlan, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[planID]
	return plan, ok
}

// ListBackupPlans returns all backup plans.
func (s *Store) ListBackupPlans() []*BackupPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*BackupPlan, 0, len(s.plans))
	for _, p := range s.plans {
		out = append(out, p)
	}
	return out
}

// DeleteBackupPlan deletes a backup plan.
func (s *Store) DeleteBackupPlan(planID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.plans[planID]; !ok {
		return false
	}
	delete(s.plans, planID)
	delete(s.selections, planID)
	return true
}

// CreateBackupVault creates a new backup vault.
func (s *Store) CreateBackupVault(name, encryptionKeyArn string) (*BackupVault, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vaults[name]; ok {
		return nil, fmt.Errorf("backup vault already exists: %s", name)
	}

	vault := &BackupVault{
		BackupVaultName:  name,
		BackupVaultArn:   s.arnPrefix() + "backup-vault:" + name,
		CreationDate:     time.Now().UTC(),
		EncryptionKeyArn: encryptionKeyArn,
	}
	s.vaults[name] = vault
	return vault, nil
}

// GetBackupVault retrieves a backup vault by name.
func (s *Store) GetBackupVault(name string) (*BackupVault, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vault, ok := s.vaults[name]
	return vault, ok
}

// ListBackupVaults returns all backup vaults.
func (s *Store) ListBackupVaults() []*BackupVault {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*BackupVault, 0, len(s.vaults))
	for _, v := range s.vaults {
		out = append(out, v)
	}
	return out
}

// DeleteBackupVault deletes a backup vault.
func (s *Store) DeleteBackupVault(name string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[name]
	if !ok {
		return false, "not_found"
	}
	if vault.Locked {
		return false, "locked"
	}
	if vault.NumberOfRecoveryPoints > 0 {
		return false, "not_empty"
	}
	delete(s.vaults, name)
	return true, ""
}

// LockBackupVault locks a vault with retention policy.
func (s *Store) LockBackupVault(name string, minRetention, maxRetention int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	vault, ok := s.vaults[name]
	if !ok {
		return fmt.Errorf("vault not found: %s", name)
	}
	now := time.Now().UTC()
	vault.Locked = true
	vault.LockDate = &now
	vault.MinRetentionDays = minRetention
	vault.MaxRetentionDays = maxRetention
	return nil
}

// StartBackupJob starts a backup job.
func (s *Store) StartBackupJob(vaultName, resourceArn, resourceType, iamRoleArn string) (*BackupJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vault, ok := s.vaults[vaultName]
	if !ok {
		return nil, fmt.Errorf("backup vault not found: %s", vaultName)
	}

	s.jobSeq++
	jobID := fmt.Sprintf("job-%012d", s.jobSeq)

	job := &BackupJob{
		BackupJobID:       jobID,
		BackupVaultName:   vaultName,
		BackupVaultArn:    vault.BackupVaultArn,
		ResourceArn:       resourceArn,
		ResourceType:      resourceType,
		State:             "CREATED",
		StatusMessage:     "Backup job created",
		CreationDate:      time.Now().UTC(),
		BackupSizeInBytes: 1073741824, // 1 GB mock
	}

	// For instant transitions (default config), complete the job synchronously
	// to avoid race conditions with async lifecycle callbacks.
	if s.lcConfig.EffectiveDelay(1*time.Second) <= 0 {
		s.completeJob(job, vault, vaultName, resourceArn, resourceType)
	} else {
		transitions := []lifecycle.Transition{
			{From: "CREATED", To: "RUNNING", Delay: 1 * time.Second},
			{From: "RUNNING", To: "COMPLETED", Delay: 3 * time.Second},
		}
		job.lifecycle = lifecycle.NewMachine("CREATED", transitions, s.lcConfig)
		job.lifecycle.OnTransition(func(from, to lifecycle.State) {
			s.mu.Lock()
			defer s.mu.Unlock()
			job.State = string(to)
			if to == "COMPLETED" {
				s.completeJob(job, vault, vaultName, resourceArn, resourceType)
			} else if to == "RUNNING" {
				job.StatusMessage = "Backup job running"
			}
		})
	}

	s.jobs[jobID] = job
	return job, nil
}

// completeJob finalizes a backup job and creates a recovery point. Must be called with s.mu held.
func (s *Store) completeJob(job *BackupJob, vault *BackupVault, vaultName, resourceArn, resourceType string) {
	now := time.Now().UTC()
	job.State = "COMPLETED"
	job.CompletionDate = &now
	job.StatusMessage = "Backup job completed"

	s.rpSeq++
	rpArn := fmt.Sprintf("%srecovery-point:%012d", s.arnPrefix(), s.rpSeq)
	job.RecoveryPointArn = rpArn

	s.recoveryPts[rpArn] = &RecoveryPoint{
		RecoveryPointArn:  rpArn,
		BackupVaultName:   vaultName,
		BackupVaultArn:    vault.BackupVaultArn,
		ResourceArn:       resourceArn,
		ResourceType:      resourceType,
		CreationDate:      now,
		Status:            "COMPLETED",
		BackupSizeInBytes: job.BackupSizeInBytes,
		IsEncrypted:       true,
	}
	vault.NumberOfRecoveryPoints++
}

// GetBackupJob retrieves a backup job.
func (s *Store) GetBackupJob(jobID string) (*BackupJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if ok && job.lifecycle != nil {
		job.State = string(job.lifecycle.State())
	}
	return job, ok
}

// ListBackupJobs returns all backup jobs.
func (s *Store) ListBackupJobs() []*BackupJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*BackupJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

// ListRecoveryPoints returns recovery points for a vault.
func (s *Store) ListRecoveryPoints(vaultName string) []*RecoveryPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RecoveryPoint, 0)
	for _, rp := range s.recoveryPts {
		if vaultName == "" || rp.BackupVaultName == vaultName {
			out = append(out, rp)
		}
	}
	return out
}

// GetRecoveryPoint retrieves a recovery point.
func (s *Store) GetRecoveryPoint(vaultName, rpArn string) (*RecoveryPoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rp, ok := s.recoveryPts[rpArn]
	if !ok || rp.BackupVaultName != vaultName {
		return nil, false
	}
	return rp, true
}

// CreateBackupSelection creates a backup selection for a plan.
func (s *Store) CreateBackupSelection(planID, selName, iamRoleArn string, resources []string) (*BackupSelection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	selMap, ok := s.selections[planID]
	if !ok {
		return nil, fmt.Errorf("backup plan not found: %s", planID)
	}

	s.selSeq++
	selID := fmt.Sprintf("sel-%012d", s.selSeq)

	sel := &BackupSelection{
		SelectionID:   selID,
		SelectionName: selName,
		BackupPlanID:  planID,
		IamRoleArn:    iamRoleArn,
		Resources:     resources,
		CreationDate:  time.Now().UTC(),
	}
	selMap[selID] = sel
	return sel, nil
}

// ListBackupSelections returns all selections for a plan.
func (s *Store) ListBackupSelections(planID string) []*BackupSelection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	selMap := s.selections[planID]
	out := make([]*BackupSelection, 0, len(selMap))
	for _, sel := range selMap {
		out = append(out, sel)
	}
	return out
}

// GetBackupSelection retrieves a backup selection.
func (s *Store) GetBackupSelection(planID, selID string) (*BackupSelection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	selMap, ok := s.selections[planID]
	if !ok {
		return nil, false
	}
	sel, ok := selMap[selID]
	return sel, ok
}

// DeleteBackupSelection deletes a backup selection.
func (s *Store) DeleteBackupSelection(planID, selID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	selMap, ok := s.selections[planID]
	if !ok {
		return false
	}
	if _, ok := selMap[selID]; !ok {
		return false
	}
	delete(selMap, selID)
	return true
}
