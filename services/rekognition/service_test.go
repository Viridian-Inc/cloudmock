package rekognition_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/rekognition"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body == nil {
		data = []byte("{}")
	} else {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "RekognitionService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/rekognition/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func mustOK(t *testing.T, w *httptest.ResponseRecorder, name string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: want 200, got %d: %s", name, w.Code, w.Body.String())
	}
}

func sampleImage() map[string]any {
	return map[string]any{
		"Bytes": "ZmFrZQ==",
	}
}

// ── Collection lifecycle ─────────────────────────────────────────────────────

func TestCollectionLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create
	w := doCall(t, h, "CreateCollection", map[string]any{
		"CollectionId": "my-collection",
		"Tags":         map[string]string{"env": "test"},
	})
	mustOK(t, w, "CreateCollection")
	var created struct {
		CollectionArn    string `json:"CollectionArn"`
		FaceModelVersion string `json:"FaceModelVersion"`
		StatusCode       int    `json:"StatusCode"`
	}
	decode(t, w, &created)
	if created.CollectionArn == "" {
		t.Fatalf("CreateCollection: missing CollectionArn: %s", w.Body.String())
	}
	if created.FaceModelVersion == "" {
		t.Fatalf("CreateCollection: missing FaceModelVersion")
	}

	// Duplicate fails
	if w := doCall(t, h, "CreateCollection", map[string]any{"CollectionId": "my-collection"}); w.Code == http.StatusOK {
		t.Fatalf("CreateCollection: duplicate should fail, got 200")
	}

	// Describe
	w = doCall(t, h, "DescribeCollection", map[string]any{"CollectionId": "my-collection"})
	mustOK(t, w, "DescribeCollection")
	var desc struct {
		CollectionARN    string `json:"CollectionARN"`
		FaceCount        int    `json:"FaceCount"`
		FaceModelVersion string `json:"FaceModelVersion"`
	}
	decode(t, w, &desc)
	if desc.CollectionARN == "" {
		t.Fatalf("DescribeCollection: missing ARN: %s", w.Body.String())
	}

	// List
	w = doCall(t, h, "ListCollections", nil)
	mustOK(t, w, "ListCollections")
	var listed struct {
		CollectionIds []string `json:"CollectionIds"`
	}
	decode(t, w, &listed)
	if len(listed.CollectionIds) != 1 || listed.CollectionIds[0] != "my-collection" {
		t.Fatalf("ListCollections: unexpected: %+v", listed)
	}

	// Delete
	mustOK(t, doCall(t, h, "DeleteCollection", map[string]any{"CollectionId": "my-collection"}), "DeleteCollection")

	// Describe missing should fail
	if w := doCall(t, h, "DescribeCollection", map[string]any{"CollectionId": "my-collection"}); w.Code == http.StatusOK {
		t.Fatalf("DescribeCollection after delete: should fail, got 200")
	}
}

// ── Face lifecycle ───────────────────────────────────────────────────────────

func TestFaceLifecycle(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "CreateCollection", map[string]any{"CollectionId": "faces"}), "CreateCollection")

	// Index a face
	w := doCall(t, h, "IndexFaces", map[string]any{
		"CollectionId":    "faces",
		"Image":           sampleImage(),
		"ExternalImageId": "person-1",
	})
	mustOK(t, w, "IndexFaces")
	var indexed struct {
		FaceRecords []struct {
			Face struct {
				FaceId          string `json:"FaceId"`
				ExternalImageId string `json:"ExternalImageId"`
			} `json:"Face"`
			FaceDetail map[string]any `json:"FaceDetail"`
		} `json:"FaceRecords"`
	}
	decode(t, w, &indexed)
	if len(indexed.FaceRecords) != 1 {
		t.Fatalf("IndexFaces: expected 1 record, got %d: %s", len(indexed.FaceRecords), w.Body.String())
	}
	faceID := indexed.FaceRecords[0].Face.FaceId
	if faceID == "" {
		t.Fatalf("IndexFaces: missing FaceId")
	}
	if indexed.FaceRecords[0].FaceDetail == nil {
		t.Fatalf("IndexFaces: missing FaceDetail")
	}

	// List the face
	w = doCall(t, h, "ListFaces", map[string]any{"CollectionId": "faces"})
	mustOK(t, w, "ListFaces")
	var listed struct {
		Faces []struct {
			FaceId string `json:"FaceId"`
		} `json:"Faces"`
	}
	decode(t, w, &listed)
	if len(listed.Faces) != 1 || listed.Faces[0].FaceId != faceID {
		t.Fatalf("ListFaces: unexpected: %+v", listed)
	}

	// Search by face id
	w = doCall(t, h, "SearchFaces", map[string]any{
		"CollectionId": "faces",
		"FaceId":       faceID,
	})
	mustOK(t, w, "SearchFaces")

	// Search by image
	w = doCall(t, h, "SearchFacesByImage", map[string]any{
		"CollectionId": "faces",
		"Image":        sampleImage(),
	})
	mustOK(t, w, "SearchFacesByImage")

	// Compare faces
	w = doCall(t, h, "CompareFaces", map[string]any{
		"SourceImage": sampleImage(),
		"TargetImage": sampleImage(),
	})
	mustOK(t, w, "CompareFaces")

	// DetectFaces (sync)
	mustOK(t, doCall(t, h, "DetectFaces", map[string]any{"Image": sampleImage()}), "DetectFaces")

	// Delete face
	w = doCall(t, h, "DeleteFaces", map[string]any{
		"CollectionId": "faces",
		"FaceIds":      []string{faceID},
	})
	mustOK(t, w, "DeleteFaces")
	var deleted struct {
		DeletedFaces []string `json:"DeletedFaces"`
	}
	decode(t, w, &deleted)
	if len(deleted.DeletedFaces) != 1 {
		t.Fatalf("DeleteFaces: expected 1, got %d", len(deleted.DeletedFaces))
	}

	// List should now be empty
	w = doCall(t, h, "ListFaces", map[string]any{"CollectionId": "faces"})
	decode(t, w, &listed)
	if len(listed.Faces) != 0 {
		t.Fatalf("ListFaces after delete: expected 0, got %d", len(listed.Faces))
	}
}

// ── User lifecycle (with face association) ──────────────────────────────────

func TestUserLifecycle(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "CreateCollection", map[string]any{"CollectionId": "users"}), "CreateCollection")

	// Create user
	mustOK(t, doCall(t, h, "CreateUser", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
	}), "CreateUser")

	// Duplicate user fails
	if w := doCall(t, h, "CreateUser", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
	}); w.Code == http.StatusOK {
		t.Fatalf("CreateUser duplicate should fail")
	}

	// List users
	w := doCall(t, h, "ListUsers", map[string]any{"CollectionId": "users"})
	mustOK(t, w, "ListUsers")
	var listed struct {
		Users []struct {
			UserId     string `json:"UserId"`
			UserStatus string `json:"UserStatus"`
		} `json:"Users"`
	}
	decode(t, w, &listed)
	if len(listed.Users) != 1 || listed.Users[0].UserId != "alice" {
		t.Fatalf("ListUsers: unexpected: %+v", listed)
	}

	// Index a face to associate
	w = doCall(t, h, "IndexFaces", map[string]any{
		"CollectionId": "users",
		"Image":        sampleImage(),
	})
	mustOK(t, w, "IndexFaces")
	var indexed struct {
		FaceRecords []struct {
			Face struct {
				FaceId string `json:"FaceId"`
			} `json:"Face"`
		} `json:"FaceRecords"`
	}
	decode(t, w, &indexed)
	faceID := indexed.FaceRecords[0].Face.FaceId

	// AssociateFaces
	w = doCall(t, h, "AssociateFaces", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
		"FaceIds":      []string{faceID},
	})
	mustOK(t, w, "AssociateFaces")
	var assoc struct {
		AssociatedFaces []map[string]any `json:"AssociatedFaces"`
		UserStatus      string           `json:"UserStatus"`
	}
	decode(t, w, &assoc)
	if len(assoc.AssociatedFaces) != 1 {
		t.Fatalf("AssociateFaces: expected 1, got %d", len(assoc.AssociatedFaces))
	}
	if assoc.UserStatus != "ACTIVE" {
		t.Fatalf("AssociateFaces: expected UserStatus ACTIVE, got %q", assoc.UserStatus)
	}

	// SearchUsers / SearchUsersByImage
	mustOK(t, doCall(t, h, "SearchUsers", map[string]any{
		"CollectionId": "users",
		"FaceId":       faceID,
	}), "SearchUsers")
	mustOK(t, doCall(t, h, "SearchUsersByImage", map[string]any{
		"CollectionId": "users",
		"Image":        sampleImage(),
	}), "SearchUsersByImage")

	// DisassociateFaces
	mustOK(t, doCall(t, h, "DisassociateFaces", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
		"FaceIds":      []string{faceID},
	}), "DisassociateFaces")

	// Delete user
	mustOK(t, doCall(t, h, "DeleteUser", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
	}), "DeleteUser")

	// Should now be missing
	if w := doCall(t, h, "DeleteUser", map[string]any{
		"CollectionId": "users",
		"UserId":       "alice",
	}); w.Code == http.StatusOK {
		t.Fatalf("DeleteUser missing should fail")
	}
}

// ── Project lifecycle ────────────────────────────────────────────────────────

func TestProjectLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create project
	w := doCall(t, h, "CreateProject", map[string]any{
		"ProjectName": "my-project",
		"Feature":     "CUSTOM_LABELS",
	})
	mustOK(t, w, "CreateProject")
	var created struct {
		ProjectArn string `json:"ProjectArn"`
	}
	decode(t, w, &created)
	if created.ProjectArn == "" {
		t.Fatalf("CreateProject: missing ProjectArn")
	}
	projectArn := created.ProjectArn

	// Duplicate fails
	if w := doCall(t, h, "CreateProject", map[string]any{
		"ProjectName": "my-project",
	}); w.Code == http.StatusOK {
		t.Fatalf("CreateProject duplicate should fail")
	}

	// Describe
	w = doCall(t, h, "DescribeProjects", nil)
	mustOK(t, w, "DescribeProjects")
	var listed struct {
		ProjectDescriptions []struct {
			ProjectArn  string `json:"ProjectArn"`
			ProjectName string `json:"ProjectName"`
		} `json:"ProjectDescriptions"`
	}
	decode(t, w, &listed)
	if len(listed.ProjectDescriptions) != 1 {
		t.Fatalf("DescribeProjects: expected 1, got %d: %s", len(listed.ProjectDescriptions), w.Body.String())
	}

	// Create a project version
	w = doCall(t, h, "CreateProjectVersion", map[string]any{
		"ProjectArn":  projectArn,
		"VersionName": "v1",
	})
	mustOK(t, w, "CreateProjectVersion")
	var version struct {
		ProjectVersionArn string `json:"ProjectVersionArn"`
	}
	decode(t, w, &version)
	if version.ProjectVersionArn == "" {
		t.Fatalf("CreateProjectVersion: missing arn")
	}
	versionArn := version.ProjectVersionArn

	// DescribeProjectVersions
	w = doCall(t, h, "DescribeProjectVersions", map[string]any{
		"ProjectArn": projectArn,
	})
	mustOK(t, w, "DescribeProjectVersions")
	var versions struct {
		ProjectVersionDescriptions []struct {
			ProjectVersionArn string `json:"ProjectVersionArn"`
			Status            string `json:"Status"`
		} `json:"ProjectVersionDescriptions"`
	}
	decode(t, w, &versions)
	if len(versions.ProjectVersionDescriptions) != 1 {
		t.Fatalf("DescribeProjectVersions: expected 1, got %d", len(versions.ProjectVersionDescriptions))
	}

	// Start version
	w = doCall(t, h, "StartProjectVersion", map[string]any{
		"ProjectVersionArn": versionArn,
		"MinInferenceUnits": 1,
	})
	mustOK(t, w, "StartProjectVersion")
	var started struct {
		Status string `json:"Status"`
	}
	decode(t, w, &started)
	if started.Status != "RUNNING" {
		t.Fatalf("StartProjectVersion: expected RUNNING, got %q", started.Status)
	}

	// DetectCustomLabels
	mustOK(t, doCall(t, h, "DetectCustomLabels", map[string]any{
		"ProjectVersionArn": versionArn,
		"Image":             sampleImage(),
	}), "DetectCustomLabels")

	// Stop version
	mustOK(t, doCall(t, h, "StopProjectVersion", map[string]any{
		"ProjectVersionArn": versionArn,
	}), "StopProjectVersion")

	// Project policy lifecycle
	w = doCall(t, h, "PutProjectPolicy", map[string]any{
		"ProjectArn":     projectArn,
		"PolicyName":     "policy1",
		"PolicyDocument": `{"Version":"2012-10-17","Statement":[]}`,
	})
	mustOK(t, w, "PutProjectPolicy")

	w = doCall(t, h, "ListProjectPolicies", map[string]any{"ProjectArn": projectArn})
	mustOK(t, w, "ListProjectPolicies")
	var policies struct {
		ProjectPolicies []map[string]any `json:"ProjectPolicies"`
	}
	decode(t, w, &policies)
	if len(policies.ProjectPolicies) != 1 {
		t.Fatalf("ListProjectPolicies: expected 1, got %d", len(policies.ProjectPolicies))
	}

	mustOK(t, doCall(t, h, "DeleteProjectPolicy", map[string]any{
		"ProjectArn": projectArn,
		"PolicyName": "policy1",
	}), "DeleteProjectPolicy")

	// Delete version then project
	mustOK(t, doCall(t, h, "DeleteProjectVersion", map[string]any{
		"ProjectVersionArn": versionArn,
	}), "DeleteProjectVersion")
	mustOK(t, doCall(t, h, "DeleteProject", map[string]any{
		"ProjectArn": projectArn,
	}), "DeleteProject")
}

// ── Dataset lifecycle ────────────────────────────────────────────────────────

func TestDatasetLifecycle(t *testing.T) {
	h := newGateway(t)

	// Need a project first
	w := doCall(t, h, "CreateProject", map[string]any{"ProjectName": "dataset-project"})
	mustOK(t, w, "CreateProject")
	var created struct {
		ProjectArn string `json:"ProjectArn"`
	}
	decode(t, w, &created)

	// Create dataset
	w = doCall(t, h, "CreateDataset", map[string]any{
		"ProjectArn":  created.ProjectArn,
		"DatasetType": "TRAIN",
	})
	mustOK(t, w, "CreateDataset")
	var dataset struct {
		DatasetArn string `json:"DatasetArn"`
	}
	decode(t, w, &dataset)
	if dataset.DatasetArn == "" {
		t.Fatalf("CreateDataset: missing arn")
	}

	// Bad type fails
	if w := doCall(t, h, "CreateDataset", map[string]any{
		"ProjectArn":  created.ProjectArn,
		"DatasetType": "BOGUS",
	}); w.Code == http.StatusOK {
		t.Fatalf("CreateDataset with bad type should fail")
	}

	// Describe
	mustOK(t, doCall(t, h, "DescribeDataset", map[string]any{"DatasetArn": dataset.DatasetArn}), "DescribeDataset")

	// Update entries
	mustOK(t, doCall(t, h, "UpdateDatasetEntries", map[string]any{
		"DatasetArn": dataset.DatasetArn,
		"Changes": map[string]any{
			"GroundTruth": "ZmFrZQ==",
		},
	}), "UpdateDatasetEntries")

	// Distribute
	mustOK(t, doCall(t, h, "DistributeDatasetEntries", map[string]any{
		"Datasets": []map[string]any{{"Arn": dataset.DatasetArn}},
	}), "DistributeDatasetEntries")

	// List entries / labels
	mustOK(t, doCall(t, h, "ListDatasetEntries", map[string]any{"DatasetArn": dataset.DatasetArn}), "ListDatasetEntries")
	mustOK(t, doCall(t, h, "ListDatasetLabels", map[string]any{"DatasetArn": dataset.DatasetArn}), "ListDatasetLabels")

	// Delete
	mustOK(t, doCall(t, h, "DeleteDataset", map[string]any{"DatasetArn": dataset.DatasetArn}), "DeleteDataset")
	if w := doCall(t, h, "DescribeDataset", map[string]any{"DatasetArn": dataset.DatasetArn}); w.Code == http.StatusOK {
		t.Fatalf("DescribeDataset after delete should fail")
	}
}

// ── Stream processor lifecycle ───────────────────────────────────────────────

func TestStreamProcessorLifecycle(t *testing.T) {
	h := newGateway(t)

	// Missing required fields should fail
	if w := doCall(t, h, "CreateStreamProcessor", map[string]any{
		"Name": "sp1",
	}); w.Code == http.StatusOK {
		t.Fatalf("CreateStreamProcessor missing fields should fail")
	}

	createBody := map[string]any{
		"Name": "sp1",
		"Input": map[string]any{
			"KinesisVideoStream": map[string]any{
				"Arn": "arn:aws:kinesisvideo:us-east-1:000000000000:stream/test/1",
			},
		},
		"Output": map[string]any{
			"KinesisDataStream": map[string]any{
				"Arn": "arn:aws:kinesis:us-east-1:000000000000:stream/test",
			},
		},
		"RoleArn": "arn:aws:iam::000000000000:role/RekSP",
		"Settings": map[string]any{
			"FaceSearch": map[string]any{
				"CollectionId":        "faces",
				"FaceMatchThreshold":  90.0,
			},
		},
	}

	// Create
	w := doCall(t, h, "CreateStreamProcessor", createBody)
	mustOK(t, w, "CreateStreamProcessor")
	var created struct {
		StreamProcessorArn string `json:"StreamProcessorArn"`
	}
	decode(t, w, &created)
	if created.StreamProcessorArn == "" {
		t.Fatalf("CreateStreamProcessor: missing arn")
	}

	// Duplicate
	if w := doCall(t, h, "CreateStreamProcessor", createBody); w.Code == http.StatusOK {
		t.Fatalf("CreateStreamProcessor duplicate should fail")
	}

	// Describe
	w = doCall(t, h, "DescribeStreamProcessor", map[string]any{"Name": "sp1"})
	mustOK(t, w, "DescribeStreamProcessor")
	var desc struct {
		Name   string `json:"Name"`
		Status string `json:"Status"`
	}
	decode(t, w, &desc)
	if desc.Status != "STOPPED" {
		t.Fatalf("DescribeStreamProcessor: expected STOPPED, got %q", desc.Status)
	}

	// List
	w = doCall(t, h, "ListStreamProcessors", nil)
	mustOK(t, w, "ListStreamProcessors")
	var listed struct {
		StreamProcessors []struct {
			Name string `json:"Name"`
		} `json:"StreamProcessors"`
	}
	decode(t, w, &listed)
	if len(listed.StreamProcessors) != 1 {
		t.Fatalf("ListStreamProcessors: expected 1, got %d", len(listed.StreamProcessors))
	}

	// Start
	mustOK(t, doCall(t, h, "StartStreamProcessor", map[string]any{"Name": "sp1"}), "StartStreamProcessor")

	// Cannot delete while running
	if w := doCall(t, h, "DeleteStreamProcessor", map[string]any{"Name": "sp1"}); w.Code == http.StatusOK {
		t.Fatalf("DeleteStreamProcessor while running should fail")
	}

	// Update
	mustOK(t, doCall(t, h, "UpdateStreamProcessor", map[string]any{
		"Name": "sp1",
		"SettingsForUpdate": map[string]any{
			"ConnectedHomeForUpdate": map[string]any{
				"Labels": []string{"PERSON"},
			},
		},
	}), "UpdateStreamProcessor")

	// Stop and delete
	mustOK(t, doCall(t, h, "StopStreamProcessor", map[string]any{"Name": "sp1"}), "StopStreamProcessor")
	mustOK(t, doCall(t, h, "DeleteStreamProcessor", map[string]any{"Name": "sp1"}), "DeleteStreamProcessor")
}

// ── Async video jobs ─────────────────────────────────────────────────────────

func TestAsyncVideoJobs(t *testing.T) {
	h := newGateway(t)

	video := map[string]any{
		"S3Object": map[string]any{
			"Bucket": "vids",
			"Name":   "movie.mp4",
		},
	}

	// helper to start one job and get its result
	pairs := []struct {
		start, get string
	}{
		{"StartLabelDetection", "GetLabelDetection"},
		{"StartTextDetection", "GetTextDetection"},
		{"StartContentModeration", "GetContentModeration"},
		{"StartFaceDetection", "GetFaceDetection"},
		{"StartFaceSearch", "GetFaceSearch"},
		{"StartPersonTracking", "GetPersonTracking"},
		{"StartCelebrityRecognition", "GetCelebrityRecognition"},
		{"StartSegmentDetection", "GetSegmentDetection"},
	}

	for _, p := range pairs {
		w := doCall(t, h, p.start, map[string]any{"Video": video})
		mustOK(t, w, p.start)
		var started struct {
			JobId string `json:"JobId"`
		}
		decode(t, w, &started)
		if started.JobId == "" {
			t.Fatalf("%s: missing JobId", p.start)
		}

		w = doCall(t, h, p.get, map[string]any{"JobId": started.JobId})
		mustOK(t, w, p.get)
		var got struct {
			JobStatus string `json:"JobStatus"`
		}
		decode(t, w, &got)
		if got.JobStatus != "SUCCEEDED" {
			t.Fatalf("%s: expected SUCCEEDED, got %q", p.get, got.JobStatus)
		}
	}

	// Missing video should fail
	if w := doCall(t, h, "StartLabelDetection", map[string]any{}); w.Code == http.StatusOK {
		t.Fatalf("StartLabelDetection without Video should fail")
	}

	// Missing JobId on get should fail
	if w := doCall(t, h, "GetLabelDetection", map[string]any{}); w.Code == http.StatusOK {
		t.Fatalf("GetLabelDetection without JobId should fail")
	}
}

// ── Media analysis jobs ──────────────────────────────────────────────────────

func TestMediaAnalysisJob(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "StartMediaAnalysisJob", map[string]any{
		"JobName": "ma1",
		"OperationsConfig": map[string]any{
			"DetectModerationLabels": map[string]any{
				"MinConfidence": 50.0,
			},
		},
		"Input": map[string]any{
			"S3Object": map[string]any{"Bucket": "in", "Name": "img.jpg"},
		},
		"OutputConfig": map[string]any{
			"S3Bucket": "out",
		},
	})
	mustOK(t, w, "StartMediaAnalysisJob")
	var started struct {
		JobId string `json:"JobId"`
	}
	decode(t, w, &started)
	if started.JobId == "" {
		t.Fatalf("StartMediaAnalysisJob: missing JobId")
	}

	w = doCall(t, h, "GetMediaAnalysisJob", map[string]any{"JobId": started.JobId})
	mustOK(t, w, "GetMediaAnalysisJob")
	var got struct {
		JobId  string `json:"JobId"`
		Status string `json:"Status"`
	}
	decode(t, w, &got)
	if got.Status != "SUCCEEDED" {
		t.Fatalf("GetMediaAnalysisJob: expected SUCCEEDED, got %q", got.Status)
	}

	w = doCall(t, h, "ListMediaAnalysisJobs", nil)
	mustOK(t, w, "ListMediaAnalysisJobs")
	var listed struct {
		MediaAnalysisJobs []map[string]any `json:"MediaAnalysisJobs"`
	}
	decode(t, w, &listed)
	if len(listed.MediaAnalysisJobs) != 1 {
		t.Fatalf("ListMediaAnalysisJobs: expected 1, got %d", len(listed.MediaAnalysisJobs))
	}

	// Missing input fails
	if w := doCall(t, h, "StartMediaAnalysisJob", map[string]any{}); w.Code == http.StatusOK {
		t.Fatalf("StartMediaAnalysisJob with empty body should fail")
	}
}

// ── Face liveness ────────────────────────────────────────────────────────────

func TestFaceLiveness(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateFaceLivenessSession", map[string]any{})
	mustOK(t, w, "CreateFaceLivenessSession")
	var created struct {
		SessionId string `json:"SessionId"`
	}
	decode(t, w, &created)
	if created.SessionId == "" {
		t.Fatalf("CreateFaceLivenessSession: missing SessionId")
	}

	w = doCall(t, h, "GetFaceLivenessSessionResults", map[string]any{
		"SessionId": created.SessionId,
	})
	mustOK(t, w, "GetFaceLivenessSessionResults")
	var got struct {
		SessionId string  `json:"SessionId"`
		Status    string  `json:"Status"`
		Confidence float64 `json:"Confidence"`
	}
	decode(t, w, &got)
	if got.SessionId != created.SessionId {
		t.Fatalf("GetFaceLivenessSessionResults: SessionId mismatch")
	}
	if got.Status != "SUCCEEDED" {
		t.Fatalf("GetFaceLivenessSessionResults: expected SUCCEEDED, got %q", got.Status)
	}

	if w := doCall(t, h, "GetFaceLivenessSessionResults", map[string]any{
		"SessionId": "missing",
	}); w.Code == http.StatusOK {
		t.Fatalf("GetFaceLivenessSessionResults missing should fail")
	}
}

// ── Sync detection ops ───────────────────────────────────────────────────────

func TestSyncDetectionOps(t *testing.T) {
	h := newGateway(t)
	body := map[string]any{"Image": sampleImage()}

	mustOK(t, doCall(t, h, "DetectLabels", body), "DetectLabels")
	mustOK(t, doCall(t, h, "DetectModerationLabels", body), "DetectModerationLabels")
	mustOK(t, doCall(t, h, "DetectText", body), "DetectText")
	mustOK(t, doCall(t, h, "DetectProtectiveEquipment", body), "DetectProtectiveEquipment")
	mustOK(t, doCall(t, h, "RecognizeCelebrities", body), "RecognizeCelebrities")
	mustOK(t, doCall(t, h, "GetCelebrityInfo", map[string]any{"Id": "celeb-1"}), "GetCelebrityInfo")
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func TestTagging(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateCollection", map[string]any{
		"CollectionId": "tagged",
		"Tags":         map[string]string{"env": "prod"},
	})
	mustOK(t, w, "CreateCollection")
	var created struct {
		CollectionArn string `json:"CollectionArn"`
	}
	decode(t, w, &created)

	// Add more tags
	mustOK(t, doCall(t, h, "TagResource", map[string]any{
		"ResourceArn": created.CollectionArn,
		"Tags":        map[string]string{"team": "ml"},
	}), "TagResource")

	// List tags
	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": created.CollectionArn})
	mustOK(t, w, "ListTagsForResource")
	var tagsResp struct {
		Tags map[string]string `json:"Tags"`
	}
	decode(t, w, &tagsResp)
	if tagsResp.Tags["env"] != "prod" || tagsResp.Tags["team"] != "ml" {
		t.Fatalf("ListTagsForResource: unexpected: %+v", tagsResp.Tags)
	}

	// Untag
	mustOK(t, doCall(t, h, "UntagResource", map[string]any{
		"ResourceArn": created.CollectionArn,
		"TagKeys":     []string{"env"},
	}), "UntagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": created.CollectionArn})
	tagsResp.Tags = nil
	decode(t, w, &tagsResp)
	if _, ok := tagsResp.Tags["env"]; ok {
		t.Fatalf("UntagResource: env still present: %+v", tagsResp.Tags)
	}
	if tagsResp.Tags["team"] != "ml" {
		t.Fatalf("UntagResource: team should be preserved: %+v", tagsResp.Tags)
	}
}

// ── Validation guards ────────────────────────────────────────────────────────

func TestValidationErrors(t *testing.T) {
	h := newGateway(t)

	cases := []struct {
		action string
		body   map[string]any
	}{
		{"CreateCollection", map[string]any{}},
		{"DeleteCollection", map[string]any{}},
		{"DescribeCollection", map[string]any{}},
		{"IndexFaces", map[string]any{}},
		{"ListFaces", map[string]any{}},
		{"DeleteFaces", map[string]any{}},
		{"SearchFaces", map[string]any{}},
		{"AssociateFaces", map[string]any{}},
		{"DisassociateFaces", map[string]any{}},
		{"CreateUser", map[string]any{}},
		{"DeleteUser", map[string]any{}},
		{"CreateProject", map[string]any{}},
		{"CreateProjectVersion", map[string]any{}},
		{"DeleteProject", map[string]any{}},
		{"CreateDataset", map[string]any{}},
		{"DescribeDataset", map[string]any{}},
		{"DeleteDataset", map[string]any{}},
		{"CreateStreamProcessor", map[string]any{}},
		{"DescribeStreamProcessor", map[string]any{}},
		{"GetCelebrityInfo", map[string]any{}},
		{"TagResource", map[string]any{}},
		{"ListTagsForResource", map[string]any{}},
	}

	for _, c := range cases {
		w := doCall(t, h, c.action, c.body)
		if w.Code == http.StatusOK {
			t.Errorf("%s: expected validation error, got 200: %s", c.action, w.Body.String())
		}
	}
}

// ── Reset ────────────────────────────────────────────────────────────────────

func TestStoreReset(t *testing.T) {
	store := svc.NewStore("000000000000", "us-east-1")
	if _, err := store.CreateCollection("c1", nil); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if _, err := store.CreateProject("p1", "CUSTOM_LABELS", "DISABLED", nil); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	store.Reset()
	if _, err := store.GetCollection("c1"); err == nil {
		t.Fatalf("Reset: collection should be gone")
	}
	if got := len(store.ListCollections()); got != 0 {
		t.Fatalf("Reset: expected 0 collections, got %d", got)
	}
	if got := len(store.ListProjects(nil, nil)); got != 0 {
		t.Fatalf("Reset: expected 0 projects, got %d", got)
	}
}
