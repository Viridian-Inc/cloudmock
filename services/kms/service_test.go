package kms_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	kmssvc "github.com/neureaux/cloudmock/services/kms"
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
func kmsReq(t *testing.T, action string, body interface{}) *http.Request {
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
func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
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
	meta, ok := m["KeyMetadata"].(map[string]interface{})
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
	meta := m["KeyMetadata"].(map[string]interface{})
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
	keyID := mCreate["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

	// Describe the key.
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, kmsReq(t, "DescribeKey", map[string]string{"KeyId": keyID}))

	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeKey: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	meta := mDesc["KeyMetadata"].(map[string]interface{})
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
		keyIDs = append(keyIDs, mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string))
	}

	// List keys.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, kmsReq(t, "ListKeys", nil))

	if wl.Code != http.StatusOK {
		t.Fatalf("ListKeys: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	ml := decodeJSON(t, wl.Body.String())
	keys, ok := ml["Keys"].([]interface{})
	if !ok {
		t.Fatalf("ListKeys: missing Keys in response\nbody: %s", wl.Body.String())
	}
	if len(keys) < 2 {
		t.Errorf("ListKeys: expected at least 2 keys, got %d", len(keys))
	}

	// Verify both created key IDs appear in the list.
	listed := make(map[string]bool)
	for _, k := range keys {
		entry := k.(map[string]interface{})
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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

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
	aliases, ok := ml["Aliases"].([]interface{})
	if !ok || len(aliases) == 0 {
		t.Fatalf("ListAliases: expected non-empty Aliases\nbody: %s", wl.Body.String())
	}

	found := false
	for _, a := range aliases {
		entry := a.(map[string]interface{})
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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

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
	state := mdesc["KeyMetadata"].(map[string]interface{})["KeyState"].(string)
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
	state2 := mdesc2["KeyMetadata"].(map[string]interface{})["KeyState"].(string)
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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

	// Schedule deletion.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, kmsReq(t, "ScheduleKeyDeletion", map[string]interface{}{
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
	state := mdesc["KeyMetadata"].(map[string]interface{})["KeyState"].(string)
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
	keyID := mc["KeyMetadata"].(map[string]interface{})["KeyId"].(string)

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

// ---- UnknownAction ----

func TestKMS_UnknownAction(t *testing.T) {
	handler := newKMSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, kmsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
