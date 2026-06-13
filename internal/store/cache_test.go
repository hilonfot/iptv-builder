package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestStore_SetGet(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "cache.json"), 24*time.Hour)

	entry := model.CacheEntry{
		URL:          "http://stream/1.ts",
		Resolution:   "1080P",
		Bitrate:      4_000_000,
		Protocol:     "ts",
		LatencyMs:    200,
		QualityScore: 62,
	}
	s.Set("CCTV1", entry)

	got, ok := s.Get("CCTV1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.LatencyMs != 200 {
		t.Errorf("LatencyMs = %d, want 200", got.LatencyMs)
	}
	if got.Resolution != "1080P" {
		t.Errorf("Resolution = %q, want 1080P", got.Resolution)
	}
}

func TestStore_Miss(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "cache.json"), 24*time.Hour)

	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestStore_Expired(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "cache.json"), 1*time.Hour)

	// Insert with an old timestamp.
	s.mu.Lock()
	s.entries["CCTV1"] = model.CacheEntry{
		URL:       "http://stream/1.ts",
		LatencyMs: 100,
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	s.mu.Unlock()

	_, ok := s.Get("CCTV1")
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	// Create, set, save.
	s1 := New(path, 24*time.Hour)
	s1.Set("CCTV1", model.CacheEntry{LatencyMs: 300, UpdatedAt: time.Now()})
	s1.Set("湖南卫视", model.CacheEntry{LatencyMs: 150, UpdatedAt: time.Now()})
	if err := s1.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load into a new store.
	s2 := New(path, 24*time.Hour)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	e1, ok := s2.Get("CCTV1")
	if !ok {
		t.Fatal("expected CCTV1 after reload")
	}
	if e1.LatencyMs != 300 {
		t.Errorf("LatencyMs = %d, want 300", e1.LatencyMs)
	}

	e2, _ := s2.Get("湖南卫视")
	if e2.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d, want 150", e2.LatencyMs)
	}
}

func TestStore_LoadMissingFile(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "no_such_file.json"), 24*time.Hour)
	if err := s.Load(); err != nil {
		t.Fatalf("Load() should not error on missing file: %v", err)
	}
	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "cache.json"), 24*time.Hour)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			s.Set("Channel", model.CacheEntry{LatencyMs: int64(i)})
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		s.Get("Channel")
	}
	<-done

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
}
