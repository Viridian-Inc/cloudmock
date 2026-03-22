package gateway

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	certDir     = ".cloudmock/certs"
	caCertFile  = "ca.crt"
	caKeyFile   = "ca.key"
	srvCertFile = "server.crt"
	srvKeyFile  = "server.key"
	certBits    = 2048
	certDays    = 365
)

// CertPair holds a TLS certificate and CA cert path for trust instructions.
type CertPair struct {
	Cert   tls.Certificate
	CACert string // path to CA cert file
}

// TLSConfig returns a *tls.Config using this certificate pair.
func (cp *CertPair) TLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cp.Cert},
		MinVersion:   tls.VersionTLS12,
	}
}

// EnsureCerts loads existing certificates from ~/.cloudmock/certs/ or
// generates new self-signed ones if they are missing or expired.
// Returns a CertPair ready for use with a TLS listener.
func EnsureCerts() (*CertPair, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, certDir)

	caCertPath := filepath.Join(dir, caCertFile)
	caKeyPath := filepath.Join(dir, caKeyFile)
	srvCertPath := filepath.Join(dir, srvCertFile)
	srvKeyPath := filepath.Join(dir, srvKeyFile)

	// Try loading existing certs
	if fileExists(srvCertPath) && fileExists(srvKeyPath) && fileExists(caCertPath) {
		cert, err := tls.LoadX509KeyPair(srvCertPath, srvKeyPath)
		if err == nil {
			// Check expiration
			leaf, parseErr := x509.ParseCertificate(cert.Certificate[0])
			if parseErr == nil && time.Now().Before(leaf.NotAfter) {
				return &CertPair{Cert: cert, CACert: caCertPath}, nil
			}
			log.Printf("certs: existing certificate expired, regenerating")
		}
	}

	// Generate new certs
	log.Printf("certs: generating self-signed CA and certificate for *.local.autotend.io")

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create cert directory: %w", err)
	}

	// Generate CA
	caKey, err := rsa.GenerateKey(rand.Reader, certBits)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: newSerial(),
		Subject: pkix.Name{
			Organization: []string{"cloudmock local CA"},
			CommonName:   "cloudmock local CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(certDays * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}

	// Generate server cert signed by CA
	srvKey, err := rsa.GenerateKey(rand.Reader, certBits)
	if err != nil {
		return nil, fmt.Errorf("generate server key: %w", err)
	}

	srvTemplate := &x509.Certificate{
		SerialNumber: newSerial(),
		Subject: pkix.Name{
			Organization: []string{"cloudmock"},
			CommonName:   "local.autotend.io",
		},
		DNSNames: []string{
			"local.autotend.io",
			"*.local.autotend.io",
		},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(certDays * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	srvCertDER, err := x509.CreateCertificate(rand.Reader, srvTemplate, caCert, &srvKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create server cert: %w", err)
	}

	// Write all files
	if err := writePEM(caCertPath, "CERTIFICATE", caCertDER); err != nil {
		return nil, err
	}
	if err := writeKeyPEM(caKeyPath, caKey); err != nil {
		return nil, err
	}
	if err := writePEM(srvCertPath, "CERTIFICATE", srvCertDER); err != nil {
		return nil, err
	}
	if err := writeKeyPEM(srvKeyPath, srvKey); err != nil {
		return nil, err
	}

	// Load the certificate pair
	cert, err := tls.LoadX509KeyPair(srvCertPath, srvKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load generated cert: %w", err)
	}

	// Print trust instructions
	fmt.Printf("\n")
	fmt.Printf("  Self-signed CA generated at: %s\n", caCertPath)
	fmt.Printf("  To trust it (macOS):\n")
	fmt.Printf("    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caCertPath)
	fmt.Printf("  To trust it (Linux):\n")
	fmt.Printf("    sudo cp %s /usr/local/share/ca-certificates/cloudmock-ca.crt && sudo update-ca-certificates\n", caCertPath)
	fmt.Printf("\n")

	return &CertPair{Cert: cert, CACert: caCertPath}, nil
}

func newSerial() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	n, _ := rand.Int(rand.Reader, max)
	return n
}

func writePEM(path string, blockType string, derBytes []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: derBytes})
}

func writeKeyPEM(path string, key *rsa.PrivateKey) error {
	return writePEM(path, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
