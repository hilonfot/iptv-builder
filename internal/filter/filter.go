package filter

import (
	"log/slog"
	"strings"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// Filter removes unwanted channels and retains only target channels.
// Order: exclude first, then keep — matching TASK.md specification.
func Filter(channels []*model.Channel, exclude, keep []string) []*model.Channel {
	before := len(channels)

	// Pass 1: exclude — mark channels matching any exclude keyword.
	if len(exclude) > 0 {
		channels = excludeChannels(channels, exclude)
	}

	// Pass 2: keep — retain only channels matching any keep keyword.
	channels = keepChannels(channels, keep)

	after := len(channels)
	slog.Info("phase: filter",
		"before", before,
		"after", after,
		"excluded", before-after,
	)

	return channels
}

// excludeChannels removes channels whose Name or Canonical contains any exclude keyword.
func excludeChannels(channels []*model.Channel, exclude []string) []*model.Channel {
	out := make([]*model.Channel, 0, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		if matchesAny(ch.Name, ch.Canonical, exclude) {
			ch.Valid = false
			continue
		}
		out = append(out, ch)
	}
	return out
}

// keepChannels retains only channels whose Name or Canonical contains any keep keyword.
func keepChannels(channels []*model.Channel, keep []string) []*model.Channel {
	out := make([]*model.Channel, 0, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		if !ch.Valid {
			continue
		}
		if matchesAny(ch.Name, ch.Canonical, keep) {
			out = append(out, ch)
		}
	}
	return out
}

// matchesAny checks whether the name or canonical contains any of the keywords (substring match).
func matchesAny(name, canonical string, keywords []string) bool {
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(name, kw) || strings.Contains(canonical, kw) {
			return true
		}
	}
	return false
}
