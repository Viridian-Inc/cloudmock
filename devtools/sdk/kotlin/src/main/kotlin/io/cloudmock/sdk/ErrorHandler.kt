package io.cloudmock.sdk

import java.io.PrintWriter
import java.io.StringWriter

/**
 * Captures uncaught exceptions on all threads and sends them to devtools.
 * Chains with any previously installed UncaughtExceptionHandler.
 */
internal class ErrorHandler(private val connection: Connection) {
    private var previousHandler: Thread.UncaughtExceptionHandler? = null
    private var isInstalled = false

    /**
     * Install the global uncaught exception handler.
     */
    fun install() {
        if (isInstalled) return
        isInstalled = true

        // Save previous handler so we can chain
        previousHandler = Thread.getDefaultUncaughtExceptionHandler()

        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            handleException(thread, throwable)

            // Chain to previous handler (important for crash reporters, Android default, etc.)
            previousHandler?.uncaughtException(thread, throwable)
        }
    }

    /**
     * Uninstall and restore the previous exception handler.
     */
    fun uninstall() {
        if (!isInstalled) return
        isInstalled = false

        Thread.setDefaultUncaughtExceptionHandler(previousHandler)
        previousHandler = null
    }

    private fun handleException(thread: Thread, throwable: Throwable) {
        val sw = StringWriter()
        throwable.printStackTrace(PrintWriter(sw))
        val stackTrace = sw.toString()

        connection.send(
            SourceEvent(
                type = "error:uncaught",
                data = mapOf(
                    "name" to (throwable::class.simpleName ?: "Unknown"),
                    "message" to (throwable.message ?: throwable.toString()),
                    "stack" to stackTrace,
                    "thread" to thread.name,
                    "threadId" to thread.id
                ),
                source = connection.appName,
                runtime = "kotlin"
            )
        )

        // Give the connection a moment to send the event before the process exits
        try {
            Thread.sleep(500)
        } catch (_: InterruptedException) {
            // Ignore
        }
    }
}
