package scm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GitHubProvider implements SCMProvider using the GitHub REST API.
type GitHubProvider struct {
	token      string
	httpClient *http.Client

	cacheMu sync.RWMutex
	cache   map[string]*cacheEntry
}

type cacheEntry struct {
	data      string
	expiresAt time.Time
}

const (
	githubAPI     = "https://api.github.com"
	cacheTTL      = 1 * time.Hour
	maxRetryAfter = 60 // seconds
)

// NewGitHubProvider creates a new GitHub provider with the given personal access token.
func NewGitHubProvider(token string) *GitHubProvider {
	return &GitHubProvider{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string]*cacheEntry),
	}
}

// GetFileContent retrieves the content of a file from a GitHub repository.
func (g *GitHubProvider) GetFileContent(repo, path, ref string) (string, error) {
	cacheKey := fmt.Sprintf("file:%s:%s:%s", repo, path, ref)
	if cached := g.getCache(cacheKey); cached != "" {
		return cached, nil
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s", githubAPI, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}

	body, err := g.doRequest("GET", url)
	if err != nil {
		return "", fmt.Errorf("get file content %s/%s: %w", repo, path, err)
	}

	var resp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse file content response: %w", err)
	}

	if resp.Encoding != "base64" {
		return "", fmt.Errorf("unexpected encoding %q (expected base64)", resp.Encoding)
	}

	// GitHub base64 content may contain newlines
	cleaned := strings.ReplaceAll(resp.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("decode base64 content: %w", err)
	}

	content := string(decoded)
	g.setCache(cacheKey, content)
	return content, nil
}

// GetBlame retrieves blame information for a range of lines in a file.
// GitHub REST API does not have a direct blame endpoint, so we use the
// commits API to find who last modified the specific lines.
func (g *GitHubProvider) GetBlame(repo, path string, startLine, endLine int) ([]BlameLine, error) {
	// Get commits that touched this file
	url := fmt.Sprintf("%s/repos/%s/commits?path=%s&per_page=100", githubAPI, repo, path)
	body, err := g.doRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("get blame for %s/%s: %w", repo, path, err)
	}

	var commits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
			Message string `json:"message"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(body, &commits); err != nil {
		return nil, fmt.Errorf("parse commits response: %w", err)
	}

	// Build blame lines from the most recent commit touching the file.
	// In a full implementation we'd use the GraphQL API for precise blame,
	// but for v1 we attribute all requested lines to the most recent commit.
	lines := make([]BlameLine, 0, endLine-startLine+1)
	for i := startLine; i <= endLine; i++ {
		bl := BlameLine{Line: i}
		if len(commits) > 0 {
			c := commits[0]
			bl.CommitHash = c.SHA
			bl.Author = c.Commit.Author.Name
			bl.Date = c.Commit.Author.Date
			bl.Message = firstLine(c.Commit.Message)
		}
		lines = append(lines, bl)
	}

	return lines, nil
}

// ListRecentCommits returns the most recent commits for a repository.
func (g *GitHubProvider) ListRecentCommits(repo string, limit int) ([]Commit, error) {
	if limit <= 0 {
		limit = 20
	}
	url := fmt.Sprintf("%s/repos/%s/commits?per_page=%d", githubAPI, repo, limit)

	body, err := g.doRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("list commits for %s: %w", repo, err)
	}

	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
			Message string `json:"message"`
		} `json:"commit"`
		Files []struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse commits response: %w", err)
	}

	commits := make([]Commit, len(raw))
	for i, r := range raw {
		files := make([]string, len(r.Files))
		for j, f := range r.Files {
			files[j] = f.Filename
		}
		commits[i] = Commit{
			Hash:         r.SHA,
			Author:       r.Commit.Author.Name,
			Message:      firstLine(r.Commit.Message),
			Date:         r.Commit.Author.Date,
			FilesChanged: files,
		}
	}

	return commits, nil
}

// doRequest performs an authenticated HTTP request to GitHub, respecting rate limits.
func (g *GitHubProvider) doRequest(method, url string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			secs, _ := strconv.Atoi(retryAfter)
			if secs > 0 && secs <= maxRetryAfter {
				time.Sleep(time.Duration(secs) * time.Second)
				return g.doRequest(method, url)
			}
		}
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		resetStr := resp.Header.Get("X-RateLimit-Reset")
		return nil, fmt.Errorf("github rate limited (remaining=%s, reset=%s)", remaining, resetStr)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (g *GitHubProvider) getCache(key string) string {
	g.cacheMu.RLock()
	defer g.cacheMu.RUnlock()
	if e, ok := g.cache[key]; ok && time.Now().Before(e.expiresAt) {
		return e.data
	}
	return ""
}

func (g *GitHubProvider) setCache(key, data string) {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	g.cache[key] = &cacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
