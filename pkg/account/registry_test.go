package account

import (
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// stubService is a minimal service.Service implementation for testing.
type stubService struct {
	name      string
	accountID string
}

func (s *stubService) Name() string                                                  { return s.name }
func (s *stubService) Actions() []service.Action                                     { return nil }
func (s *stubService) HandleRequest(_ *service.RequestContext) (*service.Response, error) { return nil, nil }
func (s *stubService) HealthCheck() error                                            { return nil }

func TestRegistry_DefaultAccount(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")

	acct, ok := r.GetAccount("111111111111")
	if !ok {
		t.Fatal("default account should exist")
	}
	if acct.ID != "111111111111" {
		t.Errorf("default account ID = %q, want %q", acct.ID, "111111111111")
	}
	if acct.Name != "Default Account" {
		t.Errorf("default account name = %q, want %q", acct.Name, "Default Account")
	}
	if r.Default() != "111111111111" {
		t.Errorf("Default() = %q, want %q", r.Default(), "111111111111")
	}
}

func TestRegistry_CreateAccount(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")

	acct, err := r.CreateAccount("222222222222", "Dev Account")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if acct.ID != "222222222222" {
		t.Errorf("account ID = %q, want %q", acct.ID, "222222222222")
	}
	if acct.Name != "Dev Account" {
		t.Errorf("account name = %q, want %q", acct.Name, "Dev Account")
	}

	// Verify it can be retrieved.
	got, ok := r.GetAccount("222222222222")
	if !ok {
		t.Fatal("created account should be retrievable")
	}
	if got.ID != "222222222222" {
		t.Errorf("GetAccount ID = %q, want %q", got.ID, "222222222222")
	}
}

func TestRegistry_GetService_LazyInit(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")

	var factoryCalls int
	r.RegisterFactory("sts", func(accountID, region string) service.Service {
		factoryCalls++
		return &stubService{name: "sts", accountID: accountID}
	})

	// First call should trigger factory.
	svc, ok := r.GetService("111111111111", "sts")
	if !ok {
		t.Fatal("GetService should return true for registered factory")
	}
	if svc.Name() != "sts" {
		t.Errorf("service name = %q, want %q", svc.Name(), "sts")
	}
	if factoryCalls != 1 {
		t.Errorf("factory should be called once, got %d", factoryCalls)
	}

	// Second call should return cached instance.
	svc2, ok := r.GetService("111111111111", "sts")
	if !ok {
		t.Fatal("GetService should return true on second call")
	}
	if factoryCalls != 1 {
		t.Errorf("factory should still be called once, got %d", factoryCalls)
	}
	if svc != svc2 {
		t.Error("second call should return the same instance")
	}

	// Different account should get a different instance.
	r.CreateAccount("222222222222", "Other")
	svc3, ok := r.GetService("222222222222", "sts")
	if !ok {
		t.Fatal("GetService should work for second account")
	}
	if factoryCalls != 2 {
		t.Errorf("factory should be called twice, got %d", factoryCalls)
	}
	if svc3.(*stubService).accountID != "222222222222" {
		t.Errorf("service should be created for account 222222222222, got %q", svc3.(*stubService).accountID)
	}
}

func TestRegistry_CredentialMapping(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")
	r.CreateAccount("222222222222", "Target")

	r.MapCredential("ASIA1234567890abcdef", "222222222222")

	acctID, ok := r.ResolveCredential("ASIA1234567890abcdef")
	if !ok {
		t.Fatal("ResolveCredential should return true for mapped credential")
	}
	if acctID != "222222222222" {
		t.Errorf("resolved account = %q, want %q", acctID, "222222222222")
	}

	// Unknown credential should return false.
	_, ok = r.ResolveCredential("ASIAUNKNOWN")
	if ok {
		t.Error("ResolveCredential should return false for unknown credential")
	}
}

func TestRegistry_ListAccounts(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")
	r.CreateAccount("222222222222", "Dev")
	r.CreateAccount("333333333333", "Staging")

	accounts := r.ListAccounts()
	if len(accounts) != 3 {
		t.Fatalf("ListAccounts should return 3 accounts (default + 2), got %d", len(accounts))
	}

	ids := make(map[string]bool)
	for _, a := range accounts {
		ids[a.ID] = true
	}
	for _, want := range []string{"111111111111", "222222222222", "333333333333"} {
		if !ids[want] {
			t.Errorf("ListAccounts missing account %q", want)
		}
	}
}

func TestRegistry_DuplicateAccount(t *testing.T) {
	r := NewRegistry("111111111111", "us-east-1")

	// Creating a duplicate of the default account should fail.
	_, err := r.CreateAccount("111111111111", "Duplicate")
	if err == nil {
		t.Fatal("creating duplicate account should return error")
	}

	// Create then duplicate a non-default account.
	_, err = r.CreateAccount("222222222222", "Dev")
	if err != nil {
		t.Fatalf("first creation: %v", err)
	}

	_, err = r.CreateAccount("222222222222", "Dev Again")
	if err == nil {
		t.Fatal("creating duplicate non-default account should return error")
	}
}
