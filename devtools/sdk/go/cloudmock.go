// Package cloudmock provides a Go SDK for the CloudMock devtools.
//
// It connects to the devtools source server over TCP and sends JSON-line events
// for HTTP traffic, log messages, and uncaught panics. The SDK is designed to be
// a silent no-op when devtools is not running.
//
// Usage:
//
//	import cloudmock "github.com/Viridian-Inc/cloudmock-sdk-go"
//
//	func main() {
//	    cloudmock.Init(cloudmock.Options{AppName: "my-service"})
//	    defer cloudmock.Close()
//
//	    // Wrap outbound HTTP client
//	    client := &http.Client{Transport: cloudmock.WrapTransport(http.DefaultTransport)}
//
//	    // Wrap inbound HTTP handler
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/api/health", healthHandler)
//	    http.ListenAndServe(":8080", cloudmock.WrapHandler(mux))
//	}
package cloudmock

import (
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Options configures the CloudMock SDK.
type Options struct {
	// AppName is the application name shown in the devtools source bar.
	// Defaults to "go-app".
	AppName string

	// Host is the devtools server host. Defaults to "localhost".
	Host string

	// Port is the devtools server TCP port. Defaults to 4580.
	Port int
}

var (
	globalMu   sync.Mutex
	globalConn *connection
)

// Init initializes the CloudMock devtools SDK and connects to the source server.
// Call once at application startup. No-ops gracefully if devtools is not running.
// Safe to call multiple times; subsequent calls are ignored.
func Init(opts Options) {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalConn != nil {
		return
	}

	if opts.Host == "" {
		opts.Host = "localhost"
	}
	if opts.Port == 0 {
		opts.Port = 4580
	}
	if opts.AppName == "" {
		opts.AppName = "go-app"
	}

	conn := newConnection(opts.Host, opts.Port, opts.AppName)
	globalConn = conn

	// Register this source
	conn.send("source:register", map[string]any{
		"runtime": "go",
		"appName": opts.AppName,
		"pid":     os.Getpid(),
		"goVersion": goVersion(),
	})
}

// Close shuts down the SDK and disconnects from the devtools server.
// Call during application shutdown (typically deferred after Init).
func Close() {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalConn == nil {
		return
	}
	globalConn.close()
	globalConn = nil
}

// WrapTransport returns an http.RoundTripper that captures outbound HTTP calls
// and sends them to the devtools server. If the SDK is not initialized, the
// inner transport is returned unchanged.
//
// Usage:
//
//	client := &http.Client{
//	    Transport: cloudmock.WrapTransport(http.DefaultTransport),
//	}
func WrapTransport(inner http.RoundTripper) http.RoundTripper {
	globalMu.Lock()
	conn := globalConn
	globalMu.Unlock()

	if conn == nil {
		if inner != nil {
			return inner
		}
		return http.DefaultTransport
	}

	if inner == nil {
		inner = http.DefaultTransport
	}

	return &captureTransport{
		inner: inner,
		conn:  conn,
	}
}

// WrapHandler returns an http.Handler middleware that captures inbound HTTP requests
// and sends them to the devtools server. If the SDK is not initialized, the handler
// is returned unchanged.
//
// Usage:
//
//	mux := http.NewServeMux()
//	http.ListenAndServe(":8080", cloudmock.WrapHandler(mux))
func WrapHandler(next http.Handler) http.Handler {
	globalMu.Lock()
	conn := globalConn
	globalMu.Unlock()

	if conn == nil {
		return next
	}

	return newInboundHandler(next, conn)
}

// Log sends a structured log message to the devtools server.
// No-ops silently if the SDK is not initialized.
func Log(level, message string) {
	globalMu.Lock()
	conn := globalConn
	globalMu.Unlock()

	if conn == nil {
		return
	}

	sendLog(conn, level, message)
}

// getConn returns the current global connection, or nil if not initialized.
func getConn() *connection {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalConn
}

func goVersion() string {
	// Use runtime version from environment or build info
	ver := os.Getenv("GOVERSION")
	if ver != "" {
		return ver
	}
	return "unknown"
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
