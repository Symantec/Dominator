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
	"sync"
	"time"

	libnet "github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

var (
	clientMetricsDir          *tricorder.DirectorySpec
	clientMetricsMutex        sync.Mutex
	numInUseClientConnections uint64
	numOpenClientConnections  uint64
)

func init() {
	registerClientMetrics()
}

func registerClientMetrics() {
	var err error
	clientMetricsDir, err = tricorder.RegisterDirectory("srpc/client")
	if err != nil {
		panic(err)
	}
	err = clientMetricsDir.RegisterMetric("num-in-use-connections",
		&numInUseClientConnections, units.None,
		"number of connections in use")
	if err != nil {
		panic(err)
	}
	err = clientMetricsDir.RegisterMetric("num-open-connections",
		&numOpenClientConnections, units.None, "number of open connections")
	if err != nil {
		panic(err)
	}
}

func dial(network, address string, dialer Dialer) (net.Conn, error) {
	hostPort := strings.SplitN(address, ":", 2)
	address = strings.SplitN(hostPort[0], "*", 2)[0] + ":" + hostPort[1]
	conn, err := dialer.Dial(network, address)
	if err != nil {
		if strings.Contains(err.Error(), ErrorConnectionRefused.Error()) {
			return nil, ErrorConnectionRefused
		}
		if strings.Contains(err.Error(), ErrorNoRouteToHost.Error()) {
			return nil, ErrorNoRouteToHost
		}
		return nil, err
	}
	if tcpConn, ok := conn.(libnet.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			conn.Close()
			return nil, err
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Minute * 5); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func dialHTTP(network, address string, tlsConfig *tls.Config,
	dialer Dialer) (*Client, error) {
	unsecuredConn, err := dial(network, address, dialer)
	if err != nil {
		return nil, err
	}
	path := rpcPath
	if tlsConfig != nil {
		path = tlsRpcPath
	}
	if err := doHTTPConnect(unsecuredConn, path); err != nil {
		unsecuredConn.Close()
		if strings.Contains(err.Error(), "malformed HTTP response") {
			// The server may do TLS on all connections: try that.
			if tlsConfig == nil {
				return nil, err
			}
			unsecuredConn, err := dial(network, address, dialer)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(unsecuredConn, tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				unsecuredConn.Close()
				if strings.Contains(err.Error(), ErrorBadCertificate.Error()) {
					return nil, ErrorBadCertificate
				}
				return nil, err
			}
			if err := doHTTPConnect(tlsConn, tlsRpcPath); err != nil {
				tlsConn.Close()
				return nil, err
			}
			return newClient(unsecuredConn, tlsConn, true), nil
		} else if err == ErrorNoSrpcEndpoint &&
			tlsConfig != nil &&
			tlsConfig.InsecureSkipVerify {
			// Fall back to insecure connection.
			unsecuredConn, err := dial(network, address, dialer)
			if err != nil {
				return nil, err
			}
			if err := doHTTPConnect(unsecuredConn, rpcPath); err != nil {
				unsecuredConn.Close()
				return nil, err
			}
			return newClient(unsecuredConn, unsecuredConn, false), nil
		} else {
			return nil, err
		}
	}
	if tlsConfig == nil {
		return newClient(unsecuredConn, unsecuredConn, false), nil
	}
	tlsConn := tls.Client(unsecuredConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		unsecuredConn.Close()
		if strings.Contains(err.Error(), ErrorBadCertificate.Error()) {
			return nil, ErrorBadCertificate
		}
		return nil, err
	}
	return newClient(unsecuredConn, tlsConn, true), nil
}

func doHTTPConnect(conn net.Conn, path string) error {
	io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")
	// Require successful HTTP response before switching to SRPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn),
		&http.Request{Method: "CONNECT"})
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrorNoSrpcEndpoint
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrorBadCertificate
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return ErrorMissingCertificate
	}
	if resp.StatusCode != http.StatusOK || resp.Status != connectString {
		return errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil
}

func newClient(rawConn, dataConn net.Conn, isEncrypted bool) *Client {
	clientMetricsMutex.Lock()
	numOpenClientConnections++
	clientMetricsMutex.Unlock()
	client := &Client{
		conn:        dataConn,
		isEncrypted: isEncrypted,
		bufrw: bufio.NewReadWriter(bufio.NewReader(dataConn),
			bufio.NewWriter(dataConn))}
	if tcpConn, ok := rawConn.(libnet.TCPConn); ok {
		client.tcpConn = tcpConn
	}
	return client
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
		clientMetricsMutex.Lock()
		numOpenClientConnections--
		clientMetricsMutex.Unlock()
		return client.conn.Close()
	}
	client.resource.resource.Release()
	clientMetricsMutex.Lock()
	if client.resource.inUse {
		numInUseClientConnections--
		client.resource.inUse = false
	}
	numOpenClientConnections--
	clientMetricsMutex.Unlock()
	return client.resource.closeError
}

func (client *Client) ping() error {
	conn, err := client.call("")
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
	return conn.requestReply(request, reply)
}

func (conn *Conn) requestReply(request interface{}, reply interface{}) error {
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str[:len(str)-1])
	}
	return gob.NewDecoder(conn).Decode(reply)
}
