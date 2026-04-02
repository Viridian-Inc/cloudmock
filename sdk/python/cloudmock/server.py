"""Manages the CloudMock binary lifecycle."""

import atexit
import contextlib
import os
import platform
import shutil
import socket
import subprocess
import time
import urllib.request

import boto3
from botocore.config import Config


def _find_binary():
    """Find the cloudmock binary — check PATH, ~/.cloudmock/bin/, or download."""
    # Check PATH
    binary = shutil.which("cloudmock")
    if binary:
        return binary

    # Check ~/.cloudmock/bin/
    home = os.path.expanduser("~")
    cached = os.path.join(home, ".cloudmock", "bin", "cloudmock")
    if os.path.isfile(cached) and os.access(cached, os.X_OK):
        return cached

    # Try npx
    npx = shutil.which("npx")
    if npx:
        return None  # Will use npx as fallback

    raise FileNotFoundError(
        "CloudMock binary not found. Install with: npm install -g cloudmock, "
        "brew install viridian-inc/tap/cloudmock, or download from GitHub."
    )


def _free_port():
    """Find a free TCP port."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))
        return s.getsockname()[1]


class CloudMock:
    """Manages a CloudMock server instance.

    Usage:
        cm = CloudMock()
        cm.start()
        s3 = cm.boto3_client("s3")
        s3.create_bucket(Bucket="test")
        cm.stop()

    Or as a context manager:
        with CloudMock() as cm:
            s3 = cm.boto3_client("s3")
    """

    def __init__(self, port=None, region="us-east-1", profile="minimal"):
        self.port = port or _free_port()
        self.region = region
        self.profile = profile
        self._process = None
        self._endpoint = f"http://localhost:{self.port}"

    def start(self):
        """Start the CloudMock server."""
        if self._process is not None:
            return self

        binary = _find_binary()
        env = {
            **os.environ,
            "CLOUDMOCK_PROFILE": self.profile,
            "CLOUDMOCK_IAM_MODE": "none",
        }

        if binary:
            cmd = [binary, "--port", str(self.port)]
        else:
            cmd = ["npx", "cloudmock", "--port", str(self.port)]

        self._process = subprocess.Popen(
            cmd,
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        atexit.register(self.stop)

        # Wait for ready
        deadline = time.time() + 30
        while time.time() < deadline:
            try:
                urllib.request.urlopen(self._endpoint, timeout=1)
                return self
            except Exception:
                if self._process.poll() is not None:
                    raise RuntimeError(f"CloudMock exited with code {self._process.returncode}")
                time.sleep(0.1)

        raise TimeoutError("CloudMock did not start within 30 seconds")

    def stop(self):
        """Stop the CloudMock server."""
        if self._process is not None:
            self._process.terminate()
            try:
                self._process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self._process.kill()
            self._process = None

    @property
    def endpoint(self):
        """The endpoint URL for this instance."""
        return self._endpoint

    def boto3_client(self, service_name, **kwargs):
        """Create a boto3 client pre-configured for this CloudMock instance."""
        return boto3.client(
            service_name,
            endpoint_url=self._endpoint,
            region_name=self.region,
            aws_access_key_id="test",
            aws_secret_access_key="test",
            config=Config(retries={"max_attempts": 1}),
            **kwargs,
        )

    def boto3_resource(self, service_name, **kwargs):
        """Create a boto3 resource pre-configured for this CloudMock instance."""
        return boto3.resource(
            service_name,
            endpoint_url=self._endpoint,
            region_name=self.region,
            aws_access_key_id="test",
            aws_secret_access_key="test",
            **kwargs,
        )

    def boto3_session(self):
        """Create a boto3 Session pre-configured for this CloudMock instance."""
        return boto3.Session(
            region_name=self.region,
            aws_access_key_id="test",
            aws_secret_access_key="test",
        )

    def __enter__(self):
        self.start()
        return self

    def __exit__(self, *args):
        self.stop()


@contextlib.contextmanager
def mock_aws(port=None, region="us-east-1", profile="minimal"):
    """Context manager that starts CloudMock and yields a CloudMock instance.

    Usage:
        with mock_aws() as cm:
            s3 = cm.boto3_client("s3")
            s3.create_bucket(Bucket="my-bucket")
    """
    cm = CloudMock(port=port, region=region, profile=profile)
    cm.start()
    try:
        yield cm
    finally:
        cm.stop()
