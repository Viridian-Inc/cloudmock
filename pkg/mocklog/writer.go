// Package mocklog provides a shared log writer that services use to write
// mock execution logs to CloudWatch Logs via the ServiceLocator.
package mocklog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator resolves other services by name.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// Writer writes mock log events to the CloudWatch Logs service.
type Writer struct {
	locator ServiceLocator
}

// NewWriter creates a log writer that uses the given locator to find the
// CloudWatch Logs service ("logs").
func NewWriter(locator ServiceLocator) *Writer {
	return &Writer{locator: locator}
}

// LogEvent represents a single log entry.
type LogEvent struct {
	Timestamp time.Time
	Message   string
}

// CreateLogGroup creates a log group in CloudWatch Logs.
func (w *Writer) CreateLogGroup(groupName string) error {
	if w.locator == nil {
		return nil // graceful degradation
	}
	logsSvc, err := w.locator.Lookup("logs")
	if err != nil {
		return nil // logs service not available, degrade gracefully
	}

	body, _ := json.Marshal(map[string]string{
		"logGroupName": groupName,
	})
	ctx := &service.RequestContext{
		Action:     "CreateLogGroup",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	_, _ = logsSvc.HandleRequest(ctx)
	return nil
}

// CreateLogStream creates a log stream within a log group.
func (w *Writer) CreateLogStream(groupName, streamName string) error {
	if w.locator == nil {
		return nil
	}
	logsSvc, err := w.locator.Lookup("logs")
	if err != nil {
		return nil
	}

	body, _ := json.Marshal(map[string]string{
		"logGroupName":  groupName,
		"logStreamName": streamName,
	})
	ctx := &service.RequestContext{
		Action:     "CreateLogStream",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	_, _ = logsSvc.HandleRequest(ctx)
	return nil
}

// PutLogEvents writes log events to a CloudWatch Logs stream.
func (w *Writer) PutLogEvents(groupName, streamName string, events []LogEvent) error {
	if w.locator == nil {
		return nil
	}
	logsSvc, err := w.locator.Lookup("logs")
	if err != nil {
		return nil
	}

	cwEvents := make([]map[string]any, len(events))
	for i, e := range events {
		cwEvents[i] = map[string]any{
			"timestamp": e.Timestamp.UnixMilli(),
			"message":   e.Message,
		}
	}

	body, _ := json.Marshal(map[string]any{
		"logGroupName":  groupName,
		"logStreamName": streamName,
		"logEvents":     cwEvents,
	})
	ctx := &service.RequestContext{
		Action:     "PutLogEvents",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	_, _ = logsSvc.HandleRequest(ctx)
	return nil
}

// WriteBuildLog is a convenience method that creates a log group + stream
// and writes a sequence of log lines. Used by CodeBuild, Glue, EMR, etc.
func (w *Writer) WriteBuildLog(groupName, streamName string, lines []string) {
	_ = w.CreateLogGroup(groupName)
	_ = w.CreateLogStream(groupName, streamName)

	events := make([]LogEvent, len(lines))
	now := time.Now().UTC()
	for i, line := range lines {
		events[i] = LogEvent{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Message:   line,
		}
	}
	_ = w.PutLogEvents(groupName, streamName, events)
}

// GenerateBuildLines produces mock build log output for a given service and action.
func GenerateBuildLines(serviceName, buildID string, phases []string) []string {
	lines := make([]string, 0, len(phases)*2+2)
	lines = append(lines, fmt.Sprintf("[%s] Starting %s build %s", serviceName, serviceName, buildID))
	for _, phase := range phases {
		lines = append(lines,
			fmt.Sprintf("[%s] Phase %s started", serviceName, phase),
			fmt.Sprintf("[%s] Phase %s succeeded", serviceName, phase),
		)
	}
	lines = append(lines, fmt.Sprintf("[%s] Build %s completed successfully", serviceName, buildID))
	return lines
}
