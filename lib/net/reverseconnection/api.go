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
	conn  *Conn
	error error
}

type Conn struct {
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

type ReverseListenerConfig struct {
	Network         string
	ServerAddress   string
	MinimumInterval time.Duration
	MaximumInterval time.Duration
}

func (conn *Conn) Close() error {
	return conn.close()
}

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

func (l *Listener) RequestConnections(serviceName string) error {
	return l.requestConnections(serviceName)
}

func NewDialer(rawDialer *net.Dialer, serveMux *http.ServeMux,
	minimumInterval, maximumInterval time.Duration,
	logger log.DebugLogger) *Dialer {
	return newDialer(rawDialer, serveMux, minimumInterval, maximumInterval,
		logger)
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dial(network, address)
}

func (d *Dialer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	d.serveHTTP(w, req)
}
