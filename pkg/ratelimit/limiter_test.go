package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAllow_UnderLimit verifies that requests within the burst capacity are all allowed.
func TestAllow_UnderLimit(t *testing.T) {
	l := New(10, 10)
	for i := 0; i < 10; i++ {
		if !l.Allow("client1") {
			t.Fatalf("request %d: expected allowed, got denied", i+1)
		}
	}
}

// TestAllow_OverLimit verifies that requests beyond the burst capacity are denied.
func TestAllow_OverLimit(t *testing.T) {
	l := New(10, 5)
	// Exhaust the burst.
	for i := 0; i < 5; i++ {
		if !l.Allow("client1") {
			t.Fatalf("request %d: expected allowed during burst, got denied", i+1)
		}
	}
	// The next request must be denied.
	if l.Allow("client1") {
		t.Fatal("expected denied after burst exhausted, got allowed")
	}
}

// TestAllow_Refill verifies that tokens refill over time and allow requests again.
func TestAllow_Refill(t *testing.T) {
	// 100 tokens/second, burst of 1 — exhausted after one request.
	l := New(100, 1)

	if !l.Allow("client1") {
		t.Fatal("first request: expected allowed")
	}
	if l.Allow("client1") {
		t.Fatal("second request immediately: expected denied")
	}

	// Wait long enough for at least one token to refill (≥10 ms at 100 tok/s).
	time.Sleep(20 * time.Millisecond)

	if !l.Allow("client1") {
		t.Fatal("after refill: expected allowed")
	}
}

// TestMiddleware_Returns429 verifies that the middleware returns 429 with the
// Retry-After header when the rate limit is exceeded.
func TestMiddleware_Returns429(t *testing.T) {
	// Burst of 1 so the second request is denied.
	l := New(1, 1)

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := l.Middleware(ok)

	// First request should pass.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: want 200, got %d", rr.Code)
	}

	// Second request should be rate-limited.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: want 429, got %d", rr2.Code)
	}
	if got := rr2.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After header: want %q, got %q", "1", got)
	}
}

// TestMiddleware_DifferentIPs verifies that different remote addresses maintain
// independent token buckets.
func TestMiddleware_DifferentIPs(t *testing.T) {
	// Burst of 1 per key.
	l := New(1, 1)

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := l.Middleware(ok)

	addrs := []string{"10.0.0.1:1111", "10.0.0.2:2222", "10.0.0.3:3333"}
	for _, addr := range addrs {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("addr %s: first request: want 200, got %d", addr, rr.Code)
		}
	}

	// A second request from each address should be rate-limited independently.
	for _, addr := range addrs {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("addr %s: second request: want 429, got %d", addr, rr.Code)
		}
	}
}
