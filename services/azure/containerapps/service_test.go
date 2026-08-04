package containerapps

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestManagedEnvironmentAndContainerAppLifecycle(t *testing.T) {
	svc := New()

	envURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a?api-version=2025-07-01"
	envPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"appLogsConfiguration":{"destination":"log-analytics"},
			"workloadProfiles":[{"name":"Consumption","workloadProfileType":"Consumption"}],
			"vnetConfiguration":{"internal":false}
		}
	}`)
	envCreate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, envURL, envPayload))
	if err != nil {
		t.Fatalf("create managed environment returned error: %v", err)
	}
	if envCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected managed environment create 201, got %d; body=%s", envCreate.StatusCode, string(envCreate.RawBody))
	}
	env := decodeContainerAppsResponse(t, envCreate)
	if env["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a" {
		t.Fatalf("unexpected managed environment id: %v", env["id"])
	}
	if env["type"] != "Microsoft.App/managedEnvironments" || env["location"] != "eastus" {
		t.Fatalf("unexpected managed environment identity fields: %v", env)
	}
	envProps := env["properties"].(map[string]any)
	if envProps["provisioningState"] != "Succeeded" || envProps["defaultDomain"] != "env-a.eastus.azurecontainerapps.io" {
		t.Fatalf("unexpected managed environment properties: %v", envProps)
	}

	appURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a?api-version=2025-07-01"
	appPayload := []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"test"},
		"properties":{
			"managedEnvironmentId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"configuration":{
				"activeRevisionsMode":"Single",
				"ingress":{"external":true,"targetPort":8080,"transport":"auto"},
				"secrets":[{"name":"api-key","value":"secret"}]
			},
			"template":{
				"containers":[{"name":"web","image":"ghcr.io/example/app:v1","resources":{"cpu":0.5,"memory":"1Gi"}}],
				"scale":{"minReplicas":1,"maxReplicas":3}
			}
		}
	}`)
	appCreate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, appURL, appPayload))
	if err != nil {
		t.Fatalf("create container app returned error: %v", err)
	}
	if appCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected container app create 201, got %d; body=%s", appCreate.StatusCode, string(appCreate.RawBody))
	}
	app := decodeContainerAppsResponse(t, appCreate)
	if app["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a" {
		t.Fatalf("unexpected container app id: %v", app["id"])
	}
	if app["type"] != "Microsoft.App/containerApps" || app["location"] != "eastus" {
		t.Fatalf("unexpected container app identity fields: %v", app)
	}
	appProps := app["properties"].(map[string]any)
	if appProps["provisioningState"] != "Succeeded" || appProps["runningStatus"] != "Running" {
		t.Fatalf("unexpected container app state: %v", appProps)
	}
	if appProps["latestRevisionName"] != "app-a--000001" || appProps["latestRevisionFqdn"] != "app-a--000001.env-a.eastus.azurecontainerapps.io" {
		t.Fatalf("unexpected revision projection: %v", appProps)
	}
	template := appProps["template"].(map[string]any)
	containers := template["containers"].([]any)
	if containers[0].(map[string]any)["image"] != "ghcr.io/example/app:v1" {
		t.Fatalf("expected container image to be preserved, got %v", containers)
	}
	configuration := appProps["configuration"].(map[string]any)
	if configuration["ingress"].(map[string]any)["targetPort"] != float64(8080) {
		t.Fatalf("expected ingress to be preserved, got %v", configuration)
	}

	appUpdate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, appURL, []byte(`{
		"location":"westus2",
		"tags":{"env":"prod"},
		"properties":{
			"managedEnvironmentId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"configuration":{"activeRevisionsMode":"Multiple"},
			"template":{"containers":[{"name":"api","image":"ghcr.io/example/app:v2"}]}
		}
	}`)))
	if err != nil {
		t.Fatalf("update container app returned error: %v", err)
	}
	if appUpdate.StatusCode != http.StatusOK {
		t.Fatalf("expected container app update 200, got %d; body=%s", appUpdate.StatusCode, string(appUpdate.RawBody))
	}

	listApps, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list container apps returned error: %v", err)
	}
	appList := decodeContainerAppsResponse(t, listApps)
	if values := appList["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "app-a" {
		t.Fatalf("expected one container app, got %v", appList)
	}

	listEnvs, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list managed environments returned error: %v", err)
	}
	envList := decodeContainerAppsResponse(t, listEnvs)
	if values := envList["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "env-a" {
		t.Fatalf("expected one managed environment, got %v", envList)
	}

	appDelete, err := svc.HandleRequest(containerAppsCtx(t, http.MethodDelete, appURL, nil))
	if err != nil {
		t.Fatalf("delete container app returned error: %v", err)
	}
	if appDelete.StatusCode != http.StatusAccepted {
		t.Fatalf("expected container app delete 202, got %d; body=%s", appDelete.StatusCode, string(appDelete.RawBody))
	}

	envDelete, err := svc.HandleRequest(containerAppsCtx(t, http.MethodDelete, envURL, nil))
	if err != nil {
		t.Fatalf("delete managed environment returned error: %v", err)
	}
	if envDelete.StatusCode != http.StatusAccepted {
		t.Fatalf("expected managed environment delete 202, got %d; body=%s", envDelete.StatusCode, string(envDelete.RawBody))
	}
}

func TestContainerAppsSubscriptionScopedLists(t *testing.T) {
	svc := New()

	createEnv := func(rawURL, location string) {
		t.Helper()
		resp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, rawURL, []byte(`{"location":"`+location+`","properties":{}}`)))
		if err != nil {
			t.Fatalf("create managed environment %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected managed environment create 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}
	createApp := func(rawURL, envID string) {
		t.Helper()
		resp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, rawURL, []byte(`{
			"location":"eastus",
			"properties":{
				"managedEnvironmentId":"`+envID+`",
				"template":{"containers":[{"name":"web","image":"nginx"}]}
			}
		}`)))
		if err != nil {
			t.Fatalf("create container app %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected container app create 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}

	envAURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a?api-version=2025-07-01"
	envBURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.App/managedEnvironments/env-b?api-version=2025-07-01"
	envOtherURL := "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.App/managedEnvironments/env-z?api-version=2025-07-01"
	createEnv(envBURL, "westus2")
	createEnv(envOtherURL, "eastus")
	createEnv(envAURL, "eastus")

	createApp("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-a?api-version=2025-07-01", "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a")
	createApp("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.App/containerApps/app-b?api-version=2025-07-01", "/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.App/managedEnvironments/env-b")
	createApp("https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.App/containerApps/app-z?api-version=2025-07-01", "/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.App/managedEnvironments/env-z")

	appsResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.App/containerApps?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list subscription container apps returned error: %v", err)
	}
	if appsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription app list status 200, got %d; body=%s", appsResp.StatusCode, string(appsResp.RawBody))
	}
	apps := decodeContainerAppsResponse(t, appsResp)
	appValues := apps["value"].([]any)
	if len(appValues) != 2 || appValues[0].(map[string]any)["name"] != "app-a" || appValues[1].(map[string]any)["name"] != "app-b" {
		t.Fatalf("expected sub-1 container apps sorted by name, got %v", apps)
	}

	envsResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.App/managedEnvironments?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list subscription managed environments returned error: %v", err)
	}
	if envsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription environment list status 200, got %d; body=%s", envsResp.StatusCode, string(envsResp.RawBody))
	}
	envs := decodeContainerAppsResponse(t, envsResp)
	envValues := envs["value"].([]any)
	if len(envValues) != 2 || envValues[0].(map[string]any)["name"] != "env-a" || envValues[1].(map[string]any)["name"] != "env-b" {
		t.Fatalf("expected sub-1 environments sorted by name, got %v", envs)
	}
}

func TestContainerAppListSecrets(t *testing.T) {
	svc := New()

	appURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-secrets?api-version=2025-07-01"
	createResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, appURL, []byte(`{
		"location":"eastus",
		"properties":{
			"managedEnvironmentId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"configuration":{
				"secrets":[
					{"name":"api-key","value":"secret-value"},
					{"name":"registry-password","value":"p@ssw0rd"}
				]
			},
			"template":{"containers":[{"name":"web","image":"nginx"}]}
		}
	}`)))
	if err != nil {
		t.Fatalf("create container app returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	secretsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-secrets/listSecrets?api-version=2025-07-01"
	secretsResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, secretsURL, nil))
	if err != nil {
		t.Fatalf("list secrets returned error: %v", err)
	}
	if secretsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list secrets status 200, got %d; body=%s", secretsResp.StatusCode, string(secretsResp.RawBody))
	}
	secrets := decodeContainerAppsResponse(t, secretsResp)
	values := secrets["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two secrets, got %v", secrets)
	}
	first := values[0].(map[string]any)
	second := values[1].(map[string]any)
	if first["name"] != "api-key" || first["value"] != "secret-value" || second["name"] != "registry-password" || second["value"] != "p@ssw0rd" {
		t.Fatalf("unexpected secret values: %v", values)
	}

	missingResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/missing/listSecrets?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list missing app secrets returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing app status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestContainerAppRevisionsListAndGet(t *testing.T) {
	svc := New()

	appURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-rev?api-version=2025-07-01"
	createResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, appURL, []byte(`{
		"location":"eastus",
		"properties":{
			"managedEnvironmentId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"template":{
				"containers":[{"name":"web","image":"nginx:1.27"}],
				"scale":{"minReplicas":1,"maxReplicas":2}
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create container app returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-rev/revisions?api-version=2025-07-01"
	listResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list revisions returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list revisions status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	revisions := decodeContainerAppsResponse(t, listResp)
	values := revisions["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one revision, got %v", revisions)
	}
	revision := values[0].(map[string]any)
	if revision["name"] != "app-rev--000001" || revision["type"] != "Microsoft.App/containerApps/revisions" {
		t.Fatalf("unexpected revision identity: %v", revision)
	}
	revisionProps := revision["properties"].(map[string]any)
	if revisionProps["active"] != true || revisionProps["runningState"] != "Running" || revisionProps["provisioningState"] != "Provisioned" {
		t.Fatalf("unexpected revision properties: %v", revisionProps)
	}
	revisionTemplate := revisionProps["template"].(map[string]any)
	revisionContainers := revisionTemplate["containers"].([]any)
	if revisionContainers[0].(map[string]any)["image"] != "nginx:1.27" {
		t.Fatalf("expected revision template to preserve app container, got %v", revisionTemplate)
	}

	getURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-rev/revisions/app-rev--000001?api-version=2025-07-01"
	getResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, getURL, nil))
	if err != nil {
		t.Fatalf("get revision returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get revision status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	gotRevision := decodeContainerAppsResponse(t, getResp)
	if gotRevision["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-rev/revisions/app-rev--000001" {
		t.Fatalf("unexpected revision id: %v", gotRevision["id"])
	}

	missingRevisionResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-rev/revisions/missing?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get missing revision returned error: %v", err)
	}
	if missingRevisionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing revision status 404, got %d; body=%s", missingRevisionResp.StatusCode, string(missingRevisionResp.RawBody))
	}

	missingAppResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/missing/revisions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list missing app revisions returned error: %v", err)
	}
	if missingAppResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing app status 404, got %d; body=%s", missingAppResp.StatusCode, string(missingAppResp.RawBody))
	}
}

func TestContainerAppRevisionActions(t *testing.T) {
	svc := New()

	appURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-actions?api-version=2025-07-01"
	createResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, appURL, []byte(`{
		"location":"eastus",
		"properties":{
			"managedEnvironmentId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"template":{"containers":[{"name":"web","image":"nginx:1.27"}]}
		}
	}`)))
	if err != nil {
		t.Fatalf("create container app returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	revisionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-actions/revisions/app-actions--000001?api-version=2025-07-01"
	revisionActionURL := func(action string) string {
		return "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-actions/revisions/app-actions--000001/" + action + "?api-version=2025-07-01"
	}

	deactivateResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, revisionActionURL("deactivate"), nil))
	if err != nil {
		t.Fatalf("deactivate revision returned error: %v", err)
	}
	if deactivateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected deactivate revision status 200, got %d; body=%s", deactivateResp.StatusCode, string(deactivateResp.RawBody))
	}
	afterDeactivate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, revisionURL, nil))
	if err != nil {
		t.Fatalf("get revision after deactivate returned error: %v", err)
	}
	deactivatedRevision := decodeContainerAppsResponse(t, afterDeactivate)
	deactivatedProps := deactivatedRevision["properties"].(map[string]any)
	if deactivatedProps["active"] != false || deactivatedProps["runningState"] != "Stopped" {
		t.Fatalf("expected deactivated revision state, got %v", deactivatedProps)
	}

	activateResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, revisionActionURL("activate"), nil))
	if err != nil {
		t.Fatalf("activate revision returned error: %v", err)
	}
	if activateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected activate revision status 200, got %d; body=%s", activateResp.StatusCode, string(activateResp.RawBody))
	}
	afterActivate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodGet, revisionURL, nil))
	if err != nil {
		t.Fatalf("get revision after activate returned error: %v", err)
	}
	activatedRevision := decodeContainerAppsResponse(t, afterActivate)
	activatedProps := activatedRevision["properties"].(map[string]any)
	if activatedProps["active"] != true || activatedProps["runningState"] != "Running" {
		t.Fatalf("expected activated revision state, got %v", activatedProps)
	}

	restartResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, revisionActionURL("restart"), nil))
	if err != nil {
		t.Fatalf("restart revision returned error: %v", err)
	}
	if restartResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restart revision status 200, got %d; body=%s", restartResp.StatusCode, string(restartResp.RawBody))
	}
	restartedRevision := decodeContainerAppsResponse(t, restartResp)
	restartedProps := restartedRevision["properties"].(map[string]any)
	if restartedProps["active"] != true || restartedProps["runningState"] != "Running" {
		t.Fatalf("expected restarted revision response state, got %v", restartedProps)
	}

	missingRevisionResp, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-actions/revisions/missing/restart?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("restart missing revision returned error: %v", err)
	}
	if missingRevisionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing revision status 404, got %d; body=%s", missingRevisionResp.StatusCode, string(missingRevisionResp.RawBody))
	}
}

func TestContainerAppsValidationTemplateProvisioningAndKeys(t *testing.T) {
	svc := New()

	badAppURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/bad?api-version=2025-07-01"
	missingTemplate, err := svc.HandleRequest(containerAppsCtx(t, http.MethodPut, badAppURL, []byte(`{"location":"eastus","properties":{"managedEnvironmentId":"env"}}`)))
	if err != nil {
		t.Fatalf("missing template request returned error: %v", err)
	}
	if missingTemplate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing template status 400, got %d; body=%s", missingTemplate.StatusCode, string(missingTemplate.RawBody))
	}

	result, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.App/containerApps",
		"name":     "app-template",
		"location": "eastus",
		"properties": map[string]any{
			"managedEnvironmentId": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/managedEnvironments/env-a",
			"template": map[string]any{
				"containers": []any{map[string]any{"name": "web", "image": "nginx"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("provision container app returned error: %v", err)
	}
	app := result.(map[string]any)
	if app["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.App/containerApps/app-template" {
		t.Fatalf("unexpected provisioned app id: %v", app["id"])
	}

	keys := svc.ServiceKeys()
	for _, expected := range []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.App/containerApps", APIVersion: "2025-07-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.App/managedEnvironments", APIVersion: "2025-07-01"},
	} {
		found := false
		for _, got := range keys {
			if got == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected service key %#v in %#v", expected, keys)
		}
	}
}

func containerAppsCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{
		Method: method,
		URL:    parsed,
		Host:   parsed.Host,
		Header: make(http.Header),
	}
	req.Header.Set("Authorization", "Bearer token")
	target := routing.DetectTarget(req)
	return &service.RequestContext{
		Service:    target.Service,
		Action:     target.Action,
		RawRequest: req,
		Body:       body,
	}
}

func decodeContainerAppsResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
