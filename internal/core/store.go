package core

import (
	"errors"
	"strconv"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

var store map[string]*Obj

// keyVersions tracks a monotonic version per key for WATCH optimistic locking.
// Missing keys are treated as version 0.
var keyVersions map[string]uint64

// ErrWrongType is returned when a command is used against a key of the wrong type.
var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

// ErrOOM is returned when the key limit is reached and no key can be evicted.
var ErrOOM = errors.New("(error) OOM command not allowed when used memory > 'maxmemory'")

type ObjType uint8
type ObjEncoding uint8

const (
	ObjString ObjType = iota
	ObjList
	ObjSet
	// Future: ObjHash, ObjZSet
)

const (
	EncodingRaw ObjEncoding = iota // plain Go string
	EncodingInt                    // integer stored as int64
	EncodingEmbstr                 // short string (tag-only in Go)
	EncodingZiplist                // compact list encoding
	EncodingQuicklist              // list of ziplist nodes
	EncodingIntset                 // sorted integer set
	EncodingHT                     // hashtable set
)

// EmbstrMaxLen is Redis's default max length for embstr encoding.
const EmbstrMaxLen = 44

type Obj struct {
	Type      ObjType
	Encoding  ObjEncoding
	Value     interface{}
	ExpiresAt int64
	LRU       uint32 // last-access clock in seconds (approximate LRU)
}

// getLRUClock returns the current coarse LRU clock (unix seconds).
func getLRUClock() uint32 {
	return uint32(time.Now().Unix())
}

// IdleTime returns how many seconds the object has been idle.
func IdleTime(obj *Obj) int64 {
	if obj == nil {
		return 0
	}
	idle := int64(getLRUClock()) - int64(obj.LRU)
	if idle < 0 {
		return 0
	}
	return idle
}

func init() {
	store = make(map[string]*Obj)
	keyVersions = make(map[string]uint64)
}

// KeyVersion returns the current optimistic-lock version for k (0 if never written).
func KeyVersion(k string) uint64 {
	return keyVersions[k]
}

func bumpVersion(k string) {
	keyVersions[k]++
}

// deleteKey removes k from the store and bumps its version. Returns whether it existed.
func deleteKey(k string) bool {
	if _, ok := store[k]; !ok {
		return false
	}
	delete(store, k)
	bumpVersion(k)
	return true
}

// chooseStringEncoding picks int, embstr, or raw for a string value.
func chooseStringEncoding(value string) (ObjEncoding, interface{}) {
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return EncodingInt, n
	}
	if len(value) <= EmbstrMaxLen {
		return EncodingEmbstr, value
	}
	return EncodingRaw, value
}

// NewStringObj creates a string object, choosing int / embstr / raw encoding.
func NewStringObj(value string, durationMs int64) *Obj {
	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}

	enc, v := chooseStringEncoding(value)
	return &Obj{
		Type:      ObjString,
		Encoding:  enc,
		Value:     v,
		ExpiresAt: expiresAt,
		LRU:       getLRUClock(),
	}
}

// GetStringValue returns the string payload for any string encoding, or ErrWrongType.
func GetStringValue(obj *Obj) (string, error) {
	if obj.Type != ObjString {
		return "", ErrWrongType
	}

	switch obj.Encoding {
	case EncodingRaw, EncodingEmbstr:
		s, ok := obj.Value.(string)
		if !ok {
			return "", ErrWrongType
		}
		return s, nil
	case EncodingInt:
		n, ok := obj.Value.(int64)
		if !ok {
			return "", ErrWrongType
		}
		return strconv.FormatInt(n, 10), nil
	default:
		return "", ErrWrongType
	}
}

// EncodingName returns the Redis-style encoding name for OBJECT ENCODING.
func EncodingName(enc ObjEncoding) string {
	switch enc {
	case EncodingInt:
		return "int"
	case EncodingEmbstr:
		return "embstr"
	case EncodingRaw:
		return "raw"
	case EncodingZiplist:
		return "ziplist"
	case EncodingQuicklist:
		return "quicklist"
	case EncodingIntset:
		return "intset"
	case EncodingHT:
		return "hashtable"
	default:
		return "unknown"
	}
}

// getInt64Value parses a string object's value as int64 for INCR-like commands.
func getInt64Value(obj *Obj) (int64, error) {
	switch obj.Encoding {
	case EncodingInt:
		n, ok := obj.Value.(int64)
		if !ok {
			return 0, errors.New("(error) ERR value is not an integer or out of range")
		}
		return n, nil
	case EncodingRaw, EncodingEmbstr:
		s, ok := obj.Value.(string)
		if !ok {
			return 0, errors.New("(error) ERR value is not an integer or out of range")
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, errors.New("(error) ERR value is not an integer or out of range")
		}
		return n, nil
	default:
		return 0, errors.New("(error) ERR value is not an integer or out of range")
	}
}

// stringObjToRaw promotes int/embstr string objects to EncodingRaw (for future mutating cmds).
func stringObjToRaw(obj *Obj) error {
	if obj.Type != ObjString {
		return ErrWrongType
	}
	if obj.Encoding == EncodingRaw {
		return nil
	}
	s, err := GetStringValue(obj)
	if err != nil {
		return err
	}
	obj.Encoding = EncodingRaw
	obj.Value = s
	return nil
}

// Put stores obj under k. On overwrite or insert it refreshes the LRU clock.
// If the store is at KeysLimit and a new key cannot be evicted into, returns ErrOOM.
func Put(k string, obj *Obj) error {
	obj.LRU = getLRUClock()
	if _, exists := store[k]; !exists && len(store) >= config.KeysLimit {
		if !evict() {
			return ErrOOM
		}
	}
	store[k] = obj
	bumpVersion(k)
	return nil
}

// Get returns the object for k, deleting it if expired. Successful lookups touch LRU.
func Get(k string) *Obj {
	v := store[k]
	if v != nil {
		if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
			deleteKey(k)
			return nil
		}
		v.LRU = getLRUClock()
	}
	return v
}

// getNoTouch looks up k without updating the LRU clock (for OBJECT IDLETIME / ENCODING).
func getNoTouch(k string) *Obj {
	v := store[k]
	if v != nil {
		if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
			deleteKey(k)
			return nil
		}
	}
	return v
}

func Del(k string) bool {
	return deleteKey(k)
}

// KeyCount returns the number of keys currently in the store.
func KeyCount() int {
	return len(store)
}

// ExpiresCount returns how many keys have an expiry set.
func ExpiresCount() int {
	n := 0
	for _, obj := range store {
		if obj != nil && obj.ExpiresAt != -1 {
			n++
		}
	}
	return n
}
