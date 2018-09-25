package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func migrateVmSubcommand(args []string, logger log.DebugLogger) {
	if err := migrateVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error migrating VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	conn, err := destHypervisor.Call("Hypervisor.MigrateVm")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	request := hyper_proto.MigrateVmRequest{
		AccessToken:      accessToken,
		IpAddress:        vmIP,
		SourceHypervisor: sourceHypervisorAddress,
	}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var reply hyper_proto.MigrateVmResponse
		if err := decoder.Decode(&reply); err != nil {
			return err
		}
		if reply.Error != "" {
			return errors.New(reply.Error)
		}
		if reply.ProgressMessage != "" {
			logger.Debugln(0, reply.ProgressMessage)
		}
		if reply.RequestCommit {
			if err := requestCommit(conn, encoder); err != nil {
				return err
			}
		}
		if reply.Final {
			break
		}
	}
	return nil
}

func requestCommit(conn *srpc.Conn, encoder srpc.Encoder) error {
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
		return fmt.Errorf("invalid response: %s", response)
	}
	if err := encoder.Encode(response); err != nil {
		return err
	}
	return conn.Flush()
}
