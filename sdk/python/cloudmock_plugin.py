"""CloudMock Plugin SDK for Python.

Build CloudMock plugins in Python by subclassing CloudMockPlugin and calling serve().

Example:

    from cloudmock_plugin import CloudMockPlugin, serve

    class MyPlugin(CloudMockPlugin):
        def describe(self):
            return {
                "name": "my-plugin",
                "version": "0.1.0",
                "protocol": "custom",
                "actions": ["DoThing"],
                "api_paths": ["/my-plugin/*"],
            }

        def handle_request(self, request):
            return {
                "status_code": 200,
                "body": b'{"ok": true}',
                "headers": {"Content-Type": "application/json"},
            }

    if __name__ == "__main__":
        serve(MyPlugin())
"""

import json
import os
import signal
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from abc import ABC, abstractmethod


class CloudMockPlugin(ABC):
    """Base class for CloudMock plugins."""

    def init(self, config: bytes, data_dir: str, log_level: str) -> None:
        """Called once when the plugin is loaded. Override to perform setup."""
        pass

    def shutdown(self) -> None:
        """Called when the core is stopping. Override to release resources."""
        pass

    def health_check(self) -> tuple:
        """Return (status, message). Status is 'healthy', 'degraded', or 'unhealthy'."""
        return ("healthy", "")

    @abstractmethod
    def describe(self) -> dict:
        """Return plugin descriptor dict with name, version, protocol, actions, api_paths."""
        ...

    @abstractmethod
    def handle_request(self, request: dict) -> dict:
        """Process a request and return response dict with status_code, body, headers."""
        ...


def serve(plugin: CloudMockPlugin):
    """Start serving the plugin. Communicates with CloudMock core via HTTP."""
    addr = os.environ.get("CLOUDMOCK_PLUGIN_ADDR", "127.0.0.1:0")
    config = os.environ.get("CLOUDMOCK_PLUGIN_CONFIG", "").encode()
    data_dir = os.environ.get("CLOUDMOCK_PLUGIN_DATA_DIR", "")
    log_level = os.environ.get("CLOUDMOCK_PLUGIN_LOG_LEVEL", "info")

    plugin.init(config, data_dir, log_level)

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path == "/describe":
                desc = plugin.describe()
                self._json_response(200, desc)
            elif self.path == "/health":
                status, msg = plugin.health_check()
                self._json_response(200, {"status": status, "message": msg})
            else:
                self._json_response(404, {"error": "not found"})

        def do_POST(self):
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length) if length else b""
            request = json.loads(body) if body else {}
            response = plugin.handle_request(request)
            self._json_response(
                response.get("status_code", 200),
                response,
            )

        def _json_response(self, status, data):
            body = json.dumps(data).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, format, *args):
            print(f"[plugin] {format % args}", file=sys.stderr)

    host, port = addr.rsplit(":", 1) if ":" in addr else ("127.0.0.1", addr)
    server = HTTPServer((host, int(port)), Handler)

    actual_addr = f"{server.server_address[0]}:{server.server_address[1]}"
    print(f"PLUGIN_ADDR={actual_addr}", flush=True)

    desc = plugin.describe()
    print(f'[plugin] starting name={desc["name"]} addr={actual_addr}', file=sys.stderr)

    def shutdown_handler(signum, frame):
        print("[plugin] shutting down", file=sys.stderr)
        plugin.shutdown()
        server.shutdown()

    signal.signal(signal.SIGTERM, shutdown_handler)
    signal.signal(signal.SIGINT, shutdown_handler)

    server.serve_forever()
