package connpool

import (
	"errors"
	"github.com/Symantec/Dominator/lib/resourcepool"
	"net"
	"time"
)

var (
	ErrorResourceLimitExceeded = errors.New("resource limit exceeded")
)

// GetResourcePool returns a global resourcepool.Pool which may be used by other
// packages which need to share a common resource pool with the connpool
// package. This is needed for efficient sharing of the underlying file
// descriptors (connection slots).
func GetResourcePool() *resourcepool.Pool {
	return getResourcePool()
}

// ConnResource manages a single Conn.
type ConnResource struct {
	network    string
	address    string
	resource   *resourcepool.Resource
	conn       *Conn
	closeError error
}

// New returns a ConnResource for the specified network address. It may be used
// later to obtain a Conn network connection which is part of a managed pool of
// connection slots (to limit consumption of resources such as file
// descriptors). Connections can be released with the Put method but the
// underlying connection may be kept open for later re-use. The Conn is placed
// on an internal list.
// A typical programming pattern is:
//   cr := New(...)
//   c := cr.Get(...)
//   defer c.Put()
//   if err { c.Close() }
//   c := cr.Get(...)
//   defer c.Put()
//   if err { c.Close() }
// This pattern ensures Get* and Put are always matched, and if there is a
// communications error, Close shuts down the client so that a subsequent Get*
// creates a new connection.
func New(network, address string) *ConnResource {
	return newConnResource(network, address)
}

// Get will return a Conn network connection. It implements the net.Conn
// interface from the standard library. If wait is true then Get will wait for
// an available resource, otherwise Get will return ErrorResourceLimitExceeded
// if a resource is not immediately available. The timeout specifies how long
// to wait (after a resource is available) to make the connection. If timeout is
// zero or less, the underlying OS timeout is used (typically 3 minutes for
// TCP).
// Get will panic if it is called without an intervening Close or Put.
func (cr *ConnResource) Get(wait bool, timeout time.Duration) (*Conn, error) {
	return cr.get(wait, timeout)
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
