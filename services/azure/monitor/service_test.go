package monitor

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestActionGroupAndMetricAlertLifecycle(t *testing.T) {
	svc := New()

	actionGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/actionGroups/ag-a?api-version=2021-09-01"
	actionGroupPayload := []byte(`{
		"location":"Global",
		"tags":{"env":"test"},
		"properties":{
			"groupShortName":"cloudmock",
			"enabled":true,
			"emailReceivers":[{"name":"ops","emailAddress":"ops@example.com","useCommonAlertSchema":true}],
			"webhookReceivers":[{"name":"hook","serviceUri":"https://example.com/hook","useCommonAlertSchema":true}]
		}
	}`)
	createActionGroupResp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, actionGroupURL, actionGroupPayload))
	if err != nil {
		t.Fatalf("create action group returned error: %v", err)
	}
	if createActionGroupResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create action group status 201, got %d; body=%s", createActionGroupResp.StatusCode, string(createActionGroupResp.RawBody))
	}
	actionGroup := decodeMonitorResponse(t, createActionGroupResp)
	if actionGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/actionGroups/ag-a" {
		t.Fatalf("unexpected action group id: %v", actionGroup["id"])
	}
	if actionGroup["name"] != "ag-a" || actionGroup["type"] != "Microsoft.Insights/actionGroups" || actionGroup["location"] != "Global" {
		t.Fatalf("unexpected action group identity fields: %v", actionGroup)
	}
	actionGroupProps := actionGroup["properties"].(map[string]any)
	if actionGroupProps["provisioningState"] != "Succeeded" || actionGroupProps["enabled"] != true {
		t.Fatalf("unexpected action group properties: %v", actionGroupProps)
	}
	if len(actionGroupProps["emailReceivers"].([]any)) != 1 {
		t.Fatalf("expected one email receiver, got %v", actionGroupProps["emailReceivers"])
	}

	getActionGroupResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, actionGroupURL, nil))
	if err != nil {
		t.Fatalf("get action group returned error: %v", err)
	}
	if getActionGroupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get action group status 200, got %d; body=%s", getActionGroupResp.StatusCode, string(getActionGroupResp.RawBody))
	}

	listActionGroupsResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/actionGroups?api-version=2021-09-01", nil))
	if err != nil {
		t.Fatalf("list action groups returned error: %v", err)
	}
	listedActionGroups := decodeMonitorResponse(t, listActionGroupsResp)
	if len(listedActionGroups["value"].([]any)) != 1 {
		t.Fatalf("expected one action group in list, got %v", listedActionGroups)
	}

	metricAlertURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/metricAlerts/cpu-alert?api-version=2024-03-01-preview"
	metricAlertPayload := []byte(`{
		"location":"global",
		"tags":{"env":"test"},
		"properties":{
			"description":"CPU alert",
			"enabled":true,
			"severity":2,
			"scopes":["/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a"],
			"evaluationFrequency":"PT1M",
			"windowSize":"PT5M",
			"criteria":{
				"odata.type":"Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria",
				"allOf":[{"name":"cpu","metricName":"Percentage CPU","operator":"GreaterThan","threshold":80,"timeAggregation":"Average"}]
			},
			"actions":[{"actionGroupId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/actionGroups/ag-a"}]
		}
	}`)
	createMetricAlertResp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, metricAlertURL, metricAlertPayload))
	if err != nil {
		t.Fatalf("create metric alert returned error: %v", err)
	}
	if createMetricAlertResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create metric alert status 201, got %d; body=%s", createMetricAlertResp.StatusCode, string(createMetricAlertResp.RawBody))
	}
	metricAlert := decodeMonitorResponse(t, createMetricAlertResp)
	if metricAlert["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/metricAlerts/cpu-alert" {
		t.Fatalf("unexpected metric alert id: %v", metricAlert["id"])
	}
	if metricAlert["name"] != "cpu-alert" || metricAlert["type"] != "Microsoft.Insights/metricAlerts" || metricAlert["location"] != "global" {
		t.Fatalf("unexpected metric alert identity fields: %v", metricAlert)
	}
	metricAlertProps := metricAlert["properties"].(map[string]any)
	if metricAlertProps["provisioningState"] != "Succeeded" || metricAlertProps["enabled"] != true {
		t.Fatalf("unexpected metric alert properties: %v", metricAlertProps)
	}
	if len(metricAlertProps["actions"].([]any)) != 1 {
		t.Fatalf("expected one metric alert action, got %v", metricAlertProps["actions"])
	}

	getMetricAlertResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, metricAlertURL, nil))
	if err != nil {
		t.Fatalf("get metric alert returned error: %v", err)
	}
	if getMetricAlertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get metric alert status 200, got %d; body=%s", getMetricAlertResp.StatusCode, string(getMetricAlertResp.RawBody))
	}

	listMetricAlertsResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/metricAlerts?api-version=2024-03-01-preview", nil))
	if err != nil {
		t.Fatalf("list metric alerts returned error: %v", err)
	}
	listedMetricAlerts := decodeMonitorResponse(t, listMetricAlertsResp)
	if len(listedMetricAlerts["value"].([]any)) != 1 {
		t.Fatalf("expected one metric alert in list, got %v", listedMetricAlerts)
	}

	deleteMetricAlertResp, err := svc.HandleRequest(monitorCtx(t, http.MethodDelete, metricAlertURL, nil))
	if err != nil {
		t.Fatalf("delete metric alert returned error: %v", err)
	}
	if deleteMetricAlertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete metric alert status 200, got %d; body=%s", deleteMetricAlertResp.StatusCode, string(deleteMetricAlertResp.RawBody))
	}

	deleteActionGroupResp, err := svc.HandleRequest(monitorCtx(t, http.MethodDelete, actionGroupURL, nil))
	if err != nil {
		t.Fatalf("delete action group returned error: %v", err)
	}
	if deleteActionGroupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete action group status 200, got %d; body=%s", deleteActionGroupResp.StatusCode, string(deleteActionGroupResp.RawBody))
	}
}

func TestApplicationInsightsComponentLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	componentURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-a?api-version=2015-05-01"
	componentPayload := []byte(`{
		"location":"eastus",
		"kind":"web",
		"tags":{"env":"test"},
		"properties":{
			"Application_Type":"web",
			"Flow_Type":"Bluefield",
			"Request_Source":"rest",
			"SamplingPercentage":75
		}
	}`)
	createResp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, componentURL, componentPayload))
	if err != nil {
		t.Fatalf("create Application Insights component returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create component status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	component := decodeMonitorResponse(t, createResp)
	if component["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-a" {
		t.Fatalf("unexpected component id: %v", component["id"])
	}
	if component["name"] != "appi-a" || component["type"] != "Microsoft.Insights/components" || component["location"] != "eastus" || component["kind"] != "web" {
		t.Fatalf("unexpected component identity fields: %v", component)
	}
	props := component["properties"].(map[string]any)
	if props["ApplicationId"] != "appi-a" || props["Application_Type"] != "web" || props["Flow_Type"] != "Bluefield" || props["Request_Source"] != "rest" {
		t.Fatalf("unexpected component properties: %v", props)
	}
	instrumentationKey := props["InstrumentationKey"].(string)
	if instrumentationKey == "" || props["ConnectionString"] != "InstrumentationKey="+instrumentationKey {
		t.Fatalf("expected deterministic instrumentation key connection string, got %v", props)
	}
	if props["AppId"] == "" || props["TenantId"] == "" || props["provisioningState"] != "Succeeded" || props["SamplingPercentage"] != float64(75) || props["RetentionInDays"] != float64(90) || props["IngestionMode"] != "ApplicationInsights" {
		t.Fatalf("unexpected component defaults: %v", props)
	}

	updateResp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, componentURL, []byte(`{
		"location":"westus2",
		"kind":"web",
		"tags":{"env":"prod"},
		"properties":{"Application_Type":"web"}
	}`)))
	if err != nil {
		t.Fatalf("update Application Insights component returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update component status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeMonitorResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if updatedProps["InstrumentationKey"] != instrumentationKey || updatedProps["ApplicationId"] != "appi-a" {
		t.Fatalf("expected read-only component IDs to be stable across update, got %v", updatedProps)
	}

	getResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, componentURL, nil))
	if err != nil {
		t.Fatalf("get Application Insights component returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get component status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	createOther := func(rawURL string) {
		t.Helper()
		resp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, rawURL, []byte(`{"location":"eastus","kind":"web","properties":{"Application_Type":"web"}}`)))
		if err != nil {
			t.Fatalf("create component %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected create component 200 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}
	createOther("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Insights/components/appi-b?api-version=2015-05-01")
	createOther("https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.Insights/components/appi-z?api-version=2015-05-01")

	listResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights/components?api-version=2015-05-01", nil))
	if err != nil {
		t.Fatalf("list Application Insights components returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list components status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeMonitorResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "appi-a" || values[1].(map[string]any)["name"] != "appi-b" {
		t.Fatalf("expected sub-1 components sorted by name, got %v", listed)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Insights/components",
		"name":     "appi-template",
		"location": "eastus",
		"kind":     "web",
		"properties": map[string]any{
			"Application_Type": "web",
		},
	})
	if err != nil {
		t.Fatalf("provision Application Insights component returned error: %v", err)
	}
	templateComponent := templateResult.(map[string]any)
	if templateComponent["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/components/appi-template" {
		t.Fatalf("unexpected provisioned component id: %v", templateComponent["id"])
	}

	deleteResp, err := svc.HandleRequest(monitorCtx(t, http.MethodDelete, componentURL, nil))
	if err != nil {
		t.Fatalf("delete Application Insights component returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete component status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleteMissingResp, err := svc.HandleRequest(monitorCtx(t, http.MethodDelete, componentURL, nil))
	if err != nil {
		t.Fatalf("delete missing Application Insights component returned error: %v", err)
	}
	if deleteMissingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing component status 204, got %d; body=%s", deleteMissingResp.StatusCode, string(deleteMissingResp.RawBody))
	}
}

func TestApplicationInsightsQueryGet(t *testing.T) {
	svc := New()

	resp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://api.applicationinsights.io/v1/apps/app-123/query?query=requests%20%7C%20take%201&timespan=PT12H", nil))
	if err != nil {
		t.Fatalf("Application Insights query returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	out := decodeMonitorResponse(t, resp)
	tables := out["tables"].([]any)
	if len(tables) != 1 {
		t.Fatalf("expected one query result table, got %v", out)
	}
	table := tables[0].(map[string]any)
	if table["name"] != "PrimaryResult" {
		t.Fatalf("expected PrimaryResult table, got %v", table)
	}
	columns := table["columns"].([]any)
	rows := table["rows"].([]any)
	if len(columns) == 0 || len(rows) != 1 {
		t.Fatalf("expected request query columns and one row, got %v", table)
	}
	if columns[0].(map[string]any)["name"] != "timestamp" || columns[0].(map[string]any)["type"] != "datetime" {
		t.Fatalf("unexpected first query column: %v", columns[0])
	}
	firstRow := rows[0].([]any)
	if len(firstRow) != len(columns) {
		t.Fatalf("expected row width %d, got %d: %v", len(columns), len(firstRow), firstRow)
	}
	if firstRow[len(firstRow)-2] != "app-123" || firstRow[len(firstRow)-1] != "request" {
		t.Fatalf("expected deterministic app id and item type in row, got %v", firstRow)
	}

	missingQueryResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://api.applicationinsights.io/v1/apps/app-123/query", nil))
	if err != nil {
		t.Fatalf("missing query request returned error: %v", err)
	}
	if missingQueryResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing query status 400, got %d; body=%s", missingQueryResp.StatusCode, string(missingQueryResp.RawBody))
	}
}

func TestMonitorTemplateProvisioning(t *testing.T) {
	svc := New()

	actionGroupResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Insights/actionGroups",
		"name":     "ag-a",
		"location": "Global",
		"properties": map[string]any{
			"groupShortName": "cloudmock",
			"enabled":        true,
		},
	})
	if err != nil {
		t.Fatalf("provision action group returned error: %v", err)
	}
	actionGroup := actionGroupResult.(map[string]any)
	if actionGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/actionGroups/ag-a" {
		t.Fatalf("unexpected provisioned action group id: %v", actionGroup["id"])
	}

	metricAlertResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Insights/metricAlerts",
		"name":     "cpu-alert",
		"location": "global",
		"properties": map[string]any{
			"enabled":             true,
			"severity":            2,
			"scopes":              []any{"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a"},
			"evaluationFrequency": "PT1M",
			"windowSize":          "PT5M",
			"criteria":            map[string]any{"odata.type": "Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria"},
		},
	})
	if err != nil {
		t.Fatalf("provision metric alert returned error: %v", err)
	}
	metricAlert := metricAlertResult.(map[string]any)
	if metricAlert["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Insights/metricAlerts/cpu-alert" {
		t.Fatalf("unexpected provisioned metric alert id: %v", metricAlert["id"])
	}
}

func TestDiagnosticSettingLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	scopeID := "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a"
	settingURL := "https://management.azure.com" + scopeID + "/providers/Microsoft.Insights/diagnosticSettings/default?api-version=2021-05-01-preview"
	settingPayload := []byte(`{
		"properties":{
			"workspaceId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a",
			"storageAccountId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/stgacct",
			"logs":[{"category":"Administrative","enabled":true,"retentionPolicy":{"enabled":false,"days":0}}],
			"metrics":[{"category":"AllMetrics","enabled":true,"retentionPolicy":{"enabled":false,"days":0}}],
			"logAnalyticsDestinationType":"Dedicated"
		}
	}`)

	createResp, err := svc.HandleRequest(monitorCtx(t, http.MethodPut, settingURL, settingPayload))
	if err != nil {
		t.Fatalf("create diagnostic setting returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create diagnostic setting status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	setting := decodeMonitorResponse(t, createResp)
	if setting["id"] != scopeID+"/providers/Microsoft.Insights/diagnosticSettings/default" {
		t.Fatalf("unexpected diagnostic setting id: %v", setting["id"])
	}
	if setting["name"] != "default" || setting["type"] != "Microsoft.Insights/diagnosticSettings" {
		t.Fatalf("unexpected diagnostic setting identity fields: %v", setting)
	}
	props := setting["properties"].(map[string]any)
	if props["workspaceId"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a" {
		t.Fatalf("unexpected workspaceId: %v", props["workspaceId"])
	}
	if len(props["logs"].([]any)) != 1 || len(props["metrics"].([]any)) != 1 {
		t.Fatalf("expected one log and one metric setting, got %v", props)
	}

	getResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, settingURL, nil))
	if err != nil {
		t.Fatalf("get diagnostic setting returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get diagnostic setting status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com"+scopeID+"/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview", nil))
	if err != nil {
		t.Fatalf("list diagnostic settings returned error: %v", err)
	}
	listed := decodeMonitorResponse(t, listResp)
	if len(listed["value"].([]any)) != 1 {
		t.Fatalf("expected one diagnostic setting in list, got %v", listed)
	}

	deleteResp, err := svc.HandleRequest(monitorCtx(t, http.MethodDelete, settingURL, nil))
	if err != nil {
		t.Fatalf("delete diagnostic setting returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete diagnostic setting status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":  "Microsoft.Insights/diagnosticSettings",
		"name":  "default",
		"scope": scopeID,
		"properties": map[string]any{
			"workspaceId":                 "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a",
			"storageAccountId":            "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/stgacct",
			"logAnalyticsDestinationType": "Dedicated",
			"logs": []any{
				map[string]any{"category": "Administrative", "enabled": true, "retentionPolicy": map[string]any{"enabled": false, "days": float64(0)}},
			},
			"metrics": []any{
				map[string]any{"category": "AllMetrics", "enabled": true, "retentionPolicy": map[string]any{"enabled": false, "days": float64(0)}},
			},
		},
	})
	if err != nil {
		t.Fatalf("provision diagnostic setting returned error: %v", err)
	}
	templateSetting := templateResult.(map[string]any)
	if templateSetting["id"] != scopeID+"/providers/Microsoft.Insights/diagnosticSettings/default" {
		t.Fatalf("unexpected provisioned diagnostic setting id: %v", templateSetting["id"])
	}
}

func TestMetricDefinitionsAndMetricsList(t *testing.T) {
	svc := New()

	scopeID := "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a"
	definitionsResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com"+scopeID+"/providers/Microsoft.Insights/metricDefinitions?api-version=2023-10-01", nil))
	if err != nil {
		t.Fatalf("list metric definitions returned error: %v", err)
	}
	if definitionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected metric definitions status 200, got %d; body=%s", definitionsResp.StatusCode, string(definitionsResp.RawBody))
	}
	definitions := decodeMonitorResponse(t, definitionsResp)
	definitionValues := definitions["value"].([]any)
	if len(definitionValues) == 0 {
		t.Fatalf("expected metric definitions, got %v", definitions)
	}
	cpuDefinition := definitionValues[0].(map[string]any)
	cpuName := cpuDefinition["name"].(map[string]any)
	if cpuName["value"] != "Percentage CPU" || cpuDefinition["namespace"] != "Microsoft.Compute/virtualMachines" || cpuDefinition["unit"] != "Percent" || cpuDefinition["primaryAggregationType"] != "Average" {
		t.Fatalf("unexpected cpu metric definition: %v", cpuDefinition)
	}

	metricsURL := "https://management.azure.com" + scopeID + "/providers/Microsoft.Insights/metrics?api-version=2023-10-01&metricnames=Percentage%20CPU&timespan=2026-06-16T00:00:00Z/2026-06-16T01:00:00Z&interval=PT1M"
	metricsResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, metricsURL, nil))
	if err != nil {
		t.Fatalf("list metrics returned error: %v", err)
	}
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d; body=%s", metricsResp.StatusCode, string(metricsResp.RawBody))
	}
	metrics := decodeMonitorResponse(t, metricsResp)
	if metrics["timespan"] != "2026-06-16T00:00:00Z/2026-06-16T01:00:00Z" || metrics["interval"] != "PT1M" {
		t.Fatalf("unexpected metrics query projection: %v", metrics)
	}
	metricValues := metrics["value"].([]any)
	if len(metricValues) != 1 {
		t.Fatalf("expected one metric value, got %v", metrics)
	}
	cpuMetric := metricValues[0].(map[string]any)
	metricName := cpuMetric["name"].(map[string]any)
	if metricName["value"] != "Percentage CPU" || cpuMetric["type"] != "Microsoft.Insights/metrics" || cpuMetric["unit"] != "Percent" {
		t.Fatalf("unexpected cpu metric: %v", cpuMetric)
	}
	timeseries := cpuMetric["timeseries"].([]any)
	data := timeseries[0].(map[string]any)["data"].([]any)
	firstPoint := data[0].(map[string]any)
	if firstPoint["average"] != float64(0) {
		t.Fatalf("expected deterministic zero average, got %v", firstPoint)
	}
}

func TestActivityLogsListAndSelect(t *testing.T) {
	svc := New()

	filter := "eventTimestamp ge '2026-06-16T00:00:00Z' and eventTimestamp le '2026-06-17T00:00:00Z' and resourceGroupName eq 'rg-a'"
	rawURL := "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights/eventtypes/management/values?api-version=2015-04-01&$filter=" + url.QueryEscape(filter)
	resp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, rawURL, nil))
	if err != nil {
		t.Fatalf("list activity logs returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected activity logs status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	out := decodeMonitorResponse(t, resp)
	values := out["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one activity log event, got %v", out)
	}
	event := values[0].(map[string]any)
	if event["subscriptionId"] != "sub-1" || event["resourceGroupName"] != "rg-a" || event["level"] != "Informational" {
		t.Fatalf("unexpected activity log identity fields: %v", event)
	}
	operationName := event["operationName"].(map[string]any)
	status := event["status"].(map[string]any)
	resourceProvider := event["resourceProviderName"].(map[string]any)
	if operationName["value"] != "Microsoft.Resources/deployments/write" || status["value"] != "Succeeded" || resourceProvider["value"] != "Microsoft.Resources" {
		t.Fatalf("unexpected activity log localized fields: %v", event)
	}
	authorization := event["authorization"].(map[string]any)
	if authorization["action"] != "Microsoft.Resources/deployments/write" || authorization["scope"] == "" {
		t.Fatalf("unexpected activity log authorization: %v", authorization)
	}

	selectURL := rawURL + "&$select=" + url.QueryEscape("eventName,id,resourceGroupName,operationName,status,eventTimestamp")
	selectResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, selectURL, nil))
	if err != nil {
		t.Fatalf("selected activity logs returned error: %v", err)
	}
	if selectResp.StatusCode != http.StatusOK {
		t.Fatalf("expected selected activity logs status 200, got %d; body=%s", selectResp.StatusCode, string(selectResp.RawBody))
	}
	selected := decodeMonitorResponse(t, selectResp)
	selectedEvent := selected["value"].([]any)[0].(map[string]any)
	if _, ok := selectedEvent["authorization"]; ok {
		t.Fatalf("expected selected activity log to omit authorization, got %v", selectedEvent)
	}
	if selectedEvent["resourceGroupName"] != "rg-a" || selectedEvent["operationName"] == nil || selectedEvent["status"] == nil {
		t.Fatalf("selected activity log omitted requested fields: %v", selectedEvent)
	}

	missingFilterResp, err := svc.HandleRequest(monitorCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights/eventtypes/management/values?api-version=2015-04-01", nil))
	if err != nil {
		t.Fatalf("missing filter activity logs returned error: %v", err)
	}
	if missingFilterResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing filter status 400, got %d; body=%s", missingFilterResp.StatusCode, string(missingFilterResp.RawBody))
	}
}

func TestMonitorServiceKeysIncludeVersionedResources(t *testing.T) {
	svc := New()

	seen := make(map[string]bool)
	for _, key := range svc.ServiceKeys() {
		seen[string(key.Provider)+"|"+key.Service+"|"+key.APIVersion] = true
	}

	for _, expected := range []string{
		"azure|Microsoft.Insights/actionGroups|2021-09-01",
		"azure|Microsoft.Insights/actionGroups|2019-06-01",
		"azure|Microsoft.Insights/metricAlerts|2024-03-01-preview",
		"azure|Microsoft.Insights/metricAlerts|2018-03-01",
		"azure|Microsoft.Insights/diagnosticSettings|2021-05-01-preview",
		"azure|Microsoft.Insights/metrics|2023-10-01",
		"azure|Microsoft.Insights/metricDefinitions|2023-10-01",
		"azure|Microsoft.Insights/components|2015-05-01",
		"azure|Microsoft.Insights/query|v1",
		"azure|Microsoft.Insights/eventtypes|2015-04-01",
	} {
		if !seen[expected] {
			t.Fatalf("expected service key %s", expected)
		}
	}
}

func monitorCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeMonitorResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
