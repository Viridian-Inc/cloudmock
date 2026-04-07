package s3_test

import (
	"encoding/json"
	"strings"
	"testing"

	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
)

func TestS3_ExportState_Empty(t *testing.T) {
	svc := s3svc.New()

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Buckets []any `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Buckets) != 0 {
		t.Errorf("expected empty buckets, got %d", len(state.Buckets))
	}
}

func TestS3_ExportState_WithBucketAndObject(t *testing.T) {
	svc := s3svc.New()

	// Seed a bucket and object.
	seed := json.RawMessage(`{"buckets":[{"name":"my-bucket","objects":[{"key":"readme.md","body_base64":"aGVsbG8=","content_type":"text/markdown"}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Buckets []struct {
			Name    string `json:"name"`
			Objects []struct {
				Key         string `json:"key"`
				BodyBase64  string `json:"body_base64"`
				ContentType string `json:"content_type"`
			} `json:"objects"`
		} `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(state.Buckets))
	}
	if state.Buckets[0].Name != "my-bucket" {
		t.Errorf("expected bucket 'my-bucket', got %q", state.Buckets[0].Name)
	}
	if len(state.Buckets[0].Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(state.Buckets[0].Objects))
	}
	obj := state.Buckets[0].Objects[0]
	if obj.Key != "readme.md" {
		t.Errorf("expected key 'readme.md', got %q", obj.Key)
	}
	if obj.ContentType != "text/markdown" {
		t.Errorf("expected content-type 'text/markdown', got %q", obj.ContentType)
	}
	if obj.BodyBase64 == "" {
		t.Error("expected non-empty body_base64")
	}
}

func TestS3_ImportState_CreatesBuckets(t *testing.T) {
	svc := s3svc.New()

	data := json.RawMessage(`{"buckets":[{"name":"bucket-a","objects":[]},{"name":"bucket-b","objects":[]}]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	names := svc.GetBucketNames()
	for _, expected := range []string{"bucket-a", "bucket-b"} {
		found := false
		for _, n := range names {
			if n == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bucket %q not found after import, got: %v", expected, names)
		}
	}
}

func TestS3_ImportState_RestoresObjectBody(t *testing.T) {
	svc := s3svc.New()

	// "hello world" in base64 is "aGVsbG8gd29ybGQ="
	data := json.RawMessage(`{"buckets":[{"name":"data-bucket","objects":[{"key":"data.txt","body_base64":"aGVsbG8gd29ybGQ=","content_type":"text/plain"}]}]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	body, err := svc.GetObjectData("data-bucket", "data.txt")
	if err != nil {
		t.Fatalf("GetObjectData: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("expected body 'hello world', got %q", string(body))
	}
}

func TestS3_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := s3svc.New()

	if err := svc.ImportState(json.RawMessage(`{"buckets":[]}`)); err != nil {
		t.Fatalf("ImportState with empty buckets: %v", err)
	}
	if len(svc.GetBucketNames()) != 0 {
		t.Error("expected no buckets after importing empty state")
	}
}

func TestS3_RoundTrip_MultipleObjects(t *testing.T) {
	svc := s3svc.New()

	seed := json.RawMessage(`{"buckets":[{"name":"multi","objects":[
		{"key":"a.txt","body_base64":"YQ==","content_type":"text/plain"},
		{"key":"b.txt","body_base64":"Yg==","content_type":"text/plain"},
		{"key":"sub/c.txt","body_base64":"Yw==","content_type":"text/plain"}
	]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	// Import into fresh service.
	svc2 := s3svc.New()
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	for _, key := range []string{"a.txt", "b.txt", "sub/c.txt"} {
		if _, err := svc2.GetObjectData("multi", key); err != nil {
			t.Errorf("object %q not restored: %v", key, err)
		}
	}
}

func TestS3_ExportState_MultipleBuckets(t *testing.T) {
	svc := s3svc.New()

	seed := json.RawMessage(`{"buckets":[{"name":"alpha","objects":[]},{"name":"beta","objects":[]},{"name":"gamma","objects":[]}]}`)
	svc.ImportState(seed)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	export := string(raw)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(export, name) {
			t.Errorf("expected bucket %q in export", name)
		}
	}
}
