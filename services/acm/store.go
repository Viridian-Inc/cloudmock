package acm

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

// CertificateStatus represents the lifecycle state of an ACM certificate.
type CertificateStatus string

const (
	StatusPendingValidation CertificateStatus = "PENDING_VALIDATION"
	StatusIssued            CertificateStatus = "ISSUED"
	StatusInactive          CertificateStatus = "INACTIVE"
	StatusRevoked           CertificateStatus = "REVOKED"
	StatusFailed            CertificateStatus = "FAILED"
	StatusExpired           CertificateStatus = "EXPIRED"
	StatusValidationTimedOut CertificateStatus = "VALIDATION_TIMED_OUT"
)

// CertificateType distinguishes between imported and Amazon-issued certificates.
type CertificateType string

const (
	CertTypeAmazonIssued CertificateType = "AMAZON_ISSUED"
	CertTypeImported     CertificateType = "IMPORTED"
	CertTypePrivate      CertificateType = "PRIVATE"
)

// ValidationMethod indicates how domain ownership is verified.
type ValidationMethod string

const (
	ValidationDNS   ValidationMethod = "DNS"
	ValidationEmail ValidationMethod = "EMAIL"
)

// DomainValidation holds per-domain validation information.
type DomainValidation struct {
	DomainName       string
	ValidationDomain string
	ValidationStatus string
	ValidationMethod string
	ResourceRecord   *ResourceRecord
}

// ResourceRecord holds DNS validation record information.
type ResourceRecord struct {
	Name  string
	Type  string
	Value string
}

// Tag represents a key-value tag.
type Tag struct {
	Key   string
	Value string
}

// Certificate holds all metadata for an ACM certificate.
type Certificate struct {
	CertificateArn                string
	DomainName                    string
	SubjectAlternativeNames       []string
	DomainValidationOptions       []DomainValidation
	Status                        CertificateStatus
	Type                          CertificateType
	ValidationMethod              ValidationMethod
	KeyAlgorithm                  string
	IssuedAt                      *time.Time
	CreatedAt                     time.Time
	ImportedAt                    *time.Time
	NotBefore                     *time.Time
	NotAfter                      *time.Time
	Serial                        string
	Subject                       string
	Issuer                        string
	RenewalEligibility            string
	Tags                          []Tag
	InUseBy                       []string
	CertificateChain              string
	CertificateBody               string
	PrivateKey                    string
	Lifecycle                     *lifecycle.Machine
}

// Store is the in-memory store for ACM certificates.
type Store struct {
	mu          sync.RWMutex
	certs       map[string]*Certificate // keyed by ARN
	accountID   string
	region      string
	lcConfig    *lifecycle.Config
	locator     ServiceLocator
}

// SetLocator sets the service locator for Route53 validation checks.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
}

// NewStore creates an empty ACM Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		certs:     make(map[string]*Certificate),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func (s *Store) buildARN(certID string) string {
	return fmt.Sprintf("arn:aws:acm:%s:%s:certificate/%s", s.region, s.accountID, certID)
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

func generateValidationValue() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("_%s.acm-validations.aws.", fmt.Sprintf("%x", b))
}

// RequestCertificate creates a new Amazon-issued certificate.
func (s *Store) RequestCertificate(domainName string, sans []string, validationMethod ValidationMethod, keyAlgorithm string, tags []Tag) (*Certificate, error) {
	certID := newUUID()
	arn := s.buildARN(certID)
	now := time.Now().UTC()

	if validationMethod == "" {
		validationMethod = ValidationDNS
	}
	if keyAlgorithm == "" {
		keyAlgorithm = "RSA_2048"
	}

	allDomains := []string{domainName}
	for _, san := range sans {
		found := false
		for _, d := range allDomains {
			if d == san {
				found = true
				break
			}
		}
		if !found {
			allDomains = append(allDomains, san)
		}
	}

	validations := make([]DomainValidation, 0, len(allDomains))
	for _, domain := range allDomains {
		dv := DomainValidation{
			DomainName:       domain,
			ValidationDomain: domain,
			ValidationStatus: "PENDING_VALIDATION",
			ValidationMethod: string(validationMethod),
		}
		if validationMethod == ValidationDNS {
			dv.ResourceRecord = &ResourceRecord{
				Name:  fmt.Sprintf("_cname.%s.", domain),
				Type:  "CNAME",
				Value: generateValidationValue(),
			}
		}
		validations = append(validations, dv)
	}

	transitions := []lifecycle.Transition{
		{From: lifecycle.State(StatusPendingValidation), To: lifecycle.State(StatusIssued), Delay: 5 * time.Second},
	}
	lc := lifecycle.NewMachine(lifecycle.State(StatusPendingValidation), transitions, s.lcConfig)

	cert := &Certificate{
		CertificateArn:          arn,
		DomainName:              domainName,
		SubjectAlternativeNames: allDomains,
		DomainValidationOptions: validations,
		Status:                  StatusPendingValidation,
		Type:                    CertTypeAmazonIssued,
		ValidationMethod:        validationMethod,
		KeyAlgorithm:            keyAlgorithm,
		CreatedAt:               now,
		Serial:                  generateSerial(),
		Subject:                 fmt.Sprintf("CN=%s", domainName),
		Issuer:                  "Amazon",
		RenewalEligibility:      "INELIGIBLE",
		Tags:                    tags,
		InUseBy:                 []string{},
		Lifecycle:               lc,
	}

	lc.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		cert.Status = CertificateStatus(to)
		if to == lifecycle.State(StatusIssued) {
			now := time.Now().UTC()
			cert.IssuedAt = &now
			notAfter := now.AddDate(1, 0, 0)
			cert.NotBefore = &now
			cert.NotAfter = &notAfter
			cert.RenewalEligibility = "ELIGIBLE"
			for i := range cert.DomainValidationOptions {
				cert.DomainValidationOptions[i].ValidationStatus = "SUCCESS"
			}
		}
	})

	s.mu.Lock()
	s.certs[arn] = cert
	s.mu.Unlock()

	return cert, nil
}

// ImportCertificate imports a certificate (body, chain, private key).
func (s *Store) ImportCertificate(certBody, certChain, privateKey string, tags []Tag, existingARN string) (*Certificate, error) {
	now := time.Now().UTC()
	var arn string
	if existingARN != "" {
		arn = existingARN
	} else {
		arn = s.buildARN(newUUID())
	}

	cert := &Certificate{
		CertificateArn:          arn,
		DomainName:              "imported.example.com",
		SubjectAlternativeNames: []string{"imported.example.com"},
		DomainValidationOptions: []DomainValidation{},
		Status:                  StatusIssued,
		Type:                    CertTypeImported,
		KeyAlgorithm:            "RSA_2048",
		CreatedAt:               now,
		ImportedAt:              &now,
		IssuedAt:                &now,
		NotBefore:               &now,
		Serial:                  generateSerial(),
		Subject:                 "CN=imported.example.com",
		Issuer:                  "Unknown",
		RenewalEligibility:      "INELIGIBLE",
		Tags:                    tags,
		InUseBy:                 []string{},
		CertificateBody:         certBody,
		CertificateChain:        certChain,
		PrivateKey:              privateKey,
	}
	notAfter := now.AddDate(1, 0, 0)
	cert.NotAfter = &notAfter

	s.mu.Lock()
	s.certs[arn] = cert
	s.mu.Unlock()

	return cert, nil
}

// GetCertificate returns a certificate by ARN.
func (s *Store) GetCertificate(arn string) (*Certificate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cert, ok := s.certs[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", arn), http.StatusBadRequest)
	}
	// Sync lifecycle state
	if cert.Lifecycle != nil {
		cert.Status = CertificateStatus(cert.Lifecycle.State())
	}
	return cert, nil
}

// ListCertificates returns all certificates.
func (s *Store) ListCertificates() []*Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Certificate, 0, len(s.certs))
	for _, c := range s.certs {
		if c.Lifecycle != nil {
			c.Status = CertificateStatus(c.Lifecycle.State())
		}
		out = append(out, c)
	}
	return out
}

// DeleteCertificate removes a certificate by ARN.
func (s *Store) DeleteCertificate(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, ok := s.certs[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", arn), http.StatusBadRequest)
	}
	if len(cert.InUseBy) > 0 {
		return service.NewAWSError("ResourceInUseException",
			"Certificate is in use.", http.StatusBadRequest)
	}
	if cert.Lifecycle != nil {
		cert.Lifecycle.Stop()
	}
	delete(s.certs, arn)
	return nil
}

// AddTags adds tags to a certificate.
func (s *Store) AddTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, ok := s.certs[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", arn), http.StatusBadRequest)
	}
	for _, newTag := range tags {
		found := false
		for i, existing := range cert.Tags {
			if existing.Key == newTag.Key {
				cert.Tags[i].Value = newTag.Value
				found = true
				break
			}
		}
		if !found {
			cert.Tags = append(cert.Tags, newTag)
		}
	}
	return nil
}

// RemoveTags removes tags from a certificate.
func (s *Store) RemoveTags(arn string, tags []Tag) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, ok := s.certs[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", arn), http.StatusBadRequest)
	}
	for _, removeTag := range tags {
		for i, existing := range cert.Tags {
			if existing.Key == removeTag.Key {
				cert.Tags = append(cert.Tags[:i], cert.Tags[i+1:]...)
				break
			}
		}
	}
	return nil
}

// RenewCertificate triggers renewal for a certificate. New validation records
// are generated and the certificate is re-validated.
func (s *Store) RenewCertificate(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, ok := s.certs[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate %s.", arn), http.StatusBadRequest)
	}
	if cert.Type != CertTypeAmazonIssued {
		return service.NewAWSError("ValidationException",
			"Only Amazon-issued certificates can be renewed.", http.StatusBadRequest)
	}
	if cert.Status != StatusIssued {
		return service.NewAWSError("InvalidStateException",
			"Certificate is not in a state that allows renewal.", http.StatusBadRequest)
	}

	// Generate new validation records for renewal.
	for i := range cert.DomainValidationOptions {
		dv := &cert.DomainValidationOptions[i]
		if dv.ValidationMethod == string(ValidationDNS) {
			dv.ResourceRecord = &ResourceRecord{
				Name:  fmt.Sprintf("_cname.%s.", dv.DomainName),
				Type:  "CNAME",
				Value: generateValidationValue(),
			}
			dv.ValidationStatus = "SUCCESS" // Auto-succeed for renewal of already-issued cert
		}
	}

	now := time.Now().UTC()
	notAfter := now.AddDate(1, 0, 0)
	cert.IssuedAt = &now
	cert.NotAfter = &notAfter
	return nil
}
