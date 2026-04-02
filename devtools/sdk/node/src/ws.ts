import net from 'node:net';
import http from 'node:http';

/**
 * Connection to cloudmock devtools. Sends captured events via:
 * 1. HTTP POST to admin API (:4599/api/source/events) — primary, debuggable
 * 2. TCP to source server (:4580) — fallback for high-throughput
 *
 * Batches events and flushes every 1s or 50 events.
 * Silently no-ops if devtools isn't running.
 */
export class Connection {
  readonly appName: string;
  private host: string;
  private tcpPort: number;
  private adminPort: number;
  private buffer: object[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private closed = false;
  private transport: 'http' | 'tcp' | 'none' = 'none';
  private httpFailed = false;

  // TCP fallback
  private socket: net.Socket | null = null;
  private tcpConnected = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(host: string, tcpPort: number, appName: string) {
    this.host = host;
    this.tcpPort = tcpPort;
    this.adminPort = parseInt(process.env.CLOUDMOCK_ADMIN_PORT || '4599', 10);
    this.appName = appName;

    // Test HTTP first, then fall back to TCP
    this.probeHTTP();

    // Flush buffer every 1 second
    this.flushTimer = setInterval(() => this.flush(), 1000);
    if (this.flushTimer && typeof this.flushTimer === 'object' && 'unref' in this.flushTimer) {
      (this.flushTimer as NodeJS.Timeout).unref();
    }
  }

  private probeHTTP(): void {
    const url = `http://${this.host}:${this.adminPort}/api/source/status`;
    const req = http.get(url, (res) => {
      if (res.statusCode === 200) {
        this.transport = 'http';
        this.httpFailed = false;
        console.log(`[cloudmock-sdk] Connected via HTTP to ${this.host}:${this.adminPort}`);
        // Flush any buffered events
        this.flush();
      } else {
        this.fallbackToTCP('HTTP probe got ' + res.statusCode);
      }
      res.resume(); // drain
    });
    req.on('error', () => {
      this.fallbackToTCP('HTTP probe failed');
    });
    req.setTimeout(2000, () => {
      req.destroy();
      this.fallbackToTCP('HTTP probe timeout');
    });
  }

  private fallbackToTCP(reason: string): void {
    if (this.closed) return;
    this.httpFailed = true;
    this.transport = 'tcp';
    console.log(`[cloudmock-sdk] ${reason}, falling back to TCP :${this.tcpPort}`);
    this.connectTCP();
  }

  private connectTCP(): void {
    if (this.closed) return;

    try {
      const socket = net.createConnection({ host: this.host, port: this.tcpPort }, () => {
        this.tcpConnected = true;
        this.socket = socket;
        this.transport = 'tcp';
        console.log(`[cloudmock-sdk] Connected via TCP to ${this.host}:${this.tcpPort}`);
        this.flush();
      });

      socket.unref();

      socket.on('error', () => {
        this.tcpConnected = false;
        this.socket = null;
        this.scheduleReconnect();
      });

      socket.on('close', () => {
        this.tcpConnected = false;
        this.socket = null;
        this.scheduleReconnect();
      });
    } catch {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.closed || this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      // Retry HTTP first, then TCP
      if (this.httpFailed) {
        this.probeHTTP();
      } else {
        this.connectTCP();
      }
    }, 5000);

    if (this.reconnectTimer && typeof this.reconnectTimer === 'object' && 'unref' in this.reconnectTimer) {
      (this.reconnectTimer as NodeJS.Timeout).unref();
    }
  }

  send(event: { type: string; data: any }): void {
    const msg = {
      ...event,
      source: this.appName,
      runtime: 'node',
      timestamp: Date.now(),
    };

    this.buffer.push(msg);

    // Flush immediately if buffer is large
    if (this.buffer.length >= 50) {
      this.flush();
    }
  }

  private flush(): void {
    if (this.buffer.length === 0) return;

    if (this.transport === 'http' && !this.httpFailed) {
      this.flushHTTP();
    } else if (this.transport === 'tcp' && this.tcpConnected && this.socket) {
      this.flushTCP();
    }
    // If neither transport is ready, keep buffering (max 500)
    if (this.buffer.length > 500) {
      this.buffer = this.buffer.slice(-500);
    }
  }

  private flushHTTP(): void {
    const events = this.buffer.splice(0);
    const body = JSON.stringify(events);

    const url = new URL(`http://${this.host}:${this.adminPort}/api/source/events`);
    const req = http.request({
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
      },
    }, (res) => {
      res.resume(); // drain
      if (res.statusCode !== 200) {
        // Put events back and fall back to TCP
        this.buffer.unshift(...events);
        this.fallbackToTCP('HTTP POST got ' + res.statusCode);
      }
    });

    req.on('error', () => {
      this.buffer.unshift(...events);
      this.fallbackToTCP('HTTP POST failed');
    });

    req.setTimeout(3000, () => {
      req.destroy();
      this.buffer.unshift(...events);
      this.fallbackToTCP('HTTP POST timeout');
    });

    req.write(body);
    req.end();
  }

  private flushTCP(): void {
    if (!this.socket) return;
    const events = this.buffer.splice(0);
    for (const msg of events) {
      try {
        this.socket.write(JSON.stringify(msg) + '\n');
      } catch {
        this.buffer.unshift(...events);
        break;
      }
    }
  }

  close(): void {
    this.closed = true;
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    // Final flush
    this.flush();
    if (this.socket) {
      try {
        this.socket.destroy();
      } catch {}
    }
    this.socket = null;
    this.tcpConnected = false;
  }
}
