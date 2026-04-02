import { useState, useEffect } from 'preact/hooks';
import type { Widget, WidgetType, AggregationFn, MetricQuery } from './types';
import { formatQuery, parseQuery, validateQuery } from './query';

interface WidgetEditorProps {
  widget?: Widget;
  onSave: (widget: Widget) => void;
  onCancel: () => void;
  onDelete?: (widgetId: string) => void;
}

const WIDGET_TYPES: { value: WidgetType; label: string }[] = [
  { value: 'timeseries', label: 'Time Series' },
  { value: 'single-stat', label: 'Single Stat' },
  { value: 'gauge', label: 'Gauge' },
  { value: 'table', label: 'Table' },
  { value: 'heatmap', label: 'Heatmap' },
];

const AGGREGATIONS: AggregationFn[] = [
  'avg', 'sum', 'min', 'max', 'count', 'p50', 'p95', 'p99',
];

const COMMON_METRICS = [
  'http.request.duration',
  'http.request.count',
  'http.error.rate',
  'system.cpu.usage',
  'system.memory.usage',
  'system.disk.io',
  'db.query.duration',
  'queue.message.count',
];

function newId(): string {
  return `w-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function WidgetEditor({ widget, onSave, onCancel, onDelete }: WidgetEditorProps) {
  const isNew = !widget;

  const [title, setTitle] = useState(widget?.title ?? '');
  const [type, setType] = useState<WidgetType>(widget?.type ?? 'timeseries');
  const [metric, setMetric] = useState(widget?.query.metric ?? 'http.request.duration');
  const [aggregation, setAggregation] = useState<AggregationFn>(widget?.query.aggregation ?? 'avg');
  const [filterText, setFilterText] = useState(
    widget ? Object.entries(widget.query.filters).map(([k, v]) => `${k}=${v}`).join(', ') : '',
  );
  const [groupBy, setGroupBy] = useState(widget?.query.groupBy ?? '');
  const [unit, setUnit] = useState(widget?.unit ?? 'ms');
  const [warningThreshold, setWarningThreshold] = useState(
    widget?.thresholds?.warning?.toString() ?? '',
  );
  const [criticalThreshold, setCriticalThreshold] = useState(
    widget?.thresholds?.critical?.toString() ?? '',
  );
  const [colSpan, setColSpan] = useState(widget?.colSpan ?? 6);
  const [errors, setErrors] = useState<string[]>([]);

  // Auto-generate title from query if title is empty
  useEffect(() => {
    if (!title || title === formatQuery(widget?.query ?? { metric: '', aggregation: 'avg', filters: {} })) {
      setTitle(`${aggregation}(${metric})`);
    }
    // Only run when metric/aggregation change, not title
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [metric, aggregation]);

  const handleSave = () => {
    // Parse filters
    const filters: Record<string, string> = {};
    if (filterText.trim()) {
      for (const part of filterText.split(',')) {
        const eqIdx = part.indexOf('=');
        if (eqIdx > 0) {
          filters[part.slice(0, eqIdx).trim()] = part.slice(eqIdx + 1).trim();
        }
      }
    }

    const query: MetricQuery = {
      metric,
      aggregation,
      filters,
      groupBy: groupBy || undefined,
    };

    const validationErrors = validateQuery(query);
    if (!title.trim()) validationErrors.push('Title is required');
    if (validationErrors.length > 0) {
      setErrors(validationErrors);
      return;
    }

    const thresholds =
      warningThreshold || criticalThreshold
        ? {
            warning: parseFloat(warningThreshold) || 0,
            critical: parseFloat(criticalThreshold) || 0,
          }
        : undefined;

    const saved: Widget = {
      id: widget?.id ?? newId(),
      title: title.trim(),
      type,
      query,
      col: widget?.col ?? 0,
      row: widget?.row ?? 0,
      colSpan,
      rowSpan: widget?.rowSpan ?? 1,
      unit: unit || undefined,
      thresholds,
    };

    onSave(saved);
  };

  const dslPreview = formatQuery({
    metric,
    aggregation,
    filters: Object.fromEntries(
      filterText
        .split(',')
        .map((p) => p.split('=').map((s) => s.trim()))
        .filter((p) => p.length === 2 && p[0]),
    ),
    groupBy: groupBy || undefined,
  });

  return (
    <div class="widget-editor">
      <div class="widget-editor-header">
        <h3 class="widget-editor-title">
          {isNew ? 'Add Widget' : 'Edit Widget'}
        </h3>
        <button class="widget-editor-close" onClick={onCancel}>
          x
        </button>
      </div>

      <div class="widget-editor-body">
        {errors.length > 0 && (
          <div class="widget-editor-errors">
            {errors.map((err, i) => (
              <div key={i} class="widget-editor-error">{err}</div>
            ))}
          </div>
        )}

        <div class="widget-editor-field">
          <label class="widget-editor-label">Title</label>
          <input
            class="widget-editor-input"
            type="text"
            value={title}
            onInput={(e) => setTitle((e.target as HTMLInputElement).value)}
            placeholder="Widget title"
          />
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Widget Type</label>
          <div class="widget-editor-type-group">
            {WIDGET_TYPES.map((wt) => (
              <button
                key={wt.value}
                class={`widget-editor-type-btn ${type === wt.value ? 'active' : ''}`}
                onClick={() => setType(wt.value)}
              >
                {wt.label}
              </button>
            ))}
          </div>
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Metric</label>
          <select
            class="widget-editor-select"
            value={metric}
            onChange={(e) => setMetric((e.target as HTMLSelectElement).value)}
          >
            {COMMON_METRICS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </select>
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Aggregation</label>
          <select
            class="widget-editor-select"
            value={aggregation}
            onChange={(e) => setAggregation((e.target as HTMLSelectElement).value as AggregationFn)}
          >
            {AGGREGATIONS.map((a) => (
              <option key={a} value={a}>{a}</option>
            ))}
          </select>
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Filters</label>
          <input
            class="widget-editor-input"
            type="text"
            value={filterText}
            onInput={(e) => setFilterText((e.target as HTMLInputElement).value)}
            placeholder="service=api, method=GET"
          />
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Group By</label>
          <input
            class="widget-editor-input"
            type="text"
            value={groupBy}
            onInput={(e) => setGroupBy((e.target as HTMLInputElement).value)}
            placeholder="e.g. service"
          />
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Unit</label>
          <select
            class="widget-editor-select"
            value={unit}
            onChange={(e) => setUnit((e.target as HTMLSelectElement).value)}
          >
            <option value="">None</option>
            <option value="ms">ms</option>
            <option value="%">%</option>
            <option value="req/s">req/s</option>
            <option value="bytes">bytes</option>
          </select>
        </div>

        <div class="widget-editor-field">
          <label class="widget-editor-label">Width (columns)</label>
          <input
            class="widget-editor-input"
            type="number"
            min="2"
            max="12"
            value={colSpan}
            onInput={(e) => setColSpan(parseInt((e.target as HTMLInputElement).value) || 6)}
          />
        </div>

        {(type === 'gauge' || type === 'single-stat') && (
          <>
            <div class="widget-editor-field">
              <label class="widget-editor-label">Warning Threshold</label>
              <input
                class="widget-editor-input"
                type="number"
                value={warningThreshold}
                onInput={(e) => setWarningThreshold((e.target as HTMLInputElement).value)}
                placeholder="e.g. 200"
              />
            </div>
            <div class="widget-editor-field">
              <label class="widget-editor-label">Critical Threshold</label>
              <input
                class="widget-editor-input"
                type="number"
                value={criticalThreshold}
                onInput={(e) => setCriticalThreshold((e.target as HTMLInputElement).value)}
                placeholder="e.g. 500"
              />
            </div>
          </>
        )}

        <div class="widget-editor-preview">
          <label class="widget-editor-label">Query Preview</label>
          <code class="widget-editor-dsl">{dslPreview}</code>
        </div>
      </div>

      <div class="widget-editor-footer">
        {!isNew && onDelete && (
          <button
            class="btn widget-editor-delete"
            onClick={() => onDelete(widget!.id)}
          >
            Delete
          </button>
        )}
        <div class="widget-editor-footer-right">
          <button class="btn" onClick={onCancel}>Cancel</button>
          <button class="btn btn-primary" onClick={handleSave}>
            {isNew ? 'Add' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
