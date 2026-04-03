package snapshot

import "encoding/json"

// StateFile is the top-level structure of a cloudmock state snapshot.
type StateFile struct {
	Version    int                        `json:"version"`
	ExportedAt string                     `json:"exported_at"`
	Services   map[string]json.RawMessage `json:"services"`
}
