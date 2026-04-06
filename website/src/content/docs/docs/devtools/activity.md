---
title: Activity View
description: Real-time stream of every AWS API request flowing through CloudMock
---

The Activity view is a live feed of every AWS API call processed by the CloudMock gateway. Each request appears as a row showing the service, action, HTTP method, path, status code, latency, and timestamp.

## Data sources

The Activity view receives data through two parallel channels:

### SSE (Server-Sent Events)

The primary data source is a persistent SSE connection to `GET /api/stream`. Each event is a JSON object containing the request details. In development mode (port 1420), the devtools connect directly to the admin API on port 4599. In production mode (port 4500), the SSE endpoint is served from the same origin.

### Polling fallback

A secondary polling loop fetches `GET /api/requests?level=all&limit=200` every 3 seconds. This ensures data is available even if SSE fails to connect (e.g., behind a reverse proxy that does not support streaming). New requests are merged with the existing list, deduplicating by ID.

The status bar shows which data source is active: **SSE** or **Polling**.

## Request list

The left panel shows the filtered list of requests, most recent first. Each row displays:

| Column | Description |
|--------|-------------|
| Timestamp | When the request was processed |
| Service | AWS service name (s3, dynamodb, sqs, etc.) |
| Action | AWS API action (PutObject, GetItem, SendMessage, etc.) |
| Status | HTTP status code, color-coded: green for 2xx, yellow for 4xx, red for 5xx |
| Latency | Response time in milliseconds |
| Method + Path | HTTP method and URL path |

The buffer holds up to 2,000 events. Older events are discarded as new ones arrive.

## Filters and search

### Text search

Type in the search box to filter by action name, path, or service name. The search is case-insensitive and matches any substring.

### Service filter

Use the service dropdown to show only requests for a specific AWS service. When navigating from another view (e.g., clicking "View Activity" from the Topology node inspector), the service filter is set automatically.

The active filter is reflected in the URL hash (`#service=dynamodb`), so you can bookmark or share filtered views.

### Status filter

Filter by HTTP status code range: 2xx, 4xx, or 5xx.

### Source filter

When multiple services are sending traffic, use the source filter sidebar to toggle visibility of individual services. The sidebar shows per-service request counts.

## Pause and pin

- **Pause** -- Stops adding new events to the list. SSE events are still received but not displayed until you unpause. Useful for inspecting a specific moment without the list scrolling.

- **Pin** -- Freezes the current event list entirely. New events are dropped (not buffered). Use this when you want to preserve the exact set of events for analysis or export.

## Request detail panel

Click any request in the list to open the detail panel on the right. The detail view shows:

- **Request headers** -- Full HTTP headers including the AWS authorization header, content type, and X-Amz-Target.
- **Response body** -- The raw response returned by CloudMock, formatted as JSON or XML.
- **Trace link** -- If the request has an associated trace ID, click to jump directly to the Traces view.
- **Replay** -- Re-send the captured request against the gateway via `POST /api/requests/{id}/replay`.

## Export as HAR

Click the **Export HAR** button to download the current request list as an [HTTP Archive (HAR)](https://w3c.github.io/web-performance/specs/HAR/Overview.html) file. The HAR file can be imported into browser developer tools, Charles Proxy, or other HTTP debugging tools for further analysis.

The export includes all currently buffered events (up to 2,000), with request method, URL, headers, status code, and timing information.

## Clear

The **Clear** button empties the request buffer and resets all filters. This is useful when starting a new test session.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/requests` | List recent requests with filtering (service, action, status, latency, caller) |
| `GET` | `/api/requests/{id}` | Get a single request by ID |
| `POST` | `/api/requests/{id}/replay` | Replay a captured request |
| `GET` | `/api/stream` | SSE stream of requests in real time |
| `GET` | `/api/explain/{requestId}` | AI-ready context for a request |
