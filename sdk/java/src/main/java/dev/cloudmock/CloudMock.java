package dev.cloudmock;

import java.io.IOException;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.URI;
import java.util.concurrent.TimeUnit;

/**
 * Manages a CloudMock server instance for local AWS emulation.
 *
 * <pre>{@code
 * try (CloudMock cm = CloudMock.start()) {
 *     S3Client s3 = S3Client.builder()
 *         .endpointOverride(cm.endpoint())
 *         .region(Region.US_EAST_1)
 *         .credentialsProvider(StaticCredentialsProvider.create(
 *             AwsBasicCredentials.create("test", "test")))
 *         .build();
 *     s3.createBucket(b -> b.bucket("test"));
 * }
 * }</pre>
 */
public class CloudMock implements AutoCloseable {
    private Process process;
    private final int port;
    private final String region;

    private CloudMock(int port, String region) {
        this.port = port;
        this.region = region;
    }

    /** Start with default options (auto port, us-east-1, minimal profile). */
    public static CloudMock start() throws IOException, InterruptedException {
        return start(new Options());
    }

    /** Start with custom options. */
    public static CloudMock start(Options opts) throws IOException, InterruptedException {
        int port = opts.port > 0 ? opts.port : findFreePort();
        CloudMock cm = new CloudMock(port, opts.region);

        ProcessBuilder pb = new ProcessBuilder("cloudmock", "--port", String.valueOf(port))
            .redirectOutput(ProcessBuilder.Redirect.DISCARD)
            .redirectError(ProcessBuilder.Redirect.DISCARD);
        pb.environment().put("CLOUDMOCK_PROFILE", opts.profile);
        pb.environment().put("CLOUDMOCK_IAM_MODE", "none");

        cm.process = pb.start();
        cm.waitForReady(30_000);
        return cm;
    }

    /** The HTTP endpoint URI. */
    public URI endpoint() {
        return URI.create("http://localhost:" + port);
    }

    /** The port number. */
    public int port() {
        return port;
    }

    /** The configured AWS region. */
    public String region() {
        return region;
    }

    /** Stop the server. */
    @Override
    public void close() {
        if (process != null && process.isAlive()) {
            process.destroy();
            try {
                process.waitFor(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                process.destroyForcibly();
            }
        }
    }

    private void waitForReady(long timeoutMs) throws InterruptedException {
        long deadline = System.currentTimeMillis() + timeoutMs;
        while (System.currentTimeMillis() < deadline) {
            try (Socket s = new Socket("127.0.0.1", port)) {
                return; // connected — server is ready
            } catch (IOException e) {
                if (!process.isAlive()) {
                    throw new RuntimeException("CloudMock process exited with code " + process.exitValue());
                }
                Thread.sleep(100);
            }
        }
        throw new RuntimeException("CloudMock did not start within " + timeoutMs + "ms");
    }

    private static int findFreePort() throws IOException {
        try (ServerSocket ss = new ServerSocket(0)) {
            return ss.getLocalPort();
        }
    }

    /** Configuration options for CloudMock. */
    public static class Options {
        public int port = 0;
        public String region = "us-east-1";
        public String profile = "minimal";

        public Options port(int port) { this.port = port; return this; }
        public Options region(String region) { this.region = region; return this; }
        public Options profile(String profile) { this.profile = profile; return this; }
    }
}
