package main

import "testing"

func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"myapp", "myapp"},
		{"My_App-1.2", "My_App-1.2"},
		{"  trimmed  ", "trimmed"},
		// Path traversal / separators must be neutralized to a single segment.
		{"../../etc", "etc"},
		{"a/b/c", "c"},
		{"..", ""},
		{".", ""},
		{"/", ""},
		// Unsafe characters become '-'.
		{"a b$c", "a-b-c"},
		{"team:proj", "team-proj"},
	}
	for _, tc := range tests {
		if got := sanitizeProjectName(tc.in); got != tc.want {
			t.Errorf("sanitizeProjectName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
