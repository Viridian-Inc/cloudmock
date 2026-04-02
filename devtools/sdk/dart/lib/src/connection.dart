import 'dart:async';
import 'dart:convert';
import 'dart:collection';
import 'dart:io';

/// TCP JSON-line client that connects to the devtools source server.
///
/// Auto-reconnects every 5 seconds and buffers up to 100 messages when
/// disconnected. All operations are non-blocking and safe to call from
/// any isolate.
class Connection {
  final String host;
  final int port;
  final String appName;

  static const int _maxBufferSize = 100;
  static const Duration _reconnectDelay = Duration(seconds: 5);
  static const Duration _connectTimeout = Duration(seconds: 3);

  Socket? _socket;
  bool _connected = false;
  bool _closed = false;
  final Queue<String> _buffer = Queue<String>();
  Timer? _reconnectTimer;

  Connection(this.host, this.port, this.appName) {
    _connect();
  }

  Future<void> _connect() async {
    if (_closed) return;

    try {
      final socket = await Socket.connect(
        host,
        port,
        timeout: _connectTimeout,
      );

      if (_closed) {
        socket.destroy();
        return;
      }

      _socket = socket;
      _connected = true;

      // Flush buffered messages
      _flushBuffer();

      // Listen for disconnect
      socket.listen(
        (_) {}, // Discard incoming data; we're send-only
        onError: (_) => _onDisconnect(socket),
        onDone: () => _onDisconnect(socket),
        cancelOnError: true,
      );
    } catch (_) {
      // Connection failed; schedule reconnect
      _scheduleReconnect();
    }
  }

  void _onDisconnect(Socket socket) {
    if (_socket == socket) {
      _connected = false;
      _socket = null;
      try {
        socket.destroy();
      } catch (_) {}
    }
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_closed || _reconnectTimer != null) return;
    _reconnectTimer = Timer(_reconnectDelay, () {
      _reconnectTimer = null;
      if (!_closed && !_connected) {
        _connect();
      }
    });
  }

  void _flushBuffer() {
    while (_buffer.isNotEmpty && _connected && _socket != null) {
      final msg = _buffer.first;
      try {
        _socket!.write(msg);
        _buffer.removeFirst();
      } catch (_) {
        break;
      }
    }
  }

  /// Send an event to the devtools server. Non-blocking.
  void send(String type, Map<String, dynamic> data) {
    if (_closed) return;

    final event = {
      'type': type,
      'data': data,
      'source': appName,
      'runtime': 'dart',
      'timestamp': DateTime.now().millisecondsSinceEpoch,
    };

    String msg;
    try {
      msg = '${jsonEncode(event)}\n';
    } catch (_) {
      return;
    }

    if (_connected && _socket != null) {
      try {
        _socket!.write(msg);
        return;
      } catch (_) {
        // Write failed; fall through to buffer
        _connected = false;
        try {
          _socket?.destroy();
        } catch (_) {}
        _socket = null;
        _scheduleReconnect();
      }
    }

    // Buffer up to _maxBufferSize messages
    if (_buffer.length < _maxBufferSize) {
      _buffer.add(msg);
    }
  }

  /// Close the connection and stop reconnection attempts.
  void close() {
    if (_closed) return;
    _closed = true;
    _connected = false;

    _reconnectTimer?.cancel();
    _reconnectTimer = null;

    try {
      _socket?.destroy();
    } catch (_) {}
    _socket = null;

    _buffer.clear();
  }
}
