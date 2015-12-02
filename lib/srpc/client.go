package srpc

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

func dialHTTP(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	io.WriteString(conn, "CONNECT "+rpcPath+" HTTP/1.0\n\n")
	// Require successful HTTP response before switching to SRPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn),
		&http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connectString {
		return newClient(conn), nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
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
