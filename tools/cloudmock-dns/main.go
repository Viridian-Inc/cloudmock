package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const marker = "# cloudmock local development"
const hostsFile = "/etc/hosts"

// resolverDir is the macOS per-domain resolver directory.
const resolverDir = "/etc/resolver"

const baseDNSPort = 15353

type domainConfig struct {
	Autotend  string
	Cloudmock string
}

var defaultDomains = domainConfig{
	Autotend:  "cloudmock.app",
	Cloudmock: "cloudmock.io",
}

func parsePulumiConfig(path string) (domainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultDomains, err
	}
	var raw struct {
		Config map[string]any `yaml:"config"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return defaultDomains, fmt.Errorf("parse %s: %w", path, err)
	}
	domainsRaw, ok := raw.Config["backend:domains"]
	if !ok {
		return defaultDomains, fmt.Errorf("no backend:domains in %s", path)
	}
	domainsMap, ok := domainsRaw.(map[string]any)
	if !ok {
		return defaultDomains, fmt.Errorf("domains is not a map in %s", path)
	}
	dc := defaultDomains
	if v, ok := domainsMap["cloudmock"].(string); ok {
		dc.Autotend = v
	}
	if v, ok := domainsMap["cloudmock"].(string); ok {
		dc.Cloudmock = v
	}
	return dc, nil
}

func (dc domainConfig) sortedDomains() []struct{ key, domain string } {
	pairs := []struct{ key, domain string }{
		{"cloudmock", dc.Autotend},
		{"cloudmock", dc.Cloudmock},
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
	return pairs
}

func (dc domainConfig) resolverEntries() []struct{ path, content string } {
	var entries []struct{ path, content string }
	for i, d := range dc.sortedDomains() {
		port := baseDNSPort + i
		entries = append(entries, struct{ path, content string }{
			path:    resolverDir + "/" + d.domain,
			content: fmt.Sprintf("nameserver 127.0.0.1\nport %d\n", port),
		})
	}
	return entries
}

func (dc domainConfig) hostsEntries() []string {
	at := "localhost." + dc.Autotend
	cm := "localhost." + dc.Cloudmock
	return []string{
		"127.0.0.1  " + at,
		"127.0.0.1  app." + at,
		"127.0.0.1  bff." + at,
		"127.0.0.1  api." + at,
		"127.0.0.1  auth." + at,
		"127.0.0.1  admin." + at,
		"127.0.0.1  graphql." + at,
		"127.0.0.1  " + cm,
	}
}

var configPath string

func main() {
	fs := flag.NewFlagSet("cloudmock-dns", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "", "path to Pulumi stack YAML config file")
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	cmd := args[0]
	fs.Parse(args[1:])

	domains := defaultDomains
	if configPath != "" {
		var err error
		domains, err = parsePulumiConfig(configPath)
		if err != nil {
			fmt.Printf("Warning: %v, using defaults\n", err)
		}
	}

	switch cmd {
	case "auto", "setup":
		if cmd == "auto" {
			autoSetup(domains)
		} else {
			setup(domains)
		}
	case "remove":
		remove(domains)
	case "status":
		status(domains)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cloudmock-dns <auto|setup|remove|status> [--config <path>]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  auto    One-time OS resolver setup (preferred — uses /etc/resolver on macOS)")
	fmt.Println("  setup   Add entries to /etc/hosts (legacy — requires sudo)")
	fmt.Println("  remove  Remove all cloudmock DNS configuration")
	fmt.Println("  status  Show current DNS configuration status")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --config <path>  Path to Pulumi stack YAML config file for domain config")
}

// autoSetup configures the OS resolver so that *.localhost.cloudmock.app queries
// are answered by cloudmock's built-in DNS server on :15353.
//
// On macOS this writes /etc/resolver/cloudmock.app (needs sudo once).
// On Linux it prints manual instructions for systemd-resolved or NetworkManager.
//
// This only needs to be done ONCE — the config persists across reboots.
func autoSetup(dc domainConfig) {
	switch runtime.GOOS {
	case "darwin":
		autoSetupMacOS(dc)
	case "linux":
		autoSetupLinux(dc)
	default:
		fmt.Printf("Auto-setup is not supported on %s.\n", runtime.GOOS)
		fmt.Println("Use 'cloudmock-dns setup' to edit /etc/hosts instead.")
	}
}

func autoSetupMacOS(dc domainConfig) {
	resolverFiles := dc.resolverEntries()

	// Check whether all resolvers are already configured.
	allConfigured := true
	for _, r := range resolverFiles {
		if _, err := os.Stat(r.path); err != nil {
			allConfigured = false
			break
		}
		data, _ := os.ReadFile(r.path)
		if string(data) != r.content {
			allConfigured = false
			break
		}
	}
	if allConfigured {
		fmt.Println("Status: CONFIGURED (macOS resolver)")
		for _, r := range resolverFiles {
			fmt.Printf("  %s\n", r.path)
		}
		printLocalDomains(dc)
		return
	}

	// Try to write directly (works if we are root).
	if tryWriteResolverFiles(dc) {
		fmt.Println("cloudmock DNS resolvers configured!")
		for _, r := range resolverFiles {
			fmt.Printf("  Created: %s\n", r.path)
		}
		printLocalDomains(dc)
		return
	}

	// Re-exec ourselves with sudo.
	fmt.Println("Configuring macOS DNS resolver (requires sudo)...")
	sudoArgs := []string{os.Args[0], "_internal_resolver_setup"}
	if configPath != "" {
		sudoArgs = append(sudoArgs, "--config", configPath)
	}
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nFailed to configure resolver: %v\n", err)
		fmt.Println("\nYou can do it manually:")
		fmt.Printf("  sudo mkdir -p %s\n", resolverDir)
		for _, r := range resolverFiles {
			fmt.Printf("  printf '%s' | sudo tee %s\n", r.content, r.path)
		}
		os.Exit(1)
	}
	printLocalDomains(dc)
}

func tryWriteResolverFiles(dc domainConfig) bool {
	if err := os.MkdirAll(resolverDir, 0755); err != nil {
		return false
	}
	for _, r := range dc.resolverEntries() {
		if err := os.WriteFile(r.path, []byte(r.content), 0644); err != nil {
			return false
		}
	}
	return true
}

func autoSetupLinux(dc domainConfig) {
	fmt.Println("Linux auto-setup:")
	fmt.Println()
	fmt.Println("Option A — systemd-resolved:")
	fmt.Println("  Create /etc/systemd/resolved.conf.d/cloudmock.conf:")
	fmt.Println("    [Resolve]")
	fmt.Println("    DNS=127.0.0.1")
	fmt.Printf("    Domains=~%s ~%s\n", dc.Autotend, dc.Cloudmock)
	fmt.Println("  Then: sudo systemctl restart systemd-resolved")
	fmt.Println()
	fmt.Println("Option B — /etc/hosts (no cloudmock DNS needed):")
	fmt.Println("  sudo cloudmock-dns setup")
}

// _internal_resolver_setup is called by autoSetupMacOS after re-execing with sudo.
// It performs the actual file write as root.
func internalResolverSetup(dc domainConfig) {
	if !tryWriteResolverFiles(dc) {
		fmt.Fprintf(os.Stderr, "failed to write resolver files\n")
		os.Exit(1)
	}
	for _, r := range dc.resolverEntries() {
		fmt.Printf("Created %s\n", r.path)
	}
}

func printLocalDomains(dc domainConfig) {
	at := dc.Autotend
	cm := dc.Cloudmock
	fmt.Println()
	fmt.Println("  Zero-config (works immediately, no setup needed):")
	fmt.Println("    http://app.localhost")
	fmt.Println("    http://cloudmock.localhost")
	fmt.Println("    http://bff.localhost")
	fmt.Println("    http://api.localhost")
	fmt.Println("    http://auth.localhost")
	fmt.Println()
	fmt.Println("  Custom domains (now configured via DNS resolver):")
	fmt.Printf("    https://localhost.%s          ← cloudmock app\n", at)
	fmt.Printf("    https://bff.localhost.%s\n", at)
	fmt.Printf("    https://api.localhost.%s\n", at)
	fmt.Printf("    https://auth.localhost.%s\n", at)
	fmt.Printf("    https://graphql.localhost.%s\n", at)
	fmt.Printf("    https://localhost.%s         ← cloudmock dashboard\n", cm)
}

// setup adds /etc/hosts entries (legacy, requires sudo).
func setup(dc domainConfig) {
	entries := dc.hostsEntries()

	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", hostsFile, err)
		fmt.Println("Try: sudo cloudmock-dns setup")
		os.Exit(1)
	}

	content := string(data)
	if strings.Contains(content, marker) {
		fmt.Println("cloudmock DNS entries already exist in /etc/hosts")
		status(dc)
		return
	}

	block := "\n" + marker + "\n" + strings.Join(entries, "\n") + "\n"
	content += block

	if err := os.WriteFile(hostsFile, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", hostsFile, err)
		fmt.Println("Try: sudo cloudmock-dns setup")
		os.Exit(1)
	}

	fmt.Println("Added cloudmock DNS entries to /etc/hosts:")
	for _, e := range entries {
		fmt.Printf("  %s\n", e)
	}
	fmt.Printf("\nYou can now access: http://localhost.%s\n", dc.Autotend)
	fmt.Println("\nTip: 'cloudmock-dns auto' sets up a DNS resolver instead (no /etc/hosts needed).")
}

// remove removes all cloudmock DNS configuration (/etc/resolver and /etc/hosts).
func remove(dc domainConfig) {
	removedAny := false

	// Remove macOS resolver files.
	for _, r := range dc.resolverEntries() {
		if _, err := os.Stat(r.path); err == nil {
			if err := os.Remove(r.path); err != nil {
				fmt.Printf("Error removing %s: %v\n", r.path, err)
				fmt.Println("Try: sudo cloudmock-dns remove")
			} else {
				fmt.Printf("Removed %s\n", r.path)
				removedAny = true
			}
		}
	}

	// Remove /etc/hosts entries.
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", hostsFile, err)
		os.Exit(1)
	}

	content := string(data)
	if strings.Contains(content, marker) {
		lines := strings.Split(content, "\n")
		var result []string
		inBlock := false
		for _, line := range lines {
			if line == marker {
				inBlock = true
				continue
			}
			if inBlock {
				if strings.HasPrefix(strings.TrimSpace(line), "127.0.0.1") &&
					(strings.Contains(line, "localhost."+dc.Autotend) || strings.Contains(line, "localhost."+dc.Cloudmock)) {
					continue
				}
				if strings.TrimSpace(line) == "" {
					inBlock = false
					continue
				}
				inBlock = false
			}
			result = append(result, line)
		}

		if err := os.WriteFile(hostsFile, []byte(strings.Join(result, "\n")), 0644); err != nil {
			fmt.Printf("Error writing %s: %v\n", hostsFile, err)
			fmt.Println("Try: sudo cloudmock-dns remove")
			os.Exit(1)
		}
		fmt.Println("Removed cloudmock DNS entries from /etc/hosts")
		removedAny = true
	}

	if !removedAny {
		fmt.Println("Nothing to remove — cloudmock DNS is not configured.")
	}
}

// status shows the current DNS configuration.
func status(dc domainConfig) {
	entries := dc.hostsEntries()
	resolverFiles := dc.resolverEntries()

	fmt.Println("cloudmock DNS status")
	fmt.Println("====================")

	// Check macOS resolver files.
	for _, r := range resolverFiles {
		if _, err := os.Stat(r.path); err == nil {
			data, _ := os.ReadFile(r.path)
			if string(data) == r.content {
				fmt.Printf("  Resolver       : CONFIGURED (%s)\n", r.path)
			} else {
				fmt.Printf("  Resolver       : EXISTS but unexpected content (%s)\n", r.path)
			}
		} else {
			fmt.Printf("  Resolver       : not configured (%s)\n", r.path)
		}
	}

	// Check /etc/hosts.
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("  /etc/hosts    : unreadable (%v)\n", err)
	} else if strings.Contains(string(data), marker) {
		fmt.Println("  /etc/hosts    : CONFIGURED")
		for _, e := range entries {
			domain := strings.Fields(e)[1]
			fmt.Printf("    %s → 127.0.0.1\n", domain)
		}
	} else {
		fmt.Println("  /etc/hosts    : not configured")
	}

	fmt.Println()
	fmt.Println("  Zero-config (always works):")
	fmt.Println("    http://app.localhost  ← Expo app (no setup needed)")
	fmt.Println("    http://cloudmock.localhost     ← cloudmock dashboard")
	fmt.Println()
	fmt.Println("  To configure custom domain: sudo cloudmock-dns auto")
}

func init() {
	// When re-execed by autoSetupMacOS with sudo, handle the internal command.
	if len(os.Args) >= 2 && os.Args[1] == "_internal_resolver_setup" {
		fs := flag.NewFlagSet("_internal", flag.ContinueOnError)
		var cfgPath string
		fs.StringVar(&cfgPath, "config", "", "")
		fs.Parse(os.Args[2:])

		domains := defaultDomains
		if cfgPath != "" {
			configPath = cfgPath
			if dc, err := parsePulumiConfig(cfgPath); err == nil {
				domains = dc
			}
		}
		internalResolverSetup(domains)
		os.Exit(0)
	}
}
