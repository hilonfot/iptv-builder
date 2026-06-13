package dedupe

import (
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestDedupe_Basic(t *testing.T) {
	channels := []*model.Channel{
		{Name: "CCTV1HD", Canonical: "CCTV1", URL: "http://a.ts"},
		{Name: "CCTV-1", Canonical: "CCTV1", URL: "http://b.ts"},
		{Name: "CCTV2", Canonical: "CCTV2", URL: "http://c.ts"},
	}

	groups := Dedupe(channels)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if len(groups["CCTV1"]) != 2 {
		t.Errorf("CCTV1 has %d lines, want 2", len(groups["CCTV1"]))
	}
	if len(groups["CCTV2"]) != 1 {
		t.Errorf("CCTV2 has %d lines, want 1", len(groups["CCTV2"]))
	}
}

func TestDedupe_SingleChannelPerGroup(t *testing.T) {
	channels := []*model.Channel{
		{Name: "CCTV1", Canonical: "CCTV1"},
		{Name: "湖南卫视", Canonical: "湖南卫视"},
	}

	groups := Dedupe(channels)

	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
}

func TestDedupe_EmptyInput(t *testing.T) {
	groups := Dedupe(nil)
	if len(groups) != 0 {
		t.Errorf("got %d groups, want 0", len(groups))
	}
}

func TestDedupe_EmptyCanonical(t *testing.T) {
	channels := []*model.Channel{
		{Name: "Unknown", Canonical: ""},
		{Name: "CCTV1", Canonical: "CCTV1"},
	}

	groups := Dedupe(channels)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if _, ok := groups[""]; ok {
		t.Error("empty canonical should be skipped")
	}
}

func TestDedupe_NilChannel(t *testing.T) {
	channels := []*model.Channel{
		nil,
		{Name: "CCTV1", Canonical: "CCTV1"},
	}

	groups := Dedupe(channels)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
}

func TestDedupe_MultiSource(t *testing.T) {
	channels := []*model.Channel{
		{Name: "CCTV1", Canonical: "CCTV1", Source: "http://source1.m3u"},
		{Name: "CCTV1", Canonical: "CCTV1", Source: "http://source2.m3u"},
		{Name: "CCTV1", Canonical: "CCTV1", Source: "http://source3.m3u"},
	}

	groups := Dedupe(channels)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}
	if len(groups["CCTV1"]) != 3 {
		t.Errorf("CCTV1 has %d lines, want 3", len(groups["CCTV1"]))
	}

	// Verify distinct sources preserved.
	seen := make(map[string]bool)
	for _, ch := range groups["CCTV1"] {
		seen[ch.Source] = true
	}
	if len(seen) != 3 {
		t.Errorf("got %d distinct sources, want 3", len(seen))
	}
}
