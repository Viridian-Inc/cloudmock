package cloudcontrol

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Resource represents a Cloud Control API resource.
type Resource struct {
	TypeName   string
	Identifier string
	Properties string // JSON string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ResourceRequest represents a Cloud Control API resource request.
type ResourceRequest struct {
	RequestToken  string
	OperationType string // CREATE, UPDATE, DELETE
	TypeName      string
	Identifier    string
	StatusCode    string // PENDING, IN_PROGRESS, SUCCESS, FAILED
	StatusMessage string
	EventTime     time.Time
	lifecycle     *lifecycle.Machine
}

// Store manages Cloud Control resources in memory.
type Store struct {
	mu         sync.RWMutex
	resources  map[string]map[string]*Resource // typeName -> identifier -> Resource
	requests   map[string]*ResourceRequest
	accountID  string
	region     string
	lcConfig   *lifecycle.Config
	reqSeq     int
}

// NewStore returns a new empty Cloud Control Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		resources: make(map[string]map[string]*Resource),
		requests:  make(map[string]*ResourceRequest),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func (s *Store) newRequestToken() string {
	s.reqSeq++
	return fmt.Sprintf("req-%012d", s.reqSeq)
}

// CreateResource creates a new resource and returns the request tracking it.
func (s *Store) CreateResource(typeName, identifier, properties string) (*ResourceRequest, *Resource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.resources[typeName]; !ok {
		s.resources[typeName] = make(map[string]*Resource)
	}
	if _, ok := s.resources[typeName][identifier]; ok {
		return nil, nil, fmt.Errorf("resource already exists: %s/%s", typeName, identifier)
	}

	now := time.Now().UTC()
	resource := &Resource{
		TypeName:   typeName,
		Identifier: identifier,
		Properties: properties,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	token := s.newRequestToken()
	transitions := []lifecycle.Transition{
		{From: "PENDING", To: "IN_PROGRESS", Delay: 500 * time.Millisecond},
		{From: "IN_PROGRESS", To: "SUCCESS", Delay: 2 * time.Second},
	}

	req := &ResourceRequest{
		RequestToken:  token,
		OperationType: "CREATE",
		TypeName:      typeName,
		Identifier:    identifier,
		StatusCode:    "PENDING",
		StatusMessage: "Create request pending",
		EventTime:     now,
	}
	req.lifecycle = lifecycle.NewMachine("PENDING", transitions, s.lcConfig)
	req.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		req.StatusCode = string(to)
		if to == "SUCCESS" {
			req.StatusMessage = "Create complete"
		} else if to == "IN_PROGRESS" {
			req.StatusMessage = "Create in progress"
		}
	})

	s.resources[typeName][identifier] = resource
	s.requests[token] = req
	return req, resource, nil
}

// GetResource retrieves a resource.
func (s *Store) GetResource(typeName, identifier string) (*Resource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	typeMap, ok := s.resources[typeName]
	if !ok {
		return nil, false
	}
	r, ok := typeMap[identifier]
	return r, ok
}

// ListResources returns all resources of a given type.
func (s *Store) ListResources(typeName string) []*Resource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	typeMap := s.resources[typeName]
	out := make([]*Resource, 0, len(typeMap))
	for _, r := range typeMap {
		out = append(out, r)
	}
	return out
}

// UpdateResource updates a resource's properties.
func (s *Store) UpdateResource(typeName, identifier, patchDocument string) (*ResourceRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	typeMap, ok := s.resources[typeName]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s/%s", typeName, identifier)
	}
	resource, ok := typeMap[identifier]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s/%s", typeName, identifier)
	}
	resource.UpdatedAt = time.Now().UTC()
	// In a real impl we'd apply the patch; for mock we just track the request
	resource.Properties = patchDocument

	token := s.newRequestToken()
	transitions := []lifecycle.Transition{
		{From: "PENDING", To: "IN_PROGRESS", Delay: 500 * time.Millisecond},
		{From: "IN_PROGRESS", To: "SUCCESS", Delay: 1 * time.Second},
	}

	req := &ResourceRequest{
		RequestToken:  token,
		OperationType: "UPDATE",
		TypeName:      typeName,
		Identifier:    identifier,
		StatusCode:    "PENDING",
		StatusMessage: "Update request pending",
		EventTime:     time.Now().UTC(),
	}
	req.lifecycle = lifecycle.NewMachine("PENDING", transitions, s.lcConfig)
	req.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		req.StatusCode = string(to)
		if to == "SUCCESS" {
			req.StatusMessage = "Update complete"
		}
	})

	s.requests[token] = req
	return req, nil
}

// DeleteResource deletes a resource.
func (s *Store) DeleteResource(typeName, identifier string) (*ResourceRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	typeMap, ok := s.resources[typeName]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s/%s", typeName, identifier)
	}
	if _, ok := typeMap[identifier]; !ok {
		return nil, fmt.Errorf("resource not found: %s/%s", typeName, identifier)
	}
	delete(typeMap, identifier)

	token := s.newRequestToken()
	transitions := []lifecycle.Transition{
		{From: "PENDING", To: "IN_PROGRESS", Delay: 500 * time.Millisecond},
		{From: "IN_PROGRESS", To: "SUCCESS", Delay: 1 * time.Second},
	}

	req := &ResourceRequest{
		RequestToken:  token,
		OperationType: "DELETE",
		TypeName:      typeName,
		Identifier:    identifier,
		StatusCode:    "PENDING",
		StatusMessage: "Delete request pending",
		EventTime:     time.Now().UTC(),
	}
	req.lifecycle = lifecycle.NewMachine("PENDING", transitions, s.lcConfig)
	req.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		req.StatusCode = string(to)
		if to == "SUCCESS" {
			req.StatusMessage = "Delete complete"
		}
	})

	s.requests[token] = req
	return req, nil
}

// GetResourceRequestStatus retrieves a request by token.
func (s *Store) GetResourceRequestStatus(token string) (*ResourceRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[token]
	return req, ok
}

// ListResourceRequests returns all requests.
func (s *Store) ListResourceRequests() []*ResourceRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ResourceRequest, 0, len(s.requests))
	for _, req := range s.requests {
		out = append(out, req)
	}
	return out
}
