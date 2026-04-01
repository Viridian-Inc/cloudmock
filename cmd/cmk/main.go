package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "start":
		cmdStart(os.Args[2:])
	case "stop":
		cmdStop()
	case "status":
		cmdStatus()
	case "logs":
		cmdLogs(os.Args[2:])
	case "config":
		cmdConfig()
	case "version":
		fmt.Printf("cmk version %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: cmk <command> [options]

Commands:
  start      Start CloudMock gateway + admin API + dashboard
  stop       Stop running CloudMock instance
  status     Show running status, port info, service count
  logs       Tail CloudMock logs
  config     Show current configuration
  version    Print version information
  help       Show this help message

Use "cmk <command> --help" for more information about a command.`)
}

// --- cmk start ---

func cmdStart(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	configFile := fs.String("config", "", "path to .cloudmock.yaml (auto-discovered if not set)")
	profile := fs.String("profile", "", "service profile: minimal | standard | full")
	gatewayPort := fs.Int("port", 0, "gateway port (default 4566)")
	foreground := fs.Bool("fg", false, "run in the foreground instead of daemonizing")
	fs.Parse(args)

	// Load configuration
	cfg := resolveConfig(*configFile)

	// Apply flag overrides
	if *profile != "" {
		cfg.Profile = *profile
	}
	if *gatewayPort != 0 {
		cfg.Gateway.Port = *gatewayPort
	}

	// Apply environment overrides last
	cfg.ApplyEnvOverrides()

	// Auto-discover IaC if not configured
	if cfg.IaC.Path == "" {
		cwd, _ := os.Getwd()
		disc := DiscoverIaC(cwd)
		if disc.Kind != IaCNone {
			cfg.IaC.Path = disc.Path
			fmt.Printf("  Discovered %s project at %s\n", disc.Kind, disc.Path)
		}
	}

	// Check if already running
	if pid, running := readPID(); running {
		fmt.Fprintf(os.Stderr, "Error: CloudMock is already running (PID %d)\n", pid)
		fmt.Fprintln(os.Stderr, "Run 'cmk stop' first, or 'cmk status' for details.")
		os.Exit(1)
	}

	// Find gateway binary
	gatewayBin := findGatewayBinary()
	if gatewayBin == "" {
		fmt.Fprintln(os.Stderr, "Error: gateway binary not found.")
		fmt.Fprintln(os.Stderr, "Build it with 'make build-gateway' or ensure it's in your PATH.")
		os.Exit(1)
	}

	// Build gateway arguments
	gatewayArgs := buildGatewayArgs(cfg)

	fmt.Println("Starting CloudMock...")
	fmt.Printf("  Gateway:   http://localhost:%d\n", cfg.Gateway.Port)
	fmt.Printf("  Admin API: http://localhost:%d\n", cfg.Admin.Port)
	if cfg.Dashboard.Enabled {
		fmt.Printf("  Dashboard: http://localhost:%d\n", cfg.Dashboard.Port)
	}
	fmt.Printf("  Profile:   %s\n", cfg.Profile)
	fmt.Printf("  Region:    %s\n", cfg.Region)

	if *foreground {
		runForeground(gatewayBin, gatewayArgs)
	} else {
		runDaemon(gatewayBin, gatewayArgs, cfg)
	}
}

func resolveConfig(explicit string) *CmkConfig {
	// If explicitly provided, use that
	if explicit != "" {
		cfg, err := LoadCmkConfig(explicit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return cfg
	}

	// Auto-discover .cloudmock.yaml
	cwd, _ := os.Getwd()
	found := FindConfigFile(cwd)
	if found != "" {
		cfg, err := LoadCmkConfig(found)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: found %s but failed to parse: %v\n", found, err)
			fmt.Fprintln(os.Stderr, "Using defaults.")
			return DefaultCmkConfig()
		}
		fmt.Printf("  Config:    %s\n", found)
		return cfg
	}

	return DefaultCmkConfig()
}

func buildGatewayArgs(cfg *CmkConfig) []string {
	// The gateway binary reads cloudmock.yml / env vars.
	// We pass configuration through environment variables so the gateway
	// picks them up regardless of its own config file.
	return []string{}
}

func runForeground(bin string, args []string) {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Forward signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(bin string, args []string, cfg *CmkConfig) {
	stateDir := ensureStateDir()
	logPath := filepath.Join(stateDir, "cloudmock.log")
	pidPath := filepath.Join(stateDir, "cloudmock.pid")

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot open log file %s: %v\n", logPath, err)
		os.Exit(1)
	}

	// Set environment for the gateway process
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("CLOUDMOCK_GATEWAY_PORT=%d", cfg.Gateway.Port),
		fmt.Sprintf("CLOUDMOCK_ADMIN_PORT=%d", cfg.Admin.Port),
		fmt.Sprintf("CLOUDMOCK_DASHBOARD_PORT=%d", cfg.Dashboard.Port),
		fmt.Sprintf("CLOUDMOCK_REGION=%s", cfg.Region),
		fmt.Sprintf("CLOUDMOCK_PROFILE=%s", cfg.Profile),
	)

	cmd := exec.Command(bin, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = env
	// Detach from the parent process group so it survives cmk exiting
	cmd.SysProcAttr = daemonSysProcAttr()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		if isPortInUse(err) {
			fmt.Fprintf(os.Stderr, "Error: port already in use. Check if another instance is running.\n")
			fmt.Fprintf(os.Stderr, "Try 'cmk status' or change the port in .cloudmock.yaml\n")
		} else if isPermissionDenied(err) {
			fmt.Fprintf(os.Stderr, "Error: permission denied starting gateway: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: failed to start gateway: %v\n", err)
		}
		os.Exit(1)
	}

	// Write PID file
	pid := cmd.Process.Pid
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write PID file: %v\n", err)
	}

	// Release the process so it's not tied to this parent
	cmd.Process.Release()
	logFile.Close()

	fmt.Printf("\nCloudMock started (PID %d)\n", pid)
	fmt.Printf("  Logs: %s\n", logPath)
	fmt.Println("\nRun 'cmk status' to check health, 'cmk logs' to tail output.")
}

// --- cmk stop ---

func cmdStop() {
	pid, running := readPID()
	if !running {
		fmt.Println("CloudMock is not running.")
		return
	}

	fmt.Printf("Stopping CloudMock (PID %d)...\n", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot find process %d: %v\n", pid, err)
		cleanupPID()
		os.Exit(1)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot signal process %d: %v\n", pid, err)
		cleanupPID()
		os.Exit(1)
	}

	// Wait for process to exit (up to 10 seconds)
	stopped := waitForExit(pid, 10*time.Second)
	if !stopped {
		fmt.Println("Process did not exit gracefully, sending SIGKILL...")
		process.Signal(syscall.SIGKILL)
		waitForExit(pid, 3*time.Second)
	}

	cleanupPID()
	fmt.Println("CloudMock stopped.")
}

// --- cmk status ---

func cmdStatus() {
	cfg := resolveConfigQuiet()
	adminAddr := cfg.AdminAddr()

	pid, running := readPID()
	if !running {
		fmt.Println("CloudMock is not running.")
		os.Exit(1)
	}

	fmt.Printf("CloudMock is running (PID %d)\n\n", pid)
	fmt.Printf("  Gateway:   http://localhost:%d\n", cfg.Gateway.Port)
	fmt.Printf("  Admin API: %s\n", adminAddr)
	if cfg.Dashboard.Enabled {
		fmt.Printf("  Dashboard: http://localhost:%d\n", cfg.Dashboard.Port)
	}
	fmt.Printf("  Profile:   %s\n", cfg.Profile)
	fmt.Printf("  Region:    %s\n\n", cfg.Region)

	// Query health endpoint
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(adminAddr + "/api/health")
	if err != nil {
		fmt.Println("  Health: unreachable (gateway may still be starting)")
		return
	}
	defer resp.Body.Close()

	var health struct {
		Status   string          `json:"status"`
		Services map[string]bool `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Println("  Health: could not parse response")
		return
	}

	healthy := 0
	for _, h := range health.Services {
		if h {
			healthy++
		}
	}
	fmt.Printf("  Health:    %s (%d/%d services healthy)\n", health.Status, healthy, len(health.Services))
}

// --- cmk logs ---

func cmdLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	lines := fs.Int("n", 50, "number of lines to show (0 for all)")
	follow := fs.Bool("f", true, "follow log output")
	fs.Parse(args)

	logPath := filepath.Join(stateDir(), "cloudmock.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "No log file found. Is CloudMock running?")
		fmt.Fprintf(os.Stderr, "Expected: %s\n", logPath)
		os.Exit(1)
	}

	// Use tail command for simplicity and cross-platform compat
	tailArgs := []string{}
	if *lines > 0 {
		tailArgs = append(tailArgs, fmt.Sprintf("-n%d", *lines))
	}
	if *follow {
		tailArgs = append(tailArgs, "-f")
	}
	tailArgs = append(tailArgs, logPath)

	tailBin := "tail"
	if runtime.GOOS == "windows" {
		// On Windows, use PowerShell's Get-Content
		runWindowsTail(logPath, *lines, *follow)
		return
	}

	cmd := exec.Command(tailBin, tailArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward interrupt to tail
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGINT)
		}
	}()

	if err := cmd.Run(); err != nil {
		// Ignore exit from ctrl-c
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWindowsTail(logPath string, lines int, follow bool) {
	args := []string{"-Command"}
	gcArgs := fmt.Sprintf("Get-Content '%s' -Tail %d", logPath, lines)
	if follow {
		gcArgs += " -Wait"
	}
	args = append(args, gcArgs)

	cmd := exec.Command("powershell", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// --- cmk config ---

func cmdConfig() {
	cfg := resolveConfigQuiet()

	data, err := cfg.ToYAML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

// --- helpers ---

func resolveConfigQuiet() *CmkConfig {
	cwd, _ := os.Getwd()
	found := FindConfigFile(cwd)
	if found != "" {
		cfg, err := LoadCmkConfig(found)
		if err == nil {
			cfg.ApplyEnvOverrides()
			return cfg
		}
	}
	cfg := DefaultCmkConfig()
	cfg.ApplyEnvOverrides()
	return cfg
}

func stateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".cloudmock")
	}
	return filepath.Join(home, ".cloudmock")
}

func ensureStateDir() string {
	dir := stateDir()
	os.MkdirAll(dir, 0755)
	return dir
}

func pidFilePath() string {
	return filepath.Join(stateDir(), "cloudmock.pid")
}

// readPID reads the PID file and checks if the process is alive.
// Returns (pid, true) if running, (0, false) otherwise.
func readPID() (int, bool) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}

	if !isProcessAlive(pid) {
		// Stale PID file
		cleanupPID()
		return 0, false
	}

	return pid, true
}

func cleanupPID() {
	os.Remove(pidFilePath())
}

// waitForExit polls until a process exits or the timeout elapses.
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func findGatewayBinary() string {
	// Check relative to the cmk binary itself
	self, err := os.Executable()
	if err == nil {
		selfDir := filepath.Dir(self)
		candidate := filepath.Join(selfDir, "gateway")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Check common locations
	candidates := []string{
		"./bin/gateway",
		"bin/gateway",
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	// Check PATH
	if p, err := exec.LookPath("gateway"); err == nil {
		return p
	}
	if p, err := exec.LookPath("cloudmock-gateway"); err == nil {
		return p
	}

	return ""
}

func isPortInUse(err error) bool {
	return strings.Contains(err.Error(), "address already in use") ||
		strings.Contains(err.Error(), "bind")
}

func isPermissionDenied(err error) bool {
	return strings.Contains(err.Error(), "permission denied")
}

// isProcessAlive checks if a process with the given PID exists and is running.
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
