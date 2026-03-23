package profiling

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Symbolizer parses Source Map v3 format and resolves generated file:line to
// original file:line.
type Symbolizer struct {
	mu   sync.RWMutex
	maps map[string]*sourceMap // key: generated file path
}

// sourceMap represents a parsed Source Map v3.
type sourceMap struct {
	Sources  []string
	Names    []string
	Mappings []mapping
}

type mapping struct {
	GeneratedLine   int
	GeneratedColumn int
	SourceIndex     int
	OriginalLine    int
	OriginalColumn  int
	NameIndex       int // -1 if not present
}

// NewSymbolizer returns an initialised Symbolizer.
func NewSymbolizer() *Symbolizer {
	return &Symbolizer{
		maps: make(map[string]*sourceMap),
	}
}

// LoadMap loads a source map for the given generated file path.
func (s *Symbolizer) LoadMap(filePath string, mapData []byte) error {
	sm, err := parseSourceMap(mapData)
	if err != nil {
		return fmt.Errorf("sourcemap: parse %q: %w", filePath, err)
	}
	s.mu.Lock()
	s.maps[filePath] = sm
	s.mu.Unlock()
	return nil
}

// Symbolicate resolves generated frames to original source locations.
// Frames without matching source maps pass through unchanged.
func (s *Symbolizer) Symbolicate(frames []StackFrame) []StackFrame {
	if len(frames) == 0 {
		return []StackFrame{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]StackFrame, len(frames))
	for i, f := range frames {
		sm, ok := s.maps[f.File]
		if !ok {
			out[i] = f
			continue
		}

		m, found := findMapping(sm.Mappings, f.Line)
		if !found {
			out[i] = f
			continue
		}

		resolved := f
		if m.SourceIndex >= 0 && m.SourceIndex < len(sm.Sources) {
			resolved.File = sm.Sources[m.SourceIndex]
		}
		resolved.Line = m.OriginalLine + 1 // 0-indexed → 1-indexed
		if m.NameIndex >= 0 && m.NameIndex < len(sm.Names) {
			resolved.Function = sm.Names[m.NameIndex]
		}
		out[i] = resolved
	}
	return out
}

// findMapping returns the closest mapping entry for the given 1-based generated
// line number (source maps use 0-based lines internally).
func findMapping(mappings []mapping, line int) (mapping, bool) {
	if len(mappings) == 0 {
		return mapping{}, false
	}
	target := line - 1 // convert to 0-based

	// Binary search for the last mapping whose GeneratedLine <= target.
	idx := sort.Search(len(mappings), func(i int) bool {
		return mappings[i].GeneratedLine > target
	})
	// idx is the first entry with GeneratedLine > target; step back one.
	if idx == 0 {
		return mapping{}, false
	}
	return mappings[idx-1], true
}

// ---- Source Map v3 parser ----

type rawSourceMap struct {
	Version  int      `json:"version"`
	Sources  []string `json:"sources"`
	Names    []string `json:"names"`
	Mappings string   `json:"mappings"`
}

func parseSourceMap(data []byte) (*sourceMap, error) {
	var raw rawSourceMap
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Version != 3 {
		return nil, fmt.Errorf("unsupported source map version %d", raw.Version)
	}

	mappings, err := decodeMappings(raw.Mappings)
	if err != nil {
		return nil, err
	}

	return &sourceMap{
		Sources:  raw.Sources,
		Names:    raw.Names,
		Mappings: mappings,
	}, nil
}

// decodeMappings decodes the VLQ-encoded mappings string into a flat slice of
// mapping entries sorted by GeneratedLine, GeneratedColumn.
func decodeMappings(encoded string) ([]mapping, error) {
	var mappings []mapping

	genLine := 0
	genCol := 0
	srcIdx := 0
	origLine := 0
	origCol := 0
	nameIdx := 0

	// Split into lines (';'), then segments (',').
	lineStart := 0
	for i := 0; i <= len(encoded); i++ {
		if i == len(encoded) || encoded[i] == ';' {
			line := encoded[lineStart:i]
			genCol = 0 // reset generated column at start of each line

			if line != "" {
				segStart := 0
				for j := 0; j <= len(line); j++ {
					if j == len(line) || line[j] == ',' {
						seg := line[segStart:j]
						if seg != "" {
							vals, err := decodeVLQ(seg)
							if err != nil {
								return nil, fmt.Errorf("decoding segment %q: %w", seg, err)
							}
							if len(vals) < 1 {
								segStart = j + 1
								continue
							}

							genCol += vals[0]
							m := mapping{
								GeneratedLine:   genLine,
								GeneratedColumn: genCol,
								SourceIndex:     -1,
								NameIndex:       -1,
							}

							if len(vals) >= 4 {
								srcIdx += vals[1]
								origLine += vals[2]
								origCol += vals[3]
								m.SourceIndex = srcIdx
								m.OriginalLine = origLine
								m.OriginalColumn = origCol
							}
							if len(vals) >= 5 {
								nameIdx += vals[4]
								m.NameIndex = nameIdx
							}

							mappings = append(mappings, m)
						}
						segStart = j + 1
					}
				}
			}

			genLine++
			lineStart = i + 1
		}
	}

	return mappings, nil
}

// ---- Base64 VLQ decoder ----

const b64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var b64Index [256]int

func init() {
	for i := range b64Index {
		b64Index[i] = -1
	}
	for i, c := range b64Chars {
		b64Index[c] = i
	}
}

// decodeVLQ decodes a sequence of Base64-VLQ values from encoded and returns
// them as a slice of signed integers.
func decodeVLQ(encoded string) ([]int, error) {
	var values []int

	i := 0
	for i < len(encoded) {
		// Decode one VLQ integer.
		var result int
		shift := 0
		for {
			if i >= len(encoded) {
				return nil, fmt.Errorf("unexpected end of VLQ data")
			}
			digit := b64Index[encoded[i]]
			if digit < 0 {
				return nil, fmt.Errorf("invalid base64 character %q", encoded[i])
			}
			i++

			hasContinuation := digit&0x20 != 0 // bit 5
			digit &= 0x1f                       // lower 5 bits

			result |= digit << shift
			shift += 5

			if !hasContinuation {
				break
			}
		}

		// Bit 0 of the final value is the sign bit.
		if result&1 != 0 {
			result = -(result >> 1)
		} else {
			result >>= 1
		}
		values = append(values, result)
	}

	return values, nil
}
