package rpc

import (
	"github.com/Symantec/Dominator/lib/net"
	"net/rpc"
)

// DialHTTPPath works like DialHTTPPath in net/rpc but accepts a custom
// Dialer
func DialHTTPPath(dialer net.Dialer, network, address, path string) (
	*rpc.Client, error) {
	return dialHTTPPath(dialer, network, address, path)
}
