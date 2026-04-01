package scm

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StackFrame represents a parsed frame from a stack trace.
type StackFrame struct {
	FilePath string
	Line     int
}

// suspectWindow is how far back to look for suspect commits.
const suspectWindow = 7 * 24 * time.Hour

// ParseStackTrace extracts file:line pairs from a stack trace string.
// Supports common formats:
//   - "at funcName (file.js:10:5)"       — JS/TS
//   - "at file.js:10:5"                   — JS/TS (no func)
//   - "File \"file.py\", line 10"         — Python
//   - "file.go:10"                        — Go
var (
	jsFrameRe     = regexp.MustCompile(`(?:at\s+(?:\S+\s+)?\(?)([^\s():]+):(\d+)(?::\d+)?\)?`)
	pythonFrameRe = regexp.MustCompile(`File "([^"]+)", line (\d+)`)
	goFrameRe     = regexp.MustCompile(`^\s*([^\s:]+\.go):(\d+)`)
)

// ParseStackTrace extracts file paths and line numbers from a stack trace.
func ParseStackTrace(stack string) []StackFrame {
	var frames []StackFrame
	seen := make(map[string]bool)

	for _, line := range strings.Split(stack, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var file string
		var lineNum int

		if m := jsFrameRe.FindStringSubmatch(line); len(m) >= 3 {
			file = m[1]
			lineNum, _ = strconv.Atoi(m[2])
		} else if m := pythonFrameRe.FindStringSubmatch(line); len(m) >= 3 {
			file = m[1]
			lineNum, _ = strconv.Atoi(m[2])
		} else if m := goFrameRe.FindStringSubmatch(line); len(m) >= 3 {
			file = m[1]
			lineNum, _ = strconv.Atoi(m[2])
		}

		if file != "" && lineNum > 0 {
			key := fmt.Sprintf("%s:%d", file, lineNum)
			if !seen[key] {
				seen[key] = true
				frames = append(frames, StackFrame{FilePath: file, Line: lineNum})
			}
		}
	}

	return frames
}

// StripPathPrefix removes a configured prefix from a file path, making it
// relative to the repository root.
func StripPathPrefix(filePath, prefix string) string {
	if prefix == "" {
		return filePath
	}
	// Handle both "services/" prefix and full path like "/app/services/"
	if idx := strings.Index(filePath, prefix); idx >= 0 {
		return filePath[idx+len(prefix):]
	}
	return filePath
}

// FindSuspectCommits analyzes blame data for stack trace files and identifies
// commits that are likely responsible for the error.
func FindSuspectCommits(provider SCMProvider, stack string, repos []RepoMapping) ([]SuspectCommit, error) {
	frames := ParseStackTrace(stack)
	if len(frames) == 0 {
		return nil, nil
	}

	cutoff := time.Now().Add(-suspectWindow)

	// Track which commits touched error lines: commitHash → SuspectCommit
	suspects := make(map[string]*SuspectCommit)

	for _, repo := range repos {
		repoFullName := repo.Owner + "/" + repo.Repo

		for _, frame := range frames {
			repoPath := StripPathPrefix(frame.FilePath, repo.PathPrefix)

			// Get blame for a small window around the error line (error line +/- 2)
			startLine := frame.Line - 2
			if startLine < 1 {
				startLine = 1
			}
			endLine := frame.Line + 2

			blameLines, err := provider.GetBlame(repoFullName, repoPath, startLine, endLine)
			if err != nil {
				// File may not exist in this repo — skip
				continue
			}

			for _, bl := range blameLines {
				if bl.Date.Before(cutoff) {
					continue
				}

				key := bl.CommitHash
				if s, ok := suspects[key]; ok {
					s.ErrorLines++
					if !containsString(s.ErrorFiles, repoPath) {
						s.ErrorFiles = append(s.ErrorFiles, repoPath)
					}
				} else {
					suspects[key] = &SuspectCommit{
						Commit: Commit{
							Hash:    bl.CommitHash,
							Author:  bl.Author,
							Message: bl.Message,
							Date:    bl.Date,
						},
						ErrorFiles: []string{repoPath},
						ErrorLines: 1,
					}
				}
			}
		}
	}

	// Convert to slice and score
	result := make([]SuspectCommit, 0, len(suspects))
	for _, s := range suspects {
		s.Score = scoreSuspect(s, cutoff)
		s.Reason = buildReason(s)
		result = append(result, *s)
	}

	// Sort by score descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	// Limit to top 10
	if len(result) > 10 {
		result = result[:10]
	}

	return result, nil
}

// scoreSuspect calculates a suspicion score based on recency and coverage.
func scoreSuspect(s *SuspectCommit, cutoff time.Time) float64 {
	// Recency score: 1.0 for now, decaying to 0.0 at the cutoff
	windowDuration := time.Now().Sub(cutoff).Seconds()
	age := time.Now().Sub(s.Date).Seconds()
	recency := 1.0 - (age / windowDuration)
	if recency < 0 {
		recency = 0
	}

	// Coverage score: more error lines touched = higher suspicion
	coverage := float64(s.ErrorLines) / 5.0 // normalize: 5 lines = max
	if coverage > 1.0 {
		coverage = 1.0
	}

	// File breadth: touching multiple error files is suspicious
	fileBreadth := float64(len(s.ErrorFiles)) / 3.0
	if fileBreadth > 1.0 {
		fileBreadth = 1.0
	}

	return (recency * 0.5) + (coverage * 0.3) + (fileBreadth * 0.2)
}

// buildReason returns a human-readable explanation for why a commit is suspect.
func buildReason(s *SuspectCommit) string {
	parts := []string{
		fmt.Sprintf("Touched %d error line(s)", s.ErrorLines),
		fmt.Sprintf("in %d file(s)", len(s.ErrorFiles)),
	}

	age := time.Since(s.Date)
	if age < 24*time.Hour {
		parts = append(parts, "committed today")
	} else {
		days := int(age.Hours() / 24)
		parts = append(parts, fmt.Sprintf("committed %d day(s) ago", days))
	}

	return strings.Join(parts, ", ")
}

// GetSourceContext retrieves source code lines around a file:line reference.
func GetSourceContext(provider SCMProvider, repo, filePath string, errorLine, contextLines int) (*SourceContext, error) {
	content, err := provider.GetFileContent(repo, filePath, "")
	if err != nil {
		return nil, err
	}

	allLines := strings.Split(content, "\n")
	startLine := errorLine - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := errorLine + contextLines
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	lines := make([]SourceLine, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		lineContent := ""
		if i-1 < len(allLines) {
			lineContent = allLines[i-1]
		}
		lines = append(lines, SourceLine{
			Number:  i,
			Content: lineContent,
			IsError: i == errorLine,
		})
	}

	return &SourceContext{
		Lines:     lines,
		StartLine: startLine,
		EndLine:   endLine,
		FilePath:  filePath,
		Repo:      repo,
	}, nil
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
