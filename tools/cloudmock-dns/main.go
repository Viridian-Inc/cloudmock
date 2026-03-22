package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const marker = "# cloudmock local development"
const hostsFile = "/etc/hosts"

// resolverDir is the macOS per-domain resolver directory.
const resolverDir = "/etc/resolver"

// resolverFile is the file that points macOS at cloudmock's DNS server
// for the autotend.io domain.
const resolverFile = resolverDir + "/autotend.io"

// resolverContent is written to /etc/resolver/autotend.io.
// It tells macOS to send all *.autotend.io queries to cloudmock's DNS
// server running on UDP :15353.
const resolverContent = "nameserver 127.0.0.1\nport 15353\n"

var entries = []string{
	"127.0.0.1  local.autotend.io",
	"127.0.0.1  bff.local.autotend.io",
	"127.0.0.1  api.local.autotend.io",
	"127.0.0.1  auth.local.autotend.io",
	"127.0.0.1  dashboard.local.autotend.io",
	"127.0.0.1  admin.local.autotend.io",
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "auto", "setup":
		if os.Args[1] == "auto" {
			autoSetup()
		} else {
			setup()
		}
	case "remove":
		remove()
	case "status":
		status()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cloudmock-dns <auto|setup|remove|status>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  auto    One-time OS resolver setup (preferred — uses /etc/resolver on macOS)")
	fmt.Println("  setup   Add entries to /etc/hosts (legacy — requires sudo)")
	fmt.Println("  remove  Remove all cloudmock DNS configuration")
	fmt.Println("  status  Show current DNS configuration status")
}

// autoSetup configures the OS resolver so that *.local.autotend.io queries
// are answered by cloudmock's built-in DNS server on :15353.
//
// On macOS this writes /etc/resolver/autotend.io (needs sudo once).
// On Linux it prints manual instructions for systemd-resolved or NetworkManager.
//
// This only needs to be done ONCE — the config persists across reboots.
func autoSetup() {
	switch runtime.GOOS {
	case "darwin":
		autoSetupMacOS()
	case "linux":
		autoSetupLinux()
	default:
		fmt.Printf("Auto-setup is not supported on %s.\n", runtime.GOOS)
		fmt.Println("Use 'cloudmock-dns setup' to edit /etc/hosts instead.")
	}
}

func autoSetupMacOS() {
	// Check whether already configured.
	if _, err := os.Stat(resolverFile); err == nil {
		data, _ := os.ReadFile(resolverFile)
		if strings.Contains(string(data), "15353") {
			fmt.Println("Status: CONFIGURED (macOS resolver)")
			fmt.Printf("  %s already points to cloudmock DNS on :15353\n", resolverFile)
			printLocalDomains()
			return
		}
	}

	// Try to write directly (works if we are root).
	if tryWriteResolverFile() {
		fmt.Println("cloudmock DNS resolver configured!")
		fmt.Printf("  Created: %s\n", resolverFile)
		printLocalDomains()
		return
	}

	// Re-exec ourselves with sudo.
	fmt.Println("Configuring macOS DNS resolver (requires sudo)...")
	cmd := exec.Command("sudo", os.Args[0], "_internal_resolver_setup")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nFailed to configure resolver: %v\n", err)
		fmt.Println("\nYou can do it manually:")
		fmt.Printf("  sudo mkdir -p %s\n", resolverDir)
		fmt.Printf("  printf '%s' | sudo tee %s\n", resolverContent, resolverFile)
		os.Exit(1)
	}
	printLocalDomains()
}

func tryWriteResolverFile() bool {
	if err := os.MkdirAll(resolverDir, 0755); err != nil {
		return false
	}
	if err := os.WriteFile(resolverFile, []byte(resolverContent), 0644); err != nil {
		return false
	}
	return true
}

func autoSetupLinux() {
	fmt.Println("Linux auto-setup:")
	fmt.Println()
	fmt.Println("Option A — systemd-resolved:")
	fmt.Println("  Create /etc/systemd/resolved.conf.d/cloudmock.conf:")
	fmt.Println("    [Resolve]")
	fmt.Println("    DNS=127.0.0.1")
	fmt.Println("    Domains=~autotend.io")
	fmt.Println("  Then: sudo systemctl restart systemd-resolved")
	fmt.Println()
	fmt.Println("Option B — /etc/hosts (no cloudmock DNS needed):")
	fmt.Println("  sudo cloudmock-dns setup")
}

// _internal_resolver_setup is called by autoSetupMacOS after re-execing with sudo.
// It performs the actual file write as root.
func internalResolverSetup() {
	if !tryWriteResolverFile() {
		fmt.Fprintf(os.Stderr, "failed to write %s\n", resolverFile)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", resolverFile)
}

func printLocalDomains() {
	fmt.Println()
	fmt.Println("  Zero-config (works immediately, no setup needed):")
	fmt.Println("    http://autotend.localhost")
	fmt.Println("    http://bff.autotend.localhost")
	fmt.Println("    http://api.autotend.localhost")
	fmt.Println("    http://auth.autotend.localhost")
	fmt.Println("    http://dashboard.autotend.localhost")
	fmt.Println()
	fmt.Println("  Custom domain (now configured via DNS resolver):")
	fmt.Println("    http://local.autotend.io")
	fmt.Println("    http://bff.local.autotend.io")
	fmt.Println("    http://api.local.autotend.io")
	fmt.Println("    http://auth.local.autotend.io")
	fmt.Println("    http://dashboard.local.autotend.io")
}

// setup adds /etc/hosts entries (legacy, requires sudo).
func setup() {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", hostsFile, err)
		fmt.Println("Try: sudo cloudmock-dns setup")
		os.Exit(1)
	}

	content := string(data)
	if strings.Contains(content, marker) {
		fmt.Println("cloudmock DNS entries already exist in /etc/hosts")
		status()
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
	fmt.Println("\nYou can now access: http://local.autotend.io")
	fmt.Println("\nTip: 'cloudmock-dns auto' sets up a DNS resolver instead (no /etc/hosts needed).")
}

// remove removes all cloudmock DNS configuration (/etc/resolver and /etc/hosts).
func remove() {
	removedAny := false

	// Remove macOS resolver file.
	if _, err := os.Stat(resolverFile); err == nil {
		if err := os.Remove(resolverFile); err != nil {
			fmt.Printf("Error removing %s: %v\n", resolverFile, err)
			fmt.Println("Try: sudo cloudmock-dns remove")
		} else {
			fmt.Printf("Removed %s\n", resolverFile)
			removedAny = true
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
				if strings.Contains(line, "autotend.io") && strings.HasPrefix(strings.TrimSpace(line), "127.0.0.1") {
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
func status() {
	fmt.Println("cloudmock DNS status")
	fmt.Println("====================")

	// Check macOS resolver.
	if _, err := os.Stat(resolverFile); err == nil {
		data, _ := os.ReadFile(resolverFile)
		if strings.Contains(string(data), "15353") {
			fmt.Printf("  Resolver file : CONFIGURED (%s)\n", resolverFile)
			fmt.Println("  DNS server    : cloudmock built-in on UDP :15353")
		} else {
			fmt.Printf("  Resolver file : EXISTS but unexpected content (%s)\n", resolverFile)
		}
	} else {
		fmt.Println("  Resolver file : not configured")
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
	fmt.Println("    http://autotend.localhost  ← use this immediately, no setup needed")
	fmt.Println()
	fmt.Println("  To configure custom domain: sudo cloudmock-dns auto")
}

func init() {
	// When re-execed by autoSetupMacOS with sudo, handle the internal command.
	if len(os.Args) >= 2 && os.Args[1] == "_internal_resolver_setup" {
		internalResolverSetup()
		os.Exit(0)
	}
}
