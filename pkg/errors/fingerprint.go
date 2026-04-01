package errors

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Fingerprint computes a stable fingerprint for an error event based on
// the message and the first 3 stack frames.
func Fingerprint(message, stack string) string {
	var parts []string
	parts = append(parts, message)

	// Take the first 3 lines of the stack trace as frames.
	lines := strings.Split(strings.TrimSpace(stack), "\n")
	limit := 3
	if len(lines) < limit {
		limit = len(lines)
	}
	for i := 0; i < limit; i++ {
		parts = append(parts, strings.TrimSpace(lines[i]))
	}

	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return fmt.Sprintf("%x", h[:16]) // 32 hex chars
}
