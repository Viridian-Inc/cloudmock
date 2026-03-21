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
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    font-size: 14px;
    background: #f0f2f5;
    color: #1a1a2e;
    min-height: 100vh;
  }

  /* Header */
  header {
    background: #16213e;
    color: #fff;
    padding: 0 24px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: sticky;
    top: 0;
    z-index: 100;
    box-shadow: 0 2px 8px rgba(0,0,0,.4);
  }
  header .brand {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: #e2e8f0;
  }
  header .brand span { color: #63b3ed; }

  #health-badge {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 500;
  }
  #health-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%%;
    background: #a0aec0;
    transition: background .3s;
  }
  #health-dot.healthy  { background: #48bb78; box-shadow: 0 0 6px #48bb7880; }
  #health-dot.degraded { background: #ed8936; box-shadow: 0 0 6px #ed893680; }
  #health-dot.error    { background: #fc8181; box-shadow: 0 0 6px #fc818180; }

  /* Layout */
  main {
    max-width: 1200px;
    margin: 0 auto;
    padding: 24px 20px;
    display: grid;
    gap: 24px;
  }

  /* Cards */
  .card {
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 1px 4px rgba(0,0,0,.08);
    overflow: hidden;
  }
  .card-header {
    padding: 14px 20px;
    border-bottom: 1px solid #e8ecf0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: #fafbfc;
  }
  .card-header h2 {
    font-size: 14px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: .6px;
    color: #4a5568;
  }
  .card-body { padding: 0; }

  /* Tables */
  table { width: 100%%; border-collapse: collapse; }
  th {
    text-align: left;
    padding: 10px 16px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: .5px;
    color: #718096;
    background: #f7f8fa;
    border-bottom: 1px solid #e8ecf0;
    white-space: nowrap;
  }
  td {
    padding: 10px 16px;
    border-bottom: 1px solid #f0f2f5;
    color: #2d3748;
  }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #f7f9fc; }

  /* Status dots */
  .dot {
    display: inline-block;
    width: 8px; height: 8px;
    border-radius: 50%%;
    margin-right: 6px;
    vertical-align: middle;
  }
  .dot-green  { background: #48bb78; }
  .dot-yellow { background: #ed8936; }
  .dot-red    { background: #fc8181; }

  /* Status codes */
  .code {
    display: inline-block;
    padding: 2px 7px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    font-family: ui-monospace, "SF Mono", "Cascadia Code", monospace;
  }
  .code-2xx { background: #c6f6d5; color: #276749; }
  .code-4xx { background: #fefcbf; color: #744210; }
  .code-5xx { background: #fed7d7; color: #822727; }
  .code-0   { background: #e2e8f0; color: #4a5568; }

  /* Service filter */
  #service-filter {
    font-size: 13px;
    padding: 5px 10px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    background: #fff;
    color: #2d3748;
    cursor: pointer;
  }

  /* Empty / loading states */
  .empty-row td {
    text-align: center;
    color: #a0aec0;
    padding: 24px;
    font-style: italic;
  }

  .mono {
    font-family: ui-monospace, "SF Mono", "Cascadia Code", monospace;
    font-size: 12px;
  }

  .refresh-note {
    font-size: 11px;
    color: #a0aec0;
  }

  .count-badge {
    display: inline-block;
    background: #ebf8ff;
    color: #2b6cb0;
    padding: 1px 8px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
  }
</style>
</head>
<body>

<header>
  <div class="brand">cloud<span>mock</span></div>
  <div id="health-badge">
    <div id="health-dot"></div>
    <span id="health-text">connecting...</span>
  </div>
</header>

<main>

  <div class="card">
    <div class="card-header">
      <h2>Services</h2>
      <span class="refresh-note">auto-refresh every 5 s</span>
    </div>
    <div class="card-body">
      <table id="services-table">
        <thead>
          <tr>
            <th>Service</th>
            <th>Status</th>
            <th>Requests</th>
          </tr>
        </thead>
        <tbody id="services-tbody">
          <tr class="empty-row"><td colspan="3">Loading...</td></tr>
        </tbody>
      </table>
    </div>
  </div>

  <div class="card">
    <div class="card-header">
      <h2>Request Log</h2>
      <select id="service-filter">
        <option value="">All services</option>
      </select>
    </div>
    <div class="card-body">
      <table id="requests-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Service</th>
            <th>Action</th>
            <th>Status</th>
            <th>Latency</th>
          </tr>
        </thead>
        <tbody id="requests-tbody">
          <tr class="empty-row"><td colspan="5">Loading...</td></tr>
        </tbody>
      </table>
    </div>
  </div>

</main>

<script>
'use strict';

const ADMIN = %q;
const REFRESH_MS = 5000;

function fmtTime(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  const pad = n => String(n).padStart(2, '0');
  return pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
}

function fmtLatency(ns) {
  if (!ns || ns === 0) return '-';
  if (ns < 1000000) return (ns / 1000).toFixed(0) + ' us';
  if (ns < 1000000000) return (ns / 1000000).toFixed(1) + ' ms';
  return (ns / 1000000000).toFixed(2) + ' s';
}

function statusClass(code) {
  if (!code) return 'code-0';
  if (code >= 200 && code < 300) return 'code-2xx';
  if (code >= 400 && code < 500) return 'code-4xx';
  if (code >= 500) return 'code-5xx';
  return 'code-0';
}

function escHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

async function apiFetch(path) {
  const resp = await fetch(ADMIN + path);
  if (!resp.ok) throw new Error('HTTP ' + resp.status);
  return resp.json();
}

async function refreshHealth() {
  const dot  = document.getElementById('health-dot');
  const text = document.getElementById('health-text');
  try {
    const data = await apiFetch('/api/health');
    const status = (data.status || '').toLowerCase();
    dot.className = '';
    if (status === 'healthy') {
      dot.classList.add('healthy');
      text.textContent = 'Healthy';
    } else if (status === 'degraded') {
      dot.classList.add('degraded');
      text.textContent = 'Degraded';
    } else {
      dot.classList.add('error');
      text.textContent = status || 'Unknown';
    }
  } catch (_) {
    dot.className = 'error';
    text.textContent = 'Unreachable';
  }
}

let knownServices = [];

async function refreshServices() {
  const tbody = document.getElementById('services-tbody');
  try {
    const [services, stats] = await Promise.all([
      apiFetch('/api/services'),
      apiFetch('/api/stats'),
    ]);

    knownServices = (services || []).map(s => s.name);
    rebuildServiceFilter(knownServices);

    if (!services || services.length === 0) {
      tbody.textContent = '';
      const tr = document.createElement('tr');
      tr.className = 'empty-row';
      const td = document.createElement('td');
      td.setAttribute('colspan', '3');
      td.textContent = 'No services registered';
      tr.appendChild(td);
      tbody.appendChild(tr);
      return;
    }

    const fragment = document.createDocumentFragment();
    services.forEach(svc => {
      const healthy = svc.healthy !== false;
      const count = (stats && stats[svc.name]) ? stats[svc.name] : 0;

      const tr = document.createElement('tr');

      const tdName = document.createElement('td');
      tdName.textContent = svc.name;

      const tdStatus = document.createElement('td');
      const dotEl = document.createElement('span');
      dotEl.className = 'dot ' + (healthy ? 'dot-green' : 'dot-yellow');
      tdStatus.appendChild(dotEl);
      tdStatus.appendChild(document.createTextNode(healthy ? 'healthy' : 'degraded'));

      const tdCount = document.createElement('td');
      const badge = document.createElement('span');
      badge.className = 'count-badge';
      badge.textContent = count;
      tdCount.appendChild(badge);

      tr.appendChild(tdName);
      tr.appendChild(tdStatus);
      tr.appendChild(tdCount);
      fragment.appendChild(tr);
    });

    tbody.textContent = '';
    tbody.appendChild(fragment);
  } catch (err) {
    tbody.textContent = '';
    const tr = document.createElement('tr');
    tr.className = 'empty-row';
    const td = document.createElement('td');
    td.setAttribute('colspan', '3');
    td.textContent = 'Failed to load services: ' + err.message;
    tr.appendChild(td);
    tbody.appendChild(tr);
  }
}

function rebuildServiceFilter(names) {
  const sel = document.getElementById('service-filter');
  const current = sel.value;
  while (sel.options.length > 1) sel.remove(1);
  names.forEach(name => {
    const opt = document.createElement('option');
    opt.value = name;
    opt.textContent = name;
    sel.appendChild(opt);
  });
  if (names.includes(current)) sel.value = current;
}

async function refreshRequests() {
  const tbody  = document.getElementById('requests-tbody');
  const filter = document.getElementById('service-filter').value;
  const qs     = '?limit=50' + (filter ? '&service=' + encodeURIComponent(filter) : '');
  try {
    const entries = await apiFetch('/api/requests' + qs);

    if (!entries || entries.length === 0) {
      tbody.textContent = '';
      const tr = document.createElement('tr');
      tr.className = 'empty-row';
      const td = document.createElement('td');
      td.setAttribute('colspan', '5');
      td.textContent = 'No requests recorded yet';
      tr.appendChild(td);
      tbody.appendChild(tr);
      return;
    }

    const fragment = document.createDocumentFragment();
    entries.forEach(e => {
      const sc = e.status_code || 0;
      const tr = document.createElement('tr');

      const tdTime = document.createElement('td');
      tdTime.className = 'mono';
      tdTime.textContent = fmtTime(e.timestamp);

      const tdSvc = document.createElement('td');
      tdSvc.textContent = e.service || '-';

      const tdAction = document.createElement('td');
      tdAction.className = 'mono';
      tdAction.textContent = e.action || '-';

      const tdStatus = document.createElement('td');
      const codeEl = document.createElement('span');
      codeEl.className = 'code ' + statusClass(sc);
      codeEl.textContent = sc || '?';
      tdStatus.appendChild(codeEl);

      const tdLatency = document.createElement('td');
      tdLatency.className = 'mono';
      tdLatency.textContent = fmtLatency(e.latency_ns);

      tr.appendChild(tdTime);
      tr.appendChild(tdSvc);
      tr.appendChild(tdAction);
      tr.appendChild(tdStatus);
      tr.appendChild(tdLatency);
      fragment.appendChild(tr);
    });

    tbody.textContent = '';
    tbody.appendChild(fragment);
  } catch (err) {
    tbody.textContent = '';
    const tr = document.createElement('tr');
    tr.className = 'empty-row';
    const td = document.createElement('td');
    td.setAttribute('colspan', '5');
    td.textContent = 'Failed to load requests: ' + err.message;
    tr.appendChild(td);
    tbody.appendChild(tr);
  }
}

async function refreshAll() {
  await Promise.allSettled([refreshHealth(), refreshServices(), refreshRequests()]);
}

document.getElementById('service-filter').addEventListener('change', refreshRequests);

refreshAll();
setInterval(refreshAll, REFRESH_MS);
</script>
</body>
</html>`
