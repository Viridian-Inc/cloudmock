package io.cloudmock.sdk

import okhttp3.Interceptor
import okhttp3.Response
import okio.Buffer
import java.util.concurrent.atomic.AtomicInteger

/**
 * OkHttp Interceptor that captures HTTP request/response details and sends them to devtools.
 *
 * Usage:
 * ```kotlin
 * val client = OkHttpClient.Builder()
 *     .addInterceptor(CloudMockInterceptor())
 *     .build()
 * ```
 */
class CloudMockInterceptor : Interceptor {
    companion object {
        private val requestCounter = AtomicInteger(0)
    }

    override fun intercept(chain: Interceptor.Chain): Response {
        val conn = CloudMock.connection ?: return chain.proceed(chain.request())

        val id = "req_${requestCounter.incrementAndGet()}_${System.currentTimeMillis()}"
        val startTime = System.currentTimeMillis()

        // Inject correlation headers
        val request = chain.request().newBuilder()
            .header("X-CloudMock-Source", conn.appName)
            .header("X-CloudMock-Request-Id", id)
            .build()

        val method = request.method
        val url = request.url.toString()
        val path = request.url.encodedPath

        val response: Response
        try {
            response = chain.proceed(request)
        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            conn.send(
                SourceEvent(
                    type = "http:error",
                    data = mapOf(
                        "id" to id,
                        "method" to method,
                        "url" to url,
                        "path" to path,
                        "error" to (e.message ?: e.toString()),
                        "duration_ms" to duration
                    ),
                    source = conn.appName,
                    runtime = "kotlin"
                )
            )
            throw e
        }

        val duration = System.currentTimeMillis() - startTime

        // Capture request headers
        val requestHeaders = mutableMapOf<String, String>()
        for (name in request.headers.names()) {
            requestHeaders[name] = request.headers[name] ?: ""
        }

        // Capture response headers
        val responseHeaders = mutableMapOf<String, String>()
        for (name in response.headers.names()) {
            responseHeaders[name] = response.headers[name] ?: ""
        }

        // Capture response body (first 4KB) without consuming it
        val responseBody = try {
            val source = response.body?.source()
            source?.request(4096)
            val buffer = source?.buffer?.clone() ?: Buffer()
            val bodyBytes = buffer.readByteArray(minOf(buffer.size, 4096))
            String(bodyBytes, Charsets.UTF_8)
        } catch (_: Exception) {
            ""
        }

        conn.send(
            SourceEvent(
                type = "http:response",
                data = mapOf(
                    "id" to id,
                    "method" to method,
                    "url" to url,
                    "path" to path,
                    "status" to response.code,
                    "duration_ms" to duration,
                    "request_headers" to requestHeaders,
                    "response_headers" to responseHeaders,
                    "response_body" to responseBody,
                    "content_length" to (responseHeaders["Content-Length"] ?: "")
                ),
                source = conn.appName,
                runtime = "kotlin"
            )
        )

        return response
    }
}
