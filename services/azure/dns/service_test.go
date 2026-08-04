package dns

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestZoneAndRecordSetLifecycle(t *testing.T) {
	svc := New()

	zoneURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com?api-version=2018-05-01"
	zonePayload := []byte(`{
		"location":"global",
		"tags":{"env":"test"},
		"properties":{"zoneType":"Public"}
	}`)

	createZoneResp, err := svc.HandleRequest(dnsCtx(t, http.MethodPut, zoneURL, zonePayload))
	if err != nil {
		t.Fatalf("create zone returned error: %v", err)
	}
	if createZoneResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create zone status 201, got %d; body=%s", createZoneResp.StatusCode, string(createZoneResp.RawBody))
	}
	createdZone := decodeDNSResponse(t, createZoneResp)
	if createdZone["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com" {
		t.Fatalf("unexpected zone id: %v", createdZone["id"])
	}
	if createdZone["name"] != "example.com" || createdZone["type"] != "Microsoft.Network/dnsZones" || createdZone["location"] != "global" {
		t.Fatalf("unexpected zone identity fields: %v", createdZone)
	}
	zoneProps := createdZone["properties"].(map[string]any)
	if zoneProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected zone properties: %v", zoneProps)
	}

	listZonesResp, err := svc.HandleRequest(dnsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones?api-version=2018-05-01", nil))
	if err != nil {
		t.Fatalf("list zones returned error: %v", err)
	}
	listedZones := decodeDNSResponse(t, listZonesResp)
	zoneValues := listedZones["value"].([]any)
	if len(zoneValues) != 1 {
		t.Fatalf("expected one zone in list, got %d in %v", len(zoneValues), listedZones)
	}

	recordSetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com/A/www?api-version=2018-05-01"
	recordSetPayload := []byte(`{
		"properties":{
			"TTL":300,
			"ARecords":[{"ipv4Address":"203.0.113.10"}],
			"metadata":{"owner":"cloudmock"}
		}
	}`)
	createRecordSetResp, err := svc.HandleRequest(dnsCtx(t, http.MethodPut, recordSetURL, recordSetPayload))
	if err != nil {
		t.Fatalf("create record set returned error: %v", err)
	}
	if createRecordSetResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create record set status 201, got %d; body=%s", createRecordSetResp.StatusCode, string(createRecordSetResp.RawBody))
	}
	createdRecordSet := decodeDNSResponse(t, createRecordSetResp)
	if createdRecordSet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com/A/www" {
		t.Fatalf("unexpected record set id: %v", createdRecordSet["id"])
	}
	if createdRecordSet["name"] != "example.com/www" || createdRecordSet["type"] != "Microsoft.Network/dnsZones/A" {
		t.Fatalf("unexpected record set identity fields: %v", createdRecordSet)
	}
	recordProps := createdRecordSet["properties"].(map[string]any)
	if recordProps["provisioningState"] != "Succeeded" || recordProps["TTL"].(float64) != 300 {
		t.Fatalf("unexpected record set properties: %v", recordProps)
	}

	listRecordSetsResp, err := svc.HandleRequest(dnsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com/recordsets?api-version=2018-05-01", nil))
	if err != nil {
		t.Fatalf("list record sets returned error: %v", err)
	}
	listedRecordSets := decodeDNSResponse(t, listRecordSetsResp)
	recordValues := listedRecordSets["value"].([]any)
	if len(recordValues) != 1 {
		t.Fatalf("expected one record set in list, got %d in %v", len(recordValues), listedRecordSets)
	}

	deleteRecordSetResp, err := svc.HandleRequest(dnsCtx(t, http.MethodDelete, recordSetURL, nil))
	if err != nil {
		t.Fatalf("delete record set returned error: %v", err)
	}
	if deleteRecordSetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete record set status 200, got %d; body=%s", deleteRecordSetResp.StatusCode, string(deleteRecordSetResp.RawBody))
	}

	deleteZoneResp, err := svc.HandleRequest(dnsCtx(t, http.MethodDelete, zoneURL, nil))
	if err != nil {
		t.Fatalf("delete zone returned error: %v", err)
	}
	if deleteZoneResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete zone status 200, got %d; body=%s", deleteZoneResp.StatusCode, string(deleteZoneResp.RawBody))
	}
}

func TestZoneRecordSetTemplateProvisioning(t *testing.T) {
	svc := New()

	zoneResource := map[string]any{
		"type":     "Microsoft.Network/dnsZones",
		"name":     "example.com",
		"location": "global",
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"zoneType": "Public",
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", zoneResource); err != nil {
		t.Fatalf("provision zone returned error: %v", err)
	}

	recordSetResource := map[string]any{
		"type": "Microsoft.Network/dnsZones/A",
		"name": "example.com/www",
		"properties": map[string]any{
			"TTL":      300,
			"ARecords": []any{map[string]any{"ipv4Address": "203.0.113.10"}},
		},
	}
	recordSetResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", recordSetResource)
	if err != nil {
		t.Fatalf("provision record set returned error: %v", err)
	}
	recordSet := recordSetResult.(map[string]any)
	if recordSet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/dnsZones/example.com/A/www" {
		t.Fatalf("unexpected provisioned record set id: %v", recordSet["id"])
	}
	if recordSet["type"] != "Microsoft.Network/dnsZones/A" {
		t.Fatalf("unexpected provisioned record set type: %v", recordSet["type"])
	}
}

func dnsCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{"Authorization": []string{"Bearer azure-token"}}
	return &service.RequestContext{
		Region:     "global",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func decodeDNSResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
