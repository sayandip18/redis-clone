package core

import (
	"testing"

	"github.com/sayandip18/redis-clone/config"
)

func TestQuickListPushPopAcrossNodes(t *testing.T) {
	prev := config.ListMaxZiplistEntries
	config.ListMaxZiplistEntries = 2
	t.Cleanup(func() { config.ListMaxZiplistEntries = prev })

	ql := newQuickList()
	ql.PushRight("a")
	ql.PushRight("b")
	ql.PushRight("c") // forces new node
	if ql.Len() != 3 {
		t.Fatalf("Len = %d, want 3", ql.Len())
	}
	v, ok := ql.PopLeft()
	if !ok || v != "a" {
		t.Fatalf("PopLeft = %q %v", v, ok)
	}
	v, ok = ql.PopRight()
	if !ok || v != "c" {
		t.Fatalf("PopRight = %q %v", v, ok)
	}
	if ql.Len() != 1 {
		t.Fatalf("Len after pops = %d, want 1", ql.Len())
	}
}
