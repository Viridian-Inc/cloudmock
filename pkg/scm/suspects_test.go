package scm

import (
	"fmt"
	"testing"
	"time"
)

// mockProvider is a test double for SCMProvider.
type mockProvider struct {
	files  map[string]string            // "owner/repo:path" → content
	blames map[string][]BlameLine       // "owner/repo:path" → blame lines
}

func (m *mockProvider) GetFileContent(repo, path, ref string) (string, error) {
	key := fmt.Sprintf("%s:%s", repo, path)
	if content, ok := m.files[key]; ok {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s/%s", repo, path)
}

func (m *mockProvider) GetBlame(repo, path string, startLine, endLine int) ([]BlameLine, error) {
	key := fmt.Sprintf("%s:%s", repo, path)
	all, ok := m.blames[key]
	if !ok {
		return nil, fmt.Errorf("file not found: %s/%s", repo, path)
	}

	var result []BlameLine
	for _, bl := range all {
		if bl.Line >= startLine && bl.Line <= endLine {
			result = append(result, bl)
		}
	}
	return result, nil
}

func (m *mockProvider) ListRecentCommits(repo string, limit int) ([]Commit, error) {
	return nil, nil
}

func TestParseStackTrace_JS(t *testing.T) {
	stack := `TypeError: Cannot read property 'id' of undefined
    at getUser (src/handlers/user.ts:42:15)
    at processRequest (src/middleware/auth.ts:18:3)
    at Server.handleRequest (src/server.ts:120:22)`

	frames := ParseStackTrace(stack)

	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}

	expected := []StackFrame{
		{FilePath: "src/handlers/user.ts", Line: 42},
		{FilePath: "src/middleware/auth.ts", Line: 18},
		{FilePath: "src/server.ts", Line: 120},
	}

	for i, exp := range expected {
		if frames[i].FilePath != exp.FilePath {
			t.Errorf("frame %d: expected file %q, got %q", i, exp.FilePath, frames[i].FilePath)
		}
		if frames[i].Line != exp.Line {
			t.Errorf("frame %d: expected line %d, got %d", i, exp.Line, frames[i].Line)
		}
	}
}

func TestParseStackTrace_Python(t *testing.T) {
	stack := `Traceback (most recent call last):
  File "app/handlers/user.py", line 42, in get_user
    return db.query(user_id)
  File "app/db/connection.py", line 15, in query
    raise ConnectionError("timeout")`

	frames := ParseStackTrace(stack)

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}

	if frames[0].FilePath != "app/handlers/user.py" || frames[0].Line != 42 {
		t.Errorf("frame 0: got %+v", frames[0])
	}
	if frames[1].FilePath != "app/db/connection.py" || frames[1].Line != 15 {
		t.Errorf("frame 1: got %+v", frames[1])
	}
}

func TestParseStackTrace_Go(t *testing.T) {
	stack := `goroutine 1 [running]:
main.handler()
	/app/cmd/server/main.go:42
main.main()
	/app/cmd/server/main.go:15`

	frames := ParseStackTrace(stack)

	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}

	if frames[0].FilePath != "/app/cmd/server/main.go" || frames[0].Line != 42 {
		t.Errorf("frame 0: got %+v", frames[0])
	}
}

func TestParseStackTrace_Empty(t *testing.T) {
	frames := ParseStackTrace("")
	if len(frames) != 0 {
		t.Fatalf("expected 0 frames, got %d", len(frames))
	}
}

func TestParseStackTrace_Deduplicates(t *testing.T) {
	stack := `Error
    at handler (src/app.ts:10:5)
    at handler (src/app.ts:10:5)`

	frames := ParseStackTrace(stack)
	if len(frames) != 1 {
		t.Fatalf("expected 1 deduplicated frame, got %d", len(frames))
	}
}

func TestStripPathPrefix(t *testing.T) {
	tests := []struct {
		filePath string
		prefix   string
		expected string
	}{
		{"services/bff/src/handler.ts", "services/", "bff/src/handler.ts"},
		{"/app/services/bff/src/handler.ts", "services/", "bff/src/handler.ts"},
		{"src/handler.ts", "", "src/handler.ts"},
		{"src/handler.ts", "other/", "src/handler.ts"}, // prefix not found
	}

	for _, tt := range tests {
		got := StripPathPrefix(tt.filePath, tt.prefix)
		if got != tt.expected {
			t.Errorf("StripPathPrefix(%q, %q) = %q, want %q", tt.filePath, tt.prefix, got, tt.expected)
		}
	}
}

func TestFindSuspectCommits(t *testing.T) {
	now := time.Now()
	recentDate := now.Add(-2 * 24 * time.Hour) // 2 days ago
	oldDate := now.Add(-30 * 24 * time.Hour)    // 30 days ago (outside window)

	provider := &mockProvider{
		blames: map[string][]BlameLine{
			"meganargyle/app:src/handlers/user.ts": {
				{Line: 40, CommitHash: "abc123", Author: "Alice", Date: recentDate, Message: "fix user handler"},
				{Line: 41, CommitHash: "abc123", Author: "Alice", Date: recentDate, Message: "fix user handler"},
				{Line: 42, CommitHash: "abc123", Author: "Alice", Date: recentDate, Message: "fix user handler"},
				{Line: 43, CommitHash: "abc123", Author: "Alice", Date: recentDate, Message: "fix user handler"},
				{Line: 44, CommitHash: "old000", Author: "Bob", Date: oldDate, Message: "initial commit"},
			},
			"meganargyle/app:src/middleware/auth.ts": {
				{Line: 16, CommitHash: "def456", Author: "Carol", Date: recentDate, Message: "update auth middleware"},
				{Line: 17, CommitHash: "def456", Author: "Carol", Date: recentDate, Message: "update auth middleware"},
				{Line: 18, CommitHash: "def456", Author: "Carol", Date: recentDate, Message: "update auth middleware"},
				{Line: 19, CommitHash: "def456", Author: "Carol", Date: recentDate, Message: "update auth middleware"},
				{Line: 20, CommitHash: "def456", Author: "Carol", Date: recentDate, Message: "update auth middleware"},
			},
		},
	}

	stack := `TypeError: Cannot read property 'id' of undefined
    at getUser (src/handlers/user.ts:42:15)
    at processRequest (src/middleware/auth.ts:18:3)`

	repos := []RepoMapping{
		{Owner: "meganargyle", Repo: "app", PathPrefix: ""},
	}

	suspects, err := FindSuspectCommits(provider, stack, repos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(suspects) == 0 {
		t.Fatal("expected at least one suspect commit")
	}

	// abc123 should be a suspect (touched error line 42 area, recent)
	found := false
	for _, s := range suspects {
		if s.Hash == "abc123" {
			found = true
			if s.ErrorLines == 0 {
				t.Error("abc123 should have error lines > 0")
			}
			if s.Score <= 0 {
				t.Error("abc123 should have score > 0")
			}
			if s.Reason == "" {
				t.Error("abc123 should have a reason")
			}
		}
	}
	if !found {
		t.Error("expected abc123 in suspect commits")
	}

	// old000 should NOT be in suspects (outside 7-day window)
	for _, s := range suspects {
		if s.Hash == "old000" {
			t.Error("old000 should not be in suspects (outside 7-day window)")
		}
	}
}

func TestFindSuspectCommits_EmptyStack(t *testing.T) {
	provider := &mockProvider{}
	suspects, err := FindSuspectCommits(provider, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suspects) != 0 {
		t.Fatalf("expected 0 suspects for empty stack, got %d", len(suspects))
	}
}

func TestFindSuspectCommits_WithPathPrefix(t *testing.T) {
	now := time.Now()
	recentDate := now.Add(-1 * 24 * time.Hour)

	provider := &mockProvider{
		blames: map[string][]BlameLine{
			"meganargyle/api:bff/src/handler.ts": {
				{Line: 8, CommitHash: "aaa111", Author: "Dan", Date: recentDate, Message: "update handler"},
				{Line: 9, CommitHash: "aaa111", Author: "Dan", Date: recentDate, Message: "update handler"},
				{Line: 10, CommitHash: "aaa111", Author: "Dan", Date: recentDate, Message: "update handler"},
				{Line: 11, CommitHash: "aaa111", Author: "Dan", Date: recentDate, Message: "update handler"},
				{Line: 12, CommitHash: "aaa111", Author: "Dan", Date: recentDate, Message: "update handler"},
			},
		},
	}

	stack := `Error: timeout
    at handle (services/bff/src/handler.ts:10:5)`

	repos := []RepoMapping{
		{Owner: "meganargyle", Repo: "api", PathPrefix: "services/"},
	}

	suspects, err := FindSuspectCommits(provider, stack, repos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(suspects) != 1 {
		t.Fatalf("expected 1 suspect, got %d", len(suspects))
	}

	if suspects[0].Hash != "aaa111" {
		t.Errorf("expected suspect hash aaa111, got %s", suspects[0].Hash)
	}
}

func TestGetSourceContext(t *testing.T) {
	provider := &mockProvider{
		files: map[string]string{
			"meganargyle/app:src/handler.ts": "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
		},
	}

	ctx, err := GetSourceContext(provider, "meganargyle/app", "src/handler.ts", 5, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.StartLine != 3 {
		t.Errorf("expected start line 3, got %d", ctx.StartLine)
	}
	if ctx.EndLine != 7 {
		t.Errorf("expected end line 7, got %d", ctx.EndLine)
	}
	if len(ctx.Lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(ctx.Lines))
	}

	// The error line should be marked
	for _, l := range ctx.Lines {
		if l.Number == 5 && !l.IsError {
			t.Error("line 5 should have IsError=true")
		}
		if l.Number != 5 && l.IsError {
			t.Errorf("line %d should not have IsError=true", l.Number)
		}
	}
}

func TestGetSourceContext_EdgeLines(t *testing.T) {
	provider := &mockProvider{
		files: map[string]string{
			"meganargyle/app:src/small.ts": "line1\nline2\nline3",
		},
	}

	// Request context around line 1 with 5 lines of context
	ctx, err := GetSourceContext(provider, "meganargyle/app", "src/small.ts", 1, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.StartLine != 1 {
		t.Errorf("expected start line 1, got %d", ctx.StartLine)
	}
	if ctx.EndLine != 3 {
		t.Errorf("expected end line 3, got %d", ctx.EndLine)
	}
}

func TestScoreSuspect(t *testing.T) {
	now := time.Now()
	cutoff := now.Add(-suspectWindow)

	// Very recent commit touching many lines should score high
	s1 := &SuspectCommit{
		Commit:     Commit{Date: now.Add(-1 * time.Hour)},
		ErrorFiles: []string{"a.ts", "b.ts", "c.ts"},
		ErrorLines: 5,
	}
	score1 := scoreSuspect(s1, cutoff)

	// Older commit touching one line should score lower
	s2 := &SuspectCommit{
		Commit:     Commit{Date: now.Add(-6 * 24 * time.Hour)},
		ErrorFiles: []string{"a.ts"},
		ErrorLines: 1,
	}
	score2 := scoreSuspect(s2, cutoff)

	if score1 <= score2 {
		t.Errorf("recent high-coverage commit (%.3f) should score higher than old low-coverage commit (%.3f)", score1, score2)
	}

	// Score should be between 0 and 1
	if score1 < 0 || score1 > 1 {
		t.Errorf("score out of range [0,1]: %.3f", score1)
	}
	if score2 < 0 || score2 > 1 {
		t.Errorf("score out of range [0,1]: %.3f", score2)
	}
}
