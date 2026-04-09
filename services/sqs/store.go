package sqs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// QueueStore manages all SQS queues, indexed by URL, by name, and by ARN.
type QueueStore struct {
	mu        sync.RWMutex
	byURL     map[string]Queue
	byName    map[string]Queue
	byARN     map[string]Queue
	accountID string
	region    string
	port      int
}

// NewStore returns a new QueueStore.
func NewStore(accountID, region string) *QueueStore {
	port := 4566
	if v := os.Getenv("CLOUDMOCK_PORT"); v != "" {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
			port = p
		}
	}
	return &QueueStore{
		byURL:     make(map[string]Queue),
		byName:    make(map[string]Queue),
		byARN:     make(map[string]Queue),
		accountID: accountID,
		region:    region,
		port:      port,
	}
}

// QueueURL builds the canonical URL for a queue name.
func (s *QueueStore) QueueURL(name string) string {
	return fmt.Sprintf("http://sqs.%s.localhost:%d/%s/%s", s.region, s.port, s.accountID, name)
}

// queueARN builds the ARN for a queue name.
func (s *QueueStore) queueARN(name string) string {
	return fmt.Sprintf("arn:aws:sqs:%s:%s:%s", s.region, s.accountID, name)
}

// CreateQueue creates a queue with the given name and attributes.
// If a queue with the same name already exists, it returns the existing queue
// (AWS behaviour: idempotent if attributes match).
func (s *QueueStore) CreateQueue(name string, attrs map[string]string) (Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if q, ok := s.byName[name]; ok {
		return q, nil
	}

	url := fmt.Sprintf("http://sqs.%s.localhost:%d/%s/%s", s.region, s.port, s.accountID, name)
	arn := s.queueARN(name)

	var q Queue
	if strings.HasSuffix(name, ".fifo") {
		q = NewFIFOQueue(name, url, attrs)
	} else {
		q = NewStandardQueue(name, url, attrs)
	}

	s.byURL[url] = q
	s.byName[name] = q
	s.byARN[arn] = q

	// Check for RedrivePolicy and wire up DLQ.
	s.wireDLQLocked(q)

	return q, nil
}

// wireDLQLocked parses the RedrivePolicy attribute and wires up the DLQ.
// Caller must hold s.mu.
func (s *QueueStore) wireDLQLocked(q Queue) {
	attrs := q.GetAttributes()
	rpJSON, ok := attrs["RedrivePolicy"]
	if !ok || rpJSON == "" {
		return
	}
	var rp struct {
		DeadLetterTargetArn string      `json:"deadLetterTargetArn"`
		MaxReceiveCount     json.Number `json:"maxReceiveCount"`
	}
	if err := json.Unmarshal([]byte(rpJSON), &rp); err != nil {
		return
	}
	if rp.DeadLetterTargetArn == "" {
		return
	}
	maxRecv := 0
	if v, err := rp.MaxReceiveCount.Int64(); err == nil {
		maxRecv = int(v)
	}
	if dlq, found := s.byARN[rp.DeadLetterTargetArn]; found && maxRecv > 0 {
		q.SetDLQ(dlq, maxRecv)
	}
}

// DeleteQueue removes a queue by URL.
func (s *QueueStore) DeleteQueue(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.byURL[url]
	if !ok {
		return false
	}
	q.Close()
	name := q.QueueName()
	arn := s.queueARN(name)
	delete(s.byURL, url)
	delete(s.byName, name)
	delete(s.byARN, arn)
	return true
}

// GetByURL retrieves a queue by its URL.
func (s *QueueStore) GetByURL(url string) (Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.byURL[url]
	return q, ok
}

// GetByName retrieves a queue by its name.
func (s *QueueStore) GetByName(name string) (Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.byName[name]
	return q, ok
}

// GetByARN retrieves a queue by its ARN.
func (s *QueueStore) GetByARN(arn string) (Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.byARN[arn]
	return q, ok
}

// ListQueues returns all queue URLs, optionally filtered by prefix.
func (s *QueueStore) ListQueues(prefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := make([]string, 0, len(s.byURL))
	for url, q := range s.byURL {
		if prefix == "" || strings.HasPrefix(q.QueueName(), prefix) {
			urls = append(urls, url)
		}
	}
	return urls
}
