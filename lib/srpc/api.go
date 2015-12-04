/*
	Package srpc is similar to the net/rpc package in the Go standard library,
	except that it provides streaming RPC access and TLS support.

	Package srpc provides access to the exported methods of an object across a
	network or other I/O connection. A server registers an object, making it
	visible as a service with the name of the type of the object. After
	registration, exported methods of the object will be accessible remotely.
	A server may register multiple objects (services) of different types but it
	is an error to register multiple objects of the same type.
*/
package srpc

import (
	"bufio"
	"crypto/tls"
	"net"
	"sync"
)

var serverTlsConfig *tls.Config
var clientTlsConfig *tls.Config
var tlsRequired bool

// RegisterName publishes in the server the set of methods of the receiver
// value that satisfy the following interface:
//   func Method(*Conn) error
// The name of the receiver (service) is given by name.
func RegisterName(name string, rcvr interface{}) error {
	return registerName(name, rcvr)
}

// RegisterServerTlsConfig registers the configuration for TLS server
// connections.
// If requireTls is true, any non-TLS connection will be rejected.
func RegisterServerTlsConfig(config *tls.Config, requireTls bool) {
	serverTlsConfig = config
	tlsRequired = requireTls
}

// RegisterClientTlsConfig registers the configuration for TLS client
// connections.
func RegisterClientTlsConfig(config *tls.Config) {
	clientTlsConfig = config
}

// DialHTTP connects to an HTTP SRPC server at the specified network address
// listening on the HTTP SRPC path.
func DialHTTP(network, address string) (*Client, error) {
	return dialHTTP(network, address, clientTlsConfig)
}

// DialHTTP connects to an HTTP SRPC TLS server at the specified network address
// listening on the HTTP SRPC TLS path.
func DialTlsHTTP(network, address string, tlsConfig *tls.Config) (
	*Client, error) {
	return dialHTTP(network, address, tlsConfig)
}

type Client struct {
	conn     net.Conn
	bufrw    *bufio.ReadWriter
	callLock sync.Mutex
}

func (client *Client) Close() error {
	client.bufrw.Flush()
	return client.conn.Close()
}

// Call opens a buffered connection to the named Service.Method function, and
// returns a connection handle and an error status. The connection handle wraps
// a *bufio.ReadWriter. Only one connection can be made per Client. The Close
// method must be called prior to attempting another Call.
func (client *Client) Call(serviceMethod string) (*Conn, error) {
	return client.call(serviceMethod)
}

type Conn struct {
	parent *Client
	*bufio.ReadWriter
}

// Close will close the connection to the Sevice.Method function, releasing the
// Client for a subsequent Call.
func (conn *Conn) Close() error {
	return conn.close()
}
