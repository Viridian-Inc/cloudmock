"""Custom Cloud Custodian actions for cloudmock."""
from c7n.actions import Action
from c7n import utils
import requests
import os

ADMIN_ENDPOINT = os.environ.get("CLOUDMOCK_ADMIN_ENDPOINT", "http://localhost:4599")

class ResetService(Action):
    """Reset a cloudmock service state."""
    schema = utils.type_schema("cloudmock-reset", service={"type": "string"})

    def process(self, resources):
        service = self.data.get("service", "")
        if service:
            requests.post(f"{ADMIN_ENDPOINT}/api/services/{service}/reset")
        else:
            requests.post(f"{ADMIN_ENDPOINT}/api/reset")

class SeedData(Action):
    """Seed cloudmock with test data."""
    schema = utils.type_schema("cloudmock-seed", file={"type": "string"})

    def process(self, resources):
        seed_file = self.data.get("file", "")
        if seed_file:
            with open(seed_file) as f:
                requests.post(f"{ADMIN_ENDPOINT}/api/seed", json={"data": f.read()})

class Snapshot(Action):
    """Export cloudmock state snapshot."""
    schema = utils.type_schema("cloudmock-snapshot", output={"type": "string"})

    def process(self, resources):
        resp = requests.get(f"{ADMIN_ENDPOINT}/api/state")
        if resp.ok:
            output = self.data.get("output", "cloudmock-snapshot.json")
            with open(output, "w") as f:
                f.write(resp.text)
