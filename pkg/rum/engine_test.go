package rum

import (
	"testing"
)

func TestClassifyWebVital(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		expect string
	}{
		{"LCP", 1500, "good"},
		{"LCP", 3000, "needs-improvement"},
		{"LCP", 5000, "poor"},
		{"FID", 50, "good"},
		{"FID", 200, "needs-improvement"},
		{"FID", 400, "poor"},
		{"CLS", 0.05, "good"},
		{"CLS", 0.15, "needs-improvement"},
		{"CLS", 0.5, "poor"},
		{"TTFB", 500, "good"},
		{"TTFB", 1000, "needs-improvement"},
		{"TTFB", 2500, "poor"},
		{"FCP", 1000, "good"},
		{"FCP", 2500, "needs-improvement"},
		{"FCP", 4000, "poor"},
		{"INP", 100, "good"},
		{"INP", 300, "needs-improvement"},
		{"INP", 600, "poor"},
	}
	for _, tt := range tests {
		got := ClassifyWebVital(tt.name, tt.value)
		if got != tt.expect {
			t.Errorf("ClassifyWebVital(%q, %v) = %q, want %q", tt.name, tt.value, got, tt.expect)
		}
	}
}

func TestClassifyWebVital_Boundaries(t *testing.T) {
	// Test exact boundaries.
	if got := ClassifyWebVital("LCP", 2500); got != "good" {
		t.Errorf("LCP=2500 should be good, got %q", got)
	}
	if got := ClassifyWebVital("LCP", 4000); got != "needs-improvement" {
		t.Errorf("LCP=4000 should be needs-improvement, got %q", got)
	}
	if got := ClassifyWebVital("FID", 100); got != "good" {
		t.Errorf("FID=100 should be good, got %q", got)
	}
	if got := ClassifyWebVital("CLS", 0.1); got != "good" {
		t.Errorf("CLS=0.1 should be good, got %q", got)
	}
}

func TestFingerprintError(t *testing.T) {
	e1 := &JSErrorEvent{
		Message: "TypeError: Cannot read property 'foo' of undefined",
		Source:  "app.js",
		Stack:   "TypeError: Cannot read property 'foo' of undefined\n    at Object.render (app.js:42:15)\n    at main (app.js:1:1)",
	}
	e2 := &JSErrorEvent{
		Message: "TypeError: Cannot read property 'foo' of undefined",
		Source:  "app.js",
		Stack:   "TypeError: Cannot read property 'foo' of undefined\n    at Object.render (app.js:42:15)\n    at other (app.js:99:1)",
	}
	e3 := &JSErrorEvent{
		Message: "ReferenceError: bar is not defined",
		Source:  "util.js",
		Stack:   "",
	}

	fp1 := FingerprintError(e1)
	fp2 := FingerprintError(e2)
	fp3 := FingerprintError(e3)

	// Same message + source + first stack frame => same fingerprint.
	if fp1 != fp2 {
		t.Errorf("expected same fingerprint for identical first frame, got %q vs %q", fp1, fp2)
	}

	// Different error => different fingerprint.
	if fp1 == fp3 {
		t.Errorf("expected different fingerprints, both got %q", fp1)
	}

	// Fingerprint is non-empty and has expected hex format.
	if len(fp1) != 16 { // 8 bytes => 16 hex chars
		t.Errorf("expected 16-char hex fingerprint, got %q (len %d)", fp1, len(fp1))
	}
}
