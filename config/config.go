package config

var (
	Host                   = "0.0.0.0"
	Port                   = 7379
	KeysLimit              = 5
	EvictionStrategy       = "volatile-lru"
	EvictionPoolSampleSize = 5
	AOFFile                = "dump.aof"
)
