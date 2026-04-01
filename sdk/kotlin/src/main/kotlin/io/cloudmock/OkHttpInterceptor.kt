// OkHttpInterceptor — automatic instrumentation for OkHttp
//
// This interceptor wraps OkHttp requests to automatically create OTel spans
// for every HTTP request made by the app. It captures:
//   - Request method, URL, and headers
//   - Response status code and content length
//   - Timing (request start, TTFB, total duration)
//   - Error details when requests fail
//
// Usage:
//   val client = OkHttpClient.Builder()
//       .addInterceptor(CloudMockOkHttpInterceptor())
//       .build()
//
// The interceptor injects W3C trace context headers (traceparent, tracestate)
// into outgoing requests for distributed trace propagation.
package io.cloudmock

/**
 * OkHttp interceptor that automatically creates OTel spans for HTTP requests.
 * Add this to your OkHttpClient.Builder to get automatic tracing.
 */
class CloudMockOkHttpInterceptor {

    /**
     * Intercept the request, create a span, inject trace context, and record the response.
     * This follows the OkHttp Interceptor contract (Chain.proceed pattern).
     */
    fun intercept(/* chain: Interceptor.Chain */): Any /* Response */ {
        // TODO: Implement OkHttp Interceptor interface:
        // 1. Start an OTel span with the request method and URL
        // 2. Inject W3C traceparent header into the request
        // 3. Proceed with the request via chain.proceed()
        // 4. Record response status code and timing on the span
        // 5. Record error details if the request fails
        // 6. End the span
        throw NotImplementedError("Stub — requires okhttp3 dependency")
    }

    /**
     * Inject W3C trace context headers into the request builder.
     * Called automatically by intercept(); can also be used manually.
     */
    fun injectTraceContext(/* requestBuilder: Request.Builder */) {
        // TODO: Read current span context, format as traceparent/tracestate,
        // add to request headers.
    }
}
