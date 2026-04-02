package dev.cloudmock

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*

class CloudMockTest {
    @Test
    fun `endpoint format`() {
        val cm = CloudMock(port = 4566)
        assertEquals("http://localhost:4566", cm.endpoint.toString())
    }

    @Test
    fun `auto port`() {
        val cm = CloudMock()
        assertTrue(cm.port > 0)
    }
}
