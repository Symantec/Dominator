package client

import (
	"io"
)

type FlushReadWriter interface {
	Flush() error
	io.ReadWriter
}

func StartTerminal(conn FlushReadWriter) error {
	return startTerminal(conn)
}
