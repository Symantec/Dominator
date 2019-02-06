package rpcd

import (
	"bytes"
	"encoding/gob"
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/bufwriter"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

type logWriterType struct {
	encoder srpc.Encoder
}

func (t *srpcType) BuildImage(conn *srpc.Conn) error {
	decoder := gob.NewDecoder(conn)
	var request proto.BuildImageRequest
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	buildLogBuffer := &bytes.Buffer{}
	var logWriter io.Writer
	var encoder srpc.Encoder
	if request.StreamBuildLog {
		writer := bufwriter.NewWriter(conn, time.Millisecond*100)
		defer writer.Flush()
		encoder = gob.NewEncoder(writer)
		logWriter = &logWriterType{encoder}
	} else {
		encoder = gob.NewEncoder(conn)
		logWriter = buildLogBuffer
	}
	name, err := t.builder.BuildImage(request.StreamName, request.ExpiresIn,
		request.GitBranch, request.MaxSourceAge, logWriter)
	reply := proto.BuildImageResponse{
		ImageName:   name,
		BuildLog:    buildLogBuffer.Bytes(),
		ErrorString: errors.ErrorToString(err),
	}
	return encoder.Encode(reply)
}

func (w *logWriterType) Write(p []byte) (int, error) {
	reply := proto.BuildImageResponse{BuildLog: p}
	if err := w.encoder.Encode(reply); err != nil {
		return 0, err
	}
	return len(p), nil
}
