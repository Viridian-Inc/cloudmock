package config

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ComplianceType represents the compliance state of a rule.
type ComplianceType string

const (
	ComplianceCompliant    ComplianceType = "COMPLIANT"
	ComplianceNonCompliant ComplianceType = "NON_COMPLIANT"
	ComplianceNotApplicable ComplianceType = "NOT_APPLICABLE"
	ComplianceInsufficientData ComplianceType = "INSUFFICIENT_DATA"
)

// ConfigRule holds a Config rule definition.
type ConfigRule struct {
	ConfigRuleName               string
	ConfigRuleArn                string
	ConfigRuleId                 string
	Description                  string
	Source                       RuleSource
	Scope                        *RuleScope
	InputParameters              string
	MaximumExecutionFrequency    string
	ConfigRuleState              string
	CreatedBy                    string
	EvaluationModes              []map[string]any
}

// RuleSource identifies the rule evaluator.
type RuleSource struct {
	Owner            string
	SourceIdentifier string
	SourceDetails    []map[string]any
}

// RuleScope narrows the resources a rule evaluates.
type RuleScope struct {
	ComplianceResourceTypes []string
	TagKey                  string
	TagValue                string
	ComplianceResourceId    string
}

// ConfigurationRecorder holds a recorder definition.
type ConfigurationRecorder struct {
	Name           string
	RoleARN        string
	RecordingGroup *RecordingGroup
	IsRecording    bool
	subscriptionID string // eventbus subscription ID (unexported)
}

// RecordingGroup controls which resource types are recorded.
type RecordingGroup struct {
	AllSupported               bool
	IncludeGlobalResourceTypes bool
	ResourceTypes              []string
}

// DeliveryChannel holds a delivery channel definition.
type DeliveryChannel struct {
	Name         string
	S3BucketName string
	S3KeyPrefix  string
	SnsTopicARN  string
	ConfigSnapshotDeliveryProperties *SnapshotDeliveryProperties
}

// SnapshotDeliveryProperties controls snapshot delivery frequency.
type SnapshotDeliveryProperties struct {
	DeliveryFrequency string
}

// ConformancePack holds a conformance pack definition.
type ConformancePack struct {
	ConformancePackName string
	ConformancePackArn  string
	ConformancePackId   string
	DeliveryS3Bucket    string
	DeliveryS3KeyPrefix string
	TemplateBody        string
	CreatedAt           time.Time
	LastUpdateRequestedTime time.Time
	ConformancePackState string
}

// EvaluationResult holds a compliance evaluation result.
type EvaluationResult struct {
	EvaluationResultIdentifier EvaluationResultIdentifier
	ComplianceType             ComplianceType
	ResultRecordedTime         time.Time
	ConfigRuleInvokedTime      time.Time
	Annotation                 string
}

// EvaluationResultIdentifier identifies which resource was evaluated.
type EvaluationResultIdentifier struct {
	EvaluationResultQualifier EvaluationResultQualifier
	OrderingTimestamp         time.Time
}

// EvaluationResultQualifier contains the rule and resource identifiers.
type EvaluationResultQualifier struct {
	ConfigRuleName string
	ResourceType   string
	ResourceId     string
}

// Store is the in-memory store for Config resources.
type Store struct {
	mu                sync.RWMutex
	rules             map[string]*ConfigRule
	recorders         map[string]*ConfigurationRecorder
	channels          map[string]*DeliveryChannel
	conformancePacks  map[string]*ConformancePack
	evaluationResults map[string][]EvaluationResult // keyed by rule name
	configItems       map[string][]ConfigurationItem // keyed by "resourceType:resourceId"
	accountID         string
	region            string
}

// NewStore creates an empty Config Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		rules:             make(map[string]*ConfigRule),
		recorders:         make(map[string]*ConfigurationRecorder),
		channels:          make(map[string]*DeliveryChannel),
		conformancePacks:  make(map[string]*ConformancePack),
		evaluationResults: make(map[string][]EvaluationResult),
		configItems:       make(map[string][]ConfigurationItem),
		accountID:         accountID,
		region:            region,
	}
}

// GetConfigHistory returns configuration items for a resource.
func (s *Store) GetConfigHistory(resourceType, resourceId string) []ConfigurationItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := resourceType + ":" + resourceId
	return s.configItems[key]
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) buildRuleARN(name string) string {
	return fmt.Sprintf("arn:aws:config:%s:%s:config-rule/%s", s.region, s.accountID, name)
}

func (s *Store) buildConformancePackARN(name string) string {
	return fmt.Sprintf("arn:aws:config:%s:%s:conformance-pack/%s", s.region, s.accountID, name)
}

// PutConfigRule creates or updates a config rule.
func (s *Store) PutConfigRule(rule *ConfigRule) (*ConfigRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ConfigRuleName == "" {
		return nil, service.ErrValidation("ConfigRuleName is required.")
	}

	existing, ok := s.rules[rule.ConfigRuleName]
	if ok {
		// Update existing rule
		existing.Description = rule.Description
		existing.Source = rule.Source
		existing.Scope = rule.Scope
		existing.InputParameters = rule.InputParameters
		existing.MaximumExecutionFrequency = rule.MaximumExecutionFrequency
		return existing, nil
	}

	rule.ConfigRuleId = newUUID()
	rule.ConfigRuleArn = s.buildRuleARN(rule.ConfigRuleName)
	if rule.ConfigRuleState == "" {
		rule.ConfigRuleState = "ACTIVE"
	}

	s.rules[rule.ConfigRuleName] = rule
	return rule, nil
}

// GetConfigRules returns rules by name or all rules if no names given.
func (s *Store) GetConfigRules(names []string) []*ConfigRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		out := make([]*ConfigRule, 0, len(s.rules))
		for _, r := range s.rules {
			out = append(out, r)
		}
		return out
	}

	out := make([]*ConfigRule, 0, len(names))
	for _, name := range names {
		if r, ok := s.rules[name]; ok {
			out = append(out, r)
		}
	}
	return out
}

// DeleteConfigRule removes a rule by name.
func (s *Store) DeleteConfigRule(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[name]; !ok {
		return service.NewAWSError("NoSuchConfigRuleException",
			fmt.Sprintf("ConfigRule %s not found.", name), http.StatusNotFound)
	}
	delete(s.rules, name)
	delete(s.evaluationResults, name)
	return nil
}

// PutConfigurationRecorder creates or updates a recorder.
func (s *Store) PutConfigurationRecorder(recorder *ConfigurationRecorder) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if recorder.Name == "" {
		recorder.Name = "default"
	}

	s.recorders[recorder.Name] = recorder
	return nil
}

// GetConfigurationRecorders returns all recorders.
func (s *Store) GetConfigurationRecorders() []*ConfigurationRecorder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ConfigurationRecorder, 0, len(s.recorders))
	for _, r := range s.recorders {
		out = append(out, r)
	}
	return out
}

// DeleteConfigurationRecorder removes a recorder.
func (s *Store) DeleteConfigurationRecorder(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.recorders[name]; !ok {
		return service.NewAWSError("NoSuchConfigurationRecorderException",
			fmt.Sprintf("Configuration recorder %s not found.", name), http.StatusNotFound)
	}
	delete(s.recorders, name)
	return nil
}

// StartRecorder starts recording.
func (s *Store) StartRecorder(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.recorders[name]
	if !ok {
		return service.NewAWSError("NoSuchConfigurationRecorderException",
			fmt.Sprintf("Configuration recorder %s not found.", name), http.StatusNotFound)
	}
	r.IsRecording = true
	return nil
}

// StopRecorder stops recording.
func (s *Store) StopRecorder(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.recorders[name]
	if !ok {
		return service.NewAWSError("NoSuchConfigurationRecorderException",
			fmt.Sprintf("Configuration recorder %s not found.", name), http.StatusNotFound)
	}
	r.IsRecording = false
	return nil
}

// PutDeliveryChannel creates or updates a delivery channel.
func (s *Store) PutDeliveryChannel(channel *DeliveryChannel) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if channel.Name == "" {
		channel.Name = "default"
	}
	if channel.S3BucketName == "" {
		return service.ErrValidation("S3BucketName is required for delivery channel.")
	}

	s.channels[channel.Name] = channel
	return nil
}

// GetDeliveryChannels returns all delivery channels.
func (s *Store) GetDeliveryChannels() []*DeliveryChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*DeliveryChannel, 0, len(s.channels))
	for _, c := range s.channels {
		out = append(out, c)
	}
	return out
}

// DeleteDeliveryChannel removes a delivery channel.
func (s *Store) DeleteDeliveryChannel(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.channels[name]; !ok {
		return service.NewAWSError("NoSuchDeliveryChannelException",
			fmt.Sprintf("Delivery channel %s not found.", name), http.StatusNotFound)
	}
	delete(s.channels, name)
	return nil
}

// PutConformancePack creates or updates a conformance pack.
func (s *Store) PutConformancePack(name, s3Bucket, s3Prefix, templateBody string) (*ConformancePack, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("ConformancePackName is required.")
	}

	now := time.Now().UTC()
	existing, ok := s.conformancePacks[name]
	if ok {
		existing.DeliveryS3Bucket = s3Bucket
		existing.DeliveryS3KeyPrefix = s3Prefix
		existing.TemplateBody = templateBody
		existing.LastUpdateRequestedTime = now
		return existing, nil
	}

	pack := &ConformancePack{
		ConformancePackName:         name,
		ConformancePackArn:          s.buildConformancePackARN(name),
		ConformancePackId:           newUUID(),
		DeliveryS3Bucket:            s3Bucket,
		DeliveryS3KeyPrefix:         s3Prefix,
		TemplateBody:                templateBody,
		CreatedAt:                   now,
		LastUpdateRequestedTime:     now,
		ConformancePackState:        "CREATE_COMPLETE",
	}

	s.conformancePacks[name] = pack
	return pack, nil
}

// GetConformancePacks returns conformance packs by name or all if no names given.
func (s *Store) GetConformancePacks(names []string) []*ConformancePack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		out := make([]*ConformancePack, 0, len(s.conformancePacks))
		for _, p := range s.conformancePacks {
			out = append(out, p)
		}
		return out
	}

	out := make([]*ConformancePack, 0, len(names))
	for _, name := range names {
		if p, ok := s.conformancePacks[name]; ok {
			out = append(out, p)
		}
	}
	return out
}

// DeleteConformancePack removes a conformance pack.
func (s *Store) DeleteConformancePack(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conformancePacks[name]; !ok {
		return service.NewAWSError("NoSuchConformancePackException",
			fmt.Sprintf("Conformance pack %s not found.", name), http.StatusNotFound)
	}
	delete(s.conformancePacks, name)
	return nil
}

// PutEvaluations stores evaluation results for a rule.
func (s *Store) PutEvaluations(ruleName string, evaluations []EvaluationResult) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evaluationResults[ruleName] = evaluations
	return nil
}

// GetComplianceByRule returns compliance results for a rule.
func (s *Store) GetComplianceByRule(ruleName string) []EvaluationResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.evaluationResults[ruleName]
}
