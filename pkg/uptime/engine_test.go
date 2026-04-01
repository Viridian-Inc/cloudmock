package uptime

import (
	"testing"
	"time"
)

// mockStore implements Store for testing without the memory package.
type mockStore struct {
	checks  map[string]Check
	results map[string][]CheckResult
}

func newMockStore() *mockStore {
	return &mockStore{
		checks:  make(map[string]Check),
		results: make(map[string][]CheckResult),
	}
}

func (m *mockStore) CreateCheck(c Check) error {
	m.checks[c.ID] = c
	m.results[c.ID] = nil
	return nil
}

func (m *mockStore) UpdateCheck(c Check) error {
	m.checks[c.ID] = c
	return nil
}

func (m *mockStore) DeleteCheck(id string) error {
	delete(m.checks, id)
	delete(m.results, id)
	return nil
}

func (m *mockStore) GetCheck(id string) (*Check, error) {
	c, ok := m.checks[id]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (m *mockStore) ListChecks() ([]Check, error) {
	out := make([]Check, 0, len(m.checks))
	for _, c := range m.checks {
		out = append(out, c)
	}
	return out, nil
}

func (m *mockStore) AddResult(r CheckResult) error {
	m.results[r.CheckID] = append(m.results[r.CheckID], r)
	return nil
}

func (m *mockStore) Results(checkID string) ([]CheckResult, error) {
	rs := m.results[checkID]
	// Return newest first.
	out := make([]CheckResult, len(rs))
	for i, r := range rs {
		out[len(rs)-1-i] = r
	}
	return out, nil
}

func (m *mockStore) LastResult(checkID string) (*CheckResult, error) {
	rs := m.results[checkID]
	if len(rs) == 0 {
		return nil, nil
	}
	r := rs[len(rs)-1]
	return &r, nil
}

func (m *mockStore) LastFailure(checkID string) (*CheckResult, error) {
	rs := m.results[checkID]
	for i := len(rs) - 1; i >= 0; i-- {
		if !rs[i].Success {
			return &rs[i], nil
		}
	}
	return nil, nil
}

func (m *mockStore) ConsecutiveFailures(checkID string) int {
	rs := m.results[checkID]
	count := 0
	for i := len(rs) - 1; i >= 0; i-- {
		if !rs[i].Success {
			count++
		} else {
			break
		}
	}
	return count
}

func TestCalcUptime(t *testing.T) {
	now := time.Now()

	results := []CheckResult{
		{Timestamp: now.Add(-1 * time.Hour), Success: true},
		{Timestamp: now.Add(-2 * time.Hour), Success: true},
		{Timestamp: now.Add(-3 * time.Hour), Success: false},
		{Timestamp: now.Add(-4 * time.Hour), Success: true},
	}

	// All 4 results are within 24h.
	uptime24h := calcUptime(results, now, 24*time.Hour)
	expected := 75.0
	if uptime24h != expected {
		t.Errorf("expected %.1f%%, got %.1f%%", expected, uptime24h)
	}

	// 2 results within 90 minutes (1h and 2h are not in 90min; only 1h is).
	// Actually: 1h ago is within 90min, 2h is not. So 1 result, 100% uptime.
	// Let's test with a window that includes exactly the failure:
	// 3h window captures all 4 results: 3 success + 1 fail = 75%.
	// 2h window captures 2 results at 1h and 2h: both success = 100%.
	// 3.5h window captures first 3: 2 success + 1 fail at 3h = 66.7%.
	uptime3_5h := calcUptime(results, now, 210*time.Minute)
	expected3_5 := (2.0 / 3.0) * 100.0
	if uptime3_5h < expected3_5-0.1 || uptime3_5h > expected3_5+0.1 {
		t.Errorf("expected ~%.1f%%, got %.1f%%", expected3_5, uptime3_5h)
	}

	// No results in window = 100%.
	uptimeEmpty := calcUptime(results, now, 1*time.Minute)
	if uptimeEmpty != 100.0 {
		t.Errorf("expected 100%%, got %.1f%%", uptimeEmpty)
	}
}

func TestSummary(t *testing.T) {
	store := newMockStore()
	// Use nil pool since we won't actually schedule.
	e := &Engine{
		store:   store,
		cfg:     DefaultEngineConfig(),
		cancels: make(map[string]func()),
	}

	check := Check{
		ID:             "c1",
		Name:           "Test",
		URL:            "https://example.com",
		Method:         "GET",
		ExpectedStatus: 200,
		Interval:       60 * time.Second,
		Timeout:        5 * time.Second,
		Enabled:        true,
	}
	store.CreateCheck(check)

	now := time.Now()
	store.AddResult(CheckResult{
		CheckID: "c1", Timestamp: now.Add(-1 * time.Hour), Success: true, ResponseMs: 100,
	})
	store.AddResult(CheckResult{
		CheckID: "c1", Timestamp: now.Add(-30 * time.Minute), Success: true, ResponseMs: 200,
	})
	store.AddResult(CheckResult{
		CheckID: "c1", Timestamp: now.Add(-10 * time.Minute), Success: false, ResponseMs: 50, Error: "timeout",
	})

	summaries, err := e.Summary()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	s := summaries[0]
	if s.Check.ID != "c1" {
		t.Errorf("expected check ID c1, got %s", s.Check.ID)
	}
	if s.AvgResponseMs == 0 {
		t.Error("expected non-zero avg response time")
	}
	if s.LastResult == nil {
		t.Error("expected non-nil last result")
	}
	if s.LastFailure == nil {
		t.Error("expected non-nil last failure")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
	if len(id1) == 0 {
		t.Error("expected non-empty ID")
	}
}
