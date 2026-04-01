package acmpca

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

// CAStatus represents the lifecycle state of a private CA.
type CAStatus string

const (
	CAStatusCreating           CAStatus = "CREATING"
	CAStatusPendingCertificate CAStatus = "PENDING_CERTIFICATE"
	CAStatusActive             CAStatus = "ACTIVE"
	CAStatusDisabled           CAStatus = "DISABLED"
	CAStatusExpired            CAStatus = "EXPIRED"
	CAStatusFailed             CAStatus = "FAILED"
	CAStatusDeleted            CAStatus = "DELETED"
)

// RevocationReason represents the reason a certificate was revoked.
type RevocationReason string

const (
	ReasonUnspecified          RevocationReason = "UNSPECIFIED"
	ReasonKeyCompromise        RevocationReason = "KEY_COMPROMISE"
	ReasonAffiliationChanged   RevocationReason = "AFFILIATION_CHANGED"
	ReasonSuperseded           RevocationReason = "SUPERSEDED"
	ReasonCessationOfOperation RevocationReason = "CESSATION_OF_OPERATION"
)

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// CASubject holds the distinguished name fields for a CA.
type CASubject struct {
	Country            string
	Organization       string
	OrganizationalUnit string
	State              string
	Locality           string
	CommonName         string
}

// CertificateAuthority holds all metadata for a private CA.
type CertificateAuthority struct {
	Arn                    string
	Type                   string // ROOT or SUBORDINATE
	ParentCAArn            string // ARN of parent CA for SUBORDINATE type
	KeyAlgorithm           string
	SigningAlgorithm       string
	Subject                CASubject
	Status                 CAStatus
	CreatedAt              time.Time
	LastStateChangeAt      time.Time
	NotBefore              *time.Time
	NotAfter               *time.Time
	Serial                 string
	Tags                   []Tag
	RevocationConfiguration map[string]any
	Lifecycle              *lifecycle.Machine
}

// IssuedCertificate holds metadata for a certificate issued by a private CA.
type IssuedCertificate struct {
	CertificateArn          string
	CertificateBody         string
	CertificateChain        string
	Serial                  string
	IssuedAt                time.Time
	NotBefore               *time.Time
	NotAfter                *time.Time
	Revoked                 bool
	RevokedAt               *time.Time
	RevocationReason        RevocationReason
	CertificateAuthorityArn string
}

// Permission represents an ACM-PCA permission grant.
type Permission struct {
	CertificateAuthorityArn string
	Principal               string
	SourceAccount           string
	Actions                 []string
	Policy                  string
	CreatedAt               time.Time
}

// Store is the in-memory store for ACM-PCA resources.
type Store struct {
	mu          sync.RWMutex
	cas         map[string]*CertificateAuthority // keyed by ARN
	certs       map[string]*IssuedCertificate    // keyed by cert ARN
	permissions map[string][]*Permission         // keyed by CA ARN
	accountID   string
	region      string
	lcConfig    *lifecycle.Config
}

// NewStore creates an empty ACM-PCA Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		cas:         make(map[string]*CertificateAuthority),
		certs:       make(map[string]*IssuedCertificate),
		permissions: make(map[string][]*Permission),
		accountID:   accountID,
		region:      region,
		lcConfig:    lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateSerial() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	s := ""
	for i, v := range b {
		if i > 0 {
			s += ":"
		}
		s += fmt.Sprintf("%02x", v)
	}
	return s
}

func (s *Store) buildCAArn(caID string) string {
	return fmt.Sprintf("arn:aws:acm-pca:%s:%s:certificate-authority/%s", s.region, s.accountID, caID)
}

func (s *Store) buildCertArn(caID, certID string) string {
	return fmt.Sprintf("arn:aws:acm-pca:%s:%s:certificate-authority/%s/certificate/%s", s.region, s.accountID, caID, certID)
}

// CreateCA creates a new private certificate authority.
// For SUBORDINATE CAs, parentCAArn should reference an existing ROOT or intermediate CA.
func (s *Store) CreateCA(caType, keyAlgo, sigAlgo string, subject CASubject, revocationConfig map[string]any, tags []Tag, parentCAArn ...string) (*CertificateAuthority, error) {
	caID := newUUID()
	arn := s.buildCAArn(caID)
	now := time.Now().UTC()

	if keyAlgo == "" {
		keyAlgo = "RSA_2048"
	}
	if sigAlgo == "" {
		sigAlgo = "SHA256WITHRSA"
	}
	if caType == "" {
		caType = "ROOT"
	}

	transitions := []lifecycle.Transition{
		{From: lifecycle.State(CAStatusCreating), To: lifecycle.State(CAStatusPendingCertificate), Delay: 2 * time.Second},
	}
	lc := lifecycle.NewMachine(lifecycle.State(CAStatusCreating), transitions, s.lcConfig)

	var parentArn string
	if len(parentCAArn) > 0 {
		parentArn = parentCAArn[0]
	}

	ca := &CertificateAuthority{
		Arn:                    arn,
		Type:                   caType,
		ParentCAArn:            parentArn,
		KeyAlgorithm:           keyAlgo,
		SigningAlgorithm:       sigAlgo,
		Subject:                subject,
		Status:                 CAStatusCreating,
		CreatedAt:              now,
		LastStateChangeAt:      now,
		Serial:                 generateSerial(),
		Tags:                   tags,
		RevocationConfiguration: revocationConfig,
		Lifecycle:              lc,
	}

	lc.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		ca.Status = CAStatus(to)
		ca.LastStateChangeAt = time.Now().UTC()
	})

	s.mu.Lock()
	s.cas[arn] = ca
	s.mu.Unlock()

	return ca, nil
}

// GetCA returns a CA by ARN.
func (s *Store) GetCA(arn string) (*CertificateAuthority, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ca, ok := s.cas[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", arn), http.StatusBadRequest)
	}
	if ca.Lifecycle != nil {
		ca.Status = CAStatus(ca.Lifecycle.State())
	}
	return ca, nil
}

// ListCAs returns all certificate authorities.
func (s *Store) ListCAs() []*CertificateAuthority {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*CertificateAuthority, 0, len(s.cas))
	for _, ca := range s.cas {
		if ca.Lifecycle != nil {
			ca.Status = CAStatus(ca.Lifecycle.State())
		}
		out = append(out, ca)
	}
	return out
}

// DeleteCA removes a CA by ARN.
func (s *Store) DeleteCA(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ca, ok := s.cas[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", arn), http.StatusBadRequest)
	}
	if ca.Lifecycle != nil {
		ca.Lifecycle.Stop()
	}
	ca.Status = CAStatusDeleted
	ca.LastStateChangeAt = time.Now().UTC()
	return nil
}

// UpdateCA updates a CA's status and revocation configuration.
func (s *Store) UpdateCA(arn string, status string, revocationConfig map[string]any) *service.AWSError {
	s.mu.Lock()
	ca, ok := s.cas[arn]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", arn), http.StatusBadRequest)
	}
	var lc *lifecycle.Machine
	if status != "" {
		ca.Status = CAStatus(status)
		ca.LastStateChangeAt = time.Now().UTC()
		lc = ca.Lifecycle
	}
	if revocationConfig != nil {
		ca.RevocationConfiguration = revocationConfig
	}
	s.mu.Unlock()

	// ForceState may trigger OnTransition callback that acquires s.mu.
	if lc != nil {
		lc.ForceState(lifecycle.State(status))
	}
	return nil
}

// IssueCertificate issues a new certificate from a CA.
// Deprecated: use IssueCertificateWithValidity for full control.
func (s *Store) IssueCertificate(caArn string) (*IssuedCertificate, *service.AWSError) {
	return s.IssueCertificateWithValidity(caArn, 365, "")
}

// GetCertificate returns an issued certificate.
func (s *Store) GetCertificate(caArn, certArn string) (*IssuedCertificate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cert, ok := s.certs[certArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", certArn), http.StatusBadRequest)
	}
	if cert.CertificateAuthorityArn != caArn {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Certificate does not belong to the specified CA.", http.StatusBadRequest)
	}
	return cert, nil
}

// RevokeCertificate revokes a certificate.
func (s *Store) RevokeCertificate(caArn, serial string, reason RevocationReason) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cert := range s.certs {
		if cert.CertificateAuthorityArn == caArn && cert.Serial == serial {
			if cert.Revoked {
				return service.NewAWSError("InvalidStateException",
					"Certificate is already revoked.", http.StatusBadRequest)
			}
			now := time.Now().UTC()
			cert.Revoked = true
			cert.RevokedAt = &now
			cert.RevocationReason = reason
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("Could not find certificate with serial %s.", serial), http.StatusBadRequest)
}

// AddTags adds tags to a CA.
func (s *Store) AddTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ca, ok := s.cas[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", arn), http.StatusBadRequest)
	}
	for _, newTag := range tags {
		found := false
		for i, existing := range ca.Tags {
			if existing.Key == newTag.Key {
				ca.Tags[i].Value = newTag.Value
				found = true
				break
			}
		}
		if !found {
			ca.Tags = append(ca.Tags, newTag)
		}
	}
	return nil
}

// RemoveTags removes tags from a CA.
func (s *Store) RemoveTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ca, ok := s.cas[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", arn), http.StatusBadRequest)
	}
	for _, removeTag := range tags {
		for i, existing := range ca.Tags {
			if existing.Key == removeTag.Key {
				ca.Tags = append(ca.Tags[:i], ca.Tags[i+1:]...)
				break
			}
		}
	}
	return nil
}

// CreatePermission creates a permission for a CA.
func (s *Store) CreatePermission(caArn, principal, sourceAccount string, actions []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.cas[caArn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}
	perm := &Permission{
		CertificateAuthorityArn: caArn,
		Principal:               principal,
		SourceAccount:           sourceAccount,
		Actions:                 actions,
		CreatedAt:               time.Now().UTC(),
	}
	s.permissions[caArn] = append(s.permissions[caArn], perm)
	return nil
}

// ListPermissions returns permissions for a CA.
func (s *Store) ListPermissions(caArn string) ([]*Permission, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cas[caArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}
	return s.permissions[caArn], nil
}

// DeletePermission removes a permission for a CA.
func (s *Store) DeletePermission(caArn, principal, sourceAccount string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.cas[caArn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}
	perms := s.permissions[caArn]
	for i, p := range perms {
		if p.Principal == principal && p.SourceAccount == sourceAccount {
			s.permissions[caArn] = append(perms[:i], perms[i+1:]...)
			return nil
		}
	}
	return service.NewAWSError("ResourceNotFoundException",
		"Permission not found.", http.StatusBadRequest)
}

func splitLast(s, sep string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
