---
title: Dashboards View
description: Custom metric dashboards with query DSL, widget grid, presets, and favorites
---

The Dashboards view provides Datadog-style custom dashboards for visualizing CloudMock metrics. Each dashboard is a grid of widgets that query metric data using a structured DSL, with configurable time windows and auto-refresh.

## Dashboard list

The landing page shows all dashboards -- preset and custom -- in a browsable list. Each entry shows the dashboard name, widget count, description, and category label.

### Label filtering

Filter dashboards by category using the label bar at the top:

| Label | Color | Scope |
|-------|-------|-------|
| **Infrastructure** | Teal | Service-level metrics, traffic, system resources |
| **Application** | Purple | Error tracking, status codes, app-specific metrics |
| **Performance** | Yellow | Latency percentiles, deep-dive analysis |
| **Security** | Pink | Auth failures, IAM denials, rate limiting |

Click a label to show only dashboards in that category. Custom dashboards always appear regardless of the active filter.

### Favorites and hiding

- **Favorite** -- Click the star icon to pin a dashboard to the top of the list. Favorites are sorted before non-favorites.
- **Hide** -- Click the eye icon on preset dashboards to remove them from the main list. Hidden dashboards are shown in a collapsible section at the bottom.

Preferences (favorites and hidden) are persisted to localStorage.

## Preset dashboards

CloudMock ships with five built-in preset dashboards:

| Preset | Label | Widgets | Purpose |
|--------|-------|---------|---------|
| **Service Overview** | Infrastructure | 6 | Request volume, P99 latency, error rate, CPU, memory |
| **Error Tracking** | Application | 4 | Error rate trend by service, 5xx/4xx counts |
| **Latency Deep Dive** | Performance | 5 | P50/P95/P99 latency, avg by service, max latency |
| **Traffic Overview** | Infrastructure | 4 | Total requests, rate, by-service breakdown, queue depth |
| **Security & IAM** | Security | 5 | Auth failures (401), forbidden (403), rate limited (429) |

Presets cannot be deleted, but they can be hidden from the list.

## Creating a custom dashboard

Click **+ New Dashboard** to create a blank dashboard. You are taken directly to the dashboard view where you can add widgets.

## Dashboard view

After selecting a dashboard, the toolbar shows:

### Time window

Select the visible time range: **15m**, **1h**, **6h**, **24h**, or **7d**. All widgets in the dashboard share the same time window.

### Refresh interval

Choose the auto-refresh rate from the dropdown: **Off**, **10s**, **30s**, **1m**, or **5m**. Each widget re-fetches its data at this interval.

### Adding widgets

Click **+ Add Widget** to open the widget editor overlay.

## Widget types

| Type | Description |
|------|-------------|
| **Timeseries** | Line chart plotting metric values over time. Supports groupBy for multi-series overlays. |
| **Single Stat** | Large number display showing the current aggregated value. Supports warning/critical thresholds for color coding. |
| **Gauge** | Radial gauge showing a percentage value against thresholds. |

## Query DSL

Each widget is driven by a metric query using the following syntax:

```
aggregation(metric){ key=value, ... } by group
```

### Examples

```
avg(http.request.duration){ service=api-gateway }
p99(http.request.duration){ service=api-gateway, method=GET } by service
count(http.request.count)
sum(http.request.count){ status=5xx } by service
```

### Aggregation functions

| Function | Description |
|----------|-------------|
| `avg` | Mean value |
| `sum` | Total sum |
| `min` | Minimum value |
| `max` | Maximum value |
| `count` | Event count |
| `p50` | 50th percentile (median) |
| `p95` | 95th percentile |
| `p99` | 99th percentile |

### Filters

Filters are key-value pairs inside `{ }`. Common filter keys include `service`, `status`, and `method`.

### Group by

Append `by field` to split the result into multiple series, one per unique value of the field. For example, `by service` produces one line per service on a timeseries chart.

## Widget grid

Widgets are arranged in a 12-column grid. Each widget has configurable column span (1-12) and row span. Widgets can be resized by dragging their edges. Click a widget to open the editor and modify its query, title, type, or thresholds.

## Persistence

Custom dashboards and widget configurations are stored in the browser's localStorage. Preset dashboards are always loaded fresh from the source code and cannot be modified.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/metrics` | Aggregate metrics used by metric queries |
| `GET` | `/api/metrics/timeline` | Time-series data for charting |
| `GET` | `/api/traces` | Trace data used for metric computation |
| `GET` | `/api/slo` | SLO window data for percentile calculations |
