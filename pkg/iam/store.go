package iam

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// User represents an IAM user in the store.
type User struct {
	Name     string
	ARN      string
	UserID   string
	Policies map[string]*Policy
}

// AccessKey represents an IAM access key and its associated identity.
type AccessKey struct {
	AccessKeyID     string
	SecretAccessKey string
	UserName        string
	AccountID       string
	IsRoot          bool
}

// Store holds IAM users and access keys for a single AWS account.
type Store struct {
	mu         sync.RWMutex
	accountID  string
	users      map[string]*User
	accessKeys map[string]*AccessKey
}

// NewStore creates a new Store for the given AWS account ID.
func NewStore(accountID string) *Store {
	return &Store{
		accountID:  accountID,
		users:      make(map[string]*User),
		accessKeys: make(map[string]*AccessKey),
	}
}

// InitRoot creates a root access key entry for the account.
func (s *Store) InitRoot(accessKeyID, secretKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.accessKeys[accessKeyID] = &AccessKey{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretKey,
		UserName:        "root",
		AccountID:       s.accountID,
		IsRoot:          true,
	}
	return nil
}

// CreateUser creates a new IAM user with the given name.
func (s *Store) CreateUser(name string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, err := generateID("AIDA", 16)
	if err != nil {
		return nil, fmt.Errorf("generating user ID: %w", err)
	}

	user := &User{
		Name:     name,
		ARN:      fmt.Sprintf("arn:aws:iam::%s:user/%s", s.accountID, name),
		UserID:   userID,
		Policies: make(map[string]*Policy),
	}
	s.users[name] = user
	return user, nil
}

// GetUser returns the named user or an error if not found.
func (s *Store) GetUser(name string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[name]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", name)
	}
	return user, nil
}

// CreateAccessKey generates a new access key for the named user.
func (s *Store) CreateAccessKey(userName string) (*AccessKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userName]; !ok {
		return nil, fmt.Errorf("user not found: %s", userName)
	}

	keyID, err := generateID("AKIA", 16)
	if err != nil {
		return nil, fmt.Errorf("generating access key ID: %w", err)
	}

	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generating secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	key := &AccessKey{
		AccessKeyID:     keyID,
		SecretAccessKey: secret,
		UserName:        userName,
		AccountID:       s.accountID,
		IsRoot:          false,
	}
	s.accessKeys[keyID] = key
	return key, nil
}

// LookupAccessKey returns the AccessKey for the given access key ID.
func (s *Store) LookupAccessKey(accessKeyID string) (*AccessKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.accessKeys[accessKeyID]
	if !ok {
		return nil, fmt.Errorf("access key not found: %s", accessKeyID)
	}
	return key, nil
}

// AttachUserPolicy attaches a named policy to the given user.
func (s *Store) AttachUserPolicy(userName, policyName string, policy *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userName]
	if !ok {
		return fmt.Errorf("user not found: %s", userName)
	}
	user.Policies[policyName] = policy
	return nil
}

// GetUserPolicies returns all policies attached to the named user.
func (s *Store) GetUserPolicies(userName string) ([]*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[userName]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", userName)
	}

	policies := make([]*Policy, 0, len(user.Policies))
	for _, p := range user.Policies {
		policies = append(policies, p)
	}
	return policies, nil
}

// generateID creates a random ID with the given prefix using hex-encoded random bytes.
func generateID(prefix string, hexLen int) (string, error) {
	b := make([]byte, hexLen/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + strings.ToUpper(hex.EncodeToString(b)), nil
}
