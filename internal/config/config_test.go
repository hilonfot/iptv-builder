package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// ---- Full config load -------------------------------------------------------

func TestLoad_Full(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.yaml", "app:\n  cache_ttl_hours: 12\n  workers: 30\n  log_level: debug\n")
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n  - http://b.m3u\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - CCTV\n  - 卫视\nexclude:\n  - 购物\n")
	writeFile(t, dir, "aliases.yaml", "央视综合: CCTV1\n湖南卫视HD: 湖南卫视\n")
	writeFile(t, dir, "quality.yaml", "quality:\n  resolution_weight: 0.4\n  bitrate_weight: 0.3\n  protocol_weight: 0.1\n  latency_weight: 0.2\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.App.CacheTTLHours != 12 {
		t.Errorf("CacheTTLHours = %d, want 12", cfg.App.CacheTTLHours)
	}
	if cfg.App.Workers != 30 {
		t.Errorf("Workers = %d, want 30", cfg.App.Workers)
	}
	if cfg.App.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.App.LogLevel)
	}
	if len(cfg.Sources) != 2 {
		t.Errorf("Sources len = %d, want 2", len(cfg.Sources))
	}
	if len(cfg.Keep) != 2 {
		t.Errorf("Keep len = %d, want 2", len(cfg.Keep))
	}
	if len(cfg.Exclude) != 1 {
		t.Errorf("Exclude len = %d, want 1", len(cfg.Exclude))
	}
	if len(cfg.Aliases) != 2 {
		t.Errorf("Aliases len = %d, want 2", len(cfg.Aliases))
	}
	if cfg.Quality.Resolution != 0.4 {
		t.Errorf("Quality.Resolution = %f, want 0.4", cfg.Quality.Resolution)
	}
	if cfg.Quality.Bitrate != 0.3 {
		t.Errorf("Quality.Bitrate = %f, want 0.3", cfg.Quality.Bitrate)
	}

	// Stats should not panic.
	s := cfg.Stats()
	if s == "" {
		t.Error("Stats() returned empty string")
	}
}

// ---- Defaults for optional files --------------------------------------------

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - CCTV\n")
	writeFile(t, dir, "aliases.yaml", "a: b\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// App defaults
	if cfg.App.CacheTTLHours != 24 {
		t.Errorf("default CacheTTLHours = %d, want 24", cfg.App.CacheTTLHours)
	}
	if cfg.App.Workers != 50 {
		t.Errorf("default Workers = %d, want 50", cfg.App.Workers)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("default LogLevel = %s, want info", cfg.App.LogLevel)
	}

	// Quality defaults
	if cfg.Quality.Resolution != 0.5 {
		t.Errorf("default Resolution = %f, want 0.5", cfg.Quality.Resolution)
	}
	if cfg.Quality.Bitrate != 0.2 {
		t.Errorf("default Bitrate = %f, want 0.2", cfg.Quality.Bitrate)
	}
	if cfg.Quality.Protocol != 0.1 {
		t.Errorf("default Protocol = %f, want 0.1", cfg.Quality.Protocol)
	}
	if cfg.Quality.Latency != 0.2 {
		t.Errorf("default Latency = %f, want 0.2", cfg.Quality.Latency)
	}
}

// ---- Validation errors ------------------------------------------------------

func TestLoad_MissingSources(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Error("expected error for missing sources.yaml")
	}
}

func TestLoad_MissingChannels(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n")
	if _, err := Load(dir); err == nil {
		t.Error("expected error for missing channels.yaml")
	}
}

func TestLoad_MissingAliases(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - CCTV\n")
	if _, err := Load(dir); err == nil {
		t.Error("expected error for missing aliases.yaml")
	}
}

func TestLoad_EmptySources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - CCTV\n")
	writeFile(t, dir, "aliases.yaml", "a: b\n")
	if _, err := Load(dir); err == nil {
		t.Error("expected error for empty sources")
	}
}

func TestLoad_EmptyKeep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n")
	writeFile(t, dir, "channels.yaml", "keep:\n")
	writeFile(t, dir, "aliases.yaml", "a: b\n")
	if _, err := Load(dir); err == nil {
		t.Error("expected error for empty keep")
	}
}

func TestLoad_EmptyAliases(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - http://a.m3u\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - CCTV\n")
	writeFile(t, dir, "aliases.yaml", "")
	if _, err := Load(dir); err == nil {
		t.Error("expected error for empty aliases")
	}
}

func TestLoad_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sources.yaml", "sources:\n  - '  http://a.m3u  '\n  - ''\n  - '  '\n")
	writeFile(t, dir, "channels.yaml", "keep:\n  - '  CCTV  '\n  - ''\nexclude:\n  - ''\n  - ' 广告 '\n")
	writeFile(t, dir, "aliases.yaml", " ' 央视综合 ' : ' CCTV1 '\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Sources) != 1 || cfg.Sources[0] != "http://a.m3u" {
		t.Errorf("Sources = %v", cfg.Sources)
	}
	if len(cfg.Keep) != 1 || cfg.Keep[0] != "CCTV" {
		t.Errorf("Keep = %v", cfg.Keep)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "广告" {
		t.Errorf("Exclude = %v", cfg.Exclude)
	}
	if cfg.Aliases["央视综合"] != "CCTV1" {
		t.Errorf("Aliases = %v", cfg.Aliases)
	}
}
