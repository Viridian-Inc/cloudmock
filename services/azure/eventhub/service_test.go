package eventhub

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestNamespaceAndEventHubLifecycle(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	namespacePayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Standard","capacity":1},
		"tags":{"env":"test"},
		"properties":{
			"zoneRedundant":true,
			"kafkaEnabled":true,
			"publicNetworkAccess":"Enabled"
		}
	}`)

	createNamespaceResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, namespacePayload))
	if err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	}
	if createNamespaceResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", createNamespaceResp.StatusCode, string(createNamespaceResp.RawBody))
	}
	createdNamespace := decodeEventHubResponse(t, createNamespaceResp)
	if createdNamespace["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a" {
		t.Fatalf("unexpected namespace id: %v", createdNamespace["id"])
	}
	if createdNamespace["name"] != "namespace-a" || createdNamespace["type"] != "Microsoft.EventHub/namespaces" || createdNamespace["location"] != "eastus" {
		t.Fatalf("unexpected namespace identity fields: %v", createdNamespace)
	}
	namespaceProps := createdNamespace["properties"].(map[string]any)
	if namespaceProps["provisioningState"] != "Succeeded" || namespaceProps["status"] != "Active" {
		t.Fatalf("unexpected namespace lifecycle properties: %v", namespaceProps)
	}
	if namespaceProps["serviceBusEndpoint"] != "https://namespace-a.servicebus.windows.net:443/" {
		t.Fatalf("unexpected namespace service bus endpoint: %v", namespaceProps["serviceBusEndpoint"])
	}

	getNamespaceResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, namespaceURL, nil))
	if err != nil {
		t.Fatalf("get namespace returned error: %v", err)
	}
	if getNamespaceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get namespace status 200, got %d; body=%s", getNamespaceResp.StatusCode, string(getNamespaceResp.RawBody))
	}

	listNamespacesResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces?api-version=2026-01-01", nil))
	if err != nil {
		t.Fatalf("list namespaces returned error: %v", err)
	}
	listedNamespaces := decodeEventHubResponse(t, listNamespacesResp)
	namespaceValues := listedNamespaces["value"].([]any)
	if len(namespaceValues) != 1 {
		t.Fatalf("expected one namespace in list, got %d in %v", len(namespaceValues), listedNamespaces)
	}

	eventHubURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a?api-version=2026-01-01"
	eventHubPayload := []byte(`{
		"properties":{
			"partitionCount":4,
			"messageRetentionInDays":3,
			"captureDescription":{
				"enabled":true,
				"encoding":"Avro"
			}
		}
	}`)
	createEventHubResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, eventHubURL, eventHubPayload))
	if err != nil {
		t.Fatalf("create event hub returned error: %v", err)
	}
	if createEventHubResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event hub status 201, got %d; body=%s", createEventHubResp.StatusCode, string(createEventHubResp.RawBody))
	}
	createdEventHub := decodeEventHubResponse(t, createEventHubResp)
	if createdEventHub["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a" {
		t.Fatalf("unexpected event hub id: %v", createdEventHub["id"])
	}
	if createdEventHub["name"] != "namespace-a/hub-a" || createdEventHub["type"] != "Microsoft.EventHub/namespaces/eventhubs" {
		t.Fatalf("unexpected event hub identity fields: %v", createdEventHub)
	}
	eventHubProps := createdEventHub["properties"].(map[string]any)
	if eventHubProps["provisioningState"] != "Succeeded" || eventHubProps["status"] != "Active" {
		t.Fatalf("unexpected event hub lifecycle properties: %v", eventHubProps)
	}
	if eventHubProps["partitionCount"].(float64) != 4 {
		t.Fatalf("unexpected event hub partition count: %v", eventHubProps["partitionCount"])
	}

	getEventHubResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, eventHubURL, nil))
	if err != nil {
		t.Fatalf("get event hub returned error: %v", err)
	}
	if getEventHubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get event hub status 200, got %d; body=%s", getEventHubResp.StatusCode, string(getEventHubResp.RawBody))
	}

	listEventHubsResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs?api-version=2026-01-01", nil))
	if err != nil {
		t.Fatalf("list event hubs returned error: %v", err)
	}
	listedEventHubs := decodeEventHubResponse(t, listEventHubsResp)
	eventHubValues := listedEventHubs["value"].([]any)
	if len(eventHubValues) != 1 {
		t.Fatalf("expected one event hub in list, got %d in %v", len(eventHubValues), listedEventHubs)
	}

	deleteEventHubResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, eventHubURL, nil))
	if err != nil {
		t.Fatalf("delete event hub returned error: %v", err)
	}
	if deleteEventHubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete event hub status 200, got %d; body=%s", deleteEventHubResp.StatusCode, string(deleteEventHubResp.RawBody))
	}
	deleteMissingEventHubResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, eventHubURL, nil))
	if err != nil {
		t.Fatalf("delete missing event hub returned error: %v", err)
	}
	if deleteMissingEventHubResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing event hub status 204, got %d; body=%s", deleteMissingEventHubResp.StatusCode, string(deleteMissingEventHubResp.RawBody))
	}

	deleteNamespaceResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, namespaceURL, nil))
	if err != nil {
		t.Fatalf("delete namespace returned error: %v", err)
	}
	if deleteNamespaceResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete namespace status 202, got %d; body=%s", deleteNamespaceResp.StatusCode, string(deleteNamespaceResp.RawBody))
	}
}

func TestNamespaceEventHubTemplateProvisioning(t *testing.T) {
	svc := New()

	namespaceResource := map[string]any{
		"type":     "Microsoft.EventHub/namespaces",
		"name":     "namespace-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Standard", "tier": "Standard", "capacity": 1},
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"kafkaEnabled": true,
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", namespaceResource); err != nil {
		t.Fatalf("provision namespace returned error: %v", err)
	}

	eventHubResource := map[string]any{
		"type": "Microsoft.EventHub/namespaces/eventhubs",
		"name": "namespace-a/hub-a",
		"properties": map[string]any{
			"partitionCount":         4,
			"messageRetentionInDays": 3,
		},
	}
	eventHubResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", eventHubResource)
	if err != nil {
		t.Fatalf("provision event hub returned error: %v", err)
	}
	eventHub := eventHubResult.(map[string]any)
	if eventHub["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a" {
		t.Fatalf("unexpected provisioned event hub id: %v", eventHub["id"])
	}
	if eventHub["type"] != "Microsoft.EventHub/namespaces/eventhubs" {
		t.Fatalf("unexpected provisioned event hub type: %v", eventHub["type"])
	}

	namespaceRuleResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.EventHub/namespaces/authorizationRules",
		"name": "namespace-a/ns-rule-a",
		"properties": map[string]any{
			"rights": []any{"Listen", "Send"},
		},
	})
	if err != nil {
		t.Fatalf("provision namespace authorization rule returned error: %v", err)
	}
	namespaceRule := namespaceRuleResult.(map[string]any)
	if namespaceRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules/ns-rule-a" {
		t.Fatalf("unexpected provisioned namespace authorization rule id: %v", namespaceRule["id"])
	}
	if namespaceRule["type"] != "Microsoft.EventHub/Namespaces/AuthorizationRules" {
		t.Fatalf("unexpected provisioned namespace authorization rule type: %v", namespaceRule["type"])
	}

	eventHubRuleResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.EventHub/namespaces/eventhubs/authorizationRules",
		"name": "namespace-a/hub-a/hub-rule-a",
		"properties": map[string]any{
			"rights": []any{"Listen", "Send"},
		},
	})
	if err != nil {
		t.Fatalf("provision event hub authorization rule returned error: %v", err)
	}
	eventHubRule := eventHubRuleResult.(map[string]any)
	if eventHubRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules/hub-rule-a" {
		t.Fatalf("unexpected provisioned event hub authorization rule id: %v", eventHubRule["id"])
	}
	if eventHubRule["type"] != "Microsoft.EventHub/Namespaces/EventHubs/AuthorizationRules" {
		t.Fatalf("unexpected provisioned event hub authorization rule type: %v", eventHubRule["type"])
	}

	consumerGroupResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.EventHub/namespaces/eventhubs/consumergroups",
		"name": "namespace-a/hub-a/cg-template",
		"properties": map[string]any{
			"userMetadata": "template-team",
		},
	})
	if err != nil {
		t.Fatalf("provision consumer group returned error: %v", err)
	}
	consumerGroup := consumerGroupResult.(map[string]any)
	if consumerGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/consumergroups/cg-template" {
		t.Fatalf("unexpected provisioned consumer group id: %v", consumerGroup["id"])
	}
	if consumerGroup["type"] != "Microsoft.EventHub/Namespaces/EventHubs/ConsumerGroups" {
		t.Fatalf("unexpected provisioned consumer group type: %v", consumerGroup["type"])
	}
	consumerGroupProps := consumerGroup["properties"].(map[string]any)
	if consumerGroupProps["userMetadata"] != "template-team" {
		t.Fatalf("unexpected provisioned consumer group properties: %v", consumerGroupProps)
	}
}

func TestRuntimeSendEventStoresEventWithUserProperties(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	eventHubURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, eventHubURL, []byte(`{"properties":{"partitionCount":2}}`))); err != nil {
		t.Fatalf("create event hub returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event hub status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ctx := eventHubCtx(t, http.MethodPost, "https://namespace-a.servicebus.windows.net/hub-a/messages?timeout=60&api-version=2014-01", []byte(`{"DeviceId":"dev-01","Temperature":"37.0"}`))
	ctx.RawRequest.Header.Set("Content-Type", "application/atom+xml;type=entry;charset=utf-8")
	ctx.RawRequest.Header.Set("Alert", "Strong Wind")
	sendResp, err := svc.HandleRequest(ctx)
	if err != nil {
		t.Fatalf("send event returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send event status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}
	if sendResp.RawContentType != "application/xml; charset=utf-8" {
		t.Fatalf("expected Event Hubs send content type, got %q", sendResp.RawContentType)
	}

	svc.mu.RLock()
	events := append([]runtimeEvent(nil), svc.runtimeEvents[runtimeEventHubKey("namespace-a", "hub-a")]...)
	svc.mu.RUnlock()
	if len(events) != 1 {
		t.Fatalf("expected one stored runtime event, got %d in %v", len(events), events)
	}
	if string(events[0].Body) != `{"DeviceId":"dev-01","Temperature":"37.0"}` {
		t.Fatalf("unexpected runtime event body: %q", string(events[0].Body))
	}
	if events[0].UserProperties["Alert"] != "Strong Wind" {
		t.Fatalf("expected Alert user property, got %v", events[0].UserProperties)
	}
}

func TestRuntimeSendBatchEventsStoresEachEntryWithBodyUserProperties(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	eventHubURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, eventHubURL, []byte(`{"properties":{"partitionCount":2}}`))); err != nil {
		t.Fatalf("create event hub returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event hub status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	batch := []byte(`[{"Body":"Message1","UserProperties":{"Alert":"Strong Wind"}},{"Body":"Message2"}]`)
	ctx := eventHubCtx(t, http.MethodPost, "https://namespace-a.servicebus.windows.net/hub-a/messages?timeout=60&api-version=2014-01", batch)
	ctx.RawRequest.Header.Set("Content-Type", "application/vnd.microsoft.servicebus.json")
	ctx.RawRequest.Header.Set("Alert", "Ignored Batch Header")
	sendResp, err := svc.HandleRequest(ctx)
	if err != nil {
		t.Fatalf("send batch events returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send batch status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}
	if len(sendResp.RawBody) != 0 {
		t.Fatalf("expected empty send batch response body, got %q", string(sendResp.RawBody))
	}

	svc.mu.RLock()
	events := append([]runtimeEvent(nil), svc.runtimeEvents[runtimeEventHubKey("namespace-a", "hub-a")]...)
	svc.mu.RUnlock()
	if len(events) != 2 {
		t.Fatalf("expected two stored runtime events from batch, got %d in %v", len(events), events)
	}
	if string(events[0].Body) != "Message1" || events[0].UserProperties["Alert"] != "Strong Wind" {
		t.Fatalf("unexpected first batch event: body=%q userProperties=%v", string(events[0].Body), events[0].UserProperties)
	}
	if string(events[1].Body) != "Message2" || len(events[1].UserProperties) != 0 {
		t.Fatalf("unexpected second batch event: body=%q userProperties=%v", string(events[1].Body), events[1].UserProperties)
	}
}

func TestRuntimeActionsExposeBatchSend(t *testing.T) {
	svc := New()
	actions := map[string]string{}
	for _, action := range svc.Actions() {
		actions[action.Name] = action.Method
	}

	if actions["SendEvent"] != http.MethodPost {
		t.Fatalf("expected SendEvent action to use POST, got %q in %v", actions["SendEvent"], actions)
	}
	if actions["SendBatchEvents"] != http.MethodPost {
		t.Fatalf("expected SendBatchEvents action to use POST, got %q in %v", actions["SendBatchEvents"], actions)
	}
}

func TestConsumerGroupLifecycle(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	eventHubURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, eventHubURL, []byte(`{"properties":{"partitionCount":2}}`))); err != nil {
		t.Fatalf("create event hub returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event hub status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/consumergroups?api-version=2026-01-01"
	listDefaultResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list default consumer groups returned error: %v", err)
	}
	if listDefaultResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list default consumer groups status 200, got %d; body=%s", listDefaultResp.StatusCode, string(listDefaultResp.RawBody))
	}
	defaultList := decodeEventHubResponse(t, listDefaultResp)
	defaultValues := defaultList["value"].([]any)
	if len(defaultValues) != 1 || defaultValues[0].(map[string]any)["name"] != "$Default" {
		t.Fatalf("expected $Default consumer group after event hub create, got %v", defaultList)
	}

	consumerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/consumergroups/cg-a?api-version=2026-01-01"
	createResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, consumerGroupURL, []byte(`{"properties":{"userMetadata":"team-a"}}`)))
	if err != nil {
		t.Fatalf("create consumer group returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create consumer group status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeEventHubResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/consumergroups/cg-a" {
		t.Fatalf("unexpected consumer group id: %v", created["id"])
	}
	if created["name"] != "cg-a" || created["type"] != "Microsoft.EventHub/Namespaces/EventHubs/ConsumerGroups" {
		t.Fatalf("unexpected consumer group identity: %v", created)
	}
	createdProps := created["properties"].(map[string]any)
	if createdProps["userMetadata"] != "team-a" || createdProps["createdAt"] == "" || createdProps["updatedAt"] == "" {
		t.Fatalf("unexpected consumer group properties: %v", createdProps)
	}

	getResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, consumerGroupURL, nil))
	if err != nil {
		t.Fatalf("get consumer group returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get consumer group status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	updateResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, consumerGroupURL, []byte(`{"properties":{"userMetadata":"team-b"}}`)))
	if err != nil {
		t.Fatalf("update consumer group returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update consumer group status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeEventHubResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if updatedProps["userMetadata"] != "team-b" || updatedProps["createdAt"] != createdProps["createdAt"] {
		t.Fatalf("expected metadata update with stable createdAt, before=%v after=%v", createdProps, updatedProps)
	}

	listResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list consumer groups returned error: %v", err)
	}
	listed := decodeEventHubResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two consumer groups, got %d in %v", len(values), listed)
	}

	deleteResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, consumerGroupURL, nil))
	if err != nil {
		t.Fatalf("delete consumer group returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete consumer group status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleteMissingResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, consumerGroupURL, nil))
	if err != nil {
		t.Fatalf("delete missing consumer group returned error: %v", err)
	}
	if deleteMissingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing consumer group status 204, got %d; body=%s", deleteMissingResp.StatusCode, string(deleteMissingResp.RawBody))
	}
}

func TestNamespaceAuthorizationRulesKeysAndRegeneration(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules?api-version=2026-01-01"
	listDefaultResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list default authorization rules returned error: %v", err)
	}
	if listDefaultResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list default authorization rules status 200, got %d; body=%s", listDefaultResp.StatusCode, string(listDefaultResp.RawBody))
	}
	defaultRules := decodeEventHubResponse(t, listDefaultResp)
	defaultValues := defaultRules["value"].([]any)
	if len(defaultValues) != 1 || defaultValues[0].(map[string]any)["name"] != "RootManageSharedAccessKey" {
		t.Fatalf("expected default RootManageSharedAccessKey rule, got %v", defaultRules)
	}

	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules/rule-a?api-version=2026-01-01"
	createResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, ruleURL, []byte(`{"properties":{"rights":["Listen","Send"]}}`)))
	if err != nil {
		t.Fatalf("create authorization rule returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create authorization rule status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeEventHubResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules/rule-a" {
		t.Fatalf("unexpected authorization rule id: %v", created["id"])
	}
	if created["name"] != "rule-a" || created["type"] != "Microsoft.EventHub/Namespaces/AuthorizationRules" {
		t.Fatalf("unexpected authorization rule identity: %v", created)
	}
	rights := created["properties"].(map[string]any)["rights"].([]any)
	if len(rights) != 2 || rights[0] != "Listen" || rights[1] != "Send" {
		t.Fatalf("unexpected authorization rule rights: %v", rights)
	}

	getResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get authorization rule returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get authorization rule status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list authorization rules returned error: %v", err)
	}
	listed := decodeEventHubResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 2 {
		t.Fatalf("expected two authorization rules, got %d in %v", len(values), listed)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules/rule-a/listKeys?api-version=2026-01-01"
	listKeysResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventHubResponse(t, listKeysResp)
	primaryKey := keys["primaryKey"].(string)
	secondaryKey := keys["secondaryKey"].(string)
	if keys["keyName"] != "rule-a" || primaryKey == "" || secondaryKey == "" || primaryKey == secondaryKey {
		t.Fatalf("unexpected access keys: %v", keys)
	}
	if keys["primaryConnectionString"] != "Endpoint=sb://namespace-a.servicebus.windows.net/;SharedAccessKeyName=rule-a;SharedAccessKey="+primaryKey {
		t.Fatalf("unexpected primary connection string: %v", keys["primaryConnectionString"])
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/authorizationRules/rule-a/regenerateKeys?api-version=2026-01-01"
	regenerateResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"PrimaryKey"}`)))
	if err != nil {
		t.Fatalf("regenerate primary key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate primary status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotated := decodeEventHubResponse(t, regenerateResp)
	if rotated["primaryKey"] == primaryKey || rotated["secondaryKey"] != secondaryKey {
		t.Fatalf("expected primary key rotation only, before=%v after=%v", keys, rotated)
	}

	regenerateSecondaryResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"SecondaryKey","key":"manual-secondary"}`)))
	if err != nil {
		t.Fatalf("regenerate secondary key returned error: %v", err)
	}
	if regenerateSecondaryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate secondary status 200, got %d; body=%s", regenerateSecondaryResp.StatusCode, string(regenerateSecondaryResp.RawBody))
	}
	manual := decodeEventHubResponse(t, regenerateSecondaryResp)
	if manual["secondaryKey"] != "manual-secondary" {
		t.Fatalf("expected manual secondary key, got %v", manual)
	}

	deleteResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete authorization rule returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete authorization rule status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleteMissingResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete missing authorization rule returned error: %v", err)
	}
	if deleteMissingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing authorization rule status 204, got %d; body=%s", deleteMissingResp.StatusCode, string(deleteMissingResp.RawBody))
	}
}

func TestEventHubAuthorizationRulesKeysAndRegeneration(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	eventHubURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a?api-version=2026-01-01"
	if resp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, eventHubURL, []byte(`{"properties":{"partitionCount":2}}`))); err != nil {
		t.Fatalf("create event hub returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event hub status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules/rule-a?api-version=2026-01-01"
	createResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPut, ruleURL, []byte(`{"properties":{"rights":["Listen","Send"]}}`)))
	if err != nil {
		t.Fatalf("create event hub authorization rule returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create event hub authorization rule status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeEventHubResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules/rule-a" {
		t.Fatalf("unexpected event hub authorization rule id: %v", created["id"])
	}
	if created["name"] != "rule-a" || created["type"] != "Microsoft.EventHub/Namespaces/EventHubs/AuthorizationRules" {
		t.Fatalf("unexpected event hub authorization rule identity: %v", created)
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules?api-version=2026-01-01"
	listResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list event hub authorization rules returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list event hub authorization rules status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeEventHubResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 1 {
		t.Fatalf("expected one event hub authorization rule, got %d in %v", len(values), listed)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules/rule-a/listKeys?api-version=2026-01-01"
	listKeysResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list event hub keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list event hub keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventHubResponse(t, listKeysResp)
	primaryKey := keys["primaryKey"].(string)
	if keys["primaryConnectionString"] != "Endpoint=sb://namespace-a.servicebus.windows.net/;SharedAccessKeyName=rule-a;SharedAccessKey="+primaryKey+";EntityPath=hub-a" {
		t.Fatalf("unexpected event hub primary connection string: %v", keys["primaryConnectionString"])
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventHub/namespaces/namespace-a/eventhubs/hub-a/authorizationRules/rule-a/regenerateKeys?api-version=2026-01-01"
	regenerateResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"PrimaryKey"}`)))
	if err != nil {
		t.Fatalf("regenerate event hub primary key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate event hub primary status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotated := decodeEventHubResponse(t, regenerateResp)
	if rotated["primaryKey"] == primaryKey {
		t.Fatalf("expected event hub primary key rotation, before=%v after=%v", keys, rotated)
	}

	deleteResp, err := svc.HandleRequest(eventHubCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete event hub authorization rule returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete event hub authorization rule status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
}

func eventHubCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeEventHubResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
