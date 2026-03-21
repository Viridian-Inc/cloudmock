package iam

import (
	"fmt"
	"net/http"
	"strings"
)

// ExtractAccessKeyID parses the AWS SigV4 Authorization header from the request
// and returns the access key ID embedded in the Credential field.
//
// Expected header format:
//
//	AWS4-HMAC-SHA256 Credential=<AKID>/<date>/<region>/<service>/aws4_request, ...
func ExtractAccessKeyID(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	// Find the Credential= part
	const credPrefix = "Credential="
	credIdx := strings.Index(authHeader, credPrefix)
	if credIdx == -1 {
		return "", fmt.Errorf("Authorization header missing Credential field")
	}

	// Extract the value after "Credential="
	credValue := authHeader[credIdx+len(credPrefix):]

	// Trim trailing comma and whitespace if present
	if commaIdx := strings.IndexByte(credValue, ','); commaIdx != -1 {
		credValue = credValue[:commaIdx]
	}
	credValue = strings.TrimSpace(credValue)

	// Split by "/" — first element is the access key ID
	parts := strings.SplitN(credValue, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("empty access key ID in Credential field")
	}

	return parts[0], nil
}
