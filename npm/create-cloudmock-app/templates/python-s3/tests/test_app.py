import io
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
        os.environ["S3_BUCKET"] = "uploads"

        s3 = boto3.client(
            "s3",
            endpoint_url=cm.endpoint(),
            region_name="us-east-1",
            aws_access_key_id="test",
            aws_secret_access_key="test",
        )
        s3.create_bucket(Bucket="uploads")
        yield cm


@pytest.fixture
async def client(cloudmock_session):
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://test"
    ) as ac:
        yield ac


@pytest.mark.asyncio
async def test_upload_file(client):
    content = b"Hello, CloudMock!"
    res = await client.post(
        "/upload?key=hello.txt",
        files={"file": ("hello.txt", io.BytesIO(content), "text/plain")},
    )
    assert res.status_code == 201
    assert res.json()["key"] == "hello.txt"


@pytest.mark.asyncio
async def test_list_files(client):
    content = b"list test"
    await client.post(
        "/upload?key=list-test.txt",
        files={"file": ("list-test.txt", io.BytesIO(content), "text/plain")},
    )
    res = await client.get("/files")
    assert res.status_code == 200
    keys = [f["key"] for f in res.json()]
    assert "list-test.txt" in keys


@pytest.mark.asyncio
async def test_download_file(client):
    content = b"download me"
    await client.post(
        "/upload?key=download.txt",
        files={"file": ("download.txt", io.BytesIO(content), "text/plain")},
    )
    res = await client.get("/files/download.txt")
    assert res.status_code == 200
    assert res.content == content


@pytest.mark.asyncio
async def test_download_missing(client):
    res = await client.get("/files/nonexistent.txt")
    assert res.status_code == 404
