package azurearm_test

import (
	"reflect"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
)

func TestParseID_ResourceGroupProviderResource(t *testing.T) {
	id, err := azurearm.ParseID("/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct")
	if err != nil {
		t.Fatalf("ParseID returned error: %v", err)
	}

	if id.SubscriptionID != "sub-1" {
		t.Errorf("expected subscription sub-1, got %q", id.SubscriptionID)
	}
	if id.ResourceGroup != "rg-a" {
		t.Errorf("expected resource group rg-a, got %q", id.ResourceGroup)
	}
	if id.Provider != "Microsoft.Storage" {
		t.Errorf("expected provider Microsoft.Storage, got %q", id.Provider)
	}
	if !reflect.DeepEqual(id.Types, []string{"storageAccounts"}) {
		t.Errorf("expected storageAccounts type, got %#v", id.Types)
	}
	if !reflect.DeepEqual(id.Names, []string{"acct"}) {
		t.Errorf("expected acct name, got %#v", id.Names)
	}
}

func TestParseID_NestedResourceTypeChain(t *testing.T) {
	id, err := azurearm.ParseID("/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging")
	if err != nil {
		t.Fatalf("ParseID returned error: %v", err)
	}

	if !reflect.DeepEqual(id.Types, []string{"sites", "slots"}) {
		t.Errorf("expected nested type chain, got %#v", id.Types)
	}
	if !reflect.DeepEqual(id.Names, []string{"site-a", "staging"}) {
		t.Errorf("expected nested name chain, got %#v", id.Names)
	}
	if id.String() != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/slots/staging" {
		t.Errorf("unexpected formatted ID: %q", id.String())
	}
}

func TestParseID_SubscriptionScopeProvider(t *testing.T) {
	id, err := azurearm.ParseID("/subscriptions/sub-1/providers/Microsoft.Authorization/policyAssignments/assign-a")
	if err != nil {
		t.Fatalf("ParseID returned error: %v", err)
	}

	if id.ResourceGroup != "" {
		t.Errorf("expected no resource group, got %q", id.ResourceGroup)
	}
	if id.Provider != "Microsoft.Authorization" {
		t.Errorf("expected provider Microsoft.Authorization, got %q", id.Provider)
	}
	if id.String() != "/subscriptions/sub-1/providers/Microsoft.Authorization/policyAssignments/assign-a" {
		t.Errorf("unexpected formatted ID: %q", id.String())
	}
}

func TestParseID_RejectsMalformedResourceChain(t *testing.T) {
	if _, err := azurearm.ParseID("/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts"); err == nil {
		t.Fatal("expected malformed type/name chain to return an error")
	}
}
