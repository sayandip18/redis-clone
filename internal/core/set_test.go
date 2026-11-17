package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sayandip18/redis-clone/config"
)

func TestSetIntsetAndConvert(t *testing.T) {
	resetStore()
	prevKeys := config.KeysLimit
	prevMax := config.SetMaxIntsetEntries
	config.KeysLimit = 100
	config.SetMaxIntsetEntries = 3
	t.Cleanup(func() {
		config.KeysLimit = prevKeys
		config.SetMaxIntsetEntries = prevMax
	})

	n, err := setAdd("s", []string{"1", "2"})
	if err != nil || n != 2 {
		t.Fatalf("SADD n=%d err=%v", n, err)
	}
	obj := Get("s")
	if obj.Encoding != EncodingIntset {
		t.Fatalf("encoding = %s, want intset", EncodingName(obj.Encoding))
	}

	n, err = setAdd("s", []string{"hello"})
	if err != nil || n != 1 {
		t.Fatalf("SADD non-int n=%d err=%v", n, err)
	}
	obj = Get("s")
	if obj.Encoding != EncodingHT {
		t.Fatalf("encoding = %s, want hashtable", EncodingName(obj.Encoding))
	}
}

func TestSetConvertOnCardinality(t *testing.T) {
	resetStore()
	prevKeys := config.KeysLimit
	prevMax := config.SetMaxIntsetEntries
	config.KeysLimit = 100
	config.SetMaxIntsetEntries = 2
	t.Cleanup(func() {
		config.KeysLimit = prevKeys
		config.SetMaxIntsetEntries = prevMax
	})

	_, _ = setAdd("s", []string{"1", "2", "3"})
	obj := Get("s")
	if obj.Encoding != EncodingHT {
		t.Fatalf("encoding = %s, want hashtable", EncodingName(obj.Encoding))
	}
}

func TestSetAlgebra(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	_, _ = setAdd("a", []string{"1", "2", "3"})
	_, _ = setAdd("b", []string{"2", "3", "4"})

	inter, err := setOp(setOpInter, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inter) != 2 {
		t.Fatalf("inter = %v, want 2 elems", inter)
	}
	union, _ := setOp(setOpUnion, []string{"a", "b"})
	if len(union) != 4 {
		t.Fatalf("union = %v, want 4 elems", union)
	}
	diff, _ := setOp(setOpDiff, []string{"a", "b"})
	if len(diff) != 1 || diff[0] != "1" {
		t.Fatalf("diff = %v, want [1]", diff)
	}

	n, err := setOpStore(setOpInter, "c", []string{"a", "b"})
	if err != nil || n != 2 {
		t.Fatalf("SINTERSTORE n=%d err=%v", n, err)
	}
}

func TestEvalSetCommands(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	var out bytes.Buffer
	if err := evalSADD([]string{"s", "a", "b", "a"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != ":2\r\n" {
		t.Fatalf("SADD = %q", out.String())
	}
	out.Reset()
	if err := evalSISMEMBER([]string{"s", "a"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != ":1\r\n" {
		t.Fatalf("SISMEMBER = %q", out.String())
	}
	out.Reset()
	if err := evalSCARD([]string{"s"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != ":2\r\n" {
		t.Fatalf("SCARD = %q", out.String())
	}
	out.Reset()
	if err := evalSMEMBERS([]string{"s"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "a") || !strings.Contains(out.String(), "b") {
		t.Fatalf("SMEMBERS = %q", out.String())
	}
	out.Reset()
	if err := evalSREM([]string{"s", "a"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalSPOP([]string{"s"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() == string(RESP_NIL) {
		t.Fatal("SPOP unexpectedly nil")
	}
}

func TestSetWrongType(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })
	_ = Put("s", NewStringObj("v", -1))
	_, err := setAdd("s", []string{"x"})
	if err != ErrWrongType {
		t.Fatalf("err = %v, want ErrWrongType", err)
	}
}
