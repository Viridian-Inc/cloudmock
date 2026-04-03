# Monitoring Stack

CloudMock + Prometheus + Grafana. Scrape CloudMock's admin metrics and visualize them.

## Services

| Service | Port | Description |
|---------|------|-------------|
| CloudMock | 4566 | AWS API emulation |
| CloudMock DevTools | 4500 | Built-in observability dashboard |
| CloudMock Admin | 4599 | Admin API + Prometheus metrics |
| Prometheus | 9090 | Metrics collection |
| Grafana | 3000 | Dashboards and visualization |

## Start

```bash
docker compose up
```

## Access

- **CloudMock DevTools**: [http://localhost:4500](http://localhost:4500) — request traces, topology, errors
- **Prometheus**: [http://localhost:9090](http://localhost:9090) — raw metrics explorer
- **Grafana**: [http://localhost:3000](http://localhost:3000) — login: admin / admin

## Add a Grafana dashboard

1. Open Grafana at http://localhost:3000
2. Go to **Connections > Data Sources > Add data source**
3. Choose **Prometheus**, set URL to `http://prometheus:9090`, click **Save & Test**
4. Go to **Dashboards > New > Import**
5. Paste a CloudMock dashboard JSON or build your own with metrics like:
   - `cloudmock_requests_total` — total API calls by service
   - `cloudmock_request_duration_seconds` — latency histogram
   - `cloudmock_errors_total` — error rate by service and error code

## Available metrics

CloudMock exposes Prometheus metrics at `http://localhost:4599/metrics`. Common metrics:

```
cloudmock_requests_total{service="s3", method="PutObject"} 42
cloudmock_request_duration_seconds_bucket{service="dynamodb", le="0.001"} 38
cloudmock_errors_total{service="sqs", code="QueueDoesNotExist"} 1
```

## Generate some traffic

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

aws s3 mb s3://test-bucket
for i in $(seq 1 20); do aws s3 cp /dev/urandom s3://test-bucket/file-$i --expected-size 1024; done
```

Then watch the metrics update in Prometheus and Grafana.
