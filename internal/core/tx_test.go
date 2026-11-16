package core

import (
	"bytes"
	"strings"
	"testing"
)

func eval(t *testing.T, client *Client, cmd string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	err := EvalAndRespond(client, &RedisCmd{Cmd: cmd, Args: args}, &out)
	if err != nil {
		return "-" + err.Error() + "\r\n"
	}
	return out.String()
}

func TestMULTIEXEC_Basic(t *testing.T) {
	resetStore()
	c := NewClient()

	if got := eval(t, c, "MULTI"); got != "+OK\r\n" {
		t.Fatalf("MULTI = %q", got)
	}
	if got := eval(t, c, "SET", "k", "v"); got != "+QUEUED\r\n" {
		t.Fatalf("SET queued = %q", got)
	}
	if got := eval(t, c, "GET", "k"); got != "+QUEUED\r\n" {
		t.Fatalf("GET queued = %q", got)
	}
	got := eval(t, c, "EXEC")
	if !strings.HasPrefix(got, "*2\r\n") {
		t.Fatalf("EXEC = %q, want array of 2", got)
	}
	if !strings.Contains(got, "+OK\r\n") || !strings.Contains(got, "$1\r\nv\r\n") {
		t.Fatalf("EXEC body = %q", got)
	}
	if c.InTx || len(c.Queue) != 0 {
		t.Fatal("expected tx cleared after EXEC")
	}
}

func TestDISCARD(t *testing.T) {
	resetStore()
	c := NewClient()

	eval(t, c, "MULTI")
	eval(t, c, "SET", "k", "v")
	if got := eval(t, c, "DISCARD"); got != "+OK\r\n" {
		t.Fatalf("DISCARD = %q", got)
	}
	if c.InTx {
		t.Fatal("expected not in tx after DISCARD")
	}
	// SET should not have run
	if got := eval(t, c, "GET", "k"); got != string(RESP_NIL) {
		t.Fatalf("GET after DISCARD = %q, want nil", got)
	}
}

func TestEXECWithoutMULTI(t *testing.T) {
	resetStore()
	c := NewClient()
	got := eval(t, c, "EXEC")
	if !strings.Contains(got, "EXEC without MULTI") {
		t.Fatalf("EXEC = %q", got)
	}
}

func TestDISCARDWithoutMULTI(t *testing.T) {
	resetStore()
	c := NewClient()
	got := eval(t, c, "DISCARD")
	if !strings.Contains(got, "DISCARD without MULTI") {
		t.Fatalf("DISCARD = %q", got)
	}
}

func TestNestedMULTI(t *testing.T) {
	resetStore()
	c := NewClient()
	eval(t, c, "MULTI")
	got := eval(t, c, "MULTI")
	if !strings.Contains(got, "MULTI calls can not be nested") {
		t.Fatalf("nested MULTI = %q", got)
	}
}

func TestWATCH_ConflictAbortsEXEC(t *testing.T) {
	resetStore()
	c1 := NewClient()
	c2 := NewClient()

	eval(t, c1, "SET", "k", "1")
	if got := eval(t, c1, "WATCH", "k"); got != "+OK\r\n" {
		t.Fatalf("WATCH = %q", got)
	}
	// Another client mutates the watched key.
	eval(t, c2, "SET", "k", "2")

	eval(t, c1, "MULTI")
	eval(t, c1, "SET", "k", "3")
	got := eval(t, c1, "EXEC")
	if got != string(RESP_NIL_ARRAY) {
		t.Fatalf("EXEC after conflict = %q, want nil array", got)
	}
	// Queued SET must not have applied.
	if got := eval(t, c1, "GET", "k"); got != "$1\r\n2\r\n" {
		t.Fatalf("GET after aborted EXEC = %q, want 2", got)
	}
}

func TestWATCH_NoConflictSucceeds(t *testing.T) {
	resetStore()
	c := NewClient()

	eval(t, c, "SET", "k", "1")
	eval(t, c, "WATCH", "k")
	eval(t, c, "MULTI")
	eval(t, c, "SET", "k", "2")
	got := eval(t, c, "EXEC")
	if !strings.HasPrefix(got, "*1\r\n") || !strings.Contains(got, "+OK\r\n") {
		t.Fatalf("EXEC = %q", got)
	}
	if got := eval(t, c, "GET", "k"); got != "$1\r\n2\r\n" {
		t.Fatalf("GET = %q", got)
	}
}

func TestWATCH_SameClientWriteDirties(t *testing.T) {
	resetStore()
	c := NewClient()

	eval(t, c, "SET", "k", "1")
	eval(t, c, "WATCH", "k")
	// Same client writes outside MULTI — Redis dirties the watch.
	eval(t, c, "SET", "k", "x")
	eval(t, c, "MULTI")
	eval(t, c, "SET", "k", "y")
	got := eval(t, c, "EXEC")
	if got != string(RESP_NIL_ARRAY) {
		t.Fatalf("EXEC = %q, want nil array", got)
	}
}

func TestWATCH_InsideMULTI_Error(t *testing.T) {
	resetStore()
	c := NewClient()
	eval(t, c, "MULTI")
	got := eval(t, c, "WATCH", "k")
	if !strings.Contains(got, "WATCH inside MULTI") {
		t.Fatalf("WATCH in MULTI = %q", got)
	}
}

func TestEXEC_EmptyQueue(t *testing.T) {
	resetStore()
	c := NewClient()
	eval(t, c, "MULTI")
	got := eval(t, c, "EXEC")
	if got != "*0\r\n" {
		t.Fatalf("empty EXEC = %q, want *0", got)
	}
}

func TestEXEC_PartialErrorNoRollback(t *testing.T) {
	resetStore()
	c := NewClient()

	eval(t, c, "SET", "b", "hello")
	eval(t, c, "MULTI")
	eval(t, c, "SET", "b", "world")
	eval(t, c, "INCR", "b") // fails: world is not an int
	eval(t, c, "SET", "c", "ok")
	got := eval(t, c, "EXEC")
	if !strings.HasPrefix(got, "*3\r\n") {
		t.Fatalf("EXEC = %q", got)
	}
	if !strings.Contains(got, "+OK\r\n") {
		t.Fatalf("EXEC missing OK: %q", got)
	}
	if !strings.Contains(got, "not an integer") {
		t.Fatalf("EXEC missing INCR error: %q", got)
	}
	// SET b and SET c should have applied despite INCR failure (no rollback).
	if got := eval(t, c, "GET", "b"); got != "$5\r\nworld\r\n" {
		t.Fatalf("GET b = %q", got)
	}
	if got := eval(t, c, "GET", "c"); got != "$2\r\nok\r\n" {
		t.Fatalf("GET c = %q", got)
	}
}

func TestQueueUnknownCommandAbortsEXEC(t *testing.T) {
	resetStore()
	c := NewClient()
	eval(t, c, "MULTI")
	got := eval(t, c, "NOTACOMMAND")
	if !strings.Contains(got, "unknown command") {
		t.Fatalf("unknown queued = %q", got)
	}
	got = eval(t, c, "EXEC")
	if !strings.Contains(got, "EXECABORT") {
		t.Fatalf("EXEC after abort = %q", got)
	}
}

func TestDISCARD_ClearsWatches(t *testing.T) {
	resetStore()
	c := NewClient()
	eval(t, c, "SET", "k", "1")
	eval(t, c, "WATCH", "k")
	eval(t, c, "MULTI")
	eval(t, c, "DISCARD")
	// Watches cleared — a mutation then MULTI/EXEC should succeed.
	eval(t, NewClient(), "SET", "k", "2")
	eval(t, c, "MULTI")
	eval(t, c, "GET", "k")
	got := eval(t, c, "EXEC")
	if !strings.Contains(got, "$1\r\n2\r\n") {
		t.Fatalf("EXEC after DISCARD cleared watches = %q", got)
	}
}
