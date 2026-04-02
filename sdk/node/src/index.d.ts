/**
 * Options for configuring a CloudMock instance.
 */
export interface CloudMockOptions {
  /**
   * TCP port for the CloudMock process to listen on.
   * Defaults to a random free port when omitted or set to `0`.
   */
  port?: number;

  /**
   * AWS region to report to SDK clients.
   * Defaults to `"us-east-1"`.
   */
  region?: string;

  /**
   * CloudMock service profile to enable (e.g. `"minimal"`, `"full"`).
   * Defaults to `"minimal"`.
   */
  profile?: string;
}

/**
 * Configuration object returned by {@link CloudMock.clientConfig}.
 * Compatible with any AWS SDK v3 client constructor.
 */
export interface AWSClientConfig {
  endpoint: string;
  region: string;
  credentials: {
    accessKeyId: string;
    secretAccessKey: string;
  };
  /** Always `true` — required for S3 path-style bucket addressing. */
  forcePathStyle: boolean;
}

/**
 * Manages the lifecycle of a local CloudMock process and exposes helpers
 * for wiring AWS SDK v3 clients to it.
 *
 * @example
 * ```ts
 * import { CloudMock } from "@cloudmock/sdk";
 * import { S3Client } from "@aws-sdk/client-s3";
 *
 * const cm = new CloudMock({ port: 4566 });
 * await cm.start();
 * const s3 = new S3Client(cm.clientConfig());
 * // ... run tests ...
 * await cm.stop();
 * ```
 */
export declare class CloudMock {
  constructor(options?: CloudMockOptions);

  /**
   * Spawns the CloudMock binary and waits until it is ready to serve requests.
   * If `port` was `0` or omitted, a free port is chosen automatically before
   * the process is started.
   */
  start(): Promise<void>;

  /**
   * Kills the CloudMock process. Safe to call even when not running.
   */
  stop(): Promise<void>;

  /**
   * Base URL of the running instance, e.g. `"http://localhost:4566"`.
   * Only valid after {@link start} has resolved.
   */
  readonly endpoint: string;

  /**
   * Returns an AWS SDK v3 client configuration object pre-wired to this
   * CloudMock instance. Pass it directly to any client constructor.
   *
   * @example
   * ```ts
   * const ddb = new DynamoDBClient(cm.clientConfig());
   * ```
   */
  clientConfig(): AWSClientConfig;
}

/**
 * Convenience helper that creates a {@link CloudMock} instance, starts it,
 * and returns it.
 *
 * @example
 * ```ts
 * import { mockAWS } from "@cloudmock/sdk";
 * import { S3Client, CreateBucketCommand } from "@aws-sdk/client-s3";
 *
 * const cm = await mockAWS();
 * const s3 = new S3Client(cm.clientConfig());
 * await s3.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
 * await cm.stop();
 * ```
 */
export declare function mockAWS(options?: CloudMockOptions): Promise<CloudMock>;
