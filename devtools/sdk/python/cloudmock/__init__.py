"""CloudMock devtools SDK for Python.

Connects to the CloudMock devtools source server and sends telemetry for
HTTP traffic, log messages, and uncaught exceptions. Designed to be a
silent no-op when devtools is not running.

Usage::

    import cloudmock
    cloudmock.init("my-service")

    # Django/Flask
    app.wsgi_app = cloudmock.get_middleware()(app.wsgi_app)

    # FastAPI
    app.add_middleware(cloudmock.get_asgi_middleware())
"""

from .client import init, close, get_middleware, get_asgi_middleware, log

__all__ = ["init", "close", "get_middleware", "get_asgi_middleware", "log"]
__version__ = "0.1.0"
