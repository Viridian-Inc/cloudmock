package iam_test

import (
	"encoding/json"
	"testing"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	iamsvc "github.com/Viridian-Inc/cloudmock/services/iam"
)

const (
	iamTestAccount = "123456789012"
)

func newIAMService(t *testing.T) *iamsvc.IAMService {
	t.Helper()
	engine := iampkg.NewEngine()
	pkgStore := iampkg.NewStore(iamTestAccount)
	return iamsvc.New(iamTestAccount, engine, pkgStore)
}

func TestIAM_ExportState_Empty(t *testing.T) {
	svc := newIAMService(t)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Users    []any `json:"users"`
		Roles    []any `json:"roles"`
		Policies []any `json:"policies"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Users) != 0 {
		t.Errorf("expected empty users, got %d", len(state.Users))
	}
	if len(state.Roles) != 0 {
		t.Errorf("expected empty roles, got %d", len(state.Roles))
	}
	if len(state.Policies) != 0 {
		t.Errorf("expected empty policies, got %d", len(state.Policies))
	}
}

func TestIAM_ExportState_WithUsersRolesPolicies(t *testing.T) {
	svc := newIAMService(t)

	seed := json.RawMessage(`{
		"users":[{"user_name":"alice"},{"user_name":"bob"}],
		"roles":[{"role_name":"app-role","assume_role_policy_document":"{\"Version\":\"2012-10-17\"}","description":"app role"}],
		"policies":[{"policy_name":"read-only","document":"{\"Version\":\"2012-10-17\"}","description":"read only policy"}]
	}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Users    []struct{ UserName string `json:"user_name"` }    `json:"users"`
		Roles    []struct{ RoleName string `json:"role_name"` }    `json:"roles"`
		Policies []struct{ PolicyName string `json:"policy_name"` } `json:"policies"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(state.Users))
	}
	if len(state.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(state.Roles))
	}
	if len(state.Policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(state.Policies))
	}
}

func TestIAM_ImportState_RestoresUsers(t *testing.T) {
	svc := newIAMService(t)

	data := json.RawMessage(`{"users":[{"user_name":"charlie"},{"user_name":"diana"}],"roles":[],"policies":[]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Users []struct{ UserName string `json:"user_name"` } `json:"users"`
	}
	json.Unmarshal(raw, &state)

	names := make(map[string]bool)
	for _, u := range state.Users {
		names[u.UserName] = true
	}
	for _, expected := range []string{"charlie", "diana"} {
		if !names[expected] {
			t.Errorf("user %q not restored", expected)
		}
	}
}

func TestIAM_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := newIAMService(t)

	if err := svc.ImportState(json.RawMessage(`{"users":[],"roles":[],"policies":[]}`)); err != nil {
		t.Fatalf("ImportState with empty state: %v", err)
	}
}

func TestIAM_RoundTrip_PreservesAllEntities(t *testing.T) {
	svc := newIAMService(t)

	seed := json.RawMessage(`{
		"users":[{"user_name":"eve","tags":{"dept":"engineering"}}],
		"roles":[{"role_name":"deploy-role","assume_role_policy_document":"{}"}],
		"policies":[{"policy_name":"deploy-policy","document":"{}"}]
	}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := newIAMService(t)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	raw2, err := svc2.ExportState()
	if err != nil {
		t.Fatalf("ExportState (svc2): %v", err)
	}

	var s1, s2 struct {
		Users    []any `json:"users"`
		Roles    []any `json:"roles"`
		Policies []any `json:"policies"`
	}
	json.Unmarshal(raw, &s1)
	json.Unmarshal(raw2, &s2)

	if len(s2.Users) != len(s1.Users) {
		t.Errorf("user count mismatch: want %d, got %d", len(s1.Users), len(s2.Users))
	}
	if len(s2.Roles) != len(s1.Roles) {
		t.Errorf("role count mismatch: want %d, got %d", len(s1.Roles), len(s2.Roles))
	}
	if len(s2.Policies) != len(s1.Policies) {
		t.Errorf("policy count mismatch: want %d, got %d", len(s1.Policies), len(s2.Policies))
	}
}
