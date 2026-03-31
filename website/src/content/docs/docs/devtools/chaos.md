---
title: Chaos Engineering
description: Inject latency, errors, and throttling into CloudMock services
---

The Chaos view lets you inject faults into any AWS service emulated by CloudMock. This is useful for testing how your application handles slow responses, error codes, and rate limiting without modifying your application code.

## Fault types

CloudMock supports three types of fault injection:

| Type | Effect | Value |
|------|--------|-------|
| **Latency** | Adds artificial delay before responding | Milliseconds to add (e.g., 2000 for 2 seconds) |
| **Error** | Returns an HTTP error status code instead of the real response | HTTP status code (e.g., 503 for Service Unavailable) |
| **Throttle** | Randomly fails a percentage of requests with a throttling error | Percentage of requests to fail (e.g., 50 for 50%) |

## Creating rules

Each chaos rule targets a specific service and optionally a specific action:

- **Service** (required) -- The AWS service name to target (e.g., `s3`, `dynamodb`, `sqs`). Use `*` to target all services.
- **Action** (optional) -- A specific API action to target (e.g., `GetObject`, `PutItem`). If omitted, the rule applies to all actions for that service.
- **Fault type** -- Latency, Error, or Throttle.
- **Value** -- The magnitude of the fault (milliseconds, status code, or percentage).

### Using the form

1. Enter the service name in the **Service** field.
2. Optionally enter an action name in the **Action** field.
3. Select the fault type from the dropdown.
4. Enter the value.
5. Click **Add** (or press Enter).
6. Toggle the **Active** switch to enable chaos injection.
7. Click **Apply Rules** to send the configuration to CloudMock.

Rules are not active until you both toggle the Active switch on and click Apply Rules. This two-step process prevents accidental fault injection.

## Presets

The Chaos view includes five built-in presets for common failure scenarios:

| Preset | Effect |
|--------|--------|
| **Slow Database** | DynamoDB + 2 second latency |
| **Auth Failure** | Cognito returns HTTP 403 |
| **Queue Backlog** | SQS + 5 second latency |
| **Network Partition** | All services return HTTP 503 |
| **Lambda Timeout** | Lambda + 30 second latency |

Click a preset to add its rules to the current rule list. You can combine multiple presets or mix presets with custom rules.

## Scheduled auto-disable

To prevent chaos rules from being left active accidentally, you can set a **duration** before applying:

| Duration | Behavior |
|----------|----------|
| Indefinite | Rules stay active until manually disabled |
| 1 min | Auto-disable after 1 minute |
| 5 min | Auto-disable after 5 minutes |
| 15 min | Auto-disable after 15 minutes |
| 30 min | Auto-disable after 30 minutes |
| 1 hour | Auto-disable after 1 hour |

When a duration is set, a countdown banner appears showing the remaining time. You can click **Stop early** to disable chaos before the timer expires.

When the timer reaches zero, the devtools automatically send a disable request to CloudMock, clearing all active rules.

## Programmatic control

You can manage chaos rules through the admin API without the devtools UI:

### List current rules

```bash
curl http://localhost:4599/api/chaos
```

### Create rules

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "active": true,
    "rules": [
      {"service": "dynamodb", "type": "latency", "value": 2000},
      {"service": "s3", "action": "GetObject", "type": "error", "value": 500}
    ]
  }'
```

### Disable all rules

```bash
curl -X DELETE http://localhost:4599/api/chaos
```

### Update a specific rule

```bash
curl -X PUT http://localhost:4599/api/chaos/RULE_ID \
  -H "Content-Type: application/json" \
  -d '{"service": "dynamodb", "type": "latency", "value": 5000}'
```

### Delete a specific rule

```bash
curl -X DELETE http://localhost:4599/api/chaos/RULE_ID
```

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/chaos` | List chaos rules and active status |
| `POST` | `/api/chaos` | Create/update chaos rules (body: `{active, rules}`) |
| `DELETE` | `/api/chaos` | Disable all chaos rules |
| `PUT` | `/api/chaos/{id}` | Update a single rule |
| `DELETE` | `/api/chaos/{id}` | Delete a single rule |
