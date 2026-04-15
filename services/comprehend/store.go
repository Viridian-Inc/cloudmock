package comprehend

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ────────────────────────────────────────────────────────────

// StoredDocumentClassifier is a persisted document classifier.
type StoredDocumentClassifier struct {
	Arn                  string
	Name                 string
	VersionName          string
	Status               string
	Message              string
	SubmitTime           time.Time
	EndTime              time.Time
	TrainingStartTime    time.Time
	TrainingEndTime      time.Time
	InputDataConfig      map[string]any
	OutputDataConfig     map[string]any
	ClassifierMetadata   map[string]any
	DataAccessRoleArn    string
	LanguageCode         string
	VolumeKmsKeyId       string
	ModelKmsKeyId        string
	VpcConfig            map[string]any
	Mode                 string
	SourceModelArn       string
	FlywheelArn          string
	Tags                 map[string]string
}

// StoredEntityRecognizer is a persisted entity recognizer.
type StoredEntityRecognizer struct {
	Arn                string
	Name               string
	VersionName        string
	Status             string
	Message            string
	SubmitTime         time.Time
	EndTime            time.Time
	TrainingStartTime  time.Time
	TrainingEndTime    time.Time
	InputDataConfig    map[string]any
	RecognizerMetadata map[string]any
	DataAccessRoleArn  string
	LanguageCode       string
	VolumeKmsKeyId     string
	ModelKmsKeyId      string
	VpcConfig          map[string]any
	SourceModelArn     string
	FlywheelArn        string
	Tags               map[string]string
}

// StoredEndpoint is a persisted real-time endpoint.
type StoredEndpoint struct {
	Arn                      string
	Name                     string
	Status                   string
	Message                  string
	ModelArn                 string
	DesiredModelArn          string
	DesiredInferenceUnits    int
	CurrentInferenceUnits    int
	CreationTime             time.Time
	LastModifiedTime         time.Time
	DataAccessRoleArn        string
	DesiredDataAccessRoleArn string
	FlywheelArn              string
	Tags                     map[string]string
}

// StoredFlywheel is a persisted flywheel.
type StoredFlywheel struct {
	Arn                       string
	Name                      string
	ActiveModelArn            string
	DataAccessRoleArn         string
	TaskConfig                map[string]any
	DataLakeS3Uri             string
	DataSecurityConfig        map[string]any
	Status                    string
	ModelType                 string
	Message                   string
	CreationTime              time.Time
	LastModifiedTime          time.Time
	LatestFlywheelIteration   string
	Iterations                map[string]*StoredFlywheelIteration
	Datasets                  map[string]*StoredFlywheelDataset
	Tags                      map[string]string
}

// StoredFlywheelIteration is a single iteration of a flywheel.
type StoredFlywheelIteration struct {
	WorkflowArn                string
	FlywheelArn                string
	IterationId                string
	CreationTime               time.Time
	EndTime                    time.Time
	Status                     string
	Message                    string
	EvaluatedModelArn          string
	EvaluatedModelMetrics      map[string]any
	TrainedModelArn            string
	TrainedModelMetrics        map[string]any
	EvaluationManifestS3Prefix string
}

// StoredFlywheelDataset is a dataset attached to a flywheel.
type StoredFlywheelDataset struct {
	Arn               string
	FlywheelArn       string
	Name              string
	Type              string
	Status            string
	Message           string
	NumberOfDocuments int64
	CreationTime      time.Time
	EndTime           time.Time
	Description       string
	InputDataConfig   map[string]any
}

// StoredJob is a persisted async detection/classification/topic job.
type StoredJob struct {
	JobId                    string
	JobArn                   string
	JobName                  string
	JobStatus                string
	Message                  string
	SubmitTime               time.Time
	EndTime                  time.Time
	InputDataConfig          map[string]any
	OutputDataConfig         map[string]any
	DataAccessRoleArn        string
	LanguageCode             string
	VolumeKmsKeyId           string
	VpcConfig                map[string]any
	ClientRequestToken       string
	FlywheelArn              string
	ModelKmsKeyId            string
	NumberOfTopics           int
	TargetEventTypes         []string
	Mode                     string
	RedactionConfig          map[string]any
	ModelArn                 string
	EntityRecognizerArn      string
	DocumentClassifierArn    string
}

// StoredResourcePolicy is a per-ARN resource policy.
type StoredResourcePolicy struct {
	ResourceArn      string
	PolicyText       string
	CreationTime     time.Time
	LastModifiedTime time.Time
	PolicyRevisionId string
}

// ── Store ───────────────────────────────────────────────────────────────────

// Store is the in-memory data store for comprehend resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	classifiers map[string]*StoredDocumentClassifier // arn -> classifier
	recognizers map[string]*StoredEntityRecognizer   // arn -> recognizer
	endpoints   map[string]*StoredEndpoint           // arn -> endpoint
	flywheels   map[string]*StoredFlywheel           // arn -> flywheel
	datasets    map[string]*StoredFlywheelDataset    // arn -> dataset

	// jobs grouped by job kind
	jobs map[string]map[string]*StoredJob // kind -> jobId -> job

	policies map[string]*StoredResourcePolicy // resource arn -> policy
	tags     map[string]map[string]string     // arn -> tags
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		accountID: accountID,
		region:    region,
	}
	s.reset()
	return s
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reset()
}

func (s *Store) reset() {
	s.classifiers = make(map[string]*StoredDocumentClassifier)
	s.recognizers = make(map[string]*StoredEntityRecognizer)
	s.endpoints = make(map[string]*StoredEndpoint)
	s.flywheels = make(map[string]*StoredFlywheel)
	s.datasets = make(map[string]*StoredFlywheelDataset)
	s.jobs = make(map[string]map[string]*StoredJob)
	s.policies = make(map[string]*StoredResourcePolicy)
	s.tags = make(map[string]map[string]string)
}

// ── ARN builders ────────────────────────────────────────────────────────────

func (s *Store) classifierArn(name, version string) string {
	if version == "" {
		return fmt.Sprintf("arn:aws:comprehend:%s:%s:document-classifier/%s", s.region, s.accountID, name)
	}
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:document-classifier/%s/version/%s", s.region, s.accountID, name, version)
}

func (s *Store) recognizerArn(name, version string) string {
	if version == "" {
		return fmt.Sprintf("arn:aws:comprehend:%s:%s:entity-recognizer/%s", s.region, s.accountID, name)
	}
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:entity-recognizer/%s/version/%s", s.region, s.accountID, name, version)
}

func (s *Store) endpointArn(name string) string {
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:endpoint/%s", s.region, s.accountID, name)
}

func (s *Store) flywheelArn(name string) string {
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:flywheel/%s", s.region, s.accountID, name)
}

func (s *Store) datasetArn(flywheelName, datasetName string) string {
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:flywheel/%s/dataset/%s", s.region, s.accountID, flywheelName, datasetName)
}

func (s *Store) jobArn(kind, jobID string) string {
	return fmt.Sprintf("arn:aws:comprehend:%s:%s:%s/%s", s.region, s.accountID, kind, jobID)
}

// ── Document classifiers ────────────────────────────────────────────────────

// CreateDocumentClassifier persists a new classifier.
func (s *Store) CreateDocumentClassifier(c *StoredDocumentClassifier) (*StoredDocumentClassifier, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.Name == "" {
		return nil, service.ErrValidation("DocumentClassifierName is required.")
	}
	c.Arn = s.classifierArn(c.Name, c.VersionName)
	if _, ok := s.classifiers[c.Arn]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			"Classifier already exists: "+c.Arn, 400)
	}
	now := time.Now().UTC()
	if c.SubmitTime.IsZero() {
		c.SubmitTime = now
	}
	if c.Status == "" {
		c.Status = "TRAINED"
	}
	if c.EndTime.IsZero() && c.Status == "TRAINED" {
		c.EndTime = now
	}
	if c.ClassifierMetadata == nil {
		c.ClassifierMetadata = map[string]any{
			"NumberOfLabels":            2,
			"NumberOfTrainedDocuments":  100,
			"NumberOfTestDocuments":     10,
			"EvaluationMetrics": map[string]any{
				"Accuracy":       0.95,
				"Precision":      0.94,
				"Recall":         0.93,
				"F1Score":        0.935,
				"MicroPrecision": 0.94,
				"MicroRecall":    0.93,
				"MicroF1Score":   0.935,
				"HammingLoss":    0.05,
			},
		}
	}
	s.classifiers[c.Arn] = c
	return c, nil
}

// GetDocumentClassifier returns the classifier with the given ARN.
func (s *Store) GetDocumentClassifier(arn string) (*StoredDocumentClassifier, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.classifiers[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"DocumentClassifier not found: "+arn, 404)
	}
	return c, nil
}

// DeleteDocumentClassifier removes the classifier with the given ARN.
func (s *Store) DeleteDocumentClassifier(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.classifiers[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"DocumentClassifier not found: "+arn, 404)
	}
	// guard against in-use endpoints
	for _, ep := range s.endpoints {
		if ep.ModelArn == arn {
			return service.NewAWSError("ResourceInUseException",
				"Classifier in use by endpoint: "+ep.Arn, 400)
		}
	}
	delete(s.classifiers, arn)
	delete(s.tags, arn)
	return nil
}

// ListDocumentClassifiers returns all classifiers.
func (s *Store) ListDocumentClassifiers() []*StoredDocumentClassifier {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredDocumentClassifier, 0, len(s.classifiers))
	for _, c := range s.classifiers {
		out = append(out, c)
	}
	return out
}

// StopTrainingDocumentClassifier transitions a classifier to STOP_REQUESTED.
func (s *Store) StopTrainingDocumentClassifier(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.classifiers[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"DocumentClassifier not found: "+arn, 404)
	}
	c.Status = "STOP_REQUESTED"
	return nil
}

// ── Entity recognizers ──────────────────────────────────────────────────────

// CreateEntityRecognizer persists a new recognizer.
func (s *Store) CreateEntityRecognizer(r *StoredEntityRecognizer) (*StoredEntityRecognizer, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Name == "" {
		return nil, service.ErrValidation("RecognizerName is required.")
	}
	r.Arn = s.recognizerArn(r.Name, r.VersionName)
	if _, ok := s.recognizers[r.Arn]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			"Recognizer already exists: "+r.Arn, 400)
	}
	now := time.Now().UTC()
	if r.SubmitTime.IsZero() {
		r.SubmitTime = now
	}
	if r.Status == "" {
		r.Status = "TRAINED"
	}
	if r.EndTime.IsZero() && r.Status == "TRAINED" {
		r.EndTime = now
	}
	if r.RecognizerMetadata == nil {
		r.RecognizerMetadata = map[string]any{
			"NumberOfTrainedDocuments": 100,
			"NumberOfTestDocuments":    10,
			"EvaluationMetrics": map[string]any{
				"Precision": 0.92,
				"Recall":    0.91,
				"F1Score":   0.915,
			},
			"EntityTypes": []map[string]any{},
		}
	}
	s.recognizers[r.Arn] = r
	return r, nil
}

// GetEntityRecognizer returns the recognizer with the given ARN.
func (s *Store) GetEntityRecognizer(arn string) (*StoredEntityRecognizer, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.recognizers[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"EntityRecognizer not found: "+arn, 404)
	}
	return r, nil
}

// DeleteEntityRecognizer removes the recognizer with the given ARN.
func (s *Store) DeleteEntityRecognizer(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.recognizers[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"EntityRecognizer not found: "+arn, 404)
	}
	for _, ep := range s.endpoints {
		if ep.ModelArn == arn {
			return service.NewAWSError("ResourceInUseException",
				"Recognizer in use by endpoint: "+ep.Arn, 400)
		}
	}
	delete(s.recognizers, arn)
	delete(s.tags, arn)
	return nil
}

// ListEntityRecognizers returns all recognizers.
func (s *Store) ListEntityRecognizers() []*StoredEntityRecognizer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredEntityRecognizer, 0, len(s.recognizers))
	for _, r := range s.recognizers {
		out = append(out, r)
	}
	return out
}

// StopTrainingEntityRecognizer transitions a recognizer to STOP_REQUESTED.
func (s *Store) StopTrainingEntityRecognizer(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.recognizers[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"EntityRecognizer not found: "+arn, 404)
	}
	r.Status = "STOP_REQUESTED"
	return nil
}

// ── Endpoints ───────────────────────────────────────────────────────────────

// CreateEndpoint persists a new endpoint.
func (s *Store) CreateEndpoint(e *StoredEndpoint) (*StoredEndpoint, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Name == "" {
		return nil, service.ErrValidation("EndpointName is required.")
	}
	e.Arn = s.endpointArn(e.Name)
	if _, ok := s.endpoints[e.Arn]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			"Endpoint already exists: "+e.Arn, 400)
	}
	now := time.Now().UTC()
	if e.CreationTime.IsZero() {
		e.CreationTime = now
	}
	e.LastModifiedTime = now
	if e.Status == "" {
		e.Status = "IN_SERVICE"
	}
	if e.CurrentInferenceUnits == 0 {
		e.CurrentInferenceUnits = e.DesiredInferenceUnits
	}
	s.endpoints[e.Arn] = e
	return e, nil
}

// GetEndpoint returns the endpoint with the given ARN.
func (s *Store) GetEndpoint(arn string) (*StoredEndpoint, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.endpoints[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Endpoint not found: "+arn, 404)
	}
	return e, nil
}

// DeleteEndpoint removes the endpoint with the given ARN.
func (s *Store) DeleteEndpoint(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Endpoint not found: "+arn, 404)
	}
	delete(s.endpoints, arn)
	delete(s.tags, arn)
	delete(s.policies, arn)
	return nil
}

// ListEndpoints returns all endpoints.
func (s *Store) ListEndpoints() []*StoredEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredEndpoint, 0, len(s.endpoints))
	for _, e := range s.endpoints {
		out = append(out, e)
	}
	return out
}

// UpdateEndpoint updates desired fields on an endpoint.
func (s *Store) UpdateEndpoint(arn string, desiredModelArn, desiredDataAccessRoleArn, flywheelArn string, desiredInferenceUnits int) (*StoredEndpoint, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.endpoints[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Endpoint not found: "+arn, 404)
	}
	if desiredModelArn != "" {
		e.DesiredModelArn = desiredModelArn
		e.ModelArn = desiredModelArn
	}
	if desiredDataAccessRoleArn != "" {
		e.DesiredDataAccessRoleArn = desiredDataAccessRoleArn
	}
	if flywheelArn != "" {
		e.FlywheelArn = flywheelArn
	}
	if desiredInferenceUnits > 0 {
		e.DesiredInferenceUnits = desiredInferenceUnits
		e.CurrentInferenceUnits = desiredInferenceUnits
	}
	e.LastModifiedTime = time.Now().UTC()
	return e, nil
}

// ── Flywheels ───────────────────────────────────────────────────────────────

// CreateFlywheel persists a new flywheel.
func (s *Store) CreateFlywheel(f *StoredFlywheel) (*StoredFlywheel, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Name == "" {
		return nil, service.ErrValidation("FlywheelName is required.")
	}
	f.Arn = s.flywheelArn(f.Name)
	if _, ok := s.flywheels[f.Arn]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			"Flywheel already exists: "+f.Arn, 400)
	}
	now := time.Now().UTC()
	f.CreationTime = now
	f.LastModifiedTime = now
	if f.Status == "" {
		f.Status = "ACTIVE"
	}
	if f.Iterations == nil {
		f.Iterations = make(map[string]*StoredFlywheelIteration)
	}
	if f.Datasets == nil {
		f.Datasets = make(map[string]*StoredFlywheelDataset)
	}
	s.flywheels[f.Arn] = f
	return f, nil
}

// GetFlywheel returns the flywheel with the given ARN.
func (s *Store) GetFlywheel(arn string) (*StoredFlywheel, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flywheels[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	return f, nil
}

// DeleteFlywheel removes the flywheel with the given ARN.
func (s *Store) DeleteFlywheel(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flywheels[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	delete(s.flywheels, arn)
	delete(s.tags, arn)
	// drop all datasets attached to this flywheel
	for k, ds := range s.datasets {
		if ds.FlywheelArn == arn {
			delete(s.datasets, k)
		}
	}
	return nil
}

// ListFlywheels returns all flywheels.
func (s *Store) ListFlywheels() []*StoredFlywheel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFlywheel, 0, len(s.flywheels))
	for _, f := range s.flywheels {
		out = append(out, f)
	}
	return out
}

// UpdateFlywheel updates fields on a flywheel.
func (s *Store) UpdateFlywheel(arn, activeModelArn, dataAccessRoleArn string, dataSecurityConfig map[string]any) (*StoredFlywheel, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flywheels[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	if activeModelArn != "" {
		f.ActiveModelArn = activeModelArn
	}
	if dataAccessRoleArn != "" {
		f.DataAccessRoleArn = dataAccessRoleArn
	}
	if dataSecurityConfig != nil {
		f.DataSecurityConfig = dataSecurityConfig
	}
	f.LastModifiedTime = time.Now().UTC()
	return f, nil
}

// StartFlywheelIteration starts a new iteration for the given flywheel.
func (s *Store) StartFlywheelIteration(arn string) (*StoredFlywheelIteration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.flywheels[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	now := time.Now().UTC()
	id := newID()
	it := &StoredFlywheelIteration{
		FlywheelArn:                arn,
		WorkflowArn:                fmt.Sprintf("%s/iteration/%s", arn, id),
		IterationId:                id,
		CreationTime:               now,
		EndTime:                    now,
		Status:                     "COMPLETED",
		EvaluatedModelArn:          f.ActiveModelArn,
		EvaluatedModelMetrics:      map[string]any{"AverageF1Score": 0.92},
		TrainedModelArn:            f.ActiveModelArn,
		TrainedModelMetrics:        map[string]any{"AverageF1Score": 0.93},
		EvaluationManifestS3Prefix: f.DataLakeS3Uri + "/iterations/" + id + "/",
	}
	f.Iterations[id] = it
	f.LatestFlywheelIteration = id
	return it, nil
}

// GetFlywheelIteration returns an iteration by flywheel ARN and iteration ID.
func (s *Store) GetFlywheelIteration(arn, iterationID string) (*StoredFlywheelIteration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flywheels[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	it, ok := f.Iterations[iterationID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"FlywheelIteration not found: "+iterationID, 404)
	}
	return it, nil
}

// ListFlywheelIterations returns all iterations for the given flywheel.
func (s *Store) ListFlywheelIterations(arn string) ([]*StoredFlywheelIteration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flywheels[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+arn, 404)
	}
	out := make([]*StoredFlywheelIteration, 0, len(f.Iterations))
	for _, it := range f.Iterations {
		out = append(out, it)
	}
	return out, nil
}

// ── Flywheel datasets ───────────────────────────────────────────────────────

// CreateDataset persists a new dataset attached to a flywheel.
func (s *Store) CreateDataset(d *StoredFlywheelDataset) (*StoredFlywheelDataset, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.Name == "" {
		return nil, service.ErrValidation("DatasetName is required.")
	}
	if d.FlywheelArn == "" {
		return nil, service.ErrValidation("FlywheelArn is required.")
	}
	f, ok := s.flywheels[d.FlywheelArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Flywheel not found: "+d.FlywheelArn, 404)
	}
	d.Arn = s.datasetArn(f.Name, d.Name)
	if _, ok := s.datasets[d.Arn]; ok {
		return nil, service.NewAWSError("ResourceInUseException",
			"Dataset already exists: "+d.Arn, 400)
	}
	now := time.Now().UTC()
	d.CreationTime = now
	d.EndTime = now
	if d.Status == "" {
		d.Status = "COMPLETED"
	}
	if d.Type == "" {
		d.Type = "TRAIN"
	}
	if d.NumberOfDocuments == 0 {
		d.NumberOfDocuments = 100
	}
	s.datasets[d.Arn] = d
	f.Datasets[d.Arn] = d
	return d, nil
}

// GetDataset returns a dataset by ARN.
func (s *Store) GetDataset(arn string) (*StoredFlywheelDataset, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.datasets[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Dataset not found: "+arn, 404)
	}
	return d, nil
}

// ListDatasets returns datasets, optionally filtered by flywheel ARN.
func (s *Store) ListDatasets(flywheelArn string) []*StoredFlywheelDataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFlywheelDataset, 0, len(s.datasets))
	for _, d := range s.datasets {
		if flywheelArn != "" && d.FlywheelArn != flywheelArn {
			continue
		}
		out = append(out, d)
	}
	return out
}

// ── Async jobs ──────────────────────────────────────────────────────────────

// CreateJob creates a new job of the given kind with COMPLETED status.
func (s *Store) CreateJob(kind string, j *StoredJob) *StoredJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.JobId == "" {
		j.JobId = newID()
	}
	j.JobArn = s.jobArn(kind, j.JobId)
	now := time.Now().UTC()
	if j.SubmitTime.IsZero() {
		j.SubmitTime = now
	}
	if j.JobStatus == "" {
		j.JobStatus = "COMPLETED"
	}
	if j.EndTime.IsZero() && j.JobStatus == "COMPLETED" {
		j.EndTime = now
	}
	if s.jobs[kind] == nil {
		s.jobs[kind] = make(map[string]*StoredJob)
	}
	s.jobs[kind][j.JobId] = j
	return j
}

// GetJob returns a job by kind and ID.
func (s *Store) GetJob(kind, id string) (*StoredJob, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.jobs[kind]; ok {
		if j, ok := m[id]; ok {
			return j, nil
		}
	}
	return nil, service.NewAWSError("JobNotFoundException",
		fmt.Sprintf("Job not found: %s/%s", kind, id), 404)
}

// ListJobs returns all jobs of the given kind.
func (s *Store) ListJobs(kind string) []*StoredJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.jobs[kind]
	out := make([]*StoredJob, 0, len(m))
	for _, j := range m {
		out = append(out, j)
	}
	return out
}

// StopJob transitions a job to STOP_REQUESTED.
func (s *Store) StopJob(kind, id string) (*StoredJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.jobs[kind]; ok {
		if j, ok := m[id]; ok {
			j.JobStatus = "STOP_REQUESTED"
			return j, nil
		}
	}
	return nil, service.NewAWSError("JobNotFoundException",
		fmt.Sprintf("Job not found: %s/%s", kind, id), 404)
}

// ── Resource policies ───────────────────────────────────────────────────────

// PutResourcePolicy creates or updates a resource policy.
func (s *Store) PutResourcePolicy(arn, policyText, expectedRevisionID string) (*StoredResourcePolicy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	p, ok := s.policies[arn]
	if ok {
		if expectedRevisionID != "" && p.PolicyRevisionId != expectedRevisionID {
			return nil, service.NewAWSError("InvalidRequestException",
				"PolicyRevisionId mismatch.", 400)
		}
		p.PolicyText = policyText
		p.LastModifiedTime = now
		p.PolicyRevisionId = newID()
		return p, nil
	}
	p = &StoredResourcePolicy{
		ResourceArn:      arn,
		PolicyText:       policyText,
		CreationTime:     now,
		LastModifiedTime: now,
		PolicyRevisionId: newID(),
	}
	s.policies[arn] = p
	return p, nil
}

// GetResourcePolicy returns the resource policy for the given ARN.
func (s *Store) GetResourcePolicy(arn string) (*StoredResourcePolicy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"ResourcePolicy not found: "+arn, 404)
	}
	return p, nil
}

// DeleteResourcePolicy removes the resource policy for the given ARN.
func (s *Store) DeleteResourcePolicy(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"ResourcePolicy not found: "+arn, 404)
	}
	delete(s.policies, arn)
	return nil
}

// ── Tags ────────────────────────────────────────────────────────────────────

// TagResource adds tags to the given ARN.
func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

// UntagResource removes the given keys from the ARN's tags.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns the tags for the given ARN.
func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.tags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
