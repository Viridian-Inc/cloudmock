package pagination

import (
	"encoding/base64"
	"strconv"
)

// Page represents a paginated result.
type Page[T any] struct {
	Items     []T
	NextToken string
	HasMore   bool
}

// Paginate takes a sorted slice, a token (opaque cursor), and maxResults.
// Returns the page of items and the next token.
// Token is a base64-encoded index. Empty token means start from beginning.
// maxResults <= 0 means use defaultMax.
func Paginate[T any](items []T, token string, maxResults, defaultMax int) Page[T] {
	return PaginateWithFilter(items, token, maxResults, defaultMax, nil)
}

// PaginateWithFilter is like Paginate but applies a filter function first.
func PaginateWithFilter[T any](items []T, token string, maxResults, defaultMax int, filter func(T) bool) Page[T] {
	// Apply filter if provided.
	var filtered []T
	if filter != nil {
		for _, item := range items {
			if filter(item) {
				filtered = append(filtered, item)
			}
		}
	} else {
		filtered = items
	}

	// Resolve page size.
	pageSize := maxResults
	if pageSize <= 0 {
		pageSize = defaultMax
	}

	// Decode start index from token.
	start := 0
	if token != "" {
		decoded, err := base64.StdEncoding.DecodeString(token)
		if err == nil {
			idx, err := strconv.Atoi(string(decoded))
			if err == nil && idx >= 0 {
				start = idx
			}
		}
		// On any decode/parse error, fall back to start=0 (resilience).
	}

	// Guard against out-of-range start.
	if start >= len(filtered) {
		return Page[T]{Items: []T{}}
	}

	end := start + pageSize
	hasMore := end < len(filtered)
	if end > len(filtered) {
		end = len(filtered)
	}

	page := Page[T]{
		Items:   filtered[start:end],
		HasMore: hasMore,
	}

	if hasMore {
		page.NextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}

	return page
}
