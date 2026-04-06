package ssm

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/pkg/service"
)

// Document holds all data for a single SSM document.
type Document struct {
	Name            string
	Content         string
	DocumentType    string // Command | Automation | Policy | Session | Package
	DocumentFormat  string // JSON | YAML | TEXT
	DocumentVersion string
	Status          string
	CreatedDate     time.Time
	Owner           string
	SHA1            string
}

// AutomationExecution holds data for an SSM automation execution.
type AutomationExecution struct {
	AutomationExecutionID string
	DocumentName          string
	Status                string
	StartTime             time.Time
}

// DocumentStore is the in-memory store for SSM documents and automation executions.
type DocumentStore struct {
	mu          sync.RWMutex
	documents   map[string]*Document           // keyed by Name
	automations map[string]*AutomationExecution // keyed by AutomationExecutionID
	accountID   string
	region      string
}

// NewDocumentStore creates an empty DocumentStore.
func NewDocumentStore(accountID, region string) *DocumentStore {
	return &DocumentStore{
		documents:   make(map[string]*Document),
		automations: make(map[string]*AutomationExecution),
		accountID:   accountID,
		region:      region,
	}
}

// ---- JSON request/response types ----

type createDocumentRequest struct {
	Name           string `json:"Name"`
	Content        string `json:"Content"`
	DocumentType   string `json:"DocumentType"`
	DocumentFormat string `json:"DocumentFormat"`
}

type documentDescription struct {
	Name            string  `json:"Name"`
	DocumentType    string  `json:"DocumentType"`
	DocumentFormat  string  `json:"DocumentFormat"`
	DocumentVersion string  `json:"DocumentVersion"`
	Status          string  `json:"Status"`
	CreatedDate     float64 `json:"CreatedDate"`
	Owner           string  `json:"Owner"`
	SHA1            string  `json:"Hash"`
	HashType        string  `json:"HashType"`
}

type createDocumentResponse struct {
	DocumentDescription documentDescription `json:"DocumentDescription"`
}

type describeDocumentRequest struct {
	Name string `json:"Name"`
}

type describeDocumentResponse struct {
	Document documentDescription `json:"Document"`
}

type getDocumentRequest struct {
	Name string `json:"Name"`
}

type getDocumentResponse struct {
	Name           string `json:"Name"`
	Content        string `json:"Content"`
	DocumentType   string `json:"DocumentType"`
	DocumentFormat string `json:"DocumentFormat"`
	Status         string `json:"Status"`
}

type listDocumentsResponse struct {
	DocumentIdentifiers []documentDescription `json:"DocumentIdentifiers"`
}

type deleteDocumentRequest struct {
	Name string `json:"Name"`
}

type startAutomationExecutionRequest struct {
	DocumentName string `json:"DocumentName"`
}

type startAutomationExecutionResponse struct {
	AutomationExecutionID string `json:"AutomationExecutionId"`
}

type automationExecutionMetadata struct {
	AutomationExecutionID string  `json:"AutomationExecutionId"`
	DocumentName          string  `json:"DocumentName"`
	Status                string  `json:"AutomationExecutionStatus"`
	StartTime             float64 `json:"ExecutionStartTime"`
}

type describeAutomationExecutionsResponse struct {
	AutomationExecutionMetadataList []automationExecutionMetadata `json:"AutomationExecutionMetadataList"`
}

// ---- helpers ----

func docToDescription(d *Document) documentDescription {
	return documentDescription{
		Name:            d.Name,
		DocumentType:    d.DocumentType,
		DocumentFormat:  d.DocumentFormat,
		DocumentVersion: d.DocumentVersion,
		Status:          d.Status,
		CreatedDate:     unixFloat(d.CreatedDate),
		Owner:           d.Owner,
		SHA1:            d.SHA1,
		HashType:        "Sha1",
	}
}

// ---- handlers ----

func handleCreateDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationException", "Name is required.", http.StatusBadRequest))
	}

	store.docStore.mu.Lock()
	defer store.docStore.mu.Unlock()

	if _, exists := store.docStore.documents[req.Name]; exists {
		return jsonErr(service.NewAWSError("DocumentAlreadyExists",
			fmt.Sprintf("Document with name %s already exists.", req.Name),
			http.StatusBadRequest))
	}

	docType := req.DocumentType
	if docType == "" {
		docType = "Command"
	}
	docFormat := req.DocumentFormat
	if docFormat == "" {
		docFormat = "JSON"
	}

	doc := &Document{
		Name:            req.Name,
		Content:         req.Content,
		DocumentType:    docType,
		DocumentFormat:  docFormat,
		DocumentVersion: "1",
		Status:          "Active",
		CreatedDate:     time.Now().UTC(),
		Owner:           store.accountID,
		SHA1:            "da39a3ee5e6b4b0d3255bfef95601890afd80709",
	}
	store.docStore.documents[req.Name] = doc

	return jsonOK(createDocumentResponse{
		DocumentDescription: docToDescription(doc),
	})
}

func handleDescribeDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationException", "Name is required.", http.StatusBadRequest))
	}

	store.docStore.mu.RLock()
	defer store.docStore.mu.RUnlock()

	doc, ok := store.docStore.documents[req.Name]
	if !ok {
		return jsonErr(service.NewAWSError("InvalidDocument",
			fmt.Sprintf("Document %s does not exist.", req.Name),
			http.StatusBadRequest))
	}

	return jsonOK(describeDocumentResponse{
		Document: docToDescription(doc),
	})
}

func handleGetDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationException", "Name is required.", http.StatusBadRequest))
	}

	store.docStore.mu.RLock()
	defer store.docStore.mu.RUnlock()

	doc, ok := store.docStore.documents[req.Name]
	if !ok {
		return jsonErr(service.NewAWSError("InvalidDocument",
			fmt.Sprintf("Document %s does not exist.", req.Name),
			http.StatusBadRequest))
	}

	return jsonOK(getDocumentResponse{
		Name:           doc.Name,
		Content:        doc.Content,
		DocumentType:   doc.DocumentType,
		DocumentFormat: doc.DocumentFormat,
		Status:         doc.Status,
	})
}

func handleListDocuments(_ *service.RequestContext, store *Store) (*service.Response, error) {
	store.docStore.mu.RLock()
	defer store.docStore.mu.RUnlock()

	docs := make([]documentDescription, 0, len(store.docStore.documents))
	for _, d := range store.docStore.documents {
		docs = append(docs, docToDescription(d))
	}

	return jsonOK(listDocumentsResponse{
		DocumentIdentifiers: docs,
	})
}

func handleDeleteDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationException", "Name is required.", http.StatusBadRequest))
	}

	store.docStore.mu.Lock()
	defer store.docStore.mu.Unlock()

	if _, ok := store.docStore.documents[req.Name]; !ok {
		return jsonErr(service.NewAWSError("InvalidDocument",
			fmt.Sprintf("Document %s does not exist.", req.Name),
			http.StatusBadRequest))
	}
	delete(store.docStore.documents, req.Name)

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleStartAutomationExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startAutomationExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DocumentName == "" {
		return jsonErr(service.NewAWSError("ValidationException", "DocumentName is required.", http.StatusBadRequest))
	}

	store.docStore.mu.Lock()
	defer store.docStore.mu.Unlock()

	// Verify the document exists
	if _, ok := store.docStore.documents[req.DocumentName]; !ok {
		return jsonErr(service.NewAWSError("InvalidDocument",
			fmt.Sprintf("Document %s does not exist.", req.DocumentName),
			http.StatusBadRequest))
	}

	execID := uuid.New().String()
	exec := &AutomationExecution{
		AutomationExecutionID: execID,
		DocumentName:          req.DocumentName,
		Status:                "Success",
		StartTime:             time.Now().UTC(),
	}
	store.docStore.automations[execID] = exec

	return jsonOK(startAutomationExecutionResponse{
		AutomationExecutionID: execID,
	})
}

func handleDescribeAutomationExecutions(_ *service.RequestContext, store *Store) (*service.Response, error) {
	store.docStore.mu.RLock()
	defer store.docStore.mu.RUnlock()

	execs := make([]automationExecutionMetadata, 0, len(store.docStore.automations))
	for _, e := range store.docStore.automations {
		execs = append(execs, automationExecutionMetadata{
			AutomationExecutionID: e.AutomationExecutionID,
			DocumentName:          e.DocumentName,
			Status:                e.Status,
			StartTime:             unixFloat(e.StartTime),
		})
	}

	return jsonOK(describeAutomationExecutionsResponse{
		AutomationExecutionMetadataList: execs,
	})
}
