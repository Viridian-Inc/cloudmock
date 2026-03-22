package main

import (
	"fmt"
	"os"
	"strings"
)

const marker = "# cloudmock local development"
const hostsFile = "/etc/hosts"

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
		fmt.Println("Usage: cloudmock-dns <setup|remove|status>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "setup":
		setup()
	case "remove":
		remove()
	case "status":
		status()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: cloudmock-dns <setup|remove|status>")
		os.Exit(1)
	}
}

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
}

func remove() {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", hostsFile, err)
		os.Exit(1)
	}

	content := string(data)
	if !strings.Contains(content, marker) {
		fmt.Println("No cloudmock DNS entries found in /etc/hosts")
		return
	}

	// Remove the block between marker and the next blank line or EOF
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false
	for _, line := range lines {
		if line == marker {
			inBlock = true
			continue
		}
		if inBlock {
			// Skip entries that start with 127.0.0.1 and contain autotend.io
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
}

func status() {
	data, err := os.ReadFile(hostsFile)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", hostsFile, err)
		os.Exit(1)
	}

	content := string(data)
	if !strings.Contains(content, marker) {
		fmt.Println("Status: NOT configured")
		fmt.Println("Run: sudo cloudmock-dns setup")
		return
	}

	fmt.Println("Status: CONFIGURED")
	fmt.Println("Entries:")
	for _, e := range entries {
		domain := strings.Fields(e)[1]
		fmt.Printf("  %s → 127.0.0.1\n", domain)
	}
}
