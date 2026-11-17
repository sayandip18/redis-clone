package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sayandip18/redis-clone/config"
)

func TestListPushPopRange(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	n, err := listPush("mylist", listRight, []string{"a", "b", "c"})
	if err != nil || n != 3 {
		t.Fatalf("RPUSH n=%d err=%v", n, err)
	}
	n, err = listPush("mylist", listLeft, []string{"z"})
	if err != nil || n != 4 {
		t.Fatalf("LPUSH n=%d err=%v", n, err)
	}
	vals, err := listRange("mylist", 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"z", "a", "b", "c"}
	if strings.Join(vals, ",") != strings.Join(want, ",") {
		t.Fatalf("LRANGE = %v, want %v", vals, want)
	}
	v, ok, err := listPop("mylist", listLeft)
	if err != nil || !ok || v != "z" {
		t.Fatalf("LPOP = %q %v %v", v, ok, err)
	}
}

func TestListConvertToQuicklist(t *testing.T) {
	resetStore()
	prevEntries := config.ListMaxZiplistEntries
	prevKeys := config.KeysLimit
	config.ListMaxZiplistEntries = 3
	config.KeysLimit = 100
	t.Cleanup(func() {
		config.ListMaxZiplistEntries = prevEntries
		config.KeysLimit = prevKeys
	})

	if _, err := listPush("q", listRight, []string{"a", "b", "c", "d"}); err != nil {
		t.Fatal(err)
	}
	obj := Get("q")
	if obj.Encoding != EncodingQuicklist {
		t.Fatalf("encoding = %v (%s), want quicklist", obj.Encoding, EncodingName(obj.Encoding))
	}
	vals, _ := listRange("q", 0, -1)
	if strings.Join(vals, ",") != "a,b,c,d" {
		t.Fatalf("values after convert = %v", vals)
	}
}

func TestListConvertOnLargeValue(t *testing.T) {
	resetStore()
	prevVal := config.ListMaxZiplistValue
	prevKeys := config.KeysLimit
	config.ListMaxZiplistValue = 4
	config.KeysLimit = 100
	t.Cleanup(func() {
		config.ListMaxZiplistValue = prevVal
		config.KeysLimit = prevKeys
	})

	if _, err := listPush("q", listRight, []string{"tiny", "toolong"}); err != nil {
		t.Fatal(err)
	}
	obj := Get("q")
	if obj.Encoding != EncodingQuicklist {
		t.Fatalf("encoding = %s, want quicklist", EncodingName(obj.Encoding))
	}
}

func TestEvalListCommands(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	var out bytes.Buffer
	if err := evalRPUSH([]string{"l", "a", "b"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalLLEN([]string{"l"}, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != ":2\r\n" {
		t.Fatalf("LLEN = %q", out.String())
	}
	out.Reset()
	if err := evalLSET([]string{"l", "0", "A"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalLINDEX([]string{"l", "0"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "A") {
		t.Fatalf("LINDEX = %q", out.String())
	}
	out.Reset()
	if err := evalLINSERT([]string{"l", "BEFORE", "b", "x"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalLREM([]string{"l", "1", "A"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalLTRIM([]string{"l", "0", "0"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := evalLRANGE([]string{"l", "0", "-1"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "x") {
		t.Fatalf("LRANGE after trim = %q", out.String())
	}
}

func TestListWrongType(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	_ = Put("s", NewStringObj("v", -1))
	_, err := listPush("s", listLeft, []string{"x"})
	if err != ErrWrongType {
		t.Fatalf("err = %v, want ErrWrongType", err)
	}
}
