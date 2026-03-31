---
title: Topology View
description: Live service map showing how your services connect to AWS resources
---

The Topology view renders a live, interactive graph of your application's service architecture. Nodes represent your services, Lambda functions, and AWS resources. Edges represent traffic between them, annotated with call counts and average latency.

## How nodes appear

CloudMock builds the topology from two sources:

1. **IaC-derived configuration** -- If you provide a topology config (via `PUT /api/topology/config`), nodes and edges are seeded from your infrastructure definition.

2. **Traffic discovery** -- As requests flow through the gateway, CloudMock discovers which services call which AWS resources and adds edges automatically. If a service appears in traffic but not in the static config, a new node is created.

Nodes are categorized by type:

| Node type | Description |
|-----------|-------------|
| **External** | Your application services (BFF, API, workers). These are the callers. |
| **Microservice** | Lambda functions, displayed individually with friendly names. |
| **AWS Service** | Collapsed representation of an AWS service (e.g., one "DynamoDB" node for all tables). |
| **AWS Category** | Super-collapsed grouping (e.g., "Data Layer" for DynamoDB + RDS + S3). Enabled via the Collapse toggle. |

### Collapsing

By default, all resources of the same AWS service type are collapsed into a single node. For example, 42 DynamoDB table resources become one "DynamoDB" node with a resource count badge.

Click **Collapse** in the toolbar to further merge AWS services into category-level nodes (Data Layer, Messaging, Auth & Security, Compute, Monitoring).

## Edges and traffic

Edges show the direction of API calls from caller to callee. When traffic data is available, edges are annotated with:

- **Call count** -- Total number of requests observed on this edge.
- **Average latency** -- Mean response time in milliseconds.

Edges are color-coded: normal traffic appears in the default color, while edges with high error rates or latency are highlighted.

## Clicking to inspect

Click any node to open the **Node Inspector** panel on the right side. The inspector shows:

- **Service name and type** -- What this node represents.
- **Health status** -- Whether the service is healthy.
- **Metrics** -- Request rate, latency percentiles (P50/P95/P99), error rate, with sparkline history charts.
- **Connected nodes** -- Upstream callers and downstream dependencies.
- **Endpoints** -- If a service manifest is loaded, the inspector shows all registered API endpoints.
- **Recent activity** -- Quick link to jump to the Activity view filtered to this service.

## Blast radius

Select a node and the topology highlights its **blast radius** -- the set of upstream services that would be affected if this node went down. CloudMock computes this by walking the dependency graph backward from the selected node.

The blast radius is also available programmatically via `GET /api/blast-radius?node=NODE_ID`.

## Timeline and time travel

Below the canvas, a **timeline strip** shows deploy events and incidents plotted on a time axis. You can:

- Click a deploy marker to see deploy details (service, version, timestamp) and select the corresponding node.
- Drag the **playhead** to "time travel" -- the topology metrics update to reflect the state at that point in time.
- Use the **time range selector** (Live, 15m, 1h, 6h, 24h, or Custom) to control the visible window.
- Switch between **Live** mode (real-time updates) and **Historical** mode (frozen at a specific time range).

## Layouts

You can pin node positions by dragging them, then save the arrangement as a **named layout**. Layouts are stored in the browser's local storage.

- **Save** -- Enter a name and save the current positions, pan, and zoom.
- **Apply** -- Load a saved layout to restore positions.
- **Set as default** -- Mark a layout to be applied automatically when the view loads.
- **Reset to auto** -- Clear all pinned positions and let the ELK layout engine arrange nodes automatically.

## Admin API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/topology` | Full topology graph (nodes and edges) |
| `GET` | `/api/topology/config` | Get IaC-derived topology configuration |
| `PUT` | `/api/topology/config` | Set IaC-derived topology configuration |
| `GET` | `/api/blast-radius?node=ID` | Compute blast radius for a node |
