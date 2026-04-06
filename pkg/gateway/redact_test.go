package gateway

import (
	"strings"
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	cfg := DefaultRedactionConfig()

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer eyJhbGciOiJSUzI1NiJ9.secret",
		"Cookie":        "session=abc123; user=alice",
		"X-API-Key":     "cmk_live_abcdef123456",
		"X-Request-Id":  "req-12345",
	}

	result := cfg.RedactRequestHeaders(headers)

	if result["Content-Type"] != "application/json" {
		t.Errorf("Content-Type should not be redacted, got %q", result["Content-Type"])
	}
	if result["X-Request-Id"] != "req-12345" {
		t.Errorf("X-Request-Id should not be redacted, got %q", result["X-Request-Id"])
	}
	if result["Authorization"] != "[REDACTED]" {
		t.Errorf("Authorization should be redacted, got %q", result["Authorization"])
	}
	if result["Cookie"] != "[REDACTED]" {
		t.Errorf("Cookie should be redacted, got %q", result["Cookie"])
	}
	if result["X-API-Key"] != "[REDACTED]" {
		t.Errorf("X-API-Key should be redacted, got %q", result["X-API-Key"])
	}
}

func TestRedactHeaders_CaseInsensitive(t *testing.T) {
	cfg := DefaultRedactionConfig()

	headers := map[string]string{
		"authorization": "Bearer secret",
		"COOKIE":        "session=abc",
	}

	result := cfg.RedactRequestHeaders(headers)

	if result["authorization"] != "[REDACTED]" {
		t.Errorf("authorization (lowercase) should be redacted, got %q", result["authorization"])
	}
	if result["COOKIE"] != "[REDACTED]" {
		t.Errorf("COOKIE (uppercase) should be redacted, got %q", result["COOKIE"])
	}
}

func TestRedactHeaders_Disabled(t *testing.T) {
	cfg := DefaultRedactionConfig()
	cfg.Enabled = false

	headers := map[string]string{
		"Authorization": "Bearer secret",
	}

	result := cfg.RedactRequestHeaders(headers)
	if result["Authorization"] != "Bearer secret" {
		t.Error("redaction should be disabled")
	}
}

func TestRedactBody_JSONFields(t *testing.T) {
	cfg := DefaultRedactionConfig()

	body := `{"username":"alice","password":"s3cret","email":"alice@example.com"}`
	result := cfg.RedactBody(body)

	if strings.Contains(result, "s3cret") {
		t.Error("password value should be redacted")
	}
	if !strings.Contains(result, `"password": "[REDACTED]"`) {
		t.Errorf("password should show [REDACTED], got: %s", result)
	}
	// username is not in the default redact list
	if !strings.Contains(result, "alice") {
		t.Error("username should not be redacted")
	}
}

func TestRedactBody_SSN(t *testing.T) {
	cfg := DefaultRedactionConfig()

	body := `{"ssn":"123-45-6789","name":"Alice"}`
	result := cfg.RedactBody(body)

	if strings.Contains(result, "123-45-6789") {
		t.Error("SSN pattern should be redacted from body")
	}
}

func TestRedactBody_MedicalFields(t *testing.T) {
	cfg := DefaultRedactionConfig()

	body := `{"diagnosis":"diabetes","medication":"metformin","name":"Alice"}`
	result := cfg.RedactBody(body)

	if strings.Contains(result, "diabetes") {
		t.Error("diagnosis value should be redacted")
	}
	if strings.Contains(result, "metformin") {
		t.Error("medication value should be redacted")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("name should not be redacted by default")
	}
}

func TestRedactBody_Disabled(t *testing.T) {
	cfg := DefaultRedactionConfig()
	cfg.Enabled = false

	body := `{"password":"secret"}`
	result := cfg.RedactBody(body)

	if result != body {
		t.Error("redaction should be disabled")
	}
}

func TestRedactBody_Empty(t *testing.T) {
	cfg := DefaultRedactionConfig()
	if cfg.RedactBody("") != "" {
		t.Error("empty body should return empty")
	}
}
