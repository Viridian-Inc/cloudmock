"""Uncaught exception handler that forwards crashes to the devtools server.

Wraps sys.excepthook to capture unhandled exceptions. The original excepthook
is preserved and called after sending the event.
"""

import sys
import traceback
import threading
from types import TracebackType
from typing import Any, Optional, Type

from .connection import Connection

_lock = threading.Lock()
_conn: Optional[Connection] = None
_original_excepthook = None


def install_error_handler(conn: Connection) -> None:
    """Install sys.excepthook wrapper to capture uncaught exceptions.

    Args:
        conn: The devtools connection to send events on.
    """
    global _conn, _original_excepthook

    with _lock:
        _conn = conn
        if _original_excepthook is None:
            _original_excepthook = sys.excepthook
            sys.excepthook = _cloudmock_excepthook

    # Also install threading excepthook for uncaught exceptions in threads
    if hasattr(threading, "excepthook"):
        threading.excepthook = _cloudmock_threading_excepthook


def uninstall_error_handler() -> None:
    """Restore the original sys.excepthook."""
    global _conn, _original_excepthook

    with _lock:
        _conn = None
        if _original_excepthook is not None:
            sys.excepthook = _original_excepthook
            _original_excepthook = None

    if hasattr(threading, "excepthook"):
        threading.excepthook = threading.__excepthook__


def _cloudmock_excepthook(
    exc_type: Type[BaseException],
    exc_value: BaseException,
    exc_tb: Optional[TracebackType],
) -> None:
    """Custom excepthook that sends error:uncaught events."""
    conn = _conn
    if conn is not None:
        try:
            stack = "".join(traceback.format_exception(exc_type, exc_value, exc_tb))
            conn.send("error:uncaught", {
                "name": exc_type.__name__,
                "message": str(exc_value),
                "stack": stack,
            })
        except Exception:
            pass

    # Call the original excepthook
    original = _original_excepthook
    if original is not None:
        original(exc_type, exc_value, exc_tb)


def _cloudmock_threading_excepthook(args: Any) -> None:
    """Custom threading.excepthook for uncaught thread exceptions."""
    conn = _conn
    if conn is not None:
        try:
            exc_type = args.exc_type
            exc_value = args.exc_value
            exc_tb = args.exc_traceback

            stack = ""
            if exc_tb is not None:
                stack = "".join(traceback.format_exception(exc_type, exc_value, exc_tb))

            conn.send("error:uncaught", {
                "name": exc_type.__name__ if exc_type else "UnknownError",
                "message": str(exc_value) if exc_value else "",
                "stack": stack,
                "thread": args.thread.name if args.thread else "unknown",
            })
        except Exception:
            pass

    # Call the default threading excepthook
    threading.__excepthook__(args)
