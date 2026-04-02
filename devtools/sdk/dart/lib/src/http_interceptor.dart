import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:dio/dio.dart';

import 'client.dart';
import 'connection.dart';

int _requestCounter = 0;

String _nextId() {
  _requestCounter++;
  return 'dart_req_${_requestCounter}_${DateTime.now().millisecondsSinceEpoch}';
}

/// HttpOverrides that captures all HTTP traffic made via dart:io HttpClient.
///
/// Usage:
/// ```dart
/// CloudMock.init(appName: 'my-app');
/// HttpOverrides.global = CloudMockHttpOverrides();
/// ```
///
/// This captures all HTTP requests made through dart:io, including those from
/// the `http` package and any library that uses HttpClient internally.
class CloudMockHttpOverrides extends HttpOverrides {
  final HttpOverrides? _previous;

  CloudMockHttpOverrides() : _previous = HttpOverrides.current;

  @override
  HttpClient createHttpClient(SecurityContext? context) {
    final inner = _previous?.createHttpClient(context) ?? super.createHttpClient(context);
    return _CloudMockHttpClient(inner);
  }
}

/// Wrapping HttpClient that intercepts open/openUrl calls.
class _CloudMockHttpClient implements HttpClient {
  final HttpClient _inner;

  _CloudMockHttpClient(this._inner);

  Future<HttpClientRequest> _wrapOpen(
    Future<HttpClientRequest> Function() openFn,
    String method,
    String url,
    String path,
  ) async {
    final conn = CloudMock.connection;
    if (conn == null) return openFn();

    final id = _nextId();
    final start = DateTime.now();

    HttpClientRequest request;
    try {
      request = await openFn();
    } catch (e) {
      final duration = DateTime.now().difference(start).inMilliseconds;
      conn.send('http:error', {
        'id': id,
        'method': method,
        'url': url,
        'path': path,
        'error': e.toString(),
        'duration_ms': duration,
      });
      rethrow;
    }

    // Inject correlation headers
    request.headers.set('X-CloudMock-Source', conn.appName);
    request.headers.set('X-CloudMock-Request-Id', id);

    return _CloudMockHttpClientRequest(request, conn, id, method, url, path, start);
  }

  @override
  Future<HttpClientRequest> open(String method, String host, int port, String path) {
    final url = '${port == 443 ? "https" : "http"}://$host:$port$path';
    return _wrapOpen(() => _inner.open(method, host, port, path), method, url, path);
  }

  @override
  Future<HttpClientRequest> openUrl(String method, Uri url) {
    return _wrapOpen(() => _inner.openUrl(method, url), method, url.toString(), url.path);
  }

  @override
  Future<HttpClientRequest> get(String host, int port, String path) =>
      open('GET', host, port, path);

  @override
  Future<HttpClientRequest> getUrl(Uri url) => openUrl('GET', url);

  @override
  Future<HttpClientRequest> post(String host, int port, String path) =>
      open('POST', host, port, path);

  @override
  Future<HttpClientRequest> postUrl(Uri url) => openUrl('POST', url);

  @override
  Future<HttpClientRequest> put(String host, int port, String path) =>
      open('PUT', host, port, path);

  @override
  Future<HttpClientRequest> putUrl(Uri url) => openUrl('PUT', url);

  @override
  Future<HttpClientRequest> delete(String host, int port, String path) =>
      open('DELETE', host, port, path);

  @override
  Future<HttpClientRequest> deleteUrl(Uri url) => openUrl('DELETE', url);

  @override
  Future<HttpClientRequest> patch(String host, int port, String path) =>
      open('PATCH', host, port, path);

  @override
  Future<HttpClientRequest> patchUrl(Uri url) => openUrl('PATCH', url);

  @override
  Future<HttpClientRequest> head(String host, int port, String path) =>
      open('HEAD', host, port, path);

  @override
  Future<HttpClientRequest> headUrl(Uri url) => openUrl('HEAD', url);

  // Delegate all other properties and methods to _inner

  @override
  set autoUncompress(bool value) => _inner.autoUncompress = value;
  @override
  bool get autoUncompress => _inner.autoUncompress;

  @override
  set connectionTimeout(Duration? value) => _inner.connectionTimeout = value;
  @override
  Duration? get connectionTimeout => _inner.connectionTimeout;

  @override
  set idleTimeout(Duration value) => _inner.idleTimeout = value;
  @override
  Duration get idleTimeout => _inner.idleTimeout;

  @override
  set maxConnectionsPerHost(int? value) => _inner.maxConnectionsPerHost = value;
  @override
  int? get maxConnectionsPerHost => _inner.maxConnectionsPerHost;

  @override
  set userAgent(String? value) => _inner.userAgent = value;
  @override
  String? get userAgent => _inner.userAgent;

  @override
  void addCredentials(Uri url, String realm, HttpClientCredentials credentials) =>
      _inner.addCredentials(url, realm, credentials);

  @override
  void addProxyCredentials(
          String host, int port, String realm, HttpClientCredentials credentials) =>
      _inner.addProxyCredentials(host, port, realm, credentials);

  @override
  set authenticate(Future<bool> Function(Uri url, String scheme, String? realm)? f) =>
      _inner.authenticate = f;

  @override
  set authenticateProxy(
          Future<bool> Function(String host, int port, String scheme, String? realm)? f) =>
      _inner.authenticateProxy = f;

  @override
  set badCertificateCallback(bool Function(X509Certificate cert, String host, int port)? callback) =>
      _inner.badCertificateCallback = callback;

  @override
  set findProxy(String Function(Uri url)? f) => _inner.findProxy = f;

  @override
  void close({bool force = false}) => _inner.close(force: force);

  @override
  set connectionFactory(
          Future<ConnectionTask<Socket>> Function(Uri url, String? proxyHost, int? proxyPort)?
              f) =>
      _inner.connectionFactory = f;

  @override
  set keyLog(Function(String line)? callback) => _inner.keyLog = callback;
}

/// Wrapping HttpClientRequest that captures the response.
class _CloudMockHttpClientRequest implements HttpClientRequest {
  final HttpClientRequest _inner;
  final Connection _conn;
  final String _id;
  final String _method;
  final String _url;
  final String _path;
  final DateTime _start;

  _CloudMockHttpClientRequest(
    this._inner,
    this._conn,
    this._id,
    this._method,
    this._url,
    this._path,
    this._start,
  );

  @override
  Future<HttpClientResponse> close() async {
    HttpClientResponse response;
    try {
      response = await _inner.close();
    } catch (e) {
      final duration = DateTime.now().difference(_start).inMilliseconds;
      _conn.send('http:error', {
        'id': _id,
        'method': _method,
        'url': _url,
        'path': _path,
        'error': e.toString(),
        'duration_ms': duration,
      });
      rethrow;
    }

    final duration = DateTime.now().difference(_start).inMilliseconds;

    // Capture response headers
    final respHeaders = <String, String>{};
    response.headers.forEach((name, values) {
      respHeaders[name] = values.join(', ');
    });

    // Capture request headers
    final reqHeaders = <String, String>{};
    _inner.headers.forEach((name, values) {
      reqHeaders[name] = values.join(', ');
    });

    // Read body in background without consuming the stream for the caller
    // We use a transformer that tees the data
    final bodyChunks = <int>[];
    const maxCapture = 4096;

    final transformedStream = response.transform(
      StreamTransformer<List<int>, List<int>>.fromHandlers(
        handleData: (data, sink) {
          if (bodyChunks.length < maxCapture) {
            final remaining = maxCapture - bodyChunks.length;
            bodyChunks.addAll(data.take(remaining));
          }
          sink.add(data);
        },
        handleDone: (sink) {
          // Send the event when the response is fully read
          String bodyStr;
          try {
            bodyStr = utf8.decode(bodyChunks, allowMalformed: true);
          } catch (_) {
            bodyStr = '';
          }

          _conn.send('http:response', {
            'id': _id,
            'method': _method,
            'url': _url,
            'path': _path,
            'status': response.statusCode,
            'duration_ms': duration,
            'request_headers': reqHeaders,
            'response_headers': respHeaders,
            'response_body': bodyStr,
            'content_length': response.contentLength.toString(),
          });

          sink.close();
        },
      ),
    );

    return _CloudMockHttpClientResponse(response, transformedStream);
  }

  // Delegate all other methods to _inner

  @override
  bool get bufferOutput => _inner.bufferOutput;
  @override
  set bufferOutput(bool value) => _inner.bufferOutput = value;

  @override
  int get contentLength => _inner.contentLength;
  @override
  set contentLength(int value) => _inner.contentLength = value;

  @override
  Encoding get encoding => _inner.encoding;
  @override
  set encoding(Encoding value) => _inner.encoding = value;

  @override
  bool get followRedirects => _inner.followRedirects;
  @override
  set followRedirects(bool value) => _inner.followRedirects = value;

  @override
  int get maxRedirects => _inner.maxRedirects;
  @override
  set maxRedirects(int value) => _inner.maxRedirects = value;

  @override
  bool get persistentConnection => _inner.persistentConnection;
  @override
  set persistentConnection(bool value) => _inner.persistentConnection = value;

  @override
  HttpHeaders get headers => _inner.headers;

  @override
  HttpConnectionInfo? get connectionInfo => _inner.connectionInfo;

  @override
  List<Cookie> get cookies => _inner.cookies;

  @override
  Future<HttpClientResponse> get done => _inner.done;

  @override
  String get method => _inner.method;

  @override
  Uri get uri => _inner.uri;

  @override
  void abort([Object? exception, StackTrace? stackTrace]) =>
      _inner.abort(exception, stackTrace);

  @override
  void add(List<int> data) => _inner.add(data);

  @override
  void addError(Object error, [StackTrace? stackTrace]) =>
      _inner.addError(error, stackTrace);

  @override
  Future<void> addStream(Stream<List<int>> stream) => _inner.addStream(stream);

  @override
  Future<void> flush() => _inner.flush();

  @override
  void write(Object? object) => _inner.write(object);

  @override
  void writeAll(Iterable<Object?> objects, [String separator = '']) =>
      _inner.writeAll(objects, separator);

  @override
  void writeCharCode(int charCode) => _inner.writeCharCode(charCode);

  @override
  void writeln([Object? object = '']) => _inner.writeln(object);
}

/// Wrapping HttpClientResponse that uses the transformed stream.
class _CloudMockHttpClientResponse extends Stream<List<int>>
    implements HttpClientResponse {
  final HttpClientResponse _inner;
  final Stream<List<int>> _transformedStream;

  _CloudMockHttpClientResponse(this._inner, this._transformedStream);

  @override
  StreamSubscription<List<int>> listen(
    void Function(List<int>)? onData, {
    Function? onError,
    void Function()? onDone,
    bool? cancelOnError,
  }) {
    return _transformedStream.listen(
      onData,
      onError: onError,
      onDone: onDone,
      cancelOnError: cancelOnError,
    );
  }

  // Delegate all properties to _inner

  @override
  X509Certificate? get certificate => _inner.certificate;

  @override
  HttpClientResponseCompressionState get compressionState => _inner.compressionState;

  @override
  HttpConnectionInfo? get connectionInfo => _inner.connectionInfo;

  @override
  int get contentLength => _inner.contentLength;

  @override
  List<Cookie> get cookies => _inner.cookies;

  @override
  Future<Socket> detachSocket() => _inner.detachSocket();

  @override
  HttpHeaders get headers => _inner.headers;

  @override
  bool get isRedirect => _inner.isRedirect;

  @override
  bool get persistentConnection => _inner.persistentConnection;

  @override
  String get reasonPhrase => _inner.reasonPhrase;

  @override
  Future<HttpClientResponse> redirect(
          [String? method, Uri? url, bool? followLoops]) =>
      _inner.redirect(method, url, followLoops);

  @override
  List<RedirectInfo> get redirects => _inner.redirects;

  @override
  int get statusCode => _inner.statusCode;
}

/// Dio interceptor that captures HTTP requests and responses.
///
/// Usage:
/// ```dart
/// final dio = Dio();
/// dio.interceptors.add(CloudMockDioInterceptor());
/// ```
class CloudMockDioInterceptor extends Interceptor {
  final Map<RequestOptions, _DioRequestInfo> _pending = {};

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    final conn = CloudMock.connection;
    if (conn == null) {
      handler.next(options);
      return;
    }

    final id = _nextId();
    final start = DateTime.now();

    // Inject correlation headers
    options.headers['X-CloudMock-Source'] = conn.appName;
    options.headers['X-CloudMock-Request-Id'] = id;

    _pending[options] = _DioRequestInfo(id, start);
    handler.next(options);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    final conn = CloudMock.connection;
    final info = _pending.remove(response.requestOptions);

    if (conn != null && info != null) {
      final duration = DateTime.now().difference(info.start).inMilliseconds;
      final options = response.requestOptions;

      // Capture response body (up to 4KB)
      String bodyStr = '';
      try {
        final data = response.data;
        if (data is String) {
          bodyStr = data.length > 4096 ? data.substring(0, 4096) : data;
        } else if (data != null) {
          final encoded = jsonEncode(data);
          bodyStr = encoded.length > 4096 ? encoded.substring(0, 4096) : encoded;
        }
      } catch (_) {}

      conn.send('http:response', {
        'id': info.id,
        'method': options.method,
        'url': options.uri.toString(),
        'path': options.uri.path,
        'status': response.statusCode,
        'duration_ms': duration,
        'request_headers': options.headers,
        'response_headers': response.headers.map,
        'response_body': bodyStr,
        'content_length': response.headers.value('content-length'),
      });
    }

    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    final conn = CloudMock.connection;
    final info = _pending.remove(err.requestOptions);

    if (conn != null && info != null) {
      final duration = DateTime.now().difference(info.start).inMilliseconds;
      final options = err.requestOptions;

      // If there's a response (e.g., 4xx/5xx), send it
      if (err.response != null) {
        String bodyStr = '';
        try {
          final data = err.response?.data;
          if (data is String) {
            bodyStr = data.length > 4096 ? data.substring(0, 4096) : data;
          } else if (data != null) {
            final encoded = jsonEncode(data);
            bodyStr = encoded.length > 4096 ? encoded.substring(0, 4096) : encoded;
          }
        } catch (_) {}

        conn.send('http:response', {
          'id': info.id,
          'method': options.method,
          'url': options.uri.toString(),
          'path': options.uri.path,
          'status': err.response?.statusCode,
          'duration_ms': duration,
          'request_headers': options.headers,
          'response_headers': err.response?.headers.map,
          'response_body': bodyStr,
        });
      } else {
        conn.send('http:error', {
          'id': info.id,
          'method': options.method,
          'url': options.uri.toString(),
          'path': options.uri.path,
          'error': err.message ?? err.toString(),
          'duration_ms': duration,
        });
      }
    }

    handler.next(err);
  }
}

class _DioRequestInfo {
  final String id;
  final DateTime start;
  _DioRequestInfo(this.id, this.start);
}
