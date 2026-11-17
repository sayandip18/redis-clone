package core

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestEvalAPPEND_CreateAndAppend(t *testing.T) {
	resetStore()
	var out bytes.Buffer

	if err := evalAPPEND([]string{"k", "Hello"}, &out); err != nil {
		t.Fatalf("APPEND create error: %v", err)
	}
	if out.String() != ":5\r\n" {
		t.Fatalf("APPEND create = %q, want :5", out.String())
	}

	out.Reset()
	if err := evalAPPEND([]string{"k", " World"}, &out); err != nil {
		t.Fatalf("APPEND append error: %v", err)
	}
	if out.String() != ":11\r\n" {
		t.Fatalf("APPEND append = %q, want :11", out.String())
	}

	obj := Get("k")
	if obj == nil || obj.Encoding != EncodingRaw {
		t.Fatalf("encoding = %v, want EncodingRaw", obj.Encoding)
	}
	s, err := GetStringValue(obj)
	if err != nil || s != "Hello World" {
		t.Fatalf("value = %q, want Hello World", s)
	}
}

func TestEvalAPPEND_PromoteInt(t *testing.T) {
	resetStore()
	_ = Put("n", NewStringObj("10", -1))
	var out bytes.Buffer
	if err := evalAPPEND([]string{"n", "0"}, &out); err != nil {
		t.Fatalf("APPEND error: %v", err)
	}
	if out.String() != ":3\r\n" {
		t.Fatalf("APPEND = %q, want :3", out.String())
	}
	obj := Get("n")
	if obj.Encoding != EncodingRaw {
		t.Fatalf("encoding = %s, want raw", EncodingName(obj.Encoding))
	}
	s, _ := GetStringValue(obj)
	if s != "100" {
		t.Fatalf("value = %q, want 100", s)
	}
}

func TestEvalAPPEND_WrongType(t *testing.T) {
	resetStore()
	_, _ = listPush("l", listRight, []string{"a"})
	var out bytes.Buffer
	err := evalAPPEND([]string{"l", "x"}, &out)
	if !errors.Is(err, ErrWrongType) {
		t.Fatalf("error = %v, want ErrWrongType", err)
	}
}

func TestEvalSTRLEN(t *testing.T) {
	resetStore()
	var out bytes.Buffer

	if err := evalSTRLEN([]string{"missing"}, &out); err != nil {
		t.Fatalf("STRLEN missing error: %v", err)
	}
	if out.String() != ":0\r\n" {
		t.Fatalf("STRLEN missing = %q, want :0", out.String())
	}

	_ = Put("e", NewStringObj("hi", -1))
	out.Reset()
	if err := evalSTRLEN([]string{"e"}, &out); err != nil {
		t.Fatalf("STRLEN embstr error: %v", err)
	}
	if out.String() != ":2\r\n" {
		t.Fatalf("STRLEN embstr = %q, want :2", out.String())
	}

	raw := strings.Repeat("z", EmbstrMaxLen+1)
	_ = Put("r", NewStringObj(raw, -1))
	out.Reset()
	if err := evalSTRLEN([]string{"r"}, &out); err != nil {
		t.Fatalf("STRLEN raw error: %v", err)
	}
	if out.String() != ":45\r\n" {
		t.Fatalf("STRLEN raw = %q, want :45", out.String())
	}
}

func TestEvalGETRANGE(t *testing.T) {
	resetStore()
	_ = Put("k", NewStringObj("Hello World", -1))

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"k", "0", "4"}, "$5\r\nHello\r\n"},
		{[]string{"k", "-5", "-1"}, "$5\r\nWorld\r\n"},
		{[]string{"missing", "0", "1"}, "$0\r\n\r\n"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		if err := evalGETRANGE(tc.args, &out); err != nil {
			t.Fatalf("GETRANGE %v error: %v", tc.args, err)
		}
		if out.String() != tc.want {
			t.Fatalf("GETRANGE %v = %q, want %q", tc.args, out.String(), tc.want)
		}
	}
}

func TestEvalSETRANGE(t *testing.T) {
	resetStore()
	var out bytes.Buffer

	if err := evalSETRANGE([]string{"k", "0", "Hello World"}, &out); err != nil {
		t.Fatalf("SETRANGE create error: %v", err)
	}
	if out.String() != ":11\r\n" {
		t.Fatalf("SETRANGE create = %q, want :11", out.String())
	}

	out.Reset()
	if err := evalSETRANGE([]string{"k", "6", "Redis"}, &out); err != nil {
		t.Fatalf("SETRANGE overwrite error: %v", err)
	}
	if out.String() != ":11\r\n" {
		t.Fatalf("SETRANGE overwrite = %q, want :11", out.String())
	}
	s, _ := GetStringValue(Get("k"))
	if s != "Hello Redis" {
		t.Fatalf("value = %q, want Hello Redis", s)
	}

	out.Reset()
	if err := evalSETRANGE([]string{"pad", "2", "x"}, &out); err != nil {
		t.Fatalf("SETRANGE pad error: %v", err)
	}
	if out.String() != ":3\r\n" {
		t.Fatalf("SETRANGE pad = %q, want :3", out.String())
	}
	s, _ = GetStringValue(Get("pad"))
	if s != "\x00\x00x" {
		t.Fatalf("pad value = %q, want \\0\\0x", s)
	}
}

func TestEvalSETRANGE_NegativeOffset(t *testing.T) {
	resetStore()
	var out bytes.Buffer
	err := evalSETRANGE([]string{"k", "-1", "x"}, &out)
	if err == nil {
		t.Fatal("expected negative offset error")
	}
}

func TestEvalSETRANGE_WrongType(t *testing.T) {
	resetStore()
	_, _ = setAdd("s", []string{"a"})
	var out bytes.Buffer
	err := evalSETRANGE([]string{"s", "0", "x"}, &out)
	if !errors.Is(err, ErrWrongType) {
		t.Fatalf("error = %v, want ErrWrongType", err)
	}
}
