package dev.cloudmock;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class CloudMockTest {

    @Test
    void endpointFormat() {
        // Test that endpoint URI is well-formed without starting the server
        var uri = java.net.URI.create("http://localhost:4566");
        assertEquals("localhost", uri.getHost());
        assertEquals(4566, uri.getPort());
        assertEquals("http", uri.getScheme());
    }

    @Test
    void optionsDefaults() {
        var opts = new CloudMock.Options();
        assertEquals(0, opts.port);
        assertEquals("us-east-1", opts.region);
        assertEquals("minimal", opts.profile);
    }

    @Test
    void optionsBuilder() {
        var opts = new CloudMock.Options()
            .port(5555)
            .region("eu-west-1")
            .profile("standard");
        assertEquals(5555, opts.port);
        assertEquals("eu-west-1", opts.region);
        assertEquals("standard", opts.profile);
    }
}
