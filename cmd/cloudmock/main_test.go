package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/health", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"services": map[string]bool{
				"s3":       true,
				"dynamodb": true,
			},
		})
	}))
	defer server.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdStatus(server.URL)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "healthy")
	assert.Contains(t, output, "s3")
	assert.Contains(t, output, "dynamodb")
}

func TestCmdServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "s3", "action_count": 5, "healthy": true},
			{"name": "sqs", "action_count": 3, "healthy": true},
		})
	}))
	defer server.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdServices(server.URL)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "s3")
	assert.Contains(t, output, "sqs")
	assert.Contains(t, output, "5")
}

func TestCmdReset_All(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/reset", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "reset",
			"services": []string{"s3", "sqs"},
		})
	}))
	defer server.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdReset(server.URL, []string{})

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "Reset 2 services")
}

func TestCmdReset_SingleService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasPrefix(r.URL.Path, "/api/services/s3/reset"))
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "reset",
			"service": "s3",
		})
	}))
	defer server.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdReset(server.URL, []string{"-service", "s3"})

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "Reset service: s3")
}

func TestCmdConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/config", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"region":     "us-east-1",
			"account_id": "000000000000",
		})
	}))
	defer server.Close()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdConfig(server.URL)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "us-east-1")
	assert.Contains(t, output, "000000000000")
}

func TestCmdVersion(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdVersion()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	assert.Contains(t, output, "cloudmock version")
	assert.Contains(t, output, version)
}
