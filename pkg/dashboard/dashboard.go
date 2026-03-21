// Package dashboard provides a single-page web dashboard for cloudmock,
// served on the dashboard port and talking to the admin API.
package dashboard

import (
	"fmt"
	"net/http"
)

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
	return fmt.Sprintf(htmlTemplate, adminBase)
}

// htmlTemplate is the complete SPA. The single %%q verb is replaced with the
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
    --brand-teal: #4AE5F8;
    --brand-orange: #F7711E;
    --brand-yellow: #FEC307;
    --success: #029662;
    --warning: #FF9A4B;
    --error: #FF4E5E;
    --info: #7CCEF2;
    --neutral-50: #F8FAFC;
    --neutral-100: #F1F5F9;
    --neutral-200: #E2E8F0;
    --neutral-300: #CBD5E1;
    --neutral-400: #94A3B8;
    --neutral-500: #64748B;
    --neutral-600: #475569;
    --neutral-700: #334155;
    --neutral-800: #1E293B;
    --neutral-900: #0F172A;
    --font-sans: 'Figtree', -apple-system, BlinkMacSystemFont, sans-serif;
    --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
    --radius-sm: 4px;
    --radius-md: 8px;
    --radius-lg: 12px;
    --radius-xl: 16px;
    --shadow-sm: 0 1px 3px rgba(0,0,0,0.1);
    --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1);
    --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1);
  }

  body {
    font-family: var(--font-sans);
    font-size: 14px;
    background: var(--neutral-50);
    color: var(--neutral-800);
    min-height: 100vh;
    overflow: hidden;
  }

  /* ─── Layout ─── */
  #app {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  /* ─── Header ─── */
  .header {
    background: var(--brand-dark);
    color: #fff;
    padding: 0 24px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    z-index: 200;
    flex-shrink: 0;
  }
  .header-brand {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: -0.5px;
    display: flex;
    align-items: center;
    gap: 8px;
    color: #fff;
  }
  .header-brand svg { width: 24px; height: 24px; }
  .header-brand span { color: var(--brand-teal); }
  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .sse-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
    color: var(--neutral-400);
  }
  .sse-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%%;
    background: var(--neutral-400);
    transition: background 0.3s ease;
  }
  .sse-dot.connected { background: var(--success); box-shadow: 0 0 6px rgba(2,150,98,0.5); }
  .sse-dot.disconnected { background: var(--error); box-shadow: 0 0 6px rgba(255,78,94,0.5); }
  .cmd-k-hint {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--neutral-400);
    background: rgba(255,255,255,0.08);
    border: 1px solid rgba(255,255,255,0.12);
    border-radius: var(--radius-sm);
    padding: 4px 10px;
    cursor: pointer;
    transition: background 0.2s ease;
  }
  .cmd-k-hint:hover { background: rgba(255,255,255,0.14); }
  .cmd-k-hint kbd {
    font-family: var(--font-mono);
    font-size: 11px;
    background: rgba(255,255,255,0.1);
    border-radius: 3px;
    padding: 1px 5px;
  }

  /* ─── Body Layout ─── */
  .body-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  /* ─── Sidebar ─── */
  .sidebar {
    width: 200px;
    background: var(--brand-dark);
    color: #fff;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    border-right: 1px solid rgba(255,255,255,0.06);
  }
  .sidebar-nav {
    flex: 1;
    padding: 12px 0;
    overflow-y: auto;
  }
  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    color: var(--neutral-400);
    cursor: pointer;
    transition: all 0.2s ease;
    border-left: 3px solid transparent;
    text-decoration: none;
  }
  .nav-item:hover {
    color: #fff;
    background: rgba(255,255,255,0.04);
  }
  .nav-item.active {
    color: #fff;
    background: rgba(9,127,245,0.12);
    border-left-color: var(--brand-blue);
  }
  .nav-item svg {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
  }
  .nav-badge {
    margin-left: auto;
    background: var(--brand-blue);
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    padding: 1px 7px;
    border-radius: 10px;
    min-width: 20px;
    text-align: center;
  }
  .sidebar-footer {
    padding: 16px;
    border-top: 1px solid rgba(255,255,255,0.06);
    font-size: 11px;
    color: var(--neutral-500);
  }

  /* ─── Main Content ─── */
  .main-content {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
  }

  /* ─── Cards ─── */
  .card {
    background: #fff;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
    transition: box-shadow 0.2s ease;
  }
  .card:hover { box-shadow: var(--shadow-md); }
  .card-header {
    padding: 16px 20px;
    border-bottom: 1px solid var(--neutral-200);
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: #fff;
  }
  .card-header h2 {
    font-size: 14px;
    font-weight: 600;
    color: var(--neutral-700);
  }
  .card-body { padding: 0; }
  .card-body-padded { padding: 20px; }

  /* ─── Stats Bar ─── */
  .stats-bar {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 16px;
    margin-bottom: 24px;
  }
  .stat-card {
    background: #fff;
    border-radius: var(--radius-lg);
    padding: 20px;
    box-shadow: var(--shadow-sm);
    transition: box-shadow 0.2s ease, transform 0.2s ease;
  }
  .stat-card:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
  .stat-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--neutral-500);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }
  .stat-value {
    font-size: 28px;
    font-weight: 700;
    color: var(--neutral-900);
  }
  .stat-value.success { color: var(--success); }
  .stat-value.brand { color: var(--brand-blue); }

  /* ─── Service Grid ─── */
  .service-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }
  @media (max-width: 1200px) { .service-grid { grid-template-columns: repeat(2, 1fr); } }
  .service-card {
    background: #fff;
    border-radius: var(--radius-lg);
    padding: 20px;
    box-shadow: var(--shadow-sm);
    cursor: pointer;
    transition: all 0.2s ease;
    border: 1px solid transparent;
  }
  .service-card:hover {
    box-shadow: var(--shadow-md);
    transform: translateY(-2px);
    border-color: var(--brand-blue);
  }
  .svc-card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }
  .svc-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--neutral-900);
  }
  .svc-tier {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 10px;
    background: var(--neutral-100);
    color: var(--neutral-500);
    text-transform: uppercase;
  }
  .svc-tier.tier-1 { background: rgba(9,127,245,0.1); color: var(--brand-blue); }
  .svc-card-meta {
    display: flex;
    align-items: center;
    gap: 16px;
    font-size: 12px;
    color: var(--neutral-500);
  }
  .svc-status {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%%;
  }
  .status-dot.healthy { background: var(--success); }
  .status-dot.degraded { background: var(--warning); }
  .status-dot.error { background: var(--error); }
  .svc-requests {
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--brand-blue);
  }

  /* ─── Tables ─── */
  table { width: 100%%; border-collapse: collapse; }
  th {
    text-align: left;
    padding: 10px 16px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--neutral-500);
    background: var(--neutral-50);
    border-bottom: 1px solid var(--neutral-200);
    white-space: nowrap;
  }
  td {
    padding: 10px 16px;
    border-bottom: 1px solid var(--neutral-100);
    color: var(--neutral-700);
    font-size: 13px;
  }
  tr:last-child td { border-bottom: none; }
  tbody tr { transition: background 0.15s ease; cursor: pointer; }
  tbody tr:hover td { background: var(--neutral-50); }

  .mono { font-family: var(--font-mono); font-size: 12px; }
  .empty-state {
    text-align: center;
    color: var(--neutral-400);
    padding: 48px 24px;
    font-size: 14px;
  }

  /* ─── Status Codes ─── */
  .status-code {
    display: inline-block;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 600;
    font-family: var(--font-mono);
  }
  .status-2xx { background: rgba(2,150,98,0.1); color: var(--success); }
  .status-3xx { background: rgba(124,206,242,0.2); color: var(--info); }
  .status-4xx { background: rgba(254,195,7,0.15); color: #92700C; }
  .status-5xx { background: rgba(255,78,94,0.1); color: var(--error); }

  /* ─── Breadcrumb ─── */
  .breadcrumb {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--neutral-500);
    margin-bottom: 20px;
  }
  .breadcrumb a {
    color: var(--brand-blue);
    text-decoration: none;
    cursor: pointer;
  }
  .breadcrumb a:hover { text-decoration: underline; }

  /* ─── Search / Filters ─── */
  .search-input {
    font-family: var(--font-sans);
    font-size: 13px;
    padding: 8px 12px;
    border: 1px solid var(--neutral-200);
    border-radius: var(--radius-md);
    background: #fff;
    color: var(--neutral-800);
    outline: none;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
    width: 240px;
  }
  .search-input:focus {
    border-color: var(--brand-blue);
    box-shadow: 0 0 0 3px rgba(9,127,245,0.1);
  }
  .filter-select {
    font-family: var(--font-sans);
    font-size: 13px;
    padding: 8px 12px;
    border: 1px solid var(--neutral-200);
    border-radius: var(--radius-md);
    background: #fff;
    color: var(--neutral-700);
    cursor: pointer;
    outline: none;
    transition: border-color 0.2s ease;
  }
  .filter-select:focus { border-color: var(--brand-blue); }
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  /* ─── Buttons ─── */
  .btn {
    font-family: var(--font-sans);
    font-size: 13px;
    font-weight: 600;
    padding: 8px 16px;
    border-radius: var(--radius-md);
    border: none;
    cursor: pointer;
    transition: all 0.2s ease;
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .btn-primary {
    background: var(--brand-blue);
    color: #fff;
  }
  .btn-primary:hover { background: #0870D9; }
  .btn-primary:active { background: #0660BF; }
  .btn-ghost {
    background: transparent;
    color: var(--neutral-600);
    border: 1px solid var(--neutral-200);
  }
  .btn-ghost:hover { background: var(--neutral-50); border-color: var(--neutral-300); }
  .btn-danger {
    background: rgba(255,78,94,0.1);
    color: var(--error);
  }
  .btn-danger:hover { background: rgba(255,78,94,0.2); }
  .btn-sm { padding: 4px 10px; font-size: 12px; }

  /* ─── Drawer ─── */
  .drawer-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.4);
    z-index: 300;
    opacity: 0;
    transition: opacity 0.2s ease;
    pointer-events: none;
  }
  .drawer-overlay.open { opacity: 1; pointer-events: auto; }
  .drawer {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: 42%%;
    min-width: 400px;
    max-width: 700px;
    background: #fff;
    z-index: 301;
    transform: translateX(100%%);
    transition: transform 0.25s ease;
    display: flex;
    flex-direction: column;
    box-shadow: -4px 0 20px rgba(0,0,0,0.15);
  }
  .drawer.open { transform: translateX(0); }
  .drawer-header {
    padding: 20px 24px;
    border-bottom: 1px solid var(--neutral-200);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .drawer-header h3 {
    font-size: 16px;
    font-weight: 600;
    color: var(--neutral-900);
  }
  .drawer-close {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--neutral-400);
    padding: 4px;
    border-radius: var(--radius-sm);
    transition: color 0.2s ease, background 0.2s ease;
  }
  .drawer-close:hover { color: var(--neutral-700); background: var(--neutral-100); }
  .drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
  }
  .drawer-tabs {
    display: flex;
    border-bottom: 1px solid var(--neutral-200);
    padding: 0 24px;
  }
  .drawer-tab {
    padding: 12px 16px;
    font-size: 13px;
    font-weight: 500;
    color: var(--neutral-500);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: all 0.2s ease;
  }
  .drawer-tab:hover { color: var(--neutral-700); }
  .drawer-tab.active { color: var(--brand-blue); border-bottom-color: var(--brand-blue); }

  /* ─── Code Block ─── */
  .code-block {
    background: var(--neutral-900);
    color: var(--neutral-200);
    border-radius: var(--radius-md);
    padding: 16px;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 1.6;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .code-block .key { color: var(--brand-teal); }
  .code-block .string { color: var(--brand-yellow); }
  .code-block .number { color: var(--brand-orange); }
  .code-block .bool { color: var(--brand-blue); }

  /* ─── Request Detail Sections ─── */
  .detail-grid {
    display: grid;
    grid-template-columns: 120px 1fr;
    gap: 8px 16px;
    font-size: 13px;
  }
  .detail-label {
    font-weight: 500;
    color: var(--neutral-500);
  }
  .detail-value {
    color: var(--neutral-800);
  }

  /* ─── Inline Expand ─── */
  .expand-row td {
    padding: 0;
    background: var(--neutral-50);
  }
  .expand-content {
    padding: 16px 20px;
    border-top: 1px solid var(--neutral-200);
    font-size: 13px;
  }

  /* ─── Command Palette ─── */
  .palette-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.5);
    z-index: 400;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 20vh;
    opacity: 0;
    transition: opacity 0.15s ease;
    pointer-events: none;
  }
  .palette-overlay.open { opacity: 1; pointer-events: auto; }
  .palette {
    background: #fff;
    border-radius: var(--radius-xl);
    width: 560px;
    max-height: 420px;
    box-shadow: var(--shadow-lg), 0 0 0 1px rgba(0,0,0,0.05);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transform: scale(0.96);
    transition: transform 0.15s ease;
  }
  .palette-overlay.open .palette { transform: scale(1); }
  .palette-input {
    font-family: var(--font-sans);
    font-size: 16px;
    padding: 16px 20px;
    border: none;
    outline: none;
    width: 100%%;
    border-bottom: 1px solid var(--neutral-200);
  }
  .palette-results {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }
  .palette-item {
    padding: 10px 14px;
    border-radius: var(--radius-md);
    font-size: 14px;
    color: var(--neutral-700);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 10px;
    transition: background 0.1s ease;
  }
  .palette-item:hover, .palette-item.selected {
    background: var(--neutral-100);
  }
  .palette-item .hint {
    margin-left: auto;
    font-size: 12px;
    color: var(--neutral-400);
    font-family: var(--font-mono);
  }

  /* ─── Resource Explorer ─── */
  .explorer-layout {
    display: flex;
    gap: 0;
    height: calc(100vh - 56px - 48px);
  }
  .explorer-sidebar {
    width: 220px;
    background: #fff;
    border-right: 1px solid var(--neutral-200);
    overflow-y: auto;
    flex-shrink: 0;
  }
  .explorer-sidebar .nav-item {
    color: var(--neutral-600);
    border-left: 3px solid transparent;
    padding: 10px 16px;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s ease;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .explorer-sidebar .nav-item:hover { background: var(--neutral-50); color: var(--neutral-800); }
  .explorer-sidebar .nav-item.active {
    background: rgba(9,127,245,0.06);
    color: var(--brand-blue);
    border-left-color: var(--brand-blue);
    font-weight: 600;
  }
  .explorer-main {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  /* ─── IAM Debugger ─── */
  .iam-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 600px;
  }
  .form-group { display: flex; flex-direction: column; gap: 6px; }
  .form-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--neutral-600);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  .form-input {
    font-family: var(--font-mono);
    font-size: 13px;
    padding: 10px 12px;
    border: 1px solid var(--neutral-200);
    border-radius: var(--radius-md);
    outline: none;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }
  .form-input:focus {
    border-color: var(--brand-blue);
    box-shadow: 0 0 0 3px rgba(9,127,245,0.1);
  }
  .iam-result {
    margin-top: 20px;
    padding: 20px;
    border-radius: var(--radius-lg);
    font-size: 14px;
  }
  .iam-result.allow { background: rgba(2,150,98,0.08); border: 1px solid rgba(2,150,98,0.2); }
  .iam-result.deny { background: rgba(255,78,94,0.06); border: 1px solid rgba(255,78,94,0.2); }
  .iam-result-decision {
    font-size: 20px;
    font-weight: 700;
    margin-bottom: 8px;
  }
  .iam-result.allow .iam-result-decision { color: var(--success); }
  .iam-result.deny .iam-result-decision { color: var(--error); }

  /* ─── SES Mailbox ─── */
  .email-preview {
    padding: 20px;
    border-bottom: 1px solid var(--neutral-100);
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .email-preview:hover { background: var(--neutral-50); }
  .email-preview:last-child { border-bottom: none; }
  .email-from { font-weight: 600; color: var(--neutral-800); margin-bottom: 4px; }
  .email-subject { color: var(--neutral-700); margin-bottom: 4px; }
  .email-to { font-size: 12px; color: var(--neutral-500); }
  .email-date { font-size: 12px; color: var(--neutral-400); }
  .email-meta {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
  }

  /* ─── Topology ─── */
  .topology-container {
    position: relative;
    min-height: 500px;
    background: #fff;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }
  .topo-node {
    position: absolute;
    padding: 12px 20px;
    background: #fff;
    border: 2px solid var(--neutral-200);
    border-radius: var(--radius-lg);
    font-size: 13px;
    font-weight: 600;
    color: var(--neutral-800);
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: var(--shadow-sm);
    z-index: 2;
  }
  .topo-node:hover {
    border-color: var(--brand-blue);
    box-shadow: var(--shadow-md);
    transform: scale(1.05);
  }
  .topo-svg {
    position: absolute;
    inset: 0;
    width: 100%%;
    height: 100%%;
    z-index: 1;
  }
  .topo-edge { stroke: var(--neutral-300); stroke-width: 2; fill: none; }
  .topo-edge-label {
    font-size: 10px;
    fill: var(--neutral-400);
  }

  /* ─── Lambda Logs ─── */
  .log-stderr td {
    color: var(--error) !important;
    background: rgba(255,78,94,0.04);
  }
  .log-stderr:hover td {
    background: rgba(255,78,94,0.08) !important;
  }
  .lambda-table-wrap {
    max-height: 600px;
    overflow-y: auto;
  }

  /* ─── Fade In Animation ─── */
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .fade-in { animation: fadeIn 0.3s ease-out; }

  @keyframes slideUp {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .slide-up { animation: slideUp 0.3s ease-out; }

  @keyframes pulse {
    0%%   { box-shadow: 0 0 0 0 rgba(9,127,245,0.3); }
    70%%  { box-shadow: 0 0 0 6px rgba(9,127,245,0); }
    100%% { box-shadow: 0 0 0 0 rgba(9,127,245,0); }
  }
  .svc-pulse { animation: pulse 0.5s ease-out; }

  /* ─── Scrollbar ─── */
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb {
    background: var(--neutral-300);
    border-radius: 3px;
  }
  ::-webkit-scrollbar-thumb:hover { background: var(--neutral-400); }

  /* ─── Page Title ─── */
  .page-title {
    font-size: 20px;
    font-weight: 700;
    color: var(--neutral-900);
    margin-bottom: 20px;
  }
  .section-gap { margin-bottom: 24px; }
</style>
</head>
<body>
<div id="app"></div>

<script type="module">
import { h, render } from 'https://esm.sh/preact@10.19.3';
import { useState, useEffect, useRef, useCallback, useMemo } from 'https://esm.sh/preact@10.19.3/hooks';
import htm from 'https://esm.sh/htm@3.1.1';
const html = htm.bind(h);

const ADMIN = %q;
const MAX_ROWS = 200;

// ─── Helpers ───

function fmtTime(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  const p = n => String(n).padStart(2, '0');
  return p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds());
}

function fmtTimeFull(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  return d.toLocaleString();
}

function fmtLatency(ns) {
  if (!ns || ns === 0) return '-';
  if (ns < 1000000) return (ns / 1000).toFixed(0) + ' us';
  if (ns < 1000000000) return (ns / 1000000).toFixed(1) + ' ms';
  return (ns / 1000000000).toFixed(2) + ' s';
}

function statusClass(code) {
  if (!code) return '';
  if (code >= 200 && code < 300) return 'status-2xx';
  if (code >= 300 && code < 400) return 'status-3xx';
  if (code >= 400 && code < 500) return 'status-4xx';
  if (code >= 500) return 'status-5xx';
  return '';
}

async function apiFetch(path, opts) {
  const resp = await fetch(ADMIN + path, opts);
  if (!resp.ok) throw new Error('HTTP ' + resp.status);
  return resp.json();
}

// Sanitize text for safe display — strips any HTML tags.
function sanitizeText(s) {
  if (!s) return '';
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

function syntaxHighlight(json) {
  if (!json) return '';
  try {
    if (typeof json === 'string') json = JSON.parse(json);
    json = JSON.stringify(json, null, 2);
  } catch(e) {
    return sanitizeText(typeof json === 'string' ? json : JSON.stringify(json));
  }
  // Only highlight JSON tokens — all values are escaped via JSON.stringify.
  return json.replace(/("(\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    function(match) {
      let cls = 'number';
      if (/^"/.test(match)) {
        cls = /:$/.test(match) ? 'key' : 'string';
      } else if (/true|false/.test(match)) {
        cls = 'bool';
      }
      return '<span class="' + cls + '">' + match + '</span>';
    });
}

function fuzzyMatch(query, text) {
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  if (t.includes(q)) return true;
  let qi = 0;
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}

// ─── SVG Icons (inline, no deps) ───

const Icons = {
  services: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>` + "`" + `,
  requests: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>` + "`" + `,
  resources: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>` + "`" + `,
  lambda: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>` + "`" + `,
  iam: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>` + "`" + `,
  mail: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22,6 12,13 2,6"/></svg>` + "`" + `,
  topology: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="2"/><circle cx="20" cy="12" r="2"/><circle cx="4" cy="12" r="2"/><circle cx="12" cy="4" r="2"/><circle cx="12" cy="20" r="2"/><line x1="14" y1="12" x2="18" y2="12"/><line x1="6" y1="12" x2="10" y2="12"/><line x1="12" y1="6" x2="12" y2="10"/><line x1="12" y1="14" x2="12" y2="18"/></svg>` + "`" + `,
  close: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>` + "`" + `,
  expand: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>` + "`" + `,
  cloud: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/></svg>` + "`" + `,
  search: html` + "`" + `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>` + "`" + `,
};

// ─── SSE Hook ───

function useSSE() {
  const [connected, setConnected] = useState(false);
  const listenersRef = useRef([]);

  useEffect(() => {
    let es;
    function connect() {
      es = new EventSource(ADMIN + '/api/stream');
      es.onopen = () => setConnected(true);
      es.onerror = () => setConnected(false);
      es.onmessage = (e) => {
        try {
          const event = JSON.parse(e.data);
          listenersRef.current.forEach(fn => fn(event));
        } catch(_) {}
      };
    }
    connect();
    return () => { if (es) es.close(); };
  }, []);

  const subscribe = useCallback((fn) => {
    listenersRef.current.push(fn);
    return () => {
      listenersRef.current = listenersRef.current.filter(f => f !== fn);
    };
  }, []);

  return { connected, subscribe };
}

// ─── Router ───

function useRouter() {
  const [route, setRoute] = useState(window.location.hash || '#/');

  useEffect(() => {
    const onHash = () => setRoute(window.location.hash || '#/');
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const navigate = useCallback((hash) => {
    window.location.hash = hash;
  }, []);

  return { route, navigate };
}

// ─── Command Palette ───

function CommandPalette({ open, onClose, navigate, services }) {
  const [query, setQuery] = useState('');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const inputRef = useRef(null);

  useEffect(() => {
    if (open && inputRef.current) {
      setQuery('');
      setSelectedIdx(0);
      setTimeout(() => inputRef.current.focus(), 50);
    }
  }, [open]);

  const commands = useMemo(() => {
    const cmds = [
      { label: 'Services Overview', action: () => navigate('#/'), hint: '#/' },
      { label: 'Request Log', action: () => navigate('#/requests'), hint: '#/requests' },
      { label: 'Resource Explorer', action: () => navigate('#/resources'), hint: '#/resources' },
      { label: 'Lambda Logs', action: () => navigate('#/lambda'), hint: '#/lambda' },
      { label: 'IAM Debugger', action: () => navigate('#/iam'), hint: '#/iam' },
      { label: 'SES Mailbox', action: () => navigate('#/mail'), hint: '#/mail' },
      { label: 'Service Topology', action: () => navigate('#/topology'), hint: '#/topology' },
      { label: 'Reset All Services', action: () => { apiFetch('/api/reset', { method: 'POST' }); }, hint: 'POST' },
    ];
    (services || []).forEach(s => {
      cmds.push({
        label: 'Jump to ' + s.name,
        action: () => navigate('#/services/' + s.name),
        hint: s.name,
      });
      cmds.push({
        label: 'Reset ' + s.name,
        action: () => { apiFetch('/api/services/' + s.name + '/reset', { method: 'POST' }); },
        hint: 'POST',
      });
    });
    return cmds;
  }, [services, navigate]);

  const filtered = useMemo(() => {
    if (!query) return commands;
    return commands.filter(c => fuzzyMatch(query, c.label));
  }, [query, commands]);

  useEffect(() => { setSelectedIdx(0); }, [query]);

  const onKeyDown = (e) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, filtered.length - 1)); }
    if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)); }
    if (e.key === 'Enter' && filtered[selectedIdx]) {
      filtered[selectedIdx].action();
      onClose();
    }
    if (e.key === 'Escape') onClose();
  };

  return html` + "`" + `
    <div class="palette-overlay ${open ? 'open' : ''}" onClick=${(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div class="palette">
        <input
          ref=${inputRef}
          class="palette-input"
          placeholder="Type a command..."
          value=${query}
          onInput=${(e) => setQuery(e.target.value)}
          onKeyDown=${onKeyDown}
        />
        <div class="palette-results">
          ${filtered.map((cmd, i) => html` + "`" + `
            <div
              class="palette-item ${i === selectedIdx ? 'selected' : ''}"
              onClick=${() => { cmd.action(); onClose(); }}
              onMouseEnter=${() => setSelectedIdx(i)}
            >
              ${cmd.label}
              <span class="hint">${cmd.hint}</span>
            </div>
          ` + "`" + `)}
          ${filtered.length === 0 ? html` + "`" + `<div class="empty-state" style="padding:20px">No matching commands</div>` + "`" + ` : null}
        </div>
      </div>
    </div>
  ` + "`" + `;
}

// ─── Sidebar ───

function Sidebar({ route, navigate, emailCount }) {
  const items = [
    { hash: '#/', label: 'Services', icon: Icons.services },
    { hash: '#/requests', label: 'Requests', icon: Icons.requests },
    { hash: '#/resources', label: 'Resources', icon: Icons.resources },
    { hash: '#/lambda', label: 'Lambda', icon: Icons.lambda },
    { hash: '#/iam', label: 'IAM', icon: Icons.iam },
    { hash: '#/mail', label: 'Mail', icon: Icons.mail, badge: emailCount },
    { hash: '#/topology', label: 'Topology', icon: Icons.topology },
  ];

  const isActive = (hash) => {
    if (hash === '#/') return route === '#/' || route === '#/services';
    return route.startsWith(hash);
  };

  return html` + "`" + `
    <div class="sidebar">
      <nav class="sidebar-nav">
        ${items.map(item => html` + "`" + `
          <div
            class="nav-item ${isActive(item.hash) ? 'active' : ''}"
            onClick=${() => navigate(item.hash)}
          >
            ${item.icon}
            ${item.label}
            ${item.badge > 0 ? html` + "`" + `<span class="nav-badge">${item.badge}</span>` + "`" + ` : null}
          </div>
        ` + "`" + `)}
      </nav>
      <div class="sidebar-footer">cloudmock v0.1.0</div>
    </div>
  ` + "`" + `;
}

// ─── Services Overview Page ───

function ServicesPage({ navigate, services, stats }) {
  const [search, setSearch] = useState('');

  const totalRequests = Object.values(stats || {}).reduce((a, b) => a + b, 0);
  const healthyCount = (services || []).filter(s => s.healthy).length;

  const filtered = useMemo(() => {
    if (!services) return [];
    if (!search) return services;
    const q = search.toLowerCase();
    return services.filter(s => s.name.toLowerCase().includes(q));
  }, [services, search]);

  const tier1 = ['s3', 'dynamodb', 'sqs', 'sns', 'lambda', 'iam', 'sts', 'cloudwatch-logs',
    'rds', 'cloudformation', 'ec2', 'ecr', 'ecs', 'secretsmanager', 'ssm',
    'kinesis', 'firehose', 'events', 'stepfunctions', 'apigateway'];

  return html` + "`" + `
    <div>
      <div class="stats-bar slide-up">
        <div class="stat-card">
          <div class="stat-label">Services</div>
          <div class="stat-value brand">${(services || []).length}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Total Requests</div>
          <div class="stat-value">${totalRequests.toLocaleString()}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Healthy</div>
          <div class="stat-value success">${healthyCount}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Uptime</div>
          <div class="stat-value success">100%%</div>
        </div>
      </div>

      <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
        <h1 class="page-title" style="margin-bottom:0">Services</h1>
        <input
          class="search-input"
          placeholder="Filter services..."
          value=${search}
          onInput=${(e) => setSearch(e.target.value)}
        />
      </div>

      <div class="service-grid">
        ${filtered.map(svc => {
          const count = (stats && stats[svc.name]) || 0;
          const t1 = tier1.includes(svc.name);
          return html` + "`" + `
            <div class="service-card fade-in" onClick=${() => navigate('#/services/' + svc.name)}>
              <div class="svc-card-top">
                <span class="svc-name">${svc.name}</span>
                <span class="svc-tier ${t1 ? 'tier-1' : ''}">${t1 ? 'Tier 1' : 'Tier 2'}</span>
              </div>
              <div class="svc-card-meta">
                <span class="svc-status">
                  <span class="status-dot ${svc.healthy ? 'healthy' : 'degraded'}"></span>
                  ${svc.healthy ? 'Healthy' : 'Degraded'}
                </span>
                <span class="svc-requests">${count} reqs</span>
                <span>${svc.action_count} actions</span>
              </div>
            </div>
          ` + "`" + `;
        })}
      </div>
      ${filtered.length === 0 ? html` + "`" + `<div class="empty-state">No services match your filter</div>` + "`" + ` : null}
    </div>
  ` + "`" + `;
}

// ─── Service Detail Page ───

function ServiceDetailPage({ name, navigate, stats }) {
  const [svc, setSvc] = useState(null);
  const [svcRequests, setSvcRequests] = useState([]);

  useEffect(() => {
    apiFetch('/api/services/' + name).then(setSvc).catch(() => {});
    apiFetch('/api/requests?service=' + name + '&limit=20').then(setSvcRequests).catch(() => {});
  }, [name]);

  if (!svc) return html` + "`" + `<div class="empty-state">Loading service...</div>` + "`" + `;

  const count = (stats && stats[name]) || 0;

  return html` + "`" + `
    <div>
      <div class="breadcrumb">
        <a onClick=${() => navigate('#/')}>Services</a>
        <span>/</span>
        <span>${name}</span>
      </div>
      <h1 class="page-title">${name}</h1>

      <div class="stats-bar section-gap">
        <div class="stat-card">
          <div class="stat-label">Status</div>
          <div class="stat-value ${svc.healthy ? 'success' : ''}">${svc.healthy ? 'Healthy' : 'Degraded'}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Actions</div>
          <div class="stat-value brand">${svc.action_count}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Requests</div>
          <div class="stat-value">${count}</div>
        </div>
      </div>

      <div class="card section-gap">
        <div class="card-header"><h2>Recent Requests</h2></div>
        <div class="card-body">
          ${svcRequests.length === 0
            ? html` + "`" + `<div class="empty-state">No requests for this service yet</div>` + "`" + `
            : html` + "`" + `
              <table>
                <thead><tr><th>Time</th><th>Action</th><th>Status</th><th>Latency</th></tr></thead>
                <tbody>
                  ${svcRequests.map(r => html` + "`" + `
                    <tr>
                      <td class="mono">${fmtTime(r.timestamp)}</td>
                      <td class="mono">${r.action || '-'}</td>
                      <td><span class="status-code ${statusClass(r.status_code)}">${r.status_code || '?'}</span></td>
                      <td class="mono">${fmtLatency(r.latency_ns)}</td>
                    </tr>
                  ` + "`" + `)}
                </tbody>
              </table>
            ` + "`" + `
          }
        </div>
      </div>
    </div>
  ` + "`" + `;
}

// ─── Request Log Page ───

function RequestLogPage({ subscribe }) {
  const [requests, setRequests] = useState([]);
  const [svcFilter, setSvcFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [search, setSearch] = useState('');
  const [services, setServices] = useState([]);
  const [expandedId, setExpandedId] = useState(null);
  const [drawer, setDrawer] = useState(null);

  useEffect(() => {
    apiFetch('/api/requests?limit=100').then(setRequests).catch(() => {});
    apiFetch('/api/services').then(s => setServices((s || []).map(x => x.name))).catch(() => {});
  }, []);

  useEffect(() => {
    return subscribe((event) => {
      if (event.type === 'request') {
        setRequests(prev => {
          const next = [event.data, ...prev];
          if (next.length > MAX_ROWS) next.length = MAX_ROWS;
          return next;
        });
      }
    });
  }, [subscribe]);

  const filtered = useMemo(() => {
    return requests.filter(r => {
      if (svcFilter && r.service !== svcFilter) return false;
      if (statusFilter) {
        const code = r.status_code || 0;
        if (statusFilter === '2xx' && (code < 200 || code >= 300)) return false;
        if (statusFilter === '4xx' && (code < 400 || code >= 500)) return false;
        if (statusFilter === '5xx' && code < 500) return false;
      }
      if (search) {
        const q = search.toLowerCase();
        const haystack = (r.service + ' ' + r.action + ' ' + r.path).toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }, [requests, svcFilter, statusFilter, search]);

  const openDrawer = (r) => {
    if (r.id) {
      apiFetch('/api/requests/' + r.id).then(setDrawer).catch(() => setDrawer(r));
    } else {
      setDrawer(r);
    }
  };

  return html` + "`" + `
    <div>
      <h1 class="page-title">Request Log</h1>
      <div class="filter-bar section-gap">
        <select class="filter-select" id="service-filter" value=${svcFilter} onChange=${(e) => setSvcFilter(e.target.value)}>
          <option value="">All services</option>
          ${services.map(s => html` + "`" + `<option value=${s}>${s}</option>` + "`" + `)}
        </select>
        <select class="filter-select" value=${statusFilter} onChange=${(e) => setStatusFilter(e.target.value)}>
          <option value="">All status</option>
          <option value="2xx">2xx</option>
          <option value="4xx">4xx</option>
          <option value="5xx">5xx</option>
        </select>
        <input class="search-input" placeholder="Search..." value=${search} onInput=${(e) => setSearch(e.target.value)} />
      </div>
      <div class="card">
        <div class="card-body">
          <table id="requests-table">
            <thead>
              <tr>
                <th>Time</th><th>Service</th><th>Action</th><th>Status</th><th>Latency</th><th>Caller</th><th></th>
              </tr>
            </thead>
            <tbody id="requests-tbody">
              ${filtered.length === 0 ? html` + "`" + `<tr><td colspan="7" class="empty-state">No requests recorded yet</td></tr>` + "`" + ` : null}
              ${filtered.map(r => html` + "`" + `
                <tr class="fade-in" onClick=${() => setExpandedId(expandedId === r.id ? null : r.id)}>
                  <td class="mono">${fmtTime(r.timestamp)}</td>
                  <td>${r.service || '-'}</td>
                  <td class="mono">${r.action || '-'}</td>
                  <td><span class="status-code ${statusClass(r.status_code)}">${r.status_code || '?'}</span></td>
                  <td class="mono">${fmtLatency(r.latency_ns)}</td>
                  <td class="mono" style="font-size:11px">${r.caller_id || '-'}</td>
                  <td>
                    <button class="btn btn-ghost btn-sm" onClick=${(e) => { e.stopPropagation(); openDrawer(r); }}>
                      ${Icons.expand}
                    </button>
                  </td>
                </tr>
                ${expandedId === r.id ? html` + "`" + `
                  <tr class="expand-row">
                    <td colspan="7">
                      <div class="expand-content">
                        <div class="detail-grid">
                          <span class="detail-label">Method</span><span class="detail-value">${r.method}</span>
                          <span class="detail-label">Path</span><span class="detail-value mono">${r.path}</span>
                          <span class="detail-label">Timestamp</span><span class="detail-value">${fmtTimeFull(r.timestamp)}</span>
                        </div>
                        ${r.request_body ? html` + "`" + `
                          <div style="margin-top:12px">
                            <div class="detail-label" style="margin-bottom:6px">Request Body</div>
                            <div class="code-block">${r.request_body}</div>
                          </div>
                        ` + "`" + ` : null}
                        ${r.response_body ? html` + "`" + `
                          <div style="margin-top:12px">
                            <div class="detail-label" style="margin-bottom:6px">Response Body</div>
                            <div class="code-block">${r.response_body}</div>
                          </div>
                        ` + "`" + ` : null}
                      </div>
                    </td>
                  </tr>
                ` + "`" + ` : null}
              ` + "`" + `)}
            </tbody>
          </table>
        </div>
      </div>

      <${RequestDrawer} entry=${drawer} onClose=${() => setDrawer(null)} />
    </div>
  ` + "`" + `;
}

// ─── Request Drawer ───

function RequestDrawer({ entry, onClose }) {
  const [tab, setTab] = useState('overview');
  const isOpen = !!entry;

  useEffect(() => { if (isOpen) setTab('overview'); }, [isOpen]);

  return html` + "`" + `
    <div class="drawer-overlay ${isOpen ? 'open' : ''}" onClick=${(e) => { if (e.target === e.currentTarget) onClose(); }}>
    </div>
    <div class="drawer ${isOpen ? 'open' : ''}">
      <div class="drawer-header">
        <h3>Request Detail</h3>
        <button class="drawer-close" onClick=${onClose}>${Icons.close}</button>
      </div>
      <div class="drawer-tabs">
        ${['overview','request','response','timing'].map(t => html` + "`" + `
          <div class="drawer-tab ${tab === t ? 'active' : ''}" onClick=${() => setTab(t)}>${t.charAt(0).toUpperCase() + t.slice(1)}</div>
        ` + "`" + `)}
      </div>
      <div class="drawer-body">
        ${entry && tab === 'overview' ? html` + "`" + `
          <div class="detail-grid">
            <span class="detail-label">Service</span><span class="detail-value">${entry.service}</span>
            <span class="detail-label">Action</span><span class="detail-value mono">${entry.action}</span>
            <span class="detail-label">Status</span><span class="detail-value"><span class="status-code ${statusClass(entry.status_code)}">${entry.status_code}</span></span>
            <span class="detail-label">Latency</span><span class="detail-value mono">${fmtLatency(entry.latency_ns)}</span>
            <span class="detail-label">Timestamp</span><span class="detail-value">${fmtTimeFull(entry.timestamp)}</span>
            <span class="detail-label">Caller</span><span class="detail-value mono">${entry.caller_id || '-'}</span>
            <span class="detail-label">Method</span><span class="detail-value">${entry.method}</span>
            <span class="detail-label">Path</span><span class="detail-value mono">${entry.path}</span>
          </div>
          <div style="margin-top:20px">
            <button class="btn btn-ghost btn-sm" onClick=${() => {
              apiFetch('/api/requests/' + entry.id + '/replay', { method: 'POST' });
            }}>Replay Request</button>
          </div>
        ` + "`" + ` : null}
        ${entry && tab === 'request' ? html` + "`" + `
          <div>
            ${entry.request_headers ? html` + "`" + `
              <div style="margin-bottom:16px">
                <div class="detail-label" style="margin-bottom:8px">Headers</div>
                <div class="code-block">${Object.entries(entry.request_headers || {}).map(([k,v]) => k + ': ' + v).join('\n')}</div>
              </div>
            ` + "`" + ` : null}
            <div class="detail-label" style="margin-bottom:8px">Body</div>
            <div class="code-block">${entry.request_body || '(empty)'}</div>
          </div>
        ` + "`" + ` : null}
        ${entry && tab === 'response' ? html` + "`" + `
          <div>
            <div class="detail-label" style="margin-bottom:8px">Body</div>
            <div class="code-block">${entry.response_body || '(empty)'}</div>
          </div>
        ` + "`" + ` : null}
        ${entry && tab === 'timing' ? html` + "`" + `
          <div class="detail-grid">
            <span class="detail-label">Total</span><span class="detail-value mono">${fmtLatency(entry.latency_ns)}</span>
            <span class="detail-label">Start</span><span class="detail-value">${fmtTimeFull(entry.timestamp)}</span>
          </div>
        ` + "`" + ` : null}
      </div>
    </div>
  ` + "`" + `;
}

// ─── Resource Explorer Page ───

function ResourceExplorerPage({ services }) {
  const [selectedSvc, setSelectedSvc] = useState(null);
  const [resources, setResources] = useState(null);
  const [loading, setLoading] = useState(false);

  const svcList = (services || []).map(s => s.name).sort();

  const loadResources = (svc) => {
    setSelectedSvc(svc);
    setLoading(true);
    setResources(null);
    apiFetch('/api/services/' + svc)
      .then(data => {
        setResources(data);
        setLoading(false);
      })
      .catch(() => {
        setResources({ error: 'Failed to load' });
        setLoading(false);
      });
  };

  return html` + "`" + `
    <div>
      <h1 class="page-title">Resource Explorer</h1>
      <div style="display:flex;gap:0;border-radius:var(--radius-lg);overflow:hidden;box-shadow:var(--shadow-sm);background:#fff;min-height:500px">
        <div class="explorer-sidebar">
          ${svcList.map(s => html` + "`" + `
            <div class="nav-item ${selectedSvc === s ? 'active' : ''}" onClick=${() => loadResources(s)}>
              ${s}
            </div>
          ` + "`" + `)}
        </div>
        <div class="explorer-main">
          ${!selectedSvc ? html` + "`" + `<div class="empty-state">Select a service from the sidebar</div>` + "`" + ` : null}
          ${loading ? html` + "`" + `<div class="empty-state">Loading...</div>` + "`" + ` : null}
          ${selectedSvc && !loading && resources ? html` + "`" + `
            <div>
              <h2 style="font-size:16px;font-weight:600;margin-bottom:16px;color:var(--neutral-900)">${selectedSvc}</h2>
              <div class="detail-grid">
                <span class="detail-label">Name</span><span class="detail-value">${resources.name || selectedSvc}</span>
                <span class="detail-label">Actions</span><span class="detail-value">${resources.action_count || 0}</span>
                <span class="detail-label">Status</span><span class="detail-value">${resources.healthy ? 'Healthy' : 'Degraded'}</span>
              </div>
            </div>
          ` + "`" + ` : null}
        </div>
      </div>
    </div>
  ` + "`" + `;
}

// ─── Lambda Logs Page ───

function LambdaLogsPage({ subscribe }) {
  const [logs, setLogs] = useState([]);
  const [fnFilter, setFnFilter] = useState('');
  const [functions, setFunctions] = useState(new Set());
  const [expandedIdx, setExpandedIdx] = useState(null);

  useEffect(() => {
    apiFetch('/api/lambda/logs?limit=100').then(entries => {
      setLogs(entries || []);
      const fns = new Set();
      (entries || []).forEach(e => { if (e.functionName) fns.add(e.functionName); });
      setFunctions(fns);
    }).catch(() => {});
  }, []);

  useEffect(() => {
    return subscribe((event) => {
      if (event.type === 'lambda_log') {
        setLogs(prev => {
          const next = [event.data, ...prev];
          if (next.length > 500) next.length = 500;
          return next;
        });
        if (event.data.functionName) {
          setFunctions(prev => {
            const next = new Set(prev);
            next.add(event.data.functionName);
            return next;
          });
        }
      }
    });
  }, [subscribe]);

  const filtered = useMemo(() => {
    if (!fnFilter) return logs;
    return logs.filter(l => l.functionName === fnFilter);
  }, [logs, fnFilter]);

  return html` + "`" + `
    <div>
      <h1 class="page-title">Lambda Logs</h1>
      <div class="filter-bar section-gap">
        <select class="filter-select" id="lambda-filter" value=${fnFilter} onChange=${(e) => setFnFilter(e.target.value)}>
          <option value="">All functions</option>
          ${[...functions].map(f => html` + "`" + `<option value=${f}>${f}</option>` + "`" + `)}
        </select>
      </div>
      <div class="card">
        <div class="card-body lambda-table-wrap">
          <table id="lambda-table">
            <thead>
              <tr><th>Time</th><th>Function</th><th>Request ID</th><th>Duration</th><th>Message</th></tr>
            </thead>
            <tbody id="lambda-tbody">
              ${filtered.length === 0 ? html` + "`" + `<tr><td colspan="5" class="empty-state">No Lambda logs yet</td></tr>` + "`" + ` : null}
              ${filtered.map((entry, i) => html` + "`" + `
                <tr class="${entry.stream === 'stderr' ? 'log-stderr' : ''} fade-in"
                    onClick=${() => setExpandedIdx(expandedIdx === i ? null : i)}>
                  <td class="mono">${fmtTime(entry.timestamp)}</td>
                  <td>${entry.functionName || '-'}</td>
                  <td class="mono" title=${entry.requestId || ''}>${(entry.requestId || '-').substring(0, 12)}</td>
                  <td class="mono">-</td>
                  <td class="mono" style="max-width:400px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${entry.message || ''}</td>
                </tr>
                ${expandedIdx === i ? html` + "`" + `
                  <tr class="expand-row">
                    <td colspan="5">
                      <div class="expand-content">
                        <div class="code-block">${entry.message || '(no output)'}</div>
                      </div>
                    </td>
                  </tr>
                ` + "`" + ` : null}
              ` + "`" + `)}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  ` + "`" + `;
}

// ─── IAM Debugger Page ───

function IAMDebuggerPage() {
  const [principal, setPrincipal] = useState('');
  const [action, setAction] = useState('');
  const [resource, setResource] = useState('*');
  const [result, setResult] = useState(null);
  const [history, setHistory] = useState([]);

  const evaluate = () => {
    apiFetch('/api/iam/evaluate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ principal, action, resource }),
    }).then(r => {
      setResult(r);
      setHistory(prev => [{ principal, action, resource, ...r, time: new Date().toISOString() }, ...prev].slice(0, 20));
    }).catch(() => {
      setResult({ decision: 'ERROR', reason: 'Failed to evaluate' });
    });
  };

  return html` + "`" + `
    <div>
      <h1 class="page-title">IAM Debugger</h1>
      <div style="display:flex;gap:32px;flex-wrap:wrap">
        <div style="flex:1;min-width:300px">
          <div class="card">
            <div class="card-header"><h2>Evaluate Policy</h2></div>
            <div class="card-body-padded">
              <div class="iam-form">
                <div class="form-group">
                  <label class="form-label">Principal</label>
                  <input class="form-input" placeholder="e.g. admin-user" value=${principal} onInput=${(e) => setPrincipal(e.target.value)} />
                </div>
                <div class="form-group">
                  <label class="form-label">Action</label>
                  <input class="form-input" placeholder="e.g. s3:GetObject" value=${action} onInput=${(e) => setAction(e.target.value)} />
                </div>
                <div class="form-group">
                  <label class="form-label">Resource ARN</label>
                  <input class="form-input" placeholder="e.g. arn:aws:s3:::my-bucket/*" value=${resource} onInput=${(e) => setResource(e.target.value)} />
                </div>
                <button class="btn btn-primary" onClick=${evaluate}>Evaluate</button>
              </div>

              ${result ? html` + "`" + `
                <div class="iam-result ${result.decision === 'ALLOW' ? 'allow' : 'deny'}">
                  <div class="iam-result-decision">${result.decision}</div>
                  <div style="color:var(--neutral-600)">${result.reason}</div>
                  ${result.matched_statement ? html` + "`" + `
                    <div style="margin-top:12px">
                      <div class="detail-label" style="margin-bottom:6px">Matched Statement</div>
                      <div class="code-block">${JSON.stringify(result.matched_statement, null, 2)}</div>
                    </div>
                  ` + "`" + ` : null}
                </div>
              ` + "`" + ` : null}
            </div>
          </div>
        </div>

        <div style="flex:1;min-width:300px">
          <div class="card">
            <div class="card-header"><h2>Evaluation History</h2></div>
            <div class="card-body">
              ${history.length === 0 ? html` + "`" + `<div class="empty-state">No evaluations yet</div>` + "`" + ` : html` + "`" + `
                <table>
                  <thead><tr><th>Time</th><th>Action</th><th>Decision</th></tr></thead>
                  <tbody>
                    ${history.map(h => html` + "`" + `
                      <tr>
                        <td class="mono">${fmtTime(h.time)}</td>
                        <td class="mono">${h.action}</td>
                        <td><span class="status-code ${h.decision === 'ALLOW' ? 'status-2xx' : 'status-5xx'}">${h.decision}</span></td>
                      </tr>
                    ` + "`" + `)}
                  </tbody>
                </table>
              ` + "`" + `}
            </div>
          </div>
        </div>
      </div>
    </div>
  ` + "`" + `;
}

// ─── SES Mailbox Page ───

function SESMailboxPage() {
  const [emails, setEmails] = useState([]);
  const [selected, setSelected] = useState(null);

  useEffect(() => {
    apiFetch('/api/ses/emails').then(setEmails).catch(() => {});
  }, []);

  const openEmail = (email) => {
    if (email.message_id) {
      apiFetch('/api/ses/emails/' + email.message_id).then(setSelected).catch(() => setSelected(email));
    } else {
      setSelected(email);
    }
  };

  return html` + "`" + `
    <div>
      <h1 class="page-title">SES Mailbox</h1>
      <div style="display:flex;gap:24px">
        <div class="card" style="flex:1">
          <div class="card-header"><h2>Captured Emails (${emails.length})</h2></div>
          <div class="card-body">
            ${emails.length === 0 ? html` + "`" + `<div class="empty-state">No captured emails yet</div>` + "`" + ` : null}
            ${emails.map(e => html` + "`" + `
              <div class="email-preview" onClick=${() => openEmail(e)}>
                <div class="email-meta">
                  <span class="email-from">${e.source || 'Unknown'}</span>
                  <span class="email-date">${fmtTimeFull(e.timestamp)}</span>
                </div>
                <div class="email-subject">${e.subject || '(no subject)'}</div>
                <div class="email-to">To: ${(e.to || []).join(', ')}</div>
              </div>
            ` + "`" + `)}
          </div>
        </div>

        ${selected ? html` + "`" + `
          <div class="card" style="flex:1.5">
            <div class="card-header">
              <h2>${selected.Subject || selected.subject || '(no subject)'}</h2>
              <button class="btn btn-ghost btn-sm" onClick=${() => setSelected(null)}>${Icons.close}</button>
            </div>
            <div class="card-body-padded">
              <div class="detail-grid" style="margin-bottom:16px">
                <span class="detail-label">From</span><span class="detail-value">${selected.Source || selected.source}</span>
                <span class="detail-label">To</span><span class="detail-value">${(selected.ToAddresses || selected.to || []).join(', ')}</span>
                <span class="detail-label">Date</span><span class="detail-value">${fmtTimeFull(selected.Timestamp || selected.timestamp)}</span>
              </div>
              ${(selected.HtmlBody || selected.html_body)
                ? html` + "`" + `<div style="border:1px solid var(--neutral-200);border-radius:var(--radius-md);padding:16px;background:var(--neutral-50)">
                    <div class="detail-label" style="margin-bottom:8px">HTML Email Body</div>
                    <pre class="code-block">${selected.HtmlBody || selected.html_body}</pre>
                  </div>` + "`" + `
                : html` + "`" + `<pre class="code-block">${selected.TextBody || selected.text_body || '(empty body)'}</pre>` + "`" + `}
            </div>
          </div>
        ` + "`" + ` : null}
      </div>
    </div>
  ` + "`" + `;
}

// ─── Topology Page ───

function TopologyPage({ navigate }) {
  const [topo, setTopo] = useState(null);

  useEffect(() => {
    apiFetch('/api/topology').then(setTopo).catch(() => {});
  }, []);

  if (!topo) return html` + "`" + `<div class="empty-state">Loading topology...</div>` + "`" + `;

  // Layout nodes in a grid-like arrangement
  const nodes = topo.nodes || [];
  const edges = topo.edges || [];
  const cols = Math.ceil(Math.sqrt(nodes.length));
  const cellW = 160;
  const cellH = 80;
  const padX = 40;
  const padY = 40;
  const width = cols * cellW + padX * 2;
  const rows = Math.ceil(nodes.length / cols);
  const height = rows * cellH + padY * 2;

  const nodePositions = {};
  nodes.forEach((n, i) => {
    const col = i %% cols;
    const row = Math.floor(i / cols);
    nodePositions[n.id] = {
      x: padX + col * cellW + cellW / 2,
      y: padY + row * cellH + cellH / 2,
    };
  });

  return html` + "`" + `
    <div>
      <h1 class="page-title">Service Topology</h1>
      <div class="topology-container" style="width:${width}px;height:${height}px;margin:0 auto">
        <svg class="topo-svg" viewBox="0 0 ${width} ${height}">
          ${edges.map(e => {
            const s = nodePositions[e.source];
            const t = nodePositions[e.target];
            if (!s || !t) return null;
            const midX = (s.x + t.x) / 2;
            const midY = (s.y + t.y) / 2;
            return html` + "`" + `
              <line class="topo-edge" x1=${s.x} y1=${s.y} x2=${t.x} y2=${t.y} />
              <text class="topo-edge-label" x=${midX} y=${midY - 6} text-anchor="middle">${e.label}</text>
            ` + "`" + `;
          })}
        </svg>
        ${nodes.map((n, i) => {
          const pos = nodePositions[n.id];
          return html` + "`" + `
            <div
              class="topo-node"
              style="left:${pos.x - 50}px;top:${pos.y - 18}px"
              onClick=${() => navigate('#/services/' + n.id)}
            >${n.name}</div>
          ` + "`" + `;
        })}
      </div>
    </div>
  ` + "`" + `;
}

// ─── App ───

function App() {
  const { route, navigate } = useRouter();
  const { connected, subscribe } = useSSE();
  const [services, setServices] = useState([]);
  const [stats, setStats] = useState({});
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [emailCount, setEmailCount] = useState(0);

  // Initial data load
  useEffect(() => {
    apiFetch('/api/services').then(setServices).catch(() => {});
    apiFetch('/api/stats').then(setStats).catch(() => {});
    apiFetch('/api/ses/emails').then(e => setEmailCount((e || []).length)).catch(() => {});
  }, []);

  // SSE updates for service counts
  useEffect(() => {
    return subscribe((event) => {
      if (event.type === 'request' && event.data && event.data.service) {
        setStats(prev => ({
          ...prev,
          [event.data.service]: (prev[event.data.service] || 0) + 1,
        }));
      }
    });
  }, [subscribe]);

  // Cmd+K shortcut
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

  // Periodic health refresh
  useEffect(() => {
    const interval = setInterval(() => {
      apiFetch('/api/services').then(setServices).catch(() => {});
      apiFetch('/api/stats').then(setStats).catch(() => {});
    }, 15000);
    return () => clearInterval(interval);
  }, []);

  // Route matching
  let content;
  if (route.startsWith('#/services/') && route !== '#/services') {
    const name = route.replace('#/services/', '');
    content = html` + "`" + `<${ServiceDetailPage} name=${name} navigate=${navigate} stats=${stats} />` + "`" + `;
  } else if (route === '#/requests') {
    content = html` + "`" + `<${RequestLogPage} subscribe=${subscribe} />` + "`" + `;
  } else if (route === '#/resources') {
    content = html` + "`" + `<${ResourceExplorerPage} services=${services} />` + "`" + `;
  } else if (route === '#/lambda') {
    content = html` + "`" + `<${LambdaLogsPage} subscribe=${subscribe} />` + "`" + `;
  } else if (route === '#/iam') {
    content = html` + "`" + `<${IAMDebuggerPage} />` + "`" + `;
  } else if (route === '#/mail') {
    content = html` + "`" + `<${SESMailboxPage} />` + "`" + `;
  } else if (route === '#/topology') {
    content = html` + "`" + `<${TopologyPage} navigate=${navigate} />` + "`" + `;
  } else {
    content = html` + "`" + `<${ServicesPage} navigate=${navigate} services=${services} stats=${stats} />` + "`" + `;
  }

  return html` + "`" + `
    <div id="app" style="display:flex;flex-direction:column;height:100vh">
      <div class="header">
        <div class="header-brand">
          ${Icons.cloud}
          cloud<span>mock</span>
        </div>
        <div class="header-right">
          <div class="sse-badge" id="sse-badge">
            <div class="sse-dot ${connected ? 'connected' : 'disconnected'}" id="sse-dot"></div>
            <span id="sse-text">${connected ? 'Connected' : 'Disconnected'}</span>
          </div>
          <div id="health-badge" class="sse-badge">
            <div class="sse-dot ${services.length > 0 ? 'connected' : ''}" id="health-dot"></div>
            <span id="health-text">${services.every(s => s.healthy) ? 'Healthy' : 'Degraded'}</span>
          </div>
          <div class="cmd-k-hint" onClick=${() => setPaletteOpen(true)}>
            ${Icons.search}
            <kbd>${navigator.platform.includes('Mac') ? 'Cmd+K' : 'Ctrl+K'}</kbd>
          </div>
        </div>
      </div>
      <div class="body-layout">
        <${Sidebar} route=${route} navigate=${navigate} emailCount=${emailCount} />
        <main class="main-content">
          ${content}
        </main>
      </div>
      <${CommandPalette} open=${paletteOpen} onClose=${() => setPaletteOpen(false)} navigate=${navigate} services=${services} />
    </div>
  ` + "`" + `;
}

render(html` + "`" + `<${App} />` + "`" + `, document.getElementById('app'));
</script>
</body>
</html>`
