---
title: Plugins
description: Extend CloudMock with custom in-process Go plugins or external gRPC plugins
---

CloudMock supports two plugin types for extending its behavior: in-process Go plugins that run inside the gateway binary, and external gRPC plugins that run as separate processes. Both types can intercept requests, modify responses, and register entirely new services.

## Plugin types

### In-process Go plugins

In-process plugins are compiled into the CloudMock binary. They have direct access to the service registry, request context, and in-memory stores. This is the highest-performance option, with zero serialization overhead.

Use in-process plugins when:
- You need sub-millisecond interception latency.
- You want to add a new Tier 1 service with custom business logic.
- You are comfortable rebuilding the CloudMock binary.

### External gRPC plugins

External plugins run as separate processes and communicate with CloudMock over gRPC. They can be written in any language that supports gRPC (Go, Python, Node.js, Rust, Java, etc.).

Use external plugins when:
- You want to write plugins in a language other than Go.
- You need to deploy plugins independently of the CloudMock binary.
- You want hot-reload without restarting CloudMock.

## Building an in-process Go plugin

### 1. Implement the Service interface

Every CloudMock service implements the `service.Service` interface:

```go
package myservice

import (
    "net/http"
    "github.com/neureaux/cloudmock/pkg/service"
)

type MyService struct {
    store *Store
}

func New() *MyService {
    return &MyService{store: NewStore()}
}

func (s *MyService) Name() string {
    return "myservice"
}

func (s *MyService) Actions() []service.Action {
    return []service.Action{
        {Name: "CreateWidget", Method: http.MethodPost, IAMAction: "myservice:CreateWidget"},
        {Name: "GetWidget", Method: http.MethodPost, IAMAction: "myservice:GetWidget"},
        {Name: "ListWidgets", Method: http.MethodPost, IAMAction: "myservice:ListWidgets"},
        {Name: "DeleteWidget", Method: http.MethodPost, IAMAction: "myservice:DeleteWidget"},
    }
}

func (s *MyService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
    switch ctx.Action {
    case "CreateWidget":
        return s.handleCreate(ctx)
    case "GetWidget":
        return s.handleGet(ctx)
    case "ListWidgets":
        return s.handleList(ctx)
    case "DeleteWidget":
        return s.handleDelete(ctx)
    default:
        return nil, service.NewAWSError("InvalidAction", "unknown action: "+ctx.Action, http.StatusBadRequest)
    }
}

func (s *MyService) HealthCheck() error {
    return nil
}
```

### 2. Create the store

The store manages in-memory state for your service:

```go
package myservice

import "sync"

type Widget struct {
    ID   string `json:"WidgetId"`
    Name string `json:"WidgetName"`
}

type Store struct {
    mu      sync.RWMutex
    widgets map[string]*Widget
}

func NewStore() *Store {
    return &Store{widgets: make(map[string]*Widget)}
}

func (s *Store) Put(w *Widget) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.widgets[w.ID] = w
}

func (s *Store) Get(id string) (*Widget, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    w, ok := s.widgets[id]
    return w, ok
}

func (s *Store) List() []*Widget {
    s.mu.RLock()
    defer s.mu.RUnlock()
    result := make([]*Widget, 0, len(s.widgets))
    for _, w := range s.widgets {
        result = append(result, w)
    }
    return result
}

func (s *Store) Delete(id string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.widgets[id]; !ok {
        return false
    }
    delete(s.widgets, id)
    return true
}
```

### 3. Register the service

Add the registration to `services/register.go`:

```go
import "github.com/neureaux/cloudmock/services/myservice"

// Inside the registration function:
registry.Register(myservice.New())
```

### 4. Rebuild CloudMock

```bash
make build
```

Your new service is now available at `http://localhost:4566` and will respond to requests with the `myservice` credential scope or `X-Amz-Target: MyService_...` header.

## Building an external gRPC plugin

### 1. Define the plugin interface

CloudMock uses a protobuf-based plugin protocol. The plugin implements the `CloudMockPlugin` gRPC service:

```protobuf
syntax = "proto3";

package cloudmock.plugin.v1;

service CloudMockPlugin {
  // Called when the plugin is registered
  rpc Register(RegisterRequest) returns (RegisterResponse);

  // Called for each matching request
  rpc HandleRequest(HandleRequestInput) returns (HandleRequestOutput);

  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message RegisterRequest {}

message RegisterResponse {
  string service_name = 1;
  repeated ActionDef actions = 2;
}

message ActionDef {
  string name = 1;
  string method = 2;
  string iam_action = 3;
}

message HandleRequestInput {
  string action = 1;
  string region = 2;
  string account_id = 3;
  bytes body = 4;
  map<string, string> headers = 5;
}

message HandleRequestOutput {
  int32 status_code = 1;
  bytes body = 2;
  map<string, string> headers = 3;
}

message HealthCheckRequest {}
message HealthCheckResponse {
  bool healthy = 1;
}
```

### 2. Implement the plugin (example in Python)

```python
import grpc
from concurrent import futures
import plugin_pb2
import plugin_pb2_grpc
import json
import uuid

class MyPlugin(plugin_pb2_grpc.CloudMockPluginServicer):
    def __init__(self):
        self.widgets = {}

    def Register(self, request, context):
        return plugin_pb2.RegisterResponse(
            service_name="myservice",
            actions=[
                plugin_pb2.ActionDef(name="CreateWidget", method="POST", iam_action="myservice:CreateWidget"),
                plugin_pb2.ActionDef(name="GetWidget", method="POST", iam_action="myservice:GetWidget"),
                plugin_pb2.ActionDef(name="ListWidgets", method="POST", iam_action="myservice:ListWidgets"),
                plugin_pb2.ActionDef(name="DeleteWidget", method="POST", iam_action="myservice:DeleteWidget"),
            ],
        )

    def HandleRequest(self, request, context):
        body = json.loads(request.body) if request.body else {}

        if request.action == "CreateWidget":
            widget_id = str(uuid.uuid4())
            self.widgets[widget_id] = {"WidgetId": widget_id, "WidgetName": body.get("WidgetName", "")}
            return plugin_pb2.HandleRequestOutput(
                status_code=200,
                body=json.dumps({"WidgetId": widget_id}).encode(),
            )
        elif request.action == "GetWidget":
            widget = self.widgets.get(body.get("WidgetId"))
            if not widget:
                return plugin_pb2.HandleRequestOutput(status_code=404, body=b'{"error": "WidgetNotFound"}')
            return plugin_pb2.HandleRequestOutput(status_code=200, body=json.dumps(widget).encode())
        elif request.action == "ListWidgets":
            return plugin_pb2.HandleRequestOutput(
                status_code=200,
                body=json.dumps({"Widgets": list(self.widgets.values())}).encode(),
            )
        elif request.action == "DeleteWidget":
            widget_id = body.get("WidgetId")
            if widget_id in self.widgets:
                del self.widgets[widget_id]
            return plugin_pb2.HandleRequestOutput(status_code=200, body=b'{}')

        return plugin_pb2.HandleRequestOutput(status_code=400, body=b'{"error": "InvalidAction"}')

    def HealthCheck(self, request, context):
        return plugin_pb2.HealthCheckResponse(healthy=True)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    plugin_pb2_grpc.add_CloudMockPluginServicer_to_server(MyPlugin(), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
```

### 3. Register the plugin with CloudMock

Tell CloudMock where to find the plugin by adding it to the config:

```yaml
plugins:
  - name: myservice
    type: grpc
    address: localhost:50051
```

Or via the admin API:

```bash
curl -X POST http://localhost:4599/api/plugins \
  -H "Content-Type: application/json" \
  -d '{"name": "myservice", "type": "grpc", "address": "localhost:50051"}'
```

## Adding a Tier 2 CRUD stub (simplest option)

If your custom service only needs basic CRUD operations (create, get, list, delete, update), you do not need a plugin at all. Add a service model entry to `services/stubs/catalog.go`:

```go
{
    ServiceName:  "myservice",
    Protocol:     "json",
    TargetPrefix: "MyService_20260101",
    Actions: map[string]stub.Action{
        "CreateWidget": createAction("CreateWidget", "widget", "WidgetId",
            []stub.Field{reqStr("WidgetName")},
            []stub.Field{optStr("WidgetName")}),
        "DescribeWidget": describeAction("DescribeWidget", "widget", "WidgetId"),
        "ListWidgets":    listAction("ListWidgets", "widget"),
        "DeleteWidget":   deleteAction("DeleteWidget", "widget", "WidgetId"),
    },
    ResourceTypes: map[string]stub.ResourceType{
        "widget": rt("Widget", "WidgetId",
            "arn:aws:myservice:{region}:{account}:widget/{id}",
            []stub.Field{optStr("WidgetName")}),
    },
},
```

No other code changes are needed. The stub engine handles routing, request parsing, response serialization, and IAM integration automatically.

## Plugin lifecycle

1. **Registration** -- On startup, CloudMock calls `Register` on each configured plugin. The plugin returns its service name and action list.
2. **Request handling** -- When a request arrives for the plugin's service, CloudMock calls `HandleRequest` with the action, region, account ID, request body, and headers.
3. **Health check** -- The admin API's `/api/health` endpoint calls `HealthCheck` on each plugin.
4. **Shutdown** -- CloudMock gracefully closes the gRPC connection on shutdown.
