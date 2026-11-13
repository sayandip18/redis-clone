//go:build linux

package server

import (
	"io"
	"syscall"
)

// FDComm wraps a raw file descriptor so it can be used as an io.ReadWriter
// with the shared command read/respond helpers.
type FDComm struct {
	Fd int
}

func (f FDComm) Read(p []byte) (int, error) {
	n, err := syscall.Read(f.Fd, p)
	if n == 0 && err == nil {
		return 0, io.EOF
	}
	return n, err
}

func (f FDComm) Write(p []byte) (int, error) {
	return syscall.Write(f.Fd, p)
}
