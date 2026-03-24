package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
)

const (
	defaultAdminAddr = "http://localhost:4599"
	version          = "1.0.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	adminAddr := os.Getenv("CLOUDMOCK_ADMIN_ADDR")
	if adminAddr == "" {
		adminAddr = defaultAdminAddr
	}

	cmd := os.Args[1]
	switch cmd {
	case "start":
		cmdStart(os.Args[2:])
	case "stop":
		cmdStop()
	case "status":
		cmdStatus(adminAddr)
	case "reset":
		cmdReset(adminAddr, os.Args[2:])
	case "services":
		cmdServices(adminAddr)
	case "version":
		cmdVersion()
	case "config":
		cmdConfig(adminAddr)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: cloudmock <command> [options]

Commands:
  start      Start the cloudmock gateway
  stop       Stop the cloudmock gateway
  status     Show health status of all services
  reset      Reset service state (all or specific)
  services   List registered services
  config     Show current configuration
  version    Print version information
  help       Show this help message

Use "cloudmock <command> --help" for more information about a command.`)
}

func cmdStart(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	configFile := fs.String("config", "cloudmock.yml", "path to config file")
	profile := fs.String("profile", "", "service profile: minimal, standard, full")
	services := fs.String("services", "", "comma-separated list of services to enable")
	fs.Parse(args)

	// Build arguments for the gateway binary.
	gatewayArgs := []string{}
	if *configFile != "" {
		gatewayArgs = append(gatewayArgs, "-config", *configFile)
	}

	// Set environment variables for profile/services overrides.
	if *profile != "" {
		os.Setenv("CLOUDMOCK_PROFILE", *profile)
	}
	if *services != "" {
		os.Setenv("CLOUDMOCK_SERVICES", *services)
	}

	// Find gateway binary.
	gatewayBin := findGatewayBinary()
	if gatewayBin == "" {
		fmt.Fprintln(os.Stderr, "Error: gateway binary not found. Build it with 'make build'.")
		os.Exit(1)
	}

	fmt.Printf("Starting cloudmock gateway (config=%s)\n", *configFile)
	cmd := exec.Command(gatewayBin, gatewayArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func findGatewayBinary() string {
	candidates := []string{
		"./bin/gateway",
		"bin/gateway",
		"gateway",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return ""
}

func cmdStop() {
	// For now, print instructions. A full implementation would track PIDs or use docker-compose.
	fmt.Println("To stop cloudmock, press Ctrl+C in the terminal where it is running,")
	fmt.Println("or kill the gateway process.")
}

func cmdStatus(adminAddr string) {
	resp, err := http.Get(adminAddr + "/api/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot reach admin API at %s: %v\n", adminAddr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var health struct {
		Status   string          `json:"status"`
		Services map[string]bool `json:"services"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n\n", health.Status)

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tHEALTHY")
	fmt.Fprintln(tw, "-------\t-------")
	for name, healthy := range health.Services {
		h := "yes"
		if !healthy {
			h = "no"
		}
		fmt.Fprintf(tw, "%s\t%s\n", name, h)
	}
	tw.Flush()
}

func cmdReset(adminAddr string, args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	svcName := fs.String("service", "", "service to reset (omit for all)")
	fs.Parse(args)

	var url string
	if *svcName != "" {
		url = adminAddr + "/api/services/" + *svcName + "/reset"
	} else {
		url = adminAddr + "/api/reset"
	}

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "Error: service %q not found\n", *svcName)
		os.Exit(1)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if json.Unmarshal(body, &result) == nil {
		if *svcName != "" {
			fmt.Printf("Reset service: %s\n", *svcName)
		} else {
			svcs, _ := result["services"].([]any)
			names := make([]string, 0, len(svcs))
			for _, s := range svcs {
				if name, ok := s.(string); ok {
					names = append(names, name)
				}
			}
			fmt.Printf("Reset %d services: %s\n", len(names), strings.Join(names, ", "))
		}
	}
}

func cmdServices(adminAddr string) {
	resp, err := http.Get(adminAddr + "/api/services")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot reach admin API at %s: %v\n", adminAddr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var services []struct {
		Name        string `json:"name"`
		ActionCount int    `json:"action_count"`
		Healthy     bool   `json:"healthy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		os.Exit(1)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tACTIONS\tHEALTHY")
	fmt.Fprintln(tw, "-------\t-------\t-------")
	for _, svc := range services {
		h := "yes"
		if !svc.Healthy {
			h = "no"
		}
		fmt.Fprintf(tw, "%s\t%d\t%s\n", svc.Name, svc.ActionCount, h)
	}
	tw.Flush()
}

func cmdVersion() {
	fmt.Printf("cloudmock version %s\n", version)
}

func cmdConfig(adminAddr string) {
	resp, err := http.Get(adminAddr + "/api/config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot reach admin API at %s: %v\n", adminAddr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var cfg map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to decode response: %v\n", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(data))
}
