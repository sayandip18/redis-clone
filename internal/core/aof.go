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

const aofChunkSize = 64

func DumpAllAOF() error {
	var sb strings.Builder
	now := time.Now().UnixMilli()
	written := 0

	for key, obj := range store {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= now {
			continue
		}

		switch obj.Type {
		case ObjString:
			value, err := GetStringValue(obj)
			if err != nil {
				continue
			}
			sb.WriteString(encodeAOFCommand("SET", key, value))
		case ObjList:
			entries, err := listEntries(obj)
			if err != nil || len(entries) == 0 {
				continue
			}
			for i := 0; i < len(entries); i += aofChunkSize {
				end := i + aofChunkSize
				if end > len(entries) {
					end = len(entries)
				}
				args := make([]string, 0, 2+end-i)
				args = append(args, "RPUSH", key)
				args = append(args, entries[i:end]...)
				sb.WriteString(encodeAOFCommand(args...))
			}
		case ObjSet:
			members, err := setMembersForAOF(obj)
			if err != nil || len(members) == 0 {
				continue
			}
			for i := 0; i < len(members); i += aofChunkSize {
				end := i + aofChunkSize
				if end > len(members) {
					end = len(members)
				}
				args := make([]string, 0, 2+end-i)
				args = append(args, "SADD", key)
				args = append(args, members[i:end]...)
				sb.WriteString(encodeAOFCommand(args...))
			}
		default:
			continue
		}

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
