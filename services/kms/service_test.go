package kms_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	kmssvc "github.com/Viridian-Inc/cloudmock/services/kms"
)

// newKMSGateway builds a full gateway stack with the KMS service registered and IAM disabled.
func newKMSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(kmssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// kmsReq builds a JSON POST request targeting the KMS service via X-Amz-Target.
func kmsReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("kmsReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "TrentService."+action)
	// Authorization header places "kms" as the service in the credential scope
	// so the gateway router can detect "kms" as the target service.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/kms/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// ---- CreateKey ----

func TestKMS_CreateKey(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "CreateKey", map[string]string{
		"Description": "test key",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("CreateKey: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	meta, ok := m["KeyMetadata"].(map[string]any)
	if !ok {
		t.Fatalf("CreateKey: missing KeyMetadata in response\nbody: %s", w.Body.String())
	}

	keyID, _ := meta["KeyId"].(string)
	if keyID == "" {
		t.Errorf("CreateKey: KeyId is empty")
	}
	arn, _ := meta["Arn"].(string)
	if !strings.Contains(arn, keyID) {
		t.Errorf("CreateKey: Arn %q does not contain KeyId %q", arn, keyID)
	}
	keyState, _ := meta["KeyState"].(string)
	if keyState != "Enabled" {
		t.Errorf("CreateKey: expected KeyState=Enabled, got %q", keyState)
	}
	desc, _ := meta["Description"].(string)
	if desc != "test key" {
		t.Errorf("CreateKey: expected Description=%q, got %q", "test key", desc)
	}
}

func TestKMS_CreateKey_DefaultKeyUsage(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "CreateKey", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("CreateKey default KeyUsage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	meta := m["KeyMetadata"].(map[string]any)
	keyUsage, _ := meta["KeyUsage"].(string)
	if keyUsage != "ENCRYPT_DECRYPT" {
		t.Errorf("CreateKey: expected KeyUsage=ENCRYPT_DECRYPT, got %q", keyUsage)
	}
}

// ---- DescribeKey ----

func TestKMS_DescribeKey(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key first.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, kmsReq(t, "CreateKey", map[string]string{"Description": "desc-key"}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wCreate.Code, wCreate.Body.String())
	}
	mCreate := decodeJSON(t, wCreate.Body.String())
	keyID := mCreate["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Describe the key.
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, kmsReq(t, "DescribeKey", map[string]string{"KeyId": keyID}))

	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeKey: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	meta := mDesc["KeyMetadata"].(map[string]any)
	if meta["KeyId"].(string) != keyID {
		t.Errorf("DescribeKey: expected KeyId=%q, got %q", keyID, meta["KeyId"])
	}
	if meta["Description"].(string) != "desc-key" {
		t.Errorf("DescribeKey: expected Description=%q, got %q", "desc-key", meta["Description"])
	}
}

func TestKMS_DescribeKey_NotFound(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "DescribeKey", map[string]string{
		"KeyId": "00000000-0000-0000-0000-000000000000",
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("DescribeKey not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- ListKeys ----

func TestKMS_ListKeys(t *testing.T) {
	handler := newKMSGateway(t)

	// Create two keys.
	var keyIDs []string
	for i := 0; i < 2; i++ {
		wc := httptest.NewRecorder()
		handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
		if wc.Code != http.StatusOK {
			t.Fatalf("setup CreateKey %d: %d %s", i, wc.Code, wc.Body.String())
		}
		mc := decodeJSON(t, wc.Body.String())
		keyIDs = append(keyIDs, mc["KeyMetadata"].(map[string]any)["KeyId"].(string))
	}

	// List keys.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, kmsReq(t, "ListKeys", nil))

	if wl.Code != http.StatusOK {
		t.Fatalf("ListKeys: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	ml := decodeJSON(t, wl.Body.String())
	keys, ok := ml["Keys"].([]any)
	if !ok {
		t.Fatalf("ListKeys: missing Keys in response\nbody: %s", wl.Body.String())
	}
	if len(keys) < 2 {
		t.Errorf("ListKeys: expected at least 2 keys, got %d", len(keys))
	}

	// Verify both created key IDs appear in the list.
	listed := make(map[string]bool)
	for _, k := range keys {
		entry := k.(map[string]any)
		listed[entry["KeyId"].(string)] = true
	}
	for _, id := range keyIDs {
		if !listed[id] {
			t.Errorf("ListKeys: KeyId %q not found in list", id)
		}
	}
}

// ---- Encrypt + Decrypt ----

func TestKMS_EncryptDecrypt_RoundTrip(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Encrypt plaintext.
	originalText := "hello, KMS world!"
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(originalText))

	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     keyID,
		"Plaintext": plaintextB64,
	}))
	if we.Code != http.StatusOK {
		t.Fatalf("Encrypt: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	me := decodeJSON(t, we.Body.String())
	ciphertextB64, _ := me["CiphertextBlob"].(string)
	if ciphertextB64 == "" {
		t.Fatal("Encrypt: missing CiphertextBlob in response")
	}

	// Decrypt ciphertext.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kmsReq(t, "Decrypt", map[string]string{
		"CiphertextBlob": ciphertextB64,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("Decrypt: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	decryptedB64, _ := md["Plaintext"].(string)
	if decryptedB64 == "" {
		t.Fatal("Decrypt: missing Plaintext in response")
	}

	decrypted, err := base64.StdEncoding.DecodeString(decryptedB64)
	if err != nil {
		t.Fatalf("Decrypt: failed to base64-decode Plaintext: %v", err)
	}

	if string(decrypted) != originalText {
		t.Errorf("Decrypt: expected %q, got %q", originalText, string(decrypted))
	}
}

func TestKMS_Decrypt_InvalidCiphertext(t *testing.T) {
	handler := newKMSGateway(t)

	badBlob := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "Decrypt", map[string]string{
		"CiphertextBlob": badBlob,
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Decrypt invalid: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- CreateAlias ----

func TestKMS_CreateAlias(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key first.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Create an alias.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/my-test-key",
		"TargetKeyId": keyID,
	}))

	if wa.Code != http.StatusOK {
		t.Fatalf("CreateAlias: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}
}

func TestKMS_CreateAlias_InvalidPrefix(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "my-test-key",
		"TargetKeyId": "00000000-0000-0000-0000-000000000000",
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateAlias bad prefix: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- ListAliases ----

func TestKMS_ListAliases(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key and alias.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/list-test",
		"TargetKeyId": keyID,
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("setup CreateAlias: %d %s", wa.Code, wa.Body.String())
	}

	// List aliases.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, kmsReq(t, "ListAliases", nil))

	if wl.Code != http.StatusOK {
		t.Fatalf("ListAliases: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	ml := decodeJSON(t, wl.Body.String())
	aliases, ok := ml["Aliases"].([]any)
	if !ok || len(aliases) == 0 {
		t.Fatalf("ListAliases: expected non-empty Aliases\nbody: %s", wl.Body.String())
	}

	found := false
	for _, a := range aliases {
		entry := a.(map[string]any)
		if entry["AliasName"].(string) == "alias/list-test" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListAliases: alias/list-test not found\nbody: %s", wl.Body.String())
	}
}

// ---- EnableKey / DisableKey ----

func TestKMS_EnableDisableKey(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Disable the key.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kmsReq(t, "DisableKey", map[string]string{"KeyId": keyID}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DisableKey: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify key is disabled (DescribeKey).
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, kmsReq(t, "DescribeKey", map[string]string{"KeyId": keyID}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeKey after disable: %d %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	state := mdesc["KeyMetadata"].(map[string]any)["KeyState"].(string)
	if state != "Disabled" {
		t.Errorf("DisableKey: expected KeyState=Disabled, got %q", state)
	}

	// Re-enable the key.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "EnableKey", map[string]string{"KeyId": keyID}))
	if we.Code != http.StatusOK {
		t.Fatalf("EnableKey: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	// Verify key is enabled again.
	wdesc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc2, kmsReq(t, "DescribeKey", map[string]string{"KeyId": keyID}))
	mdesc2 := decodeJSON(t, wdesc2.Body.String())
	state2 := mdesc2["KeyMetadata"].(map[string]any)["KeyState"].(string)
	if state2 != "Enabled" {
		t.Errorf("EnableKey: expected KeyState=Enabled, got %q", state2)
	}
}

// ---- ScheduleKeyDeletion ----

func TestKMS_ScheduleKeyDeletion(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Schedule deletion.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, kmsReq(t, "ScheduleKeyDeletion", map[string]any{
		"KeyId":               keyID,
		"PendingWindowInDays": 7,
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("ScheduleKeyDeletion: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}

	ms := decodeJSON(t, ws.Body.String())
	if ms["KeyId"].(string) != keyID {
		t.Errorf("ScheduleKeyDeletion: expected KeyId=%q, got %q", keyID, ms["KeyId"])
	}
	if _, ok := ms["DeletionDate"]; !ok {
		t.Error("ScheduleKeyDeletion: missing DeletionDate in response")
	}

	// Verify key is pending deletion.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, kmsReq(t, "DescribeKey", map[string]string{"KeyId": keyID}))
	mdesc := decodeJSON(t, wdesc.Body.String())
	state := mdesc["KeyMetadata"].(map[string]any)["KeyState"].(string)
	if state != "PendingDeletion" {
		t.Errorf("ScheduleKeyDeletion: expected KeyState=PendingDeletion, got %q", state)
	}
}

// ---- Alias-based lookup for Encrypt ----

func TestKMS_EncryptByAlias(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Create alias.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/encrypt-test",
		"TargetKeyId": keyID,
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("setup CreateAlias: %d %s", wa.Code, wa.Body.String())
	}

	// Encrypt using alias.
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte("via alias"))
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     "alias/encrypt-test",
		"Plaintext": plaintextB64,
	}))
	if we.Code != http.StatusOK {
		t.Fatalf("Encrypt via alias: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	me := decodeJSON(t, we.Body.String())
	if me["CiphertextBlob"] == "" {
		t.Error("Encrypt via alias: missing CiphertextBlob")
	}
}

// ---- Encrypt with disabled key — DisabledException ----

func TestKMS_Encrypt_DisabledKey(t *testing.T) {
	handler := newKMSGateway(t)

	// Create and disable a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kmsReq(t, "DisableKey", map[string]string{"KeyId": keyID}))
	if wd.Code != http.StatusOK {
		t.Fatalf("setup DisableKey: %d %s", wd.Code, wd.Body.String())
	}

	// Encrypt with disabled key should fail.
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte("test"))
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     keyID,
		"Plaintext": plaintextB64,
	}))
	if we.Code != http.StatusBadRequest {
		t.Fatalf("Encrypt disabled key: expected 400, got %d\nbody: %s", we.Code, we.Body.String())
	}
	body := we.Body.String()
	if !strings.Contains(body, "Disabled") && !strings.Contains(body, "disabled") {
		t.Errorf("Encrypt disabled key: expected DisabledException in body\nbody: %s", body)
	}
}

// ---- NotFoundException — DescribeKey nonexistent (explicit code check) ----

func TestKMS_DescribeKey_NotFoundException(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "DescribeKey", map[string]string{
		"KeyId": "00000000-0000-0000-0000-000000000000",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DescribeKey not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "NotFoundException") {
		t.Errorf("DescribeKey not found: expected NotFoundException in body\nbody: %s", body)
	}
}

// ---- NotFoundException — Encrypt with nonexistent key ----

func TestKMS_Encrypt_KeyNotFound(t *testing.T) {
	handler := newKMSGateway(t)

	plaintextB64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     "00000000-0000-0000-0000-000000000000",
		"Plaintext": plaintextB64,
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Encrypt key not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "NotFoundException") {
		t.Errorf("Encrypt key not found: expected NotFoundException in body\nbody: %s", body)
	}
}

// ---- CreateAlias — duplicate (AlreadyExistsException) ----

func TestKMS_CreateAlias_AlreadyExists(t *testing.T) {
	handler := newKMSGateway(t)

	// Create key + alias.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/dup-test",
		"TargetKeyId": keyID,
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("setup CreateAlias: %d %s", wa.Code, wa.Body.String())
	}

	// Create same alias again — should fail.
	wa2 := httptest.NewRecorder()
	handler.ServeHTTP(wa2, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/dup-test",
		"TargetKeyId": keyID,
	}))
	if wa2.Code != http.StatusConflict {
		t.Fatalf("CreateAlias duplicate: expected 409, got %d\nbody: %s", wa2.Code, wa2.Body.String())
	}
	body := wa2.Body.String()
	if !strings.Contains(body, "AlreadyExistsException") {
		t.Errorf("CreateAlias duplicate: expected AlreadyExistsException\nbody: %s", body)
	}
}

// ---- ScheduleKeyDeletion — encrypt after deletion scheduled ----

func TestKMS_Encrypt_AfterScheduledDeletion(t *testing.T) {
	handler := newKMSGateway(t)

	// Create a key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Schedule deletion.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, kmsReq(t, "ScheduleKeyDeletion", map[string]any{
		"KeyId":               keyID,
		"PendingWindowInDays": 7,
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("setup ScheduleKeyDeletion: %d %s", ws.Code, ws.Body.String())
	}

	// Encrypt should fail (key is PendingDeletion).
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte("test"))
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     keyID,
		"Plaintext": plaintextB64,
	}))
	if we.Code != http.StatusBadRequest {
		t.Fatalf("Encrypt after scheduled deletion: expected 400, got %d\nbody: %s", we.Code, we.Body.String())
	}

	// EnableKey on PendingDeletion key should also fail.
	wek := httptest.NewRecorder()
	handler.ServeHTTP(wek, kmsReq(t, "EnableKey", map[string]string{"KeyId": keyID}))
	if wek.Code != http.StatusBadRequest {
		t.Fatalf("EnableKey pending deletion: expected 400, got %d\nbody: %s", wek.Code, wek.Body.String())
	}
}

// ---- Decrypt roundtrip by alias ----

func TestKMS_DecryptByAlias_RoundTrip(t *testing.T) {
	handler := newKMSGateway(t)

	// Create key + alias.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateKey: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, kmsReq(t, "CreateAlias", map[string]string{
		"AliasName":   "alias/decrypt-rt",
		"TargetKeyId": keyID,
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("setup CreateAlias: %d %s", wa.Code, wa.Body.String())
	}

	// Encrypt using alias.
	originalText := "alias roundtrip test"
	plaintextB64 := base64.StdEncoding.EncodeToString([]byte(originalText))
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, kmsReq(t, "Encrypt", map[string]string{
		"KeyId":     "alias/decrypt-rt",
		"Plaintext": plaintextB64,
	}))
	if we.Code != http.StatusOK {
		t.Fatalf("Encrypt via alias: %d %s", we.Code, we.Body.String())
	}
	me := decodeJSON(t, we.Body.String())
	ciphertextB64 := me["CiphertextBlob"].(string)

	// Decrypt — KMS should find the key from the ciphertext.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kmsReq(t, "Decrypt", map[string]string{
		"CiphertextBlob": ciphertextB64,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("Decrypt alias roundtrip: %d %s", wd.Code, wd.Body.String())
	}
	md := decodeJSON(t, wd.Body.String())
	decryptedB64 := md["Plaintext"].(string)
	decrypted, err := base64.StdEncoding.DecodeString(decryptedB64)
	if err != nil {
		t.Fatalf("Decrypt base64 decode: %v", err)
	}
	if string(decrypted) != originalText {
		t.Errorf("Decrypt: expected %q, got %q", originalText, string(decrypted))
	}
}

// ---- UnknownAction ----

func TestKMS_UnknownAction(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// NEW OPERATIONS — fixes for LocalStack's broken KMS
// ===========================================================================

// ---- GenerateDataKey — real envelope encryption ----

func TestKMS_GenerateDataKey(t *testing.T) {
	handler := newKMSGateway(t)

	// Create key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// GenerateDataKey.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "GenerateDataKey", map[string]any{
		"KeyId":         keyID,
		"NumberOfBytes": 32,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GenerateDataKey: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())

	// Must have both plaintext and ciphertext.
	ptB64 := m["Plaintext"].(string)
	ctB64 := m["CiphertextBlob"].(string)
	if ptB64 == "" || ctB64 == "" {
		t.Fatal("GenerateDataKey: missing Plaintext or CiphertextBlob")
	}

	pt, _ := base64.StdEncoding.DecodeString(ptB64)
	if len(pt) != 32 {
		t.Errorf("GenerateDataKey: Plaintext length = %d, want 32", len(pt))
	}

	// Decrypt the ciphertext blob — should recover the same data key.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, kmsReq(t, "Decrypt", map[string]string{"CiphertextBlob": ctB64}))
	if wd.Code != http.StatusOK {
		t.Fatalf("Decrypt data key: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	md := decodeJSON(t, wd.Body.String())
	decrypted, _ := base64.StdEncoding.DecodeString(md["Plaintext"].(string))

	if !bytes.Equal(pt, decrypted) {
		t.Error("GenerateDataKey: decrypted ciphertext does not match plaintext data key")
	}
}

func TestKMS_GenerateDataKeyWithoutPlaintext(t *testing.T) {
	handler := newKMSGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "GenerateDataKeyWithoutPlaintext", map[string]any{
		"KeyId": keyID,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GenerateDataKeyWithoutPlaintext: %d %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	if m["CiphertextBlob"] == "" {
		t.Error("GenerateDataKeyWithoutPlaintext: missing CiphertextBlob")
	}
	if m["Plaintext"] != nil {
		t.Error("GenerateDataKeyWithoutPlaintext: should not contain Plaintext")
	}
}

// ---- HMAC Operations — LocalStack doesn't support these ----

func TestKMS_HMAC_GenerateAndVerify(t *testing.T) {
	handler := newKMSGateway(t)

	// Create HMAC key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", map[string]string{
		"KeyUsage": "GENERATE_VERIFY_MAC",
		"KeySpec":  "HMAC_256",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateKey HMAC: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	message := base64.StdEncoding.EncodeToString([]byte("HIPAA-critical patient data"))

	// GenerateMac.
	wm := httptest.NewRecorder()
	handler.ServeHTTP(wm, kmsReq(t, "GenerateMac", map[string]string{
		"KeyId":        keyID,
		"Message":      message,
		"MacAlgorithm": "HMAC_SHA_256",
	}))
	if wm.Code != http.StatusOK {
		t.Fatalf("GenerateMac: %d %s", wm.Code, wm.Body.String())
	}
	mm := decodeJSON(t, wm.Body.String())
	macB64 := mm["Mac"].(string)
	if macB64 == "" {
		t.Fatal("GenerateMac: missing Mac")
	}

	// VerifyMac — should succeed with correct MAC.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, kmsReq(t, "VerifyMac", map[string]string{
		"KeyId":        keyID,
		"Message":      message,
		"Mac":          macB64,
		"MacAlgorithm": "HMAC_SHA_256",
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("VerifyMac: %d %s", wv.Code, wv.Body.String())
	}
	mv := decodeJSON(t, wv.Body.String())
	if mv["MacValid"] != true {
		t.Error("VerifyMac: expected MacValid=true for correct MAC")
	}

	// VerifyMac — should fail with wrong MAC.
	badMac := base64.StdEncoding.EncodeToString([]byte("wrong-mac-value-0000000000000000"))
	wv2 := httptest.NewRecorder()
	handler.ServeHTTP(wv2, kmsReq(t, "VerifyMac", map[string]string{
		"KeyId":        keyID,
		"Message":      message,
		"Mac":          badMac,
		"MacAlgorithm": "HMAC_SHA_256",
	}))
	if wv2.Code != http.StatusOK {
		t.Fatalf("VerifyMac bad: %d %s", wv2.Code, wv2.Body.String())
	}
	mv2 := decodeJSON(t, wv2.Body.String())
	if mv2["MacValid"] != false {
		t.Error("VerifyMac: expected MacValid=false for incorrect MAC")
	}
}

// ---- RSA Sign/Verify — LocalStack stubs these ----

func TestKMS_RSA_SignVerify(t *testing.T) {
	handler := newKMSGateway(t)

	// Create RSA key.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", map[string]string{
		"KeyUsage": "SIGN_VERIFY",
		"KeySpec":  "RSA_2048",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateKey RSA: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	message := base64.StdEncoding.EncodeToString([]byte("document to sign"))

	// Sign.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, kmsReq(t, "Sign", map[string]string{
		"KeyId":            keyID,
		"Message":          message,
		"SigningAlgorithm": "RSASSA_PKCS1_V1_5_SHA_256",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("Sign RSA: %d %s", ws.Code, ws.Body.String())
	}
	ms := decodeJSON(t, ws.Body.String())
	sigB64 := ms["Signature"].(string)
	if sigB64 == "" {
		t.Fatal("Sign: missing Signature")
	}

	// Verify — should succeed.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, kmsReq(t, "Verify", map[string]string{
		"KeyId":            keyID,
		"Message":          message,
		"Signature":        sigB64,
		"SigningAlgorithm": "RSASSA_PKCS1_V1_5_SHA_256",
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("Verify RSA: %d %s", wv.Code, wv.Body.String())
	}
	mv := decodeJSON(t, wv.Body.String())
	if mv["SignatureValid"] != true {
		t.Error("Verify RSA: expected SignatureValid=true")
	}

	// Verify with wrong message — should fail.
	wrongMsg := base64.StdEncoding.EncodeToString([]byte("tampered document"))
	wv2 := httptest.NewRecorder()
	handler.ServeHTTP(wv2, kmsReq(t, "Verify", map[string]string{
		"KeyId":            keyID,
		"Message":          wrongMsg,
		"Signature":        sigB64,
		"SigningAlgorithm": "RSASSA_PKCS1_V1_5_SHA_256",
	}))
	mv2 := decodeJSON(t, wv2.Body.String())
	if mv2["SignatureValid"] != false {
		t.Error("Verify RSA wrong message: expected SignatureValid=false")
	}
}

// ---- ECC Sign/Verify ----

func TestKMS_ECC_SignVerify(t *testing.T) {
	handler := newKMSGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", map[string]string{
		"KeyUsage": "SIGN_VERIFY",
		"KeySpec":  "ECC_NIST_P256",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateKey ECC: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	message := base64.StdEncoding.EncodeToString([]byte("ECDSA test"))

	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, kmsReq(t, "Sign", map[string]string{
		"KeyId":            keyID,
		"Message":          message,
		"SigningAlgorithm": "ECDSA_SHA_256",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("Sign ECC: %d %s", ws.Code, ws.Body.String())
	}
	ms := decodeJSON(t, ws.Body.String())
	sigB64 := ms["Signature"].(string)

	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, kmsReq(t, "Verify", map[string]string{
		"KeyId":            keyID,
		"Message":          message,
		"Signature":        sigB64,
		"SigningAlgorithm": "ECDSA_SHA_256",
	}))
	mv := decodeJSON(t, wv.Body.String())
	if mv["SignatureValid"] != true {
		t.Error("Verify ECC: expected SignatureValid=true")
	}
}

// ---- Key Rotation — LocalStack's is broken ----

func TestKMS_KeyRotation(t *testing.T) {
	handler := newKMSGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	// Initially rotation should be disabled.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, kmsReq(t, "GetKeyRotationStatus", map[string]string{"KeyId": keyID}))
	if w1.Code != http.StatusOK {
		t.Fatalf("GetKeyRotationStatus: %d %s", w1.Code, w1.Body.String())
	}
	m1 := decodeJSON(t, w1.Body.String())
	if m1["KeyRotationEnabled"] != false {
		t.Error("expected rotation disabled by default")
	}

	// Enable rotation.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, kmsReq(t, "EnableKeyRotation", map[string]string{"KeyId": keyID}))
	if w2.Code != http.StatusOK {
		t.Fatalf("EnableKeyRotation: %d %s", w2.Code, w2.Body.String())
	}

	// Check it's enabled.
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, kmsReq(t, "GetKeyRotationStatus", map[string]string{"KeyId": keyID}))
	m3 := decodeJSON(t, w3.Body.String())
	if m3["KeyRotationEnabled"] != true {
		t.Error("expected rotation enabled after EnableKeyRotation")
	}

	// Disable rotation.
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, kmsReq(t, "DisableKeyRotation", map[string]string{"KeyId": keyID}))
	if w4.Code != http.StatusOK {
		t.Fatalf("DisableKeyRotation: %d %s", w4.Code, w4.Body.String())
	}

	// Check it's disabled again.
	w5 := httptest.NewRecorder()
	handler.ServeHTTP(w5, kmsReq(t, "GetKeyRotationStatus", map[string]string{"KeyId": keyID}))
	m5 := decodeJSON(t, w5.Body.String())
	if m5["KeyRotationEnabled"] != false {
		t.Error("expected rotation disabled after DisableKeyRotation")
	}
}

// ---- HMAC on symmetric key should fail ----

func TestKMS_HMAC_OnSymmetricKey_Fails(t *testing.T) {
	handler := newKMSGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", nil))
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	message := base64.StdEncoding.EncodeToString([]byte("test"))
	wm := httptest.NewRecorder()
	handler.ServeHTTP(wm, kmsReq(t, "GenerateMac", map[string]string{
		"KeyId":        keyID,
		"Message":      message,
		"MacAlgorithm": "HMAC_SHA_256",
	}))
	if wm.Code != http.StatusBadRequest {
		t.Errorf("GenerateMac on symmetric key: expected 400, got %d", wm.Code)
	}
}

// ---- Key rotation on asymmetric key should fail ----

func TestKMS_KeyRotation_OnAsymmetricKey_Fails(t *testing.T) {
	handler := newKMSGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, kmsReq(t, "CreateKey", map[string]string{
		"KeyUsage": "SIGN_VERIFY",
		"KeySpec":  "RSA_2048",
	}))
	mc := decodeJSON(t, wc.Body.String())
	keyID := mc["KeyMetadata"].(map[string]any)["KeyId"].(string)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "EnableKeyRotation", map[string]string{"KeyId": keyID}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("EnableKeyRotation on RSA key: expected 400, got %d", w.Code)
	}
}
