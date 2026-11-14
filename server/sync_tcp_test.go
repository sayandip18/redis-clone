package server

import (
	"bytes"
	"io"
	"testing"

	"github.com/sayandip18/redis-clone/internal/core"
)

type chunkReader struct {
	chunks [][]byte
	idx    int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[r.idx])
	r.idx++
	return n, nil
}

func TestReadCommandsPipeline(t *testing.T) {
	payload := []byte("*1\r\n$4\r\nPING\r\n*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n")
	var buf []byte
	cmds, err := readCommands(bytes.NewReader(payload), &buf)
	if err != nil && err != io.EOF {
		t.Fatalf("readCommands() error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("readCommands() len = %d, want 2", len(cmds))
	}
	if cmds[0].Cmd != "PING" || len(cmds[0].Args) != 0 {
		t.Fatalf("cmds[0] = %#v", cmds[0])
	}
	if cmds[1].Cmd != "GET" || len(cmds[1].Args) != 1 || cmds[1].Args[0] != "key" {
		t.Fatalf("cmds[1] = %#v", cmds[1])
	}
	if len(buf) != 0 {
		t.Fatalf("leftover buf = %q, want empty", buf)
	}
}

func TestReadCommandsIncompleteAcrossReads(t *testing.T) {
	r := &chunkReader{chunks: [][]byte{
		[]byte("*1\r\n$4\r\nPING\r\n*2\r\n$3\r\nGE"),
		[]byte("T\r\n$3\r\nkey\r\n"),
	}}
	var buf []byte

	cmds, err := readCommands(r, &buf)
	if err != nil {
		t.Fatalf("first readCommands() error: %v", err)
	}
	if len(cmds) != 1 || cmds[0].Cmd != "PING" {
		t.Fatalf("first cmds = %#v, want PING", cmds)
	}
	if len(buf) == 0 {
		t.Fatal("expected incomplete leftover in buf")
	}

	cmds, err = readCommands(r, &buf)
	if err != nil && err != io.EOF {
		t.Fatalf("second readCommands() error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("second cmds len = %d, want 1", len(cmds))
	}
	want := &core.RedisCmd{Cmd: "GET", Args: []string{"key"}}
	if cmds[0].Cmd != want.Cmd || len(cmds[0].Args) != 1 || cmds[0].Args[0] != "key" {
		t.Fatalf("second cmd = %#v, want %#v", cmds[0], want)
	}
	if len(buf) != 0 {
		t.Fatalf("leftover buf = %q, want empty", buf)
	}
}
