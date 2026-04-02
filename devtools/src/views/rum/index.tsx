import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import './rum.css';

interface WebVitalsOverview {
  lcp: VitalStat;
  fid: VitalStat;
  cls: VitalStat;
  ttfb: VitalStat;
  fcp: VitalStat;
  score: number;
  total_events: number;
}

interface VitalStat {
  p75: number;
  good: number;
  needs_improvement: number;
  poor: number;
  rating: string;
}

interface PagePerformance {
  route: string;
  avg_load_ms: number;
  p75_lcp_ms: number;
  avg_cls: number;
  sample_count: number;
}

interface ErrorGroup {
  fingerprint: string;
  message: string;
  count: number;
  affected_sessions: number;
  last_seen: string;
  sample_stack: string;
}

interface SessionSummary {
  session_id: string;
  start: string;
  end: string;
  page_count: number;
  error_count: number;
  duration_sec: number;
}

type Tab = 'vitals' | 'pages' | 'errors' | 'sessions';

function vitalColor(rating: string): string {
  if (rating === 'good') return '#22c55e';
  if (rating === 'needs-improvement') return '#fbbf24';
  return '#ef4444';
}

function scoreColor(score: number): string {
  if (score >= 75) return '#22c55e';
  if (score >= 50) return '#fbbf24';
  return '#ef4444';
}

export function RUMView() {
  const [tab, setTab] = useState<Tab>('vitals');
  const [minutes, setMinutes] = useState(15);
  const [vitals, setVitals] = useState<WebVitalsOverview | null>(null);
  const [pages, setPages] = useState<PagePerformance[]>([]);
  const [errors, setErrors] = useState<ErrorGroup[]>([]);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [expandedError, setExpandedError] = useState<string | null>(null);

  const loadVitals = useCallback(async () => {
    try { setVitals(await api<WebVitalsOverview>(`/api/rum/vitals?minutes=${minutes}`)); } catch { setVitals(null); }
  }, [minutes]);

  const loadPages = useCallback(async () => {
    try { setPages(await api<PagePerformance[]>(`/api/rum/pages?minutes=${minutes}`) || []); } catch { setPages([]); }
  }, [minutes]);

  const loadErrors = useCallback(async () => {
    try { setErrors(await api<ErrorGroup[]>(`/api/rum/errors?minutes=${minutes}&limit=50`) || []); } catch { setErrors([]); }
  }, [minutes]);

  const loadSessions = useCallback(async () => {
    try { setSessions(await api<SessionSummary[]>(`/api/rum/sessions?minutes=${minutes}&limit=50`) || []); } catch { setSessions([]); }
  }, [minutes]);

  useEffect(() => {
    loadVitals();
    loadPages();
    loadErrors();
    loadSessions();
    const interval = setInterval(() => { loadVitals(); loadPages(); loadErrors(); loadSessions(); }, 10000);
    return () => clearInterval(interval);
  }, [minutes]);

  return (
    <div class="rum-view">
      <div class="rum-header">
        <div class="rum-tabs">
          {(['vitals', 'pages', 'errors', 'sessions'] as Tab[]).map((t) => (
            <button key={t} class={`rum-tab ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>
              {t === 'vitals' ? 'Web Vitals' : t === 'pages' ? 'Pages' : t === 'errors' ? 'Errors' : 'Sessions'}
            </button>
          ))}
        </div>
        <div class="rum-time-selector">
          {[15, 30, 60, 120].map((m) => (
            <button key={m} class={`rum-time-btn ${minutes === m ? 'active' : ''}`} onClick={() => setMinutes(m)}>
              {m < 60 ? `${m}m` : `${m / 60}h`}
            </button>
          ))}
        </div>
      </div>

      <div class="rum-content">
        {tab === 'vitals' && (
          <div class="rum-vitals">
            {!vitals || vitals.total_events === 0 ? (
              <div class="rum-empty">
                <div class="rum-empty-title">No RUM Data</div>
                <p>Install the @cloudmock/rum SDK in your frontend app to capture web vitals.</p>
                <pre class="rum-code-block">{`import { init } from '@cloudmock/rum';\n\ninit({\n  endpoint: 'http://localhost:4599/api/rum/events',\n  appName: 'my-app',\n});`}</pre>
              </div>
            ) : (
              <>
                <div class="rum-score-card">
                  <div class="rum-score-value" style={{ color: scoreColor(vitals.score) }}>{Math.round(vitals.score)}</div>
                  <div class="rum-score-label">User Experience Score</div>
                  <div class="rum-score-events">{vitals.total_events} events</div>
                </div>
                <div class="rum-vitals-grid">
                  {(['lcp', 'fid', 'cls', 'ttfb', 'fcp'] as const).map((key) => {
                    const v = vitals[key];
                    if (!v) return null;
                    const unit = key === 'cls' ? '' : 'ms';
                    const label = key.toUpperCase();
                    return (
                      <div key={key} class="rum-vital-card">
                        <div class="rum-vital-label">{label}</div>
                        <div class="rum-vital-value" style={{ color: vitalColor(v.rating) }}>
                          {key === 'cls' ? v.p75.toFixed(3) : Math.round(v.p75)}{unit}
                        </div>
                        <div class="rum-vital-rating">{v.rating}</div>
                        <div class="rum-vital-breakdown">
                          <span class="rum-vital-good">{v.good} good</span>
                          <span class="rum-vital-warn">{v.needs_improvement} warn</span>
                          <span class="rum-vital-poor">{v.poor} poor</span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}
          </div>
        )}

        {tab === 'pages' && (
          <div class="rum-pages">
            {pages.length === 0 ? (
              <div class="rum-empty">No page performance data in the last {minutes} minutes.</div>
            ) : (
              <table class="rum-table">
                <thead><tr><th>Route</th><th>Avg Load</th><th>P75 LCP</th><th>Avg CLS</th><th>Samples</th></tr></thead>
                <tbody>
                  {pages.map((p) => (
                    <tr key={p.route}>
                      <td class="rum-route">{p.route}</td>
                      <td>{Math.round(p.avg_load_ms)}ms</td>
                      <td>{Math.round(p.p75_lcp_ms)}ms</td>
                      <td>{p.avg_cls.toFixed(3)}</td>
                      <td>{p.sample_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {tab === 'errors' && (
          <div class="rum-errors">
            {errors.length === 0 ? (
              <div class="rum-empty">No JS errors captured in the last {minutes} minutes.</div>
            ) : (
              <div class="rum-error-list">
                {errors.map((e) => (
                  <div key={e.fingerprint} class="rum-error-item">
                    <div class="rum-error-header" onClick={() => setExpandedError(expandedError === e.fingerprint ? null : e.fingerprint)}>
                      <span class="rum-error-count">{e.count}</span>
                      <span class="rum-error-message">{e.message}</span>
                      <span class="rum-error-sessions">{e.affected_sessions} sessions</span>
                    </div>
                    {expandedError === e.fingerprint && e.sample_stack && (
                      <pre class="rum-error-stack">{e.sample_stack}</pre>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'sessions' && (
          <div class="rum-sessions">
            {sessions.length === 0 ? (
              <div class="rum-empty">No sessions in the last {minutes} minutes.</div>
            ) : (
              <table class="rum-table">
                <thead><tr><th>Session</th><th>Pages</th><th>Errors</th><th>Duration</th></tr></thead>
                <tbody>
                  {sessions.map((s) => (
                    <tr key={s.session_id}>
                      <td class="rum-session-id">{s.session_id.slice(0, 8)}...</td>
                      <td>{s.page_count}</td>
                      <td class={s.error_count > 0 ? 'rum-has-errors' : ''}>{s.error_count}</td>
                      <td>{Math.round(s.duration_sec)}s</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
