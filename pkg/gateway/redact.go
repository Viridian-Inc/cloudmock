package gateway

import (
	"regexp"
	"strings"
)

// RedactionConfig controls which headers and body fields are redacted
// before storage in traces, request logs, and audit entries.
//
// When HIPAA mode is enabled, sensitive headers are replaced with
// "[REDACTED]" and body fields matching PII patterns are masked.
type RedactionConfig struct {
	// Enabled turns on field redaction for all stored data.
	Enabled bool

	// RedactHeaders is a list of header names to redact (case-insensitive).
	// Defaults: Authorization, Cookie, Set-Cookie, X-API-Key, X-Auth-Token.
	RedactHeaders []string

	// RedactBodyFields is a list of JSON field names whose values are redacted.
	// Defaults: password, secret, token, ssn, social_security, credit_card,
	// card_number, cvv, date_of_birth, dob, medical_record, diagnosis.
	RedactBodyFields []string

	// RedactBodyPatterns is a list of regex patterns to redact in body text.
	// Defaults: SSN pattern, email addresses.
	RedactBodyPatterns []*regexp.Regexp
}

// DefaultRedactionConfig returns the default HIPAA-safe redaction config.
func DefaultRedactionConfig() *RedactionConfig {
	return &RedactionConfig{
		Enabled: true,
		RedactHeaders: []string{
			"authorization",
			"cookie",
			"set-cookie",
			"x-api-key",
			"x-auth-token",
			"x-session-token",
			"x-csrf-token",
			"proxy-authorization",
		},
		RedactBodyFields: []string{
			"password", "passwd", "secret", "token", "access_token",
			"refresh_token", "id_token", "session_token", "api_key",
			"ssn", "social_security", "social_security_number",
			"credit_card", "card_number", "cvv", "cvc", "expiry",
			"date_of_birth", "dob", "birth_date",
			"medical_record", "diagnosis", "medication",
			"insurance_id", "policy_number", "member_id",
			"phone", "phone_number", "address", "zip_code",
		},
		RedactBodyPatterns: []*regexp.Regexp{
			// SSN: 123-45-6789 or 123456789
			regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
		},
	}
}

const redactedValue = "[REDACTED]"

// redactHeadersSet caches the lowercase header set for fast lookup.
var redactHeadersSet map[string]bool

// RedactHeaders returns a copy of headers with sensitive values replaced.
func (c *RedactionConfig) RedactRequestHeaders(headers map[string]string) map[string]string {
	if !c.Enabled || len(headers) == 0 {
		return headers
	}

	// Build lookup set on first call.
	if redactHeadersSet == nil {
		redactHeadersSet = make(map[string]bool, len(c.RedactHeaders))
		for _, h := range c.RedactHeaders {
			redactHeadersSet[strings.ToLower(h)] = true
		}
	}

	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if redactHeadersSet[strings.ToLower(k)] {
			result[k] = redactedValue
		} else {
			result[k] = v
		}
	}
	return result
}

// RedactBody scrubs sensitive JSON field values and PII patterns from body text.
func (c *RedactionConfig) RedactBody(body string) string {
	if !c.Enabled || body == "" {
		return body
	}

	result := body

	// Redact known JSON field values: "field_name": "value" → "field_name": "[REDACTED]"
	for _, field := range c.RedactBodyFields {
		// Match "field": "any value" or "field":"any value" (with optional whitespace)
		pattern := `"` + regexp.QuoteMeta(field) + `"\s*:\s*"[^"]*"`
		re := regexp.MustCompile("(?i)" + pattern)
		result = re.ReplaceAllString(result, `"`+field+`": "`+redactedValue+`"`)
	}

	// Redact PII patterns (SSN, etc.)
	for _, re := range c.RedactBodyPatterns {
		result = re.ReplaceAllString(result, redactedValue)
	}

	return result
}
