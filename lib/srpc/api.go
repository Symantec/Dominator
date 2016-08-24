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

// CheckTlsRequired returns true if the server requires TLS connections with
// trusted certificates. It returns false if unencrypted or unauthenticated
// connections are permitted (i.e. insecure mode).
func CheckTlsRequired() bool {
	return tlsRequired
}

// LoadCertificates loads zero or more X509 certificates from directory. Each
// certificate must be stored in a pair of PEM-encoded files, with the private
// key in a file with extension '.key' and the corresponding public key
// certificate in a file with extension 'cert'. If there is an error loading a
// certificate pair then processing stops and the error is returned.
func LoadCertificates(directory string) ([]tls.Certificate, error) {
	return loadCertificates(directory)
}

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

type Client struct {
	conn        net.Conn
	isEncrypted bool
	isManaged   bool
	free        bool
	closed      bool
	bufrw       *bufio.ReadWriter
	callLock    sync.Mutex
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

// GetHTTP is similar to DialHTTP except that the returned Client is part of a
// managed pool of connection slots (to limit consumption of resources such as
// file descriptors). Clients can be released with the Put method but the
// underlying connection may be kept open for later re-use. The Client is placed
// on an internal list. An attempt to Get a Client with the same
// (network, address) tuple again without a Put will cause a panic. If wait is
// true then the function will block if there are no free connection slots
// available, else it will return (nil, nil).
// A typical programming pattern is:
//   c := srpc.GetHttp(...)
//   defer c.Put()
//   if err { c.Close() }
// This pattern ensures Get* and Put are always matched, and if there is a
// communications error, Close shuts down the client so that a subsequent Get*
// creates a new connection.
func GetHTTP(network, address string, timeout time.Duration, wait bool) (
	*Client, error) {
	return getHTTP(network, address, clientTlsConfig, timeout, wait)
}

// GetTlsHTTP is similar to DialTlsHTTP but returns a Client that is part of a
// managed pool like GetHTTP returns.
func GetTlsHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration, wait bool) (*Client, error) {
	if tlsConfig == nil {
		tlsConfig = clientTlsConfig
	}
	return getHTTP(network, address, tlsConfig, timeout, wait)
}

// Close will close a client, immediately releasing the internal connection.
func (client *Client) Close() error {
	return client.close()
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

// Put releases a client that was previously created using one of the Get*
// functions. It may be internally closed later if required to free limited
// resources (such as file descriptors). No methods may be called after Put is
// called. If Put is called after Close, no action is taken (this is a safe
// operation and is commonly used in some programming patterns).
func (client *Client) Put() {
	client.put(false)
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
