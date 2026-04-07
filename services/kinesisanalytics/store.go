package kinesisanalytics

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// Application represents a Kinesis Analytics application.
type Application struct {
	Name                  string
	ARN                   string
	Description           string
	Status                string
	RuntimeEnvironment    string
	ServiceExecutionRole  string
	CreateTimestamp       time.Time
	LastUpdateTimestamp   time.Time
	ApplicationVersionId  int64
	Inputs                []Input
	Outputs               []Output
	Tags                  map[string]string
	Lifecycle             *lifecycle.Machine
}

// Input represents an application input configuration.
type Input struct {
	InputId              string
	NamePrefix           string
	InputSchema          InputSchema
	KinesisStreamsInput   *KinesisStreamsInput
}

// InputSchema holds input schema configuration.
type InputSchema struct {
	RecordFormat RecordFormat
	RecordColumns []RecordColumn
}

// RecordFormat describes the data format.
type RecordFormat struct {
	RecordFormatType string
}

// RecordColumn describes a column in the input schema.
type RecordColumn struct {
	Name    string
	SqlType string
	Mapping string
}

// KinesisStreamsInput holds Kinesis stream input config.
type KinesisStreamsInput struct {
	ResourceARN string
}

// Output represents an application output configuration.
type Output struct {
	OutputId    string
	Name        string
	DestinationSchema DestinationSchema
	KinesisStreamsOutput *KinesisStreamsOutput
}

// DestinationSchema holds output schema config.
type DestinationSchema struct {
	RecordFormatType string
}

// KinesisStreamsOutput holds Kinesis stream output config.
type KinesisStreamsOutput struct {
	ResourceARN string
}

// ApplicationSnapshot represents a snapshot of an application.
type ApplicationSnapshot struct {
	SnapshotName       string
	ApplicationName    string
	SnapshotStatus     string
	ApplicationVersionId int64
	SnapshotCreationTimestamp time.Time
}

// Store manages all Kinesis Analytics resources.
type Store struct {
	mu           sync.RWMutex
	applications map[string]*Application
	snapshots    map[string]map[string]*ApplicationSnapshot // appName -> snapshotName -> snapshot
	accountID    string
	region       string
	lcConfig     *lifecycle.Config
}

// NewStore creates a new Kinesis Analytics store.
func NewStore(accountID, region string) *Store {
	return &Store{
		applications: make(map[string]*Application),
		snapshots:    make(map[string]map[string]*ApplicationSnapshot),
		accountID:    accountID,
		region:       region,
		lcConfig:     lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) applicationARN(name string) string {
	return fmt.Sprintf("arn:aws:kinesisanalytics:%s:%s:application/%s", s.region, s.accountID, name)
}

func appTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "CREATING", To: "READY", Delay: 2 * time.Second},
		{From: "STARTING", To: "RUNNING", Delay: 3 * time.Second},
		{From: "STOPPING", To: "READY", Delay: 2 * time.Second},
		{From: "DELETING", To: "DELETED", Delay: 2 * time.Second},
	}
}

func (s *Store) CreateApplication(name, description, runtimeEnv, serviceRole string, tags map[string]string) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[name]; ok {
		return nil, false
	}
	now := time.Now().UTC()
	app := &Application{
		Name:                 name,
		ARN:                  s.applicationARN(name),
		Description:          description,
		RuntimeEnvironment:   runtimeEnv,
		ServiceExecutionRole: serviceRole,
		Status:               "READY",
		CreateTimestamp:      now,
		LastUpdateTimestamp:  now,
		ApplicationVersionId: 1,
		Inputs:               make([]Input, 0),
		Outputs:              make([]Output, 0),
		Tags:                 tags,
		Lifecycle:            lifecycle.NewMachine("CREATING", appTransitions(), s.lcConfig),
	}
	s.applications[name] = app
	s.snapshots[name] = make(map[string]*ApplicationSnapshot)
	return app, true
}

func (s *Store) GetApplication(name string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.applications[name]
	if ok {
		app.Status = string(app.Lifecycle.State())
	}
	return app, ok
}

func (s *Store) ListApplications() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Application, 0, len(s.applications))
	for _, app := range s.applications {
		app.Status = string(app.Lifecycle.State())
		result = append(result, app)
	}
	return result
}

func (s *Store) DeleteApplication(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[name]
	if !ok {
		return false
	}
	app.Lifecycle.Stop()
	delete(s.applications, name)
	delete(s.snapshots, name)
	return true
}

func (s *Store) UpdateApplication(name, description string) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[name]
	if !ok {
		return nil, false
	}
	if description != "" {
		app.Description = description
	}
	app.ApplicationVersionId++
	app.LastUpdateTimestamp = time.Now().UTC()
	return app, true
}

func (s *Store) StartApplication(name string) bool {
	s.mu.Lock()
	app, ok := s.applications[name]
	if !ok {
		s.mu.Unlock()
		return false
	}
	state := string(app.Lifecycle.State())
	if state != "READY" {
		s.mu.Unlock()
		return false
	}
	lc := app.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState("STARTING")
	}
	return true
}

func (s *Store) StopApplication(name string) bool {
	s.mu.Lock()
	app, ok := s.applications[name]
	if !ok {
		s.mu.Unlock()
		return false
	}
	state := string(app.Lifecycle.State())
	if state != "RUNNING" {
		s.mu.Unlock()
		return false
	}
	lc := app.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState("STOPPING")
	}
	return true
}

func (s *Store) AddInput(name string, input Input) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[name]
	if !ok {
		return nil, false
	}
	input.InputId = newUUID()
	app.Inputs = append(app.Inputs, input)
	app.ApplicationVersionId++
	app.LastUpdateTimestamp = time.Now().UTC()
	return app, true
}

func (s *Store) AddOutput(name string, output Output) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[name]
	if !ok {
		return nil, false
	}
	output.OutputId = newUUID()
	app.Outputs = append(app.Outputs, output)
	app.ApplicationVersionId++
	app.LastUpdateTimestamp = time.Now().UTC()
	return app, true
}

func (s *Store) DeleteOutput(name, outputID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[name]
	if !ok {
		return false
	}
	for i, o := range app.Outputs {
		if o.OutputId == outputID {
			app.Outputs = append(app.Outputs[:i], app.Outputs[i+1:]...)
			app.ApplicationVersionId++
			app.LastUpdateTimestamp = time.Now().UTC()
			return true
		}
	}
	return false
}

// ---- Snapshot operations ----

func (s *Store) CreateSnapshot(appName, snapshotName string) (*ApplicationSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[appName]
	if !ok {
		return nil, false
	}
	snaps := s.snapshots[appName]
	if _, exists := snaps[snapshotName]; exists {
		return nil, false
	}
	snap := &ApplicationSnapshot{
		SnapshotName:              snapshotName,
		ApplicationName:           appName,
		SnapshotStatus:            "READY",
		ApplicationVersionId:      app.ApplicationVersionId,
		SnapshotCreationTimestamp: time.Now().UTC(),
	}
	snaps[snapshotName] = snap
	return snap, true
}

func (s *Store) ListSnapshots(appName string) []*ApplicationSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snaps := s.snapshots[appName]
	result := make([]*ApplicationSnapshot, 0, len(snaps))
	for _, snap := range snaps {
		result = append(result, snap)
	}
	return result
}

func (s *Store) DeleteSnapshot(appName, snapshotName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	snaps := s.snapshots[appName]
	if snaps == nil {
		return false
	}
	if _, ok := snaps[snapshotName]; !ok {
		return false
	}
	delete(snaps, snapshotName)
	return true
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, app := range s.applications {
		if app.ARN == arn {
			for k, v := range tags {
				app.Tags[k] = v
			}
			return true
		}
	}
	return false
}

func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, app := range s.applications {
		if app.ARN == arn {
			for _, k := range keys {
				delete(app.Tags, k)
			}
			return true
		}
	}
	return false
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, app := range s.applications {
		if app.ARN == arn {
			result := make(map[string]string, len(app.Tags))
			for k, v := range app.Tags {
				result[k] = v
			}
			return result, true
		}
	}
	return nil, false
}
