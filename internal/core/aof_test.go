package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

func resetStore() {
	store = make(map[string]*Obj)
}

func withTempAOF(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dump.aof")
	prev := config.AOFFile
	config.AOFFile = path
	t.Cleanup(func() {
		config.AOFFile = prev
	})
	return path
}

func TestEncodeAOFCommand(t *testing.T) {
	got := encodeAOFCommand("SET", "foo", "bar")
	want := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	if got != want {
		t.Fatalf("encodeAOFCommand() = %q, want %q", got, want)
	}
}

func TestDumpAllAOF_EmptyStore(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	if err := DumpAllAOF(); err != nil {
		t.Fatalf("DumpAllAOF() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("AOF content = %q, want empty", data)
	}
}

func TestDumpAllAOF_SETWithoutExpiry(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	Put("foo", NewStringObj("bar", -1))

	if err := DumpAllAOF(); err != nil {
		t.Fatalf("DumpAllAOF() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	want := encodeAOFCommand("SET", "foo", "bar")
	if string(data) != want {
		t.Fatalf("AOF content = %q, want %q", data, want)
	}
}

func TestDumpAllAOF_SETWithPEXPIREAT(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	expiresAt := time.Now().UnixMilli() + 60_000
	Put("foo", &Obj{Type: ObjString, Encoding: EncodingRaw, Value: "bar", ExpiresAt: expiresAt})

	if err := DumpAllAOF(); err != nil {
		t.Fatalf("DumpAllAOF() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, encodeAOFCommand("SET", "foo", "bar")) {
		t.Fatalf("AOF missing SET command: %q", content)
	}
	wantExpire := encodeAOFCommand("PEXPIREAT", "foo", strconv.FormatInt(expiresAt, 10))
	if !strings.Contains(content, wantExpire) {
		t.Fatalf("AOF missing PEXPIREAT command: %q", content)
	}
}

func TestDumpAllAOF_IntEncodedAsString(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	Put("n", NewStringObj("42", -1))

	if err := DumpAllAOF(); err != nil {
		t.Fatalf("DumpAllAOF() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	want := encodeAOFCommand("SET", "n", "42")
	if string(data) != want {
		t.Fatalf("AOF content = %q, want %q", data, want)
	}
}

func TestDumpAllAOF_SkipsExpiredKeys(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	Put("alive", NewStringObj("ok", -1))
	Put("dead", &Obj{Type: ObjString, Encoding: EncodingRaw, Value: "gone", ExpiresAt: time.Now().UnixMilli() - 1000})

	if err := DumpAllAOF(); err != nil {
		t.Fatalf("DumpAllAOF() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, encodeAOFCommand("SET", "alive", "ok")) {
		t.Fatalf("AOF missing alive key: %q", content)
	}
	if strings.Contains(content, "dead") {
		t.Fatalf("AOF should skip expired key, got: %q", content)
	}
}
