---
title: Traces View
description: Distributed tracing with waterfall, flamegraph, and trace comparison
---

The Traces view provides distributed tracing for every request that flows through CloudMock. Each trace captures the full lifecycle of a request, from the initial gateway hit through IAM evaluation, service routing, and the service handler.

## Trace list

The left panel shows all recent traces, sorted by start time (newest first). Each row displays:

| Column | Description |
|--------|-------------|
| Time | Wall clock time when the trace started |
| Service | The root service that initiated the trace |
| Status | HTTP status code, color-coded (green/yellow/red) |
| Method + Path | HTTP method and URL path of the root span |
| Duration | Total trace duration in milliseconds |
| Spans | Number of spans in the trace |
| Error | Error indicator if any span in the trace failed |

### Filtering

Use the search box to filter traces by service name, URL path, or trace ID. The filter is case-insensitive and matches substrings.

## Waterfall view

Select a trace to open the **Waterfall** view in the right panel. The waterfall displays every span in the trace as a horizontal bar, aligned to a shared time axis:

- **Span hierarchy** -- Spans are indented to show parent-child relationships. The root span is at the top, with child spans (IAM check, routing, service handler) nested below.
- **Timing bars** -- Each bar's width is proportional to the span's duration. The bar is positioned on the time axis relative to the trace start.
- **Critical path** -- The longest sequential chain of spans is highlighted, showing which operations determined the overall trace duration.
- **Span detail** -- Click any span to see its attributes: service name, operation, start time, duration, status code, and any error messages.

## Flamegraph view

Toggle to **Flamegraph** mode using the view switcher at the top of the detail panel. The flamegraph stacks spans vertically by depth, with wider bars indicating longer durations. This view is useful for identifying which service or operation consumed the most time.

## Trace comparison

Click the **Compare** button in the trace list toolbar to enter comparison mode:

1. Select **Trace A** by clicking a trace in the list.
2. Select **Trace B** by clicking a second trace. The traces are labeled A and B in the list.
3. The detail panel switches to a **side-by-side comparison** showing both waterfalls with a diff overlay.

This is useful for comparing the same operation before and after a code change, or for diagnosing why one request was slower than another.

You can also compare a trace against its **route baseline** using the API:

```bash
curl "http://localhost:4599/api/traces/compare?a=TRACE_ID&baseline=true"
```

This returns a comparison of the trace against the average timing for its route (service + action combination).

## Cross-view navigation

Other views link into the Traces view:

- From the **Activity** view, click the trace link in the request detail panel. The URL hash is set to `#trace=TRACE_ID`, and the Traces view auto-selects that trace.
- From the **Topology** node inspector, the "Recent Traces" section links to traces involving the selected service.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/traces` | List recent traces (filter by service, error, limit) |
| `GET` | `/api/traces/{id}` | Get a single trace with all spans |
| `GET` | `/api/traces/compare?a=ID_A&b=ID_B` | Compare two traces |
| `GET` | `/api/traces/compare?a=ID&baseline=true` | Compare a trace against its route baseline |
