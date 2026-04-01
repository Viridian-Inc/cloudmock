---
title: Incidents View
description: Incident management with timeline, acknowledge/resolve workflow, root cause analysis, and related traces
---

The Incidents view provides incident tracking and management for your CloudMock environment. It displays active and historical incidents in a split-panel layout with a 24-hour timeline, filterable list, and a detailed inspection panel with root cause analysis.

## Incident timeline

At the top of the view, a horizontal SVG timeline plots incidents from the last 24 hours. Incidents that occur within 15 minutes of each other are grouped into clusters:

- **Single dots** -- One incident at that time. The dot color reflects severity (red for critical, orange for high/warning, blue for info).
- **Cluster dots** -- Multiple incidents grouped together, shown as a larger dot with a count badge. The color reflects the highest severity in the group.
- **Tooltip** -- Hover over a dot to see the incident title and severity (or the count for clusters).

Click any dot to select that incident in the list below.

## Incident list

The left panel shows all incidents with filter tabs:

| Filter | Shows |
|--------|-------|
| **All** | Every incident |
| **Open** | Unacknowledged incidents (badge shows count) |
| **Ack** | Acknowledged but not yet resolved |
| **Resolved** | Closed incidents |

Each incident row displays:

- **Severity icon** -- `!!` for critical, `!` for warning, `i` for info.
- **Title** -- Short description of the incident.
- **Service** -- The affected AWS service.
- **Relative time** -- How long ago the incident occurred.
- **Status badge** -- Color-coded status (open, acknowledged, resolved).

## Incident detail

Click an incident to open the detail panel on the right side. The detail view shows:

### Header and actions

- **Title with severity icon** -- The incident title and severity level.
- **Acknowledge** button -- Available for open incidents. Marks the incident as acknowledged via `POST /api/incidents/{id}/acknowledge`.
- **Resolve** button -- Available for open or acknowledged incidents. Marks the incident as resolved via `POST /api/incidents/{id}/resolve`.

### Fields

| Field | Description |
|-------|-------------|
| Status | Current status badge (open, acknowledged, resolved) |
| Severity | Severity level (critical, warning, info) |
| Service | The affected service name |
| Created | Timestamp when the incident was created |
| Acknowledged | Timestamp when acknowledged (if applicable) |
| Resolved | Timestamp when resolved (if applicable) |

### Message

The full incident message with context about what triggered the incident.

### Details

If the incident includes structured details, they are rendered as formatted JSON.

### Root cause analysis

The **Root Cause Analysis** section uses the AI Debug endpoint (`POST /api/explain`) to generate a suggested root cause:

1. Click **Suggest Root Cause** to initiate analysis.
2. The view constructs a prompt from the incident context (title, severity, service, affected services, message, time window) and related traces.
3. The AI response is displayed inline as formatted text.
4. Click **Re-analyze** to regenerate, or **Open in AI Debug** to navigate to the AI Debug view for deeper investigation.

If the AI Debug endpoint is not configured, a message explains how to enable it.

### Related traces

The **Related Traces** section automatically queries `GET /api/traces` and filters for traces that match the incident's affected services within the incident time window (between `first_seen` and `last_seen`). Results are shown in a table with service, method, path, status code, and duration.

## Blast radius

Each incident includes an `affected_services` array that identifies the blast radius -- which services are impacted by this incident. This information is used by the root cause analysis and related traces features.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/incidents` | List all incidents |
| `POST` | `/api/incidents/{id}/acknowledge` | Acknowledge an incident |
| `POST` | `/api/incidents/{id}/resolve` | Resolve an incident |
| `GET` | `/api/traces` | Fetch traces for correlation |
| `POST` | `/api/explain` | AI-powered root cause analysis |
