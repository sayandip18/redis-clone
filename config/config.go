package config

var (
	Host                   = "0.0.0.0"
	Port                   = 7379
	KeysLimit              = 5
	EvictionStrategy       = "volatile-lru"
	EvictionPoolSampleSize = 5
	AOFFile                = "dump.aof"

	// List encoding conversion thresholds (concept-level Redis knobs).
	ListMaxZiplistEntries = 512
	ListMaxZiplistValue   = 64

	// Set encoding conversion threshold.
	SetMaxIntsetEntries = 512
)
