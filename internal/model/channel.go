package model

// Channel represents a single IPTV channel line parsed from an M3U source.
type Channel struct {
	Name      string // Original name from M3U EXTINF line
	URL       string // Stream URL
	Group     string // group-title from M3U

	Canonical string // Normalized name after alias resolution

	Resolution string // Resolution label: "4K"|"1080P"|"720P"|"SD"|""
	Bitrate    int64  // Bitrate in bps, 0 means unknown
	Protocol   string // Transport protocol: "m3u8"|"flv"|"ts"|""

	LatencyMs int64 // Speed test latency in milliseconds; 0 = untested or failed

	QualityScore float64 // Composite quality score (0-100)

	Source string // Origin IPTV source URL
	Valid  bool   // Whether the channel passed all pipeline stages
}
