package s3_test

import (
	"testing"

	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
)

func TestS3_DeleteObjectVersionRestoresMetadata(t *testing.T) {
	objects := s3svc.NewObjectStore()
	first := objects.PutObjectVersioned("file.txt", []byte("retained"), "text/plain", map[string]string{"owner": "original"})
	second := objects.PutObjectVersioned("file.txt", []byte("latest"), "application/json", map[string]string{"owner": "new"})

	objects.DeleteObjectVersion("file.txt", second.VersionId)
	current, err := objects.GetObject("file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if current.VersionId != first.VersionId || string(current.Body) != "retained" || current.ContentType != "text/plain" || current.Metadata["owner"] != "original" {
		t.Fatalf("restored object lost original version data: %+v", current)
	}
	retained, err := objects.GetObjectVersion("file.txt", first.VersionId)
	if err != nil || retained.Metadata["owner"] != "original" {
		t.Fatalf("version metadata: object=%+v err=%v", retained, err)
	}
}
