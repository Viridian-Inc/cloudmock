// Package traceid provides lightweight unique ID generation for distributed tracing.
package traceid

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

var counter atomic.Uint64

// New generates a unique trace/span ID. It combines a timestamp component
// with a random component and an atomic counter for uniqueness.
// Uses math/rand (not crypto/rand) for performance — these IDs are used for
// log correlation and distributed tracing, not for security purposes.
func New() string {
	ts := time.Now().UnixNano()
	c := counter.Add(1)
	rnd := rand.Uint64()
	return fmt.Sprintf("%x-%x-%x", ts, rnd, c)
}
