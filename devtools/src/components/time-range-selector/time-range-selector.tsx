import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import './time-range-selector.css';

export interface TimeRangeSelectorProps {
  /** Full data range (earliest to latest) */
  dataRange: { start: number; end: number };
  /** Currently selected view window */
  selectedRange: { start: number; end: number };
  /** Called when user drags handles or scrolls to zoom */
  onRangeChange: (range: { start: number; end: number }) => void;
  /** Optional: mini chart data points for the overview */
  overviewData?: { timestamp: number; value: number }[];
  /** Whether to auto-advance the window (live mode) */
  live?: boolean;
  /** Height in pixels */
  height?: number;
}

const MIN_ZOOM_MS = 10_000; // 10 seconds minimum
const HANDLE_WIDTH = 8;
const LABEL_COUNT = 6;

type DragMode = 'left' | 'right' | 'pan' | null;

/** Format a timestamp for axis labels, adapting to the visible range */
function formatAxisLabel(ts: number, rangeDuration: number): string {
  const d = new Date(ts);
  if (rangeDuration < 120_000) {
    // < 2 min: show HH:MM:SS
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }
  if (rangeDuration < 3_600_000) {
    // < 1 hour: show HH:MM
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  if (rangeDuration < 86_400_000) {
    // < 24 hours: show HH:MM
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  // > 24 hours: show Mon DD HH:MM
  return `${d.toLocaleDateString([], { month: 'short', day: 'numeric' })} ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
}

export function TimeRangeSelector({
  dataRange,
  selectedRange,
  onRangeChange,
  overviewData,
  live = false,
  height = 48,
}: TimeRangeSelectorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{
    mode: DragMode;
    startX: number;
    startRange: { start: number; end: number };
  } | null>(null);
  const liveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const totalDuration = Math.max(1, dataRange.end - dataRange.start);
  const selectedDuration = Math.max(1, selectedRange.end - selectedRange.start);

  // Convert timestamp to fraction (0-1) of the data range
  const toFraction = useCallback(
    (ts: number) => (ts - dataRange.start) / totalDuration,
    [dataRange.start, totalDuration],
  );

  // Convert fraction (0-1) to timestamp
  const toTimestamp = useCallback(
    (frac: number) => dataRange.start + frac * totalDuration,
    [dataRange.start, totalDuration],
  );

  // Convert a client X position to a fraction
  const clientXToFraction = useCallback(
    (clientX: number): number => {
      const el = containerRef.current;
      if (!el) return 0;
      const rect = el.getBoundingClientRect();
      return Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    },
    [],
  );

  // Selected range as fractions
  const leftFrac = toFraction(selectedRange.start);
  const rightFrac = toFraction(selectedRange.end);

  // Clamp and apply a new range
  const applyRange = useCallback(
    (start: number, end: number) => {
      // Enforce minimum zoom
      if (end - start < MIN_ZOOM_MS) {
        const mid = (start + end) / 2;
        start = mid - MIN_ZOOM_MS / 2;
        end = mid + MIN_ZOOM_MS / 2;
      }
      // Enforce maximum zoom (full data range)
      if (end - start > totalDuration) {
        start = dataRange.start;
        end = dataRange.end;
      }
      // Clamp to data range
      if (start < dataRange.start) {
        const shift = dataRange.start - start;
        start += shift;
        end += shift;
      }
      if (end > dataRange.end) {
        const shift = end - dataRange.end;
        start -= shift;
        end -= shift;
      }
      start = Math.max(dataRange.start, start);
      end = Math.min(dataRange.end, end);

      onRangeChange({ start, end });
    },
    [dataRange, totalDuration, onRangeChange],
  );

  // --- Drag handlers ---
  const handlePointerDown = useCallback(
    (e: PointerEvent, mode: DragMode) => {
      e.preventDefault();
      e.stopPropagation();
      const el = containerRef.current;
      if (!el) return;
      el.setPointerCapture(e.pointerId);
      dragRef.current = {
        mode,
        startX: e.clientX,
        startRange: { ...selectedRange },
      };
    },
    [selectedRange],
  );

  const handlePointerMove = useCallback(
    (e: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag || !drag.mode) return;

      const el = containerRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const deltaFrac = (e.clientX - drag.startX) / rect.width;
      const deltaMs = deltaFrac * totalDuration;

      if (drag.mode === 'left') {
        const newStart = drag.startRange.start + deltaMs;
        const maxStart = selectedRange.end - MIN_ZOOM_MS;
        applyRange(Math.min(newStart, maxStart), selectedRange.end);
      } else if (drag.mode === 'right') {
        const newEnd = drag.startRange.end + deltaMs;
        const minEnd = selectedRange.start + MIN_ZOOM_MS;
        applyRange(selectedRange.start, Math.max(newEnd, minEnd));
      } else if (drag.mode === 'pan') {
        const newStart = drag.startRange.start + deltaMs;
        const newEnd = drag.startRange.end + deltaMs;
        applyRange(newStart, newEnd);
      }
    },
    [totalDuration, selectedRange, applyRange],
  );

  const handlePointerUp = useCallback(() => {
    dragRef.current = null;
  }, []);

  // --- Wheel zoom ---
  const handleWheel = useCallback(
    (e: WheelEvent) => {
      e.preventDefault();
      const el = containerRef.current;
      if (!el) return;

      // Zoom centered on cursor position
      const cursorFrac = clientXToFraction(e.clientX);
      const cursorTs = toTimestamp(cursorFrac);

      // Zoom factor: scroll up = zoom in, scroll down = zoom out
      const zoomFactor = e.deltaY > 0 ? 1.15 : 0.87;
      const newDuration = selectedDuration * zoomFactor;

      // Distribute the new duration around the cursor position
      const cursorPositionInRange = (cursorTs - selectedRange.start) / selectedDuration;
      const newStart = cursorTs - cursorPositionInRange * newDuration;
      const newEnd = newStart + newDuration;

      applyRange(newStart, newEnd);
    },
    [clientXToFraction, toTimestamp, selectedDuration, selectedRange, applyRange],
  );

  // Attach wheel listener (passive: false needed to prevent scroll)
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    el.addEventListener('wheel', handleWheel, { passive: false });
    return () => el.removeEventListener('wheel', handleWheel);
  }, [handleWheel]);

  // --- Live mode auto-advance ---
  const durationRef = useRef(selectedDuration);
  durationRef.current = selectedDuration;
  const onRangeChangeRef = useRef(onRangeChange);
  onRangeChangeRef.current = onRangeChange;

  useEffect(() => {
    if (liveTimerRef.current) {
      clearInterval(liveTimerRef.current);
      liveTimerRef.current = null;
    }

    if (live) {
      liveTimerRef.current = setInterval(() => {
        const now = Date.now();
        onRangeChangeRef.current({ start: now - durationRef.current, end: now });
      }, 1000);
    }

    return () => {
      if (liveTimerRef.current) clearInterval(liveTimerRef.current);
    };
  }, [live]); // Only re-run when live mode toggles, not on every range change

  // --- Overview mini chart path ---
  const overviewPath = useMemo(() => {
    if (!overviewData || overviewData.length < 2) return null;

    const chartHeight = height - 20; // leave room for time labels
    const maxVal = Math.max(1e-9, ...overviewData.map((d) => d.value));

    const points = overviewData.map((d) => {
      const x = toFraction(d.timestamp) * 100;
      const y = chartHeight - (d.value / maxVal) * (chartHeight - 4);
      return { x, y };
    });

    const linePath = points
      .map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`)
      .join(' ');

    const areaPath = `${linePath} L${points[points.length - 1].x},${chartHeight} L${points[0].x},${chartHeight} Z`;

    return { linePath, areaPath, chartHeight };
  }, [overviewData, height, toFraction]);

  // --- Time axis labels ---
  const axisLabels = useMemo(() => {
    const labels: { frac: number; text: string }[] = [];
    for (let i = 0; i <= LABEL_COUNT; i++) {
      const frac = i / LABEL_COUNT;
      const ts = dataRange.start + frac * totalDuration;
      labels.push({ frac, text: formatAxisLabel(ts, totalDuration) });
    }
    return labels;
  }, [dataRange, totalDuration]);

  const chartHeight = height - 20;

  return (
    <div
      class="trs-container"
      ref={containerRef}
      style={{ height: `${height}px` }}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
    >
      <svg
        class="trs-svg"
        width="100%"
        height={chartHeight}
        viewBox={`0 0 100 ${chartHeight}`}
        preserveAspectRatio="none"
      >
        {/* Dimmed regions outside selection */}
        <rect
          x="0"
          y="0"
          width={`${leftFrac * 100}`}
          height={chartHeight}
          class="trs-dim"
        />
        <rect
          x={`${rightFrac * 100}`}
          y="0"
          width={`${(1 - rightFrac) * 100}`}
          height={chartHeight}
          class="trs-dim"
        />

        {/* Overview data area */}
        {overviewPath && (
          <>
            <path d={overviewPath.areaPath} class="trs-area" />
            <path
              d={overviewPath.linePath}
              class="trs-line"
              vector-effect="non-scaling-stroke"
            />
          </>
        )}

        {/* Selected region highlight border */}
        <rect
          x={`${leftFrac * 100}`}
          y="0"
          width={`${(rightFrac - leftFrac) * 100}`}
          height={chartHeight}
          class="trs-selected-region"
        />
      </svg>

      {/* Draggable left handle */}
      <div
        class="trs-handle trs-handle-left"
        style={{ left: `${leftFrac * 100}%` }}
        onPointerDown={(e) => handlePointerDown(e as unknown as PointerEvent, 'left')}
      >
        <div class="trs-handle-grip" />
      </div>

      {/* Pan region (middle) */}
      <div
        class="trs-pan-region"
        style={{
          left: `${leftFrac * 100}%`,
          width: `${(rightFrac - leftFrac) * 100}%`,
        }}
        onPointerDown={(e) => handlePointerDown(e as unknown as PointerEvent, 'pan')}
      />

      {/* Draggable right handle */}
      <div
        class="trs-handle trs-handle-right"
        style={{ left: `${rightFrac * 100}%` }}
        onPointerDown={(e) => handlePointerDown(e as unknown as PointerEvent, 'right')}
      >
        <div class="trs-handle-grip" />
      </div>

      {/* Live indicator */}
      {live && (
        <div class="trs-live-badge">
          <span class="trs-live-dot" />
          LIVE
        </div>
      )}

      {/* Time axis labels */}
      <div class="trs-axis" style={{ height: '20px' }}>
        {axisLabels.map((l, i) => (
          <span
            key={i}
            class="trs-axis-label"
            style={{ left: `${l.frac * 100}%` }}
          >
            {l.text}
          </span>
        ))}
      </div>
    </div>
  );
}
