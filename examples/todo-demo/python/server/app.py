import json
import os
import uuid
from datetime import datetime, timezone

import boto3
from flask import Flask, jsonify, request

ENDPOINT = os.environ.get("CLOUDMOCK_ENDPOINT", "http://localhost:4566")
session = boto3.Session(
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)

dynamodb = session.client("dynamodb", endpoint_url=ENDPOINT)
s3 = session.client("s3", endpoint_url=ENDPOINT)
sns = session.client("sns", endpoint_url=ENDPOINT)
sqs = session.client("sqs", endpoint_url=ENDPOINT)

app = Flask(__name__)

topic_arn = None
queue_url = None


def setup():
    global topic_arn, queue_url

    try:
        dynamodb.create_table(
            TableName="todos",
            KeySchema=[{"AttributeName": "id", "KeyType": "HASH"}],
            AttributeDefinitions=[{"AttributeName": "id", "AttributeType": "S"}],
            BillingMode="PAY_PER_REQUEST",
        )
    except dynamodb.exceptions.ResourceInUseException:
        pass

    try:
        s3.create_bucket(Bucket="todo-attachments")
    except Exception:
        pass

    topic = sns.create_topic(Name="todo-completed")
    topic_arn = topic["TopicArn"]

    queue = sqs.create_queue(QueueName="todo-notifications")
    queue_url = queue["QueueUrl"]

    attrs = sqs.get_queue_attributes(QueueUrl=queue_url, AttributeNames=["QueueArn"])
    sns.subscribe(TopicArn=topic_arn, Protocol="sqs", Endpoint=attrs["Attributes"]["QueueArn"])


def item_to_dict(item):
    return {
        "id": item["id"]["S"],
        "title": item["title"]["S"],
        "description": item.get("description", {}).get("S", ""),
        "status": item["status"]["S"],
        "imageKey": item.get("imageKey", {}).get("S"),
        "createdAt": item["createdAt"]["S"],
    }


@app.post("/todos")
def create_todo():
    data = request.json
    todo_id = str(uuid.uuid4())
    dynamodb.put_item(
        TableName="todos",
        Item={
            "id": {"S": todo_id},
            "title": {"S": data["title"]},
            "description": {"S": data.get("description", "")},
            "status": {"S": "pending"},
            "createdAt": {"S": datetime.now(timezone.utc).isoformat()},
        },
    )
    return jsonify({"id": todo_id, "title": data["title"], "status": "pending"}), 201


@app.get("/todos")
def list_todos():
    result = dynamodb.scan(TableName="todos")
    return jsonify([item_to_dict(item) for item in result.get("Items", [])])


@app.get("/todos/<todo_id>")
def get_todo(todo_id):
    result = dynamodb.get_item(TableName="todos", Key={"id": {"S": todo_id}})
    if "Item" not in result:
        return jsonify({"error": "Not found"}), 404
    return jsonify(item_to_dict(result["Item"]))


@app.put("/todos/<todo_id>/image")
def upload_image(todo_id):
    image_key = f"todos/{todo_id}/image.jpg"
    s3.put_object(
        Bucket="todo-attachments",
        Key=image_key,
        Body=request.data,
        ContentType=request.content_type,
    )
    dynamodb.update_item(
        TableName="todos",
        Key={"id": {"S": todo_id}},
        UpdateExpression="SET imageKey = :key",
        ExpressionAttributeValues={":key": {"S": image_key}},
    )
    return jsonify({"imageKey": image_key})


@app.put("/todos/<todo_id>/complete")
def complete_todo(todo_id):
    result = dynamodb.get_item(TableName="todos", Key={"id": {"S": todo_id}})
    if "Item" not in result:
        return jsonify({"error": "Not found"}), 404

    dynamodb.update_item(
        TableName="todos",
        Key={"id": {"S": todo_id}},
        UpdateExpression="SET #s = :s",
        ExpressionAttributeNames={"#s": "status"},
        ExpressionAttributeValues={":s": {"S": "completed"}},
    )
    sns.publish(
        TopicArn=topic_arn,
        Subject="Todo Completed",
        Message=json.dumps({
            "todoId": todo_id,
            "title": result["Item"]["title"]["S"],
            "completedAt": datetime.now(timezone.utc).isoformat(),
        }),
    )
    return jsonify({"status": "completed"})


@app.get("/notifications")
def get_notifications():
    result = sqs.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=10, WaitTimeSeconds=1)
    notifications = []
    for msg in result.get("Messages", []):
        body = json.loads(msg["Body"])
        notifications.append(json.loads(body["Message"]))
        sqs.delete_message(QueueUrl=queue_url, ReceiptHandle=msg["ReceiptHandle"])
    return jsonify(notifications)


if __name__ == "__main__":
    setup()
    print("Todo API running on http://localhost:3000")
    app.run(host="0.0.0.0", port=3000)
