# Phase 3: Documentation Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Starlight (Astro) docs site at cloudmock.io/docs. 25 per-service reference pages, getting started guide, SDK/language guides, configuration reference, admin API reference. Search working.

**Architecture:** New `website/` directory at cloudmock repo root. Astro + Starlight. Migrate existing markdown docs from `docs/` into Starlight content structure. Deploy to Cloudflare Pages.

**Tech Stack:** Astro, Starlight, Node.js, Cloudflare Pages

**Spec:** `docs/superpowers/specs/2026-03-31-v1-saas-launch-design.md` Phase 3

---

## Task 1: Scaffold Starlight project

**Files:**
- Create: `website/` directory with Astro + Starlight

- [ ] **Step 1: Create Starlight project**

```bash
cd /Users/megan/work/neureaux/cloudmock
npm create astro@latest website -- --template starlight --no-install --no-git
cd website && npm install
```

- [ ] **Step 2: Configure starlight in astro.config.mjs**

Set up sidebar navigation, site title, social links:
- Title: "CloudMock"
- Sidebar groups: Getting Started, Services (25), Devtools, Language Guides, Reference
- Social: GitHub repo link

- [ ] **Step 3: Verify dev server works**

```bash
cd website && npm run dev
```

- [ ] **Step 4: Commit**

```bash
git add website/
git commit -m "docs: scaffold Starlight docs site"
```

---

## Task 2: Getting Started pages

**Files:**
- Create: `website/src/content/docs/getting-started/installation.md`
- Create: `website/src/content/docs/getting-started/first-request.md`
- Create: `website/src/content/docs/getting-started/with-your-stack.md`

- [ ] **Step 1: Write installation.md**

Cover all 4 install methods (npx, brew, Docker, go install) with copy-paste commands. Include system requirements. Show the startup output. Link to first-request.

- [ ] **Step 2: Write first-request.md**

30-second tutorial: create an S3 bucket, upload a file, list objects. Using curl + AWS CLI. Show expected output.

- [ ] **Step 3: Write with-your-stack.md**

SDK configuration for Node.js (AWS SDK v3), Python (boto3), Go (aws-sdk-go-v2), Java (AWS SDK v2). Each: 5-10 lines showing how to point the SDK at localhost:4566.

- [ ] **Step 4: Build and verify**

```bash
cd website && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add website/src/content/docs/getting-started/
git commit -m "docs: add Getting Started guides (install, first request, SDK config)"
```

---

## Task 3: Migrate 25 service reference pages

**Files:**
- Create: `website/src/content/docs/services/*.md` (25 files)

- [ ] **Step 1: Read existing docs**

Read `docs/services/` to understand current structure. There are 26 service-specific directories with existing documentation.

- [ ] **Step 2: Create per-service pages**

For each of the 25 Tier 1 services, create a markdown page following this template:

```markdown
---
title: S3
description: Amazon S3 (Simple Storage Service) emulation
---

## Overview
One-line description of what this service does.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateBucket | Supported | |
| PutObject | Supported | Max 5GB |
| ... | ... | ... |

## Quick Start

### curl
\`\`\`bash
curl -X PUT http://localhost:4566/my-bucket
\`\`\`

### Node.js
\`\`\`typescript
import { S3Client, CreateBucketCommand } from '@aws-sdk/client-s3';
const s3 = new S3Client({ endpoint: 'http://localhost:4566', ... });
await s3.send(new CreateBucketCommand({ Bucket: 'my-bucket' }));
\`\`\`

### Python
\`\`\`python
import boto3
s3 = boto3.client('s3', endpoint_url='http://localhost:4566', ...)
s3.create_bucket(Bucket='my-bucket')
\`\`\`

## Configuration
Service-specific config options from cloudmock.yml.

## Known Differences from AWS
List any behavioral differences.

## Error Codes
| Code | Description |
|------|-------------|
| NoSuchBucket | The specified bucket does not exist |
| ... | ... |
```

Pull operation lists from `docs/compatibility-matrix.md`. Pull details from existing `docs/services/` directories.

- [ ] **Step 3: Build and verify**

```bash
cd website && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add website/src/content/docs/services/
git commit -m "docs: add 25 Tier 1 service reference pages"
```

---

## Task 4: Devtools guide pages

**Files:**
- Create: `website/src/content/docs/devtools/*.md` (6-8 pages)

- [ ] **Step 1: Write overview.md**

What the devtools are, how to access (localhost:4500), screenshot of topology view.

- [ ] **Step 2: Write per-view pages**

One page per major view: topology, activity, traces, metrics, chaos. Brief description + what it shows + how to use it.

- [ ] **Step 3: Commit**

```bash
git add website/src/content/docs/devtools/
git commit -m "docs: add devtools UI guide pages"
```

---

## Task 5: Language guide pages

**Files:**
- Create: `website/src/content/docs/language-guides/*.md` (6 pages)

- [ ] **Step 1: Write guides for Node, Go, Python**

These have dedicated CloudMock SDKs (in `neureaux-devtools/sdk/`). Cover: install SDK, init, middleware setup, what gets captured.

- [ ] **Step 2: Write guides for Swift, Kotlin, Dart**

These are AWS SDK configuration guides (point at localhost:4566). No custom SDK needed.

- [ ] **Step 3: Commit**

```bash
git add website/src/content/docs/language-guides/
git commit -m "docs: add 6 language guide pages (Node, Go, Python, Swift, Kotlin, Dart)"
```

---

## Task 6: Reference pages

**Files:**
- Create: `website/src/content/docs/reference/configuration.md`
- Create: `website/src/content/docs/reference/admin-api.md`
- Create: `website/src/content/docs/reference/plugins.md`
- Create: `website/src/content/docs/reference/comparison.md`

- [ ] **Step 1: Write configuration.md**

Migrate from existing `docs/configuration.md`. Cover cloudmock.yml structure, profiles, ports, persistence, IAM modes.

- [ ] **Step 2: Write admin-api.md**

Migrate from existing `docs/admin-api.md`. List all 46+ endpoints with method, path, description, example request/response.

- [ ] **Step 3: Write plugins.md**

Migrate from existing docs. Cover plugin types (in-process, gRPC), how to build one, example.

- [ ] **Step 4: Write comparison.md**

Honest comparison table: CloudMock vs LocalStack vs Moto vs SAM Local. Feature matrix, pricing, service coverage.

- [ ] **Step 5: Commit**

```bash
git add website/src/content/docs/reference/
git commit -m "docs: add reference pages (config, API, plugins, comparison)"
```

---

## Task 7: Final build and verification

- [ ] **Step 1: Full build**

```bash
cd website && npm run build
```
Expected: Builds with zero errors, generates static site in `dist/`

- [ ] **Step 2: Verify page count**

Expected: 40+ pages (3 getting started + 25 services + 6 devtools + 6 language guides + 4 reference + index)

- [ ] **Step 3: Verify search works**

Starlight includes built-in search (Pagefind). Verify it indexes all pages.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "docs: Phase 3 complete — Starlight docs site with 40+ pages"
```

---

## Verification

1. `cd website && npm run build` — zero errors
2. 40+ documentation pages generated
3. Search indexes all service pages
4. Getting started guide: install → first request in under 60 seconds
5. All 25 Tier 1 services have reference pages with operation tables
6. 6 language guides (3 with SDKs, 3 with config guides)
7. Configuration, API, plugin, comparison reference pages complete
