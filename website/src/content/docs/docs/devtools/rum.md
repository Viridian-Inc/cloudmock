---
title: RUM View
description: Real User Monitoring with web vitals, page performance, error tracking, and session analysis
---

The RUM (Real User Monitoring) view collects and displays frontend performance data from your web application. It tracks Core Web Vitals, page-level performance, JavaScript errors, and user sessions.

## SDK setup

To send RUM data to CloudMock, install the `@cloudmock/rum` SDK in your frontend application:

```javascript
import { init } from '@cloudmock/rum';

init({
  endpoint: 'http://localhost:4599/api/rum/events',
  appName: 'my-app',
});
```

The SDK captures web vitals, page loads, JS errors, and session data automatically, then sends events to the CloudMock admin API.

## Time range

Select the observation window using the time selector in the header: **15m**, **30m**, **1h**, or **2h**. All tabs share the same time range. Data refreshes automatically every 10 seconds.

## Tabs

### Web Vitals

The Vitals tab shows an overview of Core Web Vitals and related metrics:

#### User Experience Score

A composite score (0-100) computed from all vitals, color-coded:

- **Green** (75+) -- Good user experience.
- **Yellow** (50-74) -- Needs improvement.
- **Red** (< 50) -- Poor experience.

The total event count is shown below the score.

#### Vital cards

Each vital is displayed as a card showing:

| Vital | Full Name | Unit | Good Threshold |
|-------|-----------|------|----------------|
| **LCP** | Largest Contentful Paint | ms | < 2500ms |
| **FID** | First Input Delay | ms | < 100ms |
| **CLS** | Cumulative Layout Shift | unitless | < 0.1 |
| **TTFB** | Time to First Byte | ms | < 800ms |
| **FCP** | First Contentful Paint | ms | < 1800ms |

Each card shows the P75 value, a rating (good/needs-improvement/poor) with color coding, and a breakdown of how many measurements fell into each category.

### Pages

The Pages tab shows per-route performance data:

| Column | Description |
|--------|-------------|
| Route | The page route or URL path |
| Avg Load | Average page load time in milliseconds |
| P75 LCP | 75th percentile Largest Contentful Paint |
| Avg CLS | Average Cumulative Layout Shift |
| Samples | Number of page view events in the time window |

### Errors

The Errors tab shows JavaScript errors grouped by fingerprint (deduplicated stack trace):

| Column | Description |
|--------|-------------|
| Count | Number of times this error occurred |
| Message | The error message |
| Sessions | Number of unique sessions affected |

Click an error to expand its sample stack trace.

### Sessions

The Sessions tab lists individual user sessions:

| Column | Description |
|--------|-------------|
| Session | Truncated session ID |
| Pages | Number of pages viewed in the session |
| Errors | Number of JS errors encountered (highlighted if > 0) |
| Duration | Session length in seconds |

## Empty state

When no RUM events have been received, the Vitals tab displays the SDK installation instructions with a code snippet.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/rum/vitals?minutes=N` | Web vitals overview with scores and breakdowns |
| `GET` | `/api/rum/pages?minutes=N` | Per-route page performance data |
| `GET` | `/api/rum/errors?minutes=N&limit=50` | Grouped JS errors with stack traces |
| `GET` | `/api/rum/sessions?minutes=N&limit=50` | Session summaries with page and error counts |
| `POST` | `/api/rum/events` | Ingest RUM events from the SDK |
