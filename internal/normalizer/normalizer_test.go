package normalizer

import (
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestNormalize_ExactMatch(t *testing.T) {
	ch := []*model.Channel{{Name: "央视综合"}}
	aliases := map[string]string{"央视综合": "CCTV1"}

	Normalize(ch, aliases)

	if ch[0].Canonical != "CCTV1" {
		t.Errorf("Canonical = %q, want CCTV1", ch[0].Canonical)
	}
}

func TestNormalize_MultipleMatches(t *testing.T) {
	channels := []*model.Channel{
		{Name: "央视综合"},
		{Name: "CCTV-1"},
		{Name: "湖南卫视HD"},
	}
	aliases := map[string]string{
		"央视综合":   "CCTV1",
		"CCTV-1":  "CCTV1",
		"湖南卫视HD": "湖南卫视",
	}

	Normalize(channels, aliases)

	if channels[0].Canonical != "CCTV1" {
		t.Errorf("ch[0].Canonical = %q, want CCTV1", channels[0].Canonical)
	}
	if channels[1].Canonical != "CCTV1" {
		t.Errorf("ch[1].Canonical = %q, want CCTV1", channels[1].Canonical)
	}
	if channels[2].Canonical != "湖南卫视" {
		t.Errorf("ch[2].Canonical = %q, want 湖南卫视", channels[2].Canonical)
	}
}

func TestNormalize_NoMatch(t *testing.T) {
	ch := []*model.Channel{{Name: "未知频道"}}
	aliases := map[string]string{"CCTV1": "CCTV1"}

	Normalize(ch, aliases)

	if ch[0].Canonical != "未知频道" {
		t.Errorf("Canonical = %q, want 未知频道", ch[0].Canonical)
	}
}

func TestNormalize_TrimmedMatch(t *testing.T) {
	ch := []*model.Channel{{Name: " 央视综合 "}}
	aliases := map[string]string{"央视综合": "CCTV1"}

	Normalize(ch, aliases)

	if ch[0].Canonical != "CCTV1" {
		t.Errorf("Canonical = %q, want CCTV1", ch[0].Canonical)
	}
}

func TestNormalize_EmptyList(t *testing.T) {
	Normalize(nil, map[string]string{"a": "b"})
	// Should not panic.
}

func TestNormalize_NilChannel(t *testing.T) {
	channels := []*model.Channel{nil, {Name: "湖南卫视HD"}}
	aliases := map[string]string{"湖南卫视HD": "湖南卫视"}

	Normalize(channels, aliases)

	if channels[1].Canonical != "湖南卫视" {
		t.Errorf("Canonical = %q, want 湖南卫视", channels[1].Canonical)
	}
}

func TestNormalize_EmptyAliases(t *testing.T) {
	ch := []*model.Channel{{Name: "CCTV1"}}
	Normalize(ch, map[string]string{})
	if ch[0].Canonical != "CCTV1" {
		t.Errorf("Canonical = %q, want CCTV1", ch[0].Canonical)
	}
}
