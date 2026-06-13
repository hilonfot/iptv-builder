package parser

import (
	"testing"
)

func TestParse_Basic(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 group-title="CCTV" ,CCTV1
http://stream.example.com/cctv1.ts
#EXTINF:-1 group-title="卫视" ,湖南卫视
http://stream.example.com/hunan.m3u8
`

	p := New()
	channels := p.Parse([]byte(content), "http://source.m3u")

	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}

	// Channel 1
	if channels[0].Name != "CCTV1" {
		t.Errorf("ch[0].Name = %q, want CCTV1", channels[0].Name)
	}
	if channels[0].Group != "CCTV" {
		t.Errorf("ch[0].Group = %q, want CCTV", channels[0].Group)
	}
	if channels[0].URL != "http://stream.example.com/cctv1.ts" {
		t.Errorf("ch[0].URL = %q", channels[0].URL)
	}
	if channels[0].Protocol != "ts" {
		t.Errorf("ch[0].Protocol = %q, want ts", channels[0].Protocol)
	}
	if channels[0].Source != "http://source.m3u" {
		t.Errorf("ch[0].Source = %q", channels[0].Source)
	}
	if !channels[0].Valid {
		t.Error("ch[0].Valid should be true")
	}

	// Channel 2
	if channels[1].Name != "湖南卫视" {
		t.Errorf("ch[1].Name = %q, want 湖南卫视", channels[1].Name)
	}
	if channels[1].Group != "卫视" {
		t.Errorf("ch[1].Group = %q, want 卫视", channels[1].Group)
	}
	if channels[1].Protocol != "m3u8" {
		t.Errorf("ch[1].Protocol = %q, want m3u8", channels[1].Protocol)
	}
}

func TestParse_NoGroupTitle(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 ,浙江卫视
http://stream/zhejiang.flv
`

	p := New()
	channels := p.Parse([]byte(content), "source")

	if len(channels) != 1 {
		t.Fatalf("got %d channels, want 1", len(channels))
	}
	if channels[0].Name != "浙江卫视" {
		t.Errorf("Name = %q, want 浙江卫视", channels[0].Name)
	}
	if channels[0].Group != "" {
		t.Errorf("Group = %q, want empty", channels[0].Group)
	}
	if channels[0].Protocol != "flv" {
		t.Errorf("Protocol = %q, want flv", channels[0].Protocol)
	}
}

func TestParse_EmptyContent(t *testing.T) {
	p := New()
	channels := p.Parse([]byte(""), "source")
	if len(channels) != 0 {
		t.Errorf("got %d channels, want 0", len(channels))
	}
}

func TestParse_CommentsIgnored(t *testing.T) {
	content := `#EXTM3U
# Some comment
#EXTINF:-1 ,CCTV1
http://stream/cctv1.ts
`
	p := New()
	channels := p.Parse([]byte(content), "source")
	if len(channels) != 1 {
		t.Fatalf("got %d channels, want 1", len(channels))
	}
}

func TestParse_URLWithoutEXTINF(t *testing.T) {
	// A URL line without a preceding EXTINF should be ignored.
	content := `#EXTM3U
http://stream/orphan.ts
#EXTINF:-1 ,CCTV1
http://stream/cctv1.ts
`
	p := New()
	channels := p.Parse([]byte(content), "source")
	if len(channels) != 1 {
		t.Fatalf("got %d channels, want 1", len(channels))
	}
	if channels[0].Name != "CCTV1" {
		t.Errorf("Name = %q, want CCTV1", channels[0].Name)
	}
}

func TestParse_MultipleChannelsSameGroup(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 group-title="CCTV" ,CCTV1
http://stream/cctv1.ts
#EXTINF:-1 group-title="CCTV" ,CCTV2
http://stream/cctv2.ts
#EXTINF:-1 group-title="CCTV" ,CCTV3
http://stream/cctv3.ts
`
	p := New()
	channels := p.Parse([]byte(content), "source")
	if len(channels) != 3 {
		t.Fatalf("got %d channels, want 3", len(channels))
	}
	for i, ch := range channels {
		if ch.Group != "CCTV" {
			t.Errorf("ch[%d].Group = %q, want CCTV", i, ch.Group)
		}
	}
}

func TestParse_ProtocolDetection(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://a.m3u8", "m3u8"},
		{"http://a.m3u", "m3u8"},
		{"http://a.flv", "flv"},
		{"http://a.ts", "ts"},
		{"http://a.ts?token=123", "ts"},
		{"http://a.m3u8?id=1&key=2", "m3u8"},
		{"http://a/stream", ""},
	}

	for _, tt := range tests {
		got := detectProtocol(tt.url)
		if got != tt.expected {
			t.Errorf("detectProtocol(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}

func TestParseEXTINF_WithGroup(t *testing.T) {
	name, group := parseEXTINF(`#EXTINF:-1 group-title="CCTV" ,CCTV1HD`)
	if name != "CCTV1HD" {
		t.Errorf("name = %q, want CCTV1HD", name)
	}
	if group != "CCTV" {
		t.Errorf("group = %q, want CCTV", group)
	}
}

func TestParseEXTINF_WithoutGroup(t *testing.T) {
	name, group := parseEXTINF(`#EXTINF:-1 ,湖南卫视`)
	if name != "湖南卫视" {
		t.Errorf("name = %q, want 湖南卫视", name)
	}
	if group != "" {
		t.Errorf("group = %q, want empty", group)
	}
}

func TestParseEXTINF_EmptyName(t *testing.T) {
	name, group := parseEXTINF(`#EXTINF:-1 group-title="卫视" ,`)
	if name != "" {
		t.Errorf("name = %q, want empty", name)
	}
	if group != "卫视" {
		t.Errorf("group = %q, want 卫视", group)
	}
}

func TestParseEXTINF_QuotesInGroupTitle(t *testing.T) {
	name, group := parseEXTINF(`#EXTINF:-1 group-title="Hello World" ,Test`)
	if name != "Test" {
		t.Errorf("name = %q, want Test", name)
	}
	if group != "Hello World" {
		t.Errorf("group = %q, want \"Hello World\"", group)
	}
}
