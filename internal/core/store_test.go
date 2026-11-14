package core

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNewStringObj_EncodingInt(t *testing.T) {
	obj := NewStringObj("123", -1)
	if obj.Type != ObjString {
		t.Fatalf("Type = %v, want ObjString", obj.Type)
	}
	if obj.Encoding != EncodingInt {
		t.Fatalf("Encoding = %v, want EncodingInt", obj.Encoding)
	}
	if obj.Value != int64(123) {
		t.Fatalf("Value = %v, want int64(123)", obj.Value)
	}
}

func TestNewStringObj_EncodingEmbstr(t *testing.T) {
	obj := NewStringObj("hello", -1)
	if obj.Encoding != EncodingEmbstr {
		t.Fatalf("Encoding = %v, want EncodingEmbstr", obj.Encoding)
	}
	if obj.Value != "hello" {
		t.Fatalf("Value = %v, want %q", obj.Value, "hello")
	}
}

func TestNewStringObj_EncodingRaw(t *testing.T) {
	value := strings.Repeat("a", EmbstrMaxLen+1)
	obj := NewStringObj(value, -1)
	if obj.Encoding != EncodingRaw {
		t.Fatalf("Encoding = %v, want EncodingRaw", obj.Encoding)
	}
	if obj.Value != value {
		t.Fatalf("Value length = %d, want %d", len(obj.Value.(string)), len(value))
	}
}

func TestNewStringObj_ParseIntAcceptsLeadingZeros(t *testing.T) {
	obj := NewStringObj("007", -1)
	if obj.Encoding != EncodingInt {
		t.Fatalf("Encoding = %v, want EncodingInt", obj.Encoding)
	}
	if obj.Value != int64(7) {
		t.Fatalf("Value = %v, want int64(7)", obj.Value)
	}
}

func TestGetStringValue_AllEncodings(t *testing.T) {
	cases := []struct {
		name string
		obj  *Obj
		want string
	}{
		{"int", NewStringObj("42", -1), "42"},
		{"embstr", NewStringObj("hi", -1), "hi"},
		{"raw", NewStringObj(strings.Repeat("x", EmbstrMaxLen+1), -1), strings.Repeat("x", EmbstrMaxLen+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetStringValue(tc.obj)
			if err != nil {
				t.Fatalf("GetStringValue() error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("GetStringValue() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetStringValue_WrongType(t *testing.T) {
	obj := &Obj{
		Type:     ObjType(99),
		Encoding: EncodingRaw,
		Value:    "x",
	}
	_, err := GetStringValue(obj)
	if !errors.Is(err, ErrWrongType) {
		t.Fatalf("GetStringValue() error = %v, want ErrWrongType", err)
	}
}

func TestStringObjToRaw(t *testing.T) {
	obj := NewStringObj("99", -1)
	if obj.Encoding != EncodingInt {
		t.Fatalf("setup: Encoding = %v, want EncodingInt", obj.Encoding)
	}
	if err := stringObjToRaw(obj); err != nil {
		t.Fatalf("stringObjToRaw() error: %v", err)
	}
	if obj.Encoding != EncodingRaw {
		t.Fatalf("Encoding = %v, want EncodingRaw", obj.Encoding)
	}
	if obj.Value != "99" {
		t.Fatalf("Value = %v, want %q", obj.Value, "99")
	}
}

func TestEvalGET_WrongType(t *testing.T) {
	resetStore()
	Put("bad", &Obj{
		Type:      ObjType(99),
		Encoding:  EncodingRaw,
		Value:     "x",
		ExpiresAt: -1,
	})

	var buf bytes.Buffer
	err := evalGET([]string{"bad"}, &buf)
	if !errors.Is(err, ErrWrongType) {
		t.Fatalf("evalGET() error = %v, want ErrWrongType", err)
	}
}

func TestEvalSET_GET_IntEncoding(t *testing.T) {
	resetStore()

	var buf bytes.Buffer
	if err := evalSET([]string{"k", "123"}, &buf); err != nil {
		t.Fatalf("evalSET() error: %v", err)
	}

	obj := Get("k")
	if obj == nil || obj.Encoding != EncodingInt {
		t.Fatalf("obj encoding = %v, want EncodingInt", obj.Encoding)
	}

	var out bytes.Buffer
	if err := evalGET([]string{"k"}, &out); err != nil {
		t.Fatalf("evalGET() error: %v", err)
	}
	if out.String() != "$3\r\n123\r\n" {
		t.Fatalf("GET response = %q, want bulk string 123", out.String())
	}
}

func TestEvalOBJECT_ENCODING(t *testing.T) {
	resetStore()

	Put("i", NewStringObj("10", -1))
	Put("e", NewStringObj("hi", -1))
	Put("r", NewStringObj(strings.Repeat("z", EmbstrMaxLen+1), -1))

	cases := []struct {
		key  string
		want string
	}{
		{"i", "$3\r\nint\r\n"},
		{"e", "$6\r\nembstr\r\n"},
		{"r", "$3\r\nraw\r\n"},
		{"missing", "$-1\r\n"},
	}

	for _, tc := range cases {
		var out bytes.Buffer
		if err := evalOBJECT([]string{"ENCODING", tc.key}, &out); err != nil {
			t.Fatalf("OBJECT ENCODING %s error: %v", tc.key, err)
		}
		if out.String() != tc.want {
			t.Fatalf("OBJECT ENCODING %s = %q, want %q", tc.key, out.String(), tc.want)
		}
	}
}
