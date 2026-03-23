package profiling

import (
	"runtime"
	"strings"
)

const maxFrames = 32

// skippedPrefixes lists runtime-internal prefixes to filter out of captured stacks.
var skippedPrefixes = []string{
	"runtime.",
	"reflect.",
	"testing.",
}

// CaptureStack captures the current goroutine's call stack at the named point.
// skip controls how many additional caller frames to skip (0 = caller of CaptureStack).
func CaptureStack(point string, skip int) SpanStack {
	var pcs [maxFrames]uintptr
	n := runtime.Callers(skip+2, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	var captured []StackFrame

	for {
		frame, more := frames.Next()
		if !shouldSkip(frame.Function) {
			sf := StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			}
			if idx := strings.LastIndex(frame.Function, "/"); idx >= 0 {
				if dot := strings.Index(frame.Function[idx:], "."); dot >= 0 {
					sf.Module = frame.Function[:idx+dot]
				}
			}
			captured = append(captured, sf)
		}
		if !more {
			break
		}
	}

	return SpanStack{
		Point:  point,
		Frames: captured,
	}
}

func shouldSkip(funcName string) bool {
	for _, prefix := range skippedPrefixes {
		if strings.HasPrefix(funcName, prefix) {
			return true
		}
	}
	return false
}
