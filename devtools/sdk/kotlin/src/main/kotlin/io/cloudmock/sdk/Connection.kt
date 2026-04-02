package io.cloudmock.sdk

import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import java.io.BufferedWriter
import java.io.OutputStreamWriter
import java.net.Socket
import java.util.concurrent.ConcurrentLinkedQueue
import java.util.concurrent.atomic.AtomicBoolean

/**
 * TCP JSON-line client that connects to the devtools source server.
 * Runs on a background daemon thread to avoid blocking the main thread.
 * Auto-reconnects every 5 seconds and buffers up to 100 messages when disconnected.
 */
internal class Connection(
    private val host: String,
    private val port: Int,
    val appName: String
) {
    private var socket: Socket? = null
    private var writer: BufferedWriter? = null
    private val connected = AtomicBoolean(false)
    private val closed = AtomicBoolean(false)
    private val buffer = ConcurrentLinkedQueue<String>()
    private val maxBufferSize = 100

    private val json = Json { encodeDefaults = true }

    init {
        startConnectionThread()
    }

    private fun startConnectionThread() {
        val thread = Thread {
            while (!closed.get()) {
                try {
                    connect()
                } catch (_: Exception) {
                    // Connection failed; wait and retry
                }

                if (!closed.get()) {
                    try {
                        Thread.sleep(5000)
                    } catch (_: InterruptedException) {
                        break
                    }
                }
            }
        }
        thread.isDaemon = true
        thread.name = "cloudmock-connection"
        thread.start()
    }

    private fun connect() {
        if (closed.get()) return

        val sock = Socket(host, port)
        socket = sock
        writer = BufferedWriter(OutputStreamWriter(sock.getOutputStream(), Charsets.UTF_8))
        connected.set(true)

        // Flush buffered messages
        flushBuffer()

        // Keep connection alive by reading (blocks until disconnect)
        try {
            val input = sock.getInputStream()
            val buf = ByteArray(256)
            while (!closed.get() && !sock.isClosed) {
                val bytesRead = input.read(buf)
                if (bytesRead == -1) break
                // Discard any server data; we're send-only
            }
        } catch (_: Exception) {
            // Connection lost
        } finally {
            connected.set(false)
            closeSocket()
        }
    }

    /**
     * Send an event to the devtools server.
     * Thread-safe; buffers messages if disconnected.
     */
    fun send(event: SourceEvent) {
        val line = serializeEvent(event) ?: return
        val message = "$line\n"

        if (connected.get()) {
            try {
                synchronized(this) {
                    writer?.write(message)
                    writer?.flush()
                }
                return
            } catch (_: Exception) {
                // Write failed; buffer it
            }
        }

        // Buffer up to maxBufferSize messages
        if (buffer.size < maxBufferSize) {
            buffer.offer(message)
        }
    }

    private fun flushBuffer() {
        val w = writer ?: return
        try {
            synchronized(this) {
                while (true) {
                    val msg = buffer.poll() ?: break
                    w.write(msg)
                }
                w.flush()
            }
        } catch (_: Exception) {
            // Flush failed
        }
    }

    private fun serializeEvent(event: SourceEvent): String? {
        return try {
            val obj = buildJsonObject {
                put("type", event.type)
                put("source", event.source)
                put("runtime", event.runtime)
                put("timestamp", event.timestamp)
                put("data", serializeMap(event.data))
            }
            json.encodeToString(obj)
        } catch (_: Exception) {
            null
        }
    }

    private fun serializeMap(map: Map<String, Any?>): JsonElement {
        return buildJsonObject {
            for ((key, value) in map) {
                put(key, toJsonElement(value))
            }
        }
    }

    private fun toJsonElement(value: Any?): JsonElement {
        return when (value) {
            null -> JsonNull
            is String -> JsonPrimitive(value)
            is Number -> JsonPrimitive(value)
            is Boolean -> JsonPrimitive(value)
            is Map<*, *> -> buildJsonObject {
                @Suppress("UNCHECKED_CAST")
                for ((k, v) in value as Map<String, Any?>) {
                    put(k, toJsonElement(v))
                }
            }
            is List<*> -> buildJsonArray {
                for (item in value) {
                    add(toJsonElement(item))
                }
            }
            else -> JsonPrimitive(value.toString())
        }
    }

    private fun closeSocket() {
        try {
            writer?.close()
        } catch (_: Exception) {}
        try {
            socket?.close()
        } catch (_: Exception) {}
        writer = null
        socket = null
    }

    /**
     * Close the connection and stop reconnection attempts.
     */
    fun close() {
        closed.set(true)
        connected.set(false)
        closeSocket()
        buffer.clear()
    }
}

/**
 * Event sent to the devtools source server.
 */
internal data class SourceEvent(
    val type: String,
    val data: Map<String, Any?>,
    val source: String,
    val runtime: String,
    val timestamp: Long = System.currentTimeMillis()
)
