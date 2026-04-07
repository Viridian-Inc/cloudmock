package snapshot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Export serialises the state of all Snapshotable services in the registry
// into a single JSON document.
func Export(registry *routing.Registry) ([]byte, error) {
	sf := StateFile{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Services:   make(map[string]json.RawMessage),
	}

	for _, svc := range registry.All() {
		snap, ok := svc.(service.Snapshotable)
		if !ok {
			continue
		}
		data, err := snap.ExportState()
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", svc.Name(), err)
		}
		if data != nil {
			sf.Services[svc.Name()] = data
		}
	}

	return json.MarshalIndent(sf, "", "  ")
}

// Import restores service state from a JSON document previously produced by Export.
// Services present in the file but not registered in the registry are silently skipped.
func Import(registry *routing.Registry, data []byte) error {
	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parse state file: %w", err)
	}

	for svcName, svcData := range sf.Services {
		svc, err := registry.Lookup(svcName)
		if err != nil {
			continue // skip unknown services
		}
		snap, ok := svc.(service.Snapshotable)
		if !ok {
			continue // service doesn't support import
		}
		if err := snap.ImportState(svcData); err != nil {
			return fmt.Errorf("import %s: %w", svcName, err)
		}
	}
	return nil
}
