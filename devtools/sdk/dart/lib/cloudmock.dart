/// CloudMock devtools SDK for Dart/Flutter.
///
/// Connects to the CloudMock devtools source server and sends telemetry for
/// HTTP traffic, log messages, and uncaught errors. Designed to be a silent
/// no-op when devtools is not running.
///
/// Usage:
/// ```dart
/// import 'package:cloudmock/cloudmock.dart';
///
/// void main() {
///   CloudMock.init(appName: 'my-flutter-app');
///   // ... your app
/// }
/// ```
library cloudmock;

export 'src/client.dart';
export 'src/http_interceptor.dart';
export 'src/error_handler.dart';
