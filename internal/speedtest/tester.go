package speedtest

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hilonfot/iptv-builder/internal/model"
	"github.com/panjf2000/ants/v2"
)

const (
	defaultTestTimeout = 5 * time.Second
	maxReadBytes       = 64 << 10 // 64 KB
)

// Tester measures stream latency for channels.
type Tester struct {
	client  *http.Client
	workers int
}

// New creates a new Tester with the given number of concurrent workers.
func New(workers int) *Tester {
	return &Tester{
		client: &http.Client{
			Timeout: defaultTestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:    100,
				IdleConnTimeout: 90 * time.Second,
				MaxConnsPerHost: 5,
			},
		},
		workers: workers,
	}
}

// CacheReader provides cached latency data.
type CacheReader interface {
	Get(canonical string) (model.CacheEntry, bool)
}

// CacheWriter persists speed test results.
type CacheWriter interface {
	Set(canonical string, entry model.CacheEntry)
	Save() error
}

// Test runs speed tests for all channels across groups, using cache when available.
func (t *Tester) Test(ctx context.Context, groups map[string][]*model.Channel, cache CacheReader) {
	pool, err := ants.NewPool(t.workers)
	if err != nil {
		slog.Error("create ants pool failed", "error", err)
		return
	}
	defer pool.Release()

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		total    int
		cached   int
		tested   int
		failed   int
	)

	for canonical, channels := range groups {
		// Check cache first — if valid, apply to all channels in group.
		if entry, ok := cache.Get(canonical); ok && !entry.IsExpired(24*time.Hour, time.Now()) {
			mu.Lock()
			for _, ch := range channels {
				if ch != nil {
					ch.LatencyMs = entry.LatencyMs
					ch.Valid = true
					total++
				}
			}
			cached++
			mu.Unlock()
			continue
		}

		for _, ch := range channels {
			if ch == nil {
				continue
			}
			mu.Lock()
			total++
			mu.Unlock()

			wg.Add(1)
			chRef := ch
			pool.Submit(func() {
				defer wg.Done()
				t.testOne(ctx, chRef)

				mu.Lock()
				if chRef.LatencyMs > 0 {
					tested++
				} else {
					failed++
					chRef.Valid = false
				}
				mu.Unlock()
			})
		}
	}

	wg.Wait()

	slog.Info("phase: speed test",
		"total", total,
		"cached_groups", cached,
		"tested", tested,
		"failed", failed,
	)
}

// testOne measures latency for a single channel.
func (t *Tester) testOne(ctx context.Context, ch *model.Channel) {
	reqCtx, cancel := context.WithTimeout(ctx, defaultTestTimeout)
	defer cancel()

	switch ch.Protocol {
	case "m3u8":
		ch.LatencyMs = t.testHLS(reqCtx, ch.URL)
	default:
		ch.LatencyMs = t.testDirect(reqCtx, ch.URL)
	}
}

// testHLS measures latency for an HLS stream by fetching the playlist and
// timing the first segment download.
func (t *Tester) testHLS(ctx context.Context, url string) int64 {
	start := time.Now()

	// 1. Fetch the m3u8 playlist.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	// Read playlist content (limited).
	limited := io.LimitReader(resp.Body, 512<<10) // 512KB max playlist
	playlist, err := io.ReadAll(limited)
	if err != nil {
		return 0
	}

	// 2. Find the first media segment URI (non-comment, non-m3u8).
	firstSegment := findFirstSegment(string(playlist), url)
	if firstSegment == "" {
		// No segment found — return playlist fetch time as baseline.
		elapsed := time.Since(start).Milliseconds()
	// Ensure at least 1ms for valid tests; httptest servers can respond in < 1ms.
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed
	}

	// 3. Fetch the first segment and measure.
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, firstSegment, nil)
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
	// Ensure at least 1ms for valid tests; httptest servers can respond in < 1ms.
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed
	}
	resp2, err := t.client.Do(req2)
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
	// Ensure at least 1ms for valid tests; httptest servers can respond in < 1ms.
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed
	}
	defer resp2.Body.Close()

	// Read enough to verify the stream is alive.
	io.CopyN(io.Discard, resp2.Body, 32<<10) // 32KB of segment data

	elapsed := time.Since(start).Milliseconds()
	// Ensure at least 1ms for valid tests; httptest servers can respond in < 1ms.
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed
}

// testDirect measures latency for a direct stream (FLV/TS) by fetching the
// first bytes.
func (t *Tester) testDirect(ctx context.Context, url string) int64 {
	start := time.Now()

	rctx, cancel := context.WithTimeout(ctx, defaultTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(rctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Warn("testDirect: create request failed", "url", url, "error", err)
		return 0
	}

	resp, err := t.client.Do(req)
	if err != nil {
		slog.Warn("testDirect: do request failed", "url", url, "error", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		slog.Warn("testDirect: unexpected status", "url", url, "status", resp.StatusCode)
		return 0
	}

	// Read up to 64KB to verify stream liveness.
	io.CopyN(io.Discard, resp.Body, maxReadBytes)

	elapsed := time.Since(start).Milliseconds()
	// Ensure at least 1ms for valid tests; httptest servers can respond in < 1ms.
	if elapsed <= 0 {
		elapsed = 1
	}
	return elapsed
}

// findFirstSegment extracts the first non-comment, non-playlist URI from HLS content.
func findFirstSegment(playlist, baseURL string) string {
	lines := strings.Split(playlist, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Skip nested playlists.
		if strings.HasSuffix(line, ".m3u8") || strings.HasSuffix(line, ".m3u") {
			continue
		}
		// Resolve relative URLs.
		return resolveURL(baseURL, line)
	}
	return ""
}

// resolveURL resolves a potentially relative URI against the base URL.
func resolveURL(base, ref string) string {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	// Strip filename from base.
	idx := strings.LastIndex(base, "/")
	if idx < 8 { // "https://"
		return ref
	}
	return base[:idx+1] + ref
}
