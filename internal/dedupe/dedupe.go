package dedupe

import (
	"log/slog"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// Dedupe groups channels by their Canonical name. Channels with empty
// Canonical are skipped.
func Dedupe(channels []*model.Channel) map[string][]*model.Channel {
	groups := make(map[string][]*model.Channel)
	skipped := 0

	for _, ch := range channels {
		if ch == nil || ch.Canonical == "" {
			skipped++
			continue
		}
		groups[ch.Canonical] = append(groups[ch.Canonical], ch)
	}

	totalLines := 0
	for _, g := range groups {
		totalLines += len(g)
	}

	slog.Info("phase: dedupe",
		"unique_canonicals", len(groups),
		"total_lines", totalLines,
		"skipped", skipped,
	)

	return groups
}
