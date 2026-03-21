"""Cloud Custodian plugin for cloudmock AWS emulator."""
import os

def register():
    """Register the cloudmock plugin.

    Auto-configures AWS endpoint URLs to point at cloudmock
    when CLOUDMOCK_ENDPOINT is set.
    """
    endpoint = os.environ.get("CLOUDMOCK_ENDPOINT", "http://localhost:4566")

    # Set AWS endpoint URL if not already set
    if "AWS_ENDPOINT_URL" not in os.environ:
        os.environ["AWS_ENDPOINT_URL"] = endpoint

    # Set default credentials if not already set
    if "AWS_ACCESS_KEY_ID" not in os.environ:
        os.environ["AWS_ACCESS_KEY_ID"] = "test"
    if "AWS_SECRET_ACCESS_KEY" not in os.environ:
        os.environ["AWS_SECRET_ACCESS_KEY"] = "test"
    if "AWS_DEFAULT_REGION" not in os.environ:
        os.environ["AWS_DEFAULT_REGION"] = "us-east-1"
