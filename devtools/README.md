# neureaux devtools

Cross-platform developer tools — unified debugging for iOS, Android, Web & Cloud.

A Tauri v2 desktop app powered by cloudmock.

## Prerequisites

- [Node.js](https://nodejs.org/) 18+
- [pnpm](https://pnpm.io/) 9+
- [Rust](https://rustup.rs/) (for Tauri backend)
- [cloudmock](../cloudmock/) gateway binary (`make build-gateway` in the cloudmock directory)

## Setup

```bash
# Install Rust (if not installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Install dependencies
pnpm install

# Build cloudmock gateway (if not already built)
cd ../cloudmock && make build-gateway && cd ../neureaux-devtools
```

## Development

```bash
# Start in dev mode (hot reload frontend + Rust backend)
pnpm tauri dev

# Or just the frontend (no Tauri shell)
pnpm dev
```

## Build

```bash
# Production build (macOS .app + .dmg)
pnpm tauri build
```

## Architecture

```
neureaux-devtools/
├── src/                    # Preact frontend
│   ├── components/         # Shared UI (icon rail, panels, status bar)
│   ├── views/              # Activity, Services, Settings, ...
│   ├── hooks/              # Tauri IPC + SSE hooks
│   └── lib/                # API client, types
├── src-tauri/              # Rust backend
│   └── src/                # Process manager, health monitor, IPC commands
└── sdk/                    # Client SDKs (future)
```

The app manages a local cloudmock gateway process and connects to its admin API (:4599) for service inspection, request logging, and cloud resource browsing.

## Connection Modes

- **Local** — starts cloudmock on your machine (default)
- **cloudmock.io** — connects to a hosted instance (coming soon)
- **Custom** — any cloudmock instance by URL
