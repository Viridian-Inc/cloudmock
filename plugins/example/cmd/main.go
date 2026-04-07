// Example CloudMock plugin that demonstrates the plugin interface.
//
// This plugin registers under the name "example" and handles requests
// at the path /example/*. It serves as a template for building new plugins.
//
// Build and run:
//
//	go build -o bin/plugins/example ./plugins/example/cmd
//	CLOUDMOCK_PLUGIN_ADDR=:9100 ./bin/plugins/example
package main

import (
	"context"
	"encoding/json"

	"github.com/Viridian-Inc/cloudmock/pkg/plugin"
	sdk "github.com/Viridian-Inc/cloudmock/sdk/go"
)

type examplePlugin struct{}

func (p *examplePlugin) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil
}

func (p *examplePlugin) Shutdown(_ context.Context) error {
	return nil
}

func (p *examplePlugin) HealthCheck(_ context.Context) (plugin.HealthStatus, string, error) {
	return plugin.HealthHealthy, "example plugin is healthy", nil
}

func (p *examplePlugin) Describe(_ context.Context) (*plugin.Descriptor, error) {
	return &plugin.Descriptor{
		Name:     "example",
		Version:  "0.1.0",
		Protocol: "custom",
		Actions:  []string{"Echo", "Ping"},
		APIPaths: []string{"/example/*"},
		Metadata: map[string]string{
			"description": "Example plugin for testing the CloudMock plugin system",
		},
	}, nil
}

func (p *examplePlugin) HandleRequest(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	response := map[string]any{
		"plugin":  "example",
		"action":  req.Action,
		"path":    req.Path,
		"method":  req.Method,
		"message": "Hello from the example plugin!",
	}

	body, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &plugin.Response{
		StatusCode: 200,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func main() {
	sdk.Serve(&examplePlugin{})
}
