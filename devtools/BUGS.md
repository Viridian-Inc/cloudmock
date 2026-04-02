# Discovered Bugs & Nice-to-Haves

## Bugs (found during audit)

- [x] **CRIT** use-sse.ts:43 — empty catch → console.warn with context
- [x] **CRIT** use-topology-metrics.ts:360 — uses timeWindowRef now
- [x] **CRIT** activity/index.tsx — uses pausedRef.current now
- [x] **HIGH** connection.tsx — Tauri imports log informative fallback
- [x] **HIGH** event-detail.tsx — getAdminBase captured fresh at call time
- [x] **HIGH** ai-debug/index.tsx — same fix applied
- [x] **HIGH** incidents/index.tsx:509 — improved error messaging
- [x] **MED** topology/index.tsx — dynamic friendlyLambdaName() replaces hardcoded map
- [x] **MED** split-panel.tsx — pointer tracking lost if cursor exits viewport
- [x] **MED** connection-picker.tsx — smart port detection for :4566→:4599
- [x] **MED** health.ts — P99 threshold configurable via parameter
- [x] **MED** chaos timer — setInterval drift over long durations
- [x] **MED** markdown rendering in ai-debug — nested code blocks break
- [x] **LOW** event-list.tsx — hash collisions for service colors (djb2 hash in 5 files)
- [x] **LOW** types.ts — mixed camelCase/PascalCase naming (JSDoc documented)

## Nice-to-Haves (discovered during development)

- [x] Split panel ratio persistence to localStorage
- [x] Keyboard navigation in service lists (arrow keys)
- [x] ARIA labels for icon rail accessibility
- [x] Request correlation IDs across views (Activity→Traces, Topology→Traces)
- [x] Offline mode / cached last-known-good state
- [x] Rate limiting awareness in polling hooks
- [x] i18n preparation (externalize UI strings)
- [x] Dark/light theme toggle (Settings → Appearance)
