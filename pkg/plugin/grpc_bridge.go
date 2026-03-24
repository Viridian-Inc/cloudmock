package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCBridge manages an external plugin process that communicates via gRPC.
type GRPCBridge struct {
	cmd     *exec.Cmd
	conn    *grpc.ClientConn
	addr    string
	mu      sync.Mutex
	started bool
}

// GRPCBridgeConfig holds configuration for launching an external plugin.
type GRPCBridgeConfig struct {
	// BinaryPath is the path to the plugin executable.
	BinaryPath string
	// Args are additional command-line arguments.
	Args []string
	// Addr is the address the plugin will listen on (e.g., "localhost:0" for auto-assign).
	// If empty, the bridge will use a Unix socket.
	Addr string
}

// NewGRPCBridge creates a bridge to an external plugin.
// The plugin binary must implement the ServicePlugin gRPC service.
func NewGRPCBridge(cfg GRPCBridgeConfig) *GRPCBridge {
	return &GRPCBridge{
		addr: cfg.Addr,
		cmd:  exec.Command(cfg.BinaryPath, cfg.Args...),
	}
}

// Connect establishes a gRPC connection to the plugin process.
// Call this after the plugin process has started and is listening.
func (b *GRPCBridge) Connect(ctx context.Context, addr string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc bridge connect: %w", err)
	}
	b.conn = conn
	b.addr = addr
	b.started = true
	return nil
}

// Close shuts down the gRPC connection and kills the plugin process.
func (b *GRPCBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if b.cmd != nil && b.cmd.Process != nil {
		if err := b.cmd.Process.Kill(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("grpc bridge close: %v", errs)
	}
	return nil
}

// Conn returns the underlying gRPC connection.
// Returns nil if not connected.
func (b *GRPCBridge) Conn() *grpc.ClientConn {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.conn
}
