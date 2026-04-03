import os
from contextlib import asynccontextmanager
from typing import Any

import boto3
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse

TABLE_NAME = os.getenv("TABLE_NAME", "items")
ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localhost:4566")
REGION = os.getenv("AWS_REGION", "us-east-1")


def get_table():
    dynamodb = boto3.resource(
        "dynamodb",
        endpoint_url=ENDPOINT_URL,
        region_name=REGION,
        aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
        aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
    )
    return dynamodb.Table(TABLE_NAME)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Ensure the table exists on startup (useful for local dev)
    client = boto3.client(
        "dynamodb",
        endpoint_url=ENDPOINT_URL,
        region_name=REGION,
        aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
        aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
    )
    existing = [t for t in client.list_tables()["TableNames"] if t == TABLE_NAME]
    if not existing:
        client.create_table(
            TableName=TABLE_NAME,
            KeySchema=[{"AttributeName": "id", "KeyType": "HASH"}],
            AttributeDefinitions=[{"AttributeName": "id", "AttributeType": "S"}],
            BillingMode="PAY_PER_REQUEST",
        )
    yield


app = FastAPI(title="{{PROJECT_NAME}}", lifespan=lifespan)


@app.post("/items", status_code=201)
async def create_item(item: dict[str, Any]):
    if "id" not in item:
        raise HTTPException(status_code=400, detail="id is required")
    get_table().put_item(Item=item)
    return item


@app.get("/items/{item_id}")
async def get_item(item_id: str):
    result = get_table().get_item(Key={"id": item_id})
    if "Item" not in result:
        raise HTTPException(status_code=404, detail="Not found")
    return result["Item"]


@app.get("/items")
async def list_items():
    result = get_table().scan()
    return result.get("Items", [])


@app.delete("/items/{item_id}", status_code=204)
async def delete_item(item_id: str):
    get_table().delete_item(Key={"id": item_id})
    return JSONResponse(status_code=204, content=None)
