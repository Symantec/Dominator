// Package bufwriter implements a simplified buffered writer, similar to the
// bufio package in the Go standard library, but adds automatic flushing.
package bufwriter

import (
	"io"
	"sync"
	"time"
)

type flushingWriter interface {
	Flush() error
	io.Writer
}

type Writer struct {
	flushDelay     time.Duration
	flushingWriter flushingWriter
	mutex          sync.Mutex // Protect everything below.
	err            error
	flushPending   bool
}

func NewWriter(writer io.Writer, flushDelay time.Duration) *Writer {
	return newWriter(writer, flushDelay)
}

func (b *Writer) Flush() error { return b.flush() }

func (b *Writer) Write(p []byte) (int, error) { return b.write(p) }
