package speedtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// ---- Resolution helpers -----------------------------------------------------

func TestFindFirstSegment(t *testing.T) {
	tests := []struct {
		playlist string
		base     string
		want     string
	}{
		{
			"#EXTM3U\n#EXTINF:10,\nsegment001.ts\n#EXTINF:10,\nsegment002.ts\n",
			"http://example.com/playlist.m3u8",
			"http://example.com/segment001.ts",
		},
		{
			"#EXTM3U\n#EXTINF:10,\nhttp://other.com/seg.ts\n",
			"http://example.com/playlist.m3u8",
			"http://other.com/seg.ts",
		},
		{
			"#EXTM3U\n#EXT-X-STREAM-INF:RESOLUTION=1920x1080\nsub.m3u8\n#EXTINF:10,\nseg.ts\n",
			"http://example.com/master.m3u8",
			"http://example.com/seg.ts",
		},
		{
			"#EXTM3U\n",
			"http://example.com/playlist.m3u8",
			"",
		},
	}

	for _, tt := range tests {
		got := findFirstSegment(tt.playlist, tt.base)
		if got != tt.want {
			t.Errorf("findFirstSegment(..., %q) = %q, want %q", tt.base, got, tt.want)
		}
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base, ref, want string
	}{
		{"http://a.com/b/c.m3u8", "d.ts", "http://a.com/b/d.ts"},
		{"http://a.com/b/c.m3u8", "http://x.com/y.ts", "http://x.com/y.ts"},
		{"http://a.com/c.m3u8", "d.ts", "http://a.com/d.ts"},
	}
	for _, tt := range tests {
		got := resolveURL(tt.base, tt.ref)
		if got != tt.want {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.ref, got, tt.want)
		}
	}
}

// ---- Mock cache -------------------------------------------------------------

type mockCache struct {
	data map[string]model.CacheEntry
}

func (m *mockCache) Get(key string) (model.CacheEntry, bool) {
	e, ok := m.data[key]
	return e, ok
}

func (m *mockCache) Set(key string, e model.CacheEntry) {
	m.data[key] = e
}

func (m *mockCache) Save() error { return nil }

// ---- Speed tests ------------------------------------------------------------

func TestTest_DirectStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 64<<10)) // 64KB
	}))
	defer srv.Close()

	tt := New(10)
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1", URL: srv.URL, Protocol: "ts", Valid: true},
		},
	}
	cache := &mockCache{data: map[string]model.CacheEntry{}}

	ctx := context.Background()
	tt.Test(ctx, groups, cache)

	ch := groups["CCTV1"][0]
	if ch.LatencyMs <= 0 {
		t.Errorf("LatencyMs = %d, expected > 0", ch.LatencyMs)
	}
}

func TestTest_HLSStream(t *testing.T) {
	// Create a server that serves both the playlist and segments.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ".m3u8") {
			// Return media playlist pointing to segments.
			w.Write([]byte("#EXTM3U\n#EXTINF:10,\n/segment.ts\n"))
		} else {
			// Return segment data.
			w.Write(make([]byte, 32<<10))
		}
	}))
	defer srv.Close()

	tt := New(10)
	playlistURL := srv.URL + "/playlist.m3u8"
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1", URL: playlistURL, Protocol: "m3u8", Valid: true},
		},
	}
	cache := &mockCache{data: map[string]model.CacheEntry{}}

	ctx := context.Background()
	tt.Test(ctx, groups, cache)

	ch := groups["CCTV1"][0]
	if ch.LatencyMs <= 0 {
		t.Errorf("LatencyMs = %d, expected > 0", ch.LatencyMs)
	}
}

func TestTest_Unreachable(t *testing.T) {
	tt := New(10)
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1", URL: "http://127.0.0.1:19999/stream.ts", Protocol: "ts", Valid: true},
		},
	}
	cache := &mockCache{data: map[string]model.CacheEntry{}}

	ctx := context.Background()
	tt.Test(ctx, groups, cache)

	ch := groups["CCTV1"][0]
	if ch.LatencyMs != 0 {
		t.Errorf("LatencyMs = %d, want 0 (unreachable)", ch.LatencyMs)
	}
	if ch.Valid {
		t.Error("Valid should be false for unreachable stream")
	}
}

func TestTest_CacheHit(t *testing.T) {
	tt := New(10)
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1", URL: "http://example.com/stream.ts", Protocol: "ts", Valid: true},
		},
	}
	cache := &mockCache{data: map[string]model.CacheEntry{
		"CCTV1": {
			LatencyMs: 150,
			UpdatedAt: time.Now(),
		},
	}}

	ctx := context.Background()
	tt.Test(ctx, groups, cache)

	ch := groups["CCTV1"][0]
	if ch.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d, want 150 (cached)", ch.LatencyMs)
	}
}

func TestTest_CacheExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 64<<10))
	}))
	defer srv.Close()

	tt := New(10)
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1", URL: srv.URL, Protocol: "ts", Valid: true},
		},
	}
	cache := &mockCache{data: map[string]model.CacheEntry{
		"CCTV1": {
			LatencyMs: 999,
			UpdatedAt: time.Now().Add(-48 * time.Hour), // expired
		},
	}}

	ctx := context.Background()
	tt.Test(ctx, groups, cache)

	ch := groups["CCTV1"][0]
	// Should NOT use the expired cached value.
	if ch.LatencyMs == 999 {
		t.Errorf("LatencyMs = 999, expected fresh test result (not expired cached)")
	}
	if ch.LatencyMs <= 0 {
		t.Errorf("LatencyMs = %d, expected > 0", ch.LatencyMs)
	}
}

func TestTest_Concurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Write(make([]byte, 64<<10))
	}))
	defer srv.Close()

	tt := New(10) // 10 concurrent workers
	groups := make(map[string][]*model.Channel)
	for i := 0; i < 20; i++ {
		key := "Channel" + string(rune('A'+i))
		groups[key] = []*model.Channel{
			{Name: key, URL: srv.URL, Protocol: "ts", Valid: true},
		}
	}
	cache := &mockCache{data: map[string]model.CacheEntry{}}

	ctx := context.Background()
	start := time.Now()
	tt.Test(ctx, groups, cache)
	elapsed := time.Since(start)

	// 20 channels × 50ms sleep ÷ 10 workers = ~100ms minimum.
	// With overhead, should be < 500ms for concurrent execution.
	if elapsed > 2*time.Second {
		t.Errorf("concurrent test took %v, expected < 2s", elapsed)
	}

	successCount := 0
	for _, chs := range groups {
		if chs[0].LatencyMs > 0 {
			successCount++
		}
	}
	if successCount < 15 {
		t.Errorf("only %d/20 succeeded", successCount)
	}
}
