package admin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neureaux/cloudmock/pkg/gateway"
)

// SourceServer accepts TCP connections from @cloudmock/node (and other SDKs)
// on a configurable port (default :4580). SDKs send JSON-line messages for
// every inbound HTTP request they capture. The source server converts these
// into gateway.RequestEntry objects and injects them into the shared RequestLog.
type SourceServer struct {
	listener net.Listener
	log      *gateway.RequestLog
	stats    *gateway.RequestStats
	bc       *EventBroadcaster

	mu          sync.Mutex
	sources     map[string]*connectedSource // appName → TCP source info
	httpSources map[string]time.Time        // appName → last seen (HTTP)
	eventCount  atomic.Int64
}

type connectedSource struct {
	AppName   string    `json:"app_name"`
	Runtime   string    `json:"runtime"`
	PID       int       `json:"pid,omitempty"`
	ConnectedAt time.Time `json:"connected_at"`
}

// sdkEvent is the JSON structure sent by the Node SDK over TCP.
type sdkEvent struct {
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Runtime   string          `json:"runtime"`
	Timestamp int64           `json:"timestamp"` // unix ms
	Data      json.RawMessage `json:"data"`
}

// httpInboundData is the payload for type "http:inbound" events.
type httpInboundData struct {
	ID              string            `json:"id"`
	Direction       string            `json:"direction"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Path            string            `json:"path"`
	Status          int               `json:"status"`
	DurationMs      float64           `json:"duration_ms"`
	RequestHeaders  map[string]string `json:"request_headers"`
	ResponseBody    string            `json:"response_body"`
	ContentLength   any               `json:"content_length"`
	UserAgent       string            `json:"user_agent"`
	RemoteAddr      string            `json:"remote_addr"`
}

// NewSourceServer creates a source server but does not start listening yet.
func NewSourceServer(log *gateway.RequestLog, stats *gateway.RequestStats, bc *EventBroadcaster) *SourceServer {
	return &SourceServer{
		log:         log,
		stats:       stats,
		bc:          bc,
		sources:     make(map[string]*connectedSource),
		httpSources: make(map[string]time.Time),
	}
}

// ListenAndServe starts accepting TCP connections on the given address.
// Blocks until the listener is closed.
func (s *SourceServer) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("source server listen %s: %w", addr, err)
	}
	s.listener = ln
	slog.Info("source server listening", "addr", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// listener closed
			return nil
		}
		go s.handleConn(conn)
	}
}

// Close shuts down the listener.
func (s *SourceServer) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// ConnectedSources returns the list of currently connected SDK sources.
func (s *SourceServer) ConnectedSources() []connectedSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]connectedSource, 0, len(s.sources))
	for _, src := range s.sources {
		out = append(out, *src)
	}
	return out
}

func (s *SourceServer) handleConn(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	slog.Info("source server: new connection", "remote", remoteAddr)

	scanner := bufio.NewScanner(conn)
	// Allow up to 64KB per line (response bodies can be large)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	var appName string

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var evt sdkEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			slog.Warn("source server: invalid JSON", "err", err, "remote", remoteAddr)
			continue
		}

		if appName == "" && evt.Source != "" {
			appName = evt.Source
		}

		switch evt.Type {
		case "source:register":
			s.mu.Lock()
			s.sources[evt.Source] = &connectedSource{
				AppName:     evt.Source,
				Runtime:     evt.Runtime,
				ConnectedAt: time.Now(),
			}
			s.mu.Unlock()
			slog.Info("source server: SDK registered", "app", evt.Source, "runtime", evt.Runtime)

		case "http:inbound":
			s.handleHTTPInbound(evt)

		case "http:response":
			// Outbound HTTP from the app — also useful
			s.handleHTTPInbound(evt)

		default:
			// Ignore unknown event types (console, error, etc.)
		}
	}

	// Connection closed — remove from sources
	if appName != "" {
		s.mu.Lock()
		delete(s.sources, appName)
		s.mu.Unlock()
		slog.Info("source server: SDK disconnected", "app", appName)
	}
}

func (s *SourceServer) handleHTTPInbound(evt sdkEvent) {
	entry, err := convertSDKEvent(evt)
	if err != nil {
		slog.Warn("source server: invalid http:inbound data", "err", err)
		return
	}
	s.ingestEntry(entry)
}

func (s *SourceServer) ingestEntry(entry gateway.RequestEntry) {
	s.log.Add(entry)
	if s.stats != nil {
		s.stats.Increment(entry.Service)
	}
	if s.bc != nil {
		s.bc.Broadcast("request", entry)
	}
	s.eventCount.Add(1)

	// Track HTTP source names
	if entry.Service != "" {
		s.mu.Lock()
		s.httpSources[entry.Service] = time.Now()
		s.mu.Unlock()
	}
}

// EventCount returns total events ingested (TCP + HTTP).
func (s *SourceServer) EventCount() int64 {
	return s.eventCount.Load()
}

// HTTPSources returns app names that submitted events via HTTP.
func (s *SourceServer) HTTPSources() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.httpSources))
	for name := range s.httpSources {
		out = append(out, name)
	}
	return out
}

// IngestSDKEvent converts and ingests a single SDK event (used by HTTP endpoint).
func (s *SourceServer) IngestSDKEvent(evt sdkEvent) error {
	entry, err := convertSDKEvent(evt)
	if err != nil {
		return err
	}
	s.ingestEntry(entry)
	return nil
}

// convertSDKEvent transforms an SDK event into a gateway.RequestEntry.
func convertSDKEvent(evt sdkEvent) (gateway.RequestEntry, error) {
	var data httpInboundData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		return gateway.RequestEntry{}, err
	}

	ts := time.Now()
	if evt.Timestamp > 0 {
		ts = time.UnixMilli(evt.Timestamp)
	}

	latencyMs := data.DurationMs
	return gateway.RequestEntry{
		ID:             data.ID,
		Timestamp:      ts,
		Service:        evt.Source,
		Action:         "",
		Method:         data.Method,
		Path:           data.Path,
		StatusCode:     data.Status,
		Latency:        time.Duration(latencyMs * float64(time.Millisecond)),
		LatencyMs:      latencyMs,
		CallerID:       data.RemoteAddr,
		Level:          "app",
		RequestHeaders: data.RequestHeaders,
		ResponseBody:   data.ResponseBody,
	}, nil
}
