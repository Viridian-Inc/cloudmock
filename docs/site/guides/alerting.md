# Alerting

CloudMock includes a monitoring engine that evaluates rules on a configurable interval and fires alerts when thresholds are breached. Alerts can be routed to webhooks, Slack, PagerDuty, or email.

## Concepts

- **Monitor** -- a rule that evaluates periodically (e.g., "error rate > 5% for 5 minutes")
- **Alert** -- fired when a monitor's condition is met
- **Incident** -- alerts grouped by service/time window into actionable incidents
- **Webhook** -- an HTTP endpoint that receives alert notifications

## Creating Monitors

### Threshold Monitor

Alert when a metric exceeds a threshold:

```bash
curl -X POST http://localhost:4599/api/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High error rate",
    "type": "threshold",
    "query": "error_rate",
    "filters": {"service": "order-service"},
    "threshold": 0.05,
    "comparison": "above",
    "window": "5m",
    "severity": "critical"
  }'
```

### SLO Monitor

Alert when an SLO is at risk:

```bash
curl -X POST http://localhost:4599/api/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DynamoDB P99 SLO breach",
    "type": "slo",
    "service": "dynamodb",
    "action": "Query",
    "slo_target": "p99_ms",
    "threshold": 500,
    "window": "15m",
    "severity": "warning"
  }'
```

### Anomaly Monitor

Alert on unusual patterns (requires regression detection enabled):

```bash
curl -X POST http://localhost:4599/api/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Latency anomaly",
    "type": "anomaly",
    "query": "latency_p95",
    "filters": {"service": "*"},
    "sensitivity": "medium",
    "severity": "warning"
  }'
```

## Managing Monitors

```bash
# List all monitors
curl http://localhost:4599/api/monitors | jq '.'

# Get a specific monitor
curl http://localhost:4599/api/monitors/mon_abc123 | jq '.'

# Update a monitor
curl -X PUT http://localhost:4599/api/monitors/mon_abc123 \
  -H "Content-Type: application/json" \
  -d '{"threshold": 0.1, "severity": "warning"}'

# Delete a monitor
curl -X DELETE http://localhost:4599/api/monitors/mon_abc123
```

## Viewing Alerts

```bash
# List recent alerts
curl http://localhost:4599/api/alerts | jq '.'

# Filter by severity
curl "http://localhost:4599/api/alerts?severity=critical" | jq '.'

# Acknowledge an alert
curl -X PUT http://localhost:4599/api/alerts/alert_xyz \
  -H "Content-Type: application/json" \
  -d '{"status": "acknowledged"}'

# Resolve an alert
curl -X PUT http://localhost:4599/api/alerts/alert_xyz \
  -H "Content-Type: application/json" \
  -d '{"status": "resolved"}'
```

## Webhook Integration

Route alerts to any HTTP endpoint:

```bash
curl -X POST http://localhost:4599/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack alerts",
    "url": "https://hooks.slack.com/services/T.../B.../xxx",
    "events": ["alert.fired", "alert.resolved", "incident.created"],
    "filters": {"severity": ["critical", "warning"]}
  }'
```

### Webhook Payload

```json
{
  "event": "alert.fired",
  "alert": {
    "id": "alert_xyz",
    "monitor_name": "High error rate",
    "severity": "critical",
    "message": "Error rate exceeded 5% for order-service (current: 8.2%)",
    "fired_at": "2026-03-31T14:23:00Z",
    "service": "order-service",
    "value": 0.082,
    "threshold": 0.05
  }
}
```

## Managing Webhooks

```bash
# List webhooks
curl http://localhost:4599/api/webhooks | jq '.'

# Update a webhook
curl -X PUT http://localhost:4599/api/webhooks/wh_123 \
  -H "Content-Type: application/json" \
  -d '{"url": "https://new-url.example.com/webhook"}'

# Delete a webhook
curl -X DELETE http://localhost:4599/api/webhooks/wh_123
```

## Incidents

Related alerts are grouped into incidents based on time window and service:

```bash
# List incidents
curl http://localhost:4599/api/incidents | jq '.'

# Get incident details
curl http://localhost:4599/api/incidents/inc_456 | jq '.'
```

Incident response:

```json
{
  "id": "inc_456",
  "title": "High error rate in order-service",
  "status": "active",
  "severity": "critical",
  "created_at": "2026-03-31T14:23:00Z",
  "alerts": ["alert_xyz", "alert_abc"],
  "services": ["order-service"],
  "timeline": [
    {"time": "2026-03-31T14:23:00Z", "event": "Incident created"},
    {"time": "2026-03-31T14:23:05Z", "event": "Alert fired: High error rate (8.2%)"},
    {"time": "2026-03-31T14:25:00Z", "event": "Alert fired: P99 latency breach (1200ms)"}
  ]
}
```

## Configuration

```yaml
# .cloudmock.yaml
monitor:
  enabled: true
  eval_interval: 30s   # How often monitors are evaluated

incidents:
  enabled: true
  group_window: 5m     # Time window for grouping alerts into incidents
```

## SLO Configuration

Define SLOs globally or per-service:

```yaml
slo:
  enabled: true
  rules:
    - service: "*"
      action: "*"
      p50_ms: 50
      p95_ms: 200
      p99_ms: 500
      error_rate: 0.01

    - service: dynamodb
      action: Query
      p50_ms: 10
      p95_ms: 50
      p99_ms: 100
      error_rate: 0.001
```

View SLO status:

```bash
curl http://localhost:4599/api/slo | jq '.'
```
