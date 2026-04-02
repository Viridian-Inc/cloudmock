"""Python logging.Handler that forwards log records to the devtools server.

Automatically installed when cloudmock.init() is called with capture_logging=True.
Attaches to the root logger so all log records are forwarded.
"""

import logging
from typing import Optional

from .connection import Connection

# Map Python log levels to devtools levels
_LEVEL_MAP = {
    logging.DEBUG: "debug",
    logging.INFO: "info",
    logging.WARNING: "warn",
    logging.ERROR: "error",
    logging.CRITICAL: "error",
}


class CloudMockLoggingHandler(logging.Handler):
    """A logging.Handler that sends log records to the CloudMock devtools server.

    Usage (automatic via init)::

        import cloudmock
        cloudmock.init("my-app")
        # All logging.* calls now appear in devtools

    Usage (manual)::

        import logging
        from cloudmock.logging_handler import CloudMockLoggingHandler

        handler = CloudMockLoggingHandler(connection)
        logging.getLogger().addHandler(handler)
    """

    def __init__(self, conn: Connection) -> None:
        super().__init__()
        self.conn = conn
        self._installed = False

    def emit(self, record: logging.LogRecord) -> None:
        """Process a log record and send it to devtools."""
        try:
            level = _LEVEL_MAP.get(record.levelno, "info")
            message = self.format(record)

            self.conn.send("console", {
                "level": level,
                "message": message,
                "file": record.pathname,
                "line": record.lineno,
                "logger": record.name,
                "func": record.funcName,
            })
        except Exception:
            # Never let logging errors propagate
            self.handleError(record)

    def install(self) -> None:
        """Attach this handler to the root logger."""
        if self._installed:
            return
        self._installed = True
        logging.getLogger().addHandler(self)

    def uninstall(self) -> None:
        """Remove this handler from the root logger."""
        if not self._installed:
            return
        self._installed = False
        logging.getLogger().removeHandler(self)
