// CloudMock Kotlin SDK — thin OTel wrapper for Android
// Provides: OkHttp interception, mobile RUM, BLE mesh topology
package io.cloudmock

data class CloudMockConfig(
    val endpoint: String = "http://10.0.2.2:4318", // Android emulator localhost
    val appName: String = "app",
    val enableRUM: Boolean = true,
    val enableBLE: Boolean = false,
    val sampleRate: Double = 1.0
)

/**
 * CloudMock is the main entry point for the Android SDK.
 * It initializes telemetry collection and provides a simple API for
 * tracking custom events, capturing errors, and flushing buffered data.
 *
 * Usage:
 * ```kotlin
 * CloudMock.initialize(CloudMockConfig(
 *     endpoint = "http://10.0.2.2:4318",
 *     appName = "my-app"
 * ))
 * CloudMock.track("checkout", mapOf("item_count" to "3"))
 * ```
 */
object CloudMock {
    private var config: CloudMockConfig? = null
    private var initialized = false

    /**
     * Initialize the CloudMock SDK with the given configuration.
     * Must be called before any other SDK methods, typically in Application.onCreate().
     */
    fun initialize(config: CloudMockConfig) {
        this.config = config
        this.initialized = true
        // TODO: Set up OTel TracerProvider, wire OkHttp interceptor,
        // start RUM session if enabled, start BLE mesh if enabled.
    }

    /**
     * Track a named event with optional string attributes.
     * Events are buffered and sent in batches to the configured endpoint.
     */
    fun track(name: String, attributes: Map<String, String> = emptyMap()) {
        if (!initialized) return
        // TODO: Create an OTel span or event with the given name and attributes.
    }

    /**
     * Capture a throwable and attach it to the current trace context.
     * The exception message and stack trace are recorded.
     */
    fun captureError(throwable: Throwable) {
        if (!initialized) return
        // TODO: Record error as an OTel span event with error attributes.
    }

    /**
     * Flush any buffered telemetry data to the endpoint immediately.
     * Call this before app termination or when going to background.
     */
    fun flush() {
        if (!initialized) return
        // TODO: Force-flush the OTel span exporter.
    }
}
