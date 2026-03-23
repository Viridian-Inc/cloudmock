package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// bucket holds the token state for a single rate-limit key.
type bucket struct {
	tokens   float64
	lastFill time.Time
}

// Limiter is a token-bucket rate limiter keyed by an arbitrary string (e.g. IP address).
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   int
}

// New returns a Limiter that allows rate tokens/second with a burst capacity of burst.
func New(rate float64, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow reports whether the request identified by key should be allowed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.burst), lastFill: time.Now()}
		l.buckets[key] = b
	}

	// Refill tokens based on elapsed time.
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastFill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware returns an http.Handler that enforces the rate limit using the
// request's RemoteAddr as the bucket key. Requests that exceed the limit
// receive a 429 Too Many Requests response with a Retry-After: 1 header.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr // IP-based
		if !l.Allow(key) {
			w.Header().Set("Retry-After", strconv.Itoa(1))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
