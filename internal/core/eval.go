package core

import (
	"bytes"
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
var RESP_NIL_ARRAY []byte = []byte("*-1\r\n")

var errExecAbort = errors.New("EXECABORT Transaction discarded because of previous errors.")

// EvalAndRespond dispatches a command for the given client, writing a RESP reply to w.
func EvalAndRespond(client *Client, cmd *RedisCmd, w io.Writer) error {
	switch cmd.Cmd {
	case "MULTI":
		return evalMULTI(client, cmd.Args, w)
	case "EXEC":
		return evalEXEC(client, cmd.Args, w)
	case "DISCARD":
		return evalDISCARD(client, cmd.Args, w)
	case "WATCH":
		return evalWATCH(client, cmd.Args, w)
	}

	if client != nil && client.InTx {
		return enqueueCommand(client, cmd, w)
	}

	return execCommand(client, cmd, w, true)
}

func execCommand(client *Client, cmd *RedisCmd, w io.Writer, allowBlock bool) error {
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
	case "APPEND":
		return evalAPPEND(cmd.Args, w)
	case "STRLEN":
		return evalSTRLEN(cmd.Args, w)
	case "GETRANGE":
		return evalGETRANGE(cmd.Args, w)
	case "SETRANGE":
		return evalSETRANGE(cmd.Args, w)
	case "INFO":
		return evalINFO(cmd.Args, w)
	case "SHUTDOWN":
		return evalSHUTDOWN(cmd.Args, w)
	case "LPUSH":
		return evalLPUSH(cmd.Args, w)
	case "RPUSH":
		return evalRPUSH(cmd.Args, w)
	case "LPOP":
		return evalLPOP(cmd.Args, w)
	case "RPOP":
		return evalRPOP(cmd.Args, w)
	case "LLEN":
		return evalLLEN(cmd.Args, w)
	case "LRANGE":
		return evalLRANGE(cmd.Args, w)
	case "LINDEX":
		return evalLINDEX(cmd.Args, w)
	case "LSET":
		return evalLSET(cmd.Args, w)
	case "LINSERT":
		return evalLINSERT(cmd.Args, w)
	case "LTRIM":
		return evalLTRIM(cmd.Args, w)
	case "LREM":
		return evalLREM(cmd.Args, w)
	case "BLPOP":
		_, err := evalBlockingPop(client, cmd.Args, w, true, allowBlock)
		return err
	case "BRPOP":
		_, err := evalBlockingPop(client, cmd.Args, w, false, allowBlock)
		return err
	case "SADD":
		return evalSADD(cmd.Args, w)
	case "SREM":
		return evalSREM(cmd.Args, w)
	case "SISMEMBER":
		return evalSISMEMBER(cmd.Args, w)
	case "SCARD":
		return evalSCARD(cmd.Args, w)
	case "SMEMBERS":
		return evalSMEMBERS(cmd.Args, w)
	case "SRANDMEMBER":
		return evalSRANDMEMBER(cmd.Args, w)
	case "SPOP":
		return evalSPOP(cmd.Args, w)
	case "SINTER":
		return evalSINTER(cmd.Args, w)
	case "SUNION":
		return evalSUNION(cmd.Args, w)
	case "SDIFF":
		return evalSDIFF(cmd.Args, w)
	case "SINTERSTORE":
		return evalSINTERSTORE(cmd.Args, w)
	case "SUNIONSTORE":
		return evalSUNIONSTORE(cmd.Args, w)
	case "SDIFFSTORE":
		return evalSDIFFSTORE(cmd.Args, w)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Cmd)
	}
}

func isKnownCommand(name string) bool {
	switch name {
	case "PING", "GET", "SET", "TTL", "DEL", "EXPIRE", "BGREWRITEAOF",
		"OBJECT", "INCR", "APPEND", "STRLEN", "GETRANGE", "SETRANGE", "INFO", "SHUTDOWN",
		"MULTI", "EXEC", "DISCARD", "WATCH",
		"LPUSH", "RPUSH", "LPOP", "RPOP", "LLEN", "LRANGE", "LINDEX", "LSET",
		"LINSERT", "LTRIM", "LREM", "BLPOP", "BRPOP",
		"SADD", "SREM", "SISMEMBER", "SCARD", "SMEMBERS", "SRANDMEMBER", "SPOP",
		"SINTER", "SUNION", "SDIFF", "SINTERSTORE", "SUNIONSTORE", "SDIFFSTORE":
		return true
	default:
		return false
	}
}

func enqueueCommand(client *Client, cmd *RedisCmd, w io.Writer) error {
	if !isKnownCommand(cmd.Cmd) {
		client.Aborted = true
		return fmt.Errorf("unknown command: %s", cmd.Cmd)
	}
	// Transaction control commands are handled before enqueue; anything else queues.
	client.Queue = append(client.Queue, cmd)
	_, err := w.Write([]byte("+QUEUED\r\n"))
	return err
}

func evalMULTI(client *Client, args []string, w io.Writer) error {
	if len(args) != 0 {
		return errors.New("(error) ERR wrong number of arguments for 'multi' command")
	}
	if client == nil {
		return errors.New("(error) ERR MULTI requires a client")
	}
	if client.InTx {
		return errors.New("(error) ERR MULTI calls can not be nested")
	}
	client.InTx = true
	client.Queue = nil
	client.Aborted = false
	_, err := w.Write([]byte("+OK\r\n"))
	return err
}

func evalEXEC(client *Client, args []string, w io.Writer) error {
	if len(args) != 0 {
		return errors.New("(error) ERR wrong number of arguments for 'exec' command")
	}
	if client == nil || !client.InTx {
		return errors.New("(error) ERR EXEC without MULTI")
	}

	aborted := client.Aborted
	queue := client.Queue
	watchesOK := client.watchesIntact()

	// Always clear transaction + watches after EXEC (Redis semantics).
	client.clearTxAndWatches()

	if aborted {
		return errExecAbort
	}
	if !watchesOK {
		_, err := w.Write(RESP_NIL_ARRAY)
		return err
	}

	var out bytes.Buffer
	out.WriteString(fmt.Sprintf("*%d\r\n", len(queue)))
	for _, queued := range queue {
		var buf bytes.Buffer
		if err := execCommand(client, queued, &buf, false); err != nil {
			buf.Reset()
			buf.WriteString(fmt.Sprintf("-%s\r\n", err.Error()))
		}
		out.Write(buf.Bytes())
	}
	_, err := w.Write(out.Bytes())
	return err
}

func evalDISCARD(client *Client, args []string, w io.Writer) error {
	if len(args) != 0 {
		return errors.New("(error) ERR wrong number of arguments for 'discard' command")
	}
	if client == nil || !client.InTx {
		return errors.New("(error) ERR DISCARD without MULTI")
	}
	client.clearTxAndWatches()
	_, err := w.Write([]byte("+OK\r\n"))
	return err
}

func evalWATCH(client *Client, args []string, w io.Writer) error {
	if len(args) == 0 {
		return errors.New("(error) ERR wrong number of arguments for 'watch' command")
	}
	if client == nil {
		return errors.New("(error) ERR WATCH requires a client")
	}
	if client.InTx {
		return errors.New("(error) ERR WATCH inside MULTI is not allowed")
	}
	if client.Watching == nil {
		client.Watching = make(map[string]uint64)
	}
	for _, key := range args {
		// Only snapshot if not already watching this key (Redis keeps first watch version).
		if _, ok := client.Watching[key]; !ok {
			client.Watching[key] = KeyVersion(key)
		}
	}
	_, err := w.Write([]byte("+OK\r\n"))
	return err
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
	_, err := c.Write([]byte("+OK\r\n"))
	return err
}

func evalGET(args []string, c io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'get' command")
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		_, err := c.Write(RESP_NIL)
		return err
	}

	value, err := GetStringValue(obj)
	if err != nil {
		return err
	}

	_, err = c.Write(resp.Encode(value, false))
	return err
}

func evalTTL(args []string, c io.Writer) error {
	if len(args) != 1 {
		return errors.New("(error) ERR wrong number of arguments for 'ttl' command")
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		_, err := c.Write([]byte(":-2\r\n"))
		return err
	}

	if obj.ExpiresAt == -1 {
		_, err := c.Write([]byte(":-1\r\n"))
		return err
	}

	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	if durationMs < 0 {
		_, err := c.Write([]byte(":-2\r\n"))
		return err
	}

	_, err := c.Write(resp.Encode(int64(durationMs/1000), false))
	return err
}

func evalDEL(args []string, c io.Writer) error {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	_, err := c.Write(resp.Encode(int64(countDeleted), false))
	return err
}

func evalEXPIRE(args []string, c io.Writer) error {
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
		_, err := c.Write([]byte(":0\r\n"))
		return err
	}

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000
	bumpVersion(key)

	_, err = c.Write([]byte(":1\r\n"))
	return err
}

func evalBGREWRITEAOF(args []string, c io.Writer) error {
	if err := DumpAllAOF(); err != nil {
		return err
	}
	_, err := c.Write([]byte("+OK\r\n"))
	return err
}

func evalSHUTDOWN(args []string, c io.Writer) error {
	if len(args) != 0 {
		return errors.New("(error) ERR wrong number of arguments for 'shutdown' command")
	}
	if _, err := c.Write([]byte("+OK\r\n")); err != nil {
		return err
	}
	RequestShutdown()
	return nil
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
	created := false
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
		created = true
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
	// Put already bumps on create; in-place updates need an explicit bump.
	if !created {
		bumpVersion(key)
	}

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
