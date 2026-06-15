package scm

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc lets a test stand in for *http.Client's transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// GraphQL blame returns ranges; GetBlame must attribute each line to the
// commit whose range covers it.
func TestGetBlame_GraphQLPerLine(t *testing.T) {
	const gql = `{"data":{"repository":{"defaultBranchRef":{"target":{"blame":{"ranges":[
		{"startingLine":1,"endingLine":3,"commit":{"oid":"aaa","message":"add foo\nbody","committedDate":"2026-01-01T00:00:00Z","author":{"name":"Alice"}}},
		{"startingLine":4,"endingLine":5,"commit":{"oid":"bbb","message":"fix bar","committedDate":"2026-02-02T00:00:00Z","author":{"name":"Bob"}}}
	]}}}}}}`

	g := NewGitHubProvider("test-token")
	g.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/graphql") || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		return jsonResp(http.StatusOK, gql), nil
	})}

	lines, err := g.GetBlame("octo/repo", "main.go", 1, 5)
	if err != nil {
		t.Fatalf("GetBlame: %v", err)
	}
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	// Lines 1-3 → Alice/aaa, lines 4-5 → Bob/bbb.
	for _, l := range lines {
		wantHash, wantAuthor := "aaa", "Alice"
		if l.Line >= 4 {
			wantHash, wantAuthor = "bbb", "Bob"
		}
		if l.CommitHash != wantHash || l.Author != wantAuthor {
			t.Errorf("line %d: got %s/%s, want %s/%s", l.Line, l.CommitHash, l.Author, wantHash, wantAuthor)
		}
	}
	// firstLine() must trim the multi-line message.
	if lines[0].Message != "add foo" {
		t.Errorf("line 1 message = %q, want %q", lines[0].Message, "add foo")
	}
}

// When GraphQL fails, GetBlame must fall back to the REST commits API.
func TestGetBlame_FallsBackToREST(t *testing.T) {
	const restCommits = `[{"sha":"ccc","commit":{"author":{"name":"Carol","date":"2026-03-03T00:00:00Z"},"message":"latest"}}]`

	g := NewGitHubProvider("test-token")
	g.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/graphql") {
			return jsonResp(http.StatusInternalServerError, "boom"), nil // force fallback
		}
		return jsonResp(http.StatusOK, restCommits), nil
	})}

	lines, err := g.GetBlame("octo/repo", "main.go", 1, 3)
	if err != nil {
		t.Fatalf("GetBlame: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for _, l := range lines {
		if l.CommitHash != "ccc" || l.Author != "Carol" {
			t.Errorf("line %d: got %s/%s, want ccc/Carol (REST fallback)", l.Line, l.CommitHash, l.Author)
		}
	}
}
