package common

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAWSEnvVars(t *testing.T) {
	vars := AWSEnvVars("http://localhost:4566", "us-east-1", "mykey", "mysecret")

	expected := map[string]string{
		"AWS_ENDPOINT_URL":       "http://localhost:4566",
		"AWS_DEFAULT_REGION":     "us-east-1",
		"AWS_REGION":             "us-east-1",
		"AWS_ACCESS_KEY_ID":      "mykey",
		"AWS_SECRET_ACCESS_KEY":  "mysecret",
		"CLOUDMOCK_ENDPOINT":     "http://localhost:4566",
	}

	if len(vars) != len(expected) {
		t.Fatalf("expected %d vars, got %d", len(expected), len(vars))
	}

	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("invalid env var format: %s", v)
		}
		key, val := parts[0], parts[1]
		if exp, ok := expected[key]; !ok {
			t.Errorf("unexpected env var: %s", key)
		} else if val != exp {
			t.Errorf("env var %s: expected %q, got %q", key, exp, val)
		}
	}
}

func TestDetectEndpoint_Default(t *testing.T) {
	os.Unsetenv("CLOUDMOCK_ENDPOINT")
	ep := DetectEndpoint()
	if ep != DefaultEndpoint {
		t.Errorf("expected %q, got %q", DefaultEndpoint, ep)
	}
}

func TestDetectEndpoint_FromEnv(t *testing.T) {
	os.Setenv("CLOUDMOCK_ENDPOINT", "http://custom:9999")
	defer os.Unsetenv("CLOUDMOCK_ENDPOINT")

	ep := DetectEndpoint()
	if ep != "http://custom:9999" {
		t.Errorf("expected %q, got %q", "http://custom:9999", ep)
	}
}

func TestWaitForHealth_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cloudmock/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := WaitForHealth(srv.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWaitForHealth_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := WaitForHealth(srv.URL, 1*time.Second)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "not healthy") {
		t.Errorf("expected 'not healthy' in error, got: %v", err)
	}
}

func TestWaitForHealth_EventuallyHealthy(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := WaitForHealth(srv.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls, got %d", calls)
	}
}
