package profiling

import (
	"strings"
	"testing"
)

func TestCaptureStack_ReturnsFrames(t *testing.T) {
	stack := CaptureStack("test-point", 0)

	if stack.Point != "test-point" {
		t.Fatalf("expected point %q, got %q", "test-point", stack.Point)
	}
	if len(stack.Frames) == 0 {
		t.Fatal("expected non-empty frames")
	}

	// The top frame should be this test function.
	found := false
	for _, f := range stack.Frames {
		if strings.Contains(f.Function, "TestCaptureStack_ReturnsFrames") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find TestCaptureStack_ReturnsFrames in captured frames")
		for i, f := range stack.Frames {
			t.Logf("  frame %d: %s (%s:%d)", i, f.Function, f.File, f.Line)
		}
	}
}

func TestCaptureStack_SkipsRuntimeFrames(t *testing.T) {
	stack := CaptureStack("skip-test", 0)

	for _, f := range stack.Frames {
		for _, prefix := range skippedPrefixes {
			if strings.HasPrefix(f.Function, prefix) {
				t.Errorf("frame %q should have been skipped (prefix %q)", f.Function, prefix)
			}
		}
	}
}

func TestCaptureStack_MaxFrames(t *testing.T) {
	stack := CaptureStack("max-test", 0)

	if len(stack.Frames) > maxFrames {
		t.Errorf("expected at most %d frames, got %d", maxFrames, len(stack.Frames))
	}
}

func TestCaptureStack_SkipParameter(t *testing.T) {
	stack := helper()

	// With skip=1 from inside helper, the helper function itself should be skipped.
	for _, f := range stack.Frames {
		if strings.HasSuffix(f.Function, ".helper") {
			t.Error("expected helper to be skipped with skip=1")
		}
	}
}

func helper() SpanStack {
	return CaptureStack("skip-helper", 1)
}
