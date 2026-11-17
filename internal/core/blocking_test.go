package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

func TestBLPOPImmediate(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	_, _ = listPush("l", listRight, []string{"v1"})
	c := NewClient()
	var out bytes.Buffer
	blocked, err := evalBlockingPop(c, []string{"l", "1"}, &out, true, true)
	if err != nil || blocked {
		t.Fatalf("blocked=%v err=%v", blocked, err)
	}
	if !strings.Contains(out.String(), "l") || !strings.Contains(out.String(), "v1") {
		t.Fatalf("reply = %q", out.String())
	}
}

func TestBLPOPWakeOnPush(t *testing.T) {
	resetStore()
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	c := NewClient()
	var out bytes.Buffer
	blocked, err := evalBlockingPop(c, []string{"l", "0"}, &out, true, true)
	if err != nil || !blocked {
		t.Fatalf("blocked=%v err=%v, want blocked", blocked, err)
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected reply while blocked: %q", out.String())
	}

	if _, err := listPush("l", listRight, []string{"hello"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("wakeup reply = %q", out.String())
	}
}

func TestBLPOPTimeout(t *testing.T) {
	resetStore()
	c := NewClient()
	var out bytes.Buffer
	blocked, err := evalBlockingPop(c, []string{"missing", "0.01"}, &out, true, true)
	if err != nil || !blocked {
		t.Fatalf("blocked=%v err=%v", blocked, err)
	}
	time.Sleep(20 * time.Millisecond)
	ProcessBlockTimeouts()
	if out.String() != string(RESP_NIL_ARRAY) {
		t.Fatalf("timeout reply = %q, want null array", out.String())
	}
}

func TestBLPOPInExecNoBlock(t *testing.T) {
	resetStore()
	c := NewClient()
	var out bytes.Buffer
	blocked, err := evalBlockingPop(c, []string{"missing", "0"}, &out, true, false)
	if err != nil || blocked {
		t.Fatalf("blocked=%v err=%v", blocked, err)
	}
	if out.String() != string(RESP_NIL_ARRAY) {
		t.Fatalf("reply = %q", out.String())
	}
}
