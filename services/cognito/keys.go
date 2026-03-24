package cognito

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"sync"
)

// KeyStore holds an RSA key pair used to sign JWTs.
type KeyStore struct {
	mu         sync.RWMutex
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
}

// NewKeyStore generates a fresh RSA-2048 key pair and a random Key ID.
func NewKeyStore() (*KeyStore, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	kid := randomKID()

	return &KeyStore{
		privateKey: priv,
		publicKey:  &priv.PublicKey,
		kid:        kid,
	}, nil
}

// KID returns the Key ID.
func (ks *KeyStore) KID() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.kid
}

// PrivateKey returns the RSA private key.
func (ks *KeyStore) PrivateKey() *rsa.PrivateKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.privateKey
}

// PublicKey returns the RSA public key.
func (ks *KeyStore) PublicKey() *rsa.PublicKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.publicKey
}

// JWK returns the public key in JWK format.
func (ks *KeyStore) JWK() map[string]any {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	return map[string]any{
		"kty": "RSA",
		"kid": ks.kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64URLEncodeBigInt(ks.publicKey.N),
		"e":   base64URLEncodeBigInt(big.NewInt(int64(ks.publicKey.E))),
	}
}

// base64URLEncodeBigInt encodes a big.Int as base64url without padding.
func base64URLEncodeBigInt(n *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(n.Bytes())
}

// randomKID generates a random key ID string.
func randomKID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
