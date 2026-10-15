package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Viridian-Inc/cloudmock/benchmarks/compat"
)

// go run ./benchmarks/compat/cmd/ --input benchmarks/results/full-cloudmock/results.json --output website/src/data/compat.json

func main() {
	input := flag.String("input", "benchmarks/results/full-cloudmock/results.json", "Path to benchmark results JSON")
	output := flag.String("output", "website/src/data/compat.json", "Path to write the compat report JSON")
	sourceSHA := flag.String("source-sha", "", "Commit SHA of the gateway source tested (optional)")
	baselinePath := flag.String("baseline", "", "Prior compat report; fail if any previously passing operation no longer passes (optional)")
	flag.Parse()

	// Read the baseline before writing output so the same report path can be
	// reused safely without comparing the new report to itself.
	var baseline *compat.CompatReport
	if *baselinePath != "" {
		data, err := os.ReadFile(*baselinePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading baseline: %v\n", err)
			os.Exit(1)
		}
		baseline = &compat.CompatReport{}
		if err := json.Unmarshal(data, baseline); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing baseline: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "Reading benchmark results from %s\n", *input)

	report, err := compat.GenerateFromBenchmark(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}
	report.SourceSHA = *sourceSHA
	if report.SourceSHA != "" {
		fmt.Fprintf(os.Stderr, "Tested gateway source SHA: %s\n", report.SourceSHA)
	}

	fmt.Fprintf(os.Stderr, "Generated report: %d services, %d ops — %.2f%% compat\n",
		len(report.Services), report.TotalOps, report.CompatPct)

	if err := compat.WriteReport(report, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Wrote compat report to %s\n", *output)

	if baseline != nil {
		regressions, err := compat.CompareToBaseline(report, baseline)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking baseline: %v\n", err)
			os.Exit(1)
		}
		if len(regressions) > 0 {
			fmt.Fprintf(os.Stderr, "Compatibility regressions against %s:\n", *baselinePath)
			for _, regression := range regressions {
				fmt.Fprintf(os.Stderr, "  %s.%s: pass -> %s\n", regression.Service, regression.Operation, regression.Status)
			}
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "No compatibility regressions against %s\n", *baselinePath)
	}
}
