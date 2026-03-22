# Dynamic Domain Configuration from Pulumi IaC

**Date:** 2026-03-21
**Status:** Approved
**Scope:** Local development domain configuration

## Problem

Domain names (`autotend.io`, `cloudmock.io`) are hardcoded across multiple files in cloudmock and the local dev orchestrator. Changes require editing 5+ files. The domains should be defined once in IaC config and flow to all consumers.

## Decision

**Single source of truth:** Pulumi stack config files (`Pulumi.*.yaml`).

A `Pulumi.local.yaml` stack defines domains for local development. All consumers — the orchestrator, cloudmock gateway, proxy, certs, DNS — read from this config rather than hardcoding domain strings.

## Config Schema

Each `Pulumi.*.yaml` gains a `domains` key:

```yaml
# Pulumi.local.yaml
config:
  autotend-backend:environment: local
  autotend-backend:domains:
    autotend: autotend.io
    cloudmock: cloudmock.io
```

The same structure applies to `Pulumi.dev.yaml`, `Pulumi.stage.yaml`, and `Pulumi.prod.yaml`, allowing domains to differ per environment if needed.

## Architecture

```
Pulumi.local.yaml
       |
       +---> autotend-infra/local/config.ts   (parses YAML, exports domains)
       |         |
       |         +---> dev.ts orchestrator     (passes as env vars to gateway)
       |         \---> dev.ts connection summary (prints correct URLs)
       |
       +---> cloudmock gateway                 (reads env vars at startup)
       |         +---> proxy.go BuildRoutes()  (generates route table)
       |         +---> certs.go EnsureCerts()  (generates SANs)
       |         \---> dns.go StartDNSServer() (one server per domain)
       |
       \---> cloudmock-dns tool                (parses YAML via --config flag)
                 +---> /etc/resolver/<domain>  (macOS resolver files)
                 \---> /etc/hosts entries      (legacy fallback)
```

## Fallback Behavior

If `Pulumi.local.yaml` does not exist or lacks the `domains` key:
- `config.ts` falls back to `{ autotend: "autotend.io", cloudmock: "cloudmock.io" }` and logs a warning.
- `cloudmock-dns` without `--config` falls back to the same defaults.
- Gateway env vars default to `autotend.io` and `cloudmock.io` if unset.

This ensures a fresh clone works without creating `Pulumi.local.yaml` first.

## Changes by File

### New Files

**`autotend-infra/pulumi/Pulumi.local.yaml`**
- New Pulumi stack config for local development.
- Defines `autotend-backend:domains` with `autotend` and `cloudmock` keys.

### Modified Files

**`autotend-infra/local/config.ts`**
- Add YAML parser dependency (`yaml` package).
- Parse `../pulumi/Pulumi.local.yaml` using `__dirname`-relative path resolution (not cwd-relative, since `dev.ts` may be invoked from different directories).
- Export `config.domains.autotend` and `config.domains.cloudmock`.
- Remove hardcoded domain values.
- Fall back to defaults if file is missing (see Fallback Behavior).

**`autotend-infra/local/dev.ts`**
- Pass `CLOUDMOCK_DOMAIN_AUTOTEND` and `CLOUDMOCK_DOMAIN_CLOUDMOCK` env vars to the gateway process, sourced from `config.domains`.
- Update connection summary to use domain values from config (all six URL references in the summary block).
- Test user email domains (`@test.autotend.io`) are out of scope — these are identity data, not routing configuration.

**`cloudmock/pkg/gateway/proxy.go`**
- Replace `DefaultAutotendRoutes()` with `BuildRoutes(autotendDomain, cloudmockDomain string) []ProxyRoute`.
- Route table generated dynamically from domain parameters.
- `.localhost` zero-config routes (RFC 6761) remain hardcoded — they are derived from service names, not domain config.
- Route mapping:
  - `localhost.<autotendDomain>` -> Expo app (:8081)
  - `autotend-app.localhost.<autotendDomain>` -> Expo app (:8081)
  - `bff.localhost.<autotendDomain>` -> BFF (:3202)
  - `api.localhost.<autotendDomain>` -> AWS API (:4566)
  - `auth.localhost.<autotendDomain>` -> Cognito (:4566)
  - `admin.localhost.<autotendDomain>` -> Admin API (:4599)
  - `graphql.localhost.<autotendDomain>` -> GraphQL (:4000)
  - `localhost.<cloudmockDomain>` -> cloudmock dashboard (:4500), with `/_cloudmock/` -> :4566 and `/api/` -> :4599
- Update `StartProxy` log to use domain parameters instead of hardcoded strings.

**`cloudmock/pkg/gateway/certs.go`**
- Change `EnsureCerts()` signature to `EnsureCerts(domains ...string) (*CertPair, error)`.
- Generate SAN entries dynamically: for each domain, add `localhost.<domain>` and `*.localhost.<domain>`.
- Always include `*.localhost`, `localhost`, and `127.0.0.1`.
- Add SAN comparison logic: parse the existing cert's SAN list and compare as a set against the expected SANs. This is new behavior — the current code only checks expiration. If SANs don't match, regenerate.
- CA trust flow (login keychain on macOS, no sudo) remains unchanged.

**`cloudmock/cmd/gateway/main.go`**
- Read `CLOUDMOCK_DOMAIN_AUTOTEND` and `CLOUDMOCK_DOMAIN_CLOUDMOCK` env vars.
- Pass to `BuildRoutes()` and `EnsureCerts()`.
- Start one DNS server per domain on sequential ports (15353, 15354, ...).
- Note: `main.go` has already been partially updated to call new signatures. The `proxy.go` and `certs.go` changes must land in the same commit to avoid a broken build.

**`cloudmock/tools/cloudmock-dns/main.go`**
- Add `--config <path>` flag that points to a `Pulumi.*.yaml` file.
- Parse the YAML to extract domain names from `autotend-backend:domains`.
- Generate resolver files, hosts entries, `printLocalDomains()` output, and `autoSetupLinux()` instructions dynamically from parsed domains.
- Each domain gets its own `/etc/resolver/<domain>` file with a unique port.
- Remove all hardcoded domain strings and hosts entries.
- Maintain backwards compatibility: if `--config` is not provided, fall back to default domains (`autotend.io`, `cloudmock.io`).
- sudo re-exec (`_internal_resolver_setup`): forward the `--config` flag when re-execing with sudo, so the subprocess can re-parse the same config and generate correct resolver files.

### Unchanged

- `.localhost` RFC 6761 zero-config routes — these use service names (`autotend-app.localhost`, `cloudmock.localhost`, `bff.localhost`, etc.) and are not domain-dependent.
- Pulumi prod/dev/stage modules — refactoring those to read from config is out of scope (separate effort).
- `cloudmock.yml` — gateway config file is not involved in domain configuration.
- Crossplane CRD API groups (`s3.cloudmock.io`, `ec2.cloudmock.io`, etc.) — these are Kubernetes API group names, not routing domains, and are intentionally not dynamic.
- Test user email domains (`@test.autotend.io`) — identity data, not routing configuration.

## Domain-to-Port Mapping

DNS servers are assigned sequential ports starting at 15353. Domains are sorted alphabetically by config key to ensure deterministic port assignment across all consumers (`config.ts`, `cmd/gateway/main.go`, and `cloudmock-dns`).

| Config Key | Domain | DNS Port | Resolver File |
|------------|--------|----------|---------------|
| autotend | autotend.io | 15353 | /etc/resolver/autotend.io |
| cloudmock | cloudmock.io | 15354 | /etc/resolver/cloudmock.io |

Additional domains (if added to config) would get 15355, 15356, etc., assigned alphabetically by key.

## Cert Regeneration

Existing certs at `~/.cloudmock/certs/` are reused if:
1. Not expired
2. SAN list covers all requested `localhost.<domain>` and `*.localhost.<domain>` entries

SAN comparison is new logic: parse the existing cert's x509 DNSNames field and compare as a set against the expected SANs generated from the domain list. If any expected SAN is missing, regenerate the cert and re-trust the CA.

## Implementation Order

Changes must be applied in this order to maintain a compilable build at each step:

1. Create `Pulumi.local.yaml`
2. Update `proxy.go` — rename `DefaultAutotendRoutes` to `BuildRoutes` with domain params
3. Update `certs.go` — add domain params to `EnsureCerts`, add SAN comparison
4. Update `cmd/gateway/main.go` — already partially done, reconcile with new signatures
5. Update `config.ts` — add YAML parsing with `__dirname`-relative path
6. Update `dev.ts` — use `config.domains` for env vars and summary
7. Update `cloudmock-dns` — add `--config` flag, forward on sudo re-exec

Steps 2-4 must land atomically (same commit) since `main.go` already references the new signatures.

## Testing

- Unit test for `BuildRoutes()` — verify correct routes generated for given domains.
- Unit test for SAN generation in `EnsureCerts()` — verify correct DNS names for given domains.
- Unit test for SAN comparison — verify regeneration triggered when domains change.
- Integration: start gateway with custom domain env vars, verify proxy routes respond correctly.
- `cloudmock-dns --config` parses sample YAML and generates correct resolver content.
- `cloudmock-dns --config` with sudo re-exec forwards the flag correctly.
- Fallback: verify `config.ts` works without `Pulumi.local.yaml` present.
