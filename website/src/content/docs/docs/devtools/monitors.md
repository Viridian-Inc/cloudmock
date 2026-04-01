---
title: Monitors View
description: Alert monitors with thresholds, notification channels, muting, and alert history
---

The Monitors view lets you define alert monitors that watch CloudMock metrics and trigger when thresholds are breached. Each monitor targets a specific service and metric, with configurable warning and critical thresholds, evaluation windows, and notification channels.

## Monitor list

The left panel shows all configured monitors with filter tabs:

| Filter | Shows |
|--------|-------|
| **All** | Every monitor |
| **OK** | Monitors within normal thresholds |
| **Warning** | Monitors that have breached the warning threshold |
| **Alert** | Monitors in critical alert state (badge shows count) |
| **No Data** | Monitors that have not received data |
| **Muted** | Monitors that are muted |

Each monitor row displays a status icon, the monitor name, target service, metric type, the time since last check, and the current value.

## Creating a monitor

Click **+ New** to open the monitor creation form. Configure the following fields:

### Monitor settings

| Field | Description |
|-------|-------------|
| **Name** | A descriptive name (e.g., "API Gateway P99 Latency") |
| **Service** | The target service from the service list |
| **Metric** | The metric to monitor (see table below) |
| **Operator** | Comparison operator (`>`, `>=`, `<`, `<=`, `==`, `!=`) |
| **Critical Threshold** | The value at which the monitor enters alert state |
| **Warning Threshold** | Optional lower threshold that triggers a warning |
| **Evaluation Window** | How far back to look when evaluating: 1m, 5m, 15m, 30m, or 1h |
| **Notification Channels** | Which channels to notify on alert (see below) |

### Available metrics

| Metric | Description |
|--------|-------------|
| `p50` | P50 (median) latency |
| `p95` | 95th percentile latency |
| `p99` | 99th percentile latency |
| `error_rate` | Percentage of failed requests |
| `request_count` | Total request volume |
| `avg_latency` | Mean response time |

### Notification channels

Select one or more channels to receive alerts:

- **Email** -- Send alert notifications via email.
- **Slack** -- Post alerts to a Slack channel.
- **PagerDuty** -- Create PagerDuty incidents.
- **Webhook** -- Send alert payloads to a custom URL.

## Monitor detail

Select a monitor from the list to view its details in the right panel:

- **Status** -- Current state (ok, warning, alert, no data, muted) with color-coded badge.
- **Service** -- Target service name.
- **Metric** -- Which metric is being monitored.
- **Condition** -- The full threshold expression (e.g., `p99 > 500 (warn: 200)`).
- **Window** -- Evaluation window duration.
- **Channels** -- Configured notification channels.
- **Last Value** -- The most recent measured value.
- **Last Checked** -- When the monitor was last evaluated.

### Actions

| Action | Description |
|--------|-------------|
| **Mute / Unmute** | Silence a monitor without deleting it. Muted monitors do not trigger notifications. |
| **Edit** | Open the monitor form to modify settings. |
| **Delete** | Remove the monitor permanently. |

### Alert history

The detail panel includes an **Alert History** table showing every past alert event:

| Column | Description |
|--------|-------------|
| Status | The alert status when the event fired (warning or alert) |
| Value | The measured value that triggered the alert |
| Threshold | The threshold that was breached |
| Time | How long ago the event occurred |
| Message | Human-readable description (e.g., "P95 latency exceeded critical threshold: 1240ms > 1000ms") |

## Persistence

Monitors are stored in the browser's localStorage under the key `neureaux:monitors`. On first load, demo monitors are generated to illustrate the feature. These can be modified or deleted.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/services` | List services for the service selector dropdown |
| `GET` | `/api/metrics` | Metric data used for monitor evaluation |
| `GET` | `/api/slo` | SLO data used for percentile calculations |
