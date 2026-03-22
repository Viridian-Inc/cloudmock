import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import { api } from '../api';
import { fmtTime, fmtDuration, copyToClipboard } from '../utils';
import { RefreshIcon, PlayIcon, CopyIcon } from '../components/Icons';

interface DebugAssistantProps {
  showToast: (msg: string) => void;
}

// ── Error suggestions by code ──────────────────────────────────────────────

function getSuggestion(errorCode: string, iamMode: string): string {
  if (errorCode.includes('ResourceNotFoundException') || errorCode.includes('TableNotFoundException')) {
    return "Table/resource doesn't exist. Did you run the seed script?";
  }
  if (errorCode.includes('ValidationException')) {
    return 'Invalid request parameters. Check the request body.';
  }
  if (errorCode.includes('AccessDeniedException') || errorCode.includes('UnauthorizedException')) {
    return `IAM policy doesn't allow this action. Check IAM mode (current: ${iamMode || 'none'})`;
  }
  if (errorCode.includes('ConditionalCheckFailedException')) {
    return 'DynamoDB condition expression failed. Item may already exist.';
  }
  if (errorCode.includes('ProvisionedThroughputExceededException')) {
    return 'Throughput exceeded. In mock mode this is unusual — check for runaway loops.';
  }
  if (errorCode.includes('ItemCollectionSizeLimitExceededException')) {
    return 'Item collection too large. Clean up old test data.';
  }
  if (errorCode.includes('RequestLimitExceeded') || errorCode.includes('ThrottlingException')) {
    return 'Request throttled. Reduce request rate or increase limits in config.';
  }
  return 'Check request parameters and service configuration.';
}

// ── Extract error code from response body ─────────────────────────────────

function extractErrorCode(req: any): string {
  const body = req.response_body || req.body || {};
  if (typeof body === 'object') {
    return body.__type || body.code || body.Code || body.errorCode || '';
  }
  if (typeof body === 'string') {
    const m = body.match(/"__type"\s*:\s*"([^"]+)"/);
    if (m) return m[1];
    const m2 = body.match(/"code"\s*:\s*"([^"]+)"/);
    if (m2) return m2[1];
  }
  return '';
}

// ── Relative time helper ───────────────────────────────────────────────────

function timeAgo(ts: string | undefined | null): string {
  if (!ts) return '';
  const diff = (Date.now() - new Date(ts).getTime()) / 1000;
  if (diff < 5) return 'just now';
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  return `${Math.floor(diff / 3600)}h ago`;
}

// ── Sub-components ─────────────────────────────────────────────────────────

function SectionHeader({ icon, title, count, countColor }: { icon: string; title: string; count?: number; countColor?: string }) {
  return (
    <div class="debug-section-header">
      <span class="debug-section-icon">{icon}</span>
      <span class="debug-section-title">{title}</span>
      {count !== undefined && count > 0 && (
        <span class="debug-count-badge" style={countColor ? `background:${countColor}20;color:${countColor}` : ''}>
          {count}
        </span>
      )}
    </div>
  );
}

function AccentCard({ accent, children }: { accent: 'red' | 'yellow' | 'green' | 'blue'; children: preact.ComponentChildren }) {
  return (
    <div class={`debug-accent-card debug-accent-${accent}`}>
      {children}
    </div>
  );
}

// ── Main component ─────────────────────────────────────────────────────────

export function DebugAssistantPage({ showToast }: DebugAssistantProps) {
  const [requests, setRequests] = useState<any[]>([]);
  const [stats, setStats] = useState<any>({});
  const [health, setHealth] = useState<any>(null);
  const [config, setConfig] = useState<any>(null);
  const [services, setServices] = useState<any[]>([]);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [loading, setLoading] = useState(true);

  // Replay state
  const [replayReq, setReplayReq] = useState<any>(null);
  const [replayBody, setReplayBody] = useState('');
  const [replayResult, setReplayResult] = useState<any>(null);
  const [replayLoading, setReplayLoading] = useState(false);

  // Action states
  const [resetting, setResetting] = useState(false);
  const [creatingTables, setCreatingTables] = useState(false);
  const [togglingIam, setTogglingIam] = useState(false);

  const load = useCallback(async () => {
    try {
      const [reqs, st, hl, svc] = await Promise.all([
        api('/api/requests?limit=200').catch(() => []),
        api('/api/stats').catch(() => ({})),
        api('/api/health').catch(() => null),
        api('/api/services').catch(() => []),
      ]);
      // Try config endpoint — may not exist in all builds
      const cfg = await api('/api/config').catch(() => null);

      setRequests(Array.isArray(reqs) ? reqs : []);
      setStats(st || {});
      setHealth(hl);
      setConfig(cfg);
      setServices(Array.isArray(svc) ? svc : []);
      setLastUpdated(new Date());
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    const iv = setInterval(load, 10000);
    return () => clearInterval(iv);
  }, [load]);

  // ── Derived: Error Analysis ──────────────────────────────────────────────

  const iamMode = useMemo(() => {
    return config?.iam_mode || health?.iam_mode || 'none';
  }, [config, health]);

  const persistenceEnabled = useMemo(() => {
    return config?.persistence === true || health?.persistence === true;
  }, [config, health]);

  // Errors in last 5 minutes
  const recentErrors = useMemo(() => {
    const cutoff = Date.now() - 5 * 60 * 1000;
    return requests.filter((r: any) => {
      const status = r.status || 200;
      if (status < 400) return false;
      const ts = new Date(r.timestamp || r.time || 0).getTime();
      return ts >= cutoff;
    });
  }, [requests]);

  // Group errors by service+action
  const errorGroups = useMemo(() => {
    const groups: Record<string, { service: string; action: string; count: number; lastTs: string; errorCode: string; requests: any[] }> = {};
    for (const r of recentErrors) {
      const key = `${r.service}:${r.action}`;
      if (!groups[key]) {
        groups[key] = { service: r.service, action: r.action, count: 0, lastTs: r.timestamp || r.time || '', errorCode: '', requests: [] };
      }
      groups[key].count++;
      groups[key].requests.push(r);
      // Keep latest timestamp
      if (r.timestamp > groups[key].lastTs) {
        groups[key].lastTs = r.timestamp || r.time || '';
      }
      // Extract error code from first occurrence
      if (!groups[key].errorCode) {
        groups[key].errorCode = extractErrorCode(r);
      }
    }
    return Object.values(groups).sort((a, b) => b.count - a.count);
  }, [recentErrors]);

  // ── Derived: Performance Insights ───────────────────────────────────────

  const perfInsights = useMemo(() => {
    if (!stats?.services) return { slowest: [], busiest: [], highErrorRate: [], trend: 'stable' as const };

    const entries = Object.entries(stats.services).map(([name, s]: [string, any]) => ({
      name,
      total: s.total || 0,
      errors: s.errors || 0,
      avgLatency: s.avg_latency_ms || s.avg_latency || 0,
      rpm: s.rpm || 0,
    }));

    const slowest = entries.filter(e => e.avgLatency > 10).sort((a, b) => b.avgLatency - a.avgLatency).slice(0, 5);
    const busiest = [...entries].sort((a, b) => b.total - a.total).filter(e => e.total > 0).slice(0, 5);
    const highErrorRate = entries
      .filter(e => e.total > 0 && (e.errors / e.total) > 0.05)
      .map(e => ({ ...e, errorRate: (e.errors / e.total) * 100 }))
      .sort((a, b) => b.errorRate - a.errorRate);

    // Simple trend: compare rpm to recent error count
    const totalRpm = entries.reduce((s, e) => s + e.rpm, 0);
    const trend = totalRpm > 10 ? 'increasing' : totalRpm > 0 ? 'stable' : 'decreasing';

    return { slowest, busiest, highErrorRate, trend };
  }, [stats]);

  // ── Derived: Config Warnings ─────────────────────────────────────────────

  const configWarnings = useMemo(() => {
    const warnings: { level: 'warn' | 'ok'; message: string }[] = [];

    if (iamMode === 'none' || iamMode === '') {
      warnings.push({ level: 'warn', message: 'IAM mode is "none" — all requests bypass authorization checks' });
    } else {
      warnings.push({ level: 'ok', message: `IAM mode is "${iamMode}" — authorization is active` });
    }

    if (!persistenceEnabled) {
      warnings.push({ level: 'warn', message: 'Persistence disabled — data will be lost on restart' });
    } else {
      warnings.push({ level: 'ok', message: 'Persistence enabled — data survives restarts' });
    }

    // DynamoDB table count
    const ddbStat = stats?.services?.dynamodb;
    if (ddbStat) {
      const tableCount = ddbStat.resources || 0;
      if (tableCount > 0) {
        warnings.push({ level: 'ok', message: `${tableCount} DynamoDB table${tableCount !== 1 ? 's' : ''} exist` });
      } else {
        warnings.push({ level: 'warn', message: 'No DynamoDB tables found — run seed script?' });
      }
    }

    // Lambda functions
    const lambdaStat = stats?.services?.lambda;
    if (lambdaStat) {
      const fnCount = lambdaStat.resources || 0;
      if (fnCount > 0) {
        warnings.push({ level: 'ok', message: `${fnCount} Lambda function${fnCount !== 1 ? 's' : ''} deployed` });
      } else {
        warnings.push({ level: 'warn', message: 'No Lambda functions deployed' });
      }
    }

    return warnings;
  }, [iamMode, persistenceEnabled, stats]);

  const warnCount = useMemo(() => configWarnings.filter(w => w.level === 'warn').length, [configWarnings]);

  // ── Derived: Failed requests for replay ──────────────────────────────────

  const replayable = useMemo(() => {
    return requests.filter((r: any) => (r.status || 200) >= 400).slice(0, 20);
  }, [requests]);

  // ── Actions ────────────────────────────────────────────────────────────

  async function handleReset() {
    setResetting(true);
    try {
      await api('/api/reset', { method: 'POST' });
      showToast('Services reset successfully');
      await load();
    } catch {
      showToast('Reset failed');
    } finally {
      setResetting(false);
    }
  }

  async function handleToggleIam() {
    setTogglingIam(true);
    const newMode = iamMode === 'none' ? 'strict' : 'none';
    try {
      await api('/api/config', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ iam_mode: newMode }),
      });
      showToast(`IAM mode set to "${newMode}"`);
      await load();
    } catch {
      showToast('Failed to toggle IAM mode');
    } finally {
      setTogglingIam(false);
    }
  }

  async function handleFixTables() {
    setCreatingTables(true);
    try {
      await api('/api/admin/create-missing-tables', { method: 'POST' });
      showToast('Missing tables created');
      await load();
    } catch {
      showToast('Fix tables not supported or failed');
    } finally {
      setCreatingTables(false);
    }
  }

  function handleExportReport() {
    const report = {
      generated: new Date().toISOString(),
      summary: {
        recentErrors: recentErrors.length,
        errorGroups: errorGroups.length,
        configWarnings: warnCount,
        iamMode,
        persistenceEnabled,
      },
      errorGroups,
      performanceInsights: perfInsights,
      configWarnings,
      recentRequests: requests.slice(0, 50),
      stats,
      health,
      config,
    };
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `cloudmock-debug-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
    showToast('Debug report exported');
  }

  function openReplay(req: any) {
    setReplayReq(req);
    setReplayBody(JSON.stringify(req.request_body || req.body || {}, null, 2));
    setReplayResult(null);
  }

  async function submitReplay() {
    if (!replayReq) return;
    setReplayLoading(true);
    try {
      const result = await api(`/api/requests/${replayReq.id}/replay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body: JSON.parse(replayBody) }),
      }).catch(() =>
        api(`/api/requests/${replayReq.id}/replay`, { method: 'POST' })
      );
      setReplayResult(result);
      showToast('Request replayed');
    } catch {
      showToast('Replay failed');
    } finally {
      setReplayLoading(false);
    }
  }

  // ── Render ─────────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div class="debug-loading">
        <div class="debug-spinner" />
        <span>Analyzing cloudmock state...</span>
      </div>
    );
  }

  return (
    <div class="debug-page">
      {/* Page header */}
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Debug Assistant</h1>
          <p class="page-desc">
            Rule-based analysis of cloudmock errors, performance, and configuration
            {lastUpdated && (
              <span class="debug-last-updated"> — updated {timeAgo(lastUpdated.toISOString())}</span>
            )}
          </p>
        </div>
        <button class="btn btn-ghost btn-sm" onClick={load}>
          <RefreshIcon /> Refresh
        </button>
      </div>

      <div class="debug-grid">

        {/* ── Error Analysis ── */}
        <div class="card debug-card">
          <div class="card-header">
            <SectionHeader
              icon="⚠"
              title="Error Analysis"
              count={recentErrors.length}
              countColor="var(--error)"
            />
          </div>
          <div class="card-body">
            {recentErrors.length === 0 ? (
              <div class="debug-ok-banner">
                <span>✓</span> No errors in the last 5 minutes
              </div>
            ) : (
              <>
                <div class="debug-summary-line">
                  <span class="debug-error-count">{recentErrors.length}</span> error{recentErrors.length !== 1 ? 's' : ''} in last 5 minutes
                </div>
                <div class="debug-group-list">
                  {errorGroups.map(g => (
                    <AccentCard accent="red" key={`${g.service}:${g.action}`}>
                      <div class="debug-group-head">
                        <span class="debug-group-key">
                          <span class="debug-svc-name">{g.service}</span>
                          <span class="debug-colon">:</span>
                          <span class="debug-action-name">{g.action}</span>
                        </span>
                        <span class="debug-group-meta">
                          {g.count} occurrence{g.count !== 1 ? 's' : ''} &bull; Last: {timeAgo(g.lastTs)}
                        </span>
                      </div>
                      {g.errorCode && (
                        <div class="debug-error-code">{g.errorCode}</div>
                      )}
                      <div class="debug-suggestion">
                        <span class="debug-bulb">💡</span>
                        {getSuggestion(g.errorCode, iamMode)}
                      </div>
                      <div class="debug-group-actions">
                        <button
                          class="btn btn-ghost btn-sm"
                          onClick={() => {
                            location.hash = '/requests';
                            setTimeout(() => {
                              const el = document.getElementById('service-filter') as HTMLSelectElement;
                              if (el) { el.value = g.service; el.dispatchEvent(new Event('change')); }
                            }, 200);
                          }}
                        >
                          View Requests
                        </button>
                        {g.service === 'dynamodb' && g.errorCode.includes('ResourceNotFoundException') && (
                          <button class="btn btn-ghost btn-sm" onClick={() => (location.hash = '/dynamodb')}>
                            Open DynamoDB
                          </button>
                        )}
                      </div>
                    </AccentCard>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>

        {/* ── Performance Insights ── */}
        <div class="card debug-card">
          <div class="card-header">
            <SectionHeader icon="⚡" title="Performance Insights" />
          </div>
          <div class="card-body">
            {perfInsights.slowest.length === 0 && perfInsights.busiest.length === 0 ? (
              <div class="debug-ok-banner">
                <span>✓</span> No performance data yet — make some requests
              </div>
            ) : (
              <div class="debug-perf-sections">
                {perfInsights.slowest.length > 0 && (
                  <AccentCard accent="yellow">
                    <div class="debug-perf-label">Slowest services (avg latency &gt; 10ms)</div>
                    <div class="debug-perf-list">
                      {perfInsights.slowest.map(s => (
                        <div class="debug-perf-row" key={s.name}>
                          <span class="debug-perf-svc">{s.name}</span>
                          <span class="debug-perf-val">{fmtDuration(s.avgLatency)}</span>
                        </div>
                      ))}
                    </div>
                  </AccentCard>
                )}

                {perfInsights.busiest.length > 0 && (
                  <AccentCard accent="blue">
                    <div class="debug-perf-label">Most-called services (potential bottleneck)</div>
                    <div class="debug-perf-list">
                      {perfInsights.busiest.map(s => (
                        <div class="debug-perf-row" key={s.name}>
                          <span class="debug-perf-svc">{s.name}</span>
                          <span class="debug-perf-val">{s.total.toLocaleString()} calls</span>
                        </div>
                      ))}
                    </div>
                  </AccentCard>
                )}

                {perfInsights.highErrorRate.length > 0 && (
                  <AccentCard accent="red">
                    <div class="debug-perf-label">High error rate (&gt;5%)</div>
                    <div class="debug-perf-list">
                      {perfInsights.highErrorRate.map((s: any) => (
                        <div class="debug-perf-row" key={s.name}>
                          <span class="debug-perf-svc">{s.name}</span>
                          <span class="debug-perf-val debug-perf-error">{s.errorRate.toFixed(1)}%</span>
                        </div>
                      ))}
                    </div>
                  </AccentCard>
                )}

                <div class="debug-trend-row">
                  <span class="debug-trend-label">Request rate trend</span>
                  <span class={`debug-trend-val debug-trend-${perfInsights.trend}`}>
                    {perfInsights.trend === 'increasing' ? '↑ Increasing' :
                      perfInsights.trend === 'stable' ? '→ Stable' : '↓ Decreasing'}
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* ── Configuration Warnings ── */}
        <div class="card debug-card">
          <div class="card-header">
            <SectionHeader
              icon="⚙"
              title="Configuration"
              count={warnCount}
              countColor="var(--warning)"
            />
          </div>
          <div class="card-body">
            <div class="debug-config-list">
              {configWarnings.map((w, i) => (
                <div class={`debug-config-item debug-config-${w.level}`} key={i}>
                  <span class="debug-config-icon">{w.level === 'ok' ? '✓' : '⚠'}</span>
                  <span class="debug-config-msg">{w.message}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* ── Quick Actions ── */}
        <div class="card debug-card">
          <div class="card-header">
            <SectionHeader icon="🔧" title="Quick Actions" />
          </div>
          <div class="card-body">
            <div class="debug-actions-grid">
              <button
                class="btn btn-danger"
                onClick={handleReset}
                disabled={resetting}
              >
                {resetting ? 'Resetting…' : 'Reset & Reseed'}
              </button>

              <button
                class="btn btn-ghost"
                onClick={handleFixTables}
                disabled={creatingTables}
              >
                {creatingTables ? 'Creating…' : 'Fix Missing Tables'}
              </button>

              <button
                class="btn btn-ghost"
                onClick={handleToggleIam}
                disabled={togglingIam}
              >
                {togglingIam ? 'Updating…' : iamMode === 'none' ? 'Enable IAM' : 'Disable IAM'}
              </button>

              <button class="btn btn-secondary" onClick={handleExportReport}>
                <CopyIcon /> Export Debug Report
              </button>
            </div>
          </div>
        </div>

        {/* ── Request Replay ── */}
        <div class="card debug-card debug-card-full">
          <div class="card-header">
            <SectionHeader icon="▶" title="Request Replay" count={replayable.length} />
          </div>
          <div class="card-body">
            {replayable.length === 0 ? (
              <div class="debug-ok-banner">
                <span>✓</span> No failed requests to replay
              </div>
            ) : (
              <div class="debug-replay-layout">
                <div class="debug-replay-list">
                  {replayable.map((r: any) => (
                    <div
                      class={`debug-replay-item ${replayReq?.id === r.id ? 'active' : ''}`}
                      key={r.id}
                      onClick={() => openReplay(r)}
                    >
                      <div class="debug-replay-item-top">
                        <span class="debug-svc-name">{r.service}</span>
                        <span class="debug-replay-status" style={`color:var(--error)`}>{r.status}</span>
                      </div>
                      <div class="debug-replay-item-action">{r.action}</div>
                      <div class="debug-replay-item-time">{fmtTime(r.timestamp || r.time)}</div>
                    </div>
                  ))}
                </div>

                {replayReq ? (
                  <div class="debug-replay-detail">
                    <div class="debug-replay-detail-header">
                      <span class="debug-group-key">
                        <span class="debug-svc-name">{replayReq.service}</span>
                        <span class="debug-colon">:</span>
                        <span class="debug-action-name">{replayReq.action}</span>
                      </span>
                      <span class="debug-group-meta">
                        Original status: <span style="color:var(--error)">{replayReq.status}</span>
                      </span>
                    </div>

                    <div class="debug-replay-section-label">Request Body (edit before replaying)</div>
                    <textarea
                      class="debug-replay-textarea"
                      value={replayBody}
                      onInput={(e) => setReplayBody((e.target as HTMLTextAreaElement).value)}
                      rows={8}
                      spellcheck={false}
                    />

                    <div class="debug-replay-btn-row">
                      <button
                        class="btn btn-primary btn-sm"
                        onClick={submitReplay}
                        disabled={replayLoading}
                      >
                        <PlayIcon /> {replayLoading ? 'Replaying…' : 'Replay Request'}
                      </button>
                      <button
                        class="btn btn-ghost btn-sm"
                        onClick={() => { copyToClipboard(replayBody); showToast('Copied'); }}
                      >
                        <CopyIcon /> Copy Body
                      </button>
                    </div>

                    {replayResult && (
                      <div class="debug-replay-result">
                        <div class="debug-replay-section-label">
                          Replay Response
                          {replayResult.status && (
                            <span class={`status-pill ${replayResult.status < 400 ? 'status-2xx' : 'status-4xx'} ml-2`}>
                              {replayResult.status}
                            </span>
                          )}
                        </div>
                        <pre class="json-view">{JSON.stringify(replayResult, null, 2)}</pre>
                      </div>
                    )}
                  </div>
                ) : (
                  <div class="debug-replay-empty">
                    <span>Select a failed request from the list to replay it</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

      </div>
    </div>
  );
}

// ── Export error count for nav badge ───────────────────────────────────────

export function useDebugErrorCount(): number {
  const [count, setCount] = useState(0);

  useEffect(() => {
    function check() {
      api('/api/requests?limit=200').then((reqs: any[]) => {
        if (!Array.isArray(reqs)) return;
        const cutoff = Date.now() - 5 * 60 * 1000;
        const errs = reqs.filter((r: any) => {
          const status = r.status || 200;
          if (status < 400) return false;
          const ts = new Date(r.timestamp || r.time || 0).getTime();
          return ts >= cutoff;
        });
        setCount(errs.length);
      }).catch(() => {});
    }
    check();
    const iv = setInterval(check, 10000);
    return () => clearInterval(iv);
  }, []);

  return count;
}
