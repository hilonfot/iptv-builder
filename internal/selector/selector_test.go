package selector

import (
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestSelectBest_HighestScore(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-A", Canonical: "CCTV1", QualityScore: 85, Valid: true},
			{Name: "CCTV1-B", Canonical: "CCTV1", QualityScore: 60, Valid: true},
			{Name: "CCTV1-C", Canonical: "CCTV1", QualityScore: 40, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
	if result[0].Name != "CCTV1-A" {
		t.Errorf("best = %s, want CCTV1-A", result[0].Name)
	}
}

func TestSelectBest_MultipleGroups(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-4K", Canonical: "CCTV1", QualityScore: 95, Valid: true},
			{Name: "CCTV1-SD", Canonical: "CCTV1", QualityScore: 20, Valid: true},
		},
		"湖南卫视": {
			{Name: "湖南卫视-1080P", Canonical: "湖南卫视", QualityScore: 70, Valid: true},
			{Name: "湖南卫视-SD", Canonical: "湖南卫视", QualityScore: 10, Valid: true},
		},
		"浙江卫视": {
			{Name: "浙江卫视-720P", Canonical: "浙江卫视", QualityScore: 50, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 3 {
		t.Fatalf("got %d, want 3", len(result))
	}

	names := make(map[string]float64)
	for _, ch := range result {
		names[ch.Name] = ch.QualityScore
	}
	if names["CCTV1-4K"] != 95 {
		t.Errorf("CCTV1-4K has score %v, want 95", names["CCTV1-4K"])
	}
}

func TestSelectBest_SkipInvalid(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-A", Canonical: "CCTV1", QualityScore: 80, Valid: false},
			{Name: "CCTV1-B", Canonical: "CCTV1", QualityScore: 50, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
	if result[0].Name != "CCTV1-B" {
		t.Errorf("got %s, want CCTV1-B", result[0].Name)
	}
}

func TestSelectBest_SkipZeroScore(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-A", Canonical: "CCTV1", QualityScore: 0, Valid: true},
			{Name: "CCTV1-B", Canonical: "CCTV1", QualityScore: -1, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 0 {
		t.Fatalf("got %d, want 0 (all zero/negative score)", len(result))
	}
}

func TestSelectBest_AllFailed(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-A", Canonical: "CCTV1", QualityScore: 0, Valid: false},
			{Name: "CCTV1-B", Canonical: "CCTV1", QualityScore: 0, Valid: false},
		},
	}

	result := SelectBest(groups)
	if len(result) != 0 {
		t.Fatalf("got %d, want 0", len(result))
	}
}

func TestSelectBest_EmptyGroup(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {},
	}

	result := SelectBest(groups)
	if len(result) != 0 {
		t.Fatalf("got %d, want 0", len(result))
	}
}

func TestSelectBest_SingleCandidate(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1-Only", Canonical: "CCTV1", QualityScore: 60, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
}

func TestSelectBest_NilChannel(t *testing.T) {
	groups := map[string][]*model.Channel{
		"CCTV1": {
			nil,
			{Name: "CCTV1-OK", Canonical: "CCTV1", QualityScore: 50, Valid: true},
		},
	}

	result := SelectBest(groups)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
	if result[0].Name != "CCTV1-OK" {
		t.Errorf("got %s, want CCTV1-OK", result[0].Name)
	}
}
