package core

import (
	"sync/atomic"
	"time"
)

var (
	startTime        time.Time
	connectedClients atomic.Int64
)

// MarkStart records the server start time used for INFO uptime.
func MarkStart() {
	startTime = time.Now()
}

// UptimeSeconds returns seconds since MarkStart. Returns 0 if MarkStart was not called.
func UptimeSeconds() int64 {
	if startTime.IsZero() {
		return 0
	}
	return int64(time.Since(startTime).Seconds())
}

// ClientConnected increments the live client connection count.
func ClientConnected() {
	connectedClients.Add(1)
}

// ClientDisconnected decrements the live client connection count.
func ClientDisconnected() {
	connectedClients.Add(-1)
}

// ConnectedClients returns the current live client connection count.
func ConnectedClients() int64 {
	return connectedClients.Load()
}
