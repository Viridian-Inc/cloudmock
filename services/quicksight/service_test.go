package quicksight_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/quicksight"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

// call invokes a quicksight action, asserting the expected status code, and
// returns the decoded response body.
func call(t *testing.T, h http.Handler, action string, body any, wantStatus int) map[string]any {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal %s: %v", action, err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "quicksight."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/quicksight/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != wantStatus {
		t.Fatalf("%s: want status %d, got %d\nbody: %s", action, wantStatus, w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: failed to decode body: %v\nbody: %s", action, err, w.Body.String())
	}
	return out
}

// callOK is a shortcut for asserting 200 OK.
func callOK(t *testing.T, h http.Handler, action string, body any) map[string]any {
	t.Helper()
	return call(t, h, action, body, http.StatusOK)
}

// callCreated is a shortcut for asserting 201 Created.
func callCreated(t *testing.T, h http.Handler, action string, body any) map[string]any {
	t.Helper()
	return call(t, h, action, body, http.StatusCreated)
}

// ── User lifecycle ───────────────────────────────────────────────────────────

func TestUserLifecycle(t *testing.T) {
	h := newGateway(t)

	// Register a user.
	resp := callCreated(t, h, "RegisterUser", map[string]any{
		"AwsAccountId": "000000000000",
		"Namespace":    "default",
		"UserName":     "alice",
		"Email":        "alice@example.com",
		"UserRole":     "AUTHOR",
		"IdentityType": "QUICKSIGHT",
	})
	user, _ := resp["User"].(map[string]any)
	if user["UserName"] != "alice" {
		t.Fatalf("RegisterUser: expected UserName alice, got %v", user["UserName"])
	}

	// Describe.
	resp = callOK(t, h, "DescribeUser", map[string]any{
		"Namespace": "default",
		"UserName":  "alice",
	})
	user, _ = resp["User"].(map[string]any)
	if user["Email"] != "alice@example.com" {
		t.Fatalf("DescribeUser: unexpected email %v", user["Email"])
	}

	// Update.
	resp = callOK(t, h, "UpdateUser", map[string]any{
		"Namespace": "default",
		"UserName":  "alice",
		"Email":     "alice2@example.com",
		"Role":      "ADMIN",
	})
	user, _ = resp["User"].(map[string]any)
	if user["Email"] != "alice2@example.com" || user["Role"] != "ADMIN" {
		t.Fatalf("UpdateUser: unexpected user %v", user)
	}

	// List.
	resp = callOK(t, h, "ListUsers", map[string]any{"Namespace": "default"})
	if list, _ := resp["UserList"].([]any); len(list) != 1 {
		t.Fatalf("ListUsers: expected 1 user, got %d", len(list))
	}

	// Delete.
	callOK(t, h, "DeleteUser", map[string]any{"Namespace": "default", "UserName": "alice"})

	// Describe should now 404.
	call(t, h, "DescribeUser", map[string]any{"Namespace": "default", "UserName": "alice"}, http.StatusNotFound)
}

// ── Group lifecycle ──────────────────────────────────────────────────────────

func TestGroupLifecycle(t *testing.T) {
	h := newGateway(t)

	// Set up a user to add to the group.
	callCreated(t, h, "RegisterUser", map[string]any{
		"AwsAccountId": "000000000000",
		"Namespace":    "default",
		"UserName":     "bob",
		"Email":        "bob@example.com",
		"UserRole":     "READER",
		"IdentityType": "QUICKSIGHT",
	})

	// Create group.
	resp := callCreated(t, h, "CreateGroup", map[string]any{
		"Namespace":   "default",
		"GroupName":   "engineering",
		"Description": "engineering team",
	})
	group, _ := resp["Group"].(map[string]any)
	if group["GroupName"] != "engineering" {
		t.Fatalf("CreateGroup: unexpected group %v", group)
	}

	// Update.
	callOK(t, h, "UpdateGroup", map[string]any{
		"Namespace":   "default",
		"GroupName":   "engineering",
		"Description": "engineering & platform",
	})

	// Describe.
	resp = callOK(t, h, "DescribeGroup", map[string]any{
		"Namespace": "default",
		"GroupName": "engineering",
	})
	group, _ = resp["Group"].(map[string]any)
	if group["Description"] != "engineering & platform" {
		t.Fatalf("DescribeGroup: unexpected description %v", group["Description"])
	}

	// Add member.
	callOK(t, h, "CreateGroupMembership", map[string]any{
		"Namespace":  "default",
		"GroupName":  "engineering",
		"MemberName": "bob",
	})

	// Describe membership.
	callOK(t, h, "DescribeGroupMembership", map[string]any{
		"Namespace":  "default",
		"GroupName":  "engineering",
		"MemberName": "bob",
	})

	// List members.
	resp = callOK(t, h, "ListGroupMemberships", map[string]any{
		"Namespace": "default",
		"GroupName": "engineering",
	})
	if list, _ := resp["GroupMemberList"].([]any); len(list) != 1 {
		t.Fatalf("ListGroupMemberships: expected 1 member, got %d", len(list))
	}

	// List user groups.
	resp = callOK(t, h, "ListUserGroups", map[string]any{
		"Namespace": "default",
		"UserName":  "bob",
	})
	if list, _ := resp["GroupList"].([]any); len(list) != 1 {
		t.Fatalf("ListUserGroups: expected 1 group, got %d", len(list))
	}

	// Remove member.
	callOK(t, h, "DeleteGroupMembership", map[string]any{
		"Namespace":  "default",
		"GroupName":  "engineering",
		"MemberName": "bob",
	})

	// Delete group.
	callOK(t, h, "DeleteGroup", map[string]any{
		"Namespace": "default",
		"GroupName": "engineering",
	})

	// Should be gone.
	call(t, h, "DescribeGroup", map[string]any{"Namespace": "default", "GroupName": "engineering"}, http.StatusNotFound)
}

// ── Namespace lifecycle ──────────────────────────────────────────────────────

func TestNamespaceLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateNamespace", map[string]any{
		"Namespace":     "tenant-1",
		"IdentityStore": "QUICKSIGHT",
	})
	if resp["Name"] != "tenant-1" {
		t.Fatalf("CreateNamespace: expected name tenant-1, got %v", resp["Name"])
	}

	resp = callOK(t, h, "DescribeNamespace", map[string]any{"Namespace": "tenant-1"})
	ns, _ := resp["Namespace"].(map[string]any)
	if ns["Name"] != "tenant-1" {
		t.Fatalf("DescribeNamespace: unexpected namespace %v", ns)
	}

	resp = callOK(t, h, "ListNamespaces", nil)
	if list, _ := resp["Namespaces"].([]any); len(list) != 1 {
		t.Fatalf("ListNamespaces: expected 1 namespace, got %d", len(list))
	}

	callOK(t, h, "DeleteNamespace", map[string]any{"Namespace": "tenant-1"})
	call(t, h, "DescribeNamespace", map[string]any{"Namespace": "tenant-1"}, http.StatusNotFound)
}

// ── DataSource lifecycle ─────────────────────────────────────────────────────

func TestDataSourceLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateDataSource", map[string]any{
		"AwsAccountId": "000000000000",
		"DataSourceId": "ds-1",
		"Name":         "MyAthena",
		"Type":         "ATHENA",
		"DataSourceParameters": map[string]any{
			"AthenaParameters": map[string]any{"WorkGroup": "primary"},
		},
	})
	if resp["DataSourceId"] != "ds-1" {
		t.Fatalf("CreateDataSource: unexpected id %v", resp["DataSourceId"])
	}

	resp = callOK(t, h, "DescribeDataSource", map[string]any{"DataSourceId": "ds-1"})
	d, _ := resp["DataSource"].(map[string]any)
	if d["Name"] != "MyAthena" {
		t.Fatalf("DescribeDataSource: unexpected datasource %v", d)
	}

	callOK(t, h, "UpdateDataSource", map[string]any{
		"DataSourceId": "ds-1",
		"Name":         "MyAthenaV2",
	})

	resp = callOK(t, h, "ListDataSources", nil)
	if list, _ := resp["DataSources"].([]any); len(list) != 1 {
		t.Fatalf("ListDataSources: expected 1 data source, got %d", len(list))
	}

	// Permissions.
	callOK(t, h, "UpdateDataSourcePermissions", map[string]any{
		"DataSourceId": "ds-1",
		"GrantPermissions": []map[string]any{
			{"Principal": "arn:aws:quicksight:us-east-1:000000000000:user/default/alice", "Actions": []string{"quicksight:DescribeDataSource"}},
		},
	})
	resp = callOK(t, h, "DescribeDataSourcePermissions", map[string]any{"DataSourceId": "ds-1"})
	if perms, _ := resp["Permissions"].([]any); len(perms) != 1 {
		t.Fatalf("DescribeDataSourcePermissions: expected 1 permission, got %d", len(perms))
	}

	callOK(t, h, "DeleteDataSource", map[string]any{"DataSourceId": "ds-1"})
	call(t, h, "DescribeDataSource", map[string]any{"DataSourceId": "ds-1"}, http.StatusNotFound)
}

// ── DataSet lifecycle ────────────────────────────────────────────────────────

func TestDataSetLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateDataSet", map[string]any{
		"AwsAccountId": "000000000000",
		"DataSetId":    "ds-1",
		"Name":         "Sales",
		"PhysicalTableMap": map[string]any{
			"table1": map[string]any{"S3Source": map[string]any{"DataSourceArn": "arn:..."}},
		},
		"ImportMode": "SPICE",
	})
	if resp["DataSetId"] != "ds-1" {
		t.Fatalf("CreateDataSet: unexpected id %v", resp["DataSetId"])
	}

	resp = callOK(t, h, "DescribeDataSet", map[string]any{"DataSetId": "ds-1"})
	d, _ := resp["DataSet"].(map[string]any)
	if d["Name"] != "Sales" || d["ImportMode"] != "SPICE" {
		t.Fatalf("DescribeDataSet: unexpected dataset %v", d)
	}

	callOK(t, h, "UpdateDataSet", map[string]any{
		"DataSetId":  "ds-1",
		"Name":       "SalesV2",
		"ImportMode": "DIRECT_QUERY",
	})

	// PutDataSetRefreshProperties
	callOK(t, h, "PutDataSetRefreshProperties", map[string]any{
		"DataSetId": "ds-1",
		"DataSetRefreshProperties": map[string]any{
			"RefreshConfiguration": map[string]any{"IncrementalRefresh": map[string]any{}},
		},
	})
	resp = callOK(t, h, "DescribeDataSetRefreshProperties", map[string]any{"DataSetId": "ds-1"})
	if props, _ := resp["DataSetRefreshProperties"].(map[string]any); props == nil {
		t.Fatalf("DescribeDataSetRefreshProperties: expected non-nil props")
	}
	callOK(t, h, "DeleteDataSetRefreshProperties", map[string]any{"DataSetId": "ds-1"})

	resp = callOK(t, h, "ListDataSets", nil)
	if list, _ := resp["DataSetSummaries"].([]any); len(list) != 1 {
		t.Fatalf("ListDataSets: expected 1, got %d", len(list))
	}

	callOK(t, h, "DeleteDataSet", map[string]any{"DataSetId": "ds-1"})
	call(t, h, "DescribeDataSet", map[string]any{"DataSetId": "ds-1"}, http.StatusNotFound)
}

// ── Dashboard lifecycle ──────────────────────────────────────────────────────

func TestDashboardLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateDashboard", map[string]any{
		"AwsAccountId": "000000000000",
		"DashboardId":  "dash-1",
		"Name":         "Sales Dashboard",
		"SourceEntity": map[string]any{
			"SourceTemplate": map[string]any{
				"Arn": "arn:aws:quicksight:us-east-1:000000000000:template/t1",
			},
		},
	})
	if resp["DashboardId"] != "dash-1" {
		t.Fatalf("CreateDashboard: unexpected id %v", resp["DashboardId"])
	}

	resp = callOK(t, h, "DescribeDashboard", map[string]any{"DashboardId": "dash-1"})
	d, _ := resp["Dashboard"].(map[string]any)
	if d["Name"] != "Sales Dashboard" {
		t.Fatalf("DescribeDashboard: unexpected name %v", d["Name"])
	}

	callOK(t, h, "UpdateDashboard", map[string]any{
		"DashboardId": "dash-1",
		"Name":        "Sales Dashboard V2",
	})
	callOK(t, h, "UpdateDashboardPublishedVersion", map[string]any{
		"DashboardId":   "dash-1",
		"VersionNumber": 2,
	})
	resp = callOK(t, h, "ListDashboardVersions", map[string]any{"DashboardId": "dash-1"})
	if vers, _ := resp["DashboardVersionSummaryList"].([]any); len(vers) != 2 {
		t.Fatalf("ListDashboardVersions: expected 2 versions, got %d", len(vers))
	}

	callOK(t, h, "UpdateDashboardPermissions", map[string]any{
		"DashboardId": "dash-1",
		"GrantPermissions": []map[string]any{
			{"Principal": "arn:aws:quicksight:us-east-1:000000000000:user/default/alice", "Actions": []string{"quicksight:DescribeDashboard"}},
		},
	})
	resp = callOK(t, h, "DescribeDashboardPermissions", map[string]any{"DashboardId": "dash-1"})
	if perms, _ := resp["Permissions"].([]any); len(perms) != 1 {
		t.Fatalf("DescribeDashboardPermissions: expected 1 permission, got %d", len(perms))
	}

	callOK(t, h, "DeleteDashboard", map[string]any{"DashboardId": "dash-1"})
	call(t, h, "DescribeDashboard", map[string]any{"DashboardId": "dash-1"}, http.StatusNotFound)
}

// ── Analysis lifecycle ───────────────────────────────────────────────────────

func TestAnalysisLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateAnalysis", map[string]any{
		"AnalysisId": "ana-1",
		"Name":       "Sales Analysis",
		"SourceEntity": map[string]any{
			"SourceTemplate": map[string]any{"Arn": "arn:..."},
		},
	})
	if resp["AnalysisId"] != "ana-1" {
		t.Fatalf("CreateAnalysis: unexpected id %v", resp["AnalysisId"])
	}

	resp = callOK(t, h, "DescribeAnalysis", map[string]any{"AnalysisId": "ana-1"})
	a, _ := resp["Analysis"].(map[string]any)
	if a["Name"] != "Sales Analysis" {
		t.Fatalf("DescribeAnalysis: unexpected name %v", a["Name"])
	}

	callOK(t, h, "UpdateAnalysis", map[string]any{
		"AnalysisId": "ana-1",
		"Name":       "Sales Analysis V2",
	})

	// Soft-delete.
	callOK(t, h, "DeleteAnalysis", map[string]any{"AnalysisId": "ana-1"})

	// Restore.
	callOK(t, h, "RestoreAnalysis", map[string]any{"AnalysisId": "ana-1"})
}

// ── Template lifecycle ───────────────────────────────────────────────────────

func TestTemplateLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateTemplate", map[string]any{
		"TemplateId": "tmpl-1",
		"Name":       "MyTemplate",
		"SourceEntity": map[string]any{
			"SourceAnalysis": map[string]any{"Arn": "arn:..."},
		},
	})
	if resp["TemplateId"] != "tmpl-1" {
		t.Fatalf("CreateTemplate: unexpected id %v", resp["TemplateId"])
	}

	callOK(t, h, "DescribeTemplate", map[string]any{"TemplateId": "tmpl-1"})

	callOK(t, h, "UpdateTemplate", map[string]any{
		"TemplateId": "tmpl-1",
		"Name":       "MyTemplateV2",
	})

	// Template aliases.
	callCreated(t, h, "CreateTemplateAlias", map[string]any{
		"TemplateId":            "tmpl-1",
		"AliasName":             "PROD",
		"TemplateVersionNumber": 1,
	})
	callOK(t, h, "DescribeTemplateAlias", map[string]any{
		"TemplateId": "tmpl-1",
		"AliasName":  "PROD",
	})
	callOK(t, h, "UpdateTemplateAlias", map[string]any{
		"TemplateId":            "tmpl-1",
		"AliasName":             "PROD",
		"TemplateVersionNumber": 2,
	})
	resp = callOK(t, h, "ListTemplateAliases", map[string]any{"TemplateId": "tmpl-1"})
	if al, _ := resp["TemplateAliasList"].([]any); len(al) != 1 {
		t.Fatalf("ListTemplateAliases: expected 1, got %d", len(al))
	}
	callOK(t, h, "DeleteTemplateAlias", map[string]any{
		"TemplateId": "tmpl-1",
		"AliasName":  "PROD",
	})

	resp = callOK(t, h, "ListTemplateVersions", map[string]any{"TemplateId": "tmpl-1"})
	if vers, _ := resp["TemplateVersionSummaryList"].([]any); len(vers) != 2 {
		t.Fatalf("ListTemplateVersions: expected 2, got %d", len(vers))
	}

	callOK(t, h, "DeleteTemplate", map[string]any{"TemplateId": "tmpl-1"})
	call(t, h, "DescribeTemplate", map[string]any{"TemplateId": "tmpl-1"}, http.StatusNotFound)
}

// ── Theme lifecycle ──────────────────────────────────────────────────────────

func TestThemeLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateTheme", map[string]any{
		"ThemeId":     "theme-1",
		"Name":        "Dark Theme",
		"BaseThemeId": "MIDNIGHT",
		"Configuration": map[string]any{
			"DataColorPalette": map[string]any{},
		},
	})
	if resp["ThemeId"] != "theme-1" {
		t.Fatalf("CreateTheme: unexpected id %v", resp["ThemeId"])
	}

	callOK(t, h, "DescribeTheme", map[string]any{"ThemeId": "theme-1"})

	callOK(t, h, "UpdateTheme", map[string]any{
		"ThemeId": "theme-1",
		"Name":    "Dark Theme V2",
	})

	callCreated(t, h, "CreateThemeAlias", map[string]any{
		"ThemeId":            "theme-1",
		"AliasName":          "PROD",
		"ThemeVersionNumber": 1,
	})
	callOK(t, h, "DescribeThemeAlias", map[string]any{
		"ThemeId":   "theme-1",
		"AliasName": "PROD",
	})
	callOK(t, h, "UpdateThemeAlias", map[string]any{
		"ThemeId":            "theme-1",
		"AliasName":          "PROD",
		"ThemeVersionNumber": 2,
	})
	resp = callOK(t, h, "ListThemeAliases", map[string]any{"ThemeId": "theme-1"})
	if al, _ := resp["ThemeAliasList"].([]any); len(al) != 1 {
		t.Fatalf("ListThemeAliases: expected 1, got %d", len(al))
	}
	callOK(t, h, "DeleteThemeAlias", map[string]any{
		"ThemeId":   "theme-1",
		"AliasName": "PROD",
	})

	callOK(t, h, "DeleteTheme", map[string]any{"ThemeId": "theme-1"})
	call(t, h, "DescribeTheme", map[string]any{"ThemeId": "theme-1"}, http.StatusNotFound)
}

// ── Folder lifecycle ─────────────────────────────────────────────────────────

func TestFolderLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateFolder", map[string]any{
		"FolderId":   "folder-1",
		"Name":       "MyFolder",
		"FolderType": "SHARED",
	})
	if resp["FolderId"] != "folder-1" {
		t.Fatalf("CreateFolder: unexpected id %v", resp["FolderId"])
	}

	callOK(t, h, "DescribeFolder", map[string]any{"FolderId": "folder-1"})

	callOK(t, h, "UpdateFolder", map[string]any{
		"FolderId": "folder-1",
		"Name":     "MyFolderV2",
	})

	// Add membership.
	callCreated(t, h, "CreateFolderMembership", map[string]any{
		"FolderId":   "folder-1",
		"MemberId":   "dash-x",
		"MemberType": "DASHBOARD",
	})
	resp = callOK(t, h, "ListFolderMembers", map[string]any{"FolderId": "folder-1"})
	if list, _ := resp["FolderMemberList"].([]any); len(list) != 1 {
		t.Fatalf("ListFolderMembers: expected 1 member, got %d", len(list))
	}
	callOK(t, h, "DeleteFolderMembership", map[string]any{
		"FolderId": "folder-1",
		"MemberId": "dash-x",
	})

	// Folder permissions.
	callOK(t, h, "UpdateFolderPermissions", map[string]any{
		"FolderId": "folder-1",
		"GrantPermissions": []map[string]any{
			{"Principal": "arn:...:user/default/alice", "Actions": []string{"quicksight:DescribeFolder"}},
		},
	})
	resp = callOK(t, h, "DescribeFolderPermissions", map[string]any{"FolderId": "folder-1"})
	if perms, _ := resp["Permissions"].([]any); len(perms) != 1 {
		t.Fatalf("DescribeFolderPermissions: expected 1 permission, got %d", len(perms))
	}

	callOK(t, h, "DeleteFolder", map[string]any{"FolderId": "folder-1"})
	call(t, h, "DescribeFolder", map[string]any{"FolderId": "folder-1"}, http.StatusNotFound)
}

// ── RefreshSchedule + Ingestion lifecycle ────────────────────────────────────

func TestRefreshScheduleAndIngestion(t *testing.T) {
	h := newGateway(t)

	// First need a dataset.
	callCreated(t, h, "CreateDataSet", map[string]any{
		"DataSetId":  "ds-1",
		"Name":       "Sales",
		"ImportMode": "SPICE",
	})

	// Create refresh schedule.
	callCreated(t, h, "CreateRefreshSchedule", map[string]any{
		"DataSetId": "ds-1",
		"Schedule": map[string]any{
			"ScheduleId": "sched-1",
			"ScheduleFrequency": map[string]any{
				"Interval": "DAILY",
				"TimeOfTheDay": "00:00",
			},
			"RefreshType": "FULL_REFRESH",
		},
	})
	callOK(t, h, "DescribeRefreshSchedule", map[string]any{
		"DataSetId":  "ds-1",
		"ScheduleId": "sched-1",
	})
	callOK(t, h, "UpdateRefreshSchedule", map[string]any{
		"DataSetId": "ds-1",
		"Schedule": map[string]any{
			"ScheduleId":  "sched-1",
			"RefreshType": "INCREMENTAL_REFRESH",
		},
	})
	resp := callOK(t, h, "ListRefreshSchedules", map[string]any{"DataSetId": "ds-1"})
	if list, _ := resp["RefreshSchedules"].([]any); len(list) != 1 {
		t.Fatalf("ListRefreshSchedules: expected 1, got %d", len(list))
	}

	// Create ingestion.
	callCreated(t, h, "CreateIngestion", map[string]any{
		"DataSetId":   "ds-1",
		"IngestionId": "ing-1",
	})
	resp = callOK(t, h, "DescribeIngestion", map[string]any{
		"DataSetId":   "ds-1",
		"IngestionId": "ing-1",
	})
	ing, _ := resp["Ingestion"].(map[string]any)
	if ing["IngestionStatus"] == "" {
		t.Fatalf("DescribeIngestion: empty status")
	}
	resp = callOK(t, h, "ListIngestions", map[string]any{"DataSetId": "ds-1"})
	if list, _ := resp["Ingestions"].([]any); len(list) != 1 {
		t.Fatalf("ListIngestions: expected 1, got %d", len(list))
	}
	callOK(t, h, "CancelIngestion", map[string]any{
		"DataSetId":   "ds-1",
		"IngestionId": "ing-1",
	})

	// Cleanup.
	callOK(t, h, "DeleteRefreshSchedule", map[string]any{
		"DataSetId":  "ds-1",
		"ScheduleId": "sched-1",
	})
}

// ── VPC Connection lifecycle ─────────────────────────────────────────────────

func TestVPCConnectionLifecycle(t *testing.T) {
	h := newGateway(t)

	resp := callCreated(t, h, "CreateVPCConnection", map[string]any{
		"VPCConnectionId":  "vpc-1",
		"Name":             "MyVPC",
		"VpcId":            "vpc-12345",
		"SecurityGroupIds": []string{"sg-1"},
		"DnsResolvers":     []string{"10.0.0.2"},
		"RoleArn":          "arn:aws:iam::000000000000:role/QuickSightVPC",
	})
	if resp["VPCConnectionId"] != "vpc-1" {
		t.Fatalf("CreateVPCConnection: unexpected id %v", resp["VPCConnectionId"])
	}

	resp = callOK(t, h, "DescribeVPCConnection", map[string]any{"VPCConnectionId": "vpc-1"})
	v, _ := resp["VPCConnection"].(map[string]any)
	if v["Name"] != "MyVPC" {
		t.Fatalf("DescribeVPCConnection: unexpected name %v", v["Name"])
	}

	callOK(t, h, "UpdateVPCConnection", map[string]any{
		"VPCConnectionId": "vpc-1",
		"Name":            "MyVPCv2",
	})

	resp = callOK(t, h, "ListVPCConnections", nil)
	if list, _ := resp["VPCConnectionSummaries"].([]any); len(list) != 1 {
		t.Fatalf("ListVPCConnections: expected 1, got %d", len(list))
	}

	callOK(t, h, "DeleteVPCConnection", map[string]any{"VPCConnectionId": "vpc-1"})
	call(t, h, "DescribeVPCConnection", map[string]any{"VPCConnectionId": "vpc-1"}, http.StatusNotFound)
}

// ── IAM Policy Assignment ────────────────────────────────────────────────────

func TestIAMPolicyAssignmentLifecycle(t *testing.T) {
	h := newGateway(t)

	callCreated(t, h, "CreateIAMPolicyAssignment", map[string]any{
		"AwsAccountId":     "000000000000",
		"Namespace":        "default",
		"AssignmentName":   "myassign",
		"AssignmentStatus": "ENABLED",
		"PolicyArn":        "arn:aws:iam::aws:policy/AdministratorAccess",
		"Identities": map[string]any{
			"User": []string{"alice"},
		},
	})

	resp := callOK(t, h, "DescribeIAMPolicyAssignment", map[string]any{
		"Namespace":      "default",
		"AssignmentName": "myassign",
	})
	a, _ := resp["IAMPolicyAssignment"].(map[string]any)
	if a["AssignmentName"] != "myassign" {
		t.Fatalf("DescribeIAMPolicyAssignment: unexpected name %v", a["AssignmentName"])
	}

	callOK(t, h, "UpdateIAMPolicyAssignment", map[string]any{
		"Namespace":        "default",
		"AssignmentName":   "myassign",
		"AssignmentStatus": "DISABLED",
	})

	resp = callOK(t, h, "ListIAMPolicyAssignments", map[string]any{"Namespace": "default"})
	if list, _ := resp["IAMPolicyAssignments"].([]any); len(list) != 1 {
		t.Fatalf("ListIAMPolicyAssignments: expected 1, got %d", len(list))
	}

	callOK(t, h, "DeleteIAMPolicyAssignment", map[string]any{
		"Namespace":      "default",
		"AssignmentName": "myassign",
	})
}

// ── Topic lifecycle ──────────────────────────────────────────────────────────

func TestTopicLifecycle(t *testing.T) {
	h := newGateway(t)

	callCreated(t, h, "CreateTopic", map[string]any{
		"TopicId": "topic-1",
		"Topic": map[string]any{
			"Name":                  "Sales Topic",
			"Description":           "Q topic for sales data",
			"UserExperienceVersion": "NEW_READER_EXPERIENCE",
		},
	})

	resp := callOK(t, h, "DescribeTopic", map[string]any{"TopicId": "topic-1"})
	topic, _ := resp["Topic"].(map[string]any)
	if topic["Name"] != "Sales Topic" {
		t.Fatalf("DescribeTopic: unexpected name %v", topic["Name"])
	}

	callOK(t, h, "UpdateTopic", map[string]any{
		"TopicId": "topic-1",
		"Topic": map[string]any{
			"Name": "Sales Topic V2",
		},
	})

	// Topic refresh schedule.
	callCreated(t, h, "CreateTopicRefreshSchedule", map[string]any{
		"TopicId": "topic-1",
		"RefreshSchedule": map[string]any{
			"IsEnabled":       true,
			"BasedOnSpiceSchedule": true,
		},
	})
	callOK(t, h, "DescribeTopicRefreshSchedule", map[string]any{
		"TopicId":    "topic-1",
		"DatasetArn": "arn:aws:quicksight:us-east-1:000000000000:dataset/ds-1",
	})
	callOK(t, h, "UpdateTopicRefreshSchedule", map[string]any{
		"TopicId": "topic-1",
		"RefreshSchedule": map[string]any{
			"IsEnabled": false,
		},
	})
	callOK(t, h, "DeleteTopicRefreshSchedule", map[string]any{"TopicId": "topic-1"})

	// Topic permissions.
	callOK(t, h, "UpdateTopicPermissions", map[string]any{
		"TopicId": "topic-1",
		"GrantPermissions": []map[string]any{
			{"Principal": "arn:...:user/default/alice", "Actions": []string{"quicksight:DescribeTopic"}},
		},
	})

	callOK(t, h, "DeleteTopic", map[string]any{"TopicId": "topic-1"})
}

// ── Custom permissions lifecycle ─────────────────────────────────────────────

func TestCustomPermissionsLifecycle(t *testing.T) {
	h := newGateway(t)

	callCreated(t, h, "CreateCustomPermissions", map[string]any{
		"CustomPermissionsName": "RestrictedReader",
		"Capabilities": map[string]any{
			"ExportToCsv": "DENY",
		},
	})

	resp := callOK(t, h, "DescribeCustomPermissions", map[string]any{
		"CustomPermissionsName": "RestrictedReader",
	})
	cp, _ := resp["CustomPermissions"].(map[string]any)
	if cp["CustomPermissionsName"] != "RestrictedReader" {
		t.Fatalf("DescribeCustomPermissions: unexpected %v", cp)
	}

	callOK(t, h, "UpdateCustomPermissions", map[string]any{
		"CustomPermissionsName": "RestrictedReader",
		"Capabilities": map[string]any{
			"ExportToCsv": "ALLOW",
		},
	})

	resp = callOK(t, h, "ListCustomPermissions", nil)
	if list, _ := resp["CustomPermissionsList"].([]any); len(list) != 1 {
		t.Fatalf("ListCustomPermissions: expected 1, got %d", len(list))
	}

	callOK(t, h, "DeleteCustomPermissions", map[string]any{"CustomPermissionsName": "RestrictedReader"})
}

// ── AssetBundleExportJob lifecycle ───────────────────────────────────────────

func TestAssetBundleExportJobLifecycle(t *testing.T) {
	h := newGateway(t)

	callOK(t, h, "StartAssetBundleExportJob", map[string]any{
		"AssetBundleExportJobId": "exp-1",
		"ResourceArns":           []string{"arn:aws:quicksight:us-east-1:000000000000:dashboard/d1"},
		"ExportFormat":           "QUICKSIGHT_JSON",
		"IncludePermissions":     true,
	})

	// First describe should return a status.
	resp := callOK(t, h, "DescribeAssetBundleExportJob", map[string]any{
		"AssetBundleExportJobId": "exp-1",
	})
	if status, _ := resp["JobStatus"].(string); status != "SUCCESSFUL" {
		t.Fatalf("DescribeAssetBundleExportJob: expected SUCCESSFUL on first describe, got %v", status)
	}
	if dl, _ := resp["DownloadUrl"].(string); dl == "" {
		t.Fatalf("DescribeAssetBundleExportJob: expected DownloadUrl, got empty")
	}

	resp = callOK(t, h, "ListAssetBundleExportJobs", nil)
	if list, _ := resp["AssetBundleExportJobSummaryList"].([]any); len(list) != 1 {
		t.Fatalf("ListAssetBundleExportJobs: expected 1, got %d", len(list))
	}
}

// ── AssetBundleImportJob lifecycle ───────────────────────────────────────────

func TestAssetBundleImportJobLifecycle(t *testing.T) {
	h := newGateway(t)

	callOK(t, h, "StartAssetBundleImportJob", map[string]any{
		"AssetBundleImportJobId": "imp-1",
		"AssetBundleImportSource": map[string]any{
			"S3Uri": "s3://my-bucket/asset.qs",
		},
		"FailureAction": "DO_NOTHING",
	})

	resp := callOK(t, h, "DescribeAssetBundleImportJob", map[string]any{
		"AssetBundleImportJobId": "imp-1",
	})
	if status, _ := resp["JobStatus"].(string); status != "SUCCESSFUL" {
		t.Fatalf("DescribeAssetBundleImportJob: expected SUCCESSFUL on first describe, got %v", status)
	}

	resp = callOK(t, h, "ListAssetBundleImportJobs", nil)
	if list, _ := resp["AssetBundleImportJobSummaryList"].([]any); len(list) != 1 {
		t.Fatalf("ListAssetBundleImportJobs: expected 1, got %d", len(list))
	}
}

// ── Embed URL handlers ───────────────────────────────────────────────────────

func TestEmbedUrlHandlers(t *testing.T) {
	h := newGateway(t)

	for _, action := range []string{
		"GetSessionEmbedUrl",
		"GetDashboardEmbedUrl",
		"GenerateEmbedUrlForRegisteredUser",
		"GenerateEmbedUrlForRegisteredUserWithIdentity",
		"GenerateEmbedUrlForAnonymousUser",
	} {
		resp := callOK(t, h, action, map[string]any{"AwsAccountId": "000000000000"})
		if url, _ := resp["EmbedUrl"].(string); url == "" {
			t.Fatalf("%s: expected non-empty EmbedUrl", action)
		}
	}
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func TestTagHandlers(t *testing.T) {
	h := newGateway(t)

	arn := "arn:aws:quicksight:us-east-1:000000000000:dataset/ds-tag"
	callOK(t, h, "TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags": []map[string]any{
			{"Key": "env", "Value": "prod"},
			{"Key": "owner", "Value": "team-a"},
		},
	})

	resp := callOK(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	tags, _ := resp["Tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("ListTagsForResource: expected 2 tags, got %d", len(tags))
	}

	callOK(t, h, "UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"env"},
	})
	resp = callOK(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	if tags, _ := resp["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("ListTagsForResource after untag: expected 1, got %d", len(tags))
	}
}

// ── Account settings handlers ────────────────────────────────────────────────

func TestAccountSettingsHandlers(t *testing.T) {
	h := newGateway(t)

	resp := callOK(t, h, "DescribeAccountSettings", nil)
	if as, _ := resp["AccountSettings"].(map[string]any); as == nil {
		t.Fatalf("DescribeAccountSettings: expected AccountSettings")
	}

	callOK(t, h, "UpdateAccountSettings", map[string]any{
		"DefaultNamespace":             "default",
		"NotificationEmail":            "ops@example.com",
		"TerminationProtectionEnabled": false,
	})

	resp = callOK(t, h, "DescribeAccountSettings", nil)
	as, _ := resp["AccountSettings"].(map[string]any)
	if as["NotificationEmail"] != "ops@example.com" {
		t.Fatalf("UpdateAccountSettings: expected ops@example.com, got %v", as["NotificationEmail"])
	}

	resp = callOK(t, h, "DescribeAccountSubscription", nil)
	if ai, _ := resp["AccountInfo"].(map[string]any); ai == nil {
		t.Fatalf("DescribeAccountSubscription: expected AccountInfo")
	}
}

// ── Smoke test: every action returns a non-error status ──────────────────────

// TestAllActionsRespondNonServerError calls every registered action with a minimal
// payload and asserts the status code is < 500. This catches handlers that crash
// on empty input rather than returning a clean validation error.
func TestAllActionsRespondNonServerError(t *testing.T) {
	h := newGateway(t)

	// Per-action minimum payloads to satisfy required-field validation.
	// Anything not listed here will be invoked with an empty body.
	payloads := map[string]map[string]any{
		"RegisterUser":                {"UserName": "u", "UserRole": "READER", "IdentityType": "QUICKSIGHT"},
		"CreateGroup":                 {"GroupName": "g"},
		"CreateNamespace":             {"Namespace": "n"},
		"CreateDataSource":            {"DataSourceId": "ds-x", "Name": "x", "Type": "ATHENA"},
		"CreateDataSet":               {"DataSetId": "set-x", "Name": "x"},
		"CreateTemplate":              {"TemplateId": "t-x", "Name": "x"},
		"CreateDashboard":             {"DashboardId": "d-x", "Name": "x"},
		"CreateAnalysis":              {"AnalysisId": "a-x", "Name": "x"},
		"CreateTheme":                 {"ThemeId": "th-x", "Name": "x"},
		"CreateFolder":                {"FolderId": "f-x", "Name": "x"},
		"CreateTopic":                 {"TopicId": "tp-x", "Topic": map[string]any{"Name": "x"}},
		"CreateVPCConnection":         {"VPCConnectionId": "vp-x", "Name": "x", "VpcId": "vpc-x"},
		"CreateCustomPermissions":     {"CustomPermissionsName": "cp-x"},
		"CreateBrand":                 {"BrandId": "b-x"},
		"CreateIAMPolicyAssignment":   {"AssignmentName": "an-x"},
		"CreateActionConnector":       {"ActionConnectorId": "ac-x", "Name": "x", "Type": "GENERIC"},
		"UpdateIdentityPropagationConfig": {"Service": "redshift"},
	}

	for _, action := range allQuickSightActions(t) {
		body := payloads[action]
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(mustJSON(t, body)))
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
		req.Header.Set("X-Amz-Target", "quicksight."+action)
		req.Header.Set("Authorization",
			"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/quicksight/aws4_request, SignedHeaders=host, Signature=abc123")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code >= 500 {
			t.Errorf("%s: unexpected server error %d, body: %s", action, w.Code, w.Body.String())
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	if v == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func allQuickSightActions(t *testing.T) []string {
	t.Helper()
	s := svc.New("000000000000", "us-east-1")
	out := make([]string, 0, len(s.Actions()))
	for _, a := range s.Actions() {
		out = append(out, a.Name)
	}
	return out
}
