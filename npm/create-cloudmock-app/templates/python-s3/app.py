import io
import os

import boto3
from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.responses import StreamingResponse

BUCKET = os.getenv("S3_BUCKET", "uploads")
ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localhost:4566")
REGION = os.getenv("AWS_REGION", "us-east-1")


def get_s3():
    return boto3.client(
        "s3",
        endpoint_url=ENDPOINT_URL,
        region_name=REGION,
        aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
        aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
    )


app = FastAPI(title="{{PROJECT_NAME}}")


@app.on_event("startup")
async def startup():
    s3 = get_s3()
    existing = [b["Name"] for b in s3.list_buckets().get("Buckets", [])]
    if BUCKET not in existing:
        s3.create_bucket(Bucket=BUCKET)


@app.post("/upload", status_code=201)
async def upload_file(key: str, file: UploadFile = File(...)):
    data = await file.read()
    get_s3().put_object(
        Bucket=BUCKET,
        Key=key,
        Body=data,
        ContentType=file.content_type or "application/octet-stream",
    )
    return {"key": key, "bucket": BUCKET, "size": len(data)}


@app.get("/files")
async def list_files():
    result = get_s3().list_objects_v2(Bucket=BUCKET)
    return [
        {"key": obj["Key"], "size": obj["Size"]}
        for obj in result.get("Contents", [])
    ]


@app.get("/files/{key:path}")
async def download_file(key: str):
    try:
        obj = get_s3().get_object(Bucket=BUCKET, Key=key)
    except get_s3().exceptions.NoSuchKey:
        raise HTTPException(status_code=404, detail="Not found")
    except Exception as e:
        if "NoSuchKey" in str(e):
            raise HTTPException(status_code=404, detail="Not found")
        raise
    return StreamingResponse(
        io.BytesIO(obj["Body"].read()),
        media_type=obj.get("ContentType", "application/octet-stream"),
    )
