package eventgrid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestTopicAndEventSubscriptionLifecycle(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	topicPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"inputSchema":"EventGridSchema",
			"publicNetworkAccess":"Enabled"
		}
	}`)

	createTopicResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, topicPayload))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	createdTopic := decodeEventGridResponse(t, createTopicResp)
	if createdTopic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a" {
		t.Fatalf("unexpected topic id: %v", createdTopic["id"])
	}
	if createdTopic["name"] != "topic-a" || createdTopic["type"] != "Microsoft.EventGrid/topics" || createdTopic["location"] != "eastus" {
		t.Fatalf("unexpected topic identity fields: %v", createdTopic)
	}
	topicProps := createdTopic["properties"].(map[string]any)
	if topicProps["provisioningState"] != "Succeeded" || topicProps["endpoint"] != "https://topic-a.eastus-1.eventgrid.azure.net/api/events" {
		t.Fatalf("unexpected topic properties: %v", topicProps)
	}

	getTopicResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, topicURL, nil))
	if err != nil {
		t.Fatalf("get topic returned error: %v", err)
	}
	if getTopicResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get topic status 200, got %d; body=%s", getTopicResp.StatusCode, string(getTopicResp.RawBody))
	}

	listTopicsResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list topics returned error: %v", err)
	}
	listedTopics := decodeEventGridResponse(t, listTopicsResp)
	topicValues := listedTopics["value"].([]any)
	if len(topicValues) != 1 {
		t.Fatalf("expected one topic in list, got %d in %v", len(topicValues), listedTopics)
	}

	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	subscriptionPayload := []byte(`{
		"properties":{
			"destination":{
				"endpointType":"WebHook",
				"properties":{"endpointUrl":"https://example.com/events"}
			},
			"filter":{"includedEventTypes":["Contoso.ItemCreated"]},
			"labels":["test"],
			"retryPolicy":{"maxDeliveryAttempts":10,"eventTimeToLiveInMinutes":60}
		}
	}`)
	createSubscriptionResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, subscriptionPayload))
	if err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	}
	if createSubscriptionResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", createSubscriptionResp.StatusCode, string(createSubscriptionResp.RawBody))
	}
	createdSubscription := decodeEventGridResponse(t, createSubscriptionResp)
	if createdSubscription["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a" {
		t.Fatalf("unexpected event subscription id: %v", createdSubscription["id"])
	}
	if createdSubscription["name"] != "sub-a" || createdSubscription["type"] != "Microsoft.EventGrid/topics/eventSubscriptions" {
		t.Fatalf("unexpected event subscription identity fields: %v", createdSubscription)
	}
	subscriptionProps := createdSubscription["properties"].(map[string]any)
	if subscriptionProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected event subscription state: %v", subscriptionProps)
	}
	destination := subscriptionProps["destination"].(map[string]any)
	if destination["endpointType"] != "WebHook" {
		t.Fatalf("unexpected event subscription destination: %v", destination)
	}

	listSubscriptionsResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list event subscriptions returned error: %v", err)
	}
	listedSubscriptions := decodeEventGridResponse(t, listSubscriptionsResp)
	subscriptionValues := listedSubscriptions["value"].([]any)
	if len(subscriptionValues) != 1 {
		t.Fatalf("expected one event subscription in list, got %d in %v", len(subscriptionValues), listedSubscriptions)
	}

	deleteSubscriptionResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, subscriptionURL, nil))
	if err != nil {
		t.Fatalf("delete event subscription returned error: %v", err)
	}
	if deleteSubscriptionResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete event subscription status 202, got %d; body=%s", deleteSubscriptionResp.StatusCode, string(deleteSubscriptionResp.RawBody))
	}

	deleteTopicResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, topicURL, nil))
	if err != nil {
		t.Fatalf("delete topic returned error: %v", err)
	}
	if deleteTopicResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete topic status 202, got %d; body=%s", deleteTopicResp.StatusCode, string(deleteTopicResp.RawBody))
	}
}

func TestTopicCreateAppliesAzureDefaults(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	createResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeEventGridResponse(t, createResp)
	props := created["properties"].(map[string]any)
	if props["disableLocalAuth"] != false {
		t.Fatalf("expected default disableLocalAuth=false, got %v in %v", props["disableLocalAuth"], props)
	}
	if props["inputSchema"] != "EventGridSchema" {
		t.Fatalf("expected default inputSchema=EventGridSchema, got %v in %v", props["inputSchema"], props)
	}
	if props["publicNetworkAccess"] != "Enabled" {
		t.Fatalf("expected default publicNetworkAccess=Enabled, got %v in %v", props["publicNetworkAccess"], props)
	}

	updateResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{
		"location":"eastus",
		"properties":{
			"disableLocalAuth":true,
			"inputSchema":"CloudEventSchemaV1_0",
			"publicNetworkAccess":"Disabled"
		}
	}`)))
	if err != nil {
		t.Fatalf("update topic returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update topic status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeEventGridResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if updatedProps["disableLocalAuth"] != true {
		t.Fatalf("expected explicit disableLocalAuth=true to be preserved, got %v in %v", updatedProps["disableLocalAuth"], updatedProps)
	}
	if updatedProps["inputSchema"] != "CloudEventSchemaV1_0" {
		t.Fatalf("expected explicit inputSchema to be preserved, got %v in %v", updatedProps["inputSchema"], updatedProps)
	}
	if updatedProps["publicNetworkAccess"] != "Disabled" {
		t.Fatalf("expected explicit publicNetworkAccess to be preserved, got %v in %v", updatedProps["publicNetworkAccess"], updatedProps)
	}
}

func TestTopicDeleteReturnsOperationHeadersAndCascades(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://example.com/events"}}}}`))); err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, topicURL, nil))
	if err != nil {
		t.Fatalf("delete topic returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete topic status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	if len(deleteResp.RawBody) != 0 {
		t.Fatalf("expected async delete topic response body to be empty, got %q", string(deleteResp.RawBody))
	}
	location := deleteResp.Headers["Location"]
	if location == "" {
		t.Fatalf("expected Location header, got %#v", deleteResp.Headers)
	}
	if !strings.Contains(location, "/providers/Microsoft.EventGrid/locations/eastus/operationStatus/default/operationId/") || !strings.Contains(location, "api-version=2025-02-15") {
		t.Fatalf("unexpected Location header: %q", location)
	}
	asyncOperation := deleteResp.Headers["Azure-AsyncOperation"]
	if asyncOperation == "" {
		t.Fatalf("expected Azure-AsyncOperation header, got %#v", deleteResp.Headers)
	}
	if !strings.Contains(asyncOperation, "/providers/Microsoft.EventGrid/locations/eastus/operationResults/") || !strings.Contains(asyncOperation, "api-version=2025-02-15") {
		t.Fatalf("unexpected Azure-AsyncOperation header: %q", asyncOperation)
	}

	getSubscriptionResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, subscriptionURL, nil))
	if err != nil {
		t.Fatalf("get cascaded event subscription returned error: %v", err)
	}
	if getSubscriptionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected cascaded event subscription to be missing, got %d; body=%s", getSubscriptionResp.StatusCode, string(getSubscriptionResp.RawBody))
	}

	repeatDeleteResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, topicURL, nil))
	if err != nil {
		t.Fatalf("repeat delete topic returned error: %v", err)
	}
	if repeatDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected repeat delete topic status 204, got %d; body=%s", repeatDeleteResp.StatusCode, string(repeatDeleteResp.RawBody))
	}
}

func TestTopicsListBySubscriptionAcrossResourceGroups(t *testing.T) {
	svc := New()

	createRequests := []struct {
		url  string
		body []byte
	}{
		{
			url:  "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.EventGrid/topics/topic-b?api-version=2025-02-15",
			body: []byte(`{"location":"westus2"}`),
		},
		{
			url:  "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15",
			body: []byte(`{"location":"eastus"}`),
		},
		{
			url:  "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-c/providers/Microsoft.EventGrid/topics/topic-c?api-version=2025-02-15",
			body: []byte(`{"location":"centralus"}`),
		},
	}
	for _, request := range createRequests {
		resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, request.url, request.body))
		if err != nil {
			t.Fatalf("create topic %s returned error: %v", request.url, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create topic status 201 for %s, got %d; body=%s", request.url, resp.StatusCode, string(resp.RawBody))
		}
	}

	listResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.EventGrid/topics?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list topics by subscription returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list topics by subscription status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeEventGridResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two topics in subscription list, got %d in %v", len(values), listed)
	}
	first := values[0].(map[string]any)
	second := values[1].(map[string]any)
	if first["name"] != "topic-a" || second["name"] != "topic-b" {
		t.Fatalf("expected stable topic name ordering, got %v then %v", first["name"], second["name"])
	}
	if !strings.Contains(first["id"].(string), "/resourceGroups/rg-a/") || !strings.Contains(second["id"].(string), "/resourceGroups/rg-b/") {
		t.Fatalf("expected resource group IDs to be preserved, got %v", values)
	}
}

func TestTopicsListAppliesNameFilterAndTop(t *testing.T) {
	svc := New()

	for _, name := range []string{"alpha-01", "alpha-02", "alpha-archive", "beta-01"} {
		topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/" + name + "?api-version=2025-02-15"
		resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`)))
		if err != nil {
			t.Fatalf("create topic %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create topic %s status 201, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics?api-version=2025-02-15&%24filter=contains(namE%2C%20%27alpha%27)%20and%20name%20ne%20%27alpha-archive%27&%24top=1"
	listResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list topics with filter returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected filtered list status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeEventGridResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected top=1 filtered list, got %d in %v", len(values), listed)
	}
	topic := values[0].(map[string]any)
	if topic["name"] != "alpha-01" {
		t.Fatalf("expected first filtered topic alpha-01, got %v", topic["name"])
	}
}

func TestTopicsListReturnsNextLinkAndContinues(t *testing.T) {
	svc := New()

	for _, name := range []string{"alpha-01", "alpha-02", "alpha-03"} {
		topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/" + name + "?api-version=2025-02-15"
		resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`)))
		if err != nil {
			t.Fatalf("create topic %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create topic %s status 201, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}

	firstPageURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics?api-version=2025-02-15&%24filter=contains(name%2C%20%27alpha%27)&%24top=2"
	firstPageResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, firstPageURL, nil))
	if err != nil {
		t.Fatalf("list first topic page returned error: %v", err)
	}
	firstPage := decodeEventGridResponse(t, firstPageResp)
	firstValues := firstPage["value"].([]any)
	if len(firstValues) != 2 {
		t.Fatalf("expected first page to contain two topics, got %d in %v", len(firstValues), firstPage)
	}
	nextLink, ok := firstPage["nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected nextLink on truncated topic list, got %v", firstPage)
	}
	if !strings.Contains(nextLink, "%24skipToken=") && !strings.Contains(nextLink, "$skipToken=") {
		t.Fatalf("expected nextLink to include skip token, got %q", nextLink)
	}

	secondPageResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, nextLink, nil))
	if err != nil {
		t.Fatalf("list second topic page returned error: %v", err)
	}
	secondPage := decodeEventGridResponse(t, secondPageResp)
	secondValues := secondPage["value"].([]any)
	if len(secondValues) != 1 {
		t.Fatalf("expected second page to contain one topic, got %d in %v", len(secondValues), secondPage)
	}
	topic := secondValues[0].(map[string]any)
	if topic["name"] != "alpha-03" {
		t.Fatalf("expected second page topic alpha-03, got %v", topic["name"])
	}
	if _, ok := secondPage["nextLink"]; ok {
		t.Fatalf("expected no nextLink on final topic page, got %v", secondPage)
	}
}

func TestTopicsListRejectsInvalidFilterAndTop(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	invalidFilterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics?api-version=2025-02-15&%24filter=location%20eq%20%27eastus%27"
	invalidFilterResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, invalidFilterURL, nil))
	if err != nil {
		t.Fatalf("list topics with invalid filter returned error: %v", err)
	}
	if invalidFilterResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid filter status 400, got %d; body=%s", invalidFilterResp.StatusCode, string(invalidFilterResp.RawBody))
	}

	invalidTopURL := "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.EventGrid/topics?api-version=2025-02-15&%24top=0"
	invalidTopResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, invalidTopURL, nil))
	if err != nil {
		t.Fatalf("list topics with invalid top returned error: %v", err)
	}
	if invalidTopResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid top status 400, got %d; body=%s", invalidTopResp.StatusCode, string(invalidTopResp.RawBody))
	}
}

func TestTopicEventSubscriptionTemplateProvisioning(t *testing.T) {
	svc := New()

	topicResource := map[string]any{
		"type":     "Microsoft.EventGrid/topics",
		"name":     "topic-a",
		"location": "eastus",
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"inputSchema": "EventGridSchema",
		},
	}
	topicResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", topicResource)
	if err != nil {
		t.Fatalf("provision topic returned error: %v", err)
	}
	topic := topicResult.(map[string]any)
	if topic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a" {
		t.Fatalf("unexpected provisioned topic id: %v", topic["id"])
	}
	if topic["type"] != "Microsoft.EventGrid/topics" {
		t.Fatalf("unexpected provisioned topic type: %v", topic["type"])
	}

	subscriptionResource := map[string]any{
		"type": "Microsoft.EventGrid/topics/eventSubscriptions",
		"name": "topic-a/sub-a",
		"properties": map[string]any{
			"destination": map[string]any{
				"endpointType": "WebHook",
				"properties":   map[string]any{"endpointUrl": "https://example.com/events"},
			},
		},
	}
	subscriptionResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", subscriptionResource)
	if err != nil {
		t.Fatalf("provision event subscription returned error: %v", err)
	}
	subscription := subscriptionResult.(map[string]any)
	if subscription["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a" {
		t.Fatalf("unexpected provisioned event subscription id: %v", subscription["id"])
	}
	if subscription["type"] != "Microsoft.EventGrid/topics/eventSubscriptions" {
		t.Fatalf("unexpected provisioned event subscription type: %v", subscription["type"])
	}
}

func TestTopicEventSubscriptionCreateAppliesAzureDefaults(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	subscriptionPayload := []byte(`{
		"properties":{
			"destination":{
				"endpointType":"WebHook",
				"properties":{"endpointUrl":"https://example.com/events"}
			}
		}
	}`)
	createResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, subscriptionPayload))
	if err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeEventGridResponse(t, createResp)
	properties := created["properties"].(map[string]any)
	if properties["topic"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a" {
		t.Fatalf("expected topic resource id, got %v in %v", properties["topic"], properties)
	}
	destination := properties["destination"].(map[string]any)
	destinationProperties := destination["properties"].(map[string]any)
	if destinationProperties["endpointBaseUrl"] != "https://example.com/events" {
		t.Fatalf("expected WebHook endpointBaseUrl projection, got %v", destinationProperties)
	}
	if properties["eventDeliverySchema"] != "EventGridSchema" {
		t.Fatalf("expected default eventDeliverySchema EventGridSchema, got %v in %v", properties["eventDeliverySchema"], properties)
	}
	retryPolicy := properties["retryPolicy"].(map[string]any)
	if retryPolicy["maxDeliveryAttempts"] != float64(30) || retryPolicy["eventTimeToLiveInMinutes"] != float64(1440) {
		t.Fatalf("unexpected default retry policy: %v", retryPolicy)
	}
}

func TestTopicEventSubscriptionGetDeliveryAttributes(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	subscriptionPayload := []byte(`{
		"properties":{
			"destination":{
				"endpointType":"WebHook",
				"properties":{
					"endpointUrl":"https://example.com/events",
					"deliveryAttributeMappings":[
						{"name":"header1","type":"Static","properties":{"value":"NormalValue","isSecret":false}},
						{"name":"header2","type":"Dynamic","properties":{"sourceField":"data.foo"}}
					]
				}
			}
		}
	}`)
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, subscriptionPayload)); err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	attributesURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a/getDeliveryAttributes?api-version=2025-02-15"
	attributesResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, attributesURL, nil))
	if err != nil {
		t.Fatalf("get delivery attributes returned error: %v", err)
	}
	if attributesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get delivery attributes status 200, got %d; body=%s", attributesResp.StatusCode, string(attributesResp.RawBody))
	}
	body := decodeEventGridResponse(t, attributesResp)
	values := body["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two delivery attributes, got %d in %v", len(values), body)
	}
	static := values[0].(map[string]any)
	if static["name"] != "header1" || static["type"] != "Static" {
		t.Fatalf("unexpected static delivery attribute: %v", static)
	}
	staticProps := static["properties"].(map[string]any)
	if staticProps["value"] != "NormalValue" || staticProps["isSecret"] != false {
		t.Fatalf("unexpected static delivery attribute properties: %v", staticProps)
	}
	dynamic := values[1].(map[string]any)
	if dynamic["name"] != "header2" || dynamic["type"] != "Dynamic" {
		t.Fatalf("unexpected dynamic delivery attribute: %v", dynamic)
	}
	dynamicProps := dynamic["properties"].(map[string]any)
	if dynamicProps["sourceField"] != "data.foo" {
		t.Fatalf("unexpected dynamic delivery attribute properties: %v", dynamicProps)
	}
}

func TestTopicEventSubscriptionsListAppliesNameFilterAndTop(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	for _, name := range []string{"alpha-01", "alpha-02", "alpha-archive", "beta-01"} {
		subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/" + name + "?api-version=2025-02-15"
		resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://example.com/events"}}}}`)))
		if err != nil {
			t.Fatalf("create event subscription %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create event subscription %s status 201, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions?api-version=2025-02-15&%24filter=contains(namE%2C%20%27alpha%27)%20and%20name%20ne%20%27alpha-archive%27&%24top=1"
	listResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list event subscriptions with filter returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected filtered event subscription list status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeEventGridResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected top=1 filtered event subscription list, got %d in %v", len(values), listed)
	}
	subscription := values[0].(map[string]any)
	if subscription["name"] != "alpha-01" {
		t.Fatalf("expected first filtered event subscription alpha-01, got %v", subscription["name"])
	}
}

func TestTopicEventSubscriptionsListReturnsNextLinkAndContinues(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	for _, name := range []string{"alpha-01", "alpha-02", "alpha-03"} {
		subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/" + name + "?api-version=2025-02-15"
		resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://example.com/events"}}}}`)))
		if err != nil {
			t.Fatalf("create event subscription %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create event subscription %s status 201, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}

	firstPageURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions?api-version=2025-02-15&%24filter=contains(name%2C%20%27alpha%27)&%24top=2"
	firstPageResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, firstPageURL, nil))
	if err != nil {
		t.Fatalf("list first event subscription page returned error: %v", err)
	}
	firstPage := decodeEventGridResponse(t, firstPageResp)
	firstValues := firstPage["value"].([]any)
	if len(firstValues) != 2 {
		t.Fatalf("expected first page to contain two event subscriptions, got %d in %v", len(firstValues), firstPage)
	}
	nextLink, ok := firstPage["nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected nextLink on truncated event subscription list, got %v", firstPage)
	}

	secondPageResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, nextLink, nil))
	if err != nil {
		t.Fatalf("list second event subscription page returned error: %v", err)
	}
	secondPage := decodeEventGridResponse(t, secondPageResp)
	secondValues := secondPage["value"].([]any)
	if len(secondValues) != 1 {
		t.Fatalf("expected second page to contain one event subscription, got %d in %v", len(secondValues), secondPage)
	}
	subscription := secondValues[0].(map[string]any)
	if subscription["name"] != "alpha-03" {
		t.Fatalf("expected second page event subscription alpha-03, got %v", subscription["name"])
	}
	if _, ok := secondPage["nextLink"]; ok {
		t.Fatalf("expected no nextLink on final event subscription page, got %v", secondPage)
	}
}

func TestTopicEventSubscriptionsListRejectsInvalidFilterAndTop(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://example.com/events"}}}}`))); err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	invalidFilterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions?api-version=2025-02-15&%24filter=topic%20eq%20%27topic-a%27"
	invalidFilterResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, invalidFilterURL, nil))
	if err != nil {
		t.Fatalf("list event subscriptions with invalid filter returned error: %v", err)
	}
	if invalidFilterResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid filter status 400, got %d; body=%s", invalidFilterResp.StatusCode, string(invalidFilterResp.RawBody))
	}

	invalidTopURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions?api-version=2025-02-15&%24top=101"
	invalidTopResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodGet, invalidTopURL, nil))
	if err != nil {
		t.Fatalf("list event subscriptions with invalid top returned error: %v", err)
	}
	if invalidTopResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid top status 400, got %d; body=%s", invalidTopResp.StatusCode, string(invalidTopResp.RawBody))
	}
}

func TestTopicEventSubscriptionDeleteReturnsOperationHeaders(t *testing.T) {
	svc := New()

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, topicURL, []byte(`{"location":"eastus"}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/eventSubscriptions/sub-a?api-version=2025-02-15"
	if resp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://example.com/events"}}}}`))); err != nil {
		t.Fatalf("create event subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create event subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, subscriptionURL, nil))
	if err != nil {
		t.Fatalf("delete event subscription returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete event subscription status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	if len(deleteResp.RawBody) != 0 {
		t.Fatalf("expected async delete response body to be empty, got %q", string(deleteResp.RawBody))
	}
	location := deleteResp.Headers["Location"]
	if location == "" {
		t.Fatalf("expected Location header, got %#v", deleteResp.Headers)
	}
	if !strings.Contains(location, "/providers/Microsoft.EventGrid/locations/eastus/operationStatus/default/operationId/") || !strings.Contains(location, "api-version=2025-02-15") {
		t.Fatalf("unexpected Location header: %q", location)
	}
	asyncOperation := deleteResp.Headers["Azure-AsyncOperation"]
	if asyncOperation == "" {
		t.Fatalf("expected Azure-AsyncOperation header, got %#v", deleteResp.Headers)
	}
	if !strings.Contains(asyncOperation, "/providers/Microsoft.EventGrid/locations/eastus/operationResults/") || !strings.Contains(asyncOperation, "api-version=2025-02-15") {
		t.Fatalf("unexpected Azure-AsyncOperation header: %q", asyncOperation)
	}

	repeatDeleteResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodDelete, subscriptionURL, nil))
	if err != nil {
		t.Fatalf("repeat delete event subscription returned error: %v", err)
	}
	if repeatDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected repeat delete event subscription status 204, got %d; body=%s", repeatDeleteResp.StatusCode, string(repeatDeleteResp.RawBody))
	}
}

func TestPublishEventsToCustomTopicDataPlane(t *testing.T) {
	svc := New()

	createTopicResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus","properties":{"inputSchema":"EventGridSchema"}}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	key1 := keys["key1"].(string)
	if key1 == "" || keys["key2"] == "" || key1 == keys["key2"] {
		t.Fatalf("expected two distinct topic keys, got %v", keys)
	}

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"id":"evt-1",
			"eventType":"Contoso.ItemCreated",
			"subject":"items/item-1",
			"eventTime":"2026-06-15T12:00:00Z",
			"data":{"id":"item-1","status":"created"}
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", key1)

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusOK {
		t.Fatalf("expected publish status 200, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}

	svc.mu.RLock()
	events := svc.publishedEvents[resourceKey("sub-1", "rg-a", "topic-a")]
	svc.mu.RUnlock()
	if len(events) != 1 {
		t.Fatalf("expected one retained event, got %d in %#v", len(events), events)
	}
	if events[0]["id"] != "evt-1" || events[0]["eventType"] != "Contoso.ItemCreated" {
		t.Fatalf("unexpected retained event: %#v", events[0])
	}
	if events[0]["topic"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a" {
		t.Fatalf("expected Event Grid topic to be stamped, got %#v", events[0])
	}
	if events[0]["dataVersion"] != "" || events[0]["metadataVersion"] != "1" {
		t.Fatalf("expected Event Grid versions to be stamped, got %#v", events[0])
	}
}

func TestPublishEventsAcceptsCloudEventsForCloudEventSchemaTopic(t *testing.T) {
	svc := New()

	createTopicResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus","properties":{"inputSchema":"CloudEventSchemaV1_0"}}`)))
	if err != nil {
		t.Fatalf("create CloudEvents topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create CloudEvents topic status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"specversion":"1.0",
			"type":"Contoso.ItemCreated",
			"source":"/contoso/items",
			"id":"evt-cloud-1",
			"time":"2026-06-15T12:00:00Z",
			"subject":"items/item-1",
			"data":{"id":"item-1","status":"created"}
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish CloudEvent returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusOK {
		t.Fatalf("expected CloudEvent publish status 200, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}

	svc.mu.RLock()
	events := svc.publishedEvents[resourceKey("sub-1", "rg-a", "topic-a")]
	svc.mu.RUnlock()
	if len(events) != 1 {
		t.Fatalf("expected one retained CloudEvent, got %d in %#v", len(events), events)
	}
	if events[0]["specversion"] != "1.0" || events[0]["type"] != "Contoso.ItemCreated" || events[0]["source"] != "/contoso/items" {
		t.Fatalf("unexpected retained CloudEvent: %#v", events[0])
	}
}

func TestTopicSharedAccessKeysRegenerateAndAuthorizePublish(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}

	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	oldKey1 := keys["key1"].(string)
	key2 := keys["key2"].(string)

	rejectCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", validEventGridPublishBody())
	rejectCtx.RawRequest.Header.Set("aeg-sas-key", "wrong")
	rejectResp, err := svc.HandleRequest(rejectCtx)
	if err != nil {
		t.Fatalf("publish with wrong key returned error: %v", err)
	}
	if rejectResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected wrong key publish status 401, got %d; body=%s", rejectResp.StatusCode, string(rejectResp.RawBody))
	}

	regenerateResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/regenerateKey?api-version=2025-02-15", []byte(`{"keyName":"key1"}`)))
	if err != nil {
		t.Fatalf("regenerate key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate key status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotatedKeys := decodeEventGridResponse(t, regenerateResp)
	newKey1 := rotatedKeys["key1"].(string)
	if newKey1 == "" || newKey1 == oldKey1 {
		t.Fatalf("expected key1 to rotate, before=%q after=%q", oldKey1, newKey1)
	}
	if rotatedKeys["key2"] != key2 {
		t.Fatalf("expected key2 to remain stable, before=%q after=%v", key2, rotatedKeys["key2"])
	}

	oldKeyCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", validEventGridPublishBody())
	oldKeyCtx.RawRequest.Header.Set("aeg-sas-key", oldKey1)
	oldKeyResp, err := svc.HandleRequest(oldKeyCtx)
	if err != nil {
		t.Fatalf("publish with old key returned error: %v", err)
	}
	if oldKeyResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected old key publish status 401, got %d; body=%s", oldKeyResp.StatusCode, string(oldKeyResp.RawBody))
	}

	newKeyCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", validEventGridPublishBody())
	newKeyCtx.RawRequest.Header.Set("aeg-sas-key", newKey1)
	newKeyResp, err := svc.HandleRequest(newKeyCtx)
	if err != nil {
		t.Fatalf("publish with new key returned error: %v", err)
	}
	if newKeyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected new key publish status 200, got %d; body=%s", newKeyResp.StatusCode, string(newKeyResp.RawBody))
	}
}

func TestPublishEventsHonorsDisableLocalAuth(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus","properties":{"disableLocalAuth":true}}`)))
	if err != nil {
		t.Fatalf("create disableLocalAuth topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	keyOnlyCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", validEventGridPublishBody())
	keyOnlyCtx.RawRequest.Header.Del("Authorization")
	keyOnlyCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))
	keyOnlyResp, err := svc.HandleRequest(keyOnlyCtx)
	if err != nil {
		t.Fatalf("key-only publish returned error: %v", err)
	}
	if keyOnlyResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected key-only publish status 401 when disableLocalAuth=true, got %d; body=%s", keyOnlyResp.StatusCode, string(keyOnlyResp.RawBody))
	}

	bearerCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", validEventGridPublishBody())
	bearerResp, err := svc.HandleRequest(bearerCtx)
	if err != nil {
		t.Fatalf("bearer publish returned error: %v", err)
	}
	if bearerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected bearer publish status 200 when disableLocalAuth=true, got %d; body=%s", bearerResp.StatusCode, string(bearerResp.RawBody))
	}
}

func TestPublishEventsAcceptsSharedAccessSignatures(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	endpoint := "https://topic-a.eastus-1.eventgrid.azure.net/api/events"
	token := eventGridSASToken(t, endpoint, keys["key1"].(string), time.Now().UTC().Add(time.Hour))

	sasHeaderCtx := eventGridCtx(t, http.MethodPost, endpoint+"?api-version=2018-01-01", validEventGridPublishBody())
	sasHeaderCtx.RawRequest.Header.Del("Authorization")
	sasHeaderCtx.RawRequest.Header.Set("aeg-sas-token", token)
	sasHeaderResp, err := svc.HandleRequest(sasHeaderCtx)
	if err != nil {
		t.Fatalf("publish with aeg-sas-token returned error: %v", err)
	}
	if sasHeaderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected aeg-sas-token publish status 200, got %d; body=%s", sasHeaderResp.StatusCode, string(sasHeaderResp.RawBody))
	}

	authHeaderCtx := eventGridCtx(t, http.MethodPost, endpoint+"?api-version=2018-01-01", validEventGridPublishBody())
	authHeaderCtx.RawRequest.Header.Set("Authorization", "SharedAccessSignature "+token)
	authHeaderResp, err := svc.HandleRequest(authHeaderCtx)
	if err != nil {
		t.Fatalf("publish with Authorization SAS returned error: %v", err)
	}
	if authHeaderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected Authorization SAS publish status 200, got %d; body=%s", authHeaderResp.StatusCode, string(authHeaderResp.RawBody))
	}

	svc.mu.RLock()
	events := svc.publishedEvents[resourceKey("sub-1", "rg-a", "topic-a")]
	svc.mu.RUnlock()
	if len(events) != 2 {
		t.Fatalf("expected two retained SAS-published events, got %d in %#v", len(events), events)
	}
}

func TestPublishEventsAcceptsAccessKeyQueryParameter(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	publishURL := "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01&aeg-sas-key=" + url.QueryEscape(keys["key1"].(string))
	publishCtx := eventGridCtx(t, http.MethodPost, publishURL, validEventGridPublishBody())
	publishCtx.RawRequest.Header.Del("Authorization")

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("query-key publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusOK {
		t.Fatalf("expected query-key publish status 200, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsMissingPublishAPIVersion(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events", validEventGridPublishBody())
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing publish api-version status 404, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
	body := decodeEventGridResponse(t, publishResp)
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "NotFound" {
		t.Fatalf("unexpected error body: %v", body)
	}
}

func TestPublishEventsRejectsInvalidCustomTopicPayload(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`{"id":"evt-1"}`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
	body := decodeEventGridResponse(t, publishResp)
	errorBody := body["error"].(map[string]any)
	if errorBody["code"] != "BadRequest" {
		t.Fatalf("unexpected error body: %v", body)
	}
}

func TestPublishEventsRejectsNonObjectEventGridData(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"id":"evt-1",
			"eventType":"Contoso.ItemCreated",
			"subject":"items/item-1",
			"eventTime":"2026-06-15T12:00:00Z",
			"data":"not-an-object"
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected non-object data publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsMismatchedEventGridTopic(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"topic":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/other-topic",
			"id":"evt-1",
			"eventType":"Contoso.ItemCreated",
			"subject":"items/item-1",
			"eventTime":"2026-06-15T12:00:00Z",
			"data":{"id":"item-1","status":"created"},
			"dataVersion":"1.0"
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected mismatched topic publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsUnsupportedEventGridMetadataVersion(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"metadataVersion":"2",
			"id":"evt-1",
			"eventType":"Contoso.ItemCreated",
			"subject":"items/item-1",
			"eventTime":"2026-06-15T12:00:00Z",
			"data":{"id":"item-1","status":"created"},
			"dataVersion":"1.0"
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unsupported metadataVersion publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsInvalidEventGridEventTime(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"id":"evt-1",
			"eventType":"Contoso.ItemCreated",
			"subject":"items/item-1",
			"eventTime":"not-a-date",
			"data":{"id":"item-1","status":"created"},
			"dataVersion":"1.0"
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid eventTime publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsInvalidCloudEventTime(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus","properties":{"inputSchema":"CloudEventSchemaV1_0"}}`)))
	if err != nil {
		t.Fatalf("create CloudEvents topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"specversion":"1.0",
			"type":"Contoso.ItemCreated",
			"source":"/contoso/items",
			"id":"evt-cloud-1",
			"time":"not-a-date",
			"subject":"items/item-1",
			"data":{"id":"item-1","status":"created"}
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish CloudEvent returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid CloudEvent time publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsUnsupportedCloudEventSpecversion(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus","properties":{"inputSchema":"CloudEventSchemaV1_0"}}`)))
	if err != nil {
		t.Fatalf("create CloudEvents topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)

	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(`[
		{
			"specversion":"2.0",
			"type":"Contoso.ItemCreated",
			"source":"/contoso/items",
			"id":"evt-cloud-1",
			"time":"2026-06-15T12:00:00Z",
			"subject":"items/item-1",
			"data":{"id":"item-1","status":"created"}
		}
	]`))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish CloudEvent returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unsupported CloudEvent specversion publish status 400, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsPayloadsOverOneMegabyte(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	oversizedData := strings.Repeat("x", 1024*1024)
	body := []byte(`[{
		"id":"evt-oversized",
		"eventType":"Contoso.ItemCreated",
		"subject":"items/item-oversized",
		"eventTime":"2026-06-15T12:00:00Z",
		"data":{"payload":"` + oversizedData + `"},
		"dataVersion":"1.0"
	}]`)
	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", body)
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish oversized payload returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized publish status 413, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestPublishEventsRejectsBatchesOverFiveThousandEvents(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	listKeysResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/listKeys?api-version=2025-02-15", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeEventGridResponse(t, listKeysResp)
	var body strings.Builder
	body.WriteByte('[')
	for i := 0; i < 5001; i++ {
		if i > 0 {
			body.WriteByte(',')
		}
		body.WriteString(`{"id":"evt-batch","eventType":"Contoso.ItemCreated","subject":"items/item","eventTime":"2026-06-15T12:00:00Z","data":{},"dataVersion":"1.0"}`)
	}
	body.WriteByte(']')
	publishCtx := eventGridCtx(t, http.MethodPost, "https://topic-a.eastus-1.eventgrid.azure.net/api/events?api-version=2018-01-01", []byte(body.String()))
	publishCtx.RawRequest.Header.Set("aeg-sas-key", keys["key1"].(string))

	publishResp, err := svc.HandleRequest(publishCtx)
	if err != nil {
		t.Fatalf("publish oversized batch returned error: %v", err)
	}
	if publishResp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected oversized batch publish status 413, got %d; body=%s", publishResp.StatusCode, string(publishResp.RawBody))
	}
}

func TestRegenerateTopicSharedAccessKeyRejectsInvalidKeyName(t *testing.T) {
	svc := New()
	_, err := svc.HandleRequest(eventGridCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a?api-version=2025-02-15", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}

	regenerateResp, err := svc.HandleRequest(eventGridCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.EventGrid/topics/topic-a/regenerateKey?api-version=2025-02-15", []byte(`{"keyName":"key3"}`)))
	if err != nil {
		t.Fatalf("regenerate invalid key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid regenerate key status 400, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
}

func TestActionsExposeDeliveryAttributes(t *testing.T) {
	svc := New()
	actions := map[string]string{}
	for _, action := range svc.Actions() {
		actions[action.Name] = action.Method
	}

	if actions["GetDeliveryAttributes"] != http.MethodPost {
		t.Fatalf("expected GetDeliveryAttributes action to use POST, got %q in %v", actions["GetDeliveryAttributes"], actions)
	}
}

func TestServiceKeysIncludePublishDataPlane(t *testing.T) {
	svc := New()

	var hasControlPlane bool
	var hasPublish bool
	for _, key := range svc.ServiceKeys() {
		if key == (routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.EventGrid/topics", APIVersion: "2025-02-15"}) {
			hasControlPlane = true
		}
		if key == (routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.EventGrid/publish", APIVersion: "2018-01-01"}) {
			hasPublish = true
		}
	}
	if !hasControlPlane || !hasPublish {
		t.Fatalf("expected Event Grid control-plane and publish service keys, got %#v", svc.ServiceKeys())
	}
}

func eventGridCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeEventGridResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}

func validEventGridPublishBody() []byte {
	return []byte(`[{"id":"evt-1","eventType":"Contoso.ItemCreated","subject":"items/item-1","eventTime":"2026-06-15T12:00:00Z","data":{"id":"item-1"},"dataVersion":"1.0"}]`)
}

func eventGridSASToken(t *testing.T, resource, key string, expiry time.Time) string {
	t.Helper()
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("decode topic key: %v", err)
	}
	unsigned := "r=" + url.QueryEscape(resource) + "&e=" + url.QueryEscape(expiry.UTC().Format(time.RFC3339))
	mac := hmac.New(sha256.New, keyBytes)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "&s=" + url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
}
