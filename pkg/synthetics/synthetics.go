// Package synthetics provides synthetic HTTP test execution and assertion evaluation.
// Tests run at configured intervals using the worker pool and results are stored
// in an in-memory circular buffer.
package synthetics

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/worker"
)

// SyntheticTest defines a scheduled HTTP test with assertions.
type SyntheticTest struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`     // "http", "browser" (browser = future)
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	Assertions []Assertion       `json:"assertions"`
	Interval   time.Duration     `json:"interval"`
	Timeout    time.Duration     `json:"timeout"`
	Enabled    bool              `json:"enabled"`
	Locations  []string          `json:"locations,omitempty"` // "local", future: edge locations
}

// Assertion defines a check to run against a test response.
type Assertion struct {
	Type     string `json:"type"`     // "status_code", "response_time", "body_contains", "header_contains"
	Operator string `json:"operator"` // "equals", "less_than", "greater_than", "contains"
	Target   string `json:"target"`   // expected value
}

// TestResult records the outcome of a single test execution.
type TestResult struct {
	TestID     string            `json:"test_id"`
	Timestamp  time.Time         `json:"timestamp"`
	Success    bool              `json:"success"`
	StatusCode int               `json:"status_code"`
	ResponseMs float64           `json:"response_ms"`
	Assertions []AssertionResult `json:"assertions"`
	Error      string            `json:"error,omitempty"`
}

// AssertionResult records whether a single assertion passed.
type AssertionResult struct {
	Assertion Assertion `json:"assertion"`
	Passed    bool      `json:"passed"`
	Actual    string    `json:"actual"`
}

// Store holds synthetic tests and their results in memory.
type Store struct {
	mu         sync.RWMutex
	tests      map[string]*SyntheticTest
	results    map[string][]TestResult // test ID → results (circular buffer)
	maxResults int
}

// NewStore creates a store with the given max results per test.
func NewStore(maxResultsPerTest int) *Store {
	if maxResultsPerTest <= 0 {
		maxResultsPerTest = 100
	}
	return &Store{
		tests:      make(map[string]*SyntheticTest),
		results:    make(map[string][]TestResult),
		maxResults: maxResultsPerTest,
	}
}

// AddTest adds a test to the store, generating an ID if needed.
func (s *Store) AddTest(test SyntheticTest) SyntheticTest {
	s.mu.Lock()
	defer s.mu.Unlock()
	if test.ID == "" {
		test.ID = generateID()
	}
	if test.Method == "" {
		test.Method = "GET"
	}
	if test.Type == "" {
		test.Type = "http"
	}
	if test.Interval == 0 {
		test.Interval = 60 * time.Second
	}
	if test.Timeout == 0 {
		test.Timeout = 10 * time.Second
	}
	if len(test.Locations) == 0 {
		test.Locations = []string{"local"}
	}
	s.tests[test.ID] = &test
	return test
}

// GetTest returns a test by ID.
func (s *Store) GetTest(id string) (*SyntheticTest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tests[id]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

// ListTests returns all tests.
func (s *Store) ListTests() []SyntheticTest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SyntheticTest, 0, len(s.tests))
	for _, t := range s.tests {
		out = append(out, *t)
	}
	return out
}

// DeleteTest removes a test and its results.
func (s *Store) DeleteTest(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tests[id]; !ok {
		return false
	}
	delete(s.tests, id)
	delete(s.results, id)
	return true
}

// AddResult appends a result, evicting the oldest if over capacity.
func (s *Store) AddResult(result TestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	results := s.results[result.TestID]
	results = append(results, result)
	if len(results) > s.maxResults {
		results = results[len(results)-s.maxResults:]
	}
	s.results[result.TestID] = results
}

// GetResults returns results for a test, newest first.
func (s *Store) GetResults(testID string, limit int) []TestResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := s.results[testID]
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	out := make([]TestResult, limit)
	for i := 0; i < limit; i++ {
		out[i] = results[len(results)-1-i]
	}
	return out
}

// Engine runs synthetic tests using a worker pool.
type Engine struct {
	store   *Store
	pool    *worker.Pool
	client  *http.Client
	mu      sync.Mutex
	cancels map[string]func()
}

// NewEngine creates a synthetics engine.
func NewEngine(store *Store, pool *worker.Pool) *Engine {
	return &Engine{
		store:   store,
		pool:    pool,
		client:  &http.Client{},
		cancels: make(map[string]func()),
	}
}

// Store returns the underlying store.
func (e *Engine) Store() *Store {
	return e.store
}

// ScheduleTest starts periodic execution of a test.
func (e *Engine) ScheduleTest(test SyntheticTest) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Cancel previous schedule if any.
	if cancel, ok := e.cancels[test.ID]; ok {
		cancel()
	}

	if !test.Enabled {
		return
	}

	cancel := e.pool.ScheduleInterval("synthetic-"+test.ID, test.Interval, func() {
		result := e.RunTest(test.ID)
		if result != nil {
			e.store.AddResult(*result)
		}
	})
	e.cancels[test.ID] = cancel
}

// StopTest stops periodic execution of a test.
func (e *Engine) StopTest(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cancel, ok := e.cancels[id]; ok {
		cancel()
		delete(e.cancels, id)
	}
}

// StartAll schedules all enabled tests.
func (e *Engine) StartAll() {
	for _, test := range e.store.ListTests() {
		if test.Enabled {
			e.ScheduleTest(test)
		}
	}
}

// RunTest executes a test immediately and returns the result.
func (e *Engine) RunTest(id string) *TestResult {
	test, ok := e.store.GetTest(id)
	if !ok {
		return nil
	}

	result := TestResult{
		TestID:    id,
		Timestamp: time.Now(),
		Success:   true,
	}

	start := time.Now()

	var bodyReader io.Reader
	if test.Body != "" {
		bodyReader = strings.NewReader(test.Body)
	}

	req, err := http.NewRequest(test.Method, test.URL, bodyReader)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return &result
	}

	for k, v := range test.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: test.Timeout}
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	result.ResponseMs = float64(elapsed.Milliseconds())

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return &result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read response body for body assertions.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	bodyStr := string(body)

	// Evaluate assertions.
	for _, assertion := range test.Assertions {
		ar := evaluateAssertion(assertion, resp, result.ResponseMs, bodyStr)
		result.Assertions = append(result.Assertions, ar)
		if !ar.Passed {
			result.Success = false
		}
	}

	return &result
}

// Status returns a summary of all tests.
func (e *Engine) Status() map[string]any {
	tests := e.store.ListTests()
	total := len(tests)
	passing := 0
	failing := 0

	for _, test := range tests {
		results := e.store.GetResults(test.ID, 1)
		if len(results) > 0 && results[0].Success {
			passing++
		} else if len(results) > 0 {
			failing++
		}
	}

	return map[string]any{
		"total":   total,
		"passing": passing,
		"failing": failing,
		"unknown": total - passing - failing,
	}
}

func evaluateAssertion(a Assertion, resp *http.Response, responseMs float64, body string) AssertionResult {
	ar := AssertionResult{Assertion: a}

	switch a.Type {
	case "status_code":
		ar.Actual = strconv.Itoa(resp.StatusCode)
		ar.Passed = evaluateNumeric(float64(resp.StatusCode), a.Operator, a.Target)

	case "response_time":
		ar.Actual = fmt.Sprintf("%.0f", responseMs)
		ar.Passed = evaluateNumeric(responseMs, a.Operator, a.Target)

	case "body_contains":
		ar.Actual = fmt.Sprintf("(body length: %d)", len(body))
		switch a.Operator {
		case "contains":
			ar.Passed = strings.Contains(body, a.Target)
		case "equals":
			ar.Passed = strings.TrimSpace(body) == a.Target
		default:
			ar.Passed = strings.Contains(body, a.Target)
		}

	case "header_contains":
		// Target format: "Header-Name: expected-value"
		parts := strings.SplitN(a.Target, ":", 2)
		if len(parts) == 2 {
			headerName := strings.TrimSpace(parts[0])
			expectedVal := strings.TrimSpace(parts[1])
			actual := resp.Header.Get(headerName)
			ar.Actual = actual
			ar.Passed = strings.Contains(actual, expectedVal)
		}

	default:
		ar.Actual = "unknown assertion type"
		ar.Passed = false
	}

	return ar
}

func evaluateNumeric(actual float64, operator string, target string) bool {
	targetF, err := strconv.ParseFloat(target, 64)
	if err != nil {
		return false
	}

	switch operator {
	case "equals":
		return actual == targetF
	case "less_than":
		return actual < targetF
	case "greater_than":
		return actual > targetF
	default:
		return actual == targetF
	}
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
