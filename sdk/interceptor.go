package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/traffic"
)

// Recorder is an http.RoundTripper that wraps another RoundTripper and
// captures all request/response pairs as traffic.CapturedEntry values.
// It is intended to be used with the AWS SDK v2 HTTP client for recording
// real AWS traffic.
type Recorder struct {
	inner     http.RoundTripper
	entries   []*traffic.CapturedEntry
	mu        sync.Mutex
	startTime time.Time
	entrySeq  int
}

// NewRecorder creates a Recorder that wraps http.DefaultTransport.
func NewRecorder() *Recorder {
	return &Recorder{
		inner:     http.DefaultTransport,
		startTime: time.Now(),
	}
}

// Wrap sets the inner RoundTripper and returns the Recorder for chaining.
func (r *Recorder) Wrap(inner http.RoundTripper) *Recorder {
	r.inner = inner
	return r
}

// RoundTrip implements http.RoundTripper. It captures the request and response
// as a CapturedEntry, then returns the original response unchanged.
func (r *Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Read and buffer the request body so we can capture it.
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	resp, err := r.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Read and buffer the response body so we can capture it,
	// then replace it so the caller can still read it.
	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
	}

	latencyMs := float64(time.Since(start).Nanoseconds()) / 1e6

	// Detect service and action.
	svc := serviceFromAuth(req)
	action := actionFromRequest(req)

	headers := make(map[string]string, len(req.Header))
	for k := range req.Header {
		headers[k] = req.Header.Get(k)
	}

	r.mu.Lock()
	r.entrySeq++
	entry := &traffic.CapturedEntry{
		ID:             fmt.Sprintf("rec-%d", r.entrySeq),
		Timestamp:      start,
		Service:        svc,
		Action:         action,
		Method:         req.Method,
		Path:           req.URL.Path,
		StatusCode:     resp.StatusCode,
		LatencyMs:      latencyMs,
		RequestHeaders: headers,
		RequestBody:    string(reqBody),
		ResponseBody:   string(respBody),
		OffsetMs:       float64(start.Sub(r.startTime).Milliseconds()),
	}
	r.entries = append(r.entries, entry)
	r.mu.Unlock()

	return resp, nil
}

// Recording converts the captured entries into a traffic.Recording.
func (r *Recorder) Recording() *traffic.Recording {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entries := make([]traffic.CapturedEntry, len(r.entries))
	for i, e := range r.entries {
		entries[i] = *e
	}

	return &traffic.Recording{
		ID:          fmt.Sprintf("sdk-rec-%d", now.UnixMilli()),
		Name:        "sdk-interceptor-recording",
		Status:      traffic.RecordingCompleted,
		StartedAt:   r.startTime,
		StoppedAt:   &now,
		EntryCount:  len(entries),
		Entries:     entries,
		DurationSec: int(now.Sub(r.startTime).Seconds()),
	}
}

// SaveToFile marshals the recording to JSON and writes it to the given path.
func (r *Recorder) SaveToFile(path string) error {
	rec := r.Recording()
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recording: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Entries returns a copy of the captured entries.
func (r *Recorder) Entries() []*traffic.CapturedEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*traffic.CapturedEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

// serviceFromAuth extracts the AWS service name from the Authorization header.
func serviceFromAuth(req *http.Request) string {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Credential="
	idx := strings.Index(auth, prefix)
	if idx < 0 {
		return ""
	}
	rest := auth[idx+len(prefix):]
	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		return ""
	}
	return strings.ToLower(parts[3])
}

// actionFromRequest extracts the AWS action from the request.
func actionFromRequest(req *http.Request) string {
	if target := req.Header.Get("X-Amz-Target"); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 {
			return target[dot+1:]
		}
	}
	return req.URL.Query().Get("Action")
}
