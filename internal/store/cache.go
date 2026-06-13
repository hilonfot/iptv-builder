package store

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// Cache persists per-channel quality data to a JSON file on disk.
// It is safe for concurrent use.
type Cache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]model.CacheEntry
	ttl     time.Duration
}

// New creates a new cache Store and ensures the parent directory exists.
func New(path string, ttl time.Duration) *Cache {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Warn("failed to create cache dir", "dir", filepath.Dir(path), "error", err)
	}
	return &Cache{
		path:    path,
		entries: make(map[string]model.CacheEntry),
		ttl:     ttl,
	}
}

// Load reads the cache file from disk. Missing or unreadable files
// result in a warning and an empty cache (no fatal error).
func (s *Cache) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("cache file not found, starting fresh", "path", s.path)
			return nil
		}
		slog.Warn("failed to read cache file", "path", s.path, "error", err)
		return nil
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		slog.Warn("failed to parse cache file, starting fresh", "path", s.path, "error", err)
		s.entries = make(map[string]model.CacheEntry)
		return nil
	}

	// Prune expired entries on load.
	now := time.Now()
	pruned := 0
	for k, e := range s.entries {
		if e.IsExpired(s.ttl, now) {
			delete(s.entries, k)
			pruned++
		}
	}

	slog.Info("cache loaded",
		"path", s.path,
		"entries", len(s.entries),
		"pruned_expired", pruned,
	)
	return nil
}

// Get returns a cache entry if present and not expired.
func (s *Cache) Get(canonical string) (model.CacheEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.entries[canonical]
	if !ok {
		return model.CacheEntry{}, false
	}
	if e.IsExpired(s.ttl, time.Now()) {
		return model.CacheEntry{}, false
	}
	return e, true
}

// Set stores a cache entry.
func (s *Cache) Set(canonical string, entry model.CacheEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry.UpdatedAt = time.Now()
	s.entries[canonical] = entry
}

// Save persists all entries to disk. Errors are logged as warnings.
func (s *Cache) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		slog.Warn("failed to marshal cache", "error", err)
		return err
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		slog.Warn("failed to create cache directory", "path", filepath.Dir(s.path), "error", err)
		return err
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		slog.Warn("failed to write cache file", "path", s.path, "error", err)
		return err
	}

	slog.Info("cache saved", "path", s.path, "entries", len(s.entries))
	return nil
}

// Len returns the number of cached entries.
func (s *Cache) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}
