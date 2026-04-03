package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/neureaux/cloudmock/benchmarks/compat"
)

// go run ./benchmarks/compat/ --input benchmarks/results/full-cloudmock/results.json --output website/src/data/compat.json

func main() {
	input := flag.String("input", "benchmarks/results/full-cloudmock/results.json", "Path to benchmark results JSON")
	output := flag.String("output", "website/src/data/compat.json", "Path to write the compat report JSON")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "Reading benchmark results from %s\n", *input)

	report, err := compat.GenerateFromBenchmark(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generated report: %d services, %d ops — %.2f%% compat\n",
		len(report.Services), report.TotalOps, report.CompatPct)

	if err := compat.WriteReport(report, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Wrote compat report to %s\n", *output)
}
