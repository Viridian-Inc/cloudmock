package report

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

func WriteJSON(results *harness.BenchmarkResults, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
