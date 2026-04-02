import 'dart:io';

import 'connection.dart';
import 'error_handler.dart';

/// Main entry point for the CloudMock devtools SDK.
///
/// Usage:
/// ```dart
/// import 'package:cloudmock/cloudmock.dart';
///
/// void main() {
///   CloudMock.init(appName: 'my-app');
///   defer CloudMock.close();
///
///   // Use CloudMockHttpOverrides for automatic HTTP capture
///   HttpOverrides.global = CloudMockHttpOverrides();
///
///   // Or use CloudMockDioInterceptor with dio
///   final dio = Dio();
///   dio.interceptors.add(CloudMockDioInterceptor());
/// }
/// ```
class CloudMock {
  static Connection? _connection;
  static bool _initialized = false;

  /// Initialize the CloudMock SDK and connect to the devtools server.
  ///
  /// Call once at application startup. No-ops gracefully if devtools is not
  /// running. Safe to call multiple times; subsequent calls are ignored.
  ///
  /// [appName] - Application name shown in the devtools source bar.
  /// [host] - Devtools server host (default: localhost).
  /// [port] - Devtools server TCP port (default: 4580).
  /// [captureErrors] - Install error handlers for uncaught errors.
  static void init({
    required String appName,
    String host = 'localhost',
    int port = 4580,
    bool captureErrors = true,
  }) {
    if (_initialized) return;
    _initialized = true;

    final conn = Connection(host, port, appName);
    _connection = conn;

    // Register this source
    conn.send('source:register', {
      'runtime': 'dart',
      'appName': appName,
      'pid': pid,
      'dartVersion': Platform.version,
    });

    if (captureErrors) {
      installErrorHandlers(conn);
    }
  }

  /// Close the SDK and disconnect from the devtools server.
  static void close() {
    if (!_initialized) return;
    _initialized = false;

    uninstallErrorHandlers();
    _connection?.close();
    _connection = null;
  }

  /// Send a log message to the devtools server.
  ///
  /// [message] - The log message.
  /// [level] - Log level (default: 'info').
  static void log(String message, {String level = 'info'}) {
    final conn = _connection;
    if (conn == null) return;

    // Get caller info from stack trace
    final trace = StackTrace.current;
    final frame = _parseCallerFrame(trace);

    conn.send('console', {
      'level': level,
      'message': message,
      'file': frame?.file,
      'line': frame?.line,
    });
  }

  /// Get the current connection, or null if not initialized.
  static Connection? get connection => _connection;

  /// Whether the SDK has been initialized.
  static bool get isInitialized => _initialized;
}

/// Parsed stack frame with file and line information.
class _CallerFrame {
  final String? file;
  final int? line;
  _CallerFrame(this.file, this.line);
}

/// Parse the first useful caller frame from a stack trace.
_CallerFrame? _parseCallerFrame(StackTrace trace) {
  final lines = trace.toString().split('\n');
  // Skip frames from this SDK (index 0 is _parseCallerFrame, 1 is log)
  for (var i = 2; i < lines.length; i++) {
    final line = lines[i].trim();
    if (line.isEmpty) continue;

    // Dart stack frames: "#2  main (package:myapp/main.dart:15:3)"
    final match = RegExp(r'\((.+):(\d+):\d+\)').firstMatch(line);
    if (match != null) {
      return _CallerFrame(
        match.group(1),
        int.tryParse(match.group(2) ?? ''),
      );
    }

    // Alternative format: "package:myapp/main.dart 15:3  main"
    final altMatch = RegExp(r'(\S+)\s+(\d+):\d+').firstMatch(line);
    if (altMatch != null) {
      return _CallerFrame(
        altMatch.group(1),
        int.tryParse(altMatch.group(2) ?? ''),
      );
    }
  }
  return null;
}
