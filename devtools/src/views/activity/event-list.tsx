import { useRef, useEffect } from 'preact/hooks';
import type { RequestEvent } from '../../lib/types';
import { EventRow } from './event-row';

interface Filters {
  service: string;
  status: string;
}

// Hash-based color palette matching topology
const SOURCE_PALETTE = [
  '#4AE5F8', '#a78bfa', '#60a5fa', '#fbbf24', '#818cf8',
  '#4ade80', '#f472b6', '#22d3ee', '#f87171', '#fb923c',
  '#34d399', '#c084fc', '#38bdf8', '#facc15', '#a3e635',
];

const sourceColorCache = new Map<string, string>();
function getSourceColor(source: string): string {
  if (!sourceColorCache.has(source)) {
    let hash = 5381;
    for (let i = 0; i < source.length; i++) {
      hash = ((hash << 5) + hash + source.charCodeAt(i)) >>> 0;
    }
    sourceColorCache.set(source, SOURCE_PALETTE[hash % SOURCE_PALETTE.length]);
  }
  return sourceColorCache.get(source)!;
}

interface EventListProps {
  events: RequestEvent[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  paused: boolean;
  onTogglePause: () => void;
  search: string;
  onSearchChange: (value: string) => void;
  filters: Filters;
  onFilterChange: (key: keyof Filters, value: string) => void;
  onClear: () => void;
  onExportHAR: () => void;
  connected: boolean;
  serviceNames: string[];
  dataSource?: 'sse' | 'polling';
  sourceFilter: Set<string>;
  onToggleSource: (source: string) => void;
  onResetSourceFilter: () => void;
  sourceCounts: Map<string, number>;
}

export function EventList({
  events,
  selectedId,
  onSelect,
  paused,
  onTogglePause,
  search,
  onSearchChange,
  filters,
  onFilterChange,
  onClear,
  onExportHAR,
  connected,
  serviceNames,
  dataSource = 'sse',
  sourceFilter,
  onToggleSource,
  onResetSourceFilter,
  sourceCounts,
}: EventListProps) {
  const listRef = useRef<HTMLDivElement>(null);
  const shouldAutoScroll = useRef(true);

  // Auto-scroll to top (newest) when not paused
  useEffect(() => {
    if (!paused && shouldAutoScroll.current && listRef.current) {
      listRef.current.scrollTop = 0;
    }
  }, [events, paused]);

  const handleScroll = () => {
    if (listRef.current) {
      shouldAutoScroll.current = listRef.current.scrollTop < 20;
    }
  };

  return (
    <div class="event-list">
      <div class="event-list-toolbar">
        <div class="event-list-toolbar-row">
          <input
            class="input event-list-search"
            type="text"
            placeholder="Search actions or paths..."
            value={search}
            onInput={(e) => onSearchChange((e.target as HTMLInputElement).value)}
          />
          <button
            class={`btn btn-ghost event-list-pause ${paused ? 'paused' : ''}`}
            onClick={onTogglePause}
            title={paused ? 'Resume' : 'Pause'}
          >
            {paused ? '\u25B6' : '\u275A\u275A'}
          </button>
          <button
            class="btn btn-ghost"
            onClick={onExportHAR}
            title="Export filtered events as HAR"
            disabled={events.length === 0}
          >
            {'\uD83D\uDCE5'} HAR
          </button>
          <button
            class="btn btn-ghost"
            onClick={onClear}
            title="Clear events"
          >
            \u2715
          </button>
        </div>
        <div class="event-list-toolbar-row">
          <select
            class="input event-list-filter"
            value={filters.service}
            onChange={(e) =>
              onFilterChange('service', (e.target as HTMLSelectElement).value)
            }
          >
            <option value="">All services</option>
            {serviceNames.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          <select
            class="input event-list-filter"
            value={filters.status}
            onChange={(e) =>
              onFilterChange('status', (e.target as HTMLSelectElement).value)
            }
          >
            <option value="">All statuses</option>
            <option value="2xx">2xx</option>
            <option value="3xx">3xx</option>
            <option value="4xx">4xx</option>
            <option value="5xx">5xx</option>
          </select>
          <span class={`event-list-connection ${connected || dataSource === 'polling' ? 'connected' : ''}`}>
            {connected ? 'Live (SSE)' : dataSource === 'polling' ? 'Live (polling)' : 'Connecting...'}
          </span>
        </div>
      </div>

      {/* Source filter chips */}
      {sourceCounts.size > 0 && (
        <div class="source-chips-row">
          <button
            class={`source-chip ${sourceFilter.size === 0 ? 'source-chip-active' : ''}`}
            onClick={onResetSourceFilter}
          >
            All
          </button>
          {[...sourceCounts.entries()]
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([source, count]) => {
              const color = getSourceColor(source);
              const active = sourceFilter.size === 0 || sourceFilter.has(source);
              return (
                <button
                  key={source}
                  class={`source-chip ${active ? 'source-chip-active' : ''}`}
                  style={{
                    '--chip-color': color,
                    opacity: active ? 1 : 0.4,
                  } as any}
                  onClick={() => onToggleSource(source)}
                  title={`${source}: ${count} events`}
                >
                  <span class="source-chip-dot" style={{ background: color }} />
                  <span class="source-chip-name">{source}</span>
                  <span class="source-chip-count">{count}</span>
                </button>
              );
            })}
        </div>
      )}

      <div class="event-list-body" ref={listRef} onScroll={handleScroll}>
        {events.length === 0 ? (
          <div class="event-list-empty">No events to display</div>
        ) : (
          events.map((evt) => (
            <EventRow
              key={evt.id}
              event={evt}
              selected={evt.id === selectedId}
              onClick={() => onSelect(evt.id)}
            />
          ))
        )}
      </div>
    </div>
  );
}
