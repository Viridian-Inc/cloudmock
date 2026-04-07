package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
	"github.com/Viridian-Inc/cloudmock/benchmarks/report"
	"github.com/Viridian-Inc/cloudmock/benchmarks/suites"
	"github.com/Viridian-Inc/cloudmock/benchmarks/suites/tier1"
	"github.com/Viridian-Inc/cloudmock/benchmarks/suites/tier2"
	"github.com/Viridian-Inc/cloudmock/benchmarks/target"
)

func main() {
	var (
		flagTargets     = flag.String("target", "cloudmock", "comma-separated targets: cloudmock,localstack,localstack-pro,native")
		flagMode        = flag.String("mode", "docker", "execution mode: docker or native")
		flagServices    = flag.String("services", "", "comma-separated service names to filter (empty = all)")
		flagTier        = flag.Int("tier", 0, "filter by tier (0 = all)")
		flagIterations  = flag.Int("iterations", 10, "number of warm iterations per operation")
		flagConcurrency = flag.Int("concurrency", 4, "concurrency for load phase")
		flagCI          = flag.Bool("ci", false, "CI mode: reduce iterations and concurrency")
		flagQuick       = flag.Bool("quick", false, "quick mode: 1 iteration, no load phase")
		flagOutput      = flag.String("output", "benchmarks/results", "output directory for reports")
		flagAPIKey      = flag.String("localstack-api-key", "", "LocalStack Pro API key")
		flagEndpoint    = flag.String("endpoint", "", "use existing endpoint (skip container management)")
	)
	flag.Parse()

	cfg := harness.Config{
		Targets:     splitCSV(*flagTargets),
		Modes:       []string{*flagMode},
		Services:    splitCSV(*flagServices),
		Tier:        *flagTier,
		Iterations:  *flagIterations,
		Concurrency: *flagConcurrency,
		CI:          *flagCI,
		Quick:       *flagQuick,
		OutputDir:   *flagOutput,
	}

	if cfg.CI {
		cfg.Iterations = 5
		cfg.Concurrency = 2
	}
	if cfg.Quick {
		cfg.Iterations = 1
		cfg.Concurrency = 0
	}

	// Build registry with all suites.
	reg := buildRegistry()

	// Select suites based on flags.
	selectedSuites := selectSuites(reg, cfg)
	if len(selectedSuites) == 0 {
		log.Fatal("no suites selected; check --services and --tier flags")
	}

	fmt.Printf("Benchmark configuration:\n")
	fmt.Printf("  Targets:     %s\n", strings.Join(cfg.Targets, ", "))
	fmt.Printf("  Mode:        %s\n", *flagMode)
	fmt.Printf("  Suites:      %d selected\n", len(selectedSuites))
	fmt.Printf("  Iterations:  %d\n", cfg.Iterations)
	fmt.Printf("  Concurrency: %d\n", cfg.Concurrency)
	fmt.Printf("  Output:      %s\n\n", cfg.OutputDir)

	results := &harness.BenchmarkResults{
		Meta: harness.Meta{
			Date:        time.Now().UTC().Format(time.RFC3339),
			Platform:    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			GoVersion:   runtime.Version(),
			Mode:        *flagMode,
			Iterations:  cfg.Iterations,
			Concurrency: cfg.Concurrency,
		},
		Startup:   make(map[string]*harness.StartupResult),
		Resources: make(map[string]*harness.ResourceStats),
		Targets:   make(map[string]*harness.TargetResults),
	}

	ctx := context.Background()

	for _, targetName := range cfg.Targets {
		for _, mode := range cfg.Modes {
			targetKey := targetName
			if len(cfg.Modes) > 1 {
				targetKey = targetName + "-" + mode
			}

			fmt.Printf("=== Target: %s (mode: %s) ===\n", targetName, mode)

			var endpoint string
			var tgt target.Target

			if *flagEndpoint != "" {
				// Use existing endpoint — skip container management.
				endpoint = *flagEndpoint
				fmt.Printf("  Using existing endpoint: %s\n", endpoint)
			} else {
				var err error
				tgt, err = createTarget(targetName, mode, *flagAPIKey)
				if err != nil {
					log.Printf("ERROR creating target %s: %v\n", targetName, err)
					continue
				}

				// Measure startup time (5 runs, take median).
				startupResult := measureStartup(ctx, tgt, 5)
				results.Startup[targetKey] = startupResult
				fmt.Printf("  Startup: median=%.0fms\n", startupResult.MedianMs)

				// Start target for benchmarking.
				if err := tgt.Start(ctx); err != nil {
					log.Printf("ERROR starting target %s: %v\n", targetName, err)
					continue
				}

				endpoint = tgt.Endpoint()
			}

			targetResults := &harness.TargetResults{
				Target:   targetName,
				Mode:     mode,
				Services: make(map[string]*harness.ServiceResult),
			}

			// Run each suite.
			for _, suite := range selectedSuites {
				fmt.Printf("  Suite: %s (tier %d)\n", suite.Name(), suite.Tier())

				svcResult := &harness.ServiceResult{
					Service:    suite.Name(),
					Tier:       suite.Tier(),
					Operations: make(map[string]*harness.OperationResult),
				}

				for _, op := range suite.Operations() {
					fmt.Printf("    op: %-30s", op.Name)

					opResult, err := harness.RunOperation(ctx, op, endpoint, cfg.Iterations, cfg.Concurrency)
					if err != nil {
						fmt.Printf(" ERROR: %v\n", err)
						continue
					}

					svcResult.Operations[op.Name] = opResult
					fmt.Printf(" cold=%.1fms p50=%.1fms correctness=%s\n",
						opResult.ColdMs, opResult.Warm.P50, opResult.Correctness)
				}

				targetResults.Services[suite.Name()] = svcResult
			}

			results.Targets[targetKey] = targetResults

			// Collect resource stats before stopping.
			if tgt != nil {
				if stats, err := tgt.ResourceStats(ctx); err == nil {
					results.Resources[targetKey] = &harness.ResourceStats{
						PeakMemoryMB: stats.MemoryMB,
						AvgMemoryMB:  stats.MemoryMB,
						PeakCPUPct:   stats.CPUPct,
						AvgCPUPct:    stats.CPUPct,
					}
				}

				if err := tgt.Stop(ctx); err != nil {
					log.Printf("WARN: stopping target %s: %v\n", targetName, err)
				}
			}

			fmt.Println()
		}
	}

	// Write reports.
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	jsonPath := filepath.Join(cfg.OutputDir, "results.json")
	if err := report.WriteJSON(results, jsonPath); err != nil {
		log.Fatalf("write JSON: %v", err)
	}
	fmt.Printf("JSON report: %s\n", jsonPath)

	mdPath := filepath.Join(cfg.OutputDir, "results.md")
	if err := report.WriteMarkdown(results, mdPath); err != nil {
		log.Fatalf("write Markdown: %v", err)
	}
	fmt.Printf("Markdown report: %s\n", mdPath)
}

// buildRegistry creates and populates the suite registry with all tier1 and tier2 suites.
func buildRegistry() *suites.Registry {
	reg := suites.NewRegistry()

	// Tier 1 suites.
	reg.Register(tier1.NewAPIGatewaySuite())
	reg.Register(tier1.NewCloudFormationSuite())
	reg.Register(tier1.NewCloudTrailSuite())
	reg.Register(tier1.NewCloudWatchSuite())
	reg.Register(tier1.NewCloudWatchLogsSuite())
	reg.Register(tier1.NewCodeBuildSuite())
	reg.Register(tier1.NewCodePipelineSuite())
	reg.Register(tier1.NewCognitoSuite())
	reg.Register(tier1.NewConfigSuite())
	reg.Register(tier1.NewDynamoDBSuite())
	reg.Register(tier1.NewEC2Suite())
	reg.Register(tier1.NewECSSuite())
	reg.Register(tier1.NewEKSSuite())
	reg.Register(tier1.NewEventBridgeSuite())
	reg.Register(tier1.NewFirehoseSuite())
	reg.Register(tier1.NewIAMSuite())
	reg.Register(tier1.NewKinesisSuite())
	reg.Register(tier1.NewKMSSuite())
	reg.Register(tier1.NewLambdaSuite())
	reg.Register(tier1.NewRDSSuite())
	reg.Register(tier1.NewRoute53Suite())
	reg.Register(tier1.NewS3Suite())
	reg.Register(tier1.NewSNSSuite())
	reg.Register(tier1.NewSQSSuite())
	reg.Register(tier1.NewSTSSuite())

	// Tier 2 generated suites.
	for _, s := range tier2.GenerateAll() {
		reg.Register(s)
	}

	return reg
}

// selectSuites filters the registry based on Config.
func selectSuites(reg *suites.Registry, cfg harness.Config) []harness.Suite {
	var all []harness.Suite
	if cfg.Tier > 0 {
		all = reg.FilterByTier(cfg.Tier)
	} else {
		all = reg.List()
	}

	if len(cfg.Services) == 0 || (len(cfg.Services) == 1 && cfg.Services[0] == "*") {
		return all
	}

	filter := make(map[string]bool, len(cfg.Services))
	for _, s := range cfg.Services {
		filter[strings.ToLower(s)] = true
	}

	var filtered []harness.Suite
	for _, s := range all {
		if filter[strings.ToLower(s.Name())] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// createTarget creates the appropriate target based on mode.
func createTarget(name, mode, apiKey string) (target.Target, error) {
	switch mode {
	case "native":
		return target.NewNativeTarget(4566), nil
	case "docker":
		return target.NewDockerTarget(name, apiKey), nil
	default:
		return nil, fmt.Errorf("unknown mode %q; use docker or native", mode)
	}
}

// measureStartup starts and stops the target multiple times, recording latencies.
func measureStartup(ctx context.Context, tgt target.Target, runs int) *harness.StartupResult {
	latencies := make([]float64, 0, runs)
	for i := 0; i < runs; i++ {
		start := time.Now()
		if err := tgt.Start(ctx); err != nil {
			// If start fails, record a high latency and continue.
			latencies = append(latencies, 999999)
			continue
		}
		ms := float64(time.Since(start).Nanoseconds()) / 1e6
		latencies = append(latencies, ms)
		tgt.Stop(ctx) //nolint:errcheck
	}

	sorted := make([]float64, len(latencies))
	copy(sorted, latencies)
	sort.Float64s(sorted)

	median := 0.0
	if n := len(sorted); n > 0 {
		median = sorted[n/2]
	}

	return &harness.StartupResult{
		MedianMs: median,
		Runs:     latencies,
	}
}

// splitCSV splits a comma-separated string, trimming spaces, ignoring empty parts.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
