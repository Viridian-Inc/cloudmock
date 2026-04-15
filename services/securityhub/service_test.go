package securityhub_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/securityhub"
)

// ── Test infrastructure ──────────────────────────────────────────────────────

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
	req.Header.Set("X-Amz-Target", "securityhub."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/securityhub/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// callOK invokes an action and returns the parsed JSON response. Fails the test on non-200.
func callOK(t *testing.T, h http.Handler, action string, body any) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d\nbody: %s", action, w.Code, w.Body.String())
	}
	var resp map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s: invalid JSON response: %v\nbody: %s", action, err, w.Body.String())
		}
	}
	return resp
}

// callStatus invokes an action and returns status code + parsed body without failing.
func callStatus(t *testing.T, h http.Handler, action string, body any) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	var resp map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w.Code, resp
}

// ── Hub lifecycle ────────────────────────────────────────────────────────────

func TestHubLifecycle(t *testing.T) {
	h := newGateway(t)

	// Enable.
	resp := callOK(t, h, "EnableSecurityHub", map[string]any{
		"EnableDefaultStandards": true,
		"Tags": map[string]any{"env": "test"},
	})
	if resp["HubArn"] == "" {
		t.Fatalf("EnableSecurityHub: expected HubArn, got %+v", resp)
	}
	hubArn := resp["HubArn"].(string)
	if !strings.Contains(hubArn, "hub/default") {
		t.Errorf("HubArn should contain 'hub/default', got %s", hubArn)
	}

	// Describe.
	desc := callOK(t, h, "DescribeHub", nil)
	if desc["HubArn"] != hubArn {
		t.Errorf("DescribeHub HubArn mismatch: %v", desc["HubArn"])
	}

	// Update config.
	callOK(t, h, "UpdateSecurityHubConfiguration", map[string]any{
		"AutoEnableControls":      false,
		"ControlFindingGenerator": "STANDARD_CONTROL",
	})
	desc = callOK(t, h, "DescribeHub", nil)
	if desc["AutoEnableControls"].(bool) {
		t.Errorf("UpdateSecurityHubConfiguration: AutoEnableControls should be false")
	}
	if desc["ControlFindingGenerator"] != "STANDARD_CONTROL" {
		t.Errorf("UpdateSecurityHubConfiguration: generator mismatch")
	}

	// Re-enabling should conflict.
	if code, _ := callStatus(t, h, "EnableSecurityHub", nil); code != http.StatusConflict {
		t.Errorf("Re-enabling should return 409, got %d", code)
	}

	// Disable.
	callOK(t, h, "DisableSecurityHub", nil)
	if code, _ := callStatus(t, h, "DescribeHub", nil); code != http.StatusBadRequest {
		t.Errorf("DescribeHub after disable should fail, got %d", code)
	}

	// Disable when not enabled fails.
	if code, _ := callStatus(t, h, "DisableSecurityHub", nil); code != http.StatusBadRequest {
		t.Errorf("DisableSecurityHub when disabled should fail, got %d", code)
	}
}

func TestHubV2Lifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callOK(t, h, "EnableSecurityHubV2", nil)
	if !strings.Contains(resp["HubArn"].(string), "hub/v2") {
		t.Errorf("EnableSecurityHubV2: HubArn should contain hub/v2, got %v", resp["HubArn"])
	}

	desc := callOK(t, h, "DescribeSecurityHubV2", nil)
	if desc["HubArn"] != resp["HubArn"] {
		t.Errorf("DescribeSecurityHubV2 mismatch")
	}

	callOK(t, h, "DisableSecurityHubV2", nil)
	if code, _ := callStatus(t, h, "DescribeSecurityHubV2", nil); code != http.StatusBadRequest {
		t.Errorf("DescribeSecurityHubV2 after disable should fail, got %d", code)
	}
}

// ── Standards ────────────────────────────────────────────────────────────────

func TestStandardsLifecycle(t *testing.T) {
	h := newGateway(t)

	stds := callOK(t, h, "DescribeStandards", nil)
	if list, ok := stds["Standards"].([]any); !ok || len(list) == 0 {
		t.Fatalf("DescribeStandards: expected non-empty list, got %+v", stds)
	}

	// Enable a standard.
	enable := callOK(t, h, "BatchEnableStandards", map[string]any{
		"StandardsSubscriptionRequests": []map[string]any{
			{
				"StandardsArn":   "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0",
				"StandardsInput": map[string]any{"input1": "value1"},
			},
		},
	})
	subs := enable["StandardsSubscriptions"].([]any)
	if len(subs) != 1 {
		t.Fatalf("BatchEnableStandards: expected 1 subscription, got %d", len(subs))
	}
	sub := subs[0].(map[string]any)
	subArn := sub["StandardsSubscriptionArn"].(string)
	if subArn == "" {
		t.Fatalf("BatchEnableStandards: missing subscription ARN")
	}

	// List enabled.
	enabled := callOK(t, h, "GetEnabledStandards", nil)
	if list := enabled["StandardsSubscriptions"].([]any); len(list) != 1 {
		t.Fatalf("GetEnabledStandards: expected 1 subscription, got %d", len(list))
	}

	// Disable.
	callOK(t, h, "BatchDisableStandards", map[string]any{
		"StandardsSubscriptionArns": []string{subArn},
	})
	enabled = callOK(t, h, "GetEnabledStandards", nil)
	if list := enabled["StandardsSubscriptions"].([]any); len(list) != 0 {
		t.Fatalf("GetEnabledStandards after disable: expected 0, got %d", len(list))
	}

	// DescribeStandardsControls returns at least the static stub.
	ctrls := callOK(t, h, "DescribeStandardsControls", map[string]any{
		"StandardsSubscriptionArn": "arn:aws:securityhub:::ruleset/test/v/1.0",
	})
	if ctrls["Controls"] == nil {
		t.Errorf("DescribeStandardsControls: expected Controls field")
	}

	// UpdateStandardsControl needs an ARN input.
	callOK(t, h, "UpdateStandardsControl", map[string]any{
		"StandardsControlArn": "arn:aws:securityhub:::standards-control/CIS.1.1",
		"ControlStatus":       "DISABLED",
	})
}

// ── Security controls ────────────────────────────────────────────────────────

func TestSecurityControls(t *testing.T) {
	h := newGateway(t)

	defs := callOK(t, h, "ListSecurityControlDefinitions", nil)
	if list, ok := defs["SecurityControlDefinitions"].([]any); !ok || len(list) == 0 {
		t.Fatalf("ListSecurityControlDefinitions: expected items, got %+v", defs)
	}

	// Resolve a single definition.
	one := callOK(t, h, "GetSecurityControlDefinition", map[string]any{
		"SecurityControlId": "IAM.1",
	})
	if one["SecurityControlDefinition"] == nil {
		t.Errorf("GetSecurityControlDefinition: expected definition")
	}

	// Unknown id should fail.
	if code, _ := callStatus(t, h, "GetSecurityControlDefinition", map[string]any{"SecurityControlId": "NOPE"}); code != http.StatusBadRequest {
		t.Errorf("GetSecurityControlDefinition: expected 400 for unknown id, got %d", code)
	}

	// Batch get.
	batch := callOK(t, h, "BatchGetSecurityControls", map[string]any{
		"SecurityControlIds": []string{"IAM.1", "S3.1", "MISSING.1"},
	})
	if list := batch["SecurityControls"].([]any); len(list) != 2 {
		t.Errorf("BatchGetSecurityControls: expected 2 found, got %d", len(list))
	}
	if list := batch["UnprocessedIds"].([]any); len(list) != 1 {
		t.Errorf("BatchGetSecurityControls: expected 1 unprocessed, got %d", len(list))
	}

	// Update.
	callOK(t, h, "UpdateSecurityControl", map[string]any{
		"SecurityControlId":     "IAM.1",
		"SecurityControlStatus": "DISABLED",
	})

	// Batch get associations.
	callOK(t, h, "BatchGetStandardsControlAssociations", map[string]any{
		"StandardsControlAssociationIds": []map[string]any{
			{"StandardsArn": "arn:aws:securityhub:::ruleset/test", "SecurityControlId": "IAM.1"},
		},
	})

	// Batch update associations.
	callOK(t, h, "BatchUpdateStandardsControlAssociations", map[string]any{
		"StandardsControlAssociationUpdates": []map[string]any{
			{"StandardsArn": "arn:aws:securityhub:::ruleset/test", "SecurityControlId": "IAM.1", "AssociationStatus": "ENABLED"},
		},
	})

	// List.
	callOK(t, h, "ListStandardsControlAssociations", nil)
}

// ── Products ─────────────────────────────────────────────────────────────────

func TestProductsLifecycle(t *testing.T) {
	h := newGateway(t)

	prods := callOK(t, h, "DescribeProducts", nil)
	if list, ok := prods["Products"].([]any); !ok || len(list) == 0 {
		t.Fatalf("DescribeProducts: expected products, got %+v", prods)
	}

	v2 := callOK(t, h, "DescribeProductsV2", nil)
	if v2["ProductsV2"] == nil {
		t.Errorf("DescribeProductsV2: expected ProductsV2")
	}

	// Enable and then disable a product.
	enable := callOK(t, h, "EnableImportFindingsForProduct", map[string]any{
		"ProductArn": "arn:aws:securityhub:us-east-1::product/aws/guardduty",
	})
	subArn := enable["ProductSubscriptionArn"].(string)
	if subArn == "" {
		t.Fatalf("EnableImportFindingsForProduct: missing subscription arn")
	}

	// Re-enabling the same product should conflict.
	if code, _ := callStatus(t, h, "EnableImportFindingsForProduct", map[string]any{
		"ProductArn": "arn:aws:securityhub:us-east-1::product/aws/guardduty",
	}); code != http.StatusConflict {
		t.Errorf("EnableImportFindingsForProduct twice: expected 409, got %d", code)
	}

	listed := callOK(t, h, "ListEnabledProductsForImport", nil)
	if list := listed["ProductSubscriptions"].([]any); len(list) != 1 {
		t.Errorf("ListEnabledProductsForImport: expected 1, got %d", len(list))
	}

	callOK(t, h, "DisableImportFindingsForProduct", map[string]any{
		"ProductSubscriptionArn": subArn,
	})

	listed = callOK(t, h, "ListEnabledProductsForImport", nil)
	if list := listed["ProductSubscriptions"].([]any); len(list) != 0 {
		t.Errorf("ListEnabledProductsForImport after disable: expected 0, got %d", len(list))
	}
}

// ── Findings ─────────────────────────────────────────────────────────────────

func TestFindingsLifecycle(t *testing.T) {
	h := newGateway(t)

	finding := map[string]any{
		"Id":            "finding-1",
		"SchemaVersion": "2018-10-08",
		"ProductArn":    "arn:aws:securityhub:us-east-1::product/aws/guardduty",
		"GeneratorId":   "gen-1",
		"AwsAccountId":  "000000000000",
		"Types":         []string{"Software and Configuration Checks/Vulnerabilities/CVE"},
		"Title":         "Mock finding",
		"Description":   "A mock security finding for testing",
		"WorkflowState": "NEW",
		"RecordState":   "ACTIVE",
	}

	imp := callOK(t, h, "BatchImportFindings", map[string]any{
		"Findings": []map[string]any{finding},
	})
	if imp["SuccessCount"].(float64) != 1 {
		t.Errorf("BatchImportFindings: expected SuccessCount=1, got %v", imp["SuccessCount"])
	}

	// Import without Id should fail.
	imp = callOK(t, h, "BatchImportFindings", map[string]any{
		"Findings": []map[string]any{{"Title": "no id"}},
	})
	if imp["FailedCount"].(float64) != 1 {
		t.Errorf("BatchImportFindings: expected FailedCount=1 for missing id, got %v", imp["FailedCount"])
	}

	// Get all findings.
	got := callOK(t, h, "GetFindings", nil)
	list := got["Findings"].([]any)
	if len(list) != 1 {
		t.Errorf("GetFindings: expected 1 finding, got %d", len(list))
	}

	// V2 variant.
	gotV2 := callOK(t, h, "GetFindingsV2", nil)
	if list := gotV2["Findings"].([]any); len(list) != 1 {
		t.Errorf("GetFindingsV2: expected 1 finding, got %d", len(list))
	}

	// Filter by ID.
	filtered := callOK(t, h, "GetFindings", map[string]any{
		"Filters": map[string]any{
			"Id": []map[string]any{{"Value": "finding-1", "Comparison": "EQUALS"}},
		},
	})
	if list := filtered["Findings"].([]any); len(list) != 1 {
		t.Errorf("GetFindings filtered: expected 1, got %d", len(list))
	}

	noMatch := callOK(t, h, "GetFindings", map[string]any{
		"Filters": map[string]any{
			"Id": []map[string]any{{"Value": "nope", "Comparison": "EQUALS"}},
		},
	})
	if list := noMatch["Findings"].([]any); len(list) != 0 {
		t.Errorf("GetFindings no-match: expected 0, got %d", len(list))
	}

	// BatchUpdateFindings updates fields by identifier.
	upd := callOK(t, h, "BatchUpdateFindings", map[string]any{
		"FindingIdentifiers": []map[string]any{{"Id": "finding-1", "ProductArn": "arn:aws:securityhub:us-east-1::product/aws/guardduty"}},
		"Workflow":           map[string]any{"Status": "RESOLVED"},
		"Confidence":         99,
	})
	if list := upd["ProcessedFindings"].([]any); len(list) != 1 {
		t.Errorf("BatchUpdateFindings: expected 1 processed, got %d", len(list))
	}

	// V2 update is identical mock behavior.
	callOK(t, h, "BatchUpdateFindingsV2", map[string]any{
		"FindingIdentifiers": []map[string]any{{"Id": "finding-1", "ProductArn": "arn:aws:securityhub:us-east-1::product/aws/guardduty"}},
		"Workflow":           map[string]any{"Status": "NEW"},
	})

	// UpdateFindings (legacy bulk update by filter).
	callOK(t, h, "UpdateFindings", map[string]any{
		"Filters": map[string]any{
			"Id": []map[string]any{{"Value": "finding-1", "Comparison": "EQUALS"}},
		},
		"RecordState": "ARCHIVED",
	})

	// Stats / history.
	stats := callOK(t, h, "GetFindingStatisticsV2", nil)
	if stats["GroupByResults"] == nil {
		t.Errorf("GetFindingStatisticsV2: expected GroupByResults")
	}
	callOK(t, h, "GetFindingHistory", map[string]any{
		"FindingIdentifier": map[string]any{"Id": "finding-1"},
	})
	callOK(t, h, "GetFindingsTrendsV2", nil)
}

// ── Insights ─────────────────────────────────────────────────────────────────

func TestInsightsLifecycle(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateInsight", map[string]any{
		"Name":             "Critical findings",
		"Filters":          map[string]any{"SeverityLabel": []map[string]any{{"Value": "CRITICAL", "Comparison": "EQUALS"}}},
		"GroupByAttribute": "ResourceId",
	})
	arn := create["InsightArn"].(string)
	if arn == "" {
		t.Fatalf("CreateInsight: expected InsightArn")
	}

	// Missing name fails.
	if code, _ := callStatus(t, h, "CreateInsight", map[string]any{"GroupByAttribute": "ResourceId"}); code != http.StatusBadRequest {
		t.Errorf("CreateInsight without name: expected 400, got %d", code)
	}

	// Update.
	callOK(t, h, "UpdateInsight", map[string]any{
		"InsightArn":       arn,
		"Name":             "Renamed insight",
		"GroupByAttribute": "Type",
	})

	// Get.
	got := callOK(t, h, "GetInsights", map[string]any{"InsightArns": []string{arn}})
	list := got["Insights"].([]any)
	if len(list) != 1 {
		t.Fatalf("GetInsights: expected 1, got %d", len(list))
	}
	if list[0].(map[string]any)["Name"] != "Renamed insight" {
		t.Errorf("UpdateInsight: name not updated")
	}

	// Get all.
	all := callOK(t, h, "GetInsights", nil)
	if list := all["Insights"].([]any); len(list) != 1 {
		t.Errorf("GetInsights all: expected 1, got %d", len(list))
	}

	// Results.
	res := callOK(t, h, "GetInsightResults", map[string]any{"InsightArn": arn})
	if res["InsightResults"] == nil {
		t.Errorf("GetInsightResults: expected InsightResults")
	}

	// Delete.
	callOK(t, h, "DeleteInsight", map[string]any{"InsightArn": arn})
	if code, _ := callStatus(t, h, "DeleteInsight", map[string]any{"InsightArn": arn}); code != http.StatusBadRequest {
		t.Errorf("DeleteInsight twice: expected 400, got %d", code)
	}
}

// ── Action targets ───────────────────────────────────────────────────────────

func TestActionTargets(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateActionTarget", map[string]any{
		"Name":        "Send to SNS",
		"Description": "Forward findings to SNS topic",
		"Id":          "send-to-sns",
	})
	arn := create["ActionTargetArn"].(string)
	if arn == "" {
		t.Fatalf("CreateActionTarget: missing ARN")
	}

	// Conflict on duplicate.
	if code, _ := callStatus(t, h, "CreateActionTarget", map[string]any{
		"Name":        "Send to SNS",
		"Description": "dup",
		"Id":          "send-to-sns",
	}); code != http.StatusConflict {
		t.Errorf("CreateActionTarget duplicate: expected 409, got %d", code)
	}

	callOK(t, h, "UpdateActionTarget", map[string]any{
		"ActionTargetArn": arn,
		"Name":            "Updated",
	})

	desc := callOK(t, h, "DescribeActionTargets", map[string]any{"ActionTargetArns": []string{arn}})
	list := desc["ActionTargets"].([]any)
	if len(list) != 1 {
		t.Fatalf("DescribeActionTargets: expected 1, got %d", len(list))
	}
	if list[0].(map[string]any)["Name"] != "Updated" {
		t.Errorf("UpdateActionTarget: name not updated")
	}

	callOK(t, h, "DeleteActionTarget", map[string]any{"ActionTargetArn": arn})
	if code, _ := callStatus(t, h, "DeleteActionTarget", map[string]any{"ActionTargetArn": arn}); code != http.StatusBadRequest {
		t.Errorf("DeleteActionTarget twice: expected 400, got %d", code)
	}
}

// ── Members & invitations ────────────────────────────────────────────────────

func TestMembersAndInvitations(t *testing.T) {
	h := newGateway(t)

	// Create.
	callOK(t, h, "CreateMembers", map[string]any{
		"AccountDetails": []map[string]any{
			{"AccountId": "111111111111", "Email": "a@example.com"},
			{"AccountId": "222222222222", "Email": "b@example.com"},
		},
	})

	// List.
	list := callOK(t, h, "ListMembers", nil)
	members := list["Members"].([]any)
	if len(members) != 2 {
		t.Errorf("ListMembers: expected 2, got %d", len(members))
	}

	// Get one + missing.
	got := callOK(t, h, "GetMembers", map[string]any{
		"AccountIds": []string{"111111111111", "999999999999"},
	})
	if found := got["Members"].([]any); len(found) != 1 {
		t.Errorf("GetMembers: expected 1 found, got %d", len(found))
	}
	if unp := got["UnprocessedAccounts"].([]any); len(unp) != 1 {
		t.Errorf("GetMembers: expected 1 unprocessed, got %d", len(unp))
	}

	// Invite.
	callOK(t, h, "InviteMembers", map[string]any{"AccountIds": []string{"111111111111"}})

	// Disassociate.
	callOK(t, h, "DisassociateMembers", map[string]any{"AccountIds": []string{"111111111111"}})

	// Delete.
	callOK(t, h, "DeleteMembers", map[string]any{"AccountIds": []string{"111111111111"}})
	list = callOK(t, h, "ListMembers", nil)
	if members := list["Members"].([]any); len(members) != 1 {
		t.Errorf("ListMembers after delete: expected 1, got %d", len(members))
	}

	// Invitations and counts.
	callOK(t, h, "ListInvitations", nil)
	cnt := callOK(t, h, "GetInvitationsCount", nil)
	if cnt["InvitationsCount"].(float64) != 0 {
		t.Errorf("GetInvitationsCount: expected 0, got %v", cnt["InvitationsCount"])
	}

	// Decline & delete invitations should not panic.
	callOK(t, h, "DeclineInvitations", map[string]any{"AccountIds": []string{"333"}})
	callOK(t, h, "DeleteInvitations", map[string]any{"AccountIds": []string{"333"}})
}

func TestAdministratorAccount(t *testing.T) {
	h := newGateway(t)

	callOK(t, h, "AcceptAdministratorInvitation", map[string]any{
		"AdministratorId": "111111111111",
		"InvitationId":    "abc123",
	})

	got := callOK(t, h, "GetAdministratorAccount", nil)
	admin := got["Administrator"].(map[string]any)
	if admin["AccountId"] != "111111111111" {
		t.Errorf("GetAdministratorAccount: expected 111111111111, got %v", admin["AccountId"])
	}

	callOK(t, h, "DisassociateFromAdministratorAccount", nil)
	got = callOK(t, h, "GetAdministratorAccount", nil)
	admin = got["Administrator"].(map[string]any)
	if admin["AccountId"] != "" {
		t.Errorf("GetAdministratorAccount after disassociate: expected empty, got %v", admin["AccountId"])
	}

	// Legacy master endpoints.
	callOK(t, h, "AcceptInvitation", map[string]any{
		"MasterId":     "222222222222",
		"InvitationId": "xyz",
	})
	master := callOK(t, h, "GetMasterAccount", nil)
	if master["Master"].(map[string]any)["AccountId"] != "222222222222" {
		t.Errorf("GetMasterAccount mismatch")
	}
	callOK(t, h, "DisassociateFromMasterAccount", nil)
}

// ── Organization ─────────────────────────────────────────────────────────────

func TestOrganization(t *testing.T) {
	h := newGateway(t)

	// EnableOrganizationAdminAccount needs an account id.
	if code, _ := callStatus(t, h, "EnableOrganizationAdminAccount", nil); code != http.StatusBadRequest {
		t.Errorf("EnableOrganizationAdminAccount empty: expected 400, got %d", code)
	}

	callOK(t, h, "EnableOrganizationAdminAccount", map[string]any{"AdminAccountId": "111111111111"})
	list := callOK(t, h, "ListOrganizationAdminAccounts", nil)
	admins := list["AdminAccounts"].([]any)
	if len(admins) != 1 {
		t.Fatalf("ListOrganizationAdminAccounts: expected 1, got %d", len(admins))
	}

	cfg := callOK(t, h, "DescribeOrganizationConfiguration", nil)
	if cfg["AutoEnable"] == nil {
		t.Errorf("DescribeOrganizationConfiguration: expected AutoEnable field")
	}

	callOK(t, h, "UpdateOrganizationConfiguration", map[string]any{
		"AutoEnable":          true,
		"AutoEnableStandards": "DEFAULT",
	})
	cfg = callOK(t, h, "DescribeOrganizationConfiguration", nil)
	if cfg["AutoEnable"] != true {
		t.Errorf("UpdateOrganizationConfiguration: AutoEnable not set")
	}

	callOK(t, h, "DisableOrganizationAdminAccount", nil)
	if code, _ := callStatus(t, h, "DisableOrganizationAdminAccount", nil); code != http.StatusBadRequest {
		t.Errorf("DisableOrganizationAdminAccount when none: expected 400, got %d", code)
	}
}

// ── Finding aggregators ──────────────────────────────────────────────────────

func TestFindingAggregators(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateFindingAggregator", map[string]any{
		"RegionLinkingMode": "SPECIFIED_REGIONS",
		"Regions":           []string{"us-east-1", "us-west-2"},
	})
	arn := create["FindingAggregatorArn"].(string)
	if arn == "" {
		t.Fatalf("CreateFindingAggregator: missing ARN")
	}

	got := callOK(t, h, "GetFindingAggregator", map[string]any{"FindingAggregatorArn": arn})
	if got["FindingAggregatorArn"] != arn {
		t.Errorf("GetFindingAggregator: ARN mismatch")
	}

	upd := callOK(t, h, "UpdateFindingAggregator", map[string]any{
		"FindingAggregatorArn": arn,
		"RegionLinkingMode":    "ALL_REGIONS",
	})
	if upd["RegionLinkingMode"] != "ALL_REGIONS" {
		t.Errorf("UpdateFindingAggregator: mode not updated")
	}

	list := callOK(t, h, "ListFindingAggregators", nil)
	if aggs := list["FindingAggregators"].([]any); len(aggs) != 1 {
		t.Errorf("ListFindingAggregators: expected 1, got %d", len(aggs))
	}

	callOK(t, h, "DeleteFindingAggregator", map[string]any{"FindingAggregatorArn": arn})
	if code, _ := callStatus(t, h, "GetFindingAggregator", map[string]any{"FindingAggregatorArn": arn}); code != http.StatusBadRequest {
		t.Errorf("GetFindingAggregator after delete: expected 400, got %d", code)
	}
}

// ── Aggregators V2 ───────────────────────────────────────────────────────────

func TestAggregatorsV2(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateAggregatorV2", map[string]any{
		"RegionLinkingMode": "ALL_REGIONS",
		"LinkedRegions":     []string{"us-east-1"},
	})
	arn := create["AggregatorV2Arn"].(string)
	if arn == "" {
		t.Fatalf("CreateAggregatorV2: missing ARN")
	}

	callOK(t, h, "GetAggregatorV2", map[string]any{"AggregatorV2Arn": arn})

	upd := callOK(t, h, "UpdateAggregatorV2", map[string]any{
		"AggregatorV2Arn":   arn,
		"RegionLinkingMode": "SPECIFIED_REGIONS",
		"LinkedRegions":     []string{"us-west-2"},
	})
	if upd["RegionLinkingMode"] != "SPECIFIED_REGIONS" {
		t.Errorf("UpdateAggregatorV2: mode not updated")
	}

	list := callOK(t, h, "ListAggregatorsV2", nil)
	if aggs := list["AggregatorsV2"].([]any); len(aggs) != 1 {
		t.Errorf("ListAggregatorsV2: expected 1, got %d", len(aggs))
	}

	callOK(t, h, "DeleteAggregatorV2", map[string]any{"AggregatorV2Arn": arn})
	if code, _ := callStatus(t, h, "GetAggregatorV2", map[string]any{"AggregatorV2Arn": arn}); code != http.StatusBadRequest {
		t.Errorf("GetAggregatorV2 after delete: expected 400, got %d", code)
	}
}

// ── Automation rules ─────────────────────────────────────────────────────────

func TestAutomationRules(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateAutomationRule", map[string]any{
		"RuleName":    "Mute info findings",
		"Description": "Suppress informational findings",
		"RuleStatus":  "ENABLED",
		"RuleOrder":   1,
		"IsTerminal":  false,
		"Criteria":    map[string]any{"SeverityLabel": []map[string]any{{"Value": "INFORMATIONAL"}}},
		"Actions": []map[string]any{
			{"Type": "FINDING_FIELDS_UPDATE"},
		},
	})
	arn := create["RuleArn"].(string)
	if arn == "" {
		t.Fatalf("CreateAutomationRule: missing RuleArn")
	}

	// Missing name should fail.
	if code, _ := callStatus(t, h, "CreateAutomationRule", nil); code != http.StatusBadRequest {
		t.Errorf("CreateAutomationRule no name: expected 400, got %d", code)
	}

	// Batch get.
	get := callOK(t, h, "BatchGetAutomationRules", map[string]any{"AutomationRulesArns": []string{arn, "missing"}})
	if rules := get["Rules"].([]any); len(rules) != 1 {
		t.Errorf("BatchGetAutomationRules: expected 1 rule, got %d", len(rules))
	}
	if unp := get["UnprocessedAutomationRules"].([]any); len(unp) != 1 {
		t.Errorf("BatchGetAutomationRules: expected 1 unprocessed, got %d", len(unp))
	}

	// Update.
	upd := callOK(t, h, "BatchUpdateAutomationRules", map[string]any{
		"UpdateAutomationRulesRequestItems": []map[string]any{
			{"RuleArn": arn, "RuleName": "Renamed", "RuleStatus": "DISABLED", "RuleOrder": 5},
		},
	})
	if proc := upd["ProcessedAutomationRules"].([]any); len(proc) != 1 {
		t.Errorf("BatchUpdateAutomationRules: expected 1 processed, got %d", len(proc))
	}

	// List.
	list := callOK(t, h, "ListAutomationRules", nil)
	if md := list["AutomationRulesMetadata"].([]any); len(md) != 1 {
		t.Errorf("ListAutomationRules: expected 1, got %d", len(md))
	}

	// Delete.
	del := callOK(t, h, "BatchDeleteAutomationRules", map[string]any{"AutomationRulesArns": []string{arn}})
	if proc := del["ProcessedAutomationRules"].([]any); len(proc) != 1 {
		t.Errorf("BatchDeleteAutomationRules: expected 1 processed, got %d", len(proc))
	}
}

func TestAutomationRulesV2(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateAutomationRuleV2", map[string]any{
		"RuleName":    "V2 rule",
		"Description": "test v2",
		"RuleStatus":  "ENABLED",
		"RuleOrder":   1.5,
		"Criteria":    map[string]any{},
		"Actions":     []map[string]any{},
	})
	id := create["RuleId"].(string)
	if id == "" {
		t.Fatalf("CreateAutomationRuleV2: missing RuleId")
	}

	got := callOK(t, h, "GetAutomationRuleV2", map[string]any{"Identifier": id})
	if got["RuleId"] != id {
		t.Errorf("GetAutomationRuleV2: id mismatch")
	}

	upd := callOK(t, h, "UpdateAutomationRuleV2", map[string]any{
		"Identifier":  id,
		"RuleName":    "Renamed v2",
		"RuleStatus":  "DISABLED",
	})
	if upd["RuleName"] != "Renamed v2" {
		t.Errorf("UpdateAutomationRuleV2: name not updated")
	}

	list := callOK(t, h, "ListAutomationRulesV2", nil)
	if rules := list["Rules"].([]any); len(rules) != 1 {
		t.Errorf("ListAutomationRulesV2: expected 1, got %d", len(rules))
	}

	callOK(t, h, "DeleteAutomationRuleV2", map[string]any{"Identifier": id})
	if code, _ := callStatus(t, h, "GetAutomationRuleV2", map[string]any{"Identifier": id}); code != http.StatusBadRequest {
		t.Errorf("GetAutomationRuleV2 after delete: expected 400, got %d", code)
	}
}

// ── Configuration policies ───────────────────────────────────────────────────

func TestConfigurationPolicies(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateConfigurationPolicy", map[string]any{
		"Name":        "default-config",
		"Description": "Baseline policy",
		"ConfigurationPolicy": map[string]any{
			"SecurityHub": map[string]any{
				"ServiceEnabled":            true,
				"EnabledStandardIdentifiers": []string{"arn:aws:securityhub:::ruleset/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	})
	id := create["Id"].(string)
	arn := create["Arn"].(string)
	if id == "" || arn == "" {
		t.Fatalf("CreateConfigurationPolicy: missing id/arn")
	}

	// Missing name.
	if code, _ := callStatus(t, h, "CreateConfigurationPolicy", nil); code != http.StatusBadRequest {
		t.Errorf("CreateConfigurationPolicy no name: expected 400, got %d", code)
	}

	// Get by id.
	got := callOK(t, h, "GetConfigurationPolicy", map[string]any{"Identifier": id})
	if got["Id"] != id {
		t.Errorf("GetConfigurationPolicy: id mismatch")
	}

	// Get by ARN.
	got = callOK(t, h, "GetConfigurationPolicy", map[string]any{"Identifier": arn})
	if got["Arn"] != arn {
		t.Errorf("GetConfigurationPolicy by ARN: mismatch")
	}

	// Update.
	upd := callOK(t, h, "UpdateConfigurationPolicy", map[string]any{
		"Identifier":  id,
		"Description": "Updated description",
	})
	if upd["Description"] != "Updated description" {
		t.Errorf("UpdateConfigurationPolicy: description not updated")
	}

	// List.
	list := callOK(t, h, "ListConfigurationPolicies", nil)
	if pols := list["ConfigurationPolicySummaries"].([]any); len(pols) != 1 {
		t.Errorf("ListConfigurationPolicies: expected 1, got %d", len(pols))
	}

	// Associate.
	assoc := callOK(t, h, "StartConfigurationPolicyAssociation", map[string]any{
		"ConfigurationPolicyIdentifier": id,
		"Target": map[string]any{"AccountId": "111111111111"},
	})
	if assoc["TargetId"] != "111111111111" {
		t.Errorf("StartConfigurationPolicyAssociation: target mismatch")
	}

	getAssoc := callOK(t, h, "GetConfigurationPolicyAssociation", map[string]any{
		"Target": map[string]any{"AccountId": "111111111111"},
	})
	if getAssoc["ConfigurationPolicyId"] != id {
		t.Errorf("GetConfigurationPolicyAssociation: policy id mismatch")
	}

	batchGet := callOK(t, h, "BatchGetConfigurationPolicyAssociations", map[string]any{
		"ConfigurationPolicyAssociationIdentifiers": []map[string]any{
			{"AccountId": "111111111111"},
			{"AccountId": "missing"},
		},
	})
	if found := batchGet["ConfigurationPolicyAssociations"].([]any); len(found) != 1 {
		t.Errorf("BatchGetConfigurationPolicyAssociations: expected 1 found, got %d", len(found))
	}

	listAssoc := callOK(t, h, "ListConfigurationPolicyAssociations", nil)
	if list := listAssoc["ConfigurationPolicyAssociationSummaries"].([]any); len(list) != 1 {
		t.Errorf("ListConfigurationPolicyAssociations: expected 1, got %d", len(list))
	}

	// Disassociate.
	callOK(t, h, "StartConfigurationPolicyDisassociation", map[string]any{
		"ConfigurationPolicyIdentifier": id,
		"Target": map[string]any{"AccountId": "111111111111"},
	})

	// Delete.
	callOK(t, h, "DeleteConfigurationPolicy", map[string]any{"Identifier": id})
	if code, _ := callStatus(t, h, "GetConfigurationPolicy", map[string]any{"Identifier": id}); code != http.StatusBadRequest {
		t.Errorf("GetConfigurationPolicy after delete: expected 400, got %d", code)
	}
}

// ── Connectors V2 ────────────────────────────────────────────────────────────

func TestConnectorsV2(t *testing.T) {
	h := newGateway(t)

	create := callOK(t, h, "CreateConnectorV2", map[string]any{
		"Name":        "jira-connector",
		"Description": "Jira ticketing integration",
		"Provider":    "JIRA_CLOUD",
		"KmsKeyArn":   "arn:aws:kms:us-east-1:000000000000:key/abc",
	})
	id := create["ConnectorId"].(string)
	if id == "" {
		t.Fatalf("CreateConnectorV2: missing ConnectorId")
	}

	got := callOK(t, h, "GetConnectorV2", map[string]any{"ConnectorId": id})
	if got["ConnectorId"] != id {
		t.Errorf("GetConnectorV2: id mismatch")
	}

	upd := callOK(t, h, "UpdateConnectorV2", map[string]any{
		"ConnectorId": id,
		"Description": "Updated",
	})
	if upd["Description"] != "Updated" {
		t.Errorf("UpdateConnectorV2: not updated")
	}

	list := callOK(t, h, "ListConnectorsV2", nil)
	if conns := list["Connectors"].([]any); len(conns) != 1 {
		t.Errorf("ListConnectorsV2: expected 1, got %d", len(conns))
	}

	// Register returns the connector.
	reg := callOK(t, h, "RegisterConnectorV2", map[string]any{"ConnectorId": id})
	if reg["ConnectorId"] != id {
		t.Errorf("RegisterConnectorV2: id mismatch")
	}

	// Create a ticket using the connector.
	ticket := callOK(t, h, "CreateTicketV2", map[string]any{"ConnectorId": id})
	if ticket["TicketId"] == "" {
		t.Errorf("CreateTicketV2: expected TicketId")
	}

	// CreateTicketV2 without ConnectorId should fail.
	if code, _ := callStatus(t, h, "CreateTicketV2", nil); code != http.StatusBadRequest {
		t.Errorf("CreateTicketV2 missing connector: expected 400, got %d", code)
	}

	// Delete.
	callOK(t, h, "DeleteConnectorV2", map[string]any{"ConnectorId": id})
	if code, _ := callStatus(t, h, "GetConnectorV2", map[string]any{"ConnectorId": id}); code != http.StatusBadRequest {
		t.Errorf("GetConnectorV2 after delete: expected 400, got %d", code)
	}
}

// ── Resources V2 ─────────────────────────────────────────────────────────────

func TestResourcesV2(t *testing.T) {
	h := newGateway(t)

	r := callOK(t, h, "GetResourcesV2", nil)
	if r["Resources"] == nil {
		t.Errorf("GetResourcesV2: expected Resources")
	}
	stats := callOK(t, h, "GetResourcesStatisticsV2", nil)
	if stats["GroupByResults"] == nil {
		t.Errorf("GetResourcesStatisticsV2: expected GroupByResults")
	}
	tr := callOK(t, h, "GetResourcesTrendsV2", nil)
	if tr["Trends"] == nil {
		t.Errorf("GetResourcesTrendsV2: expected Trends")
	}
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func TestTags(t *testing.T) {
	h := newGateway(t)

	arn := "arn:aws:securityhub:us-east-1:000000000000:hub/default"

	callOK(t, h, "TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags":        map[string]any{"env": "dev", "team": "platform"},
	})

	listed := callOK(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	tags, _ := listed["Tags"].(map[string]any)
	if tags["env"] != "dev" || tags["team"] != "platform" {
		t.Errorf("ListTagsForResource: missing tags, got %+v", tags)
	}

	callOK(t, h, "UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"team"},
	})
	listed = callOK(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	tags = listed["Tags"].(map[string]any)
	if _, has := tags["team"]; has {
		t.Errorf("UntagResource: team tag should be removed")
	}
	if tags["env"] != "dev" {
		t.Errorf("UntagResource: env tag should remain")
	}

	// Missing ARN should fail.
	if code, _ := callStatus(t, h, "TagResource", nil); code != http.StatusBadRequest {
		t.Errorf("TagResource missing arn: expected 400, got %d", code)
	}
	if code, _ := callStatus(t, h, "ListTagsForResource", nil); code != http.StatusBadRequest {
		t.Errorf("ListTagsForResource missing arn: expected 400, got %d", code)
	}
}

// ── Reset ────────────────────────────────────────────────────────────────────

func TestStoreReset(t *testing.T) {
	cfg := config.Default()
	store := svc.NewStore(cfg.AccountID, cfg.Region)

	// Enable hub then reset.
	if _, err := store.EnableHub(true, "SECURITY_CONTROL", nil); err != nil {
		t.Fatalf("EnableHub: %v", err)
	}
	store.Reset()
	if _, err := store.GetHub(); err == nil {
		t.Errorf("After Reset, hub should not exist")
	}
}

// ── Smoke test: every action returns 200 with a sensible body ────────────────

// TestAllActionsRespond exercises every registered action with minimal/empty input
// to ensure no handler panics and that we return JSON for every route. Actions
// that require setup are covered above; here we just verify the dispatch wires.
func TestAllActionsRespond(t *testing.T) {
	h := newGateway(t)

	// Pre-seed: enable hub, create a finding, create policy, etc. so subsequent
	// queries have real data to operate on.
	callOK(t, h, "EnableSecurityHub", nil)
	callOK(t, h, "BatchImportFindings", map[string]any{
		"Findings": []map[string]any{{"Id": "smoke-1", "Title": "smoke"}},
	})

	// These actions are tolerant of empty input.
	tolerant := []string{
		"DescribeHub", "DescribeStandards", "DescribeProducts", "DescribeProductsV2",
		"GetEnabledStandards", "ListEnabledProductsForImport", "ListInvitations",
		"GetInvitationsCount", "ListMembers", "ListOrganizationAdminAccounts",
		"DescribeOrganizationConfiguration", "ListAutomationRules", "ListAutomationRulesV2",
		"ListConfigurationPolicies", "ListConfigurationPolicyAssociations",
		"ListConnectorsV2", "ListAggregatorsV2", "ListFindingAggregators",
		"GetFindings", "GetFindingsV2", "GetInsights", "GetFindingStatisticsV2",
		"GetFindingsTrendsV2", "GetResourcesV2", "GetResourcesStatisticsV2",
		"GetResourcesTrendsV2", "ListSecurityControlDefinitions",
		"ListStandardsControlAssociations", "GetAdministratorAccount", "GetMasterAccount",
		"DisassociateFromAdministratorAccount", "DisassociateFromMasterAccount",
	}
	for _, action := range tolerant {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, svcReq(t, action, nil))
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d\nbody: %s", action, w.Code, w.Body.String())
		}
	}
}
