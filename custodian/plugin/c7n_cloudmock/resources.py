"""Custom Cloud Custodian resources for cloudmock."""
from c7n.manager import resources
from c7n.query import QueryResourceManager, TypeInfo
from c7n import utils
import requests
import os

ADMIN_ENDPOINT = os.environ.get("CLOUDMOCK_ADMIN_ENDPOINT", "http://localhost:4599")

@resources.register("cloudmock-service")
class CloudmockService(QueryResourceManager):
    """Query cloudmock service status."""

    class resource_type(TypeInfo):
        service = "cloudmock"
        id = "name"
        name = "name"
        enum_spec = None

    def resources(self, query=None):
        resp = requests.get(f"{ADMIN_ENDPOINT}/api/services")
        return resp.json() if resp.ok else []

@resources.register("cloudmock-request")
class CloudmockRequest(QueryResourceManager):
    """Query cloudmock request log."""

    class resource_type(TypeInfo):
        service = "cloudmock"
        id = "timestamp"
        name = "action"
        enum_spec = None

    def resources(self, query=None):
        resp = requests.get(f"{ADMIN_ENDPOINT}/api/requests?limit=1000")
        return resp.json().get("entries", []) if resp.ok else []
