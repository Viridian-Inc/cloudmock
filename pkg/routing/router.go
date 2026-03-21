package routing

import (
	"net/http"
	"strings"
)

// DetectService determines the AWS service name from the request.
// It checks the Authorization header credential scope first
// (Credential=AKID/date/region/SERVICE/aws4_request), then falls back to
// X-Amz-Target (stripping the version suffix, e.g. "DynamoDB_20120810" → "dynamodb").
// Returns lowercase service name, or empty string if not detected.
func DetectService(r *http.Request) string {
	// Check Authorization header first.
	if auth := r.Header.Get("Authorization"); auth != "" {
		if svc := serviceFromAuthorization(auth); svc != "" {
			return svc
		}
	}

	// Fall back to X-Amz-Target header.
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		return serviceFromTarget(target)
	}

	return ""
}

// DetectAction determines the AWS API action from the request.
// It checks X-Amz-Target (part after the dot) first, then the ?Action= query parameter.
// Returns empty string if not detected.
func DetectAction(r *http.Request) string {
	// Check X-Amz-Target first.
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 {
			return target[dot+1:]
		}
	}

	// Fall back to query string.
	return r.URL.Query().Get("Action")
}

// serviceFromAuthorization extracts the service name from an AWS4-HMAC-SHA256
// Authorization header. The Credential value has the form:
//
//	AKID/date/region/service/aws4_request
func serviceFromAuthorization(auth string) string {
	// Find "Credential=" token.
	const prefix = "Credential="
	idx := strings.Index(auth, prefix)
	if idx < 0 {
		return ""
	}
	rest := auth[idx+len(prefix):]

	// The credential ends at a comma or whitespace.
	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}

	// rest is now AKID/date/region/service/aws4_request
	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		return ""
	}
	// parts[0]=AKID, [1]=date, [2]=region, [3]=service
	return strings.ToLower(parts[3])
}

// serviceFromTarget extracts the service name from an X-Amz-Target value like
// "DynamoDB_20120810.CreateTable". The service portion is the part before the
// dot, with any underscore-delimited version suffix stripped, lowercased.
func serviceFromTarget(target string) string {
	// Take the part before the dot.
	svc := target
	if dot := strings.Index(target, "."); dot >= 0 {
		svc = target[:dot]
	}

	// Strip version suffix (everything after the first underscore).
	if under := strings.Index(svc, "_"); under >= 0 {
		svc = svc[:under]
	}

	return strings.ToLower(svc)
}
