package servicebus

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestNamespaceAndQueueLifecycle(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	namespacePayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Standard"},
		"tags":{"env":"test"},
		"properties":{"zoneRedundant":true}
	}`)

	createNamespaceResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, namespacePayload))
	if err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	}
	if createNamespaceResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", createNamespaceResp.StatusCode, string(createNamespaceResp.RawBody))
	}
	createdNamespace := decodeServiceBusResponse(t, createNamespaceResp)
	if createdNamespace["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a" {
		t.Fatalf("unexpected namespace id: %v", createdNamespace["id"])
	}
	if createdNamespace["name"] != "ns-a" || createdNamespace["type"] != "Microsoft.ServiceBus/namespaces" || createdNamespace["location"] != "eastus" {
		t.Fatalf("unexpected namespace identity fields: %v", createdNamespace)
	}
	nsProps := createdNamespace["properties"].(map[string]any)
	if nsProps["provisioningState"] != "Succeeded" || nsProps["status"] != "Active" {
		t.Fatalf("unexpected namespace state: %v", nsProps)
	}
	if nsProps["serviceBusEndpoint"] != "https://ns-a.servicebus.windows.net:443/" {
		t.Fatalf("unexpected namespace endpoint: %v", nsProps["serviceBusEndpoint"])
	}

	getNamespaceResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, namespaceURL, nil))
	if err != nil {
		t.Fatalf("get namespace returned error: %v", err)
	}
	if getNamespaceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get namespace status 200, got %d; body=%s", getNamespaceResp.StatusCode, string(getNamespaceResp.RawBody))
	}

	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	queuePayload := []byte(`{
		"properties":{
			"lockDuration":"PT45S",
			"maxDeliveryCount":7,
			"requiresDuplicateDetection":true
		}
	}`)

	createQueueResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, queuePayload))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueueResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", createQueueResp.StatusCode, string(createQueueResp.RawBody))
	}
	createdQueue := decodeServiceBusResponse(t, createQueueResp)
	if createdQueue["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a" {
		t.Fatalf("unexpected queue id: %v", createdQueue["id"])
	}
	if createdQueue["name"] != "ns-a/queue-a" || createdQueue["type"] != "Microsoft.ServiceBus/namespaces/queues" {
		t.Fatalf("unexpected queue identity fields: %v", createdQueue)
	}
	queueProps := createdQueue["properties"].(map[string]any)
	if queueProps["status"] != "Active" || queueProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected queue state: %v", queueProps)
	}
	if queueProps["maxDeliveryCount"].(float64) != 7 {
		t.Fatalf("unexpected maxDeliveryCount: %v", queueProps["maxDeliveryCount"])
	}

	listQueuesResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("list queues returned error: %v", err)
	}
	listedQueues := decodeServiceBusResponse(t, listQueuesResp)
	queueValues := listedQueues["value"].([]any)
	if len(queueValues) != 1 {
		t.Fatalf("expected one queue in list, got %d in %v", len(queueValues), listedQueues)
	}

	deleteQueueResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, queueURL, nil))
	if err != nil {
		t.Fatalf("delete queue returned error: %v", err)
	}
	if deleteQueueResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete queue status 202, got %d; body=%s", deleteQueueResp.StatusCode, string(deleteQueueResp.RawBody))
	}

	deleteNamespaceResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, namespaceURL, nil))
	if err != nil {
		t.Fatalf("delete namespace returned error: %v", err)
	}
	if deleteNamespaceResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete namespace status 202, got %d; body=%s", deleteNamespaceResp.StatusCode, string(deleteNamespaceResp.RawBody))
	}
}

func TestNamespaceAuthorizationRulesKeysAndRegeneration(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth/AuthorizationRules?api-version=2024-01-01"
	listDefaultResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list default authorization rules returned error: %v", err)
	}
	if listDefaultResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list default authorization rules status 200, got %d; body=%s", listDefaultResp.StatusCode, string(listDefaultResp.RawBody))
	}
	defaultRules := decodeServiceBusResponse(t, listDefaultResp)
	defaultValues := defaultRules["value"].([]any)
	if len(defaultValues) != 1 || defaultValues[0].(map[string]any)["name"] != "RootManageSharedAccessKey" {
		t.Fatalf("expected default RootManageSharedAccessKey rule, got %v", defaultRules)
	}

	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth/AuthorizationRules/rule-a?api-version=2024-01-01"
	createResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, []byte(`{"properties":{"rights":["Listen","Send"]}}`)))
	if err != nil {
		t.Fatalf("create authorization rule returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create authorization rule status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeServiceBusResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth/AuthorizationRules/rule-a" {
		t.Fatalf("unexpected authorization rule id: %v", created["id"])
	}
	if created["name"] != "rule-a" || created["type"] != "Microsoft.ServiceBus/Namespaces/AuthorizationRules" {
		t.Fatalf("unexpected authorization rule identity: %v", created)
	}
	rights := created["properties"].(map[string]any)["rights"].([]any)
	if len(rights) != 2 || rights[0] != "Listen" || rights[1] != "Send" {
		t.Fatalf("unexpected authorization rule rights: %v", rights)
	}

	getResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get authorization rule returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get authorization rule status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list authorization rules returned error: %v", err)
	}
	listed := decodeServiceBusResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 2 {
		t.Fatalf("expected two authorization rules, got %d in %v", len(values), listed)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth/AuthorizationRules/rule-a/listKeys?api-version=2024-01-01"
	listKeysResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeServiceBusResponse(t, listKeysResp)
	primaryKey := keys["primaryKey"].(string)
	secondaryKey := keys["secondaryKey"].(string)
	if keys["keyName"] != "rule-a" || primaryKey == "" || secondaryKey == "" || primaryKey == secondaryKey {
		t.Fatalf("unexpected access keys: %v", keys)
	}
	if keys["primaryConnectionString"] != "Endpoint=sb://ns-auth.servicebus.windows.net/;SharedAccessKeyName=rule-a;SharedAccessKey="+primaryKey {
		t.Fatalf("unexpected primary connection string: %v", keys["primaryConnectionString"])
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-auth/AuthorizationRules/rule-a/regenerateKeys?api-version=2024-01-01"
	regenerateResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"PrimaryKey"}`)))
	if err != nil {
		t.Fatalf("regenerate primary key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate primary status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotated := decodeServiceBusResponse(t, regenerateResp)
	if rotated["primaryKey"] == primaryKey || rotated["secondaryKey"] != secondaryKey {
		t.Fatalf("expected primary key rotation only, before=%v after=%v", keys, rotated)
	}

	regenerateSecondaryResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"SecondaryKey","key":"manual-secondary"}`)))
	if err != nil {
		t.Fatalf("regenerate secondary key returned error: %v", err)
	}
	if regenerateSecondaryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate secondary status 200, got %d; body=%s", regenerateSecondaryResp.StatusCode, string(regenerateSecondaryResp.RawBody))
	}
	secondaryRotated := decodeServiceBusResponse(t, regenerateSecondaryResp)
	if secondaryRotated["primaryKey"] != rotated["primaryKey"] || secondaryRotated["secondaryKey"] != "manual-secondary" {
		t.Fatalf("expected explicit secondary key rotation only, before=%v after=%v", rotated, secondaryRotated)
	}
}

func TestQueueAuthorizationRulesKeysAndRegeneration(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a/AuthorizationRules?api-version=2024-01-01"
	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a/AuthorizationRules/queue-send?api-version=2024-01-01"
	createResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, []byte(`{"properties":{"rights":["Send"]}}`)))
	if err != nil {
		t.Fatalf("create queue authorization rule returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create queue authorization rule status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeServiceBusResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a/AuthorizationRules/queue-send" {
		t.Fatalf("unexpected queue authorization rule id: %v", created["id"])
	}
	if created["name"] != "queue-send" || created["type"] != "Microsoft.ServiceBus/Namespaces/Queues/AuthorizationRules" {
		t.Fatalf("unexpected queue authorization rule identity: %v", created)
	}
	rights := created["properties"].(map[string]any)["rights"].([]any)
	if len(rights) != 1 || rights[0] != "Send" {
		t.Fatalf("unexpected queue authorization rule rights: %v", rights)
	}

	getResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get queue authorization rule returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get queue authorization rule status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	listResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list queue authorization rules returned error: %v", err)
	}
	listed := decodeServiceBusResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "queue-send" {
		t.Fatalf("expected one queue authorization rule, got %v", listed)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a/AuthorizationRules/queue-send/listKeys?api-version=2024-01-01"
	listKeysResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list queue authorization rule keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list queue keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeServiceBusResponse(t, listKeysResp)
	primaryKey := keys["primaryKey"].(string)
	secondaryKey := keys["secondaryKey"].(string)
	if keys["keyName"] != "queue-send" || primaryKey == "" || secondaryKey == "" || primaryKey == secondaryKey {
		t.Fatalf("unexpected queue access keys: %v", keys)
	}
	expectedPrimaryConnection := "Endpoint=sb://ns-entity-auth.servicebus.windows.net/;SharedAccessKeyName=queue-send;SharedAccessKey=" + primaryKey + ";EntityPath=queue-a"
	if keys["primaryConnectionString"] != expectedPrimaryConnection {
		t.Fatalf("unexpected queue primary connection string: %v", keys["primaryConnectionString"])
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/queues/queue-a/AuthorizationRules/queue-send/regenerateKeys?api-version=2024-01-01"
	regenerateResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"SecondaryKey","key":"queue-secondary"}`)))
	if err != nil {
		t.Fatalf("regenerate queue secondary key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate queue secondary status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotated := decodeServiceBusResponse(t, regenerateResp)
	if rotated["primaryKey"] != primaryKey || rotated["secondaryKey"] != "queue-secondary" {
		t.Fatalf("expected explicit queue secondary rotation only, before=%v after=%v", keys, rotated)
	}

	deleteResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete queue authorization rule returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete queue authorization rule status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get deleted queue authorization rule returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted queue authorization rule status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestTopicAuthorizationRulesKeysAndRegeneration(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a/AuthorizationRules?api-version=2024-01-01"
	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a/AuthorizationRules/topic-listen?api-version=2024-01-01"
	createResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, []byte(`{"properties":{"rights":["Listen"]}}`)))
	if err != nil {
		t.Fatalf("create topic authorization rule returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create topic authorization rule status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeServiceBusResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a/AuthorizationRules/topic-listen" {
		t.Fatalf("unexpected topic authorization rule id: %v", created["id"])
	}
	if created["name"] != "topic-listen" || created["type"] != "Microsoft.ServiceBus/Namespaces/Topics/AuthorizationRules" {
		t.Fatalf("unexpected topic authorization rule identity: %v", created)
	}
	rights := created["properties"].(map[string]any)["rights"].([]any)
	if len(rights) != 1 || rights[0] != "Listen" {
		t.Fatalf("unexpected topic authorization rule rights: %v", rights)
	}

	getResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get topic authorization rule returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get topic authorization rule status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	listResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list topic authorization rules returned error: %v", err)
	}
	listed := decodeServiceBusResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "topic-listen" {
		t.Fatalf("expected one topic authorization rule, got %v", listed)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a/AuthorizationRules/topic-listen/listKeys?api-version=2024-01-01"
	listKeysResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list topic authorization rule keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list topic keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	keys := decodeServiceBusResponse(t, listKeysResp)
	primaryKey := keys["primaryKey"].(string)
	secondaryKey := keys["secondaryKey"].(string)
	if keys["keyName"] != "topic-listen" || primaryKey == "" || secondaryKey == "" || primaryKey == secondaryKey {
		t.Fatalf("unexpected topic access keys: %v", keys)
	}
	expectedPrimaryConnection := "Endpoint=sb://ns-entity-auth.servicebus.windows.net/;SharedAccessKeyName=topic-listen;SharedAccessKey=" + primaryKey + ";EntityPath=topic-a"
	if keys["primaryConnectionString"] != expectedPrimaryConnection {
		t.Fatalf("unexpected topic primary connection string: %v", keys["primaryConnectionString"])
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-entity-auth/topics/topic-a/AuthorizationRules/topic-listen/regenerateKeys?api-version=2024-01-01"
	regenerateResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyType":"PrimaryKey"}`)))
	if err != nil {
		t.Fatalf("regenerate topic primary key returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate topic primary status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	rotated := decodeServiceBusResponse(t, regenerateResp)
	if rotated["primaryKey"] == primaryKey || rotated["secondaryKey"] != secondaryKey {
		t.Fatalf("expected generated topic primary rotation only, before=%v after=%v", keys, rotated)
	}

	deleteResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete topic authorization rule returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete topic authorization rule status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get deleted topic authorization rule returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted topic authorization rule status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestTopicSubscriptionLifecycle(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	namespacePayload := []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`)
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, namespacePayload)); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a?api-version=2024-01-01"
	topicPayload := []byte(`{
		"properties":{
			"defaultMessageTimeToLive":"P14D",
			"duplicateDetectionHistoryTimeWindow":"PT10M",
			"enablePartitioning":true
		}
	}`)
	createTopicResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, topicPayload))
	if err != nil {
		t.Fatalf("create topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	createdTopic := decodeServiceBusResponse(t, createTopicResp)
	if createdTopic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a" {
		t.Fatalf("unexpected topic id: %v", createdTopic["id"])
	}
	if createdTopic["name"] != "ns-a/topic-a" || createdTopic["type"] != "Microsoft.ServiceBus/namespaces/topics" {
		t.Fatalf("unexpected topic identity fields: %v", createdTopic)
	}
	topicProps := createdTopic["properties"].(map[string]any)
	if topicProps["status"] != "Active" || topicProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected topic state: %v", topicProps)
	}
	if topicProps["enablePartitioning"] != true {
		t.Fatalf("unexpected enablePartitioning: %v", topicProps["enablePartitioning"])
	}

	listTopicsResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("list topics returned error: %v", err)
	}
	listedTopics := decodeServiceBusResponse(t, listTopicsResp)
	topicValues := listedTopics["value"].([]any)
	if len(topicValues) != 1 {
		t.Fatalf("expected one topic in list, got %d in %v", len(topicValues), listedTopics)
	}

	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a?api-version=2024-01-01"
	subscriptionPayload := []byte(`{
		"properties":{
			"lockDuration":"PT30S",
			"maxDeliveryCount":5,
			"deadLetteringOnMessageExpiration":true
		}
	}`)
	createSubscriptionResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, subscriptionURL, subscriptionPayload))
	if err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	}
	if createSubscriptionResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subscription status 201, got %d; body=%s", createSubscriptionResp.StatusCode, string(createSubscriptionResp.RawBody))
	}
	createdSubscription := decodeServiceBusResponse(t, createSubscriptionResp)
	if createdSubscription["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a" {
		t.Fatalf("unexpected subscription id: %v", createdSubscription["id"])
	}
	if createdSubscription["name"] != "ns-a/topic-a/sub-a" || createdSubscription["type"] != "Microsoft.ServiceBus/namespaces/topics/subscriptions" {
		t.Fatalf("unexpected subscription identity fields: %v", createdSubscription)
	}
	subscriptionProps := createdSubscription["properties"].(map[string]any)
	if subscriptionProps["status"] != "Active" || subscriptionProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected subscription state: %v", subscriptionProps)
	}
	if subscriptionProps["maxDeliveryCount"].(float64) != 5 {
		t.Fatalf("unexpected maxDeliveryCount: %v", subscriptionProps["maxDeliveryCount"])
	}

	listSubscriptionsResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("list subscriptions returned error: %v", err)
	}
	listedSubscriptions := decodeServiceBusResponse(t, listSubscriptionsResp)
	subscriptionValues := listedSubscriptions["value"].([]any)
	if len(subscriptionValues) != 1 {
		t.Fatalf("expected one subscription in list, got %d in %v", len(subscriptionValues), listedSubscriptions)
	}

	deleteSubscriptionResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, subscriptionURL, nil))
	if err != nil {
		t.Fatalf("delete subscription returned error: %v", err)
	}
	if deleteSubscriptionResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete subscription status 202, got %d; body=%s", deleteSubscriptionResp.StatusCode, string(deleteSubscriptionResp.RawBody))
	}

	deleteTopicResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, topicURL, nil))
	if err != nil {
		t.Fatalf("delete topic returned error: %v", err)
	}
	if deleteTopicResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete topic status 202, got %d; body=%s", deleteTopicResp.StatusCode, string(deleteTopicResp.RawBody))
	}
}

func TestTopicSubscriptionTemplateProvisioning(t *testing.T) {
	svc := New()

	namespaceResource := map[string]any{
		"type":     "Microsoft.ServiceBus/namespaces",
		"name":     "ns-a",
		"location": "eastus",
		"sku": map[string]any{
			"name": "Standard",
			"tier": "Standard",
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", namespaceResource); err != nil {
		t.Fatalf("provision namespace returned error: %v", err)
	}

	topicResource := map[string]any{
		"type": "Microsoft.ServiceBus/namespaces/topics",
		"name": "ns-a/topic-a",
		"properties": map[string]any{
			"enablePartitioning": true,
		},
	}
	topicResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", topicResource)
	if err != nil {
		t.Fatalf("provision topic returned error: %v", err)
	}
	topic := topicResult.(map[string]any)
	if topic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a" {
		t.Fatalf("unexpected provisioned topic id: %v", topic["id"])
	}
	if topic["type"] != "Microsoft.ServiceBus/namespaces/topics" {
		t.Fatalf("unexpected provisioned topic type: %v", topic["type"])
	}

	subscriptionResource := map[string]any{
		"type": "Microsoft.ServiceBus/namespaces/topics/subscriptions",
		"name": "ns-a/topic-a/sub-a",
		"properties": map[string]any{
			"maxDeliveryCount": 3,
		},
	}
	subscriptionResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", subscriptionResource)
	if err != nil {
		t.Fatalf("provision subscription returned error: %v", err)
	}
	subscription := subscriptionResult.(map[string]any)
	if subscription["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a" {
		t.Fatalf("unexpected provisioned subscription id: %v", subscription["id"])
	}
	if subscription["type"] != "Microsoft.ServiceBus/namespaces/topics/subscriptions" {
		t.Fatalf("unexpected provisioned subscription type: %v", subscription["type"])
	}
}

func TestSubscriptionRuleLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"maxDeliveryCount":5}}`))); err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/rule-a?api-version=2024-01-01"
	createRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, []byte(`{
		"properties":{
			"filterType":"SqlFilter",
			"sqlFilter":{"sqlExpression":"color = 'blue'"},
			"action":{"sqlExpression":"SET handled = true"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create rule returned error: %v", err)
	}
	if createRuleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create rule status 200, got %d; body=%s", createRuleResp.StatusCode, string(createRuleResp.RawBody))
	}
	createdRule := decodeServiceBusResponse(t, createRuleResp)
	if createdRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/rule-a" {
		t.Fatalf("unexpected rule id: %v", createdRule["id"])
	}
	if createdRule["name"] != "rule-a" || createdRule["type"] != "Microsoft.ServiceBus/Namespaces/Topics/Subscriptions/Rules" {
		t.Fatalf("unexpected rule identity fields: %v", createdRule)
	}
	ruleProps := createdRule["properties"].(map[string]any)
	if ruleProps["filterType"] != "SqlFilter" {
		t.Fatalf("expected SQL filter type, got %v", ruleProps)
	}
	sqlFilter := ruleProps["sqlFilter"].(map[string]any)
	if sqlFilter["sqlExpression"] != "color = 'blue'" || sqlFilter["compatibilityLevel"].(float64) != 20 {
		t.Fatalf("expected SQL filter with compatibility level, got %v", sqlFilter)
	}
	action := ruleProps["action"].(map[string]any)
	if action["sqlExpression"] != "SET handled = true" || action["compatibilityLevel"].(float64) != 20 {
		t.Fatalf("expected action with compatibility level, got %v", action)
	}

	defaultRuleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/default-rule?api-version=2024-01-01"
	defaultRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, defaultRuleURL, []byte(`{}`)))
	if err != nil {
		t.Fatalf("create default rule returned error: %v", err)
	}
	defaultRule := decodeServiceBusResponse(t, defaultRuleResp)
	defaultProps := defaultRule["properties"].(map[string]any)
	if defaultProps["filterType"] != "SqlFilter" || defaultProps["sqlFilter"].(map[string]any)["sqlExpression"] != "1=1" {
		t.Fatalf("expected default SQL rule, got %v", defaultProps)
	}

	correlationRuleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/correlation-rule?api-version=2024-01-01"
	correlationRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, correlationRuleURL, []byte(`{
		"properties":{
			"filterType":"CorrelationFilter",
			"correlationFilter":{"correlationId":"order-123","properties":{"color":"blue"}}
		}
	}`)))
	if err != nil {
		t.Fatalf("create correlation rule returned error: %v", err)
	}
	correlationRule := decodeServiceBusResponse(t, correlationRuleResp)
	correlationProps := correlationRule["properties"].(map[string]any)
	if correlationProps["filterType"] != "CorrelationFilter" || correlationProps["correlationFilter"].(map[string]any)["correlationId"] != "order-123" {
		t.Fatalf("expected correlation rule properties, got %v", correlationProps)
	}

	getRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get rule returned error: %v", err)
	}
	if getRuleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get rule status 200, got %d; body=%s", getRuleResp.StatusCode, string(getRuleResp.RawBody))
	}

	listRulesResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("list rules returned error: %v", err)
	}
	listedRules := decodeServiceBusResponse(t, listRulesResp)
	ruleValues := listedRules["value"].([]any)
	if len(ruleValues) != 3 || ruleValues[0].(map[string]any)["name"] != "correlation-rule" || ruleValues[1].(map[string]any)["name"] != "default-rule" || ruleValues[2].(map[string]any)["name"] != "rule-a" {
		t.Fatalf("expected stable sorted rules, got %v", listedRules)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.ServiceBus/namespaces/topics/subscriptions/rules",
		"name": "ns-a/topic-a/sub-a/template-rule",
		"properties": map[string]any{
			"filterType": "SqlFilter",
			"sqlFilter":  map[string]any{"sqlExpression": "priority > 5"},
		},
	})
	if err != nil {
		t.Fatalf("provision rule returned error: %v", err)
	}
	templateRule := templateResult.(map[string]any)
	if templateRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a/rules/template-rule" {
		t.Fatalf("unexpected provisioned rule id: %v", templateRule["id"])
	}

	deleteRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete rule returned error: %v", err)
	}
	if deleteRuleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete rule status 200, got %d; body=%s", deleteRuleResp.StatusCode, string(deleteRuleResp.RawBody))
	}
	missingDeleteResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete missing rule returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing rule status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}

	missingSubscriptionResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/missing/rules/orphan?api-version=2024-01-01", []byte(`{}`)))
	if err != nil {
		t.Fatalf("create rule under missing subscription returned error: %v", err)
	}
	if missingSubscriptionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing subscription status 404, got %d; body=%s", missingSubscriptionResp.StatusCode, string(missingSubscriptionResp.RawBody))
	}
}

func TestQueueRuntimeSendPeekLockCompleteAndReceiveDelete(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("hello service bus")))
	if err != nil {
		t.Fatalf("send message returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("peek-lock message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected peek-lock status 201, got %d; body=%s", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}
	if string(peekLockResp.RawBody) != "hello service bus" {
		t.Fatalf("unexpected peek-lock body: %q", string(peekLockResp.RawBody))
	}
	location := peekLockResp.Headers["Location"]
	if location == "" {
		t.Fatalf("expected Location header, got headers=%v", peekLockResp.Headers)
	}
	var broker map[string]any
	if err := gojson.Unmarshal([]byte(peekLockResp.Headers["BrokerProperties"]), &broker); err != nil {
		t.Fatalf("decode BrokerProperties: %v; value=%s", err, peekLockResp.Headers["BrokerProperties"])
	}
	if broker["LockToken"] == "" || broker["MessageId"] == "" || broker["SequenceNumber"] == nil {
		t.Fatalf("unexpected broker properties: %v", broker)
	}

	completeResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, location, nil))
	if err != nil {
		t.Fatalf("complete message returned error: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected complete status 200, got %d; body=%s", completeResp.StatusCode, string(completeResp.RawBody))
	}

	emptyAfterComplete, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete after complete returned error: %v", err)
	}
	if emptyAfterComplete.StatusCode != http.StatusNoContent {
		t.Fatalf("expected empty receive-delete status 204, got %d; body=%s", emptyAfterComplete.StatusCode, string(emptyAfterComplete.RawBody))
	}

	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("delete me"))); err != nil {
		t.Fatalf("send second message returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected second send status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	receiveDeleteResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete message returned error: %v", err)
	}
	if receiveDeleteResp.StatusCode != http.StatusOK || string(receiveDeleteResp.RawBody) != "delete me" {
		t.Fatalf("expected receive-delete body, got status=%d body=%q", receiveDeleteResp.StatusCode, string(receiveDeleteResp.RawBody))
	}
}

func TestQueueRuntimeSendMessageBatchPreservesMessageProperties(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	batchBody := []byte(`[
		{"Body":"first batch message","BrokerProperties":{"Label":"M1"},"UserProperties":{"Priority":"Low"}},
		{"Body":"second batch message","BrokerProperties":{"Label":"M2"},"UserProperties":{"Priority":"High","Customer":"ABC"}}
	]`)
	ctx := serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages?timeout=60", batchBody)
	ctx.RawRequest.Header.Set("Content-Type", "application/vnd.microsoft.servicebus.json")
	sendResp, err := svc.HandleRequest(ctx)
	if err != nil {
		t.Fatalf("send batch returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send batch status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	firstResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive first batch message returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK || string(firstResp.RawBody) != "first batch message" {
		t.Fatalf("expected first batch body, got status=%d body=%q", firstResp.StatusCode, string(firstResp.RawBody))
	}
	if firstResp.Headers["Priority"] != "Low" {
		t.Fatalf("expected first user property Priority=Low, got headers=%v", firstResp.Headers)
	}
	var firstBroker map[string]any
	if err := gojson.Unmarshal([]byte(firstResp.Headers["BrokerProperties"]), &firstBroker); err != nil {
		t.Fatalf("decode first BrokerProperties: %v; value=%s", err, firstResp.Headers["BrokerProperties"])
	}
	if firstBroker["Label"] != "M1" {
		t.Fatalf("expected first broker label M1, got %v", firstBroker)
	}

	secondResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive second batch message returned error: %v", err)
	}
	if secondResp.StatusCode != http.StatusOK || string(secondResp.RawBody) != "second batch message" {
		t.Fatalf("expected second batch body, got status=%d body=%q", secondResp.StatusCode, string(secondResp.RawBody))
	}
	if secondResp.Headers["Priority"] != "High" || secondResp.Headers["Customer"] != "ABC" {
		t.Fatalf("expected second user properties, got headers=%v", secondResp.Headers)
	}
	var secondBroker map[string]any
	if err := gojson.Unmarshal([]byte(secondResp.Headers["BrokerProperties"]), &secondBroker); err != nil {
		t.Fatalf("decode second BrokerProperties: %v; value=%s", err, secondResp.Headers["BrokerProperties"])
	}
	if secondBroker["Label"] != "M2" {
		t.Fatalf("expected second broker label M2, got %v", secondBroker)
	}
}

func TestQueueRuntimeReceivePreservesSettableBrokerProperties(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendCtx := serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("brokered"))
	sendCtx.RawRequest.Header.Set("BrokerProperties", `{"MessageId":"client-message","Label":"M1","TimeToLive":10,"DeliveryCount":9,"SequenceNumber":99}`)
	sendResp, err := svc.HandleRequest(sendCtx)
	if err != nil {
		t.Fatalf("send brokered message returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	receiveResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive brokered message returned error: %v", err)
	}
	if receiveResp.StatusCode != http.StatusOK || string(receiveResp.RawBody) != "brokered" {
		t.Fatalf("expected brokered message body, got status=%d body=%q", receiveResp.StatusCode, string(receiveResp.RawBody))
	}
	var broker map[string]any
	if err := gojson.Unmarshal([]byte(receiveResp.Headers["BrokerProperties"]), &broker); err != nil {
		t.Fatalf("decode received BrokerProperties: %v; value=%s", err, receiveResp.Headers["BrokerProperties"])
	}
	if broker["MessageId"] != "client-message" || broker["Label"] != "M1" {
		t.Fatalf("expected settable broker properties to round-trip, got %v", broker)
	}
	if broker["TimeToLive"] != float64(10) {
		t.Fatalf("expected TimeToLive to round-trip as 10, got %v in %v", broker["TimeToLive"], broker)
	}
	if broker["DeliveryCount"] != float64(1) || broker["SequenceNumber"] != float64(1) {
		t.Fatalf("expected response-owned broker properties from runtime, got %v", broker)
	}
}

func TestQueueRuntimeMovesExpiredMessagesToDeadLetterQueue(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"deadLetteringOnMessageExpiration":true}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendCtx := serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("expired payload"))
	sendCtx.RawRequest.Header.Set("BrokerProperties", `{"MessageId":"ttl-message","TimeToLive":0}`)
	sendResp, err := svc.HandleRequest(sendCtx)
	if err != nil {
		t.Fatalf("send expiring message returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	activeResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive active queue message returned error: %v", err)
	}
	if activeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected active queue to be empty after TTL expiry, got status=%d body=%q", activeResp.StatusCode, string(activeResp.RawBody))
	}

	deadLetterResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/$deadletterqueue/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive expired dead-letter message returned error: %v", err)
	}
	if deadLetterResp.StatusCode != http.StatusOK || string(deadLetterResp.RawBody) != "expired payload" {
		t.Fatalf("expected expired dead-letter message, got status=%d body=%q", deadLetterResp.StatusCode, string(deadLetterResp.RawBody))
	}
	var broker map[string]any
	if err := gojson.Unmarshal([]byte(deadLetterResp.Headers["BrokerProperties"]), &broker); err != nil {
		t.Fatalf("decode expired dead-letter BrokerProperties: %v; value=%s", err, deadLetterResp.Headers["BrokerProperties"])
	}
	if broker["MessageId"] != "ttl-message" || broker["DeadLetterReason"] != "TTLExpiredException" {
		t.Fatalf("expected ttl-message with TTLExpiredException, got %v", broker)
	}
}

func TestQueueRuntimeReceivePreservesContentType(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendCtx := serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte(`{"kind":"json"}`))
	sendCtx.RawRequest.Header.Set("Content-Type", "application/json")
	sendResp, err := svc.HandleRequest(sendCtx)
	if err != nil {
		t.Fatalf("send JSON message returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	receiveResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive JSON message returned error: %v", err)
	}
	if receiveResp.StatusCode != http.StatusOK || string(receiveResp.RawBody) != `{"kind":"json"}` {
		t.Fatalf("expected JSON message body, got status=%d body=%q", receiveResp.StatusCode, string(receiveResp.RawBody))
	}
	if receiveResp.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected received Content-Type application/json, got %q", receiveResp.Headers["Content-Type"])
	}
	if receiveResp.RawContentType != "application/json" {
		t.Fatalf("expected raw content type application/json, got %q", receiveResp.RawContentType)
	}
}

func TestQueueRuntimeUnlockMakesLockedMessageAvailable(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("retry later"))); err != nil {
		t.Fatalf("send message returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("peek-lock message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated || string(peekLockResp.RawBody) != "retry later" {
		t.Fatalf("expected locked message, got status=%d body=%q", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}
	location := peekLockResp.Headers["Location"]
	if location == "" {
		t.Fatalf("expected locked message Location, got headers=%v", peekLockResp.Headers)
	}

	lockedReceiveResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete while locked returned error: %v", err)
	}
	if lockedReceiveResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected locked message to be unavailable, got status=%d body=%s", lockedReceiveResp.StatusCode, string(lockedReceiveResp.RawBody))
	}

	unlockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, location, nil))
	if err != nil {
		t.Fatalf("unlock message returned error: %v", err)
	}
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected unlock status 200, got %d; body=%s", unlockResp.StatusCode, string(unlockResp.RawBody))
	}

	availableAgainResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete after unlock returned error: %v", err)
	}
	if availableAgainResp.StatusCode != http.StatusOK || string(availableAgainResp.RawBody) != "retry later" {
		t.Fatalf("expected unlocked message body, got status=%d body=%q", availableAgainResp.StatusCode, string(availableAgainResp.RawBody))
	}
}

func TestQueueRuntimeExpiredLockMakesMessageAvailableAgain(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT0S","maxDeliveryCount":3}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("redeliver me"))); err != nil {
		t.Fatalf("send message returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("peek-lock message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated || string(peekLockResp.RawBody) != "redeliver me" {
		t.Fatalf("expected locked message, got status=%d body=%q", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}

	redeliveredResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete after lock expiry returned error: %v", err)
	}
	if redeliveredResp.StatusCode != http.StatusOK || string(redeliveredResp.RawBody) != "redeliver me" {
		t.Fatalf("expected expired lock message to be redelivered, got status=%d body=%q", redeliveredResp.StatusCode, string(redeliveredResp.RawBody))
	}
	var broker map[string]any
	if err := gojson.Unmarshal([]byte(redeliveredResp.Headers["BrokerProperties"]), &broker); err != nil {
		t.Fatalf("decode redelivered BrokerProperties: %v; value=%s", err, redeliveredResp.Headers["BrokerProperties"])
	}
	if broker["DeliveryCount"] != float64(2) {
		t.Fatalf("expected redelivered message DeliveryCount=2, got %v in %v", broker["DeliveryCount"], broker)
	}
}

func TestQueueRuntimeRenewLockKeepsMessageLocked(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	queueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/queues/queue-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, queueURL, []byte(`{"properties":{"lockDuration":"PT30S"}}`))); err != nil {
		t.Fatalf("create queue returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages", []byte("still processing"))); err != nil {
		t.Fatalf("send message returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("peek-lock message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated || string(peekLockResp.RawBody) != "still processing" {
		t.Fatalf("expected locked message, got status=%d body=%q", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}
	location := peekLockResp.Headers["Location"]
	if location == "" {
		t.Fatalf("expected locked message Location, got headers=%v", peekLockResp.Headers)
	}

	renewResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, location, nil))
	if err != nil {
		t.Fatalf("renew lock returned error: %v", err)
	}
	if renewResp.StatusCode != http.StatusOK {
		t.Fatalf("expected renew-lock status 200, got %d; body=%s", renewResp.StatusCode, string(renewResp.RawBody))
	}

	lockedReceiveResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, "https://ns-a.servicebus.windows.net/queue-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("receive-delete while renewed lock active returned error: %v", err)
	}
	if lockedReceiveResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected renewed lock to keep message unavailable, got status=%d body=%s", lockedReceiveResp.StatusCode, string(lockedReceiveResp.RawBody))
	}

	completeResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, location, nil))
	if err != nil {
		t.Fatalf("complete renewed message returned error: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected complete status 200, got %d; body=%s", completeResp.StatusCode, string(completeResp.RawBody))
	}
}

func TestRuntimeActionsExposeLockManagement(t *testing.T) {
	svc := New()
	actions := map[string]string{}
	for _, action := range svc.Actions() {
		actions[action.Name] = action.Method
	}

	if actions["SendMessageBatch"] != http.MethodPost {
		t.Fatalf("expected SendMessageBatch action to use POST, got %q in %v", actions["SendMessageBatch"], actions)
	}
	if actions["UnlockMessage"] != http.MethodPut {
		t.Fatalf("expected UnlockMessage action to use PUT, got %q in %v", actions["UnlockMessage"], actions)
	}
	if actions["RenewLockMessage"] != http.MethodPost {
		t.Fatalf("expected RenewLockMessage action to use POST, got %q in %v", actions["RenewLockMessage"], actions)
	}
}

func TestTopicRuntimeFansOutToSubscription(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/sub-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/topic-a/messages", []byte("hello subscriber")))
	if err != nil {
		t.Fatalf("send topic message returned error: %v", err)
	}
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected topic send status 201, got %d; body=%s", sendResp.StatusCode, string(sendResp.RawBody))
	}

	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, "https://ns-a.servicebus.windows.net/topic-a/subscriptions/sub-a/messages/head?timeout=60", nil))
	if err != nil {
		t.Fatalf("peek-lock subscription message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated || string(peekLockResp.RawBody) != "hello subscriber" {
		t.Fatalf("expected subscription message, got status=%d body=%q", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}
	location := peekLockResp.Headers["Location"]
	if location == "" || !strings.Contains(location, "/topic-a/subscriptions/sub-a/messages/") {
		t.Fatalf("unexpected subscription message location: %q", location)
	}
	completeResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, location, nil))
	if err != nil {
		t.Fatalf("complete subscription message returned error: %v", err)
	}
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected complete subscription status 200, got %d; body=%s", completeResp.StatusCode, string(completeResp.RawBody))
	}
}

func TestTopicRuntimeAppliesSubscriptionRules(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	for _, subscriptionName := range []string{"all-sub", "blue-sub", "corr-sub"} {
		subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/" + subscriptionName + "?api-version=2024-01-01"
		if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{}}`))); err != nil {
			t.Fatalf("create subscription %s returned error: %v", subscriptionName, err)
		} else if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create subscription %s status 201, got %d; body=%s", subscriptionName, resp.StatusCode, string(resp.RawBody))
		}
	}

	rules := map[string][]byte{
		"all-sub/rules/default-rule": []byte(`{"properties":{"filterType":"SqlFilter","sqlFilter":{"sqlExpression":"1=1"}}}`),
		"blue-sub/rules/blue-rule":   []byte(`{"properties":{"filterType":"SqlFilter","sqlFilter":{"sqlExpression":"color = 'blue'"}}}`),
		"corr-sub/rules/corr-rule":   []byte(`{"properties":{"filterType":"CorrelationFilter","correlationFilter":{"correlationId":"order-123","properties":{"tenant":"a"}}}}`),
	}
	for path, payload := range rules {
		ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/ns-a/topics/topic-a/subscriptions/" + path + "?api-version=2024-01-01"
		if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, payload)); err != nil {
			t.Fatalf("create rule %s returned error: %v", path, err)
		} else if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected create rule %s status 200, got %d; body=%s", path, resp.StatusCode, string(resp.RawBody))
		}
	}

	sendServiceBusMessage(t, svc, "red payload", map[string]string{
		"Color":            "red",
		"Tenant":           "a",
		"BrokerProperties": `{"CorrelationId":"order-999"}`,
	})
	sendServiceBusMessage(t, svc, "blue payload", map[string]string{
		"Color":            "blue",
		"BrokerProperties": `{"CorrelationId":"order-999"}`,
	})
	sendServiceBusMessage(t, svc, "corr payload", map[string]string{
		"Tenant":           "a",
		"BrokerProperties": `{"CorrelationId":"order-123"}`,
	})

	assertServiceBusReceiveDelete(t, svc, "all-sub", http.StatusOK, "red payload")
	assertServiceBusReceiveDelete(t, svc, "all-sub", http.StatusOK, "blue payload")
	assertServiceBusReceiveDelete(t, svc, "all-sub", http.StatusOK, "corr payload")
	assertServiceBusReceiveDelete(t, svc, "all-sub", http.StatusNoContent, "")

	assertServiceBusReceiveDelete(t, svc, "blue-sub", http.StatusOK, "blue payload")
	assertServiceBusReceiveDelete(t, svc, "blue-sub", http.StatusNoContent, "")

	assertServiceBusReceiveDelete(t, svc, "corr-sub", http.StatusOK, "corr payload")
	assertServiceBusReceiveDelete(t, svc, "corr-sub", http.StatusNoContent, "")
}

func TestTopicRuntimeMovesMessagesToDeadLetterQueueAfterMaxDeliveryCount(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	topicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns/topics/topic-sdk?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, topicURL, []byte(`{"properties":{}}`))); err != nil {
		t.Fatalf("create topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create topic status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns/topics/topic-sdk/subscriptions/sub-sdk?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, subscriptionURL, []byte(`{"properties":{"maxDeliveryCount":1}}`))); err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subscription status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "poison payload", nil)

	peekLockURL := "https://sdk-ns.servicebus.windows.net/topic-sdk/subscriptions/sub-sdk/messages/head?timeout=60"
	peekLockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPost, peekLockURL, nil))
	if err != nil {
		t.Fatalf("peek-lock subscription message returned error: %v", err)
	}
	if peekLockResp.StatusCode != http.StatusCreated || string(peekLockResp.RawBody) != "poison payload" {
		t.Fatalf("expected peek-lock message, got status=%d body=%q", peekLockResp.StatusCode, string(peekLockResp.RawBody))
	}
	unlockResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, peekLockResp.Headers["Location"], nil))
	if err != nil {
		t.Fatalf("unlock subscription message returned error: %v", err)
	}
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected unlock status 200, got %d; body=%s", unlockResp.StatusCode, string(unlockResp.RawBody))
	}

	assertServiceBusReceiveDeleteFrom(t, svc, "sdk-ns", "topic-sdk", "sub-sdk", http.StatusNoContent, "")

	deadLetterURL := "https://sdk-ns.servicebus.windows.net/topic-sdk/subscriptions/sub-sdk/$deadletterqueue/messages/head?timeout=60"
	deadLetterResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, deadLetterURL, nil))
	if err != nil {
		t.Fatalf("receive-delete dead-letter message returned error: %v", err)
	}
	if deadLetterResp.StatusCode != http.StatusOK || string(deadLetterResp.RawBody) != "poison payload" {
		t.Fatalf("expected dead-letter message, got status=%d body=%q", deadLetterResp.StatusCode, string(deadLetterResp.RawBody))
	}
	var brokerProperties map[string]any
	if err := gojson.Unmarshal([]byte(deadLetterResp.Headers["BrokerProperties"]), &brokerProperties); err != nil {
		t.Fatalf("decode dead-letter BrokerProperties: %v", err)
	}
	if brokerProperties["DeadLetterReason"] != "MaxDeliveryCountExceeded" {
		t.Fatalf("expected MaxDeliveryCountExceeded, got BrokerProperties=%v", brokerProperties)
	}
}

func TestFlociStyleServiceBusAdministrationPaths(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	namespaceInfoResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/$namespaceinfo", nil))
	if err != nil {
		t.Fatalf("get floci namespace info returned error: %v", err)
	}
	if namespaceInfoResp.StatusCode != http.StatusOK {
		t.Fatalf("expected floci namespace info status 200, got %d; body=%s", namespaceInfoResp.StatusCode, string(namespaceInfoResp.RawBody))
	}
	namespaceInfoBody := string(namespaceInfoResp.RawBody)
	if namespaceInfoResp.RawContentType != "application/atom+xml;charset=utf-8" || !strings.Contains(namespaceInfoBody, "<NamespaceInfo") || !strings.Contains(namespaceInfoBody, "<Name>sdk-ns</Name>") {
		t.Fatalf("unexpected floci namespace info response: contentType=%q body=%s", namespaceInfoResp.RawContentType, namespaceInfoBody)
	}

	createQueueResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/queue-sdk", []byte(`<entry><content><QueueDescription><RequiresSession>true</RequiresSession></QueueDescription></content></entry>`)))
	if err != nil {
		t.Fatalf("create floci queue returned error: %v", err)
	}
	if createQueueResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci queue create status 201, got %d; body=%s", createQueueResp.StatusCode, string(createQueueResp.RawBody))
	}
	queueBody := string(createQueueResp.RawBody)
	if !strings.Contains(queueBody, "<QueueDescription") || !strings.Contains(queueBody, "<title type=\"text\">queue-sdk</title>") || !strings.Contains(queueBody, "<RequiresSession>true</RequiresSession>") {
		t.Fatalf("unexpected floci queue create body: %s", queueBody)
	}

	getQueueResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/queue-sdk", nil))
	if err != nil {
		t.Fatalf("get floci queue returned error: %v", err)
	}
	if getQueueResp.StatusCode != http.StatusOK || !strings.Contains(string(getQueueResp.RawBody), "<QueueDescription") {
		t.Fatalf("expected floci queue get body, got status=%d body=%s", getQueueResp.StatusCode, string(getQueueResp.RawBody))
	}

	listQueuesResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/$Resources/queues", nil))
	if err != nil {
		t.Fatalf("list floci queues returned error: %v", err)
	}
	listQueuesBody := string(listQueuesResp.RawBody)
	if listQueuesResp.StatusCode != http.StatusOK || !strings.Contains(listQueuesBody, "<feed") || !strings.Contains(listQueuesBody, "queue-sdk") {
		t.Fatalf("expected floci queues feed to include queue-sdk, got status=%d body=%s", listQueuesResp.StatusCode, listQueuesBody)
	}

	createTopicResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk", []byte(`<entry><content><TopicDescription /></content></entry>`)))
	if err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	topicBody := string(createTopicResp.RawBody)
	if !strings.Contains(topicBody, "<TopicDescription") || !strings.Contains(topicBody, "<title type=\"text\">topic-sdk</title>") {
		t.Fatalf("unexpected floci topic create body: %s", topicBody)
	}

	getTopicResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/topic-sdk", nil))
	if err != nil {
		t.Fatalf("get floci topic returned error: %v", err)
	}
	if getTopicResp.StatusCode != http.StatusOK || !strings.Contains(string(getTopicResp.RawBody), "<TopicDescription") {
		t.Fatalf("expected floci topic get body, got status=%d body=%s", getTopicResp.StatusCode, string(getTopicResp.RawBody))
	}

	listTopicsResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/$Resources/topics", nil))
	if err != nil {
		t.Fatalf("list floci topics returned error: %v", err)
	}
	listTopicsBody := string(listTopicsResp.RawBody)
	if listTopicsResp.StatusCode != http.StatusOK || !strings.Contains(listTopicsBody, "<feed") || !strings.Contains(listTopicsBody, "topic-sdk") {
		t.Fatalf("expected floci topics feed to include topic-sdk, got status=%d body=%s", listTopicsResp.StatusCode, listTopicsBody)
	}
}

func TestFlociStyleServiceBusAdministrationPreservesEntityProperties(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	queueXML := []byte(`<entry><content><QueueDescription>` +
		`<MaxSizeInMegabytes>2048</MaxSizeInMegabytes>` +
		`<DefaultMessageTimeToLive>PT2H</DefaultMessageTimeToLive>` +
		`<LockDuration>PT45S</LockDuration>` +
		`<MaxDeliveryCount>7</MaxDeliveryCount>` +
		`<RequiresDuplicateDetection>true</RequiresDuplicateDetection>` +
		`<RequiresSession>true</RequiresSession>` +
		`<DeadLetteringOnMessageExpiration>true</DeadLetteringOnMessageExpiration>` +
		`<EnableBatchedOperations>false</EnableBatchedOperations>` +
		`<AutoDeleteOnIdle>P1D</AutoDeleteOnIdle>` +
		`<UserMetadata>owner=platform</UserMetadata>` +
		`</QueueDescription></content></entry>`)
	createQueueResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/queue-rich", queueXML))
	if err != nil {
		t.Fatalf("create floci queue returned error: %v", err)
	}
	if createQueueResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci queue create status 201, got %d; body=%s", createQueueResp.StatusCode, string(createQueueResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(createQueueResp.RawBody),
		"<MaxSizeInMegabytes>2048</MaxSizeInMegabytes>",
		"<DefaultMessageTimeToLive>PT2H</DefaultMessageTimeToLive>",
		"<LockDuration>PT45S</LockDuration>",
		"<MaxDeliveryCount>7</MaxDeliveryCount>",
		"<RequiresDuplicateDetection>true</RequiresDuplicateDetection>",
		"<RequiresSession>true</RequiresSession>",
		"<DeadLetteringOnMessageExpiration>true</DeadLetteringOnMessageExpiration>",
		"<EnableBatchedOperations>false</EnableBatchedOperations>",
		"<AutoDeleteOnIdle>P1D</AutoDeleteOnIdle>",
		"<UserMetadata>owner=platform</UserMetadata>",
	)

	topicXML := []byte(`<entry><content><TopicDescription>` +
		`<MaxSizeInMegabytes>3072</MaxSizeInMegabytes>` +
		`<DefaultMessageTimeToLive>PT3H</DefaultMessageTimeToLive>` +
		`<RequiresDuplicateDetection>true</RequiresDuplicateDetection>` +
		`<EnableBatchedOperations>false</EnableBatchedOperations>` +
		`<SupportOrdering>true</SupportOrdering>` +
		`<AutoDeleteOnIdle>P2D</AutoDeleteOnIdle>` +
		`<UserMetadata>topic-meta</UserMetadata>` +
		`</TopicDescription></content></entry>`)
	createTopicResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-rich", topicXML))
	if err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	}
	if createTopicResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", createTopicResp.StatusCode, string(createTopicResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(createTopicResp.RawBody),
		"<MaxSizeInMegabytes>3072</MaxSizeInMegabytes>",
		"<DefaultMessageTimeToLive>PT3H</DefaultMessageTimeToLive>",
		"<RequiresDuplicateDetection>true</RequiresDuplicateDetection>",
		"<EnableBatchedOperations>false</EnableBatchedOperations>",
		"<SupportOrdering>true</SupportOrdering>",
		"<AutoDeleteOnIdle>P2D</AutoDeleteOnIdle>",
		"<UserMetadata>topic-meta</UserMetadata>",
	)

	subscriptionXML := []byte(`<entry><content><SubscriptionDescription>` +
		`<LockDuration>PT1M</LockDuration>` +
		`<MaxDeliveryCount>9</MaxDeliveryCount>` +
		`<RequiresSession>true</RequiresSession>` +
		`<DefaultMessageTimeToLive>PT4H</DefaultMessageTimeToLive>` +
		`<DeadLetteringOnMessageExpiration>true</DeadLetteringOnMessageExpiration>` +
		`<DeadLetteringOnFilterEvaluationExceptions>false</DeadLetteringOnFilterEvaluationExceptions>` +
		`<EnableBatchedOperations>false</EnableBatchedOperations>` +
		`<AutoDeleteOnIdle>P3D</AutoDeleteOnIdle>` +
		`<UserMetadata>sub-meta</UserMetadata>` +
		`</SubscriptionDescription></content></entry>`)
	createSubResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-rich/subscriptions/sub-rich", subscriptionXML))
	if err != nil {
		t.Fatalf("create floci subscription returned error: %v", err)
	}
	if createSubResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci subscription create status 201, got %d; body=%s", createSubResp.StatusCode, string(createSubResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(createSubResp.RawBody),
		"<LockDuration>PT1M</LockDuration>",
		"<MaxDeliveryCount>9</MaxDeliveryCount>",
		"<RequiresSession>true</RequiresSession>",
		"<DefaultMessageTimeToLive>PT4H</DefaultMessageTimeToLive>",
		"<DeadLetteringOnMessageExpiration>true</DeadLetteringOnMessageExpiration>",
		"<DeadLetteringOnFilterEvaluationExceptions>false</DeadLetteringOnFilterEvaluationExceptions>",
		"<EnableBatchedOperations>false</EnableBatchedOperations>",
		"<AutoDeleteOnIdle>P3D</AutoDeleteOnIdle>",
		"<UserMetadata>sub-meta</UserMetadata>",
	)
}

func TestFlociStyleServiceBusSubscriptionAdministrationPaths(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk", []byte(`<entry><content><TopicDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	createSubResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk/subscriptions/sub-sdk", []byte(`<entry><content><SubscriptionDescription><RequiresSession>true</RequiresSession></SubscriptionDescription></content></entry>`)))
	if err != nil {
		t.Fatalf("create floci subscription returned error: %v", err)
	}
	if createSubResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci subscription create status 201, got %d; body=%s", createSubResp.StatusCode, string(createSubResp.RawBody))
	}
	createSubBody := string(createSubResp.RawBody)
	if !strings.Contains(createSubBody, "<SubscriptionDescription") || !strings.Contains(createSubBody, "<title type=\"text\">sub-sdk</title>") || !strings.Contains(createSubBody, "<RequiresSession>true</RequiresSession>") {
		t.Fatalf("unexpected floci subscription create body: %s", createSubBody)
	}

	getSubResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/topic-sdk/subscriptions/sub-sdk", nil))
	if err != nil {
		t.Fatalf("get floci subscription returned error: %v", err)
	}
	if getSubResp.StatusCode != http.StatusOK || !strings.Contains(string(getSubResp.RawBody), "<SubscriptionDescription") {
		t.Fatalf("expected floci subscription get body, got status=%d body=%s", getSubResp.StatusCode, string(getSubResp.RawBody))
	}

	listSubsResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/topic-sdk/subscriptions", nil))
	if err != nil {
		t.Fatalf("list floci subscriptions returned error: %v", err)
	}
	listSubsBody := string(listSubsResp.RawBody)
	if listSubsResp.StatusCode != http.StatusOK || !strings.Contains(listSubsBody, "<feed") || !strings.Contains(listSubsBody, "sub-sdk") {
		t.Fatalf("expected floci subscriptions feed to include sub-sdk, got status=%d body=%s", listSubsResp.StatusCode, listSubsBody)
	}

	deleteSubResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, baseURL+"/topic-sdk/subscriptions/sub-sdk", nil))
	if err != nil {
		t.Fatalf("delete floci subscription returned error: %v", err)
	}
	if deleteSubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected floci subscription delete status 200, got %d; body=%s", deleteSubResp.StatusCode, string(deleteSubResp.RawBody))
	}

	missingSubResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/topic-sdk/subscriptions/sub-sdk", nil))
	if err != nil {
		t.Fatalf("get deleted floci subscription returned error: %v", err)
	}
	if missingSubResp.StatusCode != http.StatusNotFound || !strings.Contains(string(missingSubResp.RawBody), "sub-sdk") {
		t.Fatalf("expected deleted floci subscription 404, got status=%d body=%s", missingSubResp.StatusCode, string(missingSubResp.RawBody))
	}
}

func TestFlociStyleServiceBusRuleAdministrationPaths(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk", []byte(`<entry><content><TopicDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk/subscriptions/sub-sdk", []byte(`<entry><content><SubscriptionDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci subscription create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := baseURL + "/topic-sdk/subscriptions/sub-sdk/rules/blue-rule"
	createRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, []byte(`<entry><content><RuleDescription><Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlFilter"><SqlExpression>color = 'blue'</SqlExpression></Filter></RuleDescription></content></entry>`)))
	if err != nil {
		t.Fatalf("create floci rule returned error: %v", err)
	}
	if createRuleResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci rule create status 201, got %d; body=%s", createRuleResp.StatusCode, string(createRuleResp.RawBody))
	}
	createRuleBody := string(createRuleResp.RawBody)
	if !strings.Contains(createRuleBody, "<RuleDescription") || !strings.Contains(createRuleBody, "<title type=\"text\">blue-rule</title>") || !strings.Contains(createRuleBody, "<SqlExpression>color = &apos;blue&apos;</SqlExpression>") {
		t.Fatalf("unexpected floci rule create body: %s", createRuleBody)
	}

	getRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get floci rule returned error: %v", err)
	}
	if getRuleResp.StatusCode != http.StatusOK || !strings.Contains(string(getRuleResp.RawBody), "<RuleDescription") {
		t.Fatalf("expected floci rule get body, got status=%d body=%s", getRuleResp.StatusCode, string(getRuleResp.RawBody))
	}

	listRulesResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, baseURL+"/topic-sdk/subscriptions/sub-sdk/rules", nil))
	if err != nil {
		t.Fatalf("list floci rules returned error: %v", err)
	}
	listRulesBody := string(listRulesResp.RawBody)
	if listRulesResp.StatusCode != http.StatusOK || !strings.Contains(listRulesBody, "<feed") || !strings.Contains(listRulesBody, "blue-rule") {
		t.Fatalf("expected floci rules feed to include blue-rule, got status=%d body=%s", listRulesResp.StatusCode, listRulesBody)
	}

	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "red payload", map[string]string{"Color": "red"})
	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "blue payload", map[string]string{"Color": "blue"})
	assertServiceBusReceiveDeleteFrom(t, svc, "sdk-ns", "topic-sdk", "sub-sdk", http.StatusOK, "blue payload")
	assertServiceBusReceiveDeleteFrom(t, svc, "sdk-ns", "topic-sdk", "sub-sdk", http.StatusNoContent, "")

	deleteRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete floci rule returned error: %v", err)
	}
	if deleteRuleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected floci rule delete status 200, got %d; body=%s", deleteRuleResp.StatusCode, string(deleteRuleResp.RawBody))
	}
	missingRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get deleted floci rule returned error: %v", err)
	}
	if missingRuleResp.StatusCode != http.StatusNotFound || !strings.Contains(string(missingRuleResp.RawBody), "blue-rule") {
		t.Fatalf("expected deleted floci rule 404, got status=%d body=%s", missingRuleResp.StatusCode, string(missingRuleResp.RawBody))
	}
}

func TestFlociStyleServiceBusRuleAdministrationPreservesSqlAction(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk", []byte(`<entry><content><TopicDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk/subscriptions/sub-sdk", []byte(`<entry><content><SubscriptionDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci subscription create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := baseURL + "/topic-sdk/subscriptions/sub-sdk/rules/action-rule"
	ruleXML := []byte(`<entry><content><RuleDescription>` +
		`<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlFilter"><SqlExpression>color = 'blue'</SqlExpression></Filter>` +
		`<Action xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlRuleAction"><SqlExpression>SET priority = 'high'</SqlExpression></Action>` +
		`</RuleDescription></content></entry>`)
	createRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, ruleXML))
	if err != nil {
		t.Fatalf("create floci rule with action returned error: %v", err)
	}
	if createRuleResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci rule create status 201, got %d; body=%s", createRuleResp.StatusCode, string(createRuleResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(createRuleResp.RawBody),
		`<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlFilter">`,
		`<SqlExpression>color = &apos;blue&apos;</SqlExpression>`,
		`<Action xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlRuleAction">`,
		`<SqlExpression>SET priority = &apos;high&apos;</SqlExpression>`,
		`<CompatibilityLevel>20</CompatibilityLevel>`,
	)

	getRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodGet, ruleURL, nil))
	if err != nil {
		t.Fatalf("get floci rule with action returned error: %v", err)
	}
	if getRuleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected floci rule get status 200, got %d; body=%s", getRuleResp.StatusCode, string(getRuleResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(getRuleResp.RawBody),
		`<Action xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="SqlRuleAction">`,
		`<SqlExpression>SET priority = &apos;high&apos;</SqlExpression>`,
	)
}

func TestFlociStyleServiceBusRuleAdministrationPreservesCorrelationFilter(t *testing.T) {
	svc := New()

	namespaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ServiceBus/namespaces/sdk-ns?api-version=2024-01-01"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, namespaceURL, []byte(`{"location":"eastus","sku":{"name":"Standard","tier":"Standard"}}`))); err != nil {
		t.Fatalf("create namespace returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create namespace status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	baseURL := "http://localhost/devstoreaccount1-servicebus"
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk", []byte(`<entry><content><TopicDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci topic returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci topic create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, baseURL+"/topic-sdk/subscriptions/corr-sub", []byte(`<entry><content><SubscriptionDescription /></content></entry>`))); err != nil {
		t.Fatalf("create floci subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci subscription create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := baseURL + "/topic-sdk/subscriptions/corr-sub/rules/corr-rule"
	ruleXML := []byte(`<entry><content><RuleDescription>` +
		`<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="CorrelationFilter">` +
		`<CorrelationId>order-123</CorrelationId>` +
		`<ContentType>application/json</ContentType>` +
		`<Properties><KeyValueOfstringanyType><Key>tenant</Key><Value>a</Value></KeyValueOfstringanyType></Properties>` +
		`</Filter>` +
		`</RuleDescription></content></entry>`)
	createRuleResp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodPut, ruleURL, ruleXML))
	if err != nil {
		t.Fatalf("create floci correlation rule returned error: %v", err)
	}
	if createRuleResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected floci correlation rule create status 201, got %d; body=%s", createRuleResp.StatusCode, string(createRuleResp.RawBody))
	}
	assertServiceBusAtomContains(t, string(createRuleResp.RawBody),
		`<Filter xmlns:i="http://www.w3.org/2001/XMLSchema-instance" i:type="CorrelationFilter">`,
		`<CorrelationId>order-123</CorrelationId>`,
		`<ContentType>application/json</ContentType>`,
		`<Properties><KeyValueOfstringanyType><Key>tenant</Key><Value>a</Value></KeyValueOfstringanyType></Properties>`,
	)

	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "wrong correlation", map[string]string{
		"Content-Type":     "application/json",
		"Tenant":           "a",
		"BrokerProperties": `{"CorrelationId":"order-999"}`,
	})
	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "wrong tenant", map[string]string{
		"Content-Type":     "application/json",
		"Tenant":           "b",
		"BrokerProperties": `{"CorrelationId":"order-123"}`,
	})
	sendServiceBusMessageTo(t, svc, "sdk-ns", "topic-sdk", "matched", map[string]string{
		"Content-Type":     "application/json",
		"Tenant":           "a",
		"BrokerProperties": `{"CorrelationId":"order-123"}`,
	})

	assertServiceBusReceiveDeleteFrom(t, svc, "sdk-ns", "topic-sdk", "corr-sub", http.StatusOK, "matched")
	assertServiceBusReceiveDeleteFrom(t, svc, "sdk-ns", "topic-sdk", "corr-sub", http.StatusNoContent, "")
}

func assertServiceBusAtomContains(t *testing.T, body string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected Atom body to contain %q, got %s", fragment, body)
		}
	}
}

func serviceBusCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func sendServiceBusMessage(t *testing.T, svc *ServiceBusService, body string, headers map[string]string) {
	t.Helper()
	sendServiceBusMessageTo(t, svc, "ns-a", "topic-a", body, headers)
}

func sendServiceBusMessageTo(t *testing.T, svc *ServiceBusService, namespaceName, topicName, body string, headers map[string]string) {
	t.Helper()
	ctx := serviceBusCtx(t, http.MethodPost, "https://"+namespaceName+".servicebus.windows.net/"+topicName+"/messages", []byte(body))
	for key, value := range headers {
		ctx.RawRequest.Header.Set(key, value)
	}
	resp, err := svc.HandleRequest(ctx)
	if err != nil {
		t.Fatalf("send %q returned error: %v", body, err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send %q status 201, got %d; body=%s", body, resp.StatusCode, string(resp.RawBody))
	}
}

func assertServiceBusReceiveDelete(t *testing.T, svc *ServiceBusService, subscriptionName string, wantStatus int, wantBody string) {
	t.Helper()
	assertServiceBusReceiveDeleteFrom(t, svc, "ns-a", "topic-a", subscriptionName, wantStatus, wantBody)
}

func assertServiceBusReceiveDeleteFrom(t *testing.T, svc *ServiceBusService, namespaceName, topicName, subscriptionName string, wantStatus int, wantBody string) {
	t.Helper()
	rawURL := "https://" + namespaceName + ".servicebus.windows.net/" + topicName + "/subscriptions/" + subscriptionName + "/messages/head?timeout=60"
	resp, err := svc.HandleRequest(serviceBusCtx(t, http.MethodDelete, rawURL, nil))
	if err != nil {
		t.Fatalf("receive-delete %s returned error: %v", subscriptionName, err)
	}
	if resp.StatusCode != wantStatus || string(resp.RawBody) != wantBody {
		t.Fatalf("receive-delete %s: expected status=%d body=%q, got status=%d body=%q", subscriptionName, wantStatus, wantBody, resp.StatusCode, string(resp.RawBody))
	}
}

func decodeServiceBusResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
