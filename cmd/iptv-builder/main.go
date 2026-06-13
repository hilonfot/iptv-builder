package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hilonfot/iptv-builder/internal/builder"
	"github.com/hilonfot/iptv-builder/internal/config"
)

var (
	configDir string
	outputDir string
	cacheDir  string
)

func main() {
	root := &cobra.Command{
		Use:   "iptv-builder",
		Short: "IPTV auto-builder — fetch, analyze, speed-test, and generate final.m3u",
		Long: `IPTV Builder fetches M3U playlists from configured IPTV sources,
parses channels, normalizes names, filters target channels,
analyzes stream quality (resolution/bitrate/protocol),
speed-tests streams, computes composite scores,
selects the best line per channel, and outputs final.m3u
for direct subscription by IPTV players.`,
		RunE: run,
	}

	root.Flags().StringVar(&configDir, "config-dir", "/config", "directory containing YAML config files")
	root.Flags().StringVar(&outputDir, "output-dir", "/output", "directory for final.m3u output")
	root.Flags().StringVar(&cacheDir, "cache-dir", "/cache", "directory for speed test cache")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Setup structured JSON logging.
	logLevel := slog.LevelInfo
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		switch lvl {
		case "debug":
			logLevel = slog.LevelDebug
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Load configuration.
	cfg, err := config.Load(configDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Override with app.yaml values if set.
	if cfg.App.LogLevel != "" {
		switch cfg.App.LogLevel {
		case "debug":
			logLevel = slog.LevelDebug
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
		slog.SetDefault(logger)
	}

	slog.Info("iptv-builder starting",
		"config_dir", configDir,
		"output_dir", outputDir,
		"cache_dir", cacheDir,
	)

	// Context that cancels on SIGINT/SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Run pipeline.
	b := builder.New(cfg, configDir, outputDir, cacheDir)
	if err := b.Run(ctx); err != nil {
		slog.Error("pipeline failed", "error", err)
		return err
	}

	return nil
}
