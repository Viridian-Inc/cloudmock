# @cloudmock/rum Browser SDK

The `@cloudmock/rum` SDK captures Real User Monitoring data from browser applications -- Web Vitals, JavaScript errors, network performance, and user sessions.

## Installation

```bash
npm install @cloudmock/rum
```

## Quick Start

```js
import { CloudMockRUM } from '@cloudmock/rum';

CloudMockRUM.init({
  endpoint: 'http://localhost:4599',
  serviceName: 'my-web-app',
});
```

Add this to your application entry point (before other code runs). The SDK starts capturing immediately.

## Configuration

```js
CloudMockRUM.init({
  // Required
  endpoint: 'http://localhost:4599',  // CloudMock admin API URL
  serviceName: 'my-web-app',          // Identifies your app in DevTools

  // Optional
  sampleRate: 1.0,           // Sampling rate (0.0 to 1.0, default 1.0)
  trackErrors: true,         // Capture JS errors (default true)
  trackNetwork: true,        // Capture fetch/XHR timing (default true)
  trackWebVitals: true,      // Capture LCP, FID, CLS, etc. (default true)
  trackUserActions: true,    // Capture clicks and navigation (default true)

  // Privacy
  allowedTracingUrls: ['http://localhost:3000'],  // URLs to inject trace headers
  ignoreUrls: [/analytics/, /tracking/],          // URLs to exclude from capture

  // Version tracking
  version: '1.2.3',          // App version (shown in error groups)
  environment: 'development', // Environment name
});
```

## What Gets Captured

### Web Vitals

Automatically captures Core Web Vitals and other performance metrics:

| Metric | Description |
|--------|-------------|
| **LCP** (Largest Contentful Paint) | Time until the largest content element is visible |
| **FID** (First Input Delay) | Time from first interaction to browser response |
| **CLS** (Cumulative Layout Shift) | Visual stability score |
| **TTFB** (Time to First Byte) | Time until first byte of the page response |
| **FCP** (First Contentful Paint) | Time until first content is painted |

View in DevTools under the RUM tab, or via API:

```bash
curl http://localhost:4599/api/rum/vitals | jq '.'
```

```json
{
  "lcp": {"p50": 1200, "p75": 1800, "p95": 3200},
  "fid": {"p50": 12, "p75": 28, "p95": 95},
  "cls": {"p50": 0.02, "p75": 0.08, "p95": 0.25},
  "ttfb": {"p50": 180, "p75": 350, "p95": 800},
  "fcp": {"p50": 800, "p75": 1200, "p95": 2100}
}
```

### JavaScript Errors

Captures all unhandled exceptions and promise rejections:

```js
// Automatically captured:
throw new Error('Something went wrong');

// Also captured:
Promise.reject('Unhandled rejection');

// And console errors:
console.error('Critical failure', { context: 'payment' });
```

Errors are grouped by fingerprint and appear in both the RUM errors view and the main Errors tab:

```bash
curl http://localhost:4599/api/rum/errors | jq '.'
```

### Network Requests

Captures timing for all `fetch` and `XMLHttpRequest` calls:

```json
{
  "url": "http://localhost:3000/api/orders",
  "method": "POST",
  "status": 201,
  "duration_ms": 45,
  "request_size": 128,
  "response_size": 256,
  "timestamp": "2026-03-31T14:23:01.234Z"
}
```

### Page Views

Tracks navigation between pages (works with SPAs):

```bash
curl http://localhost:4599/api/rum/pages | jq '.'
```

```json
[
  {"route": "/", "views": 150, "avg_load_ms": 1200},
  {"route": "/orders", "views": 89, "avg_load_ms": 800},
  {"route": "/orders/:id", "views": 45, "avg_load_ms": 600}
]
```

### User Sessions

Groups events into sessions:

```bash
curl http://localhost:4599/api/rum/sessions | jq '.'
```

## Custom Events

Track application-specific events:

```js
import { CloudMockRUM } from '@cloudmock/rum';

// Track a custom event
CloudMockRUM.addEvent('order_placed', {
  orderId: 'order-123',
  total: 99.99,
  items: 3,
});

// Track user context
CloudMockRUM.setUser({
  id: 'user-456',
  email: 'alice@example.com',
  plan: 'pro',
});

// Add global context
CloudMockRUM.setGlobalContext({
  tenant: 'acme-corp',
  feature_flags: ['new-checkout', 'dark-mode'],
});
```

## Trace Propagation

When `allowedTracingUrls` is configured, the SDK injects trace context headers into outgoing requests:

```js
CloudMockRUM.init({
  endpoint: 'http://localhost:4599',
  serviceName: 'my-web-app',
  allowedTracingUrls: ['http://localhost:3000'],  // Your API server
});

// This fetch now includes W3C Trace Context headers:
// traceparent: 00-abc123...-def456...-01
fetch('http://localhost:3000/api/orders')
```

If your backend is instrumented with OpenTelemetry, this creates end-to-end traces from browser click to database query -- all visible in CloudMock's trace viewer.

## Framework Integration

### React

```jsx
// index.jsx
import { CloudMockRUM } from '@cloudmock/rum';

if (process.env.NODE_ENV === 'development') {
  CloudMockRUM.init({
    endpoint: 'http://localhost:4599',
    serviceName: 'my-react-app',
    version: process.env.REACT_APP_VERSION,
  });
}

// Error boundary integration
class ErrorBoundary extends React.Component {
  componentDidCatch(error, errorInfo) {
    CloudMockRUM.addError(error, { componentStack: errorInfo.componentStack });
  }
  render() { return this.props.children; }
}
```

### Vue

```js
// main.js
import { CloudMockRUM } from '@cloudmock/rum';
import { createApp } from 'vue';

const app = createApp(App);

if (import.meta.env.DEV) {
  CloudMockRUM.init({
    endpoint: 'http://localhost:4599',
    serviceName: 'my-vue-app',
  });

  app.config.errorHandler = (err, vm, info) => {
    CloudMockRUM.addError(err, { info });
  };
}
```

### Svelte / SvelteKit

```js
// hooks.client.js
import { CloudMockRUM } from '@cloudmock/rum';

if (dev) {
  CloudMockRUM.init({
    endpoint: 'http://localhost:4599',
    serviceName: 'my-svelte-app',
  });
}
```

## Source Maps

Upload source maps so minified stack traces are readable in DevTools:

```bash
# After your build step
curl -X POST http://localhost:4599/api/sourcemaps \
  -F "file=@dist/assets/app-abc123.js.map" \
  -F "release=1.2.3" \
  -F "url=http://localhost:3000/assets/app-abc123.js"
```

## Disabling in Production

The SDK should only run in development. Guard initialization:

```js
if (process.env.NODE_ENV === 'development') {
  CloudMockRUM.init({ ... });
}
```

Or rely on tree-shaking -- if the endpoint is unreachable, the SDK queues events locally and discards them after the buffer fills.
