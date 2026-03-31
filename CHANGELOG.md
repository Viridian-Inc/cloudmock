# Changelog

## v1.0.0 (2026-03-31)

### Added
- 25 Tier 1 AWS services with comprehensive test coverage (1,876+ tests)
- Built-in browser devtools at localhost:4500 (topology, traces, metrics, chaos, and 8 more views)
- Starlight documentation site (46 pages) at cloudmock.io/docs
- `cmk` CLI wrapper (like awslocal for LocalStack)
- `npx cloudmock` zero-install support
- Homebrew formula for macOS/Linux
- 6 language guides (Node.js, Go, Python, Swift, Kotlin, Dart)
- Node.js, Go, Python SDKs for request capture
- AppSync promoted to Tier 1 (27 operations)
- Source server for SDK-captured HTTP requests (POST /api/source/events)
- Admin API + devtools on single port :4500 (no CORS)
- Startup banner showing ports and service count
- Homepage and pricing page at cloudmock.io

### Changed
- Devtools migrated from Tauri desktop app to browser-only SPA
- Old React dashboard replaced by Preact devtools UI
- README rewritten for 1-minute install experience

### Fixed
- Edge service filtering (split(':')[0] → .pop()!)
- Request trace panel state machine (useReducer replaces 10 useState)
- Replay via admin API (no CORS)
- Requests disappearing from topology panel
- OPTIONS request filtering
