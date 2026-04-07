package kms

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createKeyRequest struct {
	Description string `json:"Description"`
	KeyUsage    string `json:"KeyUsage"`
	KeySpec     string `json:"KeySpec"`
}

type keyMetadata struct {
	KeyId        string    `json:"KeyId"`
	Arn          string    `json:"Arn"`
	Description  string    `json:"Description"`
	KeyState     string    `json:"KeyState"`
	KeyUsage     string    `json:"KeyUsage"`
	CreationDate float64   `json:"CreationDate"` // Unix timestamp (float) as AWS SDK expects
	Enabled      bool      `json:"Enabled"`
}

type createKeyResponse struct {
	KeyMetadata keyMetadata `json:"KeyMetadata"`
}

type describeKeyRequest struct {
	KeyId string `json:"KeyId"`
}

type describeKeyResponse struct {
	KeyMetadata keyMetadata `json:"KeyMetadata"`
}

type listKeysResponse struct {
	Keys      []keyEntry `json:"Keys"`
	Truncated bool       `json:"Truncated"`
}

type keyEntry struct {
	KeyId  string `json:"KeyId"`
	KeyArn string `json:"KeyArn"`
}

type encryptRequest struct {
	KeyId     string `json:"KeyId"`
	Plaintext string `json:"Plaintext"` // base64-encoded
}

type encryptResponse struct {
	CiphertextBlob string `json:"CiphertextBlob"` // base64-encoded
	KeyId          string `json:"KeyId"`
}

type decryptRequest struct {
	CiphertextBlob string `json:"CiphertextBlob"` // base64-encoded
}

type decryptResponse struct {
	Plaintext string `json:"Plaintext"` // base64-encoded
	KeyId     string `json:"KeyId"`
}

type createAliasRequest struct {
	AliasName   string `json:"AliasName"`
	TargetKeyId string `json:"TargetKeyId"`
}

type listAliasesResponse struct {
	Aliases   []aliasEntry `json:"Aliases"`
	Truncated bool         `json:"Truncated"`
}

type aliasEntry struct {
	AliasName    string `json:"AliasName"`
	AliasArn     string `json:"AliasArn"`
	TargetKeyId  string `json:"TargetKeyId"`
}

type enableDisableKeyRequest struct {
	KeyId string `json:"KeyId"`
}

type scheduleKeyDeletionRequest struct {
	KeyId               string `json:"KeyId"`
	PendingWindowInDays int    `json:"PendingWindowInDays"`
}

type scheduleKeyDeletionResponse struct {
	KeyId        string  `json:"KeyId"`
	DeletionDate float64 `json:"DeletionDate"`
}

// ---- helpers ----

func keyToMetadata(k *Key) keyMetadata {
	return keyMetadata{
		KeyId:        k.KeyId,
		Arn:          k.Arn,
		Description:  k.Description,
		KeyState:     string(k.KeyState),
		KeyUsage:     k.KeyUsage,
		CreationDate: float64(k.CreationDate.Unix()),
		Enabled:      k.KeyState == keyStateEnabled,
	}
}

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil // empty body is OK for some actions
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- handlers ----

func handleCreateKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	if req.KeyUsage == "" {
		req.KeyUsage = "ENCRYPT_DECRYPT"
	}
	if req.KeySpec == "" {
		switch req.KeyUsage {
		case "GENERATE_VERIFY_MAC":
			req.KeySpec = "HMAC_256"
		case "SIGN_VERIFY":
			req.KeySpec = "RSA_2048"
		default:
			req.KeySpec = "SYMMETRIC_DEFAULT"
		}
	}

	key, err := store.CreateKey(req.Description, req.KeyUsage, req.KeySpec)
	if err != nil {
		return jsonErr(service.NewAWSError("KMSInternalException",
			"Failed to create key.", http.StatusInternalServerError))
	}

	return jsonOK(createKeyResponse{KeyMetadata: keyToMetadata(key)})
}

func handleDescribeKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId is required.", http.StatusBadRequest))
	}

	key, awsErr := store.DescribeKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(describeKeyResponse{KeyMetadata: keyToMetadata(key)})
}

func handleListKeys(_ *service.RequestContext, store *Store) (*service.Response, error) {
	keys := store.ListKeys()
	entries := make([]keyEntry, 0, len(keys))
	for _, k := range keys {
		entries = append(entries, keyEntry{KeyId: k.KeyId, KeyArn: k.Arn})
	}
	return jsonOK(listKeysResponse{Keys: entries, Truncated: false})
}

func handleEncrypt(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req encryptRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId is required.", http.StatusBadRequest))
	}
	if req.Plaintext == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"Plaintext is required.", http.StatusBadRequest))
	}

	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Plaintext must be base64-encoded.", http.StatusBadRequest))
	}

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException",
			"KMS key is disabled.", http.StatusBadRequest))
	}

	blob, awsErr := Encrypt(key, plaintext)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(encryptResponse{
		CiphertextBlob: base64.StdEncoding.EncodeToString(blob),
		KeyId:          key.Arn,
	})
}

func handleDecrypt(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req decryptRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.CiphertextBlob == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"CiphertextBlob is required.", http.StatusBadRequest))
	}

	blob, err := base64.StdEncoding.DecodeString(req.CiphertextBlob)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidCiphertextException",
			"CiphertextBlob must be base64-encoded.", http.StatusBadRequest))
	}

	keyID, awsErr := ExtractKeyID(blob)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	key, awsErr := store.GetKey(keyID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException",
			"KMS key is disabled.", http.StatusBadRequest))
	}

	plaintext, awsErr := Decrypt(key, blob)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(decryptResponse{
		Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		KeyId:     key.Arn,
	})
}

func handleCreateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AliasName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"AliasName is required.", http.StatusBadRequest))
	}
	if !strings.HasPrefix(req.AliasName, "alias/") {
		return jsonErr(service.NewAWSError("ValidationException",
			"AliasName must begin with 'alias/'.", http.StatusBadRequest))
	}
	if req.TargetKeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TargetKeyId is required.", http.StatusBadRequest))
	}

	if awsErr := store.CreateAlias(req.AliasName, req.TargetKeyId); awsErr != nil {
		return jsonErr(awsErr)
	}

	// CreateAlias returns an empty 200 body.
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

func handleListAliases(_ *service.RequestContext, store *Store) (*service.Response, error) {
	aliases := store.ListAliases()
	entries := make([]aliasEntry, 0, len(aliases))
	for _, a := range aliases {
		entries = append(entries, aliasEntry{
			AliasName:   a.AliasName,
			AliasArn:    a.AliasArn,
			TargetKeyId: a.TargetKeyId,
		})
	}
	return jsonOK(listAliasesResponse{Aliases: entries, Truncated: false})
}

func handleEnableKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableDisableKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId is required.", http.StatusBadRequest))
	}
	if awsErr := store.EnableKey(req.KeyId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleDisableKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableDisableKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId is required.", http.StatusBadRequest))
	}
	if awsErr := store.DisableKey(req.KeyId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleScheduleKeyDeletion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req scheduleKeyDeletionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId is required.", http.StatusBadRequest))
	}

	pendingDays := req.PendingWindowInDays
	if pendingDays <= 0 {
		pendingDays = 30
	}

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	if awsErr := store.ScheduleKeyDeletion(req.KeyId); awsErr != nil {
		return jsonErr(awsErr)
	}

	deletionDate := time.Now().UTC().AddDate(0, 0, pendingDays)
	return jsonOK(scheduleKeyDeletionResponse{
		KeyId:        key.KeyId,
		DeletionDate: float64(deletionDate.Unix()),
	})
}

// ── GenerateDataKey ─────────────────────────────────────────────────────────

type generateDataKeyRequest struct {
	KeyId         string `json:"KeyId"`
	NumberOfBytes int    `json:"NumberOfBytes"`
	KeySpec       string `json:"KeySpec"` // "AES_256" or "AES_128"
}

type generateDataKeyResponse struct {
	CiphertextBlob string `json:"CiphertextBlob"`
	Plaintext      string `json:"Plaintext"`
	KeyId          string `json:"KeyId"`
}

func handleGenerateDataKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req generateDataKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException", "KeyId is required.", http.StatusBadRequest))
	}

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException", "KMS key is disabled.", http.StatusBadRequest))
	}
	if key.KeySpec != keySpecSymmetric256 {
		return jsonErr(service.NewAWSError("InvalidKeyUsageException",
			"GenerateDataKey is only supported for symmetric encryption keys.", http.StatusBadRequest))
	}

	numBytes := req.NumberOfBytes
	if numBytes == 0 {
		switch req.KeySpec {
		case "AES_128":
			numBytes = 16
		default:
			numBytes = 32 // AES_256 default
		}
	}

	plaintext, ciphertextBlob, awsErr := GenerateDataKey(key, numBytes)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(generateDataKeyResponse{
		CiphertextBlob: base64.StdEncoding.EncodeToString(ciphertextBlob),
		Plaintext:      base64.StdEncoding.EncodeToString(plaintext),
		KeyId:          key.Arn,
	})
}

func handleGenerateDataKeyWithoutPlaintext(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req generateDataKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException", "KeyId is required.", http.StatusBadRequest))
	}

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException", "KMS key is disabled.", http.StatusBadRequest))
	}

	numBytes := req.NumberOfBytes
	if numBytes == 0 {
		numBytes = 32
	}

	_, ciphertextBlob, awsErr := GenerateDataKey(key, numBytes)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"CiphertextBlob": base64.StdEncoding.EncodeToString(ciphertextBlob),
		"KeyId":          key.Arn,
	})
}

// ── HMAC Operations ─────────────────────────────────────────────────────────

type generateMacRequest struct {
	KeyId        string `json:"KeyId"`
	Message      string `json:"Message"` // base64
	MacAlgorithm string `json:"MacAlgorithm"`
}

func handleGenerateMac(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req generateMacRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" || req.Message == "" || req.MacAlgorithm == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId, Message, and MacAlgorithm are required.", http.StatusBadRequest))
	}

	message, err := base64.StdEncoding.DecodeString(req.Message)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Message must be base64-encoded.", http.StatusBadRequest))
	}

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException", "KMS key is disabled.", http.StatusBadRequest))
	}

	mac, awsErr := GenerateMac(key, message, req.MacAlgorithm)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"Mac":          base64.StdEncoding.EncodeToString(mac),
		"MacAlgorithm": req.MacAlgorithm,
		"KeyId":        key.Arn,
	})
}

type verifyMacRequest struct {
	KeyId        string `json:"KeyId"`
	Message      string `json:"Message"` // base64
	Mac          string `json:"Mac"`     // base64
	MacAlgorithm string `json:"MacAlgorithm"`
}

func handleVerifyMac(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req verifyMacRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" || req.Message == "" || req.Mac == "" || req.MacAlgorithm == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId, Message, Mac, and MacAlgorithm are required.", http.StatusBadRequest))
	}

	message, _ := base64.StdEncoding.DecodeString(req.Message)
	mac, _ := base64.StdEncoding.DecodeString(req.Mac)

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	valid, awsErr := VerifyMac(key, message, mac, req.MacAlgorithm)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"MacValid":     valid,
		"MacAlgorithm": req.MacAlgorithm,
		"KeyId":        key.Arn,
	})
}

// ── Sign/Verify ─────────────────────────────────────────────────────────────

type signRequest struct {
	KeyId            string `json:"KeyId"`
	Message          string `json:"Message"` // base64
	SigningAlgorithm string `json:"SigningAlgorithm"`
	MessageType      string `json:"MessageType"` // "RAW" or "DIGEST"
}

func handleSign(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req signRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" || req.Message == "" || req.SigningAlgorithm == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId, Message, and SigningAlgorithm are required.", http.StatusBadRequest))
	}

	message, _ := base64.StdEncoding.DecodeString(req.Message)

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if key.KeyState != keyStateEnabled {
		return jsonErr(service.NewAWSError("DisabledException", "KMS key is disabled.", http.StatusBadRequest))
	}

	sig, awsErr := Sign(key, message, req.SigningAlgorithm)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"Signature":        base64.StdEncoding.EncodeToString(sig),
		"SigningAlgorithm": req.SigningAlgorithm,
		"KeyId":            key.Arn,
	})
}

type verifyRequest struct {
	KeyId            string `json:"KeyId"`
	Message          string `json:"Message"`   // base64
	Signature        string `json:"Signature"` // base64
	SigningAlgorithm string `json:"SigningAlgorithm"`
	MessageType      string `json:"MessageType"`
}

func handleVerify(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req verifyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" || req.Message == "" || req.Signature == "" || req.SigningAlgorithm == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyId, Message, Signature, and SigningAlgorithm are required.", http.StatusBadRequest))
	}

	message, _ := base64.StdEncoding.DecodeString(req.Message)
	sig, _ := base64.StdEncoding.DecodeString(req.Signature)

	key, awsErr := store.GetKey(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	valid, awsErr := Verify(key, message, sig, req.SigningAlgorithm)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"SignatureValid":   valid,
		"SigningAlgorithm": req.SigningAlgorithm,
		"KeyId":            key.Arn,
	})
}

// ── Key Rotation ────────────────────────────────────────────────────────────

func handleEnableKeyRotation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableDisableKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException", "KeyId is required.", http.StatusBadRequest))
	}
	if awsErr := store.EnableKeyRotation(req.KeyId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleDisableKeyRotation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableDisableKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException", "KeyId is required.", http.StatusBadRequest))
	}
	if awsErr := store.DisableKeyRotation(req.KeyId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleGetKeyRotationStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req enableDisableKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.KeyId == "" {
		return jsonErr(service.NewAWSError("ValidationException", "KeyId is required.", http.StatusBadRequest))
	}
	enabled, awsErr := store.GetKeyRotationStatus(req.KeyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"KeyRotationEnabled": enabled})
}
