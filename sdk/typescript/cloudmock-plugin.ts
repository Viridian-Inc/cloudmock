/**
 * CloudMock Plugin SDK for TypeScript/Node.js.
 *
 * Build CloudMock plugins by implementing CloudMockPlugin and calling serve().
 *
 * @example
 * ```ts
 * import { serve, type CloudMockPlugin, type PluginDescriptor, type PluginRequest, type PluginResponse } from "./cloudmock-plugin";
 *
 * const myPlugin: CloudMockPlugin = {
 *   describe: async () => ({
 *     name: "my-plugin",
 *     version: "0.1.0",
 *     protocol: "custom",
 *     actions: ["DoThing"],
 *     api_paths: ["/my-plugin/*"],
 *   }),
 *   handleRequest: async (req) => ({
 *     status_code: 200,
 *     body: Buffer.from(JSON.stringify({ ok: true })),
 *     headers: { "Content-Type": "application/json" },
 *   }),
 * };
 *
 * serve(myPlugin);
 * ```
 */

import { createServer, type IncomingMessage, type ServerResponse } from "node:http";

export interface PluginDescriptor {
  name: string;
  version: string;
  protocol: string;
  actions: string[];
  api_paths: string[];
  metadata?: Record<string, string>;
}

export interface AuthContext {
  user_id: string;
  account_id: string;
  arn: string;
  access_key_id: string;
  is_root: boolean;
  roles: string[];
  claims: Record<string, string>;
}

export interface PluginRequest {
  action: string;
  body: string; // base64 encoded
  headers: Record<string, string>;
  query_params: Record<string, string>;
  path: string;
  method: string;
  auth?: AuthContext;
}

export interface PluginResponse {
  status_code: number;
  body: string; // base64 encoded
  headers: Record<string, string>;
}

export interface CloudMockPlugin {
  init?(config: Buffer, dataDir: string, logLevel: string): Promise<void>;
  shutdown?(): Promise<void>;
  healthCheck?(): Promise<{ status: string; message: string }>;
  describe(): Promise<PluginDescriptor>;
  handleRequest(req: PluginRequest): Promise<PluginResponse>;
}

function readBody(req: IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks)));
    req.on("error", reject);
  });
}

function jsonResponse(res: ServerResponse, status: number, data: unknown) {
  const body = JSON.stringify(data);
  res.writeHead(status, {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(body).toString(),
  });
  res.end(body);
}

export function serve(plugin: CloudMockPlugin) {
  const addr = process.env.CLOUDMOCK_PLUGIN_ADDR || "127.0.0.1:0";
  const config = Buffer.from(process.env.CLOUDMOCK_PLUGIN_CONFIG || "");
  const dataDir = process.env.CLOUDMOCK_PLUGIN_DATA_DIR || "";
  const logLevel = process.env.CLOUDMOCK_PLUGIN_LOG_LEVEL || "info";

  const server = createServer(async (req, res) => {
    try {
      if (req.url === "/describe" && req.method === "GET") {
        const desc = await plugin.describe();
        jsonResponse(res, 200, desc);
      } else if (req.url === "/health" && req.method === "GET") {
        const health = plugin.healthCheck
          ? await plugin.healthCheck()
          : { status: "healthy", message: "" };
        jsonResponse(res, 200, health);
      } else if (req.method === "POST") {
        const body = await readBody(req);
        const request: PluginRequest = JSON.parse(body.toString());
        const response = await plugin.handleRequest(request);
        jsonResponse(res, 200, response);
      } else {
        jsonResponse(res, 404, { error: "not found" });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      jsonResponse(res, 500, { error: message });
    }
  });

  const [host, portStr] = addr.includes(":") ? addr.split(":") : ["127.0.0.1", addr];
  const port = parseInt(portStr, 10);

  // Initialize then start.
  (async () => {
    if (plugin.init) {
      await plugin.init(config, dataDir, logLevel);
    }

    server.listen(port, host, () => {
      const address = server.address();
      if (address && typeof address === "object") {
        const actualAddr = `${address.address}:${address.port}`;
        // Print to stdout so the core can discover the address.
        process.stdout.write(`PLUGIN_ADDR=${actualAddr}\n`);
        console.error(`[plugin] starting addr=${actualAddr}`);
      }
    });
  })();

  // Graceful shutdown.
  const shutdown = async () => {
    console.error("[plugin] shutting down");
    if (plugin.shutdown) {
      await plugin.shutdown();
    }
    server.close();
    process.exit(0);
  };

  process.on("SIGTERM", shutdown);
  process.on("SIGINT", shutdown);
}
