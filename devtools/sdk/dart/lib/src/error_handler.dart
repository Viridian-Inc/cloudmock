import 'dart:async';
import 'dart:isolate';

import 'connection.dart';

Connection? _errorConn;
bool _installed = false;

/// Install error handlers that capture uncaught errors and send them to devtools.
///
/// Captures errors via:
/// - Zone-based error handling (runZonedGuarded)
/// - Isolate.current.addErrorListener
///
/// For Flutter apps, you should also set FlutterError.onError and
/// PlatformDispatcher.instance.onError in your main() function.
/// See [runGuarded] for a convenience wrapper.
void installErrorHandlers(Connection conn) {
  if (_installed) return;
  _installed = true;
  _errorConn = conn;

  // Listen for isolate-level uncaught errors
  _installIsolateErrorListener(conn);
}

/// Remove error handlers and restore originals.
void uninstallErrorHandlers() {
  if (!_installed) return;
  _installed = false;
  _errorConn = null;
}

/// Run a callback inside a guarded zone that captures errors.
///
/// This is the recommended way to wrap your app entry point for
/// comprehensive error capture in both Dart and Flutter environments.
///
/// For Flutter apps:
/// ```dart
/// void main() {
///   CloudMock.init(appName: 'my-app');
///   runGuarded(() {
///     WidgetsFlutterBinding.ensureInitialized();
///
///     // Capture Flutter framework errors
///     FlutterError.onError = (details) {
///       CloudMock.log(details.exceptionAsString(), level: 'error');
///     };
///
///     runApp(MyApp());
///   });
/// }
/// ```
///
/// For Dart CLI/server apps:
/// ```dart
/// void main() {
///   CloudMock.init(appName: 'my-server');
///   runGuarded(() {
///     // ... server code
///   });
/// }
/// ```
void runGuarded(void Function() body) {
  final conn = _errorConn;

  runZonedGuarded(
    body,
    (error, stackTrace) {
      if (conn != null) {
        sendError(conn, error, stackTrace);
      }
    },
  );
}

void _installIsolateErrorListener(Connection conn) {
  // Use the Isolate error listener for top-level uncaught errors
  final receivePort = ReceivePort();
  receivePort.listen((message) {
    if (message is List && message.length >= 2) {
      final errorStr = message[0].toString();
      final stackStr = message[1]?.toString() ?? '';

      conn.send('error:uncaught', {
        'name': 'IsolateError',
        'message': errorStr,
        'stack': stackStr,
      });
    }
  });

  Isolate.current.addErrorListener(receivePort.sendPort);
}

/// Send an error event to the devtools server.
///
/// This is public so Flutter apps can call it from FlutterError.onError
/// and PlatformDispatcher.instance.onError handlers.
///
/// ```dart
/// FlutterError.onError = (details) {
///   sendError(
///     CloudMock.connection!,
///     details.exception,
///     details.stack ?? StackTrace.current,
///   );
/// };
/// ```
void sendError(Connection conn, Object error, StackTrace stackTrace) {
  String name;
  String message;

  if (error is Error) {
    name = error.runtimeType.toString();
    message = error.toString();
  } else if (error is Exception) {
    name = error.runtimeType.toString();
    message = error.toString();
  } else {
    name = 'UnknownError';
    message = error.toString();
  }

  conn.send('error:uncaught', {
    'name': name,
    'message': message,
    'stack': stackTrace.toString(),
  });
}
