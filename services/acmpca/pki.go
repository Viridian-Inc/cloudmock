package acmpca

import (
	"fmt"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// IssueCertificateWithValidity issues a certificate with proper validation and realistic attributes.
func (s *Store) IssueCertificateWithValidity(caArn string, validityDays int, sigAlgo string) (*IssuedCertificate, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ca, ok := s.cas[caArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}
	if ca.Lifecycle != nil {
		ca.Status = CAStatus(ca.Lifecycle.State())
	}
	if ca.Status != CAStatusActive {
		return nil, service.NewAWSError("InvalidStateException",
			"Certificate authority is not in ACTIVE state.", http.StatusBadRequest)
	}

	if validityDays <= 0 {
		validityDays = 365
	}
	if sigAlgo == "" {
		sigAlgo = ca.SigningAlgorithm
	}

	// Extract caID from ARN
	caID := ""
	parts := splitLast(caArn, "/")
	if len(parts) == 2 {
		caID = parts[1]
	}

	certID := newUUID()
	certArn := s.buildCertArn(caID, certID)
	now := time.Now().UTC()
	serial := generateSerial()
	notBefore := now
	notAfter := now.Add(time.Duration(validityDays) * 24 * time.Hour)

	// Build certificate chain
	chain := s.buildCertificateChain(ca)

	certBody := fmt.Sprintf(
		"-----BEGIN CERTIFICATE-----\n"+
			"Serial: %s\n"+
			"Issuer: CN=%s, O=%s\n"+
			"NotBefore: %s\n"+
			"NotAfter: %s\n"+
			"SignatureAlgorithm: %s\n"+
			"-----END CERTIFICATE-----",
		serial,
		ca.Subject.CommonName, ca.Subject.Organization,
		notBefore.Format(time.RFC3339),
		notAfter.Format(time.RFC3339),
		sigAlgo,
	)

	cert := &IssuedCertificate{
		CertificateArn:          certArn,
		CertificateBody:         certBody,
		CertificateChain:        chain,
		Serial:                  serial,
		IssuedAt:                now,
		NotBefore:               &notBefore,
		NotAfter:                &notAfter,
		CertificateAuthorityArn: caArn,
	}

	s.certs[certArn] = cert
	return cert, nil
}

// buildCertificateChain builds the certificate chain by walking up from the CA.
func (s *Store) buildCertificateChain(ca *CertificateAuthority) string {
	chain := fmt.Sprintf(
		"-----BEGIN CERTIFICATE-----\n"+
			"CA: %s\n"+
			"Type: %s\n"+
			"Serial: %s\n"+
			"Subject: CN=%s, O=%s\n"+
			"-----END CERTIFICATE-----",
		ca.Arn, ca.Type, ca.Serial,
		ca.Subject.CommonName, ca.Subject.Organization,
	)

	// If this is a subordinate CA with a parent, include parent chain
	if ca.Type == "SUBORDINATE" && ca.ParentCAArn != "" {
		parentCA, ok := s.cas[ca.ParentCAArn]
		if ok {
			parentChain := s.buildCertificateChain(parentCA)
			chain = chain + "\n" + parentChain
		}
	}

	return chain
}

// GetCertificateAuthorityCsr returns a mock CSR for a CA in PENDING_CERTIFICATE state.
func (s *Store) GetCertificateAuthorityCsr(caArn string) (string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ca, ok := s.cas[caArn]
	if !ok {
		return "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}
	if ca.Lifecycle != nil {
		ca.Status = CAStatus(ca.Lifecycle.State())
	}
	if ca.Status != CAStatusPendingCertificate {
		return "", service.NewAWSError("InvalidStateException",
			"Certificate authority is not in PENDING_CERTIFICATE state.", http.StatusBadRequest)
	}

	csr := fmt.Sprintf(
		"-----BEGIN CERTIFICATE REQUEST-----\n"+
			"Subject: CN=%s, O=%s, OU=%s, L=%s, ST=%s, C=%s\n"+
			"KeyAlgorithm: %s\n"+
			"SignatureAlgorithm: %s\n"+
			"-----END CERTIFICATE REQUEST-----",
		ca.Subject.CommonName, ca.Subject.Organization,
		ca.Subject.OrganizationalUnit, ca.Subject.Locality,
		ca.Subject.State, ca.Subject.Country,
		ca.KeyAlgorithm, ca.SigningAlgorithm,
	)

	return csr, nil
}

// GetCertificateAuthorityCertificate returns the CA certificate and certificate chain.
func (s *Store) GetCertificateAuthorityCertificate(caArn string) (string, string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ca, ok := s.cas[caArn]
	if !ok {
		return "", "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}

	caCert := fmt.Sprintf(
		"-----BEGIN CERTIFICATE-----\n"+
			"CA: %s\n"+
			"Type: %s\n"+
			"Serial: %s\n"+
			"Subject: CN=%s, O=%s\n"+
			"-----END CERTIFICATE-----",
		ca.Arn, ca.Type, ca.Serial,
		ca.Subject.CommonName, ca.Subject.Organization,
	)

	chain := s.buildCertificateChain(ca)
	return caCert, chain, nil
}

// ListRevokedCertificates returns all revoked certificates for a CA.
func (s *Store) ListRevokedCertificates(caArn string) ([]*IssuedCertificate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.cas[caArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Could not find certificate authority %s.", caArn), http.StatusBadRequest)
	}

	var revoked []*IssuedCertificate
	for _, cert := range s.certs {
		if cert.CertificateAuthorityArn == caArn && cert.Revoked {
			revoked = append(revoked, cert)
		}
	}
	return revoked, nil
}
