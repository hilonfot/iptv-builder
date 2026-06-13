package fetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RawSource holds the raw M3U content fetched from an IPTV source URL.
type RawSource struct {
	URL     string
	Content []byte
}

// Fetcher downloads M3U content from multiple IPTV source URLs concurrently.
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// New creates a Fetcher with the given per-source timeout.
func New(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				MaxConnsPerHost:     5,
			},
		},
		timeout: timeout,
	}
}

// Fetch concurrently downloads all source URLs. Failed sources are skipped
// with a warning log. Returns aggregated results.
func (f *Fetcher) Fetch(ctx context.Context, urls []string) []RawSource {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		results  []RawSource
		successN int
		failN    int
	)

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			content, err := f.fetchOne(ctx, u)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				slog.Warn("source fetch failed", "url", u, "error", err)
				failN++
				return
			}
			results = append(results, RawSource{URL: u, Content: content})
			successN++
		}(url)
	}

	wg.Wait()

	slog.Info("phase: fetch sources",
		"total", len(urls),
		"success", successN,
		"failed", failN,
	)

	return results
}

func (f *Fetcher) fetchOne(ctx context.Context, url string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exec request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Cap response body to prevent OOM from unexpectedly large responses.
	// Typical M3U files are < 10MB.
	limited := io.LimitReader(resp.Body, 10<<20) // 10 MiB
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return content, nil
}
