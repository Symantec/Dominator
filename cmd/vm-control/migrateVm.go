package main

import (
	"fmt"
	"net"
	"time"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func migrateVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := migrateVm(args[0], logger); err != nil {
		return fmt.Errorf("Error migrating VM: %s", err)
	}
	return nil
}

func discardAccessToken(hypervisor *srpc.Client, ipAddr net.IP) {
	request := hyper_proto.DiscardVmAccessTokenRequest{IpAddress: ipAddr}
	var reply hyper_proto.DiscardVmAccessTokenResponse
	hypervisor.RequestReply("Hypervisor.DiscardVmAccessToken", request, &reply)
}

func getVmAccessTokenClient(hypervisor *srpc.Client,
	ipAddr net.IP) ([]byte, error) {
	request := hyper_proto.GetVmAccessTokenRequest{ipAddr, time.Hour * 24}
	var reply hyper_proto.GetVmAccessTokenResponse
	err := hypervisor.RequestReply("Hypervisor.GetVmAccessToken", request,
		&reply)
	if err != nil {
		return nil, err
	}
	if err := errors.New(reply.Error); err != nil {
		return nil, err
	}
	return reply.Token, nil
}

func migrateVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := searchVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return migrateVmFromHypervisor(hypervisor, vmIP, logger)
	}
}

func migrateVmFromHypervisor(sourceHypervisorAddress string, vmIP net.IP,
	logger log.DebugLogger) error {
	destHypervisorAddress, err := getHypervisorAddress()
	if err != nil {
		return err
	}
	sourceHypervisor, err := dialHypervisor(sourceHypervisorAddress)
	if err != nil {
		return err
	}
	defer sourceHypervisor.Close()
	vmInfo, err := hyperclient.GetVmInfo(sourceHypervisor, vmIP)
	if err != nil {
		return err
	} else if vmInfo.State == hyper_proto.StateMigrating {
		return errors.New("VM is migrating")
	}
	accessToken, err := getVmAccessTokenClient(sourceHypervisor, vmIP)
	if err != nil {
		return err
	}
	defer discardAccessToken(sourceHypervisor, vmIP)
	destHypervisor, err := dialHypervisor(destHypervisorAddress)
	if err != nil {
		return err
	}
	defer destHypervisor.Close()
	logger.Debugf(0, "migrating VM to %s\n", destHypervisorAddress)
	conn, err := destHypervisor.Call("Hypervisor.MigrateVm")
	if err != nil {
		return err
	}
	defer conn.Close()
	request := hyper_proto.MigrateVmRequest{
		AccessToken:      accessToken,
		IpAddress:        vmIP,
		SourceHypervisor: sourceHypervisorAddress,
	}
	if err := conn.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var reply hyper_proto.MigrateVmResponse
		if err := conn.Decode(&reply); err != nil {
			return err
		}
		if reply.Error != "" {
			return errors.New(reply.Error)
		}
		if reply.ProgressMessage != "" {
			logger.Debugln(0, reply.ProgressMessage)
		}
		if reply.RequestCommit {
			if err := requestCommit(conn); err != nil {
				return err
			}
		}
		if reply.Final {
			break
		}
	}
	return nil
}

func requestCommit(conn *srpc.Conn) error {
	userResponse, err := askForInputChoice("Commit VM",
		[]string{"commit", "abandon"})
	if err != nil {
		return err
	}
	var response hyper_proto.MigrateVmResponseResponse
	switch userResponse {
	case "abandon":
	case "commit":
		response.Commit = true
	default:
		return fmt.Errorf("invalid response: %s", userResponse)
	}
	if err := conn.Encode(response); err != nil {
		return err
	}
	return conn.Flush()
}
