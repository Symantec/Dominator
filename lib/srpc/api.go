package srpc

import (
	"bufio"
	"crypto/tls"
	"net"
	"sync"
)

var serverTlsConfig *tls.Config
var clientTlsConfig *tls.Config

// RegisterName publishes in the server the set of methods of the receiver
// value that satisfy the following interface:
//   func Method(*Conn) error
// The name of the receiver (service) is given by name.
func RegisterName(name string, rcvr interface{}) error {
	return registerName(name, rcvr)
}

func RegisterServerTlsConfig(config *tls.Config) {
	serverTlsConfig = config
}

func RegisterClientTlsConfig(config *tls.Config) {
	clientTlsConfig = config
}

func DialHTTP(network, address string) (*Client, error) {
	return dialHTTP(network, address, clientTlsConfig)
}

func DialTlsHTTP(network, address string, tlsConfig *tls.Config) (
	*Client, error) {
	return dialHTTP(network, address, tlsConfig)
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
