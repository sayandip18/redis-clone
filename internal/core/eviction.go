package core

import (
	"time"

	"github.com/sayandip18/redis-clone/config"
)

// evictFirst deletes the first key found while iterating the map.
func evictFirst() bool {
	for k := range store {
		delete(store, k)
		return true
	}
	return false
}

// evictVolatileLRU samples up to EvictionPoolSampleSize keys that have a TTL
// and deletes the one with the oldest LRU clock (largest idle time).
// Returns false when no volatile candidate exists (persistent keys are never touched).
func evictVolatileLRU() bool {
	sampleSize := config.EvictionPoolSampleSize
	if sampleSize <= 0 {
		sampleSize = 5
	}

	now := time.Now().UnixMilli()
	var (
		victimKey string
		victimLRU uint32
		found     bool
		sampled   int
	)

	for key, obj := range store {
		if obj == nil || obj.ExpiresAt == -1 {
			continue
		}
		// Skip already-expired keys; prefer reclaiming via expire path, but
		// they are still volatile — delete them as an easy win if sampled.
		if obj.ExpiresAt <= now {
			delete(store, key)
			return true
		}

		if !found || obj.LRU < victimLRU {
			victimKey = key
			victimLRU = obj.LRU
			found = true
		}
		sampled++
		if sampled >= sampleSize {
			break
		}
	}

	if !found {
		return false
	}
	delete(store, victimKey)
	return true
}

// evict runs the configured eviction strategy. Returns whether a key was removed.
func evict() bool {
	switch config.EvictionStrategy {
	case "simple-first":
		return evictFirst()
	case "volatile-lru":
		return evictVolatileLRU()
	default:
		return false
	}
}
