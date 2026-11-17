package core

import (
	"strings"
	"testing"
)

func TestNewSDS(t *testing.T) {
	s := NewSDS("hello")
	if s.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", s.Len())
	}
	if s.alloc != 5 {
		t.Fatalf("alloc = %d, want 5", s.alloc)
	}
	if s.Avail() != 0 {
		t.Fatalf("Avail() = %d, want 0", s.Avail())
	}
	if s.String() != "hello" {
		t.Fatalf("String() = %q, want hello", s.String())
	}
}

func TestSDS_Cat_GrowthUnder1MiB(t *testing.T) {
	s := NewSDS("ab")
	s.Cat("cd")
	if s.String() != "abcd" {
		t.Fatalf("String() = %q, want abcd", s.String())
	}
	// After grow from len=2 needing +2 → newlen=4 → alloc = 4*2 = 8
	if s.alloc != 8 {
		t.Fatalf("alloc = %d, want 8", s.alloc)
	}
	if s.Avail() != 4 {
		t.Fatalf("Avail() = %d, want 4", s.Avail())
	}

	// Second append that fits in avail should not grow alloc.
	prev := s.alloc
	s.Cat("ef")
	if s.alloc != prev {
		t.Fatalf("alloc changed to %d, want unchanged %d", s.alloc, prev)
	}
	if s.String() != "abcdef" {
		t.Fatalf("String() = %q, want abcdef", s.String())
	}
}

func TestSDS_Cat_GrowthOver1MiB(t *testing.T) {
	base := strings.Repeat("x", sdsMaxPrealloc-10)
	s := NewSDS(base)
	s.Cat(strings.Repeat("y", 20))
	newlen := (sdsMaxPrealloc - 10) + 20
	wantAlloc := newlen + sdsMaxPrealloc
	if s.Len() != newlen {
		t.Fatalf("Len() = %d, want %d", s.Len(), newlen)
	}
	if s.alloc != wantAlloc {
		t.Fatalf("alloc = %d, want %d", s.alloc, wantAlloc)
	}
}

func TestSDS_Clear(t *testing.T) {
	s := NewSDS("hello")
	s.Cat("world") // grow so alloc > 0 unused after clear
	allocBefore := s.alloc
	s.Clear()
	if s.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", s.Len())
	}
	if s.alloc != allocBefore {
		t.Fatalf("alloc = %d, want %d (preserved)", s.alloc, allocBefore)
	}
	if s.String() != "" {
		t.Fatalf("String() = %q, want empty", s.String())
	}
}

func TestSDS_Range(t *testing.T) {
	s := NewSDS("Hello World")
	cases := []struct {
		start, end int
		want       string
	}{
		{0, 4, "Hello"},
		{6, 10, "World"},
		{-5, -1, "World"},
		{0, -1, "Hello World"},
		{100, 200, ""},
		{4, 2, ""},
	}
	for _, tc := range cases {
		got := s.Range(tc.start, tc.end)
		if got != tc.want {
			t.Fatalf("Range(%d,%d) = %q, want %q", tc.start, tc.end, got, tc.want)
		}
	}
}

func TestSDS_SetRange_OverwriteAndGrow(t *testing.T) {
	s := NewSDS("Hello World")
	n := s.SetRange(6, "Redis")
	if n != 11 {
		t.Fatalf("SetRange len = %d, want 11", n)
	}
	if s.String() != "Hello Redis" {
		t.Fatalf("String() = %q, want Hello Redis", s.String())
	}

	n = s.SetRange(6, "Redis Clone!")
	if n != 18 {
		t.Fatalf("SetRange grow len = %d, want 18", n)
	}
	if s.String() != "Hello Redis Clone!" {
		t.Fatalf("String() = %q, want Hello Redis Clone!", s.String())
	}
}

func TestSDS_SetRange_ZeroFill(t *testing.T) {
	s := NewSDS("abc")
	n := s.SetRange(5, "xy")
	if n != 7 {
		t.Fatalf("len = %d, want 7", n)
	}
	got := s.String()
	want := "abc\x00\x00xy"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
