package scm

import "time"

// SCMProvider is the interface for source code management integrations.
type SCMProvider interface {
	GetFileContent(repo, path, ref string) (string, error)
	GetBlame(repo, path string, startLine, endLine int) ([]BlameLine, error)
	ListRecentCommits(repo string, limit int) ([]Commit, error)
}

// BlameLine represents a single line's blame information.
type BlameLine struct {
	Line       int       `json:"line"`
	CommitHash string    `json:"commit_hash"`
	Author     string    `json:"author"`
	Date       time.Time `json:"date"`
	Message    string    `json:"message"`
}

// Commit represents a Git commit.
type Commit struct {
	Hash         string    `json:"hash"`
	Author       string    `json:"author"`
	Message      string    `json:"message"`
	Date         time.Time `json:"date"`
	FilesChanged []string  `json:"files_changed"`
}

// SourceContext holds surrounding code lines for a file:line reference.
type SourceContext struct {
	Lines      []SourceLine `json:"lines"`
	StartLine  int          `json:"start_line"`
	EndLine    int          `json:"end_line"`
	FilePath   string       `json:"file_path"`
	Repo       string       `json:"repo"`
	CommitHash string       `json:"commit_hash"`
}

// SourceLine is a single line of source code with metadata.
type SourceLine struct {
	Number  int    `json:"number"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"` // true for the error line
}

// RepoMapping maps a configured repository to path-strip rules.
type RepoMapping struct {
	Owner      string `yaml:"owner" json:"owner"`
	Repo       string `yaml:"repo" json:"repo"`
	PathPrefix string `yaml:"path_prefix" json:"path_prefix"` // strip from stack trace paths
}

// Config holds SCM integration configuration.
type Config struct {
	Provider string        `yaml:"provider" json:"provider"`
	Token    string        `yaml:"token" json:"token"`
	Repos    []RepoMapping `yaml:"repos" json:"repos"`
}

// SuspectCommit is a commit suspected of causing an error.
type SuspectCommit struct {
	Commit
	Score         float64  `json:"score"`          // higher = more likely
	ErrorFiles    []string `json:"error_files"`    // files from the stack trace this commit touched
	ErrorLines    int      `json:"error_lines"`    // how many error lines this commit touched
	Reason        string   `json:"reason"`         // human-readable explanation
}
