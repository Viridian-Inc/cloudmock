package bedrock

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

type CustomizationJobStatus string

const (
	CustomizationInProgress CustomizationJobStatus = "InProgress"
	CustomizationCompleted  CustomizationJobStatus = "Completed"
	CustomizationFailed     CustomizationJobStatus = "Failed"
	CustomizationStopping   CustomizationJobStatus = "Stopping"
	CustomizationStopped    CustomizationJobStatus = "Stopped"
)

type ProvisionedModelStatus string

const (
	ProvisionedCreating ProvisionedModelStatus = "Creating"
	ProvisionedInService ProvisionedModelStatus = "InService"
	ProvisionedUpdating  ProvisionedModelStatus = "Updating"
	ProvisionedFailed    ProvisionedModelStatus = "Failed"
)

type ModelCustomizationJob struct {
	JobName              string
	JobArn               string
	Status               CustomizationJobStatus
	BaseModelIdentifier  string
	CustomModelName      string
	CustomModelArn       string
	RoleArn              string
	CustomizationType    string
	HyperParameters      map[string]string
	TrainingDataConfig   map[string]any
	ValidationDataConfig map[string]any
	OutputDataConfig     map[string]any
	CreationTime         time.Time
	EndTime              *time.Time
	FailureMessage       string
	Tags                 map[string]string
	Lifecycle            *lifecycle.Machine
}

type ProvisionedModelThroughput struct {
	ProvisionedModelId   string
	ProvisionedModelArn  string
	ProvisionedModelName string
	ModelArn             string
	ModelUnits           int
	DesiredModelUnits    int
	Status               ProvisionedModelStatus
	CommitmentDuration   string
	CreationTime         time.Time
	LastModifiedTime     time.Time
	FailureMessage       string
	Tags                 map[string]string
	Lifecycle            *lifecycle.Machine
}

type FoundationModel struct {
	ModelId              string
	ModelArn             string
	ModelName            string
	Provider             string
	InputModalities      []string
	OutputModalities     []string
	CustomizationsSupported []string
	InferenceTypesSupported []string
	ResponseStreamingSupported bool
}

// Guardrail represents a Bedrock guardrail for content filtering.
type Guardrail struct {
	ID                      string
	ARN                     string
	Name                    string
	Description             string
	Version                 string
	BlockedInputMessaging   string
	BlockedOutputsMessaging string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Status                  string
}

type Store struct {
	mu                   sync.RWMutex
	customizationJobs    map[string]*ModelCustomizationJob      // keyed by job name
	provisionedModels    map[string]*ProvisionedModelThroughput // keyed by name
	foundationModels     []FoundationModel
	guardrails           map[string]*Guardrail
	modelEvalJobs        map[string]*ModelEvaluationJob // keyed by job name
	tagsByArn            map[string]map[string]string
	accountID            string
	region               string
	lcConfig             *lifecycle.Config
	guardrailSeq         int
}

func NewStore(accountID, region string) *Store {
	s := &Store{
		customizationJobs: make(map[string]*ModelCustomizationJob),
		provisionedModels: make(map[string]*ProvisionedModelThroughput),
		guardrails:        make(map[string]*Guardrail),
		modelEvalJobs:     make(map[string]*ModelEvaluationJob),
		tagsByArn:         make(map[string]map[string]string),
		accountID:         accountID,
		region:            region,
		lcConfig:          lifecycle.DefaultConfig(),
	}
	s.foundationModels = defaultFoundationModels(accountID, region)
	return s
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func defaultFoundationModels(accountID, region string) []FoundationModel {
	return []FoundationModel{
		{
			ModelId:         "anthropic.claude-3-5-sonnet-20241022-v2:0",
			ModelArn:        fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/anthropic.claude-3-5-sonnet-20241022-v2:0", region),
			ModelName:       "Claude 3.5 Sonnet v2",
			Provider:        "Anthropic",
			InputModalities: []string{"TEXT", "IMAGE"},
			OutputModalities: []string{"TEXT"},
			CustomizationsSupported: []string{},
			InferenceTypesSupported: []string{"ON_DEMAND", "PROVISIONED"},
			ResponseStreamingSupported: true,
		},
		{
			ModelId:         "anthropic.claude-3-haiku-20240307-v1:0",
			ModelArn:        fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/anthropic.claude-3-haiku-20240307-v1:0", region),
			ModelName:       "Claude 3 Haiku",
			Provider:        "Anthropic",
			InputModalities: []string{"TEXT", "IMAGE"},
			OutputModalities: []string{"TEXT"},
			CustomizationsSupported: []string{},
			InferenceTypesSupported: []string{"ON_DEMAND", "PROVISIONED"},
			ResponseStreamingSupported: true,
		},
		{
			ModelId:         "amazon.titan-text-express-v1",
			ModelArn:        fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/amazon.titan-text-express-v1", region),
			ModelName:       "Titan Text Express",
			Provider:        "Amazon",
			InputModalities: []string{"TEXT"},
			OutputModalities: []string{"TEXT"},
			CustomizationsSupported: []string{"FINE_TUNING"},
			InferenceTypesSupported: []string{"ON_DEMAND", "PROVISIONED"},
			ResponseStreamingSupported: true,
		},
		{
			ModelId:         "amazon.titan-embed-text-v1",
			ModelArn:        fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/amazon.titan-embed-text-v1", region),
			ModelName:       "Titan Embeddings Text",
			Provider:        "Amazon",
			InputModalities: []string{"TEXT"},
			OutputModalities: []string{"EMBEDDING"},
			CustomizationsSupported: []string{},
			InferenceTypesSupported: []string{"ON_DEMAND"},
			ResponseStreamingSupported: false,
		},
		{
			ModelId:         "meta.llama3-70b-instruct-v1:0",
			ModelArn:        fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/meta.llama3-70b-instruct-v1:0", region),
			ModelName:       "Llama 3 70B Instruct",
			Provider:        "Meta",
			InputModalities: []string{"TEXT"},
			OutputModalities: []string{"TEXT"},
			CustomizationsSupported: []string{},
			InferenceTypesSupported: []string{"ON_DEMAND"},
			ResponseStreamingSupported: true,
		},
	}
}

func (s *Store) customizationJobARN(name string) string {
	return fmt.Sprintf("arn:aws:bedrock:%s:%s:model-customization-job/%s", s.region, s.accountID, name)
}

func (s *Store) customModelARN(name string) string {
	return fmt.Sprintf("arn:aws:bedrock:%s:%s:custom-model/%s", s.region, s.accountID, name)
}

func (s *Store) provisionedModelARN(name string) string {
	return fmt.Sprintf("arn:aws:bedrock:%s:%s:provisioned-model/%s", s.region, s.accountID, name)
}

// Model customization jobs.

func (s *Store) CreateModelCustomizationJob(jobName, baseModel, customModelName, roleArn, customizationType string, hyperParams map[string]string, trainingConfig, validationConfig, outputConfig map[string]any, tags map[string]string) (*ModelCustomizationJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.customizationJobs[jobName]; exists {
		return nil, service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Model customization job %s already exists", jobName), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if customizationType == "" {
		customizationType = "FINE_TUNING"
	}

	lc := lifecycle.NewMachine(
		lifecycle.State(CustomizationInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(CustomizationInProgress), To: lifecycle.State(CustomizationCompleted), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	job := &ModelCustomizationJob{
		JobName:              jobName,
		JobArn:               s.customizationJobARN(jobName),
		Status:               CustomizationJobStatus(lc.State()),
		BaseModelIdentifier:  baseModel,
		CustomModelName:      customModelName,
		CustomModelArn:       s.customModelARN(customModelName),
		RoleArn:              roleArn,
		CustomizationType:    customizationType,
		HyperParameters:      hyperParams,
		TrainingDataConfig:   trainingConfig,
		ValidationDataConfig: validationConfig,
		OutputDataConfig:     outputConfig,
		CreationTime:         time.Now().UTC(),
		Tags:                 tags,
		Lifecycle:            lc,
	}
	s.customizationJobs[jobName] = job
	s.tagsByArn[job.JobArn] = tags
	return job, nil
}

func (s *Store) GetModelCustomizationJob(jobName string) (*ModelCustomizationJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.customizationJobs[jobName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Model customization job %s not found", jobName), http.StatusNotFound)
	}
	job.Status = CustomizationJobStatus(job.Lifecycle.State())
	if job.Status == CustomizationCompleted && job.EndTime == nil {
		now := time.Now().UTC()
		job.EndTime = &now
	}
	return job, nil
}

func (s *Store) ListModelCustomizationJobs() []*ModelCustomizationJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ModelCustomizationJob, 0, len(s.customizationJobs))
	for _, job := range s.customizationJobs {
		job.Status = CustomizationJobStatus(job.Lifecycle.State())
		out = append(out, job)
	}
	return out
}

func (s *Store) StopModelCustomizationJob(jobName string) *service.AWSError {
	s.mu.Lock()
	job, ok := s.customizationJobs[jobName]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Model customization job %s not found", jobName), http.StatusNotFound)
	}
	now := time.Now().UTC()
	job.EndTime = &now
	lc := job.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(CustomizationStopped))
	}
	return nil
}

// Provisioned model throughput.

func (s *Store) CreateProvisionedModelThroughput(name, modelArn string, modelUnits int, commitmentDuration string, tags map[string]string) (*ProvisionedModelThroughput, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.provisionedModels[name]; exists {
		return nil, service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Provisioned model %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if modelUnits <= 0 {
		modelUnits = 1
	}

	lc := lifecycle.NewMachine(
		lifecycle.State(ProvisionedCreating),
		[]lifecycle.Transition{
			{From: lifecycle.State(ProvisionedCreating), To: lifecycle.State(ProvisionedInService), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	now := time.Now().UTC()
	pm := &ProvisionedModelThroughput{
		ProvisionedModelId:   newUUID(),
		ProvisionedModelArn:  s.provisionedModelARN(name),
		ProvisionedModelName: name,
		ModelArn:             modelArn,
		ModelUnits:           modelUnits,
		DesiredModelUnits:    modelUnits,
		Status:               ProvisionedModelStatus(lc.State()),
		CommitmentDuration:   commitmentDuration,
		CreationTime:         now,
		LastModifiedTime:     now,
		Tags:                 tags,
		Lifecycle:            lc,
	}
	s.provisionedModels[name] = pm
	s.tagsByArn[pm.ProvisionedModelArn] = tags
	return pm, nil
}

func (s *Store) GetProvisionedModelThroughput(name string) (*ProvisionedModelThroughput, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pm, ok := s.provisionedModels[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Provisioned model %s not found", name), http.StatusNotFound)
	}
	pm.Status = ProvisionedModelStatus(pm.Lifecycle.State())
	return pm, nil
}

func (s *Store) ListProvisionedModelThroughputs() []*ProvisionedModelThroughput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ProvisionedModelThroughput, 0, len(s.provisionedModels))
	for _, pm := range s.provisionedModels {
		pm.Status = ProvisionedModelStatus(pm.Lifecycle.State())
		out = append(out, pm)
	}
	return out
}

func (s *Store) UpdateProvisionedModelThroughput(name string, desiredUnits int, desiredModelArn string) (*ProvisionedModelThroughput, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pm, ok := s.provisionedModels[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Provisioned model %s not found", name), http.StatusNotFound)
	}
	if desiredUnits > 0 {
		pm.DesiredModelUnits = desiredUnits
	}
	if desiredModelArn != "" {
		pm.ModelArn = desiredModelArn
	}
	pm.LastModifiedTime = time.Now().UTC()
	pm.Lifecycle.ForceState(lifecycle.State(ProvisionedUpdating))
	return pm, nil
}

func (s *Store) DeleteProvisionedModelThroughput(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	pm, ok := s.provisionedModels[name]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Provisioned model %s not found", name), http.StatusNotFound)
	}
	pm.Lifecycle.Stop()
	delete(s.provisionedModels, name)
	delete(s.tagsByArn, pm.ProvisionedModelArn)
	return nil
}

// Foundation models.

func (s *Store) GetFoundationModel(modelId string) (*FoundationModel, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.foundationModels {
		if s.foundationModels[i].ModelId == modelId {
			return &s.foundationModels[i], nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Foundation model %s not found", modelId), http.StatusNotFound)
}

func (s *Store) ListFoundationModels() []FoundationModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FoundationModel, len(s.foundationModels))
	copy(out, s.foundationModels)
	return out
}

// Tags.

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for _, k := range tagKeys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}

// Guardrail operations.

// CreateGuardrail creates a new guardrail.
func (s *Store) CreateGuardrail(name, description, blockedInputMessaging, blockedOutputsMessaging string) *Guardrail {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.guardrailSeq++
	id := fmt.Sprintf("gr-%s", newUUID()[:8])
	arn := fmt.Sprintf("arn:aws:bedrock:%s:%s:guardrail/%s", s.region, s.accountID, id)
	now := time.Now().UTC()
	g := &Guardrail{
		ID:                      id,
		ARN:                     arn,
		Name:                    name,
		Description:             description,
		Version:                 "DRAFT",
		BlockedInputMessaging:   blockedInputMessaging,
		BlockedOutputsMessaging: blockedOutputsMessaging,
		CreatedAt:               now,
		UpdatedAt:               now,
		Status:                  "READY",
	}
	s.guardrails[id] = g
	return g
}

// GetGuardrail retrieves a guardrail by ID.
func (s *Store) GetGuardrail(id string) (*Guardrail, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.guardrails[id]
	return g, ok
}

// ListGuardrails returns all guardrails.
func (s *Store) ListGuardrails() []*Guardrail {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Guardrail, 0, len(s.guardrails))
	for _, g := range s.guardrails {
		result = append(result, g)
	}
	return result
}

// UpdateGuardrail updates a guardrail's properties.
func (s *Store) UpdateGuardrail(id, name, description string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.guardrails[id]
	if !ok {
		return false
	}
	if name != "" {
		g.Name = name
	}
	if description != "" {
		g.Description = description
	}
	g.UpdatedAt = time.Now().UTC()
	return true
}

// DeleteGuardrail deletes a guardrail by ID.
func (s *Store) DeleteGuardrail(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.guardrails[id]; !ok {
		return false
	}
	delete(s.guardrails, id)
	return true
}

// ModelEvaluationJob represents a model evaluation job.
type ModelEvaluationJob struct {
	JobArn           string
	JobName          string
	Status           string
	ModelIdentifier  string
	RoleArn          string
	EvaluationConfig map[string]any
	CreationTime     time.Time
	EndTime          *time.Time
}

func (s *Store) modelEvalJobARN(name string) string {
	return fmt.Sprintf("arn:aws:bedrock:%s:%s:evaluation-job/%s", s.region, s.accountID, name)
}

// CreateModelEvaluationJob creates a new evaluation job.
func (s *Store) CreateModelEvaluationJob(name, roleArn, modelId string, evalConfig map[string]any) (*ModelEvaluationJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.modelEvalJobs[name]; ok {
		return nil, false
	}
	job := &ModelEvaluationJob{
		JobArn:           s.modelEvalJobARN(name),
		JobName:          name,
		Status:           "InProgress",
		ModelIdentifier:  modelId,
		RoleArn:          roleArn,
		EvaluationConfig: evalConfig,
		CreationTime:     time.Now().UTC(),
	}
	s.modelEvalJobs[name] = job
	return job, true
}

// GetModelEvaluationJob retrieves an evaluation job by name.
func (s *Store) GetModelEvaluationJob(name string) (*ModelEvaluationJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.modelEvalJobs[name]
	return job, ok
}

// ListModelEvaluationJobs returns all evaluation jobs.
func (s *Store) ListModelEvaluationJobs() []*ModelEvaluationJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ModelEvaluationJob, 0, len(s.modelEvalJobs))
	for _, j := range s.modelEvalJobs {
		result = append(result, j)
	}
	return result
}
