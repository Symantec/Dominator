package client

import (
	"net"

	"github.com/Symantec/Dominator/lib/srpc"
)

func DestroyVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	return destroyVm(client, ipAddr, accessToken)
}

func PrepareVmForMigration(client *srpc.Client, ipAddr net.IP,
	accessToken []byte, enable bool) error {
	return prepareVmForMigration(client, ipAddr, accessToken, enable)
}

func StartVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	return startVm(client, ipAddr, accessToken)
}

func StopVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	return stopVm(client, ipAddr, accessToken)
}
