package ses

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Email represents a sent email stored in the local mailbox.
type Email struct {
	MessageId   string
	Source      string
	ToAddresses []string
	CcAddresses []string
	BccAddresses []string
	Subject     string
	TextBody    string
	HtmlBody    string
	RawMessage  string
	Timestamp   time.Time
}

// Store holds verified identities and the local email mailbox.
type Store struct {
	mu         sync.RWMutex
	identities map[string]bool // identity → verified (always true in cloudmock)
	emails     []*Email
}

// NewStore returns an initialised Store.
func NewStore() *Store {
	return &Store{
		identities: make(map[string]bool),
		emails:     make([]*Email, 0),
	}
}

// VerifyIdentity adds an identity to the verified set (all identities auto-verify).
func (s *Store) VerifyIdentity(identity string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identities[identity] = true
}

// DeleteIdentity removes an identity. Returns false if not found.
func (s *Store) DeleteIdentity(identity string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.identities[identity] {
		return false
	}
	delete(s.identities, identity)
	return true
}

// ListIdentities returns all verified identities.
func (s *Store) ListIdentities() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.identities))
	for id := range s.identities {
		ids = append(ids, id)
	}
	return ids
}

// ListVerifiedEmailAddresses returns all verified email identities.
// In cloudmock every stored identity is considered verified.
func (s *Store) ListVerifiedEmailAddresses() []string {
	return s.ListIdentities()
}

// StoreEmail appends an email to the mailbox and returns its MessageId.
func (s *Store) StoreEmail(email *Email) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	email.MessageId = newMessageID()
	email.Timestamp = time.Now().UTC()
	s.emails = append(s.emails, email)
	return email.MessageId
}

// GetEmails returns a snapshot of all stored emails (most-recently-added last).
func (s *Store) GetEmails() []*Email {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Email, len(s.emails))
	copy(out, s.emails)
	return out
}

// newMessageID returns a unique SES-style message identifier.
func newMessageID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
