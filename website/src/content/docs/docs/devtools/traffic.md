---
title: Traffic View
description: Record, replay, and compare traffic at variable speeds
---

The Traffic view lets you record live API traffic flowing through CloudMock, replay it at variable speeds, and compare replay runs. This is useful for regression testing, load testing, and reproducing production-like traffic patterns against your local environment.

## Tabs

The view is organized into four tabs:

| Tab | Purpose |
|-----|---------|
| **Record** | Capture live traffic into named recordings |
| **Replay** | Play back a recording at configurable speed |
| **History** | View past replay runs with metrics |
| **Compare** | Compare two replay runs side by side |

## Recording traffic

On the **Record** tab:

1. Enter a **recording name** (e.g., "checkout-flow", "peak-traffic-wednesday").
2. Select a **duration**: 30 seconds, 1 minute, 5 minutes, or 10 minutes.
3. Click **Start Recording**.

While recording is active, a pulsing indicator shows the recording name and a **Stop** button to end early. CloudMock captures every API request that passes through the gateway during the recording window via `POST /api/traffic/record`.

### Saved recordings

Below the recording form, a list of saved recordings shows:

- **Name** -- The recording identifier.
- **Request count** -- How many requests were captured.
- **Duration** -- Recording length in seconds.
- **Replay** button -- Jump to the Replay tab with this recording selected.
- **Delete** button -- Remove the recording via `DELETE /api/traffic/recordings/{id}`.

## Replaying traffic

On the **Replay** tab:

1. **Select a recording** from the dropdown.
2. **Choose a speed multiplier**: 1x (real-time), 2x, 5x, or 10x.
3. Click **Start Replay**.

The replay sends each captured request back through CloudMock at the selected speed. A progress bar shows how many requests have been sent, and live stats update during the replay:

| Stat | Description |
|------|-------------|
| Sent / Total | Number of requests replayed out of total |
| Errors | Count of failed replay requests |
| P99 | 99th percentile latency of replayed requests |
| Status | Current replay state (running, completed, cancelled, failed) |

The replay is managed server-side. The view polls `GET /api/traffic/replay/{id}` every second to update progress.

## Replay history

The **History** tab shows a table of all past replay runs:

| Column | Description |
|--------|-------------|
| Recording | Name of the recording that was replayed |
| Speed | Replay speed multiplier (1x, 2x, etc.) |
| Sent | Requests sent vs. total |
| Errors | Error count (highlighted in red if > 0) |
| P99 | 99th percentile latency |
| Status | Completed, cancelled, or failed |

## Comparing runs

The **Compare** tab allows you to select two runs from the History tab and compare their latency distributions side by side. This is useful for comparing performance before and after a code change or configuration update.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/traffic/recordings` | List all saved recordings |
| `POST` | `/api/traffic/record` | Start a new recording (body: `{name, duration_sec}`) |
| `POST` | `/api/traffic/record/stop` | Stop the current recording early |
| `DELETE` | `/api/traffic/recordings/{id}` | Delete a saved recording |
| `POST` | `/api/traffic/replay` | Start a replay (body: `{recording_id, speed}`) |
| `GET` | `/api/traffic/replay/{id}` | Get replay run status and progress |
| `GET` | `/api/traffic/runs` | List all past replay runs |
