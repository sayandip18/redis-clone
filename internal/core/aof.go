package core

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sayandip18/redis-clone/config"
)

func DumpAllAOF() error {
	var sb strings.Builder
	now := time.Now().UnixMilli()
	written := 0

	for key, obj := range store {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= now {
			continue
		}

		value, err := GetStringValue(obj)
		if err != nil {
			// Only string objects are supported in AOF for now.
			continue
		}

		sb.WriteString(encodeAOFCommand("SET", key, value))

		if obj.ExpiresAt != -1 {
			sb.WriteString(encodeAOFCommand("PEXPIREAT", key, strconv.FormatInt(obj.ExpiresAt, 10)))
		}
		written++
	}

	if err := os.WriteFile(config.AOFFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write AOF file: %w", err)
	}

	log.Printf("AOF rewrite complete: wrote %d keys to %s", written, config.AOFFile)
	return nil
}

func encodeAOFCommand(args ...string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}
	return sb.String()
}
