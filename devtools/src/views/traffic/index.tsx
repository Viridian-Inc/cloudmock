import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import { api, getAdminBase } from '../../lib/api';
import './traffic.css';

interface Recording {
  id: string;
  name: string;
  status: string;
  duration_sec: number;
  entry_count: number;
  entries?: CapturedEntry[];
  started_at: string;
  stopped_at?: string;
}

interface CapturedEntry {
  id: string;
  service: string;
  action: string;
  method: string;
  path: string;
  status_code: number;
  latency_ms: number;
  offset_ms: number;
}

interface ReplayRunRaw {
  id: string;
  recording_id: string;
  recording_name?: string;
  status: string;
  speed?: number;
  speed_multiplier?: number;
  total_count?: number;
  total_requests?: number;
  replayed_count?: number;
  sent_requests?: number;
  match_count?: number;
  mismatch_count?: number;
  error_count: number;
  stats?: { p50_ms: number; p95_ms: number; p99_ms: number; avg_ms: number; min_ms: number; max_ms: number };
  latency_stats?: { p50_ms: number; p95_ms: number; p99_ms: number; avg_ms: number; min_ms: number; max_ms: number };
  results?: ReplayResult[];
  started_at: string;
  finished_at?: string | null;
  completed_at?: string | null;
}

// Normalize API response to consistent field names
interface ReplayRun {
  id: string;
  recording_id: string;
  recording_name: string;
  status: string;
  speed_multiplier: number;
  total_requests: number;
  sent_requests: number;
  error_count: number;
  latency_stats: { p50_ms: number; p95_ms: number; p99_ms: number; avg_ms: number; min_ms: number; max_ms: number };
  results?: ReplayResult[];
  started_at: string;
  completed_at: string | null;
}

function normalizeRun(raw: ReplayRunRaw): ReplayRun {
  return {
    id: raw.id,
    recording_id: raw.recording_id,
    recording_name: raw.recording_name || raw.recording_id,
    status: raw.status,
    speed_multiplier: raw.speed_multiplier || raw.speed || 1,
    total_requests: raw.total_requests || raw.total_count || 0,
    sent_requests: raw.sent_requests || raw.replayed_count || 0,
    error_count: raw.error_count || 0,
    latency_stats: raw.latency_stats || raw.stats || { p50_ms: 0, p95_ms: 0, p99_ms: 0, avg_ms: 0, min_ms: 0, max_ms: 0 },
    results: raw.results,
    started_at: raw.started_at,
    completed_at: raw.completed_at || raw.finished_at || null,
  };
}

interface ReplayResult {
  entry_id?: string;
  original_status: number;
  replay_status: number;
  original_latency_ms: number;
  replay_latency_ms: number;
  latency_delta_ms?: number;
  match: boolean;
  path?: string;
  service?: string;
}

type Tab = 'recorder' | 'replay' | 'runs' | 'compare';

function statusColor(code: number): string {
  if (code >= 500) return 'var(--error)';
  if (code >= 400) return 'var(--warning)';
  if (code >= 200 && code < 300) return 'var(--success-bright, #42FF8B)';
  return 'var(--text-tertiary)';
}

function formatMs(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(2)}s`;
  return `${Math.round(ms)}ms`;
}

function formatTime(iso: string): string {
  try { return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }); }
  catch { return iso; }
}

export function TrafficView() {
  const [tab, setTab] = useState<Tab>('recorder');
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [runs, setRuns] = useState<ReplayRun[]>([]);
  const [recordingDuration, setRecordingDuration] = useState(30);
  const [isRecording, setIsRecording] = useState(false);
  const [recordingProgress, setRecordingProgress] = useState(0);
  const [selectedRecording, setSelectedRecording] = useState<string | null>(null);
  const [replaySpeed, setReplaySpeed] = useState(1);
  const [activeReplay, setActiveReplay] = useState<ReplayRun | null>(null);
  const [expandedRun, setExpandedRun] = useState<string | null>(null);
  const [compareA, setCompareA] = useState<string | null>(null);
  const [compareB, setCompareB] = useState<string | null>(null);
  const progressRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadRecordings = useCallback(async () => {
    try {
      const data = await api<Recording[]>('/api/traffic/recordings');
      setRecordings(Array.isArray(data) ? data : []);
    } catch { setRecordings([]); }
  }, []);

  const loadRuns = useCallback(async () => {
    try {
      const data = await api<ReplayRunRaw[]>('/api/traffic/runs');
      setRuns(Array.isArray(data) ? data.map(normalizeRun) : []);
    } catch { setRuns([]); }
  }, []);

  useEffect(() => { loadRecordings(); loadRuns(); }, []);

  // Auto-generate recording name
  const genName = () => `recording-${new Date().toISOString().slice(5, 19).replace(/[T:]/g, '-')}`;

  const startRecording = async () => {
    setIsRecording(true);
    setRecordingProgress(0);
    try {
      const base = getAdminBase();
      const name = genName();
      await fetch(`${base}/api/traffic/record`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, duration_sec: recordingDuration }),
      });

      // Progress ticker
      const startTime = Date.now();
      progressRef.current = setInterval(() => {
        const elapsed = (Date.now() - startTime) / 1000;
        const pct = Math.min(100, (elapsed / recordingDuration) * 100);
        setRecordingProgress(pct);
        if (pct >= 100) {
          clearInterval(progressRef.current!);
          progressRef.current = null;
          setIsRecording(false);
          setRecordingProgress(0);
          loadRecordings();
        }
      }, 200);
    } catch { setIsRecording(false); }
  };

  const stopRecording = async () => {
    if (progressRef.current) { clearInterval(progressRef.current); progressRef.current = null; }
    const base = getAdminBase();
    await fetch(`${base}/api/traffic/record/stop`, { method: 'POST' }).catch(() => {});
    setIsRecording(false);
    setRecordingProgress(0);
    setTimeout(loadRecordings, 500);
  };

  const [replayError, setReplayError] = useState<string | null>(null);

  const startReplay = async () => {
    if (!selectedRecording) return;
    setReplayError(null);
    try {
      const base = getAdminBase();
      const res = await fetch(`${base}/api/traffic/replay`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recording_id: selectedRecording, speed: replaySpeed }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        setReplayError(err.error || `Failed: ${res.status}`);
        return;
      }
      if (res.ok) {
        const run = normalizeRun(await res.json());
        setActiveReplay(run);
        // Stay on replay tab to show live progress — don't switch to history
        const interval = setInterval(async () => {
          try {
            const updated = normalizeRun(await api<ReplayRunRaw>(`/api/traffic/replay/${run.id}`));
            setActiveReplay(updated);
            if (updated.status === 'completed' || updated.status === 'cancelled' || updated.status === 'failed') {
              clearInterval(interval);
              loadRuns();
            }
          } catch { clearInterval(interval); }
        }, 500);
      }
    } catch {}
  };

  const deleteRecording = async (id: string) => {
    const base = getAdminBase();
    await fetch(`${base}/api/traffic/recordings/${id}`, { method: 'DELETE' }).catch(() => {});
    loadRecordings();
  };

  return (
    <div class="traffic-view">
      <div class="traffic-tabs">
        {(['recorder', 'replay', 'runs', 'compare'] as Tab[]).map((t) => (
          <button key={t} class={`traffic-tab ${tab === t ? 'active' : ''}`} onClick={() => setTab(t)}>
            {t === 'recorder' ? 'Record' : t === 'replay' ? 'Replay' : t === 'runs' ? 'History' : 'Compare'}
            {t === 'recorder' && isRecording && <span class="traffic-tab-dot" />}
            {t === 'runs' && runs.length > 0 && <span class="traffic-tab-count">{runs.length}</span>}
          </button>
        ))}
      </div>

      <div class="traffic-content">
        {tab === 'recorder' && (
          <div class="traffic-recorder">
            {/* Record controls */}
            <div class="traffic-record-card">
              {isRecording ? (
                <>
                  <div class="traffic-recording-header">
                    <div class="traffic-recording-pulse" />
                    <span class="traffic-recording-label">Recording...</span>
                    <span class="traffic-recording-timer">{Math.round(recordingProgress)}%</span>
                  </div>
                  <div class="traffic-progress-bar">
                    <div class="traffic-progress-fill traffic-progress-recording" style={{ width: `${recordingProgress}%` }} />
                  </div>
                  <button class="btn" onClick={stopRecording} style={{ marginTop: '12px' }}>Stop Recording</button>
                </>
              ) : (
                <>
                  <div class="traffic-record-row">
                    <span class="traffic-record-label">Duration</span>
                    <div class="traffic-duration-selector">
                      {[10, 30, 60, 300].map((s) => (
                        <button
                          key={s}
                          class={`traffic-duration-btn ${recordingDuration === s ? 'active' : ''}`}
                          onClick={() => setRecordingDuration(s)}
                        >
                          {s < 60 ? `${s}s` : `${s / 60}m`}
                        </button>
                      ))}
                    </div>
                  </div>
                  <button class="btn btn-primary traffic-record-btn" onClick={startRecording}>
                    Start Recording
                  </button>
                  <div class="traffic-record-hint">
                    Captures all requests passing through CloudMock during the recording window.
                  </div>
                </>
              )}
            </div>

            {/* Saved recordings */}
            <div class="traffic-section-title">
              Recordings
              {recordings.length > 0 && <span class="traffic-section-count">{recordings.length}</span>}
            </div>
            {recordings.length === 0 ? (
              <div class="traffic-empty">No recordings yet. Hit "Start Recording" above, then generate some traffic.</div>
            ) : (
              <div class="traffic-recordings-list">
                {recordings.map((r) => (
                  <div key={r.id} class="traffic-recording-card">
                    <div class="traffic-recording-card-header">
                      <span class="traffic-recording-name">{r.name}</span>
                      <span class={`traffic-status traffic-status-${r.status}`}>{r.status}</span>
                    </div>
                    <div class="traffic-recording-card-stats">
                      <span>{r.entry_count} requests</span>
                      <span>{Math.round(r.duration_sec)}s duration</span>
                      <span>{formatTime(r.started_at)}</span>
                    </div>
                    <div class="traffic-recording-card-actions">
                      <button class="btn btn-sm" onClick={() => { setSelectedRecording(r.id); setTab('replay'); }}>Replay</button>
                      <button class="btn btn-ghost btn-sm" onClick={() => deleteRecording(r.id)}>Delete</button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'replay' && (
          <div class="traffic-replay">
            <div class="traffic-section-title">Replay a Recording</div>
            {recordings.filter(r => r.entry_count > 0).length === 0 ? (
              <div class="traffic-replay-card">
                <div class="traffic-empty" style={{ padding: '16px 0' }}>
                  No recordings with captured traffic.
                  <br /><br />
                  <button class="btn" onClick={() => setTab('recorder')}>Go to Record tab</button>
                </div>
              </div>
            ) : (
              <div class="traffic-replay-card">
                <div class="traffic-replay-row">
                  <span class="traffic-record-label">Recording</span>
                  <select class="traffic-select" value={selectedRecording || ''} onChange={(e) => setSelectedRecording((e.target as HTMLSelectElement).value || null)}>
                    <option value="">Select...</option>
                    {recordings.filter(r => r.entry_count > 0).map((r) => (
                      <option key={r.id} value={r.id}>{r.name} ({r.entry_count} reqs)</option>
                    ))}
                  </select>
                </div>
                <div class="traffic-replay-row">
                  <span class="traffic-record-label">Speed</span>
                  <div class="traffic-speed-selector">
                    {[1, 2, 5, 10].map((s) => (
                      <button key={s} class={`traffic-speed-btn ${replaySpeed === s ? 'active' : ''}`} onClick={() => setReplaySpeed(s)}>
                        {s}x
                      </button>
                    ))}
                  </div>
                </div>
                <button class="btn btn-primary" onClick={startReplay} disabled={!selectedRecording} style={{ marginTop: '12px' }}>
                  Start Replay
                </button>
                {!selectedRecording && (
                  <div class="traffic-record-hint">Select a recording above to replay it.</div>
                )}
                {replayError && (
                  <div style={{ color: 'var(--error)', fontSize: '12px', marginTop: '8px' }}>{replayError}</div>
                )}
              </div>
            )}

            {activeReplay && (
              <div class="traffic-replay-live">
                <div class="traffic-section-title">
                  Live Replay
                  <span class={`traffic-status traffic-status-${activeReplay.status}`}>{activeReplay.status}</span>
                </div>
                <div class="traffic-progress-bar">
                  <div class="traffic-progress-fill" style={{ width: `${activeReplay.total_requests ? (activeReplay.sent_requests / activeReplay.total_requests) * 100 : 0}%` }} />
                </div>
                <div class="traffic-replay-stats-grid">
                  <div class="traffic-stat">
                    <div class="traffic-stat-value">{activeReplay.sent_requests}/{activeReplay.total_requests}</div>
                    <div class="traffic-stat-label">Sent</div>
                  </div>
                  <div class="traffic-stat">
                    <div class="traffic-stat-value" style={{ color: activeReplay.error_count > 0 ? 'var(--error)' : 'var(--text-primary)' }}>{activeReplay.error_count}</div>
                    <div class="traffic-stat-label">Errors</div>
                  </div>
                  <div class="traffic-stat">
                    <div class="traffic-stat-value">{formatMs(activeReplay.latency_stats?.p50_ms || 0)}</div>
                    <div class="traffic-stat-label">P50</div>
                  </div>
                  <div class="traffic-stat">
                    <div class="traffic-stat-value">{formatMs(activeReplay.latency_stats?.p95_ms || 0)}</div>
                    <div class="traffic-stat-label">P95</div>
                  </div>
                  <div class="traffic-stat">
                    <div class="traffic-stat-value">{formatMs(activeReplay.latency_stats?.p99_ms || 0)}</div>
                    <div class="traffic-stat-label">P99</div>
                  </div>
                  <div class="traffic-stat">
                    <div class="traffic-stat-value">{activeReplay.speed_multiplier}x</div>
                    <div class="traffic-stat-label">Speed</div>
                  </div>
                </div>
                {/* Live results table */}
                {activeReplay.results && activeReplay.results.length > 0 && (
                  <div class="traffic-results-table-wrap">
                    <table class="traffic-results-table">
                      <thead>
                        <tr><th>Service</th><th>Path</th><th>Original</th><th>Replay</th><th>Latency</th><th>Match</th></tr>
                      </thead>
                      <tbody>
                        {activeReplay.results.slice(-20).reverse().map((res, i) => (
                          <tr key={i}>
                            <td>{res.service}</td>
                            <td class="traffic-result-path">{res.path}</td>
                            <td style={{ color: statusColor(res.original_status) }}>{res.original_status}</td>
                            <td style={{ color: statusColor(res.replay_status) }}>{res.replay_status}</td>
                            <td class="mono">{formatMs(res.replay_latency_ms)}</td>
                            <td>{res.match ? <span style={{ color: 'var(--success-bright)' }}>match</span> : <span style={{ color: 'var(--error)' }}>changed</span>}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {tab === 'runs' && (
          <div class="traffic-runs">
            <div class="traffic-section-title">
              Replay History
              {runs.length > 0 && <span class="traffic-section-count">{runs.length}</span>}
            </div>
            {runs.length === 0 ? (
              <div class="traffic-empty">No replay runs yet. Record traffic, then replay it.</div>
            ) : (
              <div class="traffic-runs-list">
                {runs.map((r) => (
                  <div key={r.id} class="traffic-run-card" onClick={() => setExpandedRun(expandedRun === r.id ? null : r.id)}>
                    <div class="traffic-run-header">
                      <span class="traffic-run-name">{r.recording_name}</span>
                      <span class="traffic-run-speed">{r.speed_multiplier}x</span>
                      <span class={`traffic-status traffic-status-${r.status}`}>{r.status}</span>
                    </div>
                    <div class="traffic-run-stats">
                      <span>{r.sent_requests}/{r.total_requests} sent</span>
                      <span class={r.error_count > 0 ? 'traffic-error' : ''}>{r.error_count} errors</span>
                      <span>p50: {formatMs(r.latency_stats?.p50_ms || 0)}</span>
                      <span>p99: {formatMs(r.latency_stats?.p99_ms || 0)}</span>
                      <span>{formatTime(r.started_at)}</span>
                    </div>
                    {expandedRun === r.id && r.latency_stats && (
                      <div class="traffic-run-detail">
                        <div class="traffic-replay-stats-grid">
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.min_ms || 0)}</div><div class="traffic-stat-label">Min</div></div>
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.p50_ms || 0)}</div><div class="traffic-stat-label">P50</div></div>
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.p95_ms || 0)}</div><div class="traffic-stat-label">P95</div></div>
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.p99_ms || 0)}</div><div class="traffic-stat-label">P99</div></div>
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.max_ms || 0)}</div><div class="traffic-stat-label">Max</div></div>
                          <div class="traffic-stat"><div class="traffic-stat-value">{formatMs(r.latency_stats.avg_ms || 0)}</div><div class="traffic-stat-label">Avg</div></div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'compare' && (
          <div class="traffic-compare">
            <div class="traffic-section-title">Compare Replay Runs</div>
            <div class="traffic-compare-selectors">
              <div class="traffic-compare-selector">
                <label class="traffic-compare-label">Run A</label>
                <select class="traffic-select" value={compareA || ''} onChange={(e) => setCompareA((e.target as HTMLSelectElement).value || null)}>
                  <option value="">Select a run...</option>
                  {runs.map((r) => (
                    <option key={r.id} value={r.id}>{r.recording_name} ({r.speed_multiplier}x) - {r.status}</option>
                  ))}
                </select>
              </div>
              <div class="traffic-compare-vs">vs</div>
              <div class="traffic-compare-selector">
                <label class="traffic-compare-label">Run B</label>
                <select class="traffic-select" value={compareB || ''} onChange={(e) => setCompareB((e.target as HTMLSelectElement).value || null)}>
                  <option value="">Select a run...</option>
                  {runs.map((r) => (
                    <option key={r.id} value={r.id}>{r.recording_name} ({r.speed_multiplier}x) - {r.status}</option>
                  ))}
                </select>
              </div>
            </div>
            {runs.length === 0 && <div class="traffic-empty">No replay runs yet.</div>}
            {runs.length > 0 && (!compareA || !compareB) && <div class="traffic-empty">Select two runs above to compare.</div>}
            {compareA && compareB && (() => {
              const runA = runs.find((r) => r.id === compareA);
              const runB = runs.find((r) => r.id === compareB);
              if (!runA || !runB) return <div class="traffic-empty">Run not found.</div>;
              const sA = runA.latency_stats || { p50_ms: 0, p95_ms: 0, p99_ms: 0, avg_ms: 0 };
              const sB = runB.latency_stats || { p50_ms: 0, p95_ms: 0, p99_ms: 0, avg_ms: 0 };
              const rows = [
                { label: 'P50', a: sA.p50_ms, b: sB.p50_ms },
                { label: 'P95', a: sA.p95_ms, b: sB.p95_ms },
                { label: 'P99', a: sA.p99_ms, b: sB.p99_ms },
                { label: 'Avg', a: sA.avg_ms, b: sB.avg_ms },
                { label: 'Errors', a: runA.error_count, b: runB.error_count },
                { label: 'Requests', a: runA.sent_requests, b: runB.sent_requests },
              ];
              return (
                <table class="traffic-compare-table">
                  <thead><tr><th>Metric</th><th>Run A</th><th>Run B</th><th>Delta</th></tr></thead>
                  <tbody>
                    {rows.map((row) => {
                      const delta = row.b - row.a;
                      const pct = row.a !== 0 ? ((delta / row.a) * 100) : 0;
                      const lower = row.label !== 'Requests';
                      const better = lower ? delta < 0 : delta > 0;
                      const worse = lower ? delta > 0 : delta < 0;
                      return (
                        <tr key={row.label}>
                          <td class="traffic-compare-metric">{row.label}</td>
                          <td class="traffic-compare-value">{row.label === 'Errors' || row.label === 'Requests' ? row.a : formatMs(row.a)}</td>
                          <td class="traffic-compare-value">{row.label === 'Errors' || row.label === 'Requests' ? row.b : formatMs(row.b)}</td>
                          <td class={`traffic-compare-delta ${better ? 'traffic-compare-improved' : ''} ${worse ? 'traffic-compare-regressed' : ''}`}>
                            {delta === 0 ? '--' : `${delta > 0 ? '+' : ''}${row.label === 'Errors' || row.label === 'Requests' ? Math.round(delta) : formatMs(delta)}`}
                            {delta !== 0 && row.a !== 0 && <span class="traffic-compare-pct"> ({pct > 0 ? '+' : ''}{Math.round(pct)}%)</span>}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              );
            })()}
          </div>
        )}
      </div>
    </div>
  );
}
