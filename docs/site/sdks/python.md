# Python Integration

Python works with CloudMock through standard OpenTelemetry and boto3. No CloudMock-specific SDK is needed.

## AWS SDK (boto3)

### Setup

```bash
pip install boto3
```

Point boto3 at CloudMock:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

boto3 v1.28+ reads `AWS_ENDPOINT_URL` automatically. For older versions, pass the endpoint explicitly:

```python
import boto3

s3 = boto3.client('s3', endpoint_url='http://localhost:4566')
dynamodb = boto3.resource('dynamodb', endpoint_url='http://localhost:4566')
```

### Examples

**S3:**

```python
import boto3

s3 = boto3.client('s3', region_name='us-east-1')

# Create bucket
s3.create_bucket(Bucket='my-uploads')

# Upload file
s3.put_object(Bucket='my-uploads', Key='data.json', Body=b'{"hello": "world"}')

# Download file
response = s3.get_object(Bucket='my-uploads', Key='data.json')
data = response['Body'].read()

# List objects
objects = s3.list_objects_v2(Bucket='my-uploads')
for obj in objects.get('Contents', []):
    print(obj['Key'], obj['Size'])
```

**DynamoDB:**

```python
import boto3

dynamodb = boto3.resource('dynamodb', region_name='us-east-1')

# Create table
table = dynamodb.create_table(
    TableName='users',
    KeySchema=[{'AttributeName': 'userId', 'KeyType': 'HASH'}],
    AttributeDefinitions=[{'AttributeName': 'userId', 'AttributeType': 'S'}],
    BillingMode='PAY_PER_REQUEST',
)
table.wait_until_exists()

# Put item
table.put_item(Item={'userId': 'user-1', 'name': 'Alice', 'email': 'alice@example.com'})

# Get item
response = table.get_item(Key={'userId': 'user-1'})
item = response['Item']

# Query
response = table.query(
    KeyConditionExpression='userId = :uid',
    ExpressionAttributeValues={':uid': 'user-1'},
)
```

**SQS:**

```python
import boto3

sqs = boto3.client('sqs', region_name='us-east-1')

# Create queue
queue = sqs.create_queue(QueueName='order-processing')
queue_url = queue['QueueUrl']

# Send message
sqs.send_message(QueueUrl=queue_url, MessageBody='{"orderId": "order-123"}')

# Receive messages
response = sqs.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=10, WaitTimeSeconds=5)
for msg in response.get('Messages', []):
    print(msg['Body'])
    sqs.delete_message(QueueUrl=queue_url, ReceiptHandle=msg['ReceiptHandle'])
```

**Lambda:**

```python
import boto3
import json

lambda_client = boto3.client('lambda', region_name='us-east-1')

# Invoke function
response = lambda_client.invoke(
    FunctionName='order-processor',
    Payload=json.dumps({'orderId': 'order-123'}),
)
result = json.loads(response['Payload'].read())
```

## OpenTelemetry

### Zero-Code Instrumentation

The fastest way to add tracing -- no code changes:

```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp-proto-http
opentelemetry-bootstrap -a install  # Auto-install instrumentations
```

Run your app with auto-instrumentation:

```bash
opentelemetry-instrument \
  --traces_exporter otlp \
  --metrics_exporter otlp \
  --logs_exporter otlp \
  --exporter_otlp_endpoint http://localhost:4318 \
  --exporter_otlp_protocol http/json \
  --service_name my-python-service \
  python app.py
```

This auto-instruments: Flask, Django, FastAPI, requests, urllib3, boto3, psycopg2, redis, celery, and more.

### Programmatic Setup

```bash
pip install opentelemetry-sdk \
  opentelemetry-exporter-otlp-proto-http \
  opentelemetry-instrumentation-boto3 \
  opentelemetry-instrumentation-flask
```

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.boto3 import Boto3Instrumentor
from opentelemetry.instrumentation.flask import FlaskInstrumentor

# Configure tracing
resource = Resource.create({"service.name": "my-python-service"})
provider = TracerProvider(resource=resource)
provider.add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces"))
)
trace.set_tracer_provider(provider)

# Auto-instrument boto3 (AWS SDK calls appear as spans)
Boto3Instrumentor().instrument()

# Auto-instrument Flask
from flask import Flask
app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)
```

### Custom Spans

```python
from opentelemetry import trace

tracer = trace.get_tracer("my-service")

def process_order(order):
    with tracer.start_as_current_span("process-order") as span:
        span.set_attribute("order.id", order["id"])
        span.set_attribute("order.total", order["total"])

        try:
            charge_payment(order)
            send_confirmation(order)
        except Exception as e:
            span.record_exception(e)
            span.set_status(trace.StatusCode.ERROR, str(e))
            raise
```

### Logging Integration

Forward Python `logging` output to CloudMock:

```python
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
import logging

logger_provider = LoggerProvider(resource=resource)
logger_provider.add_log_record_processor(
    BatchLogRecordProcessor(
        OTLPLogExporter(endpoint="http://localhost:4318/v1/logs")
    )
)

handler = LoggingHandler(logger_provider=logger_provider)
logging.getLogger().addHandler(handler)
logging.getLogger().setLevel(logging.INFO)

# Now standard logging calls appear in CloudMock's log viewer
logging.info("Order processed", extra={"order_id": "order-123"})
logging.error("Payment failed", exc_info=True)
```

## Full Example: Flask + DynamoDB

```python
# app.py
import boto3
from flask import Flask, request, jsonify
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.boto3 import Boto3Instrumentor
from opentelemetry.instrumentation.flask import FlaskInstrumentor

# Setup tracing
resource = Resource.create({"service.name": "order-api"})
provider = TracerProvider(resource=resource)
provider.add_span_processor(
    BatchSpanProcessor(OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces"))
)
trace.set_tracer_provider(provider)
Boto3Instrumentor().instrument()

# App
app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)

dynamodb = boto3.resource("dynamodb", region_name="us-east-1")
table = dynamodb.Table("orders")
tracer = trace.get_tracer("order-api")

@app.post("/orders")
def create_order():
    data = request.json
    with tracer.start_as_current_span("create-order") as span:
        span.set_attribute("order.customer", data["customer"])
        table.put_item(Item={"orderId": data["id"], "customer": data["customer"], "status": "pending"})
        return jsonify({"id": data["id"], "status": "pending"}), 201

@app.get("/orders/<order_id>")
def get_order(order_id):
    response = table.get_item(Key={"orderId": order_id})
    if "Item" not in response:
        return jsonify({"error": "not found"}), 404
    return jsonify(response["Item"])

if __name__ == "__main__":
    app.run(port=3000)
```

Run it:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
python app.py
```

Open DevTools at `http://localhost:4500` to see Flask requests, DynamoDB calls, and traces all correlated.
