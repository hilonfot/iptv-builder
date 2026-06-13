package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// AppConfig holds general runtime settings from app.yaml.
type AppConfig struct {
	CacheTTLHours int    `mapstructure:"cache_ttl_hours"`
	Workers       int    `mapstructure:"workers"`
	LogLevel      string `mapstructure:"log_level"`
}

// QualityWeights defines per-dimension scoring weights from quality.yaml.
type QualityWeights struct {
	Resolution float64 `mapstructure:"resolution_weight"`
	Bitrate    float64 `mapstructure:"bitrate_weight"`
	Protocol   float64 `mapstructure:"protocol_weight"`
	Latency    float64 `mapstructure:"latency_weight"`
}

// Config aggregates all configuration data loaded from the config directory.
type Config struct {
	App     AppConfig
	Sources []string
	Keep    []string
	Exclude []string
	Aliases map[string]string
	Quality QualityWeights
}

// Load reads and validates all YAML config files from configDir.
func Load(configDir string) (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			CacheTTLHours: 24,
			Workers:       50,
			LogLevel:      "info",
		},
		Quality: QualityWeights{
			Resolution: 0.5,
			Bitrate:    0.2,
			Protocol:   0.1,
			Latency:    0.2,
		},
	}

	// ---- app.yaml (optional) ------------------------------------------------
	if err := loadApp(configDir, cfg); err != nil {
		return nil, err
	}

	// ---- sources.yaml (required) --------------------------------------------
	if err := loadSources(configDir, cfg); err != nil {
		return nil, err
	}

	// ---- channels.yaml (required) -------------------------------------------
	if err := loadChannels(configDir, cfg); err != nil {
		return nil, err
	}

	// ---- aliases.yaml (required) --------------------------------------------
	if err := loadAliases(configDir, cfg); err != nil {
		return nil, err
	}

	// ---- quality.yaml (optional) --------------------------------------------
	if err := loadQuality(configDir, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func newViper(configDir, name string) *viper.Viper {
	v := viper.New()
	v.SetConfigName(name)
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	return v
}

// loadApp reads app.yaml. It is optional — defaults are used when absent.
func loadApp(configDir string, cfg *Config) error {
	v := newViper(configDir, "app")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // optional file
		}
		return fmt.Errorf("read app.yaml: %w", err)
	}

	var app struct {
		App AppConfig `mapstructure:"app"`
	}
	if err := v.Unmarshal(&app); err != nil {
		return fmt.Errorf("unmarshal app.yaml: %w", err)
	}

	if app.App.CacheTTLHours > 0 {
		cfg.App.CacheTTLHours = app.App.CacheTTLHours
	}
	if app.App.Workers > 0 {
		cfg.App.Workers = app.App.Workers
	}
	if app.App.LogLevel != "" {
		cfg.App.LogLevel = app.App.LogLevel
	}
	return nil
}

// loadSources reads sources.yaml (required).
func loadSources(configDir string, cfg *Config) error {
	v := newViper(configDir, "sources")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read sources.yaml: %w", err)
	}

	var wrapper struct {
		Sources []string `mapstructure:"sources"`
	}
	if err := v.Unmarshal(&wrapper); err != nil {
		return fmt.Errorf("unmarshal sources.yaml: %w", err)
	}

	// Trim whitespace and filter empty lines.
	var clean []string
	for _, s := range wrapper.Sources {
		s = strings.TrimSpace(s)
		if s != "" {
			clean = append(clean, s)
		}
	}
	if len(clean) == 0 {
		return fmt.Errorf("sources.yaml: at least one source URL is required")
	}
	cfg.Sources = clean
	return nil
}

// loadChannels reads channels.yaml (required).
func loadChannels(configDir string, cfg *Config) error {
	v := newViper(configDir, "channels")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read channels.yaml: %w", err)
	}

	var wrapper struct {
		Keep    []string `mapstructure:"keep"`
		Exclude []string `mapstructure:"exclude"`
	}
	if err := v.Unmarshal(&wrapper); err != nil {
		return fmt.Errorf("unmarshal channels.yaml: %w", err)
	}

	keep := trimStrings(wrapper.Keep)
	if len(keep) == 0 {
		return fmt.Errorf("channels.yaml: at least one keep rule is required")
	}
	cfg.Keep = keep
	cfg.Exclude = trimStrings(wrapper.Exclude)
	return nil
}

// loadAliases reads aliases.yaml (required). It is a flat map with no top-level key.
func loadAliases(configDir string, cfg *Config) error {
	v := newViper(configDir, "aliases")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read aliases.yaml: %w", err)
	}

	var raw map[string]string
	if err := v.Unmarshal(&raw); err != nil {
		return fmt.Errorf("unmarshal aliases.yaml: %w", err)
	}

	clean := make(map[string]string, len(raw))
	for k, v := range raw {
		clean[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(clean) == 0 {
		return fmt.Errorf("aliases.yaml: at least one alias is required")
	}
	cfg.Aliases = clean
	return nil
}

// loadQuality reads quality.yaml. It is optional — defaults are used when absent.
func loadQuality(configDir string, cfg *Config) error {
	v := newViper(configDir, "quality")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // optional file
		}
		return fmt.Errorf("read quality.yaml: %w", err)
	}

	var wrapper struct {
		Quality QualityWeights `mapstructure:"quality"`
	}
	if err := v.Unmarshal(&wrapper); err != nil {
		return fmt.Errorf("unmarshal quality.yaml: %w", err)
	}

	q := wrapper.Quality
	if q.Resolution > 0 {
		cfg.Quality.Resolution = q.Resolution
	}
	if q.Bitrate > 0 {
		cfg.Quality.Bitrate = q.Bitrate
	}
	if q.Protocol > 0 {
		cfg.Quality.Protocol = q.Protocol
	}
	if q.Latency > 0 {
		cfg.Quality.Latency = q.Latency
	}

	return nil
}

// trimStrings removes empty/whitespace-only entries from a slice.
func trimStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Stats returns a human-readable summary for logging.
func (c *Config) Stats() string {
	return fmt.Sprintf(
		"sources=%d keep=%d exclude=%d aliases=%d workers=%d cache_ttl=%dh log_level=%s weights[res=%.2f bitrate=%.2f proto=%.2f lat=%.2f]",
		len(c.Sources), len(c.Keep), len(c.Exclude), len(c.Aliases),
		c.App.Workers, c.App.CacheTTLHours, c.App.LogLevel,
		c.Quality.Resolution, c.Quality.Bitrate, c.Quality.Protocol, c.Quality.Latency,
	)
}
