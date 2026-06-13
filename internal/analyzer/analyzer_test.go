package analyzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func TestDetectResolutionFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"CCTV4K", "4K"},
		{"CCTV-4K测试", "4K"},
		{"CCTV1 UHD", "4K"},
		{"CCTV1 4k", "4K"},
		{"HDR频道", "4K"},
		{"CCTV1 2160P", "4K"},
		{"CCTV1 1080P", "1080P"},
		{"CCTV1 1080I", "1080P"},
		{"CCTV1 FHD", "1080P"},
		{"CCTV1 1080p", "1080P"},
		{"CCTV1 720P", "720P"},
		{"CCTV1 720p", "720P"},
		{"CCTV1", ""},
		{"湖南卫视", ""},
		{"普通频道", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectResolutionFromName(tt.name)
			if got != tt.expected {
				t.Errorf("detectResolutionFromName(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://a.m3u8", "m3u8"},
		{"http://a.m3u", "m3u8"},
		{"http://a.flv", "flv"},
		{"http://a.ts", "ts"},
		{"http://a.ts?token=abc", "ts"},
		{"http://a/stream", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := detectProtocol(tt.url)
			if got != tt.expected {
				t.Errorf("detectProtocol(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestExtractResolution(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{`#EXT-X-STREAM-INF:RESOLUTION=1920x1080`, "1080P"},
		{`#EXT-X-STREAM-INF:RESOLUTION=3840x2160`, "4K"},
		{`#EXT-X-STREAM-INF:RESOLUTION=1280x720`, "720P"},
		{`#EXT-X-STREAM-INF:BANDWIDTH=8000000,RESOLUTION=1920x1080`, "1080P"},
		{`#EXT-X-STREAM-INF:RESOLUTION=720x576`, "SD"},
		{`#EXT-X-STREAM-INF:BANDWIDTH=8000000`, ""},
		{`#EXTINF:-1 ,CCTV1`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := extractResolution(tt.line)
			if got != tt.expected {
				t.Errorf("extractResolution(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestExtractBandwidth(t *testing.T) {
	tests := []struct {
		line     string
		expected int64
	}{
		{`#EXT-X-STREAM-INF:BANDWIDTH=8000000`, 8000000},
		{`#EXT-X-STREAM-INF:BANDWIDTH=4000000,RESOLUTION=1920x1080`, 4000000},
		{`#EXT-X-STREAM-INF:BANDWIDTH=2000000`, 2000000},
		{`#EXT-X-STREAM-INF:RESOLUTION=1920x1080`, 0},
		{`#EXTINF:-1 ,CCTV1`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := extractBandwidth(tt.line)
			if got != tt.expected {
				t.Errorf("extractBandwidth(%q) = %d, want %d", tt.line, got, tt.expected)
			}
		})
	}
}

func TestAnalyze_Basic(t *testing.T) {
	a := New()
	groups := map[string][]*model.Channel{
		"CCTV1": {
			{Name: "CCTV1 4K", URL: "http://stream/a.ts", Valid: true},
			{Name: "CCTV1 1080P", URL: "http://stream/b.m3u8", Valid: true},
			{Name: "CCTV1 SD", URL: "http://stream/c.flv", Valid: true},
		},
		"湖南卫视": {
			{Name: "湖南卫视 720P", URL: "http://stream/d.m3u8", Valid: true},
		},
	}

	ctx := context.Background()
	a.Analyze(ctx, groups)

	chs := groups["CCTV1"]
	if chs[0].Resolution != "4K" {
		t.Errorf("CCTV1[0].Resolution = %q, want 4K", chs[0].Resolution)
	}
	if chs[0].Protocol != "ts" {
		t.Errorf("CCTV1[0].Protocol = %q, want ts", chs[0].Protocol)
	}
	if chs[1].Resolution != "1080P" {
		t.Errorf("CCTV1[1].Resolution = %q, want 1080P", chs[1].Resolution)
	}
	if chs[1].Protocol != "m3u8" {
		t.Errorf("CCTV1[1].Protocol = %q, want m3u8", chs[1].Protocol)
	}
	if chs[2].Resolution != "" {
		t.Errorf("CCTV1[2].Resolution = %q, want empty", chs[2].Resolution)
	}
	if chs[2].Protocol != "flv" {
		t.Errorf("CCTV1[2].Protocol = %q, want flv", chs[2].Protocol)
	}

	hn := groups["湖南卫视"][0]
	if hn.Resolution != "720P" {
		t.Errorf("湖南卫视.Resolution = %q, want 720P", hn.Resolution)
	}
}

func TestAnalyze_HLSMasterPlaylist(t *testing.T) {
	// Set up a test server that returns a master HLS playlist.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=8000000,RESOLUTION=1920x1080
http://example.com/hd.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=4000000,RESOLUTION=1280x720
http://example.com/sd.m3u8
`))
	}))
	defer srv.Close()

	a := New()
	groups := map[string][]*model.Channel{
		"Test": {
			{Name: "Test", URL: srv.URL + "/playlist.m3u8", Valid: true},
		},
	}

	ctx := context.Background()
	a.Analyze(ctx, groups)

	ch := groups["Test"][0]
	if ch.Protocol != "m3u8" {
		t.Errorf("Protocol = %q, want m3u8", ch.Protocol)
	}
	if ch.Resolution != "1080P" {
		t.Errorf("Resolution = %q, want 1080P", ch.Resolution)
	}
	if ch.Bitrate != 8000000 {
		t.Errorf("Bitrate = %d, want 8000000", ch.Bitrate)
	}
}

func TestAnalyze_HLSMediaPlaylist(t *testing.T) {
	// A media playlist without BANDWIDTH/RESOLUTION.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`#EXTM3U
#EXTINF:10,
segment001.ts
#EXTINF:10,
segment002.ts
`))
	}))
	defer srv.Close()

	a := New()
	groups := map[string][]*model.Channel{
		"Test": {
			{Name: "Test 1080P", URL: srv.URL, Valid: true},
		},
	}

	ctx := context.Background()
	a.Analyze(ctx, groups)

	ch := groups["Test"][0]
	// Name-based detection should work.
	if ch.Resolution != "1080P" {
		t.Errorf("Resolution = %q, want 1080P (from name)", ch.Resolution)
	}
	// No bitrate from media playlist.
	if ch.Bitrate != 0 {
		t.Errorf("Bitrate = %d, want 0", ch.Bitrate)
	}
}

func TestAnalyze_UnreachableServer(t *testing.T) {
	a := New()
	groups := map[string][]*model.Channel{
		"Test": {
			{Name: "Test", URL: "http://127.0.0.1:19999/stream.m3u8", Valid: true},
		},
	}

	ctx := context.Background()
	a.Analyze(ctx, groups)

	// Should not panic, just leave fields empty.
	ch := groups["Test"][0]
	if ch.Protocol != "m3u8" {
		t.Errorf("Protocol = %q, want m3u8", ch.Protocol)
	}
}
