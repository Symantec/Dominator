package main

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func dialFleetManager(address string) (*srpc.Client, error) {
	return srpc.DialHTTP("tcp", address, 0)
}

func dialHypervisor(address string) (*srpc.Client, error) {
	return srpc.DialHTTP("tcp", address, 0)
}
