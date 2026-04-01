---
title: Settings View
description: Connection management, appearance, routing configuration, domains, and account
---

The Settings view provides configuration for the devtools application itself. It is organized into six tabs, each managing a different aspect of the devtools setup.

## Connections

The Connections tab manages the connection to your CloudMock instance.

| Field | Description |
|-------|-------------|
| **Admin URL** | The admin API endpoint (default: `http://localhost:4599`) with a connection status dot (green/red) |
| **Gateway URL** | The CloudMock gateway endpoint (default: `http://localhost:4566`) |

When connected, additional details are shown:

- **Region** -- The configured AWS region.
- **Profile** -- The active AWS profile.
- **IAM Mode** -- How IAM is handled (e.g., permissive, strict).
- **Services** -- Number of registered services.
- **PID** -- The CloudMock process ID.

Click **Disconnect** to close the connection, or **Connect to Local** to reconnect to the default local instance.

## Routing

The Routing tab (within Settings) provides a compact table for toggling services between local and cloud endpoints. It mirrors a simplified version of the full Routing view.

Each service row shows:

- **Service name** -- The service identifier.
- **Mode toggle** -- Switch between Local and Cloud.
- **Endpoint** -- The active endpoint URL.
- **Health status** -- Green dot for healthy local services, cloud icon for cloud-routed services.

Bulk actions (**All Local** / **All Cloud**) and an environment selector (dev, staging, prod) are available at the top.

You can add new services by filling in the service name, local endpoint, and cloud endpoint fields at the bottom of the table. Services can also be removed individually.

## Domains

The Domains tab lets you organize your services into domain groups. Domain groups are loaded from `/service-domains.json` and can be customized in the browser.

- **Rename** -- Click a domain name to edit it inline.
- **Add/remove services** -- Use the dropdown to assign unassigned services, or click the `x` on a service chip to remove it.
- **Add domain** -- Create a new domain group.
- **Delete domain** -- Remove an entire group (services become unassigned).
- **Show diff** -- View a line-by-line diff of your changes against the original configuration.
- **Reset** -- Revert all changes to the original `/service-domains.json` values.
- **Save** -- Persist changes to localStorage.

## Config

The Config tab shows a read-only JSON view of the current CloudMock configuration, fetched from the admin API. This includes all runtime settings such as region, port bindings, IAM mode, and enabled services.

## Appearance

The Appearance tab controls the visual presentation of the devtools:

### Theme

Toggle between **Dark** and **Light** color themes. The dark theme is the default.

### Font size

Adjust the base font size with a slider ranging from 11px to 16px (default: 13px). The change applies globally to all views.

Both settings are persisted to localStorage and applied immediately.

## Account

The Account tab shows a summary of your connection and dashboard state:

- **Connection Mode** -- Shows "Local Development" for localhost connections or "cloudmock.io" for cloud-hosted instances.
- **Authentication** -- In local mode, no authentication is required. Cloud connections show organization, region, and IAM mode.
- **Dashboard Preferences** -- Summary of total dashboards (presets + custom), favorites count, and hidden count.
- **Reset Dashboard Preferences** -- Clears all favorites and hidden dashboard settings.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/config` | Get current CloudMock configuration |
| `GET` | `/api/services` | List services for routing configuration |
| `GET` | `/api/topology/config` | Topology data for routing |
