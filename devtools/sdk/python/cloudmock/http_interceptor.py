"""HTTP interceptor that monkey-patches requests and urllib3 to capture outbound traffic.

Patches:
    - requests.Session.send (if `requests` is installed)
    - urllib3.HTTPConnectionPool.urlopen (if `urllib3` is installed)

All patches are reversible via restore_requests().
"""

import time
import threading
from typing import Any, Optional

from .connection import Connection

_lock = threading.Lock()
_conn: Optional[Connection] = None
_original_session_send = None
_original_urllib3_urlopen = None
_request_counter = 0
_counter_lock = threading.Lock()


def _next_id() -> str:
    global _request_counter
    with _counter_lock:
        _request_counter += 1
        return f"py_req_{_request_counter}_{int(time.time() * 1000)}"


def intercept_requests(conn: Connection) -> None:
    """Install monkey-patches on requests and urllib3."""
    global _conn, _original_session_send, _original_urllib3_urlopen

    with _lock:
        _conn = conn

    _patch_requests()
    _patch_urllib3()


def restore_requests() -> None:
    """Remove all monkey-patches and restore original functions."""
    global _conn
    with _lock:
        _conn = None

    _unpatch_requests()
    _unpatch_urllib3()


def _patch_requests() -> None:
    """Monkey-patch requests.Session.send to capture outbound HTTP calls."""
    global _original_session_send

    try:
        import requests
    except ImportError:
        return

    if _original_session_send is not None:
        return

    _original_session_send = requests.Session.send

    def patched_send(self: Any, request: Any, **kwargs: Any) -> Any:
        conn = _conn
        if conn is None:
            return _original_session_send(self, request, **kwargs)

        req_id = _next_id()
        start = time.time()

        method = request.method or "GET"
        url = request.url or ""
        path = _parse_path(url)

        # Inject correlation headers
        request.headers["X-CloudMock-Source"] = conn.app_name
        request.headers["X-CloudMock-Request-Id"] = req_id

        try:
            response = _original_session_send(self, request, **kwargs)
        except Exception as exc:
            duration = int((time.time() - start) * 1000)
            conn.send("http:error", {
                "id": req_id,
                "method": method,
                "url": url,
                "path": path,
                "error": str(exc),
                "duration_ms": duration,
            })
            raise

        duration = int((time.time() - start) * 1000)

        # Capture response body (up to 4KB)
        body = ""
        try:
            body = response.text[:4096]
        except Exception:
            pass

        req_headers = dict(request.headers) if request.headers else {}
        resp_headers = dict(response.headers) if response.headers else {}

        conn.send("http:response", {
            "id": req_id,
            "method": method,
            "url": url,
            "path": path,
            "status": response.status_code,
            "duration_ms": duration,
            "request_headers": req_headers,
            "response_headers": resp_headers,
            "response_body": body,
            "content_length": resp_headers.get("content-length"),
        })

        return response

    requests.Session.send = patched_send


def _unpatch_requests() -> None:
    """Restore the original requests.Session.send."""
    global _original_session_send

    if _original_session_send is None:
        return

    try:
        import requests
        requests.Session.send = _original_session_send
    except ImportError:
        pass

    _original_session_send = None


def _patch_urllib3() -> None:
    """Monkey-patch urllib3.HTTPConnectionPool.urlopen to capture outbound HTTP calls."""
    global _original_urllib3_urlopen

    try:
        import urllib3
    except ImportError:
        return

    if _original_urllib3_urlopen is not None:
        return

    _original_urllib3_urlopen = urllib3.HTTPConnectionPool.urlopen

    def patched_urlopen(self: Any, method: str, url: str, body: Any = None, headers: Any = None, **kwargs: Any) -> Any:
        conn = _conn
        if conn is None:
            return _original_urllib3_urlopen(self, method, url, body=body, headers=headers, **kwargs)

        req_id = _next_id()
        start = time.time()

        full_url = f"{self.scheme}://{self.host}:{self.port}{url}" if hasattr(self, "scheme") else url
        path = _parse_path(url)

        # Inject correlation headers
        if headers is None:
            headers = {}
        else:
            headers = dict(headers)
        headers["X-CloudMock-Source"] = conn.app_name
        headers["X-CloudMock-Request-Id"] = req_id

        try:
            response = _original_urllib3_urlopen(self, method, url, body=body, headers=headers, **kwargs)
        except Exception as exc:
            duration = int((time.time() - start) * 1000)
            conn.send("http:error", {
                "id": req_id,
                "method": method,
                "url": full_url,
                "path": path,
                "error": str(exc),
                "duration_ms": duration,
            })
            raise

        duration = int((time.time() - start) * 1000)

        # Capture response body (up to 4KB)
        body_str = ""
        try:
            body_str = response.data.decode("utf-8", errors="replace")[:4096]
        except Exception:
            pass

        resp_headers = dict(response.headers) if response.headers else {}

        conn.send("http:response", {
            "id": req_id,
            "method": method,
            "url": full_url,
            "path": path,
            "status": response.status,
            "duration_ms": duration,
            "request_headers": headers,
            "response_headers": resp_headers,
            "response_body": body_str,
            "content_length": resp_headers.get("content-length"),
        })

        return response

    urllib3.HTTPConnectionPool.urlopen = patched_urlopen


def _unpatch_urllib3() -> None:
    """Restore the original urllib3.HTTPConnectionPool.urlopen."""
    global _original_urllib3_urlopen

    if _original_urllib3_urlopen is None:
        return

    try:
        import urllib3
        urllib3.HTTPConnectionPool.urlopen = _original_urllib3_urlopen
    except ImportError:
        pass

    _original_urllib3_urlopen = None


def _parse_path(url: str) -> str:
    """Extract the path component from a URL."""
    try:
        from urllib.parse import urlparse
        return urlparse(url).path or "/"
    except Exception:
        return "/"
