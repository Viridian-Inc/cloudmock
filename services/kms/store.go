package kms

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// keyState represents the lifecycle state of a KMS key.
type keyState string

const (
	keyStateEnabled         keyState = "Enabled"
	keyStateDisabled        keyState = "Disabled"
	keyStatePendingDeletion keyState = "PendingDeletion"
)

// keySpec describes the type and size of key material.
type keySpec string

const (
	keySpecSymmetric256 keySpec = "SYMMETRIC_DEFAULT"
	keySpecHMAC256      keySpec = "HMAC_256"
	keySpecHMAC384      keySpec = "HMAC_384"
	keySpecHMAC512      keySpec = "HMAC_512"
	keySpecRSA2048      keySpec = "RSA_2048"
	keySpecRSA3072      keySpec = "RSA_3072"
	keySpecRSA4096      keySpec = "RSA_4096"
	keySpecECCNistP256  keySpec = "ECC_NIST_P256"
	keySpecECCNistP384  keySpec = "ECC_NIST_P384"
)

// Key holds all metadata and key material for a KMS key.
type Key struct {
	KeyId           string
	Arn             string
	Description     string
	KeyState        keyState
	KeyUsage        string
	KeySpec         keySpec
	CreationDate    time.Time
	RotationEnabled bool
	AESKey          [32]byte    // For SYMMETRIC_DEFAULT
	HMACKey         []byte      // For HMAC_* key specs
	RSAPrivateKey   interface{} // *rsa.PrivateKey for RSA_* key specs
	ECCPrivateKey   interface{} // *ecdsa.PrivateKey for ECC_* key specs
}

// Alias maps an alias name to a target key.
type Alias struct {
	AliasName  string
	TargetKeyId string
	AliasArn   string
}

// Store is the in-memory store for KMS keys and aliases.
type Store struct {
	mu        sync.RWMutex
	keys      map[string]*Key   // keyed by KeyId
	aliases   map[string]*Alias // keyed by AliasName
	accountID string
	region    string
}

// NewStore creates an empty KMS Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		keys:      make(map[string]*Key),
		aliases:   make(map[string]*Alias),
		accountID: accountID,
		region:    region,
	}
}

// buildArn constructs an ARN for a KMS key.
func (s *Store) buildArn(keyID string) string {
	return fmt.Sprintf("arn:aws:kms:%s:%s:key/%s", s.region, s.accountID, keyID)
}

// buildAliasArn constructs an ARN for a KMS alias.
func (s *Store) buildAliasArn(aliasName string) string {
	return fmt.Sprintf("arn:aws:kms:%s:%s:%s", s.region, s.accountID, aliasName)
}

// CreateKey adds a new key to the store and returns it.
func (s *Store) CreateKey(description, keyUsage, spec string) (*Key, error) {
	keyID := newUUID()
	key := &Key{
		KeyId:        keyID,
		Arn:          s.buildArn(keyID),
		Description:  description,
		KeyState:     keyStateEnabled,
		KeyUsage:     keyUsage,
		KeySpec:      keySpec(spec),
		CreationDate: time.Now().UTC(),
	}

	if err := generateKeyMaterial(key); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.keys[keyID] = key
	s.mu.Unlock()

	return key, nil
}

// EnableKeyRotation enables automatic key rotation.
func (s *Store) EnableKeyRotation(keyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return awsErr
	}
	if key.KeySpec != keySpecSymmetric256 {
		return service.NewAWSError("UnsupportedOperationException",
			"Key rotation is only supported for symmetric keys.", http.StatusBadRequest)
	}
	key.RotationEnabled = true
	return nil
}

// DisableKeyRotation disables automatic key rotation.
func (s *Store) DisableKeyRotation(keyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return awsErr
	}
	key.RotationEnabled = false
	return nil
}

// GetKeyRotationStatus returns whether key rotation is enabled.
func (s *Store) GetKeyRotationStatus(keyID string) (bool, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return false, awsErr
	}
	return key.RotationEnabled, nil
}

// GetKey looks up a key by KeyId or alias name. Returns AWSError if not found or not usable.
func (s *Store) GetKey(keyID string) (*Key, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getKeyLocked(keyID)
}

// getKeyLocked resolves a KeyId or alias (caller holds at least read lock).
func (s *Store) getKeyLocked(keyID string) (*Key, *service.AWSError) {
	// Resolve alias first.
	if alias, ok := s.aliases[keyID]; ok {
		keyID = alias.TargetKeyId
	}
	key, ok := s.keys[keyID]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid keyId %s", keyID), http.StatusBadRequest)
	}
	return key, nil
}

// DescribeKey returns the key metadata.
func (s *Store) DescribeKey(keyID string) (*Key, *service.AWSError) {
	return s.GetKey(keyID)
}

// ListKeys returns a snapshot of all keys.
func (s *Store) ListKeys() []*Key {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Key, 0, len(s.keys))
	for _, k := range s.keys {
		out = append(out, k)
	}
	return out
}

// EnableKey sets the key state to Enabled.
func (s *Store) EnableKey(keyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return awsErr
	}
	if key.KeyState == keyStatePendingDeletion {
		return service.NewAWSError("KMSInvalidStateException",
			"KMS key is pending deletion.", http.StatusBadRequest)
	}
	key.KeyState = keyStateEnabled
	return nil
}

// DisableKey sets the key state to Disabled.
func (s *Store) DisableKey(keyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return awsErr
	}
	if key.KeyState == keyStatePendingDeletion {
		return service.NewAWSError("KMSInvalidStateException",
			"KMS key is pending deletion.", http.StatusBadRequest)
	}
	key.KeyState = keyStateDisabled
	return nil
}

// ScheduleKeyDeletion marks the key as PendingDeletion.
func (s *Store) ScheduleKeyDeletion(keyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, awsErr := s.getKeyLocked(keyID)
	if awsErr != nil {
		return awsErr
	}
	key.KeyState = keyStatePendingDeletion
	return nil
}

// CreateAlias creates a new alias pointing to targetKeyId.
func (s *Store) CreateAlias(aliasName, targetKeyID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.aliases[aliasName]; ok {
		return service.NewAWSError("AlreadyExistsException",
			fmt.Sprintf("An alias with the name %s already exists in the account", aliasName),
			http.StatusConflict)
	}

	// Verify target key exists.
	if _, ok := s.keys[targetKeyID]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Invalid keyId %s", targetKeyID), http.StatusBadRequest)
	}

	s.aliases[aliasName] = &Alias{
		AliasName:   aliasName,
		TargetKeyId: targetKeyID,
		AliasArn:    s.buildAliasArn(aliasName),
	}
	return nil
}

// ListAliases returns a snapshot of all aliases.
func (s *Store) ListAliases() []*Alias {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Alias, 0, len(s.aliases))
	for _, a := range s.aliases {
		out = append(out, a)
	}
	return out
}
