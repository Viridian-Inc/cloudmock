package textract

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
	BlockType     string           `json:"BlockType"`
	Id            string           `json:"Id"`
	Text          string           `json:"Text,omitempty"`
	Confidence    float64          `json:"Confidence"`
	Geometry      map[string]float64 `json:"Geometry,omitempty"`
	Relationships []map[string]any `json:"Relationships,omitempty"`
	EntityTypes   []string         `json:"EntityTypes,omitempty"`
	Page          int              `json:"Page,omitempty"`
	RowIndex      int              `json:"RowIndex,omitempty"`
	ColumnIndex   int              `json:"ColumnIndex,omitempty"`
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

// BoundingBox returns a realistic bounding box for a text block.
func boundingBox(top, left, width, height float64) map[string]float64 {
	return map[string]float64{
		"Top": top, "Left": left, "Width": width, "Height": height,
	}
}

func mockBlocks() []Block {
	pageID := newUUID()
	lineID := newUUID()
	wordID := newUUID()
	kvKeyID := newUUID()
	kvValueID := newUUID()
	tableID := newUUID()
	cellID := newUUID()

	return []Block{
		{
			BlockType:   "PAGE",
			Id:          pageID,
			Confidence:  99.9,
			Page:        1,
			Geometry:    boundingBox(0.0, 0.0, 1.0, 1.0),
			Relationships: []map[string]any{
				{"Type": "CHILD", "Ids": []string{lineID, kvKeyID, tableID}},
			},
		},
		{
			BlockType:   "LINE",
			Id:          lineID,
			Text:        "Invoice Number: INV-2024-0042",
			Confidence:  98.5,
			Page:        1,
			Geometry:    boundingBox(0.05, 0.05, 0.45, 0.03),
			Relationships: []map[string]any{
				{"Type": "CHILD", "Ids": []string{wordID}},
			},
		},
		{
			BlockType:  "WORD",
			Id:         wordID,
			Text:       "Invoice",
			Confidence: 99.1,
			Page:       1,
			Geometry:   boundingBox(0.05, 0.05, 0.10, 0.03),
		},
		{
			BlockType:    "KEY_VALUE_SET",
			Id:           kvKeyID,
			Text:         "Invoice Number:",
			Confidence:   96.2,
			Page:         1,
			EntityTypes:  []string{"KEY"},
			Geometry:     boundingBox(0.05, 0.05, 0.20, 0.03),
			Relationships: []map[string]any{
				{"Type": "VALUE", "Ids": []string{kvValueID}},
			},
		},
		{
			BlockType:   "KEY_VALUE_SET",
			Id:          kvValueID,
			Text:        "INV-2024-0042",
			Confidence:  97.8,
			Page:        1,
			EntityTypes: []string{"VALUE"},
			Geometry:    boundingBox(0.05, 0.26, 0.20, 0.03),
		},
		{
			BlockType:  "TABLE",
			Id:         tableID,
			Confidence: 95.5,
			Page:       1,
			Geometry:   boundingBox(0.10, 0.05, 0.90, 0.30),
			Relationships: []map[string]any{
				{"Type": "CHILD", "Ids": []string{cellID}},
			},
		},
		{
			BlockType:  "CELL",
			Id:         cellID,
			Text:       "Widget A",
			Confidence: 97.0,
			Page:       1,
			Geometry:   boundingBox(0.10, 0.05, 0.30, 0.05),
			RowIndex:   1,
			ColumnIndex: 1,
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

func blockToMap(b Block) map[string]any {
	m := map[string]any{
		"BlockType":  b.BlockType,
		"Id":         b.Id,
		"Confidence": b.Confidence,
		"Page":       b.Page,
	}
	if b.Text != "" {
		m["Text"] = b.Text
	}
	if b.Geometry != nil {
		m["Geometry"] = map[string]any{
			"BoundingBox": b.Geometry,
			"Polygon": []map[string]float64{
				{"X": b.Geometry["Left"], "Y": b.Geometry["Top"]},
				{"X": b.Geometry["Left"] + b.Geometry["Width"], "Y": b.Geometry["Top"]},
				{"X": b.Geometry["Left"] + b.Geometry["Width"], "Y": b.Geometry["Top"] + b.Geometry["Height"]},
				{"X": b.Geometry["Left"], "Y": b.Geometry["Top"] + b.Geometry["Height"]},
			},
		}
	}
	if len(b.Relationships) > 0 {
		m["Relationships"] = b.Relationships
	}
	if len(b.EntityTypes) > 0 {
		m["EntityTypes"] = b.EntityTypes
	}
	if b.RowIndex > 0 {
		m["RowIndex"] = b.RowIndex
		m["ColumnIndex"] = b.ColumnIndex
	}
	return m
}

func (s *Store) AnalyzeDocumentSync(document map[string]any, featureTypes []string) map[string]any {
	blocks := mockBlocks()
	blockMaps := make([]map[string]any, 0, len(blocks))
	for _, b := range blocks {
		blockMaps = append(blockMaps, blockToMap(b))
	}
	return map[string]any{
		"DocumentMetadata":             map[string]any{"Pages": 1},
		"Blocks":                       blockMaps,
		"AnalyzeDocumentModelVersion":  "1.0",
	}
}

func (s *Store) DetectDocumentTextSync(document map[string]any) map[string]any {
	blocks := mockBlocks()
	blockMaps := make([]map[string]any, 0, len(blocks))
	for _, b := range blocks {
		blockMaps = append(blockMaps, blockToMap(b))
	}
	return map[string]any{
		"DocumentMetadata":                map[string]any{"Pages": 1},
		"Blocks":                          blockMaps,
		"DetectDocumentTextModelVersion":  "1.0",
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
