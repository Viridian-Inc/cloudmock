"""WSGI and ASGI middleware for capturing inbound HTTP requests.

WSGI middleware works with Django, Flask, Bottle, and other WSGI frameworks.
ASGI middleware works with FastAPI, Starlette, and other ASGI frameworks.
"""

import io
import time
import random
import string
from typing import Any, Callable, List, Optional, Tuple

from .connection import Connection


def _random_id() -> str:
    suffix = "".join(random.choices(string.ascii_lowercase + string.digits, k=6))
    return f"inbound_{int(time.time() * 1000)}_{suffix}"


class CloudMockWSGIMiddleware:
    """WSGI middleware that captures inbound HTTP requests.

    Usage with Flask::

        import cloudmock
        cloudmock.init("my-flask-app")
        app.wsgi_app = CloudMockWSGIMiddleware(app.wsgi_app, connection)

    Or via the convenience function::

        app.wsgi_app = cloudmock.get_middleware()(app.wsgi_app)
    """

    def __init__(self, app: Callable, conn: Connection) -> None:
        self.app = app
        self.conn = conn

    def __call__(self, environ: dict, start_response: Callable) -> Any:
        start = time.time()
        req_id = _random_id()

        method = environ.get("REQUEST_METHOD", "GET")
        path = environ.get("PATH_INFO", "/")
        query = environ.get("QUERY_STRING", "")
        url = f"{path}?{query}" if query else path

        # Capture request headers
        req_headers = {}
        for key, value in environ.items():
            if key.startswith("HTTP_"):
                header_name = key[5:].replace("_", "-").lower()
                req_headers[header_name] = value
            elif key == "CONTENT_TYPE":
                req_headers["content-type"] = value
            elif key == "CONTENT_LENGTH":
                req_headers["content-length"] = value

        user_agent = environ.get("HTTP_USER_AGENT", "")
        remote_addr = environ.get("REMOTE_ADDR", "")

        # Capture response status and body
        captured_status: Optional[str] = None
        captured_body_chunks: List[bytes] = []
        captured_body_size = 0

        def capturing_start_response(status: str, response_headers: List[Tuple[str, str]], exc_info: Any = None) -> Callable:
            nonlocal captured_status
            captured_status = status
            return start_response(status, response_headers, exc_info)

        try:
            response = self.app(environ, capturing_start_response)

            # Collect response body chunks (up to 4KB for capture)
            output = []
            for chunk in response:
                output.append(chunk)
                if captured_body_size < 4096:
                    capture_amount = min(len(chunk), 4096 - captured_body_size)
                    captured_body_chunks.append(chunk[:capture_amount])
                    captured_body_size += capture_amount

            if hasattr(response, "close"):
                response.close()

        except Exception as exc:
            duration = int((time.time() - start) * 1000)
            self.conn.send("http:inbound", {
                "id": req_id,
                "direction": "inbound",
                "method": method,
                "url": url,
                "path": path,
                "status": 500,
                "duration_ms": duration,
                "request_headers": req_headers,
                "error": str(exc),
                "user_agent": user_agent,
                "remote_addr": remote_addr,
            })
            raise

        duration = int((time.time() - start) * 1000)

        # Parse status code
        status_code = 200
        if captured_status:
            try:
                status_code = int(captured_status.split(" ", 1)[0])
            except (ValueError, IndexError):
                pass

        body_str = ""
        try:
            body_str = b"".join(captured_body_chunks).decode("utf-8", errors="replace")
        except Exception:
            pass

        self.conn.send("http:inbound", {
            "id": req_id,
            "direction": "inbound",
            "method": method,
            "url": url,
            "path": path,
            "status": status_code,
            "duration_ms": duration,
            "request_headers": req_headers,
            "response_body": body_str,
            "user_agent": user_agent,
            "remote_addr": remote_addr,
        })

        return output


class CloudMockASGIMiddleware:
    """ASGI middleware that captures inbound HTTP requests.

    Usage with FastAPI::

        import cloudmock
        cloudmock.init("my-fastapi-app")

        from cloudmock.middleware import CloudMockASGIMiddleware
        app.add_middleware(cloudmock.get_asgi_middleware())

    Or directly::

        app = CloudMockASGIMiddleware(app, connection)
    """

    def __init__(self, app: Callable, conn: Connection) -> None:
        self.app = app
        self.conn = conn

    async def __call__(self, scope: dict, receive: Callable, send: Callable) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        start = time.time()
        req_id = _random_id()

        method = scope.get("method", "GET")
        path = scope.get("path", "/")
        query = scope.get("query_string", b"").decode("utf-8", errors="replace")
        url = f"{path}?{query}" if query else path

        # Capture request headers
        req_headers = {}
        for name, value in scope.get("headers", []):
            req_headers[name.decode("utf-8", errors="replace")] = value.decode("utf-8", errors="replace")

        user_agent = req_headers.get("user-agent", "")
        client = scope.get("client")
        remote_addr = client[0] if client else ""

        # Capture response status and body
        status_code = 200
        body_chunks: List[bytes] = []
        body_size = 0

        async def capturing_send(message: dict) -> None:
            nonlocal status_code, body_size

            if message["type"] == "http.response.start":
                status_code = message.get("status", 200)
            elif message["type"] == "http.response.body":
                chunk = message.get("body", b"")
                if body_size < 4096:
                    capture_amount = min(len(chunk), 4096 - body_size)
                    body_chunks.append(chunk[:capture_amount])
                    body_size += capture_amount

            await send(message)

        try:
            await self.app(scope, receive, capturing_send)
        except Exception as exc:
            duration = int((time.time() - start) * 1000)
            self.conn.send("http:inbound", {
                "id": req_id,
                "direction": "inbound",
                "method": method,
                "url": url,
                "path": path,
                "status": 500,
                "duration_ms": duration,
                "request_headers": req_headers,
                "error": str(exc),
                "user_agent": user_agent,
                "remote_addr": remote_addr,
            })
            raise

        duration = int((time.time() - start) * 1000)

        body_str = ""
        try:
            body_str = b"".join(body_chunks).decode("utf-8", errors="replace")
        except Exception:
            pass

        self.conn.send("http:inbound", {
            "id": req_id,
            "direction": "inbound",
            "method": method,
            "url": url,
            "path": path,
            "status": status_code,
            "duration_ms": duration,
            "request_headers": req_headers,
            "response_body": body_str,
            "user_agent": user_agent,
            "remote_addr": remote_addr,
        })
