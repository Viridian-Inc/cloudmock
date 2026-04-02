package target

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type NativeTarget struct {
	port int
	cmd  *exec.Cmd
}

func NewNativeTarget(port int) *NativeTarget {
	return &NativeTarget{port: port}
}

func (n *NativeTarget) Name() string     { return "cloudmock-native" }
func (n *NativeTarget) Endpoint() string { return fmt.Sprintf("http://localhost:%d", n.port) }

func (n *NativeTarget) Start(ctx context.Context) error {
	n.cmd = exec.CommandContext(ctx, "npx", "cloudmock", "--port", strconv.Itoa(n.port))
	n.cmd.Env = append(os.Environ(), "CLOUDMOCK_PROFILE=full", "CLOUDMOCK_IAM_MODE=none")
	n.cmd.Stdout = os.Stdout
	n.cmd.Stderr = os.Stderr
	if err := n.cmd.Start(); err != nil {
		return fmt.Errorf("start cloudmock native: %w", err)
	}
	return n.waitReady(ctx, 30*time.Second)
}

func (n *NativeTarget) Stop(ctx context.Context) error {
	if n.cmd != nil && n.cmd.Process != nil {
		n.cmd.Process.Kill()
		n.cmd.Wait()
	}
	return nil
}

func (n *NativeTarget) ResourceStats(ctx context.Context) (*Stats, error) {
	if n.cmd == nil || n.cmd.Process == nil {
		return nil, fmt.Errorf("process not running")
	}
	pid := n.cmd.Process.Pid
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("ps", "-o", "rss=,pcpu=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) < 2 {
			return nil, fmt.Errorf("unexpected ps output: %s", out)
		}
		rssKB, _ := strconv.ParseFloat(fields[0], 64)
		cpuPct, _ := strconv.ParseFloat(fields[1], 64)
		return &Stats{MemoryMB: rssKB / 1024, CPUPct: cpuPct}, nil
	}
	statm, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(statm))
	if len(fields) < 2 {
		return nil, fmt.Errorf("unexpected statm: %s", statm)
	}
	pages, _ := strconv.ParseFloat(fields[1], 64)
	memMB := pages * 4096 / 1024 / 1024
	return &Stats{MemoryMB: memMB, CPUPct: 0}, nil
}

func (n *NativeTarget) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(n.Endpoint() + "/")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("cloudmock native not ready after %s", timeout)
}
