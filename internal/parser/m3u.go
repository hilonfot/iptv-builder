package parser

import (
	"bufio"
	"bytes"
	"log/slog"
	"strings"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// Parser parses M3U/M3U8 playlist content into Channel models.
type Parser struct{}

// New creates a new Parser.
func New() *Parser {
	return &Parser{}
}

// Parse reads M3U content and returns a slice of Channels.
// Malformed lines are skipped with a warning.
func (p *Parser) Parse(content []byte, sourceURL string) []*model.Channel {
	var channels []*model.Channel

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var pendingName, pendingGroup string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments that are not EXTINF.
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#EXTINF:") {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			pendingName, pendingGroup = parseEXTINF(line)
			continue
		}

		// Non-comment line after EXTINF → this is the stream URL.
		url := strings.TrimSpace(line)
		if url == "" {
			continue
		}

		if pendingName != "" {
			ch := &model.Channel{
				Name:     pendingName,
				URL:      url,
				Group:    pendingGroup,
				Protocol: detectProtocol(url),
				Source:   sourceURL,
				Valid:    true,
			}
			channels = append(channels, ch)

			// Reset pending state.
			pendingName = ""
			pendingGroup = ""
		}
		// URL without a preceding EXTINF line is silently ignored.
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("m3u scan error", "source", sourceURL, "error", err)
	}

	slog.Info("phase: parse m3u", "source", sourceURL, "channels", len(channels))
	return channels
}

// parseEXTINF extracts the channel name and optional group-title from an EXTINF line.
// Format: #EXTINF:-1 group-title="GroupName" ,Channel Name
func parseEXTINF(line string) (name, group string) {
	// Strip prefix.
	rest := strings.TrimPrefix(line, "#EXTINF:")

	// Extract group-title if present.
	if gs := strings.Index(rest, `group-title="`); gs >= 0 {
		start := gs + len(`group-title="`)
		closeQuote := strings.Index(rest[start:], `"`)
		if closeQuote >= 0 {
			group = rest[start : start+closeQuote]
			// Remove only the group-title="..." substring so the comma-separated
			// name is preserved.
			endQuote := start + closeQuote + 1
			rest = rest[:gs] + rest[endQuote:]
		}
	}

	// Name follows the last comma.
	if idx := strings.LastIndex(rest, ","); idx >= 0 {
		name = strings.TrimSpace(rest[idx+1:])
	}

	return name, strings.TrimSpace(group)
}

// detectProtocol determines the stream transport protocol from the URL extension.
func detectProtocol(url string) string {
	url = strings.ToLower(strings.TrimSpace(url))
	// Strip query parameters for extension detection.
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
