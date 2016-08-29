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

func GetResourcePool() *resourcepool.Pool {
	return getResourcePool()
}

type ConnResource struct {
	network    string
	address    string
	resource   *resourcepool.Resource
	conn       *Conn
	closeError error
}

func New(network, address string) *ConnResource {
	return newConnResource(network, address)
}

func (cr *ConnResource) Get(wait bool, timeout time.Duration) (*Conn, error) {
	return cr.get(wait, timeout)
}

func (cr *ConnResource) ScheduleClose() {
	cr.resource.ScheduleRelease()
}

type Conn struct {
	net.Conn
	resource *ConnResource
}

func (conn *Conn) Close() error {
	return conn.close()
}

func (conn *Conn) Put() {
	conn.put()
}
