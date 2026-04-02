import { useState, useRef, useCallback } from 'preact/hooks';
import type { Widget, QueryResult } from './types';
import { TimeseriesWidget } from './widgets/timeseries';
import { SingleStatWidget } from './widgets/single-stat';
import { GaugeWidget } from './widgets/gauge';
import { TableWidget } from './widgets/table-widget';
import { HeatmapWidget } from './widgets/heatmap';

interface DashboardGridProps {
  widgets: Widget[];
  queryResults: Map<string, QueryResult | null>;
  onWidgetEdit: (widgetId: string) => void;
  onWidgetResize: (widgetId: string, colSpan: number, rowSpan: number) => void;
}

const WIDGET_COLORS = [
  'var(--brand-teal, #4AE5F8)',
  '#a78bfa',
  '#f472b6',
  '#fbbf24',
  '#60a5fa',
  '#4ade80',
];

function renderWidget(
  widget: Widget,
  data: QueryResult | null,
  colorIdx: number,
) {
  switch (widget.type) {
    case 'timeseries':
      return (
        <TimeseriesWidget
          title={widget.title}
          data={data}
          unit={widget.unit}
          color={WIDGET_COLORS[colorIdx % WIDGET_COLORS.length]}
          height={widget.rowSpan * 140}
        />
      );
    case 'single-stat':
      return (
        <SingleStatWidget
          title={widget.title}
          data={data}
          unit={widget.unit}
          thresholds={widget.thresholds}
        />
      );
    case 'gauge':
      return (
        <GaugeWidget
          title={widget.title}
          data={data}
          unit={widget.unit}
          thresholds={widget.thresholds}
        />
      );
    case 'table':
      return (
        <TableWidget
          title={widget.title}
          data={data}
          unit={widget.unit}
        />
      );
    case 'heatmap':
      return (
        <HeatmapWidget
          title={widget.title}
          data={data}
          unit={widget.unit}
          height={widget.rowSpan * 140}
        />
      );
    default:
      return <div class="dashboard-widget-unknown">Unknown widget type</div>;
  }
}

export function DashboardGrid({
  widgets,
  queryResults,
  onWidgetEdit,
  onWidgetResize,
}: DashboardGridProps) {
  const [resizing, setResizing] = useState<string | null>(null);
  const startRef = useRef<{ x: number; y: number; colSpan: number; rowSpan: number } | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  const handleResizeStart = useCallback(
    (widgetId: string, colSpan: number, rowSpan: number) =>
      (e: PointerEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setResizing(widgetId);
        startRef.current = { x: e.clientX, y: e.clientY, colSpan, rowSpan };

        const handleMove = (me: PointerEvent) => {
          if (!startRef.current || !gridRef.current) return;
          const gridRect = gridRef.current.getBoundingClientRect();
          const colWidth = gridRect.width / 12;
          const rowHeight = 160;

          const dx = me.clientX - startRef.current.x;
          const dy = me.clientY - startRef.current.y;

          const newColSpan = Math.max(
            2,
            Math.min(12, startRef.current.colSpan + Math.round(dx / colWidth)),
          );
          const newRowSpan = Math.max(
            1,
            Math.min(4, startRef.current.rowSpan + Math.round(dy / rowHeight)),
          );

          onWidgetResize(widgetId, newColSpan, newRowSpan);
        };

        const handleUp = () => {
          setResizing(null);
          window.removeEventListener('pointermove', handleMove);
          window.removeEventListener('pointerup', handleUp);
        };

        window.addEventListener('pointermove', handleMove);
        window.addEventListener('pointerup', handleUp);
      },
    [onWidgetResize],
  );

  if (widgets.length === 0) {
    return (
      <div class="dashboard-grid-empty">
        <div class="dashboard-grid-empty-text">
          No widgets yet. Click "Add Widget" to get started.
        </div>
      </div>
    );
  }

  return (
    <div
      ref={gridRef}
      class="dashboard-grid"
      style={resizing ? { userSelect: 'none' } : undefined}
    >
      {widgets.map((widget, idx) => {
        const data = queryResults.get(widget.id) ?? null;

        return (
          <div
            key={widget.id}
            class={`dashboard-grid-cell ${resizing === widget.id ? 'resizing' : ''}`}
            style={{
              gridColumn: `span ${widget.colSpan}`,
              gridRow: `span ${widget.rowSpan}`,
            }}
          >
            <button
              class="dashboard-grid-cell-edit"
              onClick={() => onWidgetEdit(widget.id)}
              title="Edit widget"
            >
              ...
            </button>
            {renderWidget(widget, data, idx)}
            <div
              class="dashboard-grid-resize-handle"
              onPointerDown={handleResizeStart(widget.id, widget.colSpan, widget.rowSpan)}
            />
          </div>
        );
      })}
    </div>
  );
}
