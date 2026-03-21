package sts

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// temporaryCredentials holds generated AWS temporary credentials.
type temporaryCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// generateCredentials returns a new set of random temporary credentials.
// The access key uses the ASIA prefix as real AWS does for STS-issued keys.
func generateCredentials() temporaryCredentials {
	return temporaryCredentials{
		AccessKeyID:     "ASIA" + randomHex(16),
		SecretAccessKey: randomHex(20),
		SessionToken:    randomHex(40),
		Expiration:      time.Now().UTC().Add(time.Hour),
	}
}

// newRequestID returns a random UUID-shaped request identifier.
func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// randomHex returns n random bytes encoded as a hex string (length 2n).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
