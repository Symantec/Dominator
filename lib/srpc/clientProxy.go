package srpc

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/net/proxy"
	proto "github.com/Symantec/Dominator/proto/proxy"
)

var (
	errorUnsupportedTransport = errors.New("unsupported transport")
	errorNotImplemented       = errors.New("not implemented")
)

type fakeAddress struct{}

type proxyConn struct {
	client *Client
	conn   *Conn
}

type proxyDialer struct {
	dialer       *net.Dialer
	proxyAddress string
}

func newProxyDialer(proxyURL string, dialer *net.Dialer) (Dialer, error) {
	if proxyURL == "" {
		return dialer, nil
	}
	if parsedProxy, err := url.Parse(proxyURL); err != nil {
		return nil, err
	} else {
		switch parsedProxy.Scheme {
		case "srpc":
			return &proxyDialer{
				dialer:       dialer,
				proxyAddress: parsedProxy.Host,
			}, nil
		default:
			return proxy.NewDialer(proxyURL, dialer)
		}
	}
}

func (fakeAddress) Network() string {
	return "tcp"
}

func (fakeAddress) String() string {
	return "not-implemented"
}

func (d *proxyDialer) Dial(network, address string) (net.Conn, error) {
	switch network {
	case "tcp":
		return d.dialTCP(address)
	case "udp":
	}
	return nil, errorUnsupportedTransport
}

func (d *proxyDialer) dialTCP(address string) (net.Conn, error) {
	client, err := dialHTTP("tcp", d.proxyAddress, clientTlsConfig, d.dialer)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client != nil {
			client.Close()
		}
	}()
	conn, err := client.Call("Proxy.Connect")
	if err != nil {
		return nil, err
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	err = conn.Encode(proto.ConnectRequest{
		Address: address,
		Network: "tcp",
		Timeout: d.dialer.Timeout,
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Flush(); err != nil {
		return nil, err
	}
	var response proto.ConnectResponse
	if err := conn.Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding: %s", err)
	}
	if err := errors.New(response.Error); err != nil {
		return nil, err
	}
	proxiedConn := proxyConn{
		client: client,
		conn:   conn,
	}
	client = nil
	conn = nil
	return &proxiedConn, nil
}

func (pc *proxyConn) Close() error {
	err1 := pc.conn.Close()
	err2 := pc.client.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (pc *proxyConn) LocalAddr() net.Addr {
	return fakeAddress{}
}

func (pc *proxyConn) Read(b []byte) (int, error) {
	return pc.conn.Read(b)
}

func (pc *proxyConn) RemoteAddr() net.Addr {
	return fakeAddress{}
}

func (pc *proxyConn) SetDeadline(t time.Time) error {
	return errorNotImplemented
}

func (pc *proxyConn) SetReadDeadline(t time.Time) error {
	return errorNotImplemented
}

func (pc *proxyConn) SetWriteDeadline(t time.Time) error {
	return errorNotImplemented
}

func (pc *proxyConn) Write(b []byte) (int, error) {
	if nWritten, err := pc.conn.Write(b); err != nil {
		pc.conn.Flush()
		return nWritten, err
	} else {
		return nWritten, pc.conn.Flush()
	}
}
