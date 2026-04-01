package memory

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/uptime"
)

const defaultBufferSize = 1000

// Store is an in-memory uptime store with a circular buffer per check.
type Store struct {
	mu         sync.RWMutex
	checks     map[string]uptime.Check
	results    map[string]*ringBuffer
	bufferSize int
}

// NewStore creates an in-memory uptime store.
func NewStore(bufferSize int) *Store {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	return &Store{
		checks:     make(map[string]uptime.Check),
		results:    make(map[string]*ringBuffer),
		bufferSize: bufferSize,
	}
}

func (s *Store) CreateCheck(check uptime.Check) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.checks[check.ID]; ok {
		return fmt.Errorf("check %q already exists", check.ID)
	}
	s.checks[check.ID] = check
	s.results[check.ID] = newRingBuffer(s.bufferSize)
	return nil
}

func (s *Store) UpdateCheck(check uptime.Check) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.checks[check.ID]; !ok {
		return fmt.Errorf("check %q not found", check.ID)
	}
	s.checks[check.ID] = check
	return nil
}

func (s *Store) DeleteCheck(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.checks[id]; !ok {
		return fmt.Errorf("check %q not found", id)
	}
	delete(s.checks, id)
	delete(s.results, id)
	return nil
}

func (s *Store) GetCheck(id string) (*uptime.Check, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.checks[id]
	if !ok {
		return nil, fmt.Errorf("check %q not found", id)
	}
	return &c, nil
}

func (s *Store) ListChecks() ([]uptime.Check, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]uptime.Check, 0, len(s.checks))
	for _, c := range s.checks {
		result = append(result, c)
	}
	return result, nil
}

func (s *Store) AddResult(result uptime.CheckResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rb, ok := s.results[result.CheckID]
	if !ok {
		return fmt.Errorf("check %q not found", result.CheckID)
	}
	rb.add(result)
	return nil
}

func (s *Store) Results(checkID string) ([]uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rb, ok := s.results[checkID]
	if !ok {
		return nil, fmt.Errorf("check %q not found", checkID)
	}
	return rb.all(), nil
}

func (s *Store) LastResult(checkID string) (*uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rb, ok := s.results[checkID]
	if !ok {
		return nil, fmt.Errorf("check %q not found", checkID)
	}
	return rb.last(), nil
}

func (s *Store) LastFailure(checkID string) (*uptime.CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rb, ok := s.results[checkID]
	if !ok {
		return nil, fmt.Errorf("check %q not found", checkID)
	}
	all := rb.all()
	for _, r := range all {
		if !r.Success {
			return &r, nil
		}
	}
	return nil, nil
}

func (s *Store) ConsecutiveFailures(checkID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rb, ok := s.results[checkID]
	if !ok {
		return 0
	}
	all := rb.all() // newest first
	count := 0
	for _, r := range all {
		if !r.Success {
			count++
		} else {
			break
		}
	}
	return count
}

// ringBuffer is a fixed-size circular buffer of CheckResults.
type ringBuffer struct {
	data []uptime.CheckResult
	cap  int
	pos  int
	full bool
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		data: make([]uptime.CheckResult, capacity),
		cap:  capacity,
	}
}

func (rb *ringBuffer) add(r uptime.CheckResult) {
	rb.data[rb.pos] = r
	rb.pos = (rb.pos + 1) % rb.cap
	if rb.pos == 0 {
		rb.full = true
	}
}

// all returns results newest first.
func (rb *ringBuffer) all() []uptime.CheckResult {
	var out []uptime.CheckResult
	if rb.full {
		out = make([]uptime.CheckResult, rb.cap)
		copy(out, rb.data[rb.pos:])
		copy(out[rb.cap-rb.pos:], rb.data[:rb.pos])
	} else {
		out = make([]uptime.CheckResult, rb.pos)
		copy(out, rb.data[:rb.pos])
	}
	// Reverse to get newest first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (rb *ringBuffer) last() *uptime.CheckResult {
	if !rb.full && rb.pos == 0 {
		return nil
	}
	idx := rb.pos - 1
	if idx < 0 {
		idx = rb.cap - 1
	}
	r := rb.data[idx]
	return &r
}
