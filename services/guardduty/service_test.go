package guardduty_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/guardduty"
)

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
	req.Header.Set("X-Amz-Target", "guardduty."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/guardduty/aws4_request, SignedHeaders=host, Signature=abc")
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

func createDetector(t *testing.T, h http.Handler) string {
	t.Helper()
	w := doCall(t, h, "CreateDetector", map[string]any{
		"enable":                     true,
		"findingPublishingFrequency": "FIFTEEN_MINUTES",
		"tags":                       map[string]any{"env": "test"},
	})
	mustOK(t, w, "CreateDetector")
	var resp struct {
		DetectorID string `json:"detectorId"`
	}
	decode(t, w, &resp)
	if resp.DetectorID == "" {
		t.Fatalf("CreateDetector returned empty detectorId: %s", w.Body.String())
	}
	return resp.DetectorID
}

// ── Detector lifecycle ──────────────────────────────────────────────────────

func TestDetectorLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	// Get
	w := doCall(t, h, "GetDetector", map[string]any{"detectorId": id})
	mustOK(t, w, "GetDetector")
	var got struct {
		Status                     string `json:"status"`
		FindingPublishingFrequency string `json:"findingPublishingFrequency"`
	}
	decode(t, w, &got)
	if got.Status != "ENABLED" {
		t.Fatalf("Detector status: want ENABLED, got %q", got.Status)
	}
	if got.FindingPublishingFrequency != "FIFTEEN_MINUTES" {
		t.Fatalf("Detector freq: want FIFTEEN_MINUTES, got %q", got.FindingPublishingFrequency)
	}

	// List
	w = doCall(t, h, "ListDetectors", nil)
	mustOK(t, w, "ListDetectors")
	var listed struct {
		DetectorIDs []string `json:"detectorIds"`
	}
	decode(t, w, &listed)
	if len(listed.DetectorIDs) != 1 || listed.DetectorIDs[0] != id {
		t.Fatalf("ListDetectors: %+v", listed)
	}

	// Update
	w = doCall(t, h, "UpdateDetector", map[string]any{
		"detectorId":                 id,
		"enable":                     false,
		"findingPublishingFrequency": "ONE_HOUR",
	})
	mustOK(t, w, "UpdateDetector")

	w = doCall(t, h, "GetDetector", map[string]any{"detectorId": id})
	mustOK(t, w, "GetDetector after update")
	decode(t, w, &got)
	if got.Status != "DISABLED" || got.FindingPublishingFrequency != "ONE_HOUR" {
		t.Fatalf("update did not stick: %+v", got)
	}

	// Delete
	w = doCall(t, h, "DeleteDetector", map[string]any{"detectorId": id})
	mustOK(t, w, "DeleteDetector")
	w = doCall(t, h, "GetDetector", map[string]any{"detectorId": id})
	if w.Code == http.StatusOK {
		t.Fatalf("GetDetector after delete should fail, got 200")
	}
}

func TestCreateDetectorRequiresValidPayload(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateDetector", nil) // empty body still works (enable=false)
	mustOK(t, w, "CreateDetector empty body")
}

// ── Filter lifecycle ────────────────────────────────────────────────────────

func TestFilterLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateFilter", map[string]any{
		"detectorId": id,
		"name":       "block-low",
		"action":     "ARCHIVE",
		"rank":       1,
		"findingCriteria": map[string]any{
			"criterion": map[string]any{
				"severity": map[string]any{"gt": 4},
			},
		},
		"tags": map[string]any{"team": "secops"},
	})
	mustOK(t, w, "CreateFilter")

	// Duplicate should fail
	if w := doCall(t, h, "CreateFilter", map[string]any{
		"detectorId": id,
		"name":       "block-low",
		"action":     "NOOP",
	}); w.Code == http.StatusOK {
		t.Fatalf("duplicate CreateFilter should fail")
	}

	// Get
	w = doCall(t, h, "GetFilter", map[string]any{"detectorId": id, "filterName": "block-low"})
	mustOK(t, w, "GetFilter")
	var f struct {
		Action string `json:"action"`
		Name   string `json:"name"`
		Rank   int    `json:"rank"`
	}
	decode(t, w, &f)
	if f.Action != "ARCHIVE" || f.Name != "block-low" || f.Rank != 1 {
		t.Fatalf("filter mismatch: %+v", f)
	}

	// Update
	w = doCall(t, h, "UpdateFilter", map[string]any{
		"detectorId":  id,
		"filterName":  "block-low",
		"description": "updated desc",
		"rank":        7,
	})
	mustOK(t, w, "UpdateFilter")

	// List
	w = doCall(t, h, "ListFilters", map[string]any{"detectorId": id})
	mustOK(t, w, "ListFilters")
	var listed struct {
		FilterNames []string `json:"filterNames"`
	}
	decode(t, w, &listed)
	if len(listed.FilterNames) != 1 || listed.FilterNames[0] != "block-low" {
		t.Fatalf("ListFilters: %+v", listed)
	}

	// Delete
	w = doCall(t, h, "DeleteFilter", map[string]any{"detectorId": id, "filterName": "block-low"})
	mustOK(t, w, "DeleteFilter")
	w = doCall(t, h, "GetFilter", map[string]any{"detectorId": id, "filterName": "block-low"})
	if w.Code == http.StatusOK {
		t.Fatal("GetFilter after delete should fail")
	}
}

// ── IPSet lifecycle ─────────────────────────────────────────────────────────

func TestIPSetLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateIPSet", map[string]any{
		"detectorId": id,
		"name":       "trusted-ips",
		"format":     "TXT",
		"location":   "https://s3.amazonaws.com/example/ipset.txt",
		"activate":   true,
		"tags":       map[string]any{"src": "audit"},
	})
	mustOK(t, w, "CreateIPSet")
	var created struct {
		IPSetID string `json:"ipSetId"`
	}
	decode(t, w, &created)
	if created.IPSetID == "" {
		t.Fatal("missing ipSetId")
	}

	w = doCall(t, h, "GetIPSet", map[string]any{"detectorId": id, "ipSetId": created.IPSetID})
	mustOK(t, w, "GetIPSet")
	var got struct {
		Format   string `json:"format"`
		Location string `json:"location"`
		Name     string `json:"name"`
		Status   string `json:"status"`
	}
	decode(t, w, &got)
	if got.Format != "TXT" || got.Status != "ACTIVE" || got.Name != "trusted-ips" {
		t.Fatalf("ipset mismatch: %+v", got)
	}

	w = doCall(t, h, "UpdateIPSet", map[string]any{
		"detectorId": id,
		"ipSetId":    created.IPSetID,
		"location":   "https://s3.amazonaws.com/example/v2.txt",
		"activate":   false,
	})
	mustOK(t, w, "UpdateIPSet")

	w = doCall(t, h, "GetIPSet", map[string]any{"detectorId": id, "ipSetId": created.IPSetID})
	mustOK(t, w, "GetIPSet after update")
	decode(t, w, &got)
	if got.Status != "INACTIVE" {
		t.Fatalf("expected INACTIVE, got %q", got.Status)
	}

	w = doCall(t, h, "ListIPSets", map[string]any{"detectorId": id})
	mustOK(t, w, "ListIPSets")
	var listed struct {
		IPSetIDs []string `json:"ipSetIds"`
	}
	decode(t, w, &listed)
	if len(listed.IPSetIDs) != 1 {
		t.Fatalf("expected 1 ipset, got %+v", listed)
	}

	w = doCall(t, h, "DeleteIPSet", map[string]any{"detectorId": id, "ipSetId": created.IPSetID})
	mustOK(t, w, "DeleteIPSet")
}

// ── ThreatIntelSet lifecycle ────────────────────────────────────────────────

func TestThreatIntelSetLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateThreatIntelSet", map[string]any{
		"detectorId": id,
		"name":       "bad-ips",
		"format":     "STIX",
		"location":   "https://s3.amazonaws.com/example/stix.xml",
		"activate":   true,
	})
	mustOK(t, w, "CreateThreatIntelSet")
	var created struct {
		ThreatIntelSetID string `json:"threatIntelSetId"`
	}
	decode(t, w, &created)

	w = doCall(t, h, "GetThreatIntelSet", map[string]any{"detectorId": id, "threatIntelSetId": created.ThreatIntelSetID})
	mustOK(t, w, "GetThreatIntelSet")

	w = doCall(t, h, "ListThreatIntelSets", map[string]any{"detectorId": id})
	mustOK(t, w, "ListThreatIntelSets")
	var listed struct {
		IDs []string `json:"threatIntelSetIds"`
	}
	decode(t, w, &listed)
	if len(listed.IDs) != 1 {
		t.Fatalf("expected 1 set, got %+v", listed)
	}

	w = doCall(t, h, "UpdateThreatIntelSet", map[string]any{
		"detectorId":       id,
		"threatIntelSetId": created.ThreatIntelSetID,
		"name":             "bad-ips-renamed",
	})
	mustOK(t, w, "UpdateThreatIntelSet")

	w = doCall(t, h, "DeleteThreatIntelSet", map[string]any{"detectorId": id, "threatIntelSetId": created.ThreatIntelSetID})
	mustOK(t, w, "DeleteThreatIntelSet")
}

// ── Threat / Trusted entity sets ────────────────────────────────────────────

func TestThreatEntitySetLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateThreatEntitySet", map[string]any{
		"detectorId": id,
		"name":       "ts1",
		"format":     "TXT",
		"location":   "s3://example/threat.txt",
		"activate":   true,
	})
	mustOK(t, w, "CreateThreatEntitySet")
	var created struct {
		ThreatEntitySetID string `json:"threatEntitySetId"`
	}
	decode(t, w, &created)

	w = doCall(t, h, "GetThreatEntitySet", map[string]any{"detectorId": id, "threatEntitySetId": created.ThreatEntitySetID})
	mustOK(t, w, "GetThreatEntitySet")

	w = doCall(t, h, "UpdateThreatEntitySet", map[string]any{
		"detectorId":        id,
		"threatEntitySetId": created.ThreatEntitySetID,
		"location":          "s3://example/v2.txt",
	})
	mustOK(t, w, "UpdateThreatEntitySet")

	w = doCall(t, h, "ListThreatEntitySets", map[string]any{"detectorId": id})
	mustOK(t, w, "ListThreatEntitySets")
	var listed struct {
		IDs []string `json:"threatEntitySetIds"`
	}
	decode(t, w, &listed)
	if len(listed.IDs) != 1 {
		t.Fatalf("expected 1, got %+v", listed)
	}

	w = doCall(t, h, "DeleteThreatEntitySet", map[string]any{"detectorId": id, "threatEntitySetId": created.ThreatEntitySetID})
	mustOK(t, w, "DeleteThreatEntitySet")
}

func TestTrustedEntitySetLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateTrustedEntitySet", map[string]any{
		"detectorId": id,
		"name":       "good1",
		"format":     "TXT",
		"location":   "s3://example/trusted.txt",
		"activate":   true,
	})
	mustOK(t, w, "CreateTrustedEntitySet")
	var created struct {
		TrustedEntitySetID string `json:"trustedEntitySetId"`
	}
	decode(t, w, &created)

	w = doCall(t, h, "GetTrustedEntitySet", map[string]any{"detectorId": id, "trustedEntitySetId": created.TrustedEntitySetID})
	mustOK(t, w, "GetTrustedEntitySet")

	w = doCall(t, h, "UpdateTrustedEntitySet", map[string]any{
		"detectorId":         id,
		"trustedEntitySetId": created.TrustedEntitySetID,
		"name":               "good1-updated",
	})
	mustOK(t, w, "UpdateTrustedEntitySet")

	w = doCall(t, h, "ListTrustedEntitySets", map[string]any{"detectorId": id})
	mustOK(t, w, "ListTrustedEntitySets")
	var listed struct {
		IDs []string `json:"trustedEntitySetIds"`
	}
	decode(t, w, &listed)
	if len(listed.IDs) != 1 {
		t.Fatalf("expected 1, got %+v", listed)
	}

	w = doCall(t, h, "DeleteTrustedEntitySet", map[string]any{"detectorId": id, "trustedEntitySetId": created.TrustedEntitySetID})
	mustOK(t, w, "DeleteTrustedEntitySet")
}

// ── Members / invitations ───────────────────────────────────────────────────

func TestMembersLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateMembers", map[string]any{
		"detectorId": id,
		"accountDetails": []map[string]any{
			{"accountId": "111122223333", "email": "a@example.com"},
			{"accountId": "444455556666", "email": "b@example.com"},
		},
	})
	mustOK(t, w, "CreateMembers")

	w = doCall(t, h, "ListMembers", map[string]any{"detectorId": id})
	mustOK(t, w, "ListMembers")
	var listed struct {
		Members []struct {
			AccountID string `json:"accountId"`
		} `json:"members"`
	}
	decode(t, w, &listed)
	if len(listed.Members) != 2 {
		t.Fatalf("expected 2 members, got %+v", listed)
	}

	w = doCall(t, h, "GetMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333", "999999999999"},
	})
	mustOK(t, w, "GetMembers")
	var got struct {
		Members []struct {
			AccountID string `json:"accountId"`
		} `json:"members"`
		Unprocessed []map[string]any `json:"unprocessedAccounts"`
	}
	decode(t, w, &got)
	if len(got.Members) != 1 || got.Members[0].AccountID != "111122223333" {
		t.Fatalf("unexpected members: %+v", got)
	}
	if len(got.Unprocessed) != 1 {
		t.Fatalf("expected 1 unprocessed, got %+v", got.Unprocessed)
	}

	w = doCall(t, h, "InviteMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333", "444455556666"},
	})
	mustOK(t, w, "InviteMembers")

	w = doCall(t, h, "GetInvitationsCount", nil)
	mustOK(t, w, "GetInvitationsCount")
	var count struct {
		Count int `json:"invitationsCount"`
	}
	decode(t, w, &count)
	if count.Count != 2 {
		t.Fatalf("expected 2 invitations, got %d", count.Count)
	}

	w = doCall(t, h, "ListInvitations", nil)
	mustOK(t, w, "ListInvitations")

	w = doCall(t, h, "StartMonitoringMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "StartMonitoringMembers")

	w = doCall(t, h, "StopMonitoringMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "StopMonitoringMembers")

	w = doCall(t, h, "UpdateMemberDetectors", map[string]any{
		"detectorId":  id,
		"accountIds":  []string{"111122223333"},
		"dataSources": map[string]any{"s3Logs": map[string]any{"enable": true}},
	})
	mustOK(t, w, "UpdateMemberDetectors")

	w = doCall(t, h, "GetMemberDetectors", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "GetMemberDetectors")

	w = doCall(t, h, "DisassociateMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "DisassociateMembers")

	w = doCall(t, h, "DeleteMembers", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333", "444455556666"},
	})
	mustOK(t, w, "DeleteMembers")

	w = doCall(t, h, "DeclineInvitations", map[string]any{
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "DeclineInvitations")

	w = doCall(t, h, "DeleteInvitations", map[string]any{
		"accountIds": []string{"444455556666"},
	})
	mustOK(t, w, "DeleteInvitations")
}

// ── Administrator/Master ────────────────────────────────────────────────────

func TestAdministratorLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	// Initially empty
	w := doCall(t, h, "GetAdministratorAccount", nil)
	mustOK(t, w, "GetAdministratorAccount")
	w = doCall(t, h, "GetMasterAccount", nil)
	mustOK(t, w, "GetMasterAccount")

	// Accept admin invitation
	w = doCall(t, h, "AcceptAdministratorInvitation", map[string]any{
		"detectorId":      id,
		"administratorId": "111122223333",
		"invitationId":    "inv-1",
	})
	mustOK(t, w, "AcceptAdministratorInvitation")

	// Now should be present
	w = doCall(t, h, "GetAdministratorAccount", nil)
	mustOK(t, w, "GetAdministratorAccount after accept")
	var admin struct {
		Administrator struct {
			AccountID string `json:"accountId"`
		} `json:"administrator"`
	}
	decode(t, w, &admin)
	if admin.Administrator.AccountID != "111122223333" {
		t.Fatalf("expected 111122223333, got %q", admin.Administrator.AccountID)
	}

	// Disassociate
	w = doCall(t, h, "DisassociateFromAdministratorAccount", map[string]any{"detectorId": id})
	mustOK(t, w, "DisassociateFromAdministratorAccount")

	// AcceptInvitation (legacy)
	w = doCall(t, h, "AcceptInvitation", map[string]any{
		"detectorId":   id,
		"masterId":     "999988887777",
		"invitationId": "inv-2",
	})
	mustOK(t, w, "AcceptInvitation")

	w = doCall(t, h, "DisassociateFromMasterAccount", map[string]any{"detectorId": id})
	mustOK(t, w, "DisassociateFromMasterAccount")
}

// ── Findings ────────────────────────────────────────────────────────────────

func TestFindingsLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreateSampleFindings", map[string]any{
		"detectorId":   id,
		"findingTypes": []string{"Recon:EC2/PortProbeUnprotectedPort", "Trojan:EC2/BlackholeTraffic"},
	})
	mustOK(t, w, "CreateSampleFindings")

	w = doCall(t, h, "ListFindings", map[string]any{"detectorId": id})
	mustOK(t, w, "ListFindings")
	var listed struct {
		IDs []string `json:"findingIds"`
	}
	decode(t, w, &listed)
	if len(listed.IDs) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(listed.IDs))
	}

	w = doCall(t, h, "GetFindings", map[string]any{
		"detectorId": id,
		"findingIds": listed.IDs,
	})
	mustOK(t, w, "GetFindings")
	var got struct {
		Findings []struct {
			ID       string  `json:"id"`
			Severity float64 `json:"severity"`
			Type     string  `json:"type"`
		} `json:"findings"`
	}
	decode(t, w, &got)
	if len(got.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(got.Findings))
	}

	w = doCall(t, h, "ArchiveFindings", map[string]any{
		"detectorId": id,
		"findingIds": listed.IDs[:1],
	})
	mustOK(t, w, "ArchiveFindings")

	w = doCall(t, h, "UnarchiveFindings", map[string]any{
		"detectorId": id,
		"findingIds": listed.IDs[:1],
	})
	mustOK(t, w, "UnarchiveFindings")

	w = doCall(t, h, "UpdateFindingsFeedback", map[string]any{
		"detectorId": id,
		"findingIds": listed.IDs,
		"feedback":   "USEFUL",
	})
	mustOK(t, w, "UpdateFindingsFeedback")

	w = doCall(t, h, "GetFindingsStatistics", map[string]any{
		"detectorId":            id,
		"findingStatisticTypes": []string{"COUNT_BY_SEVERITY"},
	})
	mustOK(t, w, "GetFindingsStatistics")
}

// ── Publishing destinations ─────────────────────────────────────────────────

func TestPublishingDestinationLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "CreatePublishingDestination", map[string]any{
		"detectorId":      id,
		"destinationType": "S3",
		"destinationProperties": map[string]any{
			"destinationArn": "arn:aws:s3:::example/findings",
			"kmsKeyArn":      "arn:aws:kms:us-east-1:000000000000:key/abc",
		},
	})
	mustOK(t, w, "CreatePublishingDestination")
	var created struct {
		DestinationID string `json:"destinationId"`
	}
	decode(t, w, &created)

	w = doCall(t, h, "DescribePublishingDestination", map[string]any{
		"detectorId":    id,
		"destinationId": created.DestinationID,
	})
	mustOK(t, w, "DescribePublishingDestination")

	w = doCall(t, h, "ListPublishingDestinations", map[string]any{"detectorId": id})
	mustOK(t, w, "ListPublishingDestinations")
	var listed struct {
		Destinations []struct {
			DestinationID string `json:"destinationId"`
		} `json:"destinations"`
	}
	decode(t, w, &listed)
	if len(listed.Destinations) != 1 {
		t.Fatalf("expected 1 destination, got %+v", listed)
	}

	w = doCall(t, h, "UpdatePublishingDestination", map[string]any{
		"detectorId":    id,
		"destinationId": created.DestinationID,
		"destinationProperties": map[string]any{
			"destinationArn": "arn:aws:s3:::example/findings-v2",
		},
	})
	mustOK(t, w, "UpdatePublishingDestination")

	w = doCall(t, h, "DeletePublishingDestination", map[string]any{
		"detectorId":    id,
		"destinationId": created.DestinationID,
	})
	mustOK(t, w, "DeletePublishingDestination")
}

// ── Organization configuration ──────────────────────────────────────────────

func TestOrganizationLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "DescribeOrganizationConfiguration", map[string]any{"detectorId": id})
	mustOK(t, w, "DescribeOrganizationConfiguration")

	w = doCall(t, h, "UpdateOrganizationConfiguration", map[string]any{
		"detectorId":                    id,
		"autoEnable":                    true,
		"autoEnableOrganizationMembers": "ALL",
	})
	mustOK(t, w, "UpdateOrganizationConfiguration")

	w = doCall(t, h, "DescribeOrganizationConfiguration", map[string]any{"detectorId": id})
	mustOK(t, w, "DescribeOrganizationConfiguration after update")
	var got struct {
		AutoEnable                    bool   `json:"autoEnable"`
		AutoEnableOrganizationMembers string `json:"autoEnableOrganizationMembers"`
	}
	decode(t, w, &got)
	if !got.AutoEnable || got.AutoEnableOrganizationMembers != "ALL" {
		t.Fatalf("org config update: %+v", got)
	}

	w = doCall(t, h, "EnableOrganizationAdminAccount", map[string]any{
		"adminAccountId": "111122223333",
	})
	mustOK(t, w, "EnableOrganizationAdminAccount")

	w = doCall(t, h, "ListOrganizationAdminAccounts", nil)
	mustOK(t, w, "ListOrganizationAdminAccounts")
	var admins struct {
		Accounts []struct {
			AdminAccountID string `json:"adminAccountId"`
		} `json:"adminAccounts"`
	}
	decode(t, w, &admins)
	if len(admins.Accounts) != 1 || admins.Accounts[0].AdminAccountID != "111122223333" {
		t.Fatalf("unexpected admins: %+v", admins)
	}

	w = doCall(t, h, "GetOrganizationStatistics", nil)
	mustOK(t, w, "GetOrganizationStatistics")

	w = doCall(t, h, "DisableOrganizationAdminAccount", map[string]any{
		"adminAccountId": "111122223333",
	})
	mustOK(t, w, "DisableOrganizationAdminAccount")

	if w := doCall(t, h, "DisableOrganizationAdminAccount", map[string]any{
		"adminAccountId": "111122223333",
	}); w.Code == http.StatusOK {
		t.Fatal("disabling unknown admin should fail")
	}
}

// ── Malware protection ──────────────────────────────────────────────────────

func TestMalwareProtectionPlanLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateMalwareProtectionPlan", map[string]any{
		"role": "arn:aws:iam::000000000000:role/MalwareScanRole",
		"protectedResource": map[string]any{
			"s3Bucket": map[string]any{"bucketName": "example-bucket"},
		},
		"actions": map[string]any{
			"tagging": map[string]any{"status": "ENABLED"},
		},
		"tags": map[string]any{"team": "secops"},
	})
	mustOK(t, w, "CreateMalwareProtectionPlan")
	var created struct {
		PlanID string `json:"malwareProtectionPlanId"`
	}
	decode(t, w, &created)

	w = doCall(t, h, "GetMalwareProtectionPlan", map[string]any{"malwareProtectionPlanId": created.PlanID})
	mustOK(t, w, "GetMalwareProtectionPlan")

	w = doCall(t, h, "ListMalwareProtectionPlans", nil)
	mustOK(t, w, "ListMalwareProtectionPlans")

	w = doCall(t, h, "UpdateMalwareProtectionPlan", map[string]any{
		"malwareProtectionPlanId": created.PlanID,
		"role":                    "arn:aws:iam::000000000000:role/MalwareScanRoleV2",
	})
	mustOK(t, w, "UpdateMalwareProtectionPlan")

	w = doCall(t, h, "DeleteMalwareProtectionPlan", map[string]any{"malwareProtectionPlanId": created.PlanID})
	mustOK(t, w, "DeleteMalwareProtectionPlan")
}

// ── Malware scan / settings ─────────────────────────────────────────────────

func TestMalwareScanLifecycle(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	w := doCall(t, h, "GetMalwareScanSettings", map[string]any{"detectorId": id})
	mustOK(t, w, "GetMalwareScanSettings")

	w = doCall(t, h, "UpdateMalwareScanSettings", map[string]any{
		"detectorId":              id,
		"ebsSnapshotPreservation": "RETENTION_WITH_FINDING",
		"scanResourceCriteria": map[string]any{
			"include": map[string]any{},
			"exclude": map[string]any{},
		},
	})
	mustOK(t, w, "UpdateMalwareScanSettings")

	w = doCall(t, h, "StartMalwareScan", map[string]any{
		"resourceArn": "arn:aws:ec2:us-east-1:000000000000:instance/i-12345",
	})
	mustOK(t, w, "StartMalwareScan")
	var scan struct {
		ScanID string `json:"scanId"`
	}
	decode(t, w, &scan)
	if scan.ScanID == "" {
		t.Fatal("missing scanId")
	}

	w = doCall(t, h, "DescribeMalwareScans", map[string]any{"detectorId": id})
	mustOK(t, w, "DescribeMalwareScans")

	w = doCall(t, h, "ListMalwareScans", nil)
	mustOK(t, w, "ListMalwareScans")

	w = doCall(t, h, "GetMalwareScan", map[string]any{"scanId": scan.ScanID})
	mustOK(t, w, "GetMalwareScan")

	w = doCall(t, h, "SendObjectMalwareScan", map[string]any{
		"resourceArn": "arn:aws:s3:::example/object",
	})
	mustOK(t, w, "SendObjectMalwareScan")
}

// ── Coverage / usage / free trial ───────────────────────────────────────────

func TestCoverageAndUsage(t *testing.T) {
	h := newGateway(t)
	id := createDetector(t, h)

	doCall(t, h, "CreateMembers", map[string]any{
		"detectorId": id,
		"accountDetails": []map[string]any{
			{"accountId": "111122223333", "email": "x@example.com"},
		},
	})

	w := doCall(t, h, "ListCoverage", map[string]any{"detectorId": id})
	mustOK(t, w, "ListCoverage")

	w = doCall(t, h, "GetCoverageStatistics", map[string]any{
		"detectorId":     id,
		"statisticsType": []string{"COUNT_BY_RESOURCE_TYPE"},
	})
	mustOK(t, w, "GetCoverageStatistics")

	w = doCall(t, h, "GetUsageStatistics", map[string]any{
		"detectorId":         id,
		"usageStatisticsType": "SUM_BY_ACCOUNT",
	})
	mustOK(t, w, "GetUsageStatistics")

	w = doCall(t, h, "GetRemainingFreeTrialDays", map[string]any{
		"detectorId": id,
		"accountIds": []string{"111122223333"},
	})
	mustOK(t, w, "GetRemainingFreeTrialDays")
}

// ── Tagging ─────────────────────────────────────────────────────────────────

func TestTagsLifecycle(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:guardduty:us-east-1:000000000000:detector/abcd"

	w := doCall(t, h, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags": map[string]any{
			"env":  "dev",
			"team": "security",
		},
	})
	mustOK(t, w, "TagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	mustOK(t, w, "ListTagsForResource")
	var listed struct {
		Tags map[string]string `json:"tags"`
	}
	decode(t, w, &listed)
	if listed.Tags["env"] != "dev" || listed.Tags["team"] != "security" {
		t.Fatalf("unexpected tags: %+v", listed.Tags)
	}

	w = doCall(t, h, "UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []string{"env"},
	})
	mustOK(t, w, "UntagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	mustOK(t, w, "ListTagsForResource after untag")
	var afterUntag struct {
		Tags map[string]string `json:"tags"`
	}
	decode(t, w, &afterUntag)
	if _, ok := afterUntag.Tags["env"]; ok {
		t.Fatalf("env tag should be removed: %+v", afterUntag.Tags)
	}
	if afterUntag.Tags["team"] != "security" {
		t.Fatalf("team tag should remain: %+v", afterUntag.Tags)
	}
}

// ── Validation: missing detectorId ──────────────────────────────────────────

func TestMissingDetectorIDValidation(t *testing.T) {
	h := newGateway(t)
	cases := []string{
		"GetDetector",
		"DeleteDetector",
		"UpdateDetector",
		"CreateFilter",
		"GetFilter",
		"ListFilters",
		"CreateIPSet",
		"GetIPSet",
		"ListIPSets",
		"ListMembers",
	}
	for _, action := range cases {
		w := doCall(t, h, action, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s with missing detectorId: want 400, got %d", action, w.Code)
		}
	}
}

// ── Resource not found returns 4xx ──────────────────────────────────────────

func TestResourceNotFound(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "GetDetector", map[string]any{"detectorId": "no-such"})
	if w.Code == http.StatusOK {
		t.Fatal("GetDetector for missing id should fail")
	}
}

// ── Reset ───────────────────────────────────────────────────────────────────

func TestStoreReset(t *testing.T) {
	store := svc.NewStore("000000000000", "us-east-1")
	d := store.CreateDetector(true, "FIFTEEN_MINUTES", nil, nil, nil)
	if _, err := store.GetDetector(d.DetectorID); err != nil {
		t.Fatalf("detector should exist before reset: %v", err)
	}
	store.Reset()
	if _, err := store.GetDetector(d.DetectorID); err == nil {
		t.Fatal("detector should not exist after reset")
	}
	if ids := store.ListDetectors(); len(ids) != 0 {
		t.Fatalf("ListDetectors after reset: %+v", ids)
	}
}
