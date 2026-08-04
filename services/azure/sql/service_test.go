package sql

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestServerDatabaseAndFirewallRuleLifecycle(t *testing.T) {
	svc := New()

	serverURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a?api-version=2025-01-01"
	serverPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"administratorLogin":"cloudmockadmin",
			"minimalTlsVersion":"1.2",
			"publicNetworkAccess":"Enabled"
		}
	}`)
	createServerResp, err := svc.HandleRequest(sqlCtx(t, http.MethodPut, serverURL, serverPayload))
	if err != nil {
		t.Fatalf("create server returned error: %v", err)
	}
	if createServerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create server status 201, got %d; body=%s", createServerResp.StatusCode, string(createServerResp.RawBody))
	}
	server := decodeSQLResponse(t, createServerResp)
	if server["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a" {
		t.Fatalf("unexpected server id: %v", server["id"])
	}
	if server["name"] != "sql-a" || server["type"] != "Microsoft.Sql/servers" || server["location"] != "eastus" {
		t.Fatalf("unexpected server identity fields: %v", server)
	}
	serverProps := server["properties"].(map[string]any)
	if serverProps["state"] != "Ready" || serverProps["provisioningState"] != "Succeeded" || serverProps["fullyQualifiedDomainName"] != "sql-a.database.windows.net" {
		t.Fatalf("unexpected server properties: %v", serverProps)
	}

	listServersResp, err := svc.HandleRequest(sqlCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers?api-version=2025-01-01", nil))
	if err != nil {
		t.Fatalf("list servers returned error: %v", err)
	}
	listedServers := decodeSQLResponse(t, listServersResp)
	if len(listedServers["value"].([]any)) != 1 {
		t.Fatalf("expected one server in list, got %v", listedServers)
	}

	databaseURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/databases/db-a?api-version=2025-01-01"
	databasePayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"GP_Gen5","tier":"GeneralPurpose","capacity":2},
		"properties":{
			"collation":"SQL_Latin1_General_CP1_CI_AS",
			"maxSizeBytes":34359738368
		}
	}`)
	createDatabaseResp, err := svc.HandleRequest(sqlCtx(t, http.MethodPut, databaseURL, databasePayload))
	if err != nil {
		t.Fatalf("create database returned error: %v", err)
	}
	if createDatabaseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create database status 201, got %d; body=%s", createDatabaseResp.StatusCode, string(createDatabaseResp.RawBody))
	}
	database := decodeSQLResponse(t, createDatabaseResp)
	if database["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/databases/db-a" {
		t.Fatalf("unexpected database id: %v", database["id"])
	}
	if database["name"] != "sql-a/db-a" || database["type"] != "Microsoft.Sql/servers/databases" || database["location"] != "eastus" {
		t.Fatalf("unexpected database identity fields: %v", database)
	}
	databaseProps := database["properties"].(map[string]any)
	if databaseProps["status"] != "Online" || databaseProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected database properties: %v", databaseProps)
	}

	listDatabasesResp, err := svc.HandleRequest(sqlCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/databases?api-version=2025-01-01", nil))
	if err != nil {
		t.Fatalf("list databases returned error: %v", err)
	}
	listedDatabases := decodeSQLResponse(t, listDatabasesResp)
	if len(listedDatabases["value"].([]any)) != 1 {
		t.Fatalf("expected one database in list, got %v", listedDatabases)
	}

	firewallURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/firewallRules/allow-office?api-version=2025-01-01"
	firewallPayload := []byte(`{
		"properties":{
			"startIpAddress":"203.0.113.10",
			"endIpAddress":"203.0.113.10"
		}
	}`)
	createFirewallResp, err := svc.HandleRequest(sqlCtx(t, http.MethodPut, firewallURL, firewallPayload))
	if err != nil {
		t.Fatalf("create firewall rule returned error: %v", err)
	}
	if createFirewallResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create firewall rule status 201, got %d; body=%s", createFirewallResp.StatusCode, string(createFirewallResp.RawBody))
	}
	firewallRule := decodeSQLResponse(t, createFirewallResp)
	if firewallRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/firewallRules/allow-office" {
		t.Fatalf("unexpected firewall rule id: %v", firewallRule["id"])
	}
	if firewallRule["name"] != "sql-a/allow-office" || firewallRule["type"] != "Microsoft.Sql/servers/firewallRules" {
		t.Fatalf("unexpected firewall rule identity fields: %v", firewallRule)
	}
	firewallProps := firewallRule["properties"].(map[string]any)
	if firewallProps["startIpAddress"] != "203.0.113.10" || firewallProps["endIpAddress"] != "203.0.113.10" {
		t.Fatalf("unexpected firewall rule properties: %v", firewallProps)
	}

	listFirewallResp, err := svc.HandleRequest(sqlCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/firewallRules?api-version=2025-01-01", nil))
	if err != nil {
		t.Fatalf("list firewall rules returned error: %v", err)
	}
	listedFirewallRules := decodeSQLResponse(t, listFirewallResp)
	if len(listedFirewallRules["value"].([]any)) != 1 {
		t.Fatalf("expected one firewall rule in list, got %v", listedFirewallRules)
	}

	deleteDatabaseResp, err := svc.HandleRequest(sqlCtx(t, http.MethodDelete, databaseURL, nil))
	if err != nil {
		t.Fatalf("delete database returned error: %v", err)
	}
	if deleteDatabaseResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete database status 202, got %d; body=%s", deleteDatabaseResp.StatusCode, string(deleteDatabaseResp.RawBody))
	}

	deleteFirewallResp, err := svc.HandleRequest(sqlCtx(t, http.MethodDelete, firewallURL, nil))
	if err != nil {
		t.Fatalf("delete firewall rule returned error: %v", err)
	}
	if deleteFirewallResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete firewall rule status 202, got %d; body=%s", deleteFirewallResp.StatusCode, string(deleteFirewallResp.RawBody))
	}
}

func TestServerDatabaseAndFirewallRuleTemplateProvisioning(t *testing.T) {
	svc := New()

	serverResource := map[string]any{
		"type":     "Microsoft.Sql/servers",
		"name":     "sql-a",
		"location": "eastus",
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"administratorLogin": "cloudmockadmin",
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", serverResource); err != nil {
		t.Fatalf("provision server returned error: %v", err)
	}

	databaseResource := map[string]any{
		"type":     "Microsoft.Sql/servers/databases",
		"name":     "sql-a/db-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "GP_Gen5", "tier": "GeneralPurpose", "capacity": 2},
		"properties": map[string]any{
			"collation": "SQL_Latin1_General_CP1_CI_AS",
		},
	}
	databaseResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", databaseResource)
	if err != nil {
		t.Fatalf("provision database returned error: %v", err)
	}
	database := databaseResult.(map[string]any)
	if database["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/databases/db-a" {
		t.Fatalf("unexpected provisioned database id: %v", database["id"])
	}
	if database["type"] != "Microsoft.Sql/servers/databases" {
		t.Fatalf("unexpected provisioned database type: %v", database["type"])
	}

	firewallResource := map[string]any{
		"type": "Microsoft.Sql/servers/firewallRules",
		"name": "sql-a/allow-office",
		"properties": map[string]any{
			"startIpAddress": "203.0.113.10",
			"endIpAddress":   "203.0.113.10",
		},
	}
	firewallResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", firewallResource)
	if err != nil {
		t.Fatalf("provision firewall rule returned error: %v", err)
	}
	firewallRule := firewallResult.(map[string]any)
	if firewallRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Sql/servers/sql-a/firewallRules/allow-office" {
		t.Fatalf("unexpected provisioned firewall rule id: %v", firewallRule["id"])
	}
	if firewallRule["type"] != "Microsoft.Sql/servers/firewallRules" {
		t.Fatalf("unexpected provisioned firewall rule type: %v", firewallRule["type"])
	}
}

func sqlCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeSQLResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
