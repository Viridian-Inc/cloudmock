package memory

import (
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/uptime"
)

func testCheck(id string) uptime.Check {
	return uptime.Check{
		ID:             id,
		Name:           "Test Check " + id,
		URL:            "https://example.com",
		Method:         "GET",
		ExpectedStatus: 200,
		Interval:       60 * time.Second,
		Timeout:        5 * time.Second,
		Enabled:        true,
	}
}

func TestCreateAndListChecks(t *testing.T) {
	s := NewStore(100)

	if err := s.CreateCheck(testCheck("c1")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCheck(testCheck("c2")); err != nil {
		t.Fatal(err)
	}

	checks, err := s.ListChecks()
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := NewStore(100)
	if err := s.CreateCheck(testCheck("c1")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCheck(testCheck("c1")); err == nil {
		t.Fatal("expected error for duplicate check ID")
	}
}

func TestUpdateCheck(t *testing.T) {
	s := NewStore(100)
	c := testCheck("c1")
	s.CreateCheck(c)

	c.Name = "Updated"
	if err := s.UpdateCheck(c); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetCheck("c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" {
		t.Errorf("expected name Updated, got %s", got.Name)
	}
}

func TestDeleteCheck(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	if err := s.DeleteCheck("c1"); err != nil {
		t.Fatal(err)
	}

	_, err := s.GetCheck("c1")
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestAddAndRetrieveResults(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	now := time.Now()
	for i := 0; i < 5; i++ {
		s.AddResult(uptime.CheckResult{
			CheckID:    "c1",
			Timestamp:  now.Add(time.Duration(i) * time.Second),
			StatusCode: 200,
			ResponseMs: 50.0,
			Success:    true,
		})
	}

	results, err := s.Results("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	// Newest first.
	if results[0].Timestamp.Before(results[4].Timestamp) {
		t.Error("results should be newest first")
	}
}

func TestCircularBuffer(t *testing.T) {
	s := NewStore(3)
	s.CreateCheck(testCheck("c1"))

	for i := 0; i < 5; i++ {
		s.AddResult(uptime.CheckResult{
			CheckID:    "c1",
			Timestamp:  time.Now().Add(time.Duration(i) * time.Second),
			StatusCode: 200,
			ResponseMs: 50.0,
			Success:    true,
		})
	}

	results, _ := s.Results("c1")
	if len(results) != 3 {
		t.Fatalf("expected 3 results (buffer wrapped), got %d", len(results))
	}
}

func TestLastResult(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	// No results yet.
	lr, _ := s.LastResult("c1")
	if lr != nil {
		t.Fatal("expected nil last result")
	}

	s.AddResult(uptime.CheckResult{
		CheckID:    "c1",
		Timestamp:  time.Now(),
		StatusCode: 200,
		Success:    true,
	})
	s.AddResult(uptime.CheckResult{
		CheckID:    "c1",
		Timestamp:  time.Now().Add(time.Second),
		StatusCode: 500,
		Success:    false,
		Error:      "server error",
	})

	lr, _ = s.LastResult("c1")
	if lr == nil {
		t.Fatal("expected non-nil last result")
	}
	if lr.StatusCode != 500 {
		t.Errorf("expected 500, got %d", lr.StatusCode)
	}
}

func TestLastFailure(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	s.AddResult(uptime.CheckResult{
		CheckID: "c1", Timestamp: time.Now(), Success: false, Error: "err1",
	})
	s.AddResult(uptime.CheckResult{
		CheckID: "c1", Timestamp: time.Now().Add(time.Second), Success: true,
	})

	lf, _ := s.LastFailure("c1")
	if lf == nil {
		t.Fatal("expected non-nil last failure")
	}
	if lf.Error != "err1" {
		t.Errorf("expected err1, got %s", lf.Error)
	}
}

func TestConsecutiveFailures(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	// Add success then 3 failures.
	s.AddResult(uptime.CheckResult{CheckID: "c1", Timestamp: time.Now(), Success: true})
	for i := 0; i < 3; i++ {
		s.AddResult(uptime.CheckResult{
			CheckID:   "c1",
			Timestamp: time.Now().Add(time.Duration(i+1) * time.Second),
			Success:   false,
		})
	}

	cf := s.ConsecutiveFailures("c1")
	if cf != 3 {
		t.Errorf("expected 3 consecutive failures, got %d", cf)
	}
}

func TestConsecutiveFailuresInterrupted(t *testing.T) {
	s := NewStore(100)
	s.CreateCheck(testCheck("c1"))

	s.AddResult(uptime.CheckResult{CheckID: "c1", Timestamp: time.Now(), Success: false})
	s.AddResult(uptime.CheckResult{CheckID: "c1", Timestamp: time.Now().Add(time.Second), Success: true})
	s.AddResult(uptime.CheckResult{CheckID: "c1", Timestamp: time.Now().Add(2 * time.Second), Success: false})

	cf := s.ConsecutiveFailures("c1")
	if cf != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", cf)
	}
}
