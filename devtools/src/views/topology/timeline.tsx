import { useState, useMemo, useCallback, useRef } from 'preact/hooks';
import type { DeployEvent } from '../../lib/health';
import type { IncidentInfo } from '../../lib/types';

export interface TimelineEvent {
  type: 'deploy' | 'incident';
  data: DeployEvent | IncidentInfo;
}

interface TimelineProps {
  deploys: DeployEvent[];
  incidents: IncidentInfo[];
  onSelectEvent: (event: TimelineEvent) => void;
  timeRange: { start: number; end: number };
  /** Current playhead timestamp (null = live / at "now") */
  playheadTime: number | null;
  /** Called when user drags playhead to a new timestamp */
  onPlayheadChange: (timestamp: number | null) => void;
  /** Whether we are in live mode (playhead is hidden / at "now") */
  isLive: boolean;
}

interface PositionedEvent {
  x: number;
  type: 'deploy' | 'incident';
  data: DeployEvent | IncidentInfo;
  label: string;
  time: number;
}

function formatTimeLabel(ts: number): string {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatPlayheadLabel(ts: number): string {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function relativeTimeShort(ts: number): string {
  const diff = Date.now() - ts;
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

const TIMELINE_HEIGHT = 48;
const PADDING_LEFT = 40;
const PADDING_RIGHT = 40;
const DOT_RADIUS = 5;

export function Timeline({
  deploys,
  incidents,
  onSelectEvent,
  timeRange,
  playheadTime,
  onPlayheadChange,
  isLive,
}: TimelineProps) {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);

  const duration = timeRange.end - timeRange.start;

  const events: PositionedEvent[] = useMemo(() => {
    const all: PositionedEvent[] = [];

    for (const d of deploys) {
      const ts = new Date(d.timestamp).getTime();
      if (ts < timeRange.start || ts > timeRange.end) continue;
      const fraction = (ts - timeRange.start) / duration;
      all.push({
        x: fraction,
        type: 'deploy',
        data: d,
        label: `Deploy: ${d.service} - ${d.message} (${relativeTimeShort(ts)})`,
        time: ts,
      });
    }

    for (const inc of incidents) {
      const ts = new Date(inc.first_seen).getTime();
      if (ts < timeRange.start || ts > timeRange.end) continue;
      const fraction = (ts - timeRange.start) / duration;
      all.push({
        x: fraction,
        type: 'incident',
        data: inc,
        label: `Incident: ${inc.title} [${inc.severity}] (${relativeTimeShort(ts)})`,
        time: ts,
      });
    }

    return all.sort((a, b) => a.time - b.time);
  }, [deploys, incidents, timeRange, duration]);

  const handleClick = useCallback((ev: PositionedEvent) => {
    onSelectEvent({ type: ev.type, data: ev.data });
  }, [onSelectEvent]);

  // Convert a mouse/pointer X position on the container to a timestamp
  const xToTimestamp = useCallback((clientX: number): number => {
    const el = containerRef.current;
    if (!el) return timeRange.end;
    const rect = el.getBoundingClientRect();
    const totalWidth = rect.width;
    const usableLeft = (PADDING_LEFT / 800) * totalWidth;
    const usableRight = totalWidth - (PADDING_RIGHT / 800) * totalWidth;
    const usableWidth = usableRight - usableLeft;
    const relX = clientX - rect.left - usableLeft;
    const fraction = Math.max(0, Math.min(1, relX / usableWidth));
    return timeRange.start + fraction * duration;
  }, [timeRange, duration]);

  // Playhead drag handlers
  const handlePointerDown = useCallback((e: PointerEvent) => {
    // Only handle clicks on the timeline background, not on event dots
    const target = e.target as HTMLElement;
    if (target.closest('.timeline-event')) return;

    isDragging.current = true;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    const ts = xToTimestamp(e.clientX);
    onPlayheadChange(ts);
  }, [xToTimestamp, onPlayheadChange]);

  const handlePointerMove = useCallback((e: PointerEvent) => {
    if (!isDragging.current) return;
    const ts = xToTimestamp(e.clientX);
    onPlayheadChange(ts);
  }, [xToTimestamp, onPlayheadChange]);

  const handlePointerUp = useCallback(() => {
    isDragging.current = false;
  }, []);

  // Playhead position as a fraction (0-1)
  const playheadFraction = useMemo(() => {
    if (playheadTime == null) return null;
    return Math.max(0, Math.min(1, (playheadTime - timeRange.start) / duration));
  }, [playheadTime, timeRange, duration]);

  // Compute shaded region: from window start to playhead (or end if live)
  const shadedEndFraction = playheadFraction ?? 1;

  // Generate time ticks
  const ticks = useMemo(() => {
    const count = 5;
    const result: { x: number; label: string }[] = [];
    for (let i = 0; i <= count; i++) {
      const fraction = i / count;
      const ts = timeRange.start + fraction * duration;
      result.push({ x: fraction, label: formatTimeLabel(ts) });
    }
    return result;
  }, [timeRange, duration]);

  return (
    <div
      class="timeline-bar"
      ref={containerRef}
      style={{ height: `${TIMELINE_HEIGHT}px`, cursor: 'crosshair' }}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
    >
      <svg width="100%" height={TIMELINE_HEIGHT} preserveAspectRatio="none">
        {/* Shaded time window region */}
        <rect
          x={`${(PADDING_LEFT / 800) * 100}%`}
          y="0"
          width={`${shadedEndFraction * (100 - (PADDING_LEFT / 800) * 100 - (PADDING_RIGHT / 800) * 100)}%`}
          height={TIMELINE_HEIGHT}
          fill="rgba(74, 229, 248, 0.04)"
          class="timeline-window-shade"
        />

        {/* Base line */}
        <line
          x1={`${(PADDING_LEFT / 800) * 100}%`}
          y1={TIMELINE_HEIGHT / 2}
          x2={`${100 - (PADDING_RIGHT / 800) * 100}%`}
          y2={TIMELINE_HEIGHT / 2}
          stroke="rgba(255,255,255,0.1)"
          stroke-width="1"
        />

        {/* Time ticks */}
        {ticks.map((tick, i) => {
          const pct = ((PADDING_LEFT + tick.x * (800 - PADDING_LEFT - PADDING_RIGHT)) / 800) * 100;
          return (
            <g key={i}>
              <line
                x1={`${pct}%`} y1={TIMELINE_HEIGHT / 2 - 4}
                x2={`${pct}%`} y2={TIMELINE_HEIGHT / 2 + 4}
                stroke="rgba(255,255,255,0.1)"
                stroke-width="1"
              />
              <text
                x={`${pct}%`} y={TIMELINE_HEIGHT - 2}
                fill="var(--text-tertiary, #5a6577)"
                font-size="8"
                text-anchor="middle"
                font-family="var(--font-mono, monospace)"
              >
                {tick.label}
              </text>
            </g>
          );
        })}

        {/* Playhead vertical line */}
        {playheadFraction != null && (
          (() => {
            const pct = ((PADDING_LEFT + playheadFraction * (800 - PADDING_LEFT - PADDING_RIGHT)) / 800) * 100;
            return (
              <g class="timeline-playhead-group">
                <line
                  x1={`${pct}%`} y1="0"
                  x2={`${pct}%`} y2={TIMELINE_HEIGHT}
                  stroke="var(--text-accent, #4AE5F8)"
                  stroke-width="1.5"
                  class="timeline-playhead-line"
                />
                {/* Playhead handle (diamond) */}
                <circle
                  cx={`${pct}%`}
                  cy="6"
                  r="4"
                  fill="var(--text-accent, #4AE5F8)"
                  class="timeline-playhead-handle"
                />
              </g>
            );
          })()
        )}

        {/* "now" label — only shown in live mode */}
        {isLive && (
          <text
            x={`${100 - (PADDING_RIGHT / 800) * 100}%`}
            y={12}
            fill="var(--text-tertiary, #5a6577)"
            font-size="9"
            text-anchor="end"
            font-family="var(--font-mono, monospace)"
          >
            now
          </text>
        )}
      </svg>

      {/* Playhead time label */}
      {playheadFraction != null && playheadTime != null && (
        (() => {
          const pct = ((PADDING_LEFT + playheadFraction * (800 - PADDING_LEFT - PADDING_RIGHT)) / 800) * 100;
          return (
            <div
              class="timeline-playhead-label"
              style={{ left: `${pct}%` }}
            >
              {formatPlayheadLabel(playheadTime)}
            </div>
          );
        })()
      )}

      {/* Event dots (HTML for easier hover/click) */}
      {events.map((ev, i) => {
        const leftPct = ((PADDING_LEFT + ev.x * (800 - PADDING_LEFT - PADDING_RIGHT)) / 800) * 100;
        const isHovered = hoveredIdx === i;
        const color = ev.type === 'deploy' ? '#3b82f6' : '#ef4444';

        return (
          <div
            key={i}
            class="timeline-event"
            style={{
              left: `${leftPct}%`,
              top: `${TIMELINE_HEIGHT / 2 - DOT_RADIUS}px`,
              width: `${DOT_RADIUS * 2}px`,
              height: `${DOT_RADIUS * 2}px`,
              background: color,
              transform: isHovered ? 'scale(1.4)' : 'scale(1)',
            }}
            onMouseEnter={() => setHoveredIdx(i)}
            onMouseLeave={() => setHoveredIdx(null)}
            onClick={(e) => { e.stopPropagation(); handleClick(ev); }}
          >
            {isHovered && (
              <div class="timeline-tooltip">
                {ev.label}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
