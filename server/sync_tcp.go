package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/sayandip18/redis-clone/config"
	"github.com/sayandip18/redis-clone/internal/core"
	"github.com/sayandip18/redis-clone/internal/resp"
)

// readCommands appends the next socket read into buf, drains complete RESP
// command arrays, and leaves any incomplete trailing bytes in buf for later.
func readCommands(r io.Reader, buf *[]byte) ([]*core.RedisCmd, error) {
	tmp := make([]byte, 4096)
	n, err := r.Read(tmp)
	if n > 0 {
		*buf = append(*buf, tmp[:n]...)
	}

	if err != nil && n == 0 {
		if len(*buf) > 0 {
			return nil, fmt.Errorf("connection closed with incomplete command: %w", err)
		}
		return nil, err
	}

	arrays, rest, derr := resp.DrainArrayStrings(*buf)
	*buf = rest
	if derr != nil {
		return nil, derr
	}

	cmds := make([]*core.RedisCmd, 0, len(arrays))
	for _, tokens := range arrays {
		if len(tokens) == 0 {
			return nil, fmt.Errorf("empty command")
		}
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  tokens[0],
			Args: tokens[1:],
		})
	}

	if err != nil {
		if len(*buf) > 0 {
			return cmds, fmt.Errorf("connection closed with incomplete command: %w", err)
		}
		return cmds, err
	}
	return cmds, nil
}

func respondError(err error, w io.Writer) error {
	_, writeErr := w.Write([]byte(fmt.Sprintf("-%s\r\n", err.Error())))
	return writeErr
}

func respond(cmd *core.RedisCmd, rw io.ReadWriter) error {
	err := core.EvalAndRespond(cmd, rw)
	if err != nil {
		return respondError(err, rw)
	}
	return nil
}

func RunSyncTCPServer() {
	log.Println("starting a synchronous TCP server on", config.Host, config.Port)

	conClients := 0
	lsnr, err := net.Listen("tcp", config.Host+":"+strconv.Itoa(config.Port))
	if err != nil {
		panic(err)
	}

	for {
		c, err := lsnr.Accept()
		if err != nil {
			panic(err)
		}
		conClients++
		log.Println("client connected with address:", c.RemoteAddr(), "concurrent clients", conClients)

		var buf []byte
		for {
			cmds, err := readCommands(c, &buf)
			for _, cmd := range cmds {
				log.Println("command", cmd)
				if err := respond(cmd, c); err != nil {
					log.Println("err write:", err)
				}
			}
			if err != nil {
				c.Close()
				conClients--
				log.Println("client disconnected", c.RemoteAddr(), "concurrent clients", conClients)
				if err != io.EOF {
					log.Println("err", err)
				}
				break
			}
		}
	}
}
