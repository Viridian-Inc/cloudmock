import { spawn, ChildProcess } from "child_process";
import { createServer } from "net";
import { request as httpRequest } from "http";

export interface CloudMockOptions {
  port?: number;
  region?: string;
  profile?: string;
}

export class CloudMock {
  private process: ChildProcess | null = null;
  private readonly port: number;
  private readonly region: string;
  private readonly profile: string;

  constructor(options: CloudMockOptions = {}) {
    this.port = options.port ?? 0;
    this.region = options.region ?? "us-east-1";
    this.profile = options.profile ?? "minimal";
  }

  async start(): Promise<void> {
    if (this.process) return;

    if (this.port === 0) {
      (this as any).port = await findFreePort();
    }

    const env = {
      ...process.env,
      CLOUDMOCK_PROFILE: this.profile,
      CLOUDMOCK_IAM_MODE: "none",
    };

    this.process = spawn("npx", ["cloudmock", "--port", String(this.port)], {
      env,
      stdio: "ignore",
    });

    this.process.on("error", (err: Error) => {
      throw new Error(`Failed to start CloudMock: ${err.message}`);
    });

    await this.waitForReady(30_000);
  }

  async stop(): Promise<void> {
    if (this.process) {
      this.process.kill();
      this.process = null;
    }
  }

  get endpoint(): string {
    return `http://localhost:${this.port}`;
  }

  /**
   * Returns AWS SDK client configuration for this CloudMock instance.
   * Pass this to any AWS SDK v3 client constructor.
   */
  clientConfig() {
    return {
      endpoint: this.endpoint,
      region: this.region,
      credentials: { accessKeyId: "test", secretAccessKey: "test" },
      forcePathStyle: true, // For S3
    };
  }

  private waitForReady(timeoutMs: number): Promise<void> {
    return new Promise((resolve, reject) => {
      const deadline = Date.now() + timeoutMs;

      const check = () => {
        if (Date.now() > deadline) {
          reject(new Error("CloudMock did not start in time"));
          return;
        }

        const req = httpRequest(
          `http://localhost:${this.port}/`,
          { timeout: 1000 },
          (res: import("http").IncomingMessage) => {
            res.resume();
            resolve();
          }
        );
        req.on("error", () => setTimeout(check, 100));
        req.end();
      };

      check();
    });
  }
}

/**
 * Start CloudMock and return a configured instance.
 *
 * @example
 * ```ts
 * import { mockAWS } from "@cloudmock/sdk";
 * import { S3Client, CreateBucketCommand } from "@aws-sdk/client-s3";
 *
 * const cm = await mockAWS();
 * const s3 = new S3Client(cm.clientConfig());
 * await s3.send(new CreateBucketCommand({ Bucket: "test" }));
 * await cm.stop();
 * ```
 */
export async function mockAWS(options?: CloudMockOptions): Promise<CloudMock> {
  const cm = new CloudMock(options);
  await cm.start();
  return cm;
}

function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.listen(0, () => {
      const addr = srv.address();
      if (addr && typeof addr === "object") {
        const port = addr.port;
        srv.close(() => resolve(port));
      } else {
        reject(new Error("Could not find free port"));
      }
    });
  });
}
