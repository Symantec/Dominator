package srpc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

func call(conn net.Conn, serviceMethod string) error {
	txt := fmt.Sprintf("CONNECT %s%s HTTP/1.0\n\n", rpcPath, serviceMethod)
	io.WriteString(conn, txt)
	// Require successful HTTP response before switching to SRPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn),
		&http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connectString {
		return nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return err
}
