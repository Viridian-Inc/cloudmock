package mocklog

import (
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

type fakeLogsService struct {
	groups  map[string]bool
	streams map[string]bool
	events  int
}

func (f *fakeLogsService) Name() string             { return "logs" }
func (f *fakeLogsService) Actions() []service.Action { return nil }
func (f *fakeLogsService) HealthCheck() error        { return nil }
func (f *fakeLogsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateLogGroup":
		f.groups[string(ctx.Body)] = true
	case "CreateLogStream":
		f.streams[string(ctx.Body)] = true
	case "PutLogEvents":
		f.events++
	}
	return &service.Response{StatusCode: 200, Body: map[string]any{}, Format: service.FormatJSON}, nil
}

type fakeLookup struct {
	svc service.Service
}

func (f *fakeLookup) Lookup(name string) (service.Service, error) {
	if name == "logs" && f.svc != nil {
		return f.svc, nil
	}
	return nil, service.NewAWSError("ServiceUnavailable", "not found", 503)
}

func TestWriteBuildLog(t *testing.T) {
	logs := &fakeLogsService{
		groups:  make(map[string]bool),
		streams: make(map[string]bool),
	}
	w := NewWriter(&fakeLookup{svc: logs})
	w.WriteBuildLog("/codebuild/builds", "build-1", []string{"line1", "line2"})

	if len(logs.groups) == 0 {
		t.Error("expected log group to be created")
	}
	if len(logs.streams) == 0 {
		t.Error("expected log stream to be created")
	}
	if logs.events == 0 {
		t.Error("expected log events to be written")
	}
}

func TestWriter_NilLocator(t *testing.T) {
	w := NewWriter(nil)
	// Should not panic
	w.WriteBuildLog("/test", "stream", []string{"line"})
}

func TestGenerateBuildLines(t *testing.T) {
	lines := GenerateBuildLines("codebuild", "build-123", []string{"INSTALL", "BUILD"})
	if len(lines) != 6 { // start + 2*(start+end) + finish
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
}
