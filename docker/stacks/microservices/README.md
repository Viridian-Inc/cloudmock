# Microservices Stack

Three services communicating via SNS fan-out to SQS queues.

## Architecture

```
POST /orders
     |
[order-service] --> DynamoDB (orders table)
                --> SNS (orders topic)
                        |
              +---------+---------+
              |                   |
        [SQS: payments]   [SQS: notifications]
              |                   |
    [payment-service]  [notification-service]
    (Python/FastAPI)      (Go)
         |
    DynamoDB (payments table)
```

## Services

| Service | Port | Language | Responsibility |
|---------|------|----------|----------------|
| CloudMock | 4566 | — | AWS API emulation |
| CloudMock DevTools | 4500 | — | Observability dashboard |
| order-service | 3001 | Node.js | Create orders, publish to SNS |
| payment-service | 3002 | Python | Process payments from SQS |
| notification-service | 3003 | Go | Send notifications from SQS |

## Start

```bash
docker compose up --build
```

The `setup` container creates all AWS resources and SNS→SQS subscriptions before the services start.

## Test

```bash
# Create an order — triggers the full pipeline
curl -X POST http://localhost:3001/orders \
  -H "Content-Type: application/json" \
  -d '{"customerId": "cust-123", "amount": 49.99, "items": ["sku-1", "sku-2"]}'

# Check payment was processed
curl http://localhost:3002/payments/<order-id>

# View topology in DevTools
open http://localhost:4500
```

## How it works

1. `order-service` writes the order to DynamoDB and publishes to the `orders` SNS topic
2. SNS fans out to two SQS queues: `payments` and `notifications`
3. `payment-service` polls `payments`, records the transaction in DynamoDB
4. `notification-service` polls `notifications`, logs the send (extend to call your email/SMS provider)
