package sqs

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// QueueStore manages all SQS queues, indexed by URL and by name.
type QueueStore struct {
	mu        sync.RWMutex
	byURL     map[string]*Queue
	byName    map[string]*Queue
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
		byURL:     make(map[string]*Queue),
		byName:    make(map[string]*Queue),
		accountID: accountID,
		region:    region,
		port:      port,
	}
}

// QueueURL builds the canonical URL for a queue name.
func (s *QueueStore) QueueURL(name string) string {
	return fmt.Sprintf("http://sqs.%s.localhost:%d/%s/%s", s.region, s.port, s.accountID, name)
}

// CreateQueue creates a queue with the given name and attributes.
// If a queue with the same name already exists, it returns the existing queue
// (AWS behaviour: idempotent if attributes match).
func (s *QueueStore) CreateQueue(name string, attrs map[string]string) (*Queue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if q, ok := s.byName[name]; ok {
		return q, nil
	}

	url := fmt.Sprintf("http://sqs.%s.localhost:%d/%s/%s", s.region, s.port, s.accountID, name)
	q := newQueue(name, url, attrs)
	s.byURL[url] = q
	s.byName[name] = q
	return q, nil
}

// DeleteQueue removes a queue by URL.
func (s *QueueStore) DeleteQueue(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.byURL[url]
	if !ok {
		return false
	}
	delete(s.byURL, url)
	delete(s.byName, q.Name)
	return true
}

// GetByURL retrieves a queue by its URL.
func (s *QueueStore) GetByURL(url string) (*Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.byURL[url]
	return q, ok
}

// GetByName retrieves a queue by its name.
func (s *QueueStore) GetByName(name string) (*Queue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.byName[name]
	return q, ok
}

// ListQueues returns all queue URLs, optionally filtered by prefix.
func (s *QueueStore) ListQueues(prefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := make([]string, 0, len(s.byURL))
	for url, q := range s.byURL {
		if prefix == "" || strings.HasPrefix(q.Name, prefix) {
			urls = append(urls, url)
		}
	}
	return urls
}
