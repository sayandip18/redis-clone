package core

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

func withEvictionConfig(t *testing.T, strategy string, keysLimit, sampleSize int) {
	t.Helper()
	prevStrategy := config.EvictionStrategy
	prevLimit := config.KeysLimit
	prevSample := config.EvictionPoolSampleSize
	config.EvictionStrategy = strategy
	config.KeysLimit = keysLimit
	config.EvictionPoolSampleSize = sampleSize
	t.Cleanup(func() {
		config.EvictionStrategy = prevStrategy
		config.KeysLimit = prevLimit
		config.EvictionPoolSampleSize = prevSample
	})
}

func TestEvictVolatileLRU_ColdKeyWins(t *testing.T) {
	resetStore()
	withEvictionConfig(t, "volatile-lru", 5, 5)

	now := time.Now().UnixMilli()
	ttl := now + 60_000
	clock := getLRUClock()

	keys := []string{"a", "b", "c", "d", "cold"}
	for _, k := range keys {
		if err := Put(k, NewStringObj(k, 60_000)); err != nil {
			t.Fatalf("Put(%s): %v", k, err)
		}
		store[k].ExpiresAt = ttl
		store[k].LRU = clock
	}
	// Make "cold" clearly the oldest idle key.
	store["cold"].LRU = clock - 100

	if err := Put("new", NewStringObj("new", 60_000)); err != nil {
		t.Fatalf("Put(new): %v", err)
	}

	if _, ok := store["cold"]; ok {
		t.Fatal("expected cold key to be evicted")
	}
	if _, ok := store["new"]; !ok {
		t.Fatal("expected new key to be stored")
	}
	if len(store) != 5 {
		t.Fatalf("store size = %d, want 5", len(store))
	}
}

func TestEvictVolatileLRU_PersistentKeysSafe(t *testing.T) {
	resetStore()
	withEvictionConfig(t, "volatile-lru", 3, 5)

	for i := 0; i < 3; i++ {
		k := fmt.Sprintf("p%d", i)
		if err := Put(k, NewStringObj(k, -1)); err != nil {
			t.Fatalf("Put(%s): %v", k, err)
		}
	}

	err := Put("extra", NewStringObj("x", 60_000))
	if !errors.Is(err, ErrOOM) {
		t.Fatalf("Put(extra) error = %v, want ErrOOM", err)
	}
	if len(store) != 3 {
		t.Fatalf("store size = %d, want 3", len(store))
	}
	if _, ok := store["extra"]; ok {
		t.Fatal("extra key must not be inserted on OOM")
	}
	for i := 0; i < 3; i++ {
		k := fmt.Sprintf("p%d", i)
		if _, ok := store[k]; !ok {
			t.Fatalf("persistent key %s was evicted", k)
		}
	}
}

func TestEvictSimpleFirst_StillWorks(t *testing.T) {
	resetStore()
	withEvictionConfig(t, "simple-first", 2, 5)

	if err := Put("a", NewStringObj("1", -1)); err != nil {
		t.Fatalf("Put(a): %v", err)
	}
	if err := Put("b", NewStringObj("2", -1)); err != nil {
		t.Fatalf("Put(b): %v", err)
	}
	if err := Put("c", NewStringObj("3", -1)); err != nil {
		t.Fatalf("Put(c): %v", err)
	}
	if len(store) != 2 {
		t.Fatalf("store size = %d, want 2 after simple-first eviction", len(store))
	}
	if _, ok := store["c"]; !ok {
		t.Fatal("expected newest key c to be present")
	}
}

func TestObjectIdleTime(t *testing.T) {
	resetStore()
	withEvictionConfig(t, "volatile-lru", 100, 5)

	if err := Put("k", NewStringObj("v", -1)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	store["k"].LRU = getLRUClock() - 7

	var out bytes.Buffer
	if err := evalOBJECT([]string{"IDLETIME", "k"}, &out); err != nil {
		t.Fatalf("OBJECT IDLETIME error: %v", err)
	}
	if out.String() != ":7\r\n" {
		t.Fatalf("IDLETIME = %q, want :7\\r\\n", out.String())
	}

	// Get touches LRU; idle time should reset near 0.
	if Get("k") == nil {
		t.Fatal("Get(k) returned nil")
	}
	out.Reset()
	if err := evalOBJECT([]string{"IDLETIME", "k"}, &out); err != nil {
		t.Fatalf("OBJECT IDLETIME after Get error: %v", err)
	}
	if out.String() != ":0\r\n" {
		t.Fatalf("IDLETIME after Get = %q, want :0\\r\\n", out.String())
	}

	// Missing key → nil bulk
	out.Reset()
	if err := evalOBJECT([]string{"IDLETIME", "missing"}, &out); err != nil {
		t.Fatalf("OBJECT IDLETIME missing error: %v", err)
	}
	if out.String() != "$-1\r\n" {
		t.Fatalf("IDLETIME missing = %q, want $-1\\r\\n", out.String())
	}
}

func TestEvalSET_OOM(t *testing.T) {
	resetStore()
	withEvictionConfig(t, "volatile-lru", 2, 5)

	if err := Put("a", NewStringObj("1", -1)); err != nil {
		t.Fatalf("Put(a): %v", err)
	}
	if err := Put("b", NewStringObj("2", -1)); err != nil {
		t.Fatalf("Put(b): %v", err)
	}

	var out bytes.Buffer
	err := evalSET([]string{"c", "3"}, &out)
	if !errors.Is(err, ErrOOM) {
		t.Fatalf("evalSET error = %v, want ErrOOM", err)
	}
}
