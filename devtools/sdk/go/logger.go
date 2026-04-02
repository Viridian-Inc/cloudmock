package cloudmock

import (
	"fmt"
	"runtime"
	"strings"
)

// sendLog sends a console event to the devtools server with caller location.
func sendLog(conn *connection, level, message string) {
	file, line := callerLocation(3)

	conn.send("console", map[string]any{
		"level":   level,
		"message": message,
		"file":    file,
		"line":    line,
	})
}

// LogDebug sends a debug-level log message to the devtools server.
func LogDebug(message string) {
	Log("debug", message)
}

// LogInfo sends an info-level log message to the devtools server.
func LogInfo(message string) {
	Log("info", message)
}

// LogWarn sends a warn-level log message to the devtools server.
func LogWarn(message string) {
	Log("warn", message)
}

// LogError sends an error-level log message to the devtools server.
func LogError(message string) {
	Log("error", message)
}

// Logf sends a formatted log message at the given level to the devtools server.
func Logf(level, format string, args ...any) {
	Log(level, fmt.Sprintf(format, args...))
}

// callerLocation returns the file and line number of the caller.
// skip indicates how many frames to skip (0 = caller of callerLocation).
func callerLocation(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}
	// Shorten path: keep last two segments (e.g., "pkg/file.go")
	if idx := lastNthIndex(file, '/', 2); idx >= 0 {
		file = file[idx+1:]
	}
	return file, line
}

func lastNthIndex(s string, sep byte, n int) int {
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}

// FormatStack formats a stack trace from runtime.Stack output.
func FormatStack(buf []byte) string {
	return strings.TrimSpace(string(buf))
}
