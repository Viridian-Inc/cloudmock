"""Main entry point for the CloudMock Python SDK.

Provides init/close lifecycle and convenience accessors for middleware.
"""

import os
import threading
from typing import Optional

from .connection import Connection
from .http_interceptor import intercept_requests, restore_requests
from .error_handler import install_error_handler, uninstall_error_handler
from .logging_handler import CloudMockLoggingHandler
from .middleware import CloudMockWSGIMiddleware, CloudMockASGIMiddleware

_lock = threading.Lock()
_conn: Optional[Connection] = None
_logging_handler: Optional[CloudMockLoggingHandler] = None


def init(
    app_name: str,
    host: str = "localhost",
    port: int = 4580,
    capture_http: bool = True,
    capture_errors: bool = True,
    capture_logging: bool = True,
) -> None:
    """Initialize the CloudMock devtools SDK.

    Call once at application startup. No-ops gracefully if devtools is not running.
    Safe to call multiple times; subsequent calls are ignored.

    Args:
        app_name: Application name shown in the devtools source bar.
        host: Devtools server host.
        port: Devtools server TCP port.
        capture_http: Monkey-patch requests/urllib3 to capture outbound HTTP.
        capture_errors: Install sys.excepthook for uncaught exceptions.
        capture_logging: Install a logging.Handler that forwards log records.
    """
    global _conn, _logging_handler

    with _lock:
        if _conn is not None:
            return

        conn = Connection(host, port, app_name)
        _conn = conn

    # Register this source
    conn.send("source:register", {
        "runtime": "python",
        "appName": app_name,
        "pid": os.getpid(),
        "pythonVersion": _python_version(),
    })

    if capture_http:
        intercept_requests(conn)

    if capture_errors:
        install_error_handler(conn)

    if capture_logging:
        handler = CloudMockLoggingHandler(conn)
        handler.install()
        with _lock:
            _logging_handler = handler


def close() -> None:
    """Disconnect from devtools and restore all intercepted functions."""
    global _conn, _logging_handler

    with _lock:
        conn = _conn
        handler = _logging_handler
        _conn = None
        _logging_handler = None

    if conn is None:
        return

    restore_requests()
    uninstall_error_handler()

    if handler is not None:
        handler.uninstall()

    conn.close()


def get_connection() -> Optional[Connection]:
    """Return the current global connection, or None if not initialized."""
    with _lock:
        return _conn


def get_middleware():
    """Return a WSGI middleware class for Django/Flask.

    Usage::

        # Flask
        app.wsgi_app = cloudmock.get_middleware()(app.wsgi_app)

        # Django (in settings.py MIDDLEWARE)
        # Use CloudMockDjangoMiddleware instead (see middleware.py)
    """
    conn = get_connection()
    if conn is None:
        # Return a pass-through middleware
        class NoOpMiddleware:
            def __init__(self, app):
                self.app = app

            def __call__(self, environ, start_response):
                return self.app(environ, start_response)

        return NoOpMiddleware

    return lambda app: CloudMockWSGIMiddleware(app, conn)


def get_asgi_middleware():
    """Return an ASGI middleware class for FastAPI/Starlette.

    Usage::

        from cloudmock import get_asgi_middleware
        app.add_middleware(get_asgi_middleware())
    """
    conn = get_connection()
    if conn is None:
        class NoOpASGIMiddleware:
            def __init__(self, app):
                self.app = app

            async def __call__(self, scope, receive, send):
                await self.app(scope, receive, send)

        return NoOpASGIMiddleware

    return lambda app: CloudMockASGIMiddleware(app, conn)


def log(level: str, message: str) -> None:
    """Send a log message to the devtools server.

    Args:
        level: Log level (debug, info, warn, error).
        message: The log message.
    """
    conn = get_connection()
    if conn is None:
        return

    import inspect
    frame = inspect.currentframe()
    caller = frame.f_back if frame else None

    file_name = None
    line_no = None
    if caller is not None:
        file_name = caller.f_code.co_filename
        line_no = caller.f_lineno

    conn.send("console", {
        "level": level,
        "message": message,
        "file": file_name,
        "line": line_no,
    })


def _python_version() -> str:
    import sys
    return f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
