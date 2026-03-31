package textract

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

type JobStatus string

const (
	JobInProgress JobStatus = "IN_PROGRESS"
	JobSucceeded  JobStatus = "SUCCEEDED"
	JobFailed     JobStatus = "FAILED"
)

type JobType string

const (
	JobTextDetection  JobType = "TEXT_DETECTION"
	JobDocAnalysis    JobType = "DOCUMENT_ANALYSIS"
	JobExpenseAnalysis JobType = "EXPENSE_ANALYSIS"
)

type Block struct {
	BlockType    string         `json:"BlockType"`
	Id           string         `json:"Id"`
	Text         string         `json:"Text,omitempty"`
	Confidence   float64        `json:"Confidence"`
	Geometry     map[string]any `json:"Geometry,omitempty"`
	Relationships []map[string]any `json:"Relationships,omitempty"`
	Page         int            `json:"Page,omitempty"`
}

type ExpenseDocument struct {
	ExpenseIndex    int              `json:"ExpenseIndex"`
	SummaryFields   []map[string]any `json:"SummaryFields,omitempty"`
	LineItemGroups  []map[string]any `json:"LineItemGroups,omitempty"`
}

type Job struct {
	JobId              string
	JobType            JobType
	Status             JobStatus
	DocumentLocation   map[string]any
	FeatureTypes       []string
	NotificationChannel map[string]any
	Blocks             []Block
	ExpenseDocuments   []ExpenseDocument
	CreationTime       time.Time
	CompletionTime     *time.Time
	StatusMessage      string
	Tags               map[string]string
	Lifecycle          *lifecycle.Machine
}

type Store struct {
	mu        sync.RWMutex
	jobs      map[string]*Job // keyed by JobId
	tagsByArn map[string]map[string]string
	accountID string
	region    string
	lcConfig  *lifecycle.Config
}

func NewStore(accountID, region string) *Store {
	return &Store{
		jobs:      make(map[string]*Job),
		tagsByArn: make(map[string]map[string]string),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) jobARN(jobId string) string {
	return fmt.Sprintf("arn:aws:textract:%s:%s:job/%s", s.region, s.accountID, jobId)
}

func mockBlocks() []Block {
	return []Block{
		{
			BlockType:  "PAGE",
			Id:         newUUID(),
			Confidence: 99.9,
			Page:       1,
		},
		{
			BlockType:  "LINE",
			Id:         newUUID(),
			Text:       "Sample detected text line",
			Confidence: 98.5,
			Page:       1,
		},
		{
			BlockType:  "WORD",
			Id:         newUUID(),
			Text:       "Sample",
			Confidence: 99.1,
			Page:       1,
		},
	}
}

func mockExpenseDocuments() []ExpenseDocument {
	return []ExpenseDocument{
		{
			ExpenseIndex: 1,
			SummaryFields: []map[string]any{
				{
					"Type":  map[string]any{"Text": "TOTAL", "Confidence": 95.0},
					"ValueDetection": map[string]any{"Text": "$100.00", "Confidence": 97.0},
				},
			},
		},
	}
}

func (s *Store) StartJob(jobType JobType, docLocation map[string]any, featureTypes []string, notifChannel map[string]any, tags map[string]string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobId := newUUID()
	if tags == nil {
		tags = make(map[string]string)
	}

	lc := lifecycle.NewMachine(
		lifecycle.State(JobInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(JobInProgress), To: lifecycle.State(JobSucceeded), Delay: 1 * time.Second},
		},
		s.lcConfig,
	)

	var blocks []Block
	var expenseDocs []ExpenseDocument
	if jobType == JobExpenseAnalysis {
		expenseDocs = mockExpenseDocuments()
	} else {
		blocks = mockBlocks()
	}

	j := &Job{
		JobId:               jobId,
		JobType:             jobType,
		Status:              JobStatus(lc.State()),
		DocumentLocation:    docLocation,
		FeatureTypes:        featureTypes,
		NotificationChannel: notifChannel,
		Blocks:              blocks,
		ExpenseDocuments:    expenseDocs,
		CreationTime:        time.Now().UTC(),
		Tags:                tags,
		Lifecycle:           lc,
	}
	s.jobs[jobId] = j
	arn := s.jobARN(jobId)
	s.tagsByArn[arn] = tags
	return j
}

func (s *Store) GetJob(jobId string) (*Job, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[jobId]
	if !ok {
		return nil, service.NewAWSError("InvalidJobIdException",
			fmt.Sprintf("Job %s not found", jobId), http.StatusNotFound)
	}
	j.Status = JobStatus(j.Lifecycle.State())
	if j.Status == JobSucceeded && j.CompletionTime == nil {
		now := time.Now().UTC()
		j.CompletionTime = &now
	}
	return j, nil
}

// Sync operations (AnalyzeDocument, DetectDocumentText).

func (s *Store) AnalyzeDocumentSync(document map[string]any, featureTypes []string) map[string]any {
	blocks := mockBlocks()
	blockMaps := make([]map[string]any, 0, len(blocks))
	for _, b := range blocks {
		blockMaps = append(blockMaps, map[string]any{
			"BlockType":  b.BlockType,
			"Id":         b.Id,
			"Text":       b.Text,
			"Confidence": b.Confidence,
			"Page":       b.Page,
		})
	}
	return map[string]any{
		"DocumentMetadata": map[string]any{"Pages": 1},
		"Blocks":           blockMaps,
		"AnalyzeDocumentModelVersion": "1.0",
	}
}

func (s *Store) DetectDocumentTextSync(document map[string]any) map[string]any {
	blocks := mockBlocks()
	blockMaps := make([]map[string]any, 0, len(blocks))
	for _, b := range blocks {
		blockMaps = append(blockMaps, map[string]any{
			"BlockType":  b.BlockType,
			"Id":         b.Id,
			"Text":       b.Text,
			"Confidence": b.Confidence,
			"Page":       b.Page,
		})
	}
	return map[string]any{
		"DocumentMetadata":            map[string]any{"Pages": 1},
		"Blocks":                      blockMaps,
		"DetectDocumentTextModelVersion": "1.0",
	}
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
