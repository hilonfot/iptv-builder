package model

import (
	"testing"
	"time"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	ttl := 24 * time.Hour

	tests := []struct {
		name      string
		updatedAt time.Time
		want      bool
	}{
		{
			name:      "within TTL",
			updatedAt: now.Add(-12 * time.Hour),
			want:      false,
		},
		{
			name:      "exactly at TTL boundary",
			updatedAt: now.Add(-24 * time.Hour),
			want:      true,
		},
		{
			name:      "beyond TTL",
			updatedAt: now.Add(-48 * time.Hour),
			want:      true,
		},
		{
			name:      "just updated",
			updatedAt: now,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := CacheEntry{UpdatedAt: tt.updatedAt}
			if got := e.IsExpired(ttl, now); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
