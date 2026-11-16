package core

import (
	"bytes"
	"testing"
)

func TestRequestShutdown(t *testing.T) {
	shutdownRequested.Store(false)
	t.Cleanup(func() { shutdownRequested.Store(false) })

	if ShutdownRequested() {
		t.Fatal("expected shutdown not requested initially")
	}
	RequestShutdown()
	if !ShutdownRequested() {
		t.Fatal("expected shutdown requested after RequestShutdown")
	}
}

func TestEvalSHUTDOWN(t *testing.T) {
	shutdownRequested.Store(false)
	t.Cleanup(func() { shutdownRequested.Store(false) })

	var buf bytes.Buffer
	if err := evalSHUTDOWN(nil, &buf); err != nil {
		t.Fatalf("evalSHUTDOWN: %v", err)
	}
	if got := buf.String(); got != "+OK\r\n" {
		t.Fatalf("response = %q, want +OK\\r\\n", got)
	}
	if !ShutdownRequested() {
		t.Fatal("expected shutdown requested after SHUTDOWN")
	}
}

func TestEvalSHUTDOWNWrongArity(t *testing.T) {
	shutdownRequested.Store(false)
	t.Cleanup(func() { shutdownRequested.Store(false) })

	var buf bytes.Buffer
	err := evalSHUTDOWN([]string{"SAVE"}, &buf)
	if err == nil {
		t.Fatal("expected arity error")
	}
	if ShutdownRequested() {
		t.Fatal("shutdown should not be requested on arity error")
	}
}
