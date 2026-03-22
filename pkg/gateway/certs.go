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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// buildSANs returns the list of DNS Subject Alternative Names for the given domains.
func buildSANs(domains ...string) []string {
	sans := []string{"localhost", "*.localhost"}
	for _, d := range domains {
		sans = append(sans, "localhost."+d, "*.localhost."+d)
	}
	return sans
}

// sansMatch returns true if all needed SANs are present in current.
func sansMatch(current, needed []string) bool {
	have := make(map[string]bool, len(current))
	for _, s := range current {
		have[s] = true
	}
	for _, s := range needed {
		if !have[s] {
			return false
		}
	}
	return true
}

// EnsureCerts loads existing certificates from ~/.cloudmock/certs/ or
// generates new self-signed ones if they are missing, expired, or have
// mismatched SANs. Returns a CertPair ready for use with a TLS listener.
func EnsureCerts(domains ...string) (*CertPair, error) {
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
				needed := buildSANs(domains...)
				if sansMatch(leaf.DNSNames, needed) {
					trustCA(caCertPath)
					return &CertPair{Cert: cert, CACert: caCertPath}, nil
				}
				log.Printf("certs: SAN mismatch (have %v, need %v), regenerating", leaf.DNSNames, needed)
			}
		}
	}

	// Generate new certs
	log.Printf("certs: generating self-signed CA and certificate for domains %v", domains)

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
			CommonName:   "localhost." + domains[0],
		},
		DNSNames: buildSANs(domains...),
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

	// Automatically trust the CA
	trustCA(caCertPath)

	return &CertPair{Cert: cert, CACert: caCertPath}, nil
}

// trustCA checks whether the cloudmock CA is already trusted by the OS
// and, if not, adds it to the system trust store. On macOS this uses
// `security add-trusted-cert`; on Linux it copies to ca-certificates.
// Falls back to sudo if direct access is denied.
func trustCA(caCertPath string) {
	if isCATrusted(caCertPath) {
		log.Printf("certs: CA already trusted")
		return
	}

	log.Printf("certs: trusting CA certificate...")

	switch runtime.GOOS {
	case "darwin":
		trustCADarwin(caCertPath)
	case "linux":
		trustCALinux(caCertPath)
	default:
		printTrustInstructions(caCertPath)
	}
}

func isCATrusted(caCertPath string) bool {
	switch runtime.GOOS {
	case "darwin":
		// security verify-cert returns 0 if the cert is trusted
		if err := exec.Command("security", "verify-cert", "-c", caCertPath).Run(); err == nil {
			return true
		}
		// Check login keychain
		out, err := exec.Command("security", "find-certificate", "-c", "cloudmock local CA",
			loginKeychainPath()).CombinedOutput()
		if err == nil && strings.Contains(string(out), "cloudmock local CA") {
			return true
		}
		// Check system keychain
		out, err = exec.Command("security", "find-certificate", "-c", "cloudmock local CA",
			"/Library/Keychains/System.keychain").CombinedOutput()
		return err == nil && strings.Contains(string(out), "cloudmock local CA")
	case "linux":
		return fileExists("/usr/local/share/ca-certificates/cloudmock-ca.crt")
	}
	return false
}

func loginKeychainPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Keychains/login.keychain-db")
}

func trustCADarwin(caCertPath string) {
	// Use the login keychain — no sudo required
	err := exec.Command("security", "add-trusted-cert", "-r", "trustRoot",
		"-k", loginKeychainPath(), caCertPath).Run()
	if err == nil {
		log.Printf("certs: CA trusted in login keychain")
		return
	}

	log.Printf("certs: login keychain failed (%v), trying system keychain with sudo...", err)
	cmd := exec.Command("sudo", "-p", "cloudmock needs sudo to trust the CA certificate: ",
		"security", "add-trusted-cert", "-d", "-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain", caCertPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("certs: failed to trust CA (%v)", err)
		printTrustInstructions(caCertPath)
		return
	}
	log.Printf("certs: CA trusted in system keychain")
}

func trustCALinux(caCertPath string) {
	dest := "/usr/local/share/ca-certificates/cloudmock-ca.crt"

	// Try direct copy first
	if err := exec.Command("cp", caCertPath, dest).Run(); err != nil {
		// Re-exec with sudo
		log.Printf("certs: requesting sudo to trust CA certificate...")
		cmd := exec.Command("sudo", "-p", "cloudmock needs sudo to trust the CA certificate: ",
			"cp", caCertPath, dest)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("certs: failed to copy CA cert (%v)", err)
			printTrustInstructions(caCertPath)
			return
		}
	}

	cmd := exec.Command("sudo", "update-ca-certificates")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("certs: update-ca-certificates failed (%v)", err)
	} else {
		log.Printf("certs: CA trusted in system store")
	}
}

func printTrustInstructions(caCertPath string) {
	fmt.Printf("\n")
	fmt.Printf("  Self-signed CA generated at: %s\n", caCertPath)
	fmt.Printf("  To trust it (macOS):\n")
	fmt.Printf("    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caCertPath)
	fmt.Printf("  To trust it (Linux):\n")
	fmt.Printf("    sudo cp %s /usr/local/share/ca-certificates/cloudmock-ca.crt && sudo update-ca-certificates\n", caCertPath)
	fmt.Printf("\n")
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
