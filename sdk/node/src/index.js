"use strict";

const { spawn } = require("child_process");
const { createServer } = require("net");
const { request: httpRequest } = require("http");

/**
 * Finds a free TCP port by briefly binding a server on port 0.
 * @returns {Promise<number>}
 */
function findFreePort() {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.listen(0, () => {
      const addr = srv.address();
      if (addr && typeof addr === "object") {
        const port = addr.port;
        srv.close(() => resolve(port));
      } else {
        reject(new Error("Could not determine a free port"));
      }
    });
    srv.on("error", reject);
  });
}

/**
 * Polls the CloudMock health endpoint until it responds or the timeout expires.
 * @param {number} port
 * @param {number} timeoutMs
 * @returns {Promise<void>}
 */
function waitForReady(port, timeoutMs) {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + timeoutMs;

    function check() {
      if (Date.now() > deadline) {
        reject(new Error(`CloudMock did not become ready within ${timeoutMs}ms`));
        return;
      }

      const req = httpRequest(
        { host: "localhost", port, path: "/", timeout: 1000 },
        (res) => {
          res.resume();
          resolve();
        }
      );
      req.on("error", () => setTimeout(check, 150));
      req.on("timeout", () => {
        req.destroy();
        setTimeout(check, 150);
      });
      req.end();
    }

    check();
  });
}

/**
 * CloudMock manages a local CloudMock process and provides pre-configured
 * AWS SDK v3 client options.
 */
class CloudMock {
  /**
   * @param {import("./index").CloudMockOptions} [options]
   */
  constructor(options = {}) {
    this._port = options.port ?? 0;
    this._region = options.region ?? "us-east-1";
    this._profile = options.profile ?? "minimal";
    this._process = null;
  }

  /**
   * Spawns the CloudMock binary and waits until it is ready to accept requests.
   * If no port was specified a free port is selected automatically.
   * @returns {Promise<void>}
   */
  async start() {
    if (this._process) return;

    if (this._port === 0) {
      this._port = await findFreePort();
    }

    const env = {
      ...process.env,
      CLOUDMOCK_PROFILE: this._profile,
      CLOUDMOCK_IAM_MODE: "none",
    };

    this._process = spawn(
      "npx",
      ["cloudmock", "--port", String(this._port)],
      { env, stdio: "ignore", detached: false }
    );

    this._process.on("error", (err) => {
      this._process = null;
      throw new Error(`Failed to start CloudMock: ${err.message}`);
    });

    await waitForReady(this._port, 30_000);
  }

  /**
   * Kills the CloudMock process if it is running.
   * @returns {Promise<void>}
   */
  async stop() {
    if (this._process) {
      this._process.kill();
      this._process = null;
    }
  }

  /**
   * The base URL of the running CloudMock instance, e.g. `http://localhost:4566`.
   * @type {string}
   */
  get endpoint() {
    return `http://localhost:${this._port}`;
  }

  /**
   * Injects a fault rule into the running CloudMock instance.
   *
   * @param {string} service - AWS service name (e.g. "s3", "dynamodb", "*" for all).
   * @param {string} action - API action (e.g. "GetObject") or "*" for all.
   * @param {string} type - Fault type: "error", "latency", "timeout", "blackhole", or "throttle".
   * @param {{ statusCode?: number, message?: string, latencyMs?: number, percentage?: number }} [opts]
   * @returns {Promise<void>}
   */
  async injectFault(service, action, type, opts = {}) {
    const body = JSON.stringify({
      service,
      action,
      type,
      enabled: true,
      errorCode: opts.statusCode ?? 500,
      errorMsg: opts.message ?? "",
      latencyMs: opts.latencyMs ?? 0,
      percentage: opts.percentage ?? 100,
    });
    await fetch(`${this.endpoint}/api/chaos`, {
      method: "POST",
      body,
      headers: { "Content-Type": "application/json" },
    });
  }

  /**
   * Disables all chaos rules on the running CloudMock instance.
   * @returns {Promise<void>}
   */
  async clearFaults() {
    await fetch(`${this.endpoint}/api/chaos`, { method: "DELETE" });
  }

  /**
   * Returns an AWS SDK v3 client configuration object.
   * Pass the return value directly to any client constructor:
   *
   * ```js
   * const s3 = new S3Client(cm.clientConfig());
   * ```
   *
   * @returns {{ endpoint: string, region: string, credentials: { accessKeyId: string, secretAccessKey: string }, forcePathStyle: boolean }}
   */
  clientConfig() {
    return {
      endpoint: this.endpoint,
      region: this._region,
      credentials: {
        accessKeyId: "test",
        secretAccessKey: "test",
      },
      forcePathStyle: true, // required for S3 path-style addressing
    };
  }
}

/**
 * Convenience helper: creates a CloudMock instance, starts it, and returns it.
 *
 * @example
 * ```js
 * const { mockAWS } = require("@cloudmock/sdk");
 * const { S3Client, CreateBucketCommand } = require("@aws-sdk/client-s3");
 *
 * const cm = await mockAWS();
 * const s3 = new S3Client(cm.clientConfig());
 * await s3.send(new CreateBucketCommand({ Bucket: "test-bucket" }));
 * await cm.stop();
 * ```
 *
 * @param {import("./index").CloudMockOptions} [options]
 * @returns {Promise<CloudMock>}
 */
async function mockAWS(options) {
  const cm = new CloudMock(options);
  await cm.start();
  return cm;
}

module.exports = { CloudMock, mockAWS };
