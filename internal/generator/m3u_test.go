package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestGenerate_Basic(t *testing.T) {
	dir := t.TempDir()

	channels := []*model.Channel{
		{Name: "CCTV1", Canonical: "CCTV1", URL: "http://stream/cctv1.ts", Group: "CCTV"},
		{Name: "CCTV2", Canonical: "CCTV2", URL: "http://stream/cctv2.ts", Group: "CCTV"},
		{Name: "湖南卫视", Canonical: "湖南卫视", URL: "http://stream/hunan.m3u8", Group: "卫视"},
		{Name: "重庆卫视", Canonical: "重庆卫视", URL: "http://stream/cq.ts", Group: "重庆"},
	}

	err := Generate(channels, dir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "final.m3u"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "#EXTM3U\n") {
		t.Error("missing #EXTM3U header")
	}
	if !strings.Contains(content, "# === CCTV ===") {
		t.Error("missing CCTV group header")
	}
	if !strings.Contains(content, "# === 重庆 ===") {
		t.Error("missing 重庆 group header")
	}
	if !strings.Contains(content, "# === 卫视 ===") {
		t.Error("missing 卫视 group header")
	}

	// Verify channel lines.
	for _, ch := range channels {
		if !strings.Contains(content, ch.URL) {
			t.Errorf("missing URL: %s", ch.URL)
		}
	}

	// CCTV should appear before 重庆, 重庆 before 卫视.
	cctvPos := strings.Index(content, "# === CCTV ===")
	cqPos := strings.Index(content, "# === 重庆 ===")
	tvPos := strings.Index(content, "# === 卫视 ===")
	if cctvPos < 0 || cqPos < 0 || tvPos < 0 {
		t.Fatal("missing group sections")
	}
	if cctvPos > cqPos || cqPos > tvPos {
		t.Errorf("group order wrong: CCTV=%d 重庆=%d 卫视=%d", cctvPos, cqPos, tvPos)
	}
}

func TestGenerate_WithResolution(t *testing.T) {
	dir := t.TempDir()

	channels := []*model.Channel{
		{Name: "CCTV1", Canonical: "CCTV1", URL: "http://s/cctv1.ts", Group: "CCTV", Resolution: "4K"},
	}

	Generate(channels, dir)

	data, _ := os.ReadFile(filepath.Join(dir, "final.m3u"))
	content := string(data)

	if !strings.Contains(content, "CCTV1 4K") {
		t.Errorf("expected 'CCTV1 4K' in output, got: %s", content)
	}
}

func TestGenerate_EmptyList(t *testing.T) {
	dir := t.TempDir()

	err := Generate(nil, dir)
	if err != nil {
		t.Fatalf("Generate() error on empty: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "final.m3u"))
	content := string(data)
	if content != "#EXTM3U\n" {
		t.Errorf("expected only header, got: %q", content)
	}
}

func TestGenerate_NilChannel(t *testing.T) {
	dir := t.TempDir()

	channels := []*model.Channel{
		nil,
		{Name: "CCTV1", Canonical: "CCTV1", URL: "http://s/cctv1.ts", Group: "CCTV"},
	}

	Generate(channels, dir)
	// Should not panic, and should include the valid channel.
	data, _ := os.ReadFile(filepath.Join(dir, "final.m3u"))
	if !strings.Contains(string(data), "http://s/cctv1.ts") {
		t.Error("missing valid channel URL after nil")
	}
}

func TestClassifyGroup(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"CCTV", "CCTV"},
		{"CCTV-1", "CCTV"},
		{"CCTV综合", "CCTV"},
		{"重庆", "重庆"},
		{"重庆卫视", "重庆"},
		{"卫视", "卫视"},
		{"湖南卫视", "卫视"},
		{"浙江卫视", "卫视"},
		{"东方卫视", "卫视"},
		{"卫視", "卫视"},
		{"", "其他"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := classifyGroup(tt.raw)
			if got != tt.want {
				t.Errorf("classifyGroup(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSortChannels(t *testing.T) {
	channels := []*model.Channel{
		{Name: "东方卫视", Group: "卫视"},
		{Name: "CCTV3", Group: "CCTV"},
		{Name: "湖南卫视", Group: "卫视"},
		{Name: "重庆卫视", Group: "重庆"},
		{Name: "CCTV1", Group: "CCTV"},
	}

	sorted := sortChannels(channels)

	groups := make([]string, len(sorted))
	for i, ch := range sorted {
		groups[i] = classifyGroup(ch.Group)
	}

	// CCTV first.
	if groups[0] != "CCTV" || groups[1] != "CCTV" {
		t.Error("first two should be CCTV")
	}
	// 重庆 second.
	if groups[2] != "重庆" {
		t.Error("third should be 重庆")
	}
	// 卫视 last.
	if groups[3] != "卫视" || groups[4] != "卫视" {
		t.Error("last two should be 卫视")
	}

	// Within CCTV: by name asc.
	if sorted[0].Name > sorted[1].Name {
		t.Errorf("CCTV not sorted by name: %s > %s", sorted[0].Name, sorted[1].Name)
	}
}

func TestGroupPriority(t *testing.T) {
	if groupPriority("CCTV") != 0 {
		t.Errorf("CCTV priority = %d, want 0", groupPriority("CCTV"))
	}
	if groupPriority("重庆") != 1 {
		t.Errorf("重庆 priority = %d, want 1", groupPriority("重庆"))
	}
	if groupPriority("卫视") != 2 {
		t.Errorf("卫视 priority = %d, want 2", groupPriority("卫视"))
	}
	if groupPriority("其他") != 3 {
		t.Errorf("其他 priority = %d, want 3", groupPriority("其他"))
	}
}
