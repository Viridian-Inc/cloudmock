# Production Readiness Progress

## Phase 1: Critical Bug Fixes ✅
- [x] 1.1 Fix silent error swallowing (30+ catches across 15 files)
- [x] 1.2 Fix stale closure in use-topology-metrics.ts (uses timeWindowRef)
- [x] 1.3 Fix SSE handler stale paused ref (uses pausedRef.current)
- [x] 1.4 Remove hardcoded LAMBDA_NAMES (dynamic friendlyLambdaName)
- [x] 1.5 Fix health.ts P99 threshold (configurable via parameter)

## Phase 2: High-Severity Fixes
- [x] 2.1 Fix Tauri import failures (logs informative fallback messages)
- [x] 2.2 Fix stale getAdminBase() (captured fresh at call time)
- [x] 2.3 Fix connection-picker port assumption (smart :4566→:4599)
- [x] 2.4 Fix split-panel pointer tracking when cursor exits viewport

## Phase 3: Data Quality
- [x] 3.1 API failure logging (all catches have context logging)
- [x] 3.2 Data freshness indicator in status bar
- [x] 3.3 Fix SSE vs polling dedup (stable IDs from TraceID)

## Phase 4: Test Setup + Tests ✅
- [x] 4.1 Set up Vitest (jsdom, globals, preact preset)
- [x] 4.2.1 health.test.ts (27 tests)
- [x] 4.2.2 api.test.ts (6 tests)
- [x] 4.2.3 domains.test.ts (15 tests)
- [x] 4.2.4 event-detail.test.ts (27 tests)
- [x] 4.2.5 collapse.test.ts (10 tests)
- [x] 4.2.6 budget.test.ts (17 tests)
- [x] 4.2.7 timer.test.ts (24 tests)
- [x] 4.2.8 percentile.test.ts (22 tests)
- [x] 4.2.9 waterfall.test.ts (16 tests)
- [x] 4.2.10 routing.test.ts (25 tests)

## Phase 5: Documentation
- [x] 5.1 PROGRESS.md created
- [x] 5.2 BUGS.md created
- [x] 5.3 TODO.md synced with audit

## Phase 6: Final Bug Fixes
- [x] 6.1 Fix markdown rendering in ai-debug (two-pass: code blocks then inline)
- [x] 6.2 Fix hash collisions for service colors (djb2 hash, 5 files)
- [x] 6.3 Standardize types.ts naming conventions (JSDoc comments)

## Final Verification
- [x] `pnpm test` — 189 tests passing
- [x] `npx tsc --noEmit` — zero errors
- [x] Zero silent catches
- [x] Zero mocked data
- [x] All audit bugs resolved (15/15)
- [ ] All 12 views verified with real data
