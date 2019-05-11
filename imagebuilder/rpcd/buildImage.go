package rpcd

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

type logWriterType struct {
	conn         *srpc.Conn
	mutex        sync.Mutex // Protect everything below.
	err          error
	flushPending bool
}

func (t *srpcType) BuildImage(conn *srpc.Conn) error {
	var request proto.BuildImageRequest
	if err := conn.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	buildLogBuffer := &bytes.Buffer{}
	var logWriter io.Writer
	if request.StreamBuildLog {
		logWriter = &logWriterType{conn: conn}
	} else {
		logWriter = buildLogBuffer
	}
	image, name, err := t.builder.BuildImage(request, conn.GetAuthInformation(),
		logWriter)
	reply := proto.BuildImageResponse{
		Image:       image,
		ImageName:   name,
		BuildLog:    buildLogBuffer.Bytes(),
		ErrorString: errors.ErrorToString(err),
	}
	return conn.Encode(reply)
}

func (w *logWriterType) delayedFlush() {
	time.Sleep(time.Millisecond * 100)
	w.flush()
}

func (w *logWriterType) flush() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.flushPending = false
	if w.err != nil {
		return w.err
	}
	w.err = w.conn.Flush()
	return w.err
}

func (w *logWriterType) lockAndScheduleFlush() {
	w.mutex.Lock()
	if w.flushPending {
		return
	}
	w.flushPending = true
	go w.delayedFlush()
}

func (w *logWriterType) Write(p []byte) (int, error) {
	w.lockAndScheduleFlush()
	defer w.mutex.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	reply := proto.BuildImageResponse{BuildLog: p}
	if err := w.conn.Encode(reply); err != nil {
		return 0, err
	}
	return len(p), nil
}
