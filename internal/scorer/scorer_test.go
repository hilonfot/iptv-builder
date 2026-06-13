package scorer

import (
	"testing"

	"github.com/hilonfot/iptv-builder/internal/config"
	"github.com/hilonfot/iptv-builder/internal/model"
)

var defaultWeights = config.QualityWeights{
	Resolution: 0.5,
	Bitrate:    0.2,
	Protocol:   0.1,
	Latency:    0.2,
}

func TestResolutionScore(t *testing.T) {
	tests := []struct {
		res      string
		expected float64
	}{
		{"4K", 100}, {"1080P", 70}, {"720P", 40}, {"SD", 10}, {"", 0},
	}
	for _, tt := range tests {
		got := resolutionScore(tt.res)
		if got != tt.expected {
			t.Errorf("resolutionScore(%q) = %v, want %v", tt.res, got, tt.expected)
		}
	}
}

func TestBitrateScore(t *testing.T) {
	tests := []struct {
		bps      int64
		expected float64
	}{
		{8_000_000, 100}, {10_000_000, 100},
		{4_000_000, 70}, {5_000_000, 70},
		{2_000_000, 40}, {3_000_000, 40},
		{1, 10}, {500_000, 10},
		{0, 0}, {-1, 0},
	}
	for _, tt := range tests {
		got := bitrateScore(tt.bps)
		if got != tt.expected {
			t.Errorf("bitrateScore(%d) = %v, want %v", tt.bps, got, tt.expected)
		}
	}
}

func TestProtocolScore(t *testing.T) {
	tests := []struct {
		proto    string
		expected float64
	}{
		{"m3u8", 100}, {"ts", 50}, {"flv", 30}, {"", 0}, {"unknown", 0},
	}
	for _, tt := range tests {
		got := protocolScore(tt.proto)
		if got != tt.expected {
			t.Errorf("protocolScore(%q) = %v, want %v", tt.proto, tt.expected, got)
		}
	}
}

func TestLatencyScore(t *testing.T) {
	tests := []struct {
		ms       int64
		expected float64
	}{
		{100, 100}, {200, 100},
		{201, 70}, {500, 70},
		{501, 40}, {1000, 40},
		{1001, 10}, {5000, 10},
		{5001, 0}, {0, 0}, {-1, 0},
	}
	for _, tt := range tests {
		got := latencyScore(tt.ms)
		if got != tt.expected {
			t.Errorf("latencyScore(%d) = %v, want %v", tt.ms, got, tt.expected)
		}
	}
}

func TestCalculateScore_Perfect(t *testing.T) {
	ch := &model.Channel{
		Resolution: "4K",
		Bitrate:    8_000_000,
		Protocol:   "m3u8",
		LatencyMs:  100,
	}
	score := CalculateScore(ch, defaultWeights)
	// 100*0.5 + 100*0.2 + 100*0.1 + 100*0.2 = 100
	if score != 100 {
		t.Errorf("perfect score = %v, want 100", score)
	}
}

func TestCalculateScore_Mixed(t *testing.T) {
	ch := &model.Channel{
		Resolution: "1080P",
		Bitrate:    3_000_000,
		Protocol:   "ts",
		LatencyMs:  350,
	}
	score := CalculateScore(ch, defaultWeights)
	// 70*0.5 + 40*0.2 + 50*0.1 + 70*0.2 = 35+8+5+14 = 62
	expected := float64(62)
	if score != expected {
		t.Errorf("score = %v, want %v", score, expected)
	}
}

func TestCalculateScore_Zero(t *testing.T) {
	ch := &model.Channel{}
	score := CalculateScore(ch, defaultWeights)
	if score != 0 {
		t.Errorf("score = %v, want 0", score)
	}
}

func TestScore_Groups(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "4K高速", Resolution: "4K", Bitrate: 10_000_000, Protocol: "m3u8", LatencyMs: 50, Valid: true},
			{Name: "1080P备用", Resolution: "1080P", Bitrate: 2_000_000, Protocol: "ts", LatencyMs: 500, Valid: true},
		},
	}

	Score(groups, defaultWeights)

	high := groups["CCTV1"][0].QualityScore
	low := groups["CCTV1"][1].QualityScore

	if high <= low {
		t.Errorf("4K高速 score(%v) should be > 1080P备用 score(%v)", high, low)
	}
}

func TestScore_CustomWeights(t *testing.T) {
	// Latency-heavy: only care about speed.
	ch := &model.Channel{
		Resolution: "1080P",
		Bitrate:    8_000_000,
		Protocol:   "m3u8",
		LatencyMs:  100,
	}
	latencyWeights := config.QualityWeights{
		Resolution: 0,
		Bitrate:    0,
		Protocol:   0,
		Latency:    1.0,
	}
	score := CalculateScore(ch, latencyWeights)
	if score != 100 {
		t.Errorf("latency-only score = %v, want 100", score)
	}
}
