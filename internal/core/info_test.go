package core

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/sayandip18/redis-clone/config"
	"github.com/sayandip18/redis-clone/internal/resp"
)

func TestEvalINFO_Arity(t *testing.T) {
	var out bytes.Buffer
	err := evalINFO([]string{"server"}, &out)
	if err == nil {
		t.Fatal("evalINFO() expected wrong arity error")
	}
}

func TestEvalINFO_EmptyStore(t *testing.T) {
	resetStore()
	MarkStart()
	connectedClients.Store(0)

	var out bytes.Buffer
	if err := evalINFO(nil, &out); err != nil {
		t.Fatalf("evalINFO() error: %v", err)
	}

	body := buildINFO()
	want := string(resp.Encode(body, false))
	if out.String() != want {
		t.Fatalf("INFO response = %q, want %q", out.String(), want)
	}

	if !strings.Contains(body, "# Server\r\n") {
		t.Fatal("missing # Server section")
	}
	if !strings.Contains(body, fmt.Sprintf("tcp_port:%d\r\n", config.Port)) {
		t.Fatalf("missing tcp_port field in %q", body)
	}
	if !strings.Contains(body, "uptime_in_seconds:") {
		t.Fatal("missing uptime_in_seconds field")
	}
	if !strings.Contains(body, "# Clients\r\n") {
		t.Fatal("missing # Clients section")
	}
	if !strings.Contains(body, "connected_clients:0\r\n") {
		t.Fatalf("want connected_clients:0, got %q", body)
	}
	if !strings.Contains(body, "# Keyspace\r\n") {
		t.Fatal("missing # Keyspace section")
	}
	if !strings.Contains(body, "db0:keys=0,expires=0,avg_ttl=0\r\n") {
		t.Fatalf("want empty keyspace, got %q", body)
	}
}

func TestEvalINFO_KeyspaceAndClients(t *testing.T) {
	resetStore()
	MarkStart()
	connectedClients.Store(0)

	Put("a", NewStringObj("1", -1))
	Put("b", NewStringObj("2", 60_000))
	ClientConnected()
	ClientConnected()
	defer func() {
		ClientDisconnected()
		ClientDisconnected()
	}()

	body := buildINFO()
	if !strings.Contains(body, "connected_clients:2\r\n") {
		t.Fatalf("want connected_clients:2, got %q", body)
	}
	if !strings.Contains(body, "db0:keys=2,expires=1,avg_ttl=0\r\n") {
		t.Fatalf("want keys=2,expires=1, got %q", body)
	}

	var out bytes.Buffer
	if err := evalINFO([]string{}, &out); err != nil {
		t.Fatalf("evalINFO() error: %v", err)
	}
	if !strings.HasPrefix(out.String(), "$") {
		t.Fatalf("INFO should be a bulk string, got %q", out.String())
	}
}

func TestKeyCountAndExpiresCount(t *testing.T) {
	resetStore()
	if KeyCount() != 0 || ExpiresCount() != 0 {
		t.Fatalf("empty store counts = %d,%d", KeyCount(), ExpiresCount())
	}
	Put("x", NewStringObj("v", -1))
	Put("y", NewStringObj("v", 5_000))
	if KeyCount() != 2 {
		t.Fatalf("KeyCount() = %d, want 2", KeyCount())
	}
	if ExpiresCount() != 1 {
		t.Fatalf("ExpiresCount() = %d, want 1", ExpiresCount())
	}
}
