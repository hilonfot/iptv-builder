package normalizer

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/hilonfot/iptv-builder/internal/model"
)

// suffixPattern matches quality suffixes like [高清], [超清], [HD], [标清], etc.
var suffixPattern = regexp.MustCompile(`\s*\[(?:高清|超清|标清|HD|SD|4K|HDR|1080[Pp]|720[Pp])\]$`)

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
// Quality suffixes like [高清] / [超清] are stripped before matching.
func lookup(name string, aliases map[string]string) string {
	// 0. Strip [高清]/[超清]/[HD] etc. suffix for cleaner matching.
	clean := suffixPattern.ReplaceAllString(strings.TrimSpace(name), "")

	// 1. Exact match (original name).
	if canonical, ok := aliases[name]; ok {
		return canonical
	}

	// 2. Exact match (clean name).
	if canonical, ok := aliases[clean]; ok {
		return canonical
	}

	// 3. Trimmed match.
	trimmed := strings.TrimSpace(name)
	if canonical, ok := aliases[trimmed]; ok {
		return canonical
	}

	// 4. Trimmed + clean match.
	cleanTrimmed := strings.TrimSpace(clean)
	if canonical, ok := aliases[cleanTrimmed]; ok {
		return canonical
	}

	// 5. Case-insensitive fallback (viper lowercases YAML keys).
	nameLower := strings.ToLower(cleanTrimmed)
	for k, v := range aliases {
		if strings.ToLower(k) == nameLower {
			return v
		}
	}

	// 6. No match — use trimmed original name as canonical.
	return trimmed
}
