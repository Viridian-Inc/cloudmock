package traffic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ComparisonConfig controls how two recordings are compared.
type ComparisonConfig struct {
	IgnorePaths   []string // JSON paths to skip (e.g., "RequestId", "ResponseMetadata")
	IgnoreHeaders []string // Headers to skip during comparison
	StrictMode    bool     // If false, only compare status codes + key fields
}

// ComparisonReport summarises the result of comparing two recordings.
type ComparisonReport struct {
	TotalRequests    int       `json:"total_requests"`
	Matched          int       `json:"matched"`
	Mismatched       int       `json:"mismatched"`
	Errors           int       `json:"errors"`
	CompatibilityPct float64   `json:"compatibility_pct"`
	Mismatches       []Mismatch `json:"mismatches,omitempty"`
}

// Mismatch describes a single entry-level discrepancy between two recordings.
type Mismatch struct {
	EntryID        string   `json:"entry_id"`
	Service        string   `json:"service"`
	Action         string   `json:"action"`
	OriginalStatus int      `json:"original_status"`
	ReplayStatus   int      `json:"replay_status"`
	Diffs          []string `json:"diffs"`
	Severity       string   `json:"severity"` // "status", "data", "schema"
}

// CompareRecordings compares two recordings entry-by-entry and returns a report.
// The original and replay recordings are matched by entry index order.
func CompareRecordings(original, replay *Recording, cfg ComparisonConfig) *ComparisonReport {
	report := &ComparisonReport{}

	// Match entries by position (both should be in the same order).
	n := len(original.Entries)
	if len(replay.Entries) < n {
		n = len(replay.Entries)
	}
	report.TotalRequests = n

	// Track extra entries in either recording.
	if len(original.Entries) != len(replay.Entries) {
		report.Errors += abs(len(original.Entries) - len(replay.Entries))
		report.TotalRequests = max(len(original.Entries), len(replay.Entries))
	}

	for i := 0; i < n; i++ {
		orig := original.Entries[i]
		rep := replay.Entries[i]

		var diffs []string
		severity := ""

		// Compare status codes.
		if orig.StatusCode != rep.StatusCode {
			diffs = append(diffs, fmt.Sprintf("status: %d -> %d", orig.StatusCode, rep.StatusCode))
			severity = "status"
		}

		if cfg.StrictMode {
			// Compare response bodies as JSON.
			bodyDiffs := CompareJSON([]byte(orig.ResponseBody), []byte(rep.ResponseBody), cfg.IgnorePaths)
			if len(bodyDiffs) > 0 {
				diffs = append(diffs, bodyDiffs...)
				if severity == "" {
					// Determine if it is a schema or data difference.
					for _, d := range bodyDiffs {
						if strings.Contains(d, "missing key") || strings.Contains(d, "extra key") {
							severity = "schema"
							break
						}
					}
					if severity == "" {
						severity = "data"
					}
				}
			}
		}

		if len(diffs) == 0 {
			report.Matched++
		} else {
			report.Mismatched++
			report.Mismatches = append(report.Mismatches, Mismatch{
				EntryID:        orig.ID,
				Service:        orig.Service,
				Action:         orig.Action,
				OriginalStatus: orig.StatusCode,
				ReplayStatus:   rep.StatusCode,
				Diffs:          diffs,
				Severity:       severity,
			})
		}
	}

	if report.TotalRequests > 0 {
		report.CompatibilityPct = float64(report.Matched) / float64(report.TotalRequests) * 100
	}

	return report
}

// CompareJSON parses two JSON byte slices and returns a list of human-readable
// diff strings. Paths listed in ignorePaths are skipped.
func CompareJSON(a, b []byte, ignorePaths []string) []string {
	var aVal, bVal any

	if err := json.Unmarshal(a, &aVal); err != nil {
		if err2 := json.Unmarshal(b, &bVal); err2 != nil {
			// Neither is valid JSON; compare as strings.
			if string(a) != string(b) {
				return []string{fmt.Sprintf("raw body differs: %q vs %q", truncate(string(a), 100), truncate(string(b), 100))}
			}
			return nil
		}
		return []string{"original is not valid JSON but replay is"}
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return []string{"replay is not valid JSON but original is"}
	}

	ignoreSet := make(map[string]bool, len(ignorePaths))
	for _, p := range ignorePaths {
		ignoreSet[p] = true
	}

	var diffs []string
	compareValues("", aVal, bVal, ignoreSet, &diffs)
	return diffs
}

func compareValues(path string, a, b any, ignore map[string]bool, diffs *[]string) {
	if ignore[path] {
		return
	}
	// Also check if the last segment of the path is ignored (simple field name match).
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		if ignore[path[idx+1:]] {
			return
		}
	} else if path != "" && ignore[path] {
		return
	}

	if a == nil && b == nil {
		return
	}
	if a == nil || b == nil {
		*diffs = append(*diffs, fmt.Sprintf("%s: %v -> %v", pathOrRoot(path), summarize(a), summarize(b)))
		return
	}

	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	if aIsMap && bIsMap {
		// Collect all keys.
		allKeys := make(map[string]bool)
		for k := range aMap {
			allKeys[k] = true
		}
		for k := range bMap {
			allKeys[k] = true
		}

		sorted := make([]string, 0, len(allKeys))
		for k := range allKeys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)

		for _, k := range sorted {
			childPath := k
			if path != "" {
				childPath = path + "." + k
			}
			if ignore[childPath] || ignore[k] {
				continue
			}

			aChild, aHas := aMap[k]
			bChild, bHas := bMap[k]

			if !aHas {
				*diffs = append(*diffs, fmt.Sprintf("%s: extra key in replay", childPath))
				continue
			}
			if !bHas {
				*diffs = append(*diffs, fmt.Sprintf("%s: missing key in replay", childPath))
				continue
			}

			compareValues(childPath, aChild, bChild, ignore, diffs)
		}
		return
	}

	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)

	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			*diffs = append(*diffs, fmt.Sprintf("%s: array length %d -> %d", pathOrRoot(path), len(aSlice), len(bSlice)))
			return
		}
		for i := range aSlice {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			compareValues(childPath, aSlice[i], bSlice[i], ignore, diffs)
		}
		return
	}

	if !reflect.DeepEqual(a, b) {
		*diffs = append(*diffs, fmt.Sprintf("%s: %v -> %v", pathOrRoot(path), summarize(a), summarize(b)))
	}
}

func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func summarize(v any) string {
	if v == nil {
		return "null"
	}
	s := fmt.Sprintf("%v", v)
	return truncate(s, 80)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
