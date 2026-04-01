---
title: AI Debug View
description: AI-powered request explanation with streaming responses and recent error suggestions
---

The AI Debug view provides intelligent, AI-generated explanations of requests that flow through CloudMock. Paste a request ID or trace ID and get a narrative walkthrough of what happened across all services, including root cause identification and timeline reconstruction.

## How to use

1. Enter a **Request ID** or **Trace ID** in the input field.
2. Click **Explain** (or press Cmd/Ctrl+Enter).
3. The AI analyzes the request context and streams a narrative explanation.

## Request input

The input field accepts:

- **Request IDs** -- From the Activity view's request list.
- **Trace IDs** -- From the Traces view.
- **Error references** -- Any identifier that the explain endpoint can resolve.

While a request is being analyzed, the **Explain** button changes to **Cancel**, allowing you to abort a long-running analysis.

## Recent error suggestions

Below the input field, the view automatically loads the 5 most recent failed traces (status code >= 400) from the traces API, refreshing every 15 seconds. Each suggestion shows:

| Column | Description |
|--------|-------------|
| Status | HTTP status code badge (4xx in yellow, 5xx in red) |
| Service | The root service of the trace |
| Action | The API action or path |
| Time | Relative time since the error occurred |

Click any suggestion to populate the input field with its trace ID, then click Explain to analyze it.

## Streaming responses

The AI Debug endpoint supports two response formats:

### Server-Sent Events (SSE)

When the response content type is `text/event-stream`, the view reads SSE frames (`data: {...}`) and streams the explanation text incrementally. A blinking cursor indicates that streaming is in progress.

### Chunked text

When the response content type is `text/plain`, the view reads the response body as a stream of text chunks and renders them as they arrive.

### JSON fallback

If the response is standard JSON, the view reads the `explanation` field from the response body.

## Explanation rendering

The AI response is rendered with Markdown formatting:

- **Headings** (`#`, `##`, `###`) are styled as section headers.
- **Bold** (`**text**`) and **inline code** (`` `code` ``) are formatted.
- **Bullet lists** (`- item`) render as indented list items.
- **Fenced code blocks** (` ```language `) render in a monospace code block.

## Features

The AI Debug analysis provides:

- **Root cause identification** -- Pinpoints the service or operation that caused a failure.
- **Service dependency mapping** -- Shows how the request traversed service boundaries.
- **Timeline reconstruction** -- Reconstructs the chronological sequence of events across spans and services.

## Cross-view integration

The AI Debug view is linked from other views:

- From the **Incidents** view, the "Open in AI Debug" link navigates here with the incident ID pre-filled.
- From the **Activity** view, the explain link in the request detail panel provides the request ID context.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/explain` | Send a request ID and get an AI-generated explanation (supports streaming) |
| `GET` | `/api/explain/{requestId}` | Get AI-ready context for a request |
| `GET` | `/api/traces` | Fetch recent traces for error suggestions |
