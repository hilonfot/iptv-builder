package generator

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// groupOrder defines the desired output ordering of channel groups.
var groupOrder = map[string]int{
	"CCTV": 0,
	"重庆": 1,
	"卫视": 2,
}

// Generate writes the final M3U playlist to the output file.
// Channels are grouped and sorted: CCTV → 重庆 → 卫视 → others.
func Generate(channels []*model.Channel, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outputPath := filepath.Join(outputDir, "final.m3u")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	// Write header.
	f.WriteString("#EXTM3U\n")

	// Sort channels: by group order, then by canonical name within group.
	sorted := sortChannels(channels)
	var lastGroup string
	for _, ch := range sorted {
		if ch == nil {
			continue
		}

		// Group comment separator.
		group := classifyGroup(ch.Group)
		if group != lastGroup {
			lastGroup = group
			fmt.Fprintf(f, "\n# === %s ===\n", group)
		}

		// Display name: prefer Canonical; append resolution if available.
		displayName := ch.Name
		if ch.Canonical != "" {
			displayName = ch.Canonical
		}
		if ch.Resolution != "" {
			displayName += " " + ch.Resolution
		}

		fmt.Fprintf(f, `#EXTINF:-1 group-title="%s" ,%s`, group, displayName)
		f.WriteString("\n")
		f.WriteString(ch.URL + "\n")
	}

	slog.Info("phase: generate m3u",
		"output", outputPath,
		"channels", len(sorted),
	)

	return nil
}

// sortChannels orders channels by group priority and then by canonical name.
func sortChannels(channels []*model.Channel) []*model.Channel {
	result := make([]*model.Channel, len(channels))
	copy(result, channels)

	sort.SliceStable(result, func(i, j int) bool {
		if result[i] == nil || result[j] == nil {
			return result[i] != nil
		}
		gi := groupPriority(classifyGroup(result[i].Group))
		gj := groupPriority(classifyGroup(result[j].Group))
		if gi != gj {
			return gi < gj
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// classifyGroup maps a raw group-title to one of the output group labels.
func classifyGroup(raw string) string {
	raw = strings.TrimSpace(raw)

	// Direct match.
	if _, ok := groupOrder[raw]; ok {
		return raw
	}

	// CCTV prefix matching.
	upper := strings.ToUpper(raw)
	if strings.HasPrefix(upper, "CCTV") {
		return "CCTV"
	}

	// 重庆 prefix.
	if strings.HasPrefix(raw, "重庆") {
		return "重庆"
	}

	// 卫视 suffix.
	if strings.Contains(raw, "卫视") || strings.HasSuffix(raw, "卫視") {
		return "卫视"
	}

	// Fallback: use the original group; if empty, classify by name pattern.
	if raw != "" {
		return raw
	}
	return "其他"
}

// groupPriority returns the sort order for a group label.
func groupPriority(g string) int {
	if p, ok := groupOrder[g]; ok {
		return p
	}
	return 3 // others after CCTV/重庆/卫视
}
