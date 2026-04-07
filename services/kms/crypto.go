package kms

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const (
	uuidLen  = 36 // length of a UUID string
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

// generateKeyMaterial populates key material based on the key spec.
func generateKeyMaterial(key *Key) error {
	switch key.KeySpec {
	case keySpecSymmetric256, "":
		key.KeySpec = keySpecSymmetric256
		aesKey, err := generateAESKey()
		if err != nil {
			return err
		}
		key.AESKey = aesKey

	case keySpecHMAC256:
		key.HMACKey = make([]byte, 32)
		if _, err := rand.Read(key.HMACKey); err != nil {
			return fmt.Errorf("kms: generate HMAC-256 key: %w", err)
		}
	case keySpecHMAC384:
		key.HMACKey = make([]byte, 48)
		if _, err := rand.Read(key.HMACKey); err != nil {
			return fmt.Errorf("kms: generate HMAC-384 key: %w", err)
		}
	case keySpecHMAC512:
		key.HMACKey = make([]byte, 64)
		if _, err := rand.Read(key.HMACKey); err != nil {
			return fmt.Errorf("kms: generate HMAC-512 key: %w", err)
		}

	case keySpecRSA2048:
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("kms: generate RSA-2048 key: %w", err)
		}
		key.RSAPrivateKey = priv
	case keySpecRSA3072:
		priv, err := rsa.GenerateKey(rand.Reader, 3072)
		if err != nil {
			return fmt.Errorf("kms: generate RSA-3072 key: %w", err)
		}
		key.RSAPrivateKey = priv
	case keySpecRSA4096:
		priv, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return fmt.Errorf("kms: generate RSA-4096 key: %w", err)
		}
		key.RSAPrivateKey = priv

	case keySpecECCNistP256:
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return fmt.Errorf("kms: generate ECC P256 key: %w", err)
		}
		key.ECCPrivateKey = priv
	case keySpecECCNistP384:
		priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return fmt.Errorf("kms: generate ECC P384 key: %w", err)
		}
		key.ECCPrivateKey = priv

	default:
		return fmt.Errorf("kms: unsupported key spec: %s", key.KeySpec)
	}
	return nil
}

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ── Symmetric Encrypt/Decrypt (AES-256-GCM) ─────────────────────────────────

// Encrypt encrypts plaintext using the key's AES-256-GCM key.
// Ciphertext blob: keyID (36 bytes) + nonce (12 bytes) + GCM ciphertext.
func Encrypt(key *Key, plaintext []byte) ([]byte, *service.AWSError) {
	block, err := aes.NewCipher(key.AESKey[:])
	if err != nil {
		return nil, internalErr("Failed to create AES cipher.")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, internalErr("Failed to create GCM cipher.")
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, internalErr("Failed to generate nonce.")
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	blob := make([]byte, uuidLen+nonceLen+len(ciphertext))
	copy(blob[:uuidLen], []byte(key.KeyId))
	copy(blob[uuidLen:uuidLen+nonceLen], nonce)
	copy(blob[uuidLen+nonceLen:], ciphertext)
	return blob, nil
}

// Decrypt decrypts a ciphertext blob produced by Encrypt.
func Decrypt(key *Key, blob []byte) ([]byte, *service.AWSError) {
	if len(blob) < uuidLen+nonceLen+1 {
		return nil, invalidCiphertext()
	}
	nonce := blob[uuidLen : uuidLen+nonceLen]
	ciphertext := blob[uuidLen+nonceLen:]

	block, err := aes.NewCipher(key.AESKey[:])
	if err != nil {
		return nil, internalErr("Failed to create AES cipher.")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, internalErr("Failed to create GCM cipher.")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, invalidCiphertext()
	}
	return plaintext, nil
}

// ExtractKeyID extracts the KeyId embedded at the start of a ciphertext blob.
func ExtractKeyID(blob []byte) (string, *service.AWSError) {
	if len(blob) < uuidLen {
		return "", invalidCiphertext()
	}
	return string(blob[:uuidLen]), nil
}

// ── GenerateDataKey ─────────────────────────────────────────────────────────

// GenerateDataKey generates a random data encryption key, returns it both
// as plaintext and encrypted under the given KMS key. This is how envelope
// encryption works — the data key encrypts your data, and the KMS key
// encrypts the data key.
func GenerateDataKey(key *Key, numberOfBytes int) (plaintext, ciphertextBlob []byte, awsErr *service.AWSError) {
	if numberOfBytes <= 0 {
		numberOfBytes = 32 // AES-256
	}
	if numberOfBytes > 1024 {
		return nil, nil, service.NewAWSError("ValidationException",
			"NumberOfBytes must be between 1 and 1024.", http.StatusBadRequest)
	}

	dataKey := make([]byte, numberOfBytes)
	if _, err := rand.Read(dataKey); err != nil {
		return nil, nil, internalErr("Failed to generate data key.")
	}

	encrypted, awsErr := Encrypt(key, dataKey)
	if awsErr != nil {
		return nil, nil, awsErr
	}

	return dataKey, encrypted, nil
}

// ── HMAC Operations (GenerateMac / VerifyMac) ───────────────────────────────

// GenerateMac computes an HMAC using the key's HMAC material.
func GenerateMac(key *Key, message []byte, algorithm string) ([]byte, *service.AWSError) {
	if len(key.HMACKey) == 0 {
		return nil, service.NewAWSError("InvalidKeyUsageException",
			"This key does not support HMAC operations.", http.StatusBadRequest)
	}
	h := hmacHash(key, algorithm)
	if h == nil {
		return nil, service.NewAWSError("InvalidParameterException",
			"Unsupported MAC algorithm: "+algorithm, http.StatusBadRequest)
	}
	h.Write(message)
	return h.Sum(nil), nil
}

// VerifyMac verifies an HMAC tag against a message.
func VerifyMac(key *Key, message, mac []byte, algorithm string) (bool, *service.AWSError) {
	expected, awsErr := GenerateMac(key, message, algorithm)
	if awsErr != nil {
		return false, awsErr
	}
	return hmac.Equal(expected, mac), nil
}

func hmacHash(key *Key, algorithm string) hash.Hash {
	switch algorithm {
	case "HMAC_SHA_256":
		return hmac.New(sha256.New, key.HMACKey)
	case "HMAC_SHA_384":
		return hmac.New(sha512.New384, key.HMACKey)
	case "HMAC_SHA_512":
		return hmac.New(sha512.New, key.HMACKey)
	default:
		return nil
	}
}

// ── RSA Sign/Verify ─────────────────────────────────────────────────────────

// Sign produces a digital signature using the key's RSA or ECC private key.
func Sign(key *Key, message []byte, algorithm string) ([]byte, *service.AWSError) {
	hashed, hashFunc, awsErr := hashForSigning(message, algorithm)
	if awsErr != nil {
		return nil, awsErr
	}

	if key.RSAPrivateKey != nil {
		priv := key.RSAPrivateKey.(*rsa.PrivateKey)
		if isRSAPSS(algorithm) {
			sig, err := rsa.SignPSS(rand.Reader, priv, hashFunc, hashed, nil)
			if err != nil {
				return nil, internalErr("RSA-PSS sign failed: " + err.Error())
			}
			return sig, nil
		}
		sig, err := rsa.SignPKCS1v15(rand.Reader, priv, hashFunc, hashed)
		if err != nil {
			return nil, internalErr("RSA sign failed: " + err.Error())
		}
		return sig, nil
	}

	if key.ECCPrivateKey != nil {
		priv := key.ECCPrivateKey.(*ecdsa.PrivateKey)
		sig, err := ecdsa.SignASN1(rand.Reader, priv, hashed)
		if err != nil {
			return nil, internalErr("ECDSA sign failed: " + err.Error())
		}
		return sig, nil
	}

	return nil, service.NewAWSError("InvalidKeyUsageException",
		"This key does not support signing.", http.StatusBadRequest)
}

// Verify checks a digital signature against a message.
func Verify(key *Key, message, signature []byte, algorithm string) (bool, *service.AWSError) {
	hashed, hashFunc, awsErr := hashForSigning(message, algorithm)
	if awsErr != nil {
		return false, awsErr
	}

	if key.RSAPrivateKey != nil {
		pub := &key.RSAPrivateKey.(*rsa.PrivateKey).PublicKey
		if isRSAPSS(algorithm) {
			err := rsa.VerifyPSS(pub, hashFunc, hashed, signature, nil)
			return err == nil, nil
		}
		err := rsa.VerifyPKCS1v15(pub, hashFunc, hashed, signature)
		return err == nil, nil
	}

	if key.ECCPrivateKey != nil {
		pub := &key.ECCPrivateKey.(*ecdsa.PrivateKey).PublicKey
		return ecdsa.VerifyASN1(pub, hashed, signature), nil
	}

	return false, service.NewAWSError("InvalidKeyUsageException",
		"This key does not support verification.", http.StatusBadRequest)
}

func hashForSigning(message []byte, algorithm string) ([]byte, crypto.Hash, *service.AWSError) {
	switch {
	case contains(algorithm, "SHA_256"):
		h := sha256.Sum256(message)
		return h[:], crypto.SHA256, nil
	case contains(algorithm, "SHA_384"):
		h := sha512.Sum384(message)
		return h[:], crypto.SHA384, nil
	case contains(algorithm, "SHA_512"):
		h := sha512.Sum512(message)
		return h[:], crypto.SHA512, nil
	default:
		return nil, 0, service.NewAWSError("InvalidParameterException",
			"Unsupported signing algorithm: "+algorithm, http.StatusBadRequest)
	}
}

func isRSAPSS(algorithm string) bool {
	return contains(algorithm, "RSASSA_PSS")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── Error helpers ────────────────────────────────────────────────────────────

func internalErr(msg string) *service.AWSError {
	return service.NewAWSError("KMSInternalException", msg, http.StatusInternalServerError)
}

func invalidCiphertext() *service.AWSError {
	return service.NewAWSError("InvalidCiphertextException",
		"The ciphertext is invalid or has been tampered with.", http.StatusBadRequest)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
