/*
	Package srpc is similar to the net/rpc package in the Go standard library,
	except that it provides streaming RPC access, TLS support and authentication
	and authorisation using X509 client certificates.

	Package srpc provides access to the exported methods of an object across a
	network or other I/O connection. A server registers an object, making it
	visible as a service with the name of the type of the object. After
	registration, exported methods of the object will be accessible remotely.
	A server may register multiple objects (services) of different types but it
	is an error to register multiple objects of the same type.

	The remainder of this documentation describes the protocol, to assist the
	development of implementations in other languages.

	Internally, multiple URL paths are registered with the HTTP default mux:
	  /_goSRPC_/              Unsecured (no TLS, no auth), GOB coder.
	  /_go_TLS_SRPC_/         Secured (TLS, full auth), GOB coder.
	  /_SRPC_/unsecured/JSON  Unsecured (no TLS, no auth), JSON coder.
	  /_SRPC_/TLS/JSON        Secured (TLS, full auth), JSON coder.
	Thus, a web server may also support SRPC on the same port.

	A client issues a HTTP CONNECT request to a server and (for secured
	connections) performs a TLS handshake. If the server requires the TLS
	handshake prior to the HTTP CONNECT, the client will retry with that mode.

	Once connected, a client may issue a sequence of RPC calls, one at a time
	per connection. The client sends the name of the RPC method to call,
	followed by a newline character (a carriage return+newline is permitted).
	For a secured connection, the server will verify if the client X509
	certificate is signed by a trusted CA and if the method is listed in the
	list of permitted methods in the certificate.
	If the method call is established, the server sends a newline character. If
	the method call is rejected then an error message followed by a newline is
	sent.

	The server then calls a registered method hander. The client and server can
	exchange messages using the appropriate coder (GOB is preferred, JSON is
	available as a fallback). Most method handlers wait for client messages and
	then respond. Once the method handler exits (without an error code), the
	server waits for another method call.
*/
package srpc

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"net"
	"sync"
	"time"

	libnet "github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/resourcepool"
)

var (
	ErrorConnectionRefused    = errors.New("connection refused")
	ErrorNoRouteToHost        = errors.New("no route to host")
	ErrorMissingCertificate   = errors.New("missing certificate")
	ErrorBadCertificate       = errors.New("bad certificate")
	ErrorNoSrpcEndpoint       = errors.New("no SRPC endpoint")
	ErrorAccessToMethodDenied = errors.New("access to method denied")

	ErrorCloseClient = errors.New("close client")
)

var (
	clientTlsConfig    *tls.Config
	fullAuthCaCertPool *x509.CertPool
	serverTlsConfig    *tls.Config
	tlsRequired        bool

	srpcProxy = flag.String("srpcProxy", "",
		"Proxy to use (only works for some operations)")
)

// CheckTlsRequired returns true if the server requires TLS connections with
// trusted certificates. It returns false if unencrypted or unauthenticated
// connections are permitted (i.e. insecure mode).
func CheckTlsRequired() bool {
	return tlsRequired
}

// GetEarliestClientCertExpiration returns the earliest expiration time of any
// certificate registered with RegisterClientTlsConfig. The zero value is
// returned if there are no certificates with an expiration time.
func GetEarliestClientCertExpiration() time.Time {
	return getEarliestClientCertExpiration()
}

// LoadCertificates loads zero or more X509 certificates from directory. Each
// certificate must be stored in a pair of PEM-encoded files, with the private
// key in a file with extension '.key' and the corresponding public key
// certificate in a file with extension 'cert'. If there is an error loading a
// certificate pair then processing stops and the error is returned.
func LoadCertificates(directory string) ([]tls.Certificate, error) {
	return loadCertificates(directory)
}

type AuthInformation struct {
	GroupList        map[string]struct{}
	HaveMethodAccess bool
	Username         string
}

// Dialer implements a dialer that can be use to create connections.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type Decoder interface {
	Decode(e interface{}) error
}

type Encoder interface {
	Encode(e interface{}) error
}

// MethodBlocker defines an interface to block method calls (after possible
// authorisation) for a receiver (passed to RegisterName). This may be used to
// attach rate limiting polcies for method calls.
type MethodBlocker interface {
	// BlockMethod is called after method access is granted, prior to calling
	// the method. After the method call completes, the returned function is
	// called. If this is nil, no function is called. If a non-nil error is
	// returned then the method call is blocked and the remote caller will
	// receive the error.
	BlockMethod(methodName string, authInfo *AuthInformation) (func(), error)
}

// MethodGranter defines an interface to grant method calls (if access is not
// granted by the built-in authorisation mechanism) for a receiver (passed to
// RegisterName).
type MethodGranter interface {
	// GrantMethod is called to check if method access should be granted. If
	// access should be granted, the method should return true.
	GrantMethod(serviceMethod string, authInfo *AuthInformation) bool
}

// RegisterName publishes in the server the set of methods of the receiver
// value that satisfy one of the following interfaces:
//   func Method(*Conn) error
//   func Method(*Conn, Decoder, Encoder) error
//   func Method(*Conn, request, *response) error
// The request/response method must not perform I/O on the Conn type. This is
// passed only to provide access to connection metadata.
// If rcvr implements MethodBlocker then the BlockMethod method will be called
// as needed.
// The name of the receiver (service) is given by name.
func RegisterName(name string, rcvr interface{}) error {
	return registerName(name, rcvr, ReceiverOptions{})
}

func RegisterNameWithOptions(name string, rcvr interface{},
	options ReceiverOptions) error {
	return registerName(name, rcvr, options)
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

// RegisterFullAuthCA registers the CA certificate pool used for full
// authentication/authorisation checks (including method checks). If not
// specified, the CA certificate pool registered with RegisterServerTlsConfig is
// used for full auth checks. This allows for distinguishing between CAs trusted
// for everything versus CAs trusted only for identity (username and groups).
func RegisterFullAuthCA(certPool *x509.CertPool) {
	fullAuthCaCertPool = certPool
}

type privateClientResource struct {
	clientResource *ClientResource
	tlsConfig      *tls.Config
	dialer         Dialer
}

type ClientResource struct {
	network               string
	address               string
	resource              *resourcepool.Resource
	privateClientResource privateClientResource
	client                *Client
	inUse                 bool
	closeError            error
}

// SetDefaultGrantMethod registers the grantMethod function which will be
// called to grant access to methods (if access is not granted by the built-in
// authorisation mechanism) for all receivers. This is overridden by receivers
// which implement the MethodGranter interface.
// The default is to not grant access to methods (if the built-in authorisation
// mechanism does not grant access).
func SetDefaultGrantMethod(grantMethod func(serviceMethod string,
	authInfo *AuthInformation) bool) {
	defaultGrantMethod = grantMethod
}

// NewClientResource returns a ClientResource which may be later used to Get*
// a Client which is part of a managed pool of connection slots (to limit
// consumption of resources such as file descriptors). Clients can be released
// with the Put method but the underlying connection may be kept open for later
// re-use. The Client is placed on an internal list.
// A typical programming pattern is:
//   cr := NewClientResource(...)
//   c := cr.GetHttp(...)
//   defer c.Put()
//   if err { c.Close() }
//   c := cr.GetHttp(...)
//   defer c.Put()
//   if err { c.Close() }
// This pattern ensures Get* and Put are always matched, and if there is a
// communications error, Close shuts down the client so that a subsequent Get*
// creates a new connection.
func NewClientResource(network, address string) *ClientResource {
	return newClientResource(network, address)
}

// GetHTTP is similar to DialHTTP except that the returned Client is part of a
// managed pool of connection slots (to limit consumption of resources such as
// file descriptors). GetHTTP will wait until a resource is available or
// a message is received on cancelChannel. If cancelChannel is nil then GetHTTP
// will wait indefinitely until a resource is available. If the wait is
// cancelled then GetHTTP will return ErrorResourceLimitExceeded. The timeout
// specifies how long to wait (after a resource is available) to make the
// connection. If timeout is zero or less, the underlying OS timeout is used
// (typically 3 minutes for TCP).
func (cr *ClientResource) GetHTTP(cancelChannel <-chan struct{},
	timeout time.Duration) (*Client, error) {
	return cr.getHTTP(clientTlsConfig, cancelChannel,
		&net.Dialer{Timeout: timeout})
}

// GetHTTPWithDialer is similar to GetHTTP except that the dialer is used to
// create the underlying connection.
func (cr *ClientResource) GetHTTPWithDialer(cancelChannel <-chan struct{},
	dialer Dialer) (*Client, error) {
	return cr.getHTTP(clientTlsConfig, cancelChannel, dialer)
}

// GetTlsHTTP is similar to DialTlsHTTP but returns a Client that is part of a
// managed pool like the GetHTTP method returns.
func (cr *ClientResource) GetTlsHTTP(tlsConfig *tls.Config,
	cancelChannel <-chan struct{}, timeout time.Duration) (*Client, error) {
	return cr.GetTlsHTTPWithDialer(tlsConfig, cancelChannel,
		&net.Dialer{Timeout: timeout})
}

// GetTlsHTTPWithDialer is similar to GetTlsHTTP except that the dialer is used
// to create the underlying connection.
func (cr *ClientResource) GetTlsHTTPWithDialer(tlsConfig *tls.Config,
	cancelChannel <-chan struct{}, dialer Dialer) (*Client, error) {
	if tlsConfig == nil {
		tlsConfig = clientTlsConfig
	}
	return cr.getHTTP(tlsConfig, cancelChannel, dialer)
}

func (cr *ClientResource) ScheduleClose() {
	cr.resource.ScheduleRelease()
}

type Client struct {
	bufrw       *bufio.ReadWriter
	callLock    sync.Mutex
	conn        net.Conn
	isEncrypted bool
	makeCoder   coderMaker
	resource    *ClientResource
	tcpConn     libnet.TCPConn // The underlying raw connection.
}

// DialHTTP connects to an HTTP SRPC server at the specified network address
// listening on the HTTP SRPC path. If timeout is zero or less, the underlying
// OS timeout is used (typically 3 minutes for TCP).
func DialHTTP(network, address string, timeout time.Duration) (*Client, error) {
	netDialer := &net.Dialer{Timeout: timeout}
	if *srpcProxy != "" {
		dialer, err := newProxyDialer(*srpcProxy, netDialer)
		if err != nil {
			return nil, err
		}
		return dialHTTP(network, address, clientTlsConfig, dialer)
	}
	return dialHTTP(network, address, clientTlsConfig, netDialer)
}

// DialHTTPWithDialer is similar to DialHTTP except that the dialer is used to
// create the underlying connection.
func DialHTTPWithDialer(network, address string, dialer Dialer) (
	*Client, error) {
	return dialHTTP(network, address, clientTlsConfig, dialer)
}

// DialTlsHTTP connects to an HTTP SRPC TLS server at the specified network
// address listening on the HTTP SRPC TLS path. If timeout is zero or less, the
// underlying OS timeout is used (typically 3 minutes for TCP).
func DialTlsHTTP(network, address string, tlsConfig *tls.Config,
	timeout time.Duration) (
	*Client, error) {
	return DialTlsHTTPWithDialer(network, address, tlsConfig,
		&net.Dialer{Timeout: timeout})
}

// DialTlsHTTPWithDialer is similar to DialTlsHTTP except that the dialer is
// used to create the underlying connection.
func DialTlsHTTPWithDialer(network, address string, tlsConfig *tls.Config,
	dialer Dialer) (
	*Client, error) {
	if tlsConfig == nil {
		tlsConfig = clientTlsConfig
	}
	return dialHTTP(network, address, tlsConfig, dialer)
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
// methods. It may be internally closed later if required to free limited
// resources (such as file descriptors). No methods may be called after Put is
// called. If Put is called after Close, no action is taken (this is a safe
// operation and is commonly used in some programming patterns).
func (client *Client) Put() {
	client.put()
}

// SetKeepAlive sets whether the operating system should send keepalive messages
// on the connection.
func (client *Client) SetKeepAlive(keepalive bool) error {
	return client.tcpConn.SetKeepAlive(keepalive)
}

// SetKeepAlivePeriod sets the period between keepalive messages.
func (client *Client) SetKeepAlivePeriod(d time.Duration) error {
	return client.tcpConn.SetKeepAlivePeriod(d)
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
	Decoder
	Encoder
	parent      *Client // nil: server-side connection.
	isEncrypted bool
	*bufio.ReadWriter
	remoteAddr       string
	groupList        map[string]struct{}
	haveMethodAccess bool
	username         string              // Empty string for unauthenticated.
	permittedMethods map[string]struct{} // nil: all, empty: none permitted.
	releaseNotifier  func()
}

// Close will close the connection to the Sevice.Method function, releasing the
// Client for a subsequent Call.
func (conn *Conn) Close() error {
	return conn.close()
}

// GetAuthInformation will return authentication information for the client who
// holds the certificate used to authenticate the connection to the server. If
// the connection was not authenticated nil is returned. If the connection is a
// client connection, then GetAuthInformation will panic.
func (conn *Conn) GetAuthInformation() *AuthInformation {
	return conn.getAuthInformation()
}

// GetCloseNotifier will create a goroutine which reads from the connection
// until it closes or there is a read error. The error (which is nil if the
// connection closed) is sent to the channel. All data read are discarded.
func (conn *Conn) GetCloseNotifier() <-chan error {
	return conn.getCloseNotifier()
}

// IsEncrypted will return true if the underlying connection is TLS-encrypted.
func (conn *Conn) IsEncrypted() bool {
	return conn.isEncrypted
}

// RemoteAddr returns the remote network address. This is currently only
// implemented for server-side connections.
func (conn *Conn) RemoteAddr() string {
	return conn.remoteAddr
}

// RequestReply sends a request message to a connection and waits for a reply.
// The request and reply messages are GOB encoded and decoded, respectively.
func (conn *Conn) RequestReply(request interface{}, reply interface{}) error {
	return conn.requestReply(request, reply)
}

// Username will return the username of the client who holds the certificate
// used to authenticate the connection to the server. If the connection was not
// authenticated the empty string is returned. If the connection is a client
// connection, then Username will panic.
func (conn *Conn) Username() string {
	return conn.getUsername()
}

type ReceiverOptions struct {
	PublicMethods []string
}
