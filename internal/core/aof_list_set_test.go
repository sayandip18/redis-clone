package core

import (
	"os"
	"strings"
	"testing"

	"github.com/sayandip18/redis-clone/config"
)

func TestDumpAllAOF_ListAndSet(t *testing.T) {
	resetStore()
	path := withTempAOF(t)
	prev := config.KeysLimit
	config.KeysLimit = 100
	t.Cleanup(func() { config.KeysLimit = prev })

	_, _ = listPush("mylist", listRight, []string{"a", "b"})
	_, _ = setAdd("myset", []string{"1", "2"})

	if err := DumpAllAOF(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, encodeAOFCommand("RPUSH", "mylist", "a", "b")) {
		t.Fatalf("AOF missing RPUSH, got:\n%s", content)
	}
	if !strings.Contains(content, "SADD") || !strings.Contains(content, "myset") {
		t.Fatalf("AOF missing SADD, got:\n%s", content)
	}
}
