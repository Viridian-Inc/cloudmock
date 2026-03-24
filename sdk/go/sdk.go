// Package sdk provides helpers for building CloudMock plugins in Go.
//
// A plugin binary implements the plugin.Plugin interface and calls
// sdk.Serve to start listening for requests from the CloudMock core.
//
// Example:
//
//	func main() {
//	    sdk.Serve(&MyPlugin{})
//	}
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/neureaux/cloudmock/pkg/plugin"
)

// Serve starts an HTTP server that bridges incoming requests to the given plugin.
// The core launches plugin binaries and communicates the listen address via the
// CLOUDMOCK_PLUGIN_ADDR environment variable (default ":0" for auto-assign).
//
// This is a simple HTTP bridge for Phase 1. It will be replaced with gRPC in
// a future phase for better streaming support.
func Serve(p plugin.Plugin) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	addr := os.Getenv("CLOUDMOCK_PLUGIN_ADDR")
	if addr == "" {
		addr = ":0"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	// Initialize the plugin.
	configBytes := []byte(os.Getenv("CLOUDMOCK_PLUGIN_CONFIG"))
	dataDir := os.Getenv("CLOUDMOCK_PLUGIN_DATA_DIR")
	logLevel := os.Getenv("CLOUDMOCK_PLUGIN_LOG_LEVEL")
	if err := p.Init(context.Background(), configBytes, dataDir, logLevel); err != nil {
		logger.Error("plugin init failed", "error", err)
		os.Exit(1)
	}

	// Get descriptor for logging.
	desc, _ := p.Describe(context.Background())
	if desc != nil {
		logger.Info("plugin starting", "name", desc.Name, "version", desc.Version, "addr", ln.Addr().String())
	}

	// Print the listen address to stdout so the core can discover it.
	fmt.Fprintf(os.Stdout, "PLUGIN_ADDR=%s\n", ln.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/describe", func(w http.ResponseWriter, r *http.Request) {
		desc, err := p.Describe(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(desc)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status, msg, err := p.HealthCheck(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  status.String(),
			"message": msg,
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req plugin.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		resp, err := p.HandleRequest(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := &http.Server{Handler: mux}

	// Graceful shutdown on SIGTERM/SIGINT.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		logger.Info("shutting down plugin")
		p.Shutdown(context.Background())
		server.Close()
	}()

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
