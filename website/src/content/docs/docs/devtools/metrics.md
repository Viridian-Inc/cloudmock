---
title: Metrics View
description: Per-service request metrics, latency percentiles, error rates, and time-series charts
---

The Metrics view aggregates request data from traces and SLO windows into a dashboard showing request volume, latency distribution, and error rates across all AWS services handled by CloudMock.

## Summary cards

At the top of the view, three summary cards show aggregate metrics:

| Card | Description |
|------|-------------|
| **Total Requests** | Total number of traced requests across all services |
| **Avg Latency** | Mean response time across all requests |
| **Error Rate** | Percentage of requests that returned a 5xx status or had an error span. Color-coded: green (< 1%), yellow (1-5%), red (> 5%) |

## Per-service breakdown

Below the summary cards, a table lists every AWS service that has received traffic:

| Column | Description |
|--------|-------------|
| Service | AWS service name (s3, dynamodb, cognito-idp, etc.) |
| Requests | Total request count for this service |
| Avg Latency | Mean response time |
| P50 | Median latency (50th percentile) |
| P95 | 95th percentile latency |
| P99 | 99th percentile latency |
| Error Rate | Percentage of failed requests, color-coded by severity |

Services are sorted by request count, highest first.

### Percentile calculation

Latency percentiles are computed from two sources, with SLO windows taking priority:

1. **SLO windows** -- If SLO rules are configured, the SLO engine maintains sliding windows with pre-computed P50, P95, and P99 values. These are used when available.
2. **Trace durations** -- If no SLO window exists for a service, percentiles are computed directly from the trace duration list.

## Service comparison

Select two or more services using the checkboxes in the per-service table, then click **Compare**. The comparison view shows:

- **Side-by-side stat cards** for each selected service, with request count, avg latency, P99, and error rate.
- **Overlaid time-series charts** for request volume, P99 latency, and error rate, with each service plotted in a different color.

This is useful for comparing traffic patterns across services -- for example, verifying that your DynamoDB call volume tracks with your API Gateway request volume.

## Time-series charts

The bottom of the view shows three line charts computed from trace data bucketed by minute:

### Request volume

Requests per minute over time. Spikes indicate bursts of traffic; gaps indicate idle periods.

### P99 latency

The 99th percentile latency per minute. This chart surfaces latency outliers that the average might hide.

### Error rate

The percentage of errored requests per minute. A sudden jump here correlates with an incident or a faulty deployment.

## SLO integration

The Metrics view pulls data from the SLO engine (`GET /api/slo`). When SLO rules are defined, the per-service percentile columns reflect the SLO window calculations, which use a sliding time window rather than the full trace history. This gives more accurate recent-state percentiles.

To configure SLO rules, use the SLOs view or the admin API:

```bash
curl -X PUT http://localhost:4599/api/slo \
  -H "Content-Type: application/json" \
  -d '[{"service":"s3","latency_p99_ms":100,"error_rate_threshold":0.01}]'
```

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/metrics` | Aggregate metrics (request count, error rate, latency percentiles) |
| `GET` | `/api/metrics/timeline` | Time-series metrics for charting |
| `GET` | `/api/compare?service=X&action=Y` | Before/after comparison |
| `GET` | `/api/slo` | Current SLO status with per-service windows |
| `GET` | `/api/traces` | Trace data used for metric computation |
