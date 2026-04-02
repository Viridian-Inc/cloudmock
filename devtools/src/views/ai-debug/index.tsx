import { useState, useRef, useEffect } from 'preact/hooks';
import { api, getAdminBase, getTraces } from '../../lib/api';
import './ai-debug.css';

interface ExplainResponse {
  explanation: string;
  streaming?: boolean;
}

interface RecentTrace {
  id: string;
  trace_id?: string;
  service: string;
  action: string;
  status_code: number;
  timestamp: string;
  path?: string;
}

export function AIDebugView() {
  const [requestId, setRequestId] = useState('');
  const [explanation, setExplanation] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [streaming, setStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const [recentErrors, setRecentErrors] = useState<RecentTrace[]>([]);
  const [suggestLoading, setSuggestLoading] = useState(false);

  // Fetch recent failed traces on mount and periodically
  useEffect(() => {
    let cancelled = false;

    async function fetchRecentErrors() {
      setSuggestLoading(true);
      try {
        const traces = await getTraces();
        if (cancelled) return;
        const errors = (traces as RecentTrace[])
          .filter((t) => t.status_code >= 400)
          .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
          .slice(0, 5);
        setRecentErrors(errors);
      } catch (e) {
        console.debug('[AIDebug] Failed to fetch recent errors (suggestions are optional):', e);
      } finally {
        if (!cancelled) setSuggestLoading(false);
      }
    }

    fetchRecentErrors();
    const interval = setInterval(fetchRecentErrors, 15000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, []);

  async function handleExplain() {
    const id = requestId.trim();
    if (!id) return;

    // Capture fresh admin base at call time
    const base = getAdminBase();

    // Cancel any in-flight request
    if (abortRef.current) {
      abortRef.current.abort();
    }

    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    setError(null);
    setExplanation(null);
    setStreaming(false);

    try {
      // First try as a streaming request
      const url = `${base}/api/explain`;
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ request_id: id }),
        signal: controller.signal,
      });

      if (!res.ok) {
        const body = await res.text().catch((_e) => '');
        throw new Error(`API ${res.status}: ${res.statusText}${body ? ` — ${body}` : ''}`);
      }

      const contentType = res.headers.get('content-type') || '';

      // Handle streaming (text/event-stream or chunked)
      if (
        contentType.includes('text/event-stream') ||
        contentType.includes('text/plain')
      ) {
        setStreaming(true);
        setLoading(false);
        const reader = res.body?.getReader();
        if (!reader) {
          throw new Error('Streaming not supported by this browser');
        }

        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          const chunk = decoder.decode(value, { stream: true });

          // Handle SSE format: lines starting with "data: "
          if (contentType.includes('text/event-stream')) {
            const lines = chunk.split('\n');
            for (const line of lines) {
              if (line.startsWith('data: ')) {
                const data = line.slice(6);
                if (data === '[DONE]') continue;
                try {
                  const parsed = JSON.parse(data);
                  buffer += parsed.text || parsed.content || parsed.explanation || '';
                } catch (_e) {
                  // Treat as raw text
                  buffer += data;
                }
              }
            }
          } else {
            buffer += chunk;
          }

          setExplanation(buffer);
        }

        setStreaming(false);
        if (!buffer) {
          setExplanation(null);
          setError('The AI explanation endpoint returned an empty response. Make sure the request ID exists in your traces.');
        }
      } else {
        // Standard JSON response
        const data: ExplainResponse = await res.json();
        if (!data.explanation) {
          setError('The AI explanation endpoint returned an empty response. Make sure the request ID exists in your traces.');
        } else {
          setExplanation(data.explanation);
        }
        setLoading(false);
      }
    } catch (err: any) {
      if (err.name === 'AbortError') return;
      setError(err.message || 'Failed to get explanation');
      setLoading(false);
      setStreaming(false);
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null;
      }
    }
  }

  function handleCancel() {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setLoading(false);
    setStreaming(false);
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleExplain();
    }
  }

  /**
   * Render markdown text into preact elements.
   * Splits by fenced code blocks first (separate pass) so that backticks
   * nested inside list items or other inline contexts don't break parsing.
   */
  function renderExplanation(text: string): preact.VNode[] {
    const elements: preact.VNode[] = [];
    const codeBlockRegex = /```(\w*)\n([\s\S]*?)```/g;
    let lastIndex = 0;
    let match;

    while ((match = codeBlockRegex.exec(text)) !== null) {
      // Render text before the code block as inline markdown
      if (match.index > lastIndex) {
        elements.push(...renderInlineMarkdown(text.slice(lastIndex, match.index)));
      }
      // Render the fenced code block
      elements.push(
        <pre class="ai-md-code-block" key={`cb-${match.index}`}>
          <code>{match[2]}</code>
        </pre>
      );
      lastIndex = match.index + match[0].length;
    }

    // Render remaining text after the last code block
    if (lastIndex < text.length) {
      elements.push(...renderInlineMarkdown(text.slice(lastIndex)));
    }

    return elements;
  }

  /** Render non-code-block markdown (headings, lists, paragraphs). */
  function renderInlineMarkdown(text: string): preact.VNode[] {
    const lines = text.split('\n');
    const elements: preact.VNode[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      if (line.startsWith('### ')) {
        elements.push(<h4 class="ai-md-h3" key={`l-${i}`}>{line.slice(4)}</h4>);
      } else if (line.startsWith('## ')) {
        elements.push(<h3 class="ai-md-h2" key={`l-${i}`}>{line.slice(3)}</h3>);
      } else if (line.startsWith('# ')) {
        elements.push(<h2 class="ai-md-h1" key={`l-${i}`}>{line.slice(2)}</h2>);
      } else if (line.startsWith('- ') || line.startsWith('* ')) {
        elements.push(
          <div class="ai-md-list-item" key={`l-${i}`}>
            <span class="ai-md-bullet">{'\u2022'}</span>
            <span>{formatInline(line.slice(2))}</span>
          </div>
        );
      } else if (line.trim() === '') {
        elements.push(<div class="ai-md-spacer" key={`l-${i}`} />);
      } else {
        elements.push(<p class="ai-md-p" key={`l-${i}`}>{formatInline(line)}</p>);
      }
    }

    return elements;
  }

  function formatInline(text: string): (string | preact.VNode)[] {
    const parts: (string | preact.VNode)[] = [];
    const regex = /(`[^`]+`|\*\*[^*]+\*\*)/g;
    let last = 0;
    let match: RegExpExecArray | null;

    while ((match = regex.exec(text)) !== null) {
      if (match.index > last) {
        parts.push(text.slice(last, match.index));
      }
      const token = match[0];
      if (token.startsWith('`')) {
        parts.push(<code class="ai-md-inline-code" key={match.index}>{token.slice(1, -1)}</code>);
      } else {
        parts.push(<strong key={match.index}>{token.slice(2, -2)}</strong>);
      }
      last = match.index + token.length;
    }

    if (last < text.length) {
      parts.push(text.slice(last));
    }

    return parts;
  }

  function formatRelativeTime(timestamp: string): string {
    const now = Date.now();
    const then = new Date(timestamp).getTime();
    const diffMs = now - then;
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return `${diffHr}h ago`;
    return `${Math.floor(diffHr / 24)}d ago`;
  }

  const platformHint = typeof navigator !== 'undefined' && navigator.platform.includes('Mac') ? 'Cmd' : 'Ctrl';
  const isActive = loading || streaming;

  return (
    <div class="ai-debug-view">
      <div class="ai-debug-header">
        <div class="ai-debug-header-left">
          <h2 class="ai-debug-title">AI Debug</h2>
          <span class="ai-debug-subtitle">Cross-stack narrative explanations</span>
        </div>
      </div>

      <div class="ai-debug-body">
        <div class="ai-debug-input-section">
          <label class="ai-debug-label">Request ID or Trace ID</label>
          <div class="ai-debug-input-row">
            <textarea
              class="input ai-debug-textarea"
              placeholder="Paste a request ID, trace ID, or error reference..."
              value={requestId}
              onInput={(e) => setRequestId((e.target as HTMLTextAreaElement).value)}
              onKeyDown={handleKeyDown}
              rows={2}
              disabled={isActive}
            />
            {isActive ? (
              <button
                class="btn btn-ghost ai-debug-explain-btn"
                onClick={handleCancel}
              >
                Cancel
              </button>
            ) : (
              <button
                class="btn btn-primary ai-debug-explain-btn"
                onClick={handleExplain}
                disabled={!requestId.trim()}
              >
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M7 1v12M3.5 4.5L7 1l3.5 3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
                Explain
              </button>
            )}
          </div>
          <div class="ai-debug-input-hint">Press {platformHint}+Enter to submit</div>
        </div>

        {recentErrors.length > 0 && !explanation && !loading && (
          <div class="ai-debug-suggestions">
            <div class="ai-debug-suggestions-header">
              <span class="ai-debug-suggestions-label">Suggest from recent</span>
              {suggestLoading && <span class="ai-debug-suggestions-loading">refreshing...</span>}
            </div>
            <div class="ai-debug-suggestions-list">
              {recentErrors.map((trace) => (
                <button
                  key={trace.id}
                  class="ai-debug-suggestion-item"
                  onClick={() => setRequestId(trace.trace_id || trace.id)}
                  disabled={isActive}
                >
                  <span class={`ai-debug-suggestion-status status-${trace.status_code >= 500 ? '5xx' : '4xx'}`}>
                    {trace.status_code}
                  </span>
                  <span class="ai-debug-suggestion-service">{trace.service}</span>
                  <span class="ai-debug-suggestion-action">{trace.action || trace.path || ''}</span>
                  <span class="ai-debug-suggestion-time">
                    {formatRelativeTime(trace.timestamp)}
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}

        <div class="ai-debug-response-section">
          {error && (
            <div class="ai-debug-error">
              <span class="ai-debug-error-icon">!</span>
              {error}
            </div>
          )}

          {loading && !explanation && (
            <div class="ai-debug-loading">
              <div class="ai-debug-loading-bar" />
              <span class="ai-debug-loading-text">Analyzing request across all services...</span>
            </div>
          )}

          {explanation && (
            <div class="ai-debug-explanation">
              <div class="ai-debug-explanation-header">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <path d="M8 1a5 5 0 015 5c0 1.7-.8 3.1-2 4v2a1 1 0 01-1 1H6a1 1 0 01-1-1v-2C3.8 9.1 3 7.7 3 6a5 5 0 015-5z" stroke="var(--brand-teal)" stroke-width="1.2" fill="rgba(74,229,248,0.08)" />
                  <path d="M6 14h4" stroke="var(--brand-teal)" stroke-width="1.2" stroke-linecap="round" />
                </svg>
                <span class="ai-debug-explanation-label">
                  {streaming ? 'Streaming...' : 'Explanation'}
                </span>
              </div>
              <div class="ai-debug-explanation-body">
                {renderExplanation(explanation)}
                {streaming && <span class="ai-debug-cursor" />}
              </div>
            </div>
          )}

          {!explanation && !loading && !error && (
            <div class="ai-debug-placeholder">
              <div class="ai-debug-placeholder-icon">
                <svg width="56" height="56" viewBox="0 0 56 56" fill="none">
                  <circle cx="28" cy="28" r="24" stroke="var(--border-default)" stroke-width="1.5" stroke-dasharray="4 4" />
                  <path d="M28 16a8 8 0 018 8c0 2.7-1.3 5-3.2 6.4-.7.5-1.3 1.2-1.3 2.1V34a1.5 1.5 0 01-1.5 1.5h-4A1.5 1.5 0 0124.5 34v-1.5c0-.9-.6-1.6-1.3-2.1A8 8 0 0128 16z" stroke="var(--text-tertiary)" stroke-width="1.5" fill="none" />
                  <path d="M25 38h6" stroke="var(--text-tertiary)" stroke-width="1.5" stroke-linecap="round" />
                  <path d="M26 40h4" stroke="var(--text-tertiary)" stroke-width="1.5" stroke-linecap="round" />
                </svg>
              </div>
              <div class="ai-debug-placeholder-title">Intelligent Request Analysis</div>
              <div class="ai-debug-placeholder-text">
                Select any request, trace, or error and get a narrative explanation with full cross-stack context.
              </div>
              <div class="ai-debug-placeholder-features">
                <div class="ai-debug-feature">
                  <span class="ai-debug-feature-dot" />
                  Root cause identification
                </div>
                <div class="ai-debug-feature">
                  <span class="ai-debug-feature-dot" />
                  Service dependency mapping
                </div>
                <div class="ai-debug-feature">
                  <span class="ai-debug-feature-dot" />
                  Timeline reconstruction
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
