package main

import (
	"flag"
	"log"

	"github.com/sayandip18/redis-clone/config"
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
	// server.RunSyncTCPServer()
	server.RunAsyncTCPServer()
}
