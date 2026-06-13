package scorer

import (
	"log/slog"

	"github.com/hilonfot/iptv-builder/internal/config"
	"github.com/hilonfot/iptv-builder/internal/model"
)

// Score computes QualityScore for each channel in all groups.
func Score(groups map[string][]*model.Channel, weights config.QualityWeights) {
	var total, scored int
	for _, channels := range groups {
		for _, ch := range channels {
			if ch == nil || !ch.Valid {
				continue
			}
			total++
			ch.QualityScore = CalculateScore(ch, weights)
			scored++
		}
	}

	slog.Info("phase: calculate score",
		"total", total,
		"scored", scored,
		"weights[res]", weights.Resolution,
		"weights[bitrate]", weights.Bitrate,
		"weights[proto]", weights.Protocol,
		"weights[lat]", weights.Latency,
	)
}

// CalculateScore computes a single channel's composite quality score
// from its Resolution, Bitrate, Protocol, and Latency.
func CalculateScore(ch *model.Channel, weights config.QualityWeights) float64 {
	return resolutionScore(ch.Resolution)*weights.Resolution +
		bitrateScore(ch.Bitrate)*weights.Bitrate +
		protocolScore(ch.Protocol)*weights.Protocol +
		latencyScore(ch.LatencyMs)*weights.Latency
}

// resolutionScore scores the resolution on a 0–100 scale.
func resolutionScore(res string) float64 {
	switch res {
	case "4K":
		return 100
	case "1080P":
		return 70
	case "720P":
		return 40
	case "SD":
		return 10
	default:
		return 0
	}
}

// bitrateScore scores the bitrate (bps) on a 0–100 scale.
func bitrateScore(bps int64) float64 {
	switch {
	case bps >= 8_000_000:
		return 100
	case bps >= 4_000_000:
		return 70
	case bps >= 2_000_000:
		return 40
	case bps > 0:
		return 10
	default:
		return 0
	}
}

// protocolScore scores the transport protocol on a 0–100 scale.
// HLS is most stable (adaptive bitrate), TS moderate, FLV least.
func protocolScore(proto string) float64 {
	switch proto {
	case "m3u8":
		return 100
	case "ts":
		return 50
	case "flv":
		return 30
	default:
		return 0
	}
}

// latencyScore scores the latency (ms) on a 0–100 scale.
// Lower latency = higher score.
func latencyScore(ms int64) float64 {
	switch {
	case ms > 0 && ms <= 200:
		return 100
	case ms > 0 && ms <= 500:
		return 70
	case ms > 0 && ms <= 1000:
		return 40
	case ms > 0 && ms <= 5000:
		return 10
	default:
		return 0 // untested / timeout
	}
}
