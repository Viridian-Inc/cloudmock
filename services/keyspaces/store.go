package keyspaces

import (
	"fmt"
	"crypto/rand"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Store is the in-memory data store for keyspaces resources.
type Store struct {
	mu        sync.RWMutex
	resources map[string]map[string]any // resourceType -> id -> resource
	accountID string
	region    string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		resources: make(map[string]map[string]any),
		accountID: accountID,
		region:    region,
	}
}

func (s *Store) put(resourceType, id string, resource any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resources[resourceType] == nil {
		s.resources[resourceType] = make(map[string]any)
	}
	s.resources[resourceType][id] = resource
}

func (s *Store) get(resourceType, id string) (any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.resources[resourceType]; ok {
		if r, ok := m[id]; ok {
			return r, nil
		}
	}
	return nil, service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %s not found", resourceType, id), http.StatusBadRequest)
}

func (s *Store) list(resourceType string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []any
	for _, r := range s.resources[resourceType] {
		out = append(out, r)
	}
	return out
}

func (s *Store) delete(resourceType, id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.resources[resourceType]; ok {
		if _, ok := m[id]; ok {
			delete(m, id)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %s not found", resourceType, id), http.StatusBadRequest)
}

func (s *Store) buildArn(resourceType, id string) string {
	return fmt.Sprintf("arn:aws:cassandra:%s:%s:%s/%s", s.region, s.accountID, resourceType, id)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var _ = time.Now // ensure time is used
