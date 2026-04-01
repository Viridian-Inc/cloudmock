---
title: SLOs View
description: Service-level objective rules, compliance windows, error budgets, and alerts
---

The SLOs view lets you define service-level objectives for your CloudMock services and monitor compliance in real time. SLO rules set latency and error rate thresholds; CloudMock evaluates them over sliding time windows and reports violations and remaining error budget.

## Health status

A global health badge at the top of the view shows the overall SLO state:

- **All Healthy** (green) -- No SLO violations detected across any service.
- **Violations Detected** (red) -- One or more services are breaching their SLO thresholds.

## Alerts

When violations occur, alert banners appear below the header. Each alert shows:

- **Message** -- A description of the violation (e.g., "P99 latency exceeds 100ms for s3").
- **Severity** -- Warning or critical, color-coded accordingly.
- **Timestamp** -- When the alert was generated.

## SLO rules

The rules table lists every configured SLO rule:

| Column | Description |
|--------|-------------|
| Service | The AWS service name this rule applies to |
| Action | Specific action (or `*` for all actions) |
| P50 | Maximum allowed 50th percentile latency |
| P95 | Maximum allowed 95th percentile latency |
| P99 | Maximum allowed 99th percentile latency |
| Error Rate | Maximum allowed error rate (as a decimal, e.g., 0.01 for 1%) |

Each rule can be deleted by clicking the delete button in its row.

## Compliance windows

The compliance windows table shows the current state of each SLO evaluation window:

| Column | Description |
|--------|-------------|
| Status | Green dot if healthy, red if in violation |
| Service | AWS service name |
| Action | Specific action or `*` |
| Total | Total requests in the window |
| Errors | Error count in the window |
| P50 / P95 / P99 | Actual latency percentiles |
| Error Rate | Actual error rate percentage |
| Violations | Tags listing which thresholds are breached (e.g., "p99", "error_rate") |

Rows with violations are highlighted with a red background.

## Error budget

Below the compliance windows, the error budget section shows a visual gauge for each service:

- The gauge bar represents the remaining error budget as a percentage.
- **Green** (> 20% remaining) -- Healthy budget.
- **Yellow** (0-20% remaining) -- Budget running low.
- **Red** (0% or exhausted) -- Budget exhausted, marked with "Budget Exhausted".

The error budget is calculated as `(1 - actualErrorRate / allowedErrorRate) * 100`. A service consuming half its allowed error rate has 50% budget remaining.

## Adding rules

The **Add Rule** form at the bottom of the view lets you create new SLO rules:

1. **Service** -- Select from a dropdown populated by `GET /api/services`.
2. **P50 (ms)** -- Median latency threshold (default: 100ms).
3. **P95 (ms)** -- 95th percentile threshold (default: 500ms).
4. **P99 (ms)** -- 99th percentile threshold (default: 1000ms).
5. **Error Rate** -- Maximum error rate as a decimal (default: 0.01).

Click **Add Rule** to save. The rule is sent to CloudMock via `POST /api/slo` along with all existing rules, and the view refreshes to show updated compliance data.

## Programmatic configuration

You can manage SLO rules through the admin API:

### Set rules

```bash
curl -X POST http://localhost:4599/api/slo \
  -H "Content-Type: application/json" \
  -d '{
    "rules": [
      {"service": "s3", "action": "*", "p50_ms": 50, "p95_ms": 200, "p99_ms": 500, "error_rate": 0.01},
      {"service": "dynamodb", "action": "*", "p50_ms": 20, "p95_ms": 100, "p99_ms": 250, "error_rate": 0.005}
    ]
  }'
```

### Get current status

```bash
curl http://localhost:4599/api/slo
```

The response includes `rules`, `windows`, `alerts`, and an overall `healthy` boolean.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/slo` | Get current SLO status (rules, windows, alerts, health) |
| `POST` | `/api/slo` | Set SLO rules (replaces all existing rules) |
| `GET` | `/api/services` | List services for the rule creation dropdown |
