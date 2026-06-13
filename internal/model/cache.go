package model

import "time"

// CacheEntry stores per-channel quality data for reuse across builds.
type CacheEntry struct {
	URL          string    `json:"url"`
	Resolution   string    `json:"resolution"`
	Bitrate      int64     `json:"bitrate"`
	Protocol     string    `json:"protocol"`
	LatencyMs    int64     `json:"latency_ms"`
	QualityScore float64   `json:"quality_score"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsExpired returns whether this cache entry exceeds the given TTL.
func (e CacheEntry) IsExpired(ttl time.Duration, now time.Time) bool {
	// Consider the entry expired at or after the TTL boundary.
	return !now.Before(e.UpdatedAt.Add(ttl))
}
