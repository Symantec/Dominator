package srpc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func dialHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration) (*Client, error) {
	unsecuredConn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		if strings.Contains(err.Error(), ErrorConnectionRefused.Error()) {
			return nil, ErrorConnectionRefused
		}
		if strings.Contains(err.Error(), ErrorNoRouteToHost.Error()) {
			return nil, ErrorNoRouteToHost
		}
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
	if resp.StatusCode == http.StatusNotFound &&
		tlsConfig != nil &&
		tlsConfig.InsecureSkipVerify {
		// Fall back to insecure connection.
		return dialHTTP(network, address, nil, timeout)
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, ErrorMissingCertificate
	}
	if resp.Status != connectString {
		return nil, errors.New("unexpected HTTP response: " + resp.Status)
	}
	if tlsConfig == nil {
		return newClient(unsecuredConn), nil
	}
	tlsConn := tls.Client(unsecuredConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		if strings.Contains(err.Error(), ErrorBadCertificate.Error()) {
			return nil, ErrorBadCertificate
		}
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
		resp := resp[:len(resp)-1]
		if resp == ErrorAccessToMethodDenied.Error() {
			return nil, ErrorAccessToMethodDenied
		}
		return nil, errors.New(resp)
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

func (client *Client) ping() error {
	conn, err := client.call("\n")
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
