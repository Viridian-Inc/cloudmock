package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

func WriteMarkdown(results *harness.BenchmarkResults, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	writeHeader(&b, results)
	writeStartup(&b, results)
	writeResources(&b, results)
	writeServiceResults(&b, results, 1, "Tier 1 Services (Full Implementations)")
	writeServiceResults(&b, results, 2, "Tier 2 Services (Stub Implementations)")
	writeCorrectnessMatrix(&b, results)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeHeader(b *strings.Builder, r *harness.BenchmarkResults) {
	fmt.Fprintf(b, "# Benchmark Results\n\n")
	fmt.Fprintf(b, "**Date:** %s | **Platform:** %s | **Mode:** %s | **Iterations:** %d\n\n---\n\n",
		r.Meta.Date, r.Meta.Platform, r.Meta.Mode, r.Meta.Iterations)
}

func writeStartup(b *strings.Builder, r *harness.BenchmarkResults) {
	fmt.Fprintf(b, "## Startup Time\n\n| Target | Median (ms) |\n|--------|------------|\n")
	for _, t := range sortedKeys(r.Startup) {
		fmt.Fprintf(b, "| %s | %.0f |\n", t, r.Startup[t].MedianMs)
	}
	fmt.Fprintf(b, "\n")
}

func writeResources(b *strings.Builder, r *harness.BenchmarkResults) {
	fmt.Fprintf(b, "## Resource Usage\n\n| Target | Peak Memory (MB) | Avg Memory (MB) | Peak CPU (%%) | Avg CPU (%%) |\n|--------|-----------------|----------------|-------------|------------|\n")
	for _, t := range sortedKeys(r.Resources) {
		s := r.Resources[t]
		fmt.Fprintf(b, "| %s | %.0f | %.0f | %.1f | %.1f |\n", t, s.PeakMemoryMB, s.AvgMemoryMB, s.PeakCPUPct, s.AvgCPUPct)
	}
	fmt.Fprintf(b, "\n")
}

func writeServiceResults(b *strings.Builder, r *harness.BenchmarkResults, tier int, title string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	services := map[string]bool{}
	for _, tr := range r.Targets {
		for name, svc := range tr.Services {
			if svc.Tier == tier {
				services[name] = true
			}
		}
	}
	sortedServices := make([]string, 0, len(services))
	for s := range services {
		sortedServices = append(sortedServices, s)
	}
	sort.Strings(sortedServices)
	targetNames := sortedKeys(r.Targets)

	for _, svcName := range sortedServices {
		fmt.Fprintf(b, "### %s\n\n", svcName)
		ops := map[string]bool{}
		for _, tr := range r.Targets {
			if svc, ok := tr.Services[svcName]; ok {
				for opName := range svc.Operations {
					ops[opName] = true
				}
			}
		}
		sortedOps := make([]string, 0, len(ops))
		for o := range ops {
			sortedOps = append(sortedOps, o)
		}
		sort.Strings(sortedOps)

		header := "| Operation |"
		sep := "|-----------|"
		for _, t := range targetNames {
			header += fmt.Sprintf(" %s P50 | %s Correct |", t, t)
			sep += "--------|---------|"
		}
		fmt.Fprintf(b, "%s\n%s\n", header, sep)

		for _, opName := range sortedOps {
			row := fmt.Sprintf("| %s |", opName)
			for _, t := range targetNames {
				tr := r.Targets[t]
				if svc, ok := tr.Services[svcName]; ok {
					if op, ok := svc.Operations[opName]; ok {
						row += fmt.Sprintf(" %.1f ms | %s |", op.Warm.P50, op.Correctness)
					} else {
						row += " - | unsupported |"
					}
				} else {
					row += " - | unsupported |"
				}
			}
			fmt.Fprintf(b, "%s\n", row)
		}
		fmt.Fprintf(b, "\n")
	}
}

func writeCorrectnessMatrix(b *strings.Builder, r *harness.BenchmarkResults) {
	fmt.Fprintf(b, "## Correctness Summary\n\n")
	targetNames := sortedKeys(r.Targets)
	header := "| Service | Tier |"
	sep := "|---------|------|"
	for _, t := range targetNames {
		header += fmt.Sprintf(" %s |", t)
		sep += "--------|"
	}
	fmt.Fprintf(b, "%s\n%s\n", header, sep)

	allServices := map[string]int{}
	for _, tr := range r.Targets {
		for name, svc := range tr.Services {
			allServices[name] = svc.Tier
		}
	}
	sortedSvcs := make([]string, 0, len(allServices))
	for s := range allServices {
		sortedSvcs = append(sortedSvcs, s)
	}
	sort.Strings(sortedSvcs)

	for _, svcName := range sortedSvcs {
		row := fmt.Sprintf("| %s | %d |", svcName, allServices[svcName])
		for _, t := range targetNames {
			tr := r.Targets[t]
			if svc, ok := tr.Services[svcName]; ok {
				pass, total := 0, 0
				for _, op := range svc.Operations {
					total++
					if op.Correctness == harness.GradePass {
						pass++
					}
				}
				row += fmt.Sprintf(" %d/%d |", pass, total)
			} else {
				row += " - |"
			}
		}
		fmt.Fprintf(b, "%s\n", row)
	}
	fmt.Fprintf(b, "\n")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
