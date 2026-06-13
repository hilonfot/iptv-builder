package selector

import (
	"log/slog"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// SelectBest picks the single best channel per canonical group based on
// the highest QualityScore. Channels with Valid=false or QualityScore <= 0
// are skipped. Groups with no valid candidates are dropped entirely.
func SelectBest(groups map[string][]*model.Channel) []*model.Channel {
	var result []*model.Channel
	dropped := 0
	totalGroups := len(groups)

	for canonical, channels := range groups {
		if len(channels) == 0 {
			dropped++
			continue
		}

		var best *model.Channel
		for _, ch := range channels {
			if ch == nil || !ch.Valid || ch.QualityScore <= 0 {
				continue
			}
			if best == nil || ch.QualityScore > best.QualityScore {
				best = ch
			}
		}

		if best == nil {
			slog.Info("no valid channel for group, dropping",
				"canonical", canonical,
				"candidates", len(channels),
			)
			dropped++
			continue
		}

		result = append(result, best)
	}

	slog.Info("phase: select best",
		"total_groups", totalGroups,
		"selected", len(result),
		"dropped", dropped,
	)

	return result
}
