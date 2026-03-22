# Dynamic Domain Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make domain configuration (`autotend.io`, `cloudmock.io`) dynamic by reading from Pulumi stack config instead of hardcoding across multiple files.

**Architecture:** Domains are defined in `Pulumi.local.yaml` and flow through two paths: (1) TypeScript orchestrator parses YAML -> passes env vars to cloudmock gateway, and (2) `cloudmock-dns` tool parses the same YAML directly. The gateway's `BuildRoutes()` and `EnsureCerts()` generate proxy routes and TLS certificates dynamically from the domain parameters.

**Tech Stack:** Go (cloudmock gateway, cloudmock-dns), TypeScript (orchestrator), YAML (Pulumi config)

**Spec:** `docs/superpowers/specs/2026-03-21-dynamic-domain-config-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `autotend-infra/pulumi/Pulumi.local.yaml` | Domain config source of truth |
| Modify | `cloudmock/pkg/gateway/proxy.go` | Dynamic route generation |
| Create | `cloudmock/pkg/gateway/proxy_test.go` | Tests for BuildRoutes |
| Modify | `cloudmock/pkg/gateway/certs.go` | Dynamic SAN generation + SAN comparison |
| Create | `cloudmock/pkg/gateway/certs_test.go` | Tests for SAN generation |
| Modify | `cloudmock/cmd/gateway/main.go` | Reconcile with new function signatures |
| Modify | `cloudmock/tools/cloudmock-dns/main.go` | --config flag, dynamic resolver generation |
| Modify | `autotend-infra/local/config.ts` | Parse Pulumi YAML for domains |
| Modify | `autotend-infra/local/dev.ts` | Use config.domains for env vars and summary |

---

### Task 1: Create Pulumi.local.yaml

**Files:**
- Create: `autotend-infra/pulumi/Pulumi.local.yaml`

- [ ] **Step 1: Create the config file**

```yaml
config:
  autotend-backend:environment: local
  autotend-backend:domains:
    autotend: autotend.io
    cloudmock: cloudmock.io
```

- [ ] **Step 2: Verify YAML is valid**

Run: `python3 -c "import yaml; yaml.safe_load(open('autotend-infra/pulumi/Pulumi.local.yaml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Commit**

```bash
git add autotend-infra/pulumi/Pulumi.local.yaml
git commit -m "feat: add Pulumi.local.yaml with domain config"
```

---

### Task 2: BuildRoutes + EnsureCerts — implement together (atomic)

> **Note:** `main.go` already calls `BuildRoutes(...)` and `EnsureCerts(...)` with domain
> params. The package will not compile until both functions exist with the new signatures.
> All changes in this task must land together.

**Files:**
- Modify: `cloudmock/pkg/gateway/proxy.go` (replace `DefaultAutotendRoutes` with `BuildRoutes`)
- Modify: `cloudmock/pkg/gateway/certs.go` (add domain params + SAN comparison)
- Create: `cloudmock/pkg/gateway/proxy_test.go`
- Create: `cloudmock/pkg/gateway/certs_test.go`

- [ ] **Step 1: Implement BuildRoutes in proxy.go**

Replace the entire `DefaultAutotendRoutes` function and its comment block with:

```go
// BuildRoutes generates the routing table for local development using the
// provided domain names. The autotendDomain is used for app service routes,
// and cloudmockDomain for the cloudmock dashboard.
func BuildRoutes(autotendDomain, cloudmockDomain string) []ProxyRoute {
	at := "localhost." + autotendDomain  // e.g. "localhost.autotend.io"
	cm := "localhost." + cloudmockDomain // e.g. "localhost.cloudmock.io"

	return []ProxyRoute{
		// ---- .localhost domains (RFC 6761, zero config) ----

		{Host: "autotend-app.localhost", Path: "/", Backend: "http://localhost:8081"},

		{Host: "cloudmock.localhost", Path: "/_cloudmock/", Backend: "http://localhost:4566"},
		{Host: "cloudmock.localhost", Path: "/api/", Backend: "http://localhost:4599"},
		{Host: "cloudmock.localhost", Path: "/", Backend: "http://localhost:4500"},

		{Host: "bff.localhost", Path: "/", Backend: "http://localhost:3202"},
		{Host: "api.localhost", Path: "/", Backend: "http://localhost:4566"},
		{Host: "auth.localhost", Path: "/", Backend: "http://localhost:4566"},
		{Host: "admin.localhost", Path: "/", Backend: "http://localhost:4599"},
		{Host: "graphql.localhost", Path: "/", Backend: "http://localhost:4000"},

		// ---- custom domain: autotend app services ----

		{Host: "autotend-app." + at, Path: "/", Backend: "http://localhost:8081"},
		{Host: "bff." + at, Path: "", Backend: "http://localhost:3202"},
		{Host: "api." + at, Path: "", Backend: "http://localhost:4566"},
		{Host: "auth." + at, Path: "", Backend: "http://localhost:4566"},
		{Host: "admin." + at, Path: "", Backend: "http://localhost:4599"},
		{Host: "graphql." + at, Path: "", Backend: "http://localhost:4000"},
		{Host: at, Path: "/", Backend: "http://localhost:8081"},

		// ---- custom domain: cloudmock dashboard ----

		{Host: cm, Path: "/_cloudmock/", Backend: "http://localhost:4566"},
		{Host: cm, Path: "/api/", Backend: "http://localhost:4599"},
		{Host: cm, Path: "/", Backend: "http://localhost:4500"},
	}
}
```

Also update the `ProxyRoute.Host` comment to:

```go
Host    string // e.g. "bff.localhost" or "bff.localhost.example.com"
```

And update the `StartProxy` HTTP log message to:

```go
log.Printf("proxy HTTP%s: routing via .localhost and custom domains", addr)
```

- [ ] **Step 2: Implement buildSANs, sansMatch, and update EnsureCerts in certs.go**

Add helper functions:

```go
// buildSANs generates the DNS names for the TLS certificate from the given domains.
func buildSANs(domains ...string) []string {
	sans := []string{"localhost", "*.localhost"}
	for _, d := range domains {
		sans = append(sans, "localhost."+d, "*.localhost."+d)
	}
	return sans
}

// sansMatch returns true if all needed SANs are present in the current SAN list.
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
```

Change `EnsureCerts` signature to `EnsureCerts(domains ...string) (*CertPair, error)`.

In the "Try loading existing certs" block, replace the expiration-only check with expiration + SAN comparison:

```go
if parseErr == nil && time.Now().Before(leaf.NotAfter) {
	needed := buildSANs(domains...)
	if sansMatch(leaf.DNSNames, needed) {
		trustCA(caCertPath)
		return &CertPair{Cert: cert, CACert: caCertPath}, nil
	}
	log.Printf("certs: SAN mismatch (have %v, need %v), regenerating", leaf.DNSNames, needed)
}
```

Replace the hardcoded `DNSNames` list with `buildSANs(domains...)`.
Replace the hardcoded `CommonName` with `"localhost." + domains[0]`.
Update the generation log message to `log.Printf("certs: generating self-signed CA and certificate for %v", buildSANs(domains...))`.

- [ ] **Step 3: Verify the full build compiles**

Run: `cd cloudmock && go build ./...`
Expected: Success (main.go already calls both functions with new signatures)

- [ ] **Step 4: Write tests**

Create `cloudmock/pkg/gateway/proxy_test.go`:

```go
package gateway

import (
	"testing"
)

func TestBuildRoutes(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	findRoute := func(host, path string) *ProxyRoute {
		for _, r := range routes {
			if r.Host == host && r.Path == path {
				return &r
			}
		}
		return nil
	}

	// .localhost zero-config routes
	if r := findRoute("autotend-app.localhost", "/"); r == nil {
		t.Error("missing autotend-app.localhost route")
	}
	if r := findRoute("cloudmock.localhost", "/"); r == nil {
		t.Error("missing cloudmock.localhost route")
	}
	if r := findRoute("bff.localhost", "/"); r == nil {
		t.Error("missing bff.localhost route")
	}

	// Custom autotend domain routes
	if r := findRoute("localhost.example.com", "/"); r == nil {
		t.Error("missing localhost.example.com default route")
	} else if r.Backend != "http://localhost:8081" {
		t.Errorf("localhost.example.com should route to :8081, got %s", r.Backend)
	}
	for _, sub := range []string{"bff", "api", "auth", "admin", "graphql"} {
		if r := findRoute(sub+".localhost.example.com", ""); r == nil {
			t.Errorf("missing %s.localhost.example.com route", sub)
		}
	}

	// Custom cloudmock domain routes
	if r := findRoute("localhost.mock.dev", "/"); r == nil {
		t.Error("missing localhost.mock.dev route")
	} else if r.Backend != "http://localhost:4500" {
		t.Errorf("localhost.mock.dev should route to :4500, got %s", r.Backend)
	}

	// No hardcoded domains
	for _, r := range routes {
		if r.Host == "localhost.autotend.io" || r.Host == "localhost.cloudmock.io" {
			t.Errorf("found hardcoded domain in route: %s", r.Host)
		}
	}
}

func TestBuildRoutesOrderMatters(t *testing.T) {
	routes := BuildRoutes("example.com", "mock.dev")

	cloudmockAPIIdx, cloudmockRootIdx := -1, -1
	for i, r := range routes {
		if r.Host == "localhost.mock.dev" && r.Path == "/_cloudmock/" {
			cloudmockAPIIdx = i
		}
		if r.Host == "localhost.mock.dev" && r.Path == "/" {
			cloudmockRootIdx = i
		}
	}
	if cloudmockAPIIdx == -1 || cloudmockRootIdx == -1 {
		t.Fatal("missing cloudmock routes")
	}
	if cloudmockAPIIdx > cloudmockRootIdx {
		t.Error("/_cloudmock/ route must come before / route")
	}
}
```

Create `cloudmock/pkg/gateway/certs_test.go`:

```go
package gateway

import (
	"sort"
	"testing"
)

func TestBuildSANs(t *testing.T) {
	sans := buildSANs("example.com", "mock.dev")
	sort.Strings(sans)

	expected := []string{
		"*.localhost",
		"*.localhost.example.com",
		"*.localhost.mock.dev",
		"localhost",
		"localhost.example.com",
		"localhost.mock.dev",
	}
	sort.Strings(expected)

	if len(sans) != len(expected) {
		t.Fatalf("expected %d SANs, got %d: %v", len(expected), len(sans), sans)
	}
	for i, s := range expected {
		if sans[i] != s {
			t.Errorf("SAN[%d]: expected %q, got %q", i, s, sans[i])
		}
	}
}

func TestSANsMatch(t *testing.T) {
	current := []string{"localhost", "*.localhost", "localhost.example.com", "*.localhost.example.com"}

	if !sansMatch(current, buildSANs("example.com")) {
		t.Error("SANs should match when all needed SANs are present")
	}
	if sansMatch(current, buildSANs("example.com", "other.io")) {
		t.Error("SANs should not match when a needed domain is missing")
	}
}
```

- [ ] **Step 5: Run all gateway tests**

Run: `cd cloudmock && go test ./pkg/gateway/ -v`
Expected: All pass

- [ ] **Step 6: Run full test suite**

Run: `cd cloudmock && go test ./... 2>&1 | tail -20`
Expected: All pass

- [ ] **Step 7: Commit**

```bash
git add cloudmock/pkg/gateway/proxy.go cloudmock/pkg/gateway/proxy_test.go cloudmock/pkg/gateway/certs.go cloudmock/pkg/gateway/certs_test.go
git commit -m "feat: BuildRoutes and EnsureCerts accept domain params dynamically"
```

---

### Task 3: Update config.ts to parse Pulumi YAML

**Files:**
- Modify: `autotend-infra/local/config.ts`

- [ ] **Step 1: Install yaml dependency**

Run: `cd autotend-infra && pnpm add -w yaml`

- [ ] **Step 2: Update config.ts**

Replace the entire file with:

```typescript
import { readFileSync } from "fs";
import { parse } from "yaml";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname_resolved = typeof __dirname !== "undefined"
  ? __dirname
  : dirname(fileURLToPath(import.meta.url));

// Parse domains from Pulumi.local.yaml (single source of truth)
function loadDomains(): { autotend: string; cloudmock: string } {
  const defaults = { autotend: "autotend.io", cloudmock: "cloudmock.io" };
  const yamlPath = resolve(__dirname_resolved, "../pulumi/Pulumi.local.yaml");
  try {
    const content = readFileSync(yamlPath, "utf8");
    const parsed = parse(content);
    const domains = parsed?.config?.["autotend-backend:domains"];
    if (!domains) {
      console.warn(`[config] No domains found in ${yamlPath}, using defaults`);
      return defaults;
    }
    return {
      autotend: domains.autotend || defaults.autotend,
      cloudmock: domains.cloudmock || defaults.cloudmock,
    };
  } catch {
    console.warn(`[config] Could not read ${yamlPath}, using default domains`);
    return defaults;
  }
}

const domains = loadDomains();

export const config = {
  domains,
  cloudmock: {
    endpoint: process.env.CLOUDMOCK_ENDPOINT || "http://localhost:4566",
    adminEndpoint: process.env.CLOUDMOCK_ADMIN || "http://localhost:4599",
    region: "us-east-1",
    accountId: "000000000000",
    accessKeyId: "test",
    secretAccessKey: "test",
  },
  graphql: {
    port: 4000,
  },
  postgres: {
    host: "localhost",
    port: 5432,
    database: "calendar_db",
    user: "postgres",
    password: "postgres",
  },
  bff: {
    port: 3202,
    path: "../../autotend-api-services/services/bff",
  },
  api: {
    port: 3000,
    path: "../../autotend-api-services",
  },
};

export function awsEnv(): Record<string, string> {
  return {
    AWS_ENDPOINT_URL: config.cloudmock.endpoint,
    AWS_ACCESS_KEY_ID: config.cloudmock.accessKeyId,
    AWS_SECRET_ACCESS_KEY: config.cloudmock.secretAccessKey,
    AWS_DEFAULT_REGION: config.cloudmock.region,
    AWS_REGION: config.cloudmock.region,
    ENVIRONMENT: "local",
    LOCAL_DB: "false",
  };
}
```

- [ ] **Step 3: Verify it parses correctly**

Run: `cd autotend-infra/local && npx tsx -e "import {config} from './config'; console.log(config.domains)"`
Expected: `{ autotend: 'autotend.io', cloudmock: 'cloudmock.io' }`

- [ ] **Step 4: Commit**

```bash
git add autotend-infra/local/config.ts autotend-infra/package.json autotend-infra/pnpm-lock.yaml
git commit -m "feat: config.ts reads domains from Pulumi.local.yaml"
```

---

### Task 4: Update dev.ts to use config.domains

**Files:**
- Modify: `autotend-infra/local/dev.ts:53,113-126`

- [ ] **Step 1: Update env vars passed to gateway**

At line 53, the env already has `CLOUDMOCK_DOMAIN_AUTOTEND` and `CLOUDMOCK_DOMAIN_CLOUDMOCK` from `config.domains`. Verify this is correct — it should read:

```typescript
CLOUDMOCK_DOMAIN_AUTOTEND: config.domains.autotend,
CLOUDMOCK_DOMAIN_CLOUDMOCK: config.domains.cloudmock,
```

- [ ] **Step 2: Update connection summary**

Replace the hardcoded summary block (lines 113-126) with:

```typescript
  const at = config.domains.autotend;
  const cm = config.domains.cloudmock;

  console.log("\n" + "=".repeat(60));
  console.log("  autotend local development environment ready!");
  console.log("=".repeat(60));
  console.log(`  cloudmock:  ${config.cloudmock.endpoint}`);
  console.log(`  dashboard:  http://localhost:4500`);
  console.log(`  admin API:  ${config.cloudmock.adminEndpoint}`);
  console.log(`  GraphQL:    http://localhost:${config.graphql.port}/graphql`);
  console.log(`  Postgres:   postgresql://localhost:${config.postgres.port}/${config.postgres.database}`);
  console.log(`  BFF:        http://localhost:${config.bff.port}`);
  console.log();
  console.log("  HTTPS proxy (requires DNS setup: sudo cloudmock-dns auto):");
  console.log(`    https://localhost.${at}                ← autotend app`);
  console.log(`    https://localhost.${cm}              ← cloudmock dashboard`);
  console.log(`    https://bff.localhost.${at}`);
  console.log("\n  Test users:");
  console.log("    admin@test.autotend.io    / TestPass123!");
  console.log("    teacher@test.autotend.io  / TestPass123!");
  console.log("    student@test.autotend.io  / TestPass123!");
  console.log("\n  Press Ctrl+C to stop all services\n");
```

- [ ] **Step 3: Verify dev.ts runs without errors**

Run: `cd autotend-infra/local && npx tsx -e "import {config} from './config'; console.log('domains:', config.domains)"`
Expected: `domains: { autotend: 'autotend.io', cloudmock: 'cloudmock.io' }`

- [ ] **Step 4: Commit**

```bash
git add autotend-infra/local/dev.ts
git commit -m "feat: dev.ts uses config.domains for env vars and summary"
```

---

### Task 5: cloudmock-dns — add --config flag

**Files:**
- Modify: `cloudmock/tools/cloudmock-dns/main.go`

- [ ] **Step 1: Add YAML parsing and --config flag**

Add imports at the top:

```go
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
```

Replace the hardcoded `resolverFiles` and `entries` vars with a config-driven approach. Add after imports:

```go
const marker = "# cloudmock local development"
const hostsFile = "/etc/hosts"
const resolverDir = "/etc/resolver"
const baseDNSPort = 15353

// domainConfig holds the parsed domain configuration.
type domainConfig struct {
	Autotend string
	Cloudmock string
}

var defaultDomains = domainConfig{
	Autotend:  "autotend.io",
	Cloudmock: "cloudmock.io",
}

// parsePulumiConfig reads domains from a Pulumi stack YAML file.
func parsePulumiConfig(path string) (domainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultDomains, err
	}
	var raw struct {
		Config map[string]interface{} `yaml:"config"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return defaultDomains, fmt.Errorf("parse %s: %w", path, err)
	}
	domainsRaw, ok := raw.Config["autotend-backend:domains"]
	if !ok {
		return defaultDomains, fmt.Errorf("no autotend-backend:domains in %s", path)
	}
	domainsMap, ok := domainsRaw.(map[string]interface{})
	if !ok {
		return defaultDomains, fmt.Errorf("domains is not a map in %s", path)
	}
	dc := defaultDomains
	if v, ok := domainsMap["autotend"].(string); ok {
		dc.Autotend = v
	}
	if v, ok := domainsMap["cloudmock"].(string); ok {
		dc.Cloudmock = v
	}
	return dc, nil
}

// sortedDomains returns domains sorted alphabetically by config key for
// deterministic port assignment.
func (dc domainConfig) sortedDomains() []struct{ key, domain string } {
	pairs := []struct{ key, domain string }{
		{"autotend", dc.Autotend},
		{"cloudmock", dc.Cloudmock},
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
	return pairs
}

// resolverEntries generates the macOS resolver file entries.
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

// hostsEntries generates /etc/hosts lines for all service subdomains.
func (dc domainConfig) hostsEntries() []string {
	at := "localhost." + dc.Autotend
	cm := "localhost." + dc.Cloudmock
	return []string{
		"127.0.0.1  " + at,
		"127.0.0.1  autotend-app." + at,
		"127.0.0.1  bff." + at,
		"127.0.0.1  api." + at,
		"127.0.0.1  auth." + at,
		"127.0.0.1  admin." + at,
		"127.0.0.1  graphql." + at,
		"127.0.0.1  " + cm,
	}
}
```

Update `main()` to parse the `--config` flag and load domains:

```go
var configPath string

func main() {
	fs := flag.NewFlagSet("cloudmock-dns", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "", "path to Pulumi stack YAML config file")
	// Parse flags after the subcommand
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
```

Update `printUsage` to mention `--config`:

```go
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
```

Update all functions (`autoSetup`, `autoSetupMacOS`, `setup`, `remove`, `status`, `printLocalDomains`, `autoSetupLinux`) to accept `domainConfig` parameter and use `dc.resolverEntries()`, `dc.hostsEntries()`, and dynamic domain strings instead of hardcoded globals.

In `remove()`, replace the hardcoded domain check for `/etc/hosts` cleanup:

```go
// Replace:
//   if (strings.Contains(line, "autotend.io") || strings.Contains(line, "cloudmock.io")) && ...
// With:
if strings.HasPrefix(strings.TrimSpace(line), "127.0.0.1") &&
	(strings.Contains(line, "localhost."+dc.Autotend) || strings.Contains(line, "localhost."+dc.Cloudmock)) {
```

Update the sudo re-exec in `autoSetupMacOS` to forward the `--config` flag:

```go
sudoArgs := []string{os.Args[0], "_internal_resolver_setup"}
if configPath != "" {
	sudoArgs = append(sudoArgs, "--config", configPath)
}
cmd := exec.Command("sudo", sudoArgs...)
```

Update `init()` to use `flag.FlagSet` for parsing `--config` when re-execed:

```go
func init() {
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
```

- [ ] **Step 2: Build and verify**

Run: `cd cloudmock && go build ./tools/cloudmock-dns/`
Expected: Success

- [ ] **Step 3: Write unit test for parsePulumiConfig**

Create `cloudmock/tools/cloudmock-dns/config_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePulumiConfig(t *testing.T) {
	yaml := `config:
  autotend-backend:environment: local
  autotend-backend:domains:
    autotend: custom.example.com
    cloudmock: mock.example.com
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Autotend != "custom.example.com" {
		t.Errorf("expected custom.example.com, got %s", dc.Autotend)
	}
	if dc.Cloudmock != "mock.example.com" {
		t.Errorf("expected mock.example.com, got %s", dc.Cloudmock)
	}
}

func TestParsePulumiConfigMissing(t *testing.T) {
	dc, err := parsePulumiConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
	// Should return defaults
	if dc.Autotend != "autotend.io" || dc.Cloudmock != "cloudmock.io" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}

func TestParsePulumiConfigNoDomains(t *testing.T) {
	yaml := `config:
  autotend-backend:environment: local
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err == nil {
		t.Error("expected error when domains key is missing")
	}
	if dc.Autotend != "autotend.io" || dc.Cloudmock != "cloudmock.io" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd cloudmock && go test ./tools/cloudmock-dns/ -v`
Expected: All pass

- [ ] **Step 5: Test with --config flag (manual)**

Run: `cd cloudmock && go run ./tools/cloudmock-dns/ status --config ../autotend-infra/pulumi/Pulumi.local.yaml`
Expected: Shows resolver status using domains from the YAML file

- [ ] **Step 6: Test without --config (fallback)**

Run: `cd cloudmock && go run ./tools/cloudmock-dns/ status`
Expected: Shows resolver status using default domains

- [ ] **Step 7: Commit**

```bash
git add cloudmock/tools/cloudmock-dns/main.go cloudmock/tools/cloudmock-dns/config_test.go
git commit -m "feat: cloudmock-dns reads domains from Pulumi config via --config flag"
```

---

### Task 6: Delete old certs and integration test

**Files:** None (manual verification)

Existing certs have hardcoded SANs. The new SAN comparison logic will detect the mismatch and regenerate on next gateway start.

- [ ] **Step 1: Delete existing certs**

Run: `rm -rf ~/.cloudmock/certs/`

- [ ] **Step 2: Rebuild cloudmock-dns binary**

Run: `cd cloudmock && go build -o bin/cloudmock-dns ./tools/cloudmock-dns/`

- [ ] **Step 3: Run full test suite**

Run: `cd cloudmock && go test ./... 2>&1 | tail -5`
Expected: All pass

- [ ] **Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "chore: clean up after dynamic domain config migration"
```
