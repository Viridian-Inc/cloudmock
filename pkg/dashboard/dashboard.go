// Package dashboard provides a single-page web dashboard for cloudmock,
// served on the dashboard port and talking to the admin API.
package dashboard

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
)

//go:embed htm-preact.min.js
var htmPreactUMD string

// Handler serves the cloudmock web dashboard as a self-contained SPA.
type Handler struct {
	html []byte
}

// New creates a dashboard Handler that constructs admin API URLs using the given admin port.
func New(adminPort int) *Handler {
	html := buildHTML(adminPort)
	return &Handler{html: []byte(html)}
}

// ServeHTTP implements http.Handler. All requests receive the dashboard HTML.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.html)
}

func buildHTML(adminPort int) string {
	adminBase := fmt.Sprintf("http://localhost:%d", adminPort)
	result := fmt.Sprintf(htmlTemplate, adminBase)
	// Go raw strings can't contain backticks, so the template uses ‹› as
	// placeholders for JS template literal backticks. Replace them here.
	result = strings.ReplaceAll(result, "\u2039", "`") // ‹ → `
	result = strings.ReplaceAll(result, "\u203a", "`") // › → `
	result = strings.Replace(result, "/*HTMPREACT*/", htmPreactUMD, 1)
	return result
}

// htmlTemplate is the complete SPA. The single %%s verb is replaced with the
// admin base URL (e.g. "http://localhost:4599").
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>cloudmock dashboard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Figtree:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
/* ─── Reset ─── */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

/* ─── Design Tokens ─── */
:root {
  --brand-blue: #097FF5;
  --brand-dark: #0A1F44;
  --primary-green: #029662;
  --accent-cyan: #7CCEF2;
  --error: #FF4E5E;
  --warning: #FF9A4B;
  --brand-yellow: #FEC307;

  --n50: #F8FAFC; --n100: #F1F5F9; --n200: #E2E8F0; --n300: #CBD5E1;
  --n400: #94A3B8; --n500: #64748B; --n600: #475569; --n700: #334155;
  --n800: #1E293B; --n900: #0F172A;

  --font-sans: 'Figtree', system-ui, -apple-system, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

  --radius-sm: 4px; --radius-md: 8px; --radius-lg: 12px; --radius-xl: 16px;
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.06);
  --shadow-md: 0 4px 12px rgba(0,0,0,0.08);
  --shadow-lg: 0 12px 32px rgba(0,0,0,0.12);

  --sidebar-width: 200px;
  --header-height: 56px;
}

html, body { height: 100%%; font-family: var(--font-sans); background: var(--n50); color: var(--n800); }
body { overflow: hidden; }
#app { height: 100%%; }

/* ─── Scrollbar ─── */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--n300); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--n400); }

/* ─── Layout ─── */
.layout { display: flex; flex-direction: column; height: 100%%; }
.header {
  height: var(--header-height); background: var(--brand-dark); color: white;
  display: flex; align-items: center; padding: 0 20px; gap: 16px; flex-shrink: 0;
  z-index: 100;
}
.header-logo { display: flex; align-items: center; gap: 8px; font-weight: 700; font-size: 16px; }
.header-logo svg { width: 24px; height: 24px; }
.header-spacer { flex: 1; }
.header-badge {
  display: flex; align-items: center; gap: 6px; font-size: 13px;
  padding: 4px 12px; border-radius: 20px; background: rgba(255,255,255,0.1);
}
.header-badge .dot { width: 8px; height: 8px; border-radius: 50%%; }
.dot-green { background: var(--primary-green); }
.dot-red { background: var(--error); }
.dot-yellow { background: var(--warning); }

.cmd-k-btn {
  display: flex; align-items: center; gap: 6px; padding: 6px 14px;
  background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.15);
  border-radius: var(--radius-md); color: var(--n400); font-size: 13px;
  cursor: pointer; transition: all 0.15s;
}
.cmd-k-btn:hover { background: rgba(255,255,255,0.14); color: white; }
.cmd-k-btn kbd {
  font-family: var(--font-sans); font-size: 11px; padding: 1px 5px;
  background: rgba(255,255,255,0.1); border-radius: 4px;
}

.body-wrap { display: flex; flex: 1; overflow: hidden; }

.sidebar {
  width: var(--sidebar-width); background: var(--brand-dark); color: var(--n300);
  display: flex; flex-direction: column; flex-shrink: 0; overflow-y: auto;
  padding: 8px 0; border-right: 1px solid rgba(255,255,255,0.06);
}
.nav-item {
  display: flex; align-items: center; gap: 10px; padding: 10px 16px;
  font-size: 14px; cursor: pointer; transition: all 0.15s;
  border-left: 3px solid transparent; text-decoration: none; color: inherit;
}
.nav-item:hover { background: rgba(255,255,255,0.06); color: white; }
.nav-item.active {
  border-left-color: var(--brand-blue); background: rgba(9,127,245,0.08); color: white;
  font-weight: 600;
}
.nav-item svg { width: 18px; height: 18px; opacity: 0.7; flex-shrink: 0; }
.nav-item.active svg { opacity: 1; }
.nav-badge {
  margin-left: auto; font-size: 11px; padding: 1px 7px;
  background: rgba(255,255,255,0.1); border-radius: 10px; font-weight: 600;
}
.nav-divider { height: 1px; background: rgba(255,255,255,0.08); margin: 8px 16px; }
.nav-footer {
  margin-top: auto; padding: 12px 16px; font-size: 12px; color: var(--n500);
  border-top: 1px solid rgba(255,255,255,0.06);
}

.main { flex: 1; overflow-y: auto; padding: 24px; }

/* ─── Cards ─── */
.card {
  background: white; border: 1px solid var(--n200); border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm); transition: all 0.2s;
}
.card-clickable { cursor: pointer; }
.card-clickable:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); border-color: var(--n300); }
.card-header { padding: 16px 20px; border-bottom: 1px solid var(--n100); display: flex; align-items: center; gap: 12px; }
.card-body { padding: 16px 20px; }

/* ─── Stat Cards ─── */
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.stat-card {
  background: white; border: 1px solid var(--n200); border-radius: var(--radius-lg);
  padding: 20px; box-shadow: var(--shadow-sm);
}
.stat-label { font-size: 13px; color: var(--n500); margin-bottom: 4px; font-weight: 500; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--n900); }
.stat-sub { font-size: 12px; color: var(--n400); margin-top: 2px; }

/* ─── Service Cards Grid ─── */
.services-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
.svc-card {
  background: white; border: 1px solid var(--n200); border-radius: var(--radius-lg);
  padding: 20px; cursor: pointer; transition: all 0.2s; box-shadow: var(--shadow-sm);
}
.svc-card:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); border-color: var(--brand-blue); }
.svc-card-head { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.svc-card-name { font-weight: 700; font-size: 15px; color: var(--n900); }
.svc-card-tier {
  font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 10px;
}
.tier-t1 { background: rgba(2,150,98,0.1); color: var(--primary-green); }
.tier-t2 { background: var(--n100); color: var(--n500); }
.svc-card-status { margin-left: auto; display: flex; align-items: center; gap: 6px; font-size: 13px; color: var(--n500); }
.svc-card-status .dot { width: 8px; height: 8px; border-radius: 50%%; }
.svc-card-meta { font-size: 13px; color: var(--n500); display: flex; gap: 16px; }

/* ─── Tables ─── */
.table-wrap { overflow-x: auto; }
table { width: 100%%; border-collapse: collapse; font-size: 14px; }
thead th {
  text-align: left; padding: 10px 16px; font-weight: 600; color: var(--n600);
  background: var(--n50); border-bottom: 2px solid var(--n200); font-size: 13px;
  white-space: nowrap; user-select: none;
}
thead th.sortable { cursor: pointer; }
thead th.sortable:hover { color: var(--brand-blue); }
tbody td { padding: 10px 16px; border-bottom: 1px solid var(--n100); }
tbody tr { transition: background 0.1s; }
tbody tr:hover { background: var(--n50); }
tbody tr.clickable { cursor: pointer; }
tbody tr.clickable:hover { background: rgba(9,127,245,0.04); }
tbody tr.expanded { background: rgba(9,127,245,0.04); }
.empty-state { text-align: center; padding: 48px 20px; color: var(--n400); }
.empty-state svg { width: 48px; height: 48px; margin-bottom: 12px; opacity: 0.4; }

/* ─── Status Badges ─── */
.status-pill {
  display: inline-flex; padding: 2px 10px; border-radius: 12px;
  font-size: 12px; font-weight: 600; font-family: var(--font-mono);
}
.status-2xx { background: rgba(2,150,98,0.1); color: var(--primary-green); }
.status-3xx { background: rgba(9,127,245,0.1); color: var(--brand-blue); }
.status-4xx { background: rgba(254,195,7,0.15); color: #B8860B; }
.status-5xx { background: rgba(255,78,94,0.1); color: var(--error); }

/* ─── Buttons ─── */
.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  font-family: var(--font-sans); font-weight: 600; border: none; cursor: pointer;
  border-radius: var(--radius-md); transition: all 0.15s; font-size: 14px;
  padding: 0 16px; height: 40px; white-space: nowrap;
}
.btn:hover { transform: scale(0.97); }
.btn:active { opacity: 0.9; }
.btn-sm { height: 32px; font-size: 13px; padding: 0 12px; }
.btn-lg { height: 48px; font-size: 15px; padding: 0 24px; }
.btn-primary { background: var(--primary-green); color: white; }
.btn-primary:hover { background: #028555; }
.btn-secondary { background: var(--brand-blue); color: white; }
.btn-secondary:hover { background: #0870D9; }
.btn-ghost { background: transparent; color: var(--n600); border: 1px solid var(--n300); }
.btn-ghost:hover { background: var(--n50); border-color: var(--n400); }
.btn-danger { background: var(--error); color: white; }
.btn-danger:hover { background: #E8404F; }
.btn-icon { width: 36px; height: 36px; padding: 0; border-radius: var(--radius-md); }
.btn-icon.btn-sm { width: 28px; height: 28px; }

/* ─── Inputs ─── */
.input, .select {
  font-family: var(--font-sans); font-size: 14px; padding: 8px 12px;
  border: 1px solid var(--n300); border-radius: var(--radius-md);
  background: white; color: var(--n800); outline: none; transition: border-color 0.15s;
}
.input:focus, .select:focus { border-color: var(--brand-blue); box-shadow: 0 0 0 3px rgba(9,127,245,0.1); }
.input-search {
  padding-left: 36px; background-image: url("data:image/svg+xml,%%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='%%2394A3B8' stroke-width='2'%%3E%%3Ccircle cx='11' cy='11' r='8'%%3E%%3C/circle%%3E%%3Cline x1='21' y1='21' x2='16.65' y2='16.65'%%3E%%3C/line%%3E%%3C/svg%%3E");
  background-repeat: no-repeat; background-position: 10px center;
}

/* ─── Tabs ─── */
.tabs { display: flex; gap: 0; border-bottom: 2px solid var(--n200); margin-bottom: 16px; }
.tab {
  padding: 10px 20px; font-size: 14px; font-weight: 500; color: var(--n500);
  cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -2px;
  transition: all 0.15s; background: none; border-top: none; border-left: none; border-right: none;
  font-family: var(--font-sans);
}
.tab:hover { color: var(--n700); }
.tab.active { color: var(--brand-blue); border-bottom-color: var(--brand-blue); font-weight: 600; }

/* ─── Modal ─── */
.modal-backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
  z-index: 1000; display: flex; align-items: center; justify-content: center;
  animation: fadeIn 0.15s ease;
}
.modal {
  background: white; border-radius: var(--radius-xl); box-shadow: var(--shadow-lg);
  max-height: 85vh; overflow-y: auto; animation: slideUp 0.2s ease;
}
.modal-sm { width: 400px; }
.modal-md { width: 500px; }
.modal-lg { width: 600px; }
.modal-xl { width: 800px; }
.modal-header {
  padding: 20px 24px; border-bottom: 1px solid var(--n100);
  display: flex; align-items: center; justify-content: space-between;
}
.modal-header h3 { font-size: 18px; font-weight: 700; }
.modal-body { padding: 24px; }
.modal-footer {
  padding: 16px 24px; border-top: 1px solid var(--n100);
  display: flex; justify-content: flex-end; gap: 8px;
}

@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes slideUp { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* ─── Drawer ─── */
.drawer-backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.3); z-index: 900;
  animation: fadeIn 0.15s ease;
}
.drawer {
  position: fixed; top: 0; right: 0; bottom: 0; width: 40%%; min-width: 480px;
  background: white; box-shadow: var(--shadow-lg); z-index: 901;
  display: flex; flex-direction: column; animation: slideIn 0.2s ease;
}
.drawer-header {
  padding: 16px 20px; border-bottom: 1px solid var(--n100);
  display: flex; align-items: center; justify-content: space-between; flex-shrink: 0;
}
.drawer-body { flex: 1; overflow-y: auto; padding: 20px; }

@keyframes slideIn { from { transform: translateX(100%%); } to { transform: translateX(0); } }

/* ─── Request Detail Expand ─── */
.req-expand {
  background: var(--n50); border-top: 1px solid var(--n200);
  padding: 16px 20px;
}
.req-expand-inner {
  background: white; border: 1px solid var(--n200); border-radius: var(--radius-lg);
  overflow: hidden;
}
.req-expand-body { padding: 16px; }

/* ─── JSON Display ─── */
.json-view {
  font-family: var(--font-mono); font-size: 13px; line-height: 1.6;
  white-space: pre-wrap; word-break: break-all; background: var(--n900);
  color: var(--n300); padding: 16px; border-radius: var(--radius-md);
  overflow-x: auto; max-height: 400px; overflow-y: auto;
}
.json-key { color: var(--brand-blue); }
.json-string { color: var(--primary-green); }
.json-number { color: var(--warning); }
.json-boolean { color: #A78BFA; }
.json-null { color: var(--n500); }

/* ─── Code/Pre ─── */
pre, code { font-family: var(--font-mono); }
.mono { font-family: var(--font-mono); }

/* ─── Filters Bar ─── */
.filters-bar {
  display: flex; gap: 12px; align-items: center; margin-bottom: 16px; flex-wrap: wrap;
}
.filters-bar .input, .filters-bar .select { height: 36px; }

/* ─── Pagination ─── */
.pagination {
  display: flex; align-items: center; justify-content: center; gap: 8px;
  padding: 16px 0; font-size: 14px; color: var(--n600);
}
.pagination button {
  padding: 6px 12px; border: 1px solid var(--n300); border-radius: var(--radius-md);
  background: white; cursor: pointer; font-family: var(--font-sans); font-size: 13px;
}
.pagination button:hover { background: var(--n50); }
.pagination button:disabled { opacity: 0.4; cursor: default; }
.pagination .page-info { font-weight: 500; }

/* ─── DynamoDB Browser ─── */
.ddb-layout { display: flex; gap: 0; height: calc(100vh - var(--header-height) - 48px); margin: -24px; }
.ddb-sidebar {
  width: 260px; border-right: 1px solid var(--n200); background: white;
  display: flex; flex-direction: column; flex-shrink: 0;
}
.ddb-sidebar-header { padding: 16px; border-bottom: 1px solid var(--n100); }
.ddb-sidebar-list { flex: 1; overflow-y: auto; }
.ddb-table-item {
  padding: 12px 16px; cursor: pointer; border-bottom: 1px solid var(--n50);
  transition: background 0.1s; display: flex; align-items: center; justify-content: space-between;
}
.ddb-table-item:hover { background: var(--n50); }
.ddb-table-item.active { background: rgba(9,127,245,0.06); border-left: 3px solid var(--brand-blue); }
.ddb-table-item .name { font-weight: 600; font-size: 14px; }
.ddb-table-item .count { font-size: 12px; color: var(--n400); }
.ddb-main { flex: 1; overflow-y: auto; padding: 24px; }
.ddb-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.ddb-key-schema { font-size: 13px; color: var(--n500); display: flex; gap: 12px; }
.ddb-key-schema span { padding: 2px 8px; background: var(--n100); border-radius: var(--radius-sm); font-family: var(--font-mono); }

/* ─── Topology ─── */
.topology-container { width: 100%%; height: calc(100vh - var(--header-height) - 48px); }
.topology-container svg { width: 100%%; height: 100%%; }

/* ─── Command Palette ─── */
.palette-backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
  z-index: 2000; display: flex; align-items: flex-start; justify-content: center;
  padding-top: 20vh; animation: fadeIn 0.1s ease;
}
.palette {
  width: 560px; background: white; border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg); overflow: hidden; animation: slideUp 0.15s ease;
}
.palette-input {
  width: 100%%; padding: 16px 20px; font-size: 16px; border: none; outline: none;
  font-family: var(--font-sans); border-bottom: 1px solid var(--n100);
}
.palette-results { max-height: 320px; overflow-y: auto; }
.palette-item {
  padding: 12px 20px; cursor: pointer; display: flex; align-items: center; gap: 12px;
  font-size: 14px; transition: background 0.1s;
}
.palette-item:hover, .palette-item.active { background: var(--n50); }
.palette-item .label { font-weight: 600; }
.palette-item .desc { color: var(--n400); font-size: 13px; margin-left: auto; }

/* ─── Lambda Logs ─── */
.log-entry { font-family: var(--font-mono); font-size: 13px; padding: 4px 0; line-height: 1.5; }
.log-entry.stderr { color: var(--error); }
.log-time { color: var(--n400); margin-right: 8px; }
.log-reqid { color: var(--brand-blue); margin-right: 8px; }

/* ─── IAM Debugger ─── */
.iam-result {
  padding: 16px 20px; border-radius: var(--radius-lg); font-size: 16px; font-weight: 700;
  margin: 16px 0; text-align: center;
}
.iam-allow { background: rgba(2,150,98,0.1); color: var(--primary-green); border: 1px solid rgba(2,150,98,0.2); }
.iam-deny { background: rgba(255,78,94,0.1); color: var(--error); border: 1px solid rgba(255,78,94,0.2); }

/* ─── Mail ─── */
.mail-list { border: 1px solid var(--n200); border-radius: var(--radius-lg); overflow: hidden; }
.mail-row {
  padding: 14px 20px; border-bottom: 1px solid var(--n100); cursor: pointer;
  display: flex; gap: 16px; align-items: center; transition: background 0.1s;
}
.mail-row:hover { background: var(--n50); }
.mail-row:last-child { border-bottom: none; }
.mail-from { font-weight: 600; width: 200px; flex-shrink: 0; font-size: 14px; }
.mail-subject { flex: 1; font-size: 14px; }
.mail-date { color: var(--n400); font-size: 13px; flex-shrink: 0; }

/* ─── Utility ─── */
.flex { display: flex; }
.flex-col { flex-direction: column; }
.items-center { align-items: center; }
.justify-between { justify-content: space-between; }
.gap-2 { gap: 8px; }
.gap-3 { gap: 12px; }
.gap-4 { gap: 16px; }
.mb-4 { margin-bottom: 16px; }
.mb-6 { margin-bottom: 24px; }
.mt-4 { margin-top: 16px; }
.mr-2 { margin-right: 8px; }
.ml-auto { margin-left: auto; }
.text-sm { font-size: 13px; }
.text-muted { color: var(--n500); }
.font-mono { font-family: var(--font-mono); }
.truncate { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.w-full { width: 100%%; }
.grid { display: grid; }
.grid-cols-2 { grid-template-columns: repeat(2, 1fr); }
.grid-cols-3 { grid-template-columns: repeat(3, 1fr); }
.grid-cols-4 { grid-template-columns: repeat(4, 1fr); }
.px-3 { padding-left: 12px; padding-right: 12px; }
.px-4 { padding-left: 16px; padding-right: 16px; }
.py-2 { padding-top: 8px; padding-bottom: 8px; }
.py-3 { padding-top: 12px; padding-bottom: 12px; }
.overflow-auto { overflow: auto; }
.overflow-x-auto { overflow-x: auto; }
.whitespace-pre { white-space: pre; }
.cursor-pointer { cursor: pointer; }
.relative { position: relative; }
.absolute { position: absolute; }
.page-title { font-size: 22px; font-weight: 700; color: var(--n900); }
.page-desc { font-size: 14px; color: var(--n500); margin-top: 2px; }
.section-title { font-size: 16px; font-weight: 700; color: var(--n800); margin-bottom: 12px; }
.copy-btn {
  padding: 4px 10px; font-size: 12px; border: 1px solid var(--n300);
  background: white; border-radius: var(--radius-sm); cursor: pointer;
  font-family: var(--font-sans); color: var(--n600);
}
.copy-btn:hover { background: var(--n50); }
.label { font-size: 13px; font-weight: 600; color: var(--n600); margin-bottom: 6px; }
.field-row { display: flex; gap: 12px; margin-bottom: 12px; }
.field-row .input, .field-row .select { flex: 1; }
.textarea {
  font-family: var(--font-mono); font-size: 13px; padding: 12px;
  border: 1px solid var(--n300); border-radius: var(--radius-md);
  background: white; color: var(--n800); outline: none; resize: vertical;
  min-height: 200px; width: 100%%; line-height: 1.5;
}
.textarea:focus { border-color: var(--brand-blue); box-shadow: 0 0 0 3px rgba(9,127,245,0.1); }

.toast {
  position: fixed; bottom: 24px; right: 24px; padding: 12px 20px;
  background: var(--n900); color: white; border-radius: var(--radius-md);
  font-size: 14px; z-index: 3000; animation: slideUp 0.2s ease;
  box-shadow: var(--shadow-lg);
}
</style>
</head>
<body>
<div id="app"></div>

<script>/*HTMPREACT*/</script>
<script>
const { html, render, Component, h, useState, useEffect, useRef, useCallback, useMemo, useReducer, useContext, createContext } = htmPreact;
const Fragment = 'div';

const ADMIN = '%s';
const GW_ENDPOINT = ADMIN.replace(/:\d+$/, ':4566');

// ─── Utility ───────────────────────────────────────────────────
function api(path, opts) {
  return fetch(ADMIN + path, opts).then(r => r.ok ? r.json() : Promise.reject(r));
}

function ddbRequest(action, body) {
  return fetch(GW_ENDPOINT, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-amz-json-1.0',
      'X-Amz-Target': 'DynamoDB_20120810.' + action,
      'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260321/us-east-1/dynamodb/aws4_request, SignedHeaders=host, Signature=fake',
    },
    body: JSON.stringify(body || {}),
  }).then(r => r.json());
}

function statusClass(code) {
  if (code >= 500) return 'status-5xx';
  if (code >= 400) return 'status-4xx';
  if (code >= 300) return 'status-3xx';
  return 'status-2xx';
}

function fmtTime(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function fmtDuration(ms) {
  if (ms === undefined || ms === null) return '';
  if (ms < 1) return '<1ms';
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(1) + 's';
}

function syntaxHighlight(jsonStr) {
  if (!jsonStr) return '';
  try {
    if (typeof jsonStr === 'object') jsonStr = JSON.stringify(jsonStr, null, 2);
    else jsonStr = JSON.stringify(JSON.parse(jsonStr), null, 2);
  } catch(e) { return jsonStr; }
  return jsonStr.replace(/("(\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    function(match) {
      let cls = 'json-number';
      if (/^"/.test(match)) {
        cls = /:$/.test(match) ? 'json-key' : 'json-string';
      } else if (/true|false/.test(match)) {
        cls = 'json-boolean';
      } else if (/null/.test(match)) {
        cls = 'json-null';
      }
      return '<span class="' + cls + '">' + match + '</span>';
    });
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {});
}

// ─── Hooks ─────────────────────────────────────────────────────
function useSSE() {
  const [connected, setConnected] = useState(false);
  const [events, setEvents] = useState([]);
  const esRef = useRef(null);

  useEffect(() => {
    const es = new EventSource(ADMIN + '/api/stream');
    esRef.current = es;
    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data);
        setEvents(prev => [event, ...prev].slice(0, 500));
      } catch(err) {}
    };
    return () => es.close();
  }, []);

  return { connected, events };
}

function useRoute() {
  const [route, setRoute] = useState(location.hash || '#/');
  useEffect(() => {
    const handler = () => setRoute(location.hash || '#/');
    window.addEventListener('hashchange', handler);
    return () => window.removeEventListener('hashchange', handler);
  }, []);
  return route;
}

function parseRoute(hash) {
  const path = hash.replace('#', '') || '/';
  const segments = path.split('/').filter(Boolean);
  return { path, segments };
}

// ─── SVG Helper ────────────────────────────────────────────────
// HTM tagged templates need backticks, but Go raw strings also use backticks.
// So we use a helper that creates SVG elements from innerHTML strings.
function svg(svgHtml) {
  return h('span', { dangerouslySetInnerHTML: { __html: svgHtml }, style: 'display:inline-flex;align-items:center;' });
}

// ─── SVG Icons ─────────────────────────────────────────────────
const icons = {
  cloud: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"></path></svg>'),
  services: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg>'),
  requests: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>'),
  database: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path></svg>'),
  resources: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path></svg>'),
  lambda: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>'),
  shield: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg>'),
  mail: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline></svg>'),
  topology: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="18" r="3"></circle><circle cx="6" cy="6" r="3"></circle><circle cx="18" cy="6" r="3"></circle><line x1="6" y1="9" x2="6" y2="21"></line><path d="M9 6h6"></path><path d="M6 21a3 3 0 0 0 3-3V9"></path></svg>'),
  expand: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="15 3 21 3 21 9"></polyline><polyline points="9 21 3 21 3 15"></polyline><line x1="21" y1="3" x2="14" y2="10"></line><line x1="3" y1="21" x2="10" y2="14"></line></svg>'),
  x: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>'),
  chevDown: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="6 9 12 15 18 9"></polyline></svg>'),
  chevRight: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="9 18 15 12 9 6"></polyline></svg>'),
  search: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>'),
  plus: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>'),
  trash: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>'),
  copy: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>'),
  play: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>'),
  refresh: svg('<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16"><polyline points="23 4 23 10 17 10"></polyline><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path></svg>'),
};

// ─── Toast ─────────────────────────────────────────────────────
let toastTimer = null;
function Toast({ message }) {
  if (!message) return null;
  return html‹<div class="toast">${message}</div>›;
}

// ─── App ───────────────────────────────────────────────────────
function App() {
  const route = useRoute();
  const sse = useSSE();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [services, setServices] = useState([]);
  const [stats, setStats] = useState({});
  const [health, setHealth] = useState(null);
  const [toast, setToast] = useState('');
  const [mailCount, setMailCount] = useState(0);

  const showToast = useCallback((msg) => {
    setToast(msg);
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => setToast(''), 3000);
  }, []);

  useEffect(() => {
    api('/api/services').then(setServices).catch(() => {});
    api('/api/stats').then(setStats).catch(() => {});
    api('/api/health').then(setHealth).catch(() => {});
    api('/api/ses/emails').then(e => setMailCount(Array.isArray(e) ? e.length : 0)).catch(() => {});
  }, []);

  // Refresh stats periodically
  useEffect(() => {
    const iv = setInterval(() => {
      api('/api/stats').then(setStats).catch(() => {});
      api('/api/ses/emails').then(e => setMailCount(Array.isArray(e) ? e.length : 0)).catch(() => {});
    }, 5000);
    return () => clearInterval(iv);
  }, []);

  // Cmd+K handler
  useEffect(() => {
    const handler = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setPaletteOpen(p => !p);
      }
      if (e.key === 'Escape') setPaletteOpen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const { path, segments } = parseRoute(route);
  const navItems = [
    { id: '/', label: 'Services', icon: 'services' },
    { id: '/requests', label: 'Requests', icon: 'requests' },
    { id: '/dynamodb', label: 'DynamoDB', icon: 'database' },
    { id: '/resources', label: 'Resources', icon: 'resources' },
    { id: '/lambda', label: 'Lambda', icon: 'lambda' },
    { id: '/iam', label: 'IAM', icon: 'shield' },
    { id: '/mail', label: 'Mail', icon: 'mail', badge: mailCount || null },
    { id: '/topology', label: 'Topology', icon: 'topology' },
  ];

  const activePath = '/' + (segments[0] || '');

  function renderPage() {
    if (segments[0] === 'requests' && segments[1]) {
      return html‹<${RequestDetailPage} id=${segments[1]} showToast=${showToast} />›;
    }
    switch(activePath) {
      case '/requests': return html‹<${RequestsPage} sse=${sse} showToast=${showToast} />›;
      case '/dynamodb': return html‹<${DynamoDBPage} showToast=${showToast} />›;
      case '/resources': return html‹<${ResourcesPage} services=${services} />›;
      case '/lambda': return html‹<${LambdaPage} sse=${sse} />›;
      case '/iam': return html‹<${IAMPage} showToast=${showToast} />›;
      case '/mail': return html‹<${MailPage} />›;
      case '/topology': return html‹<${TopologyPage} />›;
      default: return html‹<${ServicesPage} services=${services} stats=${stats} health=${health} />›;
    }
  }

  return html‹
    <div class="layout">
      <header class="header">
        <div class="header-logo">
          ${icons.cloud}
          <span>cloudmock</span>
        </div>
        <div class="header-spacer" />
        <div class="header-badge" id="health-badge">
          <span class="dot ${health && health.status === ‹healthy› ? 'dot-green' : 'dot-yellow'}" id="health-dot"></span>
          <span>${health ? (health.status === 'healthy' ? 'Healthy' : 'Degraded') : '...'}</span>
        </div>
        <div class="header-badge" id="sse-badge">
          <span class="dot ${sse.connected ? 'dot-green' : 'dot-red'}" id="sse-dot"></span>
          <span>${sse.connected ? 'Connected' : 'Disconnected'}</span>
        </div>
        <button class="cmd-k-btn" onclick=${() => setPaletteOpen(true)}>
          ${icons.search} Search <kbd>Cmd+K</kbd>
        </button>
      </header>

      <div class="body-wrap">
        <nav class="sidebar">
          ${navItems.map(item => html‹
            <a class="nav-item ${activePath === item.id ? ‹active› : ''}"
               href=${'#' + item.id}
               onclick=${(e) => { e.preventDefault(); location.hash = item.id; }}>
              ${icons[item.icon]}
              <span>${item.label}</span>
              ${item.badge ? html‹<span class="nav-badge">${item.badge}</span>› : null}
            </a>
          ')}
          <div class="nav-divider" />
          <div class="nav-footer">
            <div>v0.1.0</div>
            <div>${services.length} services</div>
          </div>
        </nav>

        <main class="main">
          ${renderPage()}
        </main>
      </div>

      ${paletteOpen && html‹<${CommandPalette} services=${services} onClose=${() => setPaletteOpen(false)} />›}
      <${Toast} message=${toast} />
    </div>
  ';
}

// ─── Services Page ─────────────────────────────────────────────
function ServicesPage({ services, stats, health }) {
  const [search, setSearch] = useState('');

  const filtered = useMemo(() => {
    if (!search) return services;
    const q = search.toLowerCase();
    return services.filter(s => s.name.toLowerCase().includes(q));
  }, [services, search]);

  const totalRequests = useMemo(() => {
    if (!stats || !stats.services) return 0;
    return Object.values(stats.services).reduce((sum, s) => sum + (s.total || 0), 0);
  }, [stats]);

  const healthyCount = useMemo(() => {
    if (!health || !health.services) return 0;
    return Object.values(health.services).filter(Boolean).length;
  }, [health]);

  return html‹
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Services</h1>
          <p class="page-desc">Registered AWS service mocks</p>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">Total Services</div>
          <div class="stat-value">${services.length}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Total Requests</div>
          <div class="stat-value">${totalRequests.toLocaleString()}</div>
          <div class="stat-sub">Requests/min tracked in /api/stats</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Healthy</div>
          <div class="stat-value">${healthyCount} / ${services.length}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Uptime</div>
          <div class="stat-value">100%%</div>
        </div>
      </div>

      <div class="mb-4">
        <input class="input input-search" style="width:320px" placeholder="Filter services..."
               value=${search} onInput=${(e) => setSearch(e.target.value)} />
      </div>

      <div class="services-grid">
        ${filtered.map(svc => {
          const tier = svc.action_count > 5 ? ‹T1› : 'T2';
          return html‹
            <div class="svc-card" onclick=${() => location.hash = ‹/resources?service=› + svc.name}>
              <div class="svc-card-head">
                <span class="svc-card-name">${svc.name}</span>
                <span class="svc-card-tier ${tier === 'T1' ? 'tier-t1' : 'tier-t2'}">${tier}</span>
                <div class="svc-card-status">
                  <span class="dot ${svc.healthy ? 'dot-green' : 'dot-red'}"></span>
                  ${svc.healthy ? 'Healthy' : 'Unhealthy'}
                </div>
              </div>
              <div class="svc-card-meta">
                <span>${svc.action_count} actions</span>
              </div>
            </div>
          ';
        })}
      </div>
    </div>
  ';
}

// ─── Requests Page ─────────────────────────────────────────────
function RequestsPage({ sse, showToast }) {
  const [requests, setRequests] = useState([]);
  const [expanded, setExpanded] = useState(null);
  const [drawer, setDrawer] = useState(null);
  const [svcFilter, setSvcFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [textFilter, setTextFilter] = useState('');
  const [services, setServices] = useState([]);
  const [detailTab, setDetailTab] = useState('overview');

  useEffect(() => {
    api('/api/requests?limit=200').then(setRequests).catch(() => {});
    api('/api/services').then(s => setServices(s.map(x => x.name).sort())).catch(() => {});
  }, []);

  // Merge SSE events
  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest && latest.type === 'request' && latest.data) {
      setRequests(prev => [latest.data, ...prev].slice(0, 500));
    }
  }, [sse.events]);

  const filtered = useMemo(() => {
    return requests.filter(r => {
      if (svcFilter && r.service !== svcFilter) return false;
      if (statusFilter) {
        const s = String(r.status);
        if (statusFilter === '2xx' && !s.startsWith('2')) return false;
        if (statusFilter === '4xx' && !s.startsWith('4')) return false;
        if (statusFilter === '5xx' && !s.startsWith('5')) return false;
      }
      if (textFilter) {
        const q = textFilter.toLowerCase();
        const haystack = (r.service + ' ' + r.action + ' ' + r.method + ' ' + (r.id || '')).toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [requests, svcFilter, statusFilter, textFilter]);

  function toggleExpand(id) {
    setExpanded(prev => prev === id ? null : id);
  }

  function openDrawer(e, req) {
    e.stopPropagation();
    setDrawer(req);
  }

  function replayRequest(id) {
    api('/api/requests/' + id + '/replay', { method: 'POST' })
      .then(() => showToast('Request replayed'))
      .catch(() => showToast('Replay failed'));
  }

  function renderDetail(req, tab) {
    if (!req) return null;
    switch(tab) {
      case 'request':
        return html‹
          <div>
            <div class="flex items-center justify-between mb-4">
              <span class="section-title" style="margin:0">Request Body</span>
              <button class="copy-btn" onclick=${() => { copyToClipboard(JSON.stringify(req.request_body || req.body || '', null, 2)); showToast('Copied'); }}>
                ${icons.copy} Copy
              </button>
            </div>
            <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(req.request_body || req.body || '(empty)') }}></div>
          </div>
        ';
      case 'response':
        return html‹
          <div>
            <div class="flex items-center justify-between mb-4">
              <span class="section-title" style="margin:0">Response Body</span>
              <button class="copy-btn" onclick=${() => { copyToClipboard(JSON.stringify(req.response_body || '', null, 2)); showToast('Copied'); }}>
                ${icons.copy} Copy
              </button>
            </div>
            <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(req.response_body || '(empty)') }}></div>
          </div>
        ';
      case 'timing':
        return html‹
          <div>
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:150px">Total Latency</td><td>${fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">${req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          </div>
        ';
      default:
        return html‹
          <div>
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:150px">Method</td><td>${req.method || ‹POST›}</td></tr>
                <tr><td style="font-weight:600">Service</td><td>${req.service}</td></tr>
                <tr><td style="font-weight:600">Action</td><td class="font-mono">${req.action}</td></tr>
                <tr><td style="font-weight:600">Status</td><td><span class="status-pill ${statusClass(req.status)}">${req.status}</span></td></tr>
                <tr><td style="font-weight:600">Latency</td><td>${fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Request ID</td><td class="font-mono text-sm">${req.id || ''}</td></tr>
                <tr><td style="font-weight:600">Time</td><td>${req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          </div>
        ';
    }
  }

  return html‹
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Request Log</h1>
          <p class="page-desc">All API requests to cloudmock services</p>
        </div>
        <button class="btn btn-ghost btn-sm" onclick=${() => api(‹/api/requests?limit=200›).then(setRequests)}>
          ${icons.refresh} Refresh
        </button>
      </div>

      <div class="filters-bar">
        <select class="select" id="service-filter" value=${svcFilter}
                onchange=${(e) => setSvcFilter(e.target.value)}>
          <option value="">All Services</option>
          ${services.map(s => html‹<option value=${s}>${s}</option>›)}
        </select>
        <select class="select" value=${statusFilter}
                onchange=${(e) => setStatusFilter(e.target.value)}>
          <option value="">All Status</option>
          <option value="2xx">2xx Success</option>
          <option value="4xx">4xx Client Error</option>
          <option value="5xx">5xx Server Error</option>
        </select>
        <input class="input input-search" placeholder="Search requests..."
               value=${textFilter} onInput=${(e) => setTextFilter(e.target.value)} />
        <span class="text-sm text-muted ml-auto">${filtered.length} requests</span>
      </div>

      <div class="card">
        <div class="table-wrap">
          <table id="requests-table">
            <thead>
              <tr>
                <th style="width:100px">Time</th>
                <th>Service</th>
                <th>Action</th>
                <th style="width:80px">Status</th>
                <th style="width:80px">Latency</th>
                <th style="width:40px"></th>
              </tr>
            </thead>
            <tbody id="requests-tbody">
              ${filtered.length === 0 ? html‹
                <tr><td colspan="6" class="empty-state">No requests recorded yet</td></tr>
              › : filtered.map(req => html‹
                <${Fragment} key=${req.id || Math.random()}>
                  <tr class="clickable ${expanded === req.id ? ‹expanded› : ''}"
                      onclick=${() => toggleExpand(req.id)}>
                    <td class="font-mono text-sm">${fmtTime(req.timestamp || req.time)}</td>
                    <td><span style="font-weight:600">${req.service}</span></td>
                    <td class="font-mono text-sm">${req.action}</td>
                    <td><span class="status-pill ${statusClass(req.status)}">${req.status}</span></td>
                    <td class="font-mono text-sm">${fmtDuration(req.latency_ms || req.duration_ms)}</td>
                    <td>
                      <button class="btn-icon btn-sm btn-ghost" title="Open in drawer"
                              onclick=${(e) => openDrawer(e, req)}>
                        ${icons.expand}
                      </button>
                    </td>
                  </tr>
                  ${expanded === req.id && html‹
                    <tr>
                      <td colspan="6" style="padding:0">
                        <div class="req-expand">
                          <div class="req-expand-inner">
                            <div class="tabs" style="padding:0 16px">
                              ${[‹overview›,'request','response','timing'].map(t => html‹
                                <button class="tab ${detailTab === t ? ‹active› : ''}"
                                        onclick=${() => setDetailTab(t)}>
                                  ${t.charAt(0).toUpperCase() + t.slice(1)}
                                </button>
                              ')}
                            </div>
                            <div class="req-expand-body">
                              ${renderDetail(req, detailTab)}
                            </div>
                          </div>
                        </div>
                      </td>
                    </tr>
                  '}
                <//>
              ')}
            </tbody>
          </table>
        </div>
      </div>

      ${drawer && html‹
        <div class="drawer-backdrop" onclick=${() => setDrawer(null)}>
          <div class="drawer" onclick=${(e) => e.stopPropagation()}>
            <div class="drawer-header">
              <h3 style="font-size:16px;font-weight:700">Request Detail</h3>
              <div class="flex gap-2">
                <button class="btn btn-sm btn-ghost" onclick=${() => replayRequest(drawer.id)}>
                  ${icons.play} Replay
                </button>
                <a class="btn btn-sm btn-secondary" href=${‹#/requests/› + drawer.id}
                   style="text-decoration:none;color:white">
                  Full Page
                </a>
                <button class="btn-icon btn-sm btn-ghost" onclick=${() => setDrawer(null)}>
                  ${icons.x}
                </button>
              </div>
            </div>
            <div class="drawer-body">
              <div class="tabs">
                ${['overview','request','response','timing'].map(t => html‹
                  <button class="tab ${detailTab === t ? ‹active› : ''}"
                          onclick=${() => setDetailTab(t)}>
                    ${t.charAt(0).toUpperCase() + t.slice(1)}
                  </button>
                ')}
              </div>
              ${renderDetail(drawer, detailTab)}
            </div>
          </div>
        </div>
      '}
    </div>
  ';
}

// ─── Request Detail Page ───────────────────────────────────────
function RequestDetailPage({ id, showToast }) {
  const [req, setReq] = useState(null);
  const [tab, setTab] = useState('overview');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api('/api/requests/' + id).then(r => { setReq(r); setLoading(false); }).catch(() => setLoading(false));
  }, [id]);

  if (loading) return html‹<div class="empty-state">Loading...</div>›;
  if (!req) return html‹<div class="empty-state">Request not found</div>›;

  function renderBody(body, label) {
    return html‹
      <div>
        <div class="flex items-center justify-between mb-4">
          <span class="section-title" style="margin:0">${label}</span>
          <button class="copy-btn" onclick=${() => { copyToClipboard(JSON.stringify(body || '', null, 2)); showToast('Copied'); }}>
            ${icons.copy} Copy
          </button>
        </div>
        <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(body || '(empty)') }}></div>
      </div>
    ';
  }

  return html‹
    <div>
      <div class="flex items-center gap-3 mb-6">
        <a href="#/requests" class="btn btn-ghost btn-sm">Back</a>
        <div>
          <h1 class="page-title">${req.service} / ${req.action}</h1>
          <p class="page-desc font-mono">${id}</p>
        </div>
        <div class="ml-auto">
          <span class="status-pill ${statusClass(req.status)}" style="font-size:16px;padding:4px 14px">${req.status}</span>
        </div>
      </div>

      <div class="card">
        <div class="tabs" style="padding:0 20px">
          ${[‹overview›,'request','response','timing'].map(t => html‹
            <button class="tab ${tab === t ? ‹active› : ''}" onclick=${() => setTab(t)}>
              ${t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ')}
        </div>
        <div class="card-body">
          ${tab === 'overview' && html‹
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:160px">Method</td><td>${req.method || ‹POST›}</td></tr>
                <tr><td style="font-weight:600">Service</td><td>${req.service}</td></tr>
                <tr><td style="font-weight:600">Action</td><td class="font-mono">${req.action}</td></tr>
                <tr><td style="font-weight:600">Status</td><td><span class="status-pill ${statusClass(req.status)}">${req.status}</span></td></tr>
                <tr><td style="font-weight:600">Latency</td><td>${fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">${req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          '}
          ${tab === 'request' && renderBody(req.request_body || req.body, 'Request Body')}
          ${tab === 'response' && renderBody(req.response_body, 'Response Body')}
          ${tab === 'timing' && html‹
            <table>
              <tbody>
                <tr><td style="font-weight:600;width:160px">Total Latency</td><td>${fmtDuration(req.latency_ms || req.duration_ms)}</td></tr>
                <tr><td style="font-weight:600">Timestamp</td><td class="font-mono">${req.timestamp || req.time || ''}</td></tr>
              </tbody>
            </table>
          '}
        </div>
      </div>
    </div>
  ';
}

// ─── DynamoDB Page ─────────────────────────────────────────────
function DynamoDBPage({ showToast }) {
  const [tables, setTables] = useState([]);
  const [selectedTable, setSelectedTable] = useState(null);
  const [tableDesc, setTableDesc] = useState(null);
  const [items, setItems] = useState([]);
  const [itemCount, setItemCount] = useState(0);
  const [page, setPage] = useState(0);
  const [lastKeys, setLastKeys] = useState([]);
  const [tableSearch, setTableSearch] = useState('');
  const [activeTab, setActiveTab] = useState('browse');
  const [editModal, setEditModal] = useState(null);
  const [createModal, setCreateModal] = useState(false);
  const [createTableModal, setCreateTableModal] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(null);
  const [queryMode, setQueryMode] = useState('scan');
  const [queryExpr, setQueryExpr] = useState('');
  const [filterExpr, setFilterExpr] = useState('');
  const [exprAttrValues, setExprAttrValues] = useState('{}');
  const [queryResults, setQueryResults] = useState(null);
  const [newTableName, setNewTableName] = useState('');
  const [newTablePK, setNewTablePK] = useState('id');
  const [newTablePKType, setNewTablePKType] = useState('S');
  const [newTableSK, setNewTableSK] = useState('');
  const [newTableSKType, setNewTableSKType] = useState('S');
  const [itemJson, setItemJson] = useState('{}');
  const PAGE_SIZE = 25;

  function loadTables() {
    ddbRequest('ListTables', {}).then(r => {
      setTables(r.TableNames || []);
    }).catch(() => {});
  }

  useEffect(() => { loadTables(); }, []);

  function selectTable(name) {
    setSelectedTable(name);
    setPage(0);
    setLastKeys([]);
    setActiveTab('browse');
    ddbRequest('DescribeTable', { TableName: name }).then(r => {
      setTableDesc(r.Table);
    }).catch(() => {});
    scanItems(name, null);
  }

  function scanItems(tableName, exclusiveStartKey) {
    const params = { TableName: tableName, Limit: PAGE_SIZE };
    if (exclusiveStartKey) params.ExclusiveStartKey = exclusiveStartKey;
    ddbRequest('Scan', params).then(r => {
      setItems(r.Items || []);
      setItemCount(r.Count || 0);
      if (r.LastEvaluatedKey) {
        setLastKeys(prev => [...prev, r.LastEvaluatedKey]);
      }
    }).catch(() => setItems([]));
  }

  function nextPage() {
    const lastKey = lastKeys[page];
    if (!lastKey) return;
    setPage(p => p + 1);
    scanItems(selectedTable, lastKey);
  }

  function prevPage() {
    if (page <= 0) return;
    const newPage = page - 1;
    setPage(newPage);
    if (newPage === 0) {
      scanItems(selectedTable, null);
    } else {
      scanItems(selectedTable, lastKeys[newPage - 1]);
    }
  }

  function deleteItem(item) {
    if (!tableDesc) return;
    const key = {};
    tableDesc.KeySchema.forEach(k => {
      key[k.AttributeName] = item[k.AttributeName];
    });
    ddbRequest('DeleteItem', { TableName: selectedTable, Key: key }).then(() => {
      showToast('Item deleted');
      scanItems(selectedTable, null);
      setPage(0);
      setLastKeys([]);
      setEditModal(null);
    }).catch(() => showToast('Delete failed'));
  }

  function saveItem(jsonStr) {
    try {
      const item = JSON.parse(jsonStr);
      ddbRequest('PutItem', { TableName: selectedTable, Item: item }).then(() => {
        showToast('Item saved');
        scanItems(selectedTable, null);
        setPage(0);
        setLastKeys([]);
        setEditModal(null);
        setCreateModal(false);
      }).catch(() => showToast('Save failed'));
    } catch(e) {
      showToast('Invalid JSON');
    }
  }

  function createTable() {
    const params = {
      TableName: newTableName,
      KeySchema: [{ AttributeName: newTablePK, KeyType: 'HASH' }],
      AttributeDefinitions: [{ AttributeName: newTablePK, AttributeType: newTablePKType }],
      BillingMode: 'PAY_PER_REQUEST',
    };
    if (newTableSK) {
      params.KeySchema.push({ AttributeName: newTableSK, KeyType: 'RANGE' });
      params.AttributeDefinitions.push({ AttributeName: newTableSK, AttributeType: newTableSKType });
    }
    ddbRequest('CreateTable', params).then(() => {
      showToast('Table created');
      loadTables();
      setCreateTableModal(false);
      setNewTableName('');
      setNewTablePK('id');
      setNewTableSK('');
    }).catch(() => showToast('Create table failed'));
  }

  function deleteTable(name) {
    ddbRequest('DeleteTable', { TableName: name }).then(() => {
      showToast('Table deleted');
      loadTables();
      if (selectedTable === name) {
        setSelectedTable(null);
        setItems([]);
        setTableDesc(null);
      }
      setDeleteConfirm(null);
    }).catch(() => showToast('Delete table failed'));
  }

  function runQuery() {
    let params = { TableName: selectedTable, Limit: PAGE_SIZE };
    try {
      if (exprAttrValues && exprAttrValues !== '{}') {
        params.ExpressionAttributeValues = JSON.parse(exprAttrValues);
      }
    } catch(e) { showToast('Invalid expression attribute values JSON'); return; }

    if (queryMode === 'query') {
      if (!queryExpr) { showToast('Key condition expression required'); return; }
      params.KeyConditionExpression = queryExpr;
      if (filterExpr) params.FilterExpression = filterExpr;
      ddbRequest('Query', params).then(r => {
        setQueryResults(r.Items || []);
      }).catch(e => showToast('Query failed'));
    } else {
      if (filterExpr) params.FilterExpression = filterExpr;
      ddbRequest('Scan', params).then(r => {
        setQueryResults(r.Items || []);
      }).catch(e => showToast('Scan failed'));
    }
  }

  const filteredTables = useMemo(() => {
    if (!tableSearch) return tables;
    const q = tableSearch.toLowerCase();
    return tables.filter(t => t.toLowerCase().includes(q));
  }, [tables, tableSearch]);

  const columns = useMemo(() => {
    if (!items || items.length === 0) return [];
    const cols = new Set();
    items.forEach(item => Object.keys(item).forEach(k => cols.add(k)));
    // Put key attributes first
    const keyAttrs = tableDesc ? tableDesc.KeySchema.map(k => k.AttributeName) : [];
    const sorted = [...keyAttrs.filter(k => cols.has(k)), ...[...cols].filter(k => !keyAttrs.includes(k)).sort()];
    return sorted;
  }, [items, tableDesc]);

  function formatDDBValue(val) {
    if (!val) return '';
    if (val.S !== undefined) return val.S;
    if (val.N !== undefined) return val.N;
    if (val.BOOL !== undefined) return String(val.BOOL);
    if (val.NULL) return 'null';
    if (val.L) return JSON.stringify(val.L);
    if (val.M) return JSON.stringify(val.M);
    if (val.SS) return val.SS.join(', ');
    if (val.NS) return val.NS.join(', ');
    return JSON.stringify(val);
  }

  return html‹
    <div class="ddb-layout">
      <div class="ddb-sidebar">
        <div class="ddb-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Tables</span>
            <button class="btn btn-primary btn-sm" onclick=${() => setCreateTableModal(true)}>
              ${icons.plus} New
            </button>
          </div>
          <input class="input input-search w-full" placeholder="Filter tables..."
                 value=${tableSearch} onInput=${(e) => setTableSearch(e.target.value)} style="height:32px;font-size:13px" />
        </div>
        <div class="ddb-sidebar-list">
          ${filteredTables.length === 0 ? html›
            <div style="padding:24px;text-align:center;color:var(--n400);font-size:13px">No tables found</div>
          ' : filteredTables.map(t => html‹
            <div class="ddb-table-item ${selectedTable === t ? ‹active› : ''}"
                 onclick=${() => selectTable(t)}>
              <span class="name">${t}</span>
            </div>
          ')}
        </div>
      </div>

      <div class="ddb-main">
        ${!selectedTable ? html‹
          <div class="empty-state">
            ${icons.database}
            <div>Select a table to browse items</div>
          </div>
        › : html‹
          <div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">${selectedTable}</h2>
                ${tableDesc && html›
                  <div class="ddb-key-schema">
                    ${tableDesc.KeySchema.map(k => html‹
                      <span>${k.AttributeName} (${k.KeyType})</span>
                    ›)}
                    <span style="color:var(--n400)">${tableDesc.ItemCount || 0} items</span>
                  </div>
                '}
              </div>
              <div class="flex gap-2">
                <button class="btn btn-primary btn-sm" onclick=${() => {
                  const template = {};
                  if (tableDesc) {
                    tableDesc.KeySchema.forEach(k => {
                      const attrDef = tableDesc.AttributeDefinitions.find(a => a.AttributeName === k.AttributeName);
                      const type = attrDef ? attrDef.AttributeType : 'S';
                      template[k.AttributeName] = { [type]: '' };
                    });
                  }
                  setItemJson(JSON.stringify(template, null, 2));
                  setCreateModal(true);
                }}>
                  ${icons.plus} New Item
                </button>
                <button class="btn btn-ghost btn-sm" onclick=${() => scanItems(selectedTable, null)}>
                  ${icons.refresh} Refresh
                </button>
                <button class="btn btn-danger btn-sm" onclick=${() => setDeleteConfirm(selectedTable)}>
                  ${icons.trash} Delete Table
                </button>
              </div>
            </div>

            <div class="tabs">
              <button class="tab ${activeTab === 'browse' ? 'active' : ''}" onclick=${() => setActiveTab('browse')}>Browse Items</button>
              <button class="tab ${activeTab === 'query' ? 'active' : ''}" onclick=${() => setActiveTab('query')}>Query / Scan</button>
              <button class="tab ${activeTab === 'info' ? 'active' : ''}" onclick=${() => setActiveTab('info')}>Table Info</button>
            </div>

            ${activeTab === 'browse' && html‹
              <div>
                <div class="card">
                  <div class="table-wrap">
                    <table>
                      <thead>
                        <tr>
                          ${columns.map(c => html›<th>${c}</th>')}
                          <th style="width:40px"></th>
                        </tr>
                      </thead>
                      <tbody>
                        ${items.length === 0 ? html‹
                          <tr><td colspan=${columns.length + 1} class="empty-state">No items</td></tr>
                        › : items.map((item, idx) => html‹
                          <tr class="clickable" onclick=${() => {
                            setItemJson(JSON.stringify(item, null, 2));
                            setEditModal(item);
                          }}>
                            ${columns.map(c => html›
                              <td class="font-mono text-sm truncate" style="max-width:250px">${formatDDBValue(item[c])}</td>
                            ')}
                            <td>
                              <button class="btn-icon btn-sm btn-ghost" title="Edit"
                                      onclick=${(e) => { e.stopPropagation(); setItemJson(JSON.stringify(item, null, 2)); setEditModal(item); }}>
                                ${icons.expand}
                              </button>
                            </td>
                          </tr>
                        ')}
                      </tbody>
                    </table>
                  </div>
                </div>
                <div class="pagination">
                  <button onclick=${prevPage} disabled=${page === 0}>Previous</button>
                  <span class="page-info">Page ${page + 1}</span>
                  <button onclick=${nextPage} disabled=${!lastKeys[page]}>Next</button>
                </div>
              </div>
            '}

            ${activeTab === 'query' && html‹
              <div>
                <div class="card" style="margin-bottom:16px">
                  <div class="card-body">
                    <div class="flex gap-3 mb-4">
                      <button class="btn btn-sm ${queryMode === ‹query› ? 'btn-secondary' : 'btn-ghost'}"
                              onclick=${() => setQueryMode('query')}>Query</button>
                      <button class="btn btn-sm ${queryMode === 'scan' ? 'btn-secondary' : 'btn-ghost'}"
                              onclick=${() => setQueryMode('scan')}>Scan</button>
                    </div>
                    ${queryMode === 'query' && html‹
                      <div class="mb-4">
                        <div class="label">Key Condition Expression</div>
                        <input class="input w-full" placeholder="pk = :pk AND sk BEGINS_WITH :prefix"
                               value=${queryExpr} onInput=${(e) => setQueryExpr(e.target.value)} />
                      </div>
                    ›}
                    <div class="mb-4">
                      <div class="label">Filter Expression</div>
                      <input class="input w-full" placeholder="attribute_exists(name)"
                             value=${filterExpr} onInput=${(e) => setFilterExpr(e.target.value)} />
                    </div>
                    <div class="mb-4">
                      <div class="label">Expression Attribute Values (JSON)</div>
                      <textarea class="textarea" style="min-height:80px" value=${exprAttrValues}
                                onInput=${(e) => setExprAttrValues(e.target.value)}></textarea>
                    </div>
                    <div class="flex gap-2">
                      <button class="btn btn-primary btn-sm" onclick=${runQuery}>
                        ${icons.play} Run ${queryMode === 'query' ? 'Query' : 'Scan'}
                      </button>
                      ${queryResults && html‹
                        <button class="btn btn-ghost btn-sm" onclick=${() => {
                          copyToClipboard(JSON.stringify(queryResults, null, 2));
                          showToast('Exported to clipboard');
                        }}>Export JSON</button>
                      '}
                    </div>
                  </div>
                </div>

                ${queryResults && html‹
                  <div class="card">
                    <div class="card-header">
                      <span style="font-weight:600">${queryResults.length} results</span>
                    </div>
                    <div class="card-body">
                      <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(queryResults) }}></div>
                    </div>
                  </div>
                ›}
              </div>
            '}

            ${activeTab === 'info' && tableDesc && html‹
              <div class="card">
                <div class="card-body">
                  <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(tableDesc) }}></div>
                </div>
              </div>
            ›}
          </div>
        '}
      </div>

      ${editModal && html‹
        <div class="modal-backdrop" onclick=${() => setEditModal(null)}>
          <div class="modal modal-lg" onclick=${(e) => e.stopPropagation()}>
            <div class="modal-header">
              <h3>Edit Item</h3>
              <button class="btn-icon btn-sm btn-ghost" onclick=${() => setEditModal(null)}>${icons.x}</button>
            </div>
            <div class="modal-body">
              <textarea class="textarea" style="min-height:300px" value=${itemJson}
                        onInput=${(e) => setItemJson(e.target.value)}></textarea>
            </div>
            <div class="modal-footer">
              <button class="btn btn-danger btn-sm" onclick=${() => deleteItem(editModal)}>Delete</button>
              <button class="btn btn-ghost btn-sm" onclick=${() => setEditModal(null)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onclick=${() => saveItem(itemJson)}>Save</button>
            </div>
          </div>
        </div>
      ›}

      ${createModal && html‹
        <div class="modal-backdrop" onclick=${() => setCreateModal(false)}>
          <div class="modal modal-lg" onclick=${(e) => e.stopPropagation()}>
            <div class="modal-header">
              <h3>New Item</h3>
              <button class="btn-icon btn-sm btn-ghost" onclick=${() => setCreateModal(false)}>${icons.x}</button>
            </div>
            <div class="modal-body">
              <textarea class="textarea" style="min-height:300px" value=${itemJson}
                        onInput=${(e) => setItemJson(e.target.value)}></textarea>
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost btn-sm" onclick=${() => setCreateModal(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onclick=${() => saveItem(itemJson)}>Create</button>
            </div>
          </div>
        </div>
      ›}

      ${createTableModal && html‹
        <div class="modal-backdrop" onclick=${() => setCreateTableModal(false)}>
          <div class="modal modal-md" onclick=${(e) => e.stopPropagation()}>
            <div class="modal-header">
              <h3>Create Table</h3>
              <button class="btn-icon btn-sm btn-ghost" onclick=${() => setCreateTableModal(false)}>${icons.x}</button>
            </div>
            <div class="modal-body">
              <div class="label">Table Name</div>
              <input class="input w-full mb-4" value=${newTableName}
                     onInput=${(e) => setNewTableName(e.target.value)} placeholder="my-table" />

              <div class="label">Partition Key</div>
              <div class="field-row mb-4">
                <input class="input" value=${newTablePK}
                       onInput=${(e) => setNewTablePK(e.target.value)} placeholder="id" />
                <select class="select" value=${newTablePKType}
                        onchange=${(e) => setNewTablePKType(e.target.value)}>
                  <option value="S">String</option>
                  <option value="N">Number</option>
                  <option value="B">Binary</option>
                </select>
              </div>

              <div class="label">Sort Key (optional)</div>
              <div class="field-row">
                <input class="input" value=${newTableSK}
                       onInput=${(e) => setNewTableSK(e.target.value)} placeholder="sk" />
                <select class="select" value=${newTableSKType}
                        onchange=${(e) => setNewTableSKType(e.target.value)}>
                  <option value="S">String</option>
                  <option value="N">Number</option>
                  <option value="B">Binary</option>
                </select>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost btn-sm" onclick=${() => setCreateTableModal(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onclick=${createTable} disabled=${!newTableName}>Create Table</button>
            </div>
          </div>
        </div>
      ›}

      ${deleteConfirm && html‹
        <div class="modal-backdrop" onclick=${() => setDeleteConfirm(null)}>
          <div class="modal modal-sm" onclick=${(e) => e.stopPropagation()}>
            <div class="modal-header">
              <h3>Delete Table</h3>
              <button class="btn-icon btn-sm btn-ghost" onclick=${() => setDeleteConfirm(null)}>${icons.x}</button>
            </div>
            <div class="modal-body">
              <p>Are you sure you want to delete <strong>${deleteConfirm}</strong>? This action cannot be undone.</p>
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost btn-sm" onclick=${() => setDeleteConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onclick=${() => deleteTable(deleteConfirm)}>Delete</button>
            </div>
          </div>
        </div>
      ›}
    </div>
  ';
}

// ─── Resources Page ────────────────────────────────────────────
function ResourcesPage({ services }) {
  const [selected, setSelected] = useState(null);
  const [resources, setResources] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(location.hash.split('?')[1] || '');
    const svc = params.get('service');
    if (svc) selectService(svc);
  }, []);

  function selectService(name) {
    setSelected(name);
    setLoading(true);
    api('/api/resources/' + name).then(r => {
      setResources(r.resources);
      setLoading(false);
    }).catch(() => { setResources(null); setLoading(false); });
  }

  return html‹
    <div>
      <div class="mb-6">
        <h1 class="page-title">Resource Explorer</h1>
        <p class="page-desc">Browse resources across all AWS services</p>
      </div>

      <div class="flex gap-4" style="height:calc(100vh - var(--header-height) - 120px)">
        <div style="width:220px;flex-shrink:0;overflow-y:auto">
          <div class="card" style="height:100%%">
            <div class="card-body" style="padding:8px">
              ${services.map(svc => html›
                <div class="nav-item ${selected === svc.name ? 'active' : ''}" style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                     onclick=${() => selectService(svc.name)}>
                  <span>${svc.name}</span>
                </div>
              ')}
            </div>
          </div>
        </div>

        <div style="flex:1;overflow-y:auto">
          ${!selected ? html‹
            <div class="empty-state">Select a service to browse resources</div>
          › : loading ? html‹
            <div class="empty-state">Loading...</div>
          › : html‹
            <div class="card">
              <div class="card-header">
                <h3 style="font-weight:700">${selected} Resources</h3>
              </div>
              <div class="card-body">
                <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(resources) }}></div>
              </div>
            </div>
          ›}
        </div>
      </div>
    </div>
  ';
}

// ─── Lambda Page ───────────────────────────────────────────────
function LambdaPage({ sse }) {
  const [logs, setLogs] = useState([]);
  const [functions, setFunctions] = useState([]);
  const [selected, setSelected] = useState('');
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadLogs('');
  }, []);

  function loadLogs(fn) {
    const params = new URLSearchParams();
    if (fn) params.set('function', fn);
    params.set('limit', '100');
    api('/api/lambda/logs?' + params.toString()).then(r => {
      setLogs(r || []);
      const fns = [...new Set((r || []).map(l => l.function_name).filter(Boolean))].sort();
      setFunctions(fns);
    }).catch(() => {});
  }

  // SSE for live lambda logs
  useEffect(() => {
    if (sse.events.length === 0) return;
    const latest = sse.events[0];
    if (latest && latest.type === 'lambda_log' && latest.data) {
      setLogs(prev => [latest.data, ...prev].slice(0, 500));
    }
  }, [sse.events]);

  function selectFunction(fn) {
    setSelected(fn);
    loadLogs(fn);
  }

  const filtered = useMemo(() => {
    if (!search) return logs;
    const q = search.toLowerCase();
    return logs.filter(l => (l.message || '').toLowerCase().includes(q) || (l.function_name || '').toLowerCase().includes(q));
  }, [logs, search]);

  return html‹
    <div>
      <div class="mb-6">
        <h1 class="page-title">Lambda Logs</h1>
        <p class="page-desc">Function execution logs and metrics</p>
      </div>

      <div class="flex gap-4" style="height:calc(100vh - var(--header-height) - 120px)">
        <div style="width:220px;flex-shrink:0">
          <div class="card" style="height:100%%">
            <div class="card-header" style="padding:12px 16px">
              <span style="font-weight:600;font-size:14px">Functions</span>
            </div>
            <div class="card-body" style="padding:4px 8px;overflow-y:auto" id="lambda-filter">
              <div class="nav-item ${!selected ? ‹active› : ''}" style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                   onclick=${() => selectFunction('')}>All Functions</div>
              ${functions.map(fn => html‹
                <div class="nav-item ${selected === fn ? ‹active› : ''}" style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                     onclick=${() => selectFunction(fn)}>
                  <span class="truncate">${fn}</span>
                </div>
              ')}
            </div>
          </div>
        </div>

        <div style="flex:1;overflow:hidden;display:flex;flex-direction:column">
          <div class="filters-bar">
            <input class="input input-search" placeholder="Search logs..." style="flex:1"
                   value=${search} onInput=${(e) => setSearch(e.target.value)} />
            <button class="btn btn-ghost btn-sm" onclick=${() => loadLogs(selected)}>
              ${icons.refresh} Refresh
            </button>
          </div>
          <div class="card" style="flex:1;overflow:hidden;display:flex;flex-direction:column">
            <div class="table-wrap" style="flex:1;overflow-y:auto">
              <table id="lambda-table">
                <thead>
                  <tr>
                    <th style="width:100px">Time</th>
                    <th style="width:180px">Function</th>
                    <th style="width:120px">Request ID</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody id="lambda-tbody">
                  ${filtered.length === 0 ? html‹
                    <tr><td colspan="4" class="empty-state">No logs</td></tr>
                  › : filtered.map(l => html‹
                    <tr class="${l.stream === ‹stderr› ? 'stderr' : ''}">
                      <td class="font-mono text-sm">${fmtTime(l.timestamp || l.time)}</td>
                      <td class="truncate" style="max-width:180px">${l.function_name || ''}</td>
                      <td class="font-mono text-sm truncate" style="max-width:120px">${l.request_id || ''}</td>
                      <td class="font-mono text-sm ${l.stream === 'stderr' ? 'stderr' : ''}" style="white-space:pre-wrap">${l.message || ''}</td>
                    </tr>
                  ')}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  ';
}

// ─── IAM Page ──────────────────────────────────────────────────
function IAMPage({ showToast }) {
  const [principal, setPrincipal] = useState('');
  const [action, setAction] = useState('');
  const [resource, setResource] = useState('');
  const [result, setResult] = useState(null);
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(false);

  function evaluate() {
    if (!principal || !action || !resource) {
      showToast('All fields are required');
      return;
    }
    setLoading(true);
    fetch(ADMIN + '/api/iam/evaluate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ principal, action, resource }),
    }).then(r => r.json()).then(r => {
      setResult(r);
      setHistory(prev => [{ principal, action, resource, decision: r.decision, time: new Date().toISOString() }, ...prev].slice(0, 50));
      setLoading(false);
    }).catch(() => { showToast('Evaluation failed'); setLoading(false); });
  }

  return html‹
    <div>
      <div class="mb-6">
        <h1 class="page-title">IAM Debugger</h1>
        <p class="page-desc">Evaluate IAM policies against principals and resources</p>
      </div>

      <div class="flex gap-4">
        <div style="flex:1">
          <div class="card">
            <div class="card-header">
              <h3 style="font-weight:700">Policy Evaluation</h3>
            </div>
            <div class="card-body">
              <div class="mb-4">
                <div class="label">Principal ARN</div>
                <input class="input w-full" placeholder="arn:aws:iam::123456789012:user/admin"
                       value=${principal} onInput=${(e) => setPrincipal(e.target.value)} />
              </div>
              <div class="mb-4">
                <div class="label">Action</div>
                <input class="input w-full" placeholder="s3:GetObject"
                       value=${action} onInput=${(e) => setAction(e.target.value)} />
              </div>
              <div class="mb-4">
                <div class="label">Resource ARN</div>
                <input class="input w-full" placeholder="arn:aws:s3:::my-bucket/*"
                       value=${resource} onInput=${(e) => setResource(e.target.value)} />
              </div>
              <button class="btn btn-primary" onclick=${evaluate} disabled=${loading}>
                ${loading ? ‹Evaluating...› : 'Evaluate'}
              </button>

              ${result && html‹
                <div class="iam-result ${result.decision === ‹ALLOW› ? 'iam-allow' : 'iam-deny'}">
                  ${result.decision}
                </div>
                ${result.reason && html‹<p class="text-sm text-muted mb-4">${result.reason}</p>›}
                ${result.matched_statement && html‹
                  <div>
                    <div class="section-title">Matched Statement</div>
                    <div class="json-view" dangerouslySetInnerHTML=${{ __html: syntaxHighlight(result.matched_statement) }}></div>
                  </div>
                ›}
              '}
            </div>
          </div>
        </div>

        <div style="width:360px">
          <div class="card">
            <div class="card-header">
              <h3 style="font-weight:700">History</h3>
            </div>
            <div class="card-body" style="max-height:500px;overflow-y:auto">
              ${history.length === 0 ? html‹
                <div class="text-sm text-muted" style="text-align:center;padding:24px">No evaluations yet</div>
              › : history.map(h => html‹
                <div style="padding:8px 0;border-bottom:1px solid var(--n100);font-size:13px">
                  <div class="flex items-center gap-2">
                    <span class="status-pill ${h.decision === ‹ALLOW› ? 'status-2xx' : 'status-5xx'}" style="font-size:11px">${h.decision}</span>
                    <span class="font-mono">${h.action}</span>
                  </div>
                  <div class="text-muted truncate" style="margin-top:2px">${h.resource}</div>
                </div>
              ')}
            </div>
          </div>
        </div>
      </div>
    </div>
  ';
}

// ─── Mail Page ─────────────────────────────────────────────────
function MailPage() {
  const [emails, setEmails] = useState([]);
  const [selected, setSelected] = useState(null);
  const [detail, setDetail] = useState(null);

  useEffect(() => {
    api('/api/ses/emails').then(setEmails).catch(() => {});
  }, []);

  function viewEmail(email) {
    setSelected(email.message_id);
    api('/api/ses/emails/' + email.message_id).then(setDetail).catch(() => {});
  }

  return html‹
    <div>
      <div class="mb-6">
        <h1 class="page-title">SES Mailbox</h1>
        <p class="page-desc">Captured email messages from /api/ses/emails</p>
      </div>

      <div class="flex gap-4">
        <div style="flex:1">
          ${emails.length === 0 ? html›
            <div class="card">
              <div class="empty-state" style="padding:48px">No emails captured yet</div>
            </div>
          ' : html‹
            <div class="mail-list">
              ${emails.map(e => html›
                <div class="mail-row ${selected === e.message_id ? 'expanded' : ''}"
                     onclick=${() => viewEmail(e)}>
                  <div class="mail-from truncate">${e.source || 'Unknown'}</div>
                  <div class="mail-subject truncate">${e.subject || '(no subject)'}</div>
                  <div class="mail-date">${e.timestamp ? fmtTime(e.timestamp) : ''}</div>
                </div>
              ')}
            </div>
          '}
        </div>

        ${detail && html‹
          <div style="width:45%%;flex-shrink:0">
            <div class="card">
              <div class="card-header">
                <div style="flex:1">
                  <h3 style="font-weight:700;font-size:16px">${detail.subject || ›(no subject)'}</h3>
                  <div class="text-sm text-muted mt-4">
                    From: ${detail.source || ''} | To: ${(detail.to_addresses || []).join(', ')}
                  </div>
                </div>
                <button class="btn-icon btn-sm btn-ghost" onclick=${() => { setDetail(null); setSelected(null); }}>${icons.x}</button>
              </div>
              <div class="card-body">
                ${detail.html_body ? html‹
                  <div dangerouslySetInnerHTML=${{ __html: detail.html_body }}></div>
                › : html‹
                  <pre style="white-space:pre-wrap;font-size:14px">${detail.text_body || detail.body || '(empty)'}</pre>
                '}
              </div>
            </div>
          </div>
        '}
      </div>
    </div>
  ';
}

// ─── Topology Page ─────────────────────────────────────────────
function TopologyPage() {
  const [topology, setTopology] = useState(null);
  const svgRef = useRef(null);

  useEffect(() => {
    api('/api/topology').then(setTopology).catch(() => {});
  }, []);

  useEffect(() => {
    if (!topology || !svgRef.current) return;
    renderTopologySVG(svgRef.current, topology);
  }, [topology]);

  return html‹
    <div>
      <div class="mb-6">
        <h1 class="page-title">Service Topology</h1>
        <p class="page-desc">Connections between AWS service mocks</p>
      </div>
      <div class="card topology-container">
        <svg ref=${svgRef}></svg>
      </div>
    </div>
  ›;
}

function renderTopologySVG(svg, data) {
  if (!data || !data.nodes || data.nodes.length === 0) {
    svg.innerHTML = '<text x="50%%" y="50%%" text-anchor="middle" fill="#94A3B8" font-family="Figtree" font-size="16">No topology data</text>';
    return;
  }

  const nodes = data.nodes;
  const edges = data.edges || [];
  const W = svg.clientWidth || 900;
  const H = svg.clientHeight || 600;

  // Simple force-directed layout (basic grid fallback)
  const cols = Math.ceil(Math.sqrt(nodes.length));
  const cellW = W / (cols + 1);
  const cellH = H / (Math.ceil(nodes.length / cols) + 1);

  const positions = {};
  nodes.forEach((node, i) => {
    const row = Math.floor(i / cols);
    const col = i %% cols;
    positions[node.id] = {
      x: cellW * (col + 1),
      y: cellH * (row + 1),
    };
  });

  let svgContent = '<defs><marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto"><polygon points="0 0, 10 3.5, 0 7" fill="#94A3B8" /></marker></defs>';

  // Edges
  edges.forEach(edge => {
    const s = positions[edge.source];
    const t = positions[edge.target];
    if (!s || !t) return;
    svgContent += '<line x1="' + s.x + '" y1="' + s.y + '" x2="' + t.x + '" y2="' + t.y + '" stroke="#CBD5E1" stroke-width="1.5" marker-end="url(#arrowhead)" />';
    const mx = (s.x + t.x) / 2;
    const my = (s.y + t.y) / 2;
    svgContent += '<text x="' + mx + '" y="' + (my - 6) + '" text-anchor="middle" font-size="10" fill="#94A3B8" font-family="Figtree">' + (edge.label || '') + '</text>';
  });

  // Nodes
  nodes.forEach(node => {
    const p = positions[node.id];
    svgContent += '<g>';
    svgContent += '<rect x="' + (p.x - 50) + '" y="' + (p.y - 18) + '" width="100" height="36" rx="8" fill="#0A1F44" stroke="#097FF5" stroke-width="1.5" />';
    svgContent += '<text x="' + p.x + '" y="' + (p.y + 5) + '" text-anchor="middle" font-size="12" fill="white" font-family="Figtree" font-weight="600">' + node.name + '</text>';
    svgContent += '</g>';
  });

  svg.innerHTML = svgContent;
}

// ─── Command Palette ───────────────────────────────────────────
function CommandPalette({ services, onClose }) {
  const [query, setQuery] = useState('');
  const [activeIdx, setActiveIdx] = useState(0);
  const inputRef = useRef(null);

  useEffect(() => {
    if (inputRef.current) inputRef.current.focus();
  }, []);

  const commands = useMemo(() => {
    const items = [
      { label: 'Services', desc: 'View all services', action: () => { location.hash = '/'; onClose(); } },
      { label: 'Requests', desc: 'Request log', action: () => { location.hash = '/requests'; onClose(); } },
      { label: 'DynamoDB', desc: 'Table browser', action: () => { location.hash = '/dynamodb'; onClose(); } },
      { label: 'Resources', desc: 'Resource explorer', action: () => { location.hash = '/resources'; onClose(); } },
      { label: 'Lambda Logs', desc: 'Function logs', action: () => { location.hash = '/lambda'; onClose(); } },
      { label: 'IAM Debugger', desc: 'Policy evaluation', action: () => { location.hash = '/iam'; onClose(); } },
      { label: 'Mail', desc: 'SES emails', action: () => { location.hash = '/mail'; onClose(); } },
      { label: 'Topology', desc: 'Service map', action: () => { location.hash = '/topology'; onClose(); } },
      { label: 'Reset All Services', desc: 'Clear all state', action: () => {
        api('/api/reset', { method: 'POST' }).then(() => { onClose(); location.reload(); });
      }},
    ];
    (services || []).forEach(svc => {
      items.push({ label: svc.name, desc: 'Service', action: () => { location.hash = '/resources?service=' + svc.name; onClose(); } });
    });
    if (!query) return items;
    const q = query.toLowerCase();
    return items.filter(i => i.label.toLowerCase().includes(q) || i.desc.toLowerCase().includes(q));
  }, [query, services]);

  function handleKeyDown(e) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActiveIdx(i => Math.min(i + 1, commands.length - 1)); }
    if (e.key === 'ArrowUp') { e.preventDefault(); setActiveIdx(i => Math.max(i - 1, 0)); }
    if (e.key === 'Enter' && commands[activeIdx]) { commands[activeIdx].action(); }
    if (e.key === 'Escape') { onClose(); }
  }

  return html‹
    <div class="palette-backdrop" onclick=${onClose}>
      <div class="palette" onclick=${(e) => e.stopPropagation()}>
        <input class="palette-input" ref=${inputRef} placeholder="Search commands, services..."
               value=${query} onInput=${(e) => { setQuery(e.target.value); setActiveIdx(0); }}
               onKeyDown=${handleKeyDown} />
        <div class="palette-results">
          ${commands.map((cmd, i) => html›
            <div class="palette-item ${i === activeIdx ? 'active' : ''}"
                 onclick=${cmd.action}
                 onMouseEnter=${() => setActiveIdx(i)}>
              <span class="label">${cmd.label}</span>
              <span class="desc">${cmd.desc}</span>
            </div>
          ')}
          ${commands.length === 0 && html‹
            <div style="padding:24px;text-align:center;color:var(--n400);font-size:14px">No results</div>
          ›}
        </div>
      </div>
    </div>
  ';
}

// ─── Mount ─────────────────────────────────────────────────────
render(html‹<${App} />›, document.getElementById('app'));
</script>
</body>
</html>
`
