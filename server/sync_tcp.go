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

func readCommand(c net.Conn) (*core.RedisCmd, error) {
	buf := make([]byte, 512)
	n, err := c.Read(buf[:])
	if err != nil {
		return nil, err
	}

	tokens, err := resp.DecodeArrayString(buf[:n])
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return &core.RedisCmd{
		Cmd:  tokens[0],
		Args: tokens[1:],
	}, nil
}

func respondError(err error, c net.Conn) error {
	_, writeErr := c.Write([]byte(fmt.Sprintf("-%s\r\n", err.Error())))
	return writeErr
}

func respond(cmd *core.RedisCmd, c net.Conn) error {
	err := core.EvalAndRespond(cmd, c)
	if err != nil {
		return respondError(err, c)
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

		for {
			cmd, err := readCommand(c)
			if err != nil {
				c.Close()
				conClients--
				log.Println("client disconnected", c.RemoteAddr(), "concurrent clients", conClients)
				if err == io.EOF {
					break
				}
				log.Println("err", err)
				break
			}
			log.Println("command", cmd)
			if err = respond(cmd, c); err != nil {
				log.Println("err write:", err)
			}
		}
	}
}
