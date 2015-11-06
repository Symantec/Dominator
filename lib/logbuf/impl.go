package logbuf

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	alsoLogToStderr = flag.Bool("alsoLogToStderr", false,
		"If true, also write logs to stderr")
)

func (lb *LogBuffer) write(p []byte) (n int, err error) {
	if *alsoLogToStderr {
		os.Stderr.Write(p)
	}
	lb.rwMutex.Lock()
	defer lb.rwMutex.Unlock()
	val := make([]byte, len(p))
	copy(val, p)
	lb.buffer.Value = val
	lb.buffer = lb.buffer.Next()
	return len(p), nil
}

func (lb *LogBuffer) dump(writer io.Writer, prefix, postfix string) error {
	lb.rwMutex.RLock()
	defer lb.rwMutex.RUnlock()
	lb.buffer.Do(func(p interface{}) {
		if p != nil {
			writer.Write([]byte(prefix))
			writer.Write(p.([]byte))
			writer.Write([]byte(postfix))
		}
	})
	return nil
}

func (lb *LogBuffer) writeHtml(writer io.Writer) {
	fmt.Fprintln(writer, "Logs:<br>")
	fmt.Fprintln(writer, "<pre>")
	lb.Dump(writer, "", "")
	fmt.Fprintln(writer, "</pre>")
}
