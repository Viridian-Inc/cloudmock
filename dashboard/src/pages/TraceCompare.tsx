import { useState, useEffect } from 'preact/hooks';
import { compareTraces, compareBaseline } from '../api';
import { fmtDuration } from '../utils';

function getCompareParams() {
  const hash = location.hash;
  const queryStart = hash.indexOf('?');
  if (queryStart === -1) return { a: '', b: '', baseline: false };
  const params = new URLSearchParams(hash.substring(queryStart + 1));
  return {
    a: params.get('a') || '',
    b: params.get('b') || '',
    baseline: params.get('baseline') === 'true',
  };
}

const inputStyle: any = {
  padding: '6px 8px', borderRadius: '4px', border: '1px solid var(--border-default)',
  background: 'var(--bg-primary)', color: 'var(--text-primary)', fontSize: '12px', boxSizing: 'border-box',
  width: '280px',
};

const thStyle: any = { padding: '8px', textAlign: 'left', fontWeight: 500, color: 'var(--text-secondary)' };
const tdStyle: any = { padding: '8px' };

export function TraceComparePage() {
  const [traceA, setTraceA] = useState('');
  const [traceB, setTraceB] = useState('');
  const [isBaseline, setIsBaseline] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const { a, b, baseline } = getCompareParams();
    if (a) {
      setTraceA(a);
      setTraceB(b);
      setIsBaseline(baseline);
      setLoading(true);
      setError(null);
      const promise = baseline ? compareBaseline(a) : compareTraces(a, b);
      promise.then(setResult).catch(e => setError(e.message)).finally(() => setLoading(false));
    }
  }, [location.hash]);

  const handleSubmit = () => {
    if (!traceA) return;
    if (isBaseline) {
      location.hash = `#/traces/compare?a=${traceA}&baseline=true`;
    } else {
      if (!traceB) return;
      location.hash = `#/traces/compare?a=${traceA}&b=${traceB}`;
    }
  };

  const { a: paramA } = getCompareParams();
  const showInput = !paramA && !loading && !result;

  function deltaStyle(delta: number) {
    if (delta > 0) return { color: 'var(--error)' }; // slower = red
    if (delta < 0) return { color: 'var(--success)' }; // faster = green
    return {};
  }

  function formatDelta(delta: number) {
    if (delta > 0) return `+${fmtDuration(delta)}`;
    if (delta < 0) return `-${fmtDuration(Math.abs(delta))}`;
    return '0ms';
  }

  return (
    <div style={{ padding: '24px' }}>
      <h2 style={{ margin: '0 0 16px', fontSize: '20px', fontWeight: 600 }}>Trace Compare</h2>

      {/* Input bar */}
      {showInput && (
        <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
          <div style={{ display: 'flex', gap: '12px', alignItems: 'flex-end', flexWrap: 'wrap' }}>
            <label style={{ fontSize: '12px' }}>
              <div style={{ marginBottom: '4px', color: 'var(--text-secondary)' }}>Trace A</div>
              <input
                type="text"
                value={traceA}
                onInput={(e: any) => setTraceA(e.target.value)}
                placeholder="Trace ID..."
                style={inputStyle}
              />
            </label>
            {!isBaseline && (
              <label style={{ fontSize: '12px' }}>
                <div style={{ marginBottom: '4px', color: 'var(--text-secondary)' }}>Trace B</div>
                <input
                  type="text"
                  value={traceB}
                  onInput={(e: any) => setTraceB(e.target.value)}
                  placeholder="Trace ID..."
                  style={inputStyle}
                />
              </label>
            )}
            <label style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer', fontSize: '12px', paddingBottom: '4px' }}>
              <input
                type="checkbox"
                checked={isBaseline}
                onChange={(e: any) => setIsBaseline(e.target.checked)}
              />
              Compare with baseline
            </label>
            <button
              onClick={handleSubmit}
              style={{
                padding: '8px 16px', borderRadius: '6px', border: 'none',
                background: 'var(--brand-blue)', color: 'white', cursor: 'pointer', fontWeight: 600, fontSize: '13px',
              }}
            >
              Compare
            </button>
          </div>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div style={{ padding: '48px', textAlign: 'center', color: 'var(--text-tertiary)' }}>
          <div class="spinner" style={{ margin: '0 auto 12px', width: '24px', height: '24px', border: '3px solid var(--border-default)', borderTopColor: 'var(--brand-blue)', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
          Comparing traces...
        </div>
      )}

      {/* Error */}
      {error && (
        <div class="card" style={{ padding: '16px', color: 'var(--error)', marginBottom: '20px' }}>
          Error: {error}
          <button
            onClick={() => { setError(null); setResult(null); location.hash = '#/traces/compare'; }}
            style={{ marginLeft: '12px', padding: '4px 10px', borderRadius: '4px', border: '1px solid var(--border-default)', background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer', fontSize: '12px' }}
          >
            Try Again
          </button>
        </div>
      )}

      {/* Results */}
      {result && !loading && (
        <div>
          {/* Back button */}
          <div style={{ marginBottom: '16px' }}>
            <button
              onClick={() => { setResult(null); setError(null); location.hash = '#/traces/compare'; }}
              style={{ padding: '4px 10px', borderRadius: '4px', border: '1px solid var(--border-default)', background: 'transparent', color: 'var(--text-primary)', cursor: 'pointer', fontSize: '12px' }}
            >
              New Comparison
            </button>
          </div>

          {/* Summary bar */}
          <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
            <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Summary</h3>
            <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap', fontSize: '13px' }}>
              <div>
                <span style={{ color: 'var(--text-secondary)' }}>Latency A: </span>
                <span class="font-mono">{fmtDuration(result.latency_a ?? result.total_latency_a)}</span>
              </div>
              <div>
                <span style={{ color: 'var(--text-secondary)' }}>Latency B: </span>
                <span class="font-mono">{fmtDuration(result.latency_b ?? result.total_latency_b)}</span>
              </div>
              <div>
                <span style={{ color: 'var(--text-secondary)' }}>Delta: </span>
                <span class="font-mono" style={deltaStyle(result.delta ?? result.total_delta ?? 0)}>
                  {formatDelta(result.delta ?? result.total_delta ?? 0)}
                </span>
              </div>
              {(result.slower ?? result.slower_count) != null && (
                <div>
                  <span style={{ color: 'var(--error)', fontWeight: 600 }}>{result.slower ?? result.slower_count}</span>
                  <span style={{ color: 'var(--text-secondary)' }}> slower</span>
                </div>
              )}
              {(result.faster ?? result.faster_count) != null && (
                <div>
                  <span style={{ color: 'var(--success)', fontWeight: 600 }}>{result.faster ?? result.faster_count}</span>
                  <span style={{ color: 'var(--text-secondary)' }}> faster</span>
                </div>
              )}
              {(result.added ?? result.added_count) != null && (
                <div>
                  <span style={{ fontWeight: 600 }}>{result.added ?? result.added_count}</span>
                  <span style={{ color: 'var(--text-secondary)' }}> added</span>
                </div>
              )}
              {(result.removed ?? result.removed_count) != null && (
                <div>
                  <span style={{ fontWeight: 600 }}>{result.removed ?? result.removed_count}</span>
                  <span style={{ color: 'var(--text-secondary)' }}> removed</span>
                </div>
              )}
            </div>
          </div>

          {/* Matched spans */}
          {result.matches && result.matches.length > 0 && (
            <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
              <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600 }}>Matched Spans ({result.matches.length})</h3>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border-default)' }}>
                    <th style={thStyle}>Service</th>
                    <th style={thStyle}>Action</th>
                    <th style={thStyle}>Latency A</th>
                    <th style={thStyle}>Latency B</th>
                    <th style={thStyle}>Delta</th>
                    <th style={thStyle}>Status Change</th>
                  </tr>
                </thead>
                <tbody>
                  {result.matches.map((m: any, i: number) => (
                    <tr key={i} style={{ borderBottom: '1px solid var(--border-default)' }}>
                      <td style={tdStyle}><span style={{ fontWeight: 600 }}>{m.service}</span></td>
                      <td style={tdStyle} class="font-mono text-sm">{m.action}</td>
                      <td style={tdStyle} class="font-mono text-sm">{fmtDuration(m.latency_a ?? m.duration_a)}</td>
                      <td style={tdStyle} class="font-mono text-sm">{fmtDuration(m.latency_b ?? m.duration_b)}</td>
                      <td style={{ ...tdStyle, ...deltaStyle(m.delta ?? 0) }} class="font-mono text-sm">
                        {formatDelta(m.delta ?? 0)}
                      </td>
                      <td style={tdStyle}>
                        {m.status_change ? (
                          <span style={{
                            display: 'inline-block', padding: '2px 6px', borderRadius: '4px', fontSize: '11px', fontWeight: 600,
                            background: 'var(--warning-50)', color: 'var(--warning)',
                          }}>
                            {m.status_change}
                          </span>
                        ) : (
                          <span style={{ color: 'var(--text-tertiary)' }}>-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Only in A */}
          {result.only_in_a && result.only_in_a.length > 0 && (
            <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
              <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600, color: 'var(--error)' }}>
                Only in A ({result.only_in_a.length})
              </h3>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border-default)' }}>
                    <th style={thStyle}>Service</th>
                    <th style={thStyle}>Action</th>
                    <th style={thStyle}>Duration</th>
                  </tr>
                </thead>
                <tbody>
                  {result.only_in_a.map((s: any, i: number) => (
                    <tr key={i} style={{ borderBottom: '1px solid var(--border-default)' }}>
                      <td style={tdStyle}><span style={{ fontWeight: 600 }}>{s.service}</span></td>
                      <td style={tdStyle} class="font-mono text-sm">{s.action}</td>
                      <td style={tdStyle} class="font-mono text-sm">{fmtDuration(s.duration_ms ?? s.duration)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Only in B */}
          {result.only_in_b && result.only_in_b.length > 0 && (
            <div class="card" style={{ padding: '16px', marginBottom: '20px' }}>
              <h3 style={{ margin: '0 0 12px', fontSize: '14px', fontWeight: 600, color: 'var(--brand-blue)' }}>
                Only in B ({result.only_in_b.length})
              </h3>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border-default)' }}>
                    <th style={thStyle}>Service</th>
                    <th style={thStyle}>Action</th>
                    <th style={thStyle}>Duration</th>
                  </tr>
                </thead>
                <tbody>
                  {result.only_in_b.map((s: any, i: number) => (
                    <tr key={i} style={{ borderBottom: '1px solid var(--border-default)' }}>
                      <td style={tdStyle}><span style={{ fontWeight: 600 }}>{s.service}</span></td>
                      <td style={tdStyle} class="font-mono text-sm">{s.action}</td>
                      <td style={tdStyle} class="font-mono text-sm">{fmtDuration(s.duration_ms ?? s.duration)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
