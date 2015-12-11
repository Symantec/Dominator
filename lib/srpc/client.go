package srpc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
)

func dialHTTP(network, address string, tlsConfig *tls.Config) (*Client, error) {
	unsecuredConn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	path := rpcPath
	if tlsConfig != nil {
		path = tlsRpcPath
	}
	io.WriteString(unsecuredConn, "CONNECT "+path+" HTTP/1.0\n\n")
	// Require successful HTTP response before switching to SRPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(unsecuredConn),
		&http.Request{Method: "CONNECT"})
	if err != nil {
		return nil, err
	}
	if resp.Status != connectString {
		return nil, errors.New("unexpected HTTP response: " + resp.Status)
	}
	if tlsConfig == nil {
		return newClient(unsecuredConn), nil
	}
	tlsConn := tls.Client(unsecuredConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return newClient(tlsConn), nil
}

func newClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
		bufrw: bufio.NewReadWriter(bufio.NewReader(conn),
			bufio.NewWriter(conn))}
}

func (client *Client) call(serviceMethod string) (*Conn, error) {
	client.callLock.Lock()
	conn, err := client.callWithLock(serviceMethod)
	if err != nil {
		client.callLock.Unlock()
	}
	return conn, err
}

func (client *Client) callWithLock(serviceMethod string) (*Conn, error) {
	_, err := client.bufrw.WriteString(serviceMethod + "\n")
	if err != nil {
		return nil, err
	}
	if err = client.bufrw.Flush(); err != nil {
		return nil, err
	}
	resp, err := client.bufrw.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if resp != "\n" {
		return nil, errors.New(resp[:len(resp)-1])
	}
	conn := new(Conn)
	conn.parent = client
	conn.ReadWriter = client.bufrw
	return conn, nil
}

func (client *Client) close() error {
	client.bufrw.Flush()
	return client.conn.Close()
}
