package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWritePidFile_RoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "cloudmock.pid")
	if err := writePidFile(p, 4242); err != nil {
		t.Fatalf("writePidFile: %v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "4242" {
		t.Errorf("pidfile contents = %q, want 4242", got)
	}
}

func TestStopGateway_MissingPidfile(t *testing.T) {
	_, err := stopGateway(filepath.Join(t.TempDir(), "nope.pid"))
	if !os.IsNotExist(err) {
		t.Fatalf("err = %v, want os.IsNotExist", err)
	}
}

func TestStopGateway_SignalsAndTerminatesProcess(t *testing.T) {
	c := exec.Command("sleep", "30")
	if err := c.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() { _ = c.Process.Kill() })

	pidPath := filepath.Join(t.TempDir(), "cloudmock.pid")
	if err := writePidFile(pidPath, c.Process.Pid); err != nil {
		t.Fatalf("writePidFile: %v", err)
	}

	pid, err := stopGateway(pidPath)
	if err != nil {
		t.Fatalf("stopGateway: %v", err)
	}
	if pid != c.Process.Pid {
		t.Errorf("pid = %d, want %d", pid, c.Process.Pid)
	}
	if _, statErr := os.Stat(pidPath); !os.IsNotExist(statErr) {
		t.Errorf("pidfile not removed after stop")
	}

	// The signaled process must actually terminate.
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("process did not terminate within 5s after SIGTERM")
	}
}
