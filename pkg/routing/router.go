package routing

import (
	"net/http"
	"strings"
)

// serviceAliases maps alternative SigV4 signing names to the canonical
// CloudMock service name. For example, Pulumi/Terraform AWS providers may
// sign S3 Control requests with "s3control" in the credential scope, but
// CloudMock handles them in the S3 service.
var serviceAliases = map[string]string{
	"s3control": "s3",
}

// DetectService determines the AWS service name from the request.
// It checks the Authorization header credential scope first
// (Credential=AKID/date/region/SERVICE/aws4_request), then falls back to
// X-Amz-Target (stripping the version suffix, e.g. "DynamoDB_20120810" → "dynamodb"),
// then checks for presigned URL query parameters (X-Amz-Credential).
// Returns lowercase service name, or empty string if not detected.
func DetectService(r *http.Request) string {
	// Fast path: in-process transport sets this header directly.
	if svc := r.Header.Get("X-Cloudmock-Service"); svc != "" {
		return svc
	}

	// Check Authorization header first.
	if auth := r.Header.Get("Authorization"); auth != "" {
		if svc := serviceFromAuthorization(auth); svc != "" {
			if alias, ok := serviceAliases[svc]; ok {
				return alias
			}
			return svc
		}
	}

	// Fall back to X-Amz-Target header.
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		svc := serviceFromTarget(target)
		if alias, ok := serviceAliases[svc]; ok {
			return alias
		}
		return svc
	}

	// Fall back to presigned URL query parameter (X-Amz-Credential).
	if cred := r.URL.Query().Get("X-Amz-Credential"); cred != "" {
		if svc := serviceFromCredential(cred); svc != "" {
			if alias, ok := serviceAliases[svc]; ok {
				return alias
			}
			return svc
		}
	}

	// Fall back to path-based detection for Cognito OAuth/OIDC endpoints.
	path := r.URL.Path
	if strings.Contains(path, "/.well-known/") || strings.HasPrefix(path, "/oauth2/") || strings.HasPrefix(path, "/login") || strings.HasPrefix(path, "/logout") || strings.HasPrefix(path, "/signup") {
		return "cognito-idp"
	}

	// S3 path-style: /{bucket} or /{bucket}/{key}
	if r.Method == http.MethodGet || r.Method == http.MethodPut || r.Method == http.MethodHead || r.Method == http.MethodDelete {
		if path != "/" && !strings.HasPrefix(path, "/_") {
			return "s3"
		}
	}

	// S3 virtual-hosted-style: bucket.s3.region.amazonaws.com or bucket.s3.localhost
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	if strings.Contains(host, ".s3.") || strings.HasSuffix(strings.Split(host, ":")[0], ".s3") {
		return "s3"
	}

	return ""
}

// serviceFromCredential extracts the service name from a presigned URL
// X-Amz-Credential query parameter value: AKID/date/region/service/aws4_request.
func serviceFromCredential(cred string) string {
	parts := strings.Split(cred, "/")
	if len(parts) < 4 {
		return ""
	}
	return strings.ToLower(parts[3])
}

// DetectAction determines the AWS API action from the request.
// It checks X-Amz-Target (part after the dot) first, then the ?Action= query parameter.
// Returns empty string if not detected.
func DetectAction(r *http.Request) string {
	// Check X-Amz-Target first — fast path, no allocation.
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 {
			return target[dot+1:]
		}
	}

	// Fall back to query string — avoid r.URL.Query() allocation when possible.
	if q := r.URL.RawQuery; q != "" {
		// Fast scan for Action= in the raw query string.
		const prefix = "Action="
		idx := strings.Index(q, prefix)
		if idx >= 0 {
			val := q[idx+len(prefix):]
			if end := strings.IndexByte(val, '&'); end >= 0 {
				return val[:end]
			}
			return val
		}
	}
	return ""
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

	// Walk to the 4th slash-delimited field (AKID/date/region/SERVICE/aws4_request)
	// without allocating a []string from Split.
	slashes := 0
	start := 0
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			slashes++
			if slashes == 3 {
				start = i + 1
			}
			if slashes == 4 {
				svc := rest[start:i]
				// Lowercase in-place check (avoid ToLower allocation for already-lowercase)
				allLower := true
				for j := 0; j < len(svc); j++ {
					if svc[j] >= 'A' && svc[j] <= 'Z' {
						allLower = false
						break
					}
				}
				if allLower {
					return svc
				}
				return strings.ToLower(svc)
			}
		}
		if rest[i] == ',' || rest[i] == ' ' {
			break
		}
	}
	return ""
}

// targetToService maps the lowercased X-Amz-Target service prefix to the
// canonical CloudMock service name. This handles services whose target
// prefix doesn't match their SigV4 signing name (e.g. Cognito uses
// "AWSCognitoIdentityProviderService" in X-Amz-Target but "cognito-idp"
// in the credential scope).
// TargetToService maps the lowercased X-Amz-Target service prefix to the
// canonical CloudMock service name.
var TargetToService = map[string]string{
	"dynamodb":                              "dynamodb",
	"dax":                                   "dynamodb",
	"amazonsqs":                             "sqs",
	"amazonses":                             "ses",
	"amazonsns":                             "sns",
	"awskms":                                "kms",
	"amazonkinesis":                         "kinesis",
	"tagging":                               "tagging",
	"logs":                                  "logs",
	"awslambda":                             "lambda",
	"awscognitoidentityproviderservice":     "cognito-idp",
	"secretsmanager":                        "secretsmanager",
}

// serviceFromTarget extracts the service name from an X-Amz-Target value like
// "DynamoDB_20120810.CreateTable". The service portion is the part before the
// dot, with any underscore-delimited version suffix stripped, lowercased.
// If the resulting name appears in targetToService, the mapped canonical
// name is returned instead.
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

	lower := strings.ToLower(svc)

	// Check for a known mapping (e.g. "awscognitoidentityproviderservice" → "cognito-idp").
	if mapped, ok := TargetToService[lower]; ok {
		return mapped
	}

	return lower
}
