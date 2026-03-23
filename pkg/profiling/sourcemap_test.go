package profiling

import (
	"testing"
)

func TestSymbolicate_WithMap(t *testing.T) {
	s := NewSymbolizer()
	// "AAAAA" → genCol=0, srcIdx=0, origLine=0, origCol=0, nameIdx=0 (hello)
	// "AACAC" → genCol=0 (delta), srcIdx=0 (delta), origLine=+1 (delta), origCol=0 (delta), nameIdx=+1 (delta=C=1 → "bar" if present)
	// names only has "hello" at index 0; AACAC has nameIdx delta=1 but no second name — that's fine, it simply won't
	// replace Function because NameIndex will be out of bounds. The important assertion is File.
	mapJSON := `{"version":3,"sources":["src/app.ts"],"names":["hello"],"mappings":"AAAAA;AACAC"}`
	err := s.LoadMap("dist/bundle.js", []byte(mapJSON))
	if err != nil {
		t.Fatal(err)
	}

	frames := []StackFrame{
		{Function: "anonymous", File: "dist/bundle.js", Line: 1},
		{Function: "unknown", File: "dist/bundle.js", Line: 2},
	}
	result := s.Symbolicate(frames)

	if result[0].File != "src/app.ts" {
		t.Errorf("expected src/app.ts, got %s", result[0].File)
	}
	if result[0].Function != "hello" {
		t.Errorf("expected function hello, got %s", result[0].Function)
	}
	if result[0].Line != 1 {
		t.Errorf("expected line 1, got %d", result[0].Line)
	}
	if result[1].File != "src/app.ts" {
		t.Errorf("expected src/app.ts for frame[1], got %s", result[1].File)
	}
	if result[1].Line != 2 {
		t.Errorf("expected line 2 for frame[1], got %d", result[1].Line)
	}
}

func TestSymbolicate_NoMap(t *testing.T) {
	s := NewSymbolizer()
	frames := []StackFrame{{Function: "foo", File: "app.go", Line: 42}}
	result := s.Symbolicate(frames)
	// Unchanged
	if result[0].File != "app.go" {
		t.Error("should pass through")
	}
	if result[0].Line != 42 {
		t.Error("should pass through")
	}
}

func TestSymbolicate_EmptyFrames(t *testing.T) {
	s := NewSymbolizer()
	result := s.Symbolicate(nil)
	if len(result) != 0 {
		t.Error("expected empty")
	}
}

func TestDecodeVLQ(t *testing.T) {
	// "AAAA" → [0, 0, 0, 0]
	vals, err := decodeVLQ("AAAA")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 4 || vals[0] != 0 {
		t.Errorf("wrong decode: got %v", vals)
	}
}

func TestDecodeVLQ_Positive(t *testing.T) {
	// "C" encodes the value 1 (base64=2, no continuation, sign bit=0, value=1)
	vals, err := decodeVLQ("C")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0] != 1 {
		t.Errorf("expected [1], got %v", vals)
	}
}

func TestDecodeVLQ_Negative(t *testing.T) {
	// "D" encodes the value -1 (base64=3, no continuation, sign bit=1, value=-(1))
	vals, err := decodeVLQ("D")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0] != -1 {
		t.Errorf("expected [-1], got %v", vals)
	}
}

func TestSymbolicate_MultipleFiles(t *testing.T) {
	s := NewSymbolizer()

	// Load a map for one file.
	mapJSON := `{"version":3,"sources":["src/main.ts"],"names":[],"mappings":"AAAA"}`
	if err := s.LoadMap("dist/main.js", []byte(mapJSON)); err != nil {
		t.Fatal(err)
	}

	frames := []StackFrame{
		{Function: "fn1", File: "dist/main.js", Line: 1},
		{Function: "fn2", File: "dist/other.js", Line: 5}, // no map loaded
	}
	result := s.Symbolicate(frames)

	if result[0].File != "src/main.ts" {
		t.Errorf("frame[0]: expected src/main.ts, got %s", result[0].File)
	}
	if result[1].File != "dist/other.js" {
		t.Errorf("frame[1]: expected dist/other.js (pass-through), got %s", result[1].File)
	}
	if result[1].Line != 5 {
		t.Errorf("frame[1]: expected line 5 (pass-through), got %d", result[1].Line)
	}
}

func TestParseSourceMap_InvalidVersion(t *testing.T) {
	data := []byte(`{"version":2,"sources":[],"names":[],"mappings":""}`)
	_, err := parseSourceMap(data)
	if err == nil {
		t.Error("expected error for version 2")
	}
}

func TestParseSourceMap_InvalidJSON(t *testing.T) {
	_, err := parseSourceMap([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
