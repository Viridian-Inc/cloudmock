import json
import time
import uuid
from datetime import datetime, timezone

import boto3

ENDPOINT = "http://localhost:4566"
session = boto3.Session(
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)

dynamodb = session.client("dynamodb", endpoint_url=ENDPOINT)
s3 = session.client("s3", endpoint_url=ENDPOINT)
sns = session.client("sns", endpoint_url=ENDPOINT)
sqs = session.client("sqs", endpoint_url=ENDPOINT)


def main():
    print("=== CloudMock Todo Demo (Python) ===\n")

    # 1. Create resources
    print("Creating AWS resources...")
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
    queue_arn = attrs["Attributes"]["QueueArn"]
    sns.subscribe(TopicArn=topic_arn, Protocol="sqs", Endpoint=queue_arn)
    print("  DynamoDB table, S3 bucket, SNS topic, SQS queue created.\n")

    # 2. Create a todo
    todo_id = str(uuid.uuid4())
    print(f'Creating todo: "{todo_id[:8]}..."')
    dynamodb.put_item(
        TableName="todos",
        Item={
            "id": {"S": todo_id},
            "title": {"S": "Try CloudMock"},
            "description": {"S": "Run the quickstart and explore the DevTools dashboard"},
            "status": {"S": "pending"},
            "createdAt": {"S": datetime.now(timezone.utc).isoformat()},
        },
    )
    print("  Todo created in DynamoDB.\n")

    # 3. Attach an image
    print("Uploading image to S3...")
    image_key = f"todos/{todo_id}/image.jpg"
    with open("../sample.jpg", "rb") as f:
        s3.put_object(Bucket="todo-attachments", Key=image_key, Body=f.read(), ContentType="image/jpeg")
    dynamodb.update_item(
        TableName="todos",
        Key={"id": {"S": todo_id}},
        UpdateExpression="SET imageKey = :key",
        ExpressionAttributeValues={":key": {"S": image_key}},
    )
    print(f"  Image uploaded to s3://todo-attachments/{image_key}\n")

    # 4. Complete the todo
    print("Marking todo as complete...")
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
            "title": "Try CloudMock",
            "completedAt": datetime.now(timezone.utc).isoformat(),
        }),
    )
    print("  Todo completed. Notification sent via SNS.\n")

    # 5. Poll for notification
    print("Polling SQS for notifications...")
    time.sleep(0.5)
    response = sqs.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=10, WaitTimeSeconds=2)
    for msg in response.get("Messages", []):
        body = json.loads(msg["Body"])
        notification = json.loads(body["Message"])
        print(f'  Notification received: "{notification["title"]}" completed at {notification["completedAt"]}')
        sqs.delete_message(QueueUrl=queue_url, ReceiptHandle=msg["ReceiptHandle"])

    # 6. Verify
    print("\nFinal state:")
    item = dynamodb.get_item(TableName="todos", Key={"id": {"S": todo_id}})["Item"]
    print(f'  Todo: "{item["title"]["S"]}" — status: {item["status"]["S"]}, image: {item.get("imageKey", {}).get("S", "none")}')

    print("\n=== Done! Open http://localhost:4500 to explore the DevTools. ===")


if __name__ == "__main__":
    main()
