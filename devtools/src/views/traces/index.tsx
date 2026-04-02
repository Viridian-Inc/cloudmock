import { useState, useEffect } from 'preact/hooks';
import { SplitPanel } from '../../components/panels/split-panel';
import { cachedApi } from '../../lib/api';
import { Waterfall } from './waterfall';
import { Flamegraph } from './flamegraph';
import { CompareView } from './compare-view';
import './traces.css';

interface TraceSummary {
  TraceID: string;
  RootService: string;
  RootAction: string;
  Method: string;
  Path: string;
  DurationMs: number;
  StatusCode: number;
  SpanCount: number;
  HasError: boolean;
  StartTime: string;
}

function formatTime(ts: string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--:--';
  return d.toTimeString().slice(0, 8);
}

function statusClass(status: number): string {
  if (status >= 200 && status < 300) return 'status-2xx';
  if (status >= 400 && status < 500) return 'status-4xx';
  if (status >= 500) return 'status-5xx';
  return '';
}

function TraceList({
  traces,
  selectedId,
  onSelect,
  compareMode,
  onToggleCompare,
  compareTraceId,
  onCompareSelect,
}: {
  traces: TraceSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  compareMode: boolean;
  onToggleCompare: () => void;
  compareTraceId: string | null;
  onCompareSelect: (id: string) => void;
}) {
  const [search, setSearch] = useState('');

  const filtered = traces.filter((t) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      t.RootService.toLowerCase().includes(q) ||
      t.Path.toLowerCase().includes(q) ||
      t.TraceID.toLowerCase().includes(q)
    );
  });

  return (
    <div class="trace-list">
      <div class="trace-list-header">
        <input
          class="input"
          style="width: 100%"
          placeholder="Filter traces..."
          value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
        />
        <div class="trace-list-toolbar">
          <button
            class={`trace-toolbar-btn ${compareMode ? 'trace-toolbar-btn-active' : ''}`}
            onClick={onToggleCompare}
            title="Compare two traces side-by-side"
          >
            Compare
          </button>
          {compareMode && (
            <span class="trace-compare-hint">
              {!selectedId
                ? 'Select trace A'
                : !compareTraceId
                  ? 'Select trace B'
                  : 'Comparing'}
            </span>
          )}
        </div>
      </div>
      <div class="trace-list-body">
        {filtered.map((t) => {
          const isA = t.TraceID === selectedId;
          const isB = t.TraceID === compareTraceId;
          const isSelected = isA || isB;

          return (
            <div
              key={t.TraceID}
              class={`trace-row ${isSelected ? 'trace-row-selected' : ''} ${isB ? 'trace-row-compare-b' : ''}`}
              onClick={() => {
                if (compareMode && selectedId && !compareTraceId && t.TraceID !== selectedId) {
                  onCompareSelect(t.TraceID);
                } else {
                  onSelect(t.TraceID);
                }
              }}
            >
              {compareMode && (
                <span class="trace-row-compare-badge">
                  {isA ? 'A' : isB ? 'B' : ''}
                </span>
              )}
              <span class="trace-row-time">{formatTime(t.StartTime)}</span>
              <span class="trace-row-service">{t.RootService}</span>
              <span class={`status-pill ${statusClass(t.StatusCode)}`}>
                {t.StatusCode}
              </span>
              <span class="trace-row-path">{t.Method} {t.Path}</span>
              <span class="trace-row-duration">
                {t.DurationMs < 1 ? t.DurationMs.toFixed(2) : Math.round(t.DurationMs)}ms
              </span>
              <span class="trace-row-spans">{t.SpanCount} span{t.SpanCount !== 1 ? 's' : ''}</span>
              {t.HasError && <span class="trace-row-error">!</span>}
            </div>
          );
        })}
        {filtered.length === 0 && (
          <div class="trace-list-empty">No traces found</div>
        )}
      </div>
    </div>
  );
}


type ViewMode = 'waterfall' | 'flamegraph';

function DetailPanel({
  selectedId,
  compareMode,
  compareTraceId,
  viewMode,
  onViewModeChange,
}: {
  selectedId: string | null;
  compareMode: boolean;
  compareTraceId: string | null;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
}) {
  // Compare mode: show comparison view when both traces are selected
  if (compareMode && selectedId && compareTraceId) {
    return <CompareView traceIdA={selectedId} traceIdB={compareTraceId} />;
  }

  return (
    <div class="trace-detail-panel">
      {selectedId && (
        <div class="trace-detail-panel-header">
          <div class="trace-view-toggle">
            <button
              class={`trace-view-toggle-btn ${viewMode === 'waterfall' ? 'trace-view-toggle-btn-active' : ''}`}
              onClick={() => onViewModeChange('waterfall')}
            >
              Waterfall
            </button>
            <button
              class={`trace-view-toggle-btn ${viewMode === 'flamegraph' ? 'trace-view-toggle-btn-active' : ''}`}
              onClick={() => onViewModeChange('flamegraph')}
            >
              Flamegraph
            </button>
          </div>
        </div>
      )}
      {viewMode === 'waterfall' ? (
        <Waterfall traceId={selectedId} />
      ) : (
        <Flamegraph traceId={selectedId} />
      )}
    </div>
  );
}

export function TracesView() {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [compareMode, setCompareMode] = useState(false);
  const [compareTraceId, setCompareTraceId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('waterfall');

  useEffect(() => {
    cachedApi<TraceSummary[]>('/api/traces', 'traces:list')
      .then((data) => {
        setTraces(data);

        // Check URL hash for #trace={id} from cross-view navigation
        const hash = window.location.hash;
        const traceMatch = hash.match(/^#trace=(.+)/);
        if (traceMatch) {
          const traceId = decodeURIComponent(traceMatch[1]);
          setSelectedId(traceId);
          // Clear the hash after consuming it
          history.replaceState(null, '', window.location.pathname + window.location.search);
        }
      })
      .catch(() => setTraces([]));
  }, []);

  const handleToggleCompare = () => {
    if (compareMode) {
      // Exiting compare mode: clear compare selection
      setCompareMode(false);
      setCompareTraceId(null);
    } else {
      setCompareMode(true);
      setCompareTraceId(null);
    }
  };

  const handleSelect = (id: string) => {
    setSelectedId(id);
    // When selecting a new primary trace in compare mode, reset the second trace
    if (compareMode) {
      setCompareTraceId(null);
    }
  };

  return (
    <div class="traces-view">
      <SplitPanel
        initialSplit={50}
        direction="horizontal"
        minSize={250}
        left={
          <TraceList
            traces={traces}
            selectedId={selectedId}
            onSelect={handleSelect}
            compareMode={compareMode}
            onToggleCompare={handleToggleCompare}
            compareTraceId={compareTraceId}
            onCompareSelect={setCompareTraceId}
          />
        }
        right={
          <DetailPanel
            selectedId={selectedId}
            compareMode={compareMode}
            compareTraceId={compareTraceId}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
          />
        }
      />
    </div>
  );
}
