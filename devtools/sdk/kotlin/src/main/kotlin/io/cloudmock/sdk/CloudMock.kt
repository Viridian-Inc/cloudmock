package io.cloudmock.sdk

/**
 * Main entry point for the CloudMock devtools SDK.
 *
 * Usage:
 * ```kotlin
 * CloudMock.init(appName = "my-android-app")
 *
 * // Add OkHttp interceptor
 * val client = OkHttpClient.Builder()
 *     .addInterceptor(CloudMockInterceptor())
 *     .build()
 * ```
 */
object CloudMock {
    internal var connection: Connection? = null
        private set

    private var errorHandler: ErrorHandler? = null
    private var logInterceptor: LogInterceptor? = null
    private var isInitialized = false

    /**
     * Initialize the CloudMock SDK and connect to devtools.
     * No-ops if devtools isn't running.
     *
     * @param appName Name shown in the devtools source bar
     * @param host Devtools host (default: localhost)
     * @param port Devtools TCP port (default: 4580)
     */
    @JvmStatic
    @JvmOverloads
    fun init(appName: String, host: String = "localhost", port: Int = 4580) {
        if (isInitialized) return
        isInitialized = true

        val conn = Connection(host, port, appName)
        this.connection = conn

        // Register this source
        conn.send(
            SourceEvent(
                type = "source:register",
                data = mapOf(
                    "runtime" to "kotlin",
                    "appName" to appName,
                    "pid" to ProcessHandle.current().pid()
                ),
                source = appName,
                runtime = "kotlin"
            )
        )

        // Set up error handler
        errorHandler = ErrorHandler(conn).also { it.install() }

        // Set up log interceptor
        logInterceptor = LogInterceptor(conn)
    }

    /**
     * Log a message to devtools.
     *
     * @param message The log message
     * @param level Log level (default: INFO)
     * @param tag Optional tag for categorization
     */
    @JvmStatic
    @JvmOverloads
    fun log(message: String, level: LogLevel = LogLevel.INFO, tag: String? = null) {
        logInterceptor?.log(message, level, tag)
    }

    /**
     * Stop the SDK and disconnect from devtools.
     */
    @JvmStatic
    fun stop() {
        if (!isInitialized) return

        errorHandler?.uninstall()
        errorHandler = null

        logInterceptor = null

        connection?.close()
        connection = null
        isInitialized = false
    }
}

/**
 * Log levels supported by the CloudMock SDK.
 */
enum class LogLevel(val value: String) {
    DEBUG("debug"),
    INFO("info"),
    WARN("warn"),
    ERROR("error")
}
