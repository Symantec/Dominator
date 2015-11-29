package srpc

import "net"

func RegisterName(name string, rcvr interface{}) error {
	return registerName(name, rcvr)
}

func Call(conn net.Conn, serviceMethod string) error {
	return call(conn, serviceMethod)
}
