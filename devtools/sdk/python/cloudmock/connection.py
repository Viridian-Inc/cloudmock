"""TCP JSON-line client for the CloudMock devtools source server.

Auto-reconnects every 5 seconds, buffers up to 100 messages when disconnected,
and operates entirely on a background daemon thread so it never blocks the caller.
"""

import json
import socket
import threading
import time
from collections import deque
from typing import Any, Dict, Optional

MAX_BUFFER_SIZE = 100
RECONNECT_DELAY = 5.0
CONNECT_TIMEOUT = 3.0


class Connection:
    """TCP JSON-line connection to the devtools source server."""

    def __init__(self, host: str, port: int, app_name: str) -> None:
        self.host = host
        self.port = port
        self.app_name = app_name

        self._sock: Optional[socket.socket] = None
        self._connected = False
        self._closed = False
        self._lock = threading.Lock()
        self._buffer: deque[str] = deque(maxlen=MAX_BUFFER_SIZE)

        # Start the connection thread as a daemon so it doesn't prevent exit
        self._thread = threading.Thread(target=self._connect_loop, daemon=True)
        self._thread.start()

    def _connect_loop(self) -> None:
        """Background loop that maintains the TCP connection."""
        # Attempt first connection immediately
        self._try_connect()

        while True:
            with self._lock:
                if self._closed:
                    return

            time.sleep(RECONNECT_DELAY)

            with self._lock:
                if self._closed:
                    return
                needs_connect = not self._connected

            if needs_connect:
                self._try_connect()

    def _try_connect(self) -> None:
        """Attempt a single TCP connection to the devtools server."""
        with self._lock:
            if self._closed:
                return

        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(CONNECT_TIMEOUT)
            sock.connect((self.host, self.port))
            sock.settimeout(None)
        except OSError:
            return

        with self._lock:
            if self._closed:
                sock.close()
                return

            self._sock = sock
            self._connected = True

            # Flush buffered messages
            failed: list[str] = []
            while self._buffer:
                msg = self._buffer.popleft()
                if not self._write_locked(msg):
                    failed.append(msg)
                    break
            for msg in failed:
                self._buffer.appendleft(msg)

        # Start a reader thread to detect disconnection
        reader = threading.Thread(target=self._read_until_close, args=(sock,), daemon=True)
        reader.start()

    def _read_until_close(self, sock: socket.socket) -> None:
        """Read from the socket until it closes (we're send-only)."""
        try:
            while True:
                data = sock.recv(256)
                if not data:
                    break
        except OSError:
            pass
        finally:
            with self._lock:
                if self._sock is sock:
                    self._connected = False
                    self._sock = None
                    try:
                        sock.close()
                    except OSError:
                        pass

    def send(self, event_type: str, data: Dict[str, Any]) -> None:
        """Send an event to the devtools server. Non-blocking."""
        event = {
            "type": event_type,
            "data": data,
            "source": self.app_name,
            "runtime": "python",
            "timestamp": int(time.time() * 1000),
        }

        try:
            msg = json.dumps(event, default=str) + "\n"
        except (TypeError, ValueError):
            return

        with self._lock:
            if self._closed:
                return

            if self._connected and self._sock is not None:
                if self._write_locked(msg):
                    return
                # Write failed; mark disconnected
                self._connected = False
                if self._sock is not None:
                    try:
                        self._sock.close()
                    except OSError:
                        pass
                    self._sock = None

            # Buffer (deque with maxlen handles overflow automatically)
            self._buffer.append(msg)

    def _write_locked(self, msg: str) -> bool:
        """Write a message to the socket. Must be called with _lock held.

        Returns True on success, False on failure.
        """
        if self._sock is None:
            return False
        try:
            self._sock.sendall(msg.encode("utf-8"))
            return True
        except OSError:
            return False

    def close(self) -> None:
        """Close the connection and stop reconnection attempts."""
        with self._lock:
            if self._closed:
                return
            self._closed = True
            self._connected = False
            if self._sock is not None:
                try:
                    self._sock.close()
                except OSError:
                    pass
                self._sock = None
            self._buffer.clear()
