// URLSessionInterceptor — automatic instrumentation for URLSession
//
// This interceptor wraps URLSession to automatically create OTel spans for
// every HTTP request made by the app. It captures:
//   - Request method, URL, and headers
//   - Response status code and content length
//   - Timing (request start, TTFB, total duration)
//   - Error details when requests fail
//
// Usage:
//   let session = URLSessionInterceptor.instrumented()
//   // Use `session` instead of URLSession.shared
//
// The interceptor uses URLProtocol subclassing to transparently intercept
// requests without requiring changes to existing networking code.

import Foundation

/// URLSessionInterceptor provides automatic HTTP tracing for iOS/macOS apps.
/// It creates OTel-compatible spans for every outgoing URLSession request.
public class URLSessionInterceptor: URLProtocol {

    /// Returns a URLSession configured with automatic tracing.
    /// All requests made through this session will generate spans.
    public static func instrumented() -> URLSession {
        let config = URLSessionConfiguration.default
        // TODO: Register custom URLProtocol that wraps requests with OTel spans.
        return URLSession(configuration: config)
    }

    /// Injects W3C trace context headers (traceparent, tracestate) into the request.
    /// Called automatically by the interceptor; can also be used manually.
    public static func injectTraceContext(into request: inout URLRequest) {
        // TODO: Read current span context, format as W3C traceparent header,
        // add to request headers.
    }

    // MARK: - URLProtocol overrides (stubs)

    override public class func canInit(with request: URLRequest) -> Bool {
        // TODO: Return true for requests we should instrument.
        return false
    }

    override public class func canonicalRequest(for request: URLRequest) -> URLRequest {
        return request
    }

    override public func startLoading() {
        // TODO: Start an OTel span, inject trace context, forward request,
        // record response/error, end span.
    }

    override public func stopLoading() {
        // TODO: Cancel in-flight request if needed.
    }
}
