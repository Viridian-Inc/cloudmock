import { useState, useCallback } from 'preact/hooks';
import { getAdminBase } from '../../lib/api';
import { peek } from '../../lib/pane-stack';
import type { RequestEvent } from '../../lib/types';
import { buildCurlCommand, byteSize, formatBytes, isXml, tokenizeJson, type JsonToken } from './event-utils';

/** Try to parse and pretty-print XML */
function formatXml(xml: string): string {
  let formatted = '';
  let indent = 0;
  const parts = xml.replace(/>\s*</g, '><').split(/(<[^>]+>)/);
  for (const part of parts) {
    if (!part.trim()) continue;
    if (part.startsWith('</')) {
      indent--;
      formatted += '  '.repeat(Math.max(0, indent)) + part + '\n';
    } else if (part.startsWith('<') && part.endsWith('/>')) {
      formatted += '  '.repeat(indent) + part + '\n';
    } else if (part.startsWith('<')) {
      formatted += '  '.repeat(indent) + part + '\n';
      if (!part.startsWith('<?')) indent++;
    } else {
      formatted += '  '.repeat(indent) + part + '\n';
    }
  }
  return formatted.trim();
}

/** Color map for JSON tokens */
const TOKEN_COLORS: Record<JsonToken['type'], string> = {
  key: '#4AE5F8',
  string: '#36d982',
  number: '#fad065',
  boolean: '#a78bfa',
  null: '#8b95a5',
  punctuation: '#8b95a5',
};

interface EventDetailProps {
  event: RequestEvent | null;
  /** All recent requests for the compare feature */
  allEvents?: RequestEvent[];
}

interface CollapsibleSectionProps {
  title: string;
  children: preact.ComponentChildren;
  defaultOpen?: boolean;
}

function CollapsibleSection({
  title,
  children,
  defaultOpen = false,
}: CollapsibleSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div class="detail-section">
      <button
        class="detail-section-header"
        onClick={() => setOpen((o) => !o)}
      >
        <span class="detail-section-chevron">{open ? '\u25BC' : '\u25B6'}</span>
        <span class="detail-section-title">{title}</span>
      </button>
      {open && <div class="detail-section-body">{children}</div>}
    </div>
  );
}

function HeadersTable({ headers }: { headers: Record<string, string> }) {
  const entries = Object.entries(headers);
  if (entries.length === 0) {
    return <div class="detail-empty">No headers</div>;
  }
  return (
    <table class="detail-headers-table">
      <tbody>
        {entries.map(([key, value]) => (
          <tr key={key}>
            <td class="detail-header-key">{key}</td>
            <td class="detail-header-value">{value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function SyntaxHighlightedBody({ data }: { data: unknown }) {
  if (data == null) {
    return <div class="detail-empty">No body</div>;
  }

  const rawStr = typeof data === 'string' ? data : JSON.stringify(data);
  const size = byteSize(data);

  // Try JSON pretty-print with syntax highlighting
  let jsonParsed: unknown = null;
  try {
    jsonParsed = typeof data === 'string' ? JSON.parse(data) : data;
  } catch (e) {
    // Not JSON — that's fine, will render as raw text
  }

  if (jsonParsed !== null && typeof jsonParsed === 'object') {
    const formatted = JSON.stringify(jsonParsed, null, 2);
    const tokens = tokenizeJson(formatted);
    return (
      <div>
        <div class="detail-body-meta">{formatBytes(size)}</div>
        <pre class="detail-json detail-json-highlighted">
          {tokens.map((tok, i) => (
            <span key={i} style={{ color: TOKEN_COLORS[tok.type] }}>{tok.text}</span>
          ))}
        </pre>
      </div>
    );
  }

  // Try XML formatting
  if (typeof rawStr === 'string' && isXml(rawStr)) {
    const formatted = formatXml(rawStr);
    return (
      <div>
        <div class="detail-body-meta">{formatBytes(size)} (XML)</div>
        <pre class="detail-json">{formatted}</pre>
      </div>
    );
  }

  // Raw text fallback
  return (
    <div>
      <div class="detail-body-meta">{formatBytes(size)}</div>
      <pre class="detail-json">{rawStr}</pre>
    </div>
  );
}

function statusClass(status: number): string {
  if (status >= 200 && status < 300) return 'status-2xx';
  if (status >= 300 && status < 400) return 'status-3xx';
  if (status >= 400 && status < 500) return 'status-4xx';
  if (status >= 500) return 'status-5xx';
  return '';
}

// --- Request Diff Component ---

interface DiffLine {
  type: 'same' | 'added' | 'removed' | 'changed';
  key: string;
  leftValue?: string;
  rightValue?: string;
}

function diffHeaders(
  left: Record<string, string>,
  right: Record<string, string>,
): DiffLine[] {
  const lines: DiffLine[] = [];
  const allKeys = new Set([...Object.keys(left), ...Object.keys(right)]);

  for (const key of [...allKeys].sort()) {
    const lv = left[key];
    const rv = right[key];
    if (lv !== undefined && rv === undefined) {
      lines.push({ type: 'removed', key, leftValue: lv });
    } else if (lv === undefined && rv !== undefined) {
      lines.push({ type: 'added', key, rightValue: rv });
    } else if (lv !== rv) {
      lines.push({ type: 'changed', key, leftValue: lv, rightValue: rv });
    } else {
      lines.push({ type: 'same', key, leftValue: lv, rightValue: rv });
    }
  }
  return lines;
}

function diffBodyLines(leftBody: unknown, rightBody: unknown): DiffLine[] {
  const leftStr = leftBody == null ? '' : (typeof leftBody === 'string' ? leftBody : JSON.stringify(leftBody, null, 2));
  const rightStr = rightBody == null ? '' : (typeof rightBody === 'string' ? rightBody : JSON.stringify(rightBody, null, 2));

  // Try to pretty-print JSON for comparison
  let leftLines: string[];
  let rightLines: string[];
  try {
    const lp = JSON.parse(leftStr);
    leftLines = JSON.stringify(lp, null, 2).split('\n');
  } catch (_e) {
    leftLines = leftStr.split('\n');
  }
  try {
    const rp = JSON.parse(rightStr);
    rightLines = JSON.stringify(rp, null, 2).split('\n');
  } catch (_e) {
    rightLines = rightStr.split('\n');
  }

  // Simple line-by-line diff
  const maxLen = Math.max(leftLines.length, rightLines.length);
  const lines: DiffLine[] = [];
  for (let i = 0; i < maxLen; i++) {
    const ll = i < leftLines.length ? leftLines[i] : undefined;
    const rl = i < rightLines.length ? rightLines[i] : undefined;
    if (ll !== undefined && rl === undefined) {
      lines.push({ type: 'removed', key: String(i + 1), leftValue: ll });
    } else if (ll === undefined && rl !== undefined) {
      lines.push({ type: 'added', key: String(i + 1), rightValue: rl });
    } else if (ll !== rl) {
      lines.push({ type: 'changed', key: String(i + 1), leftValue: ll, rightValue: rl });
    } else {
      lines.push({ type: 'same', key: String(i + 1), leftValue: ll, rightValue: rl });
    }
  }
  return lines;
}

const DIFF_COLORS: Record<DiffLine['type'], string> = {
  same: 'transparent',
  added: 'rgba(54, 217, 130, 0.1)',
  removed: 'rgba(255, 78, 94, 0.1)',
  changed: 'rgba(250, 208, 101, 0.1)',
};

const DIFF_INDICATOR: Record<DiffLine['type'], string> = {
  same: ' ',
  added: '+',
  removed: '-',
  changed: '~',
};

function RequestDiff({
  event,
  allEvents,
  onClose,
}: {
  event: RequestEvent;
  allEvents: RequestEvent[];
  onClose: () => void;
}) {
  const [compareId, setCompareId] = useState<string | null>(null);
  const compareEvent = compareId ? allEvents.find((e) => e.id === compareId) ?? null : null;

  const candidates = allEvents.filter((e) => e.id !== event.id).slice(0, 20);

  return (
    <div class="diff-overlay">
      <div class="diff-panel">
        <div class="diff-panel-header">
          <h3 class="diff-panel-title">Compare Requests</h3>
          <button class="btn btn-ghost diff-close-btn" onClick={onClose}>{'\u2715'}</button>
        </div>

        {!compareEvent ? (
          <div class="diff-picker">
            <div class="diff-picker-label">Select a request to compare with:</div>
            <div class="diff-picker-list">
              {candidates.length === 0 && (
                <div class="diff-picker-empty">No other requests available</div>
              )}
              {candidates.map((c) => (
                <button
                  key={c.id}
                  class="diff-picker-item"
                  onClick={() => setCompareId(c.id)}
                >
                  <span class={`status-pill ${statusClass(c.status_code)}`}>{c.status_code}</span>
                  <span class="diff-picker-method">{c.method}</span>
                  <span class="diff-picker-path">{c.path}</span>
                  <span class="diff-picker-latency">{c.latency_ms}ms</span>
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div class="diff-content">
            {/* Summary comparison */}
            <div class="diff-summary-row">
              <div class="diff-summary-side">
                <div class="diff-summary-label">Original</div>
                <div class="diff-summary-info">
                  <span class={`status-pill ${statusClass(event.status_code)}`}>{event.status_code}</span>
                  <span class="diff-summary-method">{event.method}</span>
                  <span class="diff-summary-path">{event.path}</span>
                </div>
              </div>
              <div class="diff-summary-side">
                <div class="diff-summary-label">Compare</div>
                <div class="diff-summary-info">
                  <span class={`status-pill ${statusClass(compareEvent.status_code)}`}>{compareEvent.status_code}</span>
                  <span class="diff-summary-method">{compareEvent.method}</span>
                  <span class="diff-summary-path">{compareEvent.path}</span>
                </div>
              </div>
            </div>

            {/* Timing diff */}
            <div class="diff-timing">
              <span class="diff-timing-label">Latency:</span>
              <span class="diff-timing-value">{event.latency_ms}ms</span>
              <span class="diff-timing-arrow">{'\u2192'}</span>
              <span class="diff-timing-value">{compareEvent.latency_ms}ms</span>
              <span class={`diff-timing-delta ${compareEvent.latency_ms > event.latency_ms ? 'diff-delta-worse' : 'diff-delta-better'}`}>
                ({compareEvent.latency_ms > event.latency_ms ? '+' : ''}{(compareEvent.latency_ms - event.latency_ms).toFixed(1)}ms)
              </span>
            </div>

            {/* Headers diff */}
            <div class="diff-section">
              <div class="diff-section-title">Request Headers</div>
              <div class="diff-table-wrap">
                <table class="diff-table">
                  <thead>
                    <tr>
                      <th class="diff-th-indicator"></th>
                      <th>Header</th>
                      <th>Original</th>
                      <th>Compare</th>
                    </tr>
                  </thead>
                  <tbody>
                    {diffHeaders(event.request_headers ?? {}, compareEvent.request_headers ?? {}).map((line) => (
                      <tr key={line.key} style={{ background: DIFF_COLORS[line.type] }}>
                        <td class="diff-indicator">{DIFF_INDICATOR[line.type]}</td>
                        <td class="diff-cell-key">{line.key}</td>
                        <td class="diff-cell-value">{line.leftValue ?? ''}</td>
                        <td class="diff-cell-value">{line.rightValue ?? ''}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Body diff */}
            <div class="diff-section">
              <div class="diff-section-title">Request Body</div>
              <div class="diff-body-wrap">
                {diffBodyLines(event.request_body, compareEvent.request_body).map((line) => (
                  <div key={line.key} class="diff-body-line" style={{ background: DIFF_COLORS[line.type] }}>
                    <span class="diff-body-indicator">{DIFF_INDICATOR[line.type]}</span>
                    <span class="diff-body-left">{line.leftValue ?? ''}</span>
                    <span class="diff-body-separator">{'\u2502'}</span>
                    <span class="diff-body-right">{line.rightValue ?? ''}</span>
                  </div>
                ))}
                {event.request_body == null && compareEvent.request_body == null && (
                  <div class="diff-body-empty">No body in either request</div>
                )}
              </div>
            </div>

            {/* Response body diff */}
            <div class="diff-section">
              <div class="diff-section-title">Response Body</div>
              <div class="diff-body-wrap">
                {diffBodyLines(event.response_body, compareEvent.response_body).map((line) => (
                  <div key={line.key} class="diff-body-line" style={{ background: DIFF_COLORS[line.type] }}>
                    <span class="diff-body-indicator">{DIFF_INDICATOR[line.type]}</span>
                    <span class="diff-body-left">{line.leftValue ?? ''}</span>
                    <span class="diff-body-separator">{'\u2502'}</span>
                    <span class="diff-body-right">{line.rightValue ?? ''}</span>
                  </div>
                ))}
                {event.response_body == null && compareEvent.response_body == null && (
                  <div class="diff-body-empty">No body in either response</div>
                )}
              </div>
            </div>

            <button class="btn btn-ghost diff-back-btn" onClick={() => setCompareId(null)}>
              {'\u2190'} Pick different request
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

export function EventDetail({ event, allEvents = [] }: EventDetailProps) {
  const [copied, setCopied] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [replayState, setReplayState] = useState<
    { status: 'idle' } | { status: 'replaying' } | { status: 'done'; code: number } | { status: 'error'; message: string }
  >({ status: 'idle' });

  const handleCopyCurl = useCallback(() => {
    if (!event) return;
    const curl = buildCurlCommand(event);
    navigator.clipboard.writeText(curl).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [event]);

  const handleReplay = useCallback(async () => {
    if (!event) return;
    setReplayState({ status: 'replaying' });
    try {
      const base = getAdminBase();
      const headers: Record<string, string> = { ...event.request_headers };
      // Remove host header to avoid conflicts
      delete headers['host'];
      delete headers['Host'];

      const body =
        event.request_body != null && event.request_body !== ''
          ? typeof event.request_body === 'string'
            ? event.request_body
            : JSON.stringify(event.request_body)
          : undefined;

      const res = await fetch(`${base}${event.path}`, {
        method: event.method,
        headers,
        body: ['GET', 'HEAD'].includes(event.method.toUpperCase()) ? undefined : body,
      });
      setReplayState({ status: 'done', code: res.status });
      setTimeout(() => setReplayState({ status: 'idle' }), 4000);
    } catch (err: any) {
      setReplayState({ status: 'error', message: err.message || 'Request failed' });
      setTimeout(() => setReplayState({ status: 'idle' }), 4000);
    }
  }, [event]);

  if (!event) {
    return (
      <div class="event-detail event-detail-placeholder">
        <div class="event-detail-placeholder-text">
          Select an event to inspect
        </div>
      </div>
    );
  }

  return (
    <div class="event-detail">
      <div class="event-detail-actions">
        <button class="btn btn-ghost btn-copy-curl" onClick={handleCopyCurl}>
          {copied ? 'Copied!' : 'Copy as curl'}
        </button>
        <button
          class="btn btn-ghost btn-compare"
          onClick={() => setShowDiff(true)}
        >
          Compare
        </button>
        <button
          class={`btn btn-ghost btn-replay ${replayState.status === 'done' ? 'btn-replay-success' : ''} ${replayState.status === 'error' ? 'btn-replay-error' : ''}`}
          onClick={handleReplay}
          disabled={replayState.status === 'replaying'}
        >
          {replayState.status === 'replaying'
            ? 'Replaying...'
            : replayState.status === 'done'
              ? `\u2713 Replayed (${replayState.code})`
              : replayState.status === 'error'
                ? `\u2717 Failed`
                : 'Replay'}
        </button>
      </div>
      <div class="event-detail-summary">
        <div class="event-detail-row">
          <span class="detail-label">Service</span>
          <span class="event-row-source">{event.service}</span>
        </div>
        <div class="event-detail-row">
          <span class="detail-label">Action</span>
          <span class="detail-value">{event.action}</span>
        </div>
        {event.trace_id && (
          <div class="event-detail-row">
            <span class="detail-label">Trace</span>
            <span class="detail-value detail-mono">
              {event.trace_id}
              <button
                class="btn btn-ghost btn-view-trace"
                onClick={() => {
                  peek({
                    view: 'traces',
                    segments: [event.trace_id!],
                    title: `Trace ${event.trace_id!.slice(0, 8)}`,
                  });
                }}
                title="Open trace in a peek pane (Esc to close)"
              >
                Peek Trace {'\u2192'}
              </button>
            </span>
          </div>
        )}
        <div class="event-detail-row">
          <span class="detail-label">Method</span>
          <span class="detail-value detail-mono">{event.method}</span>
        </div>
        <div class="event-detail-row">
          <span class="detail-label">Path</span>
          <span class="detail-value detail-mono">{event.path}</span>
        </div>
        <div class="event-detail-row">
          <span class="detail-label">Status</span>
          <span class={`status-pill ${statusClass(event.status_code)}`}>
            {event.status_code}
          </span>
        </div>
        <div class="event-detail-row">
          <span class="detail-label">Latency</span>
          <span class="detail-value detail-mono">{event.latency_ms}ms</span>
        </div>
      </div>

      <CollapsibleSection title="Request Headers">
        <HeadersTable headers={event.request_headers ?? {}} />
      </CollapsibleSection>

      <CollapsibleSection title="Request Body">
        <SyntaxHighlightedBody data={event.request_body} />
      </CollapsibleSection>

      <CollapsibleSection title="Response Headers">
        <HeadersTable headers={event.response_headers ?? {}} />
      </CollapsibleSection>

      <CollapsibleSection title="Response Body" defaultOpen>
        <SyntaxHighlightedBody data={event.response_body} />
      </CollapsibleSection>

      {showDiff && (
        <RequestDiff
          event={event}
          allEvents={allEvents}
          onClose={() => setShowDiff(false)}
        />
      )}
    </div>
  );
}
