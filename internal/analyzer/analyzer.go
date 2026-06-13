package analyzer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// resolutionPatterns matches resolution indicators in channel names.
// Match resolution keywords. No leading \b since keywords may be
// concatenated directly after letters (e.g., "CCTV4K").
var resolutionPattern = regexp.MustCompile(`(?i)(4K|2160[Pp]|UHD|HDR|1080[PpIi]|FHD|720[Pp])\b`)

// Analyzer detects Resolution, Bitrate, and Protocol for each channel.
type Analyzer struct {
	client *http.Client
}

// New creates a new Analyzer.
func New() *Analyzer {
	return &Analyzer{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Analyze processes all channels in each canonical group, setting
// Resolution, Bitrate, and Protocol fields.
func (a *Analyzer) Analyze(ctx context.Context, groups map[string][]*model.Channel) {
	var total, resDetected, brDetected int

	for _, channels := range groups {
		for _, ch := range channels {
			if ch == nil {
				continue
			}
			total++
			ch.Protocol = detectProtocol(ch.URL)

			// 1. Try name-based resolution detection.
			ch.Resolution = detectResolutionFromName(ch.Name)

			// 2. For HLS streams, try master playlist for better data.
			if ch.Protocol == "m3u8" {
				a.enrichFromHLS(ctx, ch)
			}

			if ch.Resolution != "" {
				resDetected++
			}
			if ch.Bitrate > 0 {
				brDetected++
			}
		}
	}

	slog.Info("phase: analyze quality",
		"total", total,
		"resolution_detected", resDetected,
		"bitrate_detected", brDetected,
	)
}

// detectResolutionFromName extracts resolution from channel name text.
func detectResolutionFromName(name string) string {
	match := resolutionPattern.FindString(strings.ToUpper(name))
	return normalizeResolution(match)
}

// normalizeResolution maps raw resolution strings to standard labels.
func normalizeResolution(raw string) string {
	switch strings.ToUpper(raw) {
	case "4K", "2160P", "UHD":
		return "4K"
	case "HDR":
		return "4K" // HDR typically implies 4K
	case "1080P", "1080I", "FHD":
		return "1080P"
	case "720P":
		return "720P"
	default:
		return ""
	}
}

// enrichFromHLS fetches the HLS playlist and extracts RESOLUTION / BANDWIDTH
// from a master playlist. For media playlists, best-effort from first segment.
func (a *Analyzer) enrichFromHLS(ctx context.Context, ch *model.Channel) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, ch.URL, nil)
	if err != nil {
		return
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Read first 64KB — enough for any master playlist.
	limited := io.LimitReader(resp.Body, 64<<10)
	data, err := io.ReadAll(limited)
	if err != nil {
		return
	}

	// Parse master playlist tags.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// RESOLUTION=1920x1080
		if resolution := extractResolution(line); resolution != "" {
			if ch.Resolution == "" {
				ch.Resolution = resolution
			}
		}

		// BANDWIDTH=8000000
		if bw := extractBandwidth(line); bw > 0 {
			if ch.Bitrate == 0 {
				ch.Bitrate = bw
			}
		}
	}
}

// extractResolution parses RESOLUTION=WxH from an EXT-X-STREAM-INF line.
func extractResolution(line string) string {
	idx := strings.Index(line, "RESOLUTION=")
	if idx < 0 {
		return ""
	}
	val := line[idx+len("RESOLUTION="):]
	// Value is e.g. "1920x1080" or "1920x1080,"
	if comma := strings.Index(val, ","); comma >= 0 {
		val = val[:comma]
	}
	parts := strings.SplitN(val, "x", 2)
	if len(parts) != 2 {
		return ""
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}
	switch {
	case h >= 2160:
		return "4K"
	case h >= 1080:
		return "1080P"
	case h >= 720:
		return "720P"
	default:
		return "SD"
	}
}

// extractBandwidth parses BANDWIDTH=N from an EXT-X-STREAM-INF line.
func extractBandwidth(line string) int64 {
	idx := strings.Index(line, "BANDWIDTH=")
	if idx < 0 {
		return 0
	}
	val := line[idx+len("BANDWIDTH="):]
	if comma := strings.Index(val, ","); comma >= 0 {
		val = val[:comma]
	}
	bw, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return bw
}

// detectProtocol determines the transport protocol from the URL extension.
func detectProtocol(url string) string {
	url = strings.ToLower(strings.TrimSpace(url))
	if idx := strings.Index(url, "?"); idx >= 0 {
		url = url[:idx]
	}
	switch {
	case strings.HasSuffix(url, ".m3u8") || strings.HasSuffix(url, ".m3u"):
		return "m3u8"
	case strings.HasSuffix(url, ".flv"):
		return "flv"
	case strings.HasSuffix(url, ".ts"):
		return "ts"
	default:
		return ""
	}
}

// String formats analyzer results for display.
func (a *Analyzer) String() string {
	return fmt.Sprintf("analyzer ready")
}
