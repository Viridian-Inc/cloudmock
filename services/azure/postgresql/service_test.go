package postgresql

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestFlexibleServerDatabaseAndFirewallRuleLifecycle(t *testing.T) {
	svc := New()

	serverURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a?api-version=2025-08-01"
	serverPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard_D2s_v3","tier":"GeneralPurpose"},
		"tags":{"env":"test"},
		"properties":{
			"administratorLogin":"cloudmockadmin",
			"version":"16",
			"storage":{"storageSizeGB":128},
			"backup":{"backupRetentionDays":7},
			"network":{"publicNetworkAccess":"Enabled"}
		}
	}`)
	createServerResp, err := svc.HandleRequest(postgresCtx(t, http.MethodPut, serverURL, serverPayload))
	if err != nil {
		t.Fatalf("create server returned error: %v", err)
	}
	if createServerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create server status 201, got %d; body=%s", createServerResp.StatusCode, string(createServerResp.RawBody))
	}
	server := decodePostgreSQLResponse(t, createServerResp)
	if server["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a" {
		t.Fatalf("unexpected server id: %v", server["id"])
	}
	if server["name"] != "pg-a" || server["type"] != "Microsoft.DBforPostgreSQL/flexibleServers" || server["location"] != "eastus" {
		t.Fatalf("unexpected server identity fields: %v", server)
	}
	serverProps := server["properties"].(map[string]any)
	if serverProps["state"] != "Ready" || serverProps["provisioningState"] != "Succeeded" || serverProps["fullyQualifiedDomainName"] != "pg-a.postgres.database.azure.com" {
		t.Fatalf("unexpected server properties: %v", serverProps)
	}

	listServersResp, err := svc.HandleRequest(postgresCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers?api-version=2025-08-01", nil))
	if err != nil {
		t.Fatalf("list servers returned error: %v", err)
	}
	listedServers := decodePostgreSQLResponse(t, listServersResp)
	if len(listedServers["value"].([]any)) != 1 {
		t.Fatalf("expected one server in list, got %v", listedServers)
	}

	databaseURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/databases/appdb?api-version=2025-08-01"
	databasePayload := []byte(`{
		"properties":{
			"charset":"UTF8",
			"collation":"en_US.utf8"
		}
	}`)
	createDatabaseResp, err := svc.HandleRequest(postgresCtx(t, http.MethodPut, databaseURL, databasePayload))
	if err != nil {
		t.Fatalf("create database returned error: %v", err)
	}
	if createDatabaseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create database status 201, got %d; body=%s", createDatabaseResp.StatusCode, string(createDatabaseResp.RawBody))
	}
	database := decodePostgreSQLResponse(t, createDatabaseResp)
	if database["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/databases/appdb" {
		t.Fatalf("unexpected database id: %v", database["id"])
	}
	if database["name"] != "pg-a/appdb" || database["type"] != "Microsoft.DBforPostgreSQL/flexibleServers/databases" {
		t.Fatalf("unexpected database identity fields: %v", database)
	}
	databaseProps := database["properties"].(map[string]any)
	if databaseProps["charset"] != "UTF8" || databaseProps["collation"] != "en_US.utf8" || databaseProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected database properties: %v", databaseProps)
	}

	listDatabasesResp, err := svc.HandleRequest(postgresCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/databases?api-version=2025-08-01", nil))
	if err != nil {
		t.Fatalf("list databases returned error: %v", err)
	}
	listedDatabases := decodePostgreSQLResponse(t, listDatabasesResp)
	if len(listedDatabases["value"].([]any)) != 1 {
		t.Fatalf("expected one database in list, got %v", listedDatabases)
	}

	firewallURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/firewallRules/allow-office?api-version=2025-08-01"
	firewallPayload := []byte(`{
		"properties":{
			"startIpAddress":"203.0.113.20",
			"endIpAddress":"203.0.113.20"
		}
	}`)
	createFirewallResp, err := svc.HandleRequest(postgresCtx(t, http.MethodPut, firewallURL, firewallPayload))
	if err != nil {
		t.Fatalf("create firewall rule returned error: %v", err)
	}
	if createFirewallResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create firewall rule status 201, got %d; body=%s", createFirewallResp.StatusCode, string(createFirewallResp.RawBody))
	}
	firewallRule := decodePostgreSQLResponse(t, createFirewallResp)
	if firewallRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/firewallRules/allow-office" {
		t.Fatalf("unexpected firewall rule id: %v", firewallRule["id"])
	}
	if firewallRule["name"] != "pg-a/allow-office" || firewallRule["type"] != "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules" {
		t.Fatalf("unexpected firewall rule identity fields: %v", firewallRule)
	}
	firewallProps := firewallRule["properties"].(map[string]any)
	if firewallProps["startIpAddress"] != "203.0.113.20" || firewallProps["endIpAddress"] != "203.0.113.20" {
		t.Fatalf("unexpected firewall rule properties: %v", firewallProps)
	}

	listFirewallResp, err := svc.HandleRequest(postgresCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/firewallRules?api-version=2025-08-01", nil))
	if err != nil {
		t.Fatalf("list firewall rules returned error: %v", err)
	}
	listedFirewallRules := decodePostgreSQLResponse(t, listFirewallResp)
	if len(listedFirewallRules["value"].([]any)) != 1 {
		t.Fatalf("expected one firewall rule in list, got %v", listedFirewallRules)
	}

	deleteDatabaseResp, err := svc.HandleRequest(postgresCtx(t, http.MethodDelete, databaseURL, nil))
	if err != nil {
		t.Fatalf("delete database returned error: %v", err)
	}
	if deleteDatabaseResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete database status 202, got %d; body=%s", deleteDatabaseResp.StatusCode, string(deleteDatabaseResp.RawBody))
	}

	deleteFirewallResp, err := svc.HandleRequest(postgresCtx(t, http.MethodDelete, firewallURL, nil))
	if err != nil {
		t.Fatalf("delete firewall rule returned error: %v", err)
	}
	if deleteFirewallResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete firewall rule status 202, got %d; body=%s", deleteFirewallResp.StatusCode, string(deleteFirewallResp.RawBody))
	}
}

func TestFlexibleServerDatabaseAndFirewallRuleTemplateProvisioning(t *testing.T) {
	svc := New()

	serverResource := map[string]any{
		"type":     "Microsoft.DBforPostgreSQL/flexibleServers",
		"name":     "pg-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Standard_D2s_v3", "tier": "GeneralPurpose"},
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"administratorLogin": "cloudmockadmin",
			"version":            "16",
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", serverResource); err != nil {
		t.Fatalf("provision server returned error: %v", err)
	}

	databaseResource := map[string]any{
		"type": "Microsoft.DBforPostgreSQL/flexibleServers/databases",
		"name": "pg-a/appdb",
		"properties": map[string]any{
			"charset":   "UTF8",
			"collation": "en_US.utf8",
		},
	}
	databaseResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", databaseResource)
	if err != nil {
		t.Fatalf("provision database returned error: %v", err)
	}
	database := databaseResult.(map[string]any)
	if database["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/databases/appdb" {
		t.Fatalf("unexpected provisioned database id: %v", database["id"])
	}
	if database["type"] != "Microsoft.DBforPostgreSQL/flexibleServers/databases" {
		t.Fatalf("unexpected provisioned database type: %v", database["type"])
	}

	firewallResource := map[string]any{
		"type": "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules",
		"name": "pg-a/allow-office",
		"properties": map[string]any{
			"startIpAddress": "203.0.113.20",
			"endIpAddress":   "203.0.113.20",
		},
	}
	firewallResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", firewallResource)
	if err != nil {
		t.Fatalf("provision firewall rule returned error: %v", err)
	}
	firewallRule := firewallResult.(map[string]any)
	if firewallRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DBforPostgreSQL/flexibleServers/pg-a/firewallRules/allow-office" {
		t.Fatalf("unexpected provisioned firewall rule id: %v", firewallRule["id"])
	}
	if firewallRule["type"] != "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules" {
		t.Fatalf("unexpected provisioned firewall rule type: %v", firewallRule["type"])
	}
}

func postgresCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{"Authorization": []string{"Bearer azure-token"}}
	return &service.RequestContext{
		Region:     "eastus",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func decodePostgreSQLResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
