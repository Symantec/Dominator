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
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrorConnectionRefused    = errors.New("connection refused")
	ErrorNoRouteToHost        = errors.New("no route to host")
	ErrorMissingCertificate   = errors.New("missing certificate")
	ErrorBadCertificate       = errors.New("bad certificate")
	ErrorAccessToMethodDenied = errors.New("access to method denied")
)

var serverTlsConfig *tls.Config
var clientTlsConfig *tls.Config
var tlsRequired bool

// RegisterName publishes in the server the set of methods of the receiver
// value that satisfy one of the following interfaces:
//   func Method(*Conn) error
//   func Method(*Conn, request, *response) error
// The request/response method must not perform I/O on the Conn type. This is
// passed only to provide access to connection metadata.
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
// listening on the HTTP SRPC path. If timeout is zero or less, the underlying
// OS timeout is used (typically 3 minutes for TCP).
func DialHTTP(network, address string, timeout time.Duration) (*Client, error) {
	return dialHTTP(network, address, clientTlsConfig, timeout)
}

// DialHTTP connects to an HTTP SRPC TLS server at the specified network address
// listening on the HTTP SRPC TLS path. If timeout is zero or less, the
// underlying OS timeout is used (typically 3 minutes for TCP).
func DialTlsHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration) (
	*Client, error) {
	if tlsConfig == nil {
		tlsConfig = clientTlsConfig
	}
	return dialHTTP(network, address, tlsConfig, timeout)
}

type Client struct {
	conn        net.Conn
	isEncrypted bool
	bufrw       *bufio.ReadWriter
	callLock    sync.Mutex
}

func (client *Client) Close() error {
	client.bufrw.Flush()
	return client.conn.Close()
}

// Call opens a buffered connection to the named Service.Method function, and
// returns a connection handle and an error status. The connection handle wraps
// a *bufio.ReadWriter. Only one connection can be made per Client. The Call
// method will block if another Call is in progress. The Close method must be
// called prior to attempting another Call.
func (client *Client) Call(serviceMethod string) (*Conn, error) {
	return client.call(serviceMethod)
}

// IsEncrypted will return true if the underlying connection is TLS-encrypted.
func (client *Client) IsEncrypted() bool {
	return client.isEncrypted
}

// Ping sends a short "are you alive?" request and waits for a response. No
// method permissions are required for this operation. The Ping method is a
// wrapper around the Call method and hence will block if a Call is already in
// progress.
func (client *Client) Ping() error {
	return client.ping()
}

// RequestReply sends a request message to the named Service.Method function,
// and waits for a reply. The request and reply messages are GOB encoded and
// decoded, respectively. This method is a convenience wrapper around the Call
// method.
func (client *Client) RequestReply(serviceMethod string, request interface{},
	reply interface{}) error {
	return client.requestReply(serviceMethod, request, reply)
}

type Conn struct {
	parent      *Client // nil: server-side connection.
	isEncrypted bool
	*bufio.ReadWriter
	username         string              // Empty string for unauthenticated.
	permittedMethods map[string]struct{} // nil: all, empty: none permitted.
}

// Close will close the connection to the Sevice.Method function, releasing the
// Client for a subsequent Call.
func (conn *Conn) Close() error {
	return conn.close()
}

// Username will return the username of the client who holds the certificate
// used to authenticate the connection to the server. If the connection was not
// authenticated the emtpy string is returned. If the connection is a client
// connection, then Username will panic.
func (conn *Conn) Username() string {
	return conn.getUsername()
}

// IsEncrypted will return true if the underlying connection is TLS-encrypted.
func (conn *Conn) IsEncrypted() bool {
	return conn.isEncrypted
}
