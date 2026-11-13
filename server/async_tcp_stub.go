//go:build !linux

package server

import "log"

// RunAsyncTCPServer is only implemented on Linux (epoll).
func RunAsyncTCPServer() {
	log.Fatal("RunAsyncTCPServer requires Linux (epoll); use RunSyncTCPServer on this platform")
}
