package io.cloudmock.sdk

/**
 * Forwards log messages to the CloudMock devtools connection.
 *
 * Can be used standalone via CloudMock.log() or integrated with
 * Timber (Android) as a custom Tree.
 *
 * Usage:
 * ```kotlin
 * // Direct usage
 * CloudMock.log("User signed in", LogLevel.INFO, tag = "Auth")
 *
 * // With Timber (if using Android)
 * Timber.plant(CloudMockTimberTree())
 * ```
 */
internal class LogInterceptor(private val connection: Connection) {

    /**
     * Send a log message to devtools.
     *
     * @param message The log message
     * @param level Log level
     * @param tag Optional tag for categorization
     */
    fun log(message: String, level: LogLevel = LogLevel.INFO, tag: String? = null) {
        val stackTrace = Thread.currentThread().stackTrace
        // Walk up the stack past log/CloudMock frames to find the caller
        val callerFrame = stackTrace.firstOrNull { frame ->
            !frame.className.startsWith("io.cloudmock.sdk") &&
            !frame.className.startsWith("java.lang.Thread")
        }

        val data = mutableMapOf<String, Any?>(
            "level" to level.value,
            "message" to message,
        )

        if (tag != null) {
            data["tag"] = tag
        }

        if (callerFrame != null) {
            data["file"] = callerFrame.fileName
            data["line"] = callerFrame.lineNumber
            data["className"] = callerFrame.className
            data["methodName"] = callerFrame.methodName
        }

        connection.send(
            SourceEvent(
                type = "console",
                data = data,
                source = connection.appName,
                runtime = "kotlin"
            )
        )
    }
}

/**
 * Timber Tree implementation for Android projects that use Timber.
 * Forwards all Timber logs to CloudMock devtools.
 *
 * Usage:
 * ```kotlin
 * Timber.plant(CloudMockTimberTree())
 * ```
 *
 * Note: This class can be used directly without depending on Timber.
 * It implements the same interface pattern. If Timber is not in your
 * classpath, use CloudMock.log() directly instead.
 */
class CloudMockTimberTree {
    fun log(priority: Int, tag: String?, message: String) {
        val level = when (priority) {
            2 -> LogLevel.DEBUG   // Log.VERBOSE
            3 -> LogLevel.DEBUG   // Log.DEBUG
            4 -> LogLevel.INFO    // Log.INFO
            5 -> LogLevel.WARN    // Log.WARN
            6 -> LogLevel.ERROR   // Log.ERROR
            7 -> LogLevel.ERROR   // Log.ASSERT
            else -> LogLevel.INFO
        }
        CloudMock.log(message, level, tag)
    }
}
