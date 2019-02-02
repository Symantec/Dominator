package bufwriter

import (
	"bufio"
	"io"
	"time"
)

func newWriter(writer io.Writer, flushDelay time.Duration) *Writer {
	b := &Writer{flushDelay: flushDelay}
	if bufWriter, ok := writer.(flushingWriter); ok {
		b.flushingWriter = bufWriter
	} else {
		b.flushingWriter = bufio.NewWriter(writer)
	}
	return b
}

func (b *Writer) delayedFlush() {
	time.Sleep(b.flushDelay)
	b.flush()
}

func (b *Writer) flush() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.flushPending = false
	if b.err != nil {
		return b.err
	}
	b.err = b.flushingWriter.Flush()
	return b.err
}

func (b *Writer) lockAndScheduleFlush() {
	b.mutex.Lock()
	if b.flushPending {
		return
	}
	b.flushPending = true
	go b.delayedFlush()
}

func (b *Writer) write(p []byte) (int, error) {
	b.lockAndScheduleFlush()
	defer b.mutex.Unlock()
	if b.err != nil {
		return 0, b.err
	}
	nWritten, err := b.flushingWriter.Write(p)
	b.err = err
	return nWritten, err
}
