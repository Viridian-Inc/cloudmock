package lambda

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// SQSDirectAccess provides direct access to SQS queue internals for event source mapping.
// The SQS service satisfies this interface.
type SQSDirectAccess interface {
	EnqueueDirect(queueName, messageBody string) bool
	PollMessages(queueName string, maxCount, visibilityTimeout int) (messageIDs, bodies, receiptHandles []string, ok bool)
	AckMessage(queueName, receiptHandle string) bool
}

// EventSourceMapping represents a Lambda event source mapping (SQS → Lambda).
type EventSourceMapping struct {
	UUID             string `json:"UUID"`
	EventSourceArn   string `json:"EventSourceArn"`
	FunctionArn      string `json:"FunctionArn"`
	FunctionName     string `json:"-"`
	BatchSize        int    `json:"BatchSize"`
	Enabled          bool   `json:"Enabled"`
	State            string `json:"State"`
	LastModified     string `json:"LastModified"`
	MaximumBatchingWindowInSeconds int `json:"MaximumBatchingWindowInSeconds"`
}

// EventSourceMappingStore manages event source mappings.
type EventSourceMappingStore struct {
	mu       sync.RWMutex
	mappings map[string]*EventSourceMapping // keyed by UUID
	stopChs  map[string]chan struct{}       // stop channels for pollers
}

// NewEventSourceMappingStore returns a new store.
func NewEventSourceMappingStore() *EventSourceMappingStore {
	return &EventSourceMappingStore{
		mappings: make(map[string]*EventSourceMapping),
		stopChs:  make(map[string]chan struct{}),
	}
}

// Create adds a new event source mapping.
func (s *EventSourceMappingStore) Create(m *EventSourceMapping) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mappings[m.UUID] = m
}

// Get returns a mapping by UUID.
func (s *EventSourceMappingStore) Get(uuid string) (*EventSourceMapping, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.mappings[uuid]
	return m, ok
}

// Delete removes a mapping by UUID.
func (s *EventSourceMappingStore) Delete(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.mappings[uuid]; !ok {
		return false
	}
	// Stop the poller if running.
	if ch, ok := s.stopChs[uuid]; ok {
		close(ch)
		delete(s.stopChs, uuid)
	}
	delete(s.mappings, uuid)
	return true
}

// List returns all mappings, optionally filtered by EventSourceArn or FunctionName.
func (s *EventSourceMappingStore) List(eventSourceArn, functionName string) []*EventSourceMapping {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*EventSourceMapping, 0, len(s.mappings))
	for _, m := range s.mappings {
		if eventSourceArn != "" && m.EventSourceArn != eventSourceArn {
			continue
		}
		if functionName != "" && m.FunctionName != functionName {
			continue
		}
		result = append(result, m)
	}
	return result
}

// SetStopCh records the stop channel for a poller.
func (s *EventSourceMappingStore) SetStopCh(uuid string, ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopChs[uuid] = ch
}

// StopAll stops all running pollers.
func (s *EventSourceMappingStore) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for uuid, ch := range s.stopChs {
		close(ch)
		delete(s.stopChs, uuid)
	}
}

// ---- Event source mapping API handlers ----

// eventSourceMappingResponse builds the JSON response for an event source mapping.
func eventSourceMappingResponse(m *EventSourceMapping) map[string]any {
	return map[string]any{
		"UUID":                           m.UUID,
		"EventSourceArn":                 m.EventSourceArn,
		"FunctionArn":                    m.FunctionArn,
		"BatchSize":                      m.BatchSize,
		"State":                          m.State,
		"LastModified":                   m.LastModified,
		"MaximumBatchingWindowInSeconds": m.MaximumBatchingWindowInSeconds,
	}
}

// handleCreateEventSourceMapping handles the CreateEventSourceMapping API.
func handleCreateEventSourceMapping(ctx *service.RequestContext, svc *LambdaService) (*service.Response, error) {
	var req struct {
		EventSourceArn string `json:"EventSourceArn"`
		FunctionName   string `json:"FunctionName"`
		BatchSize      int    `json:"BatchSize"`
		Enabled        *bool  `json:"Enabled"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.ErrValidation("invalid request body: " + err.Error()))
	}
	if req.EventSourceArn == "" {
		return jsonErr(service.ErrValidation("EventSourceArn is required"))
	}
	if req.FunctionName == "" {
		return jsonErr(service.ErrValidation("FunctionName is required"))
	}

	batchSize := req.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Resolve function name from ARN or name.
	funcName := req.FunctionName
	if strings.Contains(funcName, ":") {
		// It's an ARN — extract function name.
		parts := strings.Split(funcName, ":")
		funcName = parts[len(parts)-1]
	}

	// Build function ARN.
	fn, ok := svc.store.Get(funcName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", funcName), http.StatusNotFound))
	}

	uuid := newMappingUUID()
	state := "Enabled"
	if !enabled {
		state = "Disabled"
	}

	mapping := &EventSourceMapping{
		UUID:           uuid,
		EventSourceArn: req.EventSourceArn,
		FunctionArn:    fn.FunctionArn,
		FunctionName:   funcName,
		BatchSize:      batchSize,
		Enabled:        enabled,
		State:          state,
		LastModified:   time.Now().UTC().Format(time.RFC3339),
	}

	svc.esmStore.Create(mapping)

	// Start background poller if enabled.
	if enabled {
		startSQSPoller(svc, mapping)
	}

	return jsonCreated(eventSourceMappingResponse(mapping))
}

// handleGetEventSourceMapping handles the GetEventSourceMapping API.
func handleGetEventSourceMapping(ctx *service.RequestContext, svc *LambdaService, uuid string) (*service.Response, error) {
	m, ok := svc.esmStore.Get(uuid)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Event source mapping not found: %s", uuid), http.StatusNotFound))
	}
	return jsonOK(eventSourceMappingResponse(m))
}

// handleDeleteEventSourceMapping handles the DeleteEventSourceMapping API.
func handleDeleteEventSourceMapping(ctx *service.RequestContext, svc *LambdaService, uuid string) (*service.Response, error) {
	m, ok := svc.esmStore.Get(uuid)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Event source mapping not found: %s", uuid), http.StatusNotFound))
	}
	resp := eventSourceMappingResponse(m)
	svc.esmStore.Delete(uuid)
	return jsonOK(resp)
}

// handleListEventSourceMappings handles the ListEventSourceMappings API.
func handleListEventSourceMappings(ctx *service.RequestContext, svc *LambdaService) (*service.Response, error) {
	q := ctx.RawRequest.URL.Query()
	eventSourceArn := q.Get("EventSourceArn")
	functionName := q.Get("FunctionName")

	mappings := svc.esmStore.List(eventSourceArn, functionName)
	entries := make([]map[string]any, 0, len(mappings))
	for _, m := range mappings {
		entries = append(entries, eventSourceMappingResponse(m))
	}

	return jsonOK(map[string]any{
		"EventSourceMappings": entries,
	})
}

// ---- Background SQS poller ----

// startSQSPoller starts a goroutine that polls SQS and invokes Lambda.
func startSQSPoller(svc *LambdaService, mapping *EventSourceMapping) {
	stopCh := make(chan struct{})
	svc.esmStore.SetStopCh(mapping.UUID, stopCh)

	// Extract queue name from SQS ARN.
	queueName := extractQueueNameFromSQSArn(mapping.EventSourceArn)
	if queueName == "" {
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				pollAndInvoke(svc, mapping, queueName)
			}
		}
	}()
}

// pollAndInvoke polls SQS for messages and invokes the Lambda function.
func pollAndInvoke(svc *LambdaService, mapping *EventSourceMapping, queueName string) {
	if svc.locator == nil {
		return
	}

	sqsSvc, err := svc.locator.Lookup("sqs")
	if err != nil {
		return
	}

	accessor, ok := sqsSvc.(SQSDirectAccess)
	if !ok {
		return
	}

	msgIDs, bodies, receipts, ok := accessor.PollMessages(queueName, mapping.BatchSize, 30)
	if !ok || len(msgIDs) == 0 {
		return
	}

	// Build SQS event payload.
	records := make([]map[string]any, 0, len(msgIDs))
	for i := range msgIDs {
		records = append(records, map[string]any{
			"messageId":      msgIDs[i],
			"receiptHandle":  receipts[i],
			"body":           bodies[i],
			"eventSource":    "aws:sqs",
			"eventSourceARN": mapping.EventSourceArn,
			"awsRegion":      "us-east-1",
		})
	}

	payload, err := json.Marshal(map[string]any{
		"Records": records,
	})
	if err != nil {
		return
	}

	// Invoke the Lambda function.
	_, invokeErr := svc.InvokeDirect(mapping.FunctionName, payload)
	if invokeErr != nil {
		log.Printf("cloudmock: SQS→Lambda invoke error for %s: %v", mapping.FunctionName, invokeErr)
		return
	}

	// Delete successfully processed messages.
	for _, rh := range receipts {
		accessor.AckMessage(queueName, rh)
	}
}

// extractQueueNameFromSQSArn extracts the queue name from an SQS ARN.
func extractQueueNameFromSQSArn(arn string) string {
	// arn:aws:sqs:region:account:queue-name
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return ""
	}
	return parts[5]
}

// newMappingUUID returns a random UUID for event source mappings.
func newMappingUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
