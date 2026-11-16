package core

import "sync/atomic"

var shutdownRequested atomic.Bool

// RequestShutdown marks the server for graceful shutdown.
func RequestShutdown() {
	shutdownRequested.Store(true)
}

// ShutdownRequested reports whether a graceful shutdown has been requested.
func ShutdownRequested() bool {
	return shutdownRequested.Load()
}
