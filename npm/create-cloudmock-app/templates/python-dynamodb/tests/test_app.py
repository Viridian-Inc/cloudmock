import os
import pytest
import boto3
from cloudmock import mock_aws
from httpx import AsyncClient, ASGITransport
from app import app


@pytest.fixture(scope="session")
def cloudmock_session():
    with mock_aws() as cm:
        os.environ["AWS_ENDPOINT_URL"] = cm.endpoint()
        os.environ["TABLE_NAME"] = "items"

        # Create the table
        ddb = boto3.client(
            "dynamodb",
            endpoint_url=cm.endpoint(),
            region_name="us-east-1",
            aws_access_key_id="test",
            aws_secret_access_key="test",
        )
        ddb.create_table(
            TableName="items",
            KeySchema=[{"AttributeName": "id", "KeyType": "HASH"}],
            AttributeDefinitions=[{"AttributeName": "id", "AttributeType": "S"}],
            BillingMode="PAY_PER_REQUEST",
        )
        yield cm


@pytest.fixture
async def client(cloudmock_session):
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://test"
    ) as ac:
        yield ac


@pytest.mark.asyncio
async def test_create_item(client):
    res = await client.post("/items", json={"id": "1", "name": "Widget"})
    assert res.status_code == 201
    assert res.json()["id"] == "1"


@pytest.mark.asyncio
async def test_get_item(client):
    await client.post("/items", json={"id": "2", "name": "Gadget"})
    res = await client.get("/items/2")
    assert res.status_code == 200
    assert res.json()["name"] == "Gadget"


@pytest.mark.asyncio
async def test_list_items(client):
    res = await client.get("/items")
    assert res.status_code == 200
    assert isinstance(res.json(), list)


@pytest.mark.asyncio
async def test_delete_item(client):
    await client.post("/items", json={"id": "3", "name": "Doomed"})
    res = await client.delete("/items/3")
    assert res.status_code == 204
    res = await client.get("/items/3")
    assert res.status_code == 404
