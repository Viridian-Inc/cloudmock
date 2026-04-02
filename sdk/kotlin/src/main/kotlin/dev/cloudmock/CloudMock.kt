package dev.cloudmock

import java.io.File
import java.net.ServerSocket
import java.net.Socket
import java.net.URI

class CloudMock(
    val port: Int = findFreePort(),
    val region: String = "us-east-1",
    private val profile: String = "minimal"
) : AutoCloseable {

    private var process: Process? = null

    val endpoint: URI get() = URI.create("http://localhost:$port")

    fun start(): CloudMock {
        if (process != null) return this

        process = ProcessBuilder("cloudmock", "--port", port.toString())
            .apply {
                environment()["CLOUDMOCK_PROFILE"] = profile
                environment()["CLOUDMOCK_IAM_MODE"] = "none"
            }
            .redirectOutput(ProcessBuilder.Redirect.DISCARD)
            .redirectError(ProcessBuilder.Redirect.DISCARD)
            .start()

        waitForReady(30_000)
        return this
    }

    override fun close() {
        process?.destroy()
        process?.waitFor(5, java.util.concurrent.TimeUnit.SECONDS)
        process = null
    }

    private fun waitForReady(timeoutMs: Long) {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (System.currentTimeMillis() < deadline) {
            try {
                Socket("127.0.0.1", port).close()
                return
            } catch (e: Exception) {
                Thread.sleep(100)
            }
        }
        throw RuntimeException("CloudMock did not start within ${timeoutMs}ms")
    }

    companion object {
        fun start(port: Int = findFreePort(), region: String = "us-east-1"): CloudMock {
            return CloudMock(port = port, region = region).start()
        }

        private fun findFreePort(): Int = ServerSocket(0).use { it.localPort }
    }
}
