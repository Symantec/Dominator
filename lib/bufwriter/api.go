// Package bufwriter implements a simplified buffered writer, similar to the
// bufio package in the Go standard library, but adds automatic flushing.
package bufwriter

import (
	"io"
	"sync"
	"time"
)

type FlushWriter interface {
	Flush() error
	io.Writer
}

type Writer struct {
	flushDelay     time.Duration
	flushingWriter FlushWriter
	mutex          sync.Mutex // Protect everything below.
	err            error
	flushPending   bool
}

// NewAutoFlushWriter wraps a FlushWriter and returns an io.Writer. The returned
// writer will automatically call the wrapped Flush method after each Write
// call.
func NewAutoFlushWriter(w FlushWriter) io.Writer {
	return newAutoFlushWriter(w)
}

// NewWriter wraps a io.Writer and returns a *Writer. Written data are flushed
// within the time specified by flushDelay. If writer does not implement the
// FlushWriter interface then a bufio.Writer is created.
func NewWriter(writer io.Writer, flushDelay time.Duration) *Writer {
	return newWriter(writer, flushDelay)
}

func (b *Writer) Flush() error { return b.flush() }

func (b *Writer) Write(p []byte) (int, error) { return b.write(p) }
