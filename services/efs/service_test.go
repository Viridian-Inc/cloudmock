package efs_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/efs"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "elasticfilesystem."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/elasticfilesystem/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	return w
}

func mustOK(t *testing.T, w *httptest.ResponseRecorder, label string) {
	t.Helper()
	if w.Code < 200 || w.Code >= 300 {
		t.Fatalf("%s: expected 2xx, got %d: %s", label, w.Code, w.Body.String())
	}
}

func createFileSystem(t *testing.T, h http.Handler, token string) string {
	t.Helper()
	w := doCall(t, h, "CreateFileSystem", map[string]any{
		"CreationToken":   token,
		"PerformanceMode": "generalPurpose",
		"Encrypted":       true,
		"Tags": []map[string]any{
			{"Key": "Name", "Value": "test-fs"},
			{"Key": "env", "Value": "dev"},
		},
	})
	mustOK(t, w, "CreateFileSystem")
	var out struct {
		FileSystemID    string `json:"FileSystemId"`
		FileSystemArn   string `json:"FileSystemArn"`
		LifeCycleState  string `json:"LifeCycleState"`
		PerformanceMode string `json:"PerformanceMode"`
		Encrypted       bool   `json:"Encrypted"`
		OwnerID         string `json:"OwnerId"`
	}
	decode(t, w, &out)
	if out.FileSystemID == "" {
		t.Fatalf("expected FileSystemId, got empty: %s", w.Body.String())
	}
	if out.LifeCycleState != "available" {
		t.Fatalf("expected LifeCycleState=available, got %q", out.LifeCycleState)
	}
	if !out.Encrypted {
		t.Fatalf("expected Encrypted=true")
	}
	if out.OwnerID == "" {
		t.Fatalf("expected OwnerId to be set")
	}
	return out.FileSystemID
}

// ── File system lifecycle ───────────────────────────────────────────────────

func TestFileSystemLifecycle(t *testing.T) {
	h := newGateway(t)

	id := createFileSystem(t, h, "tok-1")

	// Describe should find it.
	w := doCall(t, h, "DescribeFileSystems", map[string]any{"FileSystemId": id})
	mustOK(t, w, "DescribeFileSystems")
	var listed struct {
		FileSystems []struct {
			FileSystemID    string `json:"FileSystemId"`
			LifeCycleState  string `json:"LifeCycleState"`
			PerformanceMode string `json:"PerformanceMode"`
			Tags            []struct {
				Key, Value string
			}
		}
	}
	decode(t, w, &listed)
	if len(listed.FileSystems) != 1 {
		t.Fatalf("expected 1 file system, got %d: %s", len(listed.FileSystems), w.Body.String())
	}
	if listed.FileSystems[0].FileSystemID != id {
		t.Fatalf("unexpected file system id: %q", listed.FileSystems[0].FileSystemID)
	}
	if len(listed.FileSystems[0].Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(listed.FileSystems[0].Tags))
	}

	// Update throughput.
	w = doCall(t, h, "UpdateFileSystem", map[string]any{
		"FileSystemId":                 id,
		"ThroughputMode":               "provisioned",
		"ProvisionedThroughputInMibps": 64.0,
	})
	mustOK(t, w, "UpdateFileSystem")
	var updated struct {
		ThroughputMode               string  `json:"ThroughputMode"`
		ProvisionedThroughputInMibps float64 `json:"ProvisionedThroughputInMibps"`
	}
	decode(t, w, &updated)
	if updated.ThroughputMode != "provisioned" {
		t.Fatalf("expected ThroughputMode=provisioned, got %q", updated.ThroughputMode)
	}
	if updated.ProvisionedThroughputInMibps != 64 {
		t.Fatalf("expected provisioned throughput=64, got %v", updated.ProvisionedThroughputInMibps)
	}

	// Update protection.
	w = doCall(t, h, "UpdateFileSystemProtection", map[string]any{
		"FileSystemId":                   id,
		"ReplicationOverwriteProtection": "DISABLED",
	})
	mustOK(t, w, "UpdateFileSystemProtection")
	var prot struct {
		ReplicationOverwriteProtection string
	}
	decode(t, w, &prot)
	if prot.ReplicationOverwriteProtection != "DISABLED" {
		t.Fatalf("expected DISABLED, got %q", prot.ReplicationOverwriteProtection)
	}

	// Delete.
	w = doCall(t, h, "DeleteFileSystem", map[string]any{"FileSystemId": id})
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteFileSystem: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Now empty.
	w = doCall(t, h, "DescribeFileSystems", nil)
	mustOK(t, w, "DescribeFileSystems after delete")
	decode(t, w, &listed)
	if len(listed.FileSystems) != 0 {
		t.Fatalf("expected 0 file systems after delete, got %d", len(listed.FileSystems))
	}
}

func TestCreateFileSystemDuplicateToken(t *testing.T) {
	h := newGateway(t)
	createFileSystem(t, h, "dup-token")
	w := doCall(t, h, "CreateFileSystem", map[string]any{"CreationToken": "dup-token"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFileSystemNotFound(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "DeleteFileSystem", map[string]any{"FileSystemId": "fs-deadbeef"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateMountTargetRequiresFields(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateMountTarget", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	w = doCall(t, h, "CreateMountTarget", map[string]any{"FileSystemId": "fs-x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on missing SubnetId, got %d", w.Code)
	}
	w = doCall(t, h, "CreateMountTarget", map[string]any{"FileSystemId": "fs-missing", "SubnetId": "subnet-1"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on missing fs, got %d", w.Code)
	}
}

// ── Mount target lifecycle ──────────────────────────────────────────────────

func TestMountTargetLifecycle(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "mt-tok")

	w := doCall(t, h, "CreateMountTarget", map[string]any{
		"FileSystemId":   fsID,
		"SubnetId":       "subnet-abc",
		"SecurityGroups": []string{"sg-1"},
	})
	mustOK(t, w, "CreateMountTarget")
	var mt struct {
		MountTargetID  string `json:"MountTargetId"`
		FileSystemID   string `json:"FileSystemId"`
		SubnetID       string `json:"SubnetId"`
		LifeCycleState string `json:"LifeCycleState"`
		IPAddress      string `json:"IpAddress"`
	}
	decode(t, w, &mt)
	if mt.MountTargetID == "" || mt.FileSystemID != fsID || mt.LifeCycleState != "available" {
		t.Fatalf("unexpected mount target: %+v", mt)
	}

	// File system should now report 1 mount target.
	w = doCall(t, h, "DescribeFileSystems", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeFileSystems")
	var listed struct {
		FileSystems []struct {
			NumberOfMountTargets int
		}
	}
	decode(t, w, &listed)
	if len(listed.FileSystems) != 1 || listed.FileSystems[0].NumberOfMountTargets != 1 {
		t.Fatalf("expected 1 mount target reported, got %+v", listed.FileSystems)
	}

	// Describe mount targets by file system.
	w = doCall(t, h, "DescribeMountTargets", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeMountTargets")
	var dmt struct {
		MountTargets []struct {
			MountTargetID string `json:"MountTargetId"`
		}
	}
	decode(t, w, &dmt)
	if len(dmt.MountTargets) != 1 || dmt.MountTargets[0].MountTargetID != mt.MountTargetID {
		t.Fatalf("unexpected list: %+v", dmt)
	}

	// Modify SGs.
	w = doCall(t, h, "ModifyMountTargetSecurityGroups", map[string]any{
		"MountTargetId":  mt.MountTargetID,
		"SecurityGroups": []string{"sg-2", "sg-3"},
	})
	mustOK(t, w, "ModifyMountTargetSecurityGroups")

	w = doCall(t, h, "DescribeMountTargetSecurityGroups", map[string]any{
		"MountTargetId": mt.MountTargetID,
	})
	mustOK(t, w, "DescribeMountTargetSecurityGroups")
	var dsg struct {
		SecurityGroups []string
	}
	decode(t, w, &dsg)
	if len(dsg.SecurityGroups) != 2 || dsg.SecurityGroups[0] != "sg-2" {
		t.Fatalf("unexpected security groups: %+v", dsg.SecurityGroups)
	}

	// Cannot delete file system while mount targets exist.
	w = doCall(t, h, "DeleteFileSystem", map[string]any{"FileSystemId": fsID})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 deleting fs with mount target, got %d", w.Code)
	}

	// Delete mount target then file system.
	w = doCall(t, h, "DeleteMountTarget", map[string]any{"MountTargetId": mt.MountTargetID})
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteMountTarget: expected 204, got %d", w.Code)
	}
	w = doCall(t, h, "DeleteFileSystem", map[string]any{"FileSystemId": fsID})
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteFileSystem after mt removal: expected 204, got %d", w.Code)
	}
}

// ── Access point lifecycle ──────────────────────────────────────────────────

func TestAccessPointLifecycle(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "ap-tok")

	w := doCall(t, h, "CreateAccessPoint", map[string]any{
		"FileSystemId": fsID,
		"ClientToken":  "ap-client-1",
		"Tags": []map[string]any{
			{"Key": "Name", "Value": "ap-1"},
		},
		"PosixUser": map[string]any{
			"Uid":           1000,
			"Gid":           1000,
			"SecondaryGids": []int{2000},
		},
		"RootDirectory": map[string]any{"Path": "/data"},
	})
	mustOK(t, w, "CreateAccessPoint")
	var created struct {
		AccessPointID  string `json:"AccessPointId"`
		AccessPointArn string `json:"AccessPointArn"`
		LifeCycleState string `json:"LifeCycleState"`
		Tags           []struct{ Key, Value string }
	}
	decode(t, w, &created)
	if created.AccessPointID == "" || created.LifeCycleState != "available" {
		t.Fatalf("unexpected access point: %+v", created)
	}

	// Duplicate client token should 409.
	w = doCall(t, h, "CreateAccessPoint", map[string]any{
		"FileSystemId": fsID,
		"ClientToken":  "ap-client-1",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate ClientToken, got %d", w.Code)
	}

	// Describe by file system.
	w = doCall(t, h, "DescribeAccessPoints", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeAccessPoints")
	var listed struct {
		AccessPoints []struct {
			AccessPointID string `json:"AccessPointId"`
		}
	}
	decode(t, w, &listed)
	if len(listed.AccessPoints) != 1 || listed.AccessPoints[0].AccessPointID != created.AccessPointID {
		t.Fatalf("unexpected list: %+v", listed)
	}

	// Delete.
	w = doCall(t, h, "DeleteAccessPoint", map[string]any{"AccessPointId": created.AccessPointID})
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteAccessPoint: expected 204, got %d", w.Code)
	}

	w = doCall(t, h, "DescribeAccessPoints", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeAccessPoints after delete")
	decode(t, w, &listed)
	if len(listed.AccessPoints) != 0 {
		t.Fatalf("expected 0 access points after delete, got %d", len(listed.AccessPoints))
	}
}

// ── File system policy ──────────────────────────────────────────────────────

func TestFileSystemPolicy(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "policy-tok")

	// Initially no policy.
	w := doCall(t, h, "DescribeFileSystemPolicy", map[string]any{"FileSystemId": fsID})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing policy, got %d", w.Code)
	}

	policy := `{"Version":"2012-10-17","Statement":[]}`
	w = doCall(t, h, "PutFileSystemPolicy", map[string]any{
		"FileSystemId": fsID,
		"Policy":       policy,
	})
	mustOK(t, w, "PutFileSystemPolicy")

	w = doCall(t, h, "DescribeFileSystemPolicy", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeFileSystemPolicy")
	var got struct{ Policy string }
	decode(t, w, &got)
	if got.Policy != policy {
		t.Fatalf("policy roundtrip broken, got %q", got.Policy)
	}

	w = doCall(t, h, "DeleteFileSystemPolicy", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DeleteFileSystemPolicy")

	w = doCall(t, h, "DescribeFileSystemPolicy", map[string]any{"FileSystemId": fsID})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

// ── Backup policy ───────────────────────────────────────────────────────────

func TestBackupPolicy(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "backup-tok")

	w := doCall(t, h, "DescribeBackupPolicy", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeBackupPolicy")
	var got struct {
		BackupPolicy struct{ Status string }
	}
	decode(t, w, &got)
	if got.BackupPolicy.Status != "DISABLED" {
		t.Fatalf("expected DISABLED, got %q", got.BackupPolicy.Status)
	}

	w = doCall(t, h, "PutBackupPolicy", map[string]any{
		"FileSystemId": fsID,
		"BackupPolicy": map[string]any{"Status": "ENABLED"},
	})
	mustOK(t, w, "PutBackupPolicy")
	decode(t, w, &got)
	if got.BackupPolicy.Status != "ENABLED" {
		t.Fatalf("expected ENABLED after put, got %q", got.BackupPolicy.Status)
	}
}

// ── Lifecycle configuration ─────────────────────────────────────────────────

func TestLifecycleConfiguration(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "lifecycle-tok")

	policies := []map[string]any{
		{"TransitionToIA": "AFTER_30_DAYS"},
		{"TransitionToPrimaryStorageClass": "AFTER_1_ACCESS"},
	}
	w := doCall(t, h, "PutLifecycleConfiguration", map[string]any{
		"FileSystemId":      fsID,
		"LifecyclePolicies": policies,
	})
	mustOK(t, w, "PutLifecycleConfiguration")

	w = doCall(t, h, "DescribeLifecycleConfiguration", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeLifecycleConfiguration")
	var got struct {
		LifecyclePolicies []map[string]any
	}
	decode(t, w, &got)
	if len(got.LifecyclePolicies) != 2 {
		t.Fatalf("expected 2 lifecycle policies, got %d", len(got.LifecyclePolicies))
	}
	if got.LifecyclePolicies[0]["TransitionToIA"] != "AFTER_30_DAYS" {
		t.Fatalf("unexpected policy: %+v", got.LifecyclePolicies[0])
	}
}

// ── Tags ────────────────────────────────────────────────────────────────────

func TestFileSystemTags(t *testing.T) {
	h := newGateway(t)
	fsID := createFileSystem(t, h, "tag-tok")

	// Add tags via legacy CreateTags API.
	w := doCall(t, h, "CreateTags", map[string]any{
		"FileSystemId": fsID,
		"Tags": []map[string]any{
			{"Key": "team", "Value": "platform"},
		},
	})
	mustOK(t, w, "CreateTags")

	w = doCall(t, h, "DescribeTags", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeTags")
	var dt struct {
		Tags []struct{ Key, Value string }
	}
	decode(t, w, &dt)
	if len(dt.Tags) != 3 {
		t.Fatalf("expected 3 tags (Name+env+team), got %d: %+v", len(dt.Tags), dt.Tags)
	}

	// Remove via DeleteTags.
	w = doCall(t, h, "DeleteTags", map[string]any{
		"FileSystemId": fsID,
		"TagKeys":      []string{"team"},
	})
	mustOK(t, w, "DeleteTags")
	w = doCall(t, h, "DescribeTags", map[string]any{"FileSystemId": fsID})
	mustOK(t, w, "DescribeTags after delete")
	decode(t, w, &dt)
	if len(dt.Tags) != 2 {
		t.Fatalf("expected 2 tags after delete, got %d", len(dt.Tags))
	}

	// Generic ResourceId tag API.
	w = doCall(t, h, "TagResource", map[string]any{
		"ResourceId": fsID,
		"Tags":       []map[string]any{{"Key": "owner", "Value": "alice"}},
	})
	mustOK(t, w, "TagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceId": fsID})
	mustOK(t, w, "ListTagsForResource")
	var lt struct {
		Tags []struct{ Key, Value string }
	}
	decode(t, w, &lt)
	if len(lt.Tags) != 3 {
		t.Fatalf("expected 3 tags after TagResource, got %d", len(lt.Tags))
	}

	w = doCall(t, h, "UntagResource", map[string]any{
		"ResourceId": fsID,
		"tagKeys":    []string{"owner", "env"},
	})
	mustOK(t, w, "UntagResource")
	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceId": fsID})
	mustOK(t, w, "ListTagsForResource final")
	decode(t, w, &lt)
	if len(lt.Tags) != 1 {
		t.Fatalf("expected 1 tag after untag, got %d", len(lt.Tags))
	}
}

// ── Replication configuration ───────────────────────────────────────────────

func TestReplicationLifecycle(t *testing.T) {
	h := newGateway(t)
	srcID := createFileSystem(t, h, "rep-src")

	w := doCall(t, h, "CreateReplicationConfiguration", map[string]any{
		"SourceFileSystemId": srcID,
		"Destinations": []map[string]any{
			{"Region": "us-west-2"},
		},
	})
	mustOK(t, w, "CreateReplicationConfiguration")
	var created struct {
		SourceFileSystemID string `json:"SourceFileSystemId"`
		Destinations       []struct {
			FileSystemID string `json:"FileSystemId"`
			Region       string
			Status       string
		}
	}
	decode(t, w, &created)
	if created.SourceFileSystemID != srcID {
		t.Fatalf("unexpected source: %s", created.SourceFileSystemID)
	}
	if len(created.Destinations) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(created.Destinations))
	}
	if created.Destinations[0].FileSystemID == "" {
		t.Fatalf("expected destination FileSystemId to be allocated")
	}

	// Duplicate config conflicts.
	w = doCall(t, h, "CreateReplicationConfiguration", map[string]any{
		"SourceFileSystemId": srcID,
		"Destinations": []map[string]any{
			{"Region": "us-west-2"},
		},
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate replication, got %d", w.Code)
	}

	w = doCall(t, h, "DescribeReplicationConfigurations", map[string]any{"FileSystemId": srcID})
	mustOK(t, w, "DescribeReplicationConfigurations")
	var listed struct {
		Replications []struct {
			SourceFileSystemID string `json:"SourceFileSystemId"`
		}
	}
	decode(t, w, &listed)
	if len(listed.Replications) != 1 || listed.Replications[0].SourceFileSystemID != srcID {
		t.Fatalf("unexpected list: %+v", listed)
	}

	w = doCall(t, h, "DeleteReplicationConfiguration", map[string]any{"SourceFileSystemId": srcID})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	w = doCall(t, h, "DeleteReplicationConfiguration", map[string]any{"SourceFileSystemId": srcID})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", w.Code)
	}
}

// ── Account preferences ─────────────────────────────────────────────────────

func TestAccountPreferences(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "DescribeAccountPreferences", nil)
	mustOK(t, w, "DescribeAccountPreferences")
	var got struct {
		ResourceIdPreference struct {
			ResourceIdType string
		}
	}
	decode(t, w, &got)
	if got.ResourceIdPreference.ResourceIdType != "LONG_ID" {
		t.Fatalf("expected default LONG_ID, got %q", got.ResourceIdPreference.ResourceIdType)
	}

	w = doCall(t, h, "PutAccountPreferences", map[string]any{"ResourceIdType": "SHORT_ID"})
	mustOK(t, w, "PutAccountPreferences")
	decode(t, w, &got)
	if got.ResourceIdPreference.ResourceIdType != "SHORT_ID" {
		t.Fatalf("expected SHORT_ID, got %q", got.ResourceIdPreference.ResourceIdType)
	}

	// Bogus value rejected.
	w = doCall(t, h, "PutAccountPreferences", map[string]any{"ResourceIdType": "BOGUS"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on bogus value, got %d", w.Code)
	}
}

// ── Validation ──────────────────────────────────────────────────────────────

func TestRequiredFieldValidation(t *testing.T) {
	h := newGateway(t)

	cases := []struct {
		action string
		body   map[string]any
	}{
		{"DescribeBackupPolicy", map[string]any{}},
		{"DescribeFileSystemPolicy", map[string]any{}},
		{"DescribeLifecycleConfiguration", map[string]any{}},
		{"PutBackupPolicy", map[string]any{"FileSystemId": "fs-1"}},   // missing BackupPolicy
		{"PutFileSystemPolicy", map[string]any{"FileSystemId": "fs-1"}}, // missing Policy
		{"DeleteAccessPoint", map[string]any{}},
		{"DeleteMountTarget", map[string]any{}},
		{"DescribeMountTargets", map[string]any{}},
		{"CreateTags", map[string]any{"FileSystemId": "fs-1"}},     // missing Tags
		{"DeleteTags", map[string]any{"FileSystemId": "fs-1"}},     // missing TagKeys
		{"TagResource", map[string]any{"ResourceId": "fs-1"}},      // missing Tags
		{"UntagResource", map[string]any{"ResourceId": "fs-1"}},    // missing tagKeys
		{"ListTagsForResource", map[string]any{}},
		{"PutAccountPreferences", map[string]any{}},
		{"CreateReplicationConfiguration", map[string]any{}},
		{"DeleteReplicationConfiguration", map[string]any{}},
	}
	for _, tc := range cases {
		w := doCall(t, h, tc.action, tc.body)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("%s: expected 400 (validation) or 404 (resource), got %d: %s",
				tc.action, w.Code, w.Body.String())
		}
	}
}
