package core

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sayandip18/redis-clone/config"
	"github.com/sayandip18/redis-clone/internal/resp"
)

var RESP_NIL []byte = []byte("$-1\r\n")

func EvalAndRespond(cmd *RedisCmd, w io.ReadWriter) error {
	switch cmd.Cmd {
	case "PING":
		return evalPING(cmd.Args, w)
	case "GET":
		return evalGET(cmd.Args, w)
	case "SET":
		return evalSET(cmd.Args, w)
	case "TTL":
		return evalTTL(cmd.Args, w)
	case "DEL":
		return evalDEL(cmd.Args, w)
	case "EXPIRE":
		return evalEXPIRE(cmd.Args, w)
	case "BGREWRITEAOF":
		return evalBGREWRITEAOF(cmd.Args, w)
	case "OBJECT":
		return evalOBJECT(cmd.Args, w)
	case "INCR":
		return evalINCR(cmd.Args, w)
	case "INFO":
		return evalINFO(cmd.Args, w)
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

func evalSET(args []string, c io.Writer) error {
	if len(args) <= 1 {
		return errors.New("(error) ERR wrong number of arguments for 'set' command")
	}

	var key, value string
	var exDurationMs int64 = -1

	key, value = args[0], args[1]

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return errors.New("(error) ERR syntax error")
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return errors.New("(error) ERR value is not an integer or out of range")
			}
			exDurationMs = exDurationSec * 1000
		default:
			return errors.New("(error) ERR syntax error")
		}
	}

	if err := Put(key, NewStringObj(value, exDurationMs)); err != nil {
		return err
	}
	c.Write([]byte("+OK\r\n"))
	return nil
}

func evalGET(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'get' command")
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		c.Write(RESP_NIL)
		return nil
	}

	value, err := GetStringValue(obj)
	if err != nil {
		return err
	}

	c.Write(resp.Encode(value, false))
	return nil
}

func evalTTL(args []string, c io.ReadWriter) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'ttl' command")
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	if obj.ExpiresAt == -1 {
		c.Write([]byte(":-1\r\n"))
		return nil
	}

	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	if durationMs < 0 {
		c.Write([]byte(":-2\r\n"))
		return nil
	}

	c.Write(resp.Encode(int64(durationMs/1000), false))
	return nil
}

func evalDEL(args []string, c io.ReadWriter) error {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	c.Write(resp.Encode(int64(countDeleted), false))
	return nil
}

func evalEXPIRE(args []string, c io.ReadWriter) error {
	if len(args) <= 1 {
		return errors.New("(error) ERR wrong number of arguments for 'expire' command")
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return errors.New("(error) ERR value is not an integer or out of range")
	}

	obj := Get(key)

	if obj == nil {
		c.Write([]byte(":0\r\n"))
		return nil
	}

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000

	c.Write([]byte(":1\r\n"))
	return nil
}

func evalBGREWRITEAOF(args []string, c io.ReadWriter) error {
	if err := DumpAllAOF(); err != nil {
		return err
	}
	_, err := c.Write([]byte("+OK\r\n"))
	return err
}

func evalOBJECT(args []string, c io.Writer) error {
	if len(args) == 0 {
		return errors.New("(error) ERR wrong number of arguments for 'object' command")
	}

	sub := args[0]
	switch sub {
	case "ENCODING", "encoding":
		if len(args) != 2 {
			return errors.New("(error) ERR wrong number of arguments for 'object|encoding' command")
		}
		obj := getNoTouch(args[1])
		if obj == nil {
			_, err := c.Write(RESP_NIL)
			return err
		}
		_, err := c.Write(resp.Encode(EncodingName(obj.Encoding), false))
		return err
	case "IDLETIME", "idletime":
		if len(args) != 2 {
			return errors.New("(error) ERR wrong number of arguments for 'object|idletime' command")
		}
		obj := getNoTouch(args[1])
		if obj == nil {
			_, err := c.Write(RESP_NIL)
			return err
		}
		_, err := c.Write(resp.Encode(IdleTime(obj), false))
		return err
	default:
		return fmt.Errorf("(error) ERR unknown subcommand or wrong number of arguments for '%s'. Try OBJECT HELP.", sub)
	}
}

func evalINCR(args []string, c io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'incr' command")
	}

	key := args[0]
	obj := Get(key)

	var i int64
	if obj == nil {
		obj = &Obj{
			Type:      ObjString,
			Encoding:  EncodingInt,
			Value:     int64(0),
			ExpiresAt: -1,
			LRU:       getLRUClock(),
		}
		if err := Put(key, obj); err != nil {
			return err
		}
	} else {
		if obj.Type != ObjString {
			return ErrWrongType
		}
		n, err := getInt64Value(obj)
		if err != nil {
			return err
		}
		i = n
	}

	if i == math.MaxInt64 {
		return errors.New("(error) ERR increment or decrement would overflow")
	}

	i++
	obj.Encoding = EncodingInt
	obj.Value = i

	_, err := c.Write(resp.Encode(i, false))
	return err
}
func buildINFO() string {
	var b strings.Builder
	b.WriteString("# Server\r\n")
	b.WriteString(fmt.Sprintf("tcp_port:%d\r\n", config.Port))
	b.WriteString(fmt.Sprintf("uptime_in_seconds:%d\r\n", UptimeSeconds()))
	b.WriteString("\r\n")
	b.WriteString("# Clients\r\n")
	b.WriteString(fmt.Sprintf("connected_clients:%d\r\n", ConnectedClients()))
	b.WriteString("\r\n")
	b.WriteString("# Keyspace\r\n")
	b.WriteString(fmt.Sprintf("db0:keys=%d,expires=%d,avg_ttl=0\r\n", KeyCount(), ExpiresCount()))
	return b.String()
}

func evalINFO(args []string, c io.Writer) error {
	if len(args) != 0 {
		return errors.New("(error) ERR wrong number of arguments for 'info' command")
	}

	_, err := c.Write(resp.Encode(buildINFO(), false))
	return err
}
