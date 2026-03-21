package common

import "os"

// DetectEndpoint returns the cloudmock endpoint URL.
// It checks CLOUDMOCK_ENDPOINT env var first, then falls back to DefaultEndpoint.
func DetectEndpoint() string {
	if ep := os.Getenv("CLOUDMOCK_ENDPOINT"); ep != "" {
		return ep
	}
	return DefaultEndpoint
}
