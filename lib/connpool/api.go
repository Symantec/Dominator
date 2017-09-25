/*
	Package connpool provides for managing network connections with a resource
	pool.

	Package connpool may be used to create and free network connections.
	The number of concurrent network connections that may be open is limited to
	fit within the underlying file descriptor limit. Connections may be placed
	on an internal freelist for later re-use, potentially eliminating connection
	setup overhead for frequently re-opened connections.

	An application will typically call New once for each network address it
	expects to later connect to. When the application wants to connect it calls
	the Get method and calls the Put method to release the connection.
	A typical programming pattern is:
		cr0 := New(...)
		cr1 := New(...)
		go func() {
			for ... {
				c := cr0.Get(...)
				defer c.Put()
				if err { c.Close() }
			}
		}()
		go func() {
			for ... {
				c := cr1.Get(...)
				defer c.Put()
				if err { c.Close() }
			}
		}()
	This pattern ensures Get and Put are always matched, and if there is a
	communications error, Close shuts down the connection so that a subsequent
	Get	creates a new underlying connection.

	It is resonable to create one goroutine for each resource, since the Get
	methods will block, waiting for available resources.
*/
package connpool

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/resourcepool"
)

// Dialer implements a dialer that can be use to create connections.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

// GetResourcePool returns a global resourcepool.Pool which may be used by other
// packages which need to share a common resource pool with the connpool
// package. This is needed for efficient sharing of the underlying file
// descriptors (connection slots).
func GetResourcePool() *resourcepool.Pool {
	return getResourcePool()
}

type privateConnResource struct {
	connResource *ConnResource
	dialer       Dialer
}

// ConnResource manages a single Conn.
type ConnResource struct {
	network             string
	address             string
	resource            *resourcepool.Resource
	privateConnResource privateConnResource
	conn                *Conn
	netConn             net.Conn
}

// New returns a ConnResource for the specified network address. It may be used
// later to obtain a Conn network connection which is part of a managed pool of
// connection slots (to limit consumption of resources such as file
// descriptors). Connections can be released with the Put method but the
// underlying connection may be kept open for later re-use. The Conn is placed
// on an internal list.
func New(network, address string) *ConnResource {
	return newConnResource(network, address)
}

// Get will return a Conn network connection. It implements the net.Conn
// interface from the standard library. Get will wait until a resource is
// available or a message is received on cancelChannel. If cancelChannel is nil
// then Get will wait indefinitely until a resource is available. If the wait is
// cancelled then Get will return ErrorResourceLimitExceeded. The timeout
// specifies how long to wait (after a resource is available) to make the
// connection. If timeout is zero or less, the underlying OS timeout is used
// (typically 3 minutes for TCP).
// Get will panic if it is called again without an intervening Close or Put.
func (cr *ConnResource) Get(cancelChannel <-chan struct{},
	timeout time.Duration) (*Conn, error) {
	return cr.get(cancelChannel, &net.Dialer{Timeout: timeout})
}

// GetWithDialer will return a Conn network connection which implements the
// net.Conn interface from the standard library. GetWithDialer will wait until
// a resource is available or a message is received on cancelChannel. If
// cancelChannel is nil then GetWithDialer will wait indefinitely until a
// resource is available. If the wait is cancelled then GetWithDialer will
// return ErrorResourceLimitExceeded.
// The dialer is used to perform the operation which creates the connection. A
// *net.Dialer type from the standard library satisfies the Dialer interface,
// and may be used to specify how long to wait (after a resource is available)
// to make the connection. The OS may impose it's own timeout (typically 3
// minutes for TCP). A different dialer may be used to create TLS connections.
// Note that changing dialer types does not guarantee the connection type
// returned, as the dialer may not be called on every call to GetWithDialer,
// thus changing dialer types will result in unpredictable behaviour.
// GetWithDialer will panic if it is called again without an intervening Close
// or Put.
func (cr *ConnResource) GetWithDialer(cancelChannel <-chan struct{},
	dialer Dialer) (*Conn, error) {
	return cr.get(cancelChannel, dialer)
}

// ScheduleClose will immediatly Close the associated Conn if it is not in use
// or will mark the Conn to be closed when it is next Put.
func (cr *ConnResource) ScheduleClose() {
	cr.resource.ScheduleRelease()
}

// Conn is a managed network connection. It implements the net.Conn interface
// from the standard library.
type Conn struct {
	net.Conn
	resource *ConnResource
}

// Close will close the connection, immediately freeing the underlying resource.
// It may no longer be used for communication.
func (conn *Conn) Close() error {
	return conn.close()
}

// Put will release the connection. It may no longer be used for communication.
// It may be internally closed later if required to free limited resources (such
// as file descriptors). If Put is called after Close, no action is taken (this
// is a safe operation and is commonly used in some programming patterns).
func (conn *Conn) Put() {
	conn.put()
}
