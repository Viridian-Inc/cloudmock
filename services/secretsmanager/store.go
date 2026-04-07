package secretsmanager

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Secret holds all metadata and value for a Secrets Manager secret.
type Secret struct {
	Name            string
	ARN             string
	Description     string
	SecretString    string
	SecretBinary    []byte
	VersionId       string
	VersionStages   []string
	CreatedDate     time.Time
	LastChangedDate time.Time
	DeletedDate     *time.Time
	Tags            map[string]string
	IsDeleted       bool
}

// Store is the in-memory store for Secrets Manager secrets.
type Store struct {
	mu        sync.RWMutex
	secrets   map[string]*Secret // keyed by name
	arnToName map[string]string  // ARN → name index
	accountID string
	region    string
}

// NewStore creates an empty Secrets Manager Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		secrets:   make(map[string]*Secret),
		arnToName: make(map[string]string),
		accountID: accountID,
		region:    region,
	}
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

// randomSuffix returns 6 random lowercase alphanumeric characters.
func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// buildARN constructs an ARN for a secret.
func (s *Store) buildARN(name string) string {
	return fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s-%s",
		s.region, s.accountID, name, randomSuffix())
}

// resolve looks up a secret by name or ARN (caller must hold at least read lock).
// Returns nil if not found.
func (s *Store) resolveLocked(secretID string) *Secret {
	// Try by name first.
	if sec, ok := s.secrets[secretID]; ok {
		return sec
	}
	// Try by ARN.
	if name, ok := s.arnToName[secretID]; ok {
		return s.secrets[name]
	}
	return nil
}

// CreateSecret adds a new secret to the store.
func (s *Store) CreateSecret(name, description, secretString string, secretBinary []byte, tags map[string]string) (*Secret, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.secrets[name]; ok && !existing.IsDeleted {
		return nil, service.NewAWSError("ResourceExistsException",
			fmt.Sprintf("The operation failed because the secret %s already exists.", name),
			http.StatusBadRequest)
	}

	now := time.Now().UTC()
	sec := &Secret{
		Name:            name,
		Description:     description,
		SecretString:    secretString,
		SecretBinary:    secretBinary,
		VersionId:       newUUID(),
		VersionStages:   []string{"AWSCURRENT"},
		CreatedDate:     now,
		LastChangedDate: now,
		Tags:            tags,
		IsDeleted:       false,
	}
	sec.ARN = s.buildARN(name)
	s.secrets[name] = sec
	s.arnToName[sec.ARN] = name
	return sec, nil
}

// GetSecret retrieves a secret by name or ARN.
func (s *Store) GetSecret(secretID string) (*Secret, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}
	return sec, nil
}

// PutSecretValue creates a new version of the secret value.
func (s *Store) PutSecretValue(secretID, secretString string, secretBinary []byte) (*Secret, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}

	sec.SecretString = secretString
	sec.SecretBinary = secretBinary
	sec.VersionId = newUUID()
	sec.VersionStages = []string{"AWSCURRENT"}
	sec.LastChangedDate = time.Now().UTC()
	return sec, nil
}

// UpdateSecret updates description and/or secret value.
func (s *Store) UpdateSecret(secretID, description, secretString string) (*Secret, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}

	now := time.Now().UTC()
	if description != "" {
		sec.Description = description
	}
	if secretString != "" {
		sec.SecretString = secretString
		sec.VersionId = newUUID()
		sec.VersionStages = []string{"AWSCURRENT"}
	}
	sec.LastChangedDate = now
	return sec, nil
}

// DeleteSecret marks a secret as deleted.
func (s *Store) DeleteSecret(secretID string, forceDelete bool) (*Secret, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}

	now := time.Now().UTC()
	var deletionDate time.Time
	if forceDelete {
		deletionDate = now
	} else {
		deletionDate = now.AddDate(0, 0, 30)
	}
	sec.IsDeleted = true
	sec.DeletedDate = &deletionDate
	return sec, nil
}

// RestoreSecret undeletes a secret.
func (s *Store) RestoreSecret(secretID string) (*Secret, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}
	sec.IsDeleted = false
	sec.DeletedDate = nil
	return sec, nil
}

// DescribeSecret returns metadata for a secret.
func (s *Store) DescribeSecret(secretID string) (*Secret, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}
	return sec, nil
}

// ListSecrets returns all non-deleted secrets.
func (s *Store) ListSecrets() []*Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Secret, 0, len(s.secrets))
	for _, sec := range s.secrets {
		if !sec.IsDeleted {
			out = append(out, sec)
		}
	}
	return out
}

// TagResource adds or replaces tags on a secret.
func (s *Store) TagResource(secretID string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}
	if sec.Tags == nil {
		sec.Tags = make(map[string]string)
	}
	for k, v := range tags {
		sec.Tags[k] = v
	}
	return nil
}

// UntagResource removes tags from a secret.
func (s *Store) UntagResource(secretID string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := s.resolveLocked(secretID)
	if sec == nil || sec.IsDeleted {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Secrets Manager can't find the specified secret. (%s)", secretID),
			http.StatusBadRequest)
	}
	for _, k := range tagKeys {
		delete(sec.Tags, k)
	}
	return nil
}
