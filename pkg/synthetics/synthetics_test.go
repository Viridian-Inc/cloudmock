package synthetics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStoreAddAndGetTest(t *testing.T) {
	store := NewStore(100)

	test := store.AddTest(SyntheticTest{
		Name: "Test API",
		URL:  "http://localhost/health",
	})

	if test.ID == "" {
		t.Fatal("expected ID to be generated")
	}
	if test.Method != "GET" {
		t.Errorf("expected default method GET, got %s", test.Method)
	}
	if test.Type != "http" {
		t.Errorf("expected default type http, got %s", test.Type)
	}

	got, ok := store.GetTest(test.ID)
	if !ok {
		t.Fatal("expected to find test")
	}
	if got.Name != "Test API" {
		t.Errorf("name = %q, want %q", got.Name, "Test API")
	}
}

func TestStoreListTests(t *testing.T) {
	store := NewStore(100)
	store.AddTest(SyntheticTest{Name: "A"})
	store.AddTest(SyntheticTest{Name: "B"})

	tests := store.ListTests()
	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}
}

func TestStoreDeleteTest(t *testing.T) {
	store := NewStore(100)
	test := store.AddTest(SyntheticTest{Name: "A"})

	if !store.DeleteTest(test.ID) {
		t.Fatal("delete should return true")
	}
	if store.DeleteTest(test.ID) {
		t.Fatal("second delete should return false")
	}
	if _, ok := store.GetTest(test.ID); ok {
		t.Fatal("should not find deleted test")
	}
}

func TestStoreResultsCircularBuffer(t *testing.T) {
	store := NewStore(3)
	testID := "test-1"
	store.AddTest(SyntheticTest{ID: testID, Name: "A"})

	for i := 0; i < 5; i++ {
		store.AddResult(TestResult{
			TestID:    testID,
			Timestamp: time.Now(),
			Success:   i%2 == 0,
		})
	}

	results := store.GetResults(testID, 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (circular buffer), got %d", len(results))
	}
	// Newest first
	if results[0].Timestamp.Before(results[1].Timestamp) {
		t.Error("expected newest first")
	}
}

func TestEngineRunTestHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "hello")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	store := NewStore(100)
	test := store.AddTest(SyntheticTest{
		Name:   "Health Check",
		URL:    server.URL,
		Method: "GET",
		Assertions: []Assertion{
			{Type: "status_code", Operator: "equals", Target: "200"},
			{Type: "response_time", Operator: "less_than", Target: "5000"},
			{Type: "body_contains", Operator: "contains", Target: "ok"},
			{Type: "header_contains", Operator: "contains", Target: "X-Custom: hello"},
		},
	})

	engine := NewEngine(store, nil)
	result := engine.RunTest(test.ID)

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Error)
		for _, ar := range result.Assertions {
			if !ar.Passed {
				t.Errorf("  assertion %s %s %s failed (actual: %s)",
					ar.Assertion.Type, ar.Assertion.Operator, ar.Assertion.Target, ar.Actual)
			}
		}
	}
	if result.StatusCode != 200 {
		t.Errorf("status code = %d, want 200", result.StatusCode)
	}
	if len(result.Assertions) != 4 {
		t.Errorf("expected 4 assertion results, got %d", len(result.Assertions))
	}
}

func TestEngineRunTestFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	store := NewStore(100)
	test := store.AddTest(SyntheticTest{
		Name:   "Should Fail",
		URL:    server.URL,
		Method: "GET",
		Assertions: []Assertion{
			{Type: "status_code", Operator: "equals", Target: "200"},
		},
	})

	engine := NewEngine(store, nil)
	result := engine.RunTest(test.ID)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.Success {
		t.Error("expected failure")
	}
}

func TestEngineStatus(t *testing.T) {
	store := NewStore(100)
	test := store.AddTest(SyntheticTest{Name: "A", Enabled: true})
	store.AddResult(TestResult{TestID: test.ID, Success: true})

	engine := NewEngine(store, nil)
	status := engine.Status()

	if status["total"].(int) != 1 {
		t.Errorf("total = %v, want 1", status["total"])
	}
	if status["passing"].(int) != 1 {
		t.Errorf("passing = %v, want 1", status["passing"])
	}
}

func TestEvaluateNumeric(t *testing.T) {
	if !evaluateNumeric(200, "equals", "200") {
		t.Error("200 equals 200 should be true")
	}
	if !evaluateNumeric(50, "less_than", "100") {
		t.Error("50 < 100 should be true")
	}
	if evaluateNumeric(200, "less_than", "100") {
		t.Error("200 < 100 should be false")
	}
	if !evaluateNumeric(200, "greater_than", "100") {
		t.Error("200 > 100 should be true")
	}
}
