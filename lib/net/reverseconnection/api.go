package reverseconnection

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	libnet "github.com/Symantec/Dominator/lib/net"
)

type acceptEvent struct {
	conn  *listenerConn
	error error
}

type listenerConn struct {
	libnet.TCPConn
	listener *Listener
}

type Dialer struct {
	dialer            *net.Dialer
	minimumInterval   time.Duration
	maximumInterval   time.Duration
	logger            log.DebugLogger
	connectionMapLock sync.Mutex
	connectionMap     map[string]net.Conn // Key: address (ip:port).
}

type ip4Address [4]byte

type Listener struct {
	listener          net.Listener
	portNumber        uint
	logger            log.DebugLogger
	acceptChannel     chan acceptEvent
	closed            bool
	connectionMapLock sync.Mutex
	connectionMap     map[ip4Address]uint
}

// ReverseListenerConfig describes the configuration for a remote server for
// which connections are requested.
type ReverseListenerConfig struct {
	Network         string        // May be empty or "tcp".
	ServerAddress   string        // Address of the remote server.
	MinimumInterval time.Duration // Minimum interval to request connections.
	MaximumInterval time.Duration // Maximum interval to request connections.
}

// Listen creates a listener which may be used to accept incoming connections.
// It listens on all available IP addresses on the local system.
func Listen(network string, portNumber uint, logger log.DebugLogger) (
	*Listener, error) {
	return listen(network, portNumber, logger)
}

func (l *Listener) Accept() (net.Conn, error) {
	return l.accept()
}

func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

func (l *Listener) Close() error {
	return l.close()
}

// RequestConnections starts a goroutine which will periodically attempt to
// establish a connection with a remote server if there is no incoming
// connection from the remote server. The connection that is established will be
// returned by the Accept method. The configuration information for the remote
// server is read from the JSON-encoded file with filename:
// "/etc/reverse-listeners/"+serviceName with the format ReverseListenerConfig.
func (l *Listener) RequestConnections(serviceName string) error {
	return l.requestConnections(serviceName)
}

// NewDialer creates a dialer that may be used to make connections. It also
// registers a HTTP handler for receiving connections from remote systems which
// have requested connections. When the Dial method is called, if a new
// connection has been received it is used instead of dialing out the normal
// way. If rawDialer is nil, the default dialer is used to dial out when needed.
// If serveMux is nil then the default http.ServeMux is used. NewDialer may be
// called only once per serveMux.
// The minimumInterval and maximumInterval parameters are passed back to remote
// systems which are making connections, overriding their default configuration.
func NewDialer(rawDialer *net.Dialer, serveMux *http.ServeMux,
	minimumInterval, maximumInterval time.Duration,
	logger log.DebugLogger) *Dialer {
	return newDialer(rawDialer, serveMux, minimumInterval, maximumInterval,
		logger)
}

// Dial makes a connection to a remote address, possibly consuming a connection
// that was initiated by the remote address.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dial(network, address)
}
