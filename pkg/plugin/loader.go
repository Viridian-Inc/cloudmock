package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ExternalPlugin wraps an external plugin process that communicates via HTTP.
// The plugin binary is launched as a subprocess and communicates its listen
// address via stdout (PLUGIN_ADDR=host:port).
type ExternalPlugin struct {
	cmd  *exec.Cmd
	addr string
}

// LoadExternalPlugins discovers and starts plugin binaries from the given directory.
// Each executable file in the directory is treated as a plugin.
func LoadExternalPlugins(ctx context.Context, mgr *Manager, dir string, logger *slog.Logger) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No plugin directory is fine.
		}
		return fmt.Errorf("read plugin dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		// Check if file is executable.
		info, err := entry.Info()
		if err != nil || info.Mode()&0111 == 0 {
			continue
		}

		ep, err := startExternalPlugin(ctx, path, logger)
		if err != nil {
			logger.Warn("failed to start external plugin", "path", path, "error", err)
			continue
		}

		if err := mgr.RegisterInProcess(ctx, ep); err != nil {
			logger.Warn("failed to register external plugin", "path", path, "error", err)
			continue
		}
	}

	return nil
}

func startExternalPlugin(ctx context.Context, binaryPath string, logger *slog.Logger) (*ExternalPlugin, error) {
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start plugin: %w", err)
	}

	// Read the PLUGIN_ADDR line from stdout.
	addr, err := readPluginAddr(stdout, 5*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("read plugin addr: %w", err)
	}

	logger.Info("started external plugin", "path", binaryPath, "addr", addr)

	return &ExternalPlugin{cmd: cmd, addr: addr}, nil
}

func readPluginAddr(r io.Reader, timeout time.Duration) (string, error) {
	done := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PLUGIN_ADDR=") {
				done <- strings.TrimPrefix(line, "PLUGIN_ADDR=")
				return
			}
		}
		errCh <- fmt.Errorf("plugin exited without printing PLUGIN_ADDR")
	}()

	select {
	case addr := <-done:
		return addr, nil
	case err := <-errCh:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for PLUGIN_ADDR")
	}
}

// Plugin interface implementation for ExternalPlugin — proxies to the HTTP server.

func (ep *ExternalPlugin) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil // Already initialized during startup.
}

func (ep *ExternalPlugin) Shutdown(_ context.Context) error {
	if ep.cmd != nil && ep.cmd.Process != nil {
		return ep.cmd.Process.Signal(os.Interrupt)
	}
	return nil
}

func (ep *ExternalPlugin) HealthCheck(_ context.Context) (HealthStatus, string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/health", ep.addr))
	if err != nil {
		return HealthUnhealthy, err.Error(), nil
	}
	defer resp.Body.Close()
	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	switch result.Status {
	case "healthy":
		return HealthHealthy, result.Message, nil
	case "degraded":
		return HealthDegraded, result.Message, nil
	default:
		return HealthUnhealthy, result.Message, nil
	}
}

func (ep *ExternalPlugin) Describe(_ context.Context) (*Descriptor, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/describe", ep.addr))
	if err != nil {
		return nil, fmt.Errorf("describe: %w", err)
	}
	defer resp.Body.Close()
	var desc Descriptor
	if err := json.NewDecoder(resp.Body).Decode(&desc); err != nil {
		return nil, fmt.Errorf("decode descriptor: %w", err)
	}
	return &desc, nil
}

func (ep *ExternalPlugin) HandleRequest(_ context.Context, req *Request) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpResp, err := http.Post(fmt.Sprintf("http://%s/", ep.addr), "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("plugin request: %w", err)
	}
	defer httpResp.Body.Close()
	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &resp, nil
}

var _ Plugin = (*ExternalPlugin)(nil)
