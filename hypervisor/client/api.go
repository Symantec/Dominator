package client

import (
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func AcknowledgeVm(client *srpc.Client, ipAddress net.IP) error {
	return acknowledgeVm(client, ipAddress)
}

func CreateVm(client *srpc.Client, request proto.CreateVmRequest,
	reply *proto.CreateVmResponse, logger log.DebugLogger) error {
	return createVm(client, request, reply, logger)
}

func DeleteVmVolume(client *srpc.Client, ipAddr net.IP, accessToken []byte,
	volumeIndex uint) error {
	return deleteVmVolume(client, ipAddr, accessToken, volumeIndex)
}

func DestroyVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	return destroyVm(client, ipAddr, accessToken)
}

func ExportLocalVm(client *srpc.Client, ipAddr net.IP,
	verificationCookie []byte) (proto.ExportLocalVmInfo, error) {
	return exportLocalVm(client, ipAddr, verificationCookie)
}

func GetRootCookiePath(client *srpc.Client) (string, error) {
	return getRootCookiePath(client)
}

func GetVmInfo(client *srpc.Client, ipAddr net.IP) (proto.VmInfo, error) {
	return getVmInfo(client, ipAddr)
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
