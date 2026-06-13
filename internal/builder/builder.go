package builder

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/hilonfot/iptv-builder/internal/analyzer"
	"github.com/hilonfot/iptv-builder/internal/store"
	"github.com/hilonfot/iptv-builder/internal/config"
	"github.com/hilonfot/iptv-builder/internal/dedupe"
	"github.com/hilonfot/iptv-builder/internal/fetch"
	"github.com/hilonfot/iptv-builder/internal/filter"
	"github.com/hilonfot/iptv-builder/internal/generator"
	"github.com/hilonfot/iptv-builder/internal/normalizer"
	"github.com/hilonfot/iptv-builder/internal/parser"
	"github.com/hilonfot/iptv-builder/internal/scorer"
	"github.com/hilonfot/iptv-builder/internal/selector"
	"github.com/hilonfot/iptv-builder/internal/speedtest"
	"github.com/hilonfot/iptv-builder/internal/model"
)

// Builder orchestrates the full IPTV pipeline.
type Builder struct {
	cfg       *config.Config
	configDir string
	outputDir string
	cacheDir  string
}

// New creates a new Builder.
func New(cfg *config.Config, configDir, outputDir, cacheDir string) *Builder {
	return &Builder{
		cfg:       cfg,
		configDir: configDir,
		outputDir: outputDir,
		cacheDir:  cacheDir,
	}
}

// Run executes the full pipeline and returns an error only for fatal failures.
func (b *Builder) Run(ctx context.Context) error {
	start := time.Now()
	slog.Info("=== IPTV Builder started ===", "config", b.cfg.Stats())

	// ---- Step 1: Fetch sources ----
	slog.Info("--- step 1/10: fetch sources ---")
	fetcher := fetch.New(10 * time.Second)
	rawSources := fetcher.Fetch(ctx, b.cfg.Sources)
	if len(rawSources) == 0 {
		slog.Error("all sources failed, nothing to process")
		os.Exit(1)
	}

	// ---- Step 2: Parse M3U ----
	slog.Info("--- step 2/10: parse m3u ---")
	prs := parser.New()
	var allChannels []*model.Channel
	for _, rs := range rawSources {
		chans := prs.Parse(rs.Content, rs.URL)
		allChannels = append(allChannels, chans...)
	}
	slog.Info("total parsed channels", "count", len(allChannels))

	// ---- Step 3: Normalize ----
	slog.Info("--- step 3/10: normalize ---")
	normalizer.Normalize(allChannels, b.cfg.Aliases)

	// ---- Step 4: Filter ----
	slog.Info("--- step 4/10: filter ---")
	allChannels = filter.Filter(allChannels, b.cfg.Exclude, b.cfg.Keep)
	if len(allChannels) == 0 {
		slog.Warn("all channels filtered out, nothing to output")
		return nil
	}

	// ---- Step 5: Dedupe ----
	slog.Info("--- step 5/10: dedupe ---")
	groups := dedupe.Dedupe(allChannels)

	// ---- Step 6: Analyze quality ----
	slog.Info("--- step 6/10: analyze quality ---")
	azr := analyzer.New()
	azr.Analyze(ctx, groups)

	// ---- Step 7: Speed test ----
	slog.Info("--- step 7/10: speed test ---")
	cachePath := b.cacheDir + "/quality_cache.json"
	cacheStore := store.New(cachePath, time.Duration(b.cfg.App.CacheTTLHours)*time.Hour)
	_ = cacheStore.Load() // best-effort

	tester := speedtest.New(b.cfg.App.Workers)
	tester.Test(ctx, groups, cacheStore)

	// Write back speed test results to cache.
	for canonical, chs := range groups {
		for _, ch := range chs {
			if ch != nil && ch.LatencyMs > 0 && ch.Valid {
				cacheStore.Set(canonical, model.CacheEntry{
					URL:          ch.URL,
					Resolution:   ch.Resolution,
					Bitrate:      ch.Bitrate,
					Protocol:     ch.Protocol,
					LatencyMs:    ch.LatencyMs,
					QualityScore: ch.QualityScore,
				})
			}
		}
	}

	// ---- Step 8: Calculate score ----
	slog.Info("--- step 8/10: calculate score ---")
	scorer.Score(groups, b.cfg.Quality)

	// ---- Step 9: Select best ----
	slog.Info("--- step 9/10: select best ---")
	best := selector.SelectBest(groups)
	if len(best) == 0 {
		slog.Warn("no channels after selection, nothing to output")
		_ = cacheStore.Save()
		return nil
	}

	// ---- Step 10: Generate M3U ----
	slog.Info("--- step 10/10: generate m3u ---")
	if err := generator.Generate(best, b.outputDir); err != nil {
		return err
	}

	// ---- Save cache ----
	_ = cacheStore.Save()

	slog.Info("=== IPTV Builder finished ===",
		"total_channels", len(best),
		"elapsed", time.Since(start).Round(time.Second),
	)
	return nil
}
