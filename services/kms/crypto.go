package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

const (
	uuidLen  = 36 // length of a UUID string e.g. "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	nonceLen = 12 // GCM standard nonce size
)

// generateAESKey generates a random 32-byte AES-256 key.
func generateAESKey() ([32]byte, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return key, fmt.Errorf("kms: generate AES key: %w", err)
	}
	return key, nil
}

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Set version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Encrypt encrypts plaintext using the key's AES-256-GCM key.
// The returned ciphertext blob is: keyID (36 bytes) + nonce (12 bytes) + GCM ciphertext.
func Encrypt(key *Key, plaintext []byte) ([]byte, *service.AWSError) {
	block, err := aes.NewCipher(key.AESKey[:])
	if err != nil {
		return nil, service.NewAWSError("KMSInternalException",
			"Failed to create AES cipher.", http.StatusInternalServerError)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, service.NewAWSError("KMSInternalException",
			"Failed to create GCM cipher.", http.StatusInternalServerError)
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, service.NewAWSError("KMSInternalException",
			"Failed to generate nonce.", http.StatusInternalServerError)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Blob layout: keyID (36 ASCII bytes) + nonce (12 bytes) + ciphertext
	blob := make([]byte, uuidLen+nonceLen+len(ciphertext))
	copy(blob[:uuidLen], []byte(key.KeyId))
	copy(blob[uuidLen:uuidLen+nonceLen], nonce)
	copy(blob[uuidLen+nonceLen:], ciphertext)

	return blob, nil
}

// Decrypt decrypts a ciphertext blob produced by Encrypt.
// It extracts the KeyId from the first 36 bytes, then decrypts with the provided key.
func Decrypt(key *Key, blob []byte) ([]byte, *service.AWSError) {
	if len(blob) < uuidLen+nonceLen+1 {
		return nil, service.NewAWSError("InvalidCiphertextException",
			"The ciphertext is invalid.", http.StatusBadRequest)
	}

	nonce := blob[uuidLen : uuidLen+nonceLen]
	ciphertext := blob[uuidLen+nonceLen:]

	block, err := aes.NewCipher(key.AESKey[:])
	if err != nil {
		return nil, service.NewAWSError("KMSInternalException",
			"Failed to create AES cipher.", http.StatusInternalServerError)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, service.NewAWSError("KMSInternalException",
			"Failed to create GCM cipher.", http.StatusInternalServerError)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, service.NewAWSError("InvalidCiphertextException",
			"The ciphertext is invalid or has been tampered with.", http.StatusBadRequest)
	}

	return plaintext, nil
}

// ExtractKeyID extracts the KeyId embedded at the start of a ciphertext blob.
func ExtractKeyID(blob []byte) (string, *service.AWSError) {
	if len(blob) < uuidLen {
		return "", service.NewAWSError("InvalidCiphertextException",
			"The ciphertext is invalid.", http.StatusBadRequest)
	}
	return string(blob[:uuidLen]), nil
}

// randomHex returns n random bytes encoded as hex (for request IDs etc.).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
