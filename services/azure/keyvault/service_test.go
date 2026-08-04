package keyvault_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/azure/keyvault"
)

func keyVaultCtx(t *testing.T, method, targetURL string, body []byte) *service.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, targetURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	return &service.RequestContext{
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       body,
	}
}

func decodeKeyVaultJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	if resp.RawContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", resp.RawContentType)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	return out
}

func keyVaultVersionFromID(t *testing.T, id string) string {
	t.Helper()
	version := id[strings.LastIndex(id, "/")+1:]
	if version == "" || strings.Contains(version, "secrets") {
		t.Fatalf("could not extract version from id %q", id)
	}
	return version
}

func isLowerHexVersion(version string) bool {
	if len(version) != 32 {
		return false
	}
	for _, r := range version {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func TestVaultLifecycle(t *testing.T) {
	svc := keyvault.New()
	payload := []byte(`{"location":"westus2","tags":{"env":"test"},"properties":{"tenantId":"tenant-1","sku":{"family":"A","name":"standard"},"accessPolicies":[],"enabledForTemplateDeployment":true}}`)

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults/vaulta?api-version=2024-11-01", payload))
	if err != nil {
		t.Fatalf("create vault returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createResp.StatusCode)
	}
	created := decodeKeyVaultJSON(t, createResp)
	if created["name"] != "vaulta" {
		t.Fatalf("unexpected vault name: %v", created["name"])
	}
	props := created["properties"].(map[string]any)
	if props["vaultUri"] != "https://vaulta.vault.azure.net/" {
		t.Fatalf("unexpected vault URI: %v", props["vaultUri"])
	}
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected provisioning state: %v", props["provisioningState"])
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults/vaulta?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("get vault returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("list vaults returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listResp)
	if got := len(listed["value"].([]any)); got != 1 {
		t.Fatalf("expected one vault, got %d", got)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults/vaulta?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("delete vault returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete status 202, got %d", deleteResp.StatusCode)
	}
}

func TestSecretLifecycle(t *testing.T) {
	svc := keyvault.New()

	setResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/db-password?api-version=2025-07-01", []byte(`{"value":"s3cr3t","contentType":"text/plain","attributes":{"enabled":true},"tags":{"app":"api"}}`)))
	if err != nil {
		t.Fatalf("set secret returned error: %v", err)
	}
	if setResp.StatusCode != http.StatusOK {
		t.Fatalf("expected set status 200, got %d", setResp.StatusCode)
	}
	secret := decodeKeyVaultJSON(t, setResp)
	if secret["value"] != "s3cr3t" {
		t.Fatalf("unexpected secret value: %v", secret["value"])
	}
	if secret["id"] == "" {
		t.Fatal("expected secret id")
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/db-password?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get secret returned error: %v", err)
	}
	gotSecret := decodeKeyVaultJSON(t, getResp)
	if gotSecret["value"] != "s3cr3t" {
		t.Fatalf("unexpected fetched secret value: %v", gotSecret["value"])
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list secrets returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listResp)
	if _, ok := listed["nextLink"]; !ok {
		t.Fatalf("expected secret list envelope to include nextLink, got %v", listed)
	}
	if got := len(listed["value"].([]any)); got != 1 {
		t.Fatalf("expected one secret, got %d", got)
	}
	first := listed["value"].([]any)[0].(map[string]any)
	if _, includesValue := first["value"]; includesValue {
		t.Fatal("secret list items must not include secret values")
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/db-password?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete secret returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d", deleteResp.StatusCode)
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["recoveryId"] != "https://vaulta.vault.azure.net/deletedsecrets/db-password" {
		t.Fatalf("unexpected recovery id: %v", deleted["recoveryId"])
	}
}

func TestSecretVersionsPropertiesDisableAndBackup(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/app-secret?api-version=2025-07-01", []byte(`{"value":"first","contentType":"text/plain","attributes":{"enabled":true},"tags":{"stage":"one"}}`)))
	if err != nil {
		t.Fatalf("set first secret returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first set status 200, got %d; body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	first := decodeKeyVaultJSON(t, firstResp)
	firstVersion := keyVaultVersionFromID(t, first["id"].(string))
	if !isLowerHexVersion(firstVersion) {
		t.Fatalf("expected 32-character lowercase hex version, got %q", firstVersion)
	}

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/app-secret?api-version=2025-07-01", []byte(`{"value":"second","contentType":"text/plain","attributes":{"enabled":true},"tags":{"stage":"two"}}`)))
	if err != nil {
		t.Fatalf("set second secret returned error: %v", err)
	}
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second set status 200, got %d; body=%s", secondResp.StatusCode, string(secondResp.RawBody))
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["id"].(string))
	if firstVersion == secondVersion {
		t.Fatalf("expected distinct secret versions, got %q", firstVersion)
	}

	latestResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/app-secret?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get latest returned error: %v", err)
	}
	latest := decodeKeyVaultJSON(t, latestResp)
	if latest["value"] != "second" || !strings.HasSuffix(latest["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected latest secret to use second version, got %v", latest)
	}

	firstVersionResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/app-secret/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first version returned error: %v", err)
	}
	firstVersionSecret := decodeKeyVaultJSON(t, firstVersionResp)
	if firstVersionSecret["value"] != "first" || !strings.HasSuffix(firstVersionSecret["id"].(string), "/"+firstVersion) {
		t.Fatalf("expected first version value and id, got %v", firstVersionSecret)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/app-secret/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list secret versions returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list versions status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeKeyVaultJSON(t, listResp)["value"].([]any)
	versionEnvelope := decodeKeyVaultJSON(t, listResp)
	if _, ok := versionEnvelope["nextLink"]; !ok {
		t.Fatalf("expected secret version list envelope to include nextLink, got %v", versionEnvelope)
	}
	versions := map[string]bool{}
	for _, raw := range listed {
		item := raw.(map[string]any)
		if _, hasValue := item["value"]; hasValue {
			t.Fatalf("secret version list item must not include values: %v", item)
		}
		versions[keyVaultVersionFromID(t, item["id"].(string))] = true
	}
	if !versions[firstVersion] || !versions[secondVersion] || len(versions) != 2 {
		t.Fatalf("expected both versions in list, got %v", versions)
	}

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/secrets/app-secret/"+firstVersion+"?api-version=2025-07-01", []byte(`{"contentType":"application/json","tags":{"updated":"yes"},"attributes":{"enabled":true}}`)))
	if err != nil {
		t.Fatalf("update first version returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updatedFirst := decodeKeyVaultJSON(t, updateResp)
	if updatedFirst["value"] != "first" || updatedFirst["contentType"] != "application/json" {
		t.Fatalf("expected update to preserve value and change content type, got %v", updatedFirst)
	}
	if updatedFirst["tags"].(map[string]any)["updated"] != "yes" {
		t.Fatalf("expected updated tags, got %v", updatedFirst["tags"])
	}

	disableResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/secrets/app-secret/"+secondVersion+"?api-version=2025-07-01", []byte(`{"attributes":{"enabled":false}}`)))
	if err != nil {
		t.Fatalf("disable latest version returned error: %v", err)
	}
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d; body=%s", disableResp.StatusCode, string(disableResp.RawBody))
	}
	disabledLatest, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/app-secret?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get disabled latest returned error: %v", err)
	}
	if disabledLatest.StatusCode != http.StatusForbidden {
		t.Fatalf("expected disabled secret status 403, got %d; body=%s", disabledLatest.StatusCode, string(disabledLatest.RawBody))
	}

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/secrets/app-secret/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup secret returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backup := decodeKeyVaultJSON(t, backupResp)
	if backup["value"] == "" {
		t.Fatalf("expected non-empty backup value, got %v", backup)
	}
}

func TestSecretSetAndUpdatePreservesNotBeforeAndExpiresAttributes(t *testing.T) {
	svc := keyvault.New()

	setResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/expiring-secret?api-version=2025-07-01", []byte(`{"value":"timed","attributes":{"enabled":true,"nbf":631180800,"exp":662716800},"tags":{"stage":"set"}}`)))
	if err != nil {
		t.Fatalf("set expiring secret returned error: %v", err)
	}
	if setResp.StatusCode != http.StatusOK {
		t.Fatalf("expected set status 200, got %d; body=%s", setResp.StatusCode, string(setResp.RawBody))
	}
	created := decodeKeyVaultJSON(t, setResp)
	version := keyVaultVersionFromID(t, created["id"].(string))
	createdAttrs := created["attributes"].(map[string]any)
	if createdAttrs["nbf"] != float64(631180800) || createdAttrs["exp"] != float64(662716800) {
		t.Fatalf("expected set to preserve nbf and exp attributes, got %v", createdAttrs)
	}

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/secrets/expiring-secret/"+version+"?api-version=2025-07-01", []byte(`{"attributes":{"exp":694252800},"tags":{"stage":"updated"}}`)))
	if err != nil {
		t.Fatalf("update expiring secret returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeKeyVaultJSON(t, updateResp)
	updatedAttrs := updated["attributes"].(map[string]any)
	if updatedAttrs["nbf"] != float64(631180800) || updatedAttrs["exp"] != float64(694252800) {
		t.Fatalf("expected update to change exp and preserve nbf, got %v", updatedAttrs)
	}
	if updated["value"] != "timed" || updated["tags"].(map[string]any)["stage"] != "updated" {
		t.Fatalf("expected update to preserve value and replace tags, got %v", updated)
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/expiring-secret/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get updated expiring secret returned error: %v", err)
	}
	gotAttrs := decodeKeyVaultJSON(t, getResp)["attributes"].(map[string]any)
	if gotAttrs["nbf"] != float64(631180800) || gotAttrs["exp"] != float64(694252800) {
		t.Fatalf("expected persisted nbf and exp attributes, got %v", gotAttrs)
	}
}

func TestSecretListPaginationUsesMaxResultsAndNextLink(t *testing.T) {
	svc := keyvault.New()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/"+name+"?api-version=2025-07-01", []byte(`{"value":"`+name+`"}`)))
		if err != nil {
			t.Fatalf("set secret %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected set secret %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}

	firstPageResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets?api-version=2025-07-01&maxresults=1", nil))
	if err != nil {
		t.Fatalf("list first secret page returned error: %v", err)
	}
	firstPage := decodeKeyVaultJSON(t, firstPageResp)
	firstValues := firstPage["value"].([]any)
	if len(firstValues) != 1 {
		t.Fatalf("expected one secret in first page, got %v", firstValues)
	}
	firstID := firstValues[0].(map[string]any)["id"].(string)
	secretNextLink, ok := firstPage["nextLink"].(string)
	if !ok || secretNextLink == "" {
		t.Fatalf("expected secret first page to include non-empty nextLink, got %v", firstPage)
	}
	nextPageResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, secretNextLink, nil))
	if err != nil {
		t.Fatalf("list next secret page returned error: %v", err)
	}
	nextValues := decodeKeyVaultJSON(t, nextPageResp)["value"].([]any)
	if len(nextValues) != 1 || nextValues[0].(map[string]any)["id"].(string) == firstID {
		t.Fatalf("expected next secret page to advance, first=%v next=%v", firstValues, nextValues)
	}

	for _, value := range []string{"v1", "v2", "v3"} {
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/rotating?api-version=2025-07-01", []byte(`{"value":"`+value+`"}`)))
		if err != nil {
			t.Fatalf("set rotating secret version %s returned error: %v", value, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected set rotating secret status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
		}
	}
	firstVersionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/rotating/versions?api-version=2025-07-01&maxresults=1", nil))
	if err != nil {
		t.Fatalf("list first secret-version page returned error: %v", err)
	}
	firstVersionsPage := decodeKeyVaultJSON(t, firstVersionsResp)
	firstVersions := firstVersionsPage["value"].([]any)
	if len(firstVersions) != 1 {
		t.Fatalf("expected one secret version in first page, got %v", firstVersions)
	}
	firstVersionID := firstVersions[0].(map[string]any)["id"].(string)
	versionNextLink, ok := firstVersionsPage["nextLink"].(string)
	if !ok || versionNextLink == "" {
		t.Fatalf("expected secret-version first page to include non-empty nextLink, got %v", firstVersionsPage)
	}
	nextVersionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, versionNextLink, nil))
	if err != nil {
		t.Fatalf("list next secret-version page returned error: %v", err)
	}
	nextVersions := decodeKeyVaultJSON(t, nextVersionsResp)["value"].([]any)
	if len(nextVersions) != 1 || nextVersions[0].(map[string]any)["id"].(string) == firstVersionID {
		t.Fatalf("expected next secret-version page to advance, first=%v next=%v", firstVersions, nextVersions)
	}

	for _, name := range []string{"gone-alpha", "gone-beta"} {
		_, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/"+name+"?api-version=2025-07-01", []byte(`{"value":"`+name+`"}`)))
		if err != nil {
			t.Fatalf("set deleted-list secret %s returned error: %v", name, err)
		}
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/"+name+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("delete secret %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected delete secret %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}
	firstDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets?api-version=2025-07-01&maxresults=1", nil))
	if err != nil {
		t.Fatalf("list first deleted-secret page returned error: %v", err)
	}
	firstDeletedPage := decodeKeyVaultJSON(t, firstDeletedResp)
	firstDeleted := firstDeletedPage["value"].([]any)
	if len(firstDeleted) != 1 {
		t.Fatalf("expected one deleted secret in first page, got %v", firstDeleted)
	}
	firstRecoveryID := firstDeleted[0].(map[string]any)["recoveryId"].(string)
	deletedNextLink, ok := firstDeletedPage["nextLink"].(string)
	if !ok || deletedNextLink == "" {
		t.Fatalf("expected deleted-secret first page to include non-empty nextLink, got %v", firstDeletedPage)
	}
	nextDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, deletedNextLink, nil))
	if err != nil {
		t.Fatalf("list next deleted-secret page returned error: %v", err)
	}
	nextDeleted := decodeKeyVaultJSON(t, nextDeletedResp)["value"].([]any)
	if len(nextDeleted) != 1 || nextDeleted[0].(map[string]any)["recoveryId"].(string) == firstRecoveryID {
		t.Fatalf("expected next deleted-secret page to advance, first=%v next=%v", firstDeleted, nextDeleted)
	}

	invalidResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets?api-version=2025-07-01&maxresults=26", nil))
	if err != nil {
		t.Fatalf("invalid maxresults request returned error: %v", err)
	}
	if invalidResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid maxresults status 400, got %d; body=%s", invalidResp.StatusCode, string(invalidResp.RawBody))
	}
}

func TestSecretBackupAndRestorePreservesVersions(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/restore-me?api-version=2025-07-01", []byte(`{"value":"first","contentType":"text/plain","attributes":{"enabled":true},"tags":{"stage":"first"}}`)))
	if err != nil {
		t.Fatalf("set first secret version returned error: %v", err)
	}
	first := decodeKeyVaultJSON(t, firstResp)
	firstVersion := keyVaultVersionFromID(t, first["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/restore-me?api-version=2025-07-01", []byte(`{"value":"second","contentType":"application/json","attributes":{"enabled":true},"tags":{"stage":"second"}}`)))
	if err != nil {
		t.Fatalf("set second secret version returned error: %v", err)
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["id"].(string))
	if firstVersion == secondVersion {
		t.Fatalf("expected distinct secret versions, got %q", firstVersion)
	}

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/secrets/restore-me/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup secret returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup secret status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backupValue := decodeKeyVaultJSON(t, backupResp)["value"].(string)
	if backupValue == "" || strings.Contains(backupValue, "restore-me") {
		t.Fatalf("expected opaque secret backup value, got %q", backupValue)
	}
	if _, err := base64.RawURLEncoding.DecodeString(backupValue); err != nil {
		t.Fatalf("expected secret backup value to be base64url encoded: %v", err)
	}

	restoreSvc := keyvault.New()
	restoreResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/secrets/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore secret returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restore secret status 200, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}
	restored := decodeKeyVaultJSON(t, restoreResp)
	if restored["id"] != "https://vaultb.vault.azure.net/secrets/restore-me/"+secondVersion || restored["value"] != "second" || restored["contentType"] != "application/json" {
		t.Fatalf("expected restored latest secret with target vault id, got %v", restored)
	}
	if restored["tags"].(map[string]any)["stage"] != "second" {
		t.Fatalf("expected restored latest tags, got %v", restored["tags"])
	}

	firstVersionResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/secrets/restore-me/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored first secret version returned error: %v", err)
	}
	firstRestored := decodeKeyVaultJSON(t, firstVersionResp)
	if firstRestored["id"] != "https://vaultb.vault.azure.net/secrets/restore-me/"+firstVersion || firstRestored["value"] != "first" || firstRestored["contentType"] != "text/plain" {
		t.Fatalf("expected restored first secret version, got %v", firstRestored)
	}
	if firstRestored["tags"].(map[string]any)["stage"] != "first" {
		t.Fatalf("expected restored first-version tags, got %v", firstRestored["tags"])
	}

	versionsResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/secrets/restore-me/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list restored secret versions returned error: %v", err)
	}
	versions := map[string]bool{}
	for _, raw := range decodeKeyVaultJSON(t, versionsResp)["value"].([]any) {
		versions[keyVaultVersionFromID(t, raw.(map[string]any)["id"].(string))] = true
	}
	if !versions[firstVersion] || !versions[secondVersion] || len(versions) != 2 {
		t.Fatalf("expected both restored secret versions, got %v", versions)
	}

	sourceVaultResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/restore-me?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get source vault secret from restore service returned error: %v", err)
	}
	if sourceVaultResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected restore to target vault only, got %d; body=%s", sourceVaultResp.StatusCode, string(sourceVaultResp.RawBody))
	}
}

func TestCertificateBackedSecretRestoreRewritesBackingKeyID(t *testing.T) {
	svc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("secret-restore-certificate"))

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/secret-backed/import?api-version=2025-07-01", []byte(`{"value":"`+certificateValue+`","policy":{"secret_props":{"contentType":"application/x-pkcs12"}}}`)))
	if err != nil {
		t.Fatalf("import certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	version := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, importResp)["id"].(string))

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/secrets/secret-backed/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup certificate-backed secret returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backupValue := decodeKeyVaultJSON(t, backupResp)["value"].(string)

	restoreSvc := keyvault.New()
	restoreResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/secrets/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore certificate-backed secret returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}
	restored := decodeKeyVaultJSON(t, restoreResp)
	if restored["id"] != "https://vaultb.vault.azure.net/secrets/secret-backed/"+version || restored["kid"] != "https://vaultb.vault.azure.net/keys/secret-backed/"+version || restored["managed"] != true {
		t.Fatalf("expected restored certificate-backed secret identifiers to use target vault, got %v", restored)
	}
}

func TestCertificateBackedDeletedSecretProjectsManagedKeyMetadata(t *testing.T) {
	svc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("deleted-secret-certificate"))

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/deleted-backed/import?api-version=2025-07-01", []byte(`{"value":"`+certificateValue+`","policy":{"secret_props":{"contentType":"application/x-pem-file"}}}`)))
	if err != nil {
		t.Fatalf("import certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	version := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, importResp)["id"].(string))
	wantKID := "https://vaulta.vault.azure.net/keys/deleted-backed/" + version

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/deleted-backed?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete certificate-backed secret returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["kid"] != wantKID || deleted["managed"] != true {
		t.Fatalf("expected deleted certificate-backed secret to include kid and managed metadata, got %v", deleted)
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets/deleted-backed?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted certificate-backed secret returned error: %v", err)
	}
	got := decodeKeyVaultJSON(t, getResp)
	if got["kid"] != wantKID || got["managed"] != true {
		t.Fatalf("expected get deleted secret to include kid and managed metadata, got %v", got)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list deleted secrets returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listResp)["value"].([]any)
	if len(listed) != 1 {
		t.Fatalf("expected one deleted secret, got %v", listed)
	}
	listItem := listed[0].(map[string]any)
	if listItem["kid"] != wantKID || listItem["managed"] != true {
		t.Fatalf("expected list deleted secret to include kid and managed metadata, got %v", listItem)
	}
}

func TestCertificateBackedSecretRecordsPreviousCertificateVersion(t *testing.T) {
	svc := keyvault.New()
	firstValue := base64.StdEncoding.EncodeToString([]byte("previous-version-certificate-one"))
	secondValue := base64.StdEncoding.EncodeToString([]byte("previous-version-certificate-two"))

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/rotating-backed/import?api-version=2025-07-01", []byte(`{"value":"`+firstValue+`","policy":{"secret_props":{"contentType":"application/x-pkcs12"}}}`)))
	if err != nil {
		t.Fatalf("import first certificate returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first import status 200, got %d; body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/rotating-backed/import?api-version=2025-07-01", []byte(`{"value":"`+secondValue+`","policy":{"secret_props":{"contentType":"application/x-pkcs12"}}}`)))
	if err != nil {
		t.Fatalf("import second certificate returned error: %v", err)
	}
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second import status 200, got %d; body=%s", secondResp.StatusCode, string(secondResp.RawBody))
	}
	secondVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, secondResp)["id"].(string))
	if firstVersion == secondVersion {
		t.Fatalf("expected distinct certificate versions, got %q", firstVersion)
	}

	latestSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/rotating-backed?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get latest backing secret returned error: %v", err)
	}
	latestSecret := decodeKeyVaultJSON(t, latestSecretResp)
	if latestSecret["id"] != "https://vaulta.vault.azure.net/secrets/rotating-backed/"+secondVersion || latestSecret["previousVersion"] != firstVersion {
		t.Fatalf("expected latest backing secret to reference previous certificate version, got %v", latestSecret)
	}

	firstSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/rotating-backed/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first backing secret version returned error: %v", err)
	}
	if firstSecret := decodeKeyVaultJSON(t, firstSecretResp); firstSecret["previousVersion"] != nil {
		t.Fatalf("expected first backing secret version to omit previousVersion, got %v", firstSecret)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/rotating-backed?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete backing secret returned error: %v", err)
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["previousVersion"] != firstVersion {
		t.Fatalf("expected deleted backing secret to retain previousVersion, got %v", deleted)
	}
}

func TestSecretSoftDeleteRecoverAndPurge(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/recoverable?api-version=2025-07-01", []byte(`{"value":"v1","contentType":"text/plain"}`)))
	if err != nil {
		t.Fatalf("set first recoverable secret returned error: %v", err)
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPut, "https://vaulta.vault.azure.net/secrets/recoverable?api-version=2025-07-01", []byte(`{"value":"v2","contentType":"text/plain"}`)))
	if err != nil {
		t.Fatalf("set second recoverable secret returned error: %v", err)
	}
	secondVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, secondResp)["id"].(string))

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete recoverable secret returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["recoveryId"] != "https://vaulta.vault.azure.net/deletedsecrets/recoverable" || deleted["deletedDate"] == nil || deleted["scheduledPurgeDate"] == nil {
		t.Fatalf("unexpected deleted secret bundle: %v", deleted)
	}
	if !strings.HasSuffix(deleted["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected deleted bundle to reference latest version %q, got %v", secondVersion, deleted["id"])
	}

	missingActive, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted active secret returned error: %v", err)
	}
	if missingActive.StatusCode != http.StatusNotFound {
		t.Fatalf("expected active secret to be missing after delete, got %d; body=%s", missingActive.StatusCode, string(missingActive.RawBody))
	}

	getDeleted, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted secret returned error: %v", err)
	}
	if getDeleted.StatusCode != http.StatusOK {
		t.Fatalf("expected get deleted status 200, got %d; body=%s", getDeleted.StatusCode, string(getDeleted.RawBody))
	}
	if got := decodeKeyVaultJSON(t, getDeleted); got["recoveryId"] != "https://vaulta.vault.azure.net/deletedsecrets/recoverable" {
		t.Fatalf("unexpected get deleted response: %v", got)
	}

	listDeleted, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list deleted secrets returned error: %v", err)
	}
	deletedEnvelope := decodeKeyVaultJSON(t, listDeleted)
	if _, ok := deletedEnvelope["nextLink"]; !ok {
		t.Fatalf("expected deleted secret list envelope to include nextLink, got %v", deletedEnvelope)
	}
	listed := deletedEnvelope["value"].([]any)
	if len(listed) != 1 || listed[0].(map[string]any)["recoveryId"] != "https://vaulta.vault.azure.net/deletedsecrets/recoverable" {
		t.Fatalf("unexpected deleted secret list: %v", listed)
	}

	recoverResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/deletedsecrets/recoverable/recover?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("recover deleted secret returned error: %v", err)
	}
	if recoverResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recover status 200, got %d; body=%s", recoverResp.StatusCode, string(recoverResp.RawBody))
	}
	recovered := decodeKeyVaultJSON(t, recoverResp)
	if recovered["value"] != "v2" || !strings.HasSuffix(recovered["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected recover to restore latest version, got %v", recovered)
	}

	firstVersionAfterRecover, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/recoverable/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first version after recover returned error: %v", err)
	}
	if got := decodeKeyVaultJSON(t, firstVersionAfterRecover); got["value"] != "v1" {
		t.Fatalf("expected recover to preserve older versions, got %v", got)
	}

	_, err = svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/secrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete recovered secret returned error: %v", err)
	}
	purgeResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/deletedsecrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("purge deleted secret returned error: %v", err)
	}
	if purgeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected purge status 204, got %d; body=%s", purgeResp.StatusCode, string(purgeResp.RawBody))
	}
	getPurged, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedsecrets/recoverable?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get purged deleted secret returned error: %v", err)
	}
	if getPurged.StatusCode != http.StatusNotFound {
		t.Fatalf("expected purged deleted secret status 404, got %d; body=%s", getPurged.StatusCode, string(getPurged.RawBody))
	}
}

func TestKeyCreateEncryptDecryptRoundTrip(t *testing.T) {
	svc := keyvault.New()
	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/signing-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_size":2048,"key_ops":["encrypt","decrypt"],"tags":{"purpose":"unit test"}}`)))
	if err != nil {
		t.Fatalf("create key returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create key status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeKeyVaultJSON(t, createResp)
	key := created["key"].(map[string]any)
	kid := key["kid"].(string)
	if kid == "" {
		t.Fatal("expected key identifier")
	}
	if key["kty"] != "RSA" {
		t.Fatalf("unexpected key type: %v", key["kty"])
	}
	keyOps := key["key_ops"].([]any)
	if len(keyOps) != 2 || keyOps[0] != "encrypt" || keyOps[1] != "decrypt" {
		t.Fatalf("unexpected key operations: %v", keyOps)
	}
	tags := created["tags"].(map[string]any)
	if tags["purpose"] != "unit test" {
		t.Fatalf("unexpected key tags: %v", tags)
	}

	encryptResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/encrypt?api-version=2025-07-01", []byte(`{"alg":"RSA-OAEP","value":"cGxhaW50ZXh0"}`)))
	if err != nil {
		t.Fatalf("encrypt returned error: %v", err)
	}
	if encryptResp.StatusCode != http.StatusOK {
		t.Fatalf("expected encrypt status 200, got %d; body=%s", encryptResp.StatusCode, string(encryptResp.RawBody))
	}
	encrypted := decodeKeyVaultJSON(t, encryptResp)
	if encrypted["kid"] != kid {
		t.Fatalf("unexpected encrypted kid: %v", encrypted["kid"])
	}
	ciphertext := encrypted["value"].(string)
	if ciphertext == "" || ciphertext == "cGxhaW50ZXh0" {
		t.Fatalf("expected encrypted value to differ from plaintext, got %q", ciphertext)
	}

	decryptResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/decrypt?api-version=2025-07-01", []byte(`{"alg":"RSA-OAEP","value":"`+ciphertext+`"}`)))
	if err != nil {
		t.Fatalf("decrypt returned error: %v", err)
	}
	if decryptResp.StatusCode != http.StatusOK {
		t.Fatalf("expected decrypt status 200, got %d; body=%s", decryptResp.StatusCode, string(decryptResp.RawBody))
	}
	decrypted := decodeKeyVaultJSON(t, decryptResp)
	if decrypted["kid"] != kid {
		t.Fatalf("unexpected decrypted kid: %v", decrypted["kid"])
	}
	if decrypted["value"] != "cGxhaW50ZXh0" {
		t.Fatalf("expected decrypted plaintext, got %v", decrypted["value"])
	}
}

func TestKeyCreatePreservesNotBeforeAndExpiryAttributes(t *testing.T) {
	svc := keyvault.New()

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/timed-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":["encrypt","decrypt"],"attributes":{"enabled":true,"nbf":631180800,"exp":662716800}}`)))
	if err != nil {
		t.Fatalf("create timed key returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create timed key status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeKeyVaultJSON(t, createResp)
	version := keyVaultVersionFromID(t, created["key"].(map[string]any)["kid"].(string))
	attributes := created["attributes"].(map[string]any)
	if attributes["nbf"] != float64(631180800) || attributes["exp"] != float64(662716800) {
		t.Fatalf("expected create response to preserve nbf and exp attributes, got %v", attributes)
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/timed-key/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get timed key returned error: %v", err)
	}
	gotAttributes := decodeKeyVaultJSON(t, getResp)["attributes"].(map[string]any)
	if gotAttributes["nbf"] != float64(631180800) || gotAttributes["exp"] != float64(662716800) {
		t.Fatalf("expected get response to preserve nbf and exp attributes, got %v", gotAttributes)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list timed keys returned error: %v", err)
	}
	listItem := decodeKeyVaultJSON(t, listResp)["value"].([]any)[0].(map[string]any)
	listAttributes := listItem["attributes"].(map[string]any)
	if listAttributes["nbf"] != float64(631180800) || listAttributes["exp"] != float64(662716800) {
		t.Fatalf("expected list item to preserve nbf and exp attributes, got %v", listAttributes)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/timed-key/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list timed key versions returned error: %v", err)
	}
	versionItem := decodeKeyVaultJSON(t, versionsResp)["value"].([]any)[0].(map[string]any)
	versionAttributes := versionItem["attributes"].(map[string]any)
	if versionAttributes["nbf"] != float64(631180800) || versionAttributes["exp"] != float64(662716800) {
		t.Fatalf("expected version list item to preserve nbf and exp attributes, got %v", versionAttributes)
	}
}

func TestKeySignVerifyWrapUnwrapOperations(t *testing.T) {
	svc := keyvault.New()

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/crypto-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_size":2048,"key_ops":["sign","verify","wrapKey","unwrapKey"],"attributes":{"enabled":true},"tags":{"purpose":"crypto"}}`)))
	if err != nil {
		t.Fatalf("create crypto key returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create crypto key status 200, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	kid := decodeKeyVaultJSON(t, createResp)["key"].(map[string]any)["kid"].(string)
	digest := base64.RawURLEncoding.EncodeToString([]byte("digest-to-sign"))
	otherDigest := base64.RawURLEncoding.EncodeToString([]byte("different-digest"))

	signResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/sign?api-version=2025-07-01", []byte(`{"alg":"RS256","value":"`+digest+`"}`)))
	if err != nil {
		t.Fatalf("sign returned error: %v", err)
	}
	if signResp.StatusCode != http.StatusOK {
		t.Fatalf("expected sign status 200, got %d; body=%s", signResp.StatusCode, string(signResp.RawBody))
	}
	signature := decodeKeyVaultJSON(t, signResp)
	if signature["kid"] != kid || signature["value"] == "" || signature["value"] == digest {
		t.Fatalf("unexpected sign response: %v", signature)
	}

	verifyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/verify?api-version=2025-07-01", []byte(`{"alg":"RS256","digest":"`+digest+`","value":"`+signature["value"].(string)+`"}`)))
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	if verifyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected verify status 200, got %d; body=%s", verifyResp.StatusCode, string(verifyResp.RawBody))
	}
	if verified := decodeKeyVaultJSON(t, verifyResp)["value"]; verified != true {
		t.Fatalf("expected valid signature to verify true, got %v", verified)
	}

	verifyFalseResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/verify?api-version=2025-07-01", []byte(`{"alg":"RS256","digest":"`+otherDigest+`","value":"`+signature["value"].(string)+`"}`)))
	if err != nil {
		t.Fatalf("verify false returned error: %v", err)
	}
	if verified := decodeKeyVaultJSON(t, verifyFalseResp)["value"]; verified != false {
		t.Fatalf("expected mismatched digest to verify false, got %v", verified)
	}

	plaintextKey := base64.RawURLEncoding.EncodeToString([]byte("local-symmetric-key"))
	wrapResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/wrapkey?api-version=2025-07-01", []byte(`{"alg":"RSA-OAEP-256","value":"`+plaintextKey+`"}`)))
	if err != nil {
		t.Fatalf("wrapkey returned error: %v", err)
	}
	if wrapResp.StatusCode != http.StatusOK {
		t.Fatalf("expected wrapkey status 200, got %d; body=%s", wrapResp.StatusCode, string(wrapResp.RawBody))
	}
	wrapped := decodeKeyVaultJSON(t, wrapResp)
	if wrapped["kid"] != kid || wrapped["value"] == "" || wrapped["value"] == plaintextKey {
		t.Fatalf("unexpected wrapkey response: %v", wrapped)
	}

	unwrapResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, kid+"/unwrapkey?api-version=2025-07-01", []byte(`{"alg":"RSA-OAEP-256","value":"`+wrapped["value"].(string)+`"}`)))
	if err != nil {
		t.Fatalf("unwrapkey returned error: %v", err)
	}
	if unwrapResp.StatusCode != http.StatusOK {
		t.Fatalf("expected unwrapkey status 200, got %d; body=%s", unwrapResp.StatusCode, string(unwrapResp.RawBody))
	}
	unwrapped := decodeKeyVaultJSON(t, unwrapResp)
	if unwrapped["kid"] != kid || unwrapped["value"] != plaintextKey {
		t.Fatalf("unexpected unwrapkey response: %v", unwrapped)
	}
}

func TestKeyGetListVersionsAndUpdate(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/rotating-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_size":2048,"key_ops":["encrypt","decrypt"],"attributes":{"enabled":true},"tags":{"generation":"one"}}`)))
	if err != nil {
		t.Fatalf("create first key version returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first create status 200, got %d; body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	first := decodeKeyVaultJSON(t, firstResp)
	firstKID := first["key"].(map[string]any)["kid"].(string)
	firstVersion := keyVaultVersionFromID(t, firstKID)
	if !isLowerHexVersion(firstVersion) {
		t.Fatalf("expected first key version to be lowercase hex, got %q", firstVersion)
	}

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/rotating-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_size":2048,"key_ops":["sign","verify"],"attributes":{"enabled":true},"tags":{"generation":"two"}}`)))
	if err != nil {
		t.Fatalf("create second key version returned error: %v", err)
	}
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second create status 200, got %d; body=%s", secondResp.StatusCode, string(secondResp.RawBody))
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondKID := second["key"].(map[string]any)["kid"].(string)
	secondVersion := keyVaultVersionFromID(t, secondKID)
	if firstVersion == secondVersion || !isLowerHexVersion(secondVersion) {
		t.Fatalf("expected distinct lowercase hex key versions, got first=%q second=%q", firstVersion, secondVersion)
	}

	latestResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/rotating-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get latest key returned error: %v", err)
	}
	if latestResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get latest key status 200, got %d; body=%s", latestResp.StatusCode, string(latestResp.RawBody))
	}
	latest := decodeKeyVaultJSON(t, latestResp)
	if latest["key"].(map[string]any)["kid"] != secondKID || latest["tags"].(map[string]any)["generation"] != "two" {
		t.Fatalf("expected latest key to be the second version, got %v", latest)
	}

	firstVersionResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/rotating-key/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first key version returned error: %v", err)
	}
	firstVersionKey := decodeKeyVaultJSON(t, firstVersionResp)
	if firstVersionKey["key"].(map[string]any)["kid"] != firstKID || firstVersionKey["tags"].(map[string]any)["generation"] != "one" {
		t.Fatalf("expected first key version to be addressable, got %v", firstVersionKey)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeKeyVaultJSON(t, listResp)["value"].([]any)
	if len(listed) != 1 {
		t.Fatalf("expected one base key in list, got %d: %v", len(listed), listed)
	}
	listItem := listed[0].(map[string]any)
	if listItem["kid"] != "https://vaulta.vault.azure.net/keys/rotating-key" || listItem["tags"].(map[string]any)["generation"] != "two" {
		t.Fatalf("unexpected key list item: %v", listItem)
	}
	if _, includesKeyMaterial := listItem["key"]; includesKeyMaterial {
		t.Fatalf("key list item must not include full key material: %v", listItem)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/rotating-key/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list key versions returned error: %v", err)
	}
	versions := map[string]map[string]any{}
	for _, raw := range decodeKeyVaultJSON(t, versionsResp)["value"].([]any) {
		item := raw.(map[string]any)
		versions[keyVaultVersionFromID(t, item["kid"].(string))] = item
		if _, includesKeyMaterial := item["key"]; includesKeyMaterial {
			t.Fatalf("key version list item must not include full key material: %v", item)
		}
	}
	if len(versions) != 2 || versions[firstVersion] == nil || versions[secondVersion] == nil {
		t.Fatalf("expected both key versions in list, got %v", versions)
	}

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/keys/rotating-key/"+firstVersion+"?api-version=2025-07-01", []byte(`{"key_ops":["decrypt","encrypt"],"attributes":{"enabled":false,"nbf":631180800,"exp":662716800},"tags":{"generation":"updated"}}`)))
	if err != nil {
		t.Fatalf("update first key version returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update key status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeKeyVaultJSON(t, updateResp)
	updatedKey := updated["key"].(map[string]any)
	updatedAttributes := updated["attributes"].(map[string]any)
	if updatedKey["kid"] != firstKID || updatedAttributes["enabled"] != false || updatedAttributes["nbf"] != float64(631180800) || updatedAttributes["exp"] != float64(662716800) {
		t.Fatalf("unexpected updated key response: %v", updated)
	}
	updatedOps := updatedKey["key_ops"].([]any)
	if len(updatedOps) != 2 || updatedOps[0] != "decrypt" || updatedOps[1] != "encrypt" {
		t.Fatalf("unexpected updated key operations: %v", updatedOps)
	}
	if updated["tags"].(map[string]any)["generation"] != "updated" {
		t.Fatalf("unexpected updated key tags: %v", updated["tags"])
	}

	latestAfterUpdateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/rotating-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get latest after update returned error: %v", err)
	}
	latestAfterUpdate := decodeKeyVaultJSON(t, latestAfterUpdateResp)
	if latestAfterUpdate["key"].(map[string]any)["kid"] != secondKID || latestAfterUpdate["attributes"].(map[string]any)["enabled"] != true {
		t.Fatalf("expected updating first version not to replace latest key, got %v", latestAfterUpdate)
	}
}

func TestKeyBackupAndRestorePreservesVersions(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/backup-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":["encrypt","decrypt"],"attributes":{"enabled":true},"tags":{"stage":"first"}}`)))
	if err != nil {
		t.Fatalf("create first key version returned error: %v", err)
	}
	first := decodeKeyVaultJSON(t, firstResp)
	firstVersion := keyVaultVersionFromID(t, first["key"].(map[string]any)["kid"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/backup-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":["sign","verify"],"attributes":{"enabled":true},"tags":{"stage":"second"}}`)))
	if err != nil {
		t.Fatalf("create second key version returned error: %v", err)
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["key"].(map[string]any)["kid"].(string))
	if firstVersion == secondVersion {
		t.Fatalf("expected distinct key versions, got %q", firstVersion)
	}

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/backup-key/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup key returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup key status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backupValue := decodeKeyVaultJSON(t, backupResp)["value"].(string)
	if backupValue == "" || strings.Contains(backupValue, "backup-key") {
		t.Fatalf("expected opaque key backup value, got %q", backupValue)
	}

	restoreSvc := keyvault.New()
	restoreResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/keys/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore key returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restore key status 200, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}
	restored := decodeKeyVaultJSON(t, restoreResp)
	if restored["key"].(map[string]any)["kid"] != "https://vaultb.vault.azure.net/keys/backup-key/"+secondVersion || restored["tags"].(map[string]any)["stage"] != "second" {
		t.Fatalf("expected restored latest key with target vault id, got %v", restored)
	}

	firstVersionResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/keys/backup-key/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored first key version returned error: %v", err)
	}
	firstRestored := decodeKeyVaultJSON(t, firstVersionResp)
	if firstRestored["key"].(map[string]any)["kid"] != "https://vaultb.vault.azure.net/keys/backup-key/"+firstVersion || firstRestored["tags"].(map[string]any)["stage"] != "first" {
		t.Fatalf("expected restored first version, got %v", firstRestored)
	}

	versionsResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/keys/backup-key/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list restored key versions returned error: %v", err)
	}
	versions := map[string]bool{}
	for _, raw := range decodeKeyVaultJSON(t, versionsResp)["value"].([]any) {
		versions[keyVaultVersionFromID(t, raw.(map[string]any)["kid"].(string))] = true
	}
	if !versions[firstVersion] || !versions[secondVersion] || len(versions) != 2 {
		t.Fatalf("expected both restored key versions, got %v", versions)
	}
}

func TestKeyDeleteRecoverAndPurgeLifecycle(t *testing.T) {
	svc := keyvault.New()

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/recoverable-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":["encrypt"],"attributes":{"enabled":true},"tags":{"version":"one"}}`)))
	if err != nil {
		t.Fatalf("create first key version returned error: %v", err)
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["key"].(map[string]any)["kid"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/recoverable-key/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":["decrypt"],"attributes":{"enabled":true},"tags":{"version":"two"}}`)))
	if err != nil {
		t.Fatalf("create second key version returned error: %v", err)
	}
	secondVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, secondResp)["key"].(map[string]any)["kid"].(string))

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/keys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete key returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete key status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["recoveryId"] != "https://vaulta.vault.azure.net/deletedkeys/recoverable-key" || deleted["deletedDate"] == nil || deleted["scheduledPurgeDate"] == nil {
		t.Fatalf("unexpected deleted key metadata: %v", deleted)
	}
	if deleted["key"].(map[string]any)["kid"] != "https://vaulta.vault.azure.net/keys/recoverable-key/"+secondVersion {
		t.Fatalf("expected deleted key bundle to reference latest version, got %v", deleted)
	}

	activeAfterDelete, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get key after delete returned error: %v", err)
	}
	if activeAfterDelete.StatusCode != http.StatusNotFound {
		t.Fatalf("expected active key to be removed after delete, got %d; body=%s", activeAfterDelete.StatusCode, string(activeAfterDelete.RawBody))
	}

	deletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedkeys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted key returned error: %v", err)
	}
	if deletedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get deleted key status 200, got %d; body=%s", deletedResp.StatusCode, string(deletedResp.RawBody))
	}
	if got := decodeKeyVaultJSON(t, deletedResp); got["recoveryId"] != deleted["recoveryId"] {
		t.Fatalf("unexpected get deleted key response: %v", got)
	}

	listDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedkeys?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list deleted keys returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listDeletedResp)["value"].([]any)
	if len(listed) != 1 || listed[0].(map[string]any)["recoveryId"] != deleted["recoveryId"] {
		t.Fatalf("unexpected deleted keys list: %v", listed)
	}

	recoverResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/deletedkeys/recoverable-key/recover?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("recover deleted key returned error: %v", err)
	}
	if recoverResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recover status 200, got %d; body=%s", recoverResp.StatusCode, string(recoverResp.RawBody))
	}
	recovered := decodeKeyVaultJSON(t, recoverResp)
	if recovered["key"].(map[string]any)["kid"] != "https://vaulta.vault.azure.net/keys/recoverable-key/"+secondVersion || recovered["tags"].(map[string]any)["version"] != "two" {
		t.Fatalf("expected recover to restore latest key version, got %v", recovered)
	}

	firstVersionAfterRecover, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/recoverable-key/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first key version after recover returned error: %v", err)
	}
	if got := decodeKeyVaultJSON(t, firstVersionAfterRecover); got["tags"].(map[string]any)["version"] != "one" {
		t.Fatalf("expected recover to preserve older versions, got %v", got)
	}

	_, err = svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/keys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete recovered key returned error: %v", err)
	}
	purgeResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/deletedkeys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("purge deleted key returned error: %v", err)
	}
	if purgeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected purge status 204, got %d; body=%s", purgeResp.StatusCode, string(purgeResp.RawBody))
	}
	getPurged, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedkeys/recoverable-key?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get purged deleted key returned error: %v", err)
	}
	if getPurged.StatusCode != http.StatusNotFound {
		t.Fatalf("expected purged deleted key status 404, got %d; body=%s", getPurged.StatusCode, string(getPurged.RawBody))
	}
}

func TestCertificateBackedKeyProjectsManagedMetadata(t *testing.T) {
	svc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("local-pfx-with-managed-key"))

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/managed-key-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+certificateValue+`",
		"policy":{"key_props":{"kty":"RSA"},"secret_props":{"contentType":"application/x-pkcs12"}},
		"tags":{"owner":"cert"}
	}`)))
	if err != nil {
		t.Fatalf("import certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	version := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, importResp)["id"].(string))
	versionedKID := "https://vaulta.vault.azure.net/keys/managed-key-cert/" + version

	keyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, versionedKID+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get certificate-backed key returned error: %v", err)
	}
	if keyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get key status 200, got %d; body=%s", keyResp.StatusCode, string(keyResp.RawBody))
	}
	keyBundle := decodeKeyVaultJSON(t, keyResp)
	if keyBundle["managed"] != true || keyBundle["key"].(map[string]any)["kid"] != versionedKID {
		t.Fatalf("expected certificate-backed key bundle to expose managed=true and versioned kid, got %v", keyBundle)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listResp)["value"].([]any)
	if len(listed) != 1 {
		t.Fatalf("expected one listed key, got %v", listed)
	}
	listItem := listed[0].(map[string]any)
	if listItem["kid"] != "https://vaulta.vault.azure.net/keys/managed-key-cert" || listItem["managed"] != true {
		t.Fatalf("expected certificate-backed key list item to expose managed=true, got %v", listItem)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/managed-key-cert/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list key versions returned error: %v", err)
	}
	versions := decodeKeyVaultJSON(t, versionsResp)["value"].([]any)
	if len(versions) != 1 {
		t.Fatalf("expected one listed key version, got %v", versions)
	}
	versionItem := versions[0].(map[string]any)
	if versionItem["kid"] != versionedKID || versionItem["managed"] != true {
		t.Fatalf("expected certificate-backed key version item to expose managed=true, got %v", versionItem)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/keys/managed-key-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete certificate-backed key returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete key status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["managed"] != true || deleted["key"].(map[string]any)["kid"] != versionedKID {
		t.Fatalf("expected deleted certificate-backed key bundle to expose managed=true, got %v", deleted)
	}

	getDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedkeys/managed-key-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted certificate-backed key returned error: %v", err)
	}
	getDeleted := decodeKeyVaultJSON(t, getDeletedResp)
	if getDeleted["managed"] != true || getDeleted["key"].(map[string]any)["kid"] != versionedKID {
		t.Fatalf("expected get deleted key to preserve managed=true, got %v", getDeleted)
	}

	listDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedkeys?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list deleted keys returned error: %v", err)
	}
	deletedItems := decodeKeyVaultJSON(t, listDeletedResp)["value"].([]any)
	if len(deletedItems) != 1 {
		t.Fatalf("expected one deleted key item, got %v", deletedItems)
	}
	deletedItem := deletedItems[0].(map[string]any)
	if deletedItem["kid"] != "https://vaulta.vault.azure.net/keys/managed-key-cert" || deletedItem["managed"] != true {
		t.Fatalf("expected deleted key list item to expose managed=true, got %v", deletedItem)
	}
}

func TestCertificateAndKeyListPaginationUsesMaxResultsAndNextLink(t *testing.T) {
	svc := keyvault.New()
	assertPageAdvances := func(label, targetURL, idField string) {
		t.Helper()
		firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, targetURL, nil))
		if err != nil {
			t.Fatalf("%s first page returned error: %v", label, err)
		}
		if firstResp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s first page status 200, got %d; body=%s", label, firstResp.StatusCode, string(firstResp.RawBody))
		}
		firstPage := decodeKeyVaultJSON(t, firstResp)
		firstValues := firstPage["value"].([]any)
		if len(firstValues) != 1 {
			t.Fatalf("expected one %s item in first page, got %v", label, firstValues)
		}
		firstID := firstValues[0].(map[string]any)[idField].(string)
		nextLink, ok := firstPage["nextLink"].(string)
		if !ok || nextLink == "" {
			t.Fatalf("expected %s first page to include non-empty nextLink, got %v", label, firstPage)
		}
		nextResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, nextLink, nil))
		if err != nil {
			t.Fatalf("%s next page returned error: %v", label, err)
		}
		nextValues := decodeKeyVaultJSON(t, nextResp)["value"].([]any)
		if len(nextValues) != 1 || nextValues[0].(map[string]any)[idField].(string) == firstID {
			t.Fatalf("expected %s next page to advance, first=%v next=%v", label, firstValues, nextValues)
		}
	}

	for _, name := range []string{"cert-alpha", "cert-beta"} {
		value := base64.StdEncoding.EncodeToString([]byte(name))
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/"+name+"/import?api-version=2025-07-01", []byte(`{"value":"`+value+`"}`)))
		if err != nil {
			t.Fatalf("import certificate %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected import certificate %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("certificate list", "https://vaulta.vault.azure.net/certificates?api-version=2025-07-01&maxresults=1", "id")

	for _, value := range []string{"cert-v1", "cert-v2", "cert-v3"} {
		encoded := base64.StdEncoding.EncodeToString([]byte(value))
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/cert-rotation/import?api-version=2025-07-01", []byte(`{"value":"`+encoded+`"}`)))
		if err != nil {
			t.Fatalf("import certificate version %s returned error: %v", value, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected import certificate version status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("certificate version list", "https://vaulta.vault.azure.net/certificates/cert-rotation/versions?api-version=2025-07-01&maxresults=1", "id")

	for _, name := range []string{"deleted-cert-alpha", "deleted-cert-beta"} {
		value := base64.StdEncoding.EncodeToString([]byte(name))
		_, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/"+name+"/import?api-version=2025-07-01", []byte(`{"value":"`+value+`"}`)))
		if err != nil {
			t.Fatalf("import deleted-list certificate %s returned error: %v", name, err)
		}
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/certificates/"+name+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("delete certificate %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected delete certificate %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("deleted certificate list", "https://vaulta.vault.azure.net/deletedcertificates?api-version=2025-07-01&maxresults=1", "recoveryId")

	for _, name := range []string{"key-alpha", "key-beta"} {
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/"+name+"/create?api-version=2025-07-01", []byte(`{"kty":"RSA"}`)))
		if err != nil {
			t.Fatalf("create key %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected create key %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("key list", "https://vaulta.vault.azure.net/keys?api-version=2025-07-01&maxresults=1", "kid")

	for _, ops := range []string{`["encrypt"]`, `["decrypt"]`, `["sign"]`} {
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/key-rotation/create?api-version=2025-07-01", []byte(`{"kty":"RSA","key_ops":`+ops+`}`)))
		if err != nil {
			t.Fatalf("create key version %s returned error: %v", ops, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected create key version status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("key version list", "https://vaulta.vault.azure.net/keys/key-rotation/versions?api-version=2025-07-01&maxresults=1", "kid")

	for _, name := range []string{"deleted-key-alpha", "deleted-key-beta"} {
		_, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/keys/"+name+"/create?api-version=2025-07-01", []byte(`{"kty":"RSA"}`)))
		if err != nil {
			t.Fatalf("create deleted-list key %s returned error: %v", name, err)
		}
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/keys/"+name+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("delete key %s returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected delete key %s status 200, got %d; body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
	}
	assertPageAdvances("deleted key list", "https://vaulta.vault.azure.net/deletedkeys?api-version=2025-07-01&maxresults=1", "recoveryId")
}

func TestCertificateCreateAndImportPreserveNotBeforeAndExpiryAttributes(t *testing.T) {
	svc := keyvault.New()
	assertCertificateAttributes := func(label string, bundle map[string]any) {
		t.Helper()
		attributes := bundle["attributes"].(map[string]any)
		if attributes["nbf"] != float64(631180800) || attributes["exp"] != float64(662716800) {
			t.Fatalf("expected %s to preserve nbf and exp attributes, got %v", label, attributes)
		}
	}

	importedValue := base64.StdEncoding.EncodeToString([]byte("certificate-with-expiry"))
	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/timed-import/import?api-version=2025-07-01", []byte(`{
		"value":"`+importedValue+`",
		"attributes":{"enabled":true,"nbf":631180800,"exp":662716800},
		"policy":{"secret_props":{"contentType":"application/x-pkcs12"}}
	}`)))
	if err != nil {
		t.Fatalf("import timed certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import timed certificate status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	imported := decodeKeyVaultJSON(t, importResp)
	assertCertificateAttributes("import response", imported)
	importedVersion := keyVaultVersionFromID(t, imported["id"].(string))

	getImportedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/timed-import/"+importedVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get timed imported certificate returned error: %v", err)
	}
	assertCertificateAttributes("get imported certificate response", decodeKeyVaultJSON(t, getImportedResp))

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/timed-create/create?api-version=2025-07-01", []byte(`{
		"attributes":{"enabled":true,"nbf":631180800,"exp":662716800},
		"policy":{"secret_props":{"contentType":"application/x-pem-file"}}
	}`)))
	if err != nil {
		t.Fatalf("create timed certificate returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected create timed certificate status 202, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	operation := decodeKeyVaultJSON(t, createResp)
	createdVersion := keyVaultVersionFromID(t, operation["target"].(string))

	getCreatedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/timed-create/"+createdVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get timed created certificate returned error: %v", err)
	}
	assertCertificateAttributes("get created certificate response", decodeKeyVaultJSON(t, getCreatedResp))

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list timed certificates returned error: %v", err)
	}
	for _, raw := range decodeKeyVaultJSON(t, listResp)["value"].([]any) {
		item := raw.(map[string]any)
		assertCertificateAttributes("certificate list item "+item["id"].(string), item)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/timed-import/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list timed imported certificate versions returned error: %v", err)
	}
	versionItem := decodeKeyVaultJSON(t, versionsResp)["value"].([]any)[0].(map[string]any)
	assertCertificateAttributes("certificate version list item", versionItem)
}

func TestCertificateLinkedKeyAndSecretMirrorNotBeforeAndExpiryAttributes(t *testing.T) {
	svc := keyvault.New()
	assertMirrorAttributes := func(label string, bundle map[string]any) {
		t.Helper()
		attributes := bundle["attributes"].(map[string]any)
		if attributes["nbf"] != float64(631180800) || attributes["exp"] != float64(662716800) {
			t.Fatalf("expected %s to mirror certificate nbf and exp attributes, got %v", label, attributes)
		}
	}
	assertLinkedResources := func(label string, cert map[string]any) {
		t.Helper()
		secretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, cert["sid"].(string)+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("get %s linked secret returned error: %v", label, err)
		}
		assertMirrorAttributes(label+" linked secret", decodeKeyVaultJSON(t, secretResp))

		keyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, cert["kid"].(string)+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("get %s linked key returned error: %v", label, err)
		}
		assertMirrorAttributes(label+" linked key", decodeKeyVaultJSON(t, keyResp))
	}

	importedValue := base64.StdEncoding.EncodeToString([]byte("certificate-with-mirrored-window"))
	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/mirror-import/import?api-version=2025-07-01", []byte(`{
		"value":"`+importedValue+`",
		"attributes":{"enabled":true,"nbf":631180800,"exp":662716800},
		"policy":{"secret_props":{"contentType":"application/x-pkcs12"}}
	}`)))
	if err != nil {
		t.Fatalf("import mirror certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import mirror certificate status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	assertLinkedResources("imported certificate", decodeKeyVaultJSON(t, importResp))

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/mirror-create/create?api-version=2025-07-01", []byte(`{
		"attributes":{"enabled":true,"nbf":631180800,"exp":662716800},
		"policy":{"secret_props":{"contentType":"application/x-pem-file"}}
	}`)))
	if err != nil {
		t.Fatalf("create mirror certificate returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected create mirror certificate status 202, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	operation := decodeKeyVaultJSON(t, createResp)
	certResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, operation["target"].(string)+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get created mirror certificate returned error: %v", err)
	}
	assertLinkedResources("created certificate", decodeKeyVaultJSON(t, certResp))
}

func TestCertificateUpdateMirrorsLinkedKeyAndSecretAttributes(t *testing.T) {
	svc := keyvault.New()
	importedValue := base64.StdEncoding.EncodeToString([]byte("certificate-update-mirror-window"))

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/update-mirror/import?api-version=2025-07-01", []byte(`{
		"value":"`+importedValue+`",
		"attributes":{"enabled":true,"nbf":631180800,"exp":662716800},
		"policy":{"secret_props":{"contentType":"application/x-pkcs12"}}
	}`)))
	if err != nil {
		t.Fatalf("import update-mirror certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import update-mirror certificate status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	imported := decodeKeyVaultJSON(t, importResp)
	version := keyVaultVersionFromID(t, imported["id"].(string))

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/certificates/update-mirror/"+version+"?api-version=2025-07-01", []byte(`{
		"attributes":{"enabled":true,"nbf":694224000,"exp":725846400}
	}`)))
	if err != nil {
		t.Fatalf("update certificate returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update certificate status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}

	assertLinkedAttributes := func(label, url string) {
		t.Helper()
		resp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, url+"?api-version=2025-07-01", nil))
		if err != nil {
			t.Fatalf("get %s returned error: %v", label, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected get %s status 200, got %d; body=%s", label, resp.StatusCode, string(resp.RawBody))
		}
		attributes := decodeKeyVaultJSON(t, resp)["attributes"].(map[string]any)
		if attributes["enabled"] != true || attributes["nbf"] != float64(694224000) || attributes["exp"] != float64(725846400) {
			t.Fatalf("expected %s to mirror updated certificate attributes, got %v", label, attributes)
		}
	}

	updated := decodeKeyVaultJSON(t, updateResp)
	assertLinkedAttributes("linked secret", updated["sid"].(string))
	assertLinkedAttributes("linked key", updated["kid"].(string))
}

func TestCertificateImportGetListDeleteAndLinkedSecret(t *testing.T) {
	svc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("local-pfx-with-private-key"))
	importBody := []byte(`{
		"value":"` + certificateValue + `",
		"policy":{
			"key_props":{"exportable":true,"kty":"RSA","key_size":2048,"reuse_key":false},
			"secret_props":{"contentType":"application/x-pkcs12"},
			"issuer":{"name":"Unknown"}
		},
		"attributes":{"enabled":true},
		"tags":{"app":"frontend"}
	}`)

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/frontend-cert/import?api-version=2025-07-01", importBody))
	if err != nil {
		t.Fatalf("import certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	imported := decodeKeyVaultJSON(t, importResp)
	version := keyVaultVersionFromID(t, imported["id"].(string))
	if !strings.Contains(imported["id"].(string), "/certificates/frontend-cert/") || !isLowerHexVersion(version) {
		t.Fatalf("expected certificate versioned id, got %v", imported["id"])
	}
	if imported["kid"] != "https://vaulta.vault.azure.net/keys/frontend-cert/"+version {
		t.Fatalf("unexpected certificate key id: %v", imported["kid"])
	}
	if imported["sid"] != "https://vaulta.vault.azure.net/secrets/frontend-cert/"+version {
		t.Fatalf("unexpected certificate secret id: %v", imported["sid"])
	}
	if imported["cer"] != certificateValue || imported["x5t"] == "" {
		t.Fatalf("expected stored certificate bytes and thumbprint, got %v", imported)
	}
	if imported["tags"].(map[string]any)["app"] != "frontend" {
		t.Fatalf("unexpected certificate tags: %v", imported["tags"])
	}
	policy := imported["policy"].(map[string]any)
	if policy["id"] != "https://vaulta.vault.azure.net/certificates/frontend-cert/policy" {
		t.Fatalf("unexpected certificate policy id: %v", policy["id"])
	}
	if policy["secret_props"].(map[string]any)["contentType"] != "application/x-pkcs12" {
		t.Fatalf("unexpected secret policy: %v", policy["secret_props"])
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/frontend-cert/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get certificate returned error: %v", err)
	}
	got := decodeKeyVaultJSON(t, getResp)
	if got["id"] != imported["id"] || got["cer"] != certificateValue {
		t.Fatalf("unexpected get certificate response: %v", got)
	}

	listResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list certificates returned error: %v", err)
	}
	listed := decodeKeyVaultJSON(t, listResp)["value"].([]any)
	if len(listed) != 1 {
		t.Fatalf("expected one listed certificate, got %v", listed)
	}
	listItem := listed[0].(map[string]any)
	if listItem["id"] != "https://vaulta.vault.azure.net/certificates/frontend-cert" || listItem["x5t"] != imported["x5t"] {
		t.Fatalf("unexpected listed certificate: %v", listItem)
	}
	if _, includesValue := listItem["cer"]; includesValue {
		t.Fatalf("certificate list item must not include certificate contents: %v", listItem)
	}

	linkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/frontend-cert/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get linked certificate secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["value"] != certificateValue || linkedSecret["contentType"] != "application/x-pkcs12" {
		t.Fatalf("unexpected linked certificate secret: %v", linkedSecret)
	}
	if linkedSecret["kid"] != "https://vaulta.vault.azure.net/keys/frontend-cert/"+version || linkedSecret["managed"] != true {
		t.Fatalf("expected linked certificate secret to expose backing kid and managed=true, got %v", linkedSecret)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/certificates/frontend-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete certificate returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleted := decodeKeyVaultJSON(t, deleteResp)
	if deleted["recoveryId"] != "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert" || deleted["id"] != imported["id"] {
		t.Fatalf("unexpected deleted certificate bundle: %v", deleted)
	}

	missingActive, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/frontend-cert/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted active certificate returned error: %v", err)
	}
	if missingActive.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted active certificate status 404, got %d; body=%s", missingActive.StatusCode, string(missingActive.RawBody))
	}

	getDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get deleted certificate returned error: %v", err)
	}
	if getDeletedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get deleted certificate status 200, got %d; body=%s", getDeletedResp.StatusCode, string(getDeletedResp.RawBody))
	}
	if got := decodeKeyVaultJSON(t, getDeletedResp); got["recoveryId"] != "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert" {
		t.Fatalf("unexpected get deleted certificate response: %v", got)
	}

	listDeletedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedcertificates?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list deleted certificates returned error: %v", err)
	}
	listedDeleted := decodeKeyVaultJSON(t, listDeletedResp)["value"].([]any)
	if len(listedDeleted) != 1 || listedDeleted[0].(map[string]any)["recoveryId"] != "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert" {
		t.Fatalf("unexpected deleted certificate list: %v", listedDeleted)
	}

	purgeResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("purge deleted certificate returned error: %v", err)
	}
	if purgeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected purge status 204, got %d; body=%s", purgeResp.StatusCode, string(purgeResp.RawBody))
	}
	getPurgedResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/deletedcertificates/frontend-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get purged deleted certificate returned error: %v", err)
	}
	if getPurgedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected purged deleted certificate status 404, got %d; body=%s", getPurgedResp.StatusCode, string(getPurgedResp.RawBody))
	}
}

func TestCertificateVersionsAndRecoverDeletedCertificate(t *testing.T) {
	svc := keyvault.New()
	firstValue := base64.StdEncoding.EncodeToString([]byte("first-certificate-version"))
	secondValue := base64.StdEncoding.EncodeToString([]byte("second-certificate-version"))

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/rotation-cert/import?api-version=2025-07-01", []byte(`{"value":"`+firstValue+`","policy":{"secret_props":{"contentType":"application/x-pem-file"}},"tags":{"rotation":"one"}}`)))
	if err != nil {
		t.Fatalf("import first certificate version returned error: %v", err)
	}
	first := decodeKeyVaultJSON(t, firstResp)
	firstVersion := keyVaultVersionFromID(t, first["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/rotation-cert/import?api-version=2025-07-01", []byte(`{"value":"`+secondValue+`","policy":{"secret_props":{"contentType":"application/x-pem-file"}},"tags":{"rotation":"two"}}`)))
	if err != nil {
		t.Fatalf("import second certificate version returned error: %v", err)
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["id"].(string))
	if firstVersion == secondVersion {
		t.Fatalf("expected distinct certificate versions, got %q", firstVersion)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/rotation-cert/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list certificate versions returned error: %v", err)
	}
	if versionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list versions status 200, got %d; body=%s", versionsResp.StatusCode, string(versionsResp.RawBody))
	}
	listed := decodeKeyVaultJSON(t, versionsResp)["value"].([]any)
	versions := map[string]bool{}
	for _, raw := range listed {
		item := raw.(map[string]any)
		if _, includesCert := item["cer"]; includesCert {
			t.Fatalf("certificate version list item must not include certificate contents: %v", item)
		}
		if _, includesPolicy := item["policy"]; includesPolicy {
			t.Fatalf("certificate version list item must not include full policy: %v", item)
		}
		versions[keyVaultVersionFromID(t, item["id"].(string))] = true
	}
	if !versions[firstVersion] || !versions[secondVersion] || len(versions) != 2 {
		t.Fatalf("expected both certificate versions in list, got %v", versions)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/certificates/rotation-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete certificate returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	recoverResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/deletedcertificates/rotation-cert/recover?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("recover deleted certificate returned error: %v", err)
	}
	if recoverResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recover status 200, got %d; body=%s", recoverResp.StatusCode, string(recoverResp.RawBody))
	}
	recovered := decodeKeyVaultJSON(t, recoverResp)
	if recovered["cer"] != secondValue || !strings.HasSuffix(recovered["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected recover to restore latest certificate version, got %v", recovered)
	}

	firstVersionAfterRecover, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/rotation-cert/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first certificate version after recover returned error: %v", err)
	}
	if got := decodeKeyVaultJSON(t, firstVersionAfterRecover); got["cer"] != firstValue {
		t.Fatalf("expected recover to preserve older certificate versions, got %v", got)
	}

	linkedSecretAfterRecover, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/rotation-cert/"+secondVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get recovered linked certificate secret returned error: %v", err)
	}
	if got := decodeKeyVaultJSON(t, linkedSecretAfterRecover); got["value"] != secondValue || got["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected recovered certificate to restore linked latest secret, got %v", got)
	}
}

func TestUpdateCertificatePropertiesPreservesMaterial(t *testing.T) {
	svc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("stable-certificate-material"))

	importResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/update-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+certificateValue+`",
		"policy":{
			"secret_props":{"contentType":"application/x-pkcs12"},
			"issuer":{"name":"Unknown"}
		},
		"attributes":{"enabled":true},
		"tags":{"department":"initial"}
	}`)))
	if err != nil {
		t.Fatalf("import update certificate returned error: %v", err)
	}
	imported := decodeKeyVaultJSON(t, importResp)
	version := keyVaultVersionFromID(t, imported["id"].(string))
	originalKID := imported["kid"]
	originalSID := imported["sid"]
	originalThumbprint := imported["x5t"]

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/certificates/update-cert/"+version+"?api-version=2025-07-01", []byte(`{
		"attributes":{"enabled":false,"nbf":1430344421,"exp":2208988799},
		"policy":{"issuer":{"name":"Self"},"secret_props":{"contentType":"application/x-pem-file"}},
		"tags":{"department":"KeyVaultTest","owner":"platform"}
	}`)))
	if err != nil {
		t.Fatalf("update certificate returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update certificate status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeKeyVaultJSON(t, updateResp)
	if updated["cer"] != certificateValue || updated["kid"] != originalKID || updated["sid"] != originalSID || updated["x5t"] != originalThumbprint {
		t.Fatalf("expected update to preserve certificate material and linked IDs, got %v", updated)
	}
	attrs := updated["attributes"].(map[string]any)
	if attrs["enabled"] != false || attrs["nbf"] != float64(1430344421) || attrs["exp"] != float64(2208988799) {
		t.Fatalf("unexpected updated certificate attributes: %v", attrs)
	}
	tags := updated["tags"].(map[string]any)
	if tags["department"] != "KeyVaultTest" || tags["owner"] != "platform" {
		t.Fatalf("unexpected updated certificate tags: %v", tags)
	}
	policy := updated["policy"].(map[string]any)
	if policy["id"] != "https://vaulta.vault.azure.net/certificates/update-cert/policy" {
		t.Fatalf("unexpected updated policy id: %v", policy["id"])
	}
	if policy["issuer"].(map[string]any)["name"] != "Self" || policy["secret_props"].(map[string]any)["contentType"] != "application/x-pem-file" {
		t.Fatalf("unexpected updated certificate policy: %v", policy)
	}

	getResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/update-cert/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get updated certificate returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected disabled updated certificate status 403, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listSecretsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list secrets after disabling certificate returned error: %v", err)
	}
	listedSecret := decodeKeyVaultJSON(t, listSecretsResp)["value"].([]any)[0].(map[string]any)
	listedSecretAttrs := listedSecret["attributes"].(map[string]any)
	if listedSecretAttrs["enabled"] != false || listedSecretAttrs["nbf"] != float64(1430344421) || listedSecretAttrs["exp"] != float64(2208988799) {
		t.Fatalf("expected linked secret metadata to mirror disabled certificate attributes, got %v", listedSecretAttrs)
	}
	if listedSecret["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected linked secret metadata to reflect updated certificate policy content type, got %v", listedSecret)
	}

	linkedKeyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, originalKID.(string)+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get disabled linked key returned error: %v", err)
	}
	linkedKeyAttrs := decodeKeyVaultJSON(t, linkedKeyResp)["attributes"].(map[string]any)
	if linkedKeyAttrs["enabled"] != false || linkedKeyAttrs["nbf"] != float64(1430344421) || linkedKeyAttrs["exp"] != float64(2208988799) {
		t.Fatalf("expected linked key to mirror disabled certificate attributes, got %v", linkedKeyAttrs)
	}

	disabledLinkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/update-cert/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get disabled linked secret returned error: %v", err)
	}
	if disabledLinkedSecretResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected disabled linked secret status 403, got %d; body=%s", disabledLinkedSecretResp.StatusCode, string(disabledLinkedSecretResp.RawBody))
	}

	reenableResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/certificates/update-cert/"+version+"?api-version=2025-07-01", []byte(`{
		"attributes":{"enabled":true}
	}`)))
	if err != nil {
		t.Fatalf("reenable certificate returned error: %v", err)
	}
	if reenableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected reenable certificate status 200, got %d; body=%s", reenableResp.StatusCode, string(reenableResp.RawBody))
	}

	linkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, originalSID.(string)+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get updated linked secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["value"] != certificateValue || linkedSecret["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected updated certificate policy to update linked secret properties without changing value, got %v", linkedSecret)
	}
}

func TestCertificatePolicyGetAndUpdateAffectsLatestVersion(t *testing.T) {
	svc := keyvault.New()
	firstValue := base64.StdEncoding.EncodeToString([]byte("policy-first-version"))
	secondValue := base64.StdEncoding.EncodeToString([]byte("policy-second-version"))

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/policy-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+firstValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pkcs12"},"issuer":{"name":"Unknown"}},
		"tags":{"version":"one"}
	}`)))
	if err != nil {
		t.Fatalf("import first policy certificate returned error: %v", err)
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/policy-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+secondValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pkcs12"},"issuer":{"name":"Unknown"}},
		"tags":{"version":"two"}
	}`)))
	if err != nil {
		t.Fatalf("import second policy certificate returned error: %v", err)
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["id"].(string))

	policyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/policy-cert/policy?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get certificate policy returned error: %v", err)
	}
	if policyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get policy status 200, got %d; body=%s", policyResp.StatusCode, string(policyResp.RawBody))
	}
	policy := decodeKeyVaultJSON(t, policyResp)
	if policy["id"] != "https://vaulta.vault.azure.net/certificates/policy-cert/policy" ||
		policy["secret_props"].(map[string]any)["contentType"] != "application/x-pkcs12" ||
		policy["issuer"].(map[string]any)["name"] != "Unknown" {
		t.Fatalf("unexpected initial certificate policy: %v", policy)
	}

	updateResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPatch, "https://vaulta.vault.azure.net/certificates/policy-cert/policy?api-version=2025-07-01", []byte(`{
		"secret_props":{"contentType":"application/x-pem-file"},
		"issuer":{"name":"Self"},
		"x509_props":{"subject":"CN=policy-cert","validity_months":12}
	}`)))
	if err != nil {
		t.Fatalf("update certificate policy returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update policy status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updatedPolicy := decodeKeyVaultJSON(t, updateResp)
	if updatedPolicy["id"] != "https://vaulta.vault.azure.net/certificates/policy-cert/policy" ||
		updatedPolicy["secret_props"].(map[string]any)["contentType"] != "application/x-pem-file" ||
		updatedPolicy["issuer"].(map[string]any)["name"] != "Self" ||
		updatedPolicy["x509_props"].(map[string]any)["subject"] != "CN=policy-cert" {
		t.Fatalf("unexpected updated certificate policy: %v", updatedPolicy)
	}

	latestResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/policy-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get latest policy certificate returned error: %v", err)
	}
	latest := decodeKeyVaultJSON(t, latestResp)
	if latest["cer"] != secondValue || !strings.HasSuffix(latest["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected policy update to preserve latest certificate material, got %v", latest)
	}
	latestPolicy := latest["policy"].(map[string]any)
	if latestPolicy["issuer"].(map[string]any)["name"] != "Self" || latestPolicy["secret_props"].(map[string]any)["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected latest certificate to use updated policy, got %v", latestPolicy)
	}

	olderResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/policy-cert/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get older policy certificate returned error: %v", err)
	}
	older := decodeKeyVaultJSON(t, olderResp)
	if older["cer"] != firstValue || older["policy"].(map[string]any)["secret_props"].(map[string]any)["contentType"] != "application/x-pkcs12" {
		t.Fatalf("expected policy update to leave older certificate version unchanged, got %v", older)
	}

	linkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/policy-cert/"+secondVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get policy-updated linked secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["value"] != secondValue || linkedSecret["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected updated policy to update latest linked secret properties without changing value, got %v", linkedSecret)
	}
}

func TestCertificateBackupAndRestorePreservesVersions(t *testing.T) {
	svc := keyvault.New()
	firstValue := base64.StdEncoding.EncodeToString([]byte("backup-first-version"))
	secondValue := base64.StdEncoding.EncodeToString([]byte("backup-second-version"))

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/backup-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+firstValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pem-file"}},
		"tags":{"stage":"first"}
	}`)))
	if err != nil {
		t.Fatalf("import first backup certificate version returned error: %v", err)
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/backup-cert/import?api-version=2025-07-01", []byte(`{
		"value":"`+secondValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pem-file"},"issuer":{"name":"Self"}},
		"tags":{"stage":"second"}
	}`)))
	if err != nil {
		t.Fatalf("import second backup certificate version returned error: %v", err)
	}
	second := decodeKeyVaultJSON(t, secondResp)
	secondVersion := keyVaultVersionFromID(t, second["id"].(string))

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/backup-cert/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup certificate returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backup := decodeKeyVaultJSON(t, backupResp)
	backupValue, ok := backup["value"].(string)
	if !ok || backupValue == "" || strings.Contains(backupValue, "+") || strings.Contains(backupValue, "/") || strings.Contains(backupValue, "=") {
		t.Fatalf("expected non-empty base64url backup value, got %v", backup)
	}

	deleteResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/certificates/backup-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("delete backed-up certificate returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	purgeResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodDelete, "https://vaulta.vault.azure.net/deletedcertificates/backup-cert?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("purge backed-up certificate returned error: %v", err)
	}
	if purgeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected purge status 204, got %d; body=%s", purgeResp.StatusCode, string(purgeResp.RawBody))
	}

	restoreResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore certificate returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}
	restored := decodeKeyVaultJSON(t, restoreResp)
	if restored["cer"] != secondValue || !strings.HasSuffix(restored["id"].(string), "/"+secondVersion) {
		t.Fatalf("expected restore to return latest version, got %v", restored)
	}
	if restored["policy"].(map[string]any)["issuer"].(map[string]any)["name"] != "Self" {
		t.Fatalf("expected latest certificate policy to survive restore, got %v", restored["policy"])
	}

	firstAfterRestore, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/backup-cert/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get first restored certificate version returned error: %v", err)
	}
	if got := decodeKeyVaultJSON(t, firstAfterRestore); got["cer"] != firstValue || got["tags"].(map[string]any)["stage"] != "first" {
		t.Fatalf("expected restore to preserve first version material and tags, got %v", got)
	}

	versionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/backup-cert/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list restored certificate versions returned error: %v", err)
	}
	versions := map[string]bool{}
	for _, raw := range decodeKeyVaultJSON(t, versionsResp)["value"].([]any) {
		versions[keyVaultVersionFromID(t, raw.(map[string]any)["id"].(string))] = true
	}
	if !versions[firstVersion] || !versions[secondVersion] || len(versions) != 2 {
		t.Fatalf("expected both versions after restore, got %v", versions)
	}

	linkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/backup-cert/"+secondVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored linked certificate secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["value"] != secondValue || linkedSecret["contentType"] != "application/x-pem-file" {
		t.Fatalf("expected restored linked latest secret, got %v", linkedSecret)
	}

	linkedKeyVersionsResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/backup-cert/versions?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("list restored linked certificate key versions returned error: %v", err)
	}
	keyVersions := map[string]bool{}
	for _, raw := range decodeKeyVaultJSON(t, linkedKeyVersionsResp)["value"].([]any) {
		keyVersions[keyVaultVersionFromID(t, raw.(map[string]any)["kid"].(string))] = true
	}
	if !keyVersions[firstVersion] || !keyVersions[secondVersion] || len(keyVersions) != 2 {
		t.Fatalf("expected both linked key versions after restore, got %v", keyVersions)
	}

	firstKeyResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/keys/backup-cert/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored first linked certificate key returned error: %v", err)
	}
	firstKey := decodeKeyVaultJSON(t, firstKeyResp)
	if firstKey["key"].(map[string]any)["kid"] != "https://vaulta.vault.azure.net/keys/backup-cert/"+firstVersion {
		t.Fatalf("expected first linked key version to be restored, got %v", firstKey)
	}
}

func TestCertificateRestoreRewritesTargetVaultIdentifiers(t *testing.T) {
	svc := keyvault.New()
	firstValue := base64.StdEncoding.EncodeToString([]byte("cross-vault-first-version"))
	secondValue := base64.StdEncoding.EncodeToString([]byte("cross-vault-second-version"))

	firstResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/cross-vault/import?api-version=2025-07-01", []byte(`{
		"value":"`+firstValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pem-file"}},
		"tags":{"stage":"first"}
	}`)))
	if err != nil {
		t.Fatalf("import first cross-vault certificate version returned error: %v", err)
	}
	firstVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, firstResp)["id"].(string))

	secondResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/cross-vault/import?api-version=2025-07-01", []byte(`{
		"value":"`+secondValue+`",
		"policy":{"secret_props":{"contentType":"application/x-pem-file"},"issuer":{"name":"Self"}},
		"tags":{"stage":"second"}
	}`)))
	if err != nil {
		t.Fatalf("import second cross-vault certificate version returned error: %v", err)
	}
	secondVersion := keyVaultVersionFromID(t, decodeKeyVaultJSON(t, secondResp)["id"].(string))

	backupResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/cross-vault/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup cross-vault certificate returned error: %v", err)
	}
	if backupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backup status 200, got %d; body=%s", backupResp.StatusCode, string(backupResp.RawBody))
	}
	backupValue := decodeKeyVaultJSON(t, backupResp)["value"].(string)

	restoreSvc := keyvault.New()
	restoreResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/certificates/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore cross-vault certificate returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}
	restored := decodeKeyVaultJSON(t, restoreResp)
	if restored["id"] != "https://vaultb.vault.azure.net/certificates/cross-vault/"+secondVersion {
		t.Fatalf("expected restored certificate id to use target vault, got %v", restored)
	}
	if restored["kid"] != "https://vaultb.vault.azure.net/keys/cross-vault/"+secondVersion {
		t.Fatalf("expected restored certificate kid to use target vault, got %v", restored)
	}
	if restored["sid"] != "https://vaultb.vault.azure.net/secrets/cross-vault/"+secondVersion {
		t.Fatalf("expected restored certificate sid to use target vault, got %v", restored)
	}
	if restored["policy"].(map[string]any)["id"] != "https://vaultb.vault.azure.net/certificates/cross-vault/policy" {
		t.Fatalf("expected restored certificate policy id to use target vault, got %v", restored["policy"])
	}

	firstVersionResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/certificates/cross-vault/"+firstVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored first certificate version returned error: %v", err)
	}
	firstRestored := decodeKeyVaultJSON(t, firstVersionResp)
	if firstRestored["id"] != "https://vaultb.vault.azure.net/certificates/cross-vault/"+firstVersion || firstRestored["kid"] != "https://vaultb.vault.azure.net/keys/cross-vault/"+firstVersion || firstRestored["sid"] != "https://vaultb.vault.azure.net/secrets/cross-vault/"+firstVersion {
		t.Fatalf("expected restored first version identifiers to use target vault, got %v", firstRestored)
	}

	linkedSecretResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/secrets/cross-vault/"+secondVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored linked secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["id"] != "https://vaultb.vault.azure.net/secrets/cross-vault/"+secondVersion || linkedSecret["value"] != secondValue {
		t.Fatalf("expected restored linked secret to use target vault, got %v", linkedSecret)
	}

	linkedKeyResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/keys/cross-vault/"+secondVersion+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get restored linked key returned error: %v", err)
	}
	linkedKey := decodeKeyVaultJSON(t, linkedKeyResp)
	if linkedKey["key"].(map[string]any)["kid"] != "https://vaultb.vault.azure.net/keys/cross-vault/"+secondVersion {
		t.Fatalf("expected restored linked key to use target vault, got %v", linkedKey)
	}

	sourceVaultResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/cross-vault?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get source-vault certificate from restore service returned error: %v", err)
	}
	if sourceVaultResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected restore to target vault only, got %d; body=%s", sourceVaultResp.StatusCode, string(sourceVaultResp.RawBody))
	}
}

func TestCertificateRestoreRejectsExistingActiveCertificate(t *testing.T) {
	sourceSvc := keyvault.New()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("existing-restore-certificate"))

	importResp, err := sourceSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/existing/import?api-version=2025-07-01", []byte(`{"value":"`+certificateValue+`","tags":{"source":"backup"}}`)))
	if err != nil {
		t.Fatalf("import source certificate returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("expected import status 200, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}

	backupResp, err := sourceSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/existing/backup?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("backup source certificate returned error: %v", err)
	}
	backupValue := decodeKeyVaultJSON(t, backupResp)["value"].(string)

	restoreSvc := keyvault.New()
	activeResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/certificates/existing/import?api-version=2025-07-01", []byte(`{"value":"`+base64.StdEncoding.EncodeToString([]byte("already-active"))+`","tags":{"source":"active"}}`)))
	if err != nil {
		t.Fatalf("import active target certificate returned error: %v", err)
	}
	active := decodeKeyVaultJSON(t, activeResp)
	activeID := active["id"].(string)

	restoreResp, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaultb.vault.azure.net/certificates/restore?api-version=2025-07-01", []byte(`{"value":"`+backupValue+`"}`)))
	if err != nil {
		t.Fatalf("restore over active certificate returned error: %v", err)
	}
	if restoreResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected restore conflict status 409, got %d; body=%s", restoreResp.StatusCode, string(restoreResp.RawBody))
	}

	activeAfterRestore, err := restoreSvc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaultb.vault.azure.net/certificates/existing?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get active certificate after rejected restore returned error: %v", err)
	}
	got := decodeKeyVaultJSON(t, activeAfterRestore)
	if got["id"] != activeID || got["tags"].(map[string]any)["source"] != "active" {
		t.Fatalf("expected rejected restore to preserve active certificate, got %v", got)
	}
}

func TestCreateCertificateAndGetPendingOperation(t *testing.T) {
	svc := keyvault.New()

	createResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodPost, "https://vaulta.vault.azure.net/certificates/self-signed/create?api-version=2025-07-01", []byte(`{
		"policy":{
			"key_props":{"exportable":true,"kty":"RSA","key_size":2048,"reuse_key":false},
			"secret_props":{"contentType":"application/x-pkcs12"},
			"x509_props":{"subject":"CN=self-signed.test","validity_months":12},
			"issuer":{"name":"Self"}
		},
		"attributes":{"enabled":true},
		"tags":{"source":"create"}
	}`)))
	if err != nil {
		t.Fatalf("create certificate returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected create certificate status 202, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	operation := decodeKeyVaultJSON(t, createResp)
	if operation["id"] != "https://vaulta.vault.azure.net/certificates/self-signed/pending" {
		t.Fatalf("unexpected operation id: %v", operation["id"])
	}
	if operation["status"] != "completed" || operation["status_details"] == "" || operation["request_id"] == "" || operation["csr"] == "" {
		t.Fatalf("unexpected completed operation response: %v", operation)
	}
	if operation["issuer"].(map[string]any)["name"] != "Self" {
		t.Fatalf("unexpected operation issuer: %v", operation["issuer"])
	}
	target := operation["target"].(string)
	version := keyVaultVersionFromID(t, target)
	if !strings.Contains(target, "/certificates/self-signed/") || !isLowerHexVersion(version) {
		t.Fatalf("expected operation target to point at created certificate version, got %q", target)
	}

	pendingResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/self-signed/pending?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get certificate operation returned error: %v", err)
	}
	if pendingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get operation status 200, got %d; body=%s", pendingResp.StatusCode, string(pendingResp.RawBody))
	}
	pending := decodeKeyVaultJSON(t, pendingResp)
	if pending["id"] != operation["id"] || pending["target"] != target || pending["status"] != "completed" {
		t.Fatalf("unexpected pending operation response: %v", pending)
	}

	certResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/certificates/self-signed?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get created certificate returned error: %v", err)
	}
	cert := decodeKeyVaultJSON(t, certResp)
	if cert["id"] != target || cert["cer"] == "" || cert["kid"] != "https://vaulta.vault.azure.net/keys/self-signed/"+version || cert["sid"] != "https://vaulta.vault.azure.net/secrets/self-signed/"+version {
		t.Fatalf("unexpected created certificate bundle: %v", cert)
	}
	if cert["tags"].(map[string]any)["source"] != "create" {
		t.Fatalf("unexpected created certificate tags: %v", cert["tags"])
	}
	policy := cert["policy"].(map[string]any)
	if policy["x509_props"].(map[string]any)["subject"] != "CN=self-signed.test" || policy["issuer"].(map[string]any)["name"] != "Self" {
		t.Fatalf("unexpected created certificate policy: %v", policy)
	}

	linkedSecretResp, err := svc.HandleRequest(keyVaultCtx(t, http.MethodGet, "https://vaulta.vault.azure.net/secrets/self-signed/"+version+"?api-version=2025-07-01", nil))
	if err != nil {
		t.Fatalf("get created certificate linked secret returned error: %v", err)
	}
	linkedSecret := decodeKeyVaultJSON(t, linkedSecretResp)
	if linkedSecret["value"] != cert["cer"] || linkedSecret["contentType"] != "application/x-pkcs12" {
		t.Fatalf("unexpected linked secret for created certificate: %v", linkedSecret)
	}
}
