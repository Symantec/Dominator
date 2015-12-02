package srpc

import (
	"bufio"
	"net"
	"sync"
)

func RegisterName(name string, rcvr interface{}) error {
	return registerName(name, rcvr)
}

func DialHTTP(network, address string) (*Client, error) {
	return dialHTTP(network, address)
}

type Client struct {
	conn     net.Conn
	bufrw    *bufio.ReadWriter
	callLock sync.Mutex
}

func (client *Client) Close() error {
	client.bufrw.Flush()
	return client.conn.Close()
}

func (client *Client) Call(serviceMethod string) (*Conn, error) {
	return client.call(serviceMethod)
}

type Conn struct {
	parent *Client
	*bufio.ReadWriter
}

func (conn *Conn) Close() error {
	return conn.close()
}
