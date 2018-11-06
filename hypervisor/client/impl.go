package client

import (
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func deleteVmVolume(client *srpc.Client, ipAddr net.IP, accessToken []byte,
	volumeIndex uint) error {
	request := proto.DeleteVmVolumeRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
		VolumeIndex: volumeIndex,
	}
	var reply proto.DeleteVmVolumeResponse
	err := client.RequestReply("Hypervisor.DeleteVmVolume", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func destroyVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	request := proto.DestroyVmRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
	}
	var reply proto.DestroyVmResponse
	err := client.RequestReply("Hypervisor.DestroyVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func prepareVmForMigration(client *srpc.Client, ipAddr net.IP,
	accessToken []byte, enable bool) error {
	request := proto.PrepareVmForMigrationRequest{
		AccessToken: accessToken,
		Enable:      enable,
		IpAddress:   ipAddr,
	}
	var reply proto.PrepareVmForMigrationResponse
	err := client.RequestReply("Hypervisor.PrepareVmForMigration",
		request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func startVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	request := proto.StartVmRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
	}
	var reply proto.StartVmResponse
	err := client.RequestReply("Hypervisor.StartVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func stopVm(client *srpc.Client, ipAddr net.IP, accessToken []byte) error {
	request := proto.StopVmRequest{
		AccessToken: accessToken,
		IpAddress:   ipAddr,
	}
	var reply proto.StopVmResponse
	err := client.RequestReply("Hypervisor.StopVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
