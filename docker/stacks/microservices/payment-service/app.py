import json
import os
import boto3
from fastapi import FastAPI
import threading

app = FastAPI()

endpoint = os.getenv("AWS_ENDPOINT_URL", "http://cloudmock:4566")
sqs = boto3.client(
    "sqs",
    endpoint_url=endpoint,
    region_name="us-east-1",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)
dynamo = boto3.client(
    "dynamodb",
    endpoint_url=endpoint,
    region_name="us-east-1",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)

QUEUE_URL = os.getenv("PAYMENTS_QUEUE_URL", f"{endpoint}/000000000000/payments")


def process_payment(order: dict) -> dict:
    """Simulate payment processing."""
    return {"orderId": order.get("id"), "status": "paid", "transactionId": f"txn-{order.get('id', 'unknown')[:8]}"}


def poll_queue():
    while True:
        resp = sqs.receive_message(QueueUrl=QUEUE_URL, WaitTimeSeconds=5, MaxNumberOfMessages=10)
        for msg in resp.get("Messages", []):
            try:
                order = json.loads(msg["Body"])
                result = process_payment(order)
                dynamo.put_item(
                    TableName="payments",
                    Item={
                        "orderId": {"S": result["orderId"]},
                        "status": {"S": result["status"]},
                        "transactionId": {"S": result["transactionId"]},
                    },
                )
                sqs.delete_message(QueueUrl=QUEUE_URL, ReceiptHandle=msg["ReceiptHandle"])
                print(f"Processed payment for order {result['orderId']}")
            except Exception as e:
                print(f"Error processing message: {e}")


@app.get("/health")
def health():
    return {"status": "ok"}


@app.get("/payments/{order_id}")
def get_payment(order_id: str):
    result = dynamo.get_item(TableName="payments", Key={"orderId": {"S": order_id}})
    item = result.get("Item")
    if not item:
        return {"error": "not found"}
    return {"orderId": item["orderId"]["S"], "status": item["status"]["S"], "transactionId": item["transactionId"]["S"]}


# Start polling in background thread on startup
@app.on_event("startup")
def start_polling():
    t = threading.Thread(target=poll_queue, daemon=True)
    t.start()
