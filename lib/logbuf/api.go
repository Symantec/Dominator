package logbuf

import (
	"container/ring"
	"io"
	"sync"
)

type LogBuffer struct {
	rwMutex sync.RWMutex
	buffer  *ring.Ring // Always points to next insert position.
}

func New(length uint) *LogBuffer {
	return &LogBuffer{buffer: ring.New(int(length))}
}

func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	return lb.write(p)
}

func (lb *LogBuffer) Dump(writer io.Writer, prefix, postfix string) error {
	return lb.dump(writer, prefix, postfix)
}

func (lb *LogBuffer) WriteHtml(writer io.Writer) {
	lb.writeHtml(writer)
}
