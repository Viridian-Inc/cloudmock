// Package filestore provides a generic JSON file-backed store.
// Items are persisted as individual JSON files in a directory.
// Thread-safe. Items are identified by string ID.
// Storage layout: {dir}/{id}.json
package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// JSONFileStore[T] persists items as individual JSON files in a directory.
type JSONFileStore[T any] struct {
	dir string
	mu  sync.RWMutex
	seq atomic.Int64
}

// New creates a JSONFileStore backed by the given directory.
// The directory is created if it doesn't exist. An atomic counter
// is initialized by counting existing files for ID generation.
func New[T any](dir string) (*JSONFileStore[T], error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("filestore: create dir: %w", err)
	}

	s := &JSONFileStore[T]{dir: dir}

	// Count existing .json files to seed the sequence counter.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("filestore: read dir: %w", err)
	}
	var count int64
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	s.seq.Store(count)

	return s, nil
}

// Save marshals item to JSON and writes it to {dir}/{id}.json.
func (s *JSONFileStore[T]) Save(id string, item T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("filestore: marshal: %w", err)
	}
	return os.WriteFile(s.path(id), data, 0644)
}

// Get reads and unmarshals the item with the given ID.
func (s *JSONFileStore[T]) Get(id string) (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zero T
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return zero, fmt.Errorf("filestore: %s not found", id)
		}
		return zero, fmt.Errorf("filestore: read: %w", err)
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return zero, fmt.Errorf("filestore: unmarshal: %w", err)
	}
	return item, nil
}

// List reads all .json files in the directory and returns the deserialized items.
func (s *JSONFileStore[T]) List() ([]T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("filestore: read dir: %w", err)
	}

	var items []T
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var item T
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// Delete removes the file for the given ID.
func (s *JSONFileStore[T]) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path(id))
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("filestore: %s not found", id)
	}
	return err
}

// NextID returns a monotonically increasing ID with the given prefix,
// e.g. NextID("rule") returns "rule-1", "rule-2", etc.
func (s *JSONFileStore[T]) NextID(prefix string) string {
	n := s.seq.Add(1)
	return fmt.Sprintf("%s-%d", prefix, n)
}

// path returns the file path for an ID.
func (s *JSONFileStore[T]) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}
