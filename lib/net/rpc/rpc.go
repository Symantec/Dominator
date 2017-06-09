package rpc

import (
	"bufio"
	"errors"
	"github.com/Symantec/Dominator/lib/net"
	"io"
	gonet "net"
	"net/http"
	"net/rpc"
)

func dialHTTPPath(dialer net.Dialer, network, address, path string) (
	*rpc.Client, error) {
	var err error
	conn, err := dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")

	// Require successful HTTP reponse before switching to RPC protocol
	resp, err := http.ReadResponse(
		bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	// The status value is undocumented and subject to change!
	if err == nil && resp.Status == "200 Connected to Go RPC" {
		return rpc.NewClient(conn), nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	conn.Close()
	return nil, &gonet.OpError{
		Op:   "dial-http",
		Net:  network + " " + address,
		Addr: nil,
		Err:  err,
	}
}
