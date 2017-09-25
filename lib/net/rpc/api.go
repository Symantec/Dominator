package rpc

import (
	"net/rpc"

	"github.com/Symantec/Dominator/lib/net"
)

// DialHTTPPath works like DialHTTPPath in net/rpc but accepts a custom
// dialer.
func DialHTTPPath(dialer net.Dialer, network, address, path string) (
	*rpc.Client, error) {
	return dialHTTPPath(dialer, network, address, path)
}

// Dial works like Dial in net/rpc but accepts a custom dialer.
func Dial(dialer net.Dialer, network, address string) (*rpc.Client, error) {
	return dial(dialer, network, address)
}
