package client

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func acknowledgeVm(client *srpc.Client, ipAddress net.IP) error {
	request := proto.AcknowledgeVmRequest{ipAddress}
	var reply proto.AcknowledgeVmResponse
	return client.RequestReply("Hypervisor.AcknowledgeVm", request, &reply)
}

func createVm(client *srpc.Client, request proto.CreateVmRequest,
	reply *proto.CreateVmResponse, logger log.DebugLogger) error {
	if conn, err := client.Call("Hypervisor.CreateVm"); err != nil {
		return err
	} else {
		defer conn.Close()
		if err := conn.Encode(request); err != nil {
			return err
		}
		if err := conn.Flush(); err != nil {
			return err
		}
		for {
			var response proto.CreateVmResponse
			if err := conn.Decode(&response); err != nil {
				return fmt.Errorf("error decoding: %s", err)
			}
			if response.Error != "" {
				return errors.New(response.Error)
			}
			if response.ProgressMessage != "" {
				logger.Debugln(0, response.ProgressMessage)
			}
			if response.Final {
				*reply = response
				return nil
			}
		}
	}
}

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

func exportLocalVm(client *srpc.Client, ipAddr net.IP,
	verificationCookie []byte) (proto.ExportLocalVmInfo, error) {
	request := proto.ExportLocalVmRequest{
		IpAddress:          ipAddr,
		VerificationCookie: verificationCookie,
	}
	var reply proto.ExportLocalVmResponse
	err := client.RequestReply("Hypervisor.ExportLocalVm", request, &reply)
	if err != nil {
		return proto.ExportLocalVmInfo{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return proto.ExportLocalVmInfo{}, err
	}
	return reply.VmInfo, nil
}

func getVmInfo(client *srpc.Client, ipAddr net.IP) (proto.VmInfo, error) {
	request := proto.GetVmInfoRequest{IpAddress: ipAddr}
	var reply proto.GetVmInfoResponse
	err := client.RequestReply("Hypervisor.GetVmInfo", request, &reply)
	if err != nil {
		return proto.VmInfo{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return proto.VmInfo{}, err
	}
	return reply.VmInfo, nil
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
