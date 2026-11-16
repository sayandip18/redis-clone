package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sayandip18/redis-clone/config"
	"github.com/sayandip18/redis-clone/internal/core"
	"github.com/sayandip18/redis-clone/server"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "0.0.0.0", "host for the dice server")
	flag.IntVar(&config.Port, "port", 7379, "port for the dice server")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("Rolling the dice...")
	core.MarkStart()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Println("received signal", sig, "- requesting graceful shutdown")
		core.RequestShutdown()

		sig = <-sigCh
		log.Println("received second signal", sig, "- forcing exit")
		os.Exit(1)
	}()

	// server.RunSyncTCPServer()
	server.RunAsyncTCPServer()
	log.Println("server stopped")
}
