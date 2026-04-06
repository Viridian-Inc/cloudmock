package eventbridge_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- Archives ----

func TestEB_ArchiveLifecycle(t *testing.T) {
	handler := newEBGateway(t)

	// CreateArchive
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "CreateArchive", map[string]any{
		"ArchiveName":    "my-archive",
		"EventSourceArn": "arn:aws:events:us-east-1:000000000000:event-bus/default",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateArchive: %d %s", w.Code, w.Body.String())
	}

	// DescribeArchive
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeArchive", map[string]string{
		"ArchiveName": "my-archive",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeArchive: %d %s", w.Code, w.Body.String())
	}

	// ListArchives
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListArchives", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListArchives: %d %s", w.Code, w.Body.String())
	}

	// DeleteArchive
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DeleteArchive", map[string]string{
		"ArchiveName": "my-archive",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteArchive: %d %s", w.Code, w.Body.String())
	}
}

// ---- Replay ----

func TestEB_Replay(t *testing.T) {
	handler := newEBGateway(t)

	// Create archive first
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "CreateArchive", map[string]any{
		"ArchiveName":    "replay-archive",
		"EventSourceArn": "arn:aws:events:us-east-1:000000000000:event-bus/default",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateArchive: %d %s", w.Code, w.Body.String())
	}

	// StartReplay
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "StartReplay", map[string]any{
		"ReplayName":         "my-replay",
		"EventSourceArn":     "arn:aws:events:us-east-1:000000000000:archive/replay-archive",
		"EventStartTime":     1700000000,
		"EventEndTime":       1700003600,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("StartReplay: %d %s", w.Code, w.Body.String())
	}

	// DescribeReplay
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeReplay", map[string]string{
		"ReplayName": "my-replay",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeReplay: %d %s", w.Code, w.Body.String())
	}
}
