package normalizer

import (
	"log/slog"
	"strings"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// Normalize sets the Canonical field on each Channel using the alias map.
// Match order: exact → trimmed → no match (Canonical = original Name).
func Normalize(channels []*model.Channel, aliases map[string]string) {
	normalized := 0
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		ch.Canonical = lookup(ch.Name, aliases)
		if ch.Canonical != ch.Name {
			normalized++
		}
	}

	slog.Info("phase: normalize",
		"total", len(channels),
		"normalized", normalized,
		"aliases", len(aliases),
	)
}

// lookup finds the canonical name for a raw channel name in the alias map.
// Viper lowercases map keys, so we do case-insensitive fallback matching.
func lookup(name string, aliases map[string]string) string {
	// 1. Exact match.
	if canonical, ok := aliases[name]; ok {
		return canonical
	}

	// 2. Trimmed match.
	trimmed := strings.TrimSpace(name)
	if canonical, ok := aliases[trimmed]; ok {
		return canonical
	}

	// 3. Case-insensitive fallback (viper lowercases YAML keys).
	nameLower := strings.ToLower(trimmed)
	for k, v := range aliases {
		if strings.ToLower(k) == nameLower {
			return v
		}
	}

	// 4. No match — use trimmed original name as canonical.
	return trimmed
}
