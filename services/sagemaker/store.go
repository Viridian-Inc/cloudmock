package sagemaker

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Resource states.

type NotebookInstanceStatus string

const (
	NotebookPending   NotebookInstanceStatus = "Pending"
	NotebookInService NotebookInstanceStatus = "InService"
	NotebookStopping  NotebookInstanceStatus = "Stopping"
	NotebookStopped   NotebookInstanceStatus = "Stopped"
	NotebookFailed    NotebookInstanceStatus = "Failed"
	NotebookDeleting  NotebookInstanceStatus = "Deleting"
	NotebookUpdating  NotebookInstanceStatus = "Updating"
)

type TrainingJobStatus string

const (
	TrainingInProgress TrainingJobStatus = "InProgress"
	TrainingCompleted  TrainingJobStatus = "Completed"
	TrainingFailed     TrainingJobStatus = "Failed"
	TrainingStopping   TrainingJobStatus = "Stopping"
	TrainingStopped    TrainingJobStatus = "Stopped"
)

type EndpointStatus string

const (
	EndpointCreating          EndpointStatus = "Creating"
	EndpointInService         EndpointStatus = "InService"
	EndpointUpdating          EndpointStatus = "Updating"
	EndpointDeleting          EndpointStatus = "Deleting"
	EndpointFailed            EndpointStatus = "Failed"
	EndpointSystemUpdating    EndpointStatus = "SystemUpdating"
	EndpointRollingBack       EndpointStatus = "RollingBack"
	EndpointUpdateRollbackFailed EndpointStatus = "UpdateRollbackFailed"
)

type ProcessingJobStatus string

const (
	ProcessingInProgress ProcessingJobStatus = "InProgress"
	ProcessingCompleted  ProcessingJobStatus = "Completed"
	ProcessingFailed     ProcessingJobStatus = "Failed"
	ProcessingStopping   ProcessingJobStatus = "Stopping"
	ProcessingStopped    ProcessingJobStatus = "Stopped"
)

type TransformJobStatus string

const (
	TransformInProgress TransformJobStatus = "InProgress"
	TransformCompleted  TransformJobStatus = "Completed"
	TransformFailed     TransformJobStatus = "Failed"
	TransformStopping   TransformJobStatus = "Stopping"
	TransformStopped    TransformJobStatus = "Stopped"
)

// Domain types.

type NotebookInstance struct {
	NotebookInstanceName string
	NotebookInstanceArn  string
	InstanceType         string
	RoleArn              string
	Status               NotebookInstanceStatus
	SubnetId             string
	SecurityGroupIds     []string
	DirectInternetAccess string
	VolumeSizeInGB       int
	CreationTime         time.Time
	LastModifiedTime     time.Time
	Tags                 map[string]string
	Lifecycle            *lifecycle.Machine
}

type TrainingJob struct {
	TrainingJobName                string
	TrainingJobArn                 string
	TrainingJobStatus              TrainingJobStatus
	SecondaryStatus                string
	AlgorithmSpecification         map[string]any
	RoleArn                        string
	InputDataConfig                []map[string]any
	OutputDataConfig               map[string]any
	ResourceConfig                 map[string]any
	StoppingCondition              map[string]any
	HyperParameters                map[string]string
	CreationTime                   time.Time
	TrainingStartTime              *time.Time
	TrainingEndTime                *time.Time
	LastModifiedTime               time.Time
	FailureReason                  string
	ModelArtifacts                 map[string]any
	Tags                           map[string]string
	Lifecycle                      *lifecycle.Machine
}

type Model struct {
	ModelName             string
	ModelArn              string
	PrimaryContainer      map[string]any
	ExecutionRoleArn      string
	CreationTime          time.Time
	Tags                  map[string]string
}

type EndpointConfig struct {
	EndpointConfigName string
	EndpointConfigArn  string
	ProductionVariants []map[string]any
	CreationTime       time.Time
	Tags               map[string]string
}

type Endpoint struct {
	EndpointName       string
	EndpointArn        string
	EndpointConfigName string
	EndpointStatus     EndpointStatus
	CreationTime       time.Time
	LastModifiedTime   time.Time
	FailureReason      string
	Tags               map[string]string
	Lifecycle          *lifecycle.Machine
}

type ProcessingJob struct {
	ProcessingJobName   string
	ProcessingJobArn    string
	ProcessingJobStatus ProcessingJobStatus
	RoleArn             string
	AppSpecification    map[string]any
	ProcessingResources map[string]any
	ProcessingInputs    []map[string]any
	ProcessingOutputConfig map[string]any
	StoppingCondition   map[string]any
	CreationTime        time.Time
	ProcessingStartTime *time.Time
	ProcessingEndTime   *time.Time
	FailureReason       string
	Tags                map[string]string
	Lifecycle           *lifecycle.Machine
}

type TransformJob struct {
	TransformJobName   string
	TransformJobArn    string
	TransformJobStatus TransformJobStatus
	ModelName          string
	TransformInput     map[string]any
	TransformOutput    map[string]any
	TransformResources map[string]any
	CreationTime       time.Time
	TransformStartTime *time.Time
	TransformEndTime   *time.Time
	FailureReason      string
	Tags               map[string]string
	Lifecycle          *lifecycle.Machine
}

// Store is the in-memory store for all SageMaker resources.
type Store struct {
	mu              sync.RWMutex
	notebooks       map[string]*NotebookInstance // keyed by name
	trainingJobs    map[string]*TrainingJob      // keyed by name
	models          map[string]*Model            // keyed by name
	endpointConfigs map[string]*EndpointConfig   // keyed by name
	endpoints       map[string]*Endpoint         // keyed by name
	processingJobs  map[string]*ProcessingJob    // keyed by name
	transformJobs   map[string]*TransformJob     // keyed by name
	tagsByArn       map[string]map[string]string // extra tags store for ARN-based lookups
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore creates a new SageMaker Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		notebooks:       make(map[string]*NotebookInstance),
		trainingJobs:    make(map[string]*TrainingJob),
		models:          make(map[string]*Model),
		endpointConfigs: make(map[string]*EndpointConfig),
		endpoints:       make(map[string]*Endpoint),
		processingJobs:  make(map[string]*ProcessingJob),
		transformJobs:   make(map[string]*TransformJob),
		tagsByArn:       make(map[string]map[string]string),
		accountID:       accountID,
		region:          region,
		lcConfig:        lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// generateModelArtifacts creates an S3 path for training job output.
func (s *Store) generateModelArtifacts(jobName string, outputDataConfig map[string]any) map[string]any {
	s3URI := "s3://sagemaker-" + s.region + "-" + s.accountID + "/" + jobName + "/output/model.tar.gz"
	if outputDataConfig != nil {
		if s3OutputPath, ok := outputDataConfig["S3OutputPath"].(string); ok && s3OutputPath != "" {
			s3URI = s3OutputPath + "/" + jobName + "/output/model.tar.gz"
		}
	}
	return map[string]any{"S3ModelArtifacts": s3URI}
}

// ARN builders.

func (s *Store) notebookARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:notebook-instance/%s", s.region, s.accountID, name)
}

func (s *Store) trainingJobARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:training-job/%s", s.region, s.accountID, name)
}

func (s *Store) modelARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:model/%s", s.region, s.accountID, name)
}

func (s *Store) endpointConfigARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint-config/%s", s.region, s.accountID, name)
}

func (s *Store) endpointARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:endpoint/%s", s.region, s.accountID, name)
}

func (s *Store) processingJobARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:processing-job/%s", s.region, s.accountID, name)
}

func (s *Store) transformJobARN(name string) string {
	return fmt.Sprintf("arn:aws:sagemaker:%s:%s:transform-job/%s", s.region, s.accountID, name)
}

// Notebook instances.

func (s *Store) CreateNotebookInstance(name, instanceType, roleArn, subnetId, directInternetAccess string, securityGroupIds []string, volumeSize int, tags map[string]string) (*NotebookInstance, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.notebooks[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Notebook instance %s already exists", name), http.StatusConflict)
	}

	if instanceType == "" {
		instanceType = "ml.t2.medium"
	}
	if directInternetAccess == "" {
		directInternetAccess = "Enabled"
	}
	if volumeSize <= 0 {
		volumeSize = 5
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	now := time.Now().UTC()
	lc := lifecycle.NewMachine(
		lifecycle.State(NotebookPending),
		[]lifecycle.Transition{
			{From: lifecycle.State(NotebookPending), To: lifecycle.State(NotebookInService), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	nb := &NotebookInstance{
		NotebookInstanceName: name,
		NotebookInstanceArn:  s.notebookARN(name),
		InstanceType:         instanceType,
		RoleArn:              roleArn,
		Status:               NotebookInstanceStatus(lc.State()),
		SubnetId:             subnetId,
		SecurityGroupIds:     securityGroupIds,
		DirectInternetAccess: directInternetAccess,
		VolumeSizeInGB:       volumeSize,
		CreationTime:         now,
		LastModifiedTime:     now,
		Tags:                 tags,
		Lifecycle:            lc,
	}
	s.notebooks[name] = nb
	s.tagsByArn[nb.NotebookInstanceArn] = tags
	return nb, nil
}

func (s *Store) GetNotebookInstance(name string) (*NotebookInstance, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nb, ok := s.notebooks[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Notebook instance %s not found", name), http.StatusNotFound)
	}
	nb.Status = NotebookInstanceStatus(nb.Lifecycle.State())
	return nb, nil
}

func (s *Store) ListNotebookInstances() []*NotebookInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*NotebookInstance, 0, len(s.notebooks))
	for _, nb := range s.notebooks {
		nb.Status = NotebookInstanceStatus(nb.Lifecycle.State())
		out = append(out, nb)
	}
	return out
}

func (s *Store) DeleteNotebookInstance(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	nb, ok := s.notebooks[name]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Notebook instance %s not found", name), http.StatusNotFound)
	}
	status := NotebookInstanceStatus(nb.Lifecycle.State())
	if status != NotebookStopped && status != NotebookFailed {
		return service.NewAWSError("ResourceInUse",
			"Notebook instance must be stopped before deletion", http.StatusConflict)
	}
	nb.Lifecycle.Stop()
	delete(s.notebooks, name)
	delete(s.tagsByArn, nb.NotebookInstanceArn)
	return nil
}

func (s *Store) StartNotebookInstance(name string) *service.AWSError {
	s.mu.Lock()
	nb, ok := s.notebooks[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Notebook instance %s not found", name), http.StatusNotFound)
	}
	nb.LastModifiedTime = time.Now().UTC()
	lc := nb.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(NotebookPending))
	}
	return nil
}

func (s *Store) StopNotebookInstance(name string) *service.AWSError {
	s.mu.Lock()
	nb, ok := s.notebooks[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Notebook instance %s not found", name), http.StatusNotFound)
	}
	nb.LastModifiedTime = time.Now().UTC()
	lc := nb.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(NotebookStopped))
	}
	return nil
}

func (s *Store) UpdateNotebookInstance(name, instanceType, roleArn string, volumeSize int) (*NotebookInstance, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nb, ok := s.notebooks[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Notebook instance %s not found", name), http.StatusNotFound)
	}
	if instanceType != "" {
		nb.InstanceType = instanceType
	}
	if roleArn != "" {
		nb.RoleArn = roleArn
	}
	if volumeSize > 0 {
		nb.VolumeSizeInGB = volumeSize
	}
	nb.LastModifiedTime = time.Now().UTC()
	return nb, nil
}

// Training jobs.

func (s *Store) CreateTrainingJob(name string, algorithmSpec map[string]any, roleArn string, inputDataConfig []map[string]any, outputDataConfig, resourceConfig, stoppingCondition map[string]any, hyperParams map[string]string, tags map[string]string) (*TrainingJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.trainingJobs[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Training job %s already exists", name), http.StatusConflict)
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	now := time.Now().UTC()
	startTime := now

	lc := lifecycle.NewMachine(
		lifecycle.State(TrainingInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(TrainingInProgress), To: lifecycle.State(TrainingCompleted), Delay: 3 * time.Second},
		},
		s.lcConfig,
	)

	tj := &TrainingJob{
		TrainingJobName:        name,
		TrainingJobArn:         s.trainingJobARN(name),
		TrainingJobStatus:      TrainingJobStatus(lc.State()),
		SecondaryStatus:        "Starting",
		AlgorithmSpecification: algorithmSpec,
		RoleArn:                roleArn,
		InputDataConfig:        inputDataConfig,
		OutputDataConfig:       outputDataConfig,
		ResourceConfig:         resourceConfig,
		StoppingCondition:      stoppingCondition,
		HyperParameters:        hyperParams,
		CreationTime:           now,
		TrainingStartTime:      &startTime,
		LastModifiedTime:       now,
		ModelArtifacts:         s.generateModelArtifacts(name, outputDataConfig),
		Tags:                   tags,
		Lifecycle:              lc,
	}
	s.trainingJobs[name] = tj
	s.tagsByArn[tj.TrainingJobArn] = tags
	return tj, nil
}

func (s *Store) GetTrainingJob(name string) (*TrainingJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tj, ok := s.trainingJobs[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Training job %s not found", name), http.StatusNotFound)
	}
	tj.TrainingJobStatus = TrainingJobStatus(tj.Lifecycle.State())
	if tj.TrainingJobStatus == TrainingCompleted && tj.TrainingEndTime == nil {
		now := time.Now().UTC()
		tj.TrainingEndTime = &now
		tj.SecondaryStatus = "Completed"
	}
	return tj, nil
}

func (s *Store) ListTrainingJobs() []*TrainingJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TrainingJob, 0, len(s.trainingJobs))
	for _, tj := range s.trainingJobs {
		tj.TrainingJobStatus = TrainingJobStatus(tj.Lifecycle.State())
		out = append(out, tj)
	}
	return out
}

func (s *Store) StopTrainingJob(name string) *service.AWSError {
	s.mu.Lock()
	tj, ok := s.trainingJobs[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Training job %s not found", name), http.StatusNotFound)
	}
	now := time.Now().UTC()
	tj.TrainingEndTime = &now
	tj.SecondaryStatus = "Stopped"
	lc := tj.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(TrainingStopped))
	}
	return nil
}

// Models.

func (s *Store) CreateModel(name string, primaryContainer map[string]any, roleArn string, tags map[string]string) (*Model, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.models[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Model %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	m := &Model{
		ModelName:        name,
		ModelArn:         s.modelARN(name),
		PrimaryContainer: primaryContainer,
		ExecutionRoleArn: roleArn,
		CreationTime:     time.Now().UTC(),
		Tags:             tags,
	}
	s.models[name] = m
	s.tagsByArn[m.ModelArn] = tags
	return m, nil
}

func (s *Store) GetModel(name string) (*Model, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.models[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Model %s not found", name), http.StatusNotFound)
	}
	return m, nil
}

func (s *Store) ListModels() []*Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Model, 0, len(s.models))
	for _, m := range s.models {
		out = append(out, m)
	}
	return out
}

func (s *Store) DeleteModel(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.models[name]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Model %s not found", name), http.StatusNotFound)
	}
	delete(s.models, name)
	delete(s.tagsByArn, m.ModelArn)
	return nil
}

// Endpoint configs.

func (s *Store) CreateEndpointConfig(name string, productionVariants []map[string]any, tags map[string]string) (*EndpointConfig, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.endpointConfigs[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Endpoint config %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	ec := &EndpointConfig{
		EndpointConfigName: name,
		EndpointConfigArn:  s.endpointConfigARN(name),
		ProductionVariants: productionVariants,
		CreationTime:       time.Now().UTC(),
		Tags:               tags,
	}
	s.endpointConfigs[name] = ec
	s.tagsByArn[ec.EndpointConfigArn] = tags
	return ec, nil
}

func (s *Store) GetEndpointConfig(name string) (*EndpointConfig, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ec, ok := s.endpointConfigs[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint config %s not found", name), http.StatusNotFound)
	}
	return ec, nil
}

func (s *Store) ListEndpointConfigs() []*EndpointConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*EndpointConfig, 0, len(s.endpointConfigs))
	for _, ec := range s.endpointConfigs {
		out = append(out, ec)
	}
	return out
}

func (s *Store) DeleteEndpointConfig(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ec, ok := s.endpointConfigs[name]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint config %s not found", name), http.StatusNotFound)
	}
	delete(s.endpointConfigs, name)
	delete(s.tagsByArn, ec.EndpointConfigArn)
	return nil
}

// Endpoints.

func (s *Store) CreateEndpoint(name, configName string, tags map[string]string) (*Endpoint, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.endpoints[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Endpoint %s already exists", name), http.StatusConflict)
	}
	if _, ok := s.endpointConfigs[configName]; !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint config %s not found", configName), http.StatusNotFound)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	lc := lifecycle.NewMachine(
		lifecycle.State(EndpointCreating),
		[]lifecycle.Transition{
			{From: lifecycle.State(EndpointCreating), To: lifecycle.State(EndpointInService), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)
	ep := &Endpoint{
		EndpointName:       name,
		EndpointArn:        s.endpointARN(name),
		EndpointConfigName: configName,
		EndpointStatus:     EndpointStatus(lc.State()),
		CreationTime:       now,
		LastModifiedTime:   now,
		Tags:               tags,
		Lifecycle:          lc,
	}
	s.endpoints[name] = ep
	s.tagsByArn[ep.EndpointArn] = tags
	return ep, nil
}

func (s *Store) GetEndpoint(name string) (*Endpoint, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.endpoints[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint %s not found", name), http.StatusNotFound)
	}
	ep.EndpointStatus = EndpointStatus(ep.Lifecycle.State())
	return ep, nil
}

func (s *Store) ListEndpoints() []*Endpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Endpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		ep.EndpointStatus = EndpointStatus(ep.Lifecycle.State())
		out = append(out, ep)
	}
	return out
}

func (s *Store) DeleteEndpoint(name string) *service.AWSError {
	s.mu.Lock()
	ep, ok := s.endpoints[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint %s not found", name), http.StatusNotFound)
	}
	delete(s.endpoints, name)
	delete(s.tagsByArn, ep.EndpointArn)
	lc := ep.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(EndpointDeleting))
	}
	return nil
}

func (s *Store) UpdateEndpoint(name, configName string) (*Endpoint, *service.AWSError) {
	s.mu.Lock()
	ep, ok := s.endpoints[name]
	if !ok {
		s.mu.Unlock()
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint %s not found", name), http.StatusNotFound)
	}
	if _, ok := s.endpointConfigs[configName]; !ok {
		s.mu.Unlock()
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Endpoint config %s not found", configName), http.StatusNotFound)
	}
	ep.EndpointConfigName = configName
	ep.LastModifiedTime = time.Now().UTC()
	lc := ep.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(EndpointUpdating))
	}
	return ep, nil
}

// Processing jobs.

func (s *Store) CreateProcessingJob(name, roleArn string, appSpec, resources map[string]any, inputs []map[string]any, outputConfig, stoppingCondition map[string]any, tags map[string]string) (*ProcessingJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.processingJobs[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Processing job %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	startTime := now
	lc := lifecycle.NewMachine(
		lifecycle.State(ProcessingInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(ProcessingInProgress), To: lifecycle.State(ProcessingCompleted), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)
	pj := &ProcessingJob{
		ProcessingJobName:      name,
		ProcessingJobArn:       s.processingJobARN(name),
		ProcessingJobStatus:    ProcessingJobStatus(lc.State()),
		RoleArn:                roleArn,
		AppSpecification:       appSpec,
		ProcessingResources:    resources,
		ProcessingInputs:       inputs,
		ProcessingOutputConfig: outputConfig,
		StoppingCondition:      stoppingCondition,
		CreationTime:           now,
		ProcessingStartTime:    &startTime,
		Tags:                   tags,
		Lifecycle:              lc,
	}
	s.processingJobs[name] = pj
	s.tagsByArn[pj.ProcessingJobArn] = tags
	return pj, nil
}

func (s *Store) GetProcessingJob(name string) (*ProcessingJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pj, ok := s.processingJobs[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Processing job %s not found", name), http.StatusNotFound)
	}
	pj.ProcessingJobStatus = ProcessingJobStatus(pj.Lifecycle.State())
	if pj.ProcessingJobStatus == ProcessingCompleted && pj.ProcessingEndTime == nil {
		now := time.Now().UTC()
		pj.ProcessingEndTime = &now
	}
	return pj, nil
}

func (s *Store) ListProcessingJobs() []*ProcessingJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ProcessingJob, 0, len(s.processingJobs))
	for _, pj := range s.processingJobs {
		pj.ProcessingJobStatus = ProcessingJobStatus(pj.Lifecycle.State())
		out = append(out, pj)
	}
	return out
}

func (s *Store) StopProcessingJob(name string) *service.AWSError {
	s.mu.Lock()
	pj, ok := s.processingJobs[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Processing job %s not found", name), http.StatusNotFound)
	}
	now := time.Now().UTC()
	pj.ProcessingEndTime = &now
	lc := pj.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(ProcessingStopped))
	}
	return nil
}

// Transform jobs.

func (s *Store) CreateTransformJob(name, modelName string, input, output, resources map[string]any, tags map[string]string) (*TransformJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.transformJobs[name]; exists {
		return nil, service.NewAWSError("ResourceInUse",
			fmt.Sprintf("Transform job %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	startTime := now
	lc := lifecycle.NewMachine(
		lifecycle.State(TransformInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(TransformInProgress), To: lifecycle.State(TransformCompleted), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)
	tj := &TransformJob{
		TransformJobName:   name,
		TransformJobArn:    s.transformJobARN(name),
		TransformJobStatus: TransformJobStatus(lc.State()),
		ModelName:          modelName,
		TransformInput:     input,
		TransformOutput:    output,
		TransformResources: resources,
		CreationTime:       now,
		TransformStartTime: &startTime,
		Tags:               tags,
		Lifecycle:          lc,
	}
	s.transformJobs[name] = tj
	s.tagsByArn[tj.TransformJobArn] = tags
	return tj, nil
}

func (s *Store) GetTransformJob(name string) (*TransformJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tj, ok := s.transformJobs[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Transform job %s not found", name), http.StatusNotFound)
	}
	tj.TransformJobStatus = TransformJobStatus(tj.Lifecycle.State())
	if tj.TransformJobStatus == TransformCompleted && tj.TransformEndTime == nil {
		now := time.Now().UTC()
		tj.TransformEndTime = &now
	}
	return tj, nil
}

func (s *Store) ListTransformJobs() []*TransformJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TransformJob, 0, len(s.transformJobs))
	for _, tj := range s.transformJobs {
		tj.TransformJobStatus = TransformJobStatus(tj.Lifecycle.State())
		out = append(out, tj)
	}
	return out
}

func (s *Store) StopTransformJob(name string) *service.AWSError {
	s.mu.Lock()
	tj, ok := s.transformJobs[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Transform job %s not found", name), http.StatusNotFound)
	}
	now := time.Now().UTC()
	tj.TransformEndTime = &now
	lc := tj.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(TransformStopped))
	}
	return nil
}

// Tags.

func (s *Store) AddTags(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) DeleteTags(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for _, k := range tagKeys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTags(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}
