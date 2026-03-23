package profiling

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/pprof/profile"
)

// FoldedStacks reads a captured pprof profile by ID and converts it to folded stack format
// suitable for flame graph generation. Each line has the format: "func1;func2;func3 count".
func (e *Engine) FoldedStacks(id string) (string, error) {
	fp, err := e.FilePath(id)
	if err != nil {
		return "", err
	}

	f, err := os.Open(fp)
	if err != nil {
		return "", fmt.Errorf("open profile: %w", err)
	}
	defer f.Close()

	p, err := profile.Parse(f)
	if err != nil {
		return "", fmt.Errorf("parse profile: %w", err)
	}

	return foldedFromProfile(p), nil
}

func foldedFromProfile(p *profile.Profile) string {
	var buf strings.Builder
	for _, s := range p.Sample {
		var funcs []string
		// Walk locations in reverse (bottom-up).
		for i := len(s.Location) - 1; i >= 0; i-- {
			loc := s.Location[i]
			for _, line := range loc.Line {
				if line.Function != nil {
					funcs = append(funcs, line.Function.Name)
				}
			}
		}
		if len(funcs) > 0 && len(s.Value) > 0 {
			fmt.Fprintf(&buf, "%s %d\n", strings.Join(funcs, ";"), s.Value[0])
		}
	}
	return buf.String()
}
