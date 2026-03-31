package mediaconvert

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Job represents a MediaConvert job.
type Job struct {
	ID               string
	Arn              string
	Queue            string
	Role             string
	Status           string
	StatusUpdateInterval int
	Settings         map[string]any
	CreatedAt        time.Time
	SubmittedAt      time.Time
	FinishedAt       *time.Time
	ErrorCode        int
	ErrorMessage     string
	lifecycle        *lifecycle.Machine
}

// JobTemplate represents a MediaConvert job template.
type JobTemplate struct {
	Name        string
	Arn         string
	Description string
	Category    string
	Settings    map[string]any
	Type        string
	CreatedAt   time.Time
}

// Preset represents a MediaConvert preset.
type Preset struct {
	Name        string
	Arn         string
	Description string
	Category    string
	Settings    map[string]any
	Type        string
	CreatedAt   time.Time
}

// Queue represents a MediaConvert queue.
type Queue struct {
	Name             string
	Arn              string
	Description      string
	Status           string
	PricingPlan      string
	Type             string
	SubmittedJobsCount int
	ProgressingJobsCount int
	CreatedAt        time.Time
}

// Store manages MediaConvert resources in memory.
type Store struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	templates map[string]*JobTemplate
	presets   map[string]*Preset
	queues    map[string]*Queue
	accountID string
	region    string
	lcConfig  *lifecycle.Config
	jobSeq    int
}

// NewStore returns a new empty MediaConvert Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		jobs:      make(map[string]*Job),
		templates: make(map[string]*JobTemplate),
		presets:   make(map[string]*Preset),
		queues:    make(map[string]*Queue),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
	// Default queue
	s.queues["Default"] = &Queue{
		Name:        "Default",
		Arn:         fmt.Sprintf("arn:aws:mediaconvert:%s:%s:queues/Default", region, accountID),
		Description: "Default queue",
		Status:      "ACTIVE",
		PricingPlan: "ON_DEMAND",
		Type:        "SYSTEM",
		CreatedAt:   time.Now().UTC(),
	}
	return s
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:mediaconvert:%s:%s:", s.region, s.accountID)
}

// CreateJob creates a new job.
func (s *Store) CreateJob(queue, role string, settings map[string]any) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobSeq++
	id := fmt.Sprintf("%012d", s.jobSeq)
	now := time.Now().UTC()

	transitions := []lifecycle.Transition{
		{From: "SUBMITTED", To: "PROGRESSING", Delay: 1 * time.Second},
		{From: "PROGRESSING", To: "COMPLETE", Delay: 5 * time.Second},
	}

	job := &Job{
		ID:          id,
		Arn:         s.arnPrefix() + "jobs/" + id,
		Queue:       queue,
		Role:        role,
		Status:      "SUBMITTED",
		Settings:    settings,
		CreatedAt:   now,
		SubmittedAt: now,
	}
	job.lifecycle = lifecycle.NewMachine("SUBMITTED", transitions, s.lcConfig)
	job.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		job.Status = string(to)
		if to == "COMPLETE" {
			now := time.Now().UTC()
			job.FinishedAt = &now
		}
	})

	s.jobs[id] = job
	if q, ok := s.queues[queue]; ok {
		q.SubmittedJobsCount++
	}
	return job, nil
}

// GetJob retrieves a job.
func (s *Store) GetJob(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

// ListJobs returns all jobs.
func (s *Store) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

// CancelJob cancels a job.
func (s *Store) CancelJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}
	if j.Status != "SUBMITTED" {
		return fmt.Errorf("job cannot be canceled in state: %s", j.Status)
	}
	if j.lifecycle != nil {
		j.lifecycle.Stop()
	}
	j.Status = "CANCELED"
	return nil
}

// CreateJobTemplate creates a job template.
func (s *Store) CreateJobTemplate(name, description, category string, settings map[string]any) (*JobTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.templates[name]; ok {
		return nil, fmt.Errorf("template already exists: %s", name)
	}

	tmpl := &JobTemplate{
		Name:        name,
		Arn:         s.arnPrefix() + "jobTemplates/" + name,
		Description: description,
		Category:    category,
		Settings:    settings,
		Type:        "CUSTOM",
		CreatedAt:   time.Now().UTC(),
	}
	s.templates[name] = tmpl
	return tmpl, nil
}

// GetJobTemplate retrieves a template.
func (s *Store) GetJobTemplate(name string) (*JobTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[name]
	return t, ok
}

// ListJobTemplates returns all templates.
func (s *Store) ListJobTemplates() []*JobTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*JobTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		out = append(out, t)
	}
	return out
}

// DeleteJobTemplate removes a template.
func (s *Store) DeleteJobTemplate(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[name]; !ok {
		return false
	}
	delete(s.templates, name)
	return true
}

// CreatePreset creates a preset.
func (s *Store) CreatePreset(name, description, category string, settings map[string]any) (*Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.presets[name]; ok {
		return nil, fmt.Errorf("preset already exists: %s", name)
	}

	preset := &Preset{
		Name:        name,
		Arn:         s.arnPrefix() + "presets/" + name,
		Description: description,
		Category:    category,
		Settings:    settings,
		Type:        "CUSTOM",
		CreatedAt:   time.Now().UTC(),
	}
	s.presets[name] = preset
	return preset, nil
}

// GetPreset retrieves a preset.
func (s *Store) GetPreset(name string) (*Preset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.presets[name]
	return p, ok
}

// ListPresets returns all presets.
func (s *Store) ListPresets() []*Preset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Preset, 0, len(s.presets))
	for _, p := range s.presets {
		out = append(out, p)
	}
	return out
}

// DeletePreset removes a preset.
func (s *Store) DeletePreset(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.presets[name]; !ok {
		return false
	}
	delete(s.presets, name)
	return true
}

// CreateQueue creates a queue.
func (s *Store) CreateQueue(name, description, pricingPlan string) (*Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queues[name]; ok {
		return nil, fmt.Errorf("queue already exists: %s", name)
	}
	if pricingPlan == "" {
		pricingPlan = "ON_DEMAND"
	}

	q := &Queue{
		Name:        name,
		Arn:         s.arnPrefix() + "queues/" + name,
		Description: description,
		Status:      "ACTIVE",
		PricingPlan: pricingPlan,
		Type:        "CUSTOM",
		CreatedAt:   time.Now().UTC(),
	}
	s.queues[name] = q
	return q, nil
}

// GetQueue retrieves a queue.
func (s *Store) GetQueue(name string) (*Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.queues[name]
	return q, ok
}

// ListQueues returns all queues.
func (s *Store) ListQueues() []*Queue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Queue, 0, len(s.queues))
	for _, q := range s.queues {
		out = append(out, q)
	}
	return out
}

// DeleteQueue removes a queue.
func (s *Store) DeleteQueue(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.queues[name]; !ok {
		return false
	}
	delete(s.queues, name)
	return true
}
