package connpool

import (
	"net"
	"time"
)

func newConnResource(network, address string) *ConnResource {
	connResource := &ConnResource{
		network: network,
		address: address,
	}
	connResource.privateConnResource.connResource = connResource
	rp := GetResourcePool()
	connResource.resource = rp.Create(&connResource.privateConnResource)
	return connResource
}

func (cr *ConnResource) get(cancelChannel <-chan struct{},
	timeout time.Duration) (*Conn, error) {
	cr.privateConnResource.dialTimeout = timeout
	if err := cr.resource.Get(cancelChannel); err != nil {
		return nil, err
	}
	return cr.conn, nil
}

func (conn *Conn) close() error {
	return conn.resource.resource.Release()
}

func (conn *Conn) put() {
	conn.resource.resource.Put()
}

func (pcr *privateConnResource) Allocate() error {
	cr := pcr.connResource
	netConn, err := net.DialTimeout(cr.network, cr.address, pcr.dialTimeout)
	if err != nil {
		return err
	}
	conn := &Conn{Conn: netConn, resource: cr}
	cr.conn = conn
	return nil
}

func (pcr *privateConnResource) Release() error {
	cr := pcr.connResource
	err := cr.conn.Close()
	cr.conn = nil
	return err
}
