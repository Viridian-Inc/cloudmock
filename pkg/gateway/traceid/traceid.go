// Package traceid provides lightweight unique ID generation for distributed tracing.
package traceid

import (
	"encoding/hex"
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

// GenerateW3CTraceID returns a 32-character hex string (16 random bytes)
// compatible with the W3C Trace Context specification.
func GenerateW3CTraceID() string {
	var buf [16]byte
	r1 := rand.Uint64()
	r2 := rand.Uint64()
	buf[0] = byte(r1 >> 56)
	buf[1] = byte(r1 >> 48)
	buf[2] = byte(r1 >> 40)
	buf[3] = byte(r1 >> 32)
	buf[4] = byte(r1 >> 24)
	buf[5] = byte(r1 >> 16)
	buf[6] = byte(r1 >> 8)
	buf[7] = byte(r1)
	buf[8] = byte(r2 >> 56)
	buf[9] = byte(r2 >> 48)
	buf[10] = byte(r2 >> 40)
	buf[11] = byte(r2 >> 32)
	buf[12] = byte(r2 >> 24)
	buf[13] = byte(r2 >> 16)
	buf[14] = byte(r2 >> 8)
	buf[15] = byte(r2)
	return hex.EncodeToString(buf[:])
}

// GenerateW3CSpanID returns a 16-character hex string (8 random bytes)
// compatible with the W3C Trace Context specification.
func GenerateW3CSpanID() string {
	var buf [8]byte
	r := rand.Uint64()
	buf[0] = byte(r >> 56)
	buf[1] = byte(r >> 48)
	buf[2] = byte(r >> 40)
	buf[3] = byte(r >> 32)
	buf[4] = byte(r >> 24)
	buf[5] = byte(r >> 16)
	buf[6] = byte(r >> 8)
	buf[7] = byte(r)
	return hex.EncodeToString(buf[:])
}
