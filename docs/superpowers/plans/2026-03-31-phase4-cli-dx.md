# Phase 4: CLI + Developer Experience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One-command install and run. `npx cloudmock` / `brew install cloudmock` / Docker all work. `cmk` CLI wrapper (like LocalStack's `awslocal`). Stellar README. Must be as easy as LocalStack to bootstrap.

**Architecture:** npm package with platform-detect + binary download. Homebrew tap with pre-built bottles. `cmk` shell wrapper. README rewrite.

**Tech Stack:** Node.js (npm package), shell script (cmk), Go (startup banner), Makefile

**Spec:** `docs/superpowers/specs/2026-03-31-v1-saas-launch-design.md` Phase 4

---

## Task 1: Add startup banner to cloudmock gateway

**Files:**
- Modify: `cloudmock/cmd/gateway/main.go`

- [ ] **Step 1: Find where the gateway starts listening**

Read `cmd/gateway/main.go` and find where the HTTP servers start (gateway, admin, dashboard).

- [ ] **Step 2: Add startup banner**

After all servers are started, print:

```
CloudMock v1.0.0
  Gateway:    http://localhost:4566
  Devtools:   http://localhost:4500  <-- open in browser
  Admin API:  http://localhost:4599
  Services:   25 active (standard profile)

Ready. Point your AWS SDK at http://localhost:4566
```

Use the actual configured ports from `cfg.Gateway.Port`, `cfg.Dashboard.Port`, `cfg.Admin.Port`. Count active services from the registry.

- [ ] **Step 3: Build and verify**

```bash
go build ./cmd/gateway/
./gateway --help  # verify it still works
```

- [ ] **Step 4: Commit**

```bash
git add cmd/gateway/main.go
git commit -m "feat: add startup banner with ports and service count"
```

---

## Task 2: Create `cmk` CLI wrapper

**Files:**
- Create: `cloudmock/bin/cmk`

- [ ] **Step 1: Write the cmk shell script**

```bash
#!/usr/bin/env bash
# cmk - CloudMock CLI wrapper
# Like LocalStack's awslocal, wraps the AWS CLI with --endpoint-url

CLOUDMOCK_ENDPOINT="${CLOUDMOCK_ENDPOINT:-http://localhost:4566}"

exec aws --endpoint-url="$CLOUDMOCK_ENDPOINT" "$@"
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x bin/cmk
```

- [ ] **Step 3: Test it**

```bash
# Verify it wraps aws cli correctly (will fail if cloudmock isn't running, that's fine)
./bin/cmk sts get-caller-identity 2>&1 || true
```

- [ ] **Step 4: Commit**

```bash
git add bin/cmk
git commit -m "feat: add cmk CLI wrapper (like awslocal for LocalStack)"
```

---

## Task 3: Create npm package for `npx cloudmock`

**Files:**
- Create: `cloudmock/npm/cloudmock/package.json`
- Create: `cloudmock/npm/cloudmock/bin/cloudmock.mjs`
- Create: `cloudmock/npm/cloudmock/README.md`

- [ ] **Step 1: Create package.json**

```json
{
  "name": "cloudmock",
  "version": "1.0.0",
  "description": "Local AWS emulation. 25 services. One command.",
  "bin": { "cloudmock": "bin/cloudmock.mjs" },
  "files": ["bin/"],
  "license": "Apache-2.0",
  "repository": { "type": "git", "url": "https://github.com/Viridian-Inc/cloudmock" },
  "keywords": ["aws", "mock", "local", "development", "testing", "s3", "dynamodb", "lambda"]
}
```

- [ ] **Step 2: Create the bin script**

`bin/cloudmock.mjs` — platform-detect + binary download + execute:

1. Detect `process.platform` (darwin/linux/win32) and `process.arch` (arm64/x64)
2. Map to GitHub release binary name (e.g., `cloudmock-darwin-arm64`)
3. Check cache at `~/.cloudmock/bin/cloudmock-{version}-{platform}-{arch}`
4. If not cached: download from `https://github.com/Viridian-Inc/cloudmock/releases/latest/download/{binary}`
5. Make executable (`chmod +x`)
6. Spawn the binary with `process.argv.slice(2)` as args, inheriting stdio

Handle errors gracefully: show download progress, handle network failures, suggest Docker as fallback.

- [ ] **Step 3: Test locally**

```bash
cd npm/cloudmock && node bin/cloudmock.mjs --help
```

- [ ] **Step 4: Commit**

```bash
git add npm/
git commit -m "feat: add npm package for npx cloudmock (platform-detect + binary download)"
```

---

## Task 4: Create Homebrew tap

**Files:**
- Create: `cloudmock/homebrew/cloudmock.rb`

- [ ] **Step 1: Write Homebrew formula**

```ruby
class Cloudmock < Formula
  desc "Local AWS emulation. 25 services. One binary."
  homepage "https://cloudmock.io"
  version "1.0.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.0/cloudmock-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.0/cloudmock-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.0/cloudmock-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.0/cloudmock-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "cloudmock"
    bin.install "cmk"
  end

  test do
    assert_match "CloudMock", shell_output("#{bin}/cloudmock --version")
  end
end
```

SHA256 values will be filled in during the release workflow.

- [ ] **Step 2: Commit**

```bash
git add homebrew/
git commit -m "feat: add Homebrew formula for brew install cloudmock"
```

---

## Task 5: Rewrite README.md

**Files:**
- Modify: `cloudmock/README.md`

- [ ] **Step 1: Read current README**

Read `README.md` to understand current structure and content.

- [ ] **Step 2: Rewrite following this structure**

1. **Title + one-line**: `# CloudMock` / `Local AWS. 25 services. One binary.`
2. **Install** (4 methods in a tabbed/section format):
   ```bash
   npx cloudmock          # zero install
   brew install cloudmock  # macOS/Linux
   docker run ...          # container
   go install ...          # from source
   ```
3. **"Point your SDK"** — 3 language examples (Node, Python, Go), 5-10 lines each
4. **cmk CLI** — show `cmk s3 ls` vs `aws --endpoint-url=... s3 ls`
5. **Open Devtools** — `open http://localhost:4500` with a screenshot
6. **Service table** — all 25 Tier 1 services in a clean table
7. **Configuration basics** — profiles, ports, persistence (3-4 lines)
8. **Links**: [Documentation](https://cloudmock.io/docs) | [Comparison](https://cloudmock.io/docs/reference/comparison)
9. **Contributing**
10. **License** (Apache-2.0)

Rules: No emoji. No AI fluff. No "awesome" or "blazing fast". Technical, direct, copy-paste ready. Every command must actually work.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README for v1 launch (1-minute install, no fluff)"
```

---

## Task 6: Update GitHub Actions release workflow

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Read current release workflow**

Read `.github/workflows/release.yml` to understand existing build + publish steps.

- [ ] **Step 2: Add npm publish step**

After building binaries and Docker image, add:
```yaml
- name: Publish npm package
  run: |
    cd npm/cloudmock
    npm publish --access public
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

- [ ] **Step 3: Add Homebrew formula update step**

After release assets are uploaded, update SHA256 in formula and push to tap repo (or just note this as a manual step for v1).

- [ ] **Step 4: Ensure cmk is included in release archives**

The `make release` target builds platform binaries. Verify `bin/cmk` is included in the tar.gz archives alongside the cloudmock binary.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add npm publish and cmk to release workflow"
```

---

## Task 7: Final verification

- [ ] **Step 1: Build everything**

```bash
make build
```

- [ ] **Step 2: Test startup banner**

```bash
./gateway &
# Verify banner prints with correct ports
kill %1
```

- [ ] **Step 3: Test cmk wrapper**

```bash
./bin/cmk --version  # should show aws-cli version
```

- [ ] **Step 4: Test npm package locally**

```bash
cd npm/cloudmock && node bin/cloudmock.mjs --version
```

- [ ] **Step 5: Verify README renders correctly**

View README.md in a markdown viewer. Verify all commands are correct. Verify no emoji, no fluff.

- [ ] **Step 6: Commit any final fixes**

```bash
git commit -m "feat: Phase 4 complete — CLI + DX (npx, brew, cmk, README)"
```

---

## Verification

1. Startup banner shows ports + service count
2. `cmk s3 ls` works (wraps aws CLI with endpoint)
3. npm package structure is valid (`npm pack` in npm/cloudmock/)
4. Homebrew formula has correct structure
5. README: 1-minute install, no fluff, all commands correct
6. Release workflow includes npm publish
7. `make build` succeeds
