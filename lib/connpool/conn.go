package connpool

import (
	"net"
	"time"
)

func newConnResource(network, address string) *ConnResource {
	return &ConnResource{
		network:  network,
		address:  address,
		resource: GetResourcePool().Create(),
	}
}

func (cr *ConnResource) get(wait bool, timeout time.Duration) (*Conn, error) {
	if !cr.resource.Get(wait) {
		return nil, ErrorResourceLimitExceeded
	}
	if cr.conn != nil {
		return cr.conn, nil
	}
	netConn, err := net.DialTimeout(cr.network, cr.address, timeout)
	if err != nil {
		cr.resource.Put() // Free up a slot for someone else.
		return nil, err
	}
	conn := &Conn{Conn: netConn, resource: cr}
	cr.conn = conn
	cr.resource.SetReleaseFunc(cr.releaseCallback)
	return conn, nil
}

func (cr *ConnResource) releaseCallback() {
	cr.closeError = cr.conn.Close()
	cr.conn = nil
}

func (conn *Conn) close() error {
	conn.resource.resource.Release()
	return conn.resource.closeError
}

func (conn *Conn) put() {
	conn.resource.resource.Put()
}
