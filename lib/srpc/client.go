package srpc

import (
	"bufio"
	"crypto/tls"
	"encoding/gob"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func dialHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration) (*Client, error) {
	hostPort := strings.SplitN(address, ":", 2)
	address = strings.SplitN(hostPort[0], "*", 2)[0] + ":" + hostPort[1]
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
	if tcpConn, ok := unsecuredConn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return nil, err
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Minute * 5); err != nil {
			return nil, err
		}
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
		return newClient(unsecuredConn, false), nil
	}
	tlsConn := tls.Client(unsecuredConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		if strings.Contains(err.Error(), ErrorBadCertificate.Error()) {
			return nil, ErrorBadCertificate
		}
		return nil, err
	}
	return newClient(tlsConn, true), nil
}

func newClient(conn net.Conn, isEncrypted bool) *Client {
	return &Client{
		conn:        conn,
		isEncrypted: isEncrypted,
		bufrw: bufio.NewReadWriter(bufio.NewReader(conn),
			bufio.NewWriter(conn))}
}

func (client *Client) call(serviceMethod string) (*Conn, error) {
	if client.conn == nil {
		panic("cannot call Client after Put()")
	}
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
	conn := &Conn{
		parent:      client,
		isEncrypted: client.isEncrypted,
		ReadWriter:  client.bufrw,
	}
	return conn, nil
}

func (client *Client) close() error {
	client.bufrw.Flush()
	if client.resource == nil {
		return client.conn.Close()
	}
	client.resource.resource.Release()
	return client.resource.closeError
}

func (client *Client) ping() error {
	conn, err := client.call("\n")
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (client *Client) requestReply(serviceMethod string, request interface{},
	reply interface{}) error {
	conn, err := client.Call(serviceMethod)
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	conn.Flush()
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str[:len(str)-1])
	}
	return gob.NewDecoder(conn).Decode(reply)
}
