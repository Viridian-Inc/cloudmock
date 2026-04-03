package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/tabwriter"

	"github.com/neureaux/cloudmock/pkg/proxy"
	"github.com/neureaux/cloudmock/pkg/traffic"
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
	case "record":
		cmdRecord(os.Args[2:])
	case "replay":
		cmdReplay(adminAddr, os.Args[2:])
	case "validate":
		cmdValidate(adminAddr, os.Args[2:])
	case "contract":
		cmdContract(os.Args[2:])
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
  record     Record real AWS traffic via proxy
  replay     Replay a recording against CloudMock
  validate   Replay + compare + exit code (CI mode)
  contract   Dual-mode proxy: compare real AWS vs CloudMock live
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

func cmdRecord(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	output := fs.String("output", "recording.json", "path to write the recording JSON")
	region := fs.String("region", "us-east-1", "AWS region for forwarding requests")
	port := fs.String("port", "4577", "local port for the recording proxy")
	fs.Parse(args)

	p := proxy.New(*region)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot listen on port %s: %v\n", *port, err)
		os.Exit(1)
	}

	srv := &http.Server{Handler: p}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		fmt.Printf("Recording proxy listening on :%s (region=%s)\n", *port, *region)
		fmt.Printf("Point your AWS SDK to http://localhost:%s\n", *port)
		fmt.Println("Press Ctrl+C to stop and save the recording.")
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nStopping proxy...")
	srv.Shutdown(context.Background())

	if err := p.SaveToFile(*output); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving recording: %v\n", err)
		os.Exit(1)
	}

	rec := p.Recording()
	fmt.Printf("Saved %d entries to %s\n", rec.EntryCount, *output)
}

func cmdReplay(adminAddr string, args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	input := fs.String("input", "", "path to recording JSON file")
	endpoint := fs.String("endpoint", "http://localhost:4566", "CloudMock gateway endpoint")
	speed := fs.Float64("speed", 0, "replay speed multiplier (0 = fast as possible)")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: --input is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var rec traffic.Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing recording: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Replaying %d entries against %s\n", len(rec.Entries), *endpoint)

	client := &http.Client{}
	var matched, mismatched, errors int

	for i, entry := range rec.Entries {
		var body io.Reader
		if entry.RequestBody != "" {
			body = strings.NewReader(entry.RequestBody)
		}
		url := *endpoint + entry.Path
		req, err := http.NewRequest(entry.Method, url, body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [%d] error creating request: %v\n", i+1, err)
			errors++
			continue
		}
		for k, v := range entry.RequestHeaders {
			req.Header.Set(k, v)
		}
		req.Header.Set("X-Cloudmock-Replay", entry.ID)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [%d] error: %v\n", i+1, err)
			errors++
			continue
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == entry.StatusCode {
			matched++
		} else {
			mismatched++
			fmt.Printf("  [%d] %s %s: %d (expected %d)\n", i+1, entry.Method, entry.Path, resp.StatusCode, entry.StatusCode)
		}

		// Respect speed multiplier for pacing.
		if *speed > 0 && i > 0 {
			delta := rec.Entries[i].OffsetMs - rec.Entries[i-1].OffsetMs
			if delta > 0 {
				waitMs := delta / *speed
				_ = waitMs // timing handled in engine; CLI replay is best-effort
			}
		}
	}

	total := matched + mismatched + errors
	fmt.Printf("\nResults: %d/%d matched, %d mismatched, %d errors\n", matched, total, mismatched, errors)
}

func cmdValidate(adminAddr string, args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	input := fs.String("input", "", "path to recording JSON file")
	endpoint := fs.String("endpoint", "http://localhost:4566", "CloudMock gateway endpoint")
	strict := fs.Bool("strict", false, "enable strict body comparison")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: --input is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var original traffic.Recording
	if err := json.Unmarshal(data, &original); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing recording: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Validating %d entries against %s\n", len(original.Entries), *endpoint)

	// Replay and capture results.
	client := &http.Client{}
	replayEntries := make([]traffic.CapturedEntry, 0, len(original.Entries))

	for _, entry := range original.Entries {
		var body io.Reader
		if entry.RequestBody != "" {
			body = strings.NewReader(entry.RequestBody)
		}
		url := *endpoint + entry.Path
		req, err := http.NewRequest(entry.Method, url, body)
		if err != nil {
			replayEntries = append(replayEntries, traffic.CapturedEntry{
				ID: entry.ID, StatusCode: 0,
			})
			continue
		}
		for k, v := range entry.RequestHeaders {
			req.Header.Set(k, v)
		}
		req.Header.Set("X-Cloudmock-Replay", entry.ID)

		resp, err := client.Do(req)
		if err != nil {
			replayEntries = append(replayEntries, traffic.CapturedEntry{
				ID: entry.ID, StatusCode: 0,
			})
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		replayEntries = append(replayEntries, traffic.CapturedEntry{
			ID:           entry.ID,
			Service:      entry.Service,
			Action:       entry.Action,
			Method:       entry.Method,
			Path:         entry.Path,
			StatusCode:   resp.StatusCode,
			ResponseBody: string(respBody),
		})
	}

	replay := &traffic.Recording{Entries: replayEntries}

	cfg := traffic.ComparisonConfig{
		StrictMode:  *strict,
		IgnorePaths: []string{"RequestId", "ResponseMetadata"},
	}
	report := traffic.CompareRecordings(&original, replay, cfg)

	fmt.Printf("\nCompatibility: %.1f%%\n", report.CompatibilityPct)
	fmt.Printf("  Matched:    %d\n", report.Matched)
	fmt.Printf("  Mismatched: %d\n", report.Mismatched)
	fmt.Printf("  Errors:     %d\n", report.Errors)

	for _, m := range report.Mismatches {
		fmt.Printf("\n  [%s] %s.%s: status %d -> %d (%s)\n", m.EntryID, m.Service, m.Action, m.OriginalStatus, m.ReplayStatus, m.Severity)
		for _, d := range m.Diffs {
			fmt.Printf("    - %s\n", d)
		}
	}

	if report.Mismatched > 0 || report.Errors > 0 {
		os.Exit(1)
	}
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

func cmdContract(args []string) {
	fs := flag.NewFlagSet("contract", flag.ExitOnError)
	cloudmockURL := fs.String("cloudmock", "http://localhost:4566", "CloudMock endpoint URL")
	region := fs.String("region", "us-east-1", "AWS region for forwarding requests")
	port := fs.String("port", "4577", "local port for the contract proxy")
	output := fs.String("output", "contract-report.json", "path to write the JSON report")
	ignorePaths := fs.String("ignore-paths", "RequestId,ResponseMetadata", "comma-separated JSON paths to ignore in comparison")
	runCmd := fs.String("run", "", "command to execute (proxy starts, runs command, generates report)")
	fs.Parse(args)

	var ignore []string
	if *ignorePaths != "" {
		for _, p := range strings.Split(*ignorePaths, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				ignore = append(ignore, p)
			}
		}
	}

	cp := proxy.NewContractProxy(*region, *cloudmockURL, ignore)

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot listen on port %s: %v\n", *port, err)
		os.Exit(1)
	}

	srv := &http.Server{Handler: cp}

	if *runCmd != "" {
		// Run mode: start proxy, execute command, generate report, exit.
		go func() {
			if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}()

		proxyAddr := fmt.Sprintf("http://localhost:%s", *port)
		fmt.Printf("Contract proxy running on %s\n", proxyAddr)
		fmt.Printf("Executing: %s\n", *runCmd)

		cmd := exec.Command("sh", "-c", *runCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "AWS_ENDPOINT_URL="+proxyAddr)
		cmdErr := cmd.Run()

		srv.Shutdown(context.Background())

		if err := cp.SaveReport(*output); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving report: %v\n", err)
			os.Exit(1)
		}

		report := cp.Report()
		printContractSummary(report, *output)

		if cmdErr != nil {
			fmt.Fprintf(os.Stderr, "\nCommand failed: %v\n", cmdErr)
			os.Exit(1)
		}
		if report.Mismatched > 0 {
			os.Exit(1)
		}
	} else {
		// Interactive mode: start proxy, wait for signal, generate report.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		go func() {
			proxyAddr := fmt.Sprintf("http://localhost:%s", *port)
			fmt.Printf("Contract proxy running on %s\n", proxyAddr)
			fmt.Printf("CloudMock endpoint: %s\n", *cloudmockURL)
			fmt.Printf("AWS region: %s\n", *region)
			fmt.Println("Press Ctrl+C to stop and generate the report.")
			if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}()

		<-ctx.Done()
		fmt.Println("\nStopping contract proxy...")
		srv.Shutdown(context.Background())

		if err := cp.SaveReport(*output); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving report: %v\n", err)
			os.Exit(1)
		}

		report := cp.Report()
		printContractSummary(report, *output)

		if report.Mismatched > 0 {
			os.Exit(1)
		}
	}
}

func printContractSummary(report *proxy.ContractReport, outputPath string) {
	fmt.Printf("\nContract Test Report\n")
	fmt.Printf("  Total requests:  %d\n", report.TotalRequests)
	fmt.Printf("  Matched:         %d\n", report.Matched)
	fmt.Printf("  Mismatched:      %d\n", report.Mismatched)
	fmt.Printf("  Compatibility:   %.1f%%\n", report.CompatibilityPct)

	for svc, sr := range report.ByService {
		fmt.Printf("  %s: %d/%d (%.1f%%)\n", svc, sr.Matched, sr.Total, sr.Pct)
	}

	for _, m := range report.Mismatches {
		fmt.Printf("\n  [%s] %s.%s: AWS %d vs CloudMock %d (%s)\n", m.Severity, m.Service, m.Action, m.AWSStatus, m.CloudMockStatus, m.Severity)
		for _, d := range m.Diffs {
			fmt.Printf("    - %s\n", d)
		}
	}

	fmt.Printf("\nReport written to %s\n", outputPath)
}
