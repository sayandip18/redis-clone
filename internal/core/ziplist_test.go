package core

import "testing"

func TestZipListPushPop(t *testing.T) {
	z := newZipList()
	z.PushRight("a")
	z.PushRight("b")
	z.PushLeft("z")
	if z.Len() != 3 {
		t.Fatalf("Len = %d, want 3", z.Len())
	}
	v, ok := z.PopLeft()
	if !ok || v != "z" {
		t.Fatalf("PopLeft = %q %v, want z true", v, ok)
	}
	v, ok = z.PopRight()
	if !ok || v != "b" {
		t.Fatalf("PopRight = %q %v, want b true", v, ok)
	}
}

func TestZipListRangeIndexSet(t *testing.T) {
	z := ziplistFromEntries([]string{"a", "b", "c", "d"})
	got := z.Range(1, 2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("Range = %v, want [b c]", got)
	}
	v, ok := z.Index(-1)
	if !ok || v != "d" {
		t.Fatalf("Index(-1) = %q %v, want d", v, ok)
	}
	if !z.Set(1, "B") {
		t.Fatal("Set failed")
	}
	v, _ = z.Index(1)
	if v != "B" {
		t.Fatalf("after Set Index(1) = %q", v)
	}
}

func TestZipListInsertRemTrim(t *testing.T) {
	z := ziplistFromEntries([]string{"a", "b", "a", "c"})
	if !z.InsertBefore("b", "x") {
		t.Fatal("InsertBefore failed")
	}
	if z.Rem(1, "a") != 1 {
		t.Fatal("Rem count mismatch")
	}
	z.Trim(0, 1)
	if z.Len() != 2 {
		t.Fatalf("after Trim Len = %d, want 2", z.Len())
	}
}
