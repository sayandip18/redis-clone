//go:build linux

package server

import (
	"log"
	"net"
	"syscall"
	"time"

	"github.com/sayandip18/redis-clone/config"
	"github.com/sayandip18/redis-clone/internal/core"
)

func RunAsyncTCPServer() {
	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)

	maxClients := 20000
	conClients := 0
	var cronFrequency time.Duration = 1 * time.Second
	var lastCronExecTime time.Time = time.Now()

	events := make([]syscall.EpollEvent, maxClients)

	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(serverFD)

	if err = syscall.SetsockoptInt(serverFD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		panic(err)
	}

	if err = syscall.SetNonblock(serverFD, true); err != nil {
		panic(err)
	}

	ip4 := net.ParseIP(config.Host).To4()
	if ip4 == nil {
		panic("invalid IPv4 host: " + config.Host)
	}

	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		panic(err)
	}

	if err = syscall.Listen(serverFD, maxClients); err != nil {
		panic(err)
	}

	epollFD, err := syscall.EpollCreate1(0)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(epollFD)

	serverEvent := syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(serverFD),
	}
	if err = syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, serverFD, &serverEvent); err != nil {
		panic(err)
	}

	for {
		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}
		nevents, err := syscall.EpollWait(epollFD, events, 100)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			log.Println("err epoll wait:", err)
			continue
		}

		for i := 0; i < nevents; i++ {
			fd := int(events[i].Fd)

			if fd == serverFD {
				clientFD, sa, err := syscall.Accept(serverFD)
				if err != nil {
					log.Println("err accept:", err)
					continue
				}

				if err = syscall.SetNonblock(clientFD, true); err != nil {
					log.Println("err set nonblock:", err)
					syscall.Close(clientFD)
					continue
				}

				clientEvent := syscall.EpollEvent{
					Events: syscall.EPOLLIN,
					Fd:     int32(clientFD),
				}
				if err = syscall.EpollCtl(epollFD, syscall.EPOLL_CTL_ADD, clientFD, &clientEvent); err != nil {
					log.Println("err epoll add client:", err)
					syscall.Close(clientFD)
					continue
				}

				conClients++
				log.Println("client connected with address:", sockaddrString(sa), "concurrent clients", conClients)
				continue
			}

			comm := FDComm{Fd: fd}
			cmd, err := readCommand(comm)
			if err != nil {
				syscall.Close(fd)
				conClients--
				log.Println("client disconnected", "fd", fd, "concurrent clients", conClients)
				log.Println("err", err)
				continue
			}

			log.Println("command", cmd)
			if err = respond(cmd, comm); err != nil {
				log.Println("err write:", err)
			}
		}
	}
}

func sockaddrString(sa syscall.Sockaddr) string {
	switch addr := sa.(type) {
	case *syscall.SockaddrInet4:
		return net.IP(addr.Addr[:]).String()
	case *syscall.SockaddrInet6:
		return net.IP(addr.Addr[:]).String()
	default:
		return "unknown"
	}
}
