package translate

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StoredTerminology is a persisted custom terminology.
type StoredTerminology struct {
	Name            string
	Arn             string
	Description     string
	SourceLanguage  string
	TargetLanguages []string
	TermCount       int
	EncryptionKeyID string
	Format          string
	Directionality  string
	CreatedAt       time.Time
	LastUpdatedAt   time.Time
	Data            []byte
	Tags            map[string]string
}

// StoredParallelData is a persisted parallel data resource.
type StoredParallelData struct {
	Name                string
	Arn                 string
	Description         string
	Status              string
	SourceLanguageCode  string
	TargetLanguageCodes []string
	ParallelDataConfig  map[string]any
	EncryptionKeyID     string
	CreatedAt           time.Time
	LastUpdatedAt       time.Time
	ImportedDataSize    int
	ImportedRecordCount int
	FailedRecordCount   int
	Message             string
	Tags                map[string]string
}

// StoredJob is a persisted batch translation job.
type StoredJob struct {
	JobID               string
	JobName             string
	JobStatus           string
	Message             string
	SubmittedTime       time.Time
	EndTime             time.Time
	SourceLanguageCode  string
	TargetLanguageCodes []string
	TerminologyNames    []string
	ParallelDataNames   []string
	InputDataConfig     map[string]any
	OutputDataConfig    map[string]any
	DataAccessRoleArn   string
	Settings            map[string]any
}

// Store is the in-memory data store for translate resources.
type Store struct {
	mu            sync.RWMutex
	accountID     string
	region        string
	terminologies map[string]*StoredTerminology
	parallelData  map[string]*StoredParallelData
	jobs          map[string]*StoredJob
	resourceTags  map[string]map[string]string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:     accountID,
		region:        region,
		terminologies: make(map[string]*StoredTerminology),
		parallelData:  make(map[string]*StoredParallelData),
		jobs:          make(map[string]*StoredJob),
		resourceTags:  make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.terminologies = make(map[string]*StoredTerminology)
	s.parallelData = make(map[string]*StoredParallelData)
	s.jobs = make(map[string]*StoredJob)
	s.resourceTags = make(map[string]map[string]string)
}

// ── Terminology ──────────────────────────────────────────────────────────────

func (s *Store) ImportTerminology(name, description, sourceLang string, targetLangs []string, format, directionality string, data []byte, tags map[string]string) *StoredTerminology {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	term, ok := s.terminologies[name]
	if !ok {
		term = &StoredTerminology{
			Name:      name,
			Arn:       fmt.Sprintf("arn:aws:translate:%s:%s:terminology/%s", s.region, s.accountID, name),
			CreatedAt: now,
			Tags:      map[string]string{},
		}
		s.terminologies[name] = term
	}
	term.Description = description
	term.SourceLanguage = sourceLang
	term.TargetLanguages = append([]string(nil), targetLangs...)
	term.Format = format
	term.Directionality = directionality
	term.Data = append([]byte(nil), data...)
	term.TermCount = countLines(data)
	term.LastUpdatedAt = now
	for k, v := range tags {
		term.Tags[k] = v
	}
	return term
}

func (s *Store) GetTerminology(name string) (*StoredTerminology, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.terminologies[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Terminology not found: "+name, 404)
	}
	return t, nil
}

func (s *Store) DeleteTerminology(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.terminologies[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Terminology not found: "+name, 404)
	}
	delete(s.terminologies, name)
	return nil
}

func (s *Store) ListTerminologies() []*StoredTerminology {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTerminology, 0, len(s.terminologies))
	for _, t := range s.terminologies {
		out = append(out, t)
	}
	return out
}

// ── Parallel data ────────────────────────────────────────────────────────────

func (s *Store) CreateParallelData(name, description, sourceLang string, targetLangs []string, config map[string]any, encKey string, tags map[string]string) (*StoredParallelData, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parallelData[name]; ok {
		return nil, service.NewAWSError("ConflictException",
			"Parallel data already exists: "+name, 409)
	}
	now := time.Now().UTC()
	pd := &StoredParallelData{
		Name:                name,
		Arn:                 fmt.Sprintf("arn:aws:translate:%s:%s:parallel-data/%s", s.region, s.accountID, name),
		Description:         description,
		Status:              "ACTIVE",
		SourceLanguageCode:  sourceLang,
		TargetLanguageCodes: append([]string(nil), targetLangs...),
		ParallelDataConfig:  config,
		EncryptionKeyID:     encKey,
		CreatedAt:           now,
		LastUpdatedAt:       now,
		Tags:                copyStringMap(tags),
	}
	s.parallelData[name] = pd
	return pd, nil
}

func (s *Store) GetParallelData(name string) (*StoredParallelData, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pd, ok := s.parallelData[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Parallel data not found: "+name, 404)
	}
	return pd, nil
}

func (s *Store) DeleteParallelData(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parallelData[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Parallel data not found: "+name, 404)
	}
	delete(s.parallelData, name)
	return nil
}

func (s *Store) ListParallelData() []*StoredParallelData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredParallelData, 0, len(s.parallelData))
	for _, pd := range s.parallelData {
		out = append(out, pd)
	}
	return out
}

func (s *Store) UpdateParallelData(name, description string, config map[string]any) (*StoredParallelData, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pd, ok := s.parallelData[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Parallel data not found: "+name, 404)
	}
	if description != "" {
		pd.Description = description
	}
	if config != nil {
		pd.ParallelDataConfig = config
	}
	pd.LastUpdatedAt = time.Now().UTC()
	return pd, nil
}

// ── Jobs ─────────────────────────────────────────────────────────────────────

func (s *Store) StartJob(job *StoredJob) *StoredJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.JobID == "" {
		job.JobID = newUUID()
	}
	if job.JobStatus == "" {
		job.JobStatus = "COMPLETED"
	}
	if job.SubmittedTime.IsZero() {
		job.SubmittedTime = time.Now().UTC()
	}
	if job.EndTime.IsZero() {
		job.EndTime = job.SubmittedTime
	}
	s.jobs[job.JobID] = job
	return job
}

func (s *Store) StopJob(id string) (*StoredJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Job not found: "+id, 404)
	}
	job.JobStatus = "STOPPED"
	job.EndTime = time.Now().UTC()
	return job, nil
}

func (s *Store) GetJob(id string) (*StoredJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Job not found: "+id, 404)
	}
	return job, nil
}

func (s *Store) ListJobs() []*StoredJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
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
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
