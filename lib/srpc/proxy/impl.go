package proxy

import (
	"io"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/proxy"
)

type autoFlusher struct {
	writer writeFlusher
}

type writeFlusher interface {
	Flush() error
	io.Writer
}

func (t *srpcType) Connect(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request proto.ConnectRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	requestedConn, err := net.DialTimeout(request.Network, request.Address,
		request.Timeout)
	e := encoder.Encode(proto.ConnectResponse{Error: errors.ErrorToString(err)})
	if e != nil {
		return e
	}
	defer requestedConn.Close()
	if err != nil {
		return nil
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	t.logger.Debugf(0, "made proxy connection to: %s\n", request.Address)
	closed := false
	go func() {
		defer requestedConn.Close()
		if _, err := io.Copy(requestedConn, conn); err != nil {
			if !closed {
				t.logger.Printf("error copying proxied data to %s: %s\n",
					request.Address, err)
			}
		}
		closed = true
	}()
	if _, err := io.Copy(&autoFlusher{conn}, requestedConn); err != nil {
		if !closed {
			t.logger.Printf("error copying proxied data from %s: %s\n",
				request.Address, err)
		}
	}
	closed = true
	return srpc.ErrorCloseClient
}

func (w *autoFlusher) Write(b []byte) (int, error) {
	if nWritten, err := w.writer.Write(b); err != nil {
		w.writer.Flush()
		return nWritten, err
	} else {
		return nWritten, w.writer.Flush()
	}
}
