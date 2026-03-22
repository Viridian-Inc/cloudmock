// Package traceid provides lightweight unique ID generation for distributed tracing.
package traceid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var counter atomic.Uint64

// New generates a unique trace/span ID. It combines a timestamp component
// with a random component and an atomic counter for uniqueness.
func New() string {
	ts := time.Now().UnixNano()
	c := counter.Add(1)

	var buf [8]byte
	_, _ = rand.Read(buf[:])

	return fmt.Sprintf("%x-%s-%x", ts, hex.EncodeToString(buf[:]), c)
}
