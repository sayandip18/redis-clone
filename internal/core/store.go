package core

import (
	"errors"
	"strconv"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

var store map[string]*Obj

// ErrWrongType is returned when a command is used against a key of the wrong type.
var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

type ObjType uint8
type ObjEncoding uint8

const (
	ObjString ObjType = iota
	// Future: ObjList, ObjHash, ObjSet, ObjZSet
)

const (
	EncodingRaw ObjEncoding = iota // plain Go string
	EncodingInt                    // integer stored as int64
	EncodingEmbstr                 // short string (tag-only in Go)
)

// EmbstrMaxLen is Redis's default max length for embstr encoding.
const EmbstrMaxLen = 44

type Obj struct {
	Type      ObjType
	Encoding  ObjEncoding
	Value     interface{}
	ExpiresAt int64
}

func init() {
	store = make(map[string]*Obj)
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
	default:
		return "unknown"
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

func Put(k string, obj *Obj) {
	if _, exists := store[k]; !exists && len(store) >= config.KeysLimit {
		evict()
	}
	store[k] = obj
}

func Get(k string) *Obj {
	v := store[k]
	if v != nil {
		if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
			delete(store, k)
			return nil
		}
	}
	return v
}

func Del(k string) bool {
	if _, ok := store[k]; ok {
		delete(store, k)
		return true
	}
	return false
}
