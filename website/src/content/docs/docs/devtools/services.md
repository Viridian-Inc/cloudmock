---
title: Services View
description: Browse AWS services, app services, health status, and resource inventories
---

The Services view is a split-panel browser for exploring everything running inside CloudMock. The left panel lists your application services (loaded from IaC topology config) and the AWS services that CloudMock emulates. The right panel shows details and resource inventories for the selected service.

## Service list

The left panel organizes services into three sections:

### App services (from IaC topology)

If you have seeded a topology config via `PUT /api/topology/config`, the Services view parses the node and edge definitions to display your application services grouped by category:

| Group | Icon | Examples |
|-------|------|----------|
| **Client** | Mobile/browser apps | React Native app, web dashboard |
| **API** | Backend servers | BFF, GraphQL gateway |
| **Compute** | Processing services | Workers, Lambda functions |
| **Plugins** | Extensions | Custom plugins |

Each app service row shows the service name, type badge (client, server, plugin), and a count of its AWS dependencies derived from the topology edges.

### Active AWS services

AWS services that have received at least one API call are listed under **Active AWS Services**, sorted by action count (highest first). Each row shows:

- **Health dot** -- Green if the service is healthy, red if unhealthy.
- **Service name** -- The AWS service identifier (s3, dynamodb, sqs, etc.).
- **Action count** -- Number of distinct API actions registered for this service.

### Stub services

AWS services registered in CloudMock but with zero traffic are collapsed under **Stub Services**. Click the header to expand the list.

## Search

Type in the filter box at the top of the service list to search by name. The filter applies across all three sections (app, active AWS, and stub).

## App service detail

Selecting an app service opens the **App Service Detail** panel on the right. This shows:

- **Name and icon** -- The service name with a type-appropriate icon.
- **Group and type** -- The category (Client, API, Compute) and runtime type.
- **AWS Dependencies** -- A list of AWS resources this service connects to, derived from the topology edges. Each dependency shows a health dot and the resource identifier (e.g., `users-table` for DynamoDB, `uploads-bucket` for S3).
- **Service ID** -- The internal node ID from the topology config.

## Resource browser

Selecting an AWS service opens the **Resource Browser** panel on the right. CloudMock fetches the service's resource inventory via `GET /api/services/{name}/resources` and renders it as an interactive JSON tree.

The tree view collapses empty objects and null values, making it easy to scan large resource inventories. For example, selecting DynamoDB shows all tables with their key schemas, provisioned throughput, and item counts.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/services` | List all registered services with health status and action counts |
| `GET` | `/api/services/{name}/resources` | Get the resource inventory for a specific service |
| `GET` | `/api/topology/config` | Get IaC-derived topology configuration (nodes and edges) |
| `PUT` | `/api/topology/config` | Set IaC-derived topology configuration |
