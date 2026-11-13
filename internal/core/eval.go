package core

import (
	"errors"
	"fmt"
	"io"

	"github.com/sayandip18/redis-clone/internal/resp"
)

func EvalAndRespond(cmd *RedisCmd, w io.ReadWriter) error {
	switch cmd.Cmd {
	case "PING":
		return evalPING(cmd.Args, w)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Cmd)
	}
}

func evalPING(args []string, w io.Writer) error {
	var b []byte

	if len(args) >= 2 {
		return errors.New("wrong number of arguments for 'ping' command")
	}

	if len(args) == 0 {
		b = resp.Encode("PONG", true)
	} else {
		b = resp.Encode(args[0], false)
	}

	_, err := w.Write(b)
	return err
}
